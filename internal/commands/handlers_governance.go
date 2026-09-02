package commands

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func handlePreflightExpiredAgentLaunchContinuation(
	ctx context.Context, api applicationAPI, bound handlerContext, payload json.RawMessage,
) (json.RawMessage, error) {
	var input struct {
		ReconciliationAuthorityRef string `json:"reconciliation_authority_ref"`
	}
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	result, err := api.PreflightExpiredAgentLaunchContinuationV41(ctx, bound.access,
		application.PreflightExpiredAgentLaunchContinuationRequestV41{
			RequestRef: bound.requestRef, ReconciliationAuthorityRef: input.ReconciliationAuthorityRef,
		})
	preparation := result.Preparation
	return marshalApplication(struct {
		Preparation struct {
			ReconciliationAuthorityRef string    `json:"reconciliation_authority_ref"`
			ReconciliationAttemptRef   string    `json:"reconciliation_attempt_ref"`
			EffectAttemptRef           string    `json:"effect_attempt_ref"`
			ManifestSHA256             string    `json:"manifest_sha256"`
			RequestKeySHA256           string    `json:"request_key_sha256"`
			OriginalRequestSHA256      string    `json:"original_request_sha256"`
			AMVLaunchRef               string    `json:"amv_launch_ref"`
			AMVExecutionRef            string    `json:"amv_execution_ref"`
			AMVRunRef                  string    `json:"amv_run_ref"`
			AMVFence                   uint64    `json:"amv_fence"`
			AMVGeneration              uint64    `json:"amv_generation"`
			AMVCID                     uint32    `json:"amv_cid"`
			AMVIdentitySHA256          string    `json:"amv_identity_sha256"`
			PreparedAt                 time.Time `json:"prepared_at"`
		} `json:"preparation"`
	}{Preparation: struct {
		ReconciliationAuthorityRef string    `json:"reconciliation_authority_ref"`
		ReconciliationAttemptRef   string    `json:"reconciliation_attempt_ref"`
		EffectAttemptRef           string    `json:"effect_attempt_ref"`
		ManifestSHA256             string    `json:"manifest_sha256"`
		RequestKeySHA256           string    `json:"request_key_sha256"`
		OriginalRequestSHA256      string    `json:"original_request_sha256"`
		AMVLaunchRef               string    `json:"amv_launch_ref"`
		AMVExecutionRef            string    `json:"amv_execution_ref"`
		AMVRunRef                  string    `json:"amv_run_ref"`
		AMVFence                   uint64    `json:"amv_fence"`
		AMVGeneration              uint64    `json:"amv_generation"`
		AMVCID                     uint32    `json:"amv_cid"`
		AMVIdentitySHA256          string    `json:"amv_identity_sha256"`
		PreparedAt                 time.Time `json:"prepared_at"`
	}{preparation.Binding.ReconciliationAuthorityRef, preparation.Binding.ReconciliationAttemptRef,
		preparation.Binding.EffectAttemptRef, preparation.ManifestSHA256, preparation.RequestKeySHA256,
		preparation.OriginalRequestSHA256, preparation.AMVLaunchRef, preparation.AMVExecutionRef,
		preparation.AMVRunRef, preparation.AMVFence, preparation.AMVGeneration, preparation.AMVCID,
		preparation.AMVIdentitySHA256, preparation.PreparedAt}}, err)
}

func handleConfirmExpiredAgentLaunchContinuation(
	ctx context.Context, api applicationAPI, bound handlerContext, payload json.RawMessage,
) (json.RawMessage, error) {
	var input struct {
		ReconciliationAuthorityRef string `json:"reconciliation_authority_ref"`
		ExpectedManifestSHA256     string `json:"expected_manifest_sha256"`
		Confirmation               string `json:"confirmation"`
	}
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	result, err := api.ConfirmExpiredAgentLaunchContinuationV41(ctx, bound.access,
		application.ConfirmExpiredAgentLaunchContinuationRequestV41{
			RequestRef: bound.requestRef, ReconciliationAuthorityRef: input.ReconciliationAuthorityRef,
			ExpectedManifestSHA256: input.ExpectedManifestSHA256, Confirmation: input.Confirmation,
		})
	record := result.Record
	return marshalApplication(struct {
		Authority struct {
			AuthorityRef               string `json:"authority_ref"`
			SubjectRef                 string `json:"subject_ref"`
			ReconciliationAuthorityRef string `json:"reconciliation_authority_ref"`
			ReconciliationAttemptRef   string `json:"reconciliation_attempt_ref"`
			EffectAttemptRef           string `json:"effect_attempt_ref"`
			ManifestSHA256             string `json:"manifest_sha256"`
			KeyID                      string `json:"key_id"`
			KeyEpoch                   uint64 `json:"key_epoch"`
			TrustRevision              uint64 `json:"trust_revision"`
			IssuedUnixMS               uint64 `json:"issued_unix_ms"`
			ExpiresUnixMS              uint64 `json:"expires_unix_ms"`
		} `json:"authority"`
	}{Authority: struct {
		AuthorityRef               string `json:"authority_ref"`
		SubjectRef                 string `json:"subject_ref"`
		ReconciliationAuthorityRef string `json:"reconciliation_authority_ref"`
		ReconciliationAttemptRef   string `json:"reconciliation_attempt_ref"`
		EffectAttemptRef           string `json:"effect_attempt_ref"`
		ManifestSHA256             string `json:"manifest_sha256"`
		KeyID                      string `json:"key_id"`
		KeyEpoch                   uint64 `json:"key_epoch"`
		TrustRevision              uint64 `json:"trust_revision"`
		IssuedUnixMS               uint64 `json:"issued_unix_ms"`
		ExpiresUnixMS              uint64 `json:"expires_unix_ms"`
	}{record.AuthorityRef, record.SubjectRef, record.ReconciliationAuthorityRef,
		record.ReconciliationAttemptRef, record.EffectAttemptRef, record.ManifestSHA256,
		record.KeyID, record.KeyEpoch, record.TrustRevision, record.IssuedUnixMS,
		record.ExpiresUnixMS}}, err)
}

func handleClaimDirector(ctx context.Context, api applicationAPI, bound handlerContext, payload json.RawMessage) (json.RawMessage, error) {
	var input struct {
		GoalRef string `json:"goal_ref"`
	}
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	ref, err := goal.NewGoalRef(input.GoalRef)
	if err != nil {
		return nil, err
	}
	result, err := api.ClaimDirector(ctx, bound.access, application.ClaimDirectorRequest{RequestRef: bound.requestRef, GoalRef: ref})
	return marshalApplication(struct {
		Lease leaseView `json:"lease"`
	}{projectLease(result.Lease)}, err)
}

func handleRenewDirector(ctx context.Context, api applicationAPI, bound handlerContext, payload json.RawMessage) (json.RawMessage, error) {
	var input struct {
		GoalRef string `json:"goal_ref"`
		Token   string `json:"token"`
		Fence   uint64 `json:"fence"`
	}
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	ref, err := goal.NewGoalRef(input.GoalRef)
	if err != nil {
		return nil, err
	}
	result, err := api.RenewDirector(ctx, bound.access, application.RenewDirectorRequest{RequestRef: bound.requestRef, GoalRef: ref, Token: input.Token, Fence: input.Fence})
	return marshalApplication(struct {
		Lease leaseView `json:"lease"`
	}{projectLease(result.Lease)}, err)
}

func handleProposeDirectorPlan(ctx context.Context, api applicationAPI, bound handlerContext, payload json.RawMessage) (json.RawMessage, error) {
	var input struct {
		GoalRef                  string    `json:"goal_ref"`
		ExpectedGoalRevision     uint64    `json:"expected_goal_revision"`
		ExpectedPlanGeneration   uint64    `json:"expected_plan_generation"`
		LeaseToken               string    `json:"lease_token"`
		LeaseFence               uint64    `json:"lease_fence"`
		Cause                    string    `json:"cause"`
		SourceWorkItemRef        string    `json:"source_work_item_ref"`
		ExpectedWorkItemRevision uint64    `json:"expected_work_item_revision"`
		SourceExecutionRef       string    `json:"source_execution_ref"`
		SourceExecutionAttempt   uint64    `json:"source_execution_attempt"`
		CouncilSubjectDigest     string    `json:"council_subject_digest"`
		CouncilDecisionRef       string    `json:"council_decision_ref"`
		CouncilDecisionDigest    string    `json:"council_decision_digest"`
		Reason                   string    `json:"reason"`
		Plan                     planInput `json:"plan"`
	}
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	goalRef, err := goal.NewGoalRef(input.GoalRef)
	if err != nil {
		return nil, err
	}
	replanFields := []bool{
		input.Cause != "", input.SourceWorkItemRef != "", input.ExpectedWorkItemRevision != 0,
		input.SourceExecutionRef != "", input.SourceExecutionAttempt != 0,
	}
	replanCount := 0
	for _, present := range replanFields {
		if present {
			replanCount++
		}
	}
	if replanCount != 0 && replanCount != len(replanFields) {
		return nil, commandError{code: CodeInvalidRequest, cause: errors.New("commands.director_replan_fence_partial")}
	}
	var workRef goal.WorkItemRef
	var executionRef goal.ExecutionRef
	if replanCount > 0 {
		workRef, err = goal.NewWorkItemRef(input.SourceWorkItemRef)
		if err != nil {
			return nil, err
		}
		executionRef, err = goal.NewExecutionRef(input.SourceExecutionRef)
		if err != nil {
			return nil, err
		}
	}
	plan := applicationPlan(&input.Plan)
	result, err := api.ProposeDirectorPlan(ctx, bound.access, application.ProposeDirectorPlanRequest{RequestRef: bound.requestRef, GoalRef: goalRef, ExpectedGoalRevision: goal.Revision(input.ExpectedGoalRevision), ExpectedPlanGeneration: goal.PlanGeneration(input.ExpectedPlanGeneration), LeaseToken: input.LeaseToken, LeaseFence: input.LeaseFence, Cause: goal.ReplanCause(input.Cause), SourceWorkItemRef: workRef, ExpectedWorkItemRevision: goal.Revision(input.ExpectedWorkItemRevision), SourceExecutionRef: executionRef, SourceExecutionAttempt: input.SourceExecutionAttempt, CouncilSubjectDigest: application.CouncilSubjectDigest(input.CouncilSubjectDigest), CouncilDecisionRef: input.CouncilDecisionRef, CouncilDecisionDigest: application.CouncilSubjectDigest(input.CouncilDecisionDigest), Reason: input.Reason, Plan: *plan})
	return marshalApplication(struct {
		Decision struct {
			Ref                   string `json:"ref"`
			GoalRef               string `json:"goal_ref"`
			LeaseFence            uint64 `json:"lease_fence"`
			AppliedGoalRevision   uint64 `json:"applied_goal_revision"`
			AppliedPlanGeneration uint64 `json:"applied_plan_generation"`
			CouncilSubjectDigest  string `json:"council_subject_digest"`
			CouncilDecisionRef    string `json:"council_decision_ref"`
			CouncilDecisionDigest string `json:"council_decision_digest"`
		} `json:"decision"`
	}{Decision: struct {
		Ref                   string `json:"ref"`
		GoalRef               string `json:"goal_ref"`
		LeaseFence            uint64 `json:"lease_fence"`
		AppliedGoalRevision   uint64 `json:"applied_goal_revision"`
		AppliedPlanGeneration uint64 `json:"applied_plan_generation"`
		CouncilSubjectDigest  string `json:"council_subject_digest"`
		CouncilDecisionRef    string `json:"council_decision_ref"`
		CouncilDecisionDigest string `json:"council_decision_digest"`
	}{result.Decision.Ref, result.Decision.GoalRef.String(), result.Decision.LeaseFence, uint64(result.Decision.AppliedGoalRevision), uint64(result.Decision.AppliedPlanGeneration), string(result.Decision.CouncilSubjectDigest), result.Decision.CouncilDecisionRef, string(result.Decision.CouncilDecisionDigest)}}, err)
}

func handleControl(ctx context.Context, api applicationAPI, bound handlerContext, payload json.RawMessage) (json.RawMessage, error) {
	var input struct {
		Operation                 string `json:"operation"`
		Target                    string `json:"target"`
		GoalRef                   string `json:"goal_ref"`
		ExpectedGoalRevision      uint64 `json:"expected_goal_revision"`
		ExpectedPlanGeneration    uint64 `json:"expected_plan_generation"`
		ExpectedAppSpecGeneration uint64 `json:"expected_app_spec_generation"`
		ExpectedSpecHash          string `json:"expected_spec_hash"`
		WorkItemRef               string `json:"work_item_ref,omitempty"`
		ExpectedWorkItemRevision  uint64 `json:"expected_work_item_revision,omitempty"`
		ExecutionRef              string `json:"execution_ref,omitempty"`
		ExpectedExecutionAttempt  uint64 `json:"expected_execution_attempt,omitempty"`
		Mode                      string `json:"mode,omitempty"`
		Reason                    string `json:"reason"`
	}
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	goalRef, err := goal.NewGoalRef(input.GoalRef)
	if err != nil {
		return nil, err
	}
	var workRef goal.WorkItemRef
	if input.WorkItemRef != "" {
		workRef, err = goal.NewWorkItemRef(input.WorkItemRef)
		if err != nil {
			return nil, err
		}
	}
	var executionRef goal.ExecutionRef
	if input.ExecutionRef != "" {
		executionRef, err = goal.NewExecutionRef(input.ExecutionRef)
		if err != nil {
			return nil, err
		}
	}
	result, err := api.Control(ctx, bound.access, application.ControlRequest{RequestRef: bound.requestRef, Operation: application.ControlOperation(input.Operation), Target: application.ControlTarget(input.Target), GoalRef: goalRef, ExpectedGoalRevision: goal.Revision(input.ExpectedGoalRevision), ExpectedPlanGeneration: goal.PlanGeneration(input.ExpectedPlanGeneration), ExpectedAppSpecGeneration: goal.AppSpecGeneration(input.ExpectedAppSpecGeneration), ExpectedSpecHash: input.ExpectedSpecHash, WorkItemRef: workRef, ExpectedWorkItemRevision: goal.Revision(input.ExpectedWorkItemRevision), ExecutionRef: executionRef, ExpectedExecutionAttempt: input.ExpectedExecutionAttempt, Mode: ports.AgentStopMode(input.Mode), Reason: input.Reason})
	return marshalApplication(struct {
		ControlRef string `json:"control_ref"`
	}{result.Control.Ref}, err)
}

func handleDecideEffect(ctx context.Context, api applicationAPI, bound handlerContext, payload json.RawMessage) (json.RawMessage, error) {
	var input struct {
		GoalRef              string `json:"goal_ref"`
		IntentRef            string `json:"intent_ref"`
		ExpectedIntentDigest string `json:"expected_intent_digest"`
		Decision             string `json:"decision"`
		Reason               string `json:"reason"`
	}
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	ref, err := goal.NewGoalRef(input.GoalRef)
	if err != nil {
		return nil, err
	}
	result, err := api.DecideEffect(ctx, bound.access, application.DecideEffectRequest{RequestRef: bound.requestRef, GoalRef: ref, IntentRef: input.IntentRef, ExpectedIntentDigest: input.ExpectedIntentDigest, Decision: application.EffectDecision(input.Decision), Reason: input.Reason})
	return marshalApplication(struct {
		ApprovalRef string `json:"approval_ref"`
		IntentRef   string `json:"intent_ref"`
		Decision    string `json:"decision"`
	}{result.Approval.Ref, result.Approval.IntentRef, string(result.Approval.Decision)}, err)
}

func handleReconcileTerminalAgentLaunch(
	ctx context.Context,
	api applicationAPI,
	bound handlerContext,
	payload json.RawMessage,
) (json.RawMessage, error) {
	var input struct {
		GoalRef            string `json:"goal_ref"`
		WorkItemRef        string `json:"work_item_ref"`
		ExecutionRef       string `json:"execution_ref"`
		ActionRef          string `json:"action_ref"`
		EffectIntentRef    string `json:"effect_intent_ref"`
		EffectIntentDigest string `json:"effect_intent_digest"`
		EffectAttemptRef   string `json:"effect_attempt_ref"`
		PlanGeneration     uint64 `json:"plan_generation"`
		WorkItemGeneration uint64 `json:"work_item_generation"`
		ActionFence        uint64 `json:"action_fence"`
	}
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	goalRef, err := goal.NewGoalRef(input.GoalRef)
	if err != nil {
		return nil, err
	}
	workItemRef, err := goal.NewWorkItemRef(input.WorkItemRef)
	if err != nil {
		return nil, err
	}
	executionRef, err := goal.NewExecutionRef(input.ExecutionRef)
	if err != nil {
		return nil, err
	}
	result, err := api.ReconcileTerminalAgentLaunch(
		ctx,
		bound.access,
		application.ReconcileTerminalAgentLaunchRequest{
			RequestRef: bound.requestRef, GoalRef: goalRef, WorkItemRef: workItemRef,
			ExecutionRef: executionRef, ActionRef: input.ActionRef,
			EffectIntentRef: input.EffectIntentRef, EffectIntentDigest: input.EffectIntentDigest,
			EffectAttemptRef: input.EffectAttemptRef, PlanGeneration: goal.PlanGeneration(input.PlanGeneration),
			WorkItemGeneration: goal.Revision(input.WorkItemGeneration), ActionFence: input.ActionFence,
		},
	)
	return marshalApplication(struct {
		AuthorityRef string `json:"authority_ref"`
		JobRef       string `json:"job_ref"`
	}{result.Authority.Ref, result.Authority.JobRef}, err)
}

func handleListPendingChanges(ctx context.Context, api applicationAPI, bound handlerContext, payload json.RawMessage) (json.RawMessage, error) {
	var input struct {
		Limit int `json:"limit"`
	}
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	result, err := api.ListPendingChanges(ctx, bound.access, application.ListPendingChangesRequest{Limit: input.Limit})
	changes := make([]changeView, 0, len(result.Changes))
	for _, change := range result.Changes {
		changes = append(changes, projectChange(change))
	}
	return marshalApplication(struct {
		Changes []changeView `json:"changes"`
	}{changes}, err)
}

func handleIntegrateChange(ctx context.Context, api applicationAPI, bound handlerContext, payload json.RawMessage) (json.RawMessage, error) {
	var input struct {
		GoalRef           string `json:"goal_ref"`
		ChangeRef         string `json:"change_ref"`
		ExpectedTargetOID string `json:"expected_target_oid"`
	}
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	goalRef, err := goal.NewGoalRef(input.GoalRef)
	if err != nil {
		return nil, err
	}
	changeRef, err := ports.NewChangeSetRef(input.ChangeRef)
	if err != nil {
		return nil, err
	}
	result, err := api.IntegrateChange(ctx, bound.access, application.IntegrateChangeRequest{RequestRef: bound.requestRef, GoalRef: goalRef, ChangeRef: changeRef, ExpectedTargetOID: input.ExpectedTargetOID})
	return marshalApplication(struct {
		ActionRef string `json:"action_ref"`
	}{result.Action.Ref}, err)
}
