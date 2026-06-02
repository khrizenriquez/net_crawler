package store

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/duku/net-lab/internal/model"
)

func TestValidateScheduleRejectsLongAndOverlappingWindows(t *testing.T) {
	existing := []model.CaptureSchedule{{Days: []int{1}, StartMinute: 600, DurationMinutes: 60, Channels: []int{2}, RotationMinutes: 5}}
	if err := ValidateSchedule(model.CaptureSchedule{Days: []int{2}, StartMinute: 600, DurationMinutes: 121, Channels: []int{2}, RotationMinutes: 5}, existing); err == nil {
		t.Fatal("expected duration validation error")
	}
	if err := ValidateSchedule(model.CaptureSchedule{Days: []int{1}, StartMinute: 630, DurationMinutes: 30, Channels: []int{36}, RotationMinutes: 5}, existing); err == nil {
		t.Fatal("expected overlap validation error")
	}
}

func TestValidateScheduleRejectsMissingFieldsAndAllowsAdjacentWindows(t *testing.T) {
	existing := []model.CaptureSchedule{{Days: []int{1}, StartMinute: 600, DurationMinutes: 60, Channels: []int{2}, RotationMinutes: 5}}
	for _, schedule := range []model.CaptureSchedule{
		{Days: []int{1}, DurationMinutes: 0, Channels: []int{2}, RotationMinutes: 5},
		{Days: []int{1}, DurationMinutes: 1, Channels: []int{2}, RotationMinutes: 0},
		{DurationMinutes: 1, Channels: []int{2}, RotationMinutes: 5},
		{Days: []int{1}, DurationMinutes: 1, RotationMinutes: 5},
	} {
		if err := ValidateSchedule(schedule, nil); err == nil {
			t.Fatalf("expected validation error for %+v", schedule)
		}
	}
	adjacent := model.CaptureSchedule{Days: []int{1}, StartMinute: 660, DurationMinutes: 30, Channels: []int{36}, RotationMinutes: 5}
	if err := ValidateSchedule(adjacent, existing); err != nil {
		t.Fatalf("adjacent schedule should pass: %v", err)
	}
}

func TestTickSchedulesQueuesMatchingWindowOnlyOnce(t *testing.T) {
	memory := NewMemory()
	now := time.Date(2026, 6, 1, 20, 0, 0, 0, time.Local)
	memory.automationEnabled = true
	memory.schedules = []model.CaptureSchedule{{Days: []int{int(now.Weekday())}, StartMinute: 20 * 60, DurationMinutes: 90, Channels: []int{2, 36}, RotationMinutes: 5, Enabled: true}}
	if got := memory.TickSchedules(now); len(got) != 1 {
		t.Fatalf("expected one command, got %d", len(got))
	}
	if got := memory.TickSchedules(now); len(got) != 0 {
		t.Fatalf("same minute queued duplicate command: %d", len(got))
	}
}

func TestTickSchedulesRespectsDisabledAutomationAndRunningSession(t *testing.T) {
	memory := NewMemory()
	now := time.Date(2026, 6, 1, 20, 0, 0, 0, time.Local)
	memory.schedules = []model.CaptureSchedule{{Days: []int{int(now.Weekday())}, StartMinute: 20 * 60, DurationMinutes: 90, Channels: []int{2}, RotationMinutes: 5, Enabled: true}}
	if got := memory.TickSchedules(now); len(got) != 0 {
		t.Fatalf("disabled automation queued %d commands", len(got))
	}
	memory.automationEnabled = true
	memory.sessions = []model.CaptureSession{{Status: "running"}}
	if got := memory.TickSchedules(now.Add(time.Minute)); len(got) != 0 {
		t.Fatalf("running session queued %d commands", len(got))
	}
}

func TestCompletedHostCommandsUpdateActiveSession(t *testing.T) {
	memory := NewMemory()
	start := memory.StartCapture(36)
	if err := memory.CompleteHostCommand(start.ID, "exit=0 capture started"); err != nil {
		t.Fatal(err)
	}
	if active := memory.Status().ActiveSession; active == nil || active.Channel != 36 {
		t.Fatalf("unexpected active session: %+v", active)
	}
	stop := memory.StopCapture()
	if err := memory.CompleteHostCommand(stop.ID, "exit=0 capture stopped"); err != nil {
		t.Fatal(err)
	}
	if active := memory.Status().ActiveSession; active != nil {
		t.Fatalf("session should be stopped: %+v", active)
	}
}

func TestStopCommandClosesSessionWhenHelperIsAlreadyIdle(t *testing.T) {
	memory := NewMemory()
	start := memory.StartLocalCapture()
	if err := memory.CompleteHostCommand(start.ID, "exit=0 capture started"); err != nil {
		t.Fatal(err)
	}
	stop := memory.StopCapture()
	if err := memory.CompleteHostCommand(stop.ID, "exit=1 capture is not running"); err != nil {
		t.Fatal(err)
	}
	if active := memory.Status().ActiveSession; active != nil {
		t.Fatalf("session should be reconciled closed: %+v", active)
	}
}

func TestCompletedLocalHostCommandCreatesPartialSession(t *testing.T) {
	memory := NewMemory()
	start := memory.StartLocalCapture()
	if start.Action != "start-local" {
		t.Fatalf("action=%q", start.Action)
	}
	if err := memory.CompleteHostCommand(start.ID, "exit=0 capture started"); err != nil {
		t.Fatal(err)
	}
	active := memory.Status().ActiveSession
	if active == nil || active.Origin != "local-host" || active.Coverage != model.CoveragePartial {
		t.Fatalf("unexpected active local-host session: %+v", active)
	}
}

func TestStartCaptureReusesPendingStartCommand(t *testing.T) {
	memory := NewMemory()
	first := memory.StartLocalCapture()
	second := memory.StartLocalCapture()
	if second.ID != first.ID {
		t.Fatalf("duplicate local start queued: first=%s second=%s", first.ID, second.ID)
	}
	third := memory.StartCapture(149)
	if third.ID != first.ID {
		t.Fatalf("radio start should reuse pending command: first=%s third=%s", first.ID, third.ID)
	}
}

func TestStopCommandCancelsPendingStartsAndTakesPriority(t *testing.T) {
	memory := NewMemory()
	start := memory.StartLocalCapture()
	stop := memory.StopCapture()
	next, err := memory.NextHostCommand()
	if err != nil {
		t.Fatal(err)
	}
	if next.ID != stop.ID {
		t.Fatalf("stop should have priority: next=%s stop=%s", next.ID, stop.ID)
	}
	if err := memory.CompleteHostCommand(stop.ID, "exit=1 capture is not running"); err != nil {
		t.Fatal(err)
	}
	for _, cmd := range memory.Snapshot().HostCommands {
		if cmd.ID == start.ID && cmd.Status != "canceled" {
			t.Fatalf("pending start was not canceled: %+v", cmd)
		}
	}
	if next, err := memory.NextHostCommand(); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unexpected next command after stop: %+v err=%v", next, err)
	}
}

func TestCompleteHostCommandIgnoresFailedExecutionAndMissingCommand(t *testing.T) {
	memory := NewMemory()
	start := memory.StartCapture(36)
	if err := memory.CompleteHostCommand(start.ID, "exit=1 helper failed"); err != nil {
		t.Fatal(err)
	}
	if active := memory.Status().ActiveSession; active != nil {
		t.Fatalf("failed command opened session: %+v", active)
	}
	if err := memory.CompleteHostCommand("missing", "exit=0"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestNextHostCommandAndSnapshotRestore(t *testing.T) {
	memory := NewMemory()
	if _, err := memory.NextHostCommand(); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected empty queue error, got %v", err)
	}
	cmd := memory.StartCapture(2)
	next, err := memory.NextHostCommand()
	if err != nil || next.ID != cmd.ID {
		t.Fatalf("unexpected next command %+v err=%v", next, err)
	}
	data, err := json.Marshal(memory.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatal(err)
	}
	restored := NewMemory()
	restored.Restore(snapshot)
	if err := restored.CompleteHostCommand(cmd.ID, "exit=0 capture started"); err != nil {
		t.Fatal(err)
	}
	if active := restored.Status().ActiveSession; active == nil || active.Channel != 2 {
		t.Fatalf("float JSON args were not restored: %+v", active)
	}
}

func TestSeedDemoResetAndCopies(t *testing.T) {
	memory := NewMemory()
	memory.SeedDemo()
	if len(memory.Radios()) != 2 || len(memory.Schedules()) != 1 || len(memory.Metrics()) != 3 || len(memory.Findings()) != 2 {
		t.Fatalf("incomplete demo snapshot: %+v", memory.Snapshot())
	}
	radios := memory.Radios()
	radios[0].SSID = "mutated"
	if memory.Radios()[0].SSID == "mutated" {
		t.Fatal("getter returned mutable internal slice")
	}
	memory.ResetDemo()
	if len(memory.Radios()) != 0 || len(memory.Findings()) != 0 {
		t.Fatal("demo reset kept data")
	}
}

func TestAddRadioScheduleIngestAndSortedMetrics(t *testing.T) {
	memory := NewMemory()
	radio := memory.AddRadio(model.AuthorizedRadio{SSID: "lab"})
	if radio.ID == "" || radio.CreatedAt.IsZero() {
		t.Fatalf("radio metadata missing: %+v", radio)
	}
	schedule, err := memory.AddSchedule(model.CaptureSchedule{Name: "night", Days: []int{1}, StartMinute: 10, DurationMinutes: 20, Channels: []int{2}, RotationMinutes: 5})
	if err != nil || schedule.ID == "" {
		t.Fatalf("schedule=%+v err=%v", schedule, err)
	}
	now := time.Now()
	memory.Ingest([]model.MetricBucket{{Timestamp: now.Add(time.Minute)}, {Timestamp: now}}, []model.Finding{{Category: "test"}})
	if memory.Metrics()[0].ID == "" || memory.Findings()[0].ID == "" || memory.Findings()[0].CreatedAt.IsZero() {
		t.Fatal("ingest did not assign metadata")
	}
	original := memory.Metrics()
	sorted := SortedMetrics(original)
	if !sorted[0].Timestamp.Before(sorted[1].Timestamp) || original[0].Timestamp.Before(original[1].Timestamp) {
		t.Fatal("metrics sorting mutated input or failed to sort")
	}
}
