package analyzer

import (
	"strings"
	"testing"
)

func TestParseTSharkRowsCreatesRedactedFinding(t *testing.T) {
	row := "1710000000.5\t128\t02:00:00:00:10:02\t192.0.2.24\tHTTP\t\tlab.example.test\t/login?user=hello@example.test&password=hunter2\t\t\t\t\t\t\t\t\t\n"
	report, err := ParseTSharkRows(strings.NewReader(row))
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Metrics) != 1 || len(report.Findings) != 1 {
		t.Fatalf("report=%+v", report)
	}
	sample := report.Findings[0].RedactedSample
	if strings.Contains(sample, "hunter2") || strings.Contains(sample, "hello@example.test") {
		t.Fatalf("unsafe sample: %s", sample)
	}
}

func TestParseTSharkRowsDetectsBase64AndFallbackProtocol(t *testing.T) {
	row := "\t64\tdevice\tremote\t\t\t\t/data\tdata:image/png;base64,QUJDREVGR0hJSktMTU5PUFFSU1RVVldYWVo=\t\t\t\t\t\t\t\t\n"
	report, err := ParseTSharkRows(strings.NewReader(row))
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Metrics) != 1 || report.Metrics[0].Protocol != "UNKNOWN" {
		t.Fatalf("unexpected metrics: %+v", report.Metrics)
	}
	if len(report.Findings) != 1 || report.Findings[0].Category != "base64_media" {
		t.Fatalf("unexpected findings: %+v", report.Findings)
	}
	if strings.Contains(report.Findings[0].RedactedSample, "QUJDREV") {
		t.Fatal("base64 payload leaked")
	}
}

func TestParseTSharkRowsRejectsMalformedInput(t *testing.T) {
	if _, err := ParseTSharkRows(strings.NewReader("\"unterminated\n")); err == nil {
		t.Fatal("expected malformed row error")
	}
}

func TestAnalyzePCAPReportsMissingTShark(t *testing.T) {
	if _, err := AnalyzePCAP("/does/not/exist", "fixture.pcap"); err == nil {
		t.Fatal("expected missing tshark error")
	}
}
