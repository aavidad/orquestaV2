package orquestaruncoordinator

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func TestCoordinateRunsTickChoosesHighestPriorityV0(t *testing.T) {
	deps := coordinatorDepsV0([]orquestarunqueue.RunSchedulingCandidateV0{
		candidateV0("run-low", "app", 1),
		candidateV0("run-high", "app", 9),
	})

	result, err := CoordinateRunsTickV0(context.Background(), deps, tickCommandV0(1))
	if err != nil {
		t.Fatalf("coordinate tick: %v", err)
	}

	assertRunRefsV0(t, executionRefsV0(result.Executions), []string{"run-high"})
	assertRunRefsV0(t, rankedRefsV0(result.Ranked), []string{"run-high", "run-low"})
}

func TestCoordinateRunsTickPropagaMetadataCausalDeRescateV0(t *testing.T) {
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	group := orquestarunqueue.RunQueueAttemptGroupV0{
		ConsumerRef:  "consumer",
		ObjectiveRef: "objective",
		WorkItemRef:  "topic-001",
		WriteSetRefs: []string{"topic/001"},
	}
	deps := coordinatorDepsV0([]orquestarunqueue.RunSchedulingCandidateV0{
		{
			RunRef:        "run-original",
			AppRef:        "app",
			Status:        "queued",
			PriorityScore: 1,
			UpdatedAt:     now.Add(-20 * time.Minute),
			AttemptGroup:  group,
		},
		{
			RunRef:           "run-rescue",
			AppRef:           "app",
			Status:           "queued",
			PriorityScore:    9,
			UpdatedAt:        now,
			AttemptGroup:     group,
			ParentRunRef:     "run-original",
			SupersedesRunRef: "run-original",
			RescueReason:     "estado_incierto",
		},
	})

	result, err := CoordinateRunsTickV0(context.Background(), deps, tickCommandV0(1))
	if err != nil {
		t.Fatalf("coordinate tick: %v", err)
	}

	if len(result.Executions) != 1 ||
		result.Executions[0].RunRef != "run-rescue" ||
		result.Executions[0].ActiveAttemptRef != "run-rescue" ||
		result.Executions[0].ParentRunRef != "run-original" ||
		result.Ranked[0].ActiveAttemptRef != "run-rescue" {
		t.Fatalf("result=%+v", result)
	}
	drainer := deps.Drainer.(*fakeDrainerV0)
	if len(drainer.requests) != 1 ||
		drainer.requests[0].ActiveAttemptRef != "run-rescue" ||
		drainer.requests[0].SupersedesRunRef != "run-original" ||
		drainer.requests[0].AttemptGroup.WorkItemRef != "topic-001" {
		t.Fatalf("requests=%+v", drainer.requests)
	}
}

func TestCoordinateRunsTickNoRankeaNiDrenaIntentoSupersededV0(t *testing.T) {
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	supersessionGroup := orquestarunqueue.RunQueueAttemptGroupV0{GroupRef: "attempt-group-supersession"}
	unsupersededGroup := orquestarunqueue.RunQueueAttemptGroupV0{GroupRef: "attempt-group-no-supersession"}
	deps := coordinatorDepsV0([]orquestarunqueue.RunSchedulingCandidateV0{
		{
			RunRef:        "run-superseded",
			AppRef:        "app",
			Status:        orquestarunqueue.RunStatusReadyV0,
			PriorityScore: 100,
			UpdatedAt:     now.Add(-time.Hour),
			AttemptGroup:  supersessionGroup,
		},
		{
			RunRef:           "run-active-rescue",
			AppRef:           "app",
			Status:           orquestarunqueue.RunStatusReadyV0,
			PriorityScore:    5,
			UpdatedAt:        now,
			AttemptGroup:     supersessionGroup,
			ParentRunRef:     "run-superseded",
			SupersedesRunRef: "run-superseded",
		},
		{
			RunRef:        "run-no-supersession-high",
			AppRef:        "app",
			Status:        orquestarunqueue.RunStatusReadyV0,
			PriorityScore: 9,
			UpdatedAt:     now,
			AttemptGroup:  unsupersededGroup,
		},
		{
			RunRef:        "run-no-supersession-low",
			AppRef:        "app",
			Status:        orquestarunqueue.RunStatusReadyV0,
			PriorityScore: 7,
			UpdatedAt:     now,
			AttemptGroup:  unsupersededGroup,
		},
	})
	command := tickCommandV0(3)
	command.OccurredAt = now
	command.RankingPolicy = orquestarunqueue.DefaultRunQueueRankingPolicyV0(now)

	result, err := CoordinateRunsTickV0(context.Background(), deps, command)
	if err != nil {
		t.Fatalf("coordinate tick: %v", err)
	}

	assertRunRefsV0(t, rankedRefsV0(result.Ranked), []string{
		"run-no-supersession-high",
		"run-no-supersession-low",
		"run-active-rescue",
	})
	assertRunRefsV0(t, executionRefsV0(result.Executions), []string{
		"run-no-supersession-high",
		"run-no-supersession-low",
		"run-active-rescue",
	})
	drainer := deps.Drainer.(*fakeDrainerV0)
	if len(drainer.requests) != 3 {
		t.Fatalf("drain requests=%+v", drainer.requests)
	}
	for _, request := range drainer.requests {
		if request.RunRef == "run-superseded" {
			t.Fatalf("superseded attempt reached drainer: requests=%+v", drainer.requests)
		}
	}
}

func TestCoordinateRunsTickResuelveSupersessionAntesDeQueueLimitV0(t *testing.T) {
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	queue := &fakeQueueReaderV0{
		applyLimit: true,
		candidates: []orquestarunqueue.RunSchedulingCandidateV0{
			candidateWithUpdatedAtV0("run-original", "app", 100, now.Add(-time.Hour)),
			{
				RunRef:           "run-rescue",
				AppRef:           "app",
				Status:           orquestarunqueue.RunStatusReadyV0,
				PriorityScore:    9,
				UpdatedAt:        now,
				ParentRunRef:     "run-original",
				SupersedesRunRef: "run-original",
			},
			candidateWithUpdatedAtV0("run-other", "app", 1, now),
		},
	}
	drainer := &fakeDrainerV0{}
	deps := RunCoordinatorDepsV0{QueueReader: queue, Drainer: drainer}
	command := tickCommandV0(2)
	command.QueueLimit = 1
	command.OccurredAt = now
	command.RankingPolicy = orquestarunqueue.DefaultRunQueueRankingPolicyV0(now)

	result, err := CoordinateRunsTickV0(context.Background(), deps, command)
	if err != nil {
		t.Fatalf("coordinate tick: %v", err)
	}

	if queue.request.Limit != 0 {
		t.Fatalf("reader limit=%d, want complete read", queue.request.Limit)
	}
	assertRunRefsV0(t, rankedRefsV0(result.Ranked), []string{"run-rescue"})
	assertRunRefsV0(t, executionRefsV0(result.Executions), []string{"run-rescue"})
	if len(result.Ranked) != command.QueueLimit {
		t.Fatalf("ranked output exceeds queue limit: %+v", result.Ranked)
	}
	if len(drainer.requests) != 1 || drainer.requests[0].RunRef != "run-rescue" {
		t.Fatalf("drain requests=%+v", drainer.requests)
	}
	for _, request := range drainer.requests {
		if request.RunRef == "run-original" {
			t.Fatalf("superseded original reached drainer: requests=%+v", drainer.requests)
		}
	}
}

func TestFilterSupersededAttemptCandidatesV0NoDependeDelIntentoActivoV0(t *testing.T) {
	candidates := []orquestarunqueue.RunSchedulingCandidateV0{
		{RunRef: "run-original", AttemptGroup: orquestarunqueue.RunQueueAttemptGroupV0{GroupRef: "group-chain"}, PriorityScore: 100},
		{RunRef: "run-rescue", SupersedesRunRef: "run-original", AttemptGroup: orquestarunqueue.RunQueueAttemptGroupV0{GroupRef: "group-chain"}, PriorityScore: 1},
		{RunRef: "run-other-active", AttemptGroup: orquestarunqueue.RunQueueAttemptGroupV0{GroupRef: "group-chain"}, PriorityScore: 200},
	}

	filtered := filterSupersededAttemptCandidatesV0(candidates)
	refs := make([]string, 0, len(filtered))
	for _, candidate := range filtered {
		refs = append(refs, candidate.RunRef)
	}
	assertRunRefsV0(t, refs, []string{"run-rescue", "run-other-active"})
}

func TestFilterSupersededAttemptCandidatesV0IgnoraEnlaceDeOtroGrupoV0(t *testing.T) {
	candidates := []orquestarunqueue.RunSchedulingCandidateV0{
		{RunRef: "run-group-a", AttemptGroup: orquestarunqueue.RunQueueAttemptGroupV0{GroupRef: "group-a"}},
		{
			RunRef: "run-group-b", ParentRunRef: "run-group-a", SupersedesRunRef: "run-group-a",
			AttemptGroup: orquestarunqueue.RunQueueAttemptGroupV0{GroupRef: "group-b"},
		},
	}

	filtered := filterSupersededAttemptCandidatesV0(candidates)
	refs := make([]string, 0, len(filtered))
	for _, candidate := range filtered {
		refs = append(refs, candidate.RunRef)
	}
	assertRunRefsV0(t, refs, []string{"run-group-a", "run-group-b"})
}

func TestFilterSupersededAttemptCandidatesV0LegacyExigeParentExactoV0(t *testing.T) {
	candidates := []orquestarunqueue.RunSchedulingCandidateV0{
		{RunRef: "run-legacy-a"},
		{RunRef: "run-legacy-b", ParentRunRef: "run-distinto", SupersedesRunRef: "run-legacy-a"},
	}

	if got := filterSupersededAttemptCandidatesV0(candidates); len(got) != 2 {
		t.Fatalf("supersesion legacy sin parent causal elimino candidato: %+v", got)
	}
}

func TestCoordinateRunsTickPreservaMetadataCausalAlRotarColaV0(t *testing.T) {
	now := time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC)
	claim := orquestarunqueue.WorksetClaimV0{
		SchemaVersion: orquestarunqueue.WorksetClaimSchemaVersionV0,
		ClaimRef:      "claim-ref-run-preserve",
		RunRef:        "run-preserve",
		TaskRef:       "task-preserve",
		WriteSet: []orquestarunqueue.ScopeRefV0{{
			Ref: "modulos/orquesta-run-coordinator",
		}},
	}
	candidate := candidateWithUpdatedAtV0("run-preserve", "app", 9, now.Add(-time.Minute))
	candidate.FairnessGroupRef = "fairness-preserve"
	candidate.AttemptGroup = orquestarunqueue.RunQueueAttemptGroupV0{
		GroupRef:     "attempt-group-preserve",
		ConsumerRef:  "consumer-preserve",
		ObjectiveRef: "objective-preserve",
		WorkItemRef:  "task-preserve",
		WriteSetRefs: []string{"modulos/orquesta-run-coordinator"},
	}
	candidate.ParentRunRef = "run-parent"
	candidate.SupersedesRunRef = "run-superseded"
	candidate.RescueReason = "reconcile_stale_state"
	candidate.WorksetClaims = []orquestarunqueue.WorksetClaimV0{claim}
	queue := &fakeQueueStoreV0{candidates: []orquestarunqueue.RunSchedulingCandidateV0{candidate}}
	drainer := &fakeDrainerV0{
		results: map[string]RunDrainResultV0{
			"run-preserve": {
				RunRef:      "run-preserve",
				AppRef:      "app",
				Outcome:     "quiescent",
				QueueStatus: orquestarunqueue.RunStatusDeliveredV0,
			},
		},
	}
	deps := RunCoordinatorDepsV0{
		QueueReader:   queue,
		QueueUpdater:  queue,
		ControlReader: &fakeControlReaderV0{states: map[string]orquestaruncontrol.RunControlStateV0{}, missing: map[string]bool{}},
		Drainer:       drainer,
	}
	command := tickCommandV0(1)
	command.QueueRef = "queue-preserve"
	command.OccurredAt = now
	command.RankingPolicy = orquestarunqueue.DefaultRunQueueRankingPolicyV0(now)

	result, err := CoordinateRunsTickV0(context.Background(), deps, command)
	if err != nil {
		t.Fatalf("coordinate tick: %v", err)
	}

	if len(result.Executions) != 1 || len(queue.commands) != 1 {
		t.Fatalf("result=%+v commands=%+v", result, queue.commands)
	}
	written := queue.commands[0]
	if written.Status != orquestarunqueue.RunStatusDeliveredV0 ||
		written.FairnessGroupRef != candidate.FairnessGroupRef ||
		written.AttemptGroup.WorkItemRef != candidate.AttemptGroup.WorkItemRef ||
		written.ParentRunRef != candidate.ParentRunRef ||
		written.SupersedesRunRef != candidate.SupersedesRunRef ||
		written.RescueReason != candidate.RescueReason ||
		len(written.WorksetClaims) != 1 ||
		written.WorksetClaims[0].ClaimRef != claim.ClaimRef {
		t.Fatalf("command no preserva metadata causal: %+v", written)
	}
	if queue.candidates[0].Status != orquestarunqueue.RunStatusDeliveredV0 ||
		len(queue.candidates[0].WorksetClaims) != 1 ||
		queue.candidates[0].WorksetClaims[0].ClaimRef != claim.ClaimRef {
		t.Fatalf("candidate actualizado sin metadata: %+v", queue.candidates[0])
	}
}

func TestCoordinateRunsTickRotaColaConEvidenceRefsDelDrainerV0(t *testing.T) {
	now := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	candidate := candidateWithUpdatedAtV0("run-evidence-refs", "app", 9, now.Add(-time.Minute))
	candidate.EvidenceRefs = []string{"evidence-ref-candidate", "evidence-ref-promotion"}
	queue := &fakeQueueStoreV0{candidates: []orquestarunqueue.RunSchedulingCandidateV0{candidate}}
	drainer := &fakeDrainerV0{results: map[string]RunDrainResultV0{
		"run-evidence-refs": {
			RunRef:       "run-evidence-refs",
			AppRef:       "app",
			Outcome:      "quiescent",
			QueueStatus:  orquestarunqueue.RunStatusClosedV0,
			EvidenceRefs: []string{"evidence-ref-promotion", "evidence-ref-archive"},
		},
	}}
	command := tickCommandV0(1)
	command.QueueRef = "queue-evidence-refs"
	command.OccurredAt = now

	result, err := CoordinateRunsTickV0(context.Background(), RunCoordinatorDepsV0{
		QueueReader:   queue,
		QueueUpdater:  queue,
		ControlReader: &fakeControlReaderV0{states: map[string]orquestaruncontrol.RunControlStateV0{}, missing: map[string]bool{}},
		Drainer:       drainer,
	}, command)
	if err != nil {
		t.Fatalf("CoordinateRunsTickV0: %v", err)
	}
	if len(result.Executions) != 1 ||
		!containsStringV0(result.Executions[0].EvidenceRefs, "evidence-ref-archive") {
		t.Fatalf("execution evidence_refs=%+v", result.Executions)
	}
	updated := queue.candidates[0]
	for _, want := range []string{
		"evidence-ref-candidate",
		"evidence-ref-promotion",
		"evidence-ref-archive",
		"evidence-ref-run-coordinator-executed",
	} {
		if !containsStringV0(updated.EvidenceRefs, want) {
			t.Fatalf("updated evidence_refs=%v missing %s", updated.EvidenceRefs, want)
		}
	}
	if got := countStringV0(updated.EvidenceRefs, "evidence-ref-promotion"); got != 1 {
		t.Fatalf("promotion ref duplicated %d times: %v", got, updated.EvidenceRefs)
	}
}

func TestCoordinateRunsTickSkipsPausedV0(t *testing.T) {
	deps := coordinatorDepsV0([]orquestarunqueue.RunSchedulingCandidateV0{
		candidateV0("run-paused", "app", 9),
		candidateV0("run-next", "app", 5),
	})
	deps.ControlReader.(*fakeControlReaderV0).states["run-paused"] = orquestaruncontrol.RunControlStateV0{
		RunRef: "run-paused",
		Status: orquestaruncontrol.RunControlStatusPausedV0,
	}

	result, err := CoordinateRunsTickV0(context.Background(), deps, tickCommandV0(1))
	if err != nil {
		t.Fatalf("coordinate tick: %v", err)
	}

	assertRunRefsV0(t, executionRefsV0(result.Executions), []string{"run-next"})
	assertRunRefsV0(t, skipRefsV0(result.Skips), []string{"run-paused"})
	if result.Skips[0].Status != string(orquestaruncontrol.RunControlStatusPausedV0) {
		t.Fatalf("skip status = %q", result.Skips[0].Status)
	}
}

func TestCoordinateRunsTickDrainsForcedStopRequestedV0(t *testing.T) {
	deps := coordinatorDepsV0([]orquestarunqueue.RunSchedulingCandidateV0{
		candidateV0("run-stop", "app", 9),
	})
	deps.ControlReader.(*fakeControlReaderV0).states["run-stop"] = orquestaruncontrol.RunControlStateV0{
		RunRef: "run-stop",
		Status: orquestaruncontrol.RunControlStatusStopRequestedV0,
		Forced: true,
	}

	result, err := CoordinateRunsTickV0(context.Background(), deps, tickCommandV0(1))
	if err != nil {
		t.Fatalf("coordinate tick: %v", err)
	}

	assertRunRefsV0(t, executionRefsV0(result.Executions), []string{"run-stop"})
	if len(result.Skips) != 0 {
		t.Fatalf("skips=%+v", result.Skips)
	}
	drainer := deps.Drainer.(*fakeDrainerV0)
	if len(drainer.requests) != 1 || drainer.requests[0].RunRef != "run-stop" {
		t.Fatalf("drain requests=%+v", drainer.requests)
	}
}

func TestCoordinateRunsTickSkipsStopRequestedConCheckpointPendienteV0(t *testing.T) {
	deps := coordinatorDepsV0([]orquestarunqueue.RunSchedulingCandidateV0{
		candidateV0("run-stop", "app", 9),
	})
	deps.ControlReader.(*fakeControlReaderV0).states["run-stop"] = orquestaruncontrol.RunControlStateV0{
		RunRef: "run-stop",
		Status: orquestaruncontrol.RunControlStatusStopRequestedV0,
		Forced: false,
	}

	result, err := CoordinateRunsTickV0(context.Background(), deps, tickCommandV0(1))
	if err != nil {
		t.Fatalf("coordinate tick: %v", err)
	}

	assertRunRefsV0(t, skipRefsV0(result.Skips), []string{"run-stop"})
	if len(result.Executions) != 0 {
		t.Fatalf("executions=%+v", result.Executions)
	}
}

func TestCoordinateRunsTickSincronizaControlBloqueadoConColaNoEjecutableV0(t *testing.T) {
	now := time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name        string
		control     orquestaruncontrol.RunControlStatusV0
		wantQueue   string
		wantSkipped string
	}{
		{
			name:        "stop_requested",
			control:     orquestaruncontrol.RunControlStatusStopRequestedV0,
			wantQueue:   orquestarunqueue.RunStatusStoppedV0,
			wantSkipped: string(orquestaruncontrol.RunControlStatusStopRequestedV0),
		},
		{
			name:        "cancel_requested",
			control:     orquestaruncontrol.RunControlStatusCancelRequestedV0,
			wantQueue:   orquestarunqueue.RunStatusCanceledV0,
			wantSkipped: string(orquestaruncontrol.RunControlStatusCancelRequestedV0),
		},
		{
			name:        "paused",
			control:     orquestaruncontrol.RunControlStatusPausedV0,
			wantQueue:   orquestarunqueue.RunStatusPausedV0,
			wantSkipped: string(orquestaruncontrol.RunControlStatusPausedV0),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queue := &fakeQueueStoreV0{candidates: []orquestarunqueue.RunSchedulingCandidateV0{
				candidateWithUpdatedAtV0("run-blocked", "app", 9, now.Add(-time.Hour)),
			}}
			control := &fakeControlReaderV0{
				states: map[string]orquestaruncontrol.RunControlStateV0{
					"run-blocked": {
						RunRef: "run-blocked",
						Status: tt.control,
					},
				},
				missing: map[string]bool{},
			}
			deps := RunCoordinatorDepsV0{
				QueueReader:   queue,
				QueueUpdater:  queue,
				ControlReader: control,
				Drainer:       &fakeDrainerV0{},
			}
			command := tickCommandV0(1)
			command.OccurredAt = now
			command.RankingPolicy = orquestarunqueue.DefaultRunQueueRankingPolicyV0(now)

			first, err := CoordinateRunsTickV0(context.Background(), deps, command)
			if err != nil {
				t.Fatalf("first tick: %v", err)
			}
			second, err := CoordinateRunsTickV0(context.Background(), deps, command)
			if err != nil {
				t.Fatalf("second tick: %v", err)
			}

			if len(first.Executions) != 0 ||
				len(first.Skips) != 1 ||
				first.Skips[0].Status != tt.wantSkipped {
				t.Fatalf("first=%+v", first)
			}
			if queue.candidates[0].Status != tt.wantQueue ||
				!containsStringV0(queue.candidates[0].EvidenceRefs, "evidence-ref-run-coordinator-control-blocked-queue-sync") {
				t.Fatalf("queue=%+v", queue.candidates[0])
			}
			if len(second.Ranked) != 0 || len(second.Executions) != 0 {
				t.Fatalf("second=%+v", second)
			}
		})
	}
}

func TestCoordinateRunsTickDefaultsRunningWhenControlStateMissingV0(t *testing.T) {
	deps := coordinatorDepsV0([]orquestarunqueue.RunSchedulingCandidateV0{
		candidateV0("run-missing", "app", 9),
	})
	deps.ControlReader.(*fakeControlReaderV0).missing["run-missing"] = true

	result, err := CoordinateRunsTickV0(context.Background(), deps, tickCommandV0(1))
	if err != nil {
		t.Fatalf("coordinate tick: %v", err)
	}

	assertRunRefsV0(t, executionRefsV0(result.Executions), []string{"run-missing"})
}

func TestCoordinateRunsTickPropagaDiagnosticosDelDrainV0(t *testing.T) {
	deps := coordinatorDepsV0([]orquestarunqueue.RunSchedulingCandidateV0{
		candidateV0("run-diagnostics", "app", 9),
	})
	deps.Drainer.(*fakeDrainerV0).results = map[string]RunDrainResultV0{
		"run-diagnostics": {
			RunRef:  "run-diagnostics",
			AppRef:  "app",
			Outcome: "wait_unhandled_outbox",
			Diagnostics: []RunDrainDiagnosticV0{{
				Kind:               "drain_final",
				Status:             "wait_unhandled_outbox",
				PendingOutboxCount: 2,
				PendingOutboxRefs:  []string{"outbox-ref-1", "outbox-ref-2"},
			}},
		},
	}

	result, err := CoordinateRunsTickV0(context.Background(), deps, tickCommandV0(1))
	if err != nil {
		t.Fatalf("coordinate tick: %v", err)
	}
	if len(result.Executions) != 1 ||
		len(result.Executions[0].Diagnostics) != 1 ||
		result.Executions[0].Diagnostics[0].PendingOutboxCount != 2 {
		t.Fatalf("executions=%+v", result.Executions)
	}
}

func TestCoordinateRunsTickConservaDiagnosticosSiDrainFallaV0(t *testing.T) {
	deps := coordinatorDepsV0([]orquestarunqueue.RunSchedulingCandidateV0{
		candidateV0("run-error", "app", 9),
	})
	deps.Drainer.(*fakeDrainerV0).results = map[string]RunDrainResultV0{
		"run-error": {
			RunRef:  "run-error",
			AppRef:  "app",
			Outcome: "error",
			Diagnostics: []RunDrainDiagnosticV0{{
				Kind:        "outbox_batch_dispatch_error",
				Status:      "execution_failed",
				RunRef:      "run-error",
				TargetPort:  "agent_launcher",
				MessageType: "LaunchRuntimeAgent",
				MessageID:   "outbox-launch-error-001",
				Error:       "transicion_invalida: payload.agent_request_id",
			}},
		},
	}
	deps.Drainer.(*fakeDrainerV0).errors = map[string]error{
		"run-error": errors.New("transicion_invalida: payload.agent_request_id"),
	}

	result, err := CoordinateRunsTickV0(context.Background(), deps, tickCommandV0(1))
	if err == nil {
		t.Fatalf("expected drain error")
	}
	if len(result.Executions) != 1 ||
		len(result.Executions[0].Diagnostics) != 1 ||
		result.Executions[0].Diagnostics[0].MessageID != "outbox-launch-error-001" ||
		result.Executions[0].Diagnostics[0].Error == "" {
		t.Fatalf("result sin diagnostico parcial=%+v", result)
	}
}

func TestCoordinateRunsTickContinuaTrasDrainErrorSiPoliticaLoPermiteV0(t *testing.T) {
	now := time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC)
	queue := &fakeQueueStoreV0{candidates: []orquestarunqueue.RunSchedulingCandidateV0{
		candidateWithUpdatedAtV0("run-error", "app", 10, now.Add(-time.Hour)),
		candidateWithUpdatedAtV0("run-ok", "app", 9, now.Add(-time.Hour)),
	}}
	deps := RunCoordinatorDepsV0{
		QueueReader:   queue,
		QueueUpdater:  queue,
		ControlReader: &fakeControlReaderV0{states: map[string]orquestaruncontrol.RunControlStateV0{}, missing: map[string]bool{}},
		Drainer: &fakeDrainerV0{
			errors: map[string]error{
				"run-error": errors.New("transicion_invalida: payload.phase_id"),
			},
		},
	}
	command := tickCommandV0(2)
	command.OccurredAt = now
	command.RankingPolicy = orquestarunqueue.DefaultRunQueueRankingPolicyV0(now)
	command.ContinueOnDrainError = true

	result, err := CoordinateRunsTickV0(context.Background(), deps, command)
	if err != nil {
		t.Fatalf("coordinate tick: %v", err)
	}

	assertRunRefsV0(t, executionRefsV0(result.Executions), []string{"run-error", "run-ok"})
	if len(result.Executions[0].Diagnostics) != 1 ||
		result.Executions[0].Diagnostics[0].Kind != "drain_error" ||
		result.Executions[0].Diagnostics[0].Error == "" {
		t.Fatalf("diagnostico error=%+v", result.Executions[0])
	}
	if result.Executions[1].Outcome != "drained" {
		t.Fatalf("segunda ejecucion=%+v", result.Executions[1])
	}
	if queue.candidates[0].UpdatedAt != now ||
		!containsStringV0(queue.candidates[0].EvidenceRefs, "evidence-ref-run-coordinator-executed") {
		t.Fatalf("run error no rotado en cola=%+v", queue.candidates[0])
	}
}

func TestCoordinateRunsTickAllowsMissingControlReaderV0(t *testing.T) {
	deps := coordinatorDepsV0([]orquestarunqueue.RunSchedulingCandidateV0{
		candidateV0("run-no-control", "app", 9),
	})
	deps.ControlReader = nil

	result, err := CoordinateRunsTickV0(context.Background(), deps, tickCommandV0(1))
	if err != nil {
		t.Fatalf("coordinate tick: %v", err)
	}

	assertRunRefsV0(t, executionRefsV0(result.Executions), []string{"run-no-control"})
}

func TestCoordinateRunsTickRequiresReaderAndDrainerV0(t *testing.T) {
	_, err := CoordinateRunsTickV0(context.Background(), RunCoordinatorDepsV0{}, tickCommandV0(1))
	if err == nil {
		t.Fatalf("expected queue reader error")
	}
	deps := coordinatorDepsV0([]orquestarunqueue.RunSchedulingCandidateV0{
		candidateV0("run-a", "app", 9),
	})
	deps.Drainer = nil
	_, err = CoordinateRunsTickV0(context.Background(), deps, tickCommandV0(1))
	if err == nil {
		t.Fatalf("expected drainer error")
	}
}

func TestCoordinateRunsTickExecutesUntilMaxRunsV0(t *testing.T) {
	deps := coordinatorDepsV0([]orquestarunqueue.RunSchedulingCandidateV0{
		candidateV0("run-a", "app", 9),
		candidateV0("run-b", "app", 8),
		candidateV0("run-c", "app", 7),
	})

	result, err := CoordinateRunsTickV0(context.Background(), deps, tickCommandV0(2))
	if err != nil {
		t.Fatalf("coordinate tick: %v", err)
	}

	assertRunRefsV0(t, executionRefsV0(result.Executions), []string{"run-a", "run-b"})
}

func TestCoordinateRunsTickCortaCooperativamenteTrasCancelarContextoV0(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	deps := coordinatorDepsV0([]orquestarunqueue.RunSchedulingCandidateV0{
		candidateV0("run-a", "app", 9),
		candidateV0("run-b", "app", 8),
	})
	deps.QueueUpdater = &fakeQueueStoreV0{candidates: []orquestarunqueue.RunSchedulingCandidateV0{
		candidateV0("run-a", "app", 9),
		candidateV0("run-b", "app", 8),
	}}
	drainer := deps.Drainer.(*fakeDrainerV0)
	drainer.afterDrain = func(_ context.Context, request RunDrainRequestV0) {
		if request.RunRef == "run-a" {
			cancel()
		}
	}

	result, err := CoordinateRunsTickV0(ctx, deps, tickCommandV0(2))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v want context.Canceled result=%+v", err, result)
	}
	assertRunRefsV0(t, executionRefsV0(result.Executions), []string{"run-a"})
	if len(drainer.requests) != 1 || drainer.requests[0].RunRef != "run-a" {
		t.Fatalf("drain requests=%+v", drainer.requests)
	}
}

func TestCoordinateRunsTickRotaRunsEjecutadosConMismaPrioridadV0(t *testing.T) {
	now := time.Date(2026, 5, 11, 10, 0, 0, 0, time.UTC)
	queue := &fakeQueueStoreV0{candidates: []orquestarunqueue.RunSchedulingCandidateV0{
		candidateWithUpdatedAtV0("run-a", "app", 9, now.Add(-time.Hour)),
		candidateWithUpdatedAtV0("run-b", "app", 9, now.Add(-time.Hour)),
	}}
	deps := RunCoordinatorDepsV0{
		QueueReader:   queue,
		QueueUpdater:  queue,
		ControlReader: &fakeControlReaderV0{states: map[string]orquestaruncontrol.RunControlStateV0{}, missing: map[string]bool{}},
		Drainer:       &fakeDrainerV0{},
	}
	command := tickCommandV0(1)
	command.OccurredAt = now
	command.RankingPolicy = orquestarunqueue.DefaultRunQueueRankingPolicyV0(now)

	first, err := CoordinateRunsTickV0(context.Background(), deps, command)
	if err != nil {
		t.Fatalf("first tick: %v", err)
	}
	second, err := CoordinateRunsTickV0(context.Background(), deps, command)
	if err != nil {
		t.Fatalf("second tick: %v", err)
	}

	assertRunRefsV0(t, executionRefsV0(first.Executions), []string{"run-a"})
	assertRunRefsV0(t, executionRefsV0(second.Executions), []string{"run-b"})
}

func TestCoordinateRunsTickPropagaEstadoTerminalALaColaV0(t *testing.T) {
	now := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	queue := &fakeQueueStoreV0{candidates: []orquestarunqueue.RunSchedulingCandidateV0{
		candidateWithUpdatedAtV0("run-closed", "app", 9, now.Add(-time.Hour)),
	}}
	drainer := &fakeDrainerV0{
		results: map[string]RunDrainResultV0{
			"run-closed": {
				RunRef:      "run-closed",
				AppRef:      "app",
				Outcome:     "run_terminal",
				QueueStatus: orquestarunqueue.RunStatusClosedV0,
			},
		},
	}
	deps := RunCoordinatorDepsV0{
		QueueReader:   queue,
		QueueUpdater:  queue,
		ControlReader: &fakeControlReaderV0{states: map[string]orquestaruncontrol.RunControlStateV0{}, missing: map[string]bool{}},
		Drainer:       drainer,
	}
	command := tickCommandV0(1)
	command.OccurredAt = now
	command.RankingPolicy = orquestarunqueue.DefaultRunQueueRankingPolicyV0(now)

	first, err := CoordinateRunsTickV0(context.Background(), deps, command)
	if err != nil {
		t.Fatalf("first tick: %v", err)
	}
	second, err := CoordinateRunsTickV0(context.Background(), deps, command)
	if err != nil {
		t.Fatalf("second tick: %v", err)
	}

	if len(first.Executions) != 1 ||
		first.Executions[0].QueueStatus != orquestarunqueue.RunStatusClosedV0 ||
		queue.candidates[0].Status != orquestarunqueue.RunStatusClosedV0 {
		t.Fatalf("first=%+v queue=%+v", first, queue.candidates)
	}
	if len(second.Executions) != 0 {
		t.Fatalf("second=%+v", second)
	}
}

func TestCoordinateRunsTickMarcaRunningSiDrainNoDeclaraEstadoColaV0(t *testing.T) {
	now := time.Date(2026, 5, 27, 10, 0, 0, 0, time.UTC)
	queue := &fakeQueueStoreV0{candidates: []orquestarunqueue.RunSchedulingCandidateV0{
		candidateWithUpdatedAtV0("run-active", "app", 9, now.Add(-time.Hour)),
	}}
	drainer := &fakeDrainerV0{
		results: map[string]RunDrainResultV0{
			"run-active": {
				RunRef:  "run-active",
				AppRef:  "app",
				Outcome: "external_work_pending",
			},
		},
	}
	deps := RunCoordinatorDepsV0{
		QueueReader:   queue,
		QueueUpdater:  queue,
		ControlReader: &fakeControlReaderV0{states: map[string]orquestaruncontrol.RunControlStateV0{}, missing: map[string]bool{}},
		Drainer:       drainer,
	}
	command := tickCommandV0(1)
	command.OccurredAt = now
	command.RankingPolicy = orquestarunqueue.DefaultRunQueueRankingPolicyV0(now)

	result, err := CoordinateRunsTickV0(context.Background(), deps, command)
	if err != nil {
		t.Fatalf("tick: %v", err)
	}
	if len(result.Executions) != 1 ||
		result.Executions[0].QueueStatus != orquestarunqueue.RunStatusRunningV0 ||
		queue.candidates[0].Status != orquestarunqueue.RunStatusRunningV0 {
		t.Fatalf("result=%+v queue=%+v", result, queue.candidates)
	}
}

func TestCoordinateRunsTickSkipsExcludedRunsV0(t *testing.T) {
	deps := coordinatorDepsV0([]orquestarunqueue.RunSchedulingCandidateV0{
		candidateV0("run-a", "app", 9),
		candidateV0("run-b", "app", 8),
	})
	command := tickCommandV0(1)
	command.ExcludeRunRefs = []string{" run-a "}

	result, err := CoordinateRunsTickV0(context.Background(), deps, command)
	if err != nil {
		t.Fatalf("coordinate tick: %v", err)
	}

	assertRunRefsV0(t, executionRefsV0(result.Executions), []string{"run-b"})
	if len(result.Skips) != 1 ||
		result.Skips[0].RunRef != "run-a" ||
		result.Skips[0].Reason != "run_excluded" {
		t.Fatalf("skips=%+v", result.Skips)
	}
}

func TestCoordinateRunsTickAppliesQueueLimitAfterRankingAndPropagatesTraceFieldsV0(t *testing.T) {
	deps := coordinatorDepsV0([]orquestarunqueue.RunSchedulingCandidateV0{
		candidateV0("run-a", "app", 9),
		candidateV0("run-b", "app", 8),
	})
	queue := deps.QueueReader.(*fakeQueueReaderV0)
	drainer := deps.Drainer.(*fakeDrainerV0)
	occurredAt := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)

	result, err := CoordinateRunsTickV0(context.Background(), deps, RunCoordinatorTickCommandV0{
		QueueRef:      "queue-main",
		AppRefs:       []string{"app"},
		QueueLimit:    1,
		MaxRuns:       1,
		OccurredAt:    occurredAt,
		CorrelationID: "corr-123",
		DrainLimits: RunDrainLimitsV0{
			MaxExternalWaits: 1,
		},
		RankingPolicy: orquestarunqueue.DefaultRunQueueRankingPolicyV0(occurredAt),
	})
	if err != nil {
		t.Fatalf("coordinate tick: %v", err)
	}

	if queue.request.Limit != 0 {
		t.Fatalf("queue limit = %d", queue.request.Limit)
	}
	if len(result.Ranked) != 1 || result.Ranked[0].RunRef != "run-a" {
		t.Fatalf("ranked=%+v", result.Ranked)
	}
	if len(drainer.requests) != 1 {
		t.Fatalf("drain requests = %d", len(drainer.requests))
	}
	if !drainer.requests[0].OccurredAt.Equal(occurredAt) {
		t.Fatalf("occurred_at = %s", drainer.requests[0].OccurredAt)
	}
	if drainer.requests[0].CorrelationID != "corr-123" {
		t.Fatalf("correlation_id = %q", drainer.requests[0].CorrelationID)
	}
	if drainer.requests[0].Limits.MaxExternalWaits != 1 {
		t.Fatalf("limits = %+v", drainer.requests[0].Limits)
	}
}

func TestCoordinateRunsTickDoesNotMutateCandidatesV0(t *testing.T) {
	candidates := []orquestarunqueue.RunSchedulingCandidateV0{
		candidateV0("run-a", "app", 1),
		candidateV0("run-b", "app", 9),
	}
	candidates[0].EvidenceRefs = []string{"ev-a"}
	want := cloneCandidatesV0(candidates)
	deps := coordinatorDepsV0(candidates)

	if _, err := CoordinateRunsTickV0(context.Background(), deps, tickCommandV0(2)); err != nil {
		t.Fatalf("coordinate tick: %v", err)
	}

	if !reflect.DeepEqual(candidates, want) {
		t.Fatalf("candidates mutated\nwant: %#v\ngot:  %#v", want, candidates)
	}
}

type fakeQueueReaderV0 struct {
	candidates []orquestarunqueue.RunSchedulingCandidateV0
	request    orquestarunqueue.RunQueueReadRequestV0
	applyLimit bool
}

func (fake *fakeQueueReaderV0) ListRunSchedulingCandidatesV0(
	_ context.Context,
	request orquestarunqueue.RunQueueReadRequestV0,
) ([]orquestarunqueue.RunSchedulingCandidateV0, error) {
	fake.request = request
	if fake.applyLimit && request.Limit > 0 && len(fake.candidates) > request.Limit {
		return fake.candidates[:request.Limit], nil
	}
	return fake.candidates, nil
}

type fakeQueueStoreV0 struct {
	candidates []orquestarunqueue.RunSchedulingCandidateV0
	commands   []orquestarunqueue.RunQueuePriorityCommandV0
}

func (fake *fakeQueueStoreV0) ListRunSchedulingCandidatesV0(
	_ context.Context,
	_ orquestarunqueue.RunQueueReadRequestV0,
) ([]orquestarunqueue.RunSchedulingCandidateV0, error) {
	return cloneCandidatesV0(fake.candidates), nil
}

func (fake *fakeQueueStoreV0) SetRunPriorityV0(
	_ context.Context,
	command orquestarunqueue.RunQueuePriorityCommandV0,
) (orquestarunqueue.RunSchedulingCandidateV0, error) {
	fake.commands = append(fake.commands, command)
	for index := range fake.candidates {
		if fake.candidates[index].RunRef != command.RunRef {
			continue
		}
		if command.Status != "" {
			fake.candidates[index].Status = command.Status
		}
		fake.candidates[index].UpdatedAt = command.UpdatedAt
		fake.candidates[index].PriorityScore = command.PriorityScore
		if command.FairnessGroupRef != "" {
			fake.candidates[index].FairnessGroupRef = command.FairnessGroupRef
		}
		if !orquestarunqueue.RunQueueAttemptGroupEmptyV0(command.AttemptGroup) {
			fake.candidates[index].AttemptGroup = command.AttemptGroup
		}
		if command.ParentRunRef != "" {
			fake.candidates[index].ParentRunRef = command.ParentRunRef
		}
		if command.SupersedesRunRef != "" {
			fake.candidates[index].SupersedesRunRef = command.SupersedesRunRef
		}
		if command.RescueReason != "" {
			fake.candidates[index].RescueReason = command.RescueReason
		}
		fake.candidates[index].EvidenceRefs = append([]string(nil), command.EvidenceRefs...)
		fake.candidates[index].WorksetClaims = append([]orquestarunqueue.WorksetClaimV0(nil), command.WorksetClaims...)
		return fake.candidates[index], nil
	}
	return orquestarunqueue.RunSchedulingCandidateV0{}, errors.New("run not found")
}

type fakeControlReaderV0 struct {
	states  map[string]orquestaruncontrol.RunControlStateV0
	missing map[string]bool
}

func (fake *fakeControlReaderV0) ReadRunControlStateV0(
	_ context.Context,
	request orquestaruncontrol.RunControlReadRequestV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	if fake.missing[request.RunRef] {
		return orquestaruncontrol.RunControlStateV0{},
			orquestaruncontrol.RunControlStateNotFoundErrorV0(request)
	}
	if state, ok := fake.states[request.RunRef]; ok {
		return state, nil
	}
	return orquestaruncontrol.DefaultRunControlStateV0(request.RunRef), nil
}

type fakeDrainerV0 struct {
	requests   []RunDrainRequestV0
	results    map[string]RunDrainResultV0
	errors     map[string]error
	afterDrain func(context.Context, RunDrainRequestV0)
}

func (fake *fakeDrainerV0) DrainRunV0(
	ctx context.Context,
	request RunDrainRequestV0,
) (RunDrainResultV0, error) {
	if request.RunRef == "" {
		return RunDrainResultV0{}, errors.New("missing run ref")
	}
	fake.requests = append(fake.requests, request)
	if fake.afterDrain != nil {
		fake.afterDrain(ctx, request)
	}
	if result, ok := fake.results[request.RunRef]; ok {
		return result, fake.errors[request.RunRef]
	}
	if err, ok := fake.errors[request.RunRef]; ok {
		return RunDrainResultV0{RunRef: request.RunRef, AppRef: request.AppRef, Outcome: "error"}, err
	}
	return RunDrainResultV0{RunRef: request.RunRef, AppRef: request.AppRef, Outcome: "drained"}, nil
}

func coordinatorDepsV0(candidates []orquestarunqueue.RunSchedulingCandidateV0) RunCoordinatorDepsV0 {
	return RunCoordinatorDepsV0{
		QueueReader: &fakeQueueReaderV0{candidates: candidates},
		ControlReader: &fakeControlReaderV0{
			states:  map[string]orquestaruncontrol.RunControlStateV0{},
			missing: map[string]bool{},
		},
		Drainer: &fakeDrainerV0{},
	}
}

func tickCommandV0(maxRuns int) RunCoordinatorTickCommandV0 {
	now := time.Date(2026, 5, 11, 10, 0, 0, 0, time.UTC)
	return RunCoordinatorTickCommandV0{
		MaxRuns:       maxRuns,
		OccurredAt:    now,
		CorrelationID: "corr-test",
		RankingPolicy: orquestarunqueue.DefaultRunQueueRankingPolicyV0(now),
	}
}

func candidateV0(runRef string, appRef string, priority int) orquestarunqueue.RunSchedulingCandidateV0 {
	return candidateWithUpdatedAtV0(
		runRef,
		appRef,
		priority,
		time.Date(2026, 5, 11, 9, 0, 0, 0, time.UTC),
	)
}

func candidateWithUpdatedAtV0(
	runRef string,
	appRef string,
	priority int,
	updatedAt time.Time,
) orquestarunqueue.RunSchedulingCandidateV0 {
	return orquestarunqueue.RunSchedulingCandidateV0{
		RunRef:        runRef,
		AppRef:        appRef,
		Status:        "queued",
		PriorityScore: priority,
		UpdatedAt:     updatedAt,
	}
}

func cloneCandidatesV0(
	candidates []orquestarunqueue.RunSchedulingCandidateV0,
) []orquestarunqueue.RunSchedulingCandidateV0 {
	cloned := append([]orquestarunqueue.RunSchedulingCandidateV0(nil), candidates...)
	for index := range cloned {
		cloned[index].AttemptGroup.WriteSetRefs = append([]string(nil), cloned[index].AttemptGroup.WriteSetRefs...)
		cloned[index].EvidenceRefs = append([]string(nil), cloned[index].EvidenceRefs...)
		cloned[index].WorksetClaims = append([]orquestarunqueue.WorksetClaimV0(nil), cloned[index].WorksetClaims...)
		for claimIndex := range cloned[index].WorksetClaims {
			cloned[index].WorksetClaims[claimIndex].ReadSet = append([]orquestarunqueue.ScopeRefV0(nil), cloned[index].WorksetClaims[claimIndex].ReadSet...)
			cloned[index].WorksetClaims[claimIndex].WriteSet = append([]orquestarunqueue.ScopeRefV0(nil), cloned[index].WorksetClaims[claimIndex].WriteSet...)
			cloned[index].WorksetClaims[claimIndex].DependsOn = append([]string(nil), cloned[index].WorksetClaims[claimIndex].DependsOn...)
			cloned[index].WorksetClaims[claimIndex].EvidenceRefs = append([]string(nil), cloned[index].WorksetClaims[claimIndex].EvidenceRefs...)
		}
	}
	return cloned
}

func containsStringV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func countStringV0(values []string, want string) int {
	count := 0
	for _, value := range values {
		if value == want {
			count++
		}
	}
	return count
}
