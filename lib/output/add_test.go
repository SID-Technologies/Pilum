package output_test

import (
	"testing"

	"github.com/sid-technologies/pilum/lib/output"
	"github.com/sid-technologies/pilum/lib/types"

	"github.com/stretchr/testify/require"
)

func TestPrintAddHelp(t *testing.T) {
	t.Parallel()

	config := types.Config{
		Name:        "test-template",
		Description: "A test template",
	}

	require.NotPanics(t, func() {
		output.PrintAddHelp(config)
	})
}

func TestPrintAddHelpEmpty(t *testing.T) {
	t.Parallel()

	config := types.Config{}

	require.NotPanics(t, func() {
		output.PrintAddHelp(config)
	})
}

func TestPrintNextSteps(t *testing.T) {
	t.Parallel()

	config := types.Config{
		Name: "test-template",
		Files: []types.ConfigFile{
			{Path: "src/file1.ts", OutputPath: "output/file1.ts"},
			{Path: "src/file2.ts", OutputPath: "output/file2.ts"},
		},
	}

	require.NotPanics(t, func() {
		output.PrintNextSteps(config)
	})
}

func TestPrintNextStepsNoFiles(t *testing.T) {
	t.Parallel()

	config := types.Config{
		Name:  "test-template",
		Files: []types.ConfigFile{},
	}

	require.NotPanics(t, func() {
		output.PrintNextSteps(config)
	})
}
