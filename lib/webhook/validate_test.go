package webhook

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateWebhookURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		url     string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid https URL",
			url:     "https://hooks.slack.com/services/T00/B00/xxx",
			wantErr: false,
		},
		{
			name:    "http localhost allowed",
			url:     "http://localhost:8080/webhook",
			wantErr: false,
		},
		{
			name:    "http 127.0.0.1 allowed",
			url:     "http://127.0.0.1:9090/hook",
			wantErr: false,
		},
		{
			name:    "http remote blocked",
			url:     "http://example.com/webhook",
			wantErr: true,
			errMsg:  "http is only allowed for localhost",
		},
		{
			name:    "file scheme blocked",
			url:     "file:///etc/passwd",
			wantErr: true,
			errMsg:  "unsupported scheme",
		},
		{
			name:    "ftp scheme blocked",
			url:     "ftp://example.com/file",
			wantErr: true,
			errMsg:  "unsupported scheme",
		},
		{
			name:    "empty scheme blocked",
			url:     "example.com/webhook",
			wantErr: true,
			errMsg:  "unsupported scheme",
		},
		{
			name:    "AWS metadata IP blocked",
			url:     "https://169.254.169.254/latest/meta-data/",
			wantErr: true,
			errMsg:  "private IP",
		},
		{
			name:    "private 10.x blocked",
			url:     "https://10.0.0.1/webhook",
			wantErr: true,
			errMsg:  "private IP",
		},
		{
			name:    "private 172.16.x blocked",
			url:     "https://172.16.0.1/webhook",
			wantErr: true,
			errMsg:  "private IP",
		},
		{
			name:    "private 192.168.x blocked",
			url:     "https://192.168.1.1/webhook",
			wantErr: true,
			errMsg:  "private IP",
		},
		{
			name:    "no hostname blocked",
			url:     "https:///path",
			wantErr: true,
			errMsg:  "no hostname",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateWebhookURL(tt.url)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					require.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
