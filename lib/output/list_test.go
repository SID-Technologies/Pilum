package output_test

import (
	"testing"

	"github.com/sid-technologies/pilum/lib/output"
	"github.com/sid-technologies/pilum/lib/types"
	"github.com/stretchr/testify/require"
)

func TestDisplayConfigs(t *testing.T) {
	t.Parallel()

	configs := []types.Config{
		{
			Name:        "web-stack",
			Type:        types.TemplateTypeEnum.Stack,
			Description: "A web application stack",
			Category:    "web",
			Options: []types.FlagArg{
				{Name: "url", Flag: "base-url", Required: true},
			},
		},
		{
			Name:        "api-construct",
			Type:        types.TemplateTypeEnum.Construct,
			Description: "An API construct",
			Category:    "web",
			Options: []types.FlagArg{
				{Name: "port", Flag: "port", Required: false},
			},
		},
		{
			Name:        "db-stack",
			Type:        types.TemplateTypeEnum.Stack,
			Description: "Database stack",
			Category:    "database",
		},
	}

	require.NotPanics(t, func() {
		output.DisplayConfigs(configs)
	})
}

func TestDisplayConfigsEmpty(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		output.DisplayConfigs([]types.Config{})
	})
}

func TestDisplayConfigsSingleCategory(t *testing.T) {
	t.Parallel()

	configs := []types.Config{
		{
			Name:        "template1",
			Type:        types.TemplateTypeEnum.Stack,
			Description: "First template",
			Category:    "category1",
		},
	}

	require.NotPanics(t, func() {
		output.DisplayConfigs(configs)
	})
}

func TestDisplayConfigsOnlyConstructs(t *testing.T) {
	t.Parallel()

	configs := []types.Config{
		{
			Name:        "construct1",
			Type:        types.TemplateTypeEnum.Construct,
			Description: "First construct",
			Category:    "category1",
		},
		{
			Name:     "construct2",
			Type:     types.TemplateTypeEnum.Construct,
			Category: "category1",
		},
	}

	require.NotPanics(t, func() {
		output.DisplayConfigs(configs)
	})
}

func TestDisplayConfigsNoDescription(t *testing.T) {
	t.Parallel()

	configs := []types.Config{
		{
			Name:     "stack1",
			Type:     types.TemplateTypeEnum.Stack,
			Category: "category1",
		},
	}

	require.NotPanics(t, func() {
		output.DisplayConfigs(configs)
	})
}

func TestDisplayConfigsWithOptions(t *testing.T) {
	t.Parallel()

	configs := []types.Config{
		{
			Name:        "config1",
			Type:        types.TemplateTypeEnum.Stack,
			Description: "Config with options",
			Category:    "category1",
			Options: []types.FlagArg{
				{Name: "opt1", Flag: "option1", Required: true},
				{Name: "opt2", Flag: "option2", Required: false},
			},
		},
	}

	require.NotPanics(t, func() {
		output.DisplayConfigs(configs)
	})
}
