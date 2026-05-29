package reader

import (
	"context"
	"fmt"

	"github.com/SSHcom/inventory_import_privx_dev/internal/config"
	"github.com/SSHcom/inventory_import_privx_dev/internal/output"
)

type MultiReader struct {
	readers map[Format]Reader
}

func NewMultiReader(readers map[Format]Reader) *MultiReader {
	return &MultiReader{readers: readers}
}

// ReadAll will loop through input sources and call appropriate reader for each.
func (c *MultiReader) ReadAll(ctx context.Context, out output.Output, cfg config.Config, inputs []Input) (ReadResult, error) {
	var result ReadResult

	for _, input := range inputs {
		r, ok := c.readers[input.Format]
		if !ok {
			return ReadResult{}, fmt.Errorf("unsupported input format: %s", input.Format)
		}

		sourceResult, err := r.Read(ctx, out, cfg, input)
		if err != nil {
			return ReadResult{}, err
		}

		result.Sources = append(result.Sources, sourceResult)
	}

	return result, nil
}
