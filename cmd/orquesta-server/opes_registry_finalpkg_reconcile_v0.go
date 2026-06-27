package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"strings"
	"time"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestastatefile "orquesta/modulos/orquesta-state-file"
)

const opesRegistryFinalPkgReconcilerRequestedByV0 = "opes_registry_finalpkg_reconciler"

func reconcileOPESRegistryFinalPkgCompletedRunsV0(
	ctx context.Context,
	config opesRegistryFinalPkgConfigV0,
	drifts []opesRegistryFinalPkgCompletionDriftV0,
	summary *opesRegistryFinalPkgSummaryV0,
) {
	if summary == nil || !config.ReconcileCompleted || config.DryRun || len(drifts) == 0 {
		return
	}
	limit := config.ReconcileLimit
	if limit < 1 {
		limit = defaultOPESRegistryFinalPkgReconcileLimitV0
	}
	selected := drifts
	if len(selected) > limit {
		selected = selected[:limit]
	}
	store, err := orquestastatefile.NewStoreV0(orquestastatefile.ConfigV0{
		RootDir: config.OrchestrationStateRoot,
	})
	if err != nil {
		summary.ReconcileSkipped += len(selected)
		summary.Errors = append(summary.Errors, opesRegistryFinalPkgPublicErrorV0{Code: "reconcile_store_error"})
		return
	}
	for _, drift := range selected {
		result, err := reconcileOPESRegistryFinalPkgRunV0(ctx, store, config, drift)
		if err != nil {
			result.Status = "reconcile_error"
			summary.ReconcileSkipped++
			summary.Errors = append(summary.Errors, opesRegistryFinalPkgPublicErrorV0{
				TopicID: result.TopicID,
				Code:    compactExternalBridgeErrorCodeV0(err),
			})
		}
		if result.Status == "reconciled" {
			summary.Reconciled++
		} else if result.Status != "" && result.Status != "already_closed" && err == nil {
			summary.ReconcileSkipped++
		}
		if result.Status != "" {
			summary.Results = append(summary.Results, result)
		}
	}
}

func reconcileOPESRegistryFinalPkgRunV0(
	ctx context.Context,
	store *orquestastatefile.StoreV0,
	config opesRegistryFinalPkgConfigV0,
	drift opesRegistryFinalPkgCompletionDriftV0,
) (opesRegistryFinalPkgResultV0, error) {
	runRef := strings.TrimSpace(drift.RunRef)
	topicID := strings.TrimSpace(drift.TopicID)
	result := opesRegistryFinalPkgResultV0{TopicID: topicID, RunRef: runRef}
	if runRef == "" {
		return result, errors.New("missing_run_ref")
	}
	run, err := store.LoadRunV0(ctx, runRef)
	if err != nil {
		return result, err
	}
	if topicID == "" {
		topicID, _ = opesRegistryFinalPkgTopicIDFromRunRefV0(config, run.RunID)
		result.TopicID = topicID
	}
	if run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		result.Status = "already_closed"
		return result, nil
	}
	if !opesRegistryFinalPkgPackageCompleteV0(filepath.Join(config.CourseRoot, "tema_"+topicID, "paquete_final")) {
		result.Status = "package_incomplete"
		return result, nil
	}
	if len(run.Tasks) == 0 {
		return result, errors.New("missing_task")
	}
	if len(run.Deliveries) == 0 {
		return result, errors.New("missing_delivery")
	}
	evidenceRefs := opesRegistryFinalPkgReconcileEvidenceRefsV0(run, topicID)
	run, err = reconcileOPESRegistryFinalPkgBlockersV0(ctx, store, run, evidenceRefs)
	if err != nil {
		return result, err
	}
	if run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		result.Status = "already_closed"
		return result, nil
	}
	if run.Status != orquestacoreworkflow.OrchestrationRunStatusActiveV0 {
		return result, errors.New("run_not_active")
	}
	if len(run.ClosedTasks) == 0 {
		run, err = ensureOPESRegistryFinalPkgPhaseV0(ctx, store, run, orquestacoreworkflow.OrchestrationPhaseRevisionV0, "Preparar cierre causal de paquete OPES completo.")
		if err != nil {
			return result, err
		}
		run, err = ensureOPESRegistryFinalPkgDeterministicReviewV0(ctx, store, run, topicID, evidenceRefs)
		if err != nil {
			return result, err
		}
		evidenceRefs = opesRegistryFinalPkgReconcileEvidenceRefsV0(run, topicID)
		command, err := opesRegistryFinalPkgCloseTaskCommandV0(run, evidenceRefs)
		if err != nil {
			return result, err
		}
		run, err = handleOPESRegistryFinalPkgWorkflowCommandV0(ctx, store, command)
		if err != nil {
			return result, err
		}
	}
	closedTaskRef := firstOPESRegistryFinalPkgStringV0(run.ClosedTasks)
	if closedTaskRef == "" {
		return result, errors.New("missing_closed_task")
	}
	validationRef := firstOPESRegistryFinalPkgStringV0(run.Validations)
	if validationRef == "" {
		validationRef = opesRegistryFinalPkgValidationRefV0(run.RunID, topicID)
		if !opesRegistryFinalPkgPhaseActiveV0(run, orquestacoreworkflow.OrchestrationPhaseValidacionFinalV0) &&
			!opesRegistryFinalPkgPhaseActiveV0(run, orquestacoreworkflow.OrchestrationPhaseCierreV0) {
			run, err = ensureOPESRegistryFinalPkgPhaseV0(ctx, store, run, orquestacoreworkflow.OrchestrationPhaseValidacionFinalV0, "Validar paquete OPES completo antes de cierre.")
			if err != nil {
				return result, err
			}
		}
		command, err := opesRegistryFinalPkgRegisterValidationCommandV0(run, closedTaskRef, validationRef, evidenceRefs)
		if err != nil {
			return result, err
		}
		run, err = handleOPESRegistryFinalPkgWorkflowCommandV0(ctx, store, command)
		if err != nil {
			return result, err
		}
	}
	closureRef := firstOPESRegistryFinalPkgStringV0(run.Closures)
	if closureRef == "" {
		closureRef = opesRegistryFinalPkgClosureRefV0(run.RunID, topicID)
		run, err = ensureOPESRegistryFinalPkgPhaseV0(ctx, store, run, orquestacoreworkflow.OrchestrationPhaseCierreV0, "Cerrar run OPES con paquete completo.")
		if err != nil {
			return result, err
		}
		command, err := opesRegistryFinalPkgCloseRunCommandV0(run, closureRef, validationRef, evidenceRefs)
		if err != nil {
			return result, err
		}
		run, err = handleOPESRegistryFinalPkgWorkflowCommandV0(ctx, store, command)
		if err != nil {
			return result, err
		}
	}
	if run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		return result, errors.New("run_not_closed")
	}
	result.Status = "reconciled"
	return result, nil
}

func ensureOPESRegistryFinalPkgDeterministicReviewV0(
	ctx context.Context,
	store *orquestastatefile.StoreV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	topicID string,
	evidenceRefs []string,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	if len(run.AcceptedReviews) > 0 {
		return run, nil
	}
	deliveryRef := firstOPESRegistryFinalPkgStringV0(run.Deliveries)
	if deliveryRef == "" {
		return run, errors.New("missing_delivery")
	}
	reviewRequestRef := opesRegistryFinalPkgReviewRequestRefV0(run.RunID, topicID)
	reviewResultRef := opesRegistryFinalPkgReviewResultRefV0(run.RunID, topicID)
	acceptedReviewRef := opesRegistryFinalPkgAcceptedReviewRefV0(run.RunID, topicID)
	command, err := opesRegistryFinalPkgRequestReviewCommandV0(run, reviewRequestRef, deliveryRef, evidenceRefs)
	if err != nil {
		return run, err
	}
	run, err = handleOPESRegistryFinalPkgWorkflowCommandV0(ctx, store, command)
	if err != nil {
		return run, err
	}
	command, err = opesRegistryFinalPkgRecordDeterministicReviewCommandV0(
		run,
		reviewRequestRef,
		reviewResultRef,
		deliveryRef,
		topicID,
		evidenceRefs,
	)
	if err != nil {
		return run, err
	}
	run, err = handleOPESRegistryFinalPkgWorkflowCommandV0(ctx, store, command)
	if err != nil {
		return run, err
	}
	command, err = opesRegistryFinalPkgAcceptDeterministicReviewCommandV0(
		run,
		reviewRequestRef,
		acceptedReviewRef,
		deliveryRef,
		evidenceRefs,
	)
	if err != nil {
		return run, err
	}
	return handleOPESRegistryFinalPkgWorkflowCommandV0(ctx, store, command)
}

func reconcileOPESRegistryFinalPkgBlockersV0(
	ctx context.Context,
	store *orquestastatefile.StoreV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	evidenceRefs []string,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	blockers := append([]string(nil), run.Blockers...)
	var err error
	for _, blockerID := range blockers {
		command, buildErr := opesRegistryFinalPkgResolveBlockerCommandV0(run.RunID, blockerID, evidenceRefs)
		if buildErr != nil {
			return run, buildErr
		}
		run, err = handleOPESRegistryFinalPkgWorkflowCommandV0(ctx, store, command)
		if err != nil {
			return run, err
		}
	}
	return run, nil
}

func ensureOPESRegistryFinalPkgPhaseV0(
	ctx context.Context,
	store *orquestastatefile.StoreV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	phaseID orquestacoreworkflow.OrchestrationPhaseIDV0,
	reason string,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	if opesRegistryFinalPkgPhaseActiveV0(run, phaseID) {
		return run, nil
	}
	command, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		opesRegistryFinalPkgCommandMetaV0(run.RunID, "open-"+string(phaseID), string(phaseID)),
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(phaseID),
			Reason:  reason,
		},
	)
	if err != nil {
		return run, err
	}
	return handleOPESRegistryFinalPkgWorkflowCommandV0(ctx, store, command)
}

func opesRegistryFinalPkgRequestReviewCommandV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	reviewRequestRef string,
	deliveryRef string,
	evidenceRefs []string,
) (orquestacoreworkflow.OrchestrationCommandV0, error) {
	return orquestacoreworkflow.NewRequestReviewCommandV0(
		opesRegistryFinalPkgCommandMetaV0(run.RunID, "request-deterministic-review", reviewRequestRef),
		orquestacoreworkflow.RequestReviewCommandPayloadV0{
			ReviewRequestID: reviewRequestRef,
			PhaseID:         string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			DeliveryRef:     deliveryRef,
			Summary:         "Revision determinista OPES finalpkg basada en contrato de paquete y banco de preguntas publicable.",
			EvidenceRefs:    evidenceRefs,
		},
	)
}

func opesRegistryFinalPkgRecordDeterministicReviewCommandV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	reviewRequestRef string,
	reviewResultRef string,
	deliveryRef string,
	topicID string,
	evidenceRefs []string,
) (orquestacoreworkflow.OrchestrationCommandV0, error) {
	return orquestacoreworkflow.NewRecordReviewResultCommandV0(
		opesRegistryFinalPkgCommandMetaV0(run.RunID, "record-deterministic-review", reviewResultRef),
		orquestacoreworkflow.ReviewResultV0{
			ReviewResultRef: reviewResultRef,
			ReviewRequestID: reviewRequestRef,
			DeliveryRef:     deliveryRef,
			Status:          orquestacoreworkflow.ReviewResultStatusAcceptedV0,
			Summary:         "Paquete OPES finalpkg aceptado por validacion determinista de artefactos requeridos y tests publicables.",
			EvidenceRefs:    evidenceRefs,
			QualityGateRef:  "quality-gate-ref-opes-finalpkg-" + opesRegistryFinalPkgSafeRefTokenV0(topicID),
		},
	)
}

func opesRegistryFinalPkgAcceptDeterministicReviewCommandV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	reviewRequestRef string,
	acceptedReviewRef string,
	deliveryRef string,
	evidenceRefs []string,
) (orquestacoreworkflow.OrchestrationCommandV0, error) {
	return orquestacoreworkflow.NewAcceptReviewCommandV0(
		opesRegistryFinalPkgCommandMetaV0(run.RunID, "accept-deterministic-review", acceptedReviewRef),
		orquestacoreworkflow.AcceptReviewCommandPayloadV0{
			AcceptedReviewRef: acceptedReviewRef,
			PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			ReviewRequestID:   reviewRequestRef,
			DeliveryRef:       deliveryRef,
			Summary:           "Review determinista aceptada para cierre OPES finalpkg; no depende de juicio subjetivo de agente.",
			EvidenceRefs:      evidenceRefs,
		},
	)
}

func opesRegistryFinalPkgResolveBlockerCommandV0(
	runRef string,
	blockerID string,
	evidenceRefs []string,
) (orquestacoreworkflow.OrchestrationCommandV0, error) {
	return orquestacoreworkflow.NewResolveRunBlockerCommandV0(
		opesRegistryFinalPkgCommandMetaV0(runRef, "resolve-blocker", blockerID),
		orquestacoreworkflow.ResolveRunBlockerCommandPayloadV0{
			BlockerID:    blockerID,
			ReasonCode:   "package_complete_run_non_terminal",
			Summary:      "Paquete final OPES completo; se resuelve bloqueo operativo residual para cierre causal.",
			EvidenceRefs: evidenceRefs,
		},
	)
}

func opesRegistryFinalPkgCloseTaskCommandV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	evidenceRefs []string,
) (orquestacoreworkflow.OrchestrationCommandV0, error) {
	taskID := firstOPESRegistryFinalPkgStringV0(run.Tasks)
	return orquestacoreworkflow.NewCloseTaskCommandV0(
		opesRegistryFinalPkgCommandMetaV0(run.RunID, "close-task", taskID),
		orquestacoreworkflow.CloseTaskCommandPayloadV0{
			TaskID:            taskID,
			PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			DeliveryRef:       firstOPESRegistryFinalPkgStringV0(run.Deliveries),
			AcceptedReviewRef: firstOPESRegistryFinalPkgStringV0(run.AcceptedReviews),
			Summary:           "Cierre causal de tarea OPES finalpkg con paquete completo en disco.",
			EvidenceRefs:      evidenceRefs,
		},
	)
}

func opesRegistryFinalPkgRegisterValidationCommandV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	closedTaskRef string,
	validationRef string,
	evidenceRefs []string,
) (orquestacoreworkflow.OrchestrationCommandV0, error) {
	return orquestacoreworkflow.NewRegisterFinalValidationCommandV0(
		opesRegistryFinalPkgCommandMetaV0(run.RunID, "register-validation", validationRef),
		orquestacoreworkflow.RegisterFinalValidationCommandPayloadV0{
			ValidationRef: validationRef,
			PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseValidacionFinalV0),
			ClosedTaskRef: closedTaskRef,
			Summary:       "Validacion final OPES finalpkg basada en contrato determinista de paquete, tests y review causal aceptada.",
			EvidenceRefs:  evidenceRefs,
		},
	)
}

func opesRegistryFinalPkgCloseRunCommandV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	closureRef string,
	validationRef string,
	evidenceRefs []string,
) (orquestacoreworkflow.OrchestrationCommandV0, error) {
	return orquestacoreworkflow.NewCloseRunCommandV0(
		opesRegistryFinalPkgCommandMetaV0(run.RunID, "close-run", closureRef),
		orquestacoreworkflow.CloseRunCommandPayloadV0{
			ClosureRef:    closureRef,
			PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseCierreV0),
			ValidationRef: validationRef,
			Summary:       "Run OPES finalpkg cerrado por reconciliacion causal de Orquesta.",
			EvidenceRefs:  evidenceRefs,
		},
	)
}

func handleOPESRegistryFinalPkgWorkflowCommandV0(
	ctx context.Context,
	store *orquestastatefile.StoreV0,
	command orquestacoreworkflow.OrchestrationCommandV0,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	if _, err := orquestacionnucleoapp.HandleStoredWorkflowCommandV0(ctx, store, store, command); err != nil {
		return orquestacoreworkflow.OrchestrationRunV0{}, err
	}
	return store.LoadRunV0(ctx, command.RunID)
}

func opesRegistryFinalPkgCommandMetaV0(
	runRef string,
	action string,
	subject string,
) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	key := opesRegistryFinalPkgShortHashV0(runRef + "|" + action + "|" + subject)
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      "cmd-opes-finalpkg-reconcile-" + action + "-" + key,
		RunID:          strings.TrimSpace(runRef),
		IdempotencyKey: "idem-opes-finalpkg-reconcile-" + action + "-" + key,
		CorrelationID:  "corr-opes-finalpkg-reconcile-" + opesRegistryFinalPkgShortHashV0(runRef),
		RequestedBy:    opesRegistryFinalPkgReconcilerRequestedByV0,
		OccurredAt:     time.Now().UTC().Format(time.RFC3339Nano),
	}
}

func opesRegistryFinalPkgReconcileEvidenceRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	topicID string,
) []string {
	return compactExternalBridgeStringsV0([]string{
		"evidence-ref-opes-finalpkg-package-complete",
		"evidence-ref-opes-finalpkg-deterministic-package-contract",
		"evidence-ref-opes-finalpkg-question-bank-present",
		"evidence-ref-opes-finalpkg-reconcile",
		strings.TrimSpace(run.RunID),
		"topic-ref-opes-finalpkg-" + opesRegistryFinalPkgSafeRefTokenV0(topicID),
		firstOPESRegistryFinalPkgStringV0(run.Deliveries),
		firstOPESRegistryFinalPkgStringV0(run.AcceptedReviews),
	})
}

func opesRegistryFinalPkgReviewRequestRefV0(runRef string, topicID string) string {
	return "review-request-ref-opes-finalpkg-" + opesRegistryFinalPkgShortHashV0(runRef+"|"+topicID)
}

func opesRegistryFinalPkgReviewResultRefV0(runRef string, topicID string) string {
	return "review-result-ref-opes-finalpkg-" + opesRegistryFinalPkgShortHashV0(runRef+"|"+topicID)
}

func opesRegistryFinalPkgAcceptedReviewRefV0(runRef string, topicID string) string {
	return "accepted-review-ref-opes-finalpkg-" + opesRegistryFinalPkgShortHashV0(runRef+"|"+topicID)
}

func opesRegistryFinalPkgValidationRefV0(runRef string, topicID string) string {
	return "validation-ref-opes-finalpkg-" + opesRegistryFinalPkgSafeRefTokenV0(topicID) + "-" + opesRegistryFinalPkgShortHashV0(runRef)
}

func opesRegistryFinalPkgClosureRefV0(runRef string, topicID string) string {
	return "closure-ref-opes-finalpkg-" + opesRegistryFinalPkgSafeRefTokenV0(topicID) + "-" + opesRegistryFinalPkgShortHashV0(runRef)
}

func opesRegistryFinalPkgSafeRefTokenV0(value string) string {
	token := compactExternalBridgeTokenV0(strings.ToLower(strings.TrimSpace(value)))
	if token == "" {
		return "unknown"
	}
	return token
}

func opesRegistryFinalPkgShortHashV0(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:])[:12]
}

func opesRegistryFinalPkgPhaseActiveV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	phaseID orquestacoreworkflow.OrchestrationPhaseIDV0,
) bool {
	if run.CurrentPhase != phaseID {
		return false
	}
	for _, phase := range run.Phases {
		if phase.ID == phaseID {
			return phase.Status == orquestacoreworkflow.OrchestrationPhaseStatusActiveV0
		}
	}
	return false
}

func firstOPESRegistryFinalPkgStringV0(values []string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
