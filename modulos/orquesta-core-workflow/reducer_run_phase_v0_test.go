package orquestacoreworkflow

import (
	"reflect"
	"testing"
)

func TestApplyEventV0RunStartedCreatesActiveRunWithPhaseCatalog(t *testing.T) {
	event := mustReducerRunStartedEventV0(t, "evt-run-started-reducer-001", 1)

	got, err := ApplyEventV0(OrchestrationRunV0{}, event)
	if err != nil {
		t.Fatalf("apply RunStarted: %v", err)
	}

	if got.SchemaVersion != OrchestrationRunSchemaVersionV0 {
		t.Fatalf("schema_version=%q", got.SchemaVersion)
	}
	if got.RunID != event.RunID {
		t.Fatalf("run_id=%q, want %q", got.RunID, event.RunID)
	}
	if got.ProjectRef != "project:ventas" || got.AppSpecRef != "appspec:req-001" {
		t.Fatalf("refs inesperadas: project=%q app_spec=%q", got.ProjectRef, got.AppSpecRef)
	}
	if got.Status != OrchestrationRunStatusActiveV0 {
		t.Fatalf("status=%q, want %q", got.Status, OrchestrationRunStatusActiveV0)
	}
	if got.CurrentPhase != OrchestrationPhaseDescubrimientoV0 {
		t.Fatalf("current_phase=%q, want %q", got.CurrentPhase, OrchestrationPhaseDescubrimientoV0)
	}
	if got.LastEventID != event.EventID {
		t.Fatalf("last_event_id=%q, want %q", got.LastEventID, event.EventID)
	}
	if len(got.Phases) != len(OrchestrationPhaseCatalogV0()) {
		t.Fatalf("phases=%d, want catalogo completo", len(got.Phases))
	}
	for _, phase := range got.Phases {
		if phase.Status != OrchestrationPhaseStatusPendingV0 {
			t.Fatalf("fase %s inicia con status=%q", phase.ID, phase.Status)
		}
	}
	assertReducerRunValidV0(t, got)
}

func TestApplyEventV0PhaseOpenedMarksSingleActivePhase(t *testing.T) {
	run := mustReducerStartedRunV0(t)
	run = mustApplyReducerEventV0(t, run, mustReducerPhaseOpenedEventV0(
		t,
		"evt-phase-opened-reducer-001",
		2,
		OrchestrationPhaseDescubrimientoV0,
	))
	event := mustReducerPhaseOpenedEventV0(
		t,
		"evt-phase-opened-reducer-002",
		3,
		OrchestrationPhaseProgramacionV0,
	)

	got, err := ApplyEventV0(run, event)
	if err != nil {
		t.Fatalf("apply PhaseOpened: %v", err)
	}

	if got.CurrentPhase != OrchestrationPhaseProgramacionV0 {
		t.Fatalf("current_phase=%q, want %q", got.CurrentPhase, OrchestrationPhaseProgramacionV0)
	}
	if got.LastEventID != event.EventID {
		t.Fatalf("last_event_id=%q, want %q", got.LastEventID, event.EventID)
	}
	active := activePhaseIDsV0(got)
	if len(active) != 1 || active[0] != OrchestrationPhaseProgramacionV0 {
		t.Fatalf("fases activas=%v, want solo %s", active, OrchestrationPhaseProgramacionV0)
	}
	opened := reducerPhaseByIDV0(t, got, OrchestrationPhaseProgramacionV0)
	if opened.OpenedAt != event.OccurredAt {
		t.Fatalf("opened_at=%q, want %q", opened.OpenedAt, event.OccurredAt)
	}
	assertReducerRunValidV0(t, got)
}

func TestApplyEventV0PhaseReopenPreservesOriginalOpenedAt(t *testing.T) {
	run := mustReducerStartedRunV0(t)
	first := mustApplyReducerEventV0(t, run, mustReducerPhaseOpenedEventV0(
		t,
		"evt-phase-opened-reducer-original",
		2,
		OrchestrationPhaseProgramacionV0,
	))
	original := reducerPhaseByIDV0(t, first, OrchestrationPhaseProgramacionV0).OpenedAt
	other := mustApplyReducerEventV0(t, first, mustReducerPhaseOpenedEventV0(
		t,
		"evt-phase-opened-reducer-other",
		3,
		OrchestrationPhaseRevisionV0,
	))
	reopened := mustApplyReducerEventV0(t, other, mustReducerPhaseOpenedEventV0(
		t,
		"evt-phase-opened-reducer-reopen",
		4,
		OrchestrationPhaseProgramacionV0,
	))

	got := reducerPhaseByIDV0(t, reopened, OrchestrationPhaseProgramacionV0)
	if got.OpenedAt != original {
		t.Fatalf("opened_at reescrito=%q, want original %q", got.OpenedAt, original)
	}
}

func TestApplyEventV0DoesNotMutateInputSlices(t *testing.T) {
	current := mustReducerStartedRunV0(t)
	before := cloneRunForReducerV0(current)
	event := mustReducerPhaseOpenedEventV0(
		t,
		"evt-phase-opened-reducer-003",
		2,
		OrchestrationPhaseBrainstormingArquitecturaV0,
	)

	got, err := ApplyEventV0(current, event)
	if err != nil {
		t.Fatalf("apply PhaseOpened: %v", err)
	}

	if !reflect.DeepEqual(current, before) {
		t.Fatalf("estado de entrada mutado: before=%+v after=%+v", before, current)
	}
	if reflect.DeepEqual(got, current) {
		t.Fatalf("estado nuevo no refleja apertura de fase")
	}
}
