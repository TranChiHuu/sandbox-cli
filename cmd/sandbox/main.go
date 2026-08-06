package main

import (
	"context"
	"fmt"
	"os"

	"github.com/tranchihuu/sandbox-cli/internal/cli"
)

func main() {
	if err := cli.NewRoot().ExecuteContext(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "sandbox:", err)
		os.Exit(1)
	}
}
