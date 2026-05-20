package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		fmt.Println("thinbox: run called — not implemented yet")
	case "ps":
		fmt.Println("thinbox: ps called — not implemented yet")
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