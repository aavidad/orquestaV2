package orquestaestadovivo

import (
	"reflect"
	"strconv"
	"testing"
	"time"

	"pgregory.net/rapid"
)

func TestConstruirProyeccionCicloVidaV0PropDeterminismoBajoPermutacionV0(t *testing.T) {
	// Invariant: reordenar evidencias no cambia la proyeccion publica.
	rapid.Check(t, func(rt *rapid.T) {
		evidencias := rapid.SliceOfN(evidenciaEstadoRapidV0(), 0, 8).Draw(rt, "evidencias")
		permutadas := append([]EvidenciaEstadoV0(nil), evidencias...)
		for i, j := 0, len(permutadas)-1; i < j; i, j = i+1, j-1 {
			permutadas[i], permutadas[j] = permutadas[j], permutadas[i]
		}

		primera := ConstruirProyeccionCicloVidaV0(evidencias, ahoraRapidV0(), time.Hour)
		segunda := ConstruirProyeccionCicloVidaV0(permutadas, ahoraRapidV0(), time.Hour)
		if !reflect.DeepEqual(primera, segunda) {
			rt.Fatalf("proyeccion no determinista:\nprimera=%#v\nsegunda=%#v", primera, segunda)
		}
	})
}

func TestConstruirProyeccionCicloVidaV0PropConflictoNoSilenciosoV0(t *testing.T) {
	// Invariant: proceso vivo y terminal causal de la misma run siempre declaran conflicto.
	rapid.Check(t, func(rt *rapid.T) {
		runRef := "run-conflict-" + strconv.Itoa(rapid.IntRange(0, 999).Draw(rt, "run"))
		aceptado := rapid.Bool().Draw(rt, "aceptado")
		proyeccion := ConstruirProyeccionCicloVidaV0([]EvidenciaEstadoV0{
			{RunRef: runRef, Fuente: rapid.SampledFrom([]string{"process_registry", "process_snapshot"}).Draw(rt, "proceso_fuente"), ProcesoVivo: true, ObservadoEn: "2026-07-03T09:59:00Z"},
			{RunRef: runRef, Fuente: rapid.SampledFrom([]string{"receipt", "review"}).Draw(rt, "terminal_fuente"), Terminal: true, Aceptado: aceptado, ObservadoEn: "2026-07-03T09:58:00Z"},
		}, ahoraRapidV0(), time.Hour)

		if len(proyeccion.Nodos) != 1 || proyeccion.Nodos[0].Fase != FaseConflictoV0 {
			rt.Fatalf("fase=%#v, want conflicto", proyeccion.Nodos)
		}
		if len(proyeccion.Nodos[0].Conflictos) != 1 ||
			proyeccion.Nodos[0].Conflictos[0].Codigo != CodigoConflictoProcesoVivoTrasTerminalV0 {
			rt.Fatalf("conflictos=%#v, want proceso_vivo_tras_terminal", proyeccion.Nodos[0].Conflictos)
		}
	})
}

func TestConstruirProyeccionCicloVidaV0PropTerminalAceptadoDominaPrecedenciasMenoresV0(t *testing.T) {
	// Invariant: una terminal aceptada no se degrada por evidencias no vivas ni no terminales.
	rapid.Check(t, func(rt *rapid.T) {
		runRef := "run-terminal-" + strconv.Itoa(rapid.IntRange(0, 999).Draw(rt, "run"))
		evidencias := []EvidenciaEstadoV0{{
			RunRef:      runRef,
			Fuente:      "receipt",
			Terminal:    true,
			Aceptado:    true,
			ObservadoEn: "2026-07-03T09:59:00Z",
		}}
		evidencias = append(evidencias, evidenciaMenorPrecedenciaRapidV0(runRef).Draw(rt, "menor"))

		proyeccion := ConstruirProyeccionCicloVidaV0(evidencias, ahoraRapidV0(), time.Hour)
		if len(proyeccion.Nodos) != 1 || proyeccion.Nodos[0].Fase != FaseTerminalAceptadoV0 {
			rt.Fatalf("fase=%#v, want terminal_aceptado", proyeccion.Nodos)
		}
	})
}

func TestConstruirProyeccionCicloVidaV0PropDistingueOutboxWaitYProcesoV0(t *testing.T) {
	// Invariant: outbox pendiente, wait_external y proceso vivo producen fases distintas.
	rapid.Check(t, func(rt *rapid.T) {
		suffix := strconv.Itoa(rapid.IntRange(0, 999).Draw(rt, "suffix"))
		proyeccion := ConstruirProyeccionCicloVidaV0([]EvidenciaEstadoV0{
			{RunRef: "run-outbox-" + suffix, Fuente: rapid.SampledFrom([]string{"outbox", "queue"}).Draw(rt, "outbox_fuente"), Estado: rapid.SampledFrom([]string{"pending", "queued", "pendiente"}).Draw(rt, "outbox_estado"), ObservadoEn: "2026-07-03T09:58:00Z"},
			{RunRef: "run-wait-" + suffix, Fuente: "run_store", Estado: rapid.SampledFrom([]string{"wait_external", "external_wait", "lanzado"}).Draw(rt, "wait_estado"), ObservadoEn: "2026-07-03T09:58:00Z"},
			{RunRef: "run-process-" + suffix, Fuente: "process_snapshot", ProcesoVivo: true, ObservadoEn: "2026-07-03T09:58:00Z"},
		}, ahoraRapidV0(), time.Hour)

		fases := map[string]FaseCicloVidaV0{}
		for _, nodo := range proyeccion.Nodos {
			fases[nodo.RunRef] = nodo.Fase
		}
		if fases["run-outbox-"+suffix] != FaseSolicitadoV0 ||
			fases["run-wait-"+suffix] != FaseLanzadoV0 ||
			fases["run-process-"+suffix] != FaseProcesoVivoV0 {
			rt.Fatalf("fases=%#v", fases)
		}
	})
}

func evidenciaEstadoRapidV0() *rapid.Generator[EvidenciaEstadoV0] {
	return rapid.Custom(func(rt *rapid.T) EvidenciaEstadoV0 {
		terminal := rapid.Bool().Draw(rt, "terminal")
		proceso := rapid.Bool().Draw(rt, "proceso")
		return EvidenciaEstadoV0{
			RunRef:          rapid.SampledFrom([]string{"", "run-a", "run-b", "run-c"}).Draw(rt, "run_ref"),
			GoalRef:         rapid.SampledFrom([]string{"", "goal-a", "goal-b"}).Draw(rt, "goal_ref"),
			ExternalGoalRef: rapid.SampledFrom([]string{"", "external-a", "external-b"}).Draw(rt, "external_goal_ref"),
			Fuente:          rapid.SampledFrom([]string{"run_marker", "run_store", "process_registry", "process_snapshot", "receipt", "outbox", "queue"}).Draw(rt, "fuente"),
			Estado:          rapid.SampledFrom([]string{"", "blocked", "pending", "queued", "wait_external", "delivery", "completed"}).Draw(rt, "estado"),
			ProcesoVivo:     proceso,
			Terminal:        terminal,
			Aceptado:        terminal && rapid.Bool().Draw(rt, "aceptado"),
			ObservadoEn:     rapid.SampledFrom([]string{"", "2026-07-03T08:00:00Z", "2026-07-03T09:58:00Z", "2026-07-03T09:59:00Z"}).Draw(rt, "observado_en"),
			EvidenceRefs:    rapid.SliceOfN(rapid.SampledFrom([]string{"evidence-a", "evidence-b", "evidence-c"}), 0, 3).Draw(rt, "evidence_refs"),
		}
	})
}

func evidenciaMenorPrecedenciaRapidV0(runRef string) *rapid.Generator[EvidenciaEstadoV0] {
	return rapid.Custom(func(rt *rapid.T) EvidenciaEstadoV0 {
		return rapid.SampledFrom([]EvidenciaEstadoV0{
			{RunRef: runRef, Fuente: "run_store", Estado: "blocked", ObservadoEn: "2026-07-03T09:57:00Z"},
			{RunRef: runRef, Fuente: "receipt", Estado: "delivery", EvidenceRefs: []string{"delivery-ref"}, ObservadoEn: "2026-07-03T09:57:00Z"},
			{RunRef: runRef, Fuente: "outbox", Estado: "pending", ObservadoEn: "2026-07-03T09:57:00Z"},
			{RunRef: runRef, Fuente: "run_store", Estado: "wait_external", ObservadoEn: "2026-07-03T09:57:00Z"},
		}).Draw(rt, "evidencia")
	})
}

func ahoraRapidV0() time.Time {
	return time.Date(2026, 7, 3, 10, 0, 0, 0, time.UTC)
}
