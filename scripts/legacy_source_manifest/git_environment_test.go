// Esta prueba impide que una inspección Git descargue objetos o refresque
// índices de las fuentes históricas.
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGitInspectionDisablesLazyFetchAndOptionalLocks(t *testing.T) {
	directory := t.TempDir()
	executable := filepath.Join(directory, "git")
	program := "#!/bin/sh\nprintf '%s|%s\\n' \"$GIT_NO_LAZY_FETCH\" \"$GIT_OPTIONAL_LOCKS\"\n"
	if err := os.WriteFile(executable, []byte(program), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory)

	output, err := runGit(time.Second, "", "version")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(output)) != "1|0" {
		t.Fatalf("entorno Git inseguro: %q", output)
	}
}
