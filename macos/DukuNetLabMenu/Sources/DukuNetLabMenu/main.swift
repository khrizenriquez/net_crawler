import AppKit
import DukuNetLabCore
import SwiftUI

@main
struct DukuNetLabMenuApp: App {
    @StateObject private var model = LabModel()

    var body: some Scene {
        MenuBarExtra {
            LabMenu(model: model)
        } label: {
            Image(systemName: model.status?.activeSession == nil ? "antenna.radiowaves.left.and.right" : "dot.radiowaves.left.and.right")
        }
        .menuBarExtraStyle(.window)
    }
}

struct APIStatus: Decodable {
    struct Session: Decodable { let channel: Int; let band: String; let status: String }
    let labRunning: Bool
    let routerStatus: String
    let recentFindings: Int
    let activeSession: Session?
}

@MainActor
final class LabModel: ObservableObject {
    @Published var status: APIStatus?
    @Published var message = "Laboratorio detenido"
    @Published var isBusy = false
    private var pollTask: Task<Void, Never>?
    private let apiURL = URL(string: "http://127.0.0.1:8080/api/v1/status")!
    private let dashboardURL = URL(string: "http://127.0.0.1:4173")!
    private let detectedRepositoryPath = defaultRepositoryPath(currentDirectory: FileManager.default.currentDirectoryPath, bundlePath: Bundle.main.bundleURL.path)

    init() {
        pollTask = Task { @MainActor [weak self] in
            while !Task.isCancelled {
                await self?.refresh()
                await self?.pollHostCommand()
                try? await Task.sleep(for: .seconds(5))
            }
        }
    }

    deinit {
        pollTask?.cancel()
    }

    var repositoryPath: String {
        UserDefaults.standard.string(forKey: "repositoryPath")
            ?? detectedRepositoryPath
    }

    func refresh() async {
        do {
            let (data, _) = try await URLSession.shared.data(from: apiURL)
            status = try JSONDecoder().decode(APIStatus.self, from: data)
            message = status?.activeSession == nil ? "Laboratorio listo" : "Captura temporal activa"
        } catch {
            status = nil
            message = "Stack local no disponible"
        }
    }

    func pollHostCommand() async {
        guard status != nil else {
            logBridge("skip poll: status unavailable")
            return
        }
        guard let token = hostToken else {
            logBridge("skip poll: host token unavailable at \(repositoryPath)")
            return
        }
        var request = URLRequest(url: URL(string: "http://127.0.0.1:8080/api/v1/host/commands/next")!)
        request.setValue(token, forHTTPHeaderField: "X-Duku-Host-Token")
        do {
            let (data, response) = try await URLSession.shared.data(for: request)
            let statusCode = (response as? HTTPURLResponse)?.statusCode ?? 0
            guard statusCode == 200 else {
                logBridge("poll status=\(statusCode)")
                return
            }
            let command = try JSONDecoder().decode(HostCommand.self, from: data)
            let result = execute(command)
            logBridge("executed \(command.action): \(result.prefix(80))")
            await report(command.id, result: result, token: token)
        } catch {
            logBridge("poll error: \(error.localizedDescription)")
        }
    }

    func start() {
        runPodman(["machine", "start"]) { [weak self] _ in
            self?.runCompose(["-f", "compose.yml", "up", "-d", "--build"]) { _ in
                Task { @MainActor in await self?.refresh() }
            }
        }
    }

    func stop() {
        let helper = "/usr/local/libexec/duku-capture-helper"
        if FileManager.default.isExecutableFile(atPath: helper) {
            _ = try? Process.run(URL(fileURLWithPath: "/usr/bin/sudo"), arguments: ["-n", helper, "stop"]).waitUntilExit()
        }
        runCompose(["-f", "compose.yml", "down"]) { [weak self] _ in
            Task { @MainActor in await self?.refresh() }
        }
    }

    func openDashboard() { NSWorkspace.shared.open(dashboardURL) }

    func chooseRepository() {
        let panel = NSOpenPanel()
        panel.canChooseFiles = false
        panel.canChooseDirectories = true
        panel.allowsMultipleSelection = false
        if panel.runModal() == .OK, let path = panel.url?.path {
            UserDefaults.standard.set(path, forKey: "repositoryPath")
            message = "Repositorio configurado"
        }
    }

    private var hostToken: String? {
        guard let contents = try? String(contentsOfFile: "\(repositoryPath)/.env", encoding: .utf8) else { return nil }
        return parseHostToken(contents)
    }

    private func execute(_ command: HostCommand) -> String {
        let helper = "/usr/local/libexec/duku-capture-helper"
        guard FileManager.default.isExecutableFile(atPath: helper) else { return "helper is not installed" }
        let arguments: [String]
        switch command.action {
        case "start":
            let channel = command.args["channel"]?.intValue ?? 0
            let duration = clampedCaptureDuration(command.args["durationMinutes"]?.intValue)
            let formatter = ISO8601DateFormatter()
            let name = formatter.string(from: Date()).replacingOccurrences(of: ":", with: "-")
            let path = "\(FileManager.default.homeDirectoryForCurrentUser.path)/.duku-net-lab/staging/\(name).pcap"
            arguments = ["-n", helper, "start", "en0", String(channel), String(duration), path]
        case "start-local":
            let duration = clampedCaptureDuration(command.args["durationMinutes"]?.intValue)
            let formatter = ISO8601DateFormatter()
            let name = formatter.string(from: Date()).replacingOccurrences(of: ":", with: "-")
            let path = "\(FileManager.default.homeDirectoryForCurrentUser.path)/.duku-net-lab/staging/\(name).local.pcap"
            arguments = localCaptureArguments(helper: helper, duration: duration, path: path)
        case "stop": arguments = ["-n", helper, "stop"]
        default: return "unsupported host action"
        }
        let process = Process()
        let pipe = Pipe()
        process.executableURL = URL(fileURLWithPath: "/usr/bin/sudo")
        process.arguments = arguments
        process.standardOutput = pipe
        process.standardError = pipe
        do {
            try process.run()
            process.waitUntilExit()
            let output = String(data: pipe.fileHandleForReading.readDataToEndOfFile(), encoding: .utf8) ?? ""
            return "exit=\(process.terminationStatus) \(output)"
        } catch { return error.localizedDescription }
    }

    private func report(_ id: String, result: String, token: String) async {
        var request = URLRequest(url: URL(string: "http://127.0.0.1:8080/api/v1/host/commands/\(id)/result")!)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.setValue(token, forHTTPHeaderField: "X-Duku-Host-Token")
        request.httpBody = try? JSONSerialization.data(withJSONObject: ["result": result])
        _ = try? await URLSession.shared.data(for: request)
        await refresh()
    }

    private func logBridge(_ message: String) {
        let line = "\(Date()) \(message)\n"
        let url = URL(fileURLWithPath: "/tmp/duku-net-lab-menu.log")
        if let data = line.data(using: .utf8) {
            if FileManager.default.fileExists(atPath: url.path),
               let handle = try? FileHandle(forWritingTo: url) {
                _ = try? handle.seekToEnd()
                try? handle.write(contentsOf: data)
                try? handle.close()
            } else {
                try? data.write(to: url, options: .atomic)
            }
        }
    }

    private func runPodman(_ args: [String], completion: @escaping @MainActor @Sendable (Int32) -> Void) {
        runExecutable(paths: ["/opt/homebrew/bin/podman", "/usr/local/bin/podman"], args: args, completion: completion)
    }

    private func runCompose(_ args: [String], completion: @escaping @MainActor @Sendable (Int32) -> Void) {
        runExecutable(paths: ["/opt/homebrew/bin/podman-compose", "/usr/local/bin/podman-compose"], args: args, completion: completion)
    }

    private func runExecutable(paths: [String], args: [String], completion: @escaping @MainActor @Sendable (Int32) -> Void) {
        isBusy = true
        let path = paths.first { FileManager.default.isExecutableFile(atPath: $0) } ?? paths[0]
        DispatchQueue.global(qos: .userInitiated).async { [repositoryPath] in
            let process = Process()
            process.executableURL = URL(fileURLWithPath: path)
            process.arguments = args
            process.currentDirectoryURL = URL(fileURLWithPath: repositoryPath)
            process.standardOutput = FileHandle.nullDevice
            process.standardError = FileHandle.nullDevice
            do { try process.run(); process.waitUntilExit() } catch {}
            let exitCode = process.terminationStatus
            DispatchQueue.main.async { [weak self] in self?.isBusy = false; completion(exitCode) }
        }
    }
}

struct LabMenu: View {
    @ObservedObject var model: LabModel

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            HStack {
                Image(systemName: "antenna.radiowaves.left.and.right").foregroundStyle(.green)
                VStack(alignment: .leading) {
                    Text("Duku Net Lab").font(.headline)
                    Text(model.message).font(.caption).foregroundStyle(.secondary)
                }
            }
            Divider()
            if let session = model.status?.activeSession {
                Label("Sesión \(session.status)", systemImage: "waveform.path.ecg")
                Text("\(session.band) · canal \(session.channel)").font(.caption).foregroundStyle(.secondary)
            } else {
                Label("Sin captura activa", systemImage: "pause.circle")
            }
            if let status = model.status {
                Label("Huawei: \(status.routerStatus)", systemImage: "network")
                Label("\(status.recentFindings) hallazgos recientes", systemImage: "exclamationmark.shield")
            }
            Divider()
            Button("Iniciar laboratorio") { model.start() }.disabled(model.isBusy)
            Button("Abrir dashboard") { model.openDashboard() }
            Button("Detener laboratorio") { model.stop() }.disabled(model.isBusy)
            Divider()
            Button("Seleccionar repositorio…") { model.chooseRepository() }
            Text(model.repositoryPath).font(.caption2).foregroundStyle(.secondary).lineLimit(2)
            Divider()
            Button("Salir") {
                if model.status?.activeSession != nil { model.stop() }
                NSApplication.shared.terminate(nil)
            }
        }
        .padding(16)
        .frame(width: 310)
    }
}
