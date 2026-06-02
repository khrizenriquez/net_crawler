package analyzer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var bssidRE = regexp.MustCompile(`(?i)^[0-9a-f]{2}(:[0-9a-f]{2}){5}$`)

type FilterOptions struct {
	TSharkPath      string
	RawPath         string
	AuthorizedPath  string
	AuthorizedBSSID []string
	Timeout         time.Duration
}

func BuildBSSIDFilter(bssids []string) (string, error) {
	if len(bssids) == 0 {
		return "", errors.New("at least one authorized BSSID is required")
	}
	parts := make([]string, 0, len(bssids))
	for _, bssid := range bssids {
		bssid = strings.ToLower(strings.TrimSpace(bssid))
		if !bssidRE.MatchString(bssid) {
			return "", fmt.Errorf("invalid BSSID %q", bssid)
		}
		parts = append(parts, "wlan.bssid == "+bssid)
	}
	return "(" + strings.Join(parts, " || ") + ")", nil
}

// FilterAuthorizedPCAP deliberately removes the raw channel-wide capture on
// every path. Only an allowlisted copy is eligible for retention.
func FilterAuthorizedPCAP(opts FilterOptions) error {
	if opts.Timeout == 0 {
		opts.Timeout = 90 * time.Second
	}
	if filepath.Clean(opts.RawPath) == filepath.Clean(opts.AuthorizedPath) {
		return errors.New("raw and authorized paths must differ")
	}
	defer os.Remove(opts.RawPath)
	filter, err := BuildBSSIDFilter(opts.AuthorizedBSSID)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, opts.TSharkPath, "-r", opts.RawPath, "-Y", filter, "-w", opts.AuthorizedPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		os.Remove(opts.AuthorizedPath)
		return fmt.Errorf("tshark filter failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func RetainLocalHostPCAP(rawPath, retainedPath string) error {
	rawPath, retainedPath = filepath.Clean(rawPath), filepath.Clean(retainedPath)
	if rawPath == retainedPath {
		return errors.New("raw and retained paths must differ")
	}
	if !strings.HasSuffix(rawPath, ".local.pcap") || !strings.HasSuffix(retainedPath, ".local.authorized.pcap") {
		return errors.New("local-host paths must use .local.pcap and .local.authorized.pcap suffixes")
	}
	defer os.Remove(rawPath)
	if _, err := os.Lstat(retainedPath); err == nil {
		return errors.New("retained local-host capture already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Rename(rawPath, retainedPath)
}
