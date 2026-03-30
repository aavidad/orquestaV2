package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"orquesta/db"
	"orquesta/propuestasapp"
)

func TestConstruirAgenteTickOutputPriorizaSupervisionParaSupervisorOperativo(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		SupervisorAgente:     "Codex1",
		ReserveSupervisor:    true,
		AutoCreateTasks:      true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion supervisor: %v", err)
	}
	if err := db.ActivarAsignacion("Codex2", proyectoID, "worker"); err != nil {
		t.Fatalf("activar asignacion worker: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion supervisor: %v", err)
	}
	if _, _, err := propuestasService.Create(propuestasapp.CreateProposalInput{
		Codigo:       "PROP-1",
		Titulo:       "Propuesta de prueba",
		Descripcion:  "detalle",
		Tipo:         "otro",
		Proyecto:     "orquestador",
		PropuestoPor: "tester",
		Distribuidor: "orquesta",
	}); err != nil {
		t.Fatalf("crear propuesta: %v", err)
	}

	proyecto, err := db.GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	out, err := construirAgenteTickOutput("Codex1", proyecto, nil, 100)
	if err != nil {
		t.Fatalf("construir tick: %v", err)
	}
	if out.AccionRecomendada != "supervisar_proyecto" {
		t.Fatalf("accion recomendada inesperada: %+v", out)
	}
	if !strings.Contains(strings.ToLower(out.Motivo), "supervisor operativo") {
		t.Fatalf("motivo inesperado: %q", out.Motivo)
	}
}
