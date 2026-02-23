// Package webhook provides generic HTTP webhook notifications for pipeline events.
// Webhooks are configured in .pilum.yml and fire on deploy start/success/failure.
package webhook

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/output"
)

// Event represents a pipeline lifecycle event.
type Event string

const (
	EventStart   Event = "start"
	EventSuccess Event = "success"
	EventFailure Event = "failure"
)

// Config represents a single webhook configuration from .pilum.yml.
type Config struct {
	URL    string  `mapstructure:"url"`
	Events []Event `mapstructure:"events"`
}

// Payload is the JSON body sent to webhook endpoints.
type Payload struct {
	Event    Event    `json:"event"`
	Command  string   `json:"command"`
	Tag      string   `json:"tag"`
	Services []string `json:"services"`
	Duration string   `json:"duration,omitempty"`
	Success  bool     `json:"success"`
	Error    string   `json:"error,omitempty"`
	Text     string   `json:"text"`
}

const httpTimeout = 10 * time.Second

// Send fires the webhook for each config that matches the given event.
// Errors are logged but do not fail the pipeline.
func Send(configs []Config, payload Payload) {
	for _, cfg := range configs {
		if !hasEvent(cfg.Events, payload.Event) {
			continue
		}

		if err := post(cfg.URL, payload); err != nil {
			output.Warning("Webhook failed for %s: %s", cfg.URL, err)
		}
	}
}

func hasEvent(events []Event, target Event) bool {
	if len(events) == 0 {
		// No events specified = all events
		return true
	}
	for _, e := range events {
		if e == target {
			return true
		}
	}
	return false
}

func post(url string, payload Payload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrap(err, "marshaling webhook payload")
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return errors.Wrap(err, "creating webhook request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "pilum-webhook")

	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, "sending webhook")
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return errors.New("webhook returned HTTP %d", resp.StatusCode)
	}

	return nil
}
