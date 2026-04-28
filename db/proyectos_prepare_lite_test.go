package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetProyectoPrepareLiteUsaReadOnlySQLiteIndependienteDelHandlePrincipal(t *testing.T) {
	Close()
	path := filepath.Join(t.TempDir(), "orquesta.db")
	prev := os.Getenv("ORQUESTA_DB")
	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv ORQUESTA_DB: %v", err)
	}
	defer func() {
		if prev == "" {
			_ = os.Unsetenv("ORQUESTA_DB")
		} else {
			_ = os.Setenv("ORQUESTA_DB", prev)
		}
		Close()
	}()
	if err := Open(); err != nil {
		t.Fatalf("open: %v", err)
	}
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
	if proyectoID <= 0 {
		t.Fatalf("id de proyecto invalido: %d", proyectoID)
	}
	Close()
	proyecto, err := GetProyectoPrepareLite("orquestador")
	if err != nil {
		t.Fatalf("GetProyectoPrepareLite: %v", err)
	}
	if proyecto == nil {
		t.Fatalf("GetProyectoPrepareLite devolvio nil")
	}
	if proyecto.Slug != "orquestador" {
		t.Fatalf("slug inesperado: %q", proyecto.Slug)
	}
}
