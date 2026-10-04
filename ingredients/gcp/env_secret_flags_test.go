package gcp_test

import (
	"testing"

	"github.com/sid-technologies/pilum/ingredients/gcp"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

var (
	oneEnv    = []serviceinfo.EnvVars{{Name: "LOG_LEVEL", Value: "debug"}}
	oneSecret = []serviceinfo.Secrets{{Name: "DB_PASSWORD", Value: "db-pass:latest"}}
)

// requireEnvSecretFlags checks one container's flag segment states the full env and secret sets.
func requireEnvSecretFlags(t *testing.T, seg []string, wantEnv, wantSecret string) {
	t.Helper()

	requireStatedSet(t, seg, "--set-env-vars", "--clear-env-vars", wantEnv)
	requireStatedSet(t, seg, "--set-secrets", "--clear-secrets", wantSecret)
}

// requireStatedSet expects --set-X with want when want is non-empty, otherwise --clear-X alone.
func requireStatedSet(t *testing.T, seg []string, setFlag, clearFlag, want string) {
	t.Helper()

	setIdx := flagIndex(seg, setFlag)
	clearIdx := flagIndex(seg, clearFlag)

	if want == "" {
		require.GreaterOrEqual(t, clearIdx, 0, "expected %s in %v", clearFlag, seg)
		require.Equal(t, -1, setIdx, "unexpected %s in %v", setFlag, seg)

		return
	}

	require.Equal(t, -1, clearIdx, "unexpected %s in %v", clearFlag, seg)
	require.GreaterOrEqual(t, setIdx, 0, "expected %s in %v", setFlag, seg)
	require.Less(t, setIdx+1, len(seg))
	require.Equal(t, want, seg[setIdx+1])
}

func TestEnvSecretFlags_SingleContainer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		env        []serviceinfo.EnvVars
		secrets    []serviceinfo.Secrets
		wantEnv    string
		wantSecret string
	}{
		{name: "both empty clears both"},
		{name: "env only clears secrets", env: oneEnv, wantEnv: "LOG_LEVEL=debug"},
		{name: "secrets only clears env", secrets: oneSecret, wantSecret: "DB_PASSWORD=db-pass:latest"},
		{name: "both set", env: oneEnv, secrets: oneSecret, wantEnv: "LOG_LEVEL=debug", wantSecret: "DB_PASSWORD=db-pass:latest"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := serviceinfo.ServiceInfo{Name: "api", Region: "us-central1", EnvVars: tt.env, Secrets: tt.secrets}

			cmd := gcp.GenerateGCPDeployCommand(svc, "gcr.io/p/api:v1")

			requireEnvSecretFlags(t, cmd, tt.wantEnv, tt.wantSecret)
		})
	}
}

func TestEnvSecretFlags_MultiContainerScopedPerContainer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		svc               serviceinfo.ServiceInfo
		wantIngressEnv    string
		wantIngressSecret string
		wantSidecarEnv    string
		wantSidecarSecret string
	}{
		{
			name: "everything empty clears every container",
			svc: serviceinfo.ServiceInfo{
				Sidecars: []serviceinfo.Sidecar{{Name: "logger", Image: "gcr.io/p/logger:v1"}},
			},
		},
		{
			name: "ingress set, sidecar empty",
			svc: serviceinfo.ServiceInfo{
				EnvVars:  oneEnv,
				Secrets:  oneSecret,
				Sidecars: []serviceinfo.Sidecar{{Name: "logger", Image: "gcr.io/p/logger:v1"}},
			},
			wantIngressEnv:    "LOG_LEVEL=debug",
			wantIngressSecret: "DB_PASSWORD=db-pass:latest",
		},
		{
			name: "ingress empty, sidecar set",
			svc: serviceinfo.ServiceInfo{
				Sidecars: []serviceinfo.Sidecar{{
					Name: "logger", Image: "gcr.io/p/logger:v1", EnvVars: oneEnv, Secrets: oneSecret,
				}},
			},
			wantSidecarEnv:    "LOG_LEVEL=debug",
			wantSidecarSecret: "DB_PASSWORD=db-pass:latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := tt.svc
			svc.Name = "api"
			svc.Region = "us-central1"

			cmd := gcp.GenerateGCPDeployCommand(svc, "gcr.io/p/api:v1")

			indexes := containerIndexes(cmd)
			require.Len(t, indexes, 2)

			// Container flags before the first --container would land on no container in a multi-container deploy.
			requireNoEnvSecretFlags(t, cmd[:indexes[0]])
			requireEnvSecretFlags(t, cmd[indexes[0]:indexes[1]], tt.wantIngressEnv, tt.wantIngressSecret)
			requireEnvSecretFlags(t, cmd[indexes[1]:], tt.wantSidecarEnv, tt.wantSidecarSecret)
		})
	}
}

func requireNoEnvSecretFlags(t *testing.T, seg []string) {
	t.Helper()

	for _, flag := range []string{"--set-env-vars", "--clear-env-vars", "--set-secrets", "--clear-secrets"} {
		require.Equal(t, -1, flagIndex(seg, flag), "unexpected %s before first --container", flag)
	}
}

func TestEnvSecretFlags_Job(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		env        []serviceinfo.EnvVars
		secrets    []serviceinfo.Secrets
		wantEnv    string
		wantSecret string
	}{
		{name: "both empty clears both"},
		{name: "env only clears secrets", env: oneEnv, wantEnv: "LOG_LEVEL=debug"},
		{name: "secrets only clears env", secrets: oneSecret, wantSecret: "DB_PASSWORD=db-pass:latest"},
		{name: "both set", env: oneEnv, secrets: oneSecret, wantEnv: "LOG_LEVEL=debug", wantSecret: "DB_PASSWORD=db-pass:latest"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := serviceinfo.ServiceInfo{
				Name: "migrations", Region: "us-central1", Config: map[string]any{}, EnvVars: tt.env, Secrets: tt.secrets,
			}

			cmd := gcp.GenerateDeployJobCommand(svc, "gcr.io/p/migrations:v1")

			requireEnvSecretFlags(t, cmd, tt.wantEnv, tt.wantSecret)
		})
	}
}
