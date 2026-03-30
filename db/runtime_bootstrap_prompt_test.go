package db

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildLaunchBootstrapPromptIncluyeRolOperativoAutonomo(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(t.TempDir(), "repo"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := UpsertProyectoAutonomia(&ProyectoAutonomia{
		ProyectoID:        proyectoID,
		Enabled:           true,
		SupervisorAgente:  "Codex1",
		ReserveSupervisor: true,
		EstadoAutonomia:   AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert autonomia: %v", err)
	}
	if err := ActivarAsignacion("Codex1", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion Codex1: %v", err)
	}
	if err := ActivarAsignacion("Codex2", proyectoID, "worker"); err != nil {
		t.Fatalf("activar asignacion Codex2: %v", err)
	}

	proyecto, err := GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	supervisor, err := GetAgente("Codex1")
	if err != nil {
		t.Fatalf("get agente supervisor: %v", err)
	}
	worker, err := GetAgente("Codex2")
	if err != nil {
		t.Fatalf("get agente worker: %v", err)
	}

	supervisorPrompt := BuildLaunchBootstrapPrompt(supervisor, proyecto, nil, nil, nil, nil, nil, "", "")
	if !strings.Contains(strings.ToLower(supervisorPrompt), "rol operativo en este proyecto: orquestador") {
		t.Fatalf("prompt supervisor sin rol operativo: %q", supervisorPrompt)
	}

	workerPrompt := BuildLaunchBootstrapPrompt(worker, proyecto, nil, nil, nil, nil, nil, "", "")
	if !strings.Contains(workerPrompt, "Supervisor operativo del proyecto: Codex1") {
		t.Fatalf("prompt worker sin supervisor operativo: %q", workerPrompt)
	}
}

func TestSeleccionarSupervisorAutonomiaOperativoUsaSupervisorConfigurado(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(t.TempDir(), "repo"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := UpsertProyectoAutonomia(&ProyectoAutonomia{
		ProyectoID:       proyectoID,
		Enabled:          true,
		SupervisorAgente: "Codex1",
		EstadoAutonomia:  AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert autonomia: %v", err)
	}

	supervisor, err := SeleccionarSupervisorAutonomiaOperativo(proyectoID, "")
	if err != nil {
		t.Fatalf("SeleccionarSupervisorAutonomiaOperativo: %v", err)
	}
	if supervisor == nil || supervisor.Nombre != "Codex1" {
		t.Fatalf("supervisor inesperado: %+v", supervisor)
	}
}
