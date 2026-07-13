package orquestagoal

import (
	"context"
	"errors"
	"reflect"
	"strings"
)

type GoalWorkLifecyclePortsV0 struct {
	Launcher                     GoalWorkLauncherPortV0
	Observer                     GoalWorkObservationPortV0
	ClosureValidator             GoalWorkClosureValidatorPortV0
	StateStore                   GoalWorkStateStorePortV0
	RequiredTestSpecBinder       GoalRequiredTestSpecBinderPortV0
	RequiredTestSnapshotObserver GoalRequiredTestFinalSnapshotObserverPortV0
	RequiredTestAttestor         GoalRequiredTestAttestorPortV0
	RequiredTestAttestationStore GoalRequiredTestAttestationStorePortV0
	RequiredTestIdentityVerifier GoalRequiredTestIdentityVerifierPortV0
	RequiredTestClaimPolicy      GoalRequiredTestAttestationClaimPolicyV0
}

type GoalWorkStartRequestV0 struct {
	RunRef       string         `json:"run_ref,omitempty"`
	Spec         GoalWorkSpecV0 `json:"spec"`
	EvidenceRefs []string       `json:"evidence_refs,omitempty"`
}

type GoalWorkStartResultV0 struct {
	State        GoalWorkStateV0     `json:"state"`
	Receipt      GoalLaunchReceiptV0 `json:"receipt"`
	EvidenceRefs []string            `json:"evidence_refs,omitempty"`
}

// GoalWorkReworkSuccessorStartRequestV0 replaces one completed reworkable
// attempt with its immediate successor under the same run.
type GoalWorkReworkSuccessorStartRequestV0 struct {
	ParentState      GoalWorkStateV0 `json:"parent_state"`
	ParentClosureRef string          `json:"parent_closure_ref"`
	SuccessorSpec    GoalWorkSpecV0  `json:"successor_spec"`
	EvidenceRefs     []string        `json:"evidence_refs,omitempty"`
}

type GoalWorkObserveRequestV0 struct {
	RunRef string `json:"run_ref"`
}

type GoalWorkObserveResultV0 struct {
	State            GoalWorkStateV0         `json:"state"`
	Result           GoalWorkResultV0        `json:"result,omitempty"`
	Closure          GoalClosureValidationV0 `json:"closure,omitempty"`
	Terminal         bool                    `json:"terminal,omitempty"`
	ClosureEvaluated bool                    `json:"closure_evaluated,omitempty"`
	Accepted         bool                    `json:"accepted,omitempty"`
	NeedsRework      bool                    `json:"needs_rework,omitempty"`
	EvidenceRefs     []string                `json:"evidence_refs,omitempty"`
}

type GoalWorkStateFromLaunchRequestV0 struct {
	RunRef        string              `json:"run_ref,omitempty"`
	Spec          GoalWorkSpecV0      `json:"spec"`
	LaunchReceipt GoalLaunchReceiptV0 `json:"launch_receipt"`
	EvidenceRefs  []string            `json:"evidence_refs,omitempty"`
}

type GoalWorkLifecycleIssueErrorV0 struct {
	Field  string            `json:"field"`
	Issues []GoalWorkIssueV0 `json:"issues,omitempty"`
}

func (err GoalWorkLifecycleIssueErrorV0) Error() string {
	if strings.TrimSpace(err.Field) == "" {
		return "goal_work_lifecycle_invalid"
	}
	return "goal_work_lifecycle_invalid: " + strings.TrimSpace(err.Field)
}

func StartGoalWorkV0(
	ctx context.Context,
	request GoalWorkStartRequestV0,
	ports GoalWorkLifecyclePortsV0,
) (GoalWorkStartResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if ports.Launcher == nil {
		return GoalWorkStartResultV0{}, GoalWorkLifecycleIssueErrorV0{Field: "ports.goal_launcher"}
	}
	if ports.StateStore == nil {
		return GoalWorkStartResultV0{}, GoalWorkLifecycleIssueErrorV0{Field: "ports.goal_state_store"}
	}
	spec := goalWorkSpecWithRunRefV0(request.Spec, request.RunRef)
	var err error
	if spec.ClosurePolicy.RequireIndependentRequiredTestAttestation {
		if ports.RequiredTestSpecBinder == nil {
			return GoalWorkStartResultV0{}, GoalWorkLifecycleIssueErrorV0{Field: "ports.goal_required_test_spec_binder"}
		}
		spec, err = ports.RequiredTestSpecBinder.BindGoalRequiredTestSpecV0(ctx, spec)
		if err != nil {
			return GoalWorkStartResultV0{}, err
		}
	}
	runRef, err := goalLifecycleRunRefV0(request.RunRef, spec)
	if err != nil {
		return GoalWorkStartResultV0{}, err
	}
	if issues := ValidateGoalWorkSpecV0(spec); len(issues) > 0 {
		return GoalWorkStartResultV0{}, GoalWorkLifecycleIssueErrorV0{Field: "goal_spec"}
	}
	if issues := ValidateGoalRequiredTestAttestationBindingV0(spec); len(issues) > 0 {
		return GoalWorkStartResultV0{}, GoalWorkLifecycleIssueErrorV0{Field: "goal_required_test_attestation_binding", Issues: issues}
	}
	receipt, err := ports.Launcher.LaunchGoalWorkV0(ctx, spec)
	if err != nil {
		result := persistPartialGoalLaunchStateV0(ctx, runRef, spec, receipt, request.EvidenceRefs, ports.StateStore)
		return result, err
	}
	state, err := NewGoalWorkStateFromLaunchV0(GoalWorkStateFromLaunchRequestV0{
		RunRef:        runRef,
		Spec:          spec,
		LaunchReceipt: receipt,
		EvidenceRefs:  request.EvidenceRefs,
	})
	if err != nil {
		return GoalWorkStartResultV0{Receipt: NormalizeGoalLaunchReceiptV0(receipt)}, err
	}
	result := GoalWorkStartResultV0{
		State:        state,
		Receipt:      state.LaunchReceipt,
		EvidenceRefs: append([]string(nil), state.EvidenceRefs...),
	}
	state, err = saveGoalWorkStateV0(ctx, ports.StateStore, state)
	if err != nil {
		return result, err
	}
	result.State = state
	return result, nil
}

// StartGoalWorkReworkSuccessorV0 launches and atomically persists the next
// causal rework attempt for a run. It deliberately requires a CAS store: a
// successor must replace the exact parent version, never a new version-zero
// state.
func StartGoalWorkReworkSuccessorV0(
	ctx context.Context,
	request GoalWorkReworkSuccessorStartRequestV0,
	ports GoalWorkLifecyclePortsV0,
) (GoalWorkStartResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if ports.Launcher == nil {
		return GoalWorkStartResultV0{}, GoalWorkLifecycleIssueErrorV0{Field: "ports.goal_launcher"}
	}
	if ports.StateStore == nil {
		return GoalWorkStartResultV0{}, GoalWorkLifecycleIssueErrorV0{Field: "ports.goal_state_store"}
	}
	cas, ok := ports.StateStore.(GoalWorkStateCASStorePortV0)
	if !ok {
		return GoalWorkStartResultV0{}, GoalWorkLifecycleIssueErrorV0{Field: "ports.goal_state_cas_store"}
	}
	parent, err := NewGoalWorkStateV0(request.ParentState)
	if err != nil {
		return GoalWorkStartResultV0{}, GoalWorkLifecycleIssueErrorV0{Field: "parent_state"}
	}
	spec := goalWorkSpecWithRunRefV0(request.SuccessorSpec, parent.RunRef)
	if spec.ClosurePolicy.RequireIndependentRequiredTestAttestation {
		if ports.RequiredTestSpecBinder == nil {
			return GoalWorkStartResultV0{}, GoalWorkLifecycleIssueErrorV0{Field: "ports.goal_required_test_spec_binder"}
		}
		spec, err = ports.RequiredTestSpecBinder.BindGoalRequiredTestSpecV0(ctx, spec)
		if err != nil {
			return GoalWorkStartResultV0{}, err
		}
	}
	if issues := ValidateGoalWorkReworkSuccessorV0(parent, GoalWorkStateV0{
		RunRef:  parent.RunRef,
		GoalRef: spec.GoalRef,
		Spec:    spec,
	}, request.ParentClosureRef); len(issues) > 0 {
		return GoalWorkStartResultV0{}, GoalWorkLifecycleIssueErrorV0{Field: "rework_successor_spec", Issues: issues}
	}
	if issues := ValidateGoalRequiredTestAttestationBindingV0(spec); len(issues) > 0 {
		return GoalWorkStartResultV0{}, GoalWorkLifecycleIssueErrorV0{Field: "goal_required_test_attestation_binding", Issues: issues}
	}
	receipt, err := ports.Launcher.LaunchGoalWorkV0(ctx, spec)
	if err != nil {
		return GoalWorkStartResultV0{Receipt: NormalizeGoalLaunchReceiptV0(receipt)}, err
	}
	state, err := NewGoalWorkStateFromLaunchV0(GoalWorkStateFromLaunchRequestV0{
		RunRef:        parent.RunRef,
		Spec:          spec,
		LaunchReceipt: receipt,
		EvidenceRefs:  request.EvidenceRefs,
	})
	if err != nil {
		return GoalWorkStartResultV0{Receipt: NormalizeGoalLaunchReceiptV0(receipt)}, err
	}
	state.StoreVersion = parent.StoreVersion
	result := GoalWorkStartResultV0{
		State:        state,
		Receipt:      state.LaunchReceipt,
		EvidenceRefs: append([]string(nil), state.EvidenceRefs...),
	}
	saved, err := cas.CompareAndSwapGoalWorkStateV0(ctx, parent.StoreVersion, state)
	if err != nil {
		return result, err
	}
	result.State = saved
	result.Receipt = saved.LaunchReceipt
	result.EvidenceRefs = append([]string(nil), saved.EvidenceRefs...)
	return result, nil
}

func persistPartialGoalLaunchStateV0(
	ctx context.Context,
	runRef string,
	spec GoalWorkSpecV0,
	receipt GoalLaunchReceiptV0,
	evidenceRefs []string,
	store GoalWorkStateStorePortV0,
) GoalWorkStartResultV0 {
	receipt = NormalizeGoalLaunchReceiptV0(receipt)
	if receipt.GoalRef == "" && receipt.ExternalGoalRef == "" {
		return GoalWorkStartResultV0{}
	}
	retryableGeneration := goalLaunchGenerationConflictRetryableV0(receipt)
	if !retryableGeneration &&
		(receipt.Status == "" || receipt.Status == GoalStatusAcceptedV0 || receipt.Status == GoalStatusRunningV0) {
		receipt.Status = GoalStatusInvalidV0
	}
	if !retryableGeneration {
		receipt.Issues = append(receipt.Issues, GoalWorkIssueV0{
			Code:  "goal_launch_partial_error",
			Field: "goal_launcher",
		})
	}
	state, err := NewGoalWorkStateFromLaunchV0(GoalWorkStateFromLaunchRequestV0{
		RunRef:        runRef,
		Spec:          spec,
		LaunchReceipt: receipt,
		EvidenceRefs:  evidenceRefs,
	})
	if err != nil {
		return GoalWorkStartResultV0{Receipt: receipt}
	}
	result := GoalWorkStartResultV0{
		State:        state,
		Receipt:      state.LaunchReceipt,
		EvidenceRefs: append([]string(nil), state.EvidenceRefs...),
	}
	if store != nil {
		if saved, saveErr := saveGoalWorkStateV0(ctx, store, state); saveErr == nil {
			result.State = saved
		}
	}
	return result
}

func goalLaunchGenerationConflictRetryableV0(receipt GoalLaunchReceiptV0) bool {
	if receipt.Status != GoalStatusRunningV0 ||
		receipt.ExternalGoalRef == "" || receipt.RuntimeGenerationRef == "" {
		return false
	}
	for _, issue := range receipt.Issues {
		if strings.TrimSpace(issue.Code) == "codex_app_server_tmux_generation_conflict_retryable" {
			return true
		}
	}
	return false
}

func ObserveGoalWorkV0(
	ctx context.Context,
	request GoalWorkObserveRequestV0,
	ports GoalWorkLifecyclePortsV0,
) (GoalWorkObserveResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	runRef := strings.TrimSpace(request.RunRef)
	if runRef == "" {
		return GoalWorkObserveResultV0{}, GoalWorkLifecycleIssueErrorV0{Field: "run_ref"}
	}
	if ports.StateStore == nil {
		return GoalWorkObserveResultV0{}, GoalWorkLifecycleIssueErrorV0{Field: "ports.goal_state_store"}
	}
	if ports.Observer == nil {
		return GoalWorkObserveResultV0{}, GoalWorkLifecycleIssueErrorV0{Field: "ports.goal_observer"}
	}
	state, err := ports.StateStore.LoadGoalWorkStateV0(ctx, runRef)
	if err != nil {
		return GoalWorkObserveResultV0{}, err
	}
	state, err = NewGoalWorkStateV0(state)
	if err != nil {
		return GoalWorkObserveResultV0{}, err
	}
	result, err := ports.Observer.ObserveGoalWorkV0(ctx, GoalObservationRequestFromStateV0(state))
	if err != nil {
		result = NormalizeGoalWorkResultV0(result)
		if len(result.Issues) > 0 {
			persistFailedGoalObservationV0(ctx, ports.StateStore, state, result)
			return GoalWorkObserveResultV0{}, GoalWorkLifecycleIssueErrorV0{
				Field:  "goal_result",
				Issues: append([]GoalWorkIssueV0(nil), result.Issues...),
			}
		}
		return GoalWorkObserveResultV0{}, err
	}
	result = NormalizeGoalWorkResultV0(result)
	if issues := ValidateGoalWorkResultV0(result); len(issues) > 0 {
		return GoalWorkObserveResultV0{}, GoalWorkLifecycleIssueErrorV0{Field: "goal_result"}
	}
	result = promoteGoalResultForIndependentTestAttestationV0(state.Spec, result)
	state.ContextBudget = MergeGoalContextBudgetV0(state.ContextBudget, result.ContextBudget)
	result.ContextBudget = state.ContextBudget
	state.LastResult = &result
	state.Status = result.Status
	state.EvidenceRefs = compactGoalStringsV0(append(state.EvidenceRefs, result.EvidenceRefs...))

	terminal := GoalWorkResultTerminalV0(result.Status)
	closure := GoalClosureValidationV0{}
	if terminal {
		if ports.ClosureValidator == nil {
			if !state.Spec.ClosurePolicy.RequireIndependentRequiredTestAttestation {
				return GoalWorkObserveResultV0{}, GoalWorkLifecycleIssueErrorV0{Field: "ports.goal_closure_validator"}
			}
			closure = blockedGoalRequiredTestAttestationClosureV0(
				GoalClosureValidationV0{}, ErrGoalRequiredTestAttestationMissingV0, "ports.goal_closure_validator",
			)
		}
		closureValidator := ports.ClosureValidator
		attestationClaimPending := false
		if state.Spec.ClosurePolicy.RequireIndependentRequiredTestAttestation && len(closure.Issues) == 0 {
			if ports.RequiredTestSnapshotObserver == nil || ports.RequiredTestAttestor == nil || ports.RequiredTestAttestationStore == nil || ports.RequiredTestIdentityVerifier == nil {
				closure = blockedGoalRequiredTestAttestationClosureV0(
					GoalClosureValidationV0{},
					ErrGoalRequiredTestAttestationMissingV0,
					"ports.required_test_attestation",
				)
			} else {
				if result.Status == GoalStatusCompleteV0 {
					snapshot, err := ports.RequiredTestSnapshotObserver.CaptureGoalRequiredTestFinalSnapshotV0(ctx, GoalRequiredTestFinalSnapshotRequestV0{
						RunRef: state.Spec.RunRef, GoalRef: state.Spec.GoalRef,
						WriteSet: append([]GoalWriteScopeV0(nil), state.Spec.WriteSet...), WriteSetSHA256: state.Spec.WriteSetSHA256,
					})
					if err != nil {
						return GoalWorkObserveResultV0{}, err
					}
					snapshot, err = ports.RequiredTestAttestationStore.FreezeGoalRequiredTestFinalSnapshotV0(ctx, snapshot)
					if err != nil {
						return GoalWorkObserveResultV0{}, err
					}
					existing, err := ports.RequiredTestAttestationStore.ListGoalRequiredTestAttestationsV0(ctx, GoalRequiredTestAttestationQueryV0{
						RunRef: state.Spec.RunRef, GoalRef: state.Spec.GoalRef, RevisionRef: snapshot.RevisionRef,
					})
					if err != nil {
						return GoalWorkObserveResultV0{}, err
					}
					missing := MissingGoalRequiredTestsForAttestationV0(state.Spec, existing)
					for _, test := range missing {
						claimResult, err := ports.RequiredTestAttestationStore.AcquireGoalRequiredTestAttestationClaimV0(
							ctx,
							goalRequiredTestAttestationClaimRequestV0(state.Spec, snapshot, test, ports.RequiredTestClaimPolicy),
						)
						if err != nil {
							return GoalWorkObserveResultV0{}, err
						}
						if !claimResult.Acquired {
							if claimResult.Claim.Status == GoalRequiredTestAttestationClaimStatusFailedV0 {
								failureCode := strings.TrimSpace(claimResult.Claim.FailureCode)
								if failureCode == "" {
									failureCode = ErrGoalRequiredTestAttestationFailedV0
								}
								closure = blockedGoalRequiredTestAttestationClosureV0(GoalClosureValidationV0{}, failureCode, "required_test_attestation_claim")
								closure.EvidenceRefs = compactGoalStringsV0(append(closure.EvidenceRefs, claimResult.Claim.ClaimRef))
								break
							}
							if claimResult.Claim.Status == GoalRequiredTestAttestationClaimStatusPendingV0 {
								// Otra instancia esta atestando este mismo snapshot. No
								// materializamos missing/rework: el goal durable vuelve a
								// running y el observer residente lo reconciliara cuando
								// aparezca el receipt o expire la lease.
								attestationClaimPending = true
								result.Status = GoalStatusRunningV0
								result.Issues = append(result.Issues, GoalWorkIssueV0{
									Code:  ErrGoalRequiredTestAttestationClaimedV0,
									Field: "required_test_attestation_claim",
								})
								result.EvidenceRefs = compactGoalStringsV0(append(result.EvidenceRefs, claimResult.Claim.ClaimRef))
								break
							}
							continue
						}
						attestationRequest := GoalRequiredTestAttestationRequestFromSpecV0(state.Spec, snapshot)
						attestationRequest.RequiredTests = []GoalRequiredTestV0{test}
						if _, err := RunAndPersistGoalRequiredTestAttestationsV0(
							ctx,
							attestationRequest,
							claimResult.Claim,
							ports.RequiredTestAttestor,
							ports.RequiredTestAttestationStore,
						); err != nil {
							failedClaim, failErr := ports.RequiredTestAttestationStore.FailGoalRequiredTestAttestationClaimV0(ctx, claimResult.Claim, ErrGoalRequiredTestAttestorInfrastructureFailedV0)
							if failErr != nil {
								return GoalWorkObserveResultV0{}, failErr
							}
							closure = blockedGoalRequiredTestAttestationClosureV0(GoalClosureValidationV0{}, ErrGoalRequiredTestAttestorInfrastructureFailedV0, "required_test_attestation")
							closure.EvidenceRefs = compactGoalStringsV0(append(closure.EvidenceRefs, failedClaim.ClaimRef))
							break
						}
					}
				}
				if !attestationClaimPending {
					closureValidator = EnforceIndependentGoalRequiredTestAttestationV0(
						ports.ClosureValidator,
						ports.RequiredTestAttestationStore,
						ports.RequiredTestIdentityVerifier,
					)
				}
			}
		}
		if !attestationClaimPending && closure.Issues == nil {
			closure, err = closureValidator.ValidateGoalWorkClosureV0(ctx, state.Spec, result)
			if err != nil {
				return GoalWorkObserveResultV0{}, err
			}
		}
		if attestationClaimPending {
			terminal = false
			state.Status = GoalStatusRunningV0
			state.LastResult = &result
			state.LastClosure = nil
			state.EvidenceRefs = compactGoalStringsV0(append(state.EvidenceRefs, result.EvidenceRefs...))
		} else {
			state.LastClosure = &closure
			state.EvidenceRefs = compactGoalStringsV0(append(state.EvidenceRefs, closure.EvidenceRefs...))
		}
	}
	state, err = NewGoalWorkStateV0(state)
	if err != nil {
		return GoalWorkObserveResultV0{}, err
	}
	state, err = saveGoalWorkStateV0(ctx, ports.StateStore, state)
	if err != nil {
		return GoalWorkObserveResultV0{}, err
	}
	if state.LastResult != nil {
		result = *state.LastResult
		terminal = GoalWorkResultTerminalV0(result.Status)
	}
	if state.LastClosure != nil {
		closure = *state.LastClosure
	}
	return GoalWorkObserveResultV0{
		State:            state,
		Result:           result,
		Closure:          closure,
		Terminal:         terminal,
		ClosureEvaluated: terminal,
		Accepted:         closure.Accepted,
		NeedsRework:      closure.NeedsRework,
		EvidenceRefs:     append([]string(nil), state.EvidenceRefs...),
	}, nil
}

func goalRequiredTestAttestationClaimRequestV0(
	spec GoalWorkSpecV0,
	snapshot GoalRequiredTestFinalSnapshotV0,
	test GoalRequiredTestV0,
	policy GoalRequiredTestAttestationClaimPolicyV0,
) GoalRequiredTestAttestationClaimRequestV0 {
	return NormalizeGoalRequiredTestAttestationClaimRequestLeaseV0(GoalRequiredTestAttestationClaimRequestV0{
		RunRef:                  spec.RunRef,
		GoalRef:                 spec.GoalRef,
		RevisionRef:             snapshot.RevisionRef,
		TestRef:                 test.TestRef,
		DefinitionSHA256:        test.DefinitionSHA256,
		OwnerRef:                policy.OwnerRef,
		LeaseDurationSeconds:    policy.LeaseDurationSeconds,
		ReclaimExpired:          policy.ReclaimExpired,
		ReclaimAuthorizationRef: policy.ReclaimAuthorizationRef,
	})
}

func promoteGoalResultForIndependentTestAttestationV0(spec GoalWorkSpecV0, result GoalWorkResultV0) GoalWorkResultV0 {
	if !spec.ClosurePolicy.RequireIndependentRequiredTestAttestation ||
		result.Status != GoalStatusBlockedV0 ||
		len(result.ArtifactPaths) == 0 ||
		len(result.Issues) == 0 {
		return result
	}
	for _, issue := range result.Issues {
		if issue.Code != GoalIssueRequiredTestsEnvironmentUnavailableV0 {
			return result
		}
	}
	requiredTests := make(map[string]bool, len(spec.RequiredTests))
	for _, test := range spec.RequiredTests {
		requiredTests[test.TestRef] = true
	}
	for _, missingRef := range result.Checklist.MissingRefs {
		if !requiredTests[missingRef] {
			return result
		}
	}
	result.Status = GoalStatusCompleteV0
	result.Checklist.MissingRefs = nil
	for index := range result.MaterializedArtifacts {
		if result.MaterializedArtifacts[index].Status == GoalMaterializedArtifactStatusPartialV0 {
			result.MaterializedArtifacts[index].Status = GoalMaterializedArtifactStatusValidV0
		}
	}
	result.EvidenceRefs = compactGoalStringsV0(append(
		result.EvidenceRefs,
		"evidence-ref-goal-required-tests-delegated-to-independent-attestor",
	))
	return result
}

func persistFailedGoalObservationV0(
	ctx context.Context,
	store GoalWorkStateStorePortV0,
	state GoalWorkStateV0,
	result GoalWorkResultV0,
) {
	if store == nil {
		return
	}
	if strings.TrimSpace(result.GoalRef) == "" {
		result.GoalRef = state.GoalRef
	}
	if strings.TrimSpace(result.ExternalGoalRef) == "" {
		result.ExternalGoalRef = state.ExternalGoalRef
	}
	state.LastResult = &result
	state.Status = result.Status
	state.EvidenceRefs = compactGoalStringsV0(append(state.EvidenceRefs, result.EvidenceRefs...))
	normalized, err := NewGoalWorkStateV0(state)
	if err != nil {
		return
	}
	_, _ = saveGoalWorkStateV0(ctx, store, normalized)
}

func saveGoalWorkStateV0(
	ctx context.Context,
	store GoalWorkStateStorePortV0,
	state GoalWorkStateV0,
) (GoalWorkStateV0, error) {
	if cas, ok := store.(GoalWorkStateCASStorePortV0); ok {
		saved, err := cas.CompareAndSwapGoalWorkStateV0(ctx, state.StoreVersion, state)
		if err == nil {
			return saved, nil
		}
		var conflict GoalWorkStateCASConflictErrorV0
		if !errors.As(err, &conflict) {
			return GoalWorkStateV0{}, err
		}
		current, loadErr := store.LoadGoalWorkStateV0(ctx, state.RunRef)
		if loadErr != nil {
			return GoalWorkStateV0{}, loadErr
		}
		current, normalizeErr := NewGoalWorkStateV0(current)
		if normalizeErr != nil {
			return GoalWorkStateV0{}, normalizeErr
		}
		if goalWorkStateTerminalCompatibleV0(current, state) {
			return current, nil
		}
		if current.StoreVersion <= state.StoreVersion || !goalWorkStateEqualExceptStoreVersionV0(current, state) {
			return GoalWorkStateV0{}, err
		}
		state.StoreVersion = current.StoreVersion
		saved, retryErr := cas.CompareAndSwapGoalWorkStateV0(ctx, current.StoreVersion, state)
		if retryErr == nil {
			return saved, nil
		}
		var retryConflict GoalWorkStateCASConflictErrorV0
		if !errors.As(retryErr, &retryConflict) {
			return GoalWorkStateV0{}, retryErr
		}
		current, loadErr = store.LoadGoalWorkStateV0(ctx, state.RunRef)
		if loadErr != nil {
			return GoalWorkStateV0{}, loadErr
		}
		current, normalizeErr = NewGoalWorkStateV0(current)
		if normalizeErr != nil {
			return GoalWorkStateV0{}, normalizeErr
		}
		if goalWorkStateTerminalCompatibleV0(current, state) {
			return current, nil
		}
		return GoalWorkStateV0{}, retryErr
	}
	return state, store.SaveGoalWorkStateV0(ctx, state)
}

func goalWorkStateTerminalCompatibleV0(current, desired GoalWorkStateV0) bool {
	if current.StoreVersion <= desired.StoreVersion ||
		!GoalWorkResultTerminalV0(current.Status) ||
		current.RunRef != desired.RunRef || current.GoalRef != desired.GoalRef ||
		current.ExternalGoalRef != desired.ExternalGoalRef || !reflect.DeepEqual(current.Spec, desired.Spec) ||
		current.LaunchReceipt.RuntimeGenerationRef != desired.LaunchReceipt.RuntimeGenerationRef ||
		current.LastResult == nil || current.LastResult.Status != current.Status ||
		current.LastResult.GoalRef != current.GoalRef || current.LastResult.ExternalGoalRef != current.ExternalGoalRef {
		return false
	}
	if GoalWorkResultTerminalV0(desired.Status) && current.Status != desired.Status {
		return false
	}
	if desired.LastResult != nil && !reflect.DeepEqual(current.LastResult, desired.LastResult) {
		return false
	}
	if desired.LastClosure != nil && !reflect.DeepEqual(current.LastClosure, desired.LastClosure) {
		return false
	}
	if current.LastClosure == nil {
		return true
	}
	closure := current.LastClosure
	return (closure.Accepted && !closure.NeedsRework && closure.Status == GoalStatusAcceptedV0) ||
		(!closure.Accepted && closure.NeedsRework && closure.Status == GoalStatusBlockedV0)
}

// goalWorkStateEqualExceptStoreVersionV0 permits a retry only when the
// concurrent write carried no semantic change. Retrying a merely related state
// would overwrite its result, closure or evidence.
func goalWorkStateEqualExceptStoreVersionV0(current, desired GoalWorkStateV0) bool {
	current.StoreVersion = 0
	desired.StoreVersion = 0
	return reflect.DeepEqual(current, desired)
}

func NewGoalWorkStateFromLaunchV0(
	request GoalWorkStateFromLaunchRequestV0,
) (GoalWorkStateV0, error) {
	spec := goalWorkSpecWithRunRefV0(request.Spec, request.RunRef)
	runRef, err := goalLifecycleRunRefV0(request.RunRef, spec)
	if err != nil {
		return GoalWorkStateV0{}, err
	}
	if issues := ValidateGoalWorkSpecV0(spec); len(issues) > 0 {
		return GoalWorkStateV0{}, GoalWorkLifecycleIssueErrorV0{Field: "goal_spec"}
	}
	receipt := NormalizeGoalLaunchReceiptV0(request.LaunchReceipt)
	if receipt.Status == "" {
		receipt.Status = GoalStatusRunningV0
	}
	if receipt.GoalRef == "" {
		receipt.GoalRef = spec.GoalRef
	}
	if receipt.ExternalGoalRef == "" {
		receipt.ExternalGoalRef = spec.GoalRef
	}
	if issues := ValidateGoalLaunchReceiptV0(receipt); len(issues) > 0 {
		return GoalWorkStateV0{}, GoalWorkLifecycleIssueErrorV0{Field: "launch_receipt"}
	}
	state := GoalWorkStateV0{
		SchemaVersion:   GoalWorkStateSchemaV0,
		RunRef:          runRef,
		GoalRef:         receipt.GoalRef,
		ExternalGoalRef: receipt.ExternalGoalRef,
		Status:          goalWorkStateStatusFromLaunchReceiptV0(receipt),
		Spec:            spec,
		LaunchReceipt:   receipt,
		ContextBudget:   receipt.ContextBudget,
		EvidenceRefs: compactGoalStringsV0(append(
			append(append([]string(nil), request.EvidenceRefs...), spec.EvidenceRefs...),
			receipt.EvidenceRefs...,
		)),
	}
	return NewGoalWorkStateV0(state)
}

func NormalizeGoalLaunchReceiptV0(receipt GoalLaunchReceiptV0) GoalLaunchReceiptV0 {
	receipt.SchemaVersion = strings.TrimSpace(receipt.SchemaVersion)
	if receipt.SchemaVersion == "" {
		receipt.SchemaVersion = GoalWorkLaunchReceiptSchemaV0
	}
	receipt.Status = strings.TrimSpace(receipt.Status)
	receipt.GoalRef = strings.TrimSpace(receipt.GoalRef)
	receipt.ExternalGoalRef = strings.TrimSpace(receipt.ExternalGoalRef)
	receipt.RuntimeGenerationRef = strings.TrimSpace(receipt.RuntimeGenerationRef)
	receipt.ContextBudget = NormalizeGoalContextBudgetV0(receipt.ContextBudget)
	for i := range receipt.EvidenceRefs {
		receipt.EvidenceRefs[i] = strings.TrimSpace(receipt.EvidenceRefs[i])
	}
	for i := range receipt.Issues {
		receipt.Issues[i] = normalizeGoalWorkIssueV0(receipt.Issues[i])
	}
	return receipt
}

func ValidateGoalLaunchReceiptV0(receipt GoalLaunchReceiptV0) []GoalWorkIssueV0 {
	receipt = NormalizeGoalLaunchReceiptV0(receipt)
	var issues []GoalWorkIssueV0
	if receipt.Status != "" && !validGoalLaunchReceiptStatusV0(receipt.Status) {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalStatusInvalidV0, Field: "status"})
	}
	validateGoalRefsV0(&issues, "goal_ref", receipt.GoalRef)
	validateGoalRefsV0(&issues, "external_goal_ref", receipt.ExternalGoalRef)
	validateGoalRefsV0(&issues, "runtime_generation_ref", receipt.RuntimeGenerationRef)
	for _, evidenceRef := range receipt.EvidenceRefs {
		validateRequiredGoalRefV0(&issues, "evidence_refs", evidenceRef)
	}
	return issues
}

func GoalObservationRequestFromStateV0(state GoalWorkStateV0) GoalObservationRequestV0 {
	return NormalizeGoalObservationRequestV0(GoalObservationRequestV0{
		GoalRef:              state.GoalRef,
		ExternalGoalRef:      state.ExternalGoalRef,
		RuntimeGenerationRef: state.LaunchReceipt.RuntimeGenerationRef,
	})
}

func GoalWorkResultTerminalV0(status string) bool {
	switch strings.TrimSpace(status) {
	case GoalStatusCompleteV0, GoalStatusBlockedV0, GoalStatusInvalidV0:
		return true
	default:
		return false
	}
}

func GoalWorkStatePendingObservationV0(state GoalWorkStateV0) bool {
	switch strings.TrimSpace(state.Status) {
	case GoalStatusRunningV0:
		return true
	case GoalStatusCompleteV0:
		if state.LastClosure == nil {
			return true
		}
		closureStatus := strings.TrimSpace(state.LastClosure.Status)
		if state.LastClosure.Accepted || closureStatus == GoalStatusAcceptedV0 {
			return false
		}
		if state.LastClosure.NeedsRework || closureStatus == GoalStatusBlockedV0 {
			return false
		}
		return true
	default:
		return false
	}
}

func NormalizeGoalWorkStateListRequestV0(
	request GoalWorkStateListRequestV0,
) GoalWorkStateListRequestV0 {
	request.RunRefs = compactGoalStringsV0(request.RunRefs)
	request.Statuses = compactGoalStringsV0(request.Statuses)
	if request.ActiveOnly && len(request.Statuses) == 0 {
		request.Statuses = []string{GoalStatusRunningV0, GoalStatusCompleteV0}
	}
	if request.MaxItems < 0 {
		request.MaxItems = 0
	}
	return request
}

func GoalWorkStateMatchesListRequestV0(
	state GoalWorkStateV0,
	request GoalWorkStateListRequestV0,
) bool {
	request = NormalizeGoalWorkStateListRequestV0(request)
	if len(request.RunRefs) > 0 && !goalStringInSetV0(request.RunRefs, state.RunRef) {
		return false
	}
	if len(request.Statuses) > 0 && !goalStringInSetV0(request.Statuses, state.Status) {
		return false
	}
	if request.ActiveOnly && !GoalWorkStatePendingObservationV0(state) {
		return false
	}
	return strings.TrimSpace(state.RunRef) != "" && strings.TrimSpace(state.GoalRef) != ""
}

func NormalizeGoalWorkRunMarkerListRequestV0(
	request GoalWorkRunMarkerListRequestV0,
) GoalWorkRunMarkerListRequestV0 {
	request.RunRefs = compactGoalStringsV0(request.RunRefs)
	request.Statuses = compactGoalStringsV0(request.Statuses)
	if request.ActiveOnly && len(request.Statuses) == 0 {
		request.Statuses = []string{GoalStatusRunningV0}
	}
	if request.MaxItems < 0 {
		request.MaxItems = 0
	}
	return request
}

func GoalWorkRunMarkerMatchesListRequestV0(
	marker GoalWorkRunMarkerV0,
	request GoalWorkRunMarkerListRequestV0,
) bool {
	request = NormalizeGoalWorkRunMarkerListRequestV0(request)
	if len(request.RunRefs) > 0 && !goalStringInSetV0(request.RunRefs, marker.RunRef) {
		return false
	}
	if len(request.Statuses) > 0 && !goalStringInSetV0(request.Statuses, marker.Status) {
		return false
	}
	return strings.TrimSpace(marker.RunRef) != ""
}

func goalWorkSpecWithRunRefV0(spec GoalWorkSpecV0, runRef string) GoalWorkSpecV0 {
	spec = NormalizeGoalWorkSpecV0(spec)
	runRef = strings.TrimSpace(runRef)
	if spec.RunRef == "" {
		spec.RunRef = runRef
	}
	return NormalizeGoalWorkSpecV0(spec)
}

func goalLifecycleRunRefV0(requestRunRef string, spec GoalWorkSpecV0) (string, error) {
	requestRunRef = strings.TrimSpace(requestRunRef)
	specRunRef := strings.TrimSpace(spec.RunRef)
	if requestRunRef != "" && specRunRef != "" && requestRunRef != specRunRef {
		return "", GoalWorkLifecycleIssueErrorV0{Field: "run_ref"}
	}
	runRef := firstGoalLifecycleValueV0(requestRunRef, specRunRef)
	if runRef == "" {
		return "", GoalWorkLifecycleIssueErrorV0{Field: "run_ref"}
	}
	return runRef, nil
}

func goalWorkStateStatusFromLaunchReceiptV0(receipt GoalLaunchReceiptV0) string {
	switch strings.TrimSpace(receipt.Status) {
	case GoalStatusCompleteV0, GoalStatusBlockedV0, GoalStatusInvalidV0:
		return strings.TrimSpace(receipt.Status)
	default:
		return GoalStatusRunningV0
	}
}

func validGoalLaunchReceiptStatusV0(status string) bool {
	switch strings.TrimSpace(status) {
	case GoalStatusAcceptedV0,
		GoalStatusRunningV0,
		GoalStatusCompleteV0,
		GoalStatusBlockedV0,
		GoalStatusInvalidV0:
		return true
	default:
		return false
	}
}

func firstGoalLifecycleValueV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func goalStringInSetV0(values []string, want string) bool {
	want = strings.TrimSpace(want)
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}

func compactGoalStringsV0(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
	return out
}
