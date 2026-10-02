// Command rahdump inspects and validates the original game's asset files.
// It prints file metadata, scene contents, frame counts and validation
// information without requiring any rendering/audio backend.
package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "rahdump:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return fmt.Errorf("missing command")
	}

	cmd, rest := args[0], args[1:]

	switch cmd {
	case "szn":
		return cmdSZN(rest)
	case "tcc":
		return cmdTCC(rest)
	case "a16":
		return cmdA16(rest)
	case "zbf":
		return cmdZBF(rest)
	case "acs":
		return cmdACS(rest)
	case "bmp":
		return cmdBMP(rest)
	case "scan":
		return cmdScan(rest)
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		printUsage()
		return fmt.Errorf("unknown command %q", cmd)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `usage:
  rahdump szn <file.SZN>
  rahdump tcc <file.TCC> [--png out.png]
  rahdump a16 <file.a16> [--frames out-dir/]
  rahdump zbf <file.ZBF> [--png out.png] [--mode gray|walk|depth]
  rahdump acs <file.ACS>
  rahdump bmp <file.BMP> [--png out.png]
  rahdump scan <asset-root>`)
}
