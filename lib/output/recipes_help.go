package output

import "fmt"

// RecipeListItem holds the data needed to display a recipe in the list view.
type RecipeListItem struct {
	Key         string `json:"name"`
	Description string `json:"description"`
}

// RecipeField holds field data for the detail view.
type RecipeField struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Default     string `json:"default,omitempty"`
}

// RecipeStep holds step data for the detail view.
type RecipeStep struct {
	Name    string   `json:"name"`
	Tags    []string `json:"tags,omitempty"`
	Timeout int      `json:"timeout,omitempty"`
}

// RecipeDetail holds all data needed for the detail view.
type RecipeDetail struct {
	Name           string        `json:"name"`
	Description    string        `json:"description"`
	Provider       string        `json:"provider"`
	Service        string        `json:"service,omitempty"`
	RequiredFields []RecipeField `json:"required_fields,omitempty"`
	OptionalFields []RecipeField `json:"optional_fields,omitempty"`
	Steps          []RecipeStep  `json:"steps,omitempty"`
}

// PrintRecipeList prints all available recipes in a styled list.
// Suppressed in JSON mode.
func PrintRecipeList(recipes []RecipeListItem) {
	if IsJSON() {
		return
	}
	fmt.Printf("\n%s%sAvailable Recipes%s\n\n", Bold, Primary, Reset)

	// Find max key length for alignment
	maxLen := 0
	for _, r := range recipes {
		if len(r.Key) > maxLen {
			maxLen = len(r.Key)
		}
	}

	for _, r := range recipes {
		fmt.Printf("  %s%-*s%s  %s%s%s\n", Bold, maxLen, r.Key, Reset, Muted, r.Description, Reset)
	}

	fmt.Printf("\n  %sRun %spilum recipes <name>%s%s for details on a specific recipe.%s\n\n", Muted, Accent, Reset, Muted, Reset)
}

// PrintRecipeDetail prints a detailed view of a single recipe.
// Suppressed in JSON mode.
func PrintRecipeDetail(detail RecipeDetail) {
	if IsJSON() {
		return
	}
	fmt.Printf("\n%s%s%s%s\n", Bold, Primary, detail.Name, Reset)
	fmt.Printf("  %s%s%s\n\n", Muted, detail.Description, Reset)

	// Provider info
	provider := detail.Provider
	if detail.Service != "" {
		provider += " / " + detail.Service
	}
	fmt.Printf("  %sProvider:%s  %s\n\n", Bold, Reset, provider)

	// Required fields
	if len(detail.RequiredFields) > 0 {
		fmt.Printf("%s%sRequired Fields%s\n\n", Bold, Primary, Reset)
		printFields(detail.RequiredFields)
	}

	// Optional fields
	if len(detail.OptionalFields) > 0 {
		fmt.Printf("%s%sOptional Fields%s\n\n", Bold, Primary, Reset)
		printFields(detail.OptionalFields)
	}

	// Steps
	if len(detail.Steps) > 0 {
		fmt.Printf("%s%sSteps%s\n\n", Bold, Primary, Reset)
		for i, step := range detail.Steps {
			tags := ""
			if len(step.Tags) > 0 {
				tags = fmt.Sprintf("  %s[%s]%s", Muted, joinStrings(step.Tags, ", "), Reset)
			}
			timeout := ""
			if step.Timeout > 0 {
				timeout = fmt.Sprintf("  %stimeout: %ds%s", Muted, step.Timeout, Reset)
			}
			fmt.Printf("  %s%d.%s %s%s%s\n", Muted, i+1, Reset, step.Name, tags, timeout)
		}
		fmt.Println()
	}
}

func printFields(fields []RecipeField) {
	// Find max name length for alignment
	maxLen := 0
	for _, f := range fields {
		if len(f.Name) > maxLen {
			maxLen = len(f.Name)
		}
	}

	for _, f := range fields {
		def := ""
		if f.Default != "" {
			def = fmt.Sprintf("  %s(default: %s)%s", Muted, f.Default, Reset)
		}
		fmt.Printf("  %s%-*s%s  %s%s\n", Accent, maxLen, f.Name, Reset, f.Description, def)
	}
	fmt.Println()
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for _, s := range strs[1:] {
		result += sep + s
	}
	return result
}
