// Command snagify diagnoses why a project works on one machine but not another
// by capturing and diffing environment/project snapshots.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/harshdevelops/snagify/internal/cli"
)

// version is overridable at build time via -ldflags "-X main.version=...".
var version = "0.3.2"

// coder is implemented by errors that carry a desired exit code.
type coder interface{ Code() int }

func main() {
	root := cli.NewRootCmd(version)
	if err := root.Execute(); err != nil {
		var c coder
		if errors.As(err, &c) {
			if msg := err.Error(); msg != "" {
				fmt.Fprintln(os.Stderr, "error:", msg)
			}
			os.Exit(c.Code())
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}
}
