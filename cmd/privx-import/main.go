package main

import (
	"fmt"
	"os"

	"github.com/SSHcom/inventory_import_privx_dev/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
