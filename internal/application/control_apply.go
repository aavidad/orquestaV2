package application

import (
	"context"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

func (orchestrator *Orchestrator) applyNewControl(
	ctx context.Context,
	request ControlRequest,
	fingerprint string,
	principal identity.PrincipalRef,
	projectRef goal.ProjectRef,
	authorization identity.AuthorizationReceipt,
) (ControlResult, error) {
	current, err := orchestrator.state.GetGoal(ctx, request.GoalRef)
	if err != nil {
		return ControlResult{}, err
	}
	item, execution, err := validateControlFences(current, projectRef, request)
	if err != nil {
		return orchestrator.replayControlAfterConflict(
			ctx, request, fingerprint, principal, projectRef, err,
		)
	}
	supersededOwner, err := exclusiveStopOwner(current, request)
	if err != nil {
		return orchestrator.replayControlAfterConflict(
			ctx, request, fingerprint, principal, projectRef, err,
		)
	}
	controlRef, err := orchestrator.ids.NewID(ctx, "control")
	if err != nil {
		return ControlResult{}, err
	}
	now := lifecycleTime(orchestrator.clock.Now(), current.Goal, item)
	control := newControlRecord(
		controlRef, fingerprint, principal, projectRef, request, authorization, now,
	)
	var superseded *ControlRecord
	if supersededOwner.Ref != "" {
		control.SupersedesControlRef = supersededOwner.Ref
		old := supersededOwner
		old.Status = ControlSuperseded
		old.SupersededAt = now
		old.SupersededByControlRef = control.Ref
		superseded = &old
	}
	state := ApplyControlState{
		RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		AuthorizationReceipt: authorization, PrincipalRef: principal, ProjectRef: projectRef,
		GoalRef: request.GoalRef, ExpectedGoalRevision: request.ExpectedGoalRevision,
		ExpectedPlanGeneration:   request.ExpectedPlanGeneration,
		ExpectedWorkItemRevision: request.ExpectedWorkItemRevision,
		ExpectedExecutionState:   execution.State,
		Goal:                     current.Goal, Control: control, OperationAt: now,
		SupersededControl: superseded,
	}
	if superseded != nil {
		state.RetireActionRefs = []string{
			"action:stop:" + superseded.Ref + ":" + superseded.ExecutionRef.String(),
		}
	}
	if err := orchestrator.buildInitialControl(ctx, current, item, execution, request, &state); err != nil {
		return orchestrator.replayControlAfterConflict(
			ctx, request, fingerprint, principal, projectRef, err,
		)
	}
	persisted, created, err := orchestrator.state.ApplyControl(ctx, state)
	if err != nil {
		return orchestrator.replayControlAfterConflict(
			ctx, request, fingerprint, principal, projectRef, err,
		)
	}
	if err := validateControlResult(request, fingerprint, principal, projectRef, persisted); err != nil {
		return ControlResult{}, err
	}
	return ControlResult{Control: persisted, Created: created}, nil
}
