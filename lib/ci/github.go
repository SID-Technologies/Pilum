// Package ci provides CI/CD integrations for Pilum.
// Currently supports GitHub Actions commit status updates.
package ci

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/sid-technologies/pilum/lib/errors"
)

// State represents a GitHub commit status state.
type State string

const (
	StatePending State = "pending"
	StateSuccess State = "success"
	StateFailure State = "failure"
)

// GitHubEnv holds the environment variables needed for GitHub API calls.
type GitHubEnv struct {
	Token      string
	SHA        string
	Repository string // owner/repo format
}

// statusRequest is the request body for the GitHub Statuses API.
type statusRequest struct {
	State       State  `json:"state"`
	Context     string `json:"context"`
	Description string `json:"description"`
	TargetURL   string `json:"target_url,omitempty"`
}

// DetectGitHub returns a GitHubEnv if running in GitHub Actions with all
// required environment variables set. Returns nil otherwise.
func DetectGitHub() *GitHubEnv {
	if os.Getenv("GITHUB_ACTIONS") != "true" {
		return nil
	}

	token := os.Getenv("GITHUB_TOKEN")
	sha := os.Getenv("GITHUB_SHA")
	repo := os.Getenv("GITHUB_REPOSITORY")

	if token == "" || sha == "" || repo == "" {
		return nil
	}

	return &GitHubEnv{
		Token:      token,
		SHA:        sha,
		Repository: repo,
	}
}

// PostStatus creates a commit status on the GitHub repository.
func (gh *GitHubEnv) PostStatus(state State, context, description, targetURL string) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/statuses/%s", gh.Repository, gh.SHA)

	body := statusRequest{
		State:       state,
		Context:     context,
		Description: description,
		TargetURL:   targetURL,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return errors.Wrap(err, "marshaling status request")
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return errors.Wrap(err, "creating status request")
	}

	req.Header.Set("Authorization", "Bearer "+gh.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "posting commit status")
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.New("github API returned status %d for commit status update", resp.StatusCode)
	}

	return nil
}
