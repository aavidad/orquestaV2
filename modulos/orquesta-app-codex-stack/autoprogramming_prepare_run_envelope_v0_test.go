package orquestaappcodexstack

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

type prepareRunManifestLoadStoreForTestV0 struct {
	loaded      orquestaautoprogramming.AutoprogrammingIntentManifestV0
	loadErr     error
	createCalls int
}

func (store *prepareRunManifestLoadStoreForTestV0) CreateAutoprogrammingIntentManifestIfAbsentV0(_ context.Context, manifest orquestaautoprogramming.AutoprogrammingIntentManifestV0) (orquestaautoprogramming.AutoprogrammingIntentManifestV0, error) {
	store.createCalls++
	return manifest, nil
}

func (store *prepareRunManifestLoadStoreForTestV0) LoadAutoprogrammingIntentManifestV0(context.Context, string) (orquestaautoprogramming.AutoprogrammingIntentManifestV0, error) {
	return store.loaded, store.loadErr
}

type prepareRunClaimStoreForTestV0 struct {
	calls    int
	last     orquestaautoprogramming.AutoprogrammingPrepareRunIdempotencyClaimV0
	delegate orquestaautoprogramming.AutoprogrammingPrepareRunIdempotencyClaimStorePortV0
}

func (store *prepareRunClaimStoreForTestV0) CreateAutoprogrammingPrepareRunIdempotencyClaimIfAbsentV0(ctx context.Context, claim orquestaautoprogramming.AutoprogrammingPrepareRunIdempotencyClaimV0) (orquestaautoprogramming.AutoprogrammingPrepareRunIdempotencyClaimV0, error) {
	store.calls++
	store.last = claim
	if store.delegate != nil {
		return store.delegate.CreateAutoprogrammingPrepareRunIdempotencyClaimIfAbsentV0(ctx, claim)
	}
	return claim, nil
}

func TestCodexStackPrepareRunHTTPV0PersistsEnvelopeGoalReplayAndConflictV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	launcher := &goalFirstQueueLauncherForTestV0{}
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	stack.Ports.GoalLauncher = launcher
	stack.Ports.GoalObserver = &goalFirstQueueObserverForTestV0{}
	stack.Ports.GoalRequiredTestSpecBinder = independentSpecBinderForStackTestV0{}
	stack.Ports.GoalClosureValidator = orquestagoal.DefaultGoalWorkClosureValidatorV0{}
	stack.Ports.GoalStateStore = goalStates

	executor := NewCodexStackAutoprogrammingPrepareRunExecutorV0(
		&stack,
		"2026-07-13T10:00:00Z",
		"default-operator",
		stack.Stores.RunQueue,
		stack.RunQueue,
		stack.Clock,
		stack.Codex.RuntimeWorkDir,
	)
	handler := orquestamcp.NewMCPAutoprogrammingPrepareRunHTTPHandlerV0(executor)
	request := autoprogrammingBridgeRequestForTestV0()
	request.RequestRef = "request-ref-envelope-http-replay-001"
	request.Tasks[0].TaskRef = "task-ref-envelope-http-replay-001"
	input := orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-envelope-http-replay-001",
		CorrelationID:          "corr-envelope-http-replay-001",
		IdempotencyKey:         "idem-envelope-http-replay-001",
		OccurredAt:             "2026-07-13T10:00:00Z",
		RequestedBy:            "operator-ref-envelope-http-001",
		DirectorExecutionMode:  orquestaappdirectorservice.AppDirectorExecutionModeGoalFirstV0,
		AutoprogrammingRequest: request,
		PriorityScore:          57,
	}

	first := postPrepareRunEnvelopeHTTPV0(t, handler, input)
	second := postPrepareRunEnvelopeHTTPV0(t, handler, input)
	if !first.Accepted || !second.Accepted || first.RunRef != request.RequestRef || second.RunRef != first.RunRef || len(launcher.specs) != 1 {
		t.Fatalf("first=%+v second=%+v launches=%d", first, second, len(launcher.specs))
	}
	manifest, err := stack.Stores.AutoprogrammingIntentManifestStore.LoadAutoprogrammingIntentManifestV0(context.Background(), request.RequestRef)
	if err != nil {
		t.Fatal(err)
	}
	var envelope orquestaautoprogramming.AutoprogrammingPrepareRunEnvelopeV0
	if err := json.Unmarshal(manifest.RequestJSON, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.SchemaVersion != orquestaautoprogramming.AutoprogrammingPrepareRunEnvelopeSchemaV0 ||
		envelope.RequestID != input.RequestID || envelope.CorrelationID != input.CorrelationID ||
		envelope.IdempotencyKey != input.IdempotencyKey || envelope.RequestedBy != input.RequestedBy ||
		envelope.DirectorExecutionMode != input.DirectorExecutionMode || envelope.PriorityScore != input.PriorityScore ||
		launcher.specs[0].IntentManifestRef != manifest.ManifestRef || launcher.specs[0].IntentManifestSHA256 != manifest.RequestSHA256 {
		t.Fatalf("envelope=%+v manifest=%+v spec=%+v", envelope, manifest, launcher.specs[0])
	}

	input.PriorityScore++
	conflict := postPrepareRunEnvelopeHTTPV0(t, handler, input)
	if conflict.Accepted || conflict.Estado != orquestamcp.MCPAutoprogrammingPrepareRunEstadoErrorV0 || len(launcher.specs) != 1 {
		t.Fatalf("conflict=%+v launches=%d", conflict, len(launcher.specs))
	}
}

func TestCodexStackPrepareRunHTTPV0PersistsPreBindingLaunchFailureV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	launcher := &goalFirstQueueLauncherForTestV0{
		receipt: orquestagoal.GoalLaunchReceiptV0{
			Status: orquestagoal.GoalStatusInvalidV0,
			Issues: []orquestagoal.GoalWorkIssueV0{{
				Code:  "codex_goal_workspace_unavailable",
				Field: "goal_launcher",
			}},
		},
		err: errors.New("codex_goal_workspace_unavailable: /private/workspace"),
	}
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	stack.Ports.GoalLauncher = launcher
	stack.Ports.GoalObserver = &goalFirstQueueObserverForTestV0{}
	stack.Ports.GoalRequiredTestSpecBinder = independentSpecBinderForStackTestV0{}
	stack.Ports.GoalClosureValidator = orquestagoal.DefaultGoalWorkClosureValidatorV0{}
	stack.Ports.GoalStateStore = goalStates

	executor := NewCodexStackAutoprogrammingPrepareRunExecutorV0(
		&stack,
		"2026-07-13T10:30:00Z",
		"default-operator",
		stack.Stores.RunQueue,
		stack.RunQueue,
		stack.Clock,
		stack.Codex.RuntimeWorkDir,
	)
	handler := orquestamcp.NewMCPAutoprogrammingPrepareRunHTTPHandlerV0(executor)
	request := autoprogrammingBridgeRequestForTestV0()
	request.RequestRef = "request-ref-envelope-pre-binding-failure-001"
	request.Tasks[0].TaskRef = "task-ref-envelope-pre-binding-failure-001"
	input := orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-envelope-pre-binding-failure-001",
		CorrelationID:          "corr-envelope-pre-binding-failure-001",
		IdempotencyKey:         "idem-envelope-pre-binding-failure-001",
		OccurredAt:             "2026-07-13T10:30:00Z",
		RequestedBy:            "operator-ref-envelope-pre-binding-failure-001",
		DirectorExecutionMode:  orquestaappdirectorservice.AppDirectorExecutionModeGoalFirstV0,
		AutoprogrammingRequest: request,
	}

	result := postPrepareRunEnvelopeHTTPV0(t, handler, input)
	if result.Accepted || result.Estado != orquestamcp.MCPAutoprogrammingPrepareRunEstadoErrorV0 ||
		len(result.Errores) != 1 || result.Errores[0].Code != "autoprogramming_goal_launch_failed" ||
		!strings.Contains(result.Errores[0].Message, "codex_goal_workspace_unavailable") ||
		strings.Contains(result.Errores[0].Message, "/private/") {
		t.Fatalf("result=%+v", result)
	}
	state, err := goalStates.LoadGoalWorkStateV0(context.Background(), request.RequestRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if state.Status != orquestagoal.GoalStatusInvalidV0 ||
		state.Spec.IntentManifestRef == "" || state.Spec.IntentManifestSHA256 == "" ||
		state.ExternalGoalRef != "" || state.LaunchReceipt.ExternalGoalRef != "" ||
		state.LaunchReceipt.IntentManifestRef != "" || state.LaunchReceipt.WorkspaceRef != "" ||
		state.LaunchReceipt.ProviderRef != "" || state.LaunchReceipt.RuntimeGenerationRef != "" {
		t.Fatalf("state=%+v", state)
	}
}

func TestAutoprogrammingBridgeGoalLaunchIssueV0KeepsReasonWhenStatePersistenceFailsV0(t *testing.T) {
	reason := autoprogrammingBridgeGoalLaunchFailureReasonV0(orquestagoal.GoalLaunchReceiptV0{
		Issues: []orquestagoal.GoalWorkIssueV0{{
			Code: "codex_goal_workspace_unavailable: /home/operator/private token=secret",
		}},
	}, errors.New("launcher failed at /home/operator/private token=secret"))
	issue := autoprogrammingBridgeGoalLaunchIssueV0(
		"run-ref-launch-and-state-failure-001",
		reason,
		errors.New("state store unavailable at /private/state"),
	)
	if reason != "codex_goal_workspace_unavailable" ||
		!strings.Contains(issue.Message, "codex_goal_workspace_unavailable") ||
		!strings.Contains(issue.Message, "state bloqueado no pudo persistirse") ||
		strings.Contains(issue.Message, "/home/") || strings.Contains(issue.Message, "token=") ||
		strings.Contains(issue.Message, "/private/") {
		t.Fatalf("issue=%+v", issue)
	}
}

func TestAutoprogrammingBridgeRequestFromEnvelopeAuthorityV0IgnoresDivergentLegacyDuplicatesV0(t *testing.T) {
	request, envelope := prepareRunEnvelopeAuthorityFixtureV0()
	duplicate := request
	duplicate.RequestRef = "request-ref-divergent-duplicate"
	bridge, issues := autoprogrammingBridgeRequestFromEnvelopeAuthorityV0(AutoprogrammingBridgeRequestV0{
		Request:               duplicate,
		PrepareRunEnvelope:    &envelope,
		OccurredAt:            "1999-01-01T00:00:00Z",
		CorrelationID:         "corr-divergent-duplicate",
		RequestedBy:           "operator-divergent-duplicate",
		DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
		MaxBursts:             999,
		MaxCommands:           999,
		MaxOutboxPerCycle:     999,
		MaxDispatchesPerWait:  999,
		MaxStepsPerBurst:      999,
	})
	if len(issues) != 0 || bridge.Request.RequestRef != request.RequestRef || bridge.CorrelationID != envelope.CorrelationID || bridge.RequestedBy != envelope.RequestedBy || bridge.DirectorExecutionMode != orquestaappdirectorservice.AppDirectorExecutionModeGoalFirstV0 || bridge.MaxBursts != envelope.MaxBursts || bridge.MaxCommands != envelope.MaxCommands {
		t.Fatalf("bridge=%+v issues=%+v", bridge, issues)
	}
}

func TestPersistAutoprogrammingPrepareRunAuthorityV0DistinguishesNotFoundFromLoadFailuresV0(t *testing.T) {
	request, envelope := prepareRunEnvelopeAuthorityFixtureV0()
	t.Run("not found creates claim then manifest", func(t *testing.T) {
		manifestStore := orquestaautoprogramming.NewInMemoryAutoprogrammingIntentManifestStoreV0()
		claimStore := &prepareRunClaimStoreForTestV0{delegate: manifestStore}
		stored, issues, err := persistAutoprogrammingPrepareRunAuthorityV0(context.Background(), StoresV0{
			AutoprogrammingIntentManifestStore:             manifestStore,
			AutoprogrammingPrepareRunIdempotencyClaimStore: claimStore,
		}, request, &envelope)
		if err != nil || len(issues) != 0 || claimStore.calls != 1 || stored.RequestRef != request.RequestRef {
			t.Fatalf("stored=%+v issues=%+v err=%v claims=%d", stored, issues, err, claimStore.calls)
		}
	})

	for name, setup := range map[string]func() (context.Context, orquestaautoprogramming.AutoprogrammingIntentManifestStorePortV0, error){
		"unavailable": func() (context.Context, orquestaautoprogramming.AutoprogrammingIntentManifestStorePortV0, error) {
			want := errors.New("manifest store unavailable")
			return context.Background(), &prepareRunManifestLoadStoreForTestV0{loadErr: want}, want
		},
		"canceled": func() (context.Context, orquestaautoprogramming.AutoprogrammingIntentManifestStorePortV0, error) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			return ctx, orquestaautoprogramming.NewInMemoryAutoprogrammingIntentManifestStoreV0(), context.Canceled
		},
	} {
		t.Run(name, func(t *testing.T) {
			ctx, manifestStore, wantErr := setup()
			claimStore := &prepareRunClaimStoreForTestV0{}
			_, issues, err := persistAutoprogrammingPrepareRunAuthorityV0(ctx, StoresV0{
				AutoprogrammingIntentManifestStore:             manifestStore,
				AutoprogrammingPrepareRunIdempotencyClaimStore: claimStore,
			}, request, &envelope)
			if !errors.Is(err, wantErr) || len(issues) != 0 || claimStore.calls != 0 {
				t.Fatalf("issues=%+v err=%v claims=%d", issues, err, claimStore.calls)
			}
		})
	}

	t.Run("corrupt", func(t *testing.T) {
		claimStore := &prepareRunClaimStoreForTestV0{}
		_, issues, err := persistAutoprogrammingPrepareRunAuthorityV0(context.Background(), StoresV0{
			AutoprogrammingIntentManifestStore:             &prepareRunManifestLoadStoreForTestV0{},
			AutoprogrammingPrepareRunIdempotencyClaimStore: claimStore,
		}, request, &envelope)
		if err != nil || len(issues) == 0 || claimStore.calls != 0 {
			t.Fatalf("issues=%+v err=%v claims=%d", issues, err, claimStore.calls)
		}
	})
}

func TestPersistAutoprogrammingPrepareRunAuthorityV0RejectsLegacyEvenWhenInnerRequestMatchesV0(t *testing.T) {
	request, envelope := prepareRunEnvelopeAuthorityFixtureV0()
	manifestStore := orquestaautoprogramming.NewInMemoryAutoprogrammingIntentManifestStoreV0()
	legacy, legacyIssues := orquestaautoprogramming.BuildAutoprogrammingIntentManifestV0(request)
	if len(legacyIssues) != 0 {
		t.Fatal(legacyIssues)
	}
	if _, err := manifestStore.CreateAutoprogrammingIntentManifestIfAbsentV0(context.Background(), legacy); err != nil {
		t.Fatal(err)
	}
	claimStore := &prepareRunClaimStoreForTestV0{delegate: manifestStore}
	_, issues, err := persistAutoprogrammingPrepareRunAuthorityV0(context.Background(), StoresV0{
		AutoprogrammingIntentManifestStore:             manifestStore,
		AutoprogrammingPrepareRunIdempotencyClaimStore: claimStore,
	}, request, &envelope)
	if err != nil || !hasAutoprogrammingRequestIssueCodeForEnvelopeTestV0(issues, "intent_manifest_legacy_request_conflict") || claimStore.calls != 0 {
		t.Fatalf("issues=%+v err=%v claims=%d", issues, err, claimStore.calls)
	}
	loaded, err := manifestStore.LoadAutoprogrammingIntentManifestV0(context.Background(), request.RequestRef)
	if err != nil || !bytes.Equal(loaded.RequestJSON, legacy.RequestJSON) {
		t.Fatalf("legacy substituted: loaded=%+v err=%v", loaded, err)
	}
}

func TestPersistAutoprogrammingPrepareRunAuthorityV0SameIdempotencyDifferentRequestConflictsBeforeSecondManifestV0(t *testing.T) {
	request, envelope := prepareRunEnvelopeAuthorityFixtureV0()
	manifestStore := orquestaautoprogramming.NewInMemoryAutoprogrammingIntentManifestStoreV0()
	claimStore := &prepareRunClaimStoreForTestV0{delegate: manifestStore}
	stores := StoresV0{
		AutoprogrammingIntentManifestStore:             manifestStore,
		AutoprogrammingPrepareRunIdempotencyClaimStore: claimStore,
	}
	if _, issues, err := persistAutoprogrammingPrepareRunAuthorityV0(context.Background(), stores, request, &envelope); err != nil || len(issues) != 0 {
		t.Fatalf("first issues=%+v err=%v", issues, err)
	}
	secondRequest := request
	secondRequest.RequestRef = "request-ref-envelope-authority-002"
	secondEnvelope := envelope
	secondEnvelope.RequestID = "request-envelope-authority-002"
	secondEnvelope.AutoprogrammingRequest = secondRequest
	_, issues, err := persistAutoprogrammingPrepareRunAuthorityV0(context.Background(), stores, secondRequest, &secondEnvelope)
	if err != nil || !hasAutoprogrammingRequestIssueCodeForEnvelopeTestV0(issues, "prepare_run_idempotency_claim_conflict") {
		t.Fatalf("second issues=%+v err=%v", issues, err)
	}
	if _, err := manifestStore.LoadAutoprogrammingIntentManifestV0(context.Background(), secondRequest.RequestRef); !errors.Is(err, orquestaautoprogramming.ErrAutoprogrammingIntentManifestNotFoundV0) {
		t.Fatalf("second manifest published before claim conflict: %v", err)
	}
}

func prepareRunEnvelopeAuthorityFixtureV0() (orquestaautoprogramming.AutoprogrammingRequestV0, orquestaautoprogramming.AutoprogrammingPrepareRunEnvelopeV0) {
	request := autoprogrammingBridgeRequestForTestV0()
	request.RequestRef = "request-ref-envelope-authority-001"
	request.Tasks[0].TaskRef = "task-ref-envelope-authority-001"
	return request, orquestaautoprogramming.AutoprogrammingPrepareRunEnvelopeV0{
		RequestID:              "request-envelope-authority-001",
		CorrelationID:          "corr-envelope-authority-001",
		IdempotencyKey:         "idem-envelope-authority-001",
		OccurredAt:             "2026-07-13T10:00:00Z",
		RequestedBy:            "operator-ref-envelope-authority-001",
		DirectorExecutionMode:  orquestaappdirectorservice.AppDirectorExecutionModeGoalFirstV0,
		AutoprogrammingRequest: request,
	}
}

func hasAutoprogrammingRequestIssueCodeForEnvelopeTestV0(issues []orquestaautoprogramming.AutoprogrammingRequestIssueV0, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func postPrepareRunEnvelopeHTTPV0(
	t *testing.T,
	handler http.Handler,
	input orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0,
) orquestamcp.MCPAutoprogrammingPrepareRunToolResultV0 {
	t.Helper()
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, orquestamcp.MCPAutoprogrammingPrepareRunHTTPPathV0, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK && rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPAutoprogrammingPrepareRunToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	return result
}
