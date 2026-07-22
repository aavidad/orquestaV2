package application

import (
	"errors"
	"strconv"
	"strings"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

func validateControlRequest(request ControlRequest) error {
	request.Reason = strings.TrimSpace(request.Reason)
	switch {
	case !validApplicationRef(request.RequestRef):
		return errors.New("application.request_ref_invalid")
	case request.GoalRef.String() == "":
		return errors.New("application.goal_ref_required")
	case request.ExpectedGoalRevision == 0 || request.ExpectedPlanGeneration == 0:
		return errors.New("application.control_goal_fence_required")
	case request.ExpectedAppSpecGeneration == 0 || !goal.IsCanonicalAppSpecHash(request.ExpectedSpecHash):
		return errors.New("application.control_spec_fence_invalid")
	case request.Reason == "":
		return errors.New("application.control_reason_required")
	}

	workItemRequired := request.Target == ControlTargetWorkItem || request.Target == ControlTargetExecution
	executionRequired := request.Operation == ControlStop || request.Operation == ControlRetry
	hasWorkItemRef := request.WorkItemRef.String() != ""
	hasWorkItemRevision := request.ExpectedWorkItemRevision != 0
	if hasWorkItemRef != hasWorkItemRevision || workItemRequired != hasWorkItemRef {
		return errors.New("application.control_work_item_fence_invalid")
	}
	hasExecutionRef := request.ExecutionRef.String() != ""
	hasExecutionAttempt := request.ExpectedExecutionAttempt != 0
	if hasExecutionRef != hasExecutionAttempt || executionRequired != hasExecutionRef {
		return errors.New("application.control_execution_fence_invalid")
	}

	valid := false
	switch request.Operation {
	case ControlPause, ControlResume, ControlCancel:
		valid = request.Target == ControlTargetGoal || request.Target == ControlTargetWorkItem
	case ControlStop:
		valid = request.Target == ControlTargetExecution && validStopMode(request.Mode)
	case ControlRetry:
		valid = request.Target == ControlTargetWorkItem && request.Mode == ""
	}
	if !valid {
		return errors.New("application.control_target_invalid")
	}
	if request.Operation != ControlStop && request.Mode != "" {
		return errors.New("application.control_mode_unexpected")
	}
	return nil
}

func validStopMode(mode ports.AgentStopMode) bool {
	return mode == ports.AgentStopCooperative || mode == ports.AgentStopForced
}

func controlFingerprint(
	principal identity.PrincipalRef,
	project goal.ProjectRef,
	request ControlRequest,
) string {
	return fingerprintFields(
		"orquesta.control.v1", principal.String(), project.String(), request.GoalRef.String(),
		strconv.FormatUint(uint64(request.ExpectedGoalRevision), 10),
		strconv.FormatUint(uint64(request.ExpectedPlanGeneration), 10),
		strconv.FormatUint(uint64(request.ExpectedAppSpecGeneration), 10), request.ExpectedSpecHash,
		string(request.Operation), string(request.Target), request.WorkItemRef.String(),
		strconv.FormatUint(uint64(request.ExpectedWorkItemRevision), 10), request.ExecutionRef.String(),
		strconv.FormatUint(request.ExpectedExecutionAttempt, 10), string(request.Mode), strings.TrimSpace(request.Reason),
	)
}

func validateControlResult(
	request ControlRequest,
	fingerprint string,
	principal identity.PrincipalRef,
	projectRef goal.ProjectRef,
	record ControlRecord,
) error {
	modeMatches := record.Mode == request.Mode
	if request.Operation == ControlCancel {
		modeMatches = record.Mode == "" || validStopMode(record.Mode)
	}
	authorizationValid := directorAuthorizationValid(
		record.AuthorizationReceipt, principal, projectRef, request.GoalRef,
	)
	if IsReviewCleanupControl(record) {
		authorizationValid = ReviewCleanupAuthorizationValid(record)
	}
	if record.Ref == "" || record.RequestRef != request.RequestRef ||
		record.RequestFingerprint != fingerprint || record.PrincipalRef != principal ||
		record.ProjectRef != projectRef || record.GoalRef != request.GoalRef ||
		record.WorkItemRef != request.WorkItemRef || record.WorkItemRevision != request.ExpectedWorkItemRevision ||
		record.ExecutionRef != request.ExecutionRef || record.ExecutionAttempt != request.ExpectedExecutionAttempt ||
		record.Operation != request.Operation || record.Target != request.Target || !modeMatches ||
		record.Reason != request.Reason || record.GoalRevision != request.ExpectedGoalRevision ||
		record.PlanGeneration != request.ExpectedPlanGeneration ||
		record.AppSpecGeneration != request.ExpectedAppSpecGeneration || record.SpecHash != request.ExpectedSpecHash ||
		record.RequestedAt.IsZero() || !authorizationValid {
		return &StateError{Code: StateConflict}
	}
	switch record.Status {
	case ControlRequested:
		if !record.ConfirmedAt.IsZero() || record.ReceiptRef != "" {
			return &StateError{Code: StateConflict}
		}
	case ControlConfirmed:
		if record.ConfirmedAt.IsZero() || record.ReceiptRef == "" || record.ConfirmedAt.Before(record.RequestedAt) {
			return &StateError{Code: StateConflict}
		}
	case ControlSuperseded:
		if record.Operation != ControlStop || record.Target != ControlTargetExecution ||
			record.Mode != ports.AgentStopCooperative || !record.ConfirmedAt.IsZero() || record.ReceiptRef != "" ||
			record.SupersedesControlRef != "" || record.SupersededAt.IsZero() ||
			record.SupersededAt.Before(record.RequestedAt) || !validApplicationRef(record.SupersededByControlRef) {
			return &StateError{Code: StateConflict}
		}
	default:
		return &StateError{Code: StateConflict}
	}
	if record.Status != ControlSuperseded && (!record.SupersededAt.IsZero() || record.SupersededByControlRef != "") {
		return &StateError{Code: StateConflict}
	}
	if record.SupersedesControlRef != "" &&
		(record.Operation != ControlStop || record.Target != ControlTargetExecution ||
			record.Mode != ports.AgentStopForced || !validApplicationRef(record.SupersedesControlRef)) {
		return &StateError{Code: StateConflict}
	}
	return nil
}

// ValidatePersistedControlRecord is the canonical semantic validator used by
// durable adapters during recovery. Cancel may acquire a provider stop mode
// after its request fingerprint was created, so its original request mode is
// intentionally reconstructed as empty.
func ValidatePersistedControlRecord(record ControlRecord) error {
	requestMode := record.Mode
	if record.Operation == ControlCancel {
		requestMode = ""
	}
	request := ControlRequest{
		RequestRef: record.RequestRef, Operation: record.Operation, Target: record.Target,
		GoalRef: record.GoalRef, ExpectedGoalRevision: record.GoalRevision,
		ExpectedPlanGeneration:    record.PlanGeneration,
		ExpectedAppSpecGeneration: record.AppSpecGeneration, ExpectedSpecHash: record.SpecHash,
		WorkItemRef: record.WorkItemRef, ExpectedWorkItemRevision: record.WorkItemRevision,
		ExecutionRef: record.ExecutionRef, ExpectedExecutionAttempt: record.ExecutionAttempt,
		Mode: requestMode, Reason: record.Reason,
	}
	if !validApplicationRef(record.Ref) || record.Reason != strings.TrimSpace(record.Reason) {
		return errors.New("application.control_record_invalid")
	}
	if err := validateControlRequest(request); err != nil {
		return errors.New("application.control_record_invalid")
	}
	fingerprint := controlFingerprint(record.PrincipalRef, record.ProjectRef, request)
	if err := validateControlResult(
		request, fingerprint, record.PrincipalRef, record.ProjectRef, record,
	); err != nil {
		return errors.New("application.control_record_invalid")
	}
	authorizationRequest := record.AuthorizationReceipt.Decision().Request()
	if IsReviewCleanupControl(record) {
		if !ReviewCleanupAuthorizationValid(record) {
			return errors.New("application.control_record_invalid")
		}
	} else if authorizationRequest.RequestRef() != controlAuthorizationRequestRef(record.RequestRef, fingerprint) ||
		record.AuthorizationReceipt.RecordedAt().After(record.RequestedAt) {
		return errors.New("application.control_record_invalid")
	}
	return nil
}

func ReviewCleanupAuthorizationValid(record ControlRecord) bool {
	if !IsReviewCleanupControl(record) {
		return false
	}
	request := record.AuthorizationReceipt.Decision().Request()
	return record.AuthorizationReceipt.Decision().Outcome() == identity.AuthorizationAllowed &&
		record.AuthorizationReceipt.Ref() != "" && !record.AuthorizationReceipt.RecordedAt().IsZero() &&
		identity.RoleAllows(record.AuthorizationReceipt.Decision().Role(), identity.PermissionGoalsCreate) &&
		request.Principal().Ref == record.PrincipalRef && request.ProjectRef() == record.ProjectRef &&
		request.Permission() == identity.PermissionGoalsCreate && request.ResourceRef() == record.ProjectRef.String() &&
		!record.AuthorizationReceipt.RecordedAt().After(record.RequestedAt)
}

func controlAuthorizationRequestRef(requestRef, fingerprint string) string {
	return "authorization-request:control:" +
		fingerprintFields("orquesta.control.authorization.v1", requestRef, fingerprint)
}
