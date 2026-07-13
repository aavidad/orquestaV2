package orquestaappcodexstack

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

const (
	autoprogrammingObserveSuccessorRecoveryEvidenceRefV0  = "evidence-ref-autoprogramming-observe-successor-public-recovery"
	autoprogrammingObserveSuccessorRecoveryReasonCodeV0   = "observe_degraded_successor_recovered"
	autoprogrammingObserveErrorDiagnosticEvidencePrefixV0 = "evidence-ref-autoprogramming-observe-error-sha256-"
)

type CodexStackAutoprogrammingObserveGoalExecutorV0 struct {
	stack *StackV0
}

var _ orquestamcp.MCPTransportObserveAppDirectorGoalTimeoutSnapshotExecutorV0 = CodexStackAutoprogrammingObserveGoalExecutorV0{}

func NewCodexStackAutoprogrammingObserveGoalExecutorV0(
	stack *StackV0,
) CodexStackAutoprogrammingObserveGoalExecutorV0 {
	return CodexStackAutoprogrammingObserveGoalExecutorV0{stack: stack}
}

func (executor CodexStackAutoprogrammingObserveGoalExecutorV0) Execute(
	ctx context.Context,
	input orquestamcp.MCPAutoprogrammingObserveGoalToolInputV0,
) (orquestamcp.MCPAutoprogrammingObserveGoalToolResultV0, error) {
	if executor.stack == nil {
		return orquestamcp.MCPAutoprogrammingObserveGoalToolResultV0{}, fmt.Errorf("stack requerido")
	}
	result, err := executor.stack.ObserveAppDirectorGoalV0(
		ctx,
		orquestamcp.ToAutoprogrammingObserveGoalRequestV0(input),
	)
	if err != nil {
		if recovered, ok := executor.recoverRunningSuccessorAfterObserveErrorV0(ctx, input, err); ok {
			return recovered, nil
		}
		if publicResult, ok := orquestamcp.NewMCPAutoprogrammingObserveGoalErrorResultFromErrorV0(input, err); ok {
			return executor.withPartialSnapshotAfterObserveErrorV0(ctx, input, publicResult), nil
		}
		return orquestamcp.MCPAutoprogrammingObserveGoalToolResultV0{}, err
	}
	publicResult := orquestamcp.NewMCPAutoprogrammingObserveGoalResultV0(input, result)
	return NewCodexStackObserveAppDirectorGoalExecutorV0(executor.stack).withMaterializedRefsV0(
		ctx,
		orquestamcp.MCPObserveAppDirectorGoalToolInputV0{
			RequestID:     input.RequestID,
			CorrelationID: input.CorrelationID,
			RunRef:        input.RunRef,
			OccurredAt:    input.OccurredAt,
			RequestedBy:   input.RequestedBy,
		},
		publicResult,
	), nil
}

// recoverRunningSuccessorAfterObserveErrorV0 is deliberately narrower than a
// generic error fallback. A running rework may already have been persisted
// after the observer request started; returning it avoids a spurious 500, but
// only when state and marker independently agree on that successor.
func (executor CodexStackAutoprogrammingObserveGoalExecutorV0) recoverRunningSuccessorAfterObserveErrorV0(
	ctx context.Context,
	input orquestamcp.MCPAutoprogrammingObserveGoalToolInputV0,
	observeErr error,
) (orquestamcp.MCPAutoprogrammingObserveGoalToolResultV0, bool) {
	if executor.stack == nil {
		return orquestamcp.MCPAutoprogrammingObserveGoalToolResultV0{}, false
	}
	runRef := strings.TrimSpace(input.RunRef)
	store := executor.stack.Ports.GoalStateStore
	if store == nil {
		store = executor.stack.Stores.AppGoalStateStore
	}
	markerStore := executor.stack.Ports.GoalFirstRunMarkerStore
	if runRef == "" || store == nil || markerStore == nil {
		return orquestamcp.MCPAutoprogrammingObserveGoalToolResultV0{}, false
	}
	cas, ok := store.(orquestagoal.GoalWorkStateCASStorePortV0)
	if !ok || observeErr == nil {
		return orquestamcp.MCPAutoprogrammingObserveGoalToolResultV0{}, false
	}
	state, stateErr := store.LoadGoalWorkStateV0(ctx, runRef)
	marker, markerErr := markerStore.LoadGoalWorkRunMarkerV0(ctx, runRef)
	if stateErr != nil || markerErr != nil ||
		!autoprogrammingObserveRunningSuccessorAccreditedV0(state, marker) {
		return orquestamcp.MCPAutoprogrammingObserveGoalToolResultV0{}, false
	}
	diagnosticRef := autoprogrammingObserveErrorDiagnosticEvidenceRefV0(observeErr)
	if !executor.persistObserveErrorDiagnosticV0(ctx, store, markerStore, cas, state, marker, diagnosticRef) {
		return orquestamcp.MCPAutoprogrammingObserveGoalToolResultV0{}, false
	}
	// Build the public result only from a freshly reopened state/marker pair.
	state, stateErr = store.LoadGoalWorkStateV0(ctx, runRef)
	marker, markerErr = markerStore.LoadGoalWorkRunMarkerV0(ctx, runRef)
	if stateErr != nil || markerErr != nil ||
		!autoprogrammingObserveRunningSuccessorAccreditedV0(state, marker) ||
		!autoprogrammingObserveEvidenceRefPresentV0(state.EvidenceRefs, diagnosticRef) {
		return orquestamcp.MCPAutoprogrammingObserveGoalToolResultV0{}, false
	}
	recovered, resultErr := orquestamcp.NewMCPObserveAppDirectorGoalPartialResultFromStateV0(
		orquestamcp.MCPObserveAppDirectorGoalToolInputV0{
			RequestID: input.RequestID, CorrelationID: input.CorrelationID, RunRef: runRef,
			OccurredAt: input.OccurredAt, RequestedBy: input.RequestedBy,
		}, state,
	)
	if resultErr != nil {
		return orquestamcp.MCPAutoprogrammingObserveGoalToolResultV0{}, false
	}
	recovered.Partial = true
	recovered.RecommendedAction = "observe_later"
	recovered.Summary = autoprogrammingObserveSuccessorRecoveryReasonCodeV0
	recovered.CausalVerdict = autoprogrammingObserveSuccessorRecoveryReasonCodeV0
	recovered.CausalReasonCode = autoprogrammingObserveSuccessorRecoveryReasonCodeV0
	recovered.EvidenceRefs = compactCodexStackStringsV0(append(
		recovered.EvidenceRefs,
		marker.EvidenceRefs...,
	))
	recovered.EvidenceRefs = compactCodexStackStringsV0(append(
		recovered.EvidenceRefs,
		autoprogrammingObserveSuccessorRecoveryEvidenceRefV0,
		diagnosticRef,
	))
	return NewCodexStackObserveAppDirectorGoalExecutorV0(executor.stack).withProcessRefsV0(ctx,
		orquestamcp.MCPObserveAppDirectorGoalToolInputV0{
			RequestID: input.RequestID, CorrelationID: input.CorrelationID, RunRef: runRef,
			OccurredAt: input.OccurredAt, RequestedBy: input.RequestedBy,
		}, recovered), true
}

// persistObserveErrorDiagnosticV0 writes only a digest-derived evidence ref.
// A raw observer error may contain operational details and must never be put in
// state or returned through MCP. Recovery is deliberately fail-closed unless a
// CAS write and a fresh read both confirm the durable metadata.
func (executor CodexStackAutoprogrammingObserveGoalExecutorV0) persistObserveErrorDiagnosticV0(
	ctx context.Context,
	store orquestagoal.GoalWorkStateStorePortV0,
	markerStore orquestagoal.GoalWorkRunMarkerStorePortV0,
	cas orquestagoal.GoalWorkStateCASStorePortV0,
	state orquestagoal.GoalWorkStateV0,
	marker orquestagoal.GoalWorkRunMarkerV0,
	diagnosticRef string,
) bool {
	if strings.TrimSpace(diagnosticRef) == "" || !autoprogrammingObserveRunningSuccessorAccreditedV0(state, marker) {
		return false
	}
	// At most one retry is permitted. Only the typed conflict is retried, and
	// never from stale state: both state and marker are causally reopened first.
	for attempt := 0; attempt < 2; attempt++ {
		if autoprogrammingObserveEvidenceRefPresentV0(state.EvidenceRefs, diagnosticRef) {
			return true
		}
		wantState := state
		wantState.EvidenceRefs = compactCodexStackStringsV0(append(wantState.EvidenceRefs, diagnosticRef))
		want, err := orquestagoal.NewGoalWorkStateV0(wantState)
		if err != nil {
			return false
		}
		saved, err := cas.CompareAndSwapGoalWorkStateV0(ctx, state.StoreVersion, want)
		if err == nil {
			confirmed, loadErr := store.LoadGoalWorkStateV0(ctx, state.RunRef)
			return loadErr == nil && confirmed.StoreVersion == saved.StoreVersion &&
				strings.TrimSpace(confirmed.GoalRef) == strings.TrimSpace(state.GoalRef) &&
				autoprogrammingObserveEvidenceRefPresentV0(confirmed.EvidenceRefs, diagnosticRef)
		}
		if attempt == 1 {
			return false
		}
		var conflict orquestagoal.GoalWorkStateCASConflictErrorV0
		if !errors.As(err, &conflict) {
			return false
		}
		state, err = store.LoadGoalWorkStateV0(ctx, state.RunRef)
		if err != nil {
			return false
		}
		marker, err = markerStore.LoadGoalWorkRunMarkerV0(ctx, state.RunRef)
		if err != nil || !autoprogrammingObserveRunningSuccessorAccreditedV0(state, marker) {
			return false
		}
	}
	return false
}

func autoprogrammingObserveErrorDiagnosticEvidenceRefV0(observeErr error) string {
	if observeErr == nil {
		return ""
	}
	digest := sha256.Sum256([]byte(observeErr.Error()))
	return fmt.Sprintf("%s%x", autoprogrammingObserveErrorDiagnosticEvidencePrefixV0, digest)
}

func autoprogrammingObserveEvidenceRefPresentV0(values []string, want string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}

func autoprogrammingObserveRunningSuccessorAccreditedV0(
	state orquestagoal.GoalWorkStateV0,
	marker orquestagoal.GoalWorkRunMarkerV0,
) bool {
	// Validate the persisted spelling before constructors normalize whitespace.
	// A rework ref is a causal identifier, so "01", "+1" and surrounding
	// spaces are different (and invalid), not recoverable presentation forms.
	if !autoprogrammingObserveCanonicalReworkGoalRefV0(state.GoalRef) ||
		!autoprogrammingObserveCanonicalReworkGoalRefV0(state.Spec.GoalRef) ||
		!autoprogrammingObserveCanonicalReworkGoalRefV0(state.LaunchReceipt.GoalRef) ||
		!autoprogrammingObserveCanonicalReworkGoalRefV0(marker.GoalRef) ||
		(marker.Spec != nil && !autoprogrammingObserveCanonicalReworkGoalRefV0(marker.Spec.GoalRef)) ||
		(marker.LaunchReceipt != nil && !autoprogrammingObserveCanonicalReworkGoalRefV0(marker.LaunchReceipt.GoalRef)) {
		return false
	}
	normalizedState, stateErr := orquestagoal.NewGoalWorkStateV0(state)
	normalizedMarker, markerErr := orquestagoal.NewGoalWorkRunMarkerV0(marker)
	if stateErr != nil || markerErr != nil ||
		normalizedMarker.Spec == nil || normalizedMarker.LaunchReceipt == nil ||
		!reflect.DeepEqual(normalizedState.Spec, *normalizedMarker.Spec) ||
		!reflect.DeepEqual(normalizedState.LaunchReceipt, *normalizedMarker.LaunchReceipt) {
		return false
	}
	state = normalizedState
	marker = normalizedMarker
	if state.StoreVersion == 0 ||
		strings.TrimSpace(state.Spec.DirectorKind) != orquestagoal.GoalDirectorKindCodexGoalV0 ||
		strings.TrimSpace(marker.DirectorKind) != orquestagoal.GoalDirectorKindCodexGoalV0 ||
		strings.TrimSpace(state.Status) != orquestagoal.GoalStatusRunningV0 ||
		strings.TrimSpace(marker.Status) != orquestagoal.GoalStatusRunningV0 ||
		strings.TrimSpace(state.LaunchReceipt.Status) != orquestagoal.GoalStatusRunningV0 ||
		strings.TrimSpace(state.RunRef) == "" ||
		strings.TrimSpace(state.RunRef) != strings.TrimSpace(marker.RunRef) ||
		strings.TrimSpace(state.GoalRef) == "" ||
		strings.TrimSpace(state.GoalRef) != strings.TrimSpace(state.Spec.GoalRef) ||
		strings.TrimSpace(state.GoalRef) != strings.TrimSpace(state.LaunchReceipt.GoalRef) ||
		strings.TrimSpace(state.GoalRef) != strings.TrimSpace(marker.GoalRef) ||
		state.LastResult != nil || state.LastClosure != nil {
		return false
	}
	if strings.TrimSpace(state.ExternalGoalRef) == "" ||
		strings.TrimSpace(state.ExternalGoalRef) != strings.TrimSpace(state.LaunchReceipt.ExternalGoalRef) ||
		strings.TrimSpace(state.ExternalGoalRef) != strings.TrimSpace(marker.ExternalGoalRef) {
		return false
	}
	goalRef := state.GoalRef
	markerIndex := strings.LastIndex(goalRef, "-rework-")
	suffix := goalRef[markerIndex+len("-rework-"):]
	index, _ := strconv.Atoi(suffix)
	parentGoalRef := goalRef[:markerIndex]
	if index > 1 {
		parentGoalRef += "-rework-" + strconv.Itoa(index-1)
	}
	return autoprogrammingObserveUniqueRequiredContextRefV0(state.Spec.ContextRefs, "goal", parentGoalRef) &&
		autoprogrammingObserveUniqueRequiredClosureContextRefV0(state.Spec.ContextRefs)
}

func autoprogrammingObserveCanonicalReworkGoalRefV0(goalRef string) bool {
	if goalRef == "" || goalRef != strings.TrimSpace(goalRef) {
		return false
	}
	markerIndex := strings.LastIndex(goalRef, "-rework-")
	if markerIndex <= 0 {
		return false
	}
	suffix := goalRef[markerIndex+len("-rework-"):]
	index, err := strconv.Atoi(suffix)
	return err == nil && index >= 1 && suffix == strconv.Itoa(index)
}

func autoprogrammingObserveUniqueRequiredContextRefV0(
	contextRefs []orquestagoal.GoalContextRefV0,
	kind, ref string,
) bool {
	found := 0
	for _, contextRef := range contextRefs {
		if contextRef.Required && strings.TrimSpace(contextRef.Kind) == kind {
			if contextRef.Kind != kind || contextRef.Ref != ref {
				return false
			}
			found++
		}
	}
	return found == 1
}

func autoprogrammingObserveUniqueRequiredClosureContextRefV0(contextRefs []orquestagoal.GoalContextRefV0) bool {
	found := 0
	for _, contextRef := range contextRefs {
		if contextRef.Required && strings.TrimSpace(contextRef.Kind) == "closure" {
			if contextRef.Kind != "closure" || strings.TrimSpace(contextRef.Ref) == "" || contextRef.Ref != strings.TrimSpace(contextRef.Ref) {
				return false
			}
			found++
		}
	}
	return found == 1
}

func (executor CodexStackAutoprogrammingObserveGoalExecutorV0) ObserveAppDirectorGoalTimeoutSnapshotV0(
	ctx context.Context,
	input orquestamcp.MCPObserveAppDirectorGoalToolInputV0,
) (orquestamcp.MCPObserveAppDirectorGoalToolResultV0, error) {
	return NewCodexStackObserveAppDirectorGoalExecutorV0(executor.stack).
		ObserveAppDirectorGoalTimeoutSnapshotV0(ctx, input)
}

func (executor CodexStackAutoprogrammingObserveGoalExecutorV0) withPartialSnapshotAfterObserveErrorV0(
	ctx context.Context,
	input orquestamcp.MCPAutoprogrammingObserveGoalToolInputV0,
	publicResult orquestamcp.MCPAutoprogrammingObserveGoalToolResultV0,
) orquestamcp.MCPAutoprogrammingObserveGoalToolResultV0 {
	snapshot, err := executor.ObserveAppDirectorGoalTimeoutSnapshotV0(ctx, orquestamcp.MCPObserveAppDirectorGoalToolInputV0{
		RequestID:     input.RequestID,
		CorrelationID: input.CorrelationID,
		RunRef:        input.RunRef,
		OccurredAt:    input.OccurredAt,
		RequestedBy:   input.RequestedBy,
	})
	if err != nil {
		return publicResult
	}
	return orquestamcp.NewMCPObserveAppDirectorGoalTimeoutResultWithPartialV0(publicResult, snapshot)
}
