package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	orquestaruntimeclaude "orquesta/modulos/orquesta-runtime-claude"
	orquestaruntimecodexappserver "orquesta/modulos/orquesta-runtime-codex-appserver"
	orquestaruntimegemini "orquesta/modulos/orquesta-runtime-gemini"
	orquestaserver "orquesta/modulos/orquesta-server"
)

type serverCodexGoalBackendsV0 struct {
	AppGoal  serverCodexGoalBackendV0
	IdleGoal serverCodexGoalBackendV0
}

const serverCodexGoalBackendDegradedDiagnosticCodeV0 = "codex_goal_backend_degraded"

func serverCodexGoalBackendFromEnvV0(
	config orquestaserver.ConfigV0,
) (serverCodexGoalBackendV0, error) {
	return serverCodexGoalBackendFromEnvForWorkDirV0(
		config,
		firstNonEmptyServerStackV0(config.IdleSelfImprovementProjectWorkDir, config.ProjectWorkDir),
	)
}

func serverCodexGoalBackendsFromEnvV0(
	config orquestaserver.ConfigV0,
) (serverCodexGoalBackendsV0, error) {
	appGoal, err := serverCodexGoalBackendFromEnvForWorkDirV0(config, config.ProjectWorkDir)
	if err != nil {
		return serverCodexGoalBackendsV0{}, err
	}
	idleGoal, err := serverCodexGoalBackendFromEnvForWorkDirV0(
		config,
		firstNonEmptyServerStackV0(config.IdleSelfImprovementProjectWorkDir, config.ProjectWorkDir),
	)
	if err != nil {
		return serverCodexGoalBackendsV0{}, err
	}
	return serverCodexGoalBackendsV0{
		AppGoal:  appGoal,
		IdleGoal: idleGoal,
	}, nil
}

func serverCodexGoalBackendFromEnvForWorkDirV0(
	config orquestaserver.ConfigV0,
	workDir string,
) (serverCodexGoalBackendV0, error) {
	backend := codexGoalBackendFromEnvV0()
	claudeBackend := claudeGoalBackendFromEnvV0()
	if claudeBackend != "" {
		return serverClaudeGoalBackendFromEnvForWorkDirV0(config, workDir)
	}
	geminiBackend := geminiGoalBackendFromEnvV0()
	if geminiBackend != "" {
		return serverGeminiGoalBackendFromEnvForWorkDirV0(config, workDir)
	}
	if backend == "" {
		return serverCodexGoalBackendV0{}, nil
	}
	if backend != orquestaruntimecodexappserver.CodexGoalBackendAppServerProxyV0 &&
		backend != orquestaruntimecodexappserver.CodexGoalBackendAppServerTmuxV0 &&
		backend != claudeGoalBackendFileControlV0 &&
		backend != claudeGoalBackendProcessV0 &&
		backend != geminiGoalBackendFileControlV0 &&
		backend != geminiGoalBackendProcessV0 {
		return serverCodexGoalBackendV0{}, fmt.Errorf("codex_goal_backend_no_soportado:%s", backend)
	}
	if backend == orquestaruntimecodexappserver.CodexGoalBackendAppServerProxyV0 && !codexGoalBackendProxyDiagnosticAllowedV0() {
		return serverCodexGoalBackendV0{}, fmt.Errorf("codex_goal_backend_proxy_diagnostic_opt_in_required:%s", envAllowAppServerProxyDiagnosticV0)
	}
	if backend == orquestaruntimecodexappserver.CodexGoalBackendAppServerProxyV0 {
		return serverCodexGoalBackendV0{}, fmt.Errorf("codex_goal_backend_proxy_diagnostic_not_operational")
	}
	runtimeConfig := codexRuntimeEnvConfigFromEnvV0()
	commandProtocol := orquestaruntimecodexappserver.CommandProtocolV0{
		CommandPath: runtimeConfig.CommandPath,
		Args:        codexGoalBackendArgsV0(backend),
		PathEnv:     runtimeConfig.PathEnv,
		Timeout:     time.Duration(codexGoalTimeoutMSFromEnvV0()) * time.Millisecond,
	}
	var runtimeProtocol orquestaruntimecodexappserver.ProtocolPortV0 = commandProtocol
	commandPreflightProtocol := commandProtocol
	commandPreflightProtocol.Timeout = time.Duration(codexGoalPreflightTimeoutMSFromEnvV0()) * time.Millisecond
	var preflightProtocol orquestaruntimecodexappserver.ProbePortV0 = commandPreflightProtocol
	preflightAtStartup := true
	authCheckedAtStartup := false
	var shutdownHook orquestaserver.RuntimeShutdownHookPortV0
	codeHomePath := ""
	authIssueCode := ""
	if backend == orquestaruntimecodexappserver.CodexGoalBackendAppServerTmuxV0 {
		appServerConfig := codexAppServerConfigFromServerConfigV0(config)
		socketPath, err := orquestaruntimecodexappserver.CodexAppServerTmuxSocketPathV0(appServerConfig)
		if err != nil {
			degraded := orquestaruntimecodexappserver.UnavailableGoalBackendV0{
				IssueCode: orquestaruntimecodexappserver.CodexAppServerIssueCodeForErrorV0(err, "codex_app_server_tmux_unavailable"),
			}
			return serverCodexGoalBackendV0{Starter: degraded, Observer: degraded}, nil
		}
		tmuxCodeHomePath, err := orquestaruntimecodexappserver.CodexAppServerTmuxCodeHomePathV0(appServerConfig)
		if err != nil {
			degraded := orquestaruntimecodexappserver.UnavailableGoalBackendV0{
				IssueCode: orquestaruntimecodexappserver.CodexAppServerIssueCodeForErrorV0(err, "codex_app_server_tmux_unavailable"),
			}
			return serverCodexGoalBackendV0{Starter: degraded, Observer: degraded}, nil
		}
		codeHomePath = tmuxCodeHomePath
		tmuxBackend := orquestaruntimecodexappserver.TmuxBackendV0{
			CommandPath:       runtimeConfig.CommandPath,
			PathEnv:           runtimeConfig.PathEnv,
			SocketPath:        socketPath,
			SessionName:       orquestaruntimecodexappserver.CodexAppServerTmuxSessionNameV0(appServerConfig),
			HomeDir:           runtimeConfig.HomeDir,
			CodeHomeDir:       tmuxCodeHomePath,
			RuntimeWorkDir:    config.RuntimeWorkDir,
			ProjectWorkDir:    firstNonEmptyServerStackV0(workDir, config.ProjectWorkDir),
			SourceCodeHomeDir: runtimeConfig.CodeHomeDir,
			Timeout:           orquestaruntimecodexappserver.CodexAppServerTmuxStartupTimeoutV0(time.Duration(codexGoalPreflightTimeoutMSFromEnvV0()) * time.Millisecond),
		}
		shutdownTmuxBackend := tmuxBackend
		shutdownTmuxBackend.Timeout = time.Duration(codexGoalPreflightTimeoutMSFromEnvV0()) * time.Millisecond
		shutdownHook = shutdownTmuxBackend
		authIssueCode = codexAppServerAuthIssueCodeFromDirsV0(runtimeConfig.CodeHomeDir, tmuxCodeHomePath)
		authCheckedAtStartup = true
		if authIssueCode != "" {
			degraded := orquestaruntimecodexappserver.UnavailableGoalBackendV0{IssueCode: authIssueCode}
			return serverCodexGoalBackendV0{Starter: degraded, Observer: degraded, ShutdownHook: shutdownHook}, nil
		}
		websocketProtocol := orquestaruntimecodexappserver.WebSocketProtocolV0{
			SocketPath:        socketPath,
			Timeout:           time.Duration(codexGoalTimeoutMSFromEnvV0()) * time.Millisecond,
			DiagnosticLogPath: orquestaruntimecodexappserver.CodexAppServerTmuxLogPathV0(tmuxBackend),
		}
		tmuxPreflightProtocol := orquestaruntimecodexappserver.WebSocketProtocolV0{
			SocketPath:        socketPath,
			Timeout:           time.Duration(codexGoalPreflightTimeoutMSFromEnvV0()) * time.Millisecond,
			DiagnosticLogPath: websocketProtocol.DiagnosticLogPath,
		}
		runtimeProtocol = orquestaruntimecodexappserver.LazyTmuxProtocolV0{
			Backend:   tmuxBackend,
			Inner:     websocketProtocol,
			Preflight: tmuxPreflightProtocol,
		}
		preflightProtocol = tmuxPreflightProtocol
		preflightAtStartup = false
	}
	if preflightAtStartup {
		if err := preflightProtocol.ProbeV0(context.Background()); err != nil {
			degraded := orquestaruntimecodexappserver.UnavailableGoalBackendV0{
				IssueCode: orquestaruntimecodexappserver.CodexAppServerIssueCodeForErrorV0(err, "codex_app_server_unavailable"),
			}
			return serverCodexGoalBackendV0{Starter: degraded, Observer: degraded}, nil
		}
	}
	if !authCheckedAtStartup {
		authIssueCode = codexAppServerAuthIssueCodeV0(codeHomePath)
	}
	if authIssueCode != "" {
		degraded := orquestaruntimecodexappserver.UnavailableGoalBackendV0{
			IssueCode: authIssueCode,
		}
		return serverCodexGoalBackendV0{Starter: degraded, Observer: degraded, ShutdownHook: shutdownHook}, nil
	}
	client := orquestaruntimecodexappserver.GoalBackendV0{
		Protocol:          runtimeProtocol,
		BackendShutdown:   shutdownHook,
		CWD:               firstNonEmptyServerStackV0(workDir, config.ProjectWorkDir),
		DiagnosticLogPath: orquestaruntimecodexappserver.DiagnosticLogPathForProtocolV0(runtimeProtocol),
		AuthIssueCode:     authIssueCode,
		Model:             runtimeConfig.Model,
		ReasoningEffort:   runtimeConfig.ReasoningEffort,
		Sandbox:           runtimeConfig.Sandbox,
		ApprovalPolicy:    runtimeConfig.ApprovalPolicy,
		Timeout:           time.Duration(codexGoalTimeoutMSFromEnvV0()) * time.Millisecond,
		HighTokenUsageThreshold: intEnvOrDefaultV0(
			envAutoprogrammingCheckpointOnlyHighConsumptionTokensV0,
			0,
		),
		Runtime: &orquestaruntimecodexappserver.GoalRuntimeV0{},
	}
	return serverCodexGoalBackendV0{
		Starter:      client,
		Observer:     client,
		Controller:   client,
		ShutdownHook: shutdownHook,
	}, nil
}

func codexAppServerConfigFromServerConfigV0(config orquestaserver.ConfigV0) orquestaruntimecodexappserver.ConfigV0 {
	return orquestaruntimecodexappserver.ConfigV0{
		RuntimeWorkDir: config.RuntimeWorkDir,
		ProjectWorkDir: config.ProjectWorkDir,
	}
}

func codexAppServerAuthIssueCodeV0(codeHomeDir string) string {
	return codexAppServerAuthIssueCodeFromDirsV0(codeHomeDir)
}

func codexAppServerAuthIssueCodeFromDirsV0(codeHomeDirs ...string) string {
	if codexAppServerAuthEnvPresentV0() {
		return ""
	}
	checked := false
	for _, codeHomeDir := range codeHomeDirs {
		codeHomeDir = strings.TrimSpace(codeHomeDir)
		if codeHomeDir == "" {
			continue
		}
		checked = true
		info, err := os.Stat(filepath.Join(codeHomeDir, "auth.json"))
		if err == nil && !info.IsDir() && info.Size() > 0 {
			return ""
		}
	}
	if !checked {
		return ""
	}
	return "codex_app_server_auth_missing"
}

func codexAppServerAuthEnvPresentV0() bool {
	for _, name := range []string{"OPENAI_API_KEY", "CODEX_API_KEY"} {
		if strings.TrimSpace(os.Getenv(name)) != "" {
			return true
		}
	}
	return false
}

func codexGoalBackendFromEnvV0() string {
	return strings.TrimSpace(os.Getenv(envCodexGoalBackendV0))
}

func codexGoalBackendOperationalFromEnvV0() bool {
	return codexGoalBackendFromEnvV0() == codexGoalBackendAppServerTmuxV0
}

func serverClaudeGoalBackendFromEnvForWorkDirV0(
	config orquestaserver.ConfigV0,
	workDir string,
) (serverCodexGoalBackendV0, error) {
	backend := claudeGoalBackendFromEnvV0()
	if backend == "" {
		return serverCodexGoalBackendV0{}, nil
	}
	if backend != claudeGoalBackendFileControlV0 && backend != claudeGoalBackendProcessV0 {
		return serverCodexGoalBackendV0{}, fmt.Errorf("claude_goal_backend_no_soportado:%s", backend)
	}
	projectWorkDir := firstNonEmptyServerStackV0(workDir, config.ProjectWorkDir)
	runtimeWorkDir := claudeGoalRuntimeWorkDirFromEnvV0(config)
	control := orquestaruntimeclaude.ClaudeGoalBackendV0{
		ProjectWorkDir: projectWorkDir,
		RuntimeWorkDir: runtimeWorkDir,
	}
	if backend == claudeGoalBackendProcessV0 {
		client := &orquestaruntimeclaude.ClaudeGoalProcessBackendV0{
			Control: control,
			Profile: orquestaruntimeclaude.ClaudeConnectorProfileV0{
				SchemaVersion:  orquestaruntimeclaude.ClaudeConnectorProfileSchemaVersionV0,
				OptIn:          true,
				CommandPath:    claudeCommandPathV0(),
				ProjectWorkDir: projectWorkDir,
				RuntimeWorkDir: runtimeWorkDir,
				HomeDir:        strings.TrimSpace(os.Getenv(envClaudeHomeV0)),
				PathEnv:        envOrDefaultV0(envClaudePathV0, os.Getenv("PATH")),
				Model:          strings.TrimSpace(os.Getenv(envClaudeModelV0)),
				PermissionMode: envOrDefaultV0(envClaudePermissionModeV0, "bypassPermissions"),
				OutputFormat:   envOrDefaultV0(envClaudeOutputFormatV0, "text"),
				Effort:         strings.TrimSpace(os.Getenv(envClaudeEffortV0)),
				ExtraArgs:      strings.Fields(os.Getenv(envClaudeExtraArgsV0)),
				PromptHints:    []string{"goal-first Claude process backend opt-in"},
			},
		}
		return serverCodexGoalBackendV0{
			GoalLauncher:  client,
			GoalObserver:  client,
			ClaudeControl: client,
		}, nil
	}
	client := control
	return serverCodexGoalBackendV0{
		GoalLauncher: client,
		GoalObserver: client,
	}, nil
}

func claudeGoalRuntimeWorkDirFromEnvV0(config orquestaserver.ConfigV0) string {
	if strings.TrimSpace(os.Getenv(envClaudeRuntimeWorkDirV0)) != "" {
		return absDirEnvOrDefaultV0(envClaudeRuntimeWorkDirV0, config.RuntimeWorkDir)
	}
	stateDir := strings.TrimSpace(config.StateDir)
	if stateDir != "" && filepath.IsAbs(stateDir) {
		return filepath.Join(filepath.Dir(stateDir), "claude-goal")
	}
	return filepath.Join(filepath.Dir(config.RuntimeWorkDir), ".orquesta-claude-goal")
}

func serverGeminiGoalBackendFromEnvForWorkDirV0(
	config orquestaserver.ConfigV0,
	workDir string,
) (serverCodexGoalBackendV0, error) {
	backend := geminiGoalBackendFromEnvV0()
	if backend == "" {
		return serverCodexGoalBackendV0{}, nil
	}
	if backend != geminiGoalBackendFileControlV0 && backend != geminiGoalBackendProcessV0 {
		return serverCodexGoalBackendV0{}, fmt.Errorf("gemini_goal_backend_no_soportado:%s", backend)
	}
	projectWorkDir := firstNonEmptyServerStackV0(workDir, config.ProjectWorkDir)
	runtimeWorkDir := geminiGoalRuntimeWorkDirFromEnvV0(config)
	control := orquestaruntimegemini.GeminiGoalBackendV0{
		ProjectWorkDir: projectWorkDir,
		RuntimeWorkDir: runtimeWorkDir,
	}
	if backend == geminiGoalBackendProcessV0 {
		client := &orquestaruntimegemini.GeminiGoalProcessBackendV0{
			Control: control,
			Profile: orquestaruntimegemini.GeminiConnectorProfileV0{
				SchemaVersion:  orquestaruntimegemini.GeminiConnectorProfileSchemaVersionV0,
				OptIn:          true,
				CommandPath:    geminiCommandPathV0(),
				ProjectWorkDir: projectWorkDir,
				RuntimeWorkDir: runtimeWorkDir,
				HomeDir:        strings.TrimSpace(os.Getenv(envGeminiHomeV0)),
				PathEnv:        envOrDefaultV0(envGeminiPathV0, os.Getenv("PATH")),
				Model:          strings.TrimSpace(os.Getenv(envGeminiModelV0)),
				ApprovalMode:   envOrDefaultV0(envGeminiApprovalModeV0, "auto_edit"),
				OutputFormat:   envOrDefaultV0(envGeminiOutputFormatV0, "text"),
				ExtraArgs:      strings.Fields(os.Getenv(envGeminiExtraArgsV0)),
				PromptHints:    []string{"goal-first Gemini process backend opt-in"},
			},
		}
		return serverCodexGoalBackendV0{
			GoalLauncher:  client,
			GoalObserver:  client,
			GeminiControl: client,
		}, nil
	}
	client := control
	return serverCodexGoalBackendV0{
		GoalLauncher: client,
		GoalObserver: client,
	}, nil
}

func geminiGoalRuntimeWorkDirFromEnvV0(config orquestaserver.ConfigV0) string {
	if strings.TrimSpace(os.Getenv(envGeminiRuntimeWorkDirV0)) != "" {
		return absDirEnvOrDefaultV0(envGeminiRuntimeWorkDirV0, config.RuntimeWorkDir)
	}
	stateDir := strings.TrimSpace(config.StateDir)
	if stateDir != "" && filepath.IsAbs(stateDir) {
		return filepath.Join(filepath.Dir(stateDir), "gemini-goal")
	}
	return filepath.Join(filepath.Dir(config.RuntimeWorkDir), ".orquesta-gemini-goal")
}

func serverGoalBackendOperationalFromEnvV0() bool {
	return codexGoalBackendOperationalFromEnvV0() ||
		claudeGoalBackendOperationalFromEnvV0() ||
		geminiGoalBackendOperationalFromEnvV0()
}

func serverGoalBackendDerivationSourceV0() string {
	if codexGoalBackendOperationalFromEnvV0() {
		return "derived_from_codex_goal_backend"
	}
	if claudeGoalBackendOperationalFromEnvV0() {
		return "derived_from_claude_goal_backend"
	}
	if geminiGoalBackendOperationalFromEnvV0() {
		return "derived_from_gemini_goal_backend"
	}
	return ""
}

func serverGoalBackendDerivationEnvV0() string {
	if codexGoalBackendOperationalFromEnvV0() {
		return envCodexGoalBackendV0
	}
	if claudeGoalBackendOperationalFromEnvV0() {
		return envCodexGoalBackendV0
	}
	if geminiGoalBackendOperationalFromEnvV0() {
		return envCodexGoalBackendV0
	}
	return ""
}

func serverConfigWithCodexGoalBackendDiagnosticsV0(
	config orquestaserver.ConfigV0,
	backends serverCodexGoalBackendsV0,
) orquestaserver.ConfigV0 {
	diagnostics := append([]orquestaserver.ServerDiagnosticV0(nil), config.EffectiveConfig.Diagnostics...)
	diagnostics = append(diagnostics, serverCodexGoalBackendDiagnosticV0("app_goal", backends.AppGoal)...)
	diagnostics = append(diagnostics, serverCodexGoalBackendDiagnosticV0("idle_goal", backends.IdleGoal)...)
	config.EffectiveConfig.Diagnostics = diagnostics
	config.EffectiveConfig = orquestaserver.NormalizeServerEffectiveConfigV0(config.EffectiveConfig)
	return config
}

func serverCodexGoalBackendDiagnosticV0(
	scope string,
	backend serverCodexGoalBackendV0,
) []orquestaserver.ServerDiagnosticV0 {
	issueCode := serverCodexGoalBackendUnavailableIssueCodeV0(backend)
	if issueCode == "" {
		return nil
	}
	scope = strings.TrimSpace(scope)
	if scope == "" {
		scope = "codex_goal"
	}
	return []orquestaserver.ServerDiagnosticV0{{
		Code:    serverCodexGoalBackendDegradedDiagnosticCodeV0,
		Scope:   scope,
		Message: serverCodexGoalBackendDiagnosticMessageV0(issueCode),
		EvidenceRefs: []string{
			"evidence-ref-server-codex-goal-backend-degraded-" + scope,
		},
	}}
}

func serverCodexGoalBackendUnavailableIssueCodeV0(backend serverCodexGoalBackendV0) string {
	if unavailable, ok := backend.Starter.(serverCodexUnavailableGoalBackendV0); ok {
		return strings.TrimSpace(unavailable.IssueCode)
	}
	if unavailable, ok := backend.Observer.(serverCodexUnavailableGoalBackendV0); ok {
		return strings.TrimSpace(unavailable.IssueCode)
	}
	if unavailable, ok := backend.Starter.(orquestaruntimecodexappserver.UnavailableGoalBackendV0); ok {
		return strings.TrimSpace(unavailable.IssueCode)
	}
	if unavailable, ok := backend.Observer.(orquestaruntimecodexappserver.UnavailableGoalBackendV0); ok {
		return strings.TrimSpace(unavailable.IssueCode)
	}
	return ""
}

func serverCodexGoalBackendDiagnosticMessageV0(issueCode string) string {
	issueCode = strings.TrimSpace(issueCode)
	if issueCode == "codex_app_server_wrapper_stdio_failed" {
		return "codex goal backend degradado: codex_app_server_wrapper_stdio_failed; accion: configura ORQUESTA_CODEX_COMMAND con el binario nativo de Codex y reinicia"
	}
	if issueCode == "codex_app_server_provider_unauthorized" {
		return "codex goal backend degradado: codex_app_server_provider_unauthorized; accion: revisa autenticacion de Codex/OpenAI en el CODEX_HOME aislado y reinicia"
	}
	if issueCode == "codex_app_server_auth_missing" {
		return "codex goal backend degradado: codex_app_server_auth_missing; accion: ejecuta login de Codex o proyecta auth.json al CODEX_HOME aislado y reinicia"
	}
	if issueCode == "" {
		issueCode = "codex_app_server_unavailable"
	}
	return "codex goal backend degradado: " + issueCode
}

func codexGoalBackendProxyDiagnosticAllowedV0() bool {
	return boolEnvOrDefaultV0(envAllowAppServerProxyDiagnosticV0, false)
}

func codexGoalBackendArgsV0(backend string) []string {
	switch strings.TrimSpace(backend) {
	case codexGoalBackendAppServerTmuxV0:
		return nil
	case codexGoalBackendAppServerProxyV0:
		return []string{"app-server", "proxy"}
	default:
		return nil
	}
}

func codexGoalBackendProxyArgsForSocketV0(socketPath string) []string {
	socketPath = strings.TrimSpace(socketPath)
	if socketPath == "" {
		return []string{"app-server", "proxy"}
	}
	return []string{"app-server", "proxy", "--sock", socketPath}
}
