package db

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func openTestSharedContextDB(t *testing.T) {
	t.Helper()
	Close()
	path := filepath.Join(t.TempDir(), "orquesta.db")
	prev := os.Getenv("ORQUESTA_DB")
	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv ORQUESTA_DB: %v", err)
	}
	t.Cleanup(func() {
		if prev == "" {
			_ = os.Unsetenv("ORQUESTA_DB")
		} else {
			_ = os.Setenv("ORQUESTA_DB", prev)
		}
		Close()
	})
	if err := Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
}

func TestBuildSharedContextSummaryFiltraYResume(t *testing.T) {
	openTestSharedContextDB(t)
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	if _, err := CreateSharedContextItem(&SharedContextItem{
		ProyectoID: &proyectoID,
		Tipo:       "decision",
		Titulo:     "Usar PostgreSQL como backend principal",
		Detalle:    "Evitar contención del control plane",
		Peso:       9,
		Origen:     "arquitectura",
	}); err != nil {
		t.Fatalf("CreateSharedContextItem general: %v", err)
	}
	if _, err := CreateSharedContextItem(&SharedContextItem{
		ProyectoID: &proyectoID,
		Agente:     "Gemma1",
		Tipo:       "followup",
		Titulo:     "Rematar test de scheduler local",
		Detalle:    "No tocar el write-set de premium",
		Peso:       8,
		Origen:     "pipeline_local_parallel",
	}); err != nil {
		t.Fatalf("CreateSharedContextItem agente: %v", err)
	}

	proyecto, err := GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("GetProyecto: %v", err)
	}
	items, resumen := BuildSharedContextSummary("Gemma1", proyecto)
	if len(items) != 2 {
		t.Fatalf("items compartidos inesperados: %+v", items)
	}
	if !strings.Contains(resumen, "Usar PostgreSQL como backend principal") || !strings.Contains(resumen, "Rematar test de scheduler local") {
		t.Fatalf("resumen compartido inesperado: %q", resumen)
	}
}
