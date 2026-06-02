package store

import (
	"os"
	"testing"
)

func TestPostgresPersistsDemoStateAcrossInstances(t *testing.T) {
	databaseURL := os.Getenv("DUKU_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set DUKU_TEST_DATABASE_URL to run PostgreSQL integration test")
	}
	first, err := NewPostgres(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	first.ResetDemo()
	first.SeedDemo()
	if len(first.Radios()) == 0 {
		t.Fatal("expected seeded radios")
	}
	second, err := NewPostgres(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Radios()) != len(first.Radios()) {
		t.Fatalf("state did not survive reconnect: first=%d second=%d", len(first.Radios()), len(second.Radios()))
	}
}
