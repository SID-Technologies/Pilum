//nolint:revive // types is a common package name for shared type definitions
package types

type TemplateType string

var TemplateTypeEnum = struct {
	Construct TemplateType
	Stack     TemplateType
}{
	Construct: "construct",
	Stack:     "stack",
}
