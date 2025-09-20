package types

/*
name = "example"
type = "Ingredient"
description = "descibe me"
category = "cat"

[metadata]
path = "path/path"
version = "1.0.0"
author = "Dan Flanagan"
tags = ["example"]

[[options]]
name = "URL"
flag = "base-url"
type = "string"
default = "example.com"
required = true
description = "domain name"

[[files]]
path = "utils/example.ts.tmpl"
output_path = "construct/utils/example.ts"

[documentation]
readme = """
# {{.Name}}

HELLO WORLD

"""
*/

type ConfigMetaData struct {
	Path    string   `toml:"path"`
	Version string   `toml:"version"`
	Author  string   `toml:"author"`
	Tags    []string `toml:"tags"`
}

type FlagArg struct {
	Name        string `toml:"name"`
	Flag        string `toml:"flag"`
	Type        string `toml:"type"` // string, int, float, bool
	Default     string `toml:"default"`
	Required    bool   `toml:"required"`
	Description string `toml:"description"`
}

type ConfigFile struct {
	Path       string `toml:"path"`
	OutputPath string `toml:"output_path"`
	IsOptional bool   `toml:"optional"`
	Condition  string `toml:"condition"`
}

type Documentation struct {
	Readme string `toml:"readme"`
}

type Config struct {
	Name        string       `toml:"name"`
	Type        TemplateType `toml:"type"`
	Description string       `toml:"description"`
	Category    string       `toml:"category"`

	Metadata ConfigMetaData `toml:"metadata"`

	Options []FlagArg    `toml:"options"`
	Files   []ConfigFile `toml:"files"`

	Documentation Documentation `toml:"documentation"`
}
