package orquestaappdirectorservice

import (
	"context"
	"strconv"
	"strings"
	"time"

	orquestaappdirectorintake "orquesta/modulos/orquesta-app-director-intake"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

const (
	appDirectorGoalClosureReasonAcceptedV0 = "goal_first_accepted"
	appDirectorGoalClosureReasonBlockedV0  = "goal_first_closure_blocked"
)

func startAppDirectorGoalFirstV0(
	ctx context.Context,
	request StartAppDirectorRequestV0,
	spec orquestafactory.AppSpecV0,
	prepared orquestaappdirectorintake.AppDirectorIntakePreparedV0,
	ports StartAppDirectorPortsV0,
) (StartAppDirectorResultV0, bool, error) {
	if !appDirectorGoalPortsReadyV0(ports) {
		return StartAppDirectorResultV0{}, false, nil
	}
	goalSpec := buildStartAppDirectorGoalWorkSpecV0(request, spec, prepared)
	var err error
	goalSpec, err = startAppDirectorGoalSpecWithAutonomousPolicyV0(ctx, request, prepared, ports, goalSpec)
	if err != nil {
		return StartAppDirectorResultV0{}, false, err
	}
	if issues := orquestagoal.ValidateGoalWorkSpecV0(goalSpec); len(issues) > 0 {
		return StartAppDirectorResultV0{}, false, AppDirectorServiceIssueV0{Field: "goal_spec"}
	}
	goalStarted, err := orquestagoal.StartGoalWorkV0(
		ctx,
		orquestagoal.GoalWorkStartRequestV0{
			RunRef:       prepared.Run.RunID,
			Spec:         goalSpec,
			EvidenceRefs: compactStartAppDirectorStringsV0(append([]string{"evidence-ref-app-director-goal-state-v0"}, prepared.EvidenceRefs...)),
		},
		orquestagoal.GoalWorkLifecyclePortsV0{
			Launcher:               ports.GoalLauncher,
			StateStore:             ports.GoalStateStore,
			RequiredTestSpecBinder: ports.GoalRequiredTestSpecBinder,
		},
	)
	if err != nil {
		if markerErr := saveAppDirectorGoalFirstLaunchFailedMarkerV0(
			ctx,
			prepared.Run.RunID,
			goalSpec,
			goalStarted.Receipt,
			prepared.EvidenceRefs,
			ports,
		); markerErr != nil {
			return StartAppDirectorResultV0{}, false, markerErr
		}
		return StartAppDirectorResultV0{}, false, err
	}
	if ports.GoalFirstRunMarkerStore != nil {
		if err := ports.GoalFirstRunMarkerStore.SaveGoalWorkRunMarkerV0(
			ctx,
			appDirectorGoalFirstRunMarkerFromLaunchV0(
				prepared.Run.RunID,
				goalSpec,
				goalStarted.Receipt,
				prepared.EvidenceRefs,
			),
		); err != nil {
			return StartAppDirectorResultV0{}, false, err
		}
	}
	return startAppDirectorGoalFirstResultV0(request, spec, prepared, goalStarted.Receipt), true, nil
}

func saveAppDirectorGoalFirstLaunchFailedMarkerV0(
	ctx context.Context,
	runRef string,
	spec orquestagoal.GoalWorkSpecV0,
	receipt orquestagoal.GoalLaunchReceiptV0,
	evidenceRefs []string,
	ports StartAppDirectorPortsV0,
) error {
	if ports.GoalFirstRunMarkerStore == nil {
		return nil
	}
	return ports.GoalFirstRunMarkerStore.SaveGoalWorkRunMarkerV0(
		ctx,
		AppDirectorGoalFirstRunMarkerV0{
			SchemaVersion: AppDirectorGoalFirstRunMarkerSchemaV0,
			RunRef:        strings.TrimSpace(runRef),
			GoalRef:       firstStartAppDirectorGoalValueV0(receipt.GoalRef, spec.GoalRef),
			ExternalGoalRef: firstStartAppDirectorGoalValueV0(
				receipt.ExternalGoalRef,
				spec.GoalRef,
			),
			DirectorKind:  orquestagoal.GoalDirectorKindCodexGoalV0,
			Status:        orquestagoal.GoalStatusBlockedV0,
			Spec:          &spec,
			LaunchReceipt: startAppDirectorGoalLaunchReceiptIfPresentV0(receipt),
			EvidenceRefs: compactStartAppDirectorStringsV0(append(
				[]string{
					"evidence-ref-app-director-goal-first-launch-failed-v0",
					"evidence-ref-app-director-goal-first-run-marker-v0",
				},
				append(evidenceRefs, receipt.EvidenceRefs...)...,
			)),
		},
	)
}

func startAppDirectorGoalLaunchReceiptIfPresentV0(
	receipt orquestagoal.GoalLaunchReceiptV0,
) *orquestagoal.GoalLaunchReceiptV0 {
	normalized := orquestagoal.NormalizeGoalLaunchReceiptV0(receipt)
	if strings.TrimSpace(normalized.GoalRef) == "" && strings.TrimSpace(normalized.ExternalGoalRef) == "" {
		return nil
	}
	return &normalized
}

func ObserveAppDirectorGoalV0(
	ctx context.Context,
	request ObserveAppDirectorGoalRequestV0,
	ports StartAppDirectorPortsV0,
) (ObserveAppDirectorGoalResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	request.RunRef = strings.TrimSpace(request.RunRef)
	if request.RunRef == "" {
		return ObserveAppDirectorGoalResultV0{}, AppDirectorServiceIssueV0{Field: "run_ref"}
	}
	if ports.GoalStateStore == nil {
		return ObserveAppDirectorGoalResultV0{}, AppDirectorServiceIssueV0{Field: "ports.goal_state_store"}
	}
	if terminal, ok, err := appDirectorForcedTerminalGoalObservationV0(ctx, request, ports); ok || err != nil {
		return terminal, err
	}
	if ports.GoalObserver == nil {
		return ObserveAppDirectorGoalResultV0{}, AppDirectorServiceIssueV0{Field: "ports.goal_observer"}
	}
	run := orquestacoreworkflow.OrchestrationRunV0{}
	goalObserved, err := orquestagoal.ObserveGoalWorkV0(
		ctx,
		orquestagoal.GoalWorkObserveRequestV0{RunRef: request.RunRef},
		orquestagoal.GoalWorkLifecyclePortsV0{
			Observer:                     ports.GoalObserver,
			RequiredTestSnapshotObserver: ports.GoalRequiredTestSnapshotObserver,
			RequiredTestAttestor:         ports.GoalRequiredTestAttestor,
			RequiredTestAttestationStore: ports.GoalRequiredTestAttestationStore,
			RequiredTestIdentityVerifier: ports.GoalRequiredTestIdentityVerifier,
			RequiredTestClaimPolicy:      ports.GoalRequiredTestClaimPolicy,
			ClosureValidator:             appDirectorGoalClosureValidatorV0{Base: ports.GoalClosureValidator},
			StateStore:                   ports.GoalStateStore,
		},
	)
	if err != nil {
		return ObserveAppDirectorGoalResultV0{}, err
	}
	state := goalObserved.State
	result := goalObserved.Result
	closure := goalObserved.Closure
	if goalObserved.Terminal {
		run, err = reflectAppDirectorGoalClosureInRunV0(ctx, request, state, result, closure, ports)
		if err != nil {
			return ObserveAppDirectorGoalResultV0{}, err
		}
		if refreshed, refreshErr := ports.GoalStateStore.LoadGoalWorkStateV0(ctx, request.RunRef); refreshErr == nil {
			if normalized, normalizeErr := NewAppDirectorGoalStateV0(refreshed); normalizeErr == nil {
				state = normalized
			}
		}
		result, closure = appDirectorGoalObservationForCurrentStateV0(state, result, closure)
	}
	return ObserveAppDirectorGoalResultV0{
		SchemaVersion:         ObserveAppDirectorGoalResultSchemaV0,
		Status:                state.Status,
		DirectorExecutionMode: AppDirectorExecutionModeGoalFirstV0,
		RunRef:                state.RunRef,
		GoalRef:               state.GoalRef,
		ExternalGoalRef:       state.ExternalGoalRef,
		Run:                   run,
		GoalResult:            result,
		Closure:               closure,
		EvidenceRefs:          append([]string(nil), state.EvidenceRefs...),
	}, nil
}

func appDirectorForcedTerminalGoalObservationV0(
	ctx context.Context,
	request ObserveAppDirectorGoalRequestV0,
	ports StartAppDirectorPortsV0,
) (ObserveAppDirectorGoalResultV0, bool, error) {
	state, err := ports.GoalStateStore.LoadGoalWorkStateV0(ctx, request.RunRef)
	if err != nil {
		return ObserveAppDirectorGoalResultV0{}, false, nil
	}
	state, err = NewAppDirectorGoalStateV0(state)
	if err != nil || !appDirectorForcedTerminalGoalStateV0(state) {
		return ObserveAppDirectorGoalResultV0{}, false, nil
	}
	result, closure := appDirectorForcedTerminalGoalStateObservationV0(state)
	run, err := reflectAppDirectorGoalClosureInRunV0(ctx, request, state, result, closure, ports)
	if err != nil {
		return ObserveAppDirectorGoalResultV0{}, true, err
	}
	if refreshed, refreshErr := ports.GoalStateStore.LoadGoalWorkStateV0(ctx, request.RunRef); refreshErr == nil {
		if normalized, normalizeErr := NewAppDirectorGoalStateV0(refreshed); normalizeErr == nil {
			state = normalized
		}
	}
	result, closure = appDirectorForcedTerminalGoalStateObservationV0(state)
	return ObserveAppDirectorGoalResultV0{
		SchemaVersion:         ObserveAppDirectorGoalResultSchemaV0,
		Status:                state.Status,
		DirectorExecutionMode: AppDirectorExecutionModeGoalFirstV0,
		RunRef:                state.RunRef,
		GoalRef:               state.GoalRef,
		ExternalGoalRef:       state.ExternalGoalRef,
		Run:                   run,
		GoalResult:            result,
		Closure:               closure,
		EvidenceRefs:          append([]string(nil), state.EvidenceRefs...),
	}, true, nil
}

func appDirectorForcedTerminalGoalStateObservationV0(
	state AppDirectorGoalStateV0,
) (orquestagoal.GoalWorkResultV0, orquestagoal.GoalClosureValidationV0) {
	result := orquestagoal.GoalWorkResultV0{}
	if state.LastResult != nil {
		result = *state.LastResult
	} else {
		result = orquestagoal.GoalWorkResultV0{
			SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
			Status:          orquestagoal.GoalStatusBlockedV0,
			GoalRef:         state.GoalRef,
			ExternalGoalRef: state.ExternalGoalRef,
			Summary:         "operator forced stop kept terminal goal-first state",
			EvidenceRefs:    append([]string(nil), state.EvidenceRefs...),
			Issues: []orquestagoal.GoalWorkIssueV0{{
				Code:  "operator_forced_stop_goal_first",
				Field: "goal_backend",
			}},
		}
	}
	result = orquestagoal.NormalizeGoalWorkResultV0(result)
	closure := orquestagoal.GoalClosureValidationV0{}
	if state.LastClosure != nil {
		closure = *state.LastClosure
	} else {
		closure = orquestagoal.GoalClosureValidationV0{
			Status:       orquestagoal.GoalStatusBlockedV0,
			NeedsRework:  true,
			EvidenceRefs: append([]string(nil), result.EvidenceRefs...),
			Issues:       append([]orquestagoal.GoalWorkIssueV0(nil), result.Issues...),
		}
	}
	closure.Status = strings.TrimSpace(closure.Status)
	if closure.Status == "" {
		closure.Status = orquestagoal.GoalStatusBlockedV0
	}
	return result, closure
}

func appDirectorForcedTerminalGoalStateV0(state AppDirectorGoalStateV0) bool {
	if strings.TrimSpace(state.Status) != orquestagoal.GoalStatusBlockedV0 {
		return false
	}
	return appDirectorForcedTerminalGoalEvidenceV0(state.EvidenceRefs) ||
		state.LastResult != nil && appDirectorForcedTerminalGoalResultV0(*state.LastResult) ||
		state.LastClosure != nil && appDirectorForcedTerminalGoalClosureV0(*state.LastClosure)
}

func appDirectorForcedTerminalGoalResultV0(result orquestagoal.GoalWorkResultV0) bool {
	if appDirectorForcedTerminalGoalEvidenceV0(result.EvidenceRefs) {
		return true
	}
	for _, issue := range result.Issues {
		if appDirectorForcedTerminalGoalCodeV0(issue.Code) {
			return true
		}
	}
	return appDirectorForcedTerminalGoalCodeV0(result.Summary)
}

func appDirectorForcedTerminalGoalClosureV0(closure orquestagoal.GoalClosureValidationV0) bool {
	if appDirectorForcedTerminalGoalEvidenceV0(closure.EvidenceRefs) {
		return true
	}
	for _, issue := range closure.Issues {
		if appDirectorForcedTerminalGoalCodeV0(issue.Code) {
			return true
		}
	}
	return false
}

func appDirectorForcedTerminalGoalEvidenceV0(refs []string) bool {
	for _, ref := range refs {
		if appDirectorForcedTerminalGoalCodeV0(ref) {
			return true
		}
	}
	return false
}

func appDirectorForcedTerminalGoalCodeV0(value string) bool {
	value = strings.TrimSpace(value)
	return value == "evidence-ref-run-control-goal-forced-stop-terminal" ||
		value == "evidence-ref-run-control-goal-forced-cancel-terminal" ||
		value == "evidence-ref-run-control-goal-forced-terminal-reconciled" ||
		value == "evidence-ref-server-shutdown-goal-forced-terminal-reconciled" ||
		strings.Contains(value, "operator_forced_stop") ||
		strings.Contains(value, "operator_forced_cancel")
}

func appDirectorGoalObservationForCurrentStateV0(
	state AppDirectorGoalStateV0,
	result orquestagoal.GoalWorkResultV0,
	closure orquestagoal.GoalClosureValidationV0,
) (orquestagoal.GoalWorkResultV0, orquestagoal.GoalClosureValidationV0) {
	if strings.TrimSpace(state.Status) != orquestagoal.GoalStatusRunningV0 ||
		strings.TrimSpace(state.GoalRef) == "" ||
		strings.TrimSpace(state.GoalRef) == strings.TrimSpace(result.GoalRef) {
		return result, closure
	}
	return orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusRunningV0,
		GoalRef:         strings.TrimSpace(state.GoalRef),
		ExternalGoalRef: strings.TrimSpace(state.ExternalGoalRef),
		EvidenceRefs:    append([]string(nil), state.EvidenceRefs...),
	}, orquestagoal.GoalClosureValidationV0{}
}

func reflectAppDirectorGoalClosureInRunV0(
	ctx context.Context,
	request ObserveAppDirectorGoalRequestV0,
	state AppDirectorGoalStateV0,
	result orquestagoal.GoalWorkResultV0,
	closure orquestagoal.GoalClosureValidationV0,
	ports StartAppDirectorPortsV0,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	if ports.RunStore == nil {
		return orquestacoreworkflow.OrchestrationRunV0{}, AppDirectorServiceIssueV0{Field: "ports.run_store"}
	}
	if ports.EventSink == nil {
		return orquestacoreworkflow.OrchestrationRunV0{}, AppDirectorServiceIssueV0{Field: "ports.event_sink"}
	}
	run, err := ports.RunStore.LoadRunV0(ctx, state.RunRef)
	if err != nil {
		return orquestacoreworkflow.OrchestrationRunV0{}, err
	}
	if run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		return run, nil
	}
	if !closure.Accepted {
		if reworkRun, launched, err := launchAppDirectorGoalReworkIfAllowedV0(ctx, request, run, state, result, closure, ports); launched || err != nil {
			return reworkRun, err
		}
		return blockAppDirectorGoalRunV0(ctx, request, state, result, closure, ports)
	}
	return closeAppDirectorGoalRunV0(ctx, request, run, state, result, closure, ports)
}

func launchAppDirectorGoalReworkIfAllowedV0(
	ctx context.Context,
	request ObserveAppDirectorGoalRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	state AppDirectorGoalStateV0,
	result orquestagoal.GoalWorkResultV0,
	closure orquestagoal.GoalClosureValidationV0,
	ports StartAppDirectorPortsV0,
) (orquestacoreworkflow.OrchestrationRunV0, bool, error) {
	if !appDirectorGoalShouldLaunchReworkV0(state, result, closure, ports) {
		return run, false, nil
	}
	spec, ok := appDirectorGoalReworkSpecV0(state, result, closure)
	if !ok {
		return run, false, nil
	}
	if recovered, ok, err := ReconcileAppDirectorGoalReworkStateFromMarkerV0(ctx, state, ports); err != nil {
		return run, true, err
	} else if ok {
		loaded, loadErr := ports.RunStore.LoadRunV0(ctx, recovered.RunRef)
		return loaded, true, loadErr
	}
	goalStarted, err := orquestagoal.StartGoalWorkReworkSuccessorV0(
		ctx,
		orquestagoal.GoalWorkReworkSuccessorStartRequestV0{
			ParentState:      state,
			ParentClosureRef: appDirectorGoalClosureRefV0(state),
			SuccessorSpec:    spec,
			EvidenceRefs:     appDirectorGoalCommandEvidenceRefsV0(state, result, closure),
		},
		orquestagoal.GoalWorkLifecyclePortsV0{
			Launcher:               ports.GoalReworkLauncher,
			StateStore:             ports.GoalStateStore,
			RequiredTestSpecBinder: ports.GoalRequiredTestSpecBinder,
		},
	)
	if err != nil {
		if ports.GoalFirstRunMarkerStore != nil {
			if markerErr := ports.GoalFirstRunMarkerStore.SaveGoalWorkRunMarkerV0(
				ctx,
				appDirectorGoalFirstRunMarkerFromLaunchV0(
					state.RunRef,
					spec,
					goalStarted.Receipt,
					goalStarted.EvidenceRefs,
				),
			); markerErr != nil {
				return run, true, markerErr
			}
		}
		return run, true, err
	}
	if ports.GoalFirstRunMarkerStore != nil {
		if err := ports.GoalFirstRunMarkerStore.SaveGoalWorkRunMarkerV0(
			ctx,
			appDirectorGoalFirstRunMarkerFromLaunchV0(
				state.RunRef,
				spec,
				goalStarted.Receipt,
				goalStarted.EvidenceRefs,
			),
		); err != nil {
			return run, true, err
		}
	}
	if appDirectorGoalRunHasBlockerV0(run, appDirectorGoalBlockerRefV0(state)) {
		resolved, err := resolveAppDirectorGoalReworkBlockerV0(ctx, request, run, state, result, closure, ports)
		return resolved, true, err
	}
	loaded, err := ports.RunStore.LoadRunV0(ctx, state.RunRef)
	return loaded, true, err
}

// ReconcileAppDirectorGoalReworkStateFromMarkerV0 repairs the narrow crash
// window where a rework launch receipt was persisted but its successor state
// CAS did not complete. It never launches a provider.
func ReconcileAppDirectorGoalReworkStateFromMarkerV0(
	ctx context.Context,
	parent AppDirectorGoalStateV0,
	ports StartAppDirectorPortsV0,
) (AppDirectorGoalStateV0, bool, error) {
	if ports.GoalFirstRunMarkerStore == nil || ports.GoalStateStore == nil {
		return parent, false, nil
	}
	marker, err := ports.GoalFirstRunMarkerStore.LoadGoalWorkRunMarkerV0(ctx, parent.RunRef)
	if err != nil {
		if continueAppDirectorGoalFirstMarkerMissingV0(err) {
			return parent, false, nil
		}
		return AppDirectorGoalStateV0{}, false, err
	}
	marker, err = NewAppDirectorGoalFirstRunMarkerV0(marker)
	if err != nil {
		return AppDirectorGoalStateV0{}, false, err
	}
	if marker.GoalRef == parent.GoalRef || marker.Spec == nil || marker.LaunchReceipt == nil {
		return parent, false, nil
	}
	successor, err := orquestagoal.NewGoalWorkStateFromLaunchV0(orquestagoal.GoalWorkStateFromLaunchRequestV0{
		RunRef: parent.RunRef, Spec: *marker.Spec, LaunchReceipt: *marker.LaunchReceipt,
		EvidenceRefs: compactStartAppDirectorStringsV0(append(
			[]string{"evidence-ref-app-director-goal-rework-state-repaired-from-marker-v0"},
			marker.EvidenceRefs...,
		)),
	})
	if err != nil {
		return AppDirectorGoalStateV0{}, false, err
	}
	if issues := orquestagoal.ValidateGoalWorkReworkSuccessorV0(parent, successor, appDirectorGoalClosureRefV0(parent)); len(issues) > 0 {
		return parent, false, nil
	}
	cas, ok := ports.GoalStateStore.(orquestagoal.GoalWorkStateCASStorePortV0)
	if !ok {
		return AppDirectorGoalStateV0{}, false, AppDirectorServiceIssueV0{Field: "ports.goal_state_cas_store"}
	}
	successor.StoreVersion = parent.StoreVersion
	saved, err := cas.CompareAndSwapGoalWorkStateV0(ctx, parent.StoreVersion, successor)
	if err != nil {
		return AppDirectorGoalStateV0{}, false, err
	}
	return saved, true, nil
}

func appDirectorGoalShouldLaunchReworkV0(
	state AppDirectorGoalStateV0,
	result orquestagoal.GoalWorkResultV0,
	closure orquestagoal.GoalClosureValidationV0,
	ports StartAppDirectorPortsV0,
) bool {
	if !closure.NeedsRework ||
		!state.Spec.ReworkPolicy.PreferNewGoal ||
		state.Spec.ClosurePolicy.RequireDomainReceipt ||
		ports.GoalReworkLauncher == nil ||
		ports.GoalStateStore == nil {
		return false
	}
	if !appDirectorGoalResultPermiteReworkV0(state.Spec, result, closure) {
		return false
	}
	maxRework := appDirectorGoalMaxReworkGoalsV0(state.Spec)
	if maxRework <= 0 {
		return false
	}
	return appDirectorGoalReworkIndexV0(state.Spec.GoalRef) < maxRework
}

func appDirectorGoalResultPermiteReworkV0(
	spec orquestagoal.GoalWorkSpecV0,
	result orquestagoal.GoalWorkResultV0,
	closure orquestagoal.GoalClosureValidationV0,
) bool {
	switch strings.TrimSpace(result.Status) {
	case orquestagoal.GoalStatusCompleteV0:
		return true
	case orquestagoal.GoalStatusBlockedV0, orquestagoal.GoalStatusInvalidV0:
		return appDirectorGoalOperationalReworkIssueV0(result, closure) ||
			appDirectorGoalNewAppMissingRefsReworkIssueV0(spec, result, closure)
	default:
		return false
	}
}

func appDirectorGoalOperationalReworkIssueV0(
	result orquestagoal.GoalWorkResultV0,
	closure orquestagoal.GoalClosureValidationV0,
) bool {
	for _, code := range appDirectorGoalResultAndClosureIssueCodesV0(result, closure) {
		switch strings.TrimSpace(code) {
		case "codex_app_server_goal_active_timeout",
			"goal_active_no_checkpoint_high_consumption",
			"checkpoint_only_high_consumption",
			"goal_active_timeout_backend_active":
			return true
		}
	}
	summary := strings.TrimSpace(result.Summary)
	return strings.Contains(summary, "codex_app_server_goal_active_timeout") ||
		strings.Contains(summary, "checkpoint_only_high_consumption") ||
		strings.Contains(summary, "goal_active_no_checkpoint_high_consumption")
}

func appDirectorGoalNewAppMissingRefsReworkIssueV0(
	spec orquestagoal.GoalWorkSpecV0,
	result orquestagoal.GoalWorkResultV0,
	closure orquestagoal.GoalClosureValidationV0,
) bool {
	if strings.TrimSpace(spec.WorkKind) != "new_app" {
		return false
	}
	if !appDirectorGoalIssueFieldInSetV0(closure.Issues, "checklist.missing_refs") &&
		!appDirectorGoalIssueFieldInSetV0(result.Issues, "checklist.missing_refs") &&
		len(compactStartAppDirectorStringsV0(result.Checklist.MissingRefs)) == 0 {
		return false
	}
	for _, ref := range result.Checklist.MissingRefs {
		switch strings.TrimSpace(ref) {
		case "source_tree", "handoff_report", "technical_stack_manifest", "go_app", "tests":
			return true
		}
	}
	return false
}

func appDirectorGoalIssueFieldInSetV0(issues []orquestagoal.GoalWorkIssueV0, field string) bool {
	field = strings.TrimSpace(field)
	if field == "" {
		return false
	}
	for _, issue := range issues {
		if strings.TrimSpace(issue.Field) == field {
			return true
		}
	}
	return false
}

func appDirectorGoalResultAndClosureIssueCodesV0(
	result orquestagoal.GoalWorkResultV0,
	closure orquestagoal.GoalClosureValidationV0,
) []string {
	codes := make([]string, 0, len(result.Issues)+len(closure.Issues))
	for _, issue := range result.Issues {
		codes = append(codes, issue.Code)
	}
	for _, issue := range closure.Issues {
		codes = append(codes, issue.Code)
	}
	return compactStartAppDirectorStringsV0(codes)
}

func appDirectorGoalMaxReworkGoalsV0(spec orquestagoal.GoalWorkSpecV0) int {
	if spec.ReworkPolicy.MaxReworkGoals > 0 {
		return spec.ReworkPolicy.MaxReworkGoals
	}
	return spec.Budget.MaxReworkGoals
}

func appDirectorGoalReworkSpecV0(
	state AppDirectorGoalStateV0,
	result orquestagoal.GoalWorkResultV0,
	closure orquestagoal.GoalClosureValidationV0,
) (orquestagoal.GoalWorkSpecV0, bool) {
	nextIndex := appDirectorGoalReworkIndexV0(state.Spec.GoalRef) + 1
	if nextIndex <= 0 {
		return orquestagoal.GoalWorkSpecV0{}, false
	}
	spec := state.Spec
	spec.GoalRef = appDirectorGoalReworkGoalRefV0(state.Spec.GoalRef, nextIndex)
	spec.ContextRefs = append(append([]orquestagoal.GoalContextRefV0(nil), spec.ContextRefs...),
		orquestagoal.GoalContextRefV0{
			Kind:     "goal",
			Ref:      state.GoalRef,
			Purpose:  "Goal anterior que necesita rework causal.",
			Required: true,
		},
		orquestagoal.GoalContextRefV0{
			Kind:     "closure",
			Ref:      appDirectorGoalClosureRefV0(state),
			Purpose:  "Cierre no aceptado que origina este rework.",
			Required: true,
		},
	)
	spec.EvidenceRefs = compactStartAppDirectorStringsV0(append(append(append(append(
		[]string{"evidence-ref-app-director-goal-rework-v0"},
		spec.EvidenceRefs...,
	), state.EvidenceRefs...), result.EvidenceRefs...), closure.EvidenceRefs...))
	spec.AcceptanceCriteria = compactStartAppDirectorStringsV0(append(
		append([]string(nil), spec.AcceptanceCriteria...),
		"Rework causal: resolver el cierre no aceptado del goal anterior "+state.GoalRef+"; issues: "+appDirectorGoalClosureIssueSummaryV0(closure)+".",
	))
	if spec.ReworkPolicy.PreserveArtifacts {
		spec.AcceptanceCriteria = compactStartAppDirectorStringsV0(append(
			spec.AcceptanceCriteria,
			"Conservar y reutilizar artefactos aprovechables del goal anterior; rehacer solo lo que incumpla el contrato.",
		))
	}
	spec = orquestagoal.NormalizeGoalWorkSpecV0(spec)
	if issues := orquestagoal.ValidateGoalWorkSpecV0(spec); len(issues) > 0 {
		return orquestagoal.GoalWorkSpecV0{}, false
	}
	return spec, true
}

func appDirectorGoalReworkGoalRefV0(goalRef string, index int) string {
	base := appDirectorGoalReworkBaseGoalRefV0(goalRef)
	if base == "" {
		base = strings.TrimSpace(goalRef)
	}
	return base + "-rework-" + strconv.Itoa(index)
}

func appDirectorGoalReworkBaseGoalRefV0(goalRef string) string {
	goalRef = strings.TrimSpace(goalRef)
	if marker := strings.LastIndex(goalRef, "-rework-"); marker > 0 {
		if _, err := strconv.Atoi(goalRef[marker+len("-rework-"):]); err == nil {
			return goalRef[:marker]
		}
	}
	return goalRef
}

func appDirectorGoalReworkIndexV0(goalRef string) int {
	goalRef = strings.TrimSpace(goalRef)
	marker := strings.LastIndex(goalRef, "-rework-")
	if marker <= 0 {
		return 0
	}
	value, err := strconv.Atoi(goalRef[marker+len("-rework-"):])
	if err != nil || value < 0 {
		return 0
	}
	return value
}

func appDirectorGoalClosureIssueSummaryV0(closure orquestagoal.GoalClosureValidationV0) string {
	items := make([]string, 0, len(closure.Issues))
	for _, issue := range closure.Issues {
		item := strings.TrimSpace(issue.Field)
		if code := strings.TrimSpace(issue.Code); code != "" {
			if item != "" {
				item += "="
			}
			item += code
		}
		if item != "" {
			items = append(items, item)
		}
	}
	if len(items) == 0 {
		return "sin_issue_detallado"
	}
	if len(items) > 8 {
		items = items[:8]
	}
	return strings.Join(items, ", ")
}

func resolveAppDirectorGoalReworkBlockerV0(
	ctx context.Context,
	request ObserveAppDirectorGoalRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	state AppDirectorGoalStateV0,
	result orquestagoal.GoalWorkResultV0,
	closure orquestagoal.GoalClosureValidationV0,
	ports StartAppDirectorPortsV0,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	blockerRef := appDirectorGoalBlockerRefV0(state)
	if !appDirectorGoalRunHasBlockerV0(run, blockerRef) {
		return run, nil
	}
	return applyAppDirectorGoalWorkflowCommandV0(ctx, request, state.RunRef, "resolve-goal-rework-blocker", blockerRef, ports, func(meta orquestacoreworkflow.OrchestrationCommandMetaV0) (orquestacoreworkflow.OrchestrationCommandV0, error) {
		return orquestacoreworkflow.NewResolveRunBlockerCommandV0(meta, orquestacoreworkflow.ResolveRunBlockerCommandPayloadV0{
			BlockerID:    blockerRef,
			ReasonCode:   "goal_first_rework_started",
			Summary:      "Goal-first reanudado mediante nuevo goal causal de rework.",
			EvidenceRefs: appDirectorGoalCommandEvidenceRefsV0(state, result, closure),
		})
	})
}

func blockAppDirectorGoalRunV0(
	ctx context.Context,
	request ObserveAppDirectorGoalRequestV0,
	state AppDirectorGoalStateV0,
	result orquestagoal.GoalWorkResultV0,
	closure orquestagoal.GoalClosureValidationV0,
	ports StartAppDirectorPortsV0,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	blockerRef := appDirectorGoalBlockerRefV0(state)
	run, err := ports.RunStore.LoadRunV0(ctx, state.RunRef)
	if err != nil {
		return orquestacoreworkflow.OrchestrationRunV0{}, err
	}
	if appDirectorGoalRunHasBlockerV0(run, blockerRef) {
		return run, nil
	}
	command, err := orquestacoreworkflow.NewBlockRunCommandV0(
		appDirectorGoalCommandMetaV0(request, state.RunRef, "block-goal", blockerRef),
		orquestacoreworkflow.BlockRunCommandPayloadV0{
			BlockerID:    blockerRef,
			ReasonCode:   appDirectorGoalClosureReasonBlockedV0,
			Summary:      "Goal-first bloqueado por cierre no aceptado.",
			SourceGroup:  "app-director-goal-first",
			EvidenceRefs: appDirectorGoalCommandEvidenceRefsV0(state, result, closure),
		},
	)
	if err != nil {
		return orquestacoreworkflow.OrchestrationRunV0{}, err
	}
	if _, err := orquestacionnucleoapp.HandleStoredWorkflowCommandV0(ctx, ports.RunStore, ports.EventSink, command); err != nil {
		return orquestacoreworkflow.OrchestrationRunV0{}, err
	}
	return ports.RunStore.LoadRunV0(ctx, state.RunRef)
}

func closeAppDirectorGoalRunV0(
	ctx context.Context,
	request ObserveAppDirectorGoalRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	state AppDirectorGoalStateV0,
	result orquestagoal.GoalWorkResultV0,
	closure orquestagoal.GoalClosureValidationV0,
	ports StartAppDirectorPortsV0,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	run, err := resolveAppDirectorGoalBlockerIfPresentV0(ctx, request, run, state, result, closure, ports)
	if err != nil {
		return run, err
	}
	if run.Status != orquestacoreworkflow.OrchestrationRunStatusActiveV0 {
		return run, nil
	}
	validationRef := appDirectorGoalValidationRefV0(state)
	closureRef := appDirectorGoalClosureRefV0(state)
	if run, err = openAppDirectorGoalFinalValidationIfNeededV0(ctx, request, run, state, validationRef, ports); err != nil {
		return run, err
	}
	if run, err = registerAppDirectorGoalFinalValidationIfNeededV0(ctx, request, run, state, result, closure, validationRef, ports); err != nil {
		return run, err
	}
	if run, err = openAppDirectorGoalClosureIfNeededV0(ctx, request, run, state, closureRef, ports); err != nil {
		return run, err
	}
	return closeAppDirectorGoalRunIfNeededV0(ctx, request, run, state, result, closure, validationRef, closureRef, ports)
}

func resolveAppDirectorGoalBlockerIfPresentV0(
	ctx context.Context,
	request ObserveAppDirectorGoalRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	state AppDirectorGoalStateV0,
	result orquestagoal.GoalWorkResultV0,
	closure orquestagoal.GoalClosureValidationV0,
	ports StartAppDirectorPortsV0,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	blockerRef := appDirectorGoalBlockerRefV0(state)
	if !appDirectorGoalRunHasBlockerV0(run, blockerRef) {
		return run, nil
	}
	return applyAppDirectorGoalWorkflowCommandV0(ctx, request, state.RunRef, "resolve-goal-blocker", blockerRef, ports, func(meta orquestacoreworkflow.OrchestrationCommandMetaV0) (orquestacoreworkflow.OrchestrationCommandV0, error) {
		return orquestacoreworkflow.NewResolveRunBlockerCommandV0(meta, orquestacoreworkflow.ResolveRunBlockerCommandPayloadV0{
			BlockerID:    blockerRef,
			ReasonCode:   appDirectorGoalClosureReasonAcceptedV0,
			Summary:      "Goal-first aceptado tras observacion de cierre.",
			EvidenceRefs: appDirectorGoalCommandEvidenceRefsV0(state, result, closure),
		})
	})
}

func openAppDirectorGoalFinalValidationIfNeededV0(
	ctx context.Context,
	request ObserveAppDirectorGoalRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	state AppDirectorGoalStateV0,
	validationRef string,
	ports StartAppDirectorPortsV0,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	if run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseValidacionFinalV0 &&
		run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseCierreV0 {
		return applyAppDirectorGoalWorkflowCommandV0(ctx, request, state.RunRef, "open-final-validation", validationRef, ports, func(meta orquestacoreworkflow.OrchestrationCommandMetaV0) (orquestacoreworkflow.OrchestrationCommandV0, error) {
			return orquestacoreworkflow.NewOpenPhaseCommandV0(meta, orquestacoreworkflow.OpenPhaseCommandPayloadV0{
				PhaseID: string(orquestacoreworkflow.OrchestrationPhaseValidacionFinalV0),
				Reason:  "Validacion final run-level de goal-first aceptado.",
			})
		})
	}
	return run, nil
}

func registerAppDirectorGoalFinalValidationIfNeededV0(
	ctx context.Context,
	request ObserveAppDirectorGoalRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	state AppDirectorGoalStateV0,
	result orquestagoal.GoalWorkResultV0,
	closure orquestagoal.GoalClosureValidationV0,
	validationRef string,
	ports StartAppDirectorPortsV0,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	if !appDirectorGoalStringInSetV0(run.Validations, validationRef) {
		return applyAppDirectorGoalWorkflowCommandV0(ctx, request, state.RunRef, "register-final-validation", validationRef, ports, func(meta orquestacoreworkflow.OrchestrationCommandMetaV0) (orquestacoreworkflow.OrchestrationCommandV0, error) {
			return orquestacoreworkflow.NewRegisterFinalValidationCommandV0(meta, orquestacoreworkflow.RegisterFinalValidationCommandPayloadV0{
				ValidationRef: validationRef,
				PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseValidacionFinalV0),
				Summary:       "Validacion final run-level de goal-first aceptado.",
				EvidenceRefs:  appDirectorGoalCommandEvidenceRefsV0(state, result, closure),
			})
		})
	}
	return run, nil
}

func openAppDirectorGoalClosureIfNeededV0(
	ctx context.Context,
	request ObserveAppDirectorGoalRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	state AppDirectorGoalStateV0,
	closureRef string,
	ports StartAppDirectorPortsV0,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	if run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseCierreV0 {
		return applyAppDirectorGoalWorkflowCommandV0(ctx, request, state.RunRef, "open-closure", closureRef, ports, func(meta orquestacoreworkflow.OrchestrationCommandMetaV0) (orquestacoreworkflow.OrchestrationCommandV0, error) {
			return orquestacoreworkflow.NewOpenPhaseCommandV0(meta, orquestacoreworkflow.OpenPhaseCommandPayloadV0{
				PhaseID: string(orquestacoreworkflow.OrchestrationPhaseCierreV0),
				Reason:  "Cierre de run goal-first aceptada.",
			})
		})
	}
	return run, nil
}

func closeAppDirectorGoalRunIfNeededV0(
	ctx context.Context,
	request ObserveAppDirectorGoalRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	state AppDirectorGoalStateV0,
	result orquestagoal.GoalWorkResultV0,
	closure orquestagoal.GoalClosureValidationV0,
	validationRef string,
	closureRef string,
	ports StartAppDirectorPortsV0,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	if !appDirectorGoalStringInSetV0(run.Closures, closureRef) {
		return applyAppDirectorGoalWorkflowCommandV0(ctx, request, state.RunRef, "close-run", closureRef, ports, func(meta orquestacoreworkflow.OrchestrationCommandMetaV0) (orquestacoreworkflow.OrchestrationCommandV0, error) {
			return orquestacoreworkflow.NewCloseRunCommandV0(meta, orquestacoreworkflow.CloseRunCommandPayloadV0{
				ClosureRef:    closureRef,
				PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseCierreV0),
				ValidationRef: validationRef,
				Summary:       "Run goal-first cerrada con closure aceptada.",
				EvidenceRefs:  appDirectorGoalCommandEvidenceRefsV0(state, result, closure),
			})
		})
	}
	return run, nil
}

type appDirectorGoalWorkflowCommandBuilderV0 func(orquestacoreworkflow.OrchestrationCommandMetaV0) (orquestacoreworkflow.OrchestrationCommandV0, error)

func applyAppDirectorGoalWorkflowCommandV0(
	ctx context.Context,
	request ObserveAppDirectorGoalRequestV0,
	runRef string,
	action string,
	ref string,
	ports StartAppDirectorPortsV0,
	build appDirectorGoalWorkflowCommandBuilderV0,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	command, err := build(appDirectorGoalCommandMetaV0(request, runRef, action, ref))
	if err != nil {
		return orquestacoreworkflow.OrchestrationRunV0{}, err
	}
	if _, err := orquestacionnucleoapp.HandleStoredWorkflowCommandV0(ctx, ports.RunStore, ports.EventSink, command); err != nil {
		return orquestacoreworkflow.OrchestrationRunV0{}, err
	}
	return ports.RunStore.LoadRunV0(ctx, runRef)
}

func appDirectorGoalCommandMetaV0(
	request ObserveAppDirectorGoalRequestV0,
	runRef string,
	action string,
	ref string,
) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	token := appDirectorGoalSafeTokenV0(runRef, ref, action)
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      "cmd-app-director-goal-" + strings.TrimSpace(action) + "-" + token,
		RunID:          strings.TrimSpace(runRef),
		IdempotencyKey: "idem-app-director-goal-" + strings.TrimSpace(action) + "-" + token,
		CorrelationID:  strings.TrimSpace(request.CorrelationID),
		RequestedBy:    firstStartAppDirectorGoalValueV0(request.RequestedBy, "orquesta-app-director-service"),
		OccurredAt:     appDirectorGoalOccurredAtV0(request.OccurredAt),
	}
}

func appDirectorGoalOccurredAtV0(value string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return time.Now().UTC().Format(time.RFC3339)
}

func appDirectorGoalBlockerRefV0(state AppDirectorGoalStateV0) string {
	return "blocker-ref-app-director-goal-" + appDirectorGoalSafeTokenV0(state.RunRef, state.GoalRef)
}

func appDirectorGoalValidationRefV0(state AppDirectorGoalStateV0) string {
	return "validation-ref-app-director-goal-" + appDirectorGoalSafeTokenV0(state.RunRef, state.GoalRef)
}

func appDirectorGoalClosureRefV0(state AppDirectorGoalStateV0) string {
	return "closure-ref-app-director-goal-" + appDirectorGoalSafeTokenV0(state.RunRef, state.GoalRef)
}

func appDirectorGoalCommandEvidenceRefsV0(
	state AppDirectorGoalStateV0,
	result orquestagoal.GoalWorkResultV0,
	closure orquestagoal.GoalClosureValidationV0,
) []string {
	refs := compactStartAppDirectorStringsV0(append(append(append(
		[]string{"evidence-ref-app-director-goal-observed-v0"},
		state.EvidenceRefs...,
	), result.EvidenceRefs...), closure.EvidenceRefs...))
	if len(refs) > 20 {
		return append([]string(nil), refs[:20]...)
	}
	return refs
}

func appDirectorGoalRunHasBlockerV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	blockerRef string,
) bool {
	return appDirectorGoalStringInSetV0(run.Blockers, blockerRef)
}

func appDirectorGoalStringInSetV0(values []string, want string) bool {
	want = strings.TrimSpace(want)
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}

func NewAppDirectorGoalStateV0(state AppDirectorGoalStateV0) (AppDirectorGoalStateV0, error) {
	return orquestagoal.NewGoalWorkStateV0(state)
}

func startAppDirectorGoalSpecWithAutonomousPolicyV0(
	ctx context.Context,
	request StartAppDirectorRequestV0,
	prepared orquestaappdirectorintake.AppDirectorIntakePreparedV0,
	ports StartAppDirectorPortsV0,
	spec orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalWorkSpecV0, error) {
	if ports.AutonomousDirectorPolicy == nil {
		return spec, nil
	}
	decision, err := ports.AutonomousDirectorPolicy.DecideAutonomousDirectorV0(
		ctx,
		orquestacionnucleoapp.AutonomousDirectorDecisionInputV0{
			Run:    prepared.Run,
			Stats:  orquestacionnucleoapp.BuildDirectorRunStatsV0(prepared.Run),
			Limits: startAppDirectorAutonomousPolicyLimitsV0(request, spec),
			Requests: orquestacionnucleoapp.AutonomousDirectorRequestHintsV0{
				CorrelationID: strings.TrimSpace(request.CorrelationID),
				EvidenceRefs:  []string{"evidence-ref-app-director-goal-autonomous-policy-v0"},
			},
		},
	)
	if err != nil {
		return orquestagoal.GoalWorkSpecV0{}, err
	}
	return startAppDirectorGoalSpecWithAutonomousDecisionV0(spec, decision), nil
}

func startAppDirectorAutonomousPolicyLimitsV0(
	request StartAppDirectorRequestV0,
	spec orquestagoal.GoalWorkSpecV0,
) orquestacionnucleoapp.AutonomousDirectorLimitsV0 {
	return orquestacionnucleoapp.AutonomousDirectorLimitsV0{
		MaxTeamSize:          spec.Budget.MaxSubgoals,
		MaxParallelAgents:    request.MaxDispatchesPerWait,
		MaxBursts:            request.MaxBursts,
		MaxStepsPerBurst:     request.MaxStepsPerBurst,
		MaxDispatchesPerWait: request.MaxDispatchesPerWait,
		MaxCommandsPerCycle:  request.MaxCommands,
		MaxOutboxPerCycle:    request.MaxOutboxPerCycle,
	}
}

func startAppDirectorGoalSpecWithAutonomousDecisionV0(
	spec orquestagoal.GoalWorkSpecV0,
	decision orquestacionnucleoapp.AutonomousDirectorDecisionV0,
) orquestagoal.GoalWorkSpecV0 {
	if decision.TeamSize > 0 {
		spec.Budget.MaxSubgoals = decision.TeamSize
	}
	if decision.MaxParallelAgents > spec.Budget.MaxSubgoals {
		spec.Budget.MaxSubgoals = decision.MaxParallelAgents
	}
	purpose := strings.TrimSpace(decision.Summary)
	if purpose == "" {
		purpose = "Decision de politica autonoma para dimensionar goal-first."
	}
	spec.ContextRefs = append(spec.ContextRefs, orquestagoal.GoalContextRefV0{
		Kind:     "autonomous_director_policy",
		Ref:      "autonomous_director_policy:v0",
		Purpose:  purpose,
		Required: false,
	})
	spec.EvidenceRefs = compactStartAppDirectorStringsV0(append(spec.EvidenceRefs, decision.EvidenceRefs...))
	spec.AcceptanceCriteria = compactStartAppDirectorStringsV0(append(
		spec.AcceptanceCriteria,
		startAppDirectorAutonomousQualityCriteriaV0(decision.QualityRecommendations)...,
	))
	return orquestagoal.NormalizeGoalWorkSpecV0(spec)
}

func startAppDirectorAutonomousQualityCriteriaV0(
	recommendations []orquestacionnucleoapp.AutonomousQualityRecommendationV0,
) []string {
	criteria := make([]string, 0, len(recommendations))
	for _, recommendation := range recommendations {
		action := strings.TrimSpace(recommendation.RecommendedAction)
		subject := strings.TrimSpace(recommendation.SubjectRef)
		summary := strings.TrimSpace(recommendation.Summary)
		if action == "" || subject == "" {
			continue
		}
		if summary == "" {
			summary = strings.TrimSpace(recommendation.ReasonCode)
		}
		criteria = append(criteria, "Politica autonoma de calidad: "+action+" sobre "+subject+"; "+summary+".")
		if len(criteria) >= 3 {
			break
		}
	}
	return criteria
}

func startAppDirectorGoalFirstResultV0(
	request StartAppDirectorRequestV0,
	spec orquestafactory.AppSpecV0,
	prepared orquestaappdirectorintake.AppDirectorIntakePreparedV0,
	receipt orquestagoal.GoalLaunchReceiptV0,
) StartAppDirectorResultV0 {
	status := StartAppDirectorStatusPendingV0
	switch strings.TrimSpace(receipt.Status) {
	case orquestagoal.GoalStatusAcceptedV0, orquestagoal.GoalStatusRunningV0, orquestagoal.GoalStatusCompleteV0:
		status = StartAppDirectorStatusStartedV0
	}
	return StartAppDirectorResultV0{
		SchemaVersion:         StartAppDirectorResultSchemaVersionV0,
		Status:                status,
		DirectorExecutionMode: AppDirectorExecutionModeGoalFirstV0,
		CorrelationID:         request.CorrelationID,
		AppSpec:               spec,
		Run:                   prepared.Run,
		GoalRef:               strings.TrimSpace(receipt.GoalRef),
		ExternalGoalRef:       strings.TrimSpace(receipt.ExternalGoalRef),
		GoalStatus:            strings.TrimSpace(receipt.Status),
		GoalLaunchReceipt:     &receipt,
		EvidenceRefs: compactStartAppDirectorStringsV0(append(
			append([]string{"evidence-ref-app-director-goal-first-launched-v0"}, prepared.EvidenceRefs...),
			receipt.EvidenceRefs...,
		)),
	}
}

func firstStartAppDirectorGoalValueV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func appDirectorGoalSafeTokenV0(values ...string) string {
	source := firstStartAppDirectorGoalValueV0(values...)
	source = strings.ToLower(strings.TrimSpace(source))
	var builder strings.Builder
	lastDash := false
	for _, r := range source {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if ok {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteRune('-')
			lastDash = true
		}
	}
	token := strings.Trim(builder.String(), "-")
	if token == "" {
		return "app"
	}
	return token
}
