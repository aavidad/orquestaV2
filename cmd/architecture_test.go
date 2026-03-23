package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestCmdNoUsaDBDirectoFueraDeExcepcionesControladas(t *testing.T) {
	t.Parallel()

	root := "."
	pattern := regexp.MustCompile(`db\.([A-Za-z0-9_]+)\(`)
	allowByFile := map[string]map[string]bool{
		"root.go": {
			"Open":  true,
			"Close": true,
		},
	}
	allowEverywhere := map[string]bool{
		"EstadoPropuesta":        true,
		"EstadoTarea":            true,
		"PosicionVoto":           true,
		"PrioridadTarea":         true,
		"DriverName":             true,
		"PlaceholderStyle":       true,
		"BootstrapSchemaEnabled": true,
		"QueryRebindingEnabled":  true,
	}

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		base := filepath.Base(path)
		if filepath.Ext(base) != ".go" || strings.HasSuffix(base, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		matches := pattern.FindAllStringSubmatch(string(data), -1)
		for _, match := range matches {
			call := match[1]
			if allowEverywhere[call] {
				continue
			}
			if allowByFile[base][call] {
				continue
			}
			t.Errorf("%s usa db.%s() directamente", base, call)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk cmd dir: %v", err)
	}
}
