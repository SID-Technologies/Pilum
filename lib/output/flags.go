package output

import (
	"fmt"
	"strings"

	"github.com/sid-technologies/centurion/lib/types"
)

func PrintFlags(flags []types.FlagArg) {
	var output []string
	output = append(output, fmt.Sprintf("  %sAvailable Flags:%s", Bold, Reset))
	output = append(output, fmt.Sprintf("    %sDirectory to add the template to%s", LightBlue, Reset))
	output = append(output, fmt.Sprintf("    %sDefault: \".\"%s", Cyan, Reset))

	// flag details
	for _, flag := range flags {
		flagLine := fmt.Sprintf("    %s--%s %s%s", Bold, flag.Name, flag.Flag, Reset)
		if flag.Required {
			flagLine += fmt.Sprintf(" %s(required)%s", Purple, Reset)
		}
		output = append(output, flagLine)

		if flag.Description != "" {
			output = append(output, fmt.Sprintf("        %s%s%s", Cyan, flag.Description, Reset))
		}
		valueType := "default"
		if flag.Required {
			valueType = "required"
		}

		output = append(output, fmt.Sprintf("        %s%s: %v%s", Cyan, valueType, flag.Default, Reset))
		output = append(output, "") // empty line for spacing
	}

	fmt.Println(strings.Join(output, "\n"))
}

func DisplayMissingFlags(missingFlags []types.FlagArg) {
	var output []string
	output = append(output, fmt.Sprintf("%sError:%s Missing required flags", Red, Reset))

	for _, flag := range missingFlags {
		flagLine := fmt.Sprintf("  %s--%s %s%s", Bold, flag.Name, flag.Flag, Reset)
		if flag.Description != "" {
			flagLine += fmt.Sprintf(" - %s%s%s", Cyan, flag.Description, Reset)
		}
		output = append(output, flagLine)
	}

	fmt.Println(strings.Join(output, "\n"))
}
