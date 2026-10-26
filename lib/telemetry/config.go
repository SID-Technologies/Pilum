// Package telemetry sends opt-in, anonymous build telemetry to
// https://t.pilum.dev (the Pilum-Metrics-Worker), which forwards it to
// PostHog. Nothing is sent unless the user opted in, and never service
// names, project IDs, paths or error messages. See
// https://pilum.dev/docs/reference/telemetry/.
package telemetry

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Settings is the per-user config file, os.UserConfigDir()/pilum/config.yaml.
// It is separate from a project's .pilum.yml because sharing telemetry is a
// person's choice, not a project's.
type Settings struct {
	Telemetry   string `yaml:"telemetry,omitempty"` // "enabled", "disabled", or empty when never asked
	InstallID   string `yaml:"install_id,omitempty"`
	AskedAt     string `yaml:"asked_at,omitempty"`
	PausedUntil string `yaml:"paused_until,omitempty"` // set when the endpoint answers accept:false
}

const (
	stateEnabled  = "enabled"
	stateDisabled = "disabled"
)

// Source says which setting decided whether telemetry is on.
type Source string

const (
	SourceDoNotTrack Source = "DO_NOT_TRACK"
	SourceEnv        Source = "PILUM_TELEMETRY"
	SourceConfig     Source = "config file"
	SourceDefault    Source = "default"
)

// configDir is a variable so tests can point it at a temp directory.
var configDir = func() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "pilum"), nil
}

func settingsPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// Load reads the user's settings. A missing or unreadable file is empty settings.
func Load() Settings {
	var s Settings
	p, err := settingsPath()
	if err != nil {
		return s
	}
	data, err := os.ReadFile(p) //nolint:gosec // path is the user's own config dir
	if err != nil {
		return s
	}
	_ = yaml.Unmarshal(data, &s)
	return s
}

// Save writes the user's settings.
func Save(s Settings) error {
	p, err := settingsPath()
	if err != nil {
		return err
	}
	// Telemetry must never panic or print, so plain errors here rather than
	// lib/errors (whose New prints and whose Wrap panics on nil).
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	data, err := yaml.Marshal(s)
	if err != nil {
		return errors.Join(errors.New("encoding telemetry settings"), err)
	}
	return os.WriteFile(p, data, 0o600)
}

// envOn parses PILUM_TELEMETRY. ok is false when the variable is unset or unrecognized.
func envOn(v string) (on, ok bool) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true, true
	case "0", "false", "no", "off":
		return false, true
	}
	return false, false
}

// doNotTrack follows the consoledonottrack.com convention: any value but "0" or "false".
func doNotTrack() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("DO_NOT_TRACK")))
	return v != "" && v != "0" && v != "false"
}

// Enabled reports whether telemetry is on and which setting decided it.
// DO_NOT_TRACK beats PILUM_TELEMETRY, which beats the config file; the default is off.
func Enabled() (bool, Source) {
	if doNotTrack() {
		return false, SourceDoNotTrack
	}
	if on, ok := envOn(os.Getenv("PILUM_TELEMETRY")); ok {
		return on, SourceEnv
	}
	switch Load().Telemetry {
	case stateEnabled:
		return true, SourceConfig
	case stateDisabled:
		return false, SourceConfig
	}
	return false, SourceDefault
}

// Asked reports whether the user has already answered the prompt (or ran enable/disable).
func Asked() bool {
	return Load().Telemetry != ""
}

// Enable turns telemetry on in the config file and creates the install ID.
func Enable() (Settings, error) {
	s := Load()
	s.Telemetry = stateEnabled
	if s.InstallID == "" {
		s.InstallID = newInstallID()
	}
	s.AskedAt = time.Now().UTC().Format("2006-01-02")
	return s, Save(s)
}

// Disable turns telemetry off in the config file and deletes the install ID.
func Disable() (Settings, error) {
	s := Load()
	s.Telemetry = stateDisabled
	s.InstallID = ""
	s.AskedAt = time.Now().UTC().Format("2006-01-02")
	return s, Save(s)
}

// newInstallID returns a random UUID v4. It identifies an install, not a person or machine.
func newInstallID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// installID returns the saved ID, creating one when telemetry was turned on by
// PILUM_TELEMETRY without the config file ever being written (CI runners).
func installID() string {
	s := Load()
	if s.InstallID != "" {
		return s.InstallID
	}
	// Telemetry stays unset in the file: an env-only opt-in shouldn't stop the
	// prompt from asking on an interactive machine later. On an ephemeral CI
	// runner this is a new ID per job, which is fine: it counts runs, not people.
	s.InstallID = newInstallID()
	_ = Save(s)
	return s.InstallID
}
