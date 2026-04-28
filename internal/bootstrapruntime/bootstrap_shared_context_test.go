package bootstrapruntime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/db"
)

func TestPrepararInyectaContextoCompartidoAunqueNoHayaBootstrapPrevio(t *testing.T) {
	db.Close()
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
		db.Close()
	}()
	if err := db.Open(); err != nil {
		t.Fatalf("db.Open: %v", err)
	}

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: t.TempDir(),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	proyecto, err := db.GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("GetProyecto: %v", err)
	}
	if _, err := db.CreateSharedContextItem(&db.SharedContextItem{
		ProyectoID: &proyectoID,
		Tipo:       "decision",
		Titulo:     "Mantener contrato de dispatch durable",
		Detalle:    "No reabrir receipt débil",
		Peso:       9,
		Origen:     "arquitectura",
	}); err != nil {
		t.Fatalf("CreateSharedContextItem: %v", err)
	}

	resume, state, err := Preparar("Gemma1", proyecto, nil)
	if err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	if state != nil {
		t.Fatalf("no deberia crear bootstrap state si no hay order/mailbox/checkpoint: %+v", state)
	}
	if !strings.Contains(strings.TrimSpace(resume.ResumePayloadJSON), `"shared_context"`) {
		t.Fatalf("resume payload sin shared_context: %s", resume.ResumePayloadJSON)
	}
	if !strings.Contains(strings.TrimSpace(resume.ResumenContinuidad), "Mantener contrato de dispatch durable") {
		t.Fatalf("resumen continuidad sin shared context: %s", resume.ResumenContinuidad)
	}
}
