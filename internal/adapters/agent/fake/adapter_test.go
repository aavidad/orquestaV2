package fake

import (
	"context"
	"testing"
	"time"

	"orquesta/internal/goal"
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
	observation, err := adapter.Observe(context.Background(), request.ExecutionRef)
	if err != nil {
		t.Fatalf("Observe() error = %v", err)
	}
	if err := ports.ValidateAgentObservation(observation, request.MaxOutputBytes); err != nil {
		t.Fatalf("observation contract error = %v", err)
	}
}

func validRequest(t *testing.T) ports.AgentLaunchRequest {
	t.Helper()
	executionRef, _ := goal.NewExecutionRef("execution:fake")
	goalRef, _ := goal.NewGoalRef("goal:fake")
	workItemRef, _ := goal.NewWorkItemRef("work:fake")
	actorRef, _ := goal.NewActorRef("actor:local-owner")
	projectRef, _ := goal.NewProjectRef("project:default")
	return ports.AgentLaunchRequest{
		ExecutionRef:      executionRef,
		GoalRef:           goalRef,
		WorkItemRef:       workItemRef,
		ActorRef:          actorRef,
		ProjectRef:        projectRef,
		Objective:         "produce artifact",
		ArtifactMediaType: "text/markdown",
		IdempotencyKey:    "launch:fake",
		MaxOutputBytes:    1024,
	}
}
