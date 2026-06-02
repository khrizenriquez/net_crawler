package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHelperRestrictsInterfacesChannelsAndPaths(t *testing.T) {
	if !isAllowedInterface("en0") || isAllowedInterface("en1") {
		t.Fatal("v1 must restrict capture to en0")
	}
	if !isAllowedChannel(1) || !isAllowedChannel(233) || isAllowedChannel(0) || isAllowedChannel(234) {
		t.Fatal("channel guard is incorrect")
	}
	base := filepath.Join(t.TempDir(), "staging")
	if !inside(base, filepath.Join(base, "capture.pcap")) {
		t.Fatal("expected staging path to pass")
	}
	if inside(base, filepath.Join(base, "..", "escape.pcap")) {
		t.Fatal("path traversal must fail")
	}
}

func TestChannelParser(t *testing.T) {
	channel, err := parseChannel("PHY Mode: 802.11n\nChannel: 2 (2GHz, 20MHz)")
	if err != nil || channel != 2 {
		t.Fatalf("channel=%d err=%v", channel, err)
	}
	if _, err := parseChannel("Wi-Fi is off"); err == nil {
		t.Fatal("expected missing channel error")
	}
}

func TestStateRoundTripUsesPrivateDataDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DUKU_DATA_DIR", dir)
	want := state{PID: os.Getpid(), Interface: "en0", Channel: 36, RawPath: filepath.Join(dir, "staging", "capture.pcap"), StartedAt: time.Now(), EndsAt: time.Now().Add(time.Minute)}
	if err := saveState(want); err != nil {
		t.Fatal(err)
	}
	got, err := readState()
	if err != nil {
		t.Fatal(err)
	}
	if got.PID != want.PID || got.Channel != want.Channel || statePath() != filepath.Join(dir, "helper-state.json") {
		t.Fatalf("state=%+v path=%s", got, statePath())
	}
	info, err := os.Stat(statePath())
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("state mode=%o", info.Mode().Perm())
	}
	if !processRunning(os.Getpid()) {
		t.Fatal("current test process should be visible")
	}
}

func TestValidateRawOutputPathRejectsExistingNestedAndSymlinkedPaths(t *testing.T) {
	root := t.TempDir()
	staging := filepath.Join(root, "staging")
	if err := os.Mkdir(staging, 0o700); err != nil {
		t.Fatal(err)
	}
	valid := filepath.Join(staging, "capture.pcap")
	if err := validateRawOutputPath(staging, valid); err != nil {
		t.Fatalf("new direct path should pass: %v", err)
	}
	if err := os.WriteFile(valid, []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateRawOutputPath(staging, valid); err == nil {
		t.Fatal("existing output path should fail")
	}
	if err := validateRawOutputPath(staging, filepath.Join(staging, "nested", "capture.pcap")); err == nil {
		t.Fatal("nested output path should fail")
	}
	symlinked := filepath.Join(root, "linked-staging")
	if err := os.Symlink(staging, symlinked); err != nil {
		t.Fatal(err)
	}
	if err := validateRawOutputPath(symlinked, filepath.Join(symlinked, "new.pcap")); err == nil {
		t.Fatal("symlinked staging path should fail")
	}
}
