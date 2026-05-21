package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

type CodexLaunchSpecResolverV0 struct {
	Config         CodexRuntimeConfigV0
	TaskStore      orquestacionnucleoapp.WorkflowTaskStorePortV0
	AppChangeStore orquestaappchange.AppChangeRecordSourcePortV0
}

var _ orquestacionnucleoapp.ExternalAgentLaunchSpecResolverPortV0 = CodexLaunchSpecResolverV0{}

func (resolver CodexLaunchSpecResolverV0) ResolveExternalAgentLaunchSpecV0(
	ctx context.Context,
	inbound orquestaruntime.AgentLauncherInboundV0,
) (orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0, error) {
	if inbound.Payload == nil {
		return orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0{}, fmt.Errorf("payload requerido")
	}
	agentRef := strings.TrimSpace(inbound.Payload.AgentRequestID)
	area := codexAreaV0(inbound.Payload.Role, inbound.Payload.TaskRef)
	runtimeDir, err := orquestaruntimecodexdelivery.CodexReceiptAgentRuntimeDirV0(
		resolver.Config.RuntimeWorkDir,
		inbound.Payload.RunID,
		agentRef,
	)
	if err != nil {
		return orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0{}, err
	}
	task, err := resolver.agentTaskV0(ctx, *inbound.Payload, area)
	if err != nil {
		return orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0{}, err
	}
	contextBundle, err := resolver.agentContextV0(ctx, *inbound.Payload, area, task)
	if err != nil {
		return orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0{}, err
	}
	task = taskWithContextGuardV0(task, contextBundle)
	spec := resolver.launchSpecV0(
		agentRef,
		strings.TrimSpace(inbound.CorrelationID),
		strings.TrimSpace(inbound.Payload.PhaseID),
		area,
		task,
	)
	spec.AgentPacket.Context = contextBundle
	profile := codexProfileForAreaV0(resolver.Config, runtimeDir, area)
	return orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0{
		Spec:            spec,
		CommandResolver: orquestaruntimecodex.NewCodexExecResolverV0(profile),
	}, nil
}

func codexProfileV0(
	config CodexRuntimeConfigV0,
	runtimeDir string,
) orquestaruntimecodex.CodexConnectorProfileV0 {
	return codexProfileForAreaV0(config, runtimeDir, "")
}

func codexProfileForAreaV0(
	config CodexRuntimeConfigV0,
	runtimeDir string,
	area string,
) orquestaruntimecodex.CodexConnectorProfileV0 {
	sandbox := strings.TrimSpace(config.Sandbox)
	approvalPolicy := strings.TrimSpace(config.ApprovalPolicy)
	if strings.TrimSpace(area) == "director" {
		if strings.TrimSpace(config.DirectorSandbox) != "" {
			sandbox = strings.TrimSpace(config.DirectorSandbox)
		}
		if strings.TrimSpace(config.DirectorApprovalPolicy) != "" {
			approvalPolicy = strings.TrimSpace(config.DirectorApprovalPolicy)
		}
	}
	return orquestaruntimecodex.CodexConnectorProfileV0{
		SchemaVersion:   orquestaruntimecodex.CodexConnectorProfileSchemaVersionV0,
		OptIn:           true,
		CommandPath:     strings.TrimSpace(config.CommandPath),
		ProjectWorkDir:  strings.TrimSpace(config.ProjectWorkDir),
		RuntimeWorkDir:  strings.TrimSpace(runtimeDir),
		CodeHomeDir:     strings.TrimSpace(config.CodeHomeDir),
		HomeDir:         strings.TrimSpace(config.HomeDir),
		PathEnv:         strings.TrimSpace(config.PathEnv),
		Model:           strings.TrimSpace(config.Model),
		ReasoningEffort: strings.TrimSpace(config.ReasoningEffort),
		Profile:         strings.TrimSpace(config.Profile),
		Sandbox:         sandbox,
		ApprovalPolicy:  approvalPolicy,
		ExtraArgs:       append([]string(nil), config.ExtraArgs...),
		PromptHints:     codexPromptHintsV0(config.PromptHints),
	}
}

func codexPromptHintsV0(values []string) []string {
	return compactStringsV0(append([]string{
		"Trabajo real lanzado por Orquesta; edita solo el write-set.",
		"Lee AGENTS.md, README.md y docs locales del directorio de trabajo.",
		"Ficheros pequenos; divide responsabilidades si se acerca a 300 lineas.",
	}, values...))
}

func (resolver CodexLaunchSpecResolverV0) launchSpecV0(
	agentRef string,
	correlationID string,
	phase string,
	area string,
	task orquestaruntime.AgentStartTaskV0,
) orquestaruntime.ExternalAgentLaunchSpecV0 {
	return orquestaruntime.ExternalAgentLaunchSpecV0{
		SchemaVersion: orquestaruntime.ExternalAgentLaunchSpecSchemaVersionV0,
		RequestID:     agentRef,
		CorrelationID: correlationID,
		ProfileRef:    "profile-ref-app-stack-" + area,
		ConnectorRef:  "connector-ref-app-stack-" + area,
		RuntimeKind:   "cli",
		LaunchMode:    orquestaruntime.RuntimeLaunchModeNewSessionV0,
		Command: orquestaruntime.ExternalAgentCommandV0{
			CommandRef:    "command-ref-app-stack-" + area,
			ExecutableRef: "exec-ref-app-stack-" + area,
			WorkingDirRef: "workdir-ref-app-stack-" + area,
		},
		AgentPacket: agentPacketV0(agentRef, correlationID, phase, area, task),
		Security:    codexSecurityV0(),
	}
}

func codexSecurityV0() orquestaruntime.ExternalAgentSecurityPolicyV0 {
	return orquestaruntime.ExternalAgentSecurityPolicyV0{
		OptIn:                 true,
		ShellPolicy:           orquestaruntime.ExternalAgentShellForbiddenV0,
		PathInheritancePolicy: orquestaruntime.ExternalAgentPathInheritanceForbiddenV0,
		EnvironmentPolicy:     orquestaruntime.ExternalAgentEnvExplicitRefsOnlyV0,
		SecretsPolicy:         orquestaruntime.ExternalAgentSecretsReferencesOnlyV0,
		HomePathsPolicy:       orquestaruntime.ExternalAgentHomeOpaqueRefsOnlyV0,
		NetworkPolicy:         orquestaruntime.ExternalAgentNetworkClosedV0,
		TranscriptsPolicy:     orquestaruntime.ExternalAgentTranscriptsForbiddenV0,
	}
}

func codexAreaV0(role string, taskRef string) string {
	value := strings.TrimSpace(role + "-" + taskRef)
	switch {
	case strings.Contains(value, "implementacion"):
		return "programacion"
	case strings.Contains(value, "dominio"):
		return "programacion"
	case strings.Contains(value, "director_web"):
		return "web"
	case strings.Contains(value, "director_api"):
		return "api"
	case strings.Contains(value, "director_persistencia"):
		return "persistencia"
	case strings.Contains(value, "director_i18n"):
		return "i18n"
	case strings.Contains(value, "director_calidad"):
		return "calidad"
	default:
		return "director"
	}
}

func isProgrammingPhaseV0(phase string) bool {
	return strings.TrimSpace(phase) == string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0)
}
