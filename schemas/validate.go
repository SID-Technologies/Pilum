package schemas

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/suggest"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const serviceURL = "pilum.yaml.schema.json"

// Problem is one place a pilum.yaml doesn't match the schema.
type Problem struct {
	// Path is the dotted key, e.g. "cloud_run.min_instances". Empty is the
	// top level.
	Path    string
	Message string
	// Warning is true for values Pilum still reads, such as a number where
	// the schema wants a string.
	Warning bool
}

func (p Problem) String() string {
	if p.Path == "" {
		return p.Message
	}
	return p.Path + ": " + p.Message
}

type compiled struct {
	schema *jsonschema.Schema
	raw    any
}

var load = sync.OnceValues(func() (compiled, error) {
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(Service))
	if err != nil {
		return compiled{}, errors.Wrap(err, "error reading the pilum.yaml schema")
	}
	c := jsonschema.NewCompiler()
	err = c.AddResource(serviceURL, doc)
	if err != nil {
		return compiled{}, errors.Wrap(err, "error loading the pilum.yaml schema")
	}
	schema, err := c.Compile(serviceURL)
	if err != nil {
		return compiled{}, errors.Wrap(err, "error compiling the pilum.yaml schema")
	}
	return compiled{schema: schema, raw: doc}, nil
})

// TopLevelKeys returns the keys the schema defines at the top level of
// pilum.yaml.
func TopLevelKeys() ([]string, error) {
	c, err := load()
	if err != nil {
		return nil, err
	}
	return propertyNames(c.raw), nil
}

// Validate checks a parsed pilum.yaml against the schema. Unknown keys at the
// top level are allowed (custom recipes add their own); unknown keys inside a
// block such as cloud_run are not.
func Validate(config map[string]any) ([]Problem, error) {
	c, err := load()
	if err != nil {
		return nil, err
	}

	// Round-trip through JSON so YAML's Go types become the JSON types the
	// validator expects.
	data, err := json.Marshal(config)
	if err != nil {
		return nil, errors.Wrap(err, "error reading config for validation")
	}
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return nil, errors.Wrap(err, "error reading config for validation")
	}

	err = c.schema.Validate(inst)
	if err == nil {
		return nil, nil
	}
	var verr *jsonschema.ValidationError
	if !errors.As(err, &verr) {
		return nil, errors.Wrap(err, "error validating config")
	}

	var problems []Problem
	collect(verr, c.raw, &problems)
	slices.SortStableFunc(problems, func(a, b Problem) int { return strings.Compare(a.Path, b.Path) })
	return slices.Compact(problems), nil
}

var printer = message.NewPrinter(language.English)

func collect(e *jsonschema.ValidationError, raw any, out *[]Problem) {
	path := strings.Join(e.InstanceLocation, ".")

	switch k := e.ErrorKind.(type) {
	case *kind.OneOf, *kind.AnyOf:
		// One branch per accepted shape. Branches that only say "wrong type
		// here" are the shapes the value isn't; report what's wrong inside
		// the shape it is. If it is none of them, say what it could be.
		var inner []*jsonschema.ValidationError
		var want []string
		var got string
		for _, cause := range e.Causes {
			t, ok := cause.ErrorKind.(*kind.Type)
			if ok && slices.Equal(cause.InstanceLocation, e.InstanceLocation) {
				want = append(want, t.Want...)
				got = t.Got
				continue
			}
			inner = append(inner, cause)
		}
		if len(inner) == 0 && got != "" {
			*out = append(*out, Problem{Path: path, Message: fmt.Sprintf("got %s, want %s", got, strings.Join(want, " or "))})
			return
		}
		for _, cause := range inner {
			collect(cause, raw, out)
		}
		return
	case *kind.AdditionalProperties:
		known := propertyNames(schemaAt(raw, e.SchemaURL))
		for _, key := range k.Properties {
			msg := "unknown key"
			hint := SuggestKey(key, known)
			if hint != "" {
				msg += " - " + hint
			}
			*out = append(*out, Problem{Path: joinPath(path, key), Message: msg})
		}
		return
	case *kind.Required:
		// Recipes own required fields; check reports them with the recipe's
		// descriptions.
		return
	case *kind.Type:
		// Pilum reads a number or boolean given for a string as text.
		readable := (k.Got == "number" || k.Got == "integer" || k.Got == "boolean") && slices.Contains(k.Want, "string")
		*out = append(*out, Problem{Path: path, Message: typeMessage(k, readable), Warning: readable})
		return
	}

	if len(e.Causes) == 0 {
		*out = append(*out, Problem{Path: path, Message: e.ErrorKind.LocalizedString(printer)})
		return
	}
	for _, cause := range e.Causes {
		collect(cause, raw, out)
	}
}

// SuggestKey returns a "did you mean" hint for a misspelled key, or "". It
// allows about one edit per three characters, so a short key like "type"
// isn't matched to an unrelated one like "cpu".
func SuggestKey(key string, known []string) string {
	limit := max(1, len(key)/3)
	var nearby []string
	for _, candidate := range known {
		if suggest.LevenshteinDistance(key, candidate) <= limit {
			nearby = append(nearby, candidate)
		}
	}
	return suggest.FormatSuggestion(key, nearby)
}

func typeMessage(k *kind.Type, readable bool) string {
	msg := fmt.Sprintf("got %s, want %s", k.Got, strings.Join(k.Want, " or "))
	if readable {
		msg += " (read as text; quote it, since YAML reads 1.20 as 1.2)"
	}
	return msg
}

func joinPath(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

// schemaAt resolves the schema an error came from. url ends in a JSON pointer
// to it, e.g. "...#/$defs/cloud_run".
func schemaAt(raw any, url string) any {
	_, pointer, _ := strings.Cut(url, "#")
	node := raw
	for _, seg := range strings.Split(strings.TrimPrefix(pointer, "/"), "/") {
		if seg == "" {
			continue
		}
		seg = strings.NewReplacer("~1", "/", "~0", "~").Replace(seg)
		m, ok := node.(map[string]any)
		if !ok {
			return nil
		}
		node = m[seg]
	}
	return node
}

func propertyNames(schema any) []string {
	m, ok := schema.(map[string]any)
	if !ok {
		return nil
	}
	props, ok := m["properties"].(map[string]any)
	if !ok {
		return nil
	}
	names := make([]string, 0, len(props))
	for name := range props {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}
