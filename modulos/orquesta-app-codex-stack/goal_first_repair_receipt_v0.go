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

type goalFirstReceiptRepairResultObserverV0 struct {
	Result orquestagoal.GoalWorkResultV0
}

func (observer goalFirstReceiptRepairResultObserverV0) ObserveGoalWorkV0(
	_ context.Context,
	_ orquestagoal.GoalObservationRequestV0,
) (orquestagoal.GoalWorkResultV0, error) {
	return orquestagoal.NormalizeGoalWorkResultV0(observer.Result), nil
}

type goalFirstReceiptRepairClosureValidatorV0 struct {
	Base orquestagoal.GoalWorkClosureValidatorPortV0
}

func (validator goalFirstReceiptRepairClosureValidatorV0) ValidateGoalWorkClosureV0(
	ctx context.Context,
	spec orquestagoal.GoalWorkSpecV0,
	result orquestagoal.GoalWorkResultV0,
) (orquestagoal.GoalClosureValidationV0, error) {
	base := validator.Base
	if base == nil {
		base = orquestagoal.DefaultGoalWorkClosureValidatorV0{}
	}
	closure, err := base.ValidateGoalWorkClosureV0(ctx, spec, result)
	if err != nil {
		return orquestagoal.GoalClosureValidationV0{}, err
	}
	return goalFirstReceiptRepairClosureV0(closure), nil
}

func repairGoalFirstReceiptFromMaterializedRefsV0(
	ctx context.Context,
	state orquestagoal.GoalWorkStateV0,
	refs orquestamcp.MCPDirectorGoalMaterializedRefsV0,
	ports orquestagoal.GoalWorkLifecyclePortsV0,
) (goalFirstReceiptRepairResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	state, err := orquestagoal.NewGoalWorkStateV0(state)
	if err != nil {
		return goalFirstReceiptRepairResultV0{}, err
	}
	out := goalFirstReceiptRepairResultV0{State: state}
	if ports.StateStore == nil ||
		!goalFirstStringSliceContainsV0(refs.IssueCodes, orquestamcp.MCPGoalFirstMissingTerminalReceiptAfterArtifactsPassV0) ||
		goalFirstReceiptRepairAlreadyClosedV0(state) ||
		goalFirstReceiptRepairAlreadyAttemptedV0(state) {
		return out, nil
	}
	result := goalFirstReceiptRepairResultFromRefsV0(state, refs)
	return runGoalFirstReceiptRepairLifecycleV0(ctx, state, result, ports)
}

func repairGoalFirstReceiptFromMaterializedResultV0(
	ctx context.Context,
	state orquestagoal.GoalWorkStateV0,
	result orquestagoal.GoalWorkResultV0,
	ports orquestagoal.GoalWorkLifecyclePortsV0,
) (goalFirstReceiptRepairResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	state, err := orquestagoal.NewGoalWorkStateV0(state)
	if err != nil {
		return goalFirstReceiptRepairResultV0{}, err
	}
	out := goalFirstReceiptRepairResultV0{State: state}
	if ports.StateStore == nil ||
		goalFirstReceiptRepairAlreadyClosedV0(state) {
		return out, nil
	}
	result = orquestagoal.NormalizeGoalWorkResultV0(result)
	if strings.TrimSpace(result.Status) != orquestagoal.GoalStatusCompleteV0 {
		return out, nil
	}
	result.GoalRef = strings.TrimSpace(state.GoalRef)
	result.ExternalGoalRef = strings.TrimSpace(state.ExternalGoalRef)
	result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, goalFirstRepairReceiptAttemptedEvidenceRefV0))
	return runGoalFirstReceiptRepairLifecycleV0(ctx, state, result, ports)
}

func runGoalFirstReceiptRepairLifecycleV0(
	ctx context.Context,
	state orquestagoal.GoalWorkStateV0,
	result orquestagoal.GoalWorkResultV0,
	ports orquestagoal.GoalWorkLifecyclePortsV0,
) (goalFirstReceiptRepairResultV0, error) {
	out := goalFirstReceiptRepairResultV0{State: state}
	if !goalFirstReceiptRepairLifecycleReadyV0(state.Spec, ports) {
		return out, nil
	}
	ports.Observer = goalFirstReceiptRepairResultObserverV0{Result: result}
	ports.ClosureValidator = goalFirstReceiptRepairClosureValidatorV0{Base: ports.ClosureValidator}
	observed, err := orquestagoal.ObserveGoalWorkV0(
		ctx,
		orquestagoal.GoalWorkObserveRequestV0{RunRef: state.RunRef},
		ports,
	)
	if err != nil {
		return goalFirstReceiptRepairResultV0{}, err
	}
	return goalFirstReceiptRepairResultV0{
		State:    observed.State,
		Closure:  observed.Closure,
		Repaired: true,
	}, nil
}

func goalFirstReceiptRepairLifecycleReadyV0(
	spec orquestagoal.GoalWorkSpecV0,
	ports orquestagoal.GoalWorkLifecyclePortsV0,
) bool {
	if ports.StateStore == nil {
		return false
	}
	if !spec.ClosurePolicy.RequireIndependentRequiredTestAttestation {
		return true
	}
	return ports.RequiredTestSnapshotObserver != nil &&
		ports.RequiredTestAttestor != nil &&
		ports.RequiredTestAttestationStore != nil &&
		ports.RequiredTestIdentityVerifier != nil
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
	return goalFirstStringSliceContainsV0(state.LastResult.EvidenceRefs, goalFirstRepairReceiptAttemptedEvidenceRefV0)
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
