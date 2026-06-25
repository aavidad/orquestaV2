package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
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
		readThread: serverCodexAppServerThreadReadV0{
			ID:     "thread-ref-goal-002",
			Status: "closed",
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
	if !reflect.DeepEqual(protocol.calls, []string{"thread/goal/get", "thread/read"}) {
		t.Fatalf("calls=%v", protocol.calls)
	}
	if protocol.getThreadID != "thread-ref-goal-002" {
		t.Fatalf("get thread id=%q", protocol.getThreadID)
	}
	if protocol.readThreadID != "thread-ref-goal-002" || !protocol.readIncludeTurns {
		t.Fatalf("read thread id=%q include=%v", protocol.readThreadID, protocol.readIncludeTurns)
	}
}

func TestServerCodexAppServerGoalBackendV0ObservaResultadoMarcadoV0(t *testing.T) {
	marked := `ORQUESTA_GOAL_RESULT_V0 {"summary":"cierre con {llaves}","artifact_refs":["artifact-ref-goal-summary"],"required_test_results":[{"test_ref":"test-ref-goal","status":"passed","evidence_refs":["evidence-ref-test-pass"]}],"domain_receipt_refs":["domain-receipt-ref-001"],"evidence_refs":["evidence-ref-required"]}`
	protocol := &fakeCodexAppServerProtocolV0{
		observedGoal: &serverCodexAppServerThreadGoalV0{
			ThreadID: "thread-ref-goal-003",
			Status:   "complete",
		},
		readThread: serverCodexAppServerThreadReadV0{
			ID: "thread-ref-goal-003",
			Turns: []serverCodexAppServerReadTurnV0{{
				ID:        "turn-ref-goal-003",
				Status:    "completed",
				ItemsView: "full",
				Items: []serverCodexAppServerReadItemV0{{
					ID:    "item-ref-goal-003",
					Type:  "agentMessage",
					Phase: "final_answer",
					Text:  "Hecho.\n" + marked,
				}},
			}},
		},
	}
	backend := serverCodexAppServerGoalBackendV0{Protocol: protocol}

	receipt, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		SchemaVersion:   orquestaruntimecodexgoal.CodexGoalObservationRequestSchemaV0,
		GoalRef:         "goal-ref-codex-app-server-003",
		ExternalGoalRef: "thread-ref-goal-003",
	})

	if err != nil {
		t.Fatalf("ObserveCodexGoalV0: %v", err)
	}
	if receipt.Summary != "cierre con {llaves}" ||
		!containsStringForTestV0(receipt.ArtifactRefs, "artifact-ref-goal-summary") ||
		!containsStringForTestV0(receipt.DomainReceiptRefs, "domain-receipt-ref-001") ||
		!containsStringForTestV0(receipt.EvidenceRefs, "evidence-ref-required") ||
		!containsStringForTestV0(receipt.EvidenceRefs, "evidence-ref-codex-app-server-goal-result-marker") ||
		len(receipt.RequiredTestResults) != 1 ||
		receipt.RequiredTestResults[0].TestRef != "test-ref-goal" ||
		receipt.RequiredTestResults[0].Status != "passed" {
		t.Fatalf("receipt=%+v", receipt)
	}
}

func TestCodexAppServerGoalResultMarkerV0AceptaFallbackSinPhaseV0(t *testing.T) {
	thread := serverCodexAppServerThreadReadV0{
		ID: "thread-ref-goal-004",
		Turns: []serverCodexAppServerReadTurnV0{{
			ID: "turn-ref-goal-004",
			Items: []serverCodexAppServerReadItemV0{{
				ID:   "item-ref-goal-004",
				Type: "agentMessage",
				Text: `texto ` + orquestaruntimecodexgoal.CodexGoalResultMarkerV0 + ` {"summary":"ok","evidence_refs":["evidence-ref-ok"]}`,
			}},
		}},
	}

	marked, found, err := codexAppServerGoalResultFromThreadV0(thread)

	if err != nil || !found || marked.Summary != "ok" ||
		!containsStringForTestV0(marked.EvidenceRefs, "evidence-ref-ok") {
		t.Fatalf("marked=%+v found=%v err=%v", marked, found, err)
	}
}

func TestCodexAppServerIssueCodeForErrorV0ClasificaDiagnosticosV0(t *testing.T) {
	cases := []struct {
		name    string
		message string
		want    string
	}{
		{
			name:    "standalone missing",
			message: "Error: managed standalone Codex install not found at /home/user/.codex/packages/standalone/current/codex",
			want:    "codex_app_server_standalone_missing",
		},
		{
			name:    "socket missing",
			message: "Error: failed to connect to socket at /home/user/.codex/app-server-control/app-server-control.sock",
			want:    "codex_app_server_control_socket_missing",
		},
		{
			name:    "permission denied",
			message: "Permission denied (os error 13)",
			want:    "codex_app_server_permission_denied",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := codexAppServerIssueCodeForErrorV0(errors.New(tc.message), "fallback"); got != tc.want {
				t.Fatalf("code=%q want=%q", got, tc.want)
			}
		})
	}
}

func TestCodexAppServerIssueCodeFromCommandFailureV0UsaErrorSinStderrV0(t *testing.T) {
	code := codexAppServerIssueCodeFromCommandFailureV0(
		"",
		errors.New(`exec: "codex-missing": executable file not found in $PATH`),
	)

	if code != "codex_app_server_command_missing" {
		t.Fatalf("code=%q", code)
	}
}

func TestServerCodexAppServerUnavailableBackendV0BloqueaConIssueCodeV0(t *testing.T) {
	backend := serverCodexUnavailableGoalBackendV0{IssueCode: "codex_app_server_control_socket_missing"}

	receipt, err := backend.StartCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef: "goal-ref-unavailable-001",
	})

	if err == nil ||
		receipt.Status != orquestagoal.GoalStatusInvalidV0 ||
		receipt.IssueCode != "codex_app_server_control_socket_missing" ||
		receipt.GoalRef != "goal-ref-unavailable-001" {
		t.Fatalf("receipt=%+v err=%v", receipt, err)
	}
}

func TestServerCodexGoalBackendFromEnvV0PreflightDegradadoV0(t *testing.T) {
	script := filepath.Join(t.TempDir(), "codex-fake")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho 'Error: failed to connect to socket at /tmp/app-server-control.sock' >&2\nexit 1\n"), 0o700); err != nil {
		t.Fatalf("write fake codex: %v", err)
	}
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envCodexCommandV0, script)
	t.Setenv(envCodexGoalBackendV0, codexGoalBackendAppServerProxyV0)
	t.Setenv(envCodexGoalPreflightTimeoutMSV0, "1000")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	backend, err := serverCodexGoalBackendFromEnvV0(config)
	if err != nil {
		t.Fatalf("serverCodexGoalBackendFromEnvV0: %v", err)
	}
	if backend.Starter == nil || backend.Observer == nil {
		t.Fatalf("backend degradado debe conservar puertos: %+v", backend)
	}
	receipt, err := backend.Starter.StartCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef: "goal-ref-preflight-001",
	})
	if err == nil ||
		receipt.IssueCode != "codex_app_server_control_socket_missing" {
		t.Fatalf("receipt=%+v err=%v", receipt, err)
	}
}

func TestServerCodexGoalBackendFromEnvV0PreflightOKConservaBackendRealV0(t *testing.T) {
	script := filepath.Join(t.TempDir(), "codex-fake-ok")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nprintf '{\"id\":2,\"result\":{\"data\":[]}}\\n'\n"), 0o700); err != nil {
		t.Fatalf("write fake codex: %v", err)
	}
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envCodexCommandV0, script)
	t.Setenv(envCodexGoalBackendV0, codexGoalBackendAppServerProxyV0)
	t.Setenv(envCodexGoalPreflightTimeoutMSV0, "1000")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	backend, err := serverCodexGoalBackendFromEnvV0(config)
	if err != nil {
		t.Fatalf("serverCodexGoalBackendFromEnvV0: %v", err)
	}
	if _, degraded := backend.Starter.(serverCodexUnavailableGoalBackendV0); degraded {
		t.Fatalf("backend no debe quedar degradado tras preflight OK: %+v", backend)
	}
	if _, real := backend.Starter.(serverCodexAppServerGoalBackendV0); !real {
		t.Fatalf("starter real=%T", backend.Starter)
	}
	if _, real := backend.Observer.(serverCodexAppServerGoalBackendV0); !real {
		t.Fatalf("observer real=%T", backend.Observer)
	}
}

func TestServerCodexGoalBackendsFromEnvV0SeparaWorkdirAppEIdleV0(t *testing.T) {
	script := filepath.Join(t.TempDir(), "codex-fake-ok")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nprintf '{\"id\":2,\"result\":{\"data\":[]}}\\n'\n"), 0o700); err != nil {
		t.Fatalf("write fake codex: %v", err)
	}
	root := t.TempDir()
	appDir := filepath.Join(root, "app-workdir")
	idleDir := filepath.Join(root, "orquesta-idle-workdir")
	t.Setenv(envCodexProjectWorkDirV0, appDir)
	t.Setenv(envServerIdleSelfImprovementProjectWorkDirV0, idleDir)
	t.Setenv(envCodexCommandV0, script)
	t.Setenv(envCodexGoalBackendV0, codexGoalBackendAppServerProxyV0)
	t.Setenv(envCodexGoalPreflightTimeoutMSV0, "1000")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	backends, err := serverCodexGoalBackendsFromEnvV0(config)
	if err != nil {
		t.Fatalf("serverCodexGoalBackendsFromEnvV0: %v", err)
	}
	appStarter, ok := backends.AppGoal.Starter.(serverCodexAppServerGoalBackendV0)
	if !ok {
		t.Fatalf("app starter=%T", backends.AppGoal.Starter)
	}
	idleStarter, ok := backends.IdleGoal.Starter.(serverCodexAppServerGoalBackendV0)
	if !ok {
		t.Fatalf("idle starter=%T", backends.IdleGoal.Starter)
	}
	if appStarter.CWD != filepath.Clean(appDir) {
		t.Fatalf("app CWD=%q want %q", appStarter.CWD, filepath.Clean(appDir))
	}
	if idleStarter.CWD != filepath.Clean(idleDir) {
		t.Fatalf("idle CWD=%q want %q", idleStarter.CWD, filepath.Clean(idleDir))
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

	stdout = []byte(`{"id":2,"result":{"thread":{"id":"thread-ref-003","status":"closed","turns":[{"id":"turn-ref-003","status":"completed","itemsView":"full","items":[{"id":"item-ref-003","type":"agentMessage","phase":"final_answer","text":"` + orquestaruntimecodexgoal.CodexGoalResultMarkerV0 + ` {\"summary\":\"ok\"}"}]}]}}}` + "\n")
	var threadResponse serverCodexAppServerThreadReadResponseV0
	if err := decodeCodexAppServerRPCResponseV0(stdout, 2, &threadResponse); err != nil {
		t.Fatalf("decode thread/read: %v", err)
	}
	if threadResponse.Thread.ID != "thread-ref-003" ||
		len(threadResponse.Thread.Turns) != 1 ||
		len(threadResponse.Thread.Turns[0].Items) != 1 ||
		threadResponse.Thread.Turns[0].Items[0].Phase != "final_answer" {
		t.Fatalf("thread response=%+v", threadResponse)
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
	if got := effectiveSettingValueForTestV0(settings, envCodexGoalPreflightTimeoutMSV0); got != "3000" {
		t.Fatalf("goal preflight timeout=%q", got)
	}
}

type fakeCodexAppServerProtocolV0 struct {
	calls       []string
	startParams serverCodexAppServerThreadStartParamsV0
	setParams   serverCodexAppServerThreadGoalSetParamsV0
	turnParams  serverCodexAppServerTurnStartParamsV0
	getThreadID string

	thread           serverCodexAppServerThreadV0
	goal             serverCodexAppServerThreadGoalV0
	turn             serverCodexAppServerTurnV0
	observedGoal     *serverCodexAppServerThreadGoalV0
	readThread       serverCodexAppServerThreadReadV0
	readThreadID     string
	readIncludeTurns bool
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

func (fake *fakeCodexAppServerProtocolV0) ReadThreadV0(
	_ context.Context,
	threadID string,
	includeTurns bool,
) (serverCodexAppServerThreadReadV0, error) {
	fake.calls = append(fake.calls, "thread/read")
	fake.readThreadID = threadID
	fake.readIncludeTurns = includeTurns
	return fake.readThread, nil
}
