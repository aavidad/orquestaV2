package orquestaruncoordinator

import (
	"strings"

	orquestaestadovivo "orquesta/modulos/orquesta-estado-vivo"
)

const (
	ExternalWorkPublicStatusRunningV0   = "running"
	ExternalWorkPublicStatusCompletedV0 = "completed"
	ExternalWorkPublicStatusBlockedV0   = "blocked"
	ExternalWorkPublicStatusPendingV0   = "pending"
	ExternalWorkPublicStatusUnknownV0   = "unknown"

	ExternalWorkReconcileActionObserveV0             = "observe_causal_sources"
	ExternalWorkReconcileActionIngestAckV0           = "ingest_completed_ack"
	ExternalWorkReconcileActionReopenOrBlockDomainV0 = "reopen_or_block_on_domain_job"
	ExternalWorkReconcileActionWaitDomainV0          = "wait_domain_job_progress"
	ExternalWorkReconcileActionNoopV0                = "noop"
)

type ExternalWorkReconciliationInputV0 struct {
	RunRef                  string                      `json:"run_ref,omitempty"`
	ProjectionStatus        string                      `json:"projection_status,omitempty"`
	ProjectionTaskCount     int                         `json:"projection_task_count,omitempty"`
	WorkflowTaskOpenCount   int                         `json:"workflow_task_open_count,omitempty"`
	PendingOutboxCount      int                         `json:"pending_outbox_count,omitempty"`
	DomainJobStatus         string                      `json:"domain_job_status,omitempty"`
	AgentAckCompleted       bool                        `json:"agent_ack_completed,omitempty"`
	AgentCheckpointPresent  bool                        `json:"agent_checkpoint_present,omitempty"`
	LocalArtifactPresent    bool                        `json:"local_artifact_present,omitempty"`
	ProcessRegistryChecked  bool                        `json:"process_registry_checked,omitempty"`
	ProcessAlive            bool                        `json:"process_alive,omitempty"`
	RuntimeIdentityRef      string                      `json:"runtime_identity_ref,omitempty"`
	RuntimeGenerationRef    string                      `json:"runtime_generation_ref,omitempty"`
	RuntimeIdentityMismatch bool                        `json:"runtime_identity_mismatch,omitempty"`
	Liveness                RunLivenessClassificationV0 `json:"liveness,omitempty"`
	EvidenceRefs            []string                    `json:"evidence_refs,omitempty"`
}

type ExternalWorkReconciliationDecisionV0 struct {
	RunRef           string                               `json:"run_ref,omitempty"`
	PublicStatus     string                               `json:"public_status"`
	Action           string                               `json:"action,omitempty"`
	Reason           string                               `json:"reason,omitempty"`
	NextAction       string                               `json:"next_action,omitempty"`
	Justified        bool                                 `json:"justified,omitempty"`
	Reconciled       bool                                 `json:"reconciled,omitempty"`
	EvidenceRefs     []string                             `json:"evidence_refs,omitempty"`
	CausalSourceRefs []string                             `json:"causal_source_refs,omitempty"`
	CausalVerdict    orquestaestadovivo.VeredictoCausalV0 `json:"causal_verdict"`
}

func ReconcileExternalWorkPublicStatusV0(
	input ExternalWorkReconciliationInputV0,
) ExternalWorkReconciliationDecisionV0 {
	decision := ExternalWorkReconciliationDecisionV0{
		RunRef:           strings.TrimSpace(input.RunRef),
		PublicStatus:     ExternalWorkPublicStatusUnknownV0,
		Action:           ExternalWorkReconcileActionObserveV0,
		Reason:           "insufficient_causal_evidence",
		NextAction:       "observe_run_projection_workflow_tasks_outbox_process_ack_artifact_and_domain_job",
		EvidenceRefs:     compactRunLivenessStringsV0(input.EvidenceRefs),
		CausalSourceRefs: externalWorkCausalSourceRefsV0(input),
	}
	domainStatus := normalizeExternalWorkStatusV0(input.DomainJobStatus)
	projectionStatus := normalizeExternalWorkStatusV0(input.ProjectionStatus)
	decision.CausalVerdict = externalWorkCausalVerdictV0(input, domainStatus, projectionStatus)

	if input.AgentAckCompleted {
		decision.Action = ExternalWorkReconcileActionIngestAckV0
		decision.Justified = true
		if (input.LocalArtifactPresent || domainStatus == "completed") &&
			decision.CausalVerdict.Clase == orquestaestadovivo.VeredictoTerminalByArtifactV0 {
			decision.PublicStatus = ExternalWorkPublicStatusCompletedV0
			decision.Reason = "completed_ack_observed_with_closure_evidence"
			decision.NextAction = "ingest_ack_checkpoint_and_keep_completed_projection"
			decision.Reconciled = true
			return decision
		}
		decision.PublicStatus = ExternalWorkPublicStatusBlockedV0
		decision.Reason = "completed_ack_observed_without_domain_closure"
		if input.LocalArtifactPresent || domainStatus == "completed" {
			decision.Reason = decision.CausalVerdict.ReasonCode
		}
		decision.NextAction = "ingest_ack_checkpoint_and_submit_or_confirm_domain_artifact"
		return decision
	}
	if input.LocalArtifactPresent && domainStatus == "completed" &&
		decision.CausalVerdict.Clase == orquestaestadovivo.VeredictoTerminalByArtifactV0 {
		decision.PublicStatus = ExternalWorkPublicStatusCompletedV0
		decision.Action = ExternalWorkReconcileActionNoopV0
		decision.Reason = "domain_completed_with_local_artifact"
		decision.NextAction = "keep_completed_projection"
		decision.Justified = true
		return decision
	}
	if decision.CausalVerdict.RequiereReparacion {
		decision.PublicStatus = ExternalWorkPublicStatusBlockedV0
		decision.Reason = decision.CausalVerdict.ReasonCode
		decision.NextAction = "repair_causal_evidence_before_reconcile"
		decision.Justified = true
		return decision
	}
	if externalWorkDomainPendingV0(domainStatus) &&
		externalWorkProjectionDoneV0(projectionStatus) &&
		input.ProjectionTaskCount == 0 &&
		input.WorkflowTaskOpenCount == 0 &&
		input.PendingOutboxCount == 0 {
		decision.PublicStatus = ExternalWorkPublicStatusBlockedV0
		decision.Action = ExternalWorkReconcileActionReopenOrBlockDomainV0
		decision.Reason = "projection_done_but_domain_job_pending_without_open_tasks"
		decision.NextAction = "create_causal_followup_or_mark_blocked_until_domain_job_progress"
		decision.Justified = true
		decision.Reconciled = true
		return decision
	}
	if decision.CausalVerdict.Clase == orquestaestadovivo.VeredictoRunningConfirmedV0 {
		decision.PublicStatus = ExternalWorkPublicStatusRunningV0
		decision.Action = ExternalWorkReconcileActionObserveV0
		decision.Reason = "live_process_open_task_pending_outbox_or_liveness"
		decision.NextAction = "observe_resident_worker_and_ingest_progress"
		decision.Justified = true
		return decision
	}
	if input.WorkflowTaskOpenCount > 0 || input.PendingOutboxCount > 0 {
		decision.PublicStatus = ExternalWorkPublicStatusPendingV0
		decision.Action = ExternalWorkReconcileActionObserveV0
		decision.Reason = "causal_work_pending_without_runtime_liveness"
		decision.NextAction = "observe_resident_worker_and_ingest_progress"
		decision.Justified = true
		return decision
	}
	if externalWorkDomainPendingV0(domainStatus) {
		decision.PublicStatus = ExternalWorkPublicStatusPendingV0
		decision.Action = ExternalWorkReconcileActionWaitDomainV0
		decision.Reason = "domain_job_pending_without_terminal_evidence"
		decision.NextAction = "wait_or_request_domain_progress_evidence"
		decision.Justified = true
		return decision
	}
	return decision
}

func externalWorkCausalVerdictV0(
	input ExternalWorkReconciliationInputV0,
	domainStatus string,
	projectionStatus string,
) orquestaestadovivo.VeredictoCausalV0 {
	evidencias := []orquestaestadovivo.EvidenciaEstadoV0{{
		RunRef: input.RunRef, Fuente: "run_projection", Estado: projectionStatus,
		EvidenceRefs: input.EvidenceRefs,
	}}
	if input.AgentAckCompleted && (input.LocalArtifactPresent || domainStatus == "completed") ||
		input.LocalArtifactPresent && domainStatus == "completed" {
		evidencias = append(evidencias, orquestaestadovivo.EvidenciaEstadoV0{
			RunRef: input.RunRef, Fuente: "external_work_terminal_receipt", Estado: "completed",
			Terminal: true, Aceptado: true, EvidenceRefs: input.EvidenceRefs,
		})
	}
	if input.ProcessRegistryChecked {
		evidencias = append(evidencias, orquestaestadovivo.EvidenciaEstadoV0{
			RunRef: input.RunRef, Fuente: "process_registry", Estado: projectionStatus,
			Scope:                       orquestaestadovivo.ScopeGoalExecutionV0,
			RuntimeIdentityRef:          strings.TrimSpace(input.RuntimeIdentityRef),
			RuntimeGenerationRef:        strings.TrimSpace(input.RuntimeGenerationRef),
			RuntimeIdentityMismatch:     input.RuntimeIdentityMismatch,
			RuntimeObservationAttempted: true, RuntimeObservado: true, ProcesoVivo: input.ProcessAlive,
			EvidenceRefs: input.EvidenceRefs,
		})
	}
	return orquestaestadovivo.DerivarVeredictoCausalV0(evidencias)
}

func externalWorkCausalSourceRefsV0(input ExternalWorkReconciliationInputV0) []string {
	refs := []string{}
	if strings.TrimSpace(input.ProjectionStatus) != "" {
		refs = append(refs, "run_projection")
	}
	if input.WorkflowTaskOpenCount > 0 || input.ProjectionTaskCount > 0 {
		refs = append(refs, "workflow_task_store")
	}
	if input.PendingOutboxCount > 0 {
		refs = append(refs, "outbox_ledger")
	}
	if input.ProcessRegistryChecked || input.ProcessAlive {
		refs = append(refs, "process_registry")
	}
	if input.AgentAckCompleted || input.AgentCheckpointPresent {
		refs = append(refs, "agent_ack_checkpoint")
	}
	if input.LocalArtifactPresent {
		refs = append(refs, "local_artifact")
	}
	if strings.TrimSpace(input.DomainJobStatus) != "" {
		refs = append(refs, "domain_job_state")
	}
	return compactRunLivenessStringsV0(refs)
}

func externalWorkProjectionDoneV0(status string) bool {
	switch status {
	case "done", "completed", "complete", "closed", "quiescent", "delivered":
		return true
	default:
		return false
	}
}

func externalWorkDomainPendingV0(status string) bool {
	switch status {
	case "pending", "queued", "accepted", "in_progress", "running":
		return true
	default:
		return false
	}
}

func normalizeExternalWorkStatusV0(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
