package orquestaestadovivo

import (
	"reflect"
	"testing"
	"time"
)

func TestConstruirProyeccionCicloVidaV0ProcesoVivoDominaEstadoStale(t *testing.T) {
	ahora := mustTimeV0(t, "2026-07-03T10:00:00Z")
	proyeccion := ConstruirProyeccionCicloVidaV0([]EvidenciaEstadoV0{
		{RunRef: "run-1", Fuente: "run_store", Estado: "blocked", ObservadoEn: "2026-07-03T09:00:00Z"},
		{RunRef: "run-1", Fuente: "process_registry", Scope: ScopeGoalV0, RuntimeIdentityRef: "runtime-1", RuntimeObservado: true, ProcesoVivo: true, ObservadoEn: "2026-07-03T09:59:00Z"},
	}, ahora, time.Hour)

	assertFaseUnicaV0(t, proyeccion, FaseProcesoVivoV0)
}

func TestConstruirProyeccionCicloVidaV0TerminalAceptadoSinProcesoVivo(t *testing.T) {
	ahora := mustTimeV0(t, "2026-07-03T10:00:00Z")
	proyeccion := ConstruirProyeccionCicloVidaV0([]EvidenciaEstadoV0{
		{RunRef: "run-1", Fuente: "receipt", Terminal: true, Aceptado: true, EvidenceRefs: []string{"receipt-ref"}, ObservadoEn: "2026-07-03T09:59:00Z"},
	}, ahora, time.Hour)

	assertFaseUnicaV0(t, proyeccion, FaseTerminalAceptadoV0)
}

func TestConstruirProyeccionCicloVidaV0TerminalReworkSinProcesoVivo(t *testing.T) {
	ahora := mustTimeV0(t, "2026-07-03T10:00:00Z")
	proyeccion := ConstruirProyeccionCicloVidaV0([]EvidenciaEstadoV0{
		{RunRef: "run-1", Fuente: "receipt", Terminal: true, Aceptado: false, EvidenceRefs: []string{"receipt-ref"}, ObservadoEn: "2026-07-03T09:59:00Z"},
	}, ahora, time.Hour)

	assertFaseUnicaV0(t, proyeccion, FaseTerminalReworkV0)
}

func TestConstruirProyeccionCicloVidaV0ProcesoVivoTrasTerminalEsConflicto(t *testing.T) {
	ahora := mustTimeV0(t, "2026-07-03T10:00:00Z")
	proyeccion := ConstruirProyeccionCicloVidaV0([]EvidenciaEstadoV0{
		{RunRef: "run-1", Fuente: "process_snapshot", Scope: ScopeGoalV0, RuntimeIdentityRef: "runtime-1", RuntimeObservado: true, ProcesoVivo: true, ObservadoEn: "2026-07-03T09:59:00Z"},
		{RunRef: "run-1", Fuente: "receipt", Terminal: true, Aceptado: true, EvidenceRefs: []string{"receipt-ref"}, ObservadoEn: "2026-07-03T09:58:00Z"},
	}, ahora, time.Hour)

	assertFaseUnicaV0(t, proyeccion, FaseConflictoV0)
	conflictos := proyeccion.Nodos[0].Conflictos
	if len(conflictos) != 1 {
		t.Fatalf("conflictos len=%d, want 1", len(conflictos))
	}
	if conflictos[0].Codigo != CodigoConflictoProcesoVivoTrasTerminalV0 {
		t.Fatalf("conflicto=%q, want %q", conflictos[0].Codigo, CodigoConflictoProcesoVivoTrasTerminalV0)
	}
}

func TestConstruirProyeccionCicloVidaV0MarkerViejoEsHuerfano(t *testing.T) {
	ahora := mustTimeV0(t, "2026-07-03T10:00:00Z")
	proyeccion := ConstruirProyeccionCicloVidaV0([]EvidenciaEstadoV0{
		{RunRef: "run-1", Fuente: "run_marker", ObservadoEn: "2026-07-03T08:00:00Z"},
	}, ahora, time.Hour)

	assertFaseUnicaV0(t, proyeccion, FaseHuerfanoV0)
}

func TestConstruirProyeccionCicloVidaV0SinEvidenciasEsDesconocido(t *testing.T) {
	ahora := mustTimeV0(t, "2026-07-03T10:00:00Z")
	proyeccion := ConstruirProyeccionCicloVidaV0(nil, ahora, time.Hour)

	assertFaseUnicaV0(t, proyeccion, FaseDesconocidoV0)
	if proyeccion.SchemaVersion != ProyeccionCicloVidaSchemaV0 {
		t.Fatalf("schema=%q, want %q", proyeccion.SchemaVersion, ProyeccionCicloVidaSchemaV0)
	}
}

func TestConstruirProyeccionCicloVidaV0EsDeterminista(t *testing.T) {
	ahora := mustTimeV0(t, "2026-07-03T10:00:00Z")
	entrada := []EvidenciaEstadoV0{
		{RunRef: "run-b", Fuente: "run_marker", ObservadoEn: "2026-07-03T09:50:00Z", EvidenceRefs: []string{"e2", "e1"}},
		{RunRef: "run-a", Fuente: "run_store", Estado: "blocked", ObservadoEn: "2026-07-03T09:45:00Z"},
		{RunRef: "run-b", Fuente: "process_registry", Scope: ScopeGoalV0, RuntimeIdentityRef: "runtime-b", RuntimeObservado: true, ProcesoVivo: true, ObservadoEn: "2026-07-03T09:59:00Z"},
	}
	reordenada := []EvidenciaEstadoV0{entrada[2], entrada[0], entrada[1]}

	primera := ConstruirProyeccionCicloVidaV0(entrada, ahora, time.Hour)
	segunda := ConstruirProyeccionCicloVidaV0(reordenada, ahora, time.Hour)

	if !reflect.DeepEqual(primera, segunda) {
		t.Fatalf("proyeccion no determinista:\nprimera=%#v\nsegunda=%#v", primera, segunda)
	}
}

func TestConstruirProyeccionCicloVidaV0AgrupaPorGoalRefSiFaltaRunRef(t *testing.T) {
	ahora := mustTimeV0(t, "2026-07-03T10:00:00Z")
	proyeccion := ConstruirProyeccionCicloVidaV0([]EvidenciaEstadoV0{
		{GoalRef: "goal-1", Fuente: "receipt", Terminal: true, Aceptado: true, EvidenceRefs: []string{"receipt-ref"}, ObservadoEn: "2026-07-03T09:59:00Z"},
		{GoalRef: "goal-1", Fuente: "run_store", Estado: "running", ObservadoEn: "2026-07-03T09:58:00Z"},
	}, ahora, time.Hour)

	assertFaseUnicaV0(t, proyeccion, FaseTerminalAceptadoV0)
	if proyeccion.Nodos[0].GoalRef != "goal-1" {
		t.Fatalf("goal_ref=%q, want goal-1", proyeccion.Nodos[0].GoalRef)
	}
}

func TestConstruirProyeccionCicloVidaV0DistingueOutboxWaitYProcesoExterno(t *testing.T) {
	ahora := mustTimeV0(t, "2026-07-03T10:00:00Z")
	proyeccion := ConstruirProyeccionCicloVidaV0([]EvidenciaEstadoV0{
		{RunRef: "run-outbox", Fuente: "outbox", Estado: "pending", ObservadoEn: "2026-07-03T09:58:00Z"},
		{RunRef: "run-wait", Fuente: "run_store", Estado: "wait_external", ObservadoEn: "2026-07-03T09:58:00Z"},
		{RunRef: "run-process", Fuente: "process_snapshot", Scope: ScopeGoalV0, RuntimeIdentityRef: "runtime-process", RuntimeObservado: true, ProcesoVivo: true, ObservadoEn: "2026-07-03T09:58:00Z"},
	}, ahora, time.Hour)

	fases := map[string]FaseCicloVidaV0{}
	for _, nodo := range proyeccion.Nodos {
		fases[nodo.RunRef] = nodo.Fase
	}
	if fases["run-outbox"] != FaseSolicitadoV0 {
		t.Fatalf("run-outbox fase=%q, want %q", fases["run-outbox"], FaseSolicitadoV0)
	}
	if fases["run-wait"] != FaseLanzadoV0 {
		t.Fatalf("run-wait fase=%q, want %q", fases["run-wait"], FaseLanzadoV0)
	}
	if fases["run-process"] != FaseProcesoVivoV0 {
		t.Fatalf("run-process fase=%q, want %q", fases["run-process"], FaseProcesoVivoV0)
	}
}

func mustTimeV0(t *testing.T, valor string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339Nano, valor)
	if err != nil {
		t.Fatalf("parse time %q: %v", valor, err)
	}
	return parsed
}

func assertFaseUnicaV0(t *testing.T, proyeccion ProyeccionCicloVidaV0, fase FaseCicloVidaV0) {
	t.Helper()
	if len(proyeccion.Nodos) != 1 {
		t.Fatalf("nodos len=%d, want 1: %#v", len(proyeccion.Nodos), proyeccion.Nodos)
	}
	if proyeccion.Nodos[0].Fase != fase {
		t.Fatalf("fase=%q, want %q", proyeccion.Nodos[0].Fase, fase)
	}
}
