package fake

import (
	"context"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

func TestAdapterLaunchIsIdempotentAndObservable(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	adapter, err := New(Config{
		ProviderRef: "provider:fake",
		MediaType:   "text/markdown",
		Content:     []byte("artifact"),
		Now:         func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	capabilities, err := adapter.Capabilities(context.Background())
	if err != nil {
		t.Fatalf("Capabilities() error = %v", err)
	}
	if err := ports.ValidateAgentCapabilities(capabilities); err != nil {
		t.Fatalf("capabilities contract error = %v", err)
	}
	if capabilities.ProviderRef != "provider:fake" || capabilities.ModelRef != ModelRef ||
		capabilities.AgentRef != AgentRef || !capabilities.Unrestricted {
		t.Fatalf("capabilities = %+v", capabilities)
	}
	request := validRequest(t)
	first, err := adapter.Launch(context.Background(), request)
	if err != nil {
		t.Fatalf("Launch() error = %v", err)
	}
	second, err := adapter.Launch(context.Background(), request)
	if err != nil {
		t.Fatalf("Launch() second error = %v", err)
	}
	if first != second {
		t.Fatalf("receipts differ: first=%+v second=%+v", first, second)
	}
	if first.SpecHash != request.SpecHash {
		t.Fatalf("receipt spec hash = %q, want %q", first.SpecHash, request.SpecHash)
	}
	if first.ReceiptRef == "" || first.ReceiptRef != second.ReceiptRef {
		t.Fatalf("unstable launch receipt ref: first=%q second=%q", first.ReceiptRef, second.ReceiptRef)
	}
	if err := ports.ValidateAgentLaunchReceipt(request, first); err != nil {
		t.Fatalf("receipt contract error = %v", err)
	}
	if first.GoalRef != request.GoalRef || first.WorkItemRef != request.WorkItemRef ||
		first.PlanGeneration != request.PlanGeneration || first.AppSpecGeneration != request.AppSpecGeneration ||
		first.ExecutionAttempt != request.ExecutionAttempt || first.ModelRef != ModelRef || first.AgentRef != AgentRef {
		t.Fatalf("receipt lost causal identity: %+v", first)
	}
	conflicting := request
	conflicting.SpecHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if _, err := adapter.Launch(context.Background(), conflicting); err == nil || err.Error() != "fake_agent.execution_conflict" {
		t.Fatalf("spec hash conflict error = %v", err)
	}
	observation, err := adapter.Observe(context.Background(), request.ExecutionRef)
	if err != nil {
		t.Fatalf("Observe() error = %v", err)
	}
	if err := ports.ValidateAgentObservation(observation, request.MaxOutputBytes); err != nil {
		t.Fatalf("observation contract error = %v", err)
	}
	if observation.SpecHash != request.SpecHash {
		t.Fatalf("observation spec hash = %q, want %q", observation.SpecHash, request.SpecHash)
	}
	if observation.Usage.Quality != governance.UsageQualityUnknown || observation.Usage.Known != 0 {
		t.Fatalf("fake invented usage: %+v", observation.Usage)
	}
}

func validRequest(t *testing.T) ports.AgentLaunchRequest {
	t.Helper()
	executionRef, _ := goal.NewExecutionRef("execution:fake")
	goalRef, _ := goal.NewGoalRef("goal:fake")
	workItemRef, _ := goal.NewWorkItemRef("work:fake")
	actorRef, _ := goal.NewActorRef("actor:local-owner")
	projectRef, _ := goal.NewProjectRef("project:default")
	placementRef, _ := ports.NewAgentPlacementRef("placement:fake")
	return ports.AgentLaunchRequest{
		ExecutionRef:         executionRef,
		ReferenciaColocacion: placementRef,
		GoalRef:              goalRef,
		WorkItemRef:          workItemRef,
		PlanGeneration:       2,
		AppSpecGeneration:    3,
		ExecutionAttempt:     1,
		SpecHash:             "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		ActorRef:             actorRef,
		ProjectRef:           projectRef,
		Objective:            "produce artifact",
		PhaseRef:             "phase-instance:build",
		PhaseKey:             "phase:build",
		PhaseTemplateRef:     "phase-template:program",
		PhaseInputRefs:       []string{"input:app-spec"},
		PhaseCriterionRefs:   []string{"criterion:tests-green"},
		RoleKey:              "role:worker",
		WriteSet:             []string{"internal/adapters/agent/fake"},
		OutputContract:       string(goal.OutputContractEvidenceBundle),
		ArtifactMediaType:    "text/markdown",
		IdempotencyKey:       "launch:fake",
		MaxOutputBytes:       1024,
		BudgetDemand: governance.BudgetDemand{
			Ref: "demand:fake", Resources: governance.ResourceVector{
				Tokens: 1_000, ActiveTimeNS: int64(time.Minute), ProcessSlots: 1, DiskBytes: 1024,
			},
		},
		SecurityCriticality: governance.SecurityCriticalityNormal,
		ReasoningEffort:     governance.ReasoningEffortMedium,
	}
}
