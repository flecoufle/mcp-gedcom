package gedcom

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetGedcomReaderUTF8(t *testing.T) {
	content := `0 HEAD
1 CHAR UTF-8
1 FILE test.ged
0 @I1@ INDI
1 NAME Test /Person/
`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_utf8.ged")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	reader, err := GetGedcomReader(tmpFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("unexpected error reading: %v", err)
	}

	if !strings.Contains(string(result), "Test /Person/") {
		t.Error("expected 'Test /Person/' in result for UTF-8")
	}
}

func TestGetGedcomReaderASCII(t *testing.T) {
	content := `0 HEAD
1 CHAR ASCII
1 FILE test.ged
0 @I1@ INDI
1 NAME Test /Person/
`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_ascii.ged")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	reader, err := GetGedcomReader(tmpFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("unexpected error reading: %v", err)
	}

	if !strings.Contains(string(result), "Test /Person/") {
		t.Error("expected 'Test /Person/' in result after ASCII handling")
	}
}

func TestGetGedcomReaderANSI(t *testing.T) {
	content := `0 HEAD
1 CHAR ANSI
1 FILE test.ged
0 @I1@ INDI
1 NAME Test /Person/
`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_ansi.ged")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	reader, err := GetGedcomReader(tmpFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("unexpected error reading: %v", err)
	}

	if !strings.Contains(string(result), "Test /Person/") {
		t.Error("expected 'Test /Person/' in result after ANSI handling")
	}
}

func TestGetGedcomReaderANSEL(t *testing.T) {
	content := `0 HEAD
1 CHAR ANSEL
1 FILE test.ged
0 @I1@ INDI
1 NAME Test /Person/
`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_ansel.ged")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	reader, err := GetGedcomReader(tmpFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("unexpected error reading: %v", err)
	}

	if !strings.Contains(string(result), "Test /Person/") {
		t.Error("expected 'Test /Person/' in result after ANSEL handling")
	}
}

func TestGetGedcomReaderDefault(t *testing.T) {
	content := `0 HEAD
1 CHAR UNKNOWN
1 FILE test.ged
0 @I1@ INDI
1 NAME Test /Person/
`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_default.ged")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	reader, err := GetGedcomReader(tmpFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("unexpected error reading: %v", err)
	}

	if !strings.Contains(string(result), "Test /Person/") {
		t.Error("expected 'Test /Person/' in result for default case")
	}
}

func TestGetGedcomReaderFileNotFound(t *testing.T) {
	_, err := GetGedcomReader("/nonexistent/path/test.ged")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestGetGedcomReaderMultipleEncodings(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		content string
	}{
		{"UTF-8 variant 1", "1 CHAR UTF-8", "Test /Person/"},
		{"UTF-8 variant 2", "1 CHAR UTF8", "Test /Person/"},
		{"ASCII", "1 CHAR ASCII", "Test /Person/"},
		{"ANSI", "1 CHAR ANSI", "Test /Person/"},
		{"ANSEL", "1 CHAR ANSEL", "Test /Person/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := "0 HEAD\n" + tt.header + "\n1 FILE test.ged\n0 @I1@ INDI\n1 NAME " + tt.content + "\n"

			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.ged")
			if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
				t.Fatal(err)
			}

			reader, err := GetGedcomReader(tmpFile)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			result, err := io.ReadAll(reader)
			if err != nil {
				t.Fatalf("unexpected error reading: %v", err)
			}

			if !strings.Contains(string(result), tt.content) {
				t.Errorf("expected '%s' in result", tt.content)
			}
		})
	}
}
