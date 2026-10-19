package rollback

import (
	"strings"
	"testing"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

// Trimmed from `aws lambda list-versions-by-function` (oldest first, as it emits).
const lambdaVersions = `{"Versions": [
  {"Version": "$LATEST", "Description": ""},
  {"Version": "10", "Description": "pilum v1.0.0"},
  {"Version": "11", "Description": "made by hand"},
  {"Version": "12", "Description": "pilum v1.2.0"},
  {"Version": "13", "Description": "pilum v1.3.0"}
]}`

func lambdaRollbackService() serviceinfo.ServiceInfo {
	return *serviceinfo.NewServiceInfo(map[string]any{
		"name": "fn", "provider": "aws", "type": "aws-lambda", "region": "us-east-1",
	}, "fn")
}

func lambdaResponses(aliasVersion string) map[string]string {
	return map[string]string{
		"list-aliases":              `{"Aliases": [{"Name": "staging", "FunctionVersion": "9"}, {"Name": "live", "FunctionVersion": "` + aliasVersion + `"}]}`,
		"list-versions-by-function": lambdaVersions,
	}
}

func TestPlanLambda(t *testing.T) {
	t.Parallel()

	plan, err := NewPlan(lambdaRollbackService(), "", fakeRunner(lambdaResponses("13")))
	require.NoError(t, err)
	require.Equal(t, KindLambda, plan.Kind)
	require.Equal(t, "live → 13", plan.From)
	require.Equal(t, "live → 12 (v1.2.0)", plan.To)
	require.Equal(t, "aws lambda update-alias --function-name fn --name live --function-version 12 --output json --region us-east-1",
		strings.Join(plan.Command, " "))
	require.Empty(t, plan.Wait)

	// Stepping back again reaches versions Pilum didn't publish too.
	plan, err = NewPlan(lambdaRollbackService(), "", fakeRunner(lambdaResponses("12")))
	require.NoError(t, err)
	require.Equal(t, "live → 11", plan.To)
}

func TestPlanLambdaTo(t *testing.T) {
	t.Parallel()

	tests := []struct{ to, want, err string }{
		{to: "10", want: "live → 10 (v1.0.0)"},
		{to: "v1.0.0", want: "live → 10 (v1.0.0)"},
		{to: "13", err: "version 13 (13) is already live"},
		{to: "v1.3.0", err: "version 13 (v1.3.0) is already live"},
		{to: "9", err: "no published version 9"},
		{to: "v9", err: `no version was deployed with tag "v9"`},
	}
	for _, tt := range tests {
		plan, err := NewPlan(lambdaRollbackService(), tt.to, fakeRunner(lambdaResponses("13")))
		if tt.err != "" {
			require.ErrorContains(t, err, tt.err, tt.to)
			continue
		}
		require.NoError(t, err, tt.to)
		require.Equal(t, tt.want, plan.To, tt.to)
	}
}

func TestPlanLambdaErrors(t *testing.T) {
	t.Parallel()

	_, err := NewPlan(lambdaRollbackService(), "", fakeRunner(lambdaResponses("10")))
	require.ErrorContains(t, err, "no published version before 10")

	_, err = NewPlan(lambdaRollbackService(), "", fakeRunner(lambdaResponses("$LATEST")))
	require.ErrorContains(t, err, "points at $LATEST, not a published version")

	split := map[string]string{
		"list-aliases": `{"Aliases": [{"Name": "live", "FunctionVersion": "13", "RoutingConfig": {"AdditionalVersionWeights": {"12": 0.1}}}]}`,
	}
	_, err = NewPlan(lambdaRollbackService(), "", fakeRunner(split))
	require.ErrorContains(t, err, "splits traffic between versions")

	noAlias := map[string]string{"list-aliases": `{"Aliases": []}`}
	_, err = NewPlan(lambdaRollbackService(), "", fakeRunner(noAlias))
	require.ErrorContains(t, err, "has no alias live to roll back")
}
