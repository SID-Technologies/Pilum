package writer

import (
	"bytes"
	"os"
	"path/filepath"
	"text/template"

	"github.com/sid-technologies/pilum/lib/errors"
)

type FileWriter struct {
	BaseSourcePath string
	BaseOutputPath string
}

type FileOperation struct {
	SourcePath string
	OutputPath string
	Content    string
}

func NewFileWriter(sourcePath string, outputPath string) *FileWriter {
	return &FileWriter{
		BaseSourcePath: sourcePath,
		BaseOutputPath: outputPath,
	}
}

func (*FileWriter) WriteFile(operation FileOperation) error {
	outputDir := filepath.Dir(operation.OutputPath)
	err := os.MkdirAll(outputDir, 0775)
	if err != nil {
		return errors.Wrap(err, "error creating output directory")
	}

	var content []byte
	tmpl, err := template.New(("file")).Parse(operation.Content)
	if err != nil {
		return errors.Wrap(err, "error parsing template")
	}

	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, nil)
	if err != nil {
		return errors.Wrap(err, "error executing template")
	}

	content = buf.Bytes()

	err = os.WriteFile(operation.OutputPath, content, 0644)
	if err != nil {
		return errors.Wrap(err, "error writing file")
	}

	return nil
}

func (fw *FileWriter) WriteFiles(operations []FileOperation) error {
	for _, operation := range operations {
		err := fw.WriteFile(operation)
		if err != nil {
			return errors.Wrap(err, "error writing file")
		}
	}

	return nil
}

func (fw *FileWriter) ReadAndWriteFile(sourcePath string, outputPath string) error {
	content, err := os.ReadFile(filepath.Join(fw.BaseSourcePath, sourcePath))
	if err != nil {
		return errors.Wrap(err, "error reading source file")
	}

	op := FileOperation{
		SourcePath: sourcePath,
		OutputPath: filepath.Join(fw.BaseOutputPath, outputPath),
		Content:    string(content),
	}

	return fw.WriteFile(op)
}

func (fw *FileWriter) ReadAndWriteFiles(files []struct{ Source, Output string }) error {
	for _, file := range files {
		err := fw.ReadAndWriteFile(file.Source, file.Output)
		if err != nil {
			return errors.Wrap(err, "error reading source file")
		}
	}

	return nil
}
