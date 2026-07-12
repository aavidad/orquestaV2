package main

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestacapacity "orquesta/modulos/orquesta-capacity"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimeclaude "orquesta/modulos/orquesta-runtime-claude"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
	orquestaruntimegemini "orquesta/modulos/orquesta-runtime-gemini"
	orquestaserver "orquesta/modulos/orquesta-server"
	orquestaservershutdown "orquesta/modulos/orquesta-server-shutdown"
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

func TestServerGoalObservationFingerprintFromBackendV0EsOptInV0(t *testing.T) {
	protocol := &fakeCodexAppServerProtocolV0{
		observedGoal: &serverCodexAppServerThreadGoalV0{
			ThreadID: "thread-ref-fingerprint-wiring",
			Status:   "active",
		},
	}
	backend := serverCodexAppServerGoalBackendV0{Protocol: protocol}
	if serverGoalObservationFingerprintFromBackendV0(serverCodexGoalBackendV0{Observer: backend}, false) != nil {
		t.Fatalf("fingerprint no debe exponerse sin opt-in")
	}
	if serverGoalObservationFingerprintFromBackendV0(serverCodexGoalBackendV0{Observer: backend}, true) == nil {
		t.Fatalf("backend Codex app-server debe exponer fingerprint con opt-in")
	}
}

func TestServerCodexGoalCostRoutingStarterV0AplicaModelRoutingCanonicoV0(t *testing.T) {
	tests := []struct {
		name       string
		writeSet   []orquestagoal.GoalWriteScopeV0
		wantModel  string
		wantEffort string
	}{
		{
			name: "doc markdown usa luna low",
			writeSet: []orquestagoal.GoalWriteScopeV0{{
				Path: "docs/auditoria.md",
			}},
			wantModel:  "gpt-5.6-luna",
			wantEffort: "low",
		},
		{
			name: "doc folder usa luna low",
			writeSet: []orquestagoal.GoalWriteScopeV0{{
				Path: "docs/runbooks",
			}},
			wantModel:  "gpt-5.6-luna",
			wantEffort: "low",
		},
		{
			name: "code usa terra medium sin heredar sol high",
			writeSet: []orquestagoal.GoalWriteScopeV0{{
				Path: "cmd/orquesta-server",
			}},
			wantModel:  "gpt-5.6-terra",
			wantEffort: "medium",
		},
		{
			name: "mixed usa terra medium sin heredar sol high",
			writeSet: []orquestagoal.GoalWriteScopeV0{
				{Path: "docs/auditoria.md"},
				{Path: "cmd/orquesta-server"},
			},
			wantModel:  "gpt-5.6-terra",
			wantEffort: "medium",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			protocol := &fakeCodexAppServerProtocolV0{
				thread: serverCodexAppServerThreadV0{ID: "thread-ref-cost-routing"},
				goal:   serverCodexAppServerThreadGoalV0{ThreadID: "thread-ref-cost-routing", Status: "active"},
				turn:   serverCodexAppServerTurnV0{ID: "turn-ref-cost-routing", Status: "inProgress"},
			}
			starter := serverCodexGoalCostRoutingStarterV0{
				Backend: serverCodexAppServerGoalBackendV0{
					Protocol:        protocol,
					CWD:             t.TempDir(),
					Model:           "gpt-5.6-sol",
					ReasoningEffort: "high",
					ApprovalPolicy:  "never",
				},
				ModelRouting: defaultCodexModelRoutingConfigV0(),
			}
			packet := orquestaruntimecodexgoal.CodexGoalStartPacketV0{
				GoalRef:       "goal-ref-cost-routing-" + strings.ReplaceAll(tt.name, " ", "-"),
				Objective:     "probar routing de coste goal-first",
				WriteSet:      tt.writeSet,
				TaskCostClass: orquestaruntimecodexgoal.CodexGoalTaskCostClassForWriteSetV0(tt.writeSet),
			}

			receipt, err := starter.StartCodexGoalV0(context.Background(), packet)
			if err != nil {
				t.Fatalf("StartCodexGoalV0: %v receipt=%+v", err, receipt)
			}
			if protocol.turnParams.Model != tt.wantModel || protocol.turnParams.Effort != tt.wantEffort {
				t.Fatalf("turn params model=%q effort=%q want model=%q effort=%q packet=%+v",
					protocol.turnParams.Model, protocol.turnParams.Effort, tt.wantModel, tt.wantEffort, packet)
			}
		})
	}
}

func TestServerCodexGoalCostRoutingStarterV0RechazaConfigInvalidaV0(t *testing.T) {
	protocol := &fakeCodexAppServerProtocolV0{}
	starter := serverCodexGoalCostRoutingStarterV0{
		Backend: serverCodexAppServerGoalBackendV0{Protocol: protocol, CWD: t.TempDir()},
		ModelRouting: orquestaappcodexstack.CodexModelRoutingConfigV0{
			Policy: orquestacapacity.ModelRoutingPolicyV0{Strict: true},
		},
	}
	receipt, err := starter.StartCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef:  "goal-ref-routing-invalid",
		WriteSet: []orquestagoal.GoalWriteScopeV0{{Path: "cmd/orquesta-server"}},
	})
	if err == nil || receipt.Status != orquestagoal.GoalStatusInvalidV0 || receipt.IssueCode != "codex_goal_model_routing_rejected" {
		t.Fatalf("receipt=%+v err=%v", receipt, err)
	}
	if protocol.turnParams.ThreadID != "" {
		t.Fatalf("config invalida alcanzo turn/start: %+v", protocol.turnParams)
	}
}

func TestServerCodexGoalModelRouteForPacketV0PermiteCriticalCausalV0(t *testing.T) {
	routing := defaultCodexModelRoutingConfigV0()
	goalRef := "goal-ref-routing-critical"
	routing.TaskRoutes[goalRef] = orquestacapacity.ModelRoutingRequestV0{
		Level:        orquestacapacity.ModelRoutingLevelCriticalV0,
		ReasonRef:    "reason-ref-routing-critical",
		EvidenceRefs: []string{"evidence-ref-routing-critical"},
	}
	decision, model, err := serverCodexGoalModelRouteForPacketV0(routing, orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef:  goalRef,
		WriteSet: []orquestagoal.GoalWriteScopeV0{{Path: "cmd/orquesta-server"}},
	})
	if err != nil || decision.Rejected || model != "gpt-5.6-sol" || decision.ReasoningEffort != "high" {
		t.Fatalf("decision=%+v model=%q err=%v", decision, model, err)
	}
}

func TestServerCodexGoalTaskCostClassForPacketV0IgnoraDeclaradoIncoherenteV0(t *testing.T) {
	packet := orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		TaskCostClass: orquestaruntimecodexgoal.CodexGoalTaskCostClassDocV0,
		WriteSet: []orquestagoal.GoalWriteScopeV0{{
			Path: "cmd/orquesta-server",
		}},
	}
	if got := serverCodexGoalTaskCostClassForPacketV0(packet); got != orquestaruntimecodexgoal.CodexGoalTaskCostClassCodeV0 {
		t.Fatalf("task_cost_class=%q want code", got)
	}
}

func TestServerCodexGoalBackendFromEnvV0TmuxNoArrancaAppServerEnConstruccionV0(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	runtimeDir := filepath.Join(root, "runtime")
	codeHome := filepath.Join(root, "codex-home")
	if err := os.MkdirAll(codeHome, 0o700); err != nil {
		t.Fatalf("mkdir code home: %v", err)
	}
	if err := os.WriteFile(filepath.Join(codeHome, "auth.json"), []byte(`{"ok":true}`), 0o600); err != nil {
		t.Fatalf("write auth: %v", err)
	}
	marker := filepath.Join(root, "codex-invoked")
	fakeCodex := filepath.Join(root, "codex")
	script := "#!/usr/bin/env bash\nprintf invoked > " + shellQuoteCodexAppServerWiringTestV0(marker) + "\nexit 42\n"
	if err := os.WriteFile(fakeCodex, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake codex: %v", err)
	}
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envCodexRuntimeWorkDirV0, runtimeDir)
	t.Setenv(envCodexCodeHomeV0, codeHome)
	t.Setenv(envCodexCommandV0, fakeCodex)
	t.Setenv(envCodexGoalBackendV0, codexGoalBackendAppServerTmuxV0)
	t.Setenv(envAutoprogrammingCheckpointOnlyHighConsumptionTokensV0, "37")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	backend, err := serverCodexGoalBackendFromEnvV0(config)
	if err != nil {
		t.Fatalf("serverCodexGoalBackendFromEnvV0: %v", err)
	}
	if backend.Starter == nil || backend.Observer == nil || backend.ShutdownHook == nil {
		t.Fatalf("backend no cableado: %+v", backend)
	}
	starter := codexAppServerGoalBackendFromPortForTestV0(t, backend.Starter)
	if starter.HighTokenUsageThreshold != 37 {
		t.Fatalf("starter sin umbral alto configurable: %#v", backend.Starter)
	}
	observer := codexAppServerGoalBackendFromPortForTestV0(t, backend.Observer)
	if observer.HighTokenUsageThreshold != 37 {
		t.Fatalf("observer sin umbral alto configurable: %#v", backend.Observer)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatalf("construir backend no debe invocar app-server")
	}
}

func TestServerCodexGoalBackendsFromEnvV0SeparaWorkspaceAutoprogrammingDeAppsV0(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	runtimeDir := filepath.Join(root, "runtime")
	codeHome := filepath.Join(root, "codex-home")
	if err := os.MkdirAll(codeHome, 0o700); err != nil {
		t.Fatalf("mkdir code home: %v", err)
	}
	if err := os.WriteFile(filepath.Join(codeHome, "auth.json"), []byte(`{"ok":true}`), 0o600); err != nil {
		t.Fatalf("write auth: %v", err)
	}
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envCodexRuntimeWorkDirV0, runtimeDir)
	t.Setenv(envCodexCodeHomeV0, codeHome)
	t.Setenv(envCodexCommandV0, filepath.Join(root, "codex"))
	t.Setenv(envCodexGoalBackendV0, codexGoalBackendAppServerTmuxV0)

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	backends, err := serverCodexGoalBackendsFromEnvV0(config)
	if err != nil {
		t.Fatalf("serverCodexGoalBackendsFromEnvV0: %v", err)
	}
	app := codexAppServerGoalBackendFromPortForTestV0(t, backends.AppGoal.Starter)
	autoprogramming := codexAppServerGoalBackendFromPortForTestV0(t, backends.AutoprogrammingGoal.Starter)
	if app.WorkspaceRouter != nil {
		t.Fatalf("AppGoal normal no debe provisionar workspaces fisicos: %+v", app)
	}
	if autoprogramming.WorkspaceRouter == nil {
		t.Fatalf("AutoprogrammingGoal debe resolver GoalWorkspace fisico: %+v", autoprogramming)
	}
}

func TestServerCodexGoalBackendFromEnvV0TmuxLeeGoalBackendDesdeFicheroCanonicoV0(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	runtimeDir := filepath.Join(root, "runtime")
	codeHome := filepath.Join(root, "codex-home")
	if err := os.MkdirAll(codeHome, 0o700); err != nil {
		t.Fatalf("mkdir code home: %v", err)
	}
	if err := os.WriteFile(filepath.Join(codeHome, "auth.json"), []byte(`{"ok":true}`), 0o600); err != nil {
		t.Fatalf("write auth: %v", err)
	}
	marker := filepath.Join(root, "codex-invoked")
	fakeCodex := filepath.Join(root, "codex")
	script := "#!/usr/bin/env bash\nprintf invoked > " + shellQuoteCodexAppServerWiringTestV0(marker) + "\nexit 42\n"
	if err := os.WriteFile(fakeCodex, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake codex: %v", err)
	}
	if err := os.MkdirAll(projectDir, 0o700); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	configFile := `{
		"schema_version":"orquesta_config.v0",
		"codex_runtime":{"runtime_work_dir":"` + filepath.ToSlash(runtimeDir) + `"},
		"autoprogramming":{"checkpoint_only_high_consumption_tokens":41},
		"goal_backend":{
			"kind":"app_server_tmux",
			"timeout_ms":13000,
			"preflight_timeout_ms":2400
		}
	}`
	if err := os.WriteFile(filepath.Join(projectDir, serverProjectConfigFileNameV0), []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envCodexRuntimeWorkDirV0, "")
	t.Setenv(envCodexCodeHomeV0, codeHome)
	t.Setenv(envCodexCommandV0, fakeCodex)
	t.Setenv(envCodexGoalBackendV0, "")
	t.Setenv(envCodexGoalTimeoutMSV0, "")
	t.Setenv(envCodexGoalPreflightTimeoutMSV0, "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	backend, err := serverCodexGoalBackendFromEnvV0(config)
	if err != nil {
		t.Fatalf("serverCodexGoalBackendFromEnvV0: %v", err)
	}
	starter := codexAppServerGoalBackendFromPortForTestV0(t, backend.Starter)
	if starter.Timeout != 13*time.Second || starter.HighTokenUsageThreshold != 41 {
		t.Fatalf("starter desde fichero inesperado: timeout=%s threshold=%d", starter.Timeout, starter.HighTokenUsageThreshold)
	}
	shutdownHook, ok := backend.ShutdownHook.(serverCodexAppServerTmuxBackendV0)
	if !ok || shutdownHook.Timeout != 2400*time.Millisecond {
		t.Fatalf("shutdown hook desde fichero inesperado: %#v", backend.ShutdownHook)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatalf("construir backend desde fichero no debe invocar app-server")
	}
}

func codexAppServerGoalBackendFromPortForTestV0(t *testing.T, port any) serverCodexAppServerGoalBackendV0 {
	t.Helper()
	switch value := port.(type) {
	case serverCodexAppServerGoalBackendV0:
		return value
	case serverCodexGoalCostRoutingStarterV0:
		return value.Backend
	default:
		t.Fatalf("puerto codex app-server inesperado: %#v", port)
		return serverCodexAppServerGoalBackendV0{}
	}
}

func TestServerGoalBackendFromEnvV0ClaudeFileControlExponePuertosNeutralesV0(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	stateDir := filepath.Join(root, "control", "state")
	t.Setenv(envCodexGoalBackendV0, claudeGoalBackendFileControlV0)
	if err := os.MkdirAll(projectDir, 0o700); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	configFile := `{"schema_version":"orquesta_config.v0","goal_backend":{"prompt_locale":"en-US"}}`
	if err := os.WriteFile(filepath.Join(projectDir, serverProjectConfigFileNameV0), []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	config := orquestaserver.ConfigV0{
		ProjectWorkDir:        projectDir,
		RuntimeWorkDir:        filepath.Join(projectDir, ".orquesta-runtime"),
		StateDir:              stateDir,
		ProjectConfigFilePath: filepath.Join(projectDir, serverProjectConfigFileNameV0),
	}
	backend, err := serverCodexGoalBackendFromEnvForWorkDirV0(config, projectDir)
	if err != nil {
		t.Fatalf("serverCodexGoalBackendFromEnvForWorkDirV0: %v", err)
	}
	if backend.Starter != nil || backend.Observer != nil {
		t.Fatalf("backend Claude goal no debe usar puertos Codex: %+v", backend)
	}
	if backend.GoalLauncher == nil || backend.GoalObserver == nil {
		t.Fatalf("backend Claude goal sin puertos neutrales: %+v", backend)
	}
	launcher := serverGoalWorkLauncherFromBackendV0(backend)
	observer := serverGoalWorkObserverFromBackendV0(backend)
	spec := orquestagoal.GoalWorkSpecV0{
		SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
		GoalRef:       "goal-ref-server-claude-file-control-001",
		RequestRef:    "request-ref-server-claude-file-control-001",
		RunRef:        "run-ref-server-claude-file-control-001",
		Objective:     "Probar backend Claude file-control desde servidor.",
		DirectorKind:  orquestagoal.GoalDirectorKindRuntimeGoalV0,
		WriteSet: []orquestagoal.GoalWriteScopeV0{{
			Path:    "docs",
			Purpose: "resultado durable",
		}},
		RequiredTests: []orquestagoal.GoalRequiredTestV0{{
			TestRef: "required-test-ref-server-claude-file-control",
			Command: "go test ./cmd/orquesta-server",
		}},
	}
	receipt, err := launcher.LaunchGoalWorkV0(context.Background(), spec)
	if err != nil {
		t.Fatalf("LaunchGoalWorkV0: %v", err)
	}
	if receipt.Status != orquestagoal.GoalStatusRunningV0 ||
		!strings.HasPrefix(receipt.ExternalGoalRef, "claude-goal-") {
		t.Fatalf("receipt Claude inesperado: %+v", receipt)
	}
	runtimeDir := filepath.Join(filepath.Dir(stateDir), "claude-goal")
	promptPath := filepath.Join(runtimeDir, "claude_goal_prompt_goal-ref-server-claude-file-control-001.txt")
	if _, err := os.Stat(promptPath); err != nil {
		t.Fatalf("prompt Claude no materializado: %v", err)
	}
	prompt := mustReadFileStringV0(t, promptPath)
	if !strings.Contains(prompt, "You are a Claude goal-first backend governed by Orquesta.") ||
		!strings.Contains(prompt, "NEUTRAL DURABLE RESULT") {
		t.Fatalf("prompt Claude no usa locale en-US:\n%s", prompt)
	}
	observed, err := observer.ObserveGoalWorkV0(context.Background(), orquestagoal.GoalObservationRequestV0{
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: receipt.ExternalGoalRef,
	})
	if err != nil {
		t.Fatalf("ObserveGoalWorkV0: %v", err)
	}
	if observed.Status != orquestagoal.GoalStatusRunningV0 ||
		!containsServerGoalStringV0(observed.EvidenceRefs, orquestaruntimeclaude.ClaudeGoalEvidenceResultPendingV0) {
		t.Fatalf("observed Claude inesperado: %+v", observed)
	}
}

func TestServerGoalBackendFromEnvV0ClaudeProcessLanzaYObservaResultadoV0(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	stateDir := filepath.Join(root, "control", "state")
	goalRef := "goal-ref-server-claude-process-001"
	t.Setenv(envCodexGoalBackendV0, claudeGoalBackendProcessV0)
	t.Setenv(envClaudeCommandV0, fakeClaudeGoalProcessCommandForTestV0(t, root, goalRef))

	backend, err := serverCodexGoalBackendFromEnvForWorkDirV0(orquestaserver.ConfigV0{
		ProjectWorkDir: projectDir,
		RuntimeWorkDir: filepath.Join(projectDir, ".orquesta-runtime"),
		StateDir:       stateDir,
	}, projectDir)
	if err != nil {
		t.Fatalf("serverCodexGoalBackendFromEnvForWorkDirV0: %v", err)
	}
	launcher := serverGoalWorkLauncherFromBackendV0(backend)
	observer := serverGoalWorkObserverFromBackendV0(backend)
	spec := orquestagoal.GoalWorkSpecV0{
		SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
		GoalRef:       goalRef,
		RequestRef:    "request-ref-server-claude-process-001",
		RunRef:        "run-ref-server-claude-process-001",
		Objective:     "Probar backend Claude process desde servidor.",
		DirectorKind:  orquestagoal.GoalDirectorKindRuntimeGoalV0,
		WriteSet: []orquestagoal.GoalWriteScopeV0{{
			Path:    "docs",
			Purpose: "resultado durable",
		}},
		RequiredTests: []orquestagoal.GoalRequiredTestV0{{
			TestRef: "required-test-ref-server-claude-process",
			Command: "fake claude process",
		}},
	}
	receipt, err := launcher.LaunchGoalWorkV0(context.Background(), spec)
	if err != nil {
		t.Fatalf("LaunchGoalWorkV0: %v", err)
	}
	if !containsServerGoalStringV0(receipt.EvidenceRefs, orquestaruntimeclaude.ClaudeGoalEvidenceProcessLaunchedV0) {
		t.Fatalf("receipt sin evidencia proceso: %+v", receipt)
	}
	resultPath := filepath.Join(projectDir, "docs", orquestaruntimeclaude.ClaudeGoalResultFileNameV0)
	waitForServerGoalFileV0(t, resultPath)
	observed, err := observer.ObserveGoalWorkV0(context.Background(), orquestagoal.GoalObservationRequestV0{
		GoalRef:         goalRef,
		ExternalGoalRef: receipt.ExternalGoalRef,
	})
	if err != nil {
		t.Fatalf("ObserveGoalWorkV0: %v", err)
	}
	if observed.Status != orquestagoal.GoalStatusCompleteV0 ||
		!containsServerGoalStringV0(observed.EvidenceRefs, orquestaruntimeclaude.ClaudeGoalEvidenceResultReadV0) {
		t.Fatalf("observed inesperado: %+v", observed)
	}
}

func TestServerGoalBackendFromEnvV0ClaudeProcessControlParaProcesoV0(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	stateDir := filepath.Join(root, "control", "state")
	goalRef := "goal-ref-server-claude-process-control-001"
	startedPath := filepath.Join(root, "started.txt")
	t.Setenv(envCodexGoalBackendV0, claudeGoalBackendProcessV0)
	t.Setenv(envClaudeCommandV0, fakeClaudeGoalLongRunningCommandForTestV0(t, root, startedPath))

	backend, err := serverCodexGoalBackendFromEnvForWorkDirV0(orquestaserver.ConfigV0{
		ProjectWorkDir: projectDir,
		RuntimeWorkDir: filepath.Join(projectDir, ".orquesta-runtime"),
		StateDir:       stateDir,
	}, projectDir)
	if err != nil {
		t.Fatalf("serverCodexGoalBackendFromEnvForWorkDirV0: %v", err)
	}
	launcher := serverGoalWorkLauncherFromBackendV0(backend)
	control := serverGoalBackendControlFromBackendV0(backend)
	if control == nil {
		t.Fatalf("backend Claude process debe exponer control")
	}
	_, err = launcher.LaunchGoalWorkV0(context.Background(), orquestagoal.GoalWorkSpecV0{
		SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
		GoalRef:       goalRef,
		RequestRef:    "request-ref-server-claude-process-control-001",
		RunRef:        "run-ref-server-claude-process-control-001",
		Objective:     "Probar control Claude process desde servidor.",
		DirectorKind:  orquestagoal.GoalDirectorKindRuntimeGoalV0,
		WriteSet: []orquestagoal.GoalWriteScopeV0{{
			Path:    "docs",
			Purpose: "resultado durable",
		}},
		RequiredTests: []orquestagoal.GoalRequiredTestV0{{
			TestRef: "required-test-ref-server-claude-process-control",
			Command: "fake claude process control",
		}},
	})
	if err != nil {
		t.Fatalf("LaunchGoalWorkV0: %v", err)
	}
	waitForServerGoalFileV0(t, startedPath)

	result, err := control.ControlGoalBackendV0(context.Background(), orquestaappcodexstack.GoalBackendControlRequestV0{
		GoalRef:      goalRef,
		Action:       "stop",
		Reason:       "test_stop",
		Forced:       true,
		EvidenceRefs: []string{"evidence-ref-server-claude-control"},
	})
	if err != nil {
		t.Fatalf("ControlGoalBackendV0: %v result=%+v", err, result)
	}
	if result.Status != orquestagoal.GoalStatusBlockedV0 ||
		!result.GoalStatusSet ||
		!result.BackendStopped ||
		!containsServerGoalStringV0(result.EvidenceRefs, orquestaruntimeclaude.ClaudeGoalEvidenceStopCompletedV0) {
		t.Fatalf("control result inesperado: %+v", result)
	}
}

func TestServerGoalBackendFromEnvV0GeminiFileControlExponePuertosNeutralesV0(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	stateDir := filepath.Join(root, "control", "state")
	t.Setenv(envCodexGoalBackendV0, geminiGoalBackendFileControlV0)
	if err := os.MkdirAll(projectDir, 0o700); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	configFile := `{"schema_version":"orquesta_config.v0","goal_backend":{"prompt_locale":"en-US"}}`
	if err := os.WriteFile(filepath.Join(projectDir, serverProjectConfigFileNameV0), []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	config := orquestaserver.ConfigV0{
		ProjectWorkDir:        projectDir,
		RuntimeWorkDir:        filepath.Join(projectDir, ".orquesta-runtime"),
		StateDir:              stateDir,
		ProjectConfigFilePath: filepath.Join(projectDir, serverProjectConfigFileNameV0),
	}
	backend, err := serverCodexGoalBackendFromEnvForWorkDirV0(config, projectDir)
	if err != nil {
		t.Fatalf("serverCodexGoalBackendFromEnvForWorkDirV0: %v", err)
	}
	if backend.Starter != nil || backend.Observer != nil {
		t.Fatalf("backend Gemini goal no debe usar puertos Codex: %+v", backend)
	}
	if backend.GoalLauncher == nil || backend.GoalObserver == nil {
		t.Fatalf("backend Gemini goal sin puertos neutrales: %+v", backend)
	}
	launcher := serverGoalWorkLauncherFromBackendV0(backend)
	observer := serverGoalWorkObserverFromBackendV0(backend)
	spec := orquestagoal.GoalWorkSpecV0{
		SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
		GoalRef:       "goal-ref-server-gemini-file-control-001",
		RequestRef:    "request-ref-server-gemini-file-control-001",
		RunRef:        "run-ref-server-gemini-file-control-001",
		Objective:     "Probar backend Gemini file-control desde servidor.",
		DirectorKind:  orquestagoal.GoalDirectorKindRuntimeGoalV0,
		WriteSet: []orquestagoal.GoalWriteScopeV0{{
			Path:    "docs",
			Purpose: "resultado durable",
		}},
		RequiredTests: []orquestagoal.GoalRequiredTestV0{{
			TestRef: "required-test-ref-server-gemini-file-control",
			Command: "go test ./cmd/orquesta-server",
		}},
	}
	receipt, err := launcher.LaunchGoalWorkV0(context.Background(), spec)
	if err != nil {
		t.Fatalf("LaunchGoalWorkV0: %v", err)
	}
	if receipt.Status != orquestagoal.GoalStatusRunningV0 ||
		!strings.HasPrefix(receipt.ExternalGoalRef, "gemini-goal-") {
		t.Fatalf("receipt Gemini inesperado: %+v", receipt)
	}
	runtimeDir := filepath.Join(filepath.Dir(stateDir), "gemini-goal")
	promptPath := filepath.Join(runtimeDir, "gemini_goal_prompt_goal-ref-server-gemini-file-control-001.txt")
	if _, err := os.Stat(promptPath); err != nil {
		t.Fatalf("prompt Gemini no materializado: %v", err)
	}
	prompt := mustReadFileStringV0(t, promptPath)
	if !strings.Contains(prompt, "You are a Gemini goal-first backend governed by Orquesta.") ||
		!strings.Contains(prompt, "NEUTRAL DURABLE RESULT") {
		t.Fatalf("prompt Gemini no usa locale en-US:\n%s", prompt)
	}
	observed, err := observer.ObserveGoalWorkV0(context.Background(), orquestagoal.GoalObservationRequestV0{
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: receipt.ExternalGoalRef,
	})
	if err != nil {
		t.Fatalf("ObserveGoalWorkV0: %v", err)
	}
	if observed.Status != orquestagoal.GoalStatusRunningV0 ||
		!containsServerGoalStringV0(observed.EvidenceRefs, orquestaruntimegemini.GeminiGoalEvidenceResultPendingV0) {
		t.Fatalf("observed Gemini inesperado: %+v", observed)
	}
}

func TestServerGoalBackendFromEnvV0GeminiProcessLanzaYObservaResultadoV0(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	stateDir := filepath.Join(root, "control", "state")
	goalRef := "goal-ref-server-gemini-process-001"
	t.Setenv(envCodexGoalBackendV0, geminiGoalBackendProcessV0)
	t.Setenv(envGeminiCommandV0, fakeGeminiGoalProcessCommandForTestV0(t, root, goalRef))

	backend, err := serverCodexGoalBackendFromEnvForWorkDirV0(orquestaserver.ConfigV0{
		ProjectWorkDir: projectDir,
		RuntimeWorkDir: filepath.Join(projectDir, ".orquesta-runtime"),
		StateDir:       stateDir,
	}, projectDir)
	if err != nil {
		t.Fatalf("serverCodexGoalBackendFromEnvForWorkDirV0: %v", err)
	}
	launcher := serverGoalWorkLauncherFromBackendV0(backend)
	observer := serverGoalWorkObserverFromBackendV0(backend)
	spec := orquestagoal.GoalWorkSpecV0{
		SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
		GoalRef:       goalRef,
		RequestRef:    "request-ref-server-gemini-process-001",
		RunRef:        "run-ref-server-gemini-process-001",
		Objective:     "Probar backend Gemini process desde servidor.",
		DirectorKind:  orquestagoal.GoalDirectorKindRuntimeGoalV0,
		WriteSet: []orquestagoal.GoalWriteScopeV0{{
			Path:    "docs",
			Purpose: "resultado durable",
		}},
		RequiredTests: []orquestagoal.GoalRequiredTestV0{{
			TestRef: "required-test-ref-server-gemini-process",
			Command: "fake gemini process",
		}},
	}
	receipt, err := launcher.LaunchGoalWorkV0(context.Background(), spec)
	if err != nil {
		t.Fatalf("LaunchGoalWorkV0: %v", err)
	}
	if !containsServerGoalStringV0(receipt.EvidenceRefs, orquestaruntimegemini.GeminiGoalEvidenceProcessLaunchedV0) {
		t.Fatalf("receipt sin evidencia proceso: %+v", receipt)
	}
	resultPath := filepath.Join(projectDir, "docs", orquestaruntimegemini.GeminiGoalResultFileNameV0)
	waitForServerGoalFileV0(t, resultPath)
	observed, err := observer.ObserveGoalWorkV0(context.Background(), orquestagoal.GoalObservationRequestV0{
		GoalRef:         goalRef,
		ExternalGoalRef: receipt.ExternalGoalRef,
	})
	if err != nil {
		t.Fatalf("ObserveGoalWorkV0: %v", err)
	}
	if observed.Status != orquestagoal.GoalStatusCompleteV0 ||
		!containsServerGoalStringV0(observed.EvidenceRefs, orquestaruntimegemini.GeminiGoalEvidenceResultReadV0) {
		t.Fatalf("observed inesperado: %+v", observed)
	}
}

func TestServerGoalBackendFromEnvV0GeminiProcessControlParaProcesoV0(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	stateDir := filepath.Join(root, "control", "state")
	goalRef := "goal-ref-server-gemini-process-control-001"
	startedPath := filepath.Join(root, "started.txt")
	t.Setenv(envCodexGoalBackendV0, geminiGoalBackendProcessV0)
	t.Setenv(envGeminiCommandV0, fakeGeminiGoalLongRunningCommandForTestV0(t, root, startedPath))

	backend, err := serverCodexGoalBackendFromEnvForWorkDirV0(orquestaserver.ConfigV0{
		ProjectWorkDir: projectDir,
		RuntimeWorkDir: filepath.Join(projectDir, ".orquesta-runtime"),
		StateDir:       stateDir,
	}, projectDir)
	if err != nil {
		t.Fatalf("serverCodexGoalBackendFromEnvForWorkDirV0: %v", err)
	}
	launcher := serverGoalWorkLauncherFromBackendV0(backend)
	control := serverGoalBackendControlFromBackendV0(backend)
	if control == nil {
		t.Fatalf("backend Gemini process debe exponer control")
	}
	_, err = launcher.LaunchGoalWorkV0(context.Background(), orquestagoal.GoalWorkSpecV0{
		SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
		GoalRef:       goalRef,
		RequestRef:    "request-ref-server-gemini-process-control-001",
		RunRef:        "run-ref-server-gemini-process-control-001",
		Objective:     "Probar control Gemini process desde servidor.",
		DirectorKind:  orquestagoal.GoalDirectorKindRuntimeGoalV0,
		WriteSet: []orquestagoal.GoalWriteScopeV0{{
			Path:    "docs",
			Purpose: "resultado durable",
		}},
		RequiredTests: []orquestagoal.GoalRequiredTestV0{{
			TestRef: "required-test-ref-server-gemini-process-control",
			Command: "fake gemini process control",
		}},
	})
	if err != nil {
		t.Fatalf("LaunchGoalWorkV0: %v", err)
	}
	waitForServerGoalFileV0(t, startedPath)

	result, err := control.ControlGoalBackendV0(context.Background(), orquestaappcodexstack.GoalBackendControlRequestV0{
		GoalRef:      goalRef,
		Action:       "stop",
		Reason:       "test_stop",
		Forced:       true,
		EvidenceRefs: []string{"evidence-ref-server-gemini-control"},
	})
	if err != nil {
		t.Fatalf("ControlGoalBackendV0: %v result=%+v", err, result)
	}
	if result.Status != orquestagoal.GoalStatusBlockedV0 ||
		!result.GoalStatusSet ||
		!result.BackendStopped ||
		!containsServerGoalStringV0(result.EvidenceRefs, orquestaruntimegemini.GeminiGoalEvidenceStopCompletedV0) {
		t.Fatalf("control result inesperado: %+v", result)
	}
}

func TestServerGoalBackendFromEnvV0RechazaBackendNoSoportadoV0(t *testing.T) {
	t.Setenv(envCodexGoalBackendV0, "claude_real_process")

	_, err := serverCodexGoalBackendFromEnvForWorkDirV0(orquestaserver.ConfigV0{
		ProjectWorkDir: t.TempDir(),
		RuntimeWorkDir: t.TempDir(),
		StateDir:       filepath.Join(t.TempDir(), "state"),
	}, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "codex_goal_backend_no_soportado") {
		t.Fatalf("backend no soportado no rechazado: %v", err)
	}
}

func TestServerGoalBackendFromEnvV0RechazaProxyHistoricoAunqueTengaOptInV0(t *testing.T) {
	baseConfig := func(t *testing.T) orquestaserver.ConfigV0 {
		t.Helper()
		return orquestaserver.ConfigV0{
			ProjectWorkDir: t.TempDir(),
			RuntimeWorkDir: t.TempDir(),
			StateDir:       filepath.Join(t.TempDir(), "state"),
		}
	}

	t.Run("sin opt-in exige diagnostico explicito", func(t *testing.T) {
		t.Setenv(envCodexGoalBackendV0, codexGoalBackendAppServerProxyV0)
		t.Setenv(envAllowAppServerProxyDiagnosticV0, "false")

		_, err := serverCodexGoalBackendFromEnvForWorkDirV0(baseConfig(t), t.TempDir())
		if err == nil || !strings.Contains(err.Error(), "codex_goal_backend_proxy_diagnostic_opt_in_required") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("con opt-in sigue sin ser backend operativo", func(t *testing.T) {
		t.Setenv(envCodexGoalBackendV0, codexGoalBackendAppServerProxyV0)
		t.Setenv(envAllowAppServerProxyDiagnosticV0, "true")

		_, err := serverCodexGoalBackendFromEnvForWorkDirV0(baseConfig(t), t.TempDir())
		if err == nil || !strings.Contains(err.Error(), "codex_goal_backend_proxy_diagnostic_not_operational") {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestServerGoalShutdownHooksFromBackendsV0DeduplicaMismoHookV0(t *testing.T) {
	hook := serverCodexAppServerTmuxBackendV0{
		SocketPath:  filepath.Join(t.TempDir(), "s.sock"),
		SessionName: "orquesta-goal-hook-1234567890",
	}
	hooks := serverGoalShutdownHooksFromBackendsV0(
		serverCodexGoalBackendV0{ShutdownHook: hook},
		serverCodexGoalBackendV0{ShutdownHook: hook},
		serverCodexGoalBackendV0{ShutdownHook: hook},
	)
	if len(hooks) != 1 {
		t.Fatalf("hooks=%d", len(hooks))
	}
}

func TestServerGoalShutdownHooksFromBackendsV0ConservaIdentidadesCausalesDistintasV0(t *testing.T) {
	root := t.TempDir()
	hooks := serverGoalShutdownHooksFromBackendsV0(
		serverCodexGoalBackendV0{ShutdownHook: serverCodexAppServerTmuxBackendV0{
			SocketPath: filepath.Join(root, "app.sock"), SessionName: "orquesta-goal-app-1234567890",
		}},
		serverCodexGoalBackendV0{ShutdownHook: serverCodexAppServerTmuxBackendV0{
			SocketPath: filepath.Join(root, "autoprogramming.sock"), SessionName: "orquesta-goal-autoprogramming-1234567890",
		}},
		serverCodexGoalBackendV0{ShutdownHook: serverCodexAppServerTmuxBackendV0{
			SocketPath: filepath.Join(root, "idle.sock"), SessionName: "orquesta-goal-idle-1234567890",
		}},
	)
	if len(hooks) != 3 {
		t.Fatalf("hooks=%d", len(hooks))
	}
}

func TestServerAutoprogrammingGoalWorkspaceSelectorsV0DeleganActiveShutdownWorkV0(t *testing.T) {
	app := &fakeServerGoalWorkspacePortV0{
		identity: "shutdown-identity-app",
		active: orquestaservershutdown.ActiveShutdownWorkResultV0{ActiveWorks: []orquestaservershutdown.ActiveShutdownWorkV0{{
			Kind: "goal_backend", RunRef: "run-ref-app", WorkRef: "goal-ref-app",
		}}},
	}
	autoprogramming := &fakeServerGoalWorkspacePortV0{
		identity: "shutdown-identity-autoprogramming",
		active: orquestaservershutdown.ActiveShutdownWorkResultV0{ActiveWorks: []orquestaservershutdown.ActiveShutdownWorkV0{{
			Kind: "goal_backend", RunRef: "run-ref-autoprogramming", WorkRef: "goal-ref-autoprogramming",
		}}},
	}
	launcher := serverAutoprogrammingGoalWorkspaceLauncherV0{AppGoal: app, AutoprogrammingGoal: autoprogramming}
	observer := serverAutoprogrammingGoalWorkspaceObserverV0{AppGoal: app, AutoprogrammingGoal: autoprogramming}

	active, err := launcher.ReadActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkRequestV0{})
	if err != nil || len(active.ActiveWorks) != 2 || app.readCalls != 1 || autoprogramming.readCalls != 1 {
		t.Fatalf("active=%+v app_reads=%d autoprogramming_reads=%d err=%v", active, app.readCalls, autoprogramming.readCalls, err)
	}
	cleaned, err := observer.CleanupActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkCleanupCommandV0{})
	if err != nil || cleaned.CleanedWorkCount != 2 || app.cleanupCalls != 1 || autoprogramming.cleanupCalls != 1 {
		t.Fatalf("cleaned=%+v app_cleanups=%d autoprogramming_cleanups=%d err=%v", cleaned, app.cleanupCalls, autoprogramming.cleanupCalls, err)
	}
	if launcher.ActiveShutdownWorkIdentityV0() == "" || launcher.ActiveShutdownWorkIdentityV0() != observer.ActiveShutdownWorkIdentityV0() {
		t.Fatalf("identidades launcher=%q observer=%q", launcher.ActiveShutdownWorkIdentityV0(), observer.ActiveShutdownWorkIdentityV0())
	}

	shared := &fakeServerGoalWorkspacePortV0{identity: "shutdown-identity-shared"}
	sharedLauncher := serverAutoprogrammingGoalWorkspaceLauncherV0{AppGoal: shared, AutoprogrammingGoal: shared}
	sharedObserver := serverAutoprogrammingGoalWorkspaceObserverV0{AppGoal: shared, AutoprogrammingGoal: shared}
	if _, err := sharedLauncher.ReadActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkRequestV0{}); err != nil {
		t.Fatalf("shared ReadActiveShutdownWorkV0: %v", err)
	}
	if _, err := sharedObserver.CleanupActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkCleanupCommandV0{}); err != nil {
		t.Fatalf("shared CleanupActiveShutdownWorkV0: %v", err)
	}
	if shared.readCalls != 1 || shared.cleanupCalls != 1 {
		t.Fatalf("backend compartido duplicado reads=%d cleanups=%d", shared.readCalls, shared.cleanupCalls)
	}
}

func TestServerGoalWorkPortsFromBackendV0PropaganActiveShutdownWorkV0(t *testing.T) {
	protocol := &fakeCodexAppServerProtocolV0{
		thread: serverCodexAppServerThreadV0{ID: "thread-ref-wiring-001"},
		goal:   serverCodexAppServerThreadGoalV0{ThreadID: "thread-ref-wiring-001", Status: "active"},
		turn:   serverCodexAppServerTurnV0{ID: "turn-ref-wiring-001", Status: "inProgress"},
	}
	hook := serverCodexAppServerTmuxBackendV0{
		SocketPath:  filepath.Join(t.TempDir(), "goal.sock"),
		SessionName: "orquesta-goal-active-1234567890",
		Timeout:     10 * time.Millisecond,
	}
	backend := serverCodexGoalBackendV0{
		Starter:      serverCodexAppServerGoalBackendV0{Protocol: protocol},
		Observer:     serverCodexAppServerGoalBackendV0{Protocol: protocol},
		ShutdownHook: hook,
	}
	launcher := serverGoalWorkLauncherFromBackendV0(backend)
	if _, ok := launcher.(orquestaservershutdown.ActiveShutdownWorkReaderPortV0); !ok {
		t.Fatalf("launcher debe propagar active shutdown work")
	}
	observer := serverGoalWorkObserverFromBackendV0(backend)
	if _, ok := observer.(orquestaservershutdown.ActiveShutdownWorkCleanerPortV0); !ok {
		t.Fatalf("observer debe propagar cleanup active shutdown work")
	}
	launcherIdentity, ok := launcher.(orquestaservershutdown.ActiveShutdownWorkIdentityPortV0)
	if !ok {
		t.Fatalf("launcher debe propagar identidad active shutdown work")
	}
	observerIdentity, ok := observer.(orquestaservershutdown.ActiveShutdownWorkIdentityPortV0)
	if !ok {
		t.Fatalf("observer debe propagar identidad active shutdown work")
	}
	if launcherIdentity.ActiveShutdownWorkIdentityV0() == "" ||
		launcherIdentity.ActiveShutdownWorkIdentityV0() != observerIdentity.ActiveShutdownWorkIdentityV0() {
		t.Fatalf("identidades shutdown launcher=%q observer=%q",
			launcherIdentity.ActiveShutdownWorkIdentityV0(),
			observerIdentity.ActiveShutdownWorkIdentityV0(),
		)
	}
}

func TestServerEffectiveConfigV0ExponeGoalFirstYBackendV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envServerIdleSelfImprovementGoalFirstV0, "true")
	t.Setenv(envCodexGoalBackendV0, codexGoalBackendAppServerTmuxV0)
	t.Setenv(envCodexGoalTimeoutMSV0, "12000")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	settings := config.EffectiveConfig.Settings
	if got := effectiveSettingValueForTestV0(settings, envServerIdleSelfImprovementGoalFirstV0); got != "true" {
		t.Fatalf("goal-first setting=%q", got)
	}
	if got := effectiveSettingValueForTestV0(settings, envCodexGoalBackendV0); got != codexGoalBackendAppServerTmuxV0 {
		t.Fatalf("goal backend=%q", got)
	}
	if got := effectiveSettingValueForTestV0(settings, envCodexGoalTimeoutMSV0); got != "12000" {
		t.Fatalf("goal timeout=%q", got)
	}
}

func shellQuoteCodexAppServerWiringTestV0(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func containsServerGoalStringV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func fakeClaudeGoalProcessCommandForTestV0(t *testing.T, root string, goalRef string) string {
	t.Helper()
	path := filepath.Join(root, "fake-claude-goal-process")
	script := "#!/bin/sh\nset -eu\nprompt=$(cat)\ncase \"$prompt\" in\n  *" + goalRef + "*) ;;\n  *) exit 7 ;;\nesac\nmkdir -p docs\ncat > docs/" + orquestaruntimeclaude.ClaudeGoalResultFileNameV0 + " <<'JSON'\n{\"schema_version\":\"orquesta_goal_result.v0\",\"status\":\"complete\",\"goal_ref\":\"" + goalRef + "\",\"summary\":\"server claude process complete\",\"artifact_refs\":[\"artifact-ref-server-claude-process-result\"],\"artifact_paths\":[\"docs/" + orquestaruntimeclaude.ClaudeGoalResultFileNameV0 + "\"],\"materialized_artifacts\":[{\"artifact_ref\":\"artifact-ref-server-claude-process-result\",\"path\":\"docs/" + orquestaruntimeclaude.ClaudeGoalResultFileNameV0 + "\",\"artifact_type\":\"goal_result\",\"status\":\"valid\",\"evidence_refs\":[\"evidence-ref-server-claude-process-result\"],\"issues\":[]}],\"checklist\":{\"expected_refs\":[\"server_claude_process\"],\"completed_refs\":[\"server_claude_process\"],\"missing_refs\":[],\"evidence_refs\":[\"evidence-ref-server-claude-process-result\"]},\"required_test_results\":[{\"test_ref\":\"required-test-ref-server-claude-process\",\"status\":\"passed\",\"evidence_refs\":[\"evidence-ref-server-claude-process-test\"]}],\"domain_receipt_refs\":[],\"rework_plan_refs\":[],\"evidence_refs\":[\"evidence-ref-server-claude-process-result\"]}\nJSON\n"
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake claude process: %v", err)
	}
	return path
}

func fakeClaudeGoalLongRunningCommandForTestV0(t *testing.T, root string, startedPath string) string {
	t.Helper()
	path := filepath.Join(root, "fake-claude-goal-long-running")
	script := "#!/bin/sh\nset -eu\ncat >/dev/null\nprintf started > " + shellQuoteCodexAppServerWiringTestV0(startedPath) + "\nsleep 30\n"
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake claude long running: %v", err)
	}
	return path
}

func fakeGeminiGoalProcessCommandForTestV0(t *testing.T, root string, goalRef string) string {
	t.Helper()
	path := filepath.Join(root, "fake-gemini-goal-process")
	script := "#!/bin/sh\nset -eu\nprompt=$(cat)\ncase \"$prompt\" in\n  *" + goalRef + "*) ;;\n  *) exit 7 ;;\nesac\nmkdir -p docs\ncat > docs/" + orquestaruntimegemini.GeminiGoalResultFileNameV0 + " <<'JSON'\n{\"schema_version\":\"orquesta_goal_result.v0\",\"status\":\"complete\",\"goal_ref\":\"" + goalRef + "\",\"summary\":\"server gemini process complete\",\"artifact_refs\":[\"artifact-ref-server-gemini-process-result\"],\"artifact_paths\":[\"docs/" + orquestaruntimegemini.GeminiGoalResultFileNameV0 + "\"],\"materialized_artifacts\":[{\"artifact_ref\":\"artifact-ref-server-gemini-process-result\",\"path\":\"docs/" + orquestaruntimegemini.GeminiGoalResultFileNameV0 + "\",\"artifact_type\":\"goal_result\",\"status\":\"valid\",\"evidence_refs\":[\"evidence-ref-server-gemini-process-result\"],\"issues\":[]}],\"checklist\":{\"expected_refs\":[\"server_gemini_process\"],\"completed_refs\":[\"server_gemini_process\"],\"missing_refs\":[],\"evidence_refs\":[\"evidence-ref-server-gemini-process-result\"]},\"required_test_results\":[{\"test_ref\":\"required-test-ref-server-gemini-process\",\"status\":\"passed\",\"evidence_refs\":[\"evidence-ref-server-gemini-process-test\"]}],\"domain_receipt_refs\":[],\"rework_plan_refs\":[],\"evidence_refs\":[\"evidence-ref-server-gemini-process-result\"]}\nJSON\n"
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake gemini process: %v", err)
	}
	return path
}

func fakeGeminiGoalLongRunningCommandForTestV0(t *testing.T, root string, startedPath string) string {
	t.Helper()
	path := filepath.Join(root, "fake-gemini-goal-long-running")
	script := "#!/bin/sh\nset -eu\ncat >/dev/null\nprintf started > " + shellQuoteCodexAppServerWiringTestV0(startedPath) + "\nsleep 30\n"
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake gemini long running: %v", err)
	}
	return path
}

func waitForServerGoalFileV0(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		raw, err := os.ReadFile(path)
		if err == nil && serverGoalFileReadyForTestV0(path, raw) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timeout esperando fichero %s", path)
}

func serverGoalFileReadyForTestV0(path string, raw []byte) bool {
	if filepath.Base(path) == orquestaruntimeclaude.ClaudeGoalResultFileNameV0 {
		return serverGoalResultFileCompleteForTestV0(raw)
	}
	return true
}

func serverGoalResultFileCompleteForTestV0(raw []byte) bool {
	var result struct {
		SchemaVersion string `json:"schema_version"`
		Status        string `json:"status"`
		GoalRef       string `json:"goal_ref"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return false
	}
	return strings.TrimSpace(result.SchemaVersion) != "" &&
		strings.TrimSpace(result.Status) != "" &&
		strings.TrimSpace(result.GoalRef) != ""
}

type fakeCodexAppServerProbeV0 struct{}

func (fakeCodexAppServerProbeV0) ProbeV0(context.Context) error {
	return nil
}

func TestCodexAppServerTmuxUnixServerHelperV0(t *testing.T) {
	if os.Getenv("CODEX_TEST_UNIX_SERVER_HELPER") != "1" {
		return
	}
	listener, err := net.Listen("unix", os.Args[len(os.Args)-1])
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	for {
		time.Sleep(time.Hour)
	}
}

func fakeCodexAppServerTmuxCommandForTestV0() string {
	return `#!/bin/sh
set -eu
if [ -n "${ORQUESTA_TEST_TMUX_LOG:-}" ]; then
  printf '%s\n' "$*" >> "$ORQUESTA_TEST_TMUX_LOG"
fi
case "${1:-}" in
  has-session)
    if [ -n "${ORQUESTA_TEST_TMUX_LOG:-}" ] && [ -e "${ORQUESTA_TEST_TMUX_LOG}.session" ]; then
      exit 0
    fi
	target="${3:-}"
	printf "can't find session: %s\n" "${target#=}" >&2
    exit 1
    ;;
  kill-session)
    if [ -n "${ORQUESTA_TEST_TMUX_LOG:-}" ]; then
	  if [ -e "${ORQUESTA_TEST_TMUX_LOG}.kill-fail" ]; then exit 2; fi
	  target="${3:-}"
	  if [ ! -f "${ORQUESTA_TEST_TMUX_LOG}.sid" ] || [ "$target" != "$(cat "${ORQUESTA_TEST_TMUX_LOG}.sid")" ]; then exit 1; fi
	  if [ -f "${ORQUESTA_TEST_TMUX_LOG}.pid" ]; then kill "$(cat "${ORQUESTA_TEST_TMUX_LOG}.pid")" 2>/dev/null || true; fi
      rm -f "${ORQUESTA_TEST_TMUX_LOG}.session"
    fi
    exit 0
    ;;
  display-message)
	printf '%s\t%s\t%s\n' "$(cat "${ORQUESTA_TEST_TMUX_LOG}.sid")" "$(cat "${ORQUESTA_TEST_TMUX_LOG}.created")" "$(cat "${ORQUESTA_TEST_TMUX_LOG}.pid")"
	exit 0
	;;
  show-environment)
	printf '%s=%s\n' 'ORQUESTA_CODEX_APP_SERVER_GENERATION_REF' "$(cat "${ORQUESTA_TEST_TMUX_LOG}.generation")"
	exit 0
    ;;
  new-session)
    if [ -n "${ORQUESTA_TEST_TMUX_LOG:-}" ]; then
      : > "${ORQUESTA_TEST_TMUX_LOG}.session"
    fi
    sock=""
	generation=""
    for arg in "$@"; do
      case "$arg" in
	    ORQUESTA_CODEX_APP_SERVER_GENERATION_REF=*)
	      generation="${arg#*=}"
	      ;;
        unix://*)
          sock="${arg#unix://}"
          ;;
        *unix://*)
          sock="${arg#*unix://}"
          sock="${sock%%\'*}"
          sock="${sock%%\"*}"
          sock="${sock%% *}"
          ;;
      esac
    done
    if [ -z "$sock" ]; then
      echo "socket missing" >&2
      exit 2
    fi
    mkdir -p "$(dirname "$sock")"
	CODEX_TEST_UNIX_SERVER_HELPER=1 "$ORQUESTA_TEST_BINARY" -test.run=^TestCodexAppServerTmuxUnixServerHelperV0$ -- "$sock" </dev/null >"${ORQUESTA_TEST_TMUX_LOG}.helper.log" 2>&1 &
	pid=$!
	printf '%s\n' "$pid" > "${ORQUESTA_TEST_TMUX_LOG}.pid"
	printf '%s\n' '$cmd-test' > "${ORQUESTA_TEST_TMUX_LOG}.sid"
	printf '%s\n' '100' > "${ORQUESTA_TEST_TMUX_LOG}.created"
	printf '%s\n' "$generation" > "${ORQUESTA_TEST_TMUX_LOG}.generation"
	i=0; while [ ! -S "$sock" ] && [ "$i" -lt 100 ]; do i=$((i+1)); sleep 0.01; done
	test -S "$sock"
	printf '%s\t%s\t%s\n' '$cmd-test' '100' "$pid"
	exit 0
    ;;
esac
echo "tmux args inesperados: $*" >&2
exit 2
`
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
	setGoalErr       error
	getGoalErr       error
	observedGoal     *serverCodexAppServerThreadGoalV0
	readThread       serverCodexAppServerThreadReadV0
	readThreadErr    error
	readThreadID     string
	readIncludeTurns bool
}

type fakeServerGoalActiveShutdownHookV0 struct {
	result orquestaservershutdown.ActiveShutdownWorkResultV0
}

func (hook fakeServerGoalActiveShutdownHookV0) ShutdownV0(context.Context) error {
	return nil
}

func (hook fakeServerGoalActiveShutdownHookV0) ReadActiveShutdownWorkV0(
	context.Context,
	orquestaservershutdown.ActiveShutdownWorkRequestV0,
) (orquestaservershutdown.ActiveShutdownWorkResultV0, error) {
	return hook.result, nil
}

func (hook fakeServerGoalActiveShutdownHookV0) CleanupActiveShutdownWorkV0(
	context.Context,
	orquestaservershutdown.ActiveShutdownWorkCleanupCommandV0,
) (orquestaservershutdown.ActiveShutdownWorkCleanupResultV0, error) {
	return orquestaservershutdown.ActiveShutdownWorkCleanupResultV0{}, nil
}

type fakeServerGoalWorkspacePortV0 struct {
	identity     string
	active       orquestaservershutdown.ActiveShutdownWorkResultV0
	readCalls    int
	cleanupCalls int
}

func (port *fakeServerGoalWorkspacePortV0) LaunchGoalWorkV0(
	context.Context,
	orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalLaunchReceiptV0, error) {
	return orquestagoal.GoalLaunchReceiptV0{}, nil
}

func (port *fakeServerGoalWorkspacePortV0) ObserveGoalWorkV0(
	context.Context,
	orquestagoal.GoalObservationRequestV0,
) (orquestagoal.GoalWorkResultV0, error) {
	return orquestagoal.GoalWorkResultV0{}, nil
}

func (port *fakeServerGoalWorkspacePortV0) ReadActiveShutdownWorkV0(
	context.Context,
	orquestaservershutdown.ActiveShutdownWorkRequestV0,
) (orquestaservershutdown.ActiveShutdownWorkResultV0, error) {
	port.readCalls++
	return port.active, nil
}

func (port *fakeServerGoalWorkspacePortV0) CleanupActiveShutdownWorkV0(
	context.Context,
	orquestaservershutdown.ActiveShutdownWorkCleanupCommandV0,
) (orquestaservershutdown.ActiveShutdownWorkCleanupResultV0, error) {
	port.cleanupCalls++
	return orquestaservershutdown.ActiveShutdownWorkCleanupResultV0{CleanedWorkCount: 1}, nil
}

func (port *fakeServerGoalWorkspacePortV0) ActiveShutdownWorkIdentityV0() string {
	return port.identity
}

func (fake *fakeCodexAppServerProtocolV0) StartThreadV0(
	_ context.Context,
	params serverCodexAppServerThreadStartParamsV0,
) (serverCodexAppServerThreadV0, error) {
	fake.calls = append(fake.calls, "thread/start")
	fake.startParams = params
	return fake.thread, nil
}

func (fake *fakeCodexAppServerProtocolV0) UpdateThreadSettingsV0(
	context.Context,
	serverCodexAppServerThreadSettingsUpdateParamsV0,
) error {
	return nil
}

func (fake *fakeCodexAppServerProtocolV0) SetGoalV0(
	_ context.Context,
	params serverCodexAppServerThreadGoalSetParamsV0,
) (serverCodexAppServerThreadGoalV0, error) {
	fake.calls = append(fake.calls, "thread/goal/set")
	fake.setParams = params
	if fake.setGoalErr != nil {
		return serverCodexAppServerThreadGoalV0{}, fake.setGoalErr
	}
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
	if fake.getGoalErr != nil {
		return nil, fake.getGoalErr
	}
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
	if fake.readThreadErr != nil {
		return serverCodexAppServerThreadReadV0{}, fake.readThreadErr
	}
	return fake.readThread, nil
}

var _ = orquestagoal.GoalStatusRunningV0
var _ = orquestaruntimecodexgoal.CodexGoalResultMarkerV0
