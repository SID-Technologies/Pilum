package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/sid-technologies/pilum/lib/output"
)

const (
	defaultEndpoint = "https://t.pilum.dev/v1/events"
	// sendTimeout bounds how long telemetry can delay a command's exit.
	sendTimeout = time.Second
	pauseFor    = 7 * 24 * time.Hour
	maxEvents   = 50
)

// endpoint is t.pilum.dev, or PILUM_TELEMETRY_ENDPOINT for tests and local Worker development.
func endpoint() string {
	if v := os.Getenv("PILUM_TELEMETRY_ENDPOINT"); v != "" {
		return v
	}
	return defaultEndpoint
}

func lastPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "last-telemetry.json"), nil
}

// Last returns the last payload sent, for `pilum telemetry last`.
func Last() ([]byte, error) {
	p, err := lastPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p) //nolint:gosec // path is the user's own config dir
	if err != nil {
		return nil, errors.New("no telemetry has been sent from this machine")
	}
	return data, nil
}

func paused(s Settings, now time.Time) bool {
	if s.PausedUntil == "" {
		return false
	}
	until, err := time.Parse(time.RFC3339, s.PausedUntil)
	return err == nil && now.Before(until)
}

// Flush sends the current run, if there is one and telemetry is on. It never
// returns an error and gives up after one second, so it can't slow down or
// fail a command. Call it once, after the command has finished.
func Flush(version string, runErr error) {
	r := take()
	if r == nil {
		return
	}
	if on, _ := Enabled(); !on {
		return
	}
	now := time.Now()
	if paused(Load(), now) {
		return
	}

	events := BuildEvents(r, version, runErr, now)
	if len(events) > maxEvents {
		events = events[:maxEvents]
	}
	body, err := json.Marshal(Batch{
		Schema:    schemaVersion,
		InstallID: installID(),
		SentAt:    now.UTC().Format(time.RFC3339),
		Events:    events,
	})
	if err != nil {
		return
	}
	if p, err := lastPath(); err == nil {
		_ = os.WriteFile(p, body, 0o600)
	}

	accept, err := send(body)
	if err != nil {
		output.Debugf("telemetry not sent: %v", err)
		return
	}
	if !accept {
		s := Load()
		s.PausedUntil = now.Add(pauseFor).UTC().Format(time.RFC3339)
		_ = Save(s)
	}
}

func send(body []byte) (accept bool, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint(), bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: sendTimeout}
	res, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusAccepted {
		return false, errors.New("endpoint answered " + strconv.Itoa(res.StatusCode))
	}
	var reply struct {
		Accept bool `json:"accept"`
	}
	if err := json.NewDecoder(res.Body).Decode(&reply); err != nil {
		return false, err
	}
	return reply.Accept, nil
}
