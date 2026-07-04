package orquestaappcodexstack

import (
	"context"
	"errors"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
)

const (
	runControlGoalForcedStopTerminalEvidenceV0   = "evidence-ref-run-control-goal-forced-stop-terminal"
	runControlGoalForcedCancelTerminalEvidenceV0 = "evidence-ref-run-control-goal-forced-cancel-terminal"
	runControlGoalForcedStopCompleteEvidenceV0   = "evidence-ref-run-control-terminal-after-goal-forced-stop"
	runControlGoalForcedCancelCompleteEvidenceV0 = "evidence-ref-run-control-terminal-after-goal-forced-cancel"
)

type GoalBackendControlRequestV0 struct {
	RunRef          string   `json:"run_ref,omitempty"`
	GoalRef         string   `json:"goal_ref,omitempty"`
	ExternalGoalRef string   `json:"external_goal_ref,omitempty"`
	Action          string   `json:"action,omitempty"`
	Reason          string   `json:"reason,omitempty"`
	Forced          bool     `json:"forced,omitempty"`
	EvidenceRefs    []string `json:"evidence_refs,omitempty"`
}

type GoalBackendControlResultV0 struct {
	Status          string   `json:"status,omitempty"`
	GoalRef         string   `json:"goal_ref,omitempty"`
	ExternalGoalRef string   `json:"external_goal_ref,omitempty"`
	GoalStatusSet   bool     `json:"goal_status_set,omitempty"`
	BackendStopped  bool     `json:"backend_stopped,omitempty"`
	IssueCode       string   `json:"issue_code,omitempty"`
	EvidenceRefs    []string `json:"evidence_refs,omitempty"`
}

type GoalBackendControlPortV0 interface {
	ControlGoalBackendV0(context.Context, GoalBackendControlRequestV0) (GoalBackendControlResultV0, error)
}

type goalFirstRunControlPortV0 struct {
	Inner          orquestaruncontrol.RunControlPortV0
	GoalStateStore orquestagoal.GoalWorkStateStorePortV0
	BackendControl GoalBackendControlPortV0
}

func goalFirstRunControlPortFromConfigV0(config ConfigV0) orquestaruncontrol.RunControlPortV0 {
	if config.Stores.RunControl == nil {
		return nil
	}
	if config.Stores.AppGoalStateStore == nil || config.AppGoalBackendControl == nil {
		return config.Stores.RunControl
	}
	return goalFirstRunControlPortV0{
		Inner:          config.Stores.RunControl,
		GoalStateStore: config.Stores.AppGoalStateStore,
		BackendControl: config.AppGoalBackendControl,
	}
}

func (port goalFirstRunControlPortV0) ReadRunControlStateV0(
	ctx context.Context,
	request orquestaruncontrol.RunControlReadRequestV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	if port.Inner == nil {
		return orquestaruncontrol.RunControlStateV0{}, errors.New("run_control_port_missing")
	}
	return port.Inner.ReadRunControlStateV0(ctx, request)
}

func (port goalFirstRunControlPortV0) PauseRunV0(
	ctx context.Context,
	command orquestaruncontrol.PauseRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	if port.Inner == nil {
		return orquestaruncontrol.RunControlStateV0{}, errors.New("run_control_port_missing")
	}
	return port.Inner.PauseRunV0(ctx, command)
}

func (port goalFirstRunControlPortV0) ResumeRunV0(
	ctx context.Context,
	command orquestaruncontrol.ResumeRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	if port.Inner == nil {
		return orquestaruncontrol.RunControlStateV0{}, errors.New("run_control_port_missing")
	}
	return port.Inner.ResumeRunV0(ctx, command)
}

func (port goalFirstRunControlPortV0) StopRunV0(
	ctx context.Context,
	command orquestaruncontrol.StopRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	if port.Inner == nil {
		return orquestaruncontrol.RunControlStateV0{}, errors.New("run_control_port_missing")
	}
	if command.Forced && port.forcedTerminalGoalStateExistsV0(ctx, command.RunRef) {
		return port.completeAlreadyForcedTerminalRunControlV0(ctx, goalFirstRunControlForcedCommandV0{
			RunRef:           command.RunRef,
			RequestedBy:      command.RequestedBy,
			Reason:           command.Reason,
			IdempotencyKey:   command.IdempotencyKey,
			EvidenceRefs:     command.EvidenceRefs,
			TerminalEvidence: runControlGoalForcedStopTerminalEvidenceV0,
			CompleteEvidence: runControlGoalForcedStopCompleteEvidenceV0,
			TargetStatus:     orquestaruncontrol.RunControlStatusStoppedV0,
		})
	}
	if command.Forced {
		if completed, ok, err := port.completeAlreadyTerminalGoalRunControlV0(ctx, goalFirstRunControlForcedCommandV0{
			RunRef:           command.RunRef,
			RequestedBy:      command.RequestedBy,
			Reason:           command.Reason,
			IdempotencyKey:   command.IdempotencyKey,
			EvidenceRefs:     command.EvidenceRefs,
			Action:           "stop",
			IssueField:       "goal_backend",
			IssueCode:        "operator_forced_stop_goal_first",
			NoArtifactsCode:  "operator_forced_stop_no_artifacts",
			Summary:          "forced stop accepted after goal was already terminal",
			TerminalEvidence: runControlGoalForcedStopTerminalEvidenceV0,
			CompleteEvidence: runControlGoalForcedStopCompleteEvidenceV0,
			TargetStatus:     orquestaruncontrol.RunControlStatusStoppedV0,
		}); ok || err != nil {
			return completed, err
		}
	}
	state, err := port.Inner.StopRunV0(ctx, command)
	if err != nil || !command.Forced {
		return state, err
	}
	return port.forceTerminalGoalFirstRunControlV0(ctx, goalFirstRunControlForcedCommandV0{
		RunRef:           command.RunRef,
		RequestedBy:      command.RequestedBy,
		Reason:           command.Reason,
		IdempotencyKey:   command.IdempotencyKey,
		EvidenceRefs:     command.EvidenceRefs,
		Action:           "stop",
		IssueField:       "goal_backend",
		IssueCode:        "operator_forced_stop_goal_first",
		NoArtifactsCode:  "operator_forced_stop_no_artifacts",
		Summary:          "forced stop accepted; goal backend stopped and marked blocked for rework",
		TerminalEvidence: runControlGoalForcedStopTerminalEvidenceV0,
		CompleteEvidence: runControlGoalForcedStopCompleteEvidenceV0,
		TargetStatus:     orquestaruncontrol.RunControlStatusStoppedV0,
	}, state, err)
}

func (port goalFirstRunControlPortV0) CancelRunV0(
	ctx context.Context,
	command orquestaruncontrol.CancelRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	if port.Inner == nil {
		return orquestaruncontrol.RunControlStateV0{}, errors.New("run_control_port_missing")
	}
	if command.Forced && port.forcedTerminalGoalStateExistsV0(ctx, command.RunRef) {
		return port.completeAlreadyForcedTerminalRunControlV0(ctx, goalFirstRunControlForcedCommandV0{
			RunRef:           command.RunRef,
			RequestedBy:      command.RequestedBy,
			Reason:           command.Reason,
			IdempotencyKey:   command.IdempotencyKey,
			EvidenceRefs:     command.EvidenceRefs,
			TerminalEvidence: runControlGoalForcedCancelTerminalEvidenceV0,
			CompleteEvidence: runControlGoalForcedCancelCompleteEvidenceV0,
			TargetStatus:     orquestaruncontrol.RunControlStatusCanceledV0,
		})
	}
	if command.Forced {
		if completed, ok, err := port.completeAlreadyTerminalGoalRunControlV0(ctx, goalFirstRunControlForcedCommandV0{
			RunRef:           command.RunRef,
			RequestedBy:      command.RequestedBy,
			Reason:           command.Reason,
			IdempotencyKey:   command.IdempotencyKey,
			EvidenceRefs:     command.EvidenceRefs,
			Action:           "cancel",
			IssueField:       "goal_backend",
			IssueCode:        "operator_forced_cancel_goal_first",
			NoArtifactsCode:  "operator_forced_cancel_no_artifacts",
			Summary:          "forced cancel accepted after goal was already terminal",
			TerminalEvidence: runControlGoalForcedCancelTerminalEvidenceV0,
			CompleteEvidence: runControlGoalForcedCancelCompleteEvidenceV0,
			TargetStatus:     orquestaruncontrol.RunControlStatusCanceledV0,
		}); ok || err != nil {
			return completed, err
		}
	}
	state, err := port.Inner.CancelRunV0(ctx, command)
	if err != nil || !command.Forced {
		return state, err
	}
	return port.forceTerminalGoalFirstRunControlV0(ctx, goalFirstRunControlForcedCommandV0{
		RunRef:           command.RunRef,
		RequestedBy:      command.RequestedBy,
		Reason:           command.Reason,
		IdempotencyKey:   command.IdempotencyKey,
		EvidenceRefs:     command.EvidenceRefs,
		Action:           "cancel",
		IssueField:       "goal_backend",
		IssueCode:        "operator_forced_cancel_goal_first",
		NoArtifactsCode:  "operator_forced_cancel_no_artifacts",
		Summary:          "forced cancel accepted; goal backend stopped and marked blocked for rework",
		TerminalEvidence: runControlGoalForcedCancelTerminalEvidenceV0,
		CompleteEvidence: runControlGoalForcedCancelCompleteEvidenceV0,
		TargetStatus:     orquestaruncontrol.RunControlStatusCanceledV0,
	}, state, err)
}

func (port goalFirstRunControlPortV0) RecordRunCheckpointV0(
	ctx context.Context,
	command orquestaruncontrol.RecordRunCheckpointCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	if port.Inner == nil {
		return orquestaruncontrol.RunControlStateV0{}, errors.New("run_control_port_missing")
	}
	return port.Inner.RecordRunCheckpointV0(ctx, command)
}

func (port goalFirstRunControlPortV0) CompleteRunControlV0(
	ctx context.Context,
	command orquestaruncontrol.CompleteRunControlCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	if port.Inner == nil {
		return orquestaruncontrol.RunControlStateV0{}, errors.New("run_control_port_missing")
	}
	return port.Inner.CompleteRunControlV0(ctx, command)
}

type goalFirstRunControlForcedCommandV0 struct {
	RunRef           string
	RequestedBy      string
	Reason           string
	IdempotencyKey   string
	EvidenceRefs     []string
	Action           string
	IssueField       string
	IssueCode        string
	NoArtifactsCode  string
	Summary          string
	TerminalEvidence string
	CompleteEvidence string
	TargetStatus     orquestaruncontrol.RunControlStatusV0
}

func (port goalFirstRunControlPortV0) forceTerminalGoalFirstRunControlV0(
	ctx context.Context,
	command goalFirstRunControlForcedCommandV0,
	fallback orquestaruncontrol.RunControlStateV0,
	fallbackErr error,
) (orquestaruncontrol.RunControlStateV0, error) {
	if port.GoalStateStore == nil || port.BackendControl == nil {
		return fallback, fallbackErr
	}
	state, ok := port.loadRunningGoalStateForForcedRunControlV0(ctx, command.RunRef)
	if !ok {
		return fallback, fallbackErr
	}
	control, err := port.BackendControl.ControlGoalBackendV0(ctx, GoalBackendControlRequestV0{
		RunRef:          state.RunRef,
		GoalRef:         state.GoalRef,
		ExternalGoalRef: state.ExternalGoalRef,
		Action:          command.Action,
		Reason:          command.Reason,
		Forced:          true,
		EvidenceRefs:    command.EvidenceRefs,
	})
	if err != nil || strings.TrimSpace(control.Status) != orquestagoal.GoalStatusBlockedV0 || !control.BackendStopped {
		return fallback, fallbackErr
	}
	evidenceRefs := compactStringsV0(append(
		append(append([]string(nil), command.EvidenceRefs...), control.EvidenceRefs...),
		command.TerminalEvidence,
	))
	if err := port.saveForcedTerminalGoalStateV0(ctx, state, command, evidenceRefs); err != nil {
		return fallback, fallbackErr
	}
	completed, err := port.Inner.CompleteRunControlV0(ctx, orquestaruncontrol.CompleteRunControlCommandV0{
		RunRef:         state.RunRef,
		TargetStatus:   command.TargetStatus,
		RequestedBy:    command.RequestedBy,
		Reason:         command.Reason,
		IdempotencyKey: command.IdempotencyKey,
		EvidenceRefs:   compactStringsV0(append(evidenceRefs, command.CompleteEvidence)),
	})
	if err != nil {
		return fallback, fallbackErr
	}
	completed.Forced = true
	return completed, nil
}

func (port goalFirstRunControlPortV0) completeAlreadyForcedTerminalRunControlV0(
	ctx context.Context,
	command goalFirstRunControlForcedCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	completed, err := port.Inner.CompleteRunControlV0(ctx, orquestaruncontrol.CompleteRunControlCommandV0{
		RunRef:         command.RunRef,
		TargetStatus:   command.TargetStatus,
		RequestedBy:    command.RequestedBy,
		Reason:         command.Reason,
		IdempotencyKey: command.IdempotencyKey,
		EvidenceRefs: compactStringsV0(append(
			append([]string(nil), command.EvidenceRefs...),
			command.TerminalEvidence,
			command.CompleteEvidence,
		)),
	})
	if err != nil {
		return completed, err
	}
	completed.Forced = true
	return completed, nil
}

func (port goalFirstRunControlPortV0) completeAlreadyTerminalGoalRunControlV0(
	ctx context.Context,
	command goalFirstRunControlForcedCommandV0,
) (orquestaruncontrol.RunControlStateV0, bool, error) {
	if port.GoalStateStore == nil {
		return orquestaruncontrol.RunControlStateV0{}, false, nil
	}
	state, err := port.GoalStateStore.LoadGoalWorkStateV0(ctx, strings.TrimSpace(command.RunRef))
	if err != nil {
		return orquestaruncontrol.RunControlStateV0{}, false, nil
	}
	state, err = orquestagoal.NewGoalWorkStateV0(state)
	if err != nil || !goalFirstRunControlAlreadyTerminalForForcedControlV0(state) {
		return orquestaruncontrol.RunControlStateV0{}, false, nil
	}
	evidenceRefs := compactStringsV0(append(
		append(append([]string(nil), command.EvidenceRefs...), state.EvidenceRefs...),
		command.TerminalEvidence,
	))
	state = goalFirstRunControlMarkAlreadyTerminalStateForcedV0(state, command, evidenceRefs)
	if err := port.GoalStateStore.SaveGoalWorkStateV0(ctx, state); err != nil {
		return orquestaruncontrol.RunControlStateV0{}, true, err
	}
	completed, err := port.Inner.CompleteRunControlV0(ctx, orquestaruncontrol.CompleteRunControlCommandV0{
		RunRef:         state.RunRef,
		TargetStatus:   command.TargetStatus,
		RequestedBy:    command.RequestedBy,
		Reason:         command.Reason,
		IdempotencyKey: command.IdempotencyKey,
		EvidenceRefs: compactStringsV0(append(
			evidenceRefs,
			command.CompleteEvidence,
		)),
	})
	if err != nil {
		return completed, true, err
	}
	completed.Forced = true
	return completed, true, nil
}

func goalFirstRunControlAlreadyTerminalForForcedControlV0(
	state orquestagoal.GoalWorkStateV0,
) bool {
	if strings.TrimSpace(state.Status) == orquestagoal.GoalStatusRunningV0 {
		return false
	}
	if goalFirstForcedTerminalStateV0(state) {
		return true
	}
	if orquestagoal.GoalWorkResultTerminalV0(state.Status) {
		return true
	}
	if state.LastResult != nil && orquestagoal.GoalWorkResultTerminalV0(state.LastResult.Status) {
		return true
	}
	if state.LastClosure != nil &&
		(state.LastClosure.NeedsRework ||
			strings.TrimSpace(state.LastClosure.Status) == orquestagoal.GoalStatusBlockedV0 ||
			strings.TrimSpace(state.LastClosure.Status) == orquestagoal.GoalStatusInvalidV0) {
		return true
	}
	return false
}

func goalFirstRunControlMarkAlreadyTerminalStateForcedV0(
	state orquestagoal.GoalWorkStateV0,
	command goalFirstRunControlForcedCommandV0,
	evidenceRefs []string,
) orquestagoal.GoalWorkStateV0 {
	state.Status = orquestagoal.GoalStatusBlockedV0
	state.EvidenceRefs = compactStringsV0(append(state.EvidenceRefs, evidenceRefs...))
	if state.LastResult != nil {
		result := orquestagoal.NormalizeGoalWorkResultV0(*state.LastResult)
		result.Status = orquestagoal.GoalStatusBlockedV0
		result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, evidenceRefs...))
		if len(result.Issues) == 0 {
			result.Issues = []orquestagoal.GoalWorkIssueV0{{
				Code:  command.IssueCode,
				Field: command.IssueField,
			}}
		}
		state.LastResult = &result
	} else {
		state.LastResult = &orquestagoal.GoalWorkResultV0{
			SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
			Status:          orquestagoal.GoalStatusBlockedV0,
			GoalRef:         strings.TrimSpace(state.GoalRef),
			ExternalGoalRef: strings.TrimSpace(state.ExternalGoalRef),
			Summary:         command.Summary,
			EvidenceRefs:    evidenceRefs,
			Issues: []orquestagoal.GoalWorkIssueV0{{
				Code:  command.IssueCode,
				Field: command.IssueField,
			}},
		}
	}
	if state.LastClosure != nil {
		closure := *state.LastClosure
		closure.Status = orquestagoal.GoalStatusBlockedV0
		closure.NeedsRework = true
		closure.EvidenceRefs = compactStringsV0(append(closure.EvidenceRefs, evidenceRefs...))
		if len(closure.Issues) == 0 {
			closure.Issues = []orquestagoal.GoalWorkIssueV0{{
				Code:  command.IssueCode,
				Field: command.IssueField,
			}}
		}
		state.LastClosure = &closure
	} else {
		state.LastClosure = &orquestagoal.GoalClosureValidationV0{
			Status:       orquestagoal.GoalStatusBlockedV0,
			NeedsRework:  true,
			EvidenceRefs: evidenceRefs,
			Issues: []orquestagoal.GoalWorkIssueV0{{
				Code:  command.IssueCode,
				Field: command.IssueField,
			}},
		}
	}
	return state
}

func (port goalFirstRunControlPortV0) forcedTerminalGoalStateExistsV0(
	ctx context.Context,
	runRef string,
) bool {
	if port.GoalStateStore == nil {
		return false
	}
	state, err := port.GoalStateStore.LoadGoalWorkStateV0(ctx, strings.TrimSpace(runRef))
	if err != nil {
		return false
	}
	state, err = orquestagoal.NewGoalWorkStateV0(state)
	return err == nil && goalFirstForcedTerminalStateV0(state)
}

func (port goalFirstRunControlPortV0) loadRunningGoalStateForForcedRunControlV0(
	ctx context.Context,
	runRef string,
) (orquestagoal.GoalWorkStateV0, bool) {
	state, err := port.GoalStateStore.LoadGoalWorkStateV0(ctx, strings.TrimSpace(runRef))
	if err != nil {
		return orquestagoal.GoalWorkStateV0{}, false
	}
	state, err = orquestagoal.NewGoalWorkStateV0(state)
	if err != nil || strings.TrimSpace(state.Status) != orquestagoal.GoalStatusRunningV0 {
		return orquestagoal.GoalWorkStateV0{}, false
	}
	return state, true
}

func (port goalFirstRunControlPortV0) saveForcedTerminalGoalStateV0(
	ctx context.Context,
	state orquestagoal.GoalWorkStateV0,
	command goalFirstRunControlForcedCommandV0,
	evidenceRefs []string,
) error {
	issueCode := command.IssueCode
	if len(goalFirstRunControlArtifactRefsV0(state)) == 0 && len(goalFirstRunControlDomainReceiptRefsV0(state)) == 0 {
		issueCode = command.NoArtifactsCode
	}
	state.Status = orquestagoal.GoalStatusBlockedV0
	state.LastResult = &orquestagoal.GoalWorkResultV0{
		SchemaVersion:     orquestagoal.GoalWorkResultSchemaV0,
		Status:            orquestagoal.GoalStatusBlockedV0,
		GoalRef:           strings.TrimSpace(state.GoalRef),
		ExternalGoalRef:   strings.TrimSpace(state.ExternalGoalRef),
		Summary:           command.Summary,
		ArtifactRefs:      goalFirstRunControlArtifactRefsV0(state),
		DomainReceiptRefs: goalFirstRunControlDomainReceiptRefsV0(state),
		EvidenceRefs:      evidenceRefs,
		Issues: []orquestagoal.GoalWorkIssueV0{{
			Code:  issueCode,
			Field: command.IssueField,
		}},
	}
	state.LastClosure = &orquestagoal.GoalClosureValidationV0{
		Status:       orquestagoal.GoalStatusBlockedV0,
		NeedsRework:  true,
		EvidenceRefs: evidenceRefs,
		Issues: []orquestagoal.GoalWorkIssueV0{{
			Code:  issueCode,
			Field: command.IssueField,
		}},
	}
	state.EvidenceRefs = compactStringsV0(append(state.EvidenceRefs, evidenceRefs...))
	normalized, err := orquestagoal.NewGoalWorkStateV0(state)
	if err != nil {
		return err
	}
	return port.GoalStateStore.SaveGoalWorkStateV0(ctx, normalized)
}

func goalFirstRunControlArtifactRefsV0(state orquestagoal.GoalWorkStateV0) []string {
	if state.LastResult == nil {
		return []string{}
	}
	return compactStringsV0(state.LastResult.ArtifactRefs)
}

func goalFirstRunControlDomainReceiptRefsV0(state orquestagoal.GoalWorkStateV0) []string {
	if state.LastResult == nil {
		return []string{}
	}
	return compactStringsV0(state.LastResult.DomainReceiptRefs)
}
