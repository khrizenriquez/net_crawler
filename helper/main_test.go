package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
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

func TestPartialCapturePathAndAtomicPublish(t *testing.T) {
	root := t.TempDir()
	t.Setenv("DUKU_DATA_DIR", root)
	staging := filepath.Join(root, "staging")
	if err := os.Mkdir(staging, 0o700); err != nil {
		t.Fatal(err)
	}
	final := filepath.Join(staging, "capture.pcap")
	partial := partialCapturePath(final)
	if partial != final+".partial" {
		t.Fatalf("partial=%q", partial)
	}
	if err := os.WriteFile(partial, []byte("synthetic pcap"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := publishCapture(partial, final); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(partial); !os.IsNotExist(err) {
		t.Fatalf("partial capture should be removed after publish: %v", err)
	}
	data, err := os.ReadFile(final)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "synthetic pcap" {
		t.Fatalf("published data=%q", data)
	}
	if err := publishCapture(partial, final); err != nil {
		t.Fatalf("publish should be idempotent: %v", err)
	}
}

func TestValidateRawOutputPathRejectsExistingPartialCapture(t *testing.T) {
	root := t.TempDir()
	staging := filepath.Join(root, "staging")
	if err := os.Mkdir(staging, 0o700); err != nil {
		t.Fatal(err)
	}
	final := filepath.Join(staging, "capture.pcap")
	if err := os.WriteFile(partialCapturePath(final), []byte("in progress"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateRawOutputPath(staging, final); err == nil {
		t.Fatal("existing partial output path should fail")
	}
}

func TestPublishCaptureIsSafeWhenStopAndSupervisorRace(t *testing.T) {
	root := t.TempDir()
	t.Setenv("DUKU_DATA_DIR", root)
	staging := filepath.Join(root, "staging")
	if err := os.Mkdir(staging, 0o700); err != nil {
		t.Fatal(err)
	}
	final := filepath.Join(staging, "capture.pcap")
	partial := partialCapturePath(final)
	if err := os.WriteFile(partial, []byte("synthetic pcap"), 0o600); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	errors := make(chan error, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	for range 2 {
		go func() {
			ready.Done()
			<-start
			errors <- publishCapture(partial, final)
		}()
	}
	ready.Wait()
	close(start)
	for range 2 {
		if err := <-errors; err != nil {
			t.Fatal(err)
		}
	}
}

func TestFinishCaptureIsSafeWhenStopAndSupervisorRace(t *testing.T) {
	root := t.TempDir()
	t.Setenv("DUKU_DATA_DIR", root)
	staging := filepath.Join(root, "staging")
	if err := os.Mkdir(staging, 0o700); err != nil {
		t.Fatal(err)
	}
	final := filepath.Join(staging, "capture.pcap")
	current := state{PID: os.Getpid(), RawPath: final, PartialPath: partialCapturePath(final)}
	if err := os.WriteFile(current.PartialPath, []byte("synthetic pcap"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := saveState(current); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	errors := make(chan error, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	for range 2 {
		go func() {
			ready.Done()
			<-start
			errors <- finishCapture(current)
		}()
	}
	ready.Wait()
	close(start)
	for range 2 {
		if err := <-errors; err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(statePath()); !os.IsNotExist(err) {
		t.Fatalf("state should be removed after finish: %v", err)
	}
}

func TestInvokingUserIDs(t *testing.T) {
	t.Setenv("SUDO_UID", strconv.Itoa(os.Getuid()))
	t.Setenv("SUDO_GID", strconv.Itoa(os.Getgid()))
	uid, gid, err := invokingUserIDs()
	if err != nil {
		t.Fatal(err)
	}
	if uid != os.Getuid() || gid != os.Getgid() {
		t.Fatalf("uid=%d gid=%d", uid, gid)
	}
	t.Setenv("SUDO_UID", "invalid")
	if _, _, err := invokingUserIDs(); err == nil {
		t.Fatal("invalid sudo uid should fail")
	}
}

func TestSetPrivateCaptureOwnerKeepsMode600(t *testing.T) {
	path := filepath.Join(t.TempDir(), "capture.pcap.partial")
	if err := os.WriteFile(path, []byte("synthetic pcap"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := setPrivateCaptureOwner(path, os.Getuid(), os.Getgid()); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode=%o", info.Mode().Perm())
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatal("missing unix file metadata")
	}
	if int(stat.Uid) != os.Getuid() || int(stat.Gid) != os.Getgid() {
		t.Fatalf("uid=%d gid=%d", stat.Uid, stat.Gid)
	}
}

func TestCaptureStartedMessageUsesSavedPID(t *testing.T) {
	if got := captureStartedMessage(1234, 149); got != "capture started pid=1234 channel=149" {
		t.Fatalf("message=%q", got)
	}
}

func TestValidateCaptureOutputNameSeparatesRadioAndLocalHost(t *testing.T) {
	for _, test := range []struct {
		name  string
		local bool
		valid bool
	}{
		{name: "capture.pcap", valid: true},
		{name: "capture.local.pcap", local: true, valid: true},
		{name: "capture.local.pcap", valid: false},
		{name: "capture.pcap", local: true, valid: false},
		{name: "capture.local.txt", local: true, valid: false},
	} {
		if err := validateCaptureOutputName(test.name, test.local); (err == nil) != test.valid {
			t.Fatalf("validateCaptureOutputName(%q, %t) err=%v valid=%t", test.name, test.local, err, test.valid)
		}
	}
}

func TestTCPDumpArgsSeparateMonitorAndLocalHostModes(t *testing.T) {
	monitor := tcpdumpArgs("en0", 60, true)
	local := tcpdumpArgs("en0", 60, false)
	if strings.Join(monitor, " ") != "-I -i en0 -s 4096 -G 60 -W 1 -w -" {
		t.Fatalf("monitor args=%v", monitor)
	}
	if strings.Join(local, " ") != "-i en0 -s 4096 -G 60 -W 1 -w -" {
		t.Fatalf("local args=%v", local)
	}
}

func TestCaptureDurationLimits(t *testing.T) {
	if err := validateCaptureDuration(maxDurationMinutes, true); err != nil {
		t.Fatal(err)
	}
	if err := validateCaptureDuration(localMaxDurationMinutes, false); err != nil {
		t.Fatal(err)
	}
	if err := validateCaptureDuration(localMaxDurationMinutes+1, false); err == nil {
		t.Fatal("local-host capture exceeded its segment limit")
	}
}

func TestCaptureByteLimitAppliesOnlyToLocalHost(t *testing.T) {
	if got := captureByteLimit("capture.local.pcap"); got != localMaxCaptureBytes {
		t.Fatalf("local limit=%d", got)
	}
	if got := captureByteLimit("capture.pcap"); got != 0 {
		t.Fatalf("radio limit=%d", got)
	}
}
