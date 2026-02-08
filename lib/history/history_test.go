package history

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFilePath(t *testing.T) {
	got := FilePath("/project")
	want := filepath.Join("/project", ".pilum", "history.json")
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

func TestRecordPrepends(t *testing.T) {
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

func TestRecordCapsAt100(t *testing.T) {
	dir := t.TempDir()

	// Write 100 entries
	for i := range 100 {
		entry := NewEntry("build", "v"+string(rune('0'+i%10)), true, time.Second, nil)
		if err := Record(dir, entry); err != nil {
			t.Fatalf("Record #%d failed: %v", i, err)
		}
	}

	entries, _ := Load(dir)
	if len(entries) != 100 {
		t.Fatalf("expected 100 entries, got %d", len(entries))
	}

	// Adding one more should still be 100
	entry := NewEntry("deploy", "overflow", true, time.Second, nil)
	_ = Record(dir, entry)

	entries, _ = Load(dir)
	if len(entries) != 100 {
		t.Errorf("expected 100 entries after cap, got %d", len(entries))
	}
	if entries[0].Tag != "overflow" {
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
