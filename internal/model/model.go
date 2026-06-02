package model

import "time"

type Coverage string

const (
	CoverageWAN            Coverage = "wan_total"
	CoverageWiFiObserved   Coverage = "wifi_observed"
	CoveragePartial        Coverage = "partial"
	CoverageNotDecryptable Coverage = "not_decryptable"
	CoverageOutOfScope     Coverage = "out_of_scope"
)

type AuthorizedRadio struct {
	ID        string    `json:"id"`
	SSID      string    `json:"ssid"`
	BSSID     string    `json:"bssid"`
	Band      string    `json:"band"`
	Channel   int       `json:"channel"`
	Confirmed bool      `json:"confirmed"`
	CreatedAt time.Time `json:"createdAt"`
}

type CaptureSchedule struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Days            []int     `json:"days"`
	StartMinute     int       `json:"startMinute"`
	DurationMinutes int       `json:"durationMinutes"`
	Channels        []int     `json:"channels"`
	RotationMinutes int       `json:"rotationMinutes"`
	Enabled         bool      `json:"enabled"`
	CreatedAt       time.Time `json:"createdAt"`
}

type CaptureSession struct {
	ID              string    `json:"id"`
	Origin          string    `json:"origin"`
	Status          string    `json:"status"`
	Channel         int       `json:"channel"`
	Band            string    `json:"band"`
	Coverage        Coverage  `json:"coverage"`
	StartedAt       time.Time `json:"startedAt"`
	EndedAt         time.Time `json:"endedAt,omitempty"`
	BytesObserved   int64     `json:"bytesObserved"`
	PacketsObserved int64     `json:"packetsObserved"`
	Error           string    `json:"error,omitempty"`
}

type MetricBucket struct {
	ID          string    `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	Source      Coverage  `json:"source"`
	DeviceMAC   string    `json:"deviceMac,omitempty"`
	RemoteIP    string    `json:"remoteIp,omitempty"`
	Domain      string    `json:"domain,omitempty"`
	Protocol    string    `json:"protocol"`
	BytesUp     int64     `json:"bytesUp"`
	BytesDown   int64     `json:"bytesDown"`
	PacketsUp   int64     `json:"packetsUp"`
	PacketsDown int64     `json:"packetsDown"`
}

type Device struct {
	MAC           string    `json:"mac"`
	Alias         string    `json:"alias,omitempty"`
	LastSeenAt    time.Time `json:"lastSeenAt"`
	Bands         []string  `json:"bands"`
	BytesObserved int64     `json:"bytesObserved"`
}

type Finding struct {
	ID             string    `json:"id"`
	Category       string    `json:"category"`
	Severity       string    `json:"severity"`
	Protocol       string    `json:"protocol"`
	SessionID      string    `json:"sessionId"`
	DeviceMAC      string    `json:"deviceMac,omitempty"`
	RemoteIP       string    `json:"remoteIp,omitempty"`
	RedactedSample string    `json:"redactedSample"`
	ContentHash    string    `json:"contentHash,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}

type RouterSample struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	BytesUp   int64     `json:"bytesUp"`
	BytesDown int64     `json:"bytesDown"`
	DeltaUp   int64     `json:"deltaUp"`
	DeltaDown int64     `json:"deltaDown"`
	Message   string    `json:"message,omitempty"`
}

type HostCommand struct {
	ID        string         `json:"id"`
	Action    string         `json:"action"`
	Args      map[string]any `json:"args"`
	Status    string         `json:"status"`
	CreatedAt time.Time      `json:"createdAt"`
	Result    string         `json:"result,omitempty"`
}

type Status struct {
	LabRunning        bool            `json:"labRunning"`
	AutomationEnabled bool            `json:"automationEnabled"`
	ActiveSession     *CaptureSession `json:"activeSession,omitempty"`
	RouterStatus      string          `json:"routerStatus"`
	PCAPUsedBytes     int64           `json:"pcapUsedBytes"`
	PCAPQuotaBytes    int64           `json:"pcapQuotaBytes"`
	RecentFindings    int             `json:"recentFindings"`
	LastUpdatedAt     time.Time       `json:"lastUpdatedAt"`
}
