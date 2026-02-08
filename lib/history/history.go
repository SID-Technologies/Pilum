package history

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/sid-technologies/pilum/lib/errors"
)

const (
	historyDir  = ".pilum"
	historyFile = "history.json"
	maxEntries  = 100
)

// Entry represents a single pipeline run.
type Entry struct {
	ID        string          `json:"id"`
	Timestamp time.Time       `json:"timestamp"`
	Command   string          `json:"command"`
	Tag       string          `json:"tag"`
	Success   bool            `json:"success"`
	Duration  string          `json:"duration"`
	Services  []ServiceResult `json:"services"`
}

// ServiceResult holds the outcome for one service in a pipeline run.
type ServiceResult struct {
	Name     string `json:"name"`
	Step     string `json:"step"`
	Success  bool   `json:"success"`
	Duration string `json:"duration"`
	Error    string `json:"error,omitempty"`
}

// FilePath returns the path to the history file under projectRoot.
func FilePath(projectRoot string) string {
	return filepath.Join(projectRoot, historyDir, historyFile)
}

// Load reads history entries from disk. Returns an empty slice if the file doesn't exist.
func Load(projectRoot string) ([]Entry, error) {
	fp := FilePath(projectRoot)

	data, err := os.ReadFile(fp)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "reading history file")
	}

	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, errors.Wrap(err, "parsing history file")
	}
	return entries, nil
}

// Record prepends an entry to the history file, capping at maxEntries.
func Record(projectRoot string, entry Entry) error {
	existing, err := Load(projectRoot)
	if err != nil {
		// If we can't load, start fresh
		existing = nil
	}

	entries := append([]Entry{entry}, existing...)
	if len(entries) > maxEntries {
		entries = entries[:maxEntries]
	}

	dir := filepath.Join(projectRoot, historyDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return errors.Wrap(err, "creating %s directory", historyDir)
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return errors.Wrap(err, "marshaling history")
	}

	fp := FilePath(projectRoot)
	if err := os.WriteFile(fp, data, 0o644); err != nil {
		return errors.Wrap(err, "writing history file")
	}

	return nil
}

// generateID returns a random 8-character hex string.
func generateID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// NewEntry creates a Entry with a generated ID and current timestamp.
func NewEntry(command, tag string, success bool, duration time.Duration, services []ServiceResult) Entry {
	return Entry{
		ID:        generateID(),
		Timestamp: time.Now(),
		Command:   command,
		Tag:       tag,
		Success:   success,
		Duration:  duration.Round(time.Millisecond).String(),
		Services:  services,
	}
}
