package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

const (
	codexStackGoalFirstObserveRequiredOutcomeV0 = "goal_first_observe_required"
	codexStackGoalFirstStateMissingOutcomeV0    = "goal_first_state_missing"
)

type codexStackGoalFirstSupervisorDispositionV0 struct {
	RunRef       string
	Outcome      string
	QueueStatus  string
	RuntimeState CodexSupervisorRuntimeStateV0
	EvidenceRefs []string
	Diagnostics  []orquestaruncoordinator.RunDrainDiagnosticV0
}

func (stack StackV0) goalFirstSupervisorDispositionV0(
	ctx context.Context,
	runRef string,
) (codexStackGoalFirstSupervisorDispositionV0, bool) {
	runRef = strings.TrimSpace(runRef)
	if runRef == "" {
		return codexStackGoalFirstSupervisorDispositionV0{}, false
	}
	if stack.Ports.GoalStateStore != nil {
		state, err := stack.Ports.GoalStateStore.LoadGoalWorkStateV0(ctx, runRef)
		if err == nil &&
			strings.TrimSpace(state.RunRef) != "" &&
			strings.TrimSpace(state.GoalRef) != "" {
			return codexStackGoalFirstSupervisorDispositionFromStateV0(runRef, state), true
		}
	}
	if stack.Ports.RunStore == nil {
		return codexStackGoalFirstSupervisorDispositionV0{}, false
	}
	run, err := stack.Ports.RunStore.LoadRunV0(ctx, runRef)
	if err != nil || !codexStackRunSupervisorGoalFirstContainerWithoutStateV0(run) {
		return codexStackGoalFirstSupervisorDispositionV0{}, false
	}
	evidenceRefs := []string{
		"evidence-ref-run-supervisor-goal-first-container",
		"evidence-ref-run-supervisor-goal-first-state-missing",
	}
	return codexStackGoalFirstSupervisorDispositionV0{
		RunRef:       runRef,
		Outcome:      codexStackGoalFirstStateMissingOutcomeV0,
		QueueStatus:  orquestarunqueue.RunStatusStoppedV0,
		RuntimeState: CodexSupervisorRuntimeStoppedV0,
		EvidenceRefs: evidenceRefs,
		Diagnostics: []orquestaruncoordinator.RunDrainDiagnosticV0{{
			Kind:         "goal_first",
			Status:       "state_missing",
			RunRef:       runRef,
			Error:        codexStackGoalFirstStateMissingOutcomeV0,
			TargetPort:   "goal_state_store",
			EvidenceRefs: evidenceRefs,
		}},
	}, true
}

func codexStackGoalFirstSupervisorDispositionFromStateV0(
	runRef string,
	state orquestagoal.GoalWorkStateV0,
) codexStackGoalFirstSupervisorDispositionV0 {
	evidenceRefs := compactStringsV0(append(
		append([]string{}, state.EvidenceRefs...),
		"evidence-ref-run-supervisor-goal-first-redirect",
	))
	return codexStackGoalFirstSupervisorDispositionV0{
		RunRef:       runRef,
		Outcome:      codexStackGoalFirstObserveRequiredOutcomeV0,
		QueueStatus:  orquestarunqueue.RunStatusDeliveredV0,
		RuntimeState: codexStackGoalFirstSupervisorStatusV0(state.Status),
		EvidenceRefs: evidenceRefs,
		Diagnostics: []orquestaruncoordinator.RunDrainDiagnosticV0{{
			Kind:         "goal_first",
			Status:       "observe_required",
			RunRef:       runRef,
			TargetPort:   "goal_observer",
			EvidenceRefs: evidenceRefs,
		}},
	}
}

func (disposition codexStackGoalFirstSupervisorDispositionV0) drainResultV0(
	request orquestaruncoordinator.RunDrainRequestV0,
) orquestaruncoordinator.RunDrainResultV0 {
	return orquestaruncoordinator.RunDrainResultV0{
		RunRef:       strings.TrimSpace(disposition.RunRef),
		AppRef:       strings.TrimSpace(request.AppRef),
		Outcome:      strings.TrimSpace(disposition.Outcome),
		QueueStatus:  strings.TrimSpace(disposition.QueueStatus),
		EvidenceRefs: compactStringsV0(disposition.EvidenceRefs),
		Diagnostics:  disposition.Diagnostics,
	}
}

func (disposition codexStackGoalFirstSupervisorDispositionV0) snapshotV0() CodexSupervisorRuntimeSnapshotV0 {
	return CodexSupervisorRuntimeSnapshotV0{
		Status:       disposition.RuntimeState,
		SessionRef:   strings.TrimSpace(disposition.RunRef),
		EvidenceRefs: compactStringsV0(disposition.EvidenceRefs),
		Diagnostics:  disposition.Diagnostics,
	}
}
