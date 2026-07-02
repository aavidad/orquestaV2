package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

const (
	goalFirstRepairReceiptAttemptedEvidenceRefV0      = "evidence-ref-goal-first-repair-receipt-attempted"
	goalFirstRepairReceiptAcceptedEvidenceRefV0       = "evidence-ref-goal-first-repair-receipt-accepted"
	goalFirstRepairReceiptRequiresReworkEvidenceRefV0 = "evidence-ref-goal-first-repair-receipt-requires-rework"
	goalFirstRepairReceiptRequiresReworkIssueV0       = "repair_receipt_requires_rework"
)

type goalFirstReceiptRepairResultV0 struct {
	State    orquestagoal.GoalWorkStateV0
	Closure  orquestagoal.GoalClosureValidationV0
	Repaired bool
}

func repairGoalFirstReceiptFromMaterializedRefsV0(
	ctx context.Context,
	state orquestagoal.GoalWorkStateV0,
	refs orquestamcp.MCPDirectorGoalMaterializedRefsV0,
	store orquestagoal.GoalWorkStateStorePortV0,
	validator orquestagoal.GoalWorkClosureValidatorPortV0,
) (goalFirstReceiptRepairResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	state, err := orquestagoal.NewGoalWorkStateV0(state)
	if err != nil {
		return goalFirstReceiptRepairResultV0{}, err
	}
	out := goalFirstReceiptRepairResultV0{State: state}
	if store == nil ||
		!containsStringV0(refs.IssueCodes, orquestamcp.MCPGoalFirstMissingTerminalReceiptAfterArtifactsPassV0) ||
		goalFirstReceiptRepairAlreadyClosedV0(state) ||
		goalFirstReceiptRepairAlreadyAttemptedV0(state) {
		return out, nil
	}
	if validator == nil {
		validator = orquestagoal.DefaultGoalWorkClosureValidatorV0{}
	}
	result := goalFirstReceiptRepairResultFromRefsV0(state, refs)
	closure, err := validator.ValidateGoalWorkClosureV0(ctx, state.Spec, result)
	if err != nil {
		return goalFirstReceiptRepairResultV0{}, err
	}
	closure = goalFirstReceiptRepairClosureV0(closure)
	updated := state
	updated.Status = result.Status
	updated.LastResult = &result
	updated.LastClosure = &closure
	updated.EvidenceRefs = compactStringsV0(append(
		updated.EvidenceRefs,
		result.EvidenceRefs...,
	))
	updated.EvidenceRefs = compactStringsV0(append(updated.EvidenceRefs, closure.EvidenceRefs...))
	updated, err = orquestagoal.NewGoalWorkStateV0(updated)
	if err != nil {
		return goalFirstReceiptRepairResultV0{}, err
	}
	if err := store.SaveGoalWorkStateV0(ctx, updated); err != nil {
		return goalFirstReceiptRepairResultV0{}, err
	}
	return goalFirstReceiptRepairResultV0{
		State:    updated,
		Closure:  closure,
		Repaired: true,
	}, nil
}

func goalFirstReceiptRepairResultFromRefsV0(
	state orquestagoal.GoalWorkStateV0,
	refs orquestamcp.MCPDirectorGoalMaterializedRefsV0,
) orquestagoal.GoalWorkResultV0 {
	result := orquestagoal.GoalWorkResultV0{
		SchemaVersion:     orquestagoal.GoalWorkResultSchemaV0,
		Status:            orquestagoal.GoalStatusCompleteV0,
		GoalRef:           state.GoalRef,
		ExternalGoalRef:   state.ExternalGoalRef,
		Summary:           "goal_first_repair_receipt_from_materialized_artifacts",
		ArtifactRefs:      compactStringsV0(refs.ArtifactRefs),
		DomainReceiptRefs: compactStringsV0(refs.DomainReceiptRefs),
		EvidenceRefs: compactStringsV0(append(
			[]string{
				goalFirstRepairReceiptResultRefV0(state.RunRef),
				goalFirstRepairReceiptAttemptedEvidenceRefV0,
			},
			refs.EvidenceRefs...,
		)),
	}
	if state.LastResult != nil {
		result.ArtifactRefs = compactStringsV0(append(state.LastResult.ArtifactRefs, result.ArtifactRefs...))
		result.DomainReceiptRefs = compactStringsV0(append(state.LastResult.DomainReceiptRefs, result.DomainReceiptRefs...))
		result.EvidenceRefs = compactStringsV0(append(state.LastResult.EvidenceRefs, result.EvidenceRefs...))
		result.RequiredTestResults = append([]orquestagoal.GoalRequiredTestResultV0(nil), state.LastResult.RequiredTestResults...)
	}
	return orquestagoal.NormalizeGoalWorkResultV0(result)
}

func goalFirstReceiptRepairClosureV0(
	closure orquestagoal.GoalClosureValidationV0,
) orquestagoal.GoalClosureValidationV0 {
	if closure.Accepted || strings.TrimSpace(closure.Status) == orquestagoal.GoalStatusAcceptedV0 {
		closure.Status = orquestagoal.GoalStatusAcceptedV0
		closure.Accepted = true
		closure.NeedsRework = false
		closure.EvidenceRefs = compactStringsV0(append(closure.EvidenceRefs, goalFirstRepairReceiptAcceptedEvidenceRefV0))
		return closure
	}
	closure.Status = orquestagoal.GoalStatusBlockedV0
	closure.Accepted = false
	closure.NeedsRework = true
	closure.EvidenceRefs = compactStringsV0(append(closure.EvidenceRefs, goalFirstRepairReceiptRequiresReworkEvidenceRefV0))
	if !goalFirstReceiptRepairHasIssueV0(closure.Issues, goalFirstRepairReceiptRequiresReworkIssueV0) {
		closure.Issues = append(closure.Issues, orquestagoal.GoalWorkIssueV0{
			Code:  goalFirstRepairReceiptRequiresReworkIssueV0,
			Field: "goal_first.receipt",
		})
	}
	return closure
}

func goalFirstRepairReceiptResultRefV0(runRef string) string {
	return "result-ref-goal-first-repair-receipt:" + safeGoalMaterializedRefPartV0(runRef)
}

func goalFirstReceiptRepairAlreadyClosedV0(state orquestagoal.GoalWorkStateV0) bool {
	return state.LastClosure != nil &&
		(state.LastClosure.Accepted || strings.TrimSpace(state.LastClosure.Status) == orquestagoal.GoalStatusAcceptedV0)
}

func goalFirstReceiptRepairAlreadyAttemptedV0(state orquestagoal.GoalWorkStateV0) bool {
	if state.LastResult == nil || state.LastClosure == nil {
		return false
	}
	return containsStringV0(state.LastResult.EvidenceRefs, goalFirstRepairReceiptAttemptedEvidenceRefV0)
}

func goalFirstReceiptRepairHasIssueV0(issues []orquestagoal.GoalWorkIssueV0, code string) bool {
	code = strings.TrimSpace(code)
	for _, issue := range issues {
		if strings.TrimSpace(issue.Code) == code {
			return true
		}
	}
	return false
}
