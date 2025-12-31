package git

import (
	"testing"
)

func TestServiceHasChanges(t *testing.T) {
	tests := []struct {
		name         string
		servicePath  string
		changedFiles []string
		want         bool
	}{
		{
			name:         "service has changes - direct file",
			servicePath:  "services/auth-api",
			changedFiles: []string{"services/auth-api/main.go", "services/other/file.go"},
			want:         true,
		},
		{
			name:         "service has changes - nested file",
			servicePath:  "services/auth-api",
			changedFiles: []string{"services/auth-api/pkg/handler/auth.go"},
			want:         true,
		},
		{
			name:         "service has no changes",
			servicePath:  "services/auth-api",
			changedFiles: []string{"services/other-api/main.go", "README.md"},
			want:         false,
		},
		{
			name:         "empty changed files",
			servicePath:  "services/auth-api",
			changedFiles: []string{},
			want:         false,
		},
		{
			name:         "similar prefix but different service",
			servicePath:  "services/auth",
			changedFiles: []string{"services/auth-api/main.go"},
			want:         false,
		},
		{
			name:         "root service path",
			servicePath:  ".",
			changedFiles: []string{"main.go", "pkg/util.go"},
			want:         true,
		},
		{
			name:         "service path with trailing slash normalization",
			servicePath:  "services/api/",
			changedFiles: []string{"services/api/handler.go"},
			want:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ServiceHasChanges(tt.servicePath, tt.changedFiles)
			if got != tt.want {
				t.Errorf("ServiceHasChanges(%q, %v) = %v, want %v",
					tt.servicePath, tt.changedFiles, got, tt.want)
			}
		})
	}
}

func TestParseLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "multiple lines",
			input: "file1.go\nfile2.go\nfile3.go",
			want:  []string{"file1.go", "file2.go", "file3.go"},
		},
		{
			name:  "with empty lines",
			input: "file1.go\n\nfile2.go\n\n",
			want:  []string{"file1.go", "file2.go"},
		},
		{
			name:  "with whitespace",
			input: "  file1.go  \n  file2.go  ",
			want:  []string{"file1.go", "file2.go"},
		},
		{
			name:  "empty input",
			input: "",
			want:  []string{},
		},
		{
			name:  "only whitespace",
			input: "   \n   \n   ",
			want:  []string{},
		},
		{
			name:  "single file",
			input: "file.go",
			want:  []string{"file.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLines(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("parseLines(%q) returned %d items, want %d",
					tt.input, len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseLines(%q)[%d] = %q, want %q",
						tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestIsGitRepository(t *testing.T) {
	// This test will pass if run from within the Pilum repo
	// which is a git repository
	if !IsGitRepository() {
		t.Skip("Not running in a git repository")
	}

	// If we get here, the function correctly detected we're in a git repo
}
