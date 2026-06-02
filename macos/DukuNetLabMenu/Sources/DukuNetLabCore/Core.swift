import Foundation

public struct HostCommand: Decodable {
    public let id: String
    public let action: String
    public let args: [String: JSONValue]
}

public enum JSONValue: Decodable {
    case string(String)
    case number(Double)
    case bool(Bool)
    case array([JSONValue])
    case object([String: JSONValue])
    case null

    public init(from decoder: Decoder) throws {
        let value = try decoder.singleValueContainer()
        if value.decodeNil() { self = .null; return }
        if let string = try? value.decode(String.self) { self = .string(string); return }
        if let number = try? value.decode(Double.self) { self = .number(number); return }
        if let bool = try? value.decode(Bool.self) { self = .bool(bool); return }
        if let array = try? value.decode([JSONValue].self) { self = .array(array); return }
        self = .object(try value.decode([String: JSONValue].self))
    }

    public var intValue: Int? {
        if case .number(let value) = self { return Int(value) }
        return nil
    }
}

public func parseHostToken(_ contents: String) -> String? {
    contents
        .split(separator: "\n")
        .first(where: { $0.hasPrefix("DUKU_HOST_TOKEN=") })
        .map { String($0.dropFirst("DUKU_HOST_TOKEN=".count)) }
        .flatMap { $0.isEmpty ? nil : $0 }
}

public func clampedCaptureDuration(_ requested: Int?) -> Int {
    min(max(requested ?? 120, 1), 120)
}

public func localCaptureArguments(helper: String, duration: Int, path: String) -> [String] {
    ["-n", helper, "start-local", "en0", String(min(clampedCaptureDuration(duration), 1)), path]
}

public func defaultRepositoryPath(currentDirectory: String, bundlePath: String?) -> String {
    guard let bundlePath, bundlePath.hasSuffix(".app") else { return currentDirectory }
    let bundleURL = URL(fileURLWithPath: bundlePath)
    return bundleURL.deletingLastPathComponent().deletingLastPathComponent().path
}
