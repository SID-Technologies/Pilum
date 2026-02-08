package output

import (
	"encoding/json"
	"fmt"
)

// JSON marshals v as indented JSON and prints to stdout.
func JSON(v any) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
}
