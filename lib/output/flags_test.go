package output_test

import (
	"testing"

	"github.com/sid-technologies/pilum/lib/output"
	"github.com/sid-technologies/pilum/lib/types"
	"github.com/stretchr/testify/require"
)

func TestPrintFlags(t *testing.T) {
	t.Parallel()

	flags := []types.FlagArg{
		{
			Name:        "url",
			Flag:        "base-url",
			Type:        "string",
			Default:     "example.com",
			Required:    true,
			Description: "The base URL",
		},
		{
			Name:        "port",
			Flag:        "port",
			Type:        "int",
			Default:     "8080",
			Required:    false,
			Description: "The port number",
		},
	}

	require.NotPanics(t, func() {
		output.PrintFlags(flags)
	})
}

func TestPrintFlagsEmpty(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		output.PrintFlags([]types.FlagArg{})
	})
}

func TestPrintFlagsNoDescription(t *testing.T) {
	t.Parallel()

	flags := []types.FlagArg{
		{
			Name:     "url",
			Flag:     "base-url",
			Required: true,
		},
	}

	require.NotPanics(t, func() {
		output.PrintFlags(flags)
	})
}

func TestDisplayMissingFlags(t *testing.T) {
	t.Parallel()

	missingFlags := []types.FlagArg{
		{
			Name:        "url",
			Flag:        "base-url",
			Required:    true,
			Description: "The base URL is required",
		},
		{
			Name:     "token",
			Flag:     "api-token",
			Required: true,
		},
	}

	require.NotPanics(t, func() {
		output.DisplayMissingFlags(missingFlags)
	})
}

func TestDisplayMissingFlagsEmpty(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		output.DisplayMissingFlags([]types.FlagArg{})
	})
}

func TestDisplayMissingFlagsNoDescription(t *testing.T) {
	t.Parallel()

	missingFlags := []types.FlagArg{
		{
			Name:     "url",
			Flag:     "base-url",
			Required: true,
		},
	}

	require.NotPanics(t, func() {
		output.DisplayMissingFlags(missingFlags)
	})
}
