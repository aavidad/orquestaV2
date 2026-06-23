package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"

	orquestacorereplanner "orquesta/modulos/orquesta-core-replanner"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

const reviewReworkReplanMaxRetryAgentsPerTaskV0 = 6

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
		if !ok {
			continue
		}
		result, ok := reviewResultForReworkV0(request.Run, rework)
		if !ok || !reviewResultNeedsReworkV0(result.Status) {
			continue
		}
		if reviewReworkReplanSkipSoftRailOnlyV0(request, rework, result) {
			continue
		}
		if reviewReworkReplanSkipExternalRailDocsLoopV0(rework, descriptors) {
			continue
		}
		if reviewReworkAlreadyResolvedByAcceptedFollowupV0(request.Run, rework, descriptors) {
			continue
		}
		plan := source.planForReworkV0(request, rework, result, descriptors)
		plan = reviewReworkPlanWithTaskBoundaryV0(request, plan, descriptors)
		if strings.TrimSpace(plan.CandidateRef) == "" {
			continue
		}
		if reworkRetryAgentAlreadyRequestedV0(request.Run, plan.AgentRequestID) {
			continue
		}
		if reworkRetryAgentLimitReachedV0(request.Run, plan.TaskRef) {
			continue
		}
		plans = append(plans, plan)
	}
	return plans, nil
}

func reviewReworkAlreadyResolvedByAcceptedFollowupV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	rework reviewReworkProjectionV0,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) bool {
	if codexStackReviewGateReworkAcceptanceAlreadyDoneV0(run, rework.DeliveryRef) {
		return true
	}
	_, deliveryByTask := codexStackReviewGateDescriptorTaskMapsV0(descriptors)
	for _, rawReplan := range run.ReplanDecisions {
		replan, ok := codexStackReviewGateParseReplanProjectionV0(rawReplan)
		if !ok || replan.SourceRef != rework.ReworkRequestRef {
			continue
		}
		if replan.AcceptedAction == string(orquestacoreworkflow.ReplanDecisionActionSplitTaskV0) &&
			codexStackReviewGateFollowupsAcceptedV0(run, deliveryByTask, replan.FollowupRefs) {
			return true
		}
		if reviewReworkRetryOrReplaceFollowupAcceptedV0(run, replan, descriptors) {
			return true
		}
	}
	return false
}

func reviewReworkRetryOrReplaceFollowupAcceptedV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	replan codexStackReviewGateReplanProjectionV0,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) bool {
	if replan.AcceptedAction != string(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0) &&
		replan.AcceptedAction != string(orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0) {
		return false
	}
	for _, followupRef := range replan.FollowupRefs {
		deliveryRef := reviewReworkDeliveryRefForFollowupV0(followupRef, descriptors)
		if deliveryRef == "" {
			continue
		}
		if reviewReworkDeliveryAcceptedV0(run, deliveryRef) {
			return true
		}
	}
	return false
}

func reviewReworkDeliveryRefForFollowupV0(
	followupRef string,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) string {
	followupRef = strings.TrimSpace(followupRef)
	if followupRef == "" {
		return ""
	}
	for _, descriptor := range descriptors {
		for _, candidate := range reviewReworkDescriptorAgentRefsV0(descriptor) {
			if strings.TrimSpace(candidate) == followupRef {
				return strings.TrimSpace(descriptor.Spec.AgentPacket.DeliveryRefs.AckRef)
			}
		}
		if strings.TrimSpace(descriptor.Spec.AgentPacket.Task.TaskRef) == followupRef {
			return strings.TrimSpace(descriptor.Spec.AgentPacket.DeliveryRefs.AckRef)
		}
	}
	return ""
}

func reviewReworkDeliveryAcceptedV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	deliveryRef string,
) bool {
	deliveryRef = strings.TrimSpace(deliveryRef)
	if deliveryRef == "" || !reviewReworkReplanStringInSetV0(run.Deliveries, deliveryRef) {
		return false
	}
	return reviewReworkReplanStringInSetV0(run.AcceptedReviews, "accepted-review-ref-"+deliveryRef)
}

func (source ReviewReworkReplanSourceV0) planForReworkV0(
	request orquestacionnucleoapp.ReviewReworkReplanPlanRequestV0,
	rework reviewReworkProjectionV0,
	result reviewResultProjectionV0,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) orquestacionnucleoapp.ReviewReworkReplanPlanV0 {
	taskRef, fallback := taskRefForReworkDeliveryV0(request.Run, rework.DeliveryRef, descriptors)
	suffix := reviewReworkReplanSafeRefV0(rework.ReworkRequestRef)
	agentSuffix := reviewReworkReplanAgentSuffixV0(rework, taskRef)
	missingTargets := reviewReworkMissingWriteSetTargetsV0(rework.DeliveryRef, descriptors)
	evidence := reviewReworkReplanEvidenceRefsV0(request, rework, result, fallback, missingTargets)
	summary := reviewReworkReplanSummaryV0(missingTargets)
	return orquestacionnucleoapp.ReviewReworkReplanPlanV0{
		CandidateRef:               "review-rework-replan-candidate-ref-" + suffix,
		ReplanRef:                  "replan-ref-" + suffix,
		SignalRef:                  "review-rework-signal-ref-" + suffix,
		ReworkRequestRef:           rework.ReworkRequestRef,
		TaskRef:                    taskRef,
		ReasonRef:                  "reason-ref-" + reviewReworkReplanSafeRefV0(result.ReviewResultRef),
		RequestedAction:            orquestacorereplanner.ReplanActionRetryTaskV0,
		CapacityRequestRef:         "capacity-ref-" + agentSuffix,
		AgentRequestID:             "agent-ref-" + agentSuffix,
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
	agentRef string,
) bool {
	return reviewReworkReplanStringInSetV0(run.Agents, agentRef) ||
		reviewReworkReplanStringInSetV0(run.StartedAgents, agentRef)
}

func reviewReworkProgrammingAgentAlreadyAssignedToTaskV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	taskRef string,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) bool {
	taskRef = strings.TrimSpace(taskRef)
	if taskRef == "" {
		return false
	}
	for _, descriptor := range descriptors {
		if strings.TrimSpace(descriptor.Spec.AgentPacket.Task.TaskRef) != taskRef {
			continue
		}
		for _, agentRef := range reviewReworkDescriptorAgentRefsV0(descriptor) {
			if reviewReworkRunHasProgrammingAgentV0(run, agentRef) {
				return true
			}
		}
	}
	return false
}

func reviewReworkDescriptorAgentRefsV0(
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) []string {
	return compactStringsV0([]string{
		descriptor.AgentRef,
		descriptor.Spec.RequestID,
		descriptor.Spec.AgentPacket.RequestID,
	})
}

func reviewReworkRunHasProgrammingAgentV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	agentRef string,
) bool {
	agentRef = strings.TrimSpace(agentRef)
	if agentRef == "" {
		return false
	}
	return reviewReworkReplanStringInSetV0(run.Agents, agentRef) ||
		reviewReworkReplanStringInSetV0(run.StartedAgents, agentRef) ||
		reviewReworkReplanStringInSetV0(run.DeliveredAgents, agentRef) ||
		reviewReworkReplanStringInSetV0(run.FailedAgents, agentRef) ||
		reviewReworkReplanStringInSetV0(run.LostAgents, agentRef) ||
		reviewReworkReplanStringInSetV0(run.StoppedAgents, agentRef)
}

func reworkRetryAgentLimitReachedV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	taskRef string,
) bool {
	prefix := reviewReworkReplanTaskRetryAgentPrefixV0(taskRef)
	seen := map[string]bool{}
	for _, value := range append(append([]string(nil), run.Agents...), run.StartedAgents...) {
		agentRef := strings.TrimSpace(value)
		if !strings.HasPrefix(agentRef, prefix) || seen[agentRef] {
			continue
		}
		seen[agentRef] = true
	}
	return len(seen) >= reviewReworkReplanMaxRetryAgentsPerTaskV0
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
