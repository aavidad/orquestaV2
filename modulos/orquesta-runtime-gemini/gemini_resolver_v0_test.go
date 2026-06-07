package orquestaruntimegemini

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestacontext "orquesta/modulos/orquesta-context"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestGeminiExecResolverV0MaterializaPromptWrapperYAckPath(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	runtimeDir := filepath.Join(root, "runtime")
	profile := GeminiConnectorProfileV0{
		SchemaVersion:  GeminiConnectorProfileSchemaVersionV0,
		OptIn:          true,
		CommandPath:    filepath.Join(root, "gemini"),
		ProjectWorkDir: projectDir,
		RuntimeWorkDir: runtimeDir,
		Model:          "gemini-2.5-pro",
		ApprovalMode:   "auto_edit",
	}
	resolver := NewGeminiExecResolverV0(profile)
	req, issues := resolver.ResolveExternalAgentProcessCommandV0(
		context.Background(),
		geminiSpecForTestV0(),
	)
	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	if req.CommandPath != filepath.Join(runtimeDir, GeminiWrapperFileNameV0) ||
		req.WorkingDir != projectDir {
		t.Fatalf("req=%+v", req)
	}
	prompt := mustReadGeminiFileForTestV0(t, filepath.Join(runtimeDir, GeminiAgentPromptFileNameV0))
	if !strings.Contains(prompt, "visual_asset") ||
		!strings.Contains(prompt, filepath.Join(runtimeDir, GeminiAgentAckFileNameV0)) {
		t.Fatalf("prompt inesperado: %s", prompt)
	}
	wrapper := mustReadGeminiFileForTestV0(t, req.CommandPath)
	if !strings.Contains(wrapper, "--prompt ''") ||
		!strings.Contains(wrapper, "< '"+filepath.Join(runtimeDir, GeminiAgentPromptFileNameV0)+"'") ||
		strings.Contains(wrapper, "Crear infografia") {
		t.Fatalf("wrapper inseguro o incompleto: %s", wrapper)
	}
}

func TestGeminiExecResolverV0PromptUsaControlFilesOperativos(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	runtimeDir := filepath.Join(root, "runtime")
	spec := geminiSpecForTestV0()
	spec.AgentPacket.Task.RequiredTests = []string{"go test ./..."}
	spec.AgentPacket.Context.Entries[0].Mode = orquestacontext.ContextMaterializationModeRefOnlyV0
	spec.AgentPacket.Context.Entries[0].Content = ""
	spec.AgentPacket.Context.Entries[0].Bytes = 0
	spec.AgentPacket.Context.Entries[0].RefOnlyReason = orquestacontext.ContextRefOnlyReasonMaterializationMissingV0
	spec.AgentPacket.Context.Entries[0].RequiredRefAction = orquestacontext.ContextRequiredRefActionReadLocalV0
	profile := GeminiConnectorProfileV0{
		SchemaVersion:  GeminiConnectorProfileSchemaVersionV0,
		OptIn:          true,
		CommandPath:    filepath.Join(root, "gemini"),
		ProjectWorkDir: projectDir,
		RuntimeWorkDir: runtimeDir,
		Model:          "gemini-2.5-pro",
		ApprovalMode:   "auto_edit",
	}

	_, issues := NewGeminiExecResolverV0(profile).ResolveExternalAgentProcessCommandV0(
		context.Background(),
		spec,
	)
	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	prompt := mustReadGeminiFileForTestV0(t, filepath.Join(runtimeDir, GeminiAgentPromptFileNameV0))
	for _, want := range []string{
		filepath.Join(runtimeDir, GeminiAgentPacketFileNameV0),
		filepath.Join(runtimeDir, GeminiAgentAckFileNameV0),
		filepath.Join(runtimeDir, GeminiDirectorDecisionsFileNameV0),
		filepath.Join(runtimeDir, GeminiShutdownRequestFileNameV0),
		filepath.Join(runtimeDir, GeminiShutdownCheckpointAckFileNameV0),
		"decision_path:",
		"CHECKPOINT DE APAGADO",
		"codex_shutdown_checkpoint_ack.v0",
		"checkpoint_ready",
		"CONTEXTO REF_ONLY REQUERIDO",
		"required_ref_action",
		"contexto_ref_only_resuelto",
		"test_receipts",
		"codex_required_test_receipt.v0",
		"No uses git status como criterio obligatorio",
		"No incluyas archivos de control en ACK.files",
		"files debe listar rutas reales",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt no contiene %q:\n%s", want, prompt)
		}
	}
}

func geminiSpecForTestV0() orquestaruntime.ExternalAgentLaunchSpecV0 {
	return orquestaruntime.ExternalAgentLaunchSpecV0{
		SchemaVersion: orquestaruntime.ExternalAgentLaunchSpecSchemaVersionV0,
		RequestID:     "agent-ref-gemini-test",
		CorrelationID: "correlation-ref-gemini-test",
		ProfileRef:    "profile-ref-gemini-test",
		ConnectorRef:  "connector-ref-gemini-test",
		RuntimeKind:   "cli",
		LaunchMode:    orquestaruntime.RuntimeLaunchModeNewSessionV0,
		Command: orquestaruntime.ExternalAgentCommandV0{
			CommandRef:    "command-ref-gemini-test",
			ExecutableRef: "exec-ref-gemini-test",
			WorkingDirRef: "workdir-ref-gemini-test",
		},
		Security: orquestaruntime.ExternalAgentSecurityPolicyV0{
			OptIn:                 true,
			ShellPolicy:           orquestaruntime.ExternalAgentShellForbiddenV0,
			PathInheritancePolicy: orquestaruntime.ExternalAgentPathInheritanceForbiddenV0,
			EnvironmentPolicy:     orquestaruntime.ExternalAgentEnvExplicitRefsOnlyV0,
			SecretsPolicy:         orquestaruntime.ExternalAgentSecretsReferencesOnlyV0,
			HomePathsPolicy:       orquestaruntime.ExternalAgentHomeOpaqueRefsOnlyV0,
			NetworkPolicy:         orquestaruntime.ExternalAgentNetworkClosedV0,
			TranscriptsPolicy:     orquestaruntime.ExternalAgentTranscriptsForbiddenV0,
		},
		AgentPacket: orquestaruntime.AgentStartPacketV0{
			SchemaVersion: orquestaruntime.AgentStartPacketSchemaVersionV0,
			RequestID:     "agent-ref-gemini-test",
			CorrelationID: "correlation-ref-gemini-test",
			WorkOrderRef:  "task-ref-gemini-test",
			TargetModule:  "orquesta-app-stack-visual",
			Phase:         "programacion",
			Task: orquestaruntime.AgentStartTaskV0{
				TaskRef:   "task-ref-gemini-test",
				Title:     "Crear infografia",
				Objective: "Crear visual_asset educativo.",
				WriteSet:  []string{"external/visual/asset.json"},
			},
			Context: orquestacontext.ContextMaterializedBundleV0{
				SchemaVersion: orquestacontext.ContextMaterializedBundleSchemaVersionV0,
				BundleRef:     "bundle-ref-gemini-test",
				WorkOrderRef:  "task-ref-gemini-test",
				TargetModule:  "orquesta-app-stack-visual",
				Entries: []orquestacontext.ContextMaterializedEntryV0{{
					EntryRef:  "entry-ref-gemini-test",
					Layer:     orquestacontext.ContextLayerTaskContextV0,
					Kind:      orquestacontext.ContextEntryDocRefV0,
					SourceRef: "source-ref-gemini-test",
					Mode:      orquestacontext.ContextMaterializationModeContentV0,
					Content:   "contexto visual",
					Bytes:     len("contexto visual"),
					Required:  true,
				}},
			},
			DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{
				MailboxRef:   "mailbox-ref-gemini-test",
				AckRef:       "ack-ref-gemini-test",
				ReadinessRef: "readiness-ref-gemini-test",
			},
			Policies: []string{"write_set_closed", "ack_required"},
		},
	}
}

func mustReadGeminiFileForTestV0(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	return string(data)
}
