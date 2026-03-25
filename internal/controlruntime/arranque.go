package controlruntime

import (
	"fmt"
	"os/exec"
	"strings"

	"orquesta/runtimeagente"
)

type SolicitudArranque struct {
	Agente   string
	Proyecto string
	Plan     *runtimeagente.LaunchPlan
}

type ProcesoArrancado struct {
	PID               int
	HandleKind        string
	HandleRef         string
	ExternalSessionID string
	StdinPath         string
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
	if _, err := exec.LookPath("script"); err != nil {
		return nil, fmt.Errorf("script no disponible: %w", err)
	}

	return arrancarPlanLocalPTY(req)
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
