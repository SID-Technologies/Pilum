package writer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sid-technologies/pilum/lib/writer"

	"github.com/stretchr/testify/require"
)

func TestNewFileWriter(t *testing.T) {
	t.Parallel()

	fw := writer.NewFileWriter("/source/path", "/output/path")

	require.NotNil(t, fw)
	require.Equal(t, "/source/path", fw.BaseSourcePath)
	require.Equal(t, "/output/path", fw.BaseOutputPath)
}

func TestFileWriterWriteFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	fw := writer.NewFileWriter("", tmpDir)

	op := writer.FileOperation{
		SourcePath: "source.txt",
		OutputPath: filepath.Join(tmpDir, "output.txt"),
		Content:    "Hello, World!",
	}

	err := fw.WriteFile(op)

	require.NoError(t, err)

	// Verify file was created
	content, err := os.ReadFile(filepath.Join(tmpDir, "output.txt"))
	require.NoError(t, err)
	require.Equal(t, "Hello, World!", string(content))
}

func TestFileWriterWriteFileCreatesDirectories(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	fw := writer.NewFileWriter("", tmpDir)

	// Output path has nested directories
	op := writer.FileOperation{
		SourcePath: "source.txt",
		OutputPath: filepath.Join(tmpDir, "a", "b", "c", "output.txt"),
		Content:    "Nested content",
	}

	err := fw.WriteFile(op)

	require.NoError(t, err)

	// Verify directories and file were created
	content, err := os.ReadFile(filepath.Join(tmpDir, "a", "b", "c", "output.txt"))
	require.NoError(t, err)
	require.Equal(t, "Nested content", string(content))
}

func TestFileWriterWriteFileWithTemplate(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	fw := writer.NewFileWriter("", tmpDir)

	// Content with Go template syntax (but no data to substitute)
	op := writer.FileOperation{
		SourcePath: "source.txt",
		OutputPath: filepath.Join(tmpDir, "output.txt"),
		Content:    "Static content without templates",
	}

	err := fw.WriteFile(op)

	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(tmpDir, "output.txt"))
	require.NoError(t, err)
	require.Equal(t, "Static content without templates", string(content))
}

func TestFileWriterWriteFileInvalidTemplate(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	fw := writer.NewFileWriter("", tmpDir)

	// Invalid template syntax
	op := writer.FileOperation{
		SourcePath: "source.txt",
		OutputPath: filepath.Join(tmpDir, "output.txt"),
		Content:    "{{ .InvalidSyntax",
	}

	err := fw.WriteFile(op)

	require.Error(t, err)
	require.Contains(t, err.Error(), "error parsing template")
}

func TestFileWriterWriteFiles(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	fw := writer.NewFileWriter("", tmpDir)

	ops := []writer.FileOperation{
		{
			SourcePath: "file1.txt",
			OutputPath: filepath.Join(tmpDir, "file1.txt"),
			Content:    "Content 1",
		},
		{
			SourcePath: "file2.txt",
			OutputPath: filepath.Join(tmpDir, "file2.txt"),
			Content:    "Content 2",
		},
		{
			SourcePath: "file3.txt",
			OutputPath: filepath.Join(tmpDir, "file3.txt"),
			Content:    "Content 3",
		},
	}

	err := fw.WriteFiles(ops)

	require.NoError(t, err)

	// Verify all files were created
	for i, op := range ops {
		content, err := os.ReadFile(op.OutputPath)
		require.NoError(t, err)
		require.Equal(t, ops[i].Content, string(content))
	}
}

func TestFileWriterWriteFilesEmpty(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	fw := writer.NewFileWriter("", tmpDir)

	err := fw.WriteFiles([]writer.FileOperation{})

	require.NoError(t, err)
}

func TestFileWriterWriteFilesStopsOnError(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	fw := writer.NewFileWriter("", tmpDir)

	ops := []writer.FileOperation{
		{
			SourcePath: "file1.txt",
			OutputPath: filepath.Join(tmpDir, "file1.txt"),
			Content:    "Content 1",
		},
		{
			SourcePath: "file2.txt",
			OutputPath: filepath.Join(tmpDir, "file2.txt"),
			Content:    "{{ .Invalid", // Invalid template
		},
		{
			SourcePath: "file3.txt",
			OutputPath: filepath.Join(tmpDir, "file3.txt"),
			Content:    "Content 3",
		},
	}

	err := fw.WriteFiles(ops)

	require.Error(t, err)

	// First file should exist
	_, err = os.Stat(filepath.Join(tmpDir, "file1.txt"))
	require.NoError(t, err)

	// Third file should not exist (stopped on second file error)
	_, err = os.Stat(filepath.Join(tmpDir, "file3.txt"))
	require.True(t, os.IsNotExist(err))
}

func TestFileWriterReadAndWriteFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	outputDir := filepath.Join(tmpDir, "output")

	err := os.MkdirAll(sourceDir, 0755)
	require.NoError(t, err)

	// Create source file
	sourceContent := "Source file content"
	err = os.WriteFile(filepath.Join(sourceDir, "template.txt"), []byte(sourceContent), 0644)
	require.NoError(t, err)

	fw := writer.NewFileWriter(sourceDir, outputDir)

	err = fw.ReadAndWriteFile("template.txt", "result.txt")

	require.NoError(t, err)

	// Verify output file
	content, err := os.ReadFile(filepath.Join(outputDir, "result.txt"))
	require.NoError(t, err)
	require.Equal(t, sourceContent, string(content))
}

func TestFileWriterReadAndWriteFileNotFound(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	fw := writer.NewFileWriter(tmpDir, tmpDir)

	err := fw.ReadAndWriteFile("nonexistent.txt", "output.txt")

	require.Error(t, err)
	require.Contains(t, err.Error(), "error reading source file")
}

func TestFileWriterReadAndWriteFiles(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	outputDir := filepath.Join(tmpDir, "output")

	err := os.MkdirAll(sourceDir, 0755)
	require.NoError(t, err)

	// Create source files
	files := []struct{ Source, Output string }{
		{"file1.txt", "out1.txt"},
		{"file2.txt", "out2.txt"},
	}

	for _, f := range files {
		err = os.WriteFile(filepath.Join(sourceDir, f.Source), []byte("Content of "+f.Source), 0644)
		require.NoError(t, err)
	}

	fw := writer.NewFileWriter(sourceDir, outputDir)

	err = fw.ReadAndWriteFiles(files)

	require.NoError(t, err)

	// Verify output files
	for _, f := range files {
		content, err := os.ReadFile(filepath.Join(outputDir, f.Output))
		require.NoError(t, err)
		require.Equal(t, "Content of "+f.Source, string(content))
	}
}

func TestFileWriterReadAndWriteFilesEmpty(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	fw := writer.NewFileWriter(tmpDir, tmpDir)

	err := fw.ReadAndWriteFiles([]struct{ Source, Output string }{})

	require.NoError(t, err)
}

func TestFileOperationStruct(t *testing.T) {
	t.Parallel()

	op := writer.FileOperation{
		SourcePath: "/src/template.txt",
		OutputPath: "/out/result.txt",
		Content:    "Template content",
	}

	require.Equal(t, "/src/template.txt", op.SourcePath)
	require.Equal(t, "/out/result.txt", op.OutputPath)
	require.Equal(t, "Template content", op.Content)
}

func TestFileWriterOverwritesExistingFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create existing file
	existingPath := filepath.Join(tmpDir, "existing.txt")
	err := os.WriteFile(existingPath, []byte("Old content"), 0644)
	require.NoError(t, err)

	fw := writer.NewFileWriter("", tmpDir)

	op := writer.FileOperation{
		SourcePath: "source.txt",
		OutputPath: existingPath,
		Content:    "New content",
	}

	err = fw.WriteFile(op)

	require.NoError(t, err)

	// Verify content was overwritten
	content, err := os.ReadFile(existingPath)
	require.NoError(t, err)
	require.Equal(t, "New content", string(content))
}

func TestFileWriterEmptyContent(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	fw := writer.NewFileWriter("", tmpDir)

	op := writer.FileOperation{
		SourcePath: "source.txt",
		OutputPath: filepath.Join(tmpDir, "empty.txt"),
		Content:    "",
	}

	err := fw.WriteFile(op)

	require.NoError(t, err)

	// Verify empty file was created
	content, err := os.ReadFile(filepath.Join(tmpDir, "empty.txt"))
	require.NoError(t, err)
	require.Empty(t, content)
}
