package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/Ahmedaltu/thinbox/internal/container"
	"github.com/Ahmedaltu/thinbox/internal/state"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		// Fork a child process into new Linux namespaces and run the requested command.
		// Args: <image> <command> [args...]
		if err := container.Run(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "thinbox: run error: %v\n", err)
			os.Exit(1)
		}

	case "child":
		// Internal subcommand — not called by the user directly.
		// The binary re-execs itself with "child" after clone(2) so this code
		// runs inside the new namespaces before execing the user command.
		if err := container.Child(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "thinbox: %v\n", err)
			os.Exit(1)
		}

	case "ps":
		// Read all persisted container states and print a summary table.
		states, err := state.List()
		if err != nil {
			fmt.Fprintf(os.Stderr, "thinbox: ps: %v\n", err)
			os.Exit(1)
		}
		if len(states) == 0 {
			fmt.Println("no running containers")
			return
		}
		fmt.Printf("%-12s  %-8s  %-6s  %-20s  %s\n", "ID", "IMAGE", "PID", "STARTED", "COMMAND")
		for _, s := range states {
			fmt.Printf("%-12s  %-8s  %-6d  %-20s  %s\n",
				s.ID,
				s.Image,
				s.PID,
				s.StartedAt.Format("2006-01-02T15:04:05"),
				strings.Join(s.Command, " "),
			)
		}

	case "help", "--help", "-h":
		printUsage()

	default:
		fmt.Fprintf(os.Stderr, "thinbox: unknown command %q\n", os.Args[1])
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`thinbox — lightweight Linux container runtime

Usage:
  thinbox run <image> <command>
  thinbox ps
  thinbox help`)
}
