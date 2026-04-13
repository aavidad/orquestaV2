/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package gitoperaciones

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

type EstadoTrabajoGit struct {
	HeadCommit         string
	ArchivosModificados []string
	Diff               string
}

func InspeccionarTrabajoGit(ruta string) (*EstadoTrabajoGit, error) {
	ruta = strings.TrimSpace(ruta)
	if ruta == "" {
		return nil, fmt.Errorf("ruta git obligatoria")
	}
	head, err := gitOutput(ruta, "rev-parse", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("resolver HEAD: %w", err)
	}
	estado, err := gitOutputCrudo(ruta, "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return nil, fmt.Errorf("leer git status: %w", err)
	}
	archivos := parsearArchivosDesdeStatusPorcelain(estado)
	diff, err := gitOutputOpcional(ruta, "diff", "--no-ext-diff", "--relative", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("leer git diff: %w", err)
	}
	return &EstadoTrabajoGit{
		HeadCommit:         strings.TrimSpace(head),
		ArchivosModificados: archivos,
		Diff:               strings.TrimSpace(diff),
	}, nil
}

func parsearArchivosDesdeStatusPorcelain(raw string) []string {
	lineas := strings.Split(strings.TrimRight(raw, "\n"), "\n")
	resultado := make([]string, 0, len(lineas))
	for _, linea := range lineas {
		if strings.TrimSpace(linea) == "" || len(linea) < 4 {
			continue
		}
		resto := strings.TrimSpace(linea[3:])
		if resto == "" {
			continue
		}
		if idx := strings.LastIndex(resto, " -> "); idx >= 0 {
			resto = strings.TrimSpace(resto[idx+4:])
		}
		resto = filepath.ToSlash(strings.TrimSpace(resto))
		if resto == "" || slices.Contains(resultado, resto) {
			continue
		}
		resultado = append(resultado, resto)
	}
	return resultado
}

func gitOutputOpcional(repoPath string, args ...string) (string, error) {
	out, err := gitOutput(repoPath, args...)
	if err != nil && strings.Contains(err.Error(), "exit status 1") {
		return "", nil
	}
	return out, err
}

func gitOutputCrudo(repoPath string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", repoPath}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}
