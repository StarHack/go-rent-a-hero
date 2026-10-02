package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/wok/rent-a-hero/internal/formats/acs"
)

func cmdACS(args []string) error {
	fs := flag.NewFlagSet("acs", flag.ExitOnError)
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return err
	}

	if fs.NArg() != 1 {
		return fmt.Errorf("usage: rahdump acs <file.ACS>")
	}

	path := fs.Arg(0)

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	track, err := acs.Parse(data)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	fmt.Printf("file:     %s\n", path)
	fmt.Printf("size:     %d bytes\n", len(data))
	fmt.Printf("rate:     %v Hz\n", track.Rate)
	fmt.Printf("samples:  %d\n", len(track.Samples))
	fmt.Printf("duration: %.4fs\n", track.Duration())
	fmt.Printf("values:   %v\n", track.Samples)

	return nil
}
