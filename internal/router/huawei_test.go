package router

import "testing"

func TestHuaweiCollectorRefusesPublicHostsAndReportsUnsupportedFirmware(t *testing.T) {
	if _, err := NewHuaweiCollector(HuaweiConfig{BaseURL: "https://example.com"}); err == nil {
		t.Fatal("expected public or HTTPS host rejection")
	}
	if _, err := NewHuaweiCollector(HuaweiConfig{BaseURL: "http://172.200.1.1"}); err == nil {
		t.Fatal("expected lookalike public address rejection")
	}
	collector, err := NewHuaweiCollector(HuaweiConfig{BaseURL: "http://192.168.100.1"})
	if err != nil {
		t.Fatal(err)
	}
	sample, err := collector.Poll()
	if err == nil || sample.Status != "unsupported" {
		t.Fatalf("sample=%+v err=%v", sample, err)
	}
}

func TestHuaweiCollectorAllowsPrivateHTTPHostsOnly(t *testing.T) {
	for _, url := range []string{"http://localhost", "http://10.0.0.1", "http://172.16.0.1", "http://192.168.1.1:80"} {
		if _, err := NewHuaweiCollector(HuaweiConfig{BaseURL: url}); err != nil {
			t.Fatalf("%s should pass: %v", url, err)
		}
	}
	for _, url := range []string{"", "https://192.168.1.1", "http://example.test", "http://8.8.8.8"} {
		if _, err := NewHuaweiCollector(HuaweiConfig{BaseURL: url}); err == nil {
			t.Fatalf("%s should fail", url)
		}
	}
}
