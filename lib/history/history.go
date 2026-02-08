package history

import (
	"bufio"
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
	historyFile = "history.jsonl"
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

// Load reads history entries from disk, returning most recent first.
// Returns nil if the file doesn't exist.
func Load(projectRoot string) ([]Entry, error) {
	fp := FilePath(projectRoot)

	f, err := os.Open(fp)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "reading history file")
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var e Entry
		if err := json.Unmarshal(line, &e); err != nil {
			continue // skip malformed lines
		}
		entries = append(entries, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, errors.Wrap(err, "scanning history file")
	}

	// Reverse so most recent is first (file is append-order)
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	return entries, nil
}

// Record appends an entry as a single JSON line to the history file.
func Record(projectRoot string, entry Entry) error {
	dir := filepath.Join(projectRoot, historyDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return errors.Wrap(err, "creating %s directory", historyDir)
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return errors.Wrap(err, "marshaling history entry")
	}
	data = append(data, '\n')

	fp := FilePath(projectRoot)
	f, err := os.OpenFile(fp, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return errors.Wrap(err, "opening history file")
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return errors.Wrap(err, "writing history entry")
	}

	return nil
}

// generateID returns a random 8-character hex string.
func generateID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// NewEntry creates an Entry with a generated ID and current timestamp.
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
