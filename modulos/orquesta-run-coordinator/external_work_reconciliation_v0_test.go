package orquestaruncoordinator

import "testing"

func TestReconcileExternalWorkPublicStatusV0NoDeclaraDoneSiJobDominioSiguePendingSinTareas(t *testing.T) {
	got := ReconcileExternalWorkPublicStatusV0(ExternalWorkReconciliationInputV0{
		RunRef:              "run-ref-opes-pending-001",
		ProjectionStatus:    "done",
		ProjectionTaskCount: 0,
		DomainJobStatus:     "pending",
		EvidenceRefs:        []string{"projection-ref-001", "domain-job-ref-001"},
	})

	if got.PublicStatus != ExternalWorkPublicStatusBlockedV0 ||
		got.Action != ExternalWorkReconcileActionReopenOrBlockDomainV0 ||
		got.Reason != "projection_done_but_domain_job_pending_without_open_tasks" ||
		!got.Justified ||
		!got.Reconciled ||
		!containsRunCoordinatorStringForTestV0(got.CausalSourceRefs, "run_projection") ||
		!containsRunCoordinatorStringForTestV0(got.CausalSourceRefs, "domain_job_state") {
		t.Fatalf("decision=%+v", got)
	}
	if got.PublicStatus == ExternalWorkPublicStatusCompletedV0 ||
		got.PublicStatus == ExternalWorkPublicStatusRunningV0 {
		t.Fatalf("estado publico falso: %+v", got)
	}
}

func TestReconcileExternalWorkPublicStatusV0AckCompletedSinCierreNoDeclaraDone(t *testing.T) {
	got := ReconcileExternalWorkPublicStatusV0(ExternalWorkReconciliationInputV0{
		RunRef:                 "run-ref-ack-completed-001",
		ProjectionStatus:       "running",
		ProjectionTaskCount:    1,
		DomainJobStatus:        "pending",
		AgentAckCompleted:      true,
		AgentCheckpointPresent: true,
		ProcessRegistryChecked: true,
		ProcessAlive:           false,
		Liveness: RunLivenessClassificationV0{
			Class: RunLivenessClassRunningStaleNoProcessV0,
			Stale: true,
		},
		EvidenceRefs: []string{"ack-ref-001"},
	})

	if got.PublicStatus != ExternalWorkPublicStatusBlockedV0 ||
		got.Action != ExternalWorkReconcileActionIngestAckV0 ||
		got.Reason != "completed_ack_observed_without_domain_closure" ||
		!got.Justified ||
		got.Reconciled ||
		!containsRunCoordinatorStringForTestV0(got.CausalSourceRefs, "agent_ack_checkpoint") ||
		!containsRunCoordinatorStringForTestV0(got.CausalSourceRefs, "process_registry") {
		t.Fatalf("decision=%+v", got)
	}
	if got.PublicStatus == ExternalWorkPublicStatusRunningV0 ||
		got.PublicStatus == ExternalWorkPublicStatusCompletedV0 ||
		got.Action == ExternalWorkReconcileActionObserveV0 {
		t.Fatalf("ACK completado sin cierre se publico como terminal falso: %+v", got)
	}
}

func TestReconcileExternalWorkPublicStatusV0AckCompletedConArtefactoCierra(t *testing.T) {
	got := ReconcileExternalWorkPublicStatusV0(ExternalWorkReconciliationInputV0{
		RunRef:                 "run-ref-ack-artifact-001",
		ProjectionStatus:       "running",
		DomainJobStatus:        "pending",
		AgentAckCompleted:      true,
		AgentCheckpointPresent: true,
		LocalArtifactPresent:   true,
		EvidenceRefs:           []string{"ack-ref-001", "artifact-ref-001"},
	})

	if got.PublicStatus != ExternalWorkPublicStatusCompletedV0 ||
		got.Action != ExternalWorkReconcileActionIngestAckV0 ||
		got.Reason != "completed_ack_observed_with_closure_evidence" ||
		!got.Justified ||
		!got.Reconciled ||
		!containsRunCoordinatorStringForTestV0(got.CausalSourceRefs, "local_artifact") {
		t.Fatalf("decision=%+v", got)
	}
}

func TestReconcileExternalWorkPublicStatusV0AckConArtefactoSinRefNoCierra(t *testing.T) {
	got := ReconcileExternalWorkPublicStatusV0(ExternalWorkReconciliationInputV0{
		RunRef:               "run-ref-ack-artifact-unproven-001",
		ProjectionStatus:     "running",
		AgentAckCompleted:    true,
		LocalArtifactPresent: true,
	})

	if got.PublicStatus != ExternalWorkPublicStatusBlockedV0 ||
		got.Reason != "durable_terminal_evidence_missing" ||
		got.CausalVerdict.ResultadoTerminal ||
		!got.CausalVerdict.RequiereReparacion {
		t.Fatalf("cierre sin referencia durable no debe publicarse: %+v", got)
	}
}

func TestReconcileExternalWorkPublicStatusV0ProcesoVivoExigeIdentidadV0(t *testing.T) {
	got := ReconcileExternalWorkPublicStatusV0(ExternalWorkReconciliationInputV0{
		RunRef:                 "run-ref-process-unattributed-001",
		ProjectionStatus:       "running",
		ProcessRegistryChecked: true,
		ProcessAlive:           true,
		EvidenceRefs:           []string{"process-ref-001"},
	})

	if got.PublicStatus == ExternalWorkPublicStatusRunningV0 ||
		got.CausalVerdict.ReasonCode != "goal_runtime_identity_missing" ||
		!got.CausalVerdict.RequiereReparacion {
		t.Fatalf("proceso sin identidad causal no debe confirmar running: %+v", got)
	}
}

func TestReconcileExternalWorkPublicStatusV0RepairCausalGanaAOutboxPendienteV0(t *testing.T) {
	got := ReconcileExternalWorkPublicStatusV0(ExternalWorkReconciliationInputV0{
		RunRef:                 "run-ref-process-unattributed-outbox-001",
		ProjectionStatus:       "running",
		PendingOutboxCount:     1,
		DomainJobStatus:        "pending",
		ProcessRegistryChecked: true,
		ProcessAlive:           true,
		EvidenceRefs:           []string{"process-ref-001", "outbox-ref-001"},
	})

	if got.PublicStatus != ExternalWorkPublicStatusBlockedV0 ||
		got.Reason != "goal_runtime_identity_missing" ||
		got.NextAction != "repair_causal_evidence_before_reconcile" {
		t.Fatalf("repair causal no debe ocultarse como pending: %+v", got)
	}
}

func TestReconcileExternalWorkPublicStatusV0RunningStaleNoProcessNoDeclaraRunning(t *testing.T) {
	got := ReconcileExternalWorkPublicStatusV0(ExternalWorkReconciliationInputV0{
		RunRef:                 "run-ref-stale-no-process-001",
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
		EvidenceRefs: []string{"liveness-ref-001"},
	})

	if got.PublicStatus == ExternalWorkPublicStatusRunningV0 ||
		got.Reason == "live_process_open_task_pending_outbox_or_liveness" {
		t.Fatalf("running stale sin proceso vivo no puede publicarse como running: %+v", got)
	}
}

func containsRunCoordinatorStringForTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
