package output

import (
	"fmt"

	"github.com/sid-technologies/centurion/lib/types"
)

func PrintAddHelp(config types.Config) {
	// Header
	fmt.Printf("\n* Add %s template to your project\n", config.Name)
	fmt.Printf("\nUsage:\n")
	fmt.Printf("	centurion add %s [flags]\n", config.Name)
}

func PrintNextSteps(config types.Config) {
	fmt.Printf("\n* Added %s configuration!\n", config.Name)

	fmt.Printf("\n Files Added:")
	for _, file := range config.Files {
		fmt.Printf("\n    - %s", file.OutputPath)
	}

	fmt.Printf("\nSee README.md in each service for next steps.\n")
}
