package identity

import (
	"context"
	"testing"

	"orquesta/internal/goal"
)

func TestLocalOwnerProviderReturnsStableOpaqueRefs(t *testing.T) {
	actorRef, err := goal.NewActorRef("actor:local-owner")
	if err != nil {
		t.Fatalf("NewActorRef() error = %v", err)
	}
	projectRef, err := goal.NewProjectRef("project:default")
	if err != nil {
		t.Fatalf("NewProjectRef() error = %v", err)
	}
	provider, err := NewLocalOwnerProvider(actorRef, projectRef)
	if err != nil {
		t.Fatalf("NewLocalOwnerProvider() error = %v", err)
	}

	first, err := provider.Principal(context.Background())
	if err != nil {
		t.Fatalf("Principal() error = %v", err)
	}
	second, err := provider.Principal(context.Background())
	if err != nil {
		t.Fatalf("Principal() second error = %v", err)
	}
	if first != second {
		t.Fatalf("principal changed: first=%+v second=%+v", first, second)
	}
	if first.ActorRef.String() != "actor:local-owner" || first.DefaultProjectRef.String() != "project:default" {
		t.Fatalf("unexpected principal: %+v", first)
	}
	if first.Method != LocalOwnerMethod {
		t.Fatalf("method = %q", first.Method)
	}
}

func TestLocalOwnerProviderRejectsMissingRefs(t *testing.T) {
	actorRef, _ := goal.NewActorRef("actor:local-owner")
	projectRef, _ := goal.NewProjectRef("project:default")

	if _, err := NewLocalOwnerProvider(goal.ActorRef{}, projectRef); err == nil {
		t.Fatal("missing actor ref accepted")
	}
	if _, err := NewLocalOwnerProvider(actorRef, goal.ProjectRef{}); err == nil {
		t.Fatal("missing project ref accepted")
	}
	var provider *LocalOwnerProvider
	if _, err := provider.Principal(context.Background()); err == nil {
		t.Fatal("nil provider accepted")
	}
}
