package analyzer

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPruneAuthorizedPCAPAppliesAgeAndQuota(t *testing.T) {
	dir, now := t.TempDir(), time.Now()
	write := func(name string, size int, age time.Duration) {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, make([]byte, size), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, now.Add(-age), now.Add(-age)); err != nil {
			t.Fatal(err)
		}
	}
	write("expired.authorized.pcap", 3, 73*time.Hour)
	write("oldest.authorized.pcap", 6, 2*time.Hour)
	write("newest.authorized.pcap", 6, time.Hour)
	if err := PruneAuthorizedPCAP(dir, 72*time.Hour, 6, now); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "expired.authorized.pcap")); !os.IsNotExist(err) {
		t.Fatal("expired file survived")
	}
	if _, err := os.Stat(filepath.Join(dir, "oldest.authorized.pcap")); !os.IsNotExist(err) {
		t.Fatal("oldest file should be evicted by quota")
	}
	if _, err := os.Stat(filepath.Join(dir, "newest.authorized.pcap")); err != nil {
		t.Fatal("newest file should survive")
	}
}

func TestPruneAuthorizedPCAPIgnoresMissingDirectoryAndUnrelatedFiles(t *testing.T) {
	dir := t.TempDir()
	if err := PruneAuthorizedPCAP(filepath.Join(dir, "missing"), time.Hour, 1, time.Now()); err != nil {
		t.Fatal(err)
	}
	notes := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(notes, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := PruneAuthorizedPCAP(dir, time.Hour, 1, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(notes); err != nil {
		t.Fatalf("unrelated file was removed: %v", err)
	}
}
