package orquestaappcodexstack

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
)

type CodexStackAutoprogrammingExecutorV0 struct {
	Stack *StackV0
}

func NewCodexStackAutoprogrammingExecutorV0(
	stack *StackV0,
) CodexStackAutoprogrammingExecutorV0 {
	return CodexStackAutoprogrammingExecutorV0{Stack: stack}
}

func (executor CodexStackAutoprogrammingExecutorV0) Execute(
	ctx context.Context,
	request AutoprogrammingBridgeRequestV0,
) (AutoprogrammingBridgeResultV0, error) {
	if executor.Stack == nil {
		return AutoprogrammingBridgeResultV0{}, fmt.Errorf("stack requerido")
	}
	return PrepareAutoprogrammingRunFromStackV0(ctx, *executor.Stack, request)
}

func PrepareAutoprogrammingRunFromStackV0(
	ctx context.Context,
	stack StackV0,
	request AutoprogrammingBridgeRequestV0,
) (AutoprogrammingBridgeResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var authorityIssues []orquestaautoprogramming.AutoprogrammingRequestIssueV0
	request, authorityIssues = autoprogrammingBridgeRequestFromEnvelopeAuthorityV0(request)
	if len(authorityIssues) != 0 {
		return AutoprogrammingBridgeResultV0{SchemaVersion: AutoprogrammingBridgeResultSchemaVersionV0, Issues: authorityIssues}, nil
	}
	request = normalizeAutoprogrammingBridgeRequestV0(request)
	// Canonicalization also creates an independent deep projection of nested
	// task slices, so backend marker insertion cannot mutate the captured input.
	callerRequest, callerRequestIssues := orquestaautoprogramming.CanonicalAutoprogrammingIntentRequestV0(request.Request)
	var callerEnvelope *orquestaautoprogramming.AutoprogrammingPrepareRunEnvelopeV0
	if request.PrepareRunEnvelope != nil {
		copyEnvelope := *request.PrepareRunEnvelope
		copyEnvelope.AutoprogrammingRequest = callerRequest
		callerEnvelope = &copyEnvelope
		runtimeRequest, runtimeRequestIssues := orquestaautoprogramming.CanonicalAutoprogrammingIntentRequestV0(callerRequest)
		if len(runtimeRequestIssues) != 0 {
			return AutoprogrammingBridgeResultV0{SchemaVersion: AutoprogrammingBridgeResultSchemaVersionV0, Issues: runtimeRequestIssues}, nil
		}
		request.Request = runtimeRequest
	}
	// The receipt is evidence for the caller command, not mutable runtime input.
	request.PrepareRunEnvelope = nil
	legacyRequested := autoprogrammingBridgeLegacyDirectorLoopRequestedV0(request)
	request.LegacyDirectorLoopOptInAvailable = request.LegacyDirectorLoopOptInAvailable ||
		stack.AllowLegacyAutoprogrammingRun ||
		request.AllowLegacyDirectorLoop
	if stack.AllowLegacyAutoprogrammingRun &&
		autoprogrammingBridgeLegacyDirectorLoopRequestedV0(request) {
		request.AllowLegacyDirectorLoop = true
	}
	if err := validateAutoprogrammingBridgePortsV0(stack.Ports); err != nil {
		return AutoprogrammingBridgeResultV0{}, err
	}
	request = autoprogrammingBridgeRequestWithGoalFirstBackendMarkersV0(request, stack.Ports)
	work := orquestaautoprogramming.BuildAutoprogrammingProgrammableWorkV0(request.Request)
	if work.Accepted && len(work.Work.GoalSpecs) > 0 && !legacyRequested {
		if len(callerRequestIssues) > 0 {
			return AutoprogrammingBridgeResultV0{SchemaVersion: AutoprogrammingBridgeResultSchemaVersionV0, Issues: callerRequestIssues}, nil
		}
		intentManifest, issues, err := persistAutoprogrammingPrepareRunAuthorityV0(ctx, stack.Stores, callerRequest, callerEnvelope)
		if err != nil {
			return AutoprogrammingBridgeResultV0{}, err
		}
		if len(issues) != 0 {
			return AutoprogrammingBridgeResultV0{SchemaVersion: AutoprogrammingBridgeResultSchemaVersionV0, Issues: issues}, nil
		}
		for i := range work.Work.GoalSpecs {
			work.Work.GoalSpecs[i].IntentManifestRef = intentManifest.ManifestRef
			work.Work.GoalSpecs[i].IntentManifestSHA256 = intentManifest.RequestSHA256
		}
	}
	if work.Accepted {
		snapshotStore := stack.AutoprogrammingPromotion.GoalFirstSnapshotStore
		if len(work.Work.GoalSpecs) > 1 && stack.AutoprogrammingPromotion.GoalWorkspaceProvisioner == nil {
			work.Accepted = false
			work.Issues = append(work.Issues, orquestaautoprogramming.AutoprogrammingRequestIssueV0{
				Code:    "physical_goal_workspace_required",
				Field:   "autoprogramming_promotion.goal_workspace_provisioner",
				Message: "los goals paralelos requieren una worktree fisica independiente antes del launch",
			})
		}
		if stack.AutoprogrammingPromotion.Enabled {
			if snapshotStore == nil {
				work.Accepted = false
				work.Issues = append(work.Issues, orquestaautoprogramming.AutoprogrammingRequestIssueV0{
					Code:    "worktree_baseline_store_missing",
					Field:   "autoprogramming_promotion.goal_first_snapshot_store",
					Message: "snapshot store requerido antes de preparar un goal promocionable",
				})
			}
		}
		if work.Accepted {
			prepared, issues := autoprogrammingPrepareWorktreeIsolationV0(
				ctx,
				stack.autoprogrammingCanonicalWorkDirV0(),
				work.Work,
				snapshotStore,
				stack.AutoprogrammingPromotion.GoalWorkspaceProvisioner,
				stack.AutoprogrammingPromotion.GoalWorkspaceRoot,
			)
			if len(issues) > 0 {
				work.Accepted = false
				work.Issues = append(work.Issues, issues...)
			}
			work.Work = prepared
		}
		if work.Accepted && len(work.Work.GoalSpecs) > 1 {
			prepared, batch, issues := stack.prepareAutoprogrammingBatchV0(ctx, work.Work)
			work.Work = prepared
			if len(issues) > 0 {
				work.Accepted = false
				work.Issues = append(work.Issues, issues...)
			} else {
				stack.Ports.GoalLauncher = autoprogrammingBatchGoalLauncherV0{
					Delegate: stack.Ports.GoalLauncher,
					Store:    stack.Stores.AutoprogrammingBatchStore,
					BatchRef: batch.BatchRef,
				}
			}
		}
	}
	return prepareAutoprogrammingRunWithWorkV0(ctx, request, stack.Ports, work)
}

func persistAutoprogrammingPrepareRunAuthorityV0(
	ctx context.Context,
	stores StoresV0,
	callerRequest orquestaautoprogramming.AutoprogrammingRequestV0,
	callerEnvelope *orquestaautoprogramming.AutoprogrammingPrepareRunEnvelopeV0,
) (orquestaautoprogramming.AutoprogrammingIntentManifestV0, []orquestaautoprogramming.AutoprogrammingRequestIssueV0, error) {
	if stores.AutoprogrammingIntentManifestStore == nil {
		return orquestaautoprogramming.AutoprogrammingIntentManifestV0{}, []orquestaautoprogramming.AutoprogrammingRequestIssueV0{{Code: "intent_manifest_store_missing", Field: "autoprogramming_intent_manifest_store", Message: "intent_manifest_store_missing"}}, nil
	}
	if callerEnvelope == nil {
		expected, issues := orquestaautoprogramming.BuildAutoprogrammingIntentManifestV0(callerRequest)
		if len(issues) != 0 {
			return orquestaautoprogramming.AutoprogrammingIntentManifestV0{}, issues, nil
		}
		stored, err := stores.AutoprogrammingIntentManifestStore.CreateAutoprogrammingIntentManifestIfAbsentV0(ctx, expected)
		if err != nil {
			return orquestaautoprogramming.AutoprogrammingIntentManifestV0{}, nil, err
		}
		if issues := validateStoredAutoprogrammingIntentManifestV0(stored, expected); len(issues) != 0 {
			return orquestaautoprogramming.AutoprogrammingIntentManifestV0{}, issues, nil
		}
		return stored, nil, nil
	}
	claimStore := stores.AutoprogrammingPrepareRunIdempotencyClaimStore
	if claimStore == nil {
		claimStore, _ = stores.AutoprogrammingIntentManifestStore.(orquestaautoprogramming.AutoprogrammingPrepareRunIdempotencyClaimStorePortV0)
	}
	if claimStore == nil {
		return orquestaautoprogramming.AutoprogrammingIntentManifestV0{}, []orquestaautoprogramming.AutoprogrammingRequestIssueV0{{Code: "prepare_run_idempotency_claim_store_missing", Field: "autoprogramming_prepare_run_idempotency_claim_store", Message: "prepare_run_idempotency_claim_store_missing"}}, nil
	}
	commandManifest, issues := orquestaautoprogramming.BuildAutoprogrammingIntentManifestFromPrepareRunEnvelopeV0(*callerEnvelope)
	if len(issues) != 0 {
		return orquestaautoprogramming.AutoprogrammingIntentManifestV0{}, issues, nil
	}
	effectiveManifest, alreadyStored, issues, err := resolveAutoprogrammingPrepareRunEffectiveManifestV0(
		ctx,
		stores.AutoprogrammingIntentManifestStore,
		commandManifest,
	)
	if err != nil {
		return orquestaautoprogramming.AutoprogrammingIntentManifestV0{}, nil, err
	}
	if len(issues) != 0 {
		return orquestaautoprogramming.AutoprogrammingIntentManifestV0{}, issues, nil
	}
	claim, issues := orquestaautoprogramming.BuildAutoprogrammingPrepareRunIdempotencyClaimV0(
		callerEnvelope.IdempotencyKey,
		commandManifest,
		effectiveManifest,
	)
	if len(issues) != 0 {
		return orquestaautoprogramming.AutoprogrammingIntentManifestV0{}, issues, nil
	}
	storedClaim, err := claimStore.CreateAutoprogrammingPrepareRunIdempotencyClaimIfAbsentV0(ctx, claim)
	if err != nil {
		code := "prepare_run_idempotency_claim_unavailable"
		if errors.Is(err, orquestaautoprogramming.ErrAutoprogrammingPrepareRunIdempotencyClaimConflictV0) {
			code = "prepare_run_idempotency_claim_conflict"
		}
		return orquestaautoprogramming.AutoprogrammingIntentManifestV0{}, []orquestaautoprogramming.AutoprogrammingRequestIssueV0{{Code: code, Field: "idempotency_key", Message: code}}, nil
	}
	if !orquestaautoprogramming.EqualAutoprogrammingPrepareRunIdempotencyClaimV0(storedClaim, claim) {
		return orquestaautoprogramming.AutoprogrammingIntentManifestV0{}, []orquestaautoprogramming.AutoprogrammingRequestIssueV0{{Code: "prepare_run_idempotency_claim_substitution", Field: "autoprogramming_prepare_run_idempotency_claim_store", Message: "prepare_run_idempotency_claim_substitution"}}, nil
	}
	if alreadyStored {
		return effectiveManifest, nil, nil
	}
	stored, err := stores.AutoprogrammingIntentManifestStore.CreateAutoprogrammingIntentManifestIfAbsentV0(ctx, commandManifest)
	if err != nil {
		return orquestaautoprogramming.AutoprogrammingIntentManifestV0{}, nil, err
	}
	if issues := validateStoredAutoprogrammingIntentManifestV0(stored, commandManifest); len(issues) != 0 {
		return orquestaautoprogramming.AutoprogrammingIntentManifestV0{}, issues, nil
	}
	return stored, nil, nil
}

func resolveAutoprogrammingPrepareRunEffectiveManifestV0(
	ctx context.Context,
	store orquestaautoprogramming.AutoprogrammingIntentManifestStorePortV0,
	commandManifest orquestaautoprogramming.AutoprogrammingIntentManifestV0,
) (orquestaautoprogramming.AutoprogrammingIntentManifestV0, bool, []orquestaautoprogramming.AutoprogrammingRequestIssueV0, error) {
	existing, err := store.LoadAutoprogrammingIntentManifestV0(ctx, commandManifest.RequestRef)
	if errors.Is(err, orquestaautoprogramming.ErrAutoprogrammingIntentManifestNotFoundV0) {
		return commandManifest, false, nil, nil
	}
	if err != nil {
		return orquestaautoprogramming.AutoprogrammingIntentManifestV0{}, false, nil, err
	}
	if issues := orquestaautoprogramming.ValidateAutoprogrammingIntentManifestV0(existing); len(issues) != 0 {
		return orquestaautoprogramming.AutoprogrammingIntentManifestV0{}, false, issues, nil
	}
	if existing.RequestSHA256 == commandManifest.RequestSHA256 && bytes.Equal(existing.RequestJSON, commandManifest.RequestJSON) {
		return existing, true, nil, nil
	}
	return orquestaautoprogramming.AutoprogrammingIntentManifestV0{}, false, []orquestaautoprogramming.AutoprogrammingRequestIssueV0{{Code: "intent_manifest_legacy_request_conflict", Field: "autoprogramming_request", Message: "intent_manifest_legacy_request_conflict"}}, nil
}

func validateStoredAutoprogrammingIntentManifestV0(
	stored orquestaautoprogramming.AutoprogrammingIntentManifestV0,
	expected orquestaautoprogramming.AutoprogrammingIntentManifestV0,
) []orquestaautoprogramming.AutoprogrammingRequestIssueV0 {
	if issues := orquestaautoprogramming.ValidateAutoprogrammingIntentManifestV0(stored); len(issues) != 0 {
		return issues
	}
	if stored.ManifestRef != expected.ManifestRef || stored.RequestRef != expected.RequestRef || stored.RequestSHA256 != expected.RequestSHA256 || !bytes.Equal(stored.RequestJSON, expected.RequestJSON) {
		return []orquestaautoprogramming.AutoprogrammingRequestIssueV0{{Code: "intent_manifest_store_substitution", Field: "autoprogramming_intent_manifest_store", Message: "intent_manifest_store_substitution"}}
	}
	return nil
}
