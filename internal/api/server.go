package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/duku/net-lab/internal/model"
	"github.com/duku/net-lab/internal/store"
)

type Server struct {
	store     store.Repository
	hostToken string
	mu        sync.Mutex
	clients   map[chan string]struct{}
}

func New(repository store.Repository, hostToken string) *Server {
	return &Server{store: repository, hostToken: hostToken, clients: map[chan string]struct{}{}}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/status", s.getStatus)
	mux.HandleFunc("GET /api/v1/events", s.events)
	mux.HandleFunc("GET /api/v1/captures", s.listCaptures)
	mux.HandleFunc("POST /api/v1/captures/start", s.startCapture)
	mux.HandleFunc("POST /api/v1/captures/stop", s.stopCapture)
	mux.HandleFunc("GET /api/v1/radios", s.listRadios)
	mux.HandleFunc("POST /api/v1/radios", s.addRadio)
	mux.HandleFunc("POST /api/v1/radios/scan", s.scanRadios)
	mux.HandleFunc("GET /api/v1/schedules", s.listSchedules)
	mux.HandleFunc("POST /api/v1/schedules", s.addSchedule)
	mux.HandleFunc("GET /api/v1/metrics", s.listMetrics)
	mux.HandleFunc("GET /api/v1/devices", s.listDevices)
	mux.HandleFunc("GET /api/v1/findings", s.listFindings)
	mux.HandleFunc("GET /api/v1/router/status", s.routerStatus)
	mux.HandleFunc("POST /api/v1/router/test", s.routerTest)
	mux.HandleFunc("GET /api/v1/exports", s.export)
	mux.HandleFunc("POST /api/v1/exports", s.export)
	mux.HandleFunc("POST /api/v1/demo/seed", s.seedDemo)
	mux.HandleFunc("POST /api/v1/demo/reset", s.resetDemo)
	mux.HandleFunc("POST /api/v1/ingest", s.requireHost(s.ingest))
	mux.HandleFunc("GET /api/v1/host/commands/next", s.requireHost(s.nextHostCommand))
	mux.HandleFunc("POST /api/v1/host/commands/{id}/result", s.requireHost(s.completeHostCommand))
	return cors(mux)
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("Origin") {
		case "http://127.0.0.1:4173", "http://127.0.0.1:5173":
			w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		default:
			w.Header().Set("Access-Control-Allow-Origin", "http://127.0.0.1:4173")
		}
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Duku-Host-Token")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return false
	}
	return true
}
func (s *Server) publish(event string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for client := range s.clients {
		select {
		case client <- event:
		default:
		}
	}
}
func (s *Server) getStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Status())
}
func (s *Server) listCaptures(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Sessions())
}
func (s *Server) listRadios(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Radios())
}
func (s *Server) listSchedules(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Schedules())
}
func (s *Server) listMetrics(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, store.SortedMetrics(s.store.Metrics()))
}
func (s *Server) listDevices(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Devices())
}
func (s *Server) listFindings(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Findings())
}
func (s *Server) routerStatus(w http.ResponseWriter, _ *http.Request) {
	samples := s.store.RouterSamples()
	if len(samples) == 0 {
		writeJSON(w, http.StatusOK, model.RouterSample{Status: "unsupported", Message: "Huawei adapter has not been configured"})
		return
	}
	writeJSON(w, http.StatusOK, samples[len(samples)-1])
}
func (s *Server) routerTest(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "unsupported", "message": "Firmware-specific WAN counter discovery is optional in v1"})
}
func (s *Server) scanRadios(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, []model.AuthorizedRadio{})
}
func (s *Server) addRadio(w http.ResponseWriter, r *http.Request) {
	var radio model.AuthorizedRadio
	if !decodeJSON(w, r, &radio) {
		return
	}
	writeJSON(w, http.StatusCreated, s.store.AddRadio(radio))
	s.publish("radio.updated")
}
func (s *Server) addSchedule(w http.ResponseWriter, r *http.Request) {
	var schedule model.CaptureSchedule
	if !decodeJSON(w, r, &schedule) {
		return
	}
	created, err := s.store.AddSchedule(schedule)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, created)
	s.publish("schedule.updated")
}
func (s *Server) startCapture(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Channel int `json:"channel"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	cmd := s.store.StartCapture(body.Channel)
	writeJSON(w, http.StatusAccepted, cmd)
	s.publish("capture.requested")
}
func (s *Server) stopCapture(w http.ResponseWriter, _ *http.Request) {
	cmd := s.store.StopCapture()
	writeJSON(w, http.StatusAccepted, cmd)
	s.publish("capture.stop_requested")
}
func (s *Server) seedDemo(w http.ResponseWriter, _ *http.Request) {
	s.store.SeedDemo()
	writeJSON(w, http.StatusOK, map[string]string{"status": "seeded"})
	s.publish("demo.seeded")
}
func (s *Server) resetDemo(w http.ResponseWriter, _ *http.Request) {
	s.store.ResetDemo()
	writeJSON(w, http.StatusOK, map[string]string{"status": "reset"})
	s.publish("demo.reset")
}

func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	client := make(chan string, 8)
	s.mu.Lock()
	s.clients[client] = struct{}{}
	s.mu.Unlock()
	defer func() { s.mu.Lock(); delete(s.clients, client); s.mu.Unlock() }()
	fmt.Fprint(w, "data: connected\n\n")
	flusher.Flush()
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case event := <-client:
			fmt.Fprintf(w, "data: %s\n\n", event)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprint(w, "data: heartbeat\n\n")
			flusher.Flush()
		}
	}
}

func (s *Server) requireHost(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.hostToken == "" || r.Header.Get("X-Duku-Host-Token") != s.hostToken {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid host token"})
			return
		}
		next(w, r)
	}
}
func (s *Server) nextHostCommand(w http.ResponseWriter, _ *http.Request) {
	cmd, err := s.store.NextHostCommand()
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	writeJSON(w, http.StatusOK, cmd)
}
func (s *Server) completeHostCommand(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Result string `json:"result"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if err := s.store.CompleteHostCommand(r.PathValue("id"), body.Result); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "completed"})
	s.publish("host.command_completed")
}

func (s *Server) ingest(w http.ResponseWriter, r *http.Request) {
	var report struct {
		Metrics  []model.MetricBucket `json:"metrics"`
		Findings []model.Finding      `json:"findings"`
	}
	if !decodeJSON(w, r, &report) {
		return
	}
	s.store.Ingest(report.Metrics, report.Findings)
	writeJSON(w, http.StatusAccepted, map[string]int{"metrics": len(report.Metrics), "findings": len(report.Findings)})
	s.publish("ingest.completed")
}

func (s *Server) export(w http.ResponseWriter, r *http.Request) {
	format := strings.ToLower(r.URL.Query().Get("format"))
	if format == "" {
		format = "json"
	}
	findings := s.store.Findings()
	if format == "csv" {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", `attachment; filename="duku-findings.csv"`)
		out := csv.NewWriter(w)
		_ = out.Write([]string{"id", "category", "severity", "protocol", "session_id", "device_mac", "remote_ip", "redacted_sample", "created_at"})
		for _, f := range findings {
			_ = out.Write([]string{f.ID, f.Category, f.Severity, f.Protocol, f.SessionID, f.DeviceMAC, f.RemoteIP, f.RedactedSample, f.CreatedAt.Format(time.RFC3339)})
		}
		out.Flush()
		return
	}
	if format != "json" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "format must be csv or json"})
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="duku-findings.json"`)
	writeJSON(w, http.StatusOK, findings)
}

func ParsePort(raw string) (int, error) {
	port, err := strconv.Atoi(raw)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("invalid port %q", raw)
	}
	return port, nil
}
