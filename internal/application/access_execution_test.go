package application

import (
	"errors"
	"testing"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

func TestAccessWithoutExecutionBindingRemainsValid(t *testing.T) {
	principal, projectRef, executionRef := executionAccessFixture(t)
	access, err := NewAccess(principal, projectRef)
	if err != nil {
		t.Fatalf("NewAccess() error=%v", err)
	}
	gotPrincipal, gotProject, err := access.values()
	if err != nil || gotPrincipal != principal || gotProject != projectRef {
		t.Fatalf("normal access values=%+v/%s err=%v", gotPrincipal, gotProject.String(), err)
	}
	if _, err = access.authenticatedExecution(executionRef); !errors.Is(err, errForbidden) {
		t.Fatalf("unbound access authenticated execution: %v", err)
	}
}

func TestExecutionAccessReturnsOnlyExactAuthenticatedExecution(t *testing.T) {
	principal, projectRef, executionRef := executionAccessFixture(t)
	access, err := NewExecutionAccess(principal, projectRef, executionRef)
	if err != nil {
		t.Fatalf("NewExecutionAccess() error=%v", err)
	}
	got, err := access.authenticatedExecution(executionRef)
	if err != nil || got != executionRef {
		t.Fatalf("authenticated execution=%s err=%v", got.String(), err)
	}
	gotPrincipal, gotProject, err := access.values()
	if err != nil || gotPrincipal != principal || gotProject != projectRef {
		t.Fatalf("execution access changed identity/scope: %+v/%s err=%v", gotPrincipal, gotProject.String(), err)
	}
}

func TestExecutionAccessRejectsMissingAndSuccessorExecution(t *testing.T) {
	principal, projectRef, executionRef := executionAccessFixture(t)
	if _, err := NewExecutionAccess(principal, projectRef, goal.ExecutionRef{}); err == nil ||
		err.Error() != "application.execution_ref_required" {
		t.Fatalf("missing execution error=%v", err)
	}
	access, err := NewExecutionAccess(principal, projectRef, executionRef)
	if err != nil {
		t.Fatal(err)
	}
	successor, err := goal.NewExecutionRef("execution:access-successor")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = access.authenticatedExecution(successor); !errors.Is(err, errForbidden) {
		t.Fatalf("successor execution accepted: %v", err)
	}
	if _, err = access.authenticatedExecution(goal.ExecutionRef{}); !errors.Is(err, errForbidden) {
		t.Fatalf("empty expected execution accepted: %v", err)
	}
}

func executionAccessFixture(t *testing.T) (identity.Principal, goal.ProjectRef, goal.ExecutionRef) {
	t.Helper()
	principalRef, err := identity.NewPrincipalRef("principal:execution-access")
	if err != nil {
		t.Fatal(err)
	}
	actorRef, err := goal.NewActorRef("actor:execution-access")
	if err != nil {
		t.Fatal(err)
	}
	principal, err := identity.NewPrincipal(
		principalRef, actorRef, identity.PrincipalKindService, "test",
	)
	if err != nil {
		t.Fatal(err)
	}
	projectRef, err := goal.NewProjectRef("project:execution-access")
	if err != nil {
		t.Fatal(err)
	}
	executionRef, err := goal.NewExecutionRef("execution:access-current")
	if err != nil {
		t.Fatal(err)
	}
	return principal, projectRef, executionRef
}
