// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "DukuNetLabMenu",
    platforms: [.macOS(.v14)],
    products: [.executable(name: "DukuNetLabMenu", targets: ["DukuNetLabMenu"])],
    targets: [
        .target(name: "DukuNetLabCore"),
        .executableTarget(name: "DukuNetLabMenu", dependencies: ["DukuNetLabCore"]),
        .testTarget(name: "DukuNetLabCoreTests", dependencies: ["DukuNetLabCore"]),
    ]
)
