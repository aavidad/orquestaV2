package orquestaruncoordinator

import (
	"testing"

	"pgregory.net/rapid"
)

func TestReconcileExternalWorkPublicStatusV0PropNuncaCompletedSinEvidenciaTerminalV0(t *testing.T) {
	// Invariant: completed publico exige artefacto local o job de dominio completed.
	rapid.Check(t, func(rt *rapid.T) {
		input := externalWorkInputRapidV0().Draw(rt, "input")
		got := ReconcileExternalWorkPublicStatusV0(input)
		if got.PublicStatus != ExternalWorkPublicStatusCompletedV0 {
			return
		}
		if !input.LocalArtifactPresent && normalizeExternalWorkStatusV0(input.DomainJobStatus) != "completed" {
			rt.Fatalf("completed sin evidencia terminal: input=%+v decision=%+v", input, got)
		}
	})
}

func TestReconcileExternalWorkPublicStatusV0PropAckSinCierreNoCompletaV0(t *testing.T) {
	// Invariant: ACK completed sin artefacto ni cierre de dominio queda bloqueado para reconciliar.
	rapid.Check(t, func(rt *rapid.T) {
		input := externalWorkInputRapidV0().Draw(rt, "input")
		input.AgentAckCompleted = true
		input.AgentCheckpointPresent = true
		input.LocalArtifactPresent = false
		input.DomainJobStatus = rapid.SampledFrom([]string{"", "pending", "running", "accepted"}).Draw(rt, "domain")
		got := ReconcileExternalWorkPublicStatusV0(input)
		if got.PublicStatus != ExternalWorkPublicStatusBlockedV0 ||
			got.Action != ExternalWorkReconcileActionIngestAckV0 ||
			got.PublicStatus == ExternalWorkPublicStatusCompletedV0 {
			rt.Fatalf("ACK sin cierre mal reconciliado: input=%+v decision=%+v", input, got)
		}
	})
}

func TestReconcileExternalWorkPublicStatusV0PropOutboxPendienteMantienePendingV0(t *testing.T) {
	// Invariant: outbox pendiente prueba trabajo solicitado, no un proceso vivo.
	rapid.Check(t, func(rt *rapid.T) {
		input := ExternalWorkReconciliationInputV0{
			RunRef:                "run-outbox-property",
			ProjectionStatus:      rapid.SampledFrom([]string{"done", "completed", "quiescent"}).Draw(rt, "projection"),
			ProjectionTaskCount:   0,
			WorkflowTaskOpenCount: 0,
			PendingOutboxCount:    rapid.IntRange(1, 5).Draw(rt, "outbox"),
			DomainJobStatus:       rapid.SampledFrom([]string{"pending", "queued", "running"}).Draw(rt, "domain"),
			EvidenceRefs:          []string{"projection-ref", "outbox-ref", "domain-ref"},
		}
		got := ReconcileExternalWorkPublicStatusV0(input)
		if got.PublicStatus != ExternalWorkPublicStatusPendingV0 ||
			got.Reason != "causal_work_pending_without_runtime_liveness" ||
			!containsRunCoordinatorStringForTestV0(got.CausalSourceRefs, "outbox_ledger") {
			rt.Fatalf("outbox pendiente no mantuvo pending: input=%+v decision=%+v", input, got)
		}
	})
}

func TestReconcileExternalWorkPublicStatusV0PropDoneDominioPendienteSinTrabajoBloqueaV0(t *testing.T) {
	// Invariant: proyeccion cerrada con dominio pendiente y sin trabajo causal abre bloqueo/rework.
	rapid.Check(t, func(rt *rapid.T) {
		input := ExternalWorkReconciliationInputV0{
			RunRef:                "run-domain-pending-property",
			ProjectionStatus:      rapid.SampledFrom([]string{"done", "completed", "quiescent"}).Draw(rt, "projection"),
			ProjectionTaskCount:   0,
			WorkflowTaskOpenCount: 0,
			PendingOutboxCount:    0,
			DomainJobStatus:       rapid.SampledFrom([]string{"pending", "queued", "running"}).Draw(rt, "domain"),
			EvidenceRefs:          []string{"projection-ref", "domain-ref"},
		}
		got := ReconcileExternalWorkPublicStatusV0(input)
		if got.PublicStatus != ExternalWorkPublicStatusBlockedV0 ||
			got.Action != ExternalWorkReconcileActionReopenOrBlockDomainV0 ||
			!got.Reconciled {
			rt.Fatalf("dominio pendiente sin trabajo no bloqueo: input=%+v decision=%+v", input, got)
		}
	})
}

func TestReconcileExternalWorkPublicStatusV0PropStaleSinProcesoNoRunningV0(t *testing.T) {
	// Invariant: liveness stale con proceso descartado no se publica como running por si solo.
	rapid.Check(t, func(rt *rapid.T) {
		input := ExternalWorkReconciliationInputV0{
			RunRef:                 "run-stale-property",
			ProjectionStatus:       "running",
			ProcessRegistryChecked: true,
			ProcessAlive:           false,
			Liveness: RunLivenessClassificationV0{
				Class:                  RunLivenessClassRunningStaleNoProcessV0,
				Running:                true,
				Stale:                  true,
				ConfirmedNoLiveProcess: true,
				Verifiable:             true,
				SafeToReconcile:        true,
			},
			DomainJobStatus: rapid.SampledFrom([]string{"", "pending"}).Draw(rt, "domain"),
			EvidenceRefs:    []string{"liveness-ref"},
		}
		got := ReconcileExternalWorkPublicStatusV0(input)
		if got.PublicStatus == ExternalWorkPublicStatusRunningV0 {
			rt.Fatalf("stale sin proceso publicado como running: input=%+v decision=%+v", input, got)
		}
	})
}

func externalWorkInputRapidV0() *rapid.Generator[ExternalWorkReconciliationInputV0] {
	return rapid.Custom(func(rt *rapid.T) ExternalWorkReconciliationInputV0 {
		return ExternalWorkReconciliationInputV0{
			RunRef:                 rapid.SampledFrom([]string{"", "run-a", "run-b"}).Draw(rt, "run"),
			ProjectionStatus:       rapid.SampledFrom([]string{"", "running", "done", "completed", "quiescent", "blocked"}).Draw(rt, "projection"),
			ProjectionTaskCount:    rapid.IntRange(0, 3).Draw(rt, "projection_tasks"),
			WorkflowTaskOpenCount:  rapid.IntRange(0, 3).Draw(rt, "open_tasks"),
			PendingOutboxCount:     rapid.IntRange(0, 3).Draw(rt, "pending_outbox"),
			DomainJobStatus:        rapid.SampledFrom([]string{"", "pending", "queued", "running", "completed", "blocked"}).Draw(rt, "domain"),
			AgentAckCompleted:      rapid.Bool().Draw(rt, "ack"),
			AgentCheckpointPresent: rapid.Bool().Draw(rt, "checkpoint"),
			LocalArtifactPresent:   rapid.Bool().Draw(rt, "artifact"),
			ProcessRegistryChecked: rapid.Bool().Draw(rt, "process_checked"),
			ProcessAlive:           rapid.Bool().Draw(rt, "process_alive"),
			Liveness:               livenessRapidV0().Draw(rt, "liveness"),
			EvidenceRefs:           rapid.SliceOfN(rapid.SampledFrom([]string{"evidence-a", "evidence-b", "evidence-c"}), 0, 3).Draw(rt, "evidence_refs"),
		}
	})
}

func livenessRapidV0() *rapid.Generator[RunLivenessClassificationV0] {
	return rapid.Custom(func(rt *rapid.T) RunLivenessClassificationV0 {
		class := rapid.SampledFrom([]string{
			"",
			RunLivenessClassRunningLiveV0,
			RunLivenessClassRunningStaleNoProcessV0,
			RunLivenessClassRunningWithoutRecentStatsV0,
			RunLivenessClassCompletedV0,
			RunLivenessClassFailedV0,
			RunLivenessClassBlockedV0,
		}).Draw(rt, "class")
		return RunLivenessClassificationV0{
			Class:                  class,
			Running:                rapid.Bool().Draw(rt, "running"),
			Live:                   class == RunLivenessClassRunningLiveV0 && rapid.Bool().Draw(rt, "live"),
			Stale:                  class == RunLivenessClassRunningStaleNoProcessV0,
			ConfirmedNoLiveProcess: class == RunLivenessClassRunningStaleNoProcessV0,
			Verifiable:             rapid.Bool().Draw(rt, "verifiable"),
			SafeToReconcile:        rapid.Bool().Draw(rt, "safe"),
		}
	})
}
