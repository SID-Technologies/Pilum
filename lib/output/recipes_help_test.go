package output_test

import (
	"testing"

	"github.com/sid-technologies/pilum/lib/output"

	"github.com/stretchr/testify/require"
)

func TestRecipeListItemFields(t *testing.T) {
	t.Parallel()

	item := output.RecipeListItem{
		Key:         "gcp-cloud-run",
		Description: "Deploy to Google Cloud Run",
	}

	require.Equal(t, "gcp-cloud-run", item.Key)
	require.Equal(t, "Deploy to Google Cloud Run", item.Description)
}

func TestRecipeFieldFields(t *testing.T) {
	t.Parallel()

	field := output.RecipeField{
		Name:        "project",
		Description: "GCP project ID",
		Type:        "string",
		Default:     "my-gcp-project",
	}

	require.Equal(t, "project", field.Name)
	require.Equal(t, "GCP project ID", field.Description)
	require.Equal(t, "string", field.Type)
	require.Equal(t, "my-gcp-project", field.Default)
}

func TestRecipeStepFields(t *testing.T) {
	t.Parallel()

	step := output.RecipeStep{
		Name:    "build binary",
		Tags:    []string{"build"},
		Timeout: 300,
	}

	require.Equal(t, "build binary", step.Name)
	require.Equal(t, []string{"build"}, step.Tags)
	require.Equal(t, 300, step.Timeout)
}

func TestRecipeDetailFields(t *testing.T) {
	t.Parallel()

	detail := output.RecipeDetail{
		Name:        "gcp-cloud-run",
		Description: "Deploy to Google Cloud Run",
		Provider:    "gcp",
		Service:     "cloud-run",
		RequiredFields: []output.RecipeField{
			{Name: "project", Description: "GCP project ID", Type: "string"},
			{Name: "name", Description: "Service name", Type: "string"},
		},
		OptionalFields: []output.RecipeField{
			{Name: "cloud_run.memory", Description: "Memory per instance", Type: "string", Default: "512Mi"},
		},
		Steps: []output.RecipeStep{
			{Name: "build binary", Tags: []string{"build"}, Timeout: 300},
			{Name: "deploy to cloud run", Tags: []string{"deploy"}, Timeout: 180},
		},
	}

	require.Equal(t, "gcp-cloud-run", detail.Name)
	require.Equal(t, "Deploy to Google Cloud Run", detail.Description)
	require.Equal(t, "gcp", detail.Provider)
	require.Equal(t, "cloud-run", detail.Service)
	require.Len(t, detail.RequiredFields, 2)
	require.Len(t, detail.OptionalFields, 1)
	require.Len(t, detail.Steps, 2)
	require.Equal(t, "512Mi", detail.OptionalFields[0].Default)
}

func TestPrintRecipeListSuppressedInJSON(t *testing.T) {
	t.Parallel()

	// Set JSON mode — PrintRecipeList should return without printing
	output.SetMode(output.ModeJSON)
	defer output.SetMode(output.ModeNormal)

	// Should not panic or error
	output.PrintRecipeList([]output.RecipeListItem{
		{Key: "test", Description: "test recipe"},
	})
}

func TestPrintRecipeDetailSuppressedInJSON(t *testing.T) {
	t.Parallel()

	output.SetMode(output.ModeJSON)
	defer output.SetMode(output.ModeNormal)

	// Should not panic or error
	output.PrintRecipeDetail(output.RecipeDetail{
		Name:        "test",
		Description: "test recipe",
	})
}

func TestPrintRecipeListNormal(t *testing.T) {
	t.Parallel()

	// Should not panic — just verifying it runs without error
	output.PrintRecipeList([]output.RecipeListItem{
		{Key: "gcp-cloud-run", Description: "Deploy to Google Cloud Run"},
		{Key: "npm", Description: "Publish npm packages"},
	})
}

func TestPrintRecipeDetailNormal(t *testing.T) {
	t.Parallel()

	// Should not panic — just verifying it runs without error
	output.PrintRecipeDetail(output.RecipeDetail{
		Name:        "gcp-cloud-run",
		Description: "Deploy to Google Cloud Run",
		Provider:    "gcp",
		Service:     "cloud-run",
		RequiredFields: []output.RecipeField{
			{Name: "project", Description: "GCP project ID", Type: "string", Default: "my-gcp-project"},
			{Name: "name", Description: "Service name", Type: "string"},
		},
		OptionalFields: []output.RecipeField{
			{Name: "cloud_run.memory", Description: "Memory per instance", Type: "string", Default: "512Mi"},
		},
		Steps: []output.RecipeStep{
			{Name: "build binary", Tags: []string{"build"}, Timeout: 300},
			{Name: "deploy", Tags: []string{"deploy"}, Timeout: 180},
		},
	})
}

func TestPrintRecipeDetailEmptyFields(t *testing.T) {
	t.Parallel()

	// Should handle empty slices gracefully
	output.PrintRecipeDetail(output.RecipeDetail{
		Name:        "minimal",
		Description: "A minimal recipe",
		Provider:    "test",
	})
}
