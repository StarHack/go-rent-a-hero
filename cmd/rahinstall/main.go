package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <source> [source ...]\n", os.Args[0])
		os.Exit(2)
	}

	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "rahinstall: executable path: %v\n", err)
		os.Exit(1)
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		fmt.Fprintf(os.Stderr, "rahinstall: executable path: %v\n", err)
		os.Exit(1)
	}
	target := filepath.Join(filepath.Dir(exe), "data")
	if err := os.MkdirAll(target, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "rahinstall: create target: %v\n", err)
		os.Exit(1)
	}

	for _, name := range os.Args[1:] {
		src, err := openSource(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "rahinstall: %s: %v\n", name, err)
			os.Exit(1)
		}

		if err := installSource(src, target); err != nil {
			src.Close()
			fmt.Fprintf(os.Stderr, "rahinstall: %s: %v\n", name, err)
			os.Exit(1)
		}
		if err := src.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "rahinstall: %s: close: %v\n", name, err)
			os.Exit(1)
		}
	}

	fmt.Println("Installation data prepared successfully.")
}

func installSource(src sourceFS, target string) error {
	if sourceDirExists(src, "GAME") {
		fmt.Printf("Copying GAME to %s\n", target)
		if err := copyTree(src, "GAME", target); err != nil {
			return fmt.Errorf("copy GAME: %w", err)
		}
	}

	if sourceDirExists(src, "Commons") {
		fmt.Printf("Copying Commons to %s\n", target)
		if err := copyTree(src, "Commons", target); err != nil {
			return fmt.Errorf("copy Commons: %w", err)
		}
	}

	if sourceFileExists(src, "SETUP/data1.cab") {
		fmt.Println("Extracting Common data from SETUP/data1.cab")
		if err := extractInstallShieldCommon(src, "SETUP/data1.cab", target); err != nil {
			return fmt.Errorf("extract Common: %w", err)
		}
	}

	return nil
}

func sourceDirExists(src sourceFS, name string) bool {
	_, err := src.ReadDir(name)
	return err == nil
}

func sourceFileExists(src sourceFS, name string) bool {
	f, err := src.Open(name)
	if err != nil {
		return false
	}
	return f.Close() == nil
}
