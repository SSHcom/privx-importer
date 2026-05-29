package output

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// RecordLookup returns optional row data for a source and line.
type RecordLookup func(source string, line int) map[string]string

func DebugDecodePagedItems[T any](data []byte) error {
	type rawListResponse struct {
		Count int               `json:"count"`
		Items []json.RawMessage `json:"items"`
	}

	var raw rawListResponse
	if err := decodeJSON(data, &raw, true); err != nil {
		return err
	}

	for i, item := range raw.Items {
		var v T
		if err := decodeJSON(item, &v, true); err != nil {
			fmt.Printf("failed to decode item %d:\n%s\n", i, string(item))
			return fmt.Errorf("decode item %d: %w", i, err)
		}
	}

	return nil
}

// decodeJSON decodes JSON into v and prints debug info on error if debug is true.
func decodeJSON(data []byte, v any, debug bool) error {
	dec := json.NewDecoder(bytes.NewReader(data))

	if err := dec.Decode(v); err != nil {
		if debug {
			offset := dec.InputOffset()

			start := int(offset) - 100
			if start < 0 {
				start = 0
			}
			end := int(offset) + 100
			if end > len(data) {
				end = len(data)
			}

			fmt.Printf("json decode error: %v\n", err)
			fmt.Printf("offset: %d\n", offset)
			fmt.Printf("context:\n%s\n", string(data[start:end]))
		}
		return fmt.Errorf("decode response body: %w", err)
	}

	return nil
}
