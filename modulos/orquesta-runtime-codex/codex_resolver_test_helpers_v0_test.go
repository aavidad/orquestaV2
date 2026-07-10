package orquestaruntimecodex

import (
	"os"
	"path/filepath"
	"testing"

	orquestacontext "orquesta/modulos/orquesta-context"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func codexProfileForTestV0(t *testing.T) CodexConnectorProfileV0 {
	t.Helper()
	root := t.TempDir()
	return CodexConnectorProfileV0{
		SchemaVersion:   CodexConnectorProfileSchemaVersionV0,
		OptIn:           true,
		CommandPath:     filepath.Join(root, "bin", "codex"),
		ProjectWorkDir:  filepath.Join(root, "project"),
		RuntimeWorkDir:  filepath.Join(root, "runtime", "agent"),
		Model:           "model-concrete-test",
		ReasoningEffort: "medium",
		ModelRouting: CodexModelRoutingReceiptV0{
			Level: "normal", SelectedModelRef: "model-ref-test", PolicyRef: "policy-ref-test",
		},
		Sandbox:        "workspace-write",
		ApprovalPolicy: "never",
	}
}

func codexSpecForTestV0() orquestaruntime.ExternalAgentLaunchSpecV0 {
	packet := codexPacketForTestV0()
	return orquestaruntime.ExternalAgentLaunchSpecV0{
		SchemaVersion: orquestaruntime.ExternalAgentLaunchSpecSchemaVersionV0,
		RequestID:     packet.RequestID,
		CorrelationID: packet.CorrelationID,
		ProfileRef:    "codex-profile-ref-001",
		ConnectorRef:  "codex-connector-ref-001",
		RuntimeKind:   "cli",
		LaunchMode:    orquestaruntime.RuntimeLaunchModeNewSessionV0,
		Command: orquestaruntime.ExternalAgentCommandV0{
			CommandRef:    "codex-command-ref-001",
			ExecutableRef: "codex-executable-ref-001",
			WorkingDirRef: "codex-workdir-ref-001",
		},
		AgentPacket: packet,
		Security:    codexSecurityForTestV0(),
	}
}

func codexSecurityForTestV0() orquestaruntime.ExternalAgentSecurityPolicyV0 {
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

func codexPacketForTestV0() orquestaruntime.AgentStartPacketV0 {
	return orquestaruntime.AgentStartPacketV0{
		SchemaVersion: orquestaruntime.AgentStartPacketSchemaVersionV0,
		RequestID:     "req-codex-001",
		CorrelationID: "corr-codex-001",
		WorkOrderRef:  "task-ref-001",
		TargetModule:  "orquesta-generated-app",
		Phase:         "programacion",
		CapacityLevel: "medium",
		Locale:        "es-ES",
		Task: orquestaruntime.AgentStartTaskV0{
			TaskRef:       "task-ref-001",
			Priority:      "normal",
			Title:         "Crear app",
			Objective:     "Crear una app pequena.",
			TargetSymbol:  "App",
			WriteSet:      []string{"README.md"},
			RequiredTests: []string{"go test ./..."},
			DoneCriteria:  []string{"ACK escrito."},
		},
		Context: codexContextForTestV0(),
		DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{
			MailboxRef:   "mailbox-ref-001",
			AckRef:       "ack-ref-001",
			ReadinessRef: "readiness-ref-001",
		},
		Policies: []string{"write_set_closed"},
	}
}

func codexContextForTestV0() orquestacontext.ContextMaterializedBundleV0 {
	return orquestacontext.ContextMaterializedBundleV0{
		SchemaVersion: orquestacontext.ContextMaterializedBundleSchemaVersionV0,
		BundleRef:     "bundle-ref-001",
		WorkOrderRef:  "task-ref-001",
		TargetModule:  "orquesta-generated-app",
		Entries: []orquestacontext.ContextMaterializedEntryV0{{
			EntryRef:  "entry-ref-001",
			Layer:     orquestacontext.ContextLayerTaskContextV0,
			Kind:      orquestacontext.ContextEntryDocRefV0,
			SourceRef: "README.md",
			Mode:      orquestacontext.ContextMaterializationModeContentV0,
			Content:   "contexto local",
			Bytes:     13,
			Required:  true,
		}},
	}
}

func requireCodexIssueV0(
	t *testing.T,
	issues []orquestaruntime.ExternalAgentConnectorErrorV0,
	code CodexConnectorIssueCodeV0,
) {
	t.Helper()
	for _, issue := range issues {
		if issue.Code == orquestaruntime.ExternalAgentConnectorErrorCodeV0(code) {
			return
		}
	}
	t.Fatalf("no se encontro issue %q en %+v", code, issues)
}

func requireFileExistsV0(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("archivo %s no existe: %v", path, err)
	}
}
