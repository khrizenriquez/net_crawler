import { describe, expect, it } from "vitest";
import { coverage, formatBytes, when } from "./format";

describe("formatBytes", () => {
  it("formats compact local storage units", () => {
    expect(formatBytes()).toBe("0.0 KB");
    expect(formatBytes(12_000)).toBe("12.0 KB");
    expect(formatBytes(2_500_000)).toBe("2.5 MB");
    expect(formatBytes(2_500_000_000)).toBe("2.50 GB");
  });
});

describe("coverage", () => {
  it("keeps dashboard coverage states explicit", () => {
    expect(coverage("wan_total")).toBe("WAN TOTAL");
    expect(coverage("wifi_observed")).toBe("WI-FI OBSERVADO");
    expect(coverage("partial")).toBe("PARCIAL");
    expect(coverage("not_decryptable")).toBe("NO DESCIFRABLE");
    expect(coverage("out_of_scope")).toBe("FUERA DE ALCANCE");
    expect(coverage("future_source")).toBe("FUTURE_SOURCE");
  });
});

describe("when", () => {
  it("returns a localized date and time", () => {
    const result = when("2026-06-02T12:30:00-06:00");
    expect(result).toMatch(/0?2\/06\/26/);
    expect(result).toContain("12:30");
  });
});
