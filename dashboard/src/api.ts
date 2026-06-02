import type { Capture, Device, Finding, Metric, Radio, Schedule, Status } from "./types";

const API = "http://127.0.0.1:8080/api/v1";
const get = async <T,>(path: string): Promise<T> => {
  const response = await fetch(`${API}${path}`);
  if (!response.ok) throw new Error(`API ${response.status}`);
  return response.json() as Promise<T>;
};
const post = async <T,>(path: string, body?: unknown): Promise<T> => {
  const response = await fetch(`${API}${path}`, { method: "POST", headers: { "Content-Type": "application/json" }, body: body === undefined ? undefined : JSON.stringify(body) });
  if (!response.ok) throw new Error(`API ${response.status}`);
  return response.json() as Promise<T>;
};
export const api = {
  load: async () => {
    const [status, captures, metrics, devices, findings, radios, schedules] = await Promise.all([
      get<Status>("/status"), get<Capture[]>("/captures"), get<Metric[]>("/metrics"),
      get<Device[]>("/devices"), get<Finding[]>("/findings"), get<Radio[]>("/radios"), get<Schedule[]>("/schedules"),
    ]);
    return { status, captures, metrics, devices, findings, radios, schedules };
  },
  start: (channel: number) => post("/captures/start", { channel }),
  startLocal: () => post("/captures/start-local"),
  stop: () => post("/captures/stop"),
  seedDemo: () => post("/demo/seed"),
  resetDemo: () => post("/demo/reset"),
  exportURL: (format: "csv" | "json") => `${API}/exports?format=${format}`,
  events: () => new EventSource(`${API}/events`),
};
