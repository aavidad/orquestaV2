package orquestaappcodexstack

import (
	"context"
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
		t.Fatalf("issues=%+v", issues)
	}
	if !strings.HasPrefix(req.CommandPath, geminiRuntimeDir) {
		t.Fatalf("command path=%s want under %s", req.CommandPath, geminiRuntimeDir)
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

func geminiProviderResolverForTestV0(
	t *testing.T,
	workKind string,
) (providerLaunchSpecResolverV0, orquestaruntime.AgentLauncherInboundV0, string) {
	t.Helper()
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	codexRuntimeDir := filepath.Join(root, "codex-runtime")
	geminiRuntimeDir := filepath.Join(root, "gemini-runtime")
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
	}, geminiRuntimeDir
}

func orquestaruntimecodexdeliveryRequestForTestV0() orquestaruntimecodexdelivery.CodexReceiptAckPathRequestV0 {
	return orquestaruntimecodexdelivery.CodexReceiptAckPathRequestV0{
		RunID:    "run-ref-provider-visual",
		AgentRef: "agent-ref-provider-visual",
		Spec: orquestaruntime.ExternalAgentLaunchSpecV0{
			ConnectorRef: "connector-ref-app-stack-gemini-visual",
		},
	}
}
