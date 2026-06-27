package orquestaappdirectorservice

import (
	"encoding/json"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"strings"
)

func operationalDirectorPlanProjectionReflectedV0(ref string, values []string) bool {
	ref = strings.TrimSpace(ref)
	for _, existing := range values {
		existing = strings.TrimSpace(existing)
		if existing == ref || strings.HasPrefix(existing, ref+"#") {
			return true
		}
	}
	return false
}

func operationalDirectorPlanReviewTraceFromEventsV0(
	events []orquestacoreworkflow.OrchestrationEventV0,
) operationalDirectorPlanReviewTraceV0 {
	trace := operationalDirectorPlanReviewTraceV0{
		Deliveries:      map[string]orquestacoreworkflow.DeliveryRegisteredPayloadV0{},
		ReviewRequests:  map[string]orquestacoreworkflow.ReviewRequestedPayloadV0{},
		ReviewResults:   map[string]orquestacoreworkflow.ReviewResultV0{},
		AcceptedReviews: map[string]orquestacoreworkflow.ReviewAcceptedPayloadV0{},
		ReworkRequests:  map[string]orquestacoreworkflow.ReworkRequestedPayloadV0{},
		ReplanDecisions: map[string]orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{},
		QualityGates:    map[string]orquestacoreworkflow.QualityGateRecordedPayloadV0{},
	}
	for _, event := range events {
		switch event.EventType {
		case orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0:
			var payload orquestacoreworkflow.DeliveryRegisteredPayloadV0
			if operationalDirectorPlanDecodeEventPayloadV0(event, &payload) {
				ref := strings.TrimSpace(payload.DeliveryRef)
				if ref != "" && trace.Deliveries[ref].DeliveryRef == "" {
					trace.DeliveryRefs = append(trace.DeliveryRefs, ref)
				}
				trace.Deliveries[ref] = payload
			}
		case orquestacoreworkflow.OrchestrationEventReviewRequestedV0:
			var payload orquestacoreworkflow.ReviewRequestedPayloadV0
			if operationalDirectorPlanDecodeEventPayloadV0(event, &payload) {
				ref := strings.TrimSpace(payload.ReviewRequestID)
				if ref != "" && trace.ReviewRequests[ref].ReviewRequestID == "" {
					trace.ReviewRequestRefs = append(trace.ReviewRequestRefs, ref)
				}
				trace.ReviewRequests[ref] = payload
			}
		case orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0:
			var payload orquestacoreworkflow.ReviewResultV0
			if operationalDirectorPlanDecodeEventPayloadV0(event, &payload) {
				ref := strings.TrimSpace(payload.ReviewResultRef)
				if ref != "" && trace.ReviewResults[ref].ReviewResultRef == "" {
					trace.ReviewResultRefs = append(trace.ReviewResultRefs, ref)
				}
				trace.ReviewResults[ref] = payload
			}
		case orquestacoreworkflow.OrchestrationEventReviewAcceptedV0:
			var payload orquestacoreworkflow.ReviewAcceptedPayloadV0
			if operationalDirectorPlanDecodeEventPayloadV0(event, &payload) {
				ref := strings.TrimSpace(payload.AcceptedReviewRef)
				if ref != "" && trace.AcceptedReviews[ref].AcceptedReviewRef == "" {
					trace.AcceptedReviewRefs = append(trace.AcceptedReviewRefs, ref)
				}
				trace.AcceptedReviews[ref] = payload
			}
		case orquestacoreworkflow.OrchestrationEventReworkRequestedV0:
			var payload orquestacoreworkflow.ReworkRequestedPayloadV0
			if operationalDirectorPlanDecodeEventPayloadV0(event, &payload) {
				ref := strings.TrimSpace(payload.ReworkRequestRef)
				if ref != "" && trace.ReworkRequests[ref].ReworkRequestRef == "" {
					trace.ReworkRequestRefs = append(trace.ReworkRequestRefs, ref)
				}
				trace.ReworkRequests[ref] = payload
			}
		case orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0:
			var payload orquestacoreworkflow.ReplanDecisionRecordedPayloadV0
			if operationalDirectorPlanDecodeEventPayloadV0(event, &payload) {
				ref := strings.TrimSpace(payload.ReplanRef)
				if ref != "" && trace.ReplanDecisions[ref].ReplanRef == "" {
					trace.ReplanDecisionRefs = append(trace.ReplanDecisionRefs, ref)
				}
				trace.ReplanDecisions[ref] = payload
			}
		case orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0:
			var payload orquestacoreworkflow.QualityGateRecordedPayloadV0
			if operationalDirectorPlanDecodeEventPayloadV0(event, &payload) {
				ref := strings.TrimSpace(payload.GateRef)
				if ref != "" && trace.QualityGates[ref].GateRef == "" {
					trace.QualityGateRefs = append(trace.QualityGateRefs, ref)
				}
				trace.QualityGates[ref] = payload
			}
		}
	}
	return trace
}

func operationalDirectorPlanDecodeEventPayloadV0(
	event orquestacoreworkflow.OrchestrationEventV0,
	out any,
) bool {
	return json.Unmarshal(event.Payload, out) == nil
}

func operationalDirectorPlanStateStepIDByKindV0(
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	kind orquestadirectoroperativo.OperationalDirectorStepKindV0,
) string {
	for _, step := range state.Steps {
		if step.Kind == kind {
			return strings.TrimSpace(step.StepID)
		}
	}
	return ""
}

func operationalDirectorPlanReviewMatchDeliveryRefsV0(
	matches []operationalDirectorPlanAcceptedReviewMatchV0,
) []string {
	refs := make([]string, 0, len(matches))
	for _, match := range matches {
		refs = append(refs, match.DeliveryRef)
	}
	return compactServiceRefsV0(refs)
}

func operationalDirectorPlanReviewMatchReviewResultRefsV0(
	matches []operationalDirectorPlanAcceptedReviewMatchV0,
) []string {
	refs := make([]string, 0, len(matches))
	for _, match := range matches {
		refs = append(refs, match.ReviewResultRef)
	}
	return compactServiceRefsV0(refs)
}

func operationalDirectorPlanReviewMatchAcceptedReviewRefsV0(
	matches []operationalDirectorPlanAcceptedReviewMatchV0,
) []string {
	refs := make([]string, 0, len(matches))
	for _, match := range matches {
		refs = append(refs, match.AcceptedReviewRef)
	}
	return compactServiceRefsV0(refs)
}

func operationalDirectorPlanReviewMatchEvidenceRefsV0(
	matches []operationalDirectorPlanAcceptedReviewMatchV0,
) []string {
	refs := make([]string, 0, len(matches))
	for _, match := range matches {
		refs = append(refs, match.EvidenceRefs...)
	}
	return compactServiceRefsV0(refs)
}

func allServiceRefsInSetV0(values []string, available []string) bool {
	for _, value := range compactServiceRefsV0(values) {
		if !startAppDirectorStringInSetV0(available, value) {
			return false
		}
	}
	return len(compactServiceRefsV0(values)) > 0
}

func firstOperationalDirectorTaskWaveRefV0(tasks []orquestacoreworkflow.WorkflowTaskV0) string {
	for _, task := range tasks {
		if value := strings.TrimSpace(task.WaveRef); value != "" {
			return value
		}
	}
	return ""
}

func firstOperationalDirectorTaskCohortRefV0(tasks []orquestacoreworkflow.WorkflowTaskV0) string {
	for _, task := range tasks {
		if value := strings.TrimSpace(task.CohortRef); value != "" {
			return value
		}
	}
	return ""
}

func firstOperationalDirectorTaskParentTaskRefV0(tasks []orquestacoreworkflow.WorkflowTaskV0) string {
	for _, task := range tasks {
		if value := strings.TrimSpace(task.ParentTaskRef); value != "" {
			return value
		}
	}
	return ""
}

func normalizeServiceWorkflowFunctionContractRefsV0(
	refs []orquestacoreworkflow.WorkflowFunctionContractRefV0,
) []orquestacoreworkflow.WorkflowFunctionContractRefV0 {
	out := make([]orquestacoreworkflow.WorkflowFunctionContractRefV0, 0, len(refs))
	for _, ref := range refs {
		compact := orquestacoreworkflow.WorkflowFunctionContractRefV0{
			ContractRef:  strings.TrimSpace(ref.ContractRef),
			FunctionName: strings.TrimSpace(ref.FunctionName),
		}
		if compact.ContractRef == "" && compact.FunctionName == "" {
			continue
		}
		out = append(out, compact)
	}
	return out
}
