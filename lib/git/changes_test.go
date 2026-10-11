package git

import (
	"testing"
)

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
