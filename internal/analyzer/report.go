package analyzer

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/duku/net-lab/internal/model"
)

type Report struct {
	Metrics  []model.MetricBucket `json:"metrics"`
	Findings []model.Finding      `json:"findings"`
}

func AnalyzePCAP(tsharkPath, path string) (Report, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	args := []string{"-r", path, "-T", "fields", "-E", "separator=\t", "-E", "occurrence=f",
		"-e", "frame.time_epoch", "-e", "frame.len", "-e", "wlan.sa", "-e", "ip.dst",
		"-e", "_ws.col.Protocol", "-e", "dns.qry.name", "-e", "http.host", "-e", "http.request.uri",
		"-e", "http.file_data", "-e", "ftp.request.command", "-e", "ftp.request.arg",
		"-e", "telnet.data", "-e", "smtp.req.command", "-e", "smtp.req.parameter",
		"-e", "pop.request", "-e", "imap.request"}
	cmd := exec.CommandContext(ctx, tsharkPath, args...)
	output, err := cmd.Output()
	if err != nil {
		return Report{}, fmt.Errorf("tshark analyze failed: %w", err)
	}
	return ParseTSharkRows(strings.NewReader(string(output)))
}

func ParseTSharkRows(input io.Reader) (Report, error) {
	reader := csv.NewReader(bufio.NewReader(input))
	reader.Comma = '\t'
	reader.FieldsPerRecord = -1
	var report Report
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return Report{}, err
		}
		for len(row) < 17 {
			row = append(row, "")
		}
		timestamp := parseEpoch(row[0])
		size, _ := strconv.ParseInt(row[1], 10, 64)
		domain := firstNonEmpty(row[6], row[5])
		protocol := strings.ToUpper(firstNonEmpty(row[4], "UNKNOWN"))
		report.Metrics = append(report.Metrics, model.MetricBucket{Timestamp: timestamp, Source: model.CoverageWiFiObserved, DeviceMAC: row[2], RemoteIP: row[3], Domain: domain, Protocol: protocol, BytesUp: size, PacketsUp: 1})
		sample := strings.TrimSpace(strings.Join(row[7:], " "))
		if sample == "" {
			continue
		}
		category, severity := "", "info"
		switch {
		case LooksSensitive(sample):
			category, severity = "possible_credential", "high"
		case strings.Contains(strings.ToLower(sample), "base64"):
			category = "base64_media"
		}
		if category != "" {
			report.Findings = append(report.Findings, model.Finding{Category: category, Severity: severity, Protocol: protocol, DeviceMAC: row[2], RemoteIP: row[3], RedactedSample: Redact(sample), ContentHash: HashContent(sample), CreatedAt: timestamp})
		}
	}
	return report, nil
}

func parseEpoch(raw string) time.Time {
	seconds, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return time.Now()
	}
	whole := int64(seconds)
	return time.Unix(whole, int64((seconds-float64(whole))*1_000_000_000))
}
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
