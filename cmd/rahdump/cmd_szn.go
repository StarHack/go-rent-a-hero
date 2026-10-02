package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/wok/rent-a-hero/internal/formats/szn"
)

func cmdSZN(args []string) error {
	fs := flag.NewFlagSet("szn", flag.ExitOnError)
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return err
	}

	if fs.NArg() != 1 {
		return fmt.Errorf("usage: rahdump szn <file.SZN>")
	}

	path := fs.Arg(0)

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	def, err := szn.Load(path, data)
	if err != nil {
		return err
	}

	fmt.Printf("file: %s\n", path)
	fmt.Printf("backgrounds: %d, characters: %d, layers: %d, areas: %d\n",
		len(def.Backgrounds), len(def.Characters), len(def.Layers), len(def.Areas))

	for _, name := range def.General.Backgrounds {
		b := def.Backgrounds[name]
		fmt.Printf("  [background] %s filename=%s zBuf=%s wBuf=%s fps=%d extra=%v\n",
			name, b.Filename, b.ZBuf, b.WBuf, b.FPS, b.Extra)
	}

	for _, name := range def.General.Characters {
		c := def.Characters[name]
		fmt.Printf("  [character]  %s filename=%s X=%d Y=%d ZPos=%d Zoom=%d Direction=%d type=%s extra=%v\n",
			name, c.Filename, c.X, c.Y, c.ZPos, c.ZoomVal, c.Direction, formatIntPtr(c.Type), c.Extra)
		for _, z := range c.Zones {
			fmt.Printf("      zone[%d] R=%s G=%s B=%s X=%s Y=%s Range=%s\n",
				z.Index, formatIntPtr(z.Red), formatIntPtr(z.Green), formatIntPtr(z.Blue),
				formatIntPtr(z.X), formatIntPtr(z.Y), formatIntPtr(z.Range))
		}
	}

	for _, name := range def.General.Layers {
		l := def.Layers[name]
		fmt.Printf("  [layer]      %s filename=%s X=%d Y=%d Z=%s Zoom=%s fps=%s LoadCond=%s extra=%v\n",
			name, l.Filename, l.X, l.Y, formatIntPtr(l.Z), formatIntPtr(l.ZoomVal), formatIntPtr(l.FPS), formatIntPtr(l.LoadCond), l.Extra)
	}

	for _, name := range def.General.Areas {
		a := def.Areas[name]
		fmt.Printf("  [area]       %s rect=(%d,%d)-(%d,%d) cursorType=%s extra=%v\n",
			name, a.X1, a.Y1, a.X2, a.Y2, formatIntPtr(a.CursorType), a.Extra)
	}

	if len(def.General.Extra) > 0 {
		fmt.Printf("  [general extra] %v\n", def.General.Extra)
	}

	return nil
}

func formatIntPtr(p *int) string {
	if p == nil {
		return "<absent>"
	}
	return fmt.Sprintf("%d", *p)
}
