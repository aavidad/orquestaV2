package main

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestServerSupervisorWithCodexGoalBackendV0ExponePuertosSoloConBackendV0(t *testing.T) {
	base := serverStackSupervisorV0{}
	plain := serverSupervisorWithCodexGoalBackendV0(base, serverCodexGoalBackendV0{})
	if _, ok := plain.(orquestaserver.IdleSelfImprovementGoalLauncherPortV0); ok {
		t.Fatalf("supervisor sin backend no debe exponer launcher goal")
	}
	if _, ok := plain.(orquestaserver.IdleSelfImprovementGoalObserverPortV0); ok {
		t.Fatalf("supervisor sin backend no debe exponer observer goal")
	}

	protocol := &fakeCodexAppServerProtocolV0{}
	backend := serverCodexAppServerGoalBackendV0{Protocol: protocol}
	wrapped := serverSupervisorWithCodexGoalBackendV0(base, serverCodexGoalBackendV0{
		Starter:  backend,
		Observer: backend,
	})
	if _, ok := wrapped.(orquestaserver.IdleSelfImprovementGoalLauncherPortV0); !ok {
		t.Fatalf("supervisor con backend debe exponer launcher goal")
	}
	if _, ok := wrapped.(orquestaserver.IdleSelfImprovementGoalObserverPortV0); !ok {
		t.Fatalf("supervisor con backend debe exponer observer goal")
	}
}

func TestServerCodexAppServerGoalBackendV0LanzaThreadGoalYTurnV0(t *testing.T) {
	protocol := &fakeCodexAppServerProtocolV0{
		thread: serverCodexAppServerThreadV0{ID: "thread-ref-goal-001"},
		goal:   serverCodexAppServerThreadGoalV0{ThreadID: "thread-ref-goal-001", Status: "active"},
		turn:   serverCodexAppServerTurnV0{ID: "turn-ref-goal-001", Status: "inProgress"},
	}
	backend := serverCodexAppServerGoalBackendV0{
		Protocol:        protocol,
		CWD:             "/tmp/orquesta-goal-fixture",
		Model:           "gpt-5.4",
		ReasoningEffort: "medium",
		Sandbox:         "workspace-write",
		ApprovalPolicy:  "never",
	}

	receipt, err := backend.StartCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		SchemaVersion: orquestaruntimecodexgoal.CodexGoalStartPacketSchemaV0,
		GoalRef:       "goal-ref-codex-app-server-001",
		Objective:     "programar una mejora acotada",
		Prompt:        "prompt compacto",
		Budget:        orquestagoal.GoalBudgetV0{TokenBudget: 123},
	})

	if err != nil {
		t.Fatalf("StartCodexGoalV0: %v", err)
	}
	if receipt.Status != orquestagoal.GoalStatusRunningV0 ||
		receipt.GoalRef != "goal-ref-codex-app-server-001" ||
		receipt.ExternalGoalRef != "thread-ref-goal-001" {
		t.Fatalf("receipt=%+v", receipt)
	}
	wantCalls := []string{"thread/start", "thread/goal/set", "turn/start"}
	if !reflect.DeepEqual(protocol.calls, wantCalls) {
		t.Fatalf("calls=%v want=%v", protocol.calls, wantCalls)
	}
	if protocol.startParams.Ephemeral ||
		protocol.startParams.CWD != "/tmp/orquesta-goal-fixture" ||
		protocol.startParams.ApprovalPolicy != "never" {
		t.Fatalf("start params=%+v", protocol.startParams)
	}
	if protocol.setParams.ThreadID != "thread-ref-goal-001" ||
		protocol.setParams.Objective != "programar una mejora acotada" ||
		protocol.setParams.Status != "active" ||
		protocol.setParams.TokenBudget != 123 {
		t.Fatalf("set params=%+v", protocol.setParams)
	}
	if protocol.turnParams.ThreadID != "thread-ref-goal-001" ||
		protocol.turnParams.InputText != "prompt compacto" ||
		protocol.turnParams.ClientMessageID != "goal-ref-codex-app-server-001" ||
		protocol.turnParams.Effort != "medium" {
		t.Fatalf("turn params=%+v", protocol.turnParams)
	}
}

func TestServerCodexAppServerGoalBackendV0ObservaGoalPorThreadIDV0(t *testing.T) {
	protocol := &fakeCodexAppServerProtocolV0{
		observedGoal: &serverCodexAppServerThreadGoalV0{
			ThreadID: "thread-ref-goal-002",
			Status:   "complete",
		},
	}
	backend := serverCodexAppServerGoalBackendV0{Protocol: protocol}

	receipt, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		SchemaVersion:   orquestaruntimecodexgoal.CodexGoalObservationRequestSchemaV0,
		GoalRef:         "goal-ref-codex-app-server-002",
		ExternalGoalRef: "thread-ref-goal-002",
	})

	if err != nil {
		t.Fatalf("ObserveCodexGoalV0: %v", err)
	}
	if receipt.Status != orquestagoal.GoalStatusCompleteV0 ||
		receipt.GoalRef != "goal-ref-codex-app-server-002" ||
		receipt.ExternalGoalRef != "thread-ref-goal-002" ||
		!containsStringForTestV0(receipt.EvidenceRefs, "evidence-ref-codex-app-server-goal-observed") {
		t.Fatalf("receipt=%+v", receipt)
	}
	if !reflect.DeepEqual(protocol.calls, []string{"thread/goal/get"}) {
		t.Fatalf("calls=%v", protocol.calls)
	}
	if protocol.getThreadID != "thread-ref-goal-002" {
		t.Fatalf("get thread id=%q", protocol.getThreadID)
	}
}

func TestCodexAppServerRPCPayloadYDecodeV0(t *testing.T) {
	payload, err := codexAppServerRPCPayloadV0("thread/goal/get", map[string]interface{}{"threadId": "thread-ref-003"})
	if err != nil {
		t.Fatalf("payload: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(payload), "\n")
	if len(lines) != 2 {
		t.Fatalf("payload lines=%q", payload)
	}
	var request map[string]interface{}
	if err := json.Unmarshal([]byte(lines[1]), &request); err != nil {
		t.Fatalf("json request: %v", err)
	}
	if request["method"] != "thread/goal/get" || int(request["id"].(float64)) != 2 {
		t.Fatalf("request=%+v", request)
	}

	stdout := []byte(`{"method":"thread/goal/updated","params":{"goal":{"threadId":"thread-ref-003","objective":"x","status":"active","createdAt":1,"updatedAt":2,"tokensUsed":3,"timeUsedSeconds":4}}}` + "\n" +
		`{"id":2,"result":{"goal":{"threadId":"thread-ref-003","objective":"x","status":"active","createdAt":1,"updatedAt":2,"tokensUsed":3,"timeUsedSeconds":4}}}` + "\n")
	var response serverCodexAppServerThreadGoalGetResponseV0
	if err := decodeCodexAppServerRPCResponseV0(stdout, 2, &response); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if response.Goal == nil ||
		response.Goal.ThreadID != "thread-ref-003" ||
		response.Goal.Status != "active" {
		t.Fatalf("response=%+v", response)
	}
}

func TestServerEffectiveConfigV0ExponeGoalFirstYBackendV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envServerIdleSelfImprovementGoalFirstV0, "true")
	t.Setenv(envCodexGoalBackendV0, codexGoalBackendAppServerProxyV0)
	t.Setenv(envCodexGoalTimeoutMSV0, "12000")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	settings := config.EffectiveConfig.Settings
	if got := effectiveSettingValueForTestV0(settings, envServerIdleSelfImprovementGoalFirstV0); got != "true" {
		t.Fatalf("goal-first setting=%q", got)
	}
	if got := effectiveSettingValueForTestV0(settings, envCodexGoalBackendV0); got != codexGoalBackendAppServerProxyV0 {
		t.Fatalf("goal backend=%q", got)
	}
	if got := effectiveSettingValueForTestV0(settings, envCodexGoalTimeoutMSV0); got != "12000" {
		t.Fatalf("goal timeout=%q", got)
	}
}

type fakeCodexAppServerProtocolV0 struct {
	calls       []string
	startParams serverCodexAppServerThreadStartParamsV0
	setParams   serverCodexAppServerThreadGoalSetParamsV0
	turnParams  serverCodexAppServerTurnStartParamsV0
	getThreadID string

	thread       serverCodexAppServerThreadV0
	goal         serverCodexAppServerThreadGoalV0
	turn         serverCodexAppServerTurnV0
	observedGoal *serverCodexAppServerThreadGoalV0
}

func (fake *fakeCodexAppServerProtocolV0) StartThreadV0(
	_ context.Context,
	params serverCodexAppServerThreadStartParamsV0,
) (serverCodexAppServerThreadV0, error) {
	fake.calls = append(fake.calls, "thread/start")
	fake.startParams = params
	return fake.thread, nil
}

func (fake *fakeCodexAppServerProtocolV0) SetGoalV0(
	_ context.Context,
	params serverCodexAppServerThreadGoalSetParamsV0,
) (serverCodexAppServerThreadGoalV0, error) {
	fake.calls = append(fake.calls, "thread/goal/set")
	fake.setParams = params
	return fake.goal, nil
}

func (fake *fakeCodexAppServerProtocolV0) StartTurnV0(
	_ context.Context,
	params serverCodexAppServerTurnStartParamsV0,
) (serverCodexAppServerTurnV0, error) {
	fake.calls = append(fake.calls, "turn/start")
	fake.turnParams = params
	return fake.turn, nil
}

func (fake *fakeCodexAppServerProtocolV0) GetGoalV0(
	_ context.Context,
	threadID string,
) (*serverCodexAppServerThreadGoalV0, error) {
	fake.calls = append(fake.calls, "thread/goal/get")
	fake.getThreadID = threadID
	return fake.observedGoal, nil
}
