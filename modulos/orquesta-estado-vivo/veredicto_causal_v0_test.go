package orquestaestadovivo

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

func TestDerivarVeredictoCausalV0TablaFuentesEnDesacuerdo(t *testing.T) {
	tests := []struct {
		nombre           string
		runtimeObservado bool
		procesoVivo      bool
		terminal         bool
		wantClase        ClaseVeredictoCausalV0
		wantReason       string
		wantRunning      bool
		wantRepair       bool
	}{
		{
			nombre:           "state running y runtime vivo",
			runtimeObservado: true, procesoVivo: true,
			wantClase: VeredictoRunningConfirmedV0, wantReason: RazonVeredictoRuntimeConfirmadoV0, wantRunning: true,
		},
		{
			nombre:           "resultado durable terminal sin proceso vivo",
			runtimeObservado: true, terminal: true,
			wantClase: VeredictoTerminalByArtifactV0, wantReason: RazonVeredictoResultadoDurableTerminalV0,
		},
		{
			nombre:           "state running y runtime confirma proceso muerto",
			runtimeObservado: true,
			wantClase:        VeredictoProcessDeadStateStaleV0, wantReason: RazonVeredictoProcesoMuertoEstadoStaleV0, wantRepair: true,
		},
		{
			nombre:           "resultado durable terminal y proceso vivo",
			runtimeObservado: true, procesoVivo: true, terminal: true,
			wantClase: VeredictoDivergentNeedsRepairV0, wantReason: RazonVeredictoProcesoVivoTrasTerminalV0, wantRepair: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.nombre, func(t *testing.T) {
			evidencias := []EvidenciaEstadoV0{
				{RunRef: "run-1", GoalRef: "goal-1", Fuente: "goal_state", Estado: "running", EvidenceRefs: []string{"state-ref"}},
				{RunRef: "run-1", GoalRef: "goal-1", Fuente: "runtime_identity", Scope: ScopeGoalV0, RuntimeIdentityRef: "runtime-1", RuntimeObservado: tt.runtimeObservado, ProcesoVivo: tt.procesoVivo, EvidenceRefs: []string{"runtime-ref"}},
			}
			if tt.terminal {
				evidencias = append(evidencias, EvidenciaEstadoV0{RunRef: "run-1", GoalRef: "goal-1", Fuente: "durable_result", Estado: "blocked", Terminal: true, EvidenceRefs: []string{"result-ref"}})
			}

			got := DerivarVeredictoCausalV0(evidencias)
			if got.Clase != tt.wantClase || got.ReasonCode != tt.wantReason {
				t.Fatalf("veredicto=(%q,%q), want (%q,%q): %#v", got.Clase, got.ReasonCode, tt.wantClase, tt.wantReason, got)
			}
			if got.PublicarRunning != tt.wantRunning || got.RequiereReparacion != tt.wantRepair {
				t.Fatalf("running=%t repair=%t, want running=%t repair=%t", got.PublicarRunning, got.RequiereReparacion, tt.wantRunning, tt.wantRepair)
			}
			if got.SchemaVersion != VeredictoCausalSchemaV0 || got.RunRef != "run-1" || got.GoalRef != "goal-1" {
				t.Fatalf("identidad/schema no preservada: %#v", got)
			}
		})
	}
}

func TestDerivarVeredictoCausalV0RunningSinLivenessNoPublicaRunning(t *testing.T) {
	got := DerivarVeredictoCausalV0([]EvidenciaEstadoV0{{
		RunRef: "run-stale", Fuente: "goal_state", Estado: "running", EvidenceRefs: []string{"state-ref"},
	}})

	if got.Clase != VeredictoIndeterminateV0 || got.ReasonCode != RazonVeredictoLivenessNoConfirmadoV0 {
		t.Fatalf("veredicto=%#v, want indeterminate/runtime_liveness_unconfirmed", got)
	}
	if got.PublicarRunning {
		t.Fatal("state running sin identidad runtime viva no puede publicar running")
	}
}

func TestDerivarVeredictoCausalV0EsDeterministaYPreservaRefs(t *testing.T) {
	evidencias := []EvidenciaEstadoV0{
		{RunRef: "run-1", Fuente: "durable_result", Terminal: true, EvidenceRefs: []string{"result-b", "result-a"}},
		{RunRef: "run-1", Fuente: "runtime_identity", Scope: ScopeGoalV0, RuntimeGenerationRef: "generation-1", RuntimeObservado: true, EvidenceRefs: []string{"runtime-a", "result-a"}},
	}
	reordenadas := []EvidenciaEstadoV0{evidencias[1], evidencias[0]}

	primero := DerivarVeredictoCausalV0(evidencias)
	segundo := DerivarVeredictoCausalV0(reordenadas)
	if !reflect.DeepEqual(primero, segundo) {
		t.Fatalf("veredicto no determinista:\nprimero=%#v\nsegundo=%#v", primero, segundo)
	}
	wantRefs := []string{"result-a", "result-b", "runtime-a"}
	if !reflect.DeepEqual(primero.EvidenceRefs, wantRefs) {
		t.Fatalf("evidence_refs=%v, want %v", primero.EvidenceRefs, wantRefs)
	}
}

func TestDerivarVeredictoCausalV0PropCuatroCombinacionesV0(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		procesoVivo := rapid.Bool().Draw(rt, "proceso_vivo")
		terminal := rapid.Bool().Draw(rt, "terminal")
		got := DerivarVeredictoCausalV0([]EvidenciaEstadoV0{
			{RunRef: "run-property", Fuente: "goal_state", Estado: "running"},
			{RunRef: "run-property", Fuente: "runtime_identity", Scope: ScopeGoalV0, RuntimeIdentityRef: "runtime-property", RuntimeObservado: true, ProcesoVivo: procesoVivo},
			{RunRef: "run-property", Fuente: "durable_result", Terminal: terminal, EvidenceRefs: []string{"result-ref"}},
		})

		want := VeredictoProcessDeadStateStaleV0
		switch {
		case procesoVivo && terminal:
			want = VeredictoDivergentNeedsRepairV0
		case procesoVivo:
			want = VeredictoRunningConfirmedV0
		case terminal:
			want = VeredictoTerminalByArtifactV0
		}
		if got.Clase != want {
			rt.Fatalf("proceso_vivo=%t terminal=%t clase=%q, want %q", procesoVivo, terminal, got.Clase, want)
		}
		if got.PublicarRunning != (got.Clase == VeredictoRunningConfirmedV0) {
			rt.Fatalf("clase=%q publicar_running=%t", got.Clase, got.PublicarRunning)
		}
	})
}

func TestDerivarVeredictoCausalV0TablaScopeIdentidadYObservacion(t *testing.T) {
	tests := []struct {
		nombre      string
		runtime     EvidenciaEstadoV0
		terminal    bool
		wantClase   ClaseVeredictoCausalV0
		wantReason  string
		wantRunning bool
	}{
		{
			nombre:    "goal observado vivo con identidad",
			runtime:   EvidenciaEstadoV0{Scope: ScopeGoalV0, RuntimeIdentityRef: "runtime-1", RuntimeObservado: true, ProcesoVivo: true},
			wantClase: VeredictoRunningConfirmedV0, wantReason: RazonVeredictoRuntimeConfirmadoV0, wantRunning: true,
		},
		{
			nombre:    "goal observado muerto con generation",
			runtime:   EvidenciaEstadoV0{Scope: ScopeGoalV0, RuntimeGenerationRef: "generation-1", RuntimeObservado: true},
			wantClase: VeredictoProcessDeadStateStaleV0, wantReason: RazonVeredictoProcesoMuertoEstadoStaleV0,
		},
		{
			nombre:    "goal vivo no observado no confirma running",
			runtime:   EvidenciaEstadoV0{Scope: ScopeGoalV0, RuntimeIdentityRef: "runtime-1", ProcesoVivo: true},
			wantClase: VeredictoIndeterminateV0, wantReason: RazonVeredictoObservacionRuntimeIncompletaV0,
		},
		{
			nombre:    "goal observado sin identidad no es atribuible",
			runtime:   EvidenciaEstadoV0{Scope: ScopeGoalV0, RuntimeObservado: true, ProcesoVivo: true},
			wantClase: VeredictoDivergentNeedsRepairV0, wantReason: RazonVeredictoIdentidadRuntimeFaltanteV0,
		},
		{
			nombre:    "backend compartido vivo no confirma goal",
			runtime:   EvidenciaEstadoV0{Scope: ScopeBackendServiceV0, RuntimeIdentityRef: "backend-1", RuntimeObservado: true, ProcesoVivo: true},
			wantClase: VeredictoIndeterminateV0, wantReason: RazonVeredictoLivenessNoConfirmadoV0,
		},
		{
			nombre: "identidad runtime distinta no confirma goal",
			runtime: EvidenciaEstadoV0{
				Scope: ScopeGoalExecutionV0, RuntimeIdentityRef: "runtime-distinto", RuntimeObservado: true, ProcesoVivo: true,
			},
			wantClase: VeredictoDivergentNeedsRepairV0, wantReason: RazonVeredictoIdentidadNoCoincidenteV0,
		},
		{
			nombre:  "terminal durable prevalece sobre backend vivo",
			runtime: EvidenciaEstadoV0{Scope: ScopeBackendServiceV0, RuntimeIdentityRef: "backend-1", RuntimeObservado: true, ProcesoVivo: true}, terminal: true,
			wantClase: VeredictoTerminalByArtifactV0, wantReason: RazonVeredictoResultadoDurableTerminalV0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.nombre, func(t *testing.T) {
			runtime := tt.runtime
			runtime.RunRef = "run-1"
			runtime.GoalRef = "goal-1"
			runtime.Fuente = "runtime"
			evidencias := []EvidenciaEstadoV0{
				{RunRef: "run-1", GoalRef: "goal-1", Fuente: "goal_state", Estado: "running", Scope: ScopeGoalExecutionV0, RuntimeIdentityRef: "runtime-1"},
				runtime,
			}
			if tt.terminal {
				evidencias = append(evidencias, EvidenciaEstadoV0{RunRef: "run-1", GoalRef: "goal-1", Fuente: "result", Terminal: true, EvidenceRefs: []string{"result-ref"}})
			}
			got := DerivarVeredictoCausalV0(evidencias)
			if got.Clase != tt.wantClase || got.ReasonCode != tt.wantReason || got.PublicarRunning != tt.wantRunning {
				t.Fatalf("got=(%q,%q,running=%t), want=(%q,%q,running=%t): %#v", got.Clase, got.ReasonCode, got.PublicarRunning, tt.wantClase, tt.wantReason, tt.wantRunning, got)
			}
		})
	}
}

func TestDerivarVeredictoCausalV0RunningExigeScopeGoalExecutionLiteralV0(t *testing.T) {
	got := DerivarVeredictoCausalV0([]EvidenciaEstadoV0{
		{RunRef: "run-1", GoalRef: "goal-1", Fuente: "goal_state", Estado: "running"},
		{RunRef: "run-1", GoalRef: "goal-1", Fuente: "runtime", Scope: ScopeEvidenciaEstadoV0("goal"), RuntimeIdentityRef: "runtime-1", RuntimeObservado: true, ProcesoVivo: true},
	})

	if got.Clase != VeredictoIndeterminateV0 || got.PublicarRunning {
		t.Fatalf("scope legacy goal no puede confirmar running: %#v", got)
	}
	if ScopeGoalExecutionV0 != "goal_execution" || ScopeGoalV0 != ScopeGoalExecutionV0 {
		t.Fatalf("scope tipado inesperado: goal_execution=%q alias=%q", ScopeGoalExecutionV0, ScopeGoalV0)
	}
}

func TestDerivarVeredictoCausalV0ProcesoLegacySinIdentidadNoOcultaDivergenciaV0(t *testing.T) {
	got := DerivarVeredictoCausalV0([]EvidenciaEstadoV0{
		{RunRef: "run-legacy", Fuente: "process_snapshot", ProcesoVivo: true},
		{RunRef: "run-legacy", Fuente: "result", Terminal: true, Aceptado: true, EvidenceRefs: []string{"result-ref"}},
	})
	if got.Clase != VeredictoDivergentNeedsRepairV0 || got.ReasonCode != RazonVeredictoIdentidadRuntimeFaltanteV0 {
		t.Fatalf("proceso sin identidad no puede caer a terminal legacy: %#v", got)
	}
	if got.PublicarRunning {
		t.Fatalf("proceso legacy incompleto no confirma running: %#v", got)
	}
}

func TestDerivarVeredictoCausalV0TimeoutEsIndeterminadoAunqueHayaTerminalV0(t *testing.T) {
	got := DerivarVeredictoCausalV0([]EvidenciaEstadoV0{
		{RunRef: "run-timeout", Fuente: "goal_state", Estado: "running"},
		{RunRef: "run-timeout", Fuente: "result", Terminal: true, EvidenceRefs: []string{"result-ref"}},
		{
			RunRef: "run-timeout", Fuente: "runtime", Scope: ScopeGoalExecutionV0,
			RuntimeIdentityRef: "runtime-timeout", RuntimeObservationAttempted: true,
		},
	})

	if got.Clase != VeredictoIndeterminateV0 || got.ReasonCode != RazonVeredictoObservacionRuntimeIncompletaV0 {
		t.Fatalf("timeout no debe cerrar por precedencia legacy: %#v", got)
	}
	if got.PublicarRunning || got.RequiereReparacion {
		t.Fatalf("timeout requiere reobservacion, no running ni reparacion afirmada: %#v", got)
	}
}

func TestDerivarVeredictoCausalV0TerminalSinEvidenciaDurableNoCierraV0(t *testing.T) {
	got := DerivarVeredictoCausalV0([]EvidenciaEstadoV0{
		{RunRef: "run-unproven", Fuente: "result", Terminal: true, Aceptado: true},
	})

	if got.Clase != VeredictoIndeterminateV0 || got.ReasonCode != RazonVeredictoEvidenciaTerminalFaltanteV0 {
		t.Fatalf("terminal sin referencia durable debe quedar indeterminado: %#v", got)
	}
	if got.ResultadoTerminal || got.ResultadoAceptado || !got.RequiereReparacion {
		t.Fatalf("terminal sin referencia no puede publicarse como cierre: %#v", got)
	}
}

func TestDerivarVeredictoCausalV0TerminalMalformadoNoOcultaConflictoDemostradoV0(t *testing.T) {
	got := DerivarVeredictoCausalV0([]EvidenciaEstadoV0{
		{RunRef: "run-conflict-proven", Fuente: "registry", Scope: ScopeGoalExecutionV0, RuntimeIdentityRef: "runtime-1"},
		{RunRef: "run-conflict-proven", Fuente: "snapshot", Scope: ScopeGoalExecutionV0, RuntimeIdentityRef: "runtime-1", RuntimeObservationAttempted: true, RuntimeObservado: true, ProcesoVivo: true},
		{RunRef: "run-conflict-proven", Fuente: "receipt", Terminal: true, EvidenceRefs: []string{"receipt-ref"}},
		{RunRef: "run-conflict-proven", Fuente: "stale_result", Terminal: true},
	})

	if got.Clase != VeredictoDivergentNeedsRepairV0 || got.ReasonCode != RazonVeredictoProcesoVivoTrasTerminalV0 {
		t.Fatalf("terminal malformado no debe ocultar conflicto demostrado: %#v", got)
	}
}

func TestDerivarVeredictoCausalV0AdmiteMultiplesProcesosConIdentidadCoincidenteV0(t *testing.T) {
	got := DerivarVeredictoCausalV0([]EvidenciaEstadoV0{
		{RunRef: "run-multi", Fuente: "registry", Scope: ScopeGoalExecutionV0, RuntimeIdentityRef: "process-a"},
		{RunRef: "run-multi", Fuente: "registry", Scope: ScopeGoalExecutionV0, RuntimeIdentityRef: "process-b"},
		{RunRef: "run-multi", Fuente: "snapshot", Scope: ScopeGoalExecutionV0, RuntimeIdentityRef: "process-a", RuntimeObservationAttempted: true, RuntimeObservado: true, ProcesoVivo: true},
		{RunRef: "run-multi", Fuente: "snapshot", Scope: ScopeGoalExecutionV0, RuntimeIdentityRef: "process-b", RuntimeObservationAttempted: true, RuntimeObservado: true},
	})

	if got.Clase != VeredictoRunningConfirmedV0 || !got.PublicarRunning {
		t.Fatalf("procesos causales del mismo run no son identidades divergentes: %#v", got)
	}
	if !reflect.DeepEqual(got.RuntimeIdentityRefs, []string{"process-a", "process-b"}) {
		t.Fatalf("runtime_identity_refs=%v", got.RuntimeIdentityRefs)
	}
}

func TestConstruirProyeccionCicloVidaV0TransportaVeredictoTipadoJSONEIdempotenteV0(t *testing.T) {
	evidencias := []EvidenciaEstadoV0{
		{RunRef: "run-json", Fuente: "state", Estado: "running"},
		{RunRef: "run-json", Fuente: "registry", Scope: ScopeGoalExecutionV0, RuntimeIdentityRef: "runtime-json"},
		{RunRef: "run-json", Fuente: "snapshot", Scope: ScopeGoalExecutionV0, RuntimeIdentityRef: "runtime-json", RuntimeObservationAttempted: true, RuntimeObservado: true, ProcesoVivo: true},
	}
	primera := ConstruirProyeccionCicloVidaV0(evidencias, ahoraRapidV0(), 0)
	segunda := ConstruirProyeccionCicloVidaV0(evidencias, ahoraRapidV0(), 0)
	if !reflect.DeepEqual(primera, segunda) {
		t.Fatalf("replay no idempotente:\n%#v\n%#v", primera, segunda)
	}
	if len(primera.Nodos) != 1 || primera.Nodos[0].Veredicto.Clase != VeredictoRunningConfirmedV0 {
		t.Fatalf("nodo no transporta veredicto: %#v", primera.Nodos)
	}
	body, err := json.Marshal(primera.Nodos[0])
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	for _, token := range []string{`"veredicto_causal"`, `"class":"running_confirmed"`, `"publish_running":true`} {
		if !strings.Contains(string(body), token) {
			t.Fatalf("JSON sin contrato tipado %s: %s", token, body)
		}
	}
}
