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
		flag_line := fmt.Sprintf("    %s--%s %s%s", Bold, flag.Name, flag.Flag, Reset)
		if flag.Required {
			flag_line += fmt.Sprintf(" %s(required)%s", Purple, Reset)
		}
		output = append(output, flag_line)

		if flag.Description != "" {
			output = append(output, fmt.Sprintf("        %s%s%s", Cyan, flag.Description, Reset))
		}
		value_type := "default"
		if flag.Required {
			value_type = "required"
		}

		output = append(output, fmt.Sprintf("        %s%s: %v%s", Cyan, value_type, flag.Default, Reset))
		output = append(output, "") // empty line for spacing
	}

	fmt.Println(strings.Join(output, "\n"))
}

func DisplayMissingFlags(missing_flags []types.FlagArg) {
	var output []string
	output = append(output, fmt.Sprintf("%sError:%s Missing required flags", Red, Reset))

	for _, flag := range missing_flags {
		flag_line := fmt.Sprintf("  %s--%s %s%s", Bold, flag.Name, flag.Flag, Reset)
		if flag.Description != "" {
			flag_line += fmt.Sprintf(" - %s%s%s", Cyan, flag.Description, Reset)
		}
		output = append(output, flag_line)
	}

	fmt.Println(strings.Join(output, "\n"))
}
