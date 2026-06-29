package orquestaruntimeclaude

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestacontext "orquesta/modulos/orquesta-context"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestClaudeExecResolverV0MaterializaControlFiles(t *testing.T) {
	root := t.TempDir()
	runtimeDir := filepath.Join(root, "runtime")
	resolver := NewClaudeExecResolverV0(ClaudeConnectorProfileV0{
		SchemaVersion:  ClaudeConnectorProfileSchemaVersionV0,
		OptIn:          true,
		CommandPath:    filepath.Join(root, "claude"),
		ProjectWorkDir: filepath.Join(root, "project"),
		RuntimeWorkDir: runtimeDir,
		PermissionMode: "dontAsk",
		OutputFormat:   "text",
	})
	req, issues := resolver.ResolveExternalAgentProcessCommandV0(
		context.Background(),
		claudeSpecForTestV0(),
	)
	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	if req.CommandPath != filepath.Join(runtimeDir, ClaudeWrapperFileNameV0) {
		t.Fatalf("command path=%s", req.CommandPath)
	}
	for _, name := range []string{
		ClaudeAgentPacketFileNameV0,
		ClaudeAgentPromptFileNameV0,
		ClaudeWrapperFileNameV0,
	} {
		if _, err := os.Stat(filepath.Join(runtimeDir, name)); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

func TestClaudeExecResolverV0PromptUsaControlFilesOperativos(t *testing.T) {
	root := t.TempDir()
	runtimeDir := filepath.Join(root, "runtime")
	spec := claudeSpecForTestV0()
	spec.AgentPacket.Task.RequiredTests = []string{"go test ./..."}
	spec.AgentPacket.Context.Entries[0].Mode = orquestacontext.ContextMaterializationModeRefOnlyV0
	spec.AgentPacket.Context.Entries[0].Content = ""
	spec.AgentPacket.Context.Entries[0].Bytes = 0
	spec.AgentPacket.Context.Entries[0].RefOnlyReason = orquestacontext.ContextRefOnlyReasonMaterializationMissingV0
	spec.AgentPacket.Context.Entries[0].RequiredRefAction = orquestacontext.ContextRequiredRefActionReadLocalV0
	resolver := NewClaudeExecResolverV0(ClaudeConnectorProfileV0{
		SchemaVersion:  ClaudeConnectorProfileSchemaVersionV0,
		OptIn:          true,
		CommandPath:    filepath.Join(root, "claude"),
		ProjectWorkDir: filepath.Join(root, "project"),
		RuntimeWorkDir: runtimeDir,
		PermissionMode: "dontAsk",
		OutputFormat:   "text",
	})

	_, issues := resolver.ResolveExternalAgentProcessCommandV0(
		context.Background(),
		spec,
	)
	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	prompt := mustReadClaudeFileForTestV0(t, filepath.Join(runtimeDir, ClaudeAgentPromptFileNameV0))
	for _, want := range []string{
		filepath.Join(runtimeDir, ClaudeAgentPacketFileNameV0),
		filepath.Join(runtimeDir, ClaudeAgentAckFileNameV0),
		filepath.Join(runtimeDir, ClaudeDirectorDecisionsFileNameV0),
		filepath.Join(runtimeDir, ClaudeShutdownRequestFileNameV0),
		filepath.Join(runtimeDir, ClaudeShutdownCheckpointAckFileNameV0),
		"decision_path:",
		"CHECKPOINT DE APAGADO",
		"codex_shutdown_checkpoint_ack.v0",
		"checkpoint_ready",
		"CONTEXTO REF_ONLY REQUERIDO",
		"required_ref_action",
		"contexto_ref_only_resuelto",
		"orquesta_agent_ack.v0",
		"codex_agent_ack.v0",
		"test_receipts",
		"orquesta_required_test_receipt.v0",
		"No uses git status como criterio obligatorio",
		"No incluyas archivos de control en ACK.files",
		"files debe listar rutas reales",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt no contiene %q:\n%s", want, prompt)
		}
	}
}

func claudeSpecForTestV0() orquestaruntime.ExternalAgentLaunchSpecV0 {
	return orquestaruntime.ExternalAgentLaunchSpecV0{
		SchemaVersion: orquestaruntime.ExternalAgentLaunchSpecSchemaVersionV0,
		RequestID:     "agent-ref-claude-test",
		CorrelationID: "correlation-ref-claude-test",
		ProfileRef:    "profile-ref-claude-test",
		ConnectorRef:  "connector-ref-claude-test",
		RuntimeKind:   "cli",
		LaunchMode:    orquestaruntime.RuntimeLaunchModeNewSessionV0,
		Command: orquestaruntime.ExternalAgentCommandV0{
			CommandRef:    "command-ref-claude-test",
			ExecutableRef: "exec-ref-claude-test",
			WorkingDirRef: "workdir-ref-claude-test",
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
			RequestID:     "agent-ref-claude-test",
			CorrelationID: "correlation-ref-claude-test",
			WorkOrderRef:  "task-ref-claude-test",
			TargetModule:  "orquesta-app-stack-revision",
			Phase:         "programacion",
			Task: orquestaruntime.AgentStartTaskV0{
				TaskRef:   "task-ref-claude-test",
				Title:     "Revisar paquete",
				Objective: "Crear informe de revision",
				WriteSet:  []string{"external/opes/review_claude/report.json"},
			},
			Context: orquestacontext.ContextMaterializedBundleV0{
				SchemaVersion: orquestacontext.ContextMaterializedBundleSchemaVersionV0,
				BundleRef:     "bundle-ref-claude-test",
				WorkOrderRef:  "task-ref-claude-test",
				TargetModule:  "orquesta-app-stack-revision",
				Entries: []orquestacontext.ContextMaterializedEntryV0{{
					EntryRef:  "entry-ref-claude-test",
					Layer:     orquestacontext.ContextLayerTaskContextV0,
					Kind:      orquestacontext.ContextEntryDocRefV0,
					SourceRef: "source-ref-claude-test",
					Mode:      orquestacontext.ContextMaterializationModeContentV0,
					Content:   "contexto revision",
					Bytes:     len("contexto revision"),
					Required:  true,
				}},
			},
			DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{
				MailboxRef:   "mailbox-ref-claude-test",
				AckRef:       "ack-ref-claude-test",
				ReadinessRef: "readiness-ref-claude-test",
			},
			Policies: []string{"write_set_closed", "ack_required"},
		},
	}
}

func mustReadClaudeFileForTestV0(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	return string(data)
}
