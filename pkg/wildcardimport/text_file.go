package wildcardimport

import (
	"bufio"
	"io"
	"io/fs"
	"os"
	"strings"
)

type ImportLineNotFoundError struct {
	Filename   string
	TargetLine string
}

func (e *ImportLineNotFoundError) Error() string {
	return e.Filename + ": Import Line Not Found: " + e.TargetLine
}

type TextFile struct {
	filename string
	info     fs.FileInfo

	beforeLines []string
	targetLine  string
	afterLines  []string
}

// NewTextFileFromFilename constructs a new text file split on the target line.
func NewTextFileFromFilename(filename string, targetLine string) (*TextFile, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	return NewTextFileFromReader(filename, info, f, targetLine)
}

// NewTextFileFromFilename constructs a new text file split on the target line.
func NewTextFileFromReader(filename string, info fs.FileInfo, in io.Reader, targetLine string) (*TextFile, error) {

	file := new(TextFile)
	file.filename = filename
	file.info = info

	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		line := scanner.Text()
		if line == targetLine {
			file.targetLine = targetLine
			continue
		}
		if line == "// "+targetLine { // already commented out (subsequent run)
			file.targetLine = targetLine
			continue
		}
		if file.targetLine == "" {
			file.beforeLines = append(file.beforeLines, line)
		} else {
			file.afterLines = append(file.afterLines, line)
		}
	}
	if file.targetLine == "" {
		return nil, &ImportLineNotFoundError{TargetLine: targetLine, Filename: filename}
	}

	// add a final entry to afterLines so that the file ends with a single newline
	file.afterLines = append(file.afterLines, "")

	return file, nil
}

func (f *TextFile) WriteOriginal() error {
	return f.Write(f.targetLine)
}

func (f *TextFile) WriteCommented() error {
	return f.Write("// " + f.targetLine)
}

func (f *TextFile) Write(targetLine string) error {
	lines := append(f.beforeLines, targetLine)
	lines = append(lines, f.afterLines...)
	content := strings.Join(lines, "\n")
	data := []byte(content)
	if err := os.WriteFile(f.filename, data, f.info.Mode()); err != nil {
		return err
	}
	return nil
}
