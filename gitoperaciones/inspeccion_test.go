/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package gitoperaciones

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestInspeccionarTrabajoGitDetectaArchivosModificadosYNoTrackeados(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	mustRunGit(t, "", "init", "-b", "main", repo)
	mustRunGit(t, repo, "config", "user.name", "Orquesta Test")
	mustRunGit(t, repo, "config", "user.email", "orquesta@example.test")

	if err := os.WriteFile(filepath.Join(repo, "uno.go"), []byte("package demo\n"), 0o644); err != nil {
		t.Fatalf("write uno.go: %v", err)
	}
	mustRunGit(t, repo, "add", "uno.go")
	mustRunGit(t, repo, "commit", "-m", "base")

	if err := os.WriteFile(filepath.Join(repo, "uno.go"), []byte("package demo\n\nfunc Uno() {}\n"), 0o644); err != nil {
		t.Fatalf("rewrite uno.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "dos.go"), []byte("package demo\n"), 0o644); err != nil {
		t.Fatalf("write dos.go: %v", err)
	}

	estado, err := InspeccionarTrabajoGit(repo)
	if err != nil {
		t.Fatalf("InspeccionarTrabajoGit: %v", err)
	}
	if estado.HeadCommit == "" {
		t.Fatal("faltaba head commit")
	}
	if !slices.Contains(estado.ArchivosModificados, "uno.go") || !slices.Contains(estado.ArchivosModificados, "dos.go") {
		t.Fatalf("archivos modificados inesperados: %#v", estado.ArchivosModificados)
	}
}

func TestParsearArchivosDesdeStatusPorcelainToleraRenombre(t *testing.T) {
	items := parsearArchivosDesdeStatusPorcelain("R  viejo.go -> nuevo.go\n?? extra.go\n")
	if len(items) != 2 || items[0] != "nuevo.go" || items[1] != "extra.go" {
		t.Fatalf("items inesperados: %#v", items)
	}
}
