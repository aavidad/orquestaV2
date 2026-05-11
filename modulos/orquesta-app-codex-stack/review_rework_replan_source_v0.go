package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"

	orquestacorereplanner "orquesta/modulos/orquesta-core-replanner"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

type ReviewReworkReplanSourceV0 struct {
	Store    orquestaruntimecodexdelivery.CodexReceiptDescriptorStorePortV0
	Capacity CapacityConfigV0
}

var _ orquestacionnucleoapp.ReviewReworkReplanPlanProviderPortV0 = ReviewReworkReplanSourceV0{}

type reviewReworkProjectionV0 struct {
	ReworkRequestRef string
	ReviewResultRef  string
	ReviewRequestID  string
	DeliveryRef      string
}

type reviewResultProjectionV0 struct {
	ReviewResultRef string
	Status          orquestacoreworkflow.ReviewResultStatusV0
	ReviewRequestID string
	DeliveryRef     string
}

func (source ReviewReworkReplanSourceV0) BuildReviewReworkReplanPlansV0(
	ctx context.Context,
	request orquestacionnucleoapp.ReviewReworkReplanPlanRequestV0,
) ([]orquestacionnucleoapp.ReviewReworkReplanPlanV0, error) {
	if source.Store == nil {
		return nil, fmt.Errorf("review_rework_replan_source: receipt_store requerido")
	}
	descriptors, err := source.Store.ListCodexReceiptDescriptorsV0(
		ctx,
		orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{RunID: request.Run.RunID},
	)
	if err != nil {
		return nil, err
	}
	plans := make([]orquestacionnucleoapp.ReviewReworkReplanPlanV0, 0, len(request.Run.ReworkRequests))
	for _, raw := range request.Run.ReworkRequests {
		rework, ok := parseReviewReworkProjectionV0(raw)
		if !ok || reworkRetryAgentAlreadyRequestedV0(request.Run, rework.ReworkRequestRef) {
			continue
		}
		result, ok := reviewResultForReworkV0(request.Run, rework)
		if !ok || !reviewResultNeedsReworkV0(result.Status) {
			continue
		}
		plans = append(plans, source.planForReworkV0(request, rework, result, descriptors))
	}
	return plans, nil
}

func (source ReviewReworkReplanSourceV0) planForReworkV0(
	request orquestacionnucleoapp.ReviewReworkReplanPlanRequestV0,
	rework reviewReworkProjectionV0,
	result reviewResultProjectionV0,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) orquestacionnucleoapp.ReviewReworkReplanPlanV0 {
	taskRef, fallback := taskRefForReworkDeliveryV0(request.Run, rework.DeliveryRef, descriptors)
	suffix := reviewReworkReplanSafeRefV0(rework.ReworkRequestRef)
	evidence := reviewReworkReplanEvidenceRefsV0(request, rework, result, fallback)
	summary := "Repetir tarea tras revision no aceptada."
	return orquestacionnucleoapp.ReviewReworkReplanPlanV0{
		CandidateRef:               "review-rework-replan-candidate-ref-" + suffix,
		ReplanRef:                  "replan-ref-" + suffix,
		SignalRef:                  "review-rework-signal-ref-" + suffix,
		ReworkRequestRef:           rework.ReworkRequestRef,
		TaskRef:                    taskRef,
		ReasonRef:                  "reason-ref-" + reviewReworkReplanSafeRefV0(result.ReviewResultRef),
		RequestedAction:            orquestacorereplanner.ReplanActionRetryTaskV0,
		CapacityRequestRef:         "capacity-ref-" + suffix,
		AgentRequestID:             "agent-ref-" + suffix,
		AgentRole:                  "implementacion",
		MinimumRecommendedCapacity: reviewReworkReplanCapacityV0(source.Capacity),
		Summary:                    summary,
		EvidenceRefs:               evidence,
		ReviewResult: orquestacoreworkflow.ReviewResultV0{
			ReviewResultRef: result.ReviewResultRef,
			ReviewRequestID: result.ReviewRequestID,
			DeliveryRef:     result.DeliveryRef,
			Status:          result.Status,
			Summary:         summary,
			EvidenceRefs:    evidence,
			QualityGateRef:  "quality-gate-ref-" + reviewReworkReplanSafeRefV0(result.DeliveryRef),
		},
	}
}

func taskRefForReworkDeliveryV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	deliveryRef string,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) (string, bool) {
	deliveryRef = strings.TrimSpace(deliveryRef)
	for _, descriptor := range descriptors {
		if strings.TrimSpace(descriptor.Spec.AgentPacket.DeliveryRefs.AckRef) != deliveryRef {
			continue
		}
		taskRef := strings.TrimSpace(descriptor.Spec.AgentPacket.Task.TaskRef)
		if taskRef != "" {
			return taskRef, false
		}
	}
	return firstReviewReworkTaskRefV0(run), true
}

func firstReviewReworkTaskRefV0(run orquestacoreworkflow.OrchestrationRunV0) string {
	tasks := compactStringsV0(run.Tasks)
	if len(tasks) == 0 {
		return ""
	}
	return tasks[0]
}

func reviewResultForReworkV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	rework reviewReworkProjectionV0,
) (reviewResultProjectionV0, bool) {
	for _, raw := range run.ReviewResults {
		result, ok := parseReviewResultProjectionV0(raw)
		if !ok || result.ReviewResultRef != rework.ReviewResultRef {
			continue
		}
		if result.ReviewRequestID == rework.ReviewRequestID &&
			result.DeliveryRef == rework.DeliveryRef {
			return result, true
		}
	}
	return reviewResultProjectionV0{}, false
}

func parseReviewReworkProjectionV0(value string) (reviewReworkProjectionV0, bool) {
	reworkRef, tail, ok := strings.Cut(strings.TrimSpace(value), "#review_result:")
	if !ok {
		return reviewReworkProjectionV0{}, false
	}
	resultRef, tail, ok := strings.Cut(tail, "#review_request:")
	if !ok {
		return reviewReworkProjectionV0{}, false
	}
	reviewRef, deliveryRef, ok := strings.Cut(tail, "#delivery:")
	if !ok {
		return reviewReworkProjectionV0{}, false
	}
	projection := reviewReworkProjectionV0{
		ReworkRequestRef: strings.TrimSpace(reworkRef),
		ReviewResultRef:  strings.TrimSpace(resultRef),
		ReviewRequestID:  strings.TrimSpace(reviewRef),
		DeliveryRef:      strings.TrimSpace(deliveryRef),
	}
	return projection, projection.ReworkRequestRef != "" && projection.ReviewResultRef != "" &&
		projection.ReviewRequestID != "" && projection.DeliveryRef != ""
}

func parseReviewResultProjectionV0(value string) (reviewResultProjectionV0, bool) {
	resultRef, tail, ok := strings.Cut(strings.TrimSpace(value), "#review_result:")
	if !ok {
		return reviewResultProjectionV0{}, false
	}
	status, tail, ok := strings.Cut(tail, "#review_request:")
	if !ok {
		return reviewResultProjectionV0{}, false
	}
	reviewRef, deliveryRef, ok := strings.Cut(tail, "#delivery:")
	if !ok {
		return reviewResultProjectionV0{}, false
	}
	projection := reviewResultProjectionV0{
		ReviewResultRef: strings.TrimSpace(resultRef),
		Status:          orquestacoreworkflow.ReviewResultStatusV0(strings.TrimSpace(status)),
		ReviewRequestID: strings.TrimSpace(reviewRef),
		DeliveryRef:     strings.TrimSpace(deliveryRef),
	}
	return projection, projection.ReviewResultRef != "" &&
		projection.ReviewRequestID != "" && projection.DeliveryRef != ""
}

func reworkRetryAgentAlreadyRequestedV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	reworkRef string,
) bool {
	agentRef := "agent-ref-" + reviewReworkReplanSafeRefV0(reworkRef)
	return reviewReworkReplanStringInSetV0(run.Agents, agentRef) ||
		reviewReworkReplanStringInSetV0(run.StartedAgents, agentRef)
}

func reviewReworkReplanStringInSetV0(values []string, want string) bool {
	want = strings.TrimSpace(want)
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}

func reviewResultNeedsReworkV0(status orquestacoreworkflow.ReviewResultStatusV0) bool {
	return status == orquestacoreworkflow.ReviewResultStatusChangesRequestedV0 ||
		status == orquestacoreworkflow.ReviewResultStatusRejectedV0
}

func reviewReworkReplanCapacityV0(
	config CapacityConfigV0,
) orquestacoreworkflow.OrchestrationCapacityRecommendationV0 {
	if strings.TrimSpace(string(config.Tier)) != "" {
		return config.Tier
	}
	return orquestacoreworkflow.OrchestrationCapacityHighV0
}

func reviewReworkReplanEvidenceRefsV0(
	request orquestacionnucleoapp.ReviewReworkReplanPlanRequestV0,
	rework reviewReworkProjectionV0,
	result reviewResultProjectionV0,
	fallback bool,
) []string {
	refs := []string{
		"evidence-ref-review-rework-replan",
		rework.ReworkRequestRef,
		result.ReviewResultRef,
		rework.DeliveryRef,
	}
	if fallback {
		refs = append(refs, "evidence-ref-review-rework-task-fallback")
	}
	return compactStringsV0(append(refs, request.EvidenceRefs...))
}

func reviewReworkReplanSafeRefV0(value string) string {
	value = strings.TrimSpace(value)
	replacer := strings.NewReplacer("\\", "-", "/", "-", " ", "-", "#", "-", ":", "-")
	value = strings.Trim(replacer.Replace(value), "-")
	if value == "" {
		return "sin-ref"
	}
	return value
}
