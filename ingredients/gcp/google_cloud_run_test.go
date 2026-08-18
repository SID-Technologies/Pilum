package gcp_test

import (
	"strings"
	"testing"

	"github.com/sid-technologies/pilum/ingredients/gcp"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

func TestGenerateGCPDeployCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		service       serviceinfo.ServiceInfo
		imageName     string
		minLen        int
		expectedFirst string
		containsFlag  string
		containsValue string
	}{
		{
			name: "basic deploy command",
			service: serviceinfo.ServiceInfo{
				Name:   "myservice",
				Region: "us-central1",
				Config: map[string]any{}, // Empty config means no cloud_run settings
			},
			imageName:     "gcr.io/project/myservice:latest",
			minLen:        11,
			expectedFirst: "gcloud",
			containsFlag:  "--region",
			containsValue: "us-central1",
		},
		{
			name: "deploy with different region",
			service: serviceinfo.ServiceInfo{
				Name:   "api-service",
				Region: "europe-west1",
				Config: map[string]any{}, // Empty config means no cloud_run settings
			},
			imageName:     "gcr.io/project/api-service:v1.0.0",
			minLen:        11,
			expectedFirst: "gcloud",
			containsFlag:  "--region",
			containsValue: "europe-west1",
		},
		{
			name: "deploy with project",
			service: serviceinfo.ServiceInfo{
				Name:    "myservice",
				Region:  "us-central1",
				Project: "my-project",
			},
			imageName:     "gcr.io/project/myservice:latest",
			minLen:        13, // 11 + 2 for --project value
			expectedFirst: "gcloud",
			containsFlag:  "--project",
			containsValue: "my-project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := gcp.GenerateGCPDeployCommand(tt.service, tt.imageName)

			require.GreaterOrEqual(t, len(cmd), tt.minLen)
			require.Equal(t, tt.expectedFirst, cmd[0])
			require.Equal(t, "run", cmd[1])
			require.Equal(t, "deploy", cmd[2])
			require.Equal(t, tt.service.Name, cmd[3])

			// Check flag and value are present
			foundFlag := false
			for i, arg := range cmd {
				if arg == tt.containsFlag && i+1 < len(cmd) {
					require.Equal(t, tt.containsValue, cmd[i+1])
					foundFlag = true
					break
				}
			}
			require.True(t, foundFlag, "expected flag %s not found", tt.containsFlag)
		})
	}
}

func TestGenerateGCPDeployCommandWithSecrets(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:   "myservice",
		Region: "us-central1",
		Config: map[string]any{},
		Secrets: []serviceinfo.Secrets{
			{Name: "DB_PASSWORD", Value: "projects/123/secrets/db-pass:latest"},
			{Name: "API_KEY", Value: "projects/123/secrets/api-key:latest"},
		},
	}

	cmd := gcp.GenerateGCPDeployCommand(service, "gcr.io/project/myservice:latest")

	// Should have --set-secrets flag
	foundSecrets := false
	for i, arg := range cmd {
		if arg == "--set-secrets" && i+1 < len(cmd) {
			secretsValue := cmd[i+1]
			require.Contains(t, secretsValue, "DB_PASSWORD=")
			require.Contains(t, secretsValue, "API_KEY=")
			require.Contains(t, secretsValue, ",") // Multiple secrets joined
			foundSecrets = true
			break
		}
	}
	require.True(t, foundSecrets, "--set-secrets flag not found")
}

func TestGenerateGCPDeployCommandNoSecrets(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:    "myservice",
		Region:  "us-central1",
		Config:  map[string]any{},
		Secrets: []serviceinfo.Secrets{},
	}

	cmd := gcp.GenerateGCPDeployCommand(service, "gcr.io/project/myservice:latest")

	// Should NOT have --set-secrets flag
	for _, arg := range cmd {
		require.NotEqual(t, "--set-secrets", arg)
	}
}

func TestGenerateGCPDeployCommandImageName(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:   "myservice",
		Region: "us-central1",
		Config: map[string]any{},
	}

	imageName := "us-central1-docker.pkg.dev/my-project/repo/myservice:v2.0.0"
	cmd := gcp.GenerateGCPDeployCommand(service, imageName)

	// Check --image flag
	foundImage := false
	for i, arg := range cmd {
		if arg == "--image" && i+1 < len(cmd) {
			require.Equal(t, imageName, cmd[i+1])
			foundImage = true
			break
		}
	}
	require.True(t, foundImage, "--image flag not found")
}

func TestGenerateGCPDeployCommandPlatformManaged(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:   "myservice",
		Region: "us-central1",
		Config: map[string]any{},
	}

	cmd := gcp.GenerateGCPDeployCommand(service, "gcr.io/project/myservice:latest")

	// Should have --platform managed
	foundPlatform := false
	for i, arg := range cmd {
		if arg == "--platform" && i+1 < len(cmd) {
			require.Equal(t, "managed", cmd[i+1])
			foundPlatform = true
			break
		}
	}
	require.True(t, foundPlatform, "--platform flag not found")
}

func TestGenerateGCPDeployCommandAllowUnauthenticated(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:   "myservice",
		Region: "us-central1",
		Config: map[string]any{},
	}

	cmd := gcp.GenerateGCPDeployCommand(service, "gcr.io/project/myservice:latest")

	// A service that does not ask to be public is deployed private. Exposing
	// every service to the internet by default gets that wrong silently, and
	// nothing in the deploy output would say so.
	found := false
	for _, arg := range cmd {
		if arg == "--no-allow-unauthenticated" {
			found = true
			break
		}
	}
	require.True(t, found, "a service with no cloud_run config must deploy private")
}

func TestGenerateGCPDeployCommandSingleSecret(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:   "myservice",
		Region: "us-central1",
		Config: map[string]any{},
		Secrets: []serviceinfo.Secrets{
			{Name: "API_KEY", Value: "projects/123/secrets/key:latest"},
		},
	}

	cmd := gcp.GenerateGCPDeployCommand(service, "gcr.io/project/myservice:latest")

	// Should have --set-secrets flag — already in gcloud format, pass through
	for i, arg := range cmd {
		if arg == "--set-secrets" && i+1 < len(cmd) {
			secretsValue := cmd[i+1]
			require.Equal(t, "API_KEY=projects/123/secrets/key:latest", secretsValue)
			return
		}
	}
	t.Fatal("--set-secrets flag not found")
}

func TestGenerateGCPDeployCommandSecretResourcePath(t *testing.T) {
	t.Parallel()

	// Full resource path format from pilum.yaml (projects/PROJECT/secrets/NAME/versions/VERSION)
	service := serviceinfo.ServiceInfo{
		Name:   "myservice",
		Region: "us-central1",
		Config: map[string]any{},
		Secrets: []serviceinfo.Secrets{
			{Name: "DATABASE_URL", Value: "projects/romans-dev/secrets/platform-database-url/versions/latest"},
		},
	}

	cmd := gcp.GenerateGCPDeployCommand(service, "gcr.io/project/myservice:latest")

	// Should convert to gcloud format: SECRET_NAME:VERSION
	for i, arg := range cmd {
		if arg == "--set-secrets" && i+1 < len(cmd) {
			secretsValue := cmd[i+1]
			require.Equal(t, "DATABASE_URL=platform-database-url:latest", secretsValue)
			return
		}
	}
	t.Fatal("--set-secrets flag not found")
}

func TestGenerateGCPDeployCommandCloudRunConfig(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:    "myservice",
		Region:  "us-central1",
		Project: "my-project",
		Config: map[string]any{
			"cloud_run": map[string]any{
				"min_instances":   0,
				"max_instances":   10,
				"cpu_throttling":  true,
				"memory":          "2048Mi",
				"cpu":             "2",
				"concurrency":     80,
				"timeout_seconds": 300,
			},
		},
	}

	cmd := gcp.GenerateGCPDeployCommand(service, "gcr.io/project/myservice:latest")

	// Check all Cloud Run config flags are present
	cmdStr := ""
	for _, arg := range cmd {
		cmdStr += arg + " "
	}

	require.Contains(t, cmdStr, "--min-instances=0")
	require.Contains(t, cmdStr, "--max-instances=10")
	require.Contains(t, cmdStr, "--cpu-throttling")
	require.Contains(t, cmdStr, "--memory 2048Mi")
	require.Contains(t, cmdStr, "--cpu 2")
	require.Contains(t, cmdStr, "--concurrency=80")
	require.Contains(t, cmdStr, "--timeout=300")
	require.Contains(t, cmdStr, "--project my-project")
}

func TestGenerateGCPDeployCommandCPUThrottlingDisabled(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:   "myservice",
		Region: "us-central1",
		Config: map[string]any{
			"cloud_run": map[string]any{
				"cpu_throttling": false,
			},
		},
	}

	cmd := gcp.GenerateGCPDeployCommand(service, "gcr.io/project/myservice:latest")

	// Should NOT have --cpu-throttling when set to false
	for _, arg := range cmd {
		require.NotEqual(t, "--cpu-throttling", arg)
	}
}

func TestGenerateGCPDeployCommandCloudSQLInstances(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:    "myservice",
		Region:  "us-central1",
		Project: "my-project",
		Config: map[string]any{
			"cloud_run": map[string]any{
				"cloudsql_instances": []any{
					"my-project:us-central1:my-db",
				},
			},
		},
	}

	cmd := gcp.GenerateGCPDeployCommand(service, "gcr.io/project/myservice:latest")

	cmdStr := ""
	for _, arg := range cmd {
		cmdStr += arg + " "
	}

	require.Contains(t, cmdStr, "--add-cloudsql-instances my-project:us-central1:my-db")
}

func TestGenerateGCPDeployCommandMultipleCloudSQLInstances(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:   "myservice",
		Region: "us-central1",
		Config: map[string]any{
			"cloud_run": map[string]any{
				"cloudsql_instances": []any{
					"proj:us-central1:db-one",
					"proj:us-central1:db-two",
				},
			},
		},
	}

	cmd := gcp.GenerateGCPDeployCommand(service, "gcr.io/project/myservice:latest")

	cmdStr := ""
	for _, arg := range cmd {
		cmdStr += arg + " "
	}

	require.Contains(t, cmdStr, "--add-cloudsql-instances proj:us-central1:db-one,proj:us-central1:db-two")
}

func TestGenerateGCPDeployCommandNoCloudSQLInstances(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:   "myservice",
		Region: "us-central1",
		Config: map[string]any{
			"cloud_run": map[string]any{
				"memory": "1Gi",
			},
		},
	}

	cmd := gcp.GenerateGCPDeployCommand(service, "gcr.io/project/myservice:latest")

	for _, arg := range cmd {
		require.NotEqual(t, "--add-cloudsql-instances", arg)
	}
}

func TestGenerateGCPDeployCommandWithEnvVars(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:   "myservice",
		Region: "us-central1",
		Config: map[string]any{},
		EnvVars: []serviceinfo.EnvVars{
			{Name: "PLATFORM_ENV", Value: "production"},
			{Name: "API_URL", Value: "https://api.example.com"},
		},
	}

	cmd := gcp.GenerateGCPDeployCommand(service, "gcr.io/project/myservice:latest")

	// Should have --set-env-vars flag
	foundEnvVars := false
	for i, arg := range cmd {
		if arg == "--set-env-vars" && i+1 < len(cmd) {
			envValue := cmd[i+1]
			require.Contains(t, envValue, "PLATFORM_ENV=production")
			require.Contains(t, envValue, "API_URL=https://api.example.com")
			require.Contains(t, envValue, ",") // Multiple env vars joined
			foundEnvVars = true
			break
		}
	}
	require.True(t, foundEnvVars, "--set-env-vars flag not found")
}

func TestGenerateGCPDeployCommandNoEnvVars(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:    "myservice",
		Region:  "us-central1",
		Config:  map[string]any{},
		EnvVars: []serviceinfo.EnvVars{},
	}

	cmd := gcp.GenerateGCPDeployCommand(service, "gcr.io/project/myservice:latest")

	// Should NOT have --set-env-vars flag
	for _, arg := range cmd {
		require.NotEqual(t, "--set-env-vars", arg)
	}
}

func TestGenerateGCPDeployCommandSingleEnvVar(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:   "myservice",
		Region: "us-central1",
		Config: map[string]any{},
		EnvVars: []serviceinfo.EnvVars{
			{Name: "APP_ENV", Value: "staging"},
		},
	}

	cmd := gcp.GenerateGCPDeployCommand(service, "gcr.io/project/myservice:latest")

	for i, arg := range cmd {
		if arg == "--set-env-vars" && i+1 < len(cmd) {
			require.Equal(t, "APP_ENV=staging", cmd[i+1])
			return
		}
	}
	t.Fatal("--set-env-vars flag not found")
}

func TestGenerateGCPDeployCommandEmptyCloudRunConfig(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:   "myservice",
		Region: "us-central1",
		Config: map[string]any{}, // No cloud_run section
	}

	cmd := gcp.GenerateGCPDeployCommand(service, "gcr.io/project/myservice:latest")

	// Should not have any Cloud Run specific flags when config is empty
	for _, arg := range cmd {
		require.NotContains(t, arg, "--min-instances")
		require.NotContains(t, arg, "--max-instances")
		require.NotEqual(t, "--cpu-throttling", arg)
		require.NotEqual(t, "--memory", arg)
		require.NotEqual(t, "--cpu", arg)
		require.NotContains(t, arg, "--concurrency")
		require.NotContains(t, arg, "--timeout")
	}
}

// TestGenerateGCPDeployCommandEnvVarWithComma locks in that when an env value
// contains a comma, the recipe switches to gcloud's ^DELIM^ alternative-
// delimiter syntax. The default --set-env-vars parser treats commas as KEY=VAL
// pair separators, so a comma in any value breaks parsing — caught in prod
// when a CORS allow-list env var (multi-origin CSV) was rejected.
func TestGenerateGCPDeployCommandEnvVarWithComma(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:   "myservice",
		Region: "us-central1",
		Config: map[string]any{},
		EnvVars: []serviceinfo.EnvVars{
			{Name: "ALLOWED_ORIGINS", Value: "https://a.com,https://b.com,https://c.com"},
			{Name: "OTHER", Value: "plain"},
		},
	}

	cmd := gcp.GenerateGCPDeployCommand(service, "gcr.io/project/myservice:latest")

	for i, arg := range cmd {
		if arg == "--set-env-vars" && i+1 < len(cmd) {
			val := cmd[i+1]
			// Must use alt-delimiter form ^X^…
			require.True(t, strings.HasPrefix(val, "^"), "expected ^DELIM^ prefix; got %q", val)
			// Must contain both env vars in the picked-delimiter form
			require.Contains(t, val, "ALLOWED_ORIGINS=https://a.com,https://b.com,https://c.com")
			require.Contains(t, val, "OTHER=plain")
			return
		}
	}

	t.Fatal("--set-env-vars flag not found")
}

// TestGenerateGCPDeployCommandEnvVarWithoutComma ensures the alt-delimiter
// form is NOT used when no value contains a comma — keeps emitted commands
// readable in the common case and preserves backward compatibility.
func TestGenerateGCPDeployCommandEnvVarWithoutComma(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:   "myservice",
		Region: "us-central1",
		Config: map[string]any{},
		EnvVars: []serviceinfo.EnvVars{
			{Name: "FOO", Value: "bar"},
			{Name: "BAZ", Value: "qux"},
		},
	}

	cmd := gcp.GenerateGCPDeployCommand(service, "gcr.io/project/myservice:latest")

	for i, arg := range cmd {
		if arg == "--set-env-vars" && i+1 < len(cmd) {
			val := cmd[i+1]
			require.False(t, strings.HasPrefix(val, "^"), "should not use ^DELIM^ form when no commas in values; got %q", val)
			require.Equal(t, "FOO=bar,BAZ=qux", val)
			return
		}
	}

	t.Fatal("--set-env-vars flag not found")
}

// TestGenerateGCPDeployCommandEnvVarDelimiterCollision covers the case where
// the first-choice delimiter (`|`) appears in a value — the picker must fall
// through to a non-conflicting candidate (`@`).
func TestGenerateGCPDeployCommandEnvVarDelimiterCollision(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:   "myservice",
		Region: "us-central1",
		Config: map[string]any{},
		EnvVars: []serviceinfo.EnvVars{
			{Name: "REGEX", Value: "foo|bar|baz"}, // contains |
			{Name: "LIST", Value: "a,b,c"},        // forces alt-delimiter
		},
	}

	cmd := gcp.GenerateGCPDeployCommand(service, "gcr.io/project/myservice:latest")

	for i, arg := range cmd {
		if arg == "--set-env-vars" && i+1 < len(cmd) {
			val := cmd[i+1]
			require.True(t, strings.HasPrefix(val, "^"), "expected ^DELIM^ prefix")
			require.False(t, strings.HasPrefix(val, "^|^"), "must not pick `|` since values contain it; got %q", val)
			require.Contains(t, val, "REGEX=foo|bar|baz")
			require.Contains(t, val, "LIST=a,b,c")
			return
		}
	}

	t.Fatal("--set-env-vars flag not found")
}

// Secure by default: reaching the internet is a property a service must ask
// for, so that a config which says nothing cannot silently publish one.
func TestGenerateGCPDeployCommand_DefaultsToPrivate(t *testing.T) {
	svc := serviceinfo.ServiceInfo{Name: "svc", Region: "us-central1"}

	cmd := gcp.GenerateGCPDeployCommand(svc, "img")

	if !containsFlag(cmd, "--no-allow-unauthenticated") {
		t.Error("a service that does not opt in must deploy private")
	}

	if containsFlag(cmd, "--allow-unauthenticated") {
		t.Error("must not expose a service that never asked to be public")
	}
}

// Edge services still opt in explicitly.
func TestGenerateGCPDeployCommand_OptsIntoPublic(t *testing.T) {
	svc := serviceinfo.ServiceInfo{
		Name:   "svc",
		Region: "us-central1",
		Config: map[string]any{
			"cloud_run": map[string]any{"allow_unauthenticated": true},
		},
	}

	cmd := gcp.GenerateGCPDeployCommand(svc, "img")

	if !containsFlag(cmd, "--allow-unauthenticated") {
		t.Error("allow_unauthenticated:true must expose the service")
	}
}

// A deploy must be able to REMOVE public access, not merely decline to add it:
// omitting the flag leaves the existing IAM policy untouched, so a service
// locked down by hand was silently made public again on the next deploy.
func TestGenerateGCPDeployCommand_CanLockDown(t *testing.T) {
	svc := serviceinfo.ServiceInfo{
		Name:   "svc",
		Region: "us-central1",
		Config: map[string]any{
			"cloud_run": map[string]any{"allow_unauthenticated": false},
		},
	}

	cmd := gcp.GenerateGCPDeployCommand(svc, "img")

	if !containsFlag(cmd, "--no-allow-unauthenticated") {
		t.Error("allow_unauthenticated:false must emit --no-allow-unauthenticated")
	}

	if containsFlag(cmd, "--allow-unauthenticated") {
		t.Error("must not also pass the public flag")
	}
}

func containsFlag(args []string, want string) bool {
	for _, a := range args {
		if a == want {
			return true
		}
	}

	return false
}
