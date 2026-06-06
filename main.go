// Command snagify diagnoses why a project works on one machine but not another
// by capturing and diffing environment/project snapshots.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/HarshDevelops/snagify/internal/cli"
)

// version is overridable at build time via -ldflags "-X main.version=...".
var version = "0.1.0"

// coder is implemented by errors that carry a desired exit code.
type coder interface{ Code() int }

func main() {
	root := cli.NewRootCmd(version)
	if err := root.Execute(); err != nil {
		var c coder
		if errors.As(err, &c) {
			os.Exit(c.Code())
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}
}
