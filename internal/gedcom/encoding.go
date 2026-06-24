package gedcom

import (
	"io"
	"os"
	"strings"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

func GetGedcomReader(path string) (io.Reader, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := string(data)

	switch {
	case strings.Contains(content, "1 CHAR UTF-8"), strings.Contains(content, "1 CHAR UTF8"):
		return strings.NewReader(content), nil

	case strings.Contains(content, "1 CHAR ASCII"), strings.Contains(content, "1 CHAR ANSI"):
		return convertFromWindows1252(content)

	case strings.Contains(content, "1 CHAR ANSEL"):
		return convertFromANSEL(content)

	default:
		return strings.NewReader(content), nil
	}
}

func convertFromWindows1252(content string) (io.Reader, error) {
	decoder := charmap.Windows1252.NewDecoder()
	utf8Content, _, err := transform.String(decoder, content)
	if err != nil {
		return strings.NewReader(content), nil
	}
	return strings.NewReader(utf8Content), nil
}

func convertFromANSEL(content string) (io.Reader, error) {
	decoder := charmap.ISO8859_1.NewDecoder()
	utf8Content, _, err := transform.String(decoder, content)
	if err != nil {
		return strings.NewReader(content), nil
	}
	return strings.NewReader(utf8Content), nil
}
