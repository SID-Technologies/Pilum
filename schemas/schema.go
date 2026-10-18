// Package schemas holds the JSON Schema for pilum.yaml.
package schemas

import _ "embed"

// Service is the JSON Schema for pilum.yaml.
//
//go:embed pilum.yaml.schema.json
var Service []byte
