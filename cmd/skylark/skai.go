package main

import (
	"fmt"
	"os"

	"github.com/butter-bot-machines/skylark/pkg/cmd"
)

func main() {
	cli := cmd.NewCLI()
	if err := cli.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
