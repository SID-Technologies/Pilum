// Package suggest provides fuzzy matching and "did you mean?" suggestions
// using Levenshtein distance for typo detection.
package suggest

import (
	"sort"
	"strings"
)

// MaxDistance is the maximum Levenshtein distance to consider for suggestions.
// Matches beyond this distance are not considered similar enough.
const MaxDistance = 3

// Match represents a potential suggestion with its distance score.
type Match struct {
	Value    string
	Distance int
}

// LevenshteinDistance calculates the minimum number of single-character edits
// (insertions, deletions, or substitutions) required to change one string into another.
func LevenshteinDistance(a, b string) int {
	a = strings.ToLower(a)
	b = strings.ToLower(b)

	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Create two work vectors of integer distances
	v0 := make([]int, len(b)+1)
	v1 := make([]int, len(b)+1)

	// Initialize v0 (the previous row of distances)
	for i := range v0 {
		v0[i] = i
	}

	for i := 0; i < len(a); i++ {
		// First element of v1 is edit distance from empty string to a[0..i]
		v1[0] = i + 1

		for j := 0; j < len(b); j++ {
			deletionCost := v0[j+1] + 1
			insertionCost := v1[j] + 1

			var substitutionCost int
			if a[i] == b[j] {
				substitutionCost = v0[j]
			} else {
				substitutionCost = v0[j] + 1
			}

			v1[j+1] = min(deletionCost, insertionCost, substitutionCost)
		}

		// Swap v0 and v1
		v0, v1 = v1, v0
	}

	return v0[len(b)]
}

// FindClosest returns the closest matching strings from candidates for the given input.
// It returns up to maxResults matches that are within MaxDistance.
// Results are sorted by distance (closest first).
func FindClosest(input string, candidates []string, maxResults int) []Match {
	if len(candidates) == 0 || maxResults <= 0 {
		return nil
	}

	var matches []Match

	for _, candidate := range candidates {
		dist := LevenshteinDistance(input, candidate)
		if dist <= MaxDistance && dist > 0 {
			matches = append(matches, Match{
				Value:    candidate,
				Distance: dist,
			})
		}
	}

	// Sort by distance (closest first), then alphabetically for ties
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Distance != matches[j].Distance {
			return matches[i].Distance < matches[j].Distance
		}
		return matches[i].Value < matches[j].Value
	})

	if len(matches) > maxResults {
		matches = matches[:maxResults]
	}

	return matches
}

// FormatSuggestion returns a formatted "did you mean?" suggestion string.
// Returns empty string if no close matches are found.
func FormatSuggestion(input string, candidates []string) string {
	matches := FindClosest(input, candidates, 3)
	if len(matches) == 0 {
		return ""
	}

	if len(matches) == 1 {
		return "did you mean '" + matches[0].Value + "'?"
	}

	// Multiple suggestions
	var suggestions []string
	for _, m := range matches {
		suggestions = append(suggestions, "'"+m.Value+"'")
	}

	return "did you mean " + strings.Join(suggestions[:len(suggestions)-1], ", ") +
		" or " + suggestions[len(suggestions)-1] + "?"
}

// HasCloseMatch returns true if there's at least one candidate within MaxDistance.
func HasCloseMatch(input string, candidates []string) bool {
	for _, candidate := range candidates {
		if LevenshteinDistance(input, candidate) <= MaxDistance {
			return true
		}
	}
	return false
}
