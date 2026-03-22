/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNombreRespaldoFechableSanitizaEtiqueta(t *testing.T) {
	t.Parallel()

	got := nombreRespaldoFechable(time.Date(2026, 3, 22, 10, 11, 12, 123456789, time.UTC), "Manual Final")
	if !strings.HasPrefix(got, "2026-03-22_10-11-12.123456789_manual_final") {
		t.Fatalf("nombre inesperado: %s", got)
	}
	if !strings.HasSuffix(got, "_orquesta.db.bak") {
		t.Fatalf("sufijo inesperado: %s", got)
	}
}

func TestRespaldoBDCreaFicheroYAplicaRetencion(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	destino := filepath.Join(tmp, "respaldos")

	anteriorAhora := ahoraRespaldo
	ahoraBase := time.Date(2026, 3, 22, 11, 0, 0, 0, time.UTC)
	llamadas := 0
	ahoraRespaldo = func() time.Time {
		valor := ahoraBase.Add(time.Duration(llamadas) * time.Second)
		llamadas++
		return valor
	}
	t.Cleanup(func() {
		ahoraRespaldo = anteriorAhora
	})

	for i := 0; i < 3; i++ {
		if err := respaldoBDCmd.Flags().Set("destino", destino); err != nil {
			t.Fatalf("set destino: %v", err)
		}
		if err := respaldoBDCmd.Flags().Set("etiqueta", "Manual Final"); err != nil {
			t.Fatalf("set etiqueta: %v", err)
		}
		if err := respaldoBDCmd.Flags().Set("retener", "2"); err != nil {
			t.Fatalf("set retener: %v", err)
		}

		out := capturarStdout(t, func() {
			if err := respaldoBDCmd.RunE(respaldoBDCmd, nil); err != nil {
				t.Fatalf("run respaldo: %v", err)
			}
		})
		if !strings.Contains(out, "Respaldo creado:") {
			t.Fatalf("salida inesperada: %s", out)
		}
	}

	ficheros, err := filepath.Glob(filepath.Join(destino, "*_orquesta.db.bak"))
	if err != nil {
		t.Fatalf("glob respaldos: %v", err)
	}
	if len(ficheros) != 2 {
		t.Fatalf("retencion inesperada, se esperaban 2 ficheros y hay %d: %v", len(ficheros), ficheros)
	}
	for _, nombre := range ficheros {
		info, err := os.Stat(nombre)
		if err != nil {
			t.Fatalf("stat respaldo: %v", err)
		}
		if info.Size() == 0 {
			t.Fatalf("respaldo vacio: %s", nombre)
		}
	}
}
