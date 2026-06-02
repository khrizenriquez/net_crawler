package api

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/duku/net-lab/internal/store"
)

func TestDashboardEndpointsAndHostTokenBoundary(t *testing.T) {
	memory := store.NewMemory()
	memory.SeedDemo()
	server := httptest.NewServer(New(memory, "local-secret").Handler())
	defer server.Close()
	resp, err := http.Get(server.URL + "/api/v1/status")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	req, _ := http.NewRequest(http.MethodGet, server.URL+"/api/v1/host/commands/next", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("host command endpoint must require token, got %d", resp.StatusCode)
	}
}

func TestScheduleOverlapReturnsConflict(t *testing.T) {
	memory := store.NewMemory()
	memory.SeedDemo()
	server := httptest.NewServer(New(memory, "local-secret").Handler())
	defer server.Close()
	body := `{"name":"overlap","days":[1],"startMinute":1230,"durationMinutes":30,"channels":[2],"rotationMinutes":5,"enabled":true}`
	resp, err := http.Post(server.URL+"/api/v1/schedules", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestConfigurationCaptureDemoAndReadEndpoints(t *testing.T) {
	memory := store.NewMemory()
	server := httptest.NewServer(New(memory, "local-secret").Handler())
	defer server.Close()
	for _, path := range []string{"/status", "/captures", "/radios", "/radios/scan", "/schedules", "/metrics", "/devices", "/findings", "/router/status", "/router/test"} {
		method := http.MethodGet
		if path == "/radios/scan" || path == "/router/test" {
			method = http.MethodPost
		}
		resp := request(t, method, server.URL+"/api/v1"+path, "", "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s %s returned %d", method, path, resp.StatusCode)
		}
		resp.Body.Close()
	}
	resp := request(t, http.MethodPost, server.URL+"/api/v1/radios", `{"ssid":"lab","bssid":"aa:bb:cc:dd:ee:ff","channel":2}`, "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("radio status=%d", resp.StatusCode)
	}
	resp.Body.Close()
	resp = request(t, http.MethodPost, server.URL+"/api/v1/schedules", `{"name":"night","days":[1],"startMinute":60,"durationMinutes":30,"channels":[2],"rotationMinutes":5}`, "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("schedule status=%d", resp.StatusCode)
	}
	resp.Body.Close()
	resp = request(t, http.MethodPost, server.URL+"/api/v1/captures/start", `{"channel":2}`, "")
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("start status=%d", resp.StatusCode)
	}
	resp.Body.Close()
	resp = request(t, http.MethodPost, server.URL+"/api/v1/captures/stop", "", "")
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("stop status=%d", resp.StatusCode)
	}
	resp.Body.Close()
	resp = request(t, http.MethodPost, server.URL+"/api/v1/demo/seed", "", "")
	if resp.StatusCode != http.StatusOK || len(memory.Metrics()) == 0 {
		t.Fatalf("seed status=%d metrics=%d", resp.StatusCode, len(memory.Metrics()))
	}
	resp.Body.Close()
	resp = request(t, http.MethodPost, server.URL+"/api/v1/demo/reset", "", "")
	if resp.StatusCode != http.StatusOK || len(memory.Metrics()) != 0 {
		t.Fatalf("reset status=%d metrics=%d", resp.StatusCode, len(memory.Metrics()))
	}
	resp.Body.Close()
}

func TestMalformedJSONAndHostCommandLifecycle(t *testing.T) {
	memory := store.NewMemory()
	server := httptest.NewServer(New(memory, "local-secret").Handler())
	defer server.Close()
	resp := request(t, http.MethodPost, server.URL+"/api/v1/radios", "{", "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad JSON status=%d", resp.StatusCode)
	}
	resp.Body.Close()
	resp = request(t, http.MethodGet, server.URL+"/api/v1/host/commands/next", "", "local-secret")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("empty queue status=%d", resp.StatusCode)
	}
	resp.Body.Close()
	cmd := memory.StartCapture(36)
	resp = request(t, http.MethodGet, server.URL+"/api/v1/host/commands/next", "", "local-secret")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("next status=%d", resp.StatusCode)
	}
	resp.Body.Close()
	resp = request(t, http.MethodPost, server.URL+"/api/v1/host/commands/"+cmd.ID+"/result", `{"result":"exit=0 capture started"}`, "local-secret")
	if resp.StatusCode != http.StatusOK || memory.Status().ActiveSession == nil {
		t.Fatalf("completion status=%d active=%+v", resp.StatusCode, memory.Status().ActiveSession)
	}
	resp.Body.Close()
	resp = request(t, http.MethodPost, server.URL+"/api/v1/host/commands/missing/result", `{"result":"exit=0"}`, "local-secret")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("missing completion status=%d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestIngestExportsAndCORS(t *testing.T) {
	memory := store.NewMemory()
	server := httptest.NewServer(New(memory, "local-secret").Handler())
	defer server.Close()
	report := `{"metrics":[{"protocol":"HTTP","bytesUp":12}],"findings":[{"category":"possible_credential","redactedSample":"password=[REDACTED]"}]}`
	resp := request(t, http.MethodPost, server.URL+"/api/v1/ingest", report, "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthorized ingest status=%d", resp.StatusCode)
	}
	resp.Body.Close()
	resp = request(t, http.MethodPost, server.URL+"/api/v1/ingest", report, "local-secret")
	if resp.StatusCode != http.StatusAccepted || len(memory.Metrics()) != 1 || len(memory.Findings()) != 1 {
		t.Fatalf("ingest status=%d metrics=%d findings=%d", resp.StatusCode, len(memory.Metrics()), len(memory.Findings()))
	}
	resp.Body.Close()
	for _, format := range []string{"json", "csv"} {
		resp = request(t, http.MethodGet, server.URL+"/api/v1/exports?format="+format, "", "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s export status=%d", format, resp.StatusCode)
		}
		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if !strings.Contains(string(data), "possible_credential") {
			t.Fatalf("%s export missing fixture: %q", format, data)
		}
	}
	resp = request(t, http.MethodPost, server.URL+"/api/v1/exports?format=xml", "", "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid export status=%d", resp.StatusCode)
	}
	resp.Body.Close()
	req, _ := http.NewRequest(http.MethodOptions, server.URL+"/api/v1/status", nil)
	req.Header.Set("Origin", "http://127.0.0.1:4173")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent || resp.Header.Get("Access-Control-Allow-Origin") != "http://127.0.0.1:4173" {
		t.Fatalf("unexpected CORS response status=%d headers=%v", resp.StatusCode, resp.Header)
	}
}

func TestEventsConnectAndParsePort(t *testing.T) {
	server := httptest.NewServer(New(store.NewMemory(), "local-secret").Handler())
	defer server.Close()
	resp, err := http.Get(server.URL + "/api/v1/events")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	line, err := bufio.NewReader(resp.Body).ReadString('\n')
	if err != nil || line != "data: connected\n" {
		t.Fatalf("event line=%q err=%v", line, err)
	}
	for _, raw := range []string{"", "0", "65536", "abc"} {
		if _, err := ParsePort(raw); err == nil {
			t.Fatalf("expected %q to fail", raw)
		}
	}
	if port, err := ParsePort("8080"); err != nil || port != 8080 {
		t.Fatalf("port=%d err=%v", port, err)
	}
}

func request(t *testing.T, method, url, body, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("X-Duku-Host-Token", token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func decode[T any](t *testing.T, response *http.Response) T {
	t.Helper()
	defer response.Body.Close()
	var out T
	if err := json.NewDecoder(response.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}
