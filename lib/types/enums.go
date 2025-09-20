package types

type TemplateType string

var TemplateTypeEnum = struct {
	Construct TemplateType
	Stack     TemplateType
}{
	Construct: "construct",
	Stack:     "stack",
}
