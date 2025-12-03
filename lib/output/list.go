package output

import (
	"fmt"
	"strings"

	"github.com/sid-technologies/pilum/lib/types"
)

func DisplayConfigs(configs []types.Config) {
	var output []string
	categoryMap := make(map[string]map[types.TemplateType][]types.Config)

	for _, config := range configs {
		if _, exists := categoryMap[config.Category]; !exists {
			categoryMap[config.Category] = make(map[types.TemplateType][]types.Config)
		}
		categoryMap[config.Category][config.Type] = append(categoryMap[config.Category][config.Type], config)
	}

	for category, typeConfigs := range categoryMap {
		stackOutput := []string{}
		constructOutput := []string{}

		stackOutput = append(stackOutput, fmt.Sprintf("     %sStacks:%s", Bold, Reset))
		constructOutput = append(constructOutput, fmt.Sprintf("     %sConstructs:%s", Bold, Reset))

		output = append(output, fmt.Sprintf("   %s%s%s%s", Bold, Purple, category, Reset))
		stacks := typeConfigs[types.TemplateTypeEnum.Stack]
		construct := typeConfigs[types.TemplateTypeEnum.Construct]

		for _, stack := range stacks {
			stackOutput = append(stackOutput, formatConfig(stack)...)
		}
		for _, construct := range construct {
			constructOutput = append(constructOutput, formatConfig(construct)...)
		}
		if len(stacks) > 0 {
			output = append(output, stackOutput...)
		}

		if len(construct) > 0 {
			output = append(output, constructOutput...)
		}
		output = append(output, "")
	}

	fmt.Println(strings.Join(output, "\n"))
}

func formatConfig(config types.Config) []string {
	var lines []string

	nameLine := fmt.Sprintf("       %s%s%s", Bold, config.Name, Reset)
	if config.Description != "" {
		nameLine += fmt.Sprintf(": %s", config.Description)
	}
	lines = append(lines, nameLine)

	if len(config.Options) > 0 {
		lines = append(lines, fmt.Sprintf("       %sFlags:%s", Cyan, Reset))
		for _, flag := range config.Options {
			flagLine := fmt.Sprintf("         --%s%s%s", Bold, flag.Flag, Reset)
			if flag.Required {
				flagLine += fmt.Sprintf(" %s(required)%s", Purple, Reset)
			}
			lines = append(lines, flagLine)
		}
	}

	return lines
}
