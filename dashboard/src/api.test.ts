import { afterEach, describe, expect, it, vi } from "vitest";
import { api } from "./api";

const response = (body: unknown, ok = true, status = 200) =>
  ({ ok, status, json: async () => body }) as Response;

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("api", () => {
  it("loads all dashboard resources from loopback", async () => {
    const fetchMock = vi.fn(async (_input: RequestInfo | URL) => response([]));
    vi.stubGlobal("fetch", fetchMock);
    await api.load();
    expect(fetchMock).toHaveBeenCalledTimes(7);
    for (const [url] of fetchMock.mock.calls) {
      expect(String(url)).toMatch(/^http:\/\/127\.0\.0\.1:8080\/api\/v1\//);
    }
  });

  it("starts captures with JSON and exposes local exports", async () => {
    const fetchMock = vi.fn(async () => response({}));
    vi.stubGlobal("fetch", fetchMock);
    await api.start(36);
    expect(fetchMock).toHaveBeenCalledWith(
      "http://127.0.0.1:8080/api/v1/captures/start",
      expect.objectContaining({ method: "POST", body: '{"channel":36}' }),
    );
    await api.startLocal();
    expect(fetchMock).toHaveBeenCalledWith(
      "http://127.0.0.1:8080/api/v1/captures/start-local",
      expect.objectContaining({ method: "POST" }),
    );
    expect(api.exportURL("csv")).toBe("http://127.0.0.1:8080/api/v1/exports?format=csv");
  });

  it("rejects non-success API responses", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => response({}, false, 503)));
    await expect(api.stop()).rejects.toThrow("API 503");
  });

  it("creates a loopback SSE client", () => {
    const eventSource = vi.fn();
    vi.stubGlobal("EventSource", eventSource);
    api.events();
    expect(eventSource).toHaveBeenCalledWith("http://127.0.0.1:8080/api/v1/events");
  });
});
