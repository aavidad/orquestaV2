package bootstrap

import (
	"context"
	"encoding/json"
	"orquesta/internal/adapters/auth/executiontoken"
	"orquesta/internal/application"
	commandcore "orquesta/internal/commands"
	"orquesta/internal/credentials"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"sync/atomic"
	"testing"
)

func TestProductionBuildCreatesDurableExecutionAuthorityResolver(t *testing.T) {
	root := t.TempDir()
	configPath := writeTestConfig(t, root)
	runtime := buildExecutionAuthorityRuntime(t, configPath)
	principal, hierarchy, err := localIdentityComposition(runtime.config)
	v22NoError(t, err)
	created := runtime.dispatcher.Dispatch(context.Background(), commandcore.Invocation{
		CommandID: "orquesta.goals.create", CommandVersion: "1",
		RequestRef: "request:v22-production-authority:create",
		ProjectRef: hierarchy.ProjectRef().String(), Principal: principal,
		Payload: json.RawMessage(`{"statement":"hold one execution for durable authority","confirm":true}`),
	})
	if created.Failure != nil {
		t.Fatalf("create Goal: %+v", created.Failure)
	}
	var createdData struct {
		Goal struct {
			GoalRef string `json:"goal_ref"`
		} `json:"goal"`
	}
	if err := json.Unmarshal(created.Data, &createdData); err != nil {
		t.Fatal(err)
	}
	goalRef, err := goal.NewGoalRef(createdData.Goal.GoalRef)
	v22NoError(t, err)
	if result, err := runtime.orchestrator.ProcessNext(
		context.Background(), "worker:v22-production-authority",
	); err != nil || !result.Processed {
		t.Fatalf("launch execution result=%+v err=%v", result, err)
	}
	access, err := application.NewAccess(principal, hierarchy.ProjectRef())
	v22NoError(t, err)
	record, err := runtime.orchestrator.GetGoal(context.Background(), access, goalRef)
	if err != nil || len(record.Executions) != 1 {
		t.Fatalf("running Goal executions=%d err=%v", len(record.Executions), err)
	}
	execution := record.Executions[0]
	if execution.State != application.ExecutionRunning {
		t.Fatalf("execution state=%s want running", execution.State)
	}
	assertExecutionBoundDispatcher(t, runtime, record, execution, "before-restart")
	shutdownRuntime(t, runtime)
	restarted := buildExecutionAuthorityRuntime(t, configPath)
	t.Cleanup(func() { shutdownRuntime(t, restarted) })
	restartedRecord, err := restarted.repository.GetGoal(context.Background(), goalRef)
	if err != nil || len(restartedRecord.Executions) != 1 {
		t.Fatalf("restart Goal executions=%d err=%v", len(restartedRecord.Executions), err)
	}
	assertExecutionBoundDispatcher(
		t, restarted, restartedRecord, restartedRecord.Executions[0], "after-restart",
	)
}

func buildExecutionAuthorityRuntime(t *testing.T, configPath string) *Runtime {
	t.Helper()
	runtime, err := Build(context.Background(), Options{
		ConfigPath: configPath, Version: "v22-production-authority",
		AgentFactory: countingFactory(new(atomic.Int64)),
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return runtime
}

func assertExecutionBoundDispatcher(
	t *testing.T,
	runtime *Runtime,
	record application.GoalRecord,
	execution application.ExecutionRecord,
	suffix string,
) {
	t.Helper()
	store, ok := any(runtime.credentialStore).(credentials.Store)
	if !ok {
		t.Fatalf("runtime credential store does not expose credentials.Store: %T", runtime.credentialStore)
	}
	broker, err := executiontoken.New(store, runtime.repository)
	v22NoError(t, err)
	authority, err := runtime.repository.ExecutionSessionAuthority(
		context.Background(), execution.Ref, executiontoken.AuthenticationMethod,
	)
	v22NoError(t, err)
	var service identity.Principal
	err = broker.UseToken(
		context.Background(), authority.Request,
		func(material []byte) error {
			credential, err := identity.NewCredential(material)
			if err != nil {
				return err
			}
			service, err = broker.Authenticate(context.Background(), credential)
			return err
		},
	)
	if err != nil || service != authority.ServicePrincipal {
		t.Fatalf("execution authentication principal=%+v err=%v", service, err)
	}
	payload, err := json.Marshal(map[string]any{
		"goal_ref":                record.Goal.Ref().String(),
		"recipient_work_item_ref": execution.WorkItemRef.String(),
		"limit":                   1,
	})
	v22NoError(t, err)
	result := runtime.dispatcher.Dispatch(context.Background(), commandcore.Invocation{
		CommandID: "orquesta.mailbox.list", CommandVersion: "1",
		RequestRef: "request:v22-production-authority:" + suffix,
		ProjectRef: record.Goal.Project().String(), Principal: service,
		ClaimedExecutionRef: execution.Ref.String(), Payload: payload,
	})
	if result.Failure != nil || result.AuditRef == "" {
		t.Fatalf("execution-bound dispatch result=%+v", result)
	}
	var output struct {
		Messages []json.RawMessage `json:"messages"`
	}
	if err := json.Unmarshal(result.Data, &output); err != nil || output.Messages == nil {
		t.Fatalf("execution-bound output=%s err=%v", result.Data, err)
	}
}
