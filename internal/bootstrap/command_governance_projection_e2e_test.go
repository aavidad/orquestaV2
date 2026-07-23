package bootstrap

import (
	"context"
	"encoding/json"
	"reflect"
	"sync/atomic"
	"testing"

	"orquesta/internal/commands"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

func TestRuntimeDispatcherPersistsAndProjectsControlClosure(t *testing.T) {
	root := t.TempDir()
	configPath := writeTestConfig(t, root)
	runtime, err := Build(context.Background(), Options{
		ConfigPath: configPath, Version: "governance-projection",
		AgentFactory: countingFactory(new(atomic.Int64)),
	})
	if err != nil {
		t.Fatal(err)
	}
	principal, hierarchy, err := localIdentityComposition(runtime.config)
	if err != nil {
		t.Fatal(err)
	}
	create := dispatchRuntimeCommand(t, runtime, principal, hierarchy.ProjectRef().String(),
		"orquesta.goals.create", "request:projection-create", map[string]any{
			"statement": "create then close one governed goal", "confirm": true,
		})
	var created struct {
		Goal struct {
			GoalRef string `json:"goal_ref"`
		} `json:"goal"`
	}
	if err := json.Unmarshal(create.Data, &created); err != nil || created.Goal.GoalRef == "" {
		t.Fatalf("create=%+v err=%v", create, err)
	}
	createdGoalRef, err := goalRef(created.Goal.GoalRef)
	if err != nil {
		t.Fatal(err)
	}
	record, err := runtime.repository.GetGoal(context.Background(), createdGoalRef)
	if err != nil {
		t.Fatal(err)
	}
	controlPayload := map[string]any{
		"operation": "cancel", "target": "goal", "goal_ref": createdGoalRef.String(),
		"expected_goal_revision":       record.Goal.Revision(),
		"expected_plan_generation":     record.Goal.PlanGeneration(),
		"expected_app_spec_generation": record.Goal.AppSpec().Generation(),
		"expected_spec_hash":           record.Goal.SpecHash(),
		"reason":                       "close the governed projection fixture",
	}
	control := dispatchRuntimeCommand(t, runtime, principal, hierarchy.ProjectRef().String(),
		"orquesta.goals.control", "request:projection-control", controlPayload)
	projection, err := runtime.repository.ProjectCommandGovernance(
		context.Background(), control.AuditRef,
	)
	if err != nil || projection.FactKind != ports.CommandGovernanceClosureControl ||
		projection.FactRef == "" || projection.AuthorizationReceiptRef == "" {
		t.Fatalf("projection=%+v err=%v", projection, err)
	}
	required := map[ports.CommandGovernanceFactKind]bool{
		ports.CommandGovernanceAuthorization:  false,
		ports.CommandGovernanceClosureControl: false,
		ports.CommandGovernanceCausalEvent:    false,
		ports.CommandGovernanceTerminalEvent:  false,
	}
	for _, fact := range projection.Facts {
		if fact.Ref == "" || fact.CausalRef == "" {
			t.Fatalf("incomplete fact=%+v", fact)
		}
		if _, wanted := required[fact.Kind]; wanted {
			required[fact.Kind] = true
		}
	}
	for kind, found := range required {
		if !found {
			t.Fatalf("missing %s: %+v", kind, projection.Facts)
		}
	}
	replay := dispatchRuntimeCommand(t, runtime, principal, hierarchy.ProjectRef().String(),
		"orquesta.goals.control", "request:projection-control", controlPayload)
	afterReplay, err := runtime.repository.GetGoal(context.Background(), createdGoalRef)
	if err != nil || !reflect.DeepEqual(replay, control) || len(afterReplay.Controls) != 1 {
		t.Fatalf("replay=%+v control=%+v controls=%d err=%v",
			replay, control, len(afterReplay.Controls), err)
	}
	shutdownRuntime(t, runtime)

	restarted, err := Build(context.Background(), Options{
		ConfigPath: configPath, Version: "governance-projection",
		AgentFactory: countingFactory(new(atomic.Int64)),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { shutdownRuntime(t, restarted) })
	restartedProjection, err := restarted.repository.ProjectCommandGovernance(
		context.Background(), control.AuditRef,
	)
	if err != nil || !reflect.DeepEqual(restartedProjection, projection) {
		t.Fatalf("restart projection=%+v want=%+v err=%v", restartedProjection, projection, err)
	}
}

func dispatchRuntimeCommand(
	t *testing.T,
	runtime *Runtime,
	principal identity.Principal,
	projectRef, commandID, requestRef string,
	payload any,
) commands.Result {
	t.Helper()
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	result := runtime.dispatcher.Dispatch(context.Background(), commands.Invocation{
		CommandID: commandID, CommandVersion: "1", RequestRef: requestRef,
		ProjectRef: projectRef, Principal: principal, Payload: encoded,
	})
	if result.Failure != nil || result.AuditRef == "" {
		t.Fatalf("dispatch %s=%+v", commandID, result)
	}
	return result
}
