package orquestaruntimecodex

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestCodexExecResolverV0AceptaSandboxDangerFullAccess(t *testing.T) {
	profile := codexProfileForTestV0(t)
	profile.Sandbox = "danger-full-access"

	req, issues := NewCodexExecResolverV0(profile).
		ResolveExternalAgentProcessCommandV0(context.Background(), codexSpecForTestV0())

	if len(issues) != 0 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}
	if req.CommandPath == "" {
		t.Fatalf("request no materializado")
	}
}

func TestCodexExecResolverV0BloqueaExtraArgsQueEscapanWorkspace(t *testing.T) {
	profile := codexProfileForTestV0(t)
	profile.ExtraArgs = []string{"--add-dir", "/tmp"}

	_, issues := NewCodexExecResolverV0(profile).
		ResolveExternalAgentProcessCommandV0(context.Background(), codexSpecForTestV0())

	requireCodexIssueV0(t, issues, CodexConnectorValueInvalidV0)
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
	profile.RuntimeWorkDir = profile.ProjectWorkDir

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
		"PROTOCOLO ACK FUERA DEL PROYECTO",
		"No uses apply_patch para escribirlo",
		"No crees docs/codigo en ese directorio de control",
		"La linea visible ACK no sustituye este JSON",
		"Es obligatorio solo si objetivo o criterios de cierre lo piden.",
		"Comunicacion compacta",
		"$caveman full",
		"PASO FINAL OBLIGATORIO",
		"escribe SIEMPRE el ACK de control",
		"No uses git status como criterio obligatorio",
		"ACK status completed aunque git no aplique",
		"No imprimas diffs ni pegues artefactos completos",
		"Si escribes decision_path, completa antes los ficheros pedidos del write-set",
		"files debe listar rutas reales de archivos de producto tocados",
		"Seguridad",
		"no borres",
		"no salgas del workdir del proyecto",
		"Si detectas que algun archivo necesario queda fuera del write-set",
		"registra el faltante como nota o tarea derivada",
		"Las rutas del write-set son relativas al workdir del proyecto",
		"No incluyas archivos de control en ACK.files",
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

func TestCodexExecResolverV0PromptNoSugiereGlobsComoFilesDelACK(t *testing.T) {
	packet := codexPacketForTestV0()
	packet.Task.WriteSet = []string{"go.mod", "cmd/server/**", "internal/**", "README.md", "web"}

	prompt := BuildCodexAgentPromptV0(packet, nil)
	ackBlock := prompt
	if before, after, ok := strings.Cut(prompt, "ACK esperado:"); ok {
		ackBlock = after
		if block, _, ok := strings.Cut(ackBlock, "Titulo:"); ok {
			ackBlock = block
		} else {
			ackBlock = before + after
		}
	}

	if strings.Contains(ackBlock, "cmd/server/**") || strings.Contains(ackBlock, "internal/**") ||
		strings.Contains(ackBlock, `"web"`) {
		t.Fatalf("ACK esperado no debe sugerir globs/directorios como files:\n%s", ackBlock)
	}
	if !strings.Contains(ackBlock, `"files":["go.mod","README.md"]`) {
		t.Fatalf("ACK esperado debe sugerir solo ficheros concretos:\n%s", ackBlock)
	}
}

func TestCodexExecResolverV0PromptExigeDecisionPathAlDirector(t *testing.T) {
	packet := codexPacketForTestV0()
	packet.TargetModule = "orquesta-app-stack-director"
	prompt := BuildCodexAgentPromptWithControlFilesV0(
		packet,
		nil,
		CodexControlFilesV0{
			PacketPath:   "/runtime/agent_packet.json",
			AckPath:      "/runtime/agent_ack.json",
			DecisionPath: "/runtime/director_decisions.json",
		},
	)

	for _, want := range []string{
		"Como target_module de director",
		"escribe decision_path con las decisiones ejecutables que puedas",
		"conserva el diagnostico y deja follow-up en ACK",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt no contiene %q:\n%s", want, prompt)
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
		"CONTEXTO TRUNCADO",
		"contexto_truncado_resuelto",
		`"notes":["contexto_truncado_resuelto: \u003cmotivo\u003e"]`,
		"guarda avance parcial",
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
