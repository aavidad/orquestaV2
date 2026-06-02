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
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
	orquestaruntimegemini "orquesta/modulos/orquesta-runtime-gemini"
)

const geminiVisualAreaV0 = "visual"

type providerLaunchSpecResolverV0 struct {
	Codex  CodexLaunchSpecResolverV0
	Gemini GeminiLaunchSpecResolverV0
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
	return domainWorkArtifactTypeForWorkKindV0(work.WorkKind) ==
		orquestadomainwork.DomainWorkArtifactTypeVisualAssetV0, nil
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
		"Para infografias educativas prioriza SVG autocontenido, Mermaid o HTML simple con alt_text.",
	}, values...))
}

type providerAwareAckPathResolverV0 struct {
	Codex  CodexRuntimeConfigV0
	Gemini GeminiRuntimeConfigV0
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
	return orquestaruntimecodexdelivery.AgentScopedCodexReceiptAckPathResolverV0{
		BaseDir:        baseDir,
		ProjectWorkDir: projectDir,
	}.ResolveCodexReceiptAckPathV0(ctx, request)
}

func providerAwareSpecUsesGeminiV0(spec orquestaruntime.ExternalAgentLaunchSpecV0) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(spec.ConnectorRef)), "gemini")
}
