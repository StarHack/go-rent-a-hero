package main

import "strings"

// reorderFlags moves every "--name value" or "--name=value" pair to the
// front and positional arguments to the back, so callers can write
// `rahdump tcc file.TCC --png out.png` instead of being forced to put flags
// first (the stdlib flag package stops parsing at the first non-flag
// argument). All flags used by rahdump take exactly one value.
func reorderFlags(args []string) []string {
	var flags, positional []string

	for i := 0; i < len(args); i++ {
		a := args[i]

		if strings.HasPrefix(a, "--") || strings.HasPrefix(a, "-") {
			flags = append(flags, a)

			if !strings.Contains(a, "=") && i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}

			continue
		}

		positional = append(positional, a)
	}

	return append(flags, positional...)
}
