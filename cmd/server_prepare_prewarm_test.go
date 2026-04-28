package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"orquesta/db"
)

func TestLoadServerAutobootstrapConfigDisabledByDefault(t *testing.T) {
	prepararDBTemporalCmd(t)

	cfg := loadServerAutobootstrapConfig()
	if cfg.Enabled {
		t.Fatalf("autobootstrap no deberia activarse por defecto: %+v", cfg)
	}
}

func TestServerPrepareContextAgentsIncluyeSupervisorYSoloWorkersCalientes(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	t.Setenv("PWD", tmp)

	projectID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "repo"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.RegistrarAgente("Codex1", "admin"); err != nil {
		t.Fatalf("registrar supervisor: %v", err)
	}
	if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar worker caliente: %v", err)
	}
	if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar worker frio: %v", err)
	}
	if err := db.ActivarAsignacion("Codex2", projectID, "test_hot"); err != nil {
		t.Fatalf("activar asignacion caliente: %v", err)
	}

	agents := serverPrepareContextAgents(serverAutobootstrapConfig{
		Enabled:         true,
		ProjectSlug:     "orquestador",
		SupervisorAgent: "Codex1",
		WorkerAgents:    []string{"Codex2", "Codex3"},
	})
	got := strings.Join(agents, ",")
	if got != "Codex1,Codex2" {
		t.Fatalf("agentes prewarm inesperados: %q", got)
	}
}
