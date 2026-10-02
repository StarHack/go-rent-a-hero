package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "usage: %s <source> <target>\n", os.Args[0])
		os.Exit(2)
	}

	src, err := openSource(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "rahinstall: %v\n", err)
		os.Exit(1)
	}
	defer src.Close()

	target := os.Args[2]

	if err := os.MkdirAll(target, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "rahinstall: create target: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Copying GAME to %s\n", target)

	if err := copyTree(src, "GAME", target); err != nil {
		fmt.Fprintf(os.Stderr, "rahinstall: copy GAME: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Extracting Common data from SETUP/data1.cab\n")

	if err := extractInstallShieldCommon(src, "SETUP/data1.cab", target); err != nil {
		fmt.Fprintf(os.Stderr, "rahinstall: extract Common: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Installation data prepared successfully.")
}
