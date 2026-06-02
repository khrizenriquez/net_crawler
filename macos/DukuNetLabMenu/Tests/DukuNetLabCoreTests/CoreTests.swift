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

    func testHostCommandDecodesNumbersAndBooleans() throws {
        let data = Data(#"{"id":"host-0001","action":"start","args":{"channel":36,"enabled":true}}"#.utf8)
        let command = try JSONDecoder().decode(HostCommand.self, from: data)
        XCTAssertEqual(command.id, "host-0001")
        XCTAssertEqual(command.action, "start")
        XCTAssertEqual(command.args["channel"]?.intValue, 36)
        XCTAssertNil(command.args["enabled"]?.intValue)
    }
}
