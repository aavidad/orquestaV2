package orquestaruntimecodex

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestacontext "orquesta/modulos/orquesta-context"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestCodexExecResolverV0BloqueaSinOptIn(t *testing.T) {
	profile := codexProfileForTestV0(t)
	profile.OptIn = false

	_, issues := NewCodexExecResolverV0(profile).
		ResolveExternalAgentProcessCommandV0(context.Background(), codexSpecForTestV0())

	requireCodexIssueV0(t, issues, CodexConnectorOptInRequiredV0)
}

func TestCodexExecResolverV0BloqueaCamposRequeridos(t *testing.T) {
	profile := codexProfileForTestV0(t)
	profile.CommandPath = ""
	profile.ProjectWorkDir = ""
	profile.RuntimeWorkDir = ""

	_, issues := NewCodexExecResolverV0(profile).
		ResolveExternalAgentProcessCommandV0(context.Background(), codexSpecForTestV0())

	requireCodexIssueV0(t, issues, CodexConnectorPathInvalidV0)
}

func TestCodexExecResolverV0MaterializaPacketPromptYWrapper(t *testing.T) {
	profile := codexProfileForTestV0(t)
	req, issues := NewCodexExecResolverV0(profile).
		ResolveExternalAgentProcessCommandV0(context.Background(), codexSpecForTestV0())
	if len(issues) != 0 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}

	requireFileExistsV0(t, filepath.Join(profile.RuntimeWorkDir, CodexAgentPacketFileNameV0))
	requireFileExistsV0(t, filepath.Join(profile.RuntimeWorkDir, CodexAgentPromptFileNameV0))
	requireFileExistsV0(t, filepath.Join(profile.RuntimeWorkDir, CodexWrapperFileNameV0))
	if req.CommandPath != filepath.Join(profile.RuntimeWorkDir, CodexWrapperFileNameV0) {
		t.Fatalf("command path=%q", req.CommandPath)
	}
	if req.WorkingDir != profile.ProjectWorkDir {
		t.Fatalf("working dir=%q", req.WorkingDir)
	}
	if len(req.Args) != 0 || len(req.Env) != 0 {
		t.Fatalf("request publico filtra args/env: %+v", req)
	}
	if _, err := os.Stat(filepath.Join(profile.ProjectWorkDir, CodexAgentPacketFileNameV0)); err == nil {
		t.Fatalf("packet no debe materializarse en project workdir compartido")
	}
}

func TestCodexExecResolverV0NoMaterializaSiProcessRequestEsInvalido(t *testing.T) {
	profile := codexProfileForTestV0(t)
	profile.RuntimeWorkDir = filepath.Join(t.TempDir(), "token-work")

	_, issues := NewCodexExecResolverV0(profile).
		ResolveExternalAgentProcessCommandV0(context.Background(), codexSpecForTestV0())
	requireCodexIssueV0(t, issues, CodexConnectorPathInvalidV0)

	for _, name := range []string{
		CodexAgentPacketFileNameV0,
		CodexAgentPromptFileNameV0,
		CodexWrapperFileNameV0,
	} {
		if _, err := os.Stat(filepath.Join(profile.RuntimeWorkDir, name)); err == nil {
			t.Fatalf("no debe materializar %s si el request de proceso es invalido", name)
		}
	}
}

func TestCodexExecResolverV0WrapperFijaCWDDelProyectoYControlEnRuntime(t *testing.T) {
	profile := codexProfileForTestV0(t)
	_, issues := NewCodexExecResolverV0(profile).
		ResolveExternalAgentProcessCommandV0(context.Background(), codexSpecForTestV0())
	if len(issues) != 0 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}

	wrapper, err := os.ReadFile(filepath.Join(profile.RuntimeWorkDir, CodexWrapperFileNameV0))
	if err != nil {
		t.Fatalf("read wrapper: %v", err)
	}
	got := string(wrapper)
	for _, want := range []string{
		"cd " + shellQuoteV0(profile.ProjectWorkDir),
		"-C " + shellQuoteV0(profile.ProjectWorkDir),
		shellQuoteV0(filepath.Join(profile.RuntimeWorkDir, CodexAgentPromptFileNameV0)),
		shellQuoteV0(filepath.Join(profile.RuntimeWorkDir, CodexStdoutFileNameV0)),
		shellQuoteV0(filepath.Join(profile.RuntimeWorkDir, CodexStderrFileNameV0)),
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("wrapper no contiene %q:\n%s", want, got)
		}
	}
}

func TestCodexExecResolverV0PromptUsaControlFilesDelRuntime(t *testing.T) {
	profile := codexProfileForTestV0(t)
	_, issues := NewCodexExecResolverV0(profile).
		ResolveExternalAgentProcessCommandV0(context.Background(), codexSpecForTestV0())
	if len(issues) != 0 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}

	prompt, err := os.ReadFile(filepath.Join(profile.RuntimeWorkDir, CodexAgentPromptFileNameV0))
	if err != nil {
		t.Fatalf("read prompt: %v", err)
	}
	for _, want := range []string{
		filepath.Join(profile.RuntimeWorkDir, CodexAgentPacketFileNameV0),
		filepath.Join(profile.RuntimeWorkDir, CodexAgentAckFileNameV0),
		filepath.Join(profile.RuntimeWorkDir, CodexDirectorDecisionsFileNameV0),
		filepath.Join(profile.RuntimeWorkDir, CodexShutdownRequestFileNameV0),
		filepath.Join(profile.RuntimeWorkDir, CodexShutdownCheckpointAckFileNameV0),
		"decision_path:",
		"CHECKPOINT DE APAGADO",
		"codex_shutdown_checkpoint_ack.v0",
		"checkpoint_ready",
		"Es obligatorio solo si objetivo o criterios de cierre lo piden.",
		"PROTOCOLO COMPACTO OBLIGATORIO",
		"$caveman full",
		"Incumplir este protocolo invalida la entrega.",
		"Si escribes decision_path, despues escribe ACK y termina",
		"Write-set permitido:",
		"README.md",
		"Tests obligatorios:",
		"go test ./...",
		`"files":["README.md"]`,
		`"tests":["go test ./..."]`,
	} {
		if !strings.Contains(string(prompt), want) {
			t.Fatalf("prompt no contiene ruta de control %q:\n%s", want, string(prompt))
		}
	}
}

func TestCodexExecResolverV0PromptAdvierteContextoRequeridoTruncado(t *testing.T) {
	profile := codexProfileForTestV0(t)
	spec := codexSpecForTestV0()
	spec.AgentPacket.Context.Entries[0].Truncated = true
	_, issues := NewCodexExecResolverV0(profile).
		ResolveExternalAgentProcessCommandV0(context.Background(), spec)
	if len(issues) != 0 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}

	prompt, err := os.ReadFile(filepath.Join(profile.RuntimeWorkDir, CodexAgentPromptFileNameV0))
	if err != nil {
		t.Fatalf("read prompt: %v", err)
	}
	for _, want := range []string{
		"CONTEXTO TRUNCADO REQUERIDO",
		"contexto_truncado_resuelto",
		"CONSULTA AL DIRECTOR",
	} {
		if !strings.Contains(string(prompt), want) {
			t.Fatalf("prompt no contiene %q:\n%s", want, string(prompt))
		}
	}
}

func TestCodexExecResolverV0NoFiltraDetallesOperacionalesAlRequest(t *testing.T) {
	profile := codexProfileForTestV0(t)
	profile.CodeHomeDir = filepath.Join(t.TempDir(), "codex-home")
	profile.HomeDir = filepath.Join(t.TempDir(), "home")
	profile.Model = "gpt-test"
	profile.Profile = "premium-test"

	req, issues := NewCodexExecResolverV0(profile).
		ResolveExternalAgentProcessCommandV0(context.Background(), codexSpecForTestV0())
	if len(issues) != 0 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}

	public := strings.Join(append(append([]string{req.CommandPath, req.WorkingDir}, req.Args...), req.Env...), " ")
	for _, forbidden := range []string{"gpt-test", "premium-test", "codex-home", "director_decisions.json", "HOME=", "CODEX_HOME", "oauth", "token"} {
		if strings.Contains(public, forbidden) {
			t.Fatalf("request publico filtra %q: %+v", forbidden, req)
		}
	}
}

func codexProfileForTestV0(t *testing.T) CodexConnectorProfileV0 {
	t.Helper()
	root := t.TempDir()
	return CodexConnectorProfileV0{
		SchemaVersion:  CodexConnectorProfileSchemaVersionV0,
		OptIn:          true,
		CommandPath:    filepath.Join(root, "bin", "codex"),
		ProjectWorkDir: filepath.Join(root, "project"),
		RuntimeWorkDir: filepath.Join(root, "project", ".orquesta-codex-runtime", "agent"),
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
		Context: orquestacontext.ContextMaterializedBundleV0{
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
		},
		DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{
			MailboxRef:   "mailbox-ref-001",
			AckRef:       "ack-ref-001",
			ReadinessRef: "readiness-ref-001",
		},
		Policies: []string{"write_set_closed"},
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
