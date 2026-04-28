package supervisionapp

import "testing"

func TestBuildResidentSupervisorEventDrivenPolicyInputNormalizaPerfilCanonico(t *testing.T) {
	got, err := BuildResidentSupervisorEventDrivenPolicyInput(PolicyInput{
		ObjetivoGeneral:      "  cerrar backlog  ",
		DefinitionOfDoneJSON: "  {\"tests\":\"green\"}  ",
		MaxWorkers:           2,
		SupervisorAgente:     "  Codex1  ",
		ReviewerAgente:       "  Codex2  ",
		ReviewRequired:       true,
		EstadoAutonomia:      "  activo  ",
	})
	if err != nil {
		t.Fatalf("BuildResidentSupervisorEventDrivenPolicyInput error: %v", err)
	}
	if !got.Enabled {
		t.Fatal("la politica residente debe quedar enabled")
	}
	if !got.ReserveSupervisor {
		t.Fatal("la politica residente debe reservar supervisor")
	}
	if !got.AutoCreateTasks {
		t.Fatal("la politica residente debe activar auto_create_tasks")
	}
	if !got.ReserveReviewer {
		t.Fatal("review_required debe reservar reviewer")
	}
	if got.SupervisorAgente != "Codex1" || got.ReviewerAgente != "Codex2" {
		t.Fatalf("agentes normalizados inesperados: %+v", got)
	}
	if got.ObjetivoGeneral != "cerrar backlog" {
		t.Fatalf("objetivo_general inesperado: %q", got.ObjetivoGeneral)
	}
	if got.DefinitionOfDoneJSON != "{\"tests\":\"green\"}" {
		t.Fatalf("definition_of_done_json inesperado: %q", got.DefinitionOfDoneJSON)
	}
	if got.EstadoAutonomia != AutonomiaProyectoActiva {
		t.Fatalf("estado_autonomia inesperado: %q", got.EstadoAutonomia)
	}
}

func TestBuildResidentSupervisorEventDrivenPolicyInputCompletaDefaultsMinimos(t *testing.T) {
	got, err := BuildResidentSupervisorEventDrivenPolicyInput(PolicyInput{
		SupervisorAgente: "Codex1",
		MaxWorkers:       0,
	})
	if err != nil {
		t.Fatalf("BuildResidentSupervisorEventDrivenPolicyInput error: %v", err)
	}
	if got.DefinitionOfDoneJSON != "{}" {
		t.Fatalf("definition_of_done_json default inesperado: %q", got.DefinitionOfDoneJSON)
	}
	if got.EstadoAutonomia != AutonomiaProyectoActiva {
		t.Fatalf("estado_autonomia default inesperado: %q", got.EstadoAutonomia)
	}
	if !got.ReserveSupervisor || !got.AutoCreateTasks || !got.Enabled {
		t.Fatalf("defaults residentes inesperados: %+v", got)
	}
}

func TestBuildResidentSupervisorEventDrivenPolicyInputRechazaSupervisorVacio(t *testing.T) {
	if _, err := BuildResidentSupervisorEventDrivenPolicyInput(PolicyInput{}); err == nil {
		t.Fatal("deberia exigir supervisor_agente")
	}
}

func TestBuildResidentSupervisorEventDrivenPolicyInputRechazaReviewerFaltanteSiReviewEsObligatoria(t *testing.T) {
	if _, err := BuildResidentSupervisorEventDrivenPolicyInput(PolicyInput{
		SupervisorAgente: "Codex1",
		ReviewRequired:   true,
	}); err == nil {
		t.Fatal("deberia exigir reviewer_agente cuando review_required=true")
	}
}

func TestBuildResidentSupervisorEventDrivenPolicyInputRechazaMaxWorkersNegativo(t *testing.T) {
	if _, err := BuildResidentSupervisorEventDrivenPolicyInput(PolicyInput{
		SupervisorAgente: "Codex1",
		MaxWorkers:       -1,
	}); err == nil {
		t.Fatal("deberia rechazar max_workers negativo")
	}
}
