package store

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/duku/net-lab/internal/model"
)

var ErrNotFound = errors.New("not found")

type Memory struct {
	mu                sync.RWMutex
	radios            []model.AuthorizedRadio
	schedules         []model.CaptureSchedule
	sessions          []model.CaptureSession
	metrics           []model.MetricBucket
	devices           []model.Device
	findings          []model.Finding
	routerSamples     []model.RouterSample
	hostCommands      []model.HostCommand
	automationEnabled bool
	lastScheduleTick  string
	seq               int
}

func NewMemory() *Memory { return &Memory{} }

type Snapshot struct {
	Radios            []model.AuthorizedRadio `json:"radios"`
	Schedules         []model.CaptureSchedule `json:"schedules"`
	Sessions          []model.CaptureSession  `json:"sessions"`
	Metrics           []model.MetricBucket    `json:"metrics"`
	Devices           []model.Device          `json:"devices"`
	Findings          []model.Finding         `json:"findings"`
	RouterSamples     []model.RouterSample    `json:"routerSamples"`
	HostCommands      []model.HostCommand     `json:"hostCommands"`
	AutomationEnabled bool                    `json:"automationEnabled"`
	LastScheduleTick  string                  `json:"lastScheduleTick"`
	Sequence          int                     `json:"sequence"`
}

func (m *Memory) Snapshot() Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return Snapshot{
		Radios: append([]model.AuthorizedRadio(nil), m.radios...), Schedules: append([]model.CaptureSchedule(nil), m.schedules...),
		Sessions: append([]model.CaptureSession(nil), m.sessions...), Metrics: append([]model.MetricBucket(nil), m.metrics...),
		Devices: append([]model.Device(nil), m.devices...), Findings: append([]model.Finding(nil), m.findings...),
		RouterSamples: append([]model.RouterSample(nil), m.routerSamples...), HostCommands: append([]model.HostCommand(nil), m.hostCommands...),
		AutomationEnabled: m.automationEnabled, LastScheduleTick: m.lastScheduleTick, Sequence: m.seq,
	}
}

func (m *Memory) Restore(snapshot Snapshot) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.radios, m.schedules, m.sessions, m.metrics = snapshot.Radios, snapshot.Schedules, snapshot.Sessions, snapshot.Metrics
	m.devices, m.findings, m.routerSamples, m.hostCommands = snapshot.Devices, snapshot.Findings, snapshot.RouterSamples, snapshot.HostCommands
	m.automationEnabled, m.lastScheduleTick, m.seq = snapshot.AutomationEnabled, snapshot.LastScheduleTick, snapshot.Sequence
}

func (m *Memory) id(prefix string) string {
	m.seq++
	return fmt.Sprintf("%s-%04d", prefix, m.seq)
}

func (m *Memory) SeedDemo() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.radios, m.schedules, m.sessions, m.metrics, m.devices, m.findings, m.routerSamples, m.hostCommands = nil, nil, nil, nil, nil, nil, nil, nil
	now := time.Now().Truncate(time.Minute)
	m.radios = []model.AuthorizedRadio{
		{ID: m.id("radio"), SSID: "DUKU-LAB-24", BSSID: "02:00:00:00:02:24", Band: "2.4 GHz", Channel: 2, Confirmed: true, CreatedAt: now},
		{ID: m.id("radio"), SSID: "DUKU-LAB-5", BSSID: "02:00:00:00:05:36", Band: "5 GHz", Channel: 36, Confirmed: true, CreatedAt: now},
	}
	m.schedules = []model.CaptureSchedule{{ID: m.id("schedule"), Name: "Ventana nocturna", Days: []int{1, 3, 5}, StartMinute: 20 * 60, DurationMinutes: 90, Channels: []int{2, 36}, RotationMinutes: 5, Enabled: true, CreatedAt: now}}
	session := model.CaptureSession{ID: m.id("capture"), Origin: "demo", Status: "completed", Channel: 2, Band: "2.4 GHz", Coverage: model.CoverageWiFiObserved, StartedAt: now.Add(-55 * time.Minute), EndedAt: now.Add(-25 * time.Minute), BytesObserved: 842_770_120, PacketsObserved: 284_302}
	m.sessions = []model.CaptureSession{session}
	m.devices = []model.Device{
		{MAC: "02:00:00:00:10:01", Alias: "TV sala", LastSeenAt: now.Add(-3 * time.Minute), Bands: []string{"5 GHz"}, BytesObserved: 522_110_450},
		{MAC: "02:00:00:00:10:02", Alias: "Tablet lab", LastSeenAt: now.Add(-7 * time.Minute), Bands: []string{"2.4 GHz"}, BytesObserved: 202_009_670},
		{MAC: "02:00:00:00:10:03", Alias: "Sensor pruebas", LastSeenAt: now.Add(-12 * time.Minute), Bands: []string{"2.4 GHz"}, BytesObserved: 118_650_000},
	}
	m.metrics = []model.MetricBucket{
		{ID: m.id("metric"), Timestamp: now.Add(-50 * time.Minute), Source: model.CoverageWiFiObserved, DeviceMAC: m.devices[0].MAC, RemoteIP: "198.51.100.14", Domain: "video.example.test", Protocol: "TLS", BytesUp: 8_440_120, BytesDown: 500_700_000, PacketsUp: 4_200, PacketsDown: 120_000},
		{ID: m.id("metric"), Timestamp: now.Add(-45 * time.Minute), Source: model.CoverageWiFiObserved, DeviceMAC: m.devices[1].MAC, RemoteIP: "192.0.2.24", Domain: "lab.example.test", Protocol: "HTTP", BytesUp: 800_000, BytesDown: 12_500_000, PacketsUp: 500, PacketsDown: 4_000},
		{ID: m.id("metric"), Timestamp: now.Add(-15 * time.Minute), Source: model.CoverageWAN, Protocol: "WAN", BytesUp: 132_300_000, BytesDown: 1_850_000_000, PacketsUp: 0, PacketsDown: 0},
	}
	m.findings = []model.Finding{
		{ID: m.id("finding"), Category: "possible_credential", Severity: "high", Protocol: "HTTP", SessionID: session.ID, DeviceMAC: m.devices[1].MAC, RemoteIP: "192.0.2.24", RedactedSample: "username=t***@example.test&password=[REDACTED]", CreatedAt: now.Add(-44 * time.Minute)},
		{ID: m.id("finding"), Category: "base64_media", Severity: "info", Protocol: "HTTP", SessionID: session.ID, DeviceMAC: m.devices[1].MAC, RemoteIP: "192.0.2.24", RedactedSample: "image/png; base64 payload omitted", ContentHash: "sha256:demo-image-fixture", CreatedAt: now.Add(-43 * time.Minute)},
	}
	m.routerSamples = []model.RouterSample{{ID: m.id("router"), Status: "unsupported", Timestamp: now, Message: "Demo: firmware counters not configured"}}
	m.automationEnabled = true
}

func (m *Memory) Status() model.Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var active *model.CaptureSession
	for i := range m.sessions {
		if m.sessions[i].Status == "running" {
			copy := m.sessions[i]
			active = &copy
		}
	}
	router := "unsupported"
	if len(m.routerSamples) > 0 {
		router = m.routerSamples[len(m.routerSamples)-1].Status
	}
	return model.Status{LabRunning: true, AutomationEnabled: m.automationEnabled, ActiveSession: active, RouterStatus: router, PCAPUsedBytes: 0, PCAPQuotaBytes: 5 * 1024 * 1024 * 1024, RecentFindings: len(m.findings), LastUpdatedAt: time.Now()}
}

func (m *Memory) Sessions() []model.CaptureSession {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]model.CaptureSession(nil), m.sessions...)
}
func (m *Memory) Radios() []model.AuthorizedRadio {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]model.AuthorizedRadio(nil), m.radios...)
}
func (m *Memory) Schedules() []model.CaptureSchedule {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]model.CaptureSchedule(nil), m.schedules...)
}
func (m *Memory) Metrics() []model.MetricBucket {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]model.MetricBucket(nil), m.metrics...)
}
func (m *Memory) Devices() []model.Device {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]model.Device(nil), m.devices...)
}
func (m *Memory) Findings() []model.Finding {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]model.Finding(nil), m.findings...)
}
func (m *Memory) RouterSamples() []model.RouterSample {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]model.RouterSample(nil), m.routerSamples...)
}

func (m *Memory) AddRadio(r model.AuthorizedRadio) model.AuthorizedRadio {
	m.mu.Lock()
	defer m.mu.Unlock()
	r.ID, r.CreatedAt = m.id("radio"), time.Now()
	m.radios = append(m.radios, r)
	return r
}

func windowsOverlap(a, b model.CaptureSchedule) bool {
	for _, da := range a.Days {
		for _, db := range b.Days {
			if da == db && a.StartMinute < b.StartMinute+b.DurationMinutes && b.StartMinute < a.StartMinute+a.DurationMinutes {
				return true
			}
		}
	}
	return false
}

func ValidateSchedule(s model.CaptureSchedule, existing []model.CaptureSchedule) error {
	if s.DurationMinutes < 1 || s.DurationMinutes > 120 {
		return errors.New("duration must be between 1 and 120 minutes")
	}
	if s.RotationMinutes < 1 {
		return errors.New("rotation must be positive")
	}
	if len(s.Days) == 0 || len(s.Channels) == 0 {
		return errors.New("days and channels are required")
	}
	for _, current := range existing {
		if windowsOverlap(s, current) {
			return errors.New("schedule overlaps an existing window")
		}
	}
	return nil
}

func (m *Memory) AddSchedule(s model.CaptureSchedule) (model.CaptureSchedule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := ValidateSchedule(s, m.schedules); err != nil {
		return model.CaptureSchedule{}, err
	}
	s.ID, s.CreatedAt = m.id("schedule"), time.Now()
	m.schedules = append(m.schedules, s)
	return s, nil
}

func (m *Memory) StartCapture(channel int) model.HostCommand {
	m.mu.Lock()
	defer m.mu.Unlock()
	cmd := m.startCapture(channel, 120, []int{channel}, 5)
	m.hostCommands = append(m.hostCommands, cmd)
	return cmd
}

func (m *Memory) startCapture(channel, duration int, channels []int, rotation int) model.HostCommand {
	return model.HostCommand{ID: m.id("host"), Action: "start", Args: map[string]any{"channel": channel, "channels": channels, "durationMinutes": duration, "rotationMinutes": rotation}, Status: "pending", CreatedAt: time.Now()}
}

func (m *Memory) TickSchedules(now time.Time) []model.HostCommand {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := now.Format("2006-01-02T15:04")
	if !m.automationEnabled || m.lastScheduleTick == key {
		return nil
	}
	m.lastScheduleTick = key
	for _, session := range m.sessions {
		if session.Status == "running" {
			return nil
		}
	}
	minute, weekday := now.Hour()*60+now.Minute(), int(now.Weekday())
	for _, schedule := range m.schedules {
		if !schedule.Enabled || schedule.StartMinute != minute || !contains(schedule.Days, weekday) || len(schedule.Channels) == 0 {
			continue
		}
		cmd := m.startCapture(schedule.Channels[0], schedule.DurationMinutes, schedule.Channels, schedule.RotationMinutes)
		m.hostCommands = append(m.hostCommands, cmd)
		return []model.HostCommand{cmd}
	}
	return nil
}

func contains(values []int, wanted int) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
func argInt(value any) int {
	switch number := value.(type) {
	case int:
		return number
	case float64:
		return int(number)
	default:
		return 0
	}
}

func (m *Memory) StopCapture() model.HostCommand {
	m.mu.Lock()
	defer m.mu.Unlock()
	cmd := model.HostCommand{ID: m.id("host"), Action: "stop", Args: map[string]any{}, Status: "pending", CreatedAt: time.Now()}
	m.hostCommands = append(m.hostCommands, cmd)
	return cmd
}

func (m *Memory) NextHostCommand() (model.HostCommand, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, cmd := range m.hostCommands {
		if cmd.Status == "pending" {
			return cmd, nil
		}
	}
	return model.HostCommand{}, ErrNotFound
}

func (m *Memory) CompleteHostCommand(id, result string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.hostCommands {
		if m.hostCommands[i].ID == id {
			m.hostCommands[i].Status, m.hostCommands[i].Result = "completed", result
			if strings.HasPrefix(result, "exit=0") {
				switch m.hostCommands[i].Action {
				case "start":
					channel := argInt(m.hostCommands[i].Args["channel"])
					m.sessions = append(m.sessions, model.CaptureSession{ID: m.id("capture"), Origin: "manual", Status: "running", Channel: channel, Coverage: model.CoverageWiFiObserved, StartedAt: time.Now()})
				case "stop":
					for j := range m.sessions {
						if m.sessions[j].Status == "running" {
							m.sessions[j].Status, m.sessions[j].EndedAt = "completed", time.Now()
						}
					}
				}
			}
			return nil
		}
	}
	return ErrNotFound
}

func (m *Memory) ResetDemo() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.radios, m.schedules, m.sessions, m.metrics, m.devices, m.findings, m.routerSamples, m.hostCommands = nil, nil, nil, nil, nil, nil, nil, nil
}

func (m *Memory) Ingest(metrics []model.MetricBucket, findings []model.Finding) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range metrics {
		if metrics[i].ID == "" {
			metrics[i].ID = m.id("metric")
		}
	}
	for i := range findings {
		if findings[i].ID == "" {
			findings[i].ID = m.id("finding")
		}
		if findings[i].CreatedAt.IsZero() {
			findings[i].CreatedAt = time.Now()
		}
	}
	m.metrics, m.findings = append(m.metrics, metrics...), append(m.findings, findings...)
}

func SortedMetrics(in []model.MetricBucket) []model.MetricBucket {
	out := append([]model.MetricBucket(nil), in...)
	sort.Slice(out, func(i, j int) bool { return out[i].Timestamp.Before(out[j].Timestamp) })
	return out
}
