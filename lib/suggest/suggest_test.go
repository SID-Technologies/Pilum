package suggest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected int
	}{
		{"identical strings", "hello", "hello", 0},
		{"empty strings", "", "", 0},
		{"one empty", "hello", "", 5},
		{"other empty", "", "world", 5},
		{"single substitution", "cat", "bat", 1},
		{"single insertion", "cat", "cats", 1},
		{"single deletion", "cats", "cat", 1},
		{"two substitutions", "cat", "dog", 3},
		{"case insensitive", "Hello", "hello", 0},
		{"case insensitive typo", "GCP", "gpc", 2},
		{"transposition", "gcp", "gpc", 2},
		{"common typo - missing char", "homebrew", "homebew", 1},
		{"common typo - extra char", "aws", "awss", 1},
		{"common typo - wrong char", "gcp", "gsp", 1},
		{"completely different", "apple", "orange", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LevenshteinDistance(tt.a, tt.b)
			assert.Equal(t, tt.expected, result, "LevenshteinDistance(%q, %q)", tt.a, tt.b)
		})
	}
}

func TestFindClosest(t *testing.T) {
	candidates := []string{"gcp", "aws", "azure", "homebrew", "kubernetes"}

	tests := []struct {
		name       string
		input      string
		maxResults int
		expected   []Match
	}{
		{
			name:       "exact match excluded but similar returned",
			input:      "gcp",
			maxResults: 3,
			expected:   []Match{{Value: "aws", Distance: 3}}, // gcp->aws is distance 3
		},
		{
			name:       "single typo - gpc instead of gcp",
			input:      "gpc",
			maxResults: 1, // limit to 1 to only get closest
			expected:   []Match{{Value: "gcp", Distance: 2}},
		},
		{
			name:       "single typo - awz instead of aws",
			input:      "awz",
			maxResults: 1,
			expected:   []Match{{Value: "aws", Distance: 1}},
		},
		{
			name:       "close to multiple - az returns closest first",
			input:      "az",
			maxResults: 1,
			expected:   []Match{{Value: "aws", Distance: 2}},
		},
		{
			name:       "homebew typo",
			input:      "homebew",
			maxResults: 3,
			expected:   []Match{{Value: "homebrew", Distance: 1}},
		},
		{
			name:       "no close match",
			input:      "docker",
			maxResults: 3,
			expected:   nil,
		},
		{
			name:       "limit results",
			input:      "a",
			maxResults: 1,
			expected:   []Match{{Value: "aws", Distance: 2}},
		},
		{
			name:       "empty candidates",
			input:      "gcp",
			maxResults: 3,
			expected:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCandidates := candidates
			if tt.name == "empty candidates" {
				testCandidates = nil
			}
			result := FindClosest(tt.input, testCandidates, tt.maxResults)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatSuggestion(t *testing.T) {
	candidates := []string{"kubernetes", "homebrew", "cloudflare"}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single suggestion - homebrew typo",
			input:    "homebrw",
			expected: "did you mean 'homebrew'?",
		},
		{
			name:     "no suggestion for exact match",
			input:    "kubernetes",
			expected: "",
		},
		{
			name:     "no suggestion for distant string",
			input:    "docker",
			expected: "",
		},
		{
			name:     "kubernetes typo",
			input:    "kuberntes",
			expected: "did you mean 'kubernetes'?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSuggestion(tt.input, candidates)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatSuggestion_MultipleSuggestions(t *testing.T) {
	// Create candidates where input is close to multiple
	candidates := []string{"test", "text", "tent", "best"}

	result := FormatSuggestion("taste", candidates)
	// Should suggest multiple close matches
	assert.Contains(t, result, "did you mean")
	assert.Contains(t, result, "test")
}

func TestHasCloseMatch(t *testing.T) {
	candidates := []string{"kubernetes", "homebrew", "cloudflare"}

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"has close match - typo", "homebrw", true},
		{"no close match", "docker", false},
		{"exact match counts as close", "kubernetes", true}, // distance 0 is <= MaxDistance
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasCloseMatch(tt.input, candidates)
			assert.Equal(t, tt.expected, result)
		})
	}
}
