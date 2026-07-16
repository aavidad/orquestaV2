package application

import (
	"context"
	"errors"
	"strings"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

// Control applies one authenticated, exactly fenced lifecycle control. Stop
// effects remain asynchronous: this call only persists their existing-outbox
// work; ProcessNext executes the provider effect later.
func (orchestrator *Orchestrator) Control(
	ctx context.Context,
	access Access,
	request ControlRequest,
) (ControlResult, error) {
	if orchestrator == nil {
		return ControlResult{}, errors.New("application.unavailable")
	}
	request.Reason = strings.TrimSpace(request.Reason)
	if err := validateControlRequest(request); err != nil {
		return ControlResult{}, err
	}
	principal, projectRef, err := access.values()
	if err != nil {
		return ControlResult{}, err
	}
	fingerprint := controlFingerprint(principal.Ref, projectRef, request)
	if err := orchestrator.requireCurrentDirectorAccess(ctx, principal.Ref, projectRef); err != nil {
		return ControlResult{}, err
	}
	replayed, found, err := orchestrator.state.ControlReplay(ctx, ControlReplayRequest{
		RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		PrincipalRef: principal.Ref, ProjectRef: projectRef, GoalRef: request.GoalRef,
	})
	if err != nil {
		return ControlResult{}, err
	}
	if found {
		if err := validateControlResult(request, fingerprint, principal.Ref, projectRef, replayed); err != nil {
			return ControlResult{}, err
		}
		return ControlResult{Control: replayed}, nil
	}
	authorization, err := orchestrator.authorizeIdempotentWithRequestRef(
		ctx, access, identity.PermissionGoalsDirect, request.GoalRef.String(), orchestrator.clock.Now().UTC(),
		controlAuthorizationRequestRef(request.RequestRef, fingerprint),
	)
	if err != nil {
		return ControlResult{}, err
	}
	return orchestrator.applyNewControl(
		ctx, request, fingerprint, principal.Ref, projectRef, authorization,
	)
}

// replayControlAfterConflict closes the narrow concurrent-idempotency race in
// which two identical requests both miss their first replay read and one then
// loses a lifecycle CAS after the other commits. Only an exact persisted
// semantic match converts that conflict into a replay; unrelated/stale
// conflicts retain their original error.
func (orchestrator *Orchestrator) replayControlAfterConflict(
	ctx context.Context,
	request ControlRequest,
	fingerprint string,
	principal identity.PrincipalRef,
	projectRef goal.ProjectRef,
	conflict error,
) (ControlResult, error) {
	if !IsStateError(conflict, StateConflict) {
		return ControlResult{}, conflict
	}
	replayed, found, err := orchestrator.state.ControlReplay(ctx, ControlReplayRequest{
		RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		PrincipalRef: principal, ProjectRef: projectRef, GoalRef: request.GoalRef,
	})
	if err != nil || !found {
		return ControlResult{}, conflict
	}
	if err := validateControlResult(request, fingerprint, principal, projectRef, replayed); err != nil {
		return ControlResult{}, conflict
	}
	return ControlResult{Control: replayed}, nil
}
