import Foundation
import XCTest
@testable import DukuNetLabCore

final class CoreTests: XCTestCase {
    func testParseHostToken() {
        XCTAssertEqual(parseHostToken("OTHER=value\nDUKU_HOST_TOKEN=local-secret\n"), "local-secret")
        XCTAssertNil(parseHostToken("OTHER=value\n"))
        XCTAssertNil(parseHostToken("DUKU_HOST_TOKEN=\n"))
    }

    func testCaptureDurationIsClamped() {
        XCTAssertEqual(clampedCaptureDuration(nil), 120)
        XCTAssertEqual(clampedCaptureDuration(0), 1)
        XCTAssertEqual(clampedCaptureDuration(30), 30)
        XCTAssertEqual(clampedCaptureDuration(500), 120)
    }

    func testLocalCaptureArgumentsAreRestrictedToEn0() {
        XCTAssertEqual(
            localCaptureArguments(helper: "/helper", duration: 500, path: "/staging/test.local.pcap"),
            ["-n", "/helper", "start-local", "en0", "1", "/staging/test.local.pcap"]
        )
    }

    func testDefaultRepositoryPathInfersRepoFromPackagedApp() {
        XCTAssertEqual(
            defaultRepositoryPath(currentDirectory: "/", bundlePath: "/repo/dist/Duku Net Lab.app"),
            "/repo"
        )
        XCTAssertEqual(
            defaultRepositoryPath(currentDirectory: "/repo", bundlePath: nil),
            "/repo"
        )
    }

    func testHostCommandDecodesNumbersAndBooleans() throws {
        let data = Data(#"{"id":"host-0001","action":"start","args":{"channel":36,"enabled":true,"channels":[36,149],"metadata":{"safe":true},"empty":null}}"#.utf8)
        let command = try JSONDecoder().decode(HostCommand.self, from: data)
        XCTAssertEqual(command.id, "host-0001")
        XCTAssertEqual(command.action, "start")
        XCTAssertEqual(command.args["channel"]?.intValue, 36)
        XCTAssertNil(command.args["enabled"]?.intValue)
    }
}
