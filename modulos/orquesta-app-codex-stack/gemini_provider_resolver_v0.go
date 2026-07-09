package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimeclaude "orquesta/modulos/orquesta-runtime-claude"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
	orquestaruntimegemini "orquesta/modulos/orquesta-runtime-gemini"
)

const geminiVisualAreaV0 = "visual"
const claudeReviewAreaV0 = "revision"

type providerLaunchSpecResolverV0 struct {
	Codex  CodexLaunchSpecResolverV0
	Gemini GeminiLaunchSpecResolverV0
	Claude ClaudeLaunchSpecResolverV0
}

var _ orquestacionnucleoapp.ExternalAgentLaunchSpecResolverPortV0 = providerLaunchSpecResolverV0{}

func (resolver providerLaunchSpecResolverV0) ResolveExternalAgentLaunchSpecV0(
	ctx context.Context,
	inbound orquestaruntime.AgentLauncherInboundV0,
) (orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0, error) {
	useGemini, err := resolver.Gemini.ShouldHandleInboundV0(ctx, inbound)
	if err != nil {
		return orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0{}, err
	}
	if useGemini {
		return resolver.Gemini.ResolveExternalAgentLaunchSpecV0(ctx, inbound)
	}
	useClaude, err := resolver.Claude.ShouldHandleInboundV0(ctx, inbound)
	if err != nil {
		return orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0{}, err
	}
	if useClaude {
		return resolver.Claude.ResolveExternalAgentLaunchSpecV0(ctx, inbound)
	}
	return resolver.Codex.ResolveExternalAgentLaunchSpecV0(ctx, inbound)
}

type GeminiLaunchSpecResolverV0 struct {
	Config         GeminiRuntimeConfigV0
	CodexConfig    CodexRuntimeConfigV0
	TaskStore      orquestacionnucleoapp.WorkflowTaskStorePortV0
	AppChangeStore orquestaappchange.AppChangeRecordSourcePortV0
}

var _ orquestacionnucleoapp.ExternalAgentLaunchSpecResolverPortV0 = GeminiLaunchSpecResolverV0{}

func (resolver GeminiLaunchSpecResolverV0) ShouldHandleInboundV0(
	ctx context.Context,
	inbound orquestaruntime.AgentLauncherInboundV0,
) (bool, error) {
	if !resolver.Config.Enabled ||
		inbound.Payload == nil ||
		!isProgrammingPhaseV0(inbound.Payload.PhaseID) ||
		resolver.TaskStore == nil ||
		resolver.AppChangeStore == nil {
		return false, nil
	}
	task, ok, err := resolver.workflowTaskForPayloadV0(ctx, *inbound.Payload)
	if err != nil || !ok || !workflowTaskHasDomainWorkContractV0(task) {
		return false, err
	}
	work, ok, err := resolver.externalWorkForTaskV0(ctx, inbound.Payload.RunID, task.TaskID)
	if err != nil || !ok {
		return false, err
	}
	return geminiShouldHandleDomainWorkV0(work.WorkKind), nil
}

func (resolver GeminiLaunchSpecResolverV0) ResolveExternalAgentLaunchSpecV0(
	ctx context.Context,
	inbound orquestaruntime.AgentLauncherInboundV0,
) (orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0, error) {
	if inbound.Payload == nil {
		return orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0{}, fmt.Errorf("payload requerido")
	}
	agentRef := strings.TrimSpace(inbound.Payload.AgentRequestID)
	runtimeDir, err := orquestaruntimecodexdelivery.CodexReceiptAgentRuntimeDirV0(
		geminiRuntimeWorkDirV0(resolver.Config, resolver.CodexConfig),
		inbound.Payload.RunID,
		agentRef,
	)
	if err != nil {
		return orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0{}, err
	}
	helper := resolver.codexHelperV0()
	task, err := helper.agentTaskV0(ctx, *inbound.Payload, geminiVisualAreaV0)
	if err != nil {
		return orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0{}, err
	}
	if err := helper.ensureLaunchWorktreeIsolationV0(ctx, *inbound.Payload); err != nil {
		return orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0{}, err
	}
	contextBundle, err := helper.agentContextV0(ctx, *inbound.Payload, geminiVisualAreaV0, task)
	if err != nil {
		return orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0{}, err
	}
	contextBundle = sanitizeCodexStackAgentContextV0(contextBundle, resolver.CodexConfig.ContextSanitizer)
	task = taskWithContextGuardV0(task, contextBundle)
	spec := helper.launchSpecV0(
		agentRef,
		strings.TrimSpace(inbound.CorrelationID),
		strings.TrimSpace(inbound.Payload.PhaseID),
		geminiVisualAreaV0,
		task,
	)
	spec.ProfileRef = "profile-ref-app-stack-gemini-visual"
	spec.ConnectorRef = "connector-ref-app-stack-gemini-visual"
	spec.Command.CommandRef = "command-ref-app-stack-gemini-visual"
	spec.Command.ExecutableRef = "exec-ref-app-stack-gemini-visual"
	spec.Command.WorkingDirRef = "workdir-ref-app-stack-gemini-visual"
	spec.AgentPacket.Context = contextBundle
	spec.AgentPacket.Policies = packetPoliciesWithContextGuardV0(spec.AgentPacket.Policies, contextBundle)
	profile := geminiProfileV0(resolver.Config, resolver.CodexConfig, runtimeDir)
	return orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0{
		Spec:            spec,
		CommandResolver: orquestaruntimegemini.NewGeminiExecResolverV0(profile),
	}, nil
}

func (resolver GeminiLaunchSpecResolverV0) codexHelperV0() CodexLaunchSpecResolverV0 {
	config := resolver.CodexConfig
	config.ProjectWorkDir = geminiProjectWorkDirV0(resolver.Config, resolver.CodexConfig)
	config.RuntimeWorkDir = geminiRuntimeWorkDirV0(resolver.Config, resolver.CodexConfig)
	return CodexLaunchSpecResolverV0{
		Config:         config,
		TaskStore:      resolver.TaskStore,
		AppChangeStore: resolver.AppChangeStore,
	}
}

func (resolver GeminiLaunchSpecResolverV0) workflowTaskForPayloadV0(
	ctx context.Context,
	payload orquestaruntime.LaunchRuntimeAgentRequestV0,
) (orquestacoreworkflow.WorkflowTaskV0, bool, error) {
	tasks, err := resolver.TaskStore.LoadWorkflowTasksV0(
		ctx,
		strings.TrimSpace(payload.RunID),
		[]string{strings.TrimSpace(payload.TaskRef)},
	)
	if err != nil || len(tasks) != 1 {
		return orquestacoreworkflow.WorkflowTaskV0{}, false, err
	}
	return tasks[0], true, nil
}

func (resolver GeminiLaunchSpecResolverV0) externalWorkForTaskV0(
	ctx context.Context,
	runRef string,
	taskRef string,
) (*orquestaappchange.AppChangeExternalWorkV0, bool, error) {
	return resolver.codexHelperV0().externalWorkForTaskV0(ctx, runRef, taskRef)
}

func geminiProfileV0(
	config GeminiRuntimeConfigV0,
	codexConfig CodexRuntimeConfigV0,
	runtimeDir string,
) orquestaruntimegemini.GeminiConnectorProfileV0 {
	return orquestaruntimegemini.GeminiConnectorProfileV0{
		SchemaVersion:  orquestaruntimegemini.GeminiConnectorProfileSchemaVersionV0,
		OptIn:          true,
		CommandPath:    strings.TrimSpace(config.CommandPath),
		ProjectWorkDir: geminiProjectWorkDirV0(config, codexConfig),
		RuntimeWorkDir: strings.TrimSpace(runtimeDir),
		HomeDir:        strings.TrimSpace(config.HomeDir),
		PathEnv:        strings.TrimSpace(config.PathEnv),
		Model:          strings.TrimSpace(config.Model),
		ApprovalMode:   strings.TrimSpace(config.ApprovalMode),
		OutputFormat:   strings.TrimSpace(config.OutputFormat),
		PromptLocale:   strings.TrimSpace(config.PromptLocale),
		ExtraArgs:      append([]string(nil), config.ExtraArgs...),
		PromptHints:    geminiPromptHintsV0(config.PromptHints),
	}
}

func geminiProjectWorkDirV0(config GeminiRuntimeConfigV0, codexConfig CodexRuntimeConfigV0) string {
	if strings.TrimSpace(config.ProjectWorkDir) != "" {
		return strings.TrimSpace(config.ProjectWorkDir)
	}
	return strings.TrimSpace(codexConfig.ProjectWorkDir)
}

func geminiRuntimeWorkDirV0(config GeminiRuntimeConfigV0, codexConfig CodexRuntimeConfigV0) string {
	if strings.TrimSpace(config.RuntimeWorkDir) != "" {
		return strings.TrimSpace(config.RuntimeWorkDir)
	}
	return strings.TrimSpace(codexConfig.RuntimeWorkDir)
}

func geminiPromptHintsV0(values []string) []string {
	return compactStringsV0(append([]string{
		"Trabajo visual neutral lanzado por Orquesta; no llames REST ni APIs de dominio.",
		"Entrega el recurso como fichero de producto dentro del write-set y declaralo en ACK.files.",
		"Si tu canal Gemini puede generar raster profesional, entrega PNG/JPG/WebP comprimible con alt_text y evita bocetos, clip-art, cajas/flechas simples o diagramas generados por codigo.",
		"Si este canal Gemini es solo CLI textual y no puede crear imagen raster real, entrega brief_visual o agent_review_report; no presentes SVG, Mermaid, HTML simple, PIL/canvas o diagramas rapidos como arte_final_visual.",
		"Si el trabajo es una revision OPES asignada a Gemini, entrega agent_review_report o agent_pair_review_report con hallazgos visuales, UX, accesibilidad, riesgos y rework causal.",
	}, values...))
}

func geminiShouldHandleDomainWorkV0(workKind string) bool {
	switch strings.TrimSpace(workKind) {
	case "review_gemini", "review_pair_codex_gemini",
		"generate_agent_candidate_gemini", "vote_agent_candidates_gemini":
		return true
	default:
		return domainWorkArtifactTypeForWorkKindV0(workKind) ==
			orquestadomainwork.DomainWorkArtifactTypeVisualAssetV0
	}
}

type ClaudeLaunchSpecResolverV0 struct {
	Config         ClaudeRuntimeConfigV0
	CodexConfig    CodexRuntimeConfigV0
	TaskStore      orquestacionnucleoapp.WorkflowTaskStorePortV0
	AppChangeStore orquestaappchange.AppChangeRecordSourcePortV0
}

var _ orquestacionnucleoapp.ExternalAgentLaunchSpecResolverPortV0 = ClaudeLaunchSpecResolverV0{}

func (resolver ClaudeLaunchSpecResolverV0) ShouldHandleInboundV0(
	ctx context.Context,
	inbound orquestaruntime.AgentLauncherInboundV0,
) (bool, error) {
	if !resolver.Config.Enabled ||
		inbound.Payload == nil ||
		!isProgrammingPhaseV0(inbound.Payload.PhaseID) ||
		resolver.TaskStore == nil ||
		resolver.AppChangeStore == nil {
		return false, nil
	}
	task, ok, err := resolver.workflowTaskForPayloadV0(ctx, *inbound.Payload)
	if err != nil || !ok || !workflowTaskHasDomainWorkContractV0(task) {
		return false, err
	}
	work, ok, err := resolver.externalWorkForTaskV0(ctx, inbound.Payload.RunID, task.TaskID)
	if err != nil || !ok {
		return false, err
	}
	return claudeShouldHandleDomainWorkV0(work.WorkKind), nil
}

func (resolver ClaudeLaunchSpecResolverV0) ResolveExternalAgentLaunchSpecV0(
	ctx context.Context,
	inbound orquestaruntime.AgentLauncherInboundV0,
) (orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0, error) {
	if inbound.Payload == nil {
		return orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0{}, fmt.Errorf("payload requerido")
	}
	agentRef := strings.TrimSpace(inbound.Payload.AgentRequestID)
	runtimeDir, err := orquestaruntimecodexdelivery.CodexReceiptAgentRuntimeDirV0(
		claudeRuntimeWorkDirV0(resolver.Config, resolver.CodexConfig),
		inbound.Payload.RunID,
		agentRef,
	)
	if err != nil {
		return orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0{}, err
	}
	helper := resolver.codexHelperV0()
	task, err := helper.agentTaskV0(ctx, *inbound.Payload, claudeReviewAreaV0)
	if err != nil {
		return orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0{}, err
	}
	if err := helper.ensureLaunchWorktreeIsolationV0(ctx, *inbound.Payload); err != nil {
		return orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0{}, err
	}
	contextBundle, err := helper.agentContextV0(ctx, *inbound.Payload, claudeReviewAreaV0, task)
	if err != nil {
		return orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0{}, err
	}
	contextBundle = sanitizeCodexStackAgentContextV0(contextBundle, resolver.CodexConfig.ContextSanitizer)
	task = taskWithContextGuardV0(task, contextBundle)
	spec := helper.launchSpecV0(
		agentRef,
		strings.TrimSpace(inbound.CorrelationID),
		strings.TrimSpace(inbound.Payload.PhaseID),
		claudeReviewAreaV0,
		task,
	)
	spec.ProfileRef = "profile-ref-app-stack-claude-review"
	spec.ConnectorRef = "connector-ref-app-stack-claude-review"
	spec.Command.CommandRef = "command-ref-app-stack-claude-review"
	spec.Command.ExecutableRef = "exec-ref-app-stack-claude-review"
	spec.Command.WorkingDirRef = "workdir-ref-app-stack-claude-review"
	spec.AgentPacket.Context = contextBundle
	spec.AgentPacket.Policies = packetPoliciesWithContextGuardV0(spec.AgentPacket.Policies, contextBundle)
	profile := claudeProfileV0(resolver.Config, resolver.CodexConfig, runtimeDir)
	return orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0{
		Spec:            spec,
		CommandResolver: orquestaruntimeclaude.NewClaudeExecResolverV0(profile),
	}, nil
}

func (resolver ClaudeLaunchSpecResolverV0) codexHelperV0() CodexLaunchSpecResolverV0 {
	config := resolver.CodexConfig
	config.ProjectWorkDir = claudeProjectWorkDirV0(resolver.Config, resolver.CodexConfig)
	config.RuntimeWorkDir = claudeRuntimeWorkDirV0(resolver.Config, resolver.CodexConfig)
	return CodexLaunchSpecResolverV0{
		Config:         config,
		TaskStore:      resolver.TaskStore,
		AppChangeStore: resolver.AppChangeStore,
	}
}

func (resolver ClaudeLaunchSpecResolverV0) workflowTaskForPayloadV0(
	ctx context.Context,
	payload orquestaruntime.LaunchRuntimeAgentRequestV0,
) (orquestacoreworkflow.WorkflowTaskV0, bool, error) {
	tasks, err := resolver.TaskStore.LoadWorkflowTasksV0(
		ctx,
		strings.TrimSpace(payload.RunID),
		[]string{strings.TrimSpace(payload.TaskRef)},
	)
	if err != nil || len(tasks) != 1 {
		return orquestacoreworkflow.WorkflowTaskV0{}, false, err
	}
	return tasks[0], true, nil
}

func (resolver ClaudeLaunchSpecResolverV0) externalWorkForTaskV0(
	ctx context.Context,
	runRef string,
	taskRef string,
) (*orquestaappchange.AppChangeExternalWorkV0, bool, error) {
	return resolver.codexHelperV0().externalWorkForTaskV0(ctx, runRef, taskRef)
}

func claudeProfileV0(
	config ClaudeRuntimeConfigV0,
	codexConfig CodexRuntimeConfigV0,
	runtimeDir string,
) orquestaruntimeclaude.ClaudeConnectorProfileV0 {
	return orquestaruntimeclaude.ClaudeConnectorProfileV0{
		SchemaVersion:  orquestaruntimeclaude.ClaudeConnectorProfileSchemaVersionV0,
		OptIn:          true,
		CommandPath:    strings.TrimSpace(config.CommandPath),
		ProjectWorkDir: claudeProjectWorkDirV0(config, codexConfig),
		RuntimeWorkDir: strings.TrimSpace(runtimeDir),
		HomeDir:        strings.TrimSpace(config.HomeDir),
		PathEnv:        strings.TrimSpace(config.PathEnv),
		Model:          strings.TrimSpace(config.Model),
		PermissionMode: strings.TrimSpace(config.PermissionMode),
		OutputFormat:   strings.TrimSpace(config.OutputFormat),
		Effort:         strings.TrimSpace(config.Effort),
		PromptLocale:   strings.TrimSpace(config.PromptLocale),
		ExtraArgs:      append([]string(nil), config.ExtraArgs...),
		PromptHints:    claudePromptHintsV0(config.PromptHints),
	}
}

func claudeProjectWorkDirV0(config ClaudeRuntimeConfigV0, codexConfig CodexRuntimeConfigV0) string {
	if strings.TrimSpace(config.ProjectWorkDir) != "" {
		return strings.TrimSpace(config.ProjectWorkDir)
	}
	return strings.TrimSpace(codexConfig.ProjectWorkDir)
}

func claudeRuntimeWorkDirV0(config ClaudeRuntimeConfigV0, codexConfig CodexRuntimeConfigV0) string {
	if strings.TrimSpace(config.RuntimeWorkDir) != "" {
		return strings.TrimSpace(config.RuntimeWorkDir)
	}
	return strings.TrimSpace(codexConfig.RuntimeWorkDir)
}

func claudePromptHintsV0(values []string) []string {
	return compactStringsV0(append([]string{
		"Trabajo de revision neutral lanzado por Orquesta; no llames REST ni APIs de dominio.",
		"Entrega el informe como fichero de producto dentro del write-set y declaralo en ACK.files.",
		"Revisa pedagogia, editorial, tests, tutor y discrepancias; conserva material recuperable y propone rework causal.",
	}, values...))
}

func claudeShouldHandleDomainWorkV0(workKind string) bool {
	switch strings.TrimSpace(workKind) {
	case "review_claude", "review_pair_codex_claude", "review_pair_gemini_claude",
		"generate_agent_candidate_claude", "vote_agent_candidates_claude":
		return true
	default:
		return false
	}
}

type providerAwareAckPathResolverV0 struct {
	Codex  CodexRuntimeConfigV0
	Gemini GeminiRuntimeConfigV0
	Claude ClaudeRuntimeConfigV0
}

func (resolver providerAwareAckPathResolverV0) ResolveCodexReceiptAckPathV0(
	ctx context.Context,
	request orquestaruntimecodexdelivery.CodexReceiptAckPathRequestV0,
) (orquestaruntimecodexdelivery.CodexReceiptAckPathResolutionV0, error) {
	baseDir := strings.TrimSpace(resolver.Codex.RuntimeWorkDir)
	projectDir := strings.TrimSpace(resolver.Codex.ProjectWorkDir)
	if providerAwareSpecUsesGeminiV0(request.Spec) {
		baseDir = geminiRuntimeWorkDirV0(resolver.Gemini, resolver.Codex)
		projectDir = geminiProjectWorkDirV0(resolver.Gemini, resolver.Codex)
	}
	if providerAwareSpecUsesClaudeV0(request.Spec) {
		baseDir = claudeRuntimeWorkDirV0(resolver.Claude, resolver.Codex)
		projectDir = claudeProjectWorkDirV0(resolver.Claude, resolver.Codex)
	}
	return orquestaruntimecodexdelivery.AgentScopedCodexReceiptAckPathResolverV0{
		BaseDir:        baseDir,
		ProjectWorkDir: projectDir,
	}.ResolveCodexReceiptAckPathV0(ctx, request)
}

func providerAwareSpecUsesGeminiV0(spec orquestaruntime.ExternalAgentLaunchSpecV0) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(spec.ConnectorRef)), "gemini")
}

func providerAwareSpecUsesClaudeV0(spec orquestaruntime.ExternalAgentLaunchSpecV0) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(spec.ConnectorRef)), "claude")
}
