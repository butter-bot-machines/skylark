package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	// Command line flags
	initCmd := flag.NewFlagSet("init", flag.ExitOnError)
	watchCmd := flag.NewFlagSet("watch", flag.ExitOnError)
	runCmd := flag.NewFlagSet("run", flag.ExitOnError)
	versionCmd := flag.NewFlagSet("version", flag.ExitOnError)

	if len(os.Args) < 2 {
		fmt.Println("expected 'init', 'watch', 'run' or 'version' subcommands")
		os.Exit(1)
	}

	// Parse command
	switch os.Args[1] {
	case "init":
		initCmd.Parse(os.Args[2:])
		// TODO: Implement project initialization
	case "watch":
		watchCmd.Parse(os.Args[2:])
		// TODO: Implement file watching
	case "run":
		runCmd.Parse(os.Args[2:])
		// TODO: Implement one-time processing
	case "version":
		versionCmd.Parse(os.Args[2:])
		fmt.Println("Skylark v0.1.0")
	default:
		fmt.Printf("%q is not valid command.\n", os.Args[1])
		os.Exit(1)
	}
}
