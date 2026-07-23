package commands

import (
	"context"
	"errors"

	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

const (
	CodeInvalidRequest  = "invalid_request"
	CodeUnauthenticated = "unauthenticated"
	CodeForbidden       = "forbidden"
	CodeNotFound        = "not_found"
	CodeConflict        = "conflict"
	CodeUnavailable     = "unavailable"
	CodeInternal        = "internal"
)

var stableErrorCodes = []string{
	CodeInvalidRequest, CodeUnauthenticated, CodeForbidden, CodeNotFound,
	CodeConflict, CodeUnavailable, CodeInternal,
}

type commandError struct {
	code  string
	cause error
}

func (err commandError) Error() string { return "commands." + err.code }
func (err commandError) Unwrap() error { return err.cause }

func failure(code string) *Failure {
	if !isFailureCode(code) {
		code = CodeInternal
	}
	return &Failure{Code: code, MessageKey: "error." + code}
}

func applicationFailure(err error) *Failure {
	if err == nil {
		return nil
	}
	var boundary commandError
	if errors.As(err, &boundary) {
		return failure(boundary.code)
	}
	return failure(classifyApplicationError(err))
}

func normalizeApplicationError(err error) error {
	if err == nil {
		return nil
	}
	var boundary commandError
	if errors.As(err, &boundary) {
		return err
	}
	return commandError{code: classifyApplicationError(err), cause: err}
}

func classifyApplicationError(err error) string {
	switch {
	case errors.Is(err, application.ErrForbidden),
		errors.Is(err, application.ErrEffectCriticalSeparationRequired),
		errors.Is(err, application.ErrEffectCriticalProjectAuthorityRequired):
		return CodeForbidden
	case errors.Is(err, council.ErrSubjectMismatch):
		return CodeConflict
	case errors.Is(err, council.ErrInvalidSubject), errors.Is(err, council.ErrInvalidPolicy),
		errors.Is(err, council.ErrInvalidContribution), errors.Is(err, council.ErrInvalidFact),
		errors.Is(err, council.ErrInvalidSkip):
		return CodeInvalidRequest
	case application.IsStateError(err, application.StateNotFound):
		return CodeNotFound
	case application.IsStateError(err, application.StateConflict),
		application.IsStateError(err, application.StateAlreadyClaimed):
		return CodeConflict
	case application.IsStateError(err, application.StateInvalid):
		return CodeInternal
	case errors.Is(err, application.ErrPlanParentUnknown),
		errors.Is(err, application.ErrPlanDependencyUnknown):
		return CodeInvalidRequest
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return CodeUnavailable
	}
	switch goal.ErrorCodeOf(err) {
	case goal.ErrorInvalidArgument, goal.ErrorInvalidRef, goal.ErrorInvalidPlan,
		goal.ErrorDuplicateWorkItem, goal.ErrorWorkItemsRequired:
		return CodeInvalidRequest
	case goal.ErrorRevisionConflict, goal.ErrorInvalidTransition, goal.ErrorScopeConflict:
		return CodeConflict
	case "":
	default:
		return CodeInternal
	}
	if ports.ArtifactContractErrorCode(err) != "" || ports.WorkspaceContractErrorCode(err) != "" ||
		ports.VersionControlContractErrorCode(err) != "" {
		return CodeInvalidRequest
	}
	switch err.Error() {
	case "application.unavailable", "test_attestor.unavailable", "workspace.unavailable":
		return CodeUnavailable
	case "artifact.not_found":
		return CodeNotFound
	case "application.confirmation_required", "application.request_ref_invalid", "application.statement_required",
		"application.query_invalid", "application.pending_changes_request_invalid", "application.integrate_change_request_invalid",
		"application.membership_request_invalid", "application.mailbox_query_invalid", "application.mailbox_resolution_invalid",
		"application.mailbox_envelope_too_large", "identity.invalid_principal_ref", "identity.invalid_principal_kind",
		"identity.invalid_authentication_method", "identity.invalid_role", "identity.invalid_permission",
		"identity.membership_grant_invalid", "identity.membership_revoke_invalid",
		"application.source_goal_ref_required", "application.source_revision_required", "application.source_spec_hash_invalid",
		"application.amendment_reason_required", "application.plan_required", "application.plan_phase_duplicate",
		"application.plan_phase_unknown", "application.plan_fanout_exceeded", "application.plan_item_key_invalid",
		"application.plan_item_key_duplicate", "application.plan_item_key_ref_collision",
		"application.goal_ref_required", "application.director_lease_token_invalid", "application.director_lease_fence_invalid",
		"application.director_plan_revision_required", "application.director_plan_reason_required",
		"application.director_plan_work_items_required", "application.director_plan_replan_fence_unexpected",
		"application.director_plan_replan_fence_invalid", "application.director_plan_council_fence_invalid",
		"application.director_plan_council_fence_unexpected", "application.control_goal_fence_required",
		"application.control_spec_fence_invalid", "application.control_target_invalid",
		"application.control_operation_invalid",
		"application.control_work_item_fence_invalid", "application.control_execution_fence_invalid",
		"application.control_mode_unexpected", "application.control_reason_required",
		"application.control_stop_unsupported", "application.control_cancel_unsupported",
		"application.effect_ref_invalid", "application.effect_digest_invalid", "application.effect_decision_invalid",
		"application.effect_reason_required", "application.mailbox_admission_invalid",
		"application.mailbox_artifact_ref_invalid", "application.mailbox_request_invalid",
		"application.mailbox_claim_invalid", "application.mailbox_message_ref_invalid",
		"council.open_request_invalid", "council.skip_request_invalid", "council.policy_invalid",
		"council.resolution_invalid", "council.skip_invalid", "council.contribution_invalid",
		"council.fact_invalid",
		"commands.payload_invalid", "commands.output_invalid":
		return CodeInvalidRequest
	case "council.subject_invalid", "council.subject_mismatch", "council.review_gate_required",
		"council.policy_required", "council.resolution_required", "council.decision_invalid":
		return CodeConflict
	case "commands.output_contract_invalid":
		return CodeInternal
	default:
		return CodeInternal
	}
}

func isFailureCode(value string) bool {
	for _, code := range stableErrorCodes {
		if value == code {
			return true
		}
	}
	return false
}

var errContract = errors.New("commands.contract_invalid")
