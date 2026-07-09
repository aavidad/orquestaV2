package orquestaappcodexstack

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaappchangedirectorsource "orquesta/modulos/orquesta-app-change-director-source"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestProviderLaunchSpecResolverV0RuteaVisualAssetAGemini(t *testing.T) {
	resolver, inbound, geminiRuntimeDir := geminiProviderResolverForTestV0(
		t,
		"generate_visual_asset",
	)
	resolution, err := resolver.ResolveExternalAgentLaunchSpecV0(context.Background(), inbound)
	if err != nil {
		t.Fatalf("ResolveExternalAgentLaunchSpecV0: %v", err)
	}
	if !strings.Contains(resolution.Spec.ConnectorRef, "gemini") ||
		resolution.Spec.AgentPacket.TargetModule != "orquesta-app-stack-visual" {
		t.Fatalf("spec no ruteada a Gemini: %+v", resolution.Spec)
	}
	req, issues := resolution.CommandResolver.ResolveExternalAgentProcessCommandV0(
		context.Background(),
		resolution.Spec,
	)
	if len(issues) != 0 {
		t.Fatalf("issues=%+v evidence=%v", issues, issues[0].Evidence)
	}
	if !strings.HasPrefix(req.CommandPath, geminiRuntimeDir) {
		t.Fatalf("command path=%s want under %s", req.CommandPath, geminiRuntimeDir)
	}
}

func TestProviderLaunchSpecResolverV0RuteaReviewGeminiAGemini(t *testing.T) {
	resolver, inbound, geminiRuntimeDir := geminiProviderResolverForTestV0(
		t,
		"review_gemini",
	)
	resolution, err := resolver.ResolveExternalAgentLaunchSpecV0(context.Background(), inbound)
	if err != nil {
		t.Fatalf("ResolveExternalAgentLaunchSpecV0: %v", err)
	}
	if !strings.Contains(resolution.Spec.ConnectorRef, "gemini") {
		t.Fatalf("spec no ruteada a Gemini: %+v", resolution.Spec)
	}
	req, issues := resolution.CommandResolver.ResolveExternalAgentProcessCommandV0(
		context.Background(),
		resolution.Spec,
	)
	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	if !strings.HasPrefix(req.CommandPath, geminiRuntimeDir) {
		t.Fatalf("command path=%s want under %s", req.CommandPath, geminiRuntimeDir)
	}
}

func TestProviderLaunchSpecResolverV0RuteaCandidatoGeminiAGemini(t *testing.T) {
	resolver, inbound, geminiRuntimeDir := geminiProviderResolverForTestV0(
		t,
		"generate_agent_candidate_gemini",
	)
	resolution, err := resolver.ResolveExternalAgentLaunchSpecV0(context.Background(), inbound)
	if err != nil {
		t.Fatalf("ResolveExternalAgentLaunchSpecV0: %v", err)
	}
	if !strings.Contains(resolution.Spec.ConnectorRef, "gemini") {
		t.Fatalf("spec no ruteada a Gemini: %+v", resolution.Spec)
	}
	req, issues := resolution.CommandResolver.ResolveExternalAgentProcessCommandV0(
		context.Background(),
		resolution.Spec,
	)
	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	if !strings.HasPrefix(req.CommandPath, geminiRuntimeDir) {
		t.Fatalf("command path=%s want under %s", req.CommandPath, geminiRuntimeDir)
	}
}

func TestProviderLaunchSpecResolverV0RuteaReviewClaudeAClaude(t *testing.T) {
	resolver, inbound, _, claudeRuntimeDir := providerResolverForTestV0(
		t,
		"review_claude",
	)
	resolution, err := resolver.ResolveExternalAgentLaunchSpecV0(context.Background(), inbound)
	if err != nil {
		t.Fatalf("ResolveExternalAgentLaunchSpecV0: %v", err)
	}
	if !strings.Contains(resolution.Spec.ConnectorRef, "claude") ||
		resolution.Spec.AgentPacket.TargetModule != "orquesta-app-stack-revision" {
		t.Fatalf("spec no ruteada a Claude: %+v", resolution.Spec)
	}
	req, issues := resolution.CommandResolver.ResolveExternalAgentProcessCommandV0(
		context.Background(),
		resolution.Spec,
	)
	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	if !strings.HasPrefix(req.CommandPath, claudeRuntimeDir) {
		t.Fatalf("command path=%s want under %s", req.CommandPath, claudeRuntimeDir)
	}
}

func TestProviderLaunchSpecResolverV0RuteaVotoClaudeAClaude(t *testing.T) {
	resolver, inbound, _, claudeRuntimeDir := providerResolverForTestV0(
		t,
		"vote_agent_candidates_claude",
	)
	resolution, err := resolver.ResolveExternalAgentLaunchSpecV0(context.Background(), inbound)
	if err != nil {
		t.Fatalf("ResolveExternalAgentLaunchSpecV0: %v", err)
	}
	if !strings.Contains(resolution.Spec.ConnectorRef, "claude") {
		t.Fatalf("spec no ruteada a Claude: %+v", resolution.Spec)
	}
	req, issues := resolution.CommandResolver.ResolveExternalAgentProcessCommandV0(
		context.Background(),
		resolution.Spec,
	)
	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	if !strings.HasPrefix(req.CommandPath, claudeRuntimeDir) {
		t.Fatalf("command path=%s want under %s", req.CommandPath, claudeRuntimeDir)
	}
}

func TestProviderLaunchSpecResolverV0PropagaPromptLocaleAProveedores(t *testing.T) {
	geminiResolver, geminiInbound, geminiRuntimeDir := geminiProviderResolverForTestV0(
		t,
		"generate_visual_asset",
	)
	geminiResolver.Gemini.Config.PromptLocale = "en-US"
	geminiResolution, err := geminiResolver.ResolveExternalAgentLaunchSpecV0(context.Background(), geminiInbound)
	if err != nil {
		t.Fatalf("ResolveExternalAgentLaunchSpecV0 gemini: %v", err)
	}
	geminiReq, issues := geminiResolution.CommandResolver.ResolveExternalAgentProcessCommandV0(
		context.Background(),
		geminiResolution.Spec,
	)
	if len(issues) != 0 {
		t.Fatalf("gemini issues=%+v", issues)
	}
	if !strings.HasPrefix(geminiReq.CommandPath, geminiRuntimeDir) {
		t.Fatalf("gemini command path=%s want under %s", geminiReq.CommandPath, geminiRuntimeDir)
	}
	geminiPrompt := mustReadProviderResolverFileForTestV0(t, filepath.Join(filepath.Dir(geminiReq.CommandPath), "agent_prompt.txt"))
	if !strings.Contains(geminiPrompt, "You are an external agent governed by OrquestaV2.") ||
		strings.Contains(geminiPrompt, "Eres un agente externo gobernado") {
		t.Fatalf("prompt gemini no localizado:\n%s", geminiPrompt)
	}

	claudeResolver, claudeInbound, _, claudeRuntimeDir := providerResolverForTestV0(
		t,
		"review_claude",
	)
	claudeResolver.Claude.Config.PromptLocale = "en-US"
	claudeResolution, err := claudeResolver.ResolveExternalAgentLaunchSpecV0(context.Background(), claudeInbound)
	if err != nil {
		t.Fatalf("ResolveExternalAgentLaunchSpecV0 claude: %v", err)
	}
	claudeReq, issues := claudeResolution.CommandResolver.ResolveExternalAgentProcessCommandV0(
		context.Background(),
		claudeResolution.Spec,
	)
	if len(issues) != 0 {
		t.Fatalf("claude issues=%+v", issues)
	}
	if !strings.HasPrefix(claudeReq.CommandPath, claudeRuntimeDir) {
		t.Fatalf("claude command path=%s want under %s", claudeReq.CommandPath, claudeRuntimeDir)
	}
	claudePrompt := mustReadProviderResolverFileForTestV0(t, filepath.Join(filepath.Dir(claudeReq.CommandPath), "agent_prompt.txt"))
	if !strings.Contains(claudePrompt, "You are an external agent governed by OrquestaV2.") ||
		strings.Contains(claudePrompt, "Eres un agente externo gobernado") {
		t.Fatalf("prompt claude no localizado:\n%s", claudePrompt)
	}
}

func TestProviderLaunchSpecResolverV0MantieneCodexParaTrabajoNoVisual(t *testing.T) {
	resolver, inbound, _ := geminiProviderResolverForTestV0(
		t,
		"draft_content_block",
	)
	resolution, err := resolver.ResolveExternalAgentLaunchSpecV0(context.Background(), inbound)
	if err != nil {
		t.Fatalf("ResolveExternalAgentLaunchSpecV0: %v", err)
	}
	if strings.Contains(resolution.Spec.ConnectorRef, "gemini") ||
		resolution.Spec.AgentPacket.TargetModule != "orquesta-app-stack-programacion" {
		t.Fatalf("spec no debe usar Gemini: %+v", resolution.Spec)
	}
}

func TestProviderAwareAckPathResolverV0UsaRuntimeGeminiParaSpecGemini(t *testing.T) {
	root := t.TempDir()
	codexRuntime := filepath.Join(root, "codex-runtime")
	geminiRuntime := filepath.Join(root, "gemini-runtime")
	path, err := (providerAwareAckPathResolverV0{
		Codex: CodexRuntimeConfigV0{
			ProjectWorkDir: filepath.Join(root, "project"),
			RuntimeWorkDir: codexRuntime,
		},
		Gemini: GeminiRuntimeConfigV0{
			Enabled:        true,
			ProjectWorkDir: filepath.Join(root, "project"),
			RuntimeWorkDir: geminiRuntime,
		},
	}).ResolveCodexReceiptAckPathV0(context.Background(), orquestaruntimecodexdeliveryRequestForTestV0())
	if err != nil {
		t.Fatalf("ResolveCodexReceiptAckPathV0: %v", err)
	}
	if !strings.HasPrefix(path.AckPath, geminiRuntime) {
		t.Fatalf("ack_path=%s want under %s", path.AckPath, geminiRuntime)
	}
}

func TestProviderAwareAckPathResolverV0UsaRuntimeClaudeParaSpecClaude(t *testing.T) {
	root := t.TempDir()
	codexRuntime := filepath.Join(root, "codex-runtime")
	claudeRuntime := filepath.Join(root, "claude-runtime")
	path, err := (providerAwareAckPathResolverV0{
		Codex: CodexRuntimeConfigV0{
			ProjectWorkDir: filepath.Join(root, "project"),
			RuntimeWorkDir: codexRuntime,
		},
		Claude: ClaudeRuntimeConfigV0{
			Enabled:        true,
			ProjectWorkDir: filepath.Join(root, "project"),
			RuntimeWorkDir: claudeRuntime,
		},
	}).ResolveCodexReceiptAckPathV0(context.Background(), orquestaruntimecodexdeliveryRequestWithConnectorForTestV0("connector-ref-app-stack-claude-review"))
	if err != nil {
		t.Fatalf("ResolveCodexReceiptAckPathV0: %v", err)
	}
	if !strings.HasPrefix(path.AckPath, claudeRuntime) {
		t.Fatalf("ack_path=%s want under %s", path.AckPath, claudeRuntime)
	}
}

func geminiProviderResolverForTestV0(
	t *testing.T,
	workKind string,
) (providerLaunchSpecResolverV0, orquestaruntime.AgentLauncherInboundV0, string) {
	resolver, inbound, geminiRuntimeDir, _ := providerResolverForTestV0(t, workKind)
	return resolver, inbound, geminiRuntimeDir
}

func providerResolverForTestV0(
	t *testing.T,
	workKind string,
) (providerLaunchSpecResolverV0, orquestaruntime.AgentLauncherInboundV0, string, string) {
	t.Helper()
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	codexRuntimeDir := filepath.Join(root, "codex-runtime")
	geminiRuntimeDir := filepath.Join(root, "gemini-runtime")
	claudeRuntimeDir := filepath.Join(root, "claude-runtime")
	changeRef := "change-ref-provider-visual"
	runRef := "run-ref-provider-visual"
	taskRef := orquestaappchangedirectorsource.AppChangeTaskRefV0(changeRef)
	task := orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion: orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:        taskRef,
		RunID:         runRef,
		PhaseID:       orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Title:         "Crear derivado de dominio",
		Summary:       "Crear trabajo externo.",
		WriteSet:      []string{"external/visual/asset.json"},
		AcceptanceCriteria: []string{
			"entrega de dominio valida",
		},
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			ContractRef:  "contract:function:app-change:visual:v0",
			FunctionName: "ApplyExternalDomainWorkV0",
		}},
	}
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(task)
	appChangeStore := orquestaappchange.NewInMemoryAppChangeStoreV0(
		orquestaappchange.AppChangeRecordV0{
			Request: orquestaappchange.AppChangeRequestV0{
				RunRef:     runRef,
				ChangeRef:  changeRef,
				UserIntent: "Crear derivado visual.",
				ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
					ProjectRef: "external-app",
					JobRef:     "job-ref-provider-visual",
					WorkKind:   workKind,
					InputFields: []orquestadomainwork.DomainWorkFieldV0{
						{Name: "expected_artifact_type", Value: "visual_asset"},
					},
				},
			},
		},
	)
	codexConfig := CodexRuntimeConfigV0{
		CommandPath:     filepath.Join(root, "codex"),
		ProjectWorkDir:  projectDir,
		RuntimeWorkDir:  codexRuntimeDir,
		Sandbox:         "workspace-write",
		ApprovalPolicy:  "never",
		ReasoningEffort: string(orquestacoreworkflow.OrchestrationCapacityMediumV0),
	}
	resolver := providerLaunchSpecResolverV0{
		Codex: CodexLaunchSpecResolverV0{
			Config:         codexConfig,
			TaskStore:      taskStore,
			AppChangeStore: appChangeStore,
		},
		Gemini: GeminiLaunchSpecResolverV0{
			Config: GeminiRuntimeConfigV0{
				Enabled:        true,
				CommandPath:    filepath.Join(root, "gemini"),
				ProjectWorkDir: projectDir,
				RuntimeWorkDir: geminiRuntimeDir,
				ApprovalMode:   "auto_edit",
			},
			CodexConfig:    codexConfig,
			TaskStore:      taskStore,
			AppChangeStore: appChangeStore,
		},
		Claude: ClaudeLaunchSpecResolverV0{
			Config: ClaudeRuntimeConfigV0{
				Enabled:        true,
				CommandPath:    filepath.Join(root, "claude"),
				ProjectWorkDir: projectDir,
				RuntimeWorkDir: claudeRuntimeDir,
				PermissionMode: "dontAsk",
				OutputFormat:   "text",
			},
			CodexConfig:    codexConfig,
			TaskStore:      taskStore,
			AppChangeStore: appChangeStore,
		},
	}
	return resolver, orquestaruntime.AgentLauncherInboundV0{
		CorrelationID: "correlation-ref-provider-visual",
		Payload: &orquestaruntime.LaunchRuntimeAgentRequestV0{
			RunID:              runRef,
			AgentRequestID:     "agent-ref-provider-visual",
			PhaseID:            string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			Role:               "dominio",
			TaskRef:            taskRef,
			CapacityRequestRef: "capacity-ref-provider-visual",
			Summary:            "external_work",
		},
	}, geminiRuntimeDir, claudeRuntimeDir
}

func orquestaruntimecodexdeliveryRequestForTestV0() orquestaruntimecodexdelivery.CodexReceiptAckPathRequestV0 {
	return orquestaruntimecodexdeliveryRequestWithConnectorForTestV0("connector-ref-app-stack-gemini-visual")
}

func orquestaruntimecodexdeliveryRequestWithConnectorForTestV0(connectorRef string) orquestaruntimecodexdelivery.CodexReceiptAckPathRequestV0 {
	return orquestaruntimecodexdelivery.CodexReceiptAckPathRequestV0{
		RunID:    "run-ref-provider-visual",
		AgentRef: "agent-ref-provider-visual",
		Spec: orquestaruntime.ExternalAgentLaunchSpecV0{
			ConnectorRef: connectorRef,
		},
	}
}

func mustReadProviderResolverFileForTestV0(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	return string(data)
}
