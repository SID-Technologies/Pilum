package output

import (
	"fmt"
	"strings"

	"github.com/sid-technologies/centurion/lib/types"
)

func DisplayConfigs(configs []types.Config) {
	var output []string
	category_map := make(map[string]map[types.TemplateType][]types.Config)

	for _, config := range configs {
		if _, exists := category_map[config.Category]; !exists {
			category_map[config.Category] = make(map[types.TemplateType][]types.Config)
		}
		category_map[config.Category][config.Type] = append(category_map[config.Category][config.Type], config)
	}

	for category, type_configs := range category_map {
		stack_output := []string{}
		construct_output := []string{}

		stack_output = append(stack_output, fmt.Sprintf("     %sStacks:%s", Bold, Reset))
		construct_output = append(construct_output, fmt.Sprintf("     %sConstructs:%s", Bold, Reset))

		output = append(output, fmt.Sprintf("   %s%s%s%s", Bold, Purple, category, Reset))
		stacks := type_configs[types.TemplateTypeEnum.Stack]
		construct := type_configs[types.TemplateTypeEnum.Construct]

		for _, stack := range stacks {
			stack_output = append(stack_output, formatConfig(stack)...)
		}
		for _, construct := range construct {
			construct_output = append(construct_output, formatConfig(construct)...)
		}
		if len(stacks) > 0 {
			output = append(output, stack_output...)
		}

		if len(construct) > 0 {
			output = append(output, construct_output...)
		}
		output = append(output, "")
	}

	fmt.Println(strings.Join(output, "\n"))
}

func formatConfig(config types.Config) []string {
	var lines []string

	name_line := fmt.Sprintf("       %s%s%s", Bold, config.Name, Reset)
	if config.Description != "" {
		name_line += fmt.Sprintf(": %s", config.Description)
	}
	lines = append(lines, name_line)

	if len(config.Options) > 0 {
		lines = append(lines, fmt.Sprintf("       %sFlags:%s", Cyan, Reset))
		for _, flag := range config.Options {
			flag_line := fmt.Sprintf("         --%s%s%s", Bold, flag.Flag, Reset)
			if flag.Required {
				flag_line += fmt.Sprintf(" %s(required)%s", Purple, Reset)
			}
			lines = append(lines, flag_line)
		}
	}

	return lines
}
