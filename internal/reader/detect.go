package reader

import (
	"fmt"
	"path/filepath"
	"strings"
)

// DetectFormat detects the format of a single input file.
func DetectFormat(path string) (Format, error) {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".csv":
		return FormatCSV, nil
	case ".json":
		return FormatJSON, nil
	case ".xml":
		return FormatXML, nil
	default:
		return "", fmt.Errorf("unsupported input format for file %q", path)
	}
}

// buildInputs detects the format of all input files.
func BuildInputs(paths []string) ([]Input, error) {
	var inputs []Input

	for _, path := range paths {
		format, err := DetectFormat(path)
		if err != nil {
			return nil, err
		}

		inputs = append(inputs, Input{
			Source: path,
			Format: format,
			Path:   path,
		})
	}

	return inputs, nil
}
