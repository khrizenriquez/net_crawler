package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/duku/net-lab/internal/analyzer"
)

type config struct {
	StagingDir      string   `json:"stagingDir"`
	AuthorizedDir   string   `json:"authorizedDir"`
	AuthorizedBSSID []string `json:"authorizedBssid"`
	APIURL          string   `json:"apiUrl"`
	HostToken       string   `json:"hostToken"`
}

func main() {
	cfg := config{StagingDir: env("DUKU_STAGING_DIR", "/data/staging"), AuthorizedDir: env("DUKU_AUTHORIZED_DIR", "/data/authorized"), AuthorizedBSSID: split(env("DUKU_AUTHORIZED_BSSID", "")), APIURL: env("DUKU_API_URL", "http://api:8080/api/v1/ingest"), HostToken: os.Getenv("DUKU_HOST_TOKEN")}
	if raw := os.Getenv("DUKU_WORKER_CONFIG"); raw != "" {
		data, err := os.ReadFile(raw)
		if err != nil {
			log.Fatal(err)
		}
		if err := json.Unmarshal(data, &cfg); err != nil {
			log.Fatal(err)
		}
	}
	if err := os.MkdirAll(cfg.AuthorizedDir, 0o700); err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(cfg.StagingDir, 0o700); err != nil {
		log.Fatal(err)
	}
	log.Printf("worker watching %s with %d authorized BSSID values", cfg.StagingDir, len(cfg.AuthorizedBSSID))
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		process(cfg)
	}
}

func process(cfg config) {
	if err := analyzer.PruneAuthorizedPCAP(cfg.AuthorizedDir, 72*time.Hour, 5*1024*1024*1024, time.Now()); err != nil {
		log.Printf("prune retained captures: %v", err)
	}
	entries, err := os.ReadDir(cfg.StagingDir)
	if err != nil {
		log.Printf("read staging: %v", err)
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || !isCompletedCapture(entry.Name()) {
			continue
		}
		raw := filepath.Join(cfg.StagingDir, entry.Name())
		authorized := filepath.Join(cfg.AuthorizedDir, strings.TrimSuffix(entry.Name(), ".pcap")+".authorized.pcap")
		if err := analyzer.FilterAuthorizedPCAP(analyzer.FilterOptions{TSharkPath: env("TSHARK_PATH", "tshark"), RawPath: raw, AuthorizedPath: authorized, AuthorizedBSSID: cfg.AuthorizedBSSID}); err != nil {
			log.Printf("filter %s: %v", entry.Name(), err)
			continue
		}
		report, err := analyzer.AnalyzePCAP(env("TSHARK_PATH", "tshark"), authorized)
		if err != nil {
			log.Printf("analyze %s: %v", authorized, err)
			continue
		}
		if err := uploadReport(cfg, report); err != nil {
			log.Printf("upload report %s: %v", authorized, err)
			continue
		}
		log.Printf("retained authorized capture %s", authorized)
	}
}

func isCompletedCapture(name string) bool {
	return strings.HasSuffix(name, ".pcap")
}

func uploadReport(cfg config, report analyzer.Report) error {
	payload, err := json.Marshal(report)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, cfg.APIURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Duku-Host-Token", cfg.HostToken)
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		return fmt.Errorf("ingest returned %s", response.Status)
	}
	return nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
func split(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var values []string
	for _, value := range strings.Split(raw, ",") {
		if value = strings.TrimSpace(value); value != "" {
			values = append(values, value)
		}
	}
	return values
}
