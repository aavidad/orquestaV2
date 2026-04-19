/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"encoding/json"
	"net/http"
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
	if !strings.HasSuffix(got, "_orquesta.sqlite.bak") {
		t.Fatalf("sufijo inesperado: %s", got)
	}
}

func TestNombreRespaldoFechableUsaSufijoDelBackendActual(t *testing.T) {
	t.Setenv("ORQUESTA_DB_DRIVER", "postgres")
	t.Setenv("ORQUESTA_DB_BACKEND", "")
	t.Setenv("ORQUESTA_DB_DSN", "postgres://user:pass@localhost/orquesta")
	t.Setenv("ORQUESTA_DB", "")

	got := nombreRespaldoFechable(time.Date(2026, 3, 22, 10, 11, 12, 123456789, time.UTC), "Manual Final")
	if !strings.HasSuffix(got, "_orquesta.postgres.bak") {
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
		ruta, err := ejecutarRespaldoBD(destino, "Manual Final", 2)
		if err != nil {
			t.Fatalf("ejecutar respaldo helper: %v", err)
		}
		if !strings.Contains(ruta, destino) {
			t.Fatalf("ruta inesperada: %s", ruta)
		}
	}

	ficheros, err := filepath.Glob(filepath.Join(destino, "*_orquesta.sqlite.bak"))
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

func TestRespaldoBDUsaAPI(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/respaldo/bd", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiRespaldoBDResponse{
			OK:   true,
			Ruta: "/tmp/orquesta-api.bak",
		})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	if err := respaldoBDCmd.Flags().Set("destino", ""); err != nil {
		t.Fatalf("set destino: %v", err)
	}
	if err := respaldoBDCmd.Flags().Set("etiqueta", ""); err != nil {
		t.Fatalf("set etiqueta: %v", err)
	}
	if err := respaldoBDCmd.Flags().Set("retener", "0"); err != nil {
		t.Fatalf("set retener: %v", err)
	}

	out := capturarStdout(t, func() {
		if err := respaldoBDCmd.RunE(respaldoBDCmd, nil); err != nil {
			t.Fatalf("run respaldo api: %v", err)
		}
	})
	if !strings.Contains(out, "/tmp/orquesta-api.bak") {
		t.Fatalf("salida respaldo via API inesperada: %s", out)
	}
}

func TestAplicarRetencionRespaldoFiltraPorSufijoDelBackendActual(t *testing.T) {
	t.Setenv("ORQUESTA_DB_DRIVER", "postgres")
	t.Setenv("ORQUESTA_DB_BACKEND", "")
	t.Setenv("ORQUESTA_DB_DSN", "postgres://user:pass@localhost/orquesta")
	t.Setenv("ORQUESTA_DB", "")

	destino := t.TempDir()
	ficheros := []string{
		"2026-03-22_10-00-00_orquesta.postgres.bak",
		"2026-03-22_10-00-01_orquesta.postgres.bak",
		"2026-03-22_10-00-02_orquesta.sqlite.bak",
	}
	for _, nombre := range ficheros {
		if err := os.WriteFile(filepath.Join(destino, nombre), []byte("x"), 0o644); err != nil {
			t.Fatalf("write backup %s: %v", nombre, err)
		}
	}

	if err := aplicarRetencionRespaldo(destino, 1); err != nil {
		t.Fatalf("retencion: %v", err)
	}

	postgresFicheros, err := filepath.Glob(filepath.Join(destino, "*_orquesta.postgres.bak"))
	if err != nil {
		t.Fatalf("glob postgres: %v", err)
	}
	if len(postgresFicheros) != 1 {
		t.Fatalf("retencion postgres inesperada: %v", postgresFicheros)
	}

	if _, err := os.Stat(filepath.Join(destino, "2026-03-22_10-00-02_orquesta.sqlite.bak")); err != nil {
		t.Fatalf("el backup sqlite no deberia tocarse: %v", err)
	}
}

func TestAplicarRetencionRespaldoSQLiteAceptaSufijoLegado(t *testing.T) {
	t.Setenv("ORQUESTA_DB_DRIVER", "")
	t.Setenv("ORQUESTA_DB_BACKEND", "")
	t.Setenv("ORQUESTA_DB_DSN", "")
	t.Setenv("ORQUESTA_DB", filepath.Join(t.TempDir(), "orquesta.db"))

	destino := t.TempDir()
	for _, nombre := range []string{
		"2026-03-22_10-00-00_orquesta.db.bak",
		"2026-03-22_10-00-01_orquesta.sqlite.bak",
		"2026-03-22_10-00-02_orquesta.sqlite.bak",
	} {
		if err := os.WriteFile(filepath.Join(destino, nombre), []byte("x"), 0o644); err != nil {
			t.Fatalf("write backup %s: %v", nombre, err)
		}
	}

	if err := aplicarRetencionRespaldo(destino, 2); err != nil {
		t.Fatalf("retencion sqlite: %v", err)
	}

	ficheros, err := listarRespaldosCompatibles(destino)
	if err != nil {
		t.Fatalf("listar respaldos compatibles: %v", err)
	}
	if len(ficheros) != 2 {
		t.Fatalf("retencion sqlite inesperada: %v", ficheros)
	}
	for _, nombre := range []string{
		filepath.Join(destino, "2026-03-22_10-00-01_orquesta.sqlite.bak"),
		filepath.Join(destino, "2026-03-22_10-00-02_orquesta.sqlite.bak"),
	} {
		if _, err := os.Stat(nombre); err != nil {
			t.Fatalf("respaldo esperado ausente %s: %v", nombre, err)
		}
	}
}
