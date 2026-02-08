package history

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFilePath(t *testing.T) {
	got := FilePath("/project")
	want := filepath.Join("/project", ".pilum", "history.jsonl")
	if got != want {
		t.Errorf("FilePath = %q, want %q", got, want)
	}
}

func TestLoadEmptyDir(t *testing.T) {
	dir := t.TempDir()
	entries, err := Load(dir)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if entries != nil {
		t.Errorf("expected nil entries, got %d", len(entries))
	}
}

func TestRecordAndLoad(t *testing.T) {
	dir := t.TempDir()

	entry := NewEntry("deploy", "v1.0.0", true, 2*time.Second, []ServiceResult{
		{Name: "api", Step: "build", Success: true, Duration: "1s"},
		{Name: "web", Step: "build", Success: true, Duration: "1.5s"},
	})

	if err := Record(dir, entry); err != nil {
		t.Fatalf("Record returned error: %v", err)
	}

	entries, err := Load(dir)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	got := entries[0]
	if got.Command != "deploy" {
		t.Errorf("Command = %q, want %q", got.Command, "deploy")
	}
	if got.Tag != "v1.0.0" {
		t.Errorf("Tag = %q, want %q", got.Tag, "v1.0.0")
	}
	if !got.Success {
		t.Error("expected Success=true")
	}
	if len(got.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(got.Services))
	}
}

func TestRecordMostRecentFirst(t *testing.T) {
	dir := t.TempDir()

	first := NewEntry("build", "v1", true, time.Second, nil)
	second := NewEntry("deploy", "v2", true, time.Second, nil)

	_ = Record(dir, first)
	_ = Record(dir, second)

	entries, _ := Load(dir)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Command != "deploy" {
		t.Errorf("most recent entry should be first, got %q", entries[0].Command)
	}
	if entries[1].Command != "build" {
		t.Errorf("older entry should be second, got %q", entries[1].Command)
	}
}

func TestRecordAppendsToFile(t *testing.T) {
	dir := t.TempDir()

	for i := range 5 {
		entry := NewEntry("build", "v"+string(rune('0'+i)), true, time.Second, nil)
		if err := Record(dir, entry); err != nil {
			t.Fatalf("Record #%d failed: %v", i, err)
		}
	}

	entries, _ := Load(dir)
	if len(entries) != 5 {
		t.Fatalf("expected 5 entries, got %d", len(entries))
	}

	// Most recent (last written) should be first in slice
	if entries[0].Tag != "v4" {
		t.Errorf("newest entry should be first, got tag=%q", entries[0].Tag)
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	if len(id1) != 8 {
		t.Errorf("expected 8-char ID, got %d chars: %q", len(id1), id1)
	}
	if id1 == id2 {
		t.Error("expected unique IDs")
	}
}

func TestRecordCreatesDirectory(t *testing.T) {
	dir := t.TempDir()

	entry := NewEntry("build", "v1", true, time.Second, nil)
	if err := Record(dir, entry); err != nil {
		t.Fatalf("Record returned error: %v", err)
	}

	pilumDir := filepath.Join(dir, ".pilum")
	info, err := os.Stat(pilumDir)
	if err != nil {
		t.Fatalf(".pilum directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error(".pilum should be a directory")
	}
}

func TestNewEntry(t *testing.T) {
	entry := NewEntry("push", "latest", false, 3500*time.Millisecond, []ServiceResult{
		{Name: "svc", Step: "push", Success: false, Error: "timeout"},
	})

	if entry.ID == "" {
		t.Error("expected non-empty ID")
	}
	if entry.Command != "push" {
		t.Errorf("Command = %q, want %q", entry.Command, "push")
	}
	if entry.Success {
		t.Error("expected Success=false")
	}
	if entry.Duration != "3.5s" {
		t.Errorf("Duration = %q, want %q", entry.Duration, "3.5s")
	}
	if entry.Services[0].Error != "timeout" {
		t.Errorf("Service error = %q, want %q", entry.Services[0].Error, "timeout")
	}
}

func TestLoadSkipsMalformedLines(t *testing.T) {
	dir := t.TempDir()
	pilumDir := filepath.Join(dir, ".pilum")
	_ = os.MkdirAll(pilumDir, 0o755)

	// Write a file with one good line and one bad line
	content := `{"id":"abc","timestamp":"2025-01-01T00:00:00Z","command":"build","tag":"v1","success":true,"duration":"1s","services":[]}
not valid json
{"id":"def","timestamp":"2025-01-02T00:00:00Z","command":"deploy","tag":"v2","success":true,"duration":"2s","services":[]}
`
	fp := FilePath(dir)
	if err := os.WriteFile(fp, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := Load(dir)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 valid entries (skipping malformed), got %d", len(entries))
	}
	// Most recent first (reversed from file order)
	if entries[0].ID != "def" {
		t.Errorf("expected most recent entry first, got ID=%q", entries[0].ID)
	}
}
