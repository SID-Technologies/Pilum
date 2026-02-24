package registry

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidatePathContainment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		path    string
		baseDir string
		wantErr bool
	}{
		{
			name:    "valid template path",
			path:    "./_templates/golang-api.v1.dockerfile",
			baseDir: "./_templates",
			wantErr: false,
		},
		{
			name:    "valid nested template",
			path:    "./_templates/subdir/template.dockerfile",
			baseDir: "./_templates",
			wantErr: false,
		},
		{
			name:    "path traversal to parent",
			path:    "./_templates/../../etc/passwd",
			baseDir: "./_templates",
			wantErr: true,
		},
		{
			name:    "path traversal with sibling",
			path:    "./_templates/../sibling/template",
			baseDir: "./_templates",
			wantErr: true,
		},
		{
			name:    "absolute path escape",
			path:    "/etc/passwd",
			baseDir: "./_templates",
			wantErr: true,
		},
		{
			name:    "base dir itself",
			path:    "./_templates",
			baseDir: "./_templates",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validatePathContainment(tt.path, tt.baseDir)
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "escapes base directory")
			} else {
				require.NoError(t, err)
			}
		})
	}
}
