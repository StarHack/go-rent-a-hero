package game

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/wok/rent-a-hero/internal/assets"
	"github.com/wok/rent-a-hero/internal/datapath"
)

func builtinController(location int) Controller {
	switch location {
	case 1:
		return LOC01Controller{}
	case 2:
		return LOC02Controller{}
	case 3:
		return LOC03Controller{}
	case 4:
		return LOC04Controller{}
	case 5:
		return LOC05Controller{}
	case 6:
		return LOC06Controller{}
	case 7:
		return LOC07Controller{}
	case 8:
		return LOC08Controller{}
	case 9:
		return LOC09Controller{}
	case 10:
		return LOC10Controller{}
	case 12:
		return LOC12Controller{}
	case 14:
		return LOC14Controller{}
	case 15:
		return LOC15Controller{}
	case 16:
		return LOC16Controller{}
	case 17:
		return LOC17Controller{}
	case 18:
		return LOC18Controller{}
	case 19:
		return LOC19Controller{}
	case 20:
		return LOC20Controller{}
	case 21:
		return LOC21Controller{}
	case 22:
		return LOC22Controller{}
	case 23:
		return LOC23Controller{}
	case 24:
		return LOC24Controller{}
	case 25:
		return LOC25Controller{}
	case 26:
		return LOC26Controller{}
	case 27:
		return LOC27Controller{}
	case 28:
		return LOC28Controller{}
	case 29:
		return LOC29Controller{}
	case 30:
		return LOC30Controller{}
	case 31:
		return LOC31Controller{}
	case 35:
		return LOC35Controller{}
	case 37:
		return LOC37Controller{}
	case 41:
		return LOC41Controller{}
	default:
		return nil
	}
}

func resolveBuiltinLocation(location int) (*assets.Index, Controller, bool, error) {
	locRoot, found, err := resolveLocationAssetRoot(location)
	if err != nil {
		return nil, nil, true, err
	}
	if !found {
		return nil, nil, false, nil
	}
	commonInstall, err := datapath.Resolve(filepath.Join("data", "Common"))
	if err != nil {
		return nil, nil, true, err
	}
	commonCD, err := datapath.Resolve(filepath.Join("data", "COMMONCD"))
	if err != nil {
		return nil, nil, true, err
	}
	idx, err := assets.NewIndex(locRoot, commonInstall, commonCD)
	if err != nil {
		return nil, nil, true, err
	}
	return idx, builtinController(location), true, nil
}

func resolveLocationAssetRoot(location int) (string, bool, error) {
	exact := filepath.Join("data", fmt.Sprintf("LOC%02d", location))
	if root, err := datapath.Resolve(exact); err == nil {
		return root, true, nil
	}
	gameRoot, err := datapath.Resolve(filepath.Join("data"))
	if err != nil {
		return "", false, nil
	}
	entries, err := os.ReadDir(gameRoot)
	if err != nil {
		return "", false, err
	}
	var match string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := strings.ToUpper(entry.Name())
		if !strings.HasPrefix(name, "LOC") {
			continue
		}
		n, err := strconv.Atoi(strings.TrimPrefix(name, "LOC"))
		if err != nil {
			continue
		}
		if n != location {
			continue
		}
		candidate := filepath.Join(gameRoot, entry.Name())
		if match != "" && !strings.EqualFold(match, candidate) {
			return "", true, fmt.Errorf("game: multiple asset directories match location %d: %q and %q", location, match, candidate)
		}
		match = candidate
	}
	if match == "" {
		return "", false, nil
	}
	return match, true, nil
}
