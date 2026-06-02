package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/duku/net-lab/internal/analyzer"
)

func TestEnvAndSplit(t *testing.T) {
	t.Setenv("DUKU_TEST_VALUE", "")
	if got := env("DUKU_TEST_VALUE", "fallback"); got != "fallback" {
		t.Fatalf("got %q", got)
	}
	t.Setenv("DUKU_TEST_VALUE", "configured")
	if got := env("DUKU_TEST_VALUE", "fallback"); got != "configured" {
		t.Fatalf("got %q", got)
	}
	got := split(" a, ,b ")
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("split=%v", got)
	}
	if got := split(" "); got != nil {
		t.Fatalf("blank split=%v", got)
	}
}

func TestUploadReport(t *testing.T) {
	for _, test := range []struct {
		name       string
		status     int
		wantError  bool
		checkToken bool
	}{
		{name: "accepted", status: http.StatusAccepted, checkToken: true},
		{name: "rejected", status: http.StatusBadRequest, wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" {
					t.Fatalf("unexpected request: %s %v", r.Method, r.Header)
				}
				if test.checkToken && r.Header.Get("X-Duku-Host-Token") != "token" {
					t.Fatalf("missing token: %v", r.Header)
				}
				w.WriteHeader(test.status)
			}))
			defer server.Close()
			err := uploadReport(config{APIURL: server.URL, HostToken: "token"}, analyzer.Report{})
			if (err != nil) != test.wantError {
				t.Fatalf("err=%v wantError=%v", err, test.wantError)
			}
		})
	}
}

func TestIsCompletedCapture(t *testing.T) {
	for _, test := range []struct {
		name string
		want bool
	}{
		{name: "capture.pcap", want: true},
		{name: "capture.pcap.partial", want: false},
		{name: "capture.partial", want: false},
		{name: "capture.txt", want: false},
	} {
		if got := isCompletedCapture(test.name); got != test.want {
			t.Fatalf("isCompletedCapture(%q)=%t want %t", test.name, got, test.want)
		}
	}
}

func TestCaptureSourceSeparatesRadioAndLocalHost(t *testing.T) {
	for _, test := range []struct {
		name string
		want captureSource
		ok   bool
	}{
		{name: "capture.pcap", want: sourceAuthorizedRadio, ok: true},
		{name: "capture.local.pcap", want: sourceLocalHost, ok: true},
		{name: "capture.local.pcap.partial", ok: false},
		{name: "capture.txt", ok: false},
	} {
		got, ok := classifyCapture(test.name)
		if got != test.want || ok != test.ok {
			t.Fatalf("classifyCapture(%q)=(%q, %t) want (%q, %t)", test.name, got, ok, test.want, test.ok)
		}
	}
}
