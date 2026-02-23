package webhook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSend_MatchingEvent(t *testing.T) {
	var received Payload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))
		require.Equal(t, "pilum-webhook", r.Header.Get("User-Agent"))
		_ = json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	configs := []Config{
		{URL: server.URL, Events: []Event{EventFailure}},
	}

	payload := Payload{
		Event:    EventFailure,
		Command:  "deploy",
		Tag:      "v1.0.0",
		Services: []string{"api", "worker"},
		Success:  false,
		Error:    "build failed",
		Text:     "Deploy failed: build failed",
	}

	Send(configs, payload)

	require.Equal(t, EventFailure, received.Event)
	require.Equal(t, "deploy", received.Command)
	require.Equal(t, "v1.0.0", received.Tag)
	require.Equal(t, []string{"api", "worker"}, received.Services)
	require.False(t, received.Success)
	require.Equal(t, "build failed", received.Error)
}

func TestSend_NonMatchingEvent(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	configs := []Config{
		{URL: server.URL, Events: []Event{EventFailure}},
	}

	payload := Payload{Event: EventSuccess}
	Send(configs, payload)

	require.False(t, called)
}

func TestSend_EmptyEventsMatchesAll(t *testing.T) {
	callCount := 0
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		callCount++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	configs := []Config{
		{URL: server.URL, Events: []Event{}},
	}

	for _, event := range []Event{EventStart, EventSuccess, EventFailure} {
		Send(configs, Payload{Event: event})
	}

	require.Equal(t, 3, callCount)
}

func TestSend_MultipleWebhooks(t *testing.T) {
	var mu sync.Mutex
	urls := make(map[string]bool)

	handler := func(id string) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			urls[id] = true
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
		})
	}

	s1 := httptest.NewServer(handler("s1"))
	defer s1.Close()
	s2 := httptest.NewServer(handler("s2"))
	defer s2.Close()

	configs := []Config{
		{URL: s1.URL, Events: []Event{EventFailure}},
		{URL: s2.URL, Events: []Event{EventStart, EventFailure}},
	}

	Send(configs, Payload{Event: EventFailure})

	require.True(t, urls["s1"])
	require.True(t, urls["s2"])
}

func TestSend_HTTPErrorDoesNotPanic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	configs := []Config{
		{URL: server.URL, Events: []Event{EventFailure}},
	}

	// Should not panic — errors are logged, not returned
	Send(configs, Payload{Event: EventFailure})
}

func TestSend_BadURLDoesNotPanic(t *testing.T) {
	configs := []Config{
		{URL: "http://localhost:1", Events: []Event{EventFailure}},
	}

	// Should not panic — connection refused is logged
	Send(configs, Payload{Event: EventFailure})
}

func TestSend_EmptyConfigs(t *testing.T) {
	// Should be a no-op
	Send(nil, Payload{Event: EventFailure})
	Send([]Config{}, Payload{Event: EventSuccess})
}

func TestHasEvent(t *testing.T) {
	require.True(t, hasEvent([]Event{EventStart, EventFailure}, EventFailure))
	require.False(t, hasEvent([]Event{EventStart, EventFailure}, EventSuccess))
	require.True(t, hasEvent([]Event{}, EventSuccess), "empty events should match all")
	require.True(t, hasEvent(nil, EventStart), "nil events should match all")
}

func TestPayload_JSONRoundTrip(t *testing.T) {
	p := Payload{
		Event:    EventSuccess,
		Command:  "deploy",
		Tag:      "v2.0.0",
		Services: []string{"api"},
		Duration: "12s",
		Success:  true,
		Text:     "Deployed 1 service(s) in 12s",
	}

	data, err := json.Marshal(p)
	require.NoError(t, err)

	var decoded Payload
	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Equal(t, p, decoded)

	// Verify "error" field is omitted when empty
	var raw map[string]any
	require.NoError(t, json.Unmarshal(data, &raw))
	_, hasError := raw["error"]
	require.False(t, hasError, "error field should be omitted when empty")
}
