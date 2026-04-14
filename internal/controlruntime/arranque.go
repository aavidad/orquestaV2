package controlruntime

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"orquesta/runtimeagente"
)

type SolicitudArranque struct {
	Agente   string
	Proyecto string
	Plan         *runtimeagente.LaunchPlan
	TimeoutReady time.Duration
}

type ProcesoArrancado struct {
	PID               int
	HandleKind        string
	HandleRef         string
	ExternalSessionID string
	StdinPath         string
	StdinRawPath      string
	LogPath           string
	WorkingDir        string
	WrappedCommand    string
	RenderedCommand   string
	MetadataJSON      string
	CapabilitiesJSON  string
}

func ArrancarPlan(req SolicitudArranque) (*ProcesoArrancado, error) {
	if req.Plan == nil {
		return nil, fmt.Errorf("plan obligatorio")
	}
	if isRemoteTransport(req.Plan.Transporte) {
		return arrancarPlanRemoto(req)
	}
	return arrancarPlanLocal(req)
}

func arrancarPlanLocal(req SolicitudArranque) (*ProcesoArrancado, error) {
	if strings.TrimSpace(req.Plan.Comando) == "" {
		return nil, fmt.Errorf("comando obligatorio")
	}
	backend := localTerminalBackend(req)
	switch backend {
	case "tmux":
		return arrancarPlanLocalTMUX(req)
	case "pty":
		return arrancarPlanLocalPTY(req)
	}
	if localTerminalRuntimeRequiresTMUX(req) {
		return nil, fmt.Errorf("runtime local process_pty_cli deshabilitado para este agente; requiere tmux")
	}
	return arrancarPlanLocalPTY(req)
}

func localTerminalBackend(req SolicitudArranque) string {
	if req.Plan != nil && req.Plan.Env != nil {
		if backend := strings.ToLower(strings.TrimSpace(req.Plan.Env["ORQUESTA_TERMINAL_BACKEND"])); backend != "" {
			return backend
		}
	}
	if backend := strings.ToLower(strings.TrimSpace(os.Getenv("ORQUESTA_TERMINAL_BACKEND"))); backend != "" {
		return backend
	}
	if localTerminalBackendShouldPreferTMUX(req) {
		return "tmux"
	}
	return ""
}

func localTerminalBackendShouldPreferTMUX(req SolicitudArranque) bool {
	if req.Plan == nil {
		return false
	}
	if !localTerminalRuntimeRequiresTMUX(req) {
		return false
	}
	if path := strings.TrimSpace(commandPathFromLaunchPlan(req.Plan)); path != "" {
		return true
	}
	_, err := exec.LookPath("tmux")
	return err == nil
}

func localTerminalRuntimeRequiresTMUX(req SolicitudArranque) bool {
	if req.Plan == nil {
		return false
	}
	rendered := runtimeagente.RenderCommand(req.Plan)
	return renderedCommandLooksLikeTMUXPreferredCLI(rendered)
}

func RenderedCommandLooksLikeTMUXPreferredCLI(rendered string) bool {
	return renderedCommandLooksLikeTMUXPreferredCLI(rendered)
}

func renderedCommandLooksLikeTMUXPreferredCLI(rendered string) bool {
	lower := strings.ToLower(strings.TrimSpace(rendered))
	if lower == "" {
		return false
	}
	if renderedCommandLooksLikeCodexCLI(rendered) {
		return true
	}
	first := lower
	if fields := strings.Fields(lower); len(fields) > 0 {
		first = fields[0]
	}
	first = strings.Trim(first, "'\"")
	base := filepath.Base(first)
	switch {
	case strings.Contains(lower, "claude-perfil"),
		strings.Contains(lower, "claude-code"),
		strings.Contains(lower, "/claude"),
		lower == "claude",
		strings.HasPrefix(lower, "claude "),
		base == "claude",
		base == "claude-code",
		strings.Contains(lower, "gemini-perfil"),
		strings.Contains(lower, "/gemini"),
		lower == "gemini",
		strings.HasPrefix(lower, "gemini "),
		base == "gemini",
		strings.Contains(lower, "/ollama"),
		lower == "ollama",
		strings.HasPrefix(lower, "ollama "),
		base == "ollama",
		// Modelos locales que corren vía ollama CLI
		strings.HasPrefix(lower, "ollama run "),
		strings.HasPrefix(lower, "ollama serve "):
		return true
	default:
		return false
	}
}

func commandPathFromLaunchPlan(plan *runtimeagente.LaunchPlan) string {
	if plan == nil || plan.Env == nil {
		return ""
	}
	if path := strings.TrimSpace(plan.Env["ORQUESTA_TMUX_BIN"]); path != "" {
		return path
	}
	return strings.TrimSpace(os.Getenv("ORQUESTA_TMUX_BIN"))
}

func sanitizePathFragment(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	if v == "" {
		return "runtime"
	}
	var b strings.Builder
	for _, r := range v {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}
