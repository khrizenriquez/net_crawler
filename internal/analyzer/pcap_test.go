package analyzer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildBSSIDFilterRejectsInvalidValue(t *testing.T) {
	if _, err := BuildBSSIDFilter([]string{"not-a-bssid"}); err == nil {
		t.Fatal("expected invalid BSSID error")
	}
	got, err := BuildBSSIDFilter([]string{"02:00:00:00:02:24"})
	if err != nil || !strings.Contains(got, "02:00:00:00:02:24") {
		t.Fatalf("unexpected filter: %q, %v", got, err)
	}
}

func TestBuildBSSIDFilterRequiresAllowlistAndCombinesValues(t *testing.T) {
	if _, err := BuildBSSIDFilter(nil); err == nil {
		t.Fatal("expected empty allowlist error")
	}
	got, err := BuildBSSIDFilter([]string{" AA:BB:CC:DD:EE:FF ", "11:22:33:44:55:66"})
	if err != nil {
		t.Fatal(err)
	}
	for _, wanted := range []string{"wlan.bssid == aa:bb:cc:dd:ee:ff", "wlan.bssid == 11:22:33:44:55:66", " || "} {
		if !strings.Contains(got, wanted) {
			t.Fatalf("filter %q does not contain %q", got, wanted)
		}
	}
}

func TestFilterAuthorizedPCAPDeletesRawOnFailure(t *testing.T) {
	dir := t.TempDir()
	raw := filepath.Join(dir, "raw.pcap")
	if err := os.WriteFile(raw, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := FilterAuthorizedPCAP(FilterOptions{TSharkPath: "/does/not/exist", RawPath: raw, AuthorizedPath: filepath.Join(dir, "filtered.pcap"), AuthorizedBSSID: []string{"02:00:00:00:02:24"}})
	if err == nil {
		t.Fatal("expected tshark error")
	}
	if _, statErr := os.Stat(raw); !os.IsNotExist(statErr) {
		t.Fatal("raw capture should always be removed")
	}
}

func TestFilterAuthorizedPCAPDeletesRawAfterSuccess(t *testing.T) {
	dir := t.TempDir()
	raw := filepath.Join(dir, "raw.pcap")
	out := filepath.Join(dir, "filtered.pcap")
	fakeTshark := filepath.Join(dir, "tshark")
	if err := os.WriteFile(raw, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fakeTshark, []byte("#!/bin/sh\ncp \"$2\" \"$6\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	err := FilterAuthorizedPCAP(FilterOptions{TSharkPath: fakeTshark, RawPath: raw, AuthorizedPath: out, AuthorizedBSSID: []string{"02:00:00:00:02:24"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(raw); !os.IsNotExist(err) {
		t.Fatalf("raw capture should always be removed: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "fixture" {
		t.Fatalf("unexpected filtered content %q", data)
	}
}

func TestFilterAuthorizedPCAPRejectsIdenticalPathsBeforeDeletion(t *testing.T) {
	dir := t.TempDir()
	raw := filepath.Join(dir, "raw.pcap")
	if err := os.WriteFile(raw, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := FilterAuthorizedPCAP(FilterOptions{TSharkPath: "tshark", RawPath: raw, AuthorizedPath: raw, AuthorizedBSSID: []string{"02:00:00:00:02:24"}})
	if err == nil {
		t.Fatal("expected identical path error")
	}
	if _, err := os.Stat(raw); err != nil {
		t.Fatalf("validation must not delete the input: %v", err)
	}
}
