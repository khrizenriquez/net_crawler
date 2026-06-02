export type Coverage = "wan_total" | "wifi_observed" | "partial" | "not_decryptable" | "out_of_scope";

export interface Status {
  labRunning: boolean;
  automationEnabled: boolean;
  routerStatus: string;
  pcapUsedBytes: number;
  pcapQuotaBytes: number;
  recentFindings: number;
  lastUpdatedAt: string;
  activeSession?: Capture;
}
export interface Capture { id: string; origin: string; status: string; channel: number; band: string; coverage: Coverage; startedAt: string; endedAt?: string; bytesObserved: number; packetsObserved: number; error?: string }
export interface Metric { id: string; timestamp: string; source: Coverage; deviceMac?: string; remoteIp?: string; domain?: string; protocol: string; bytesUp: number; bytesDown: number }
export interface Device { mac: string; alias?: string; lastSeenAt: string; bands: string[]; bytesObserved: number }
export interface Finding { id: string; category: string; severity: string; protocol: string; sessionId: string; deviceMac?: string; remoteIp?: string; redactedSample: string; createdAt: string }
export interface Radio { id: string; ssid: string; bssid: string; band: string; channel: number; confirmed: boolean }
export interface Schedule { id: string; name: string; days: number[]; startMinute: number; durationMinutes: number; channels: number[]; rotationMinutes: number; enabled: boolean }

