package output_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/sid-technologies/pilum/lib/output"

	"github.com/stretchr/testify/require"
)

func TestJSON(t *testing.T) {
	t.Parallel()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	output.JSON(map[string]any{"success": true, "count": 3})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	require.Contains(t, got, `"success": true`)
	require.Contains(t, got, `"count": 3`)
}

func TestJSONNil(t *testing.T) {
	t.Parallel()

	// Should not panic on nil
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	output.JSON(nil)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	require.Equal(t, "null\n", got)
}

func TestJSONStruct(t *testing.T) {
	t.Parallel()

	type result struct {
		Name    string `json:"name"`
		Success bool   `json:"success"`
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	output.JSON(result{Name: "test", Success: true})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	require.Contains(t, got, `"name": "test"`)
	require.Contains(t, got, `"success": true`)
}
