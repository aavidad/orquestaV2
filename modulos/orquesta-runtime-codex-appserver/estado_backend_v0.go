package orquestaruntimecodexappserver

import "strings"

type EstadoBackendAppServerV0 string

const (
	BackendInexistenteV0     EstadoBackendAppServerV0 = "inexistente"
	BackendPreparandoV0      EstadoBackendAppServerV0 = "preparando"
	BackendSocketPendienteV0 EstadoBackendAppServerV0 = "socket_pendiente"
	BackendListoV0           EstadoBackendAppServerV0 = "listo"
	BackendDegradadoV0       EstadoBackendAppServerV0 = "degradado"
	BackendApagandoV0        EstadoBackendAppServerV0 = "apagando"
	BackendApagadoV0         EstadoBackendAppServerV0 = "apagado"
	BackendHuerfanoV0        EstadoBackendAppServerV0 = "huerfano"
)

const (
	AccionBackendEnsureV0   = "ensure"
	AccionBackendShutdownV0 = "shutdown"
	AccionBackendCleanupV0  = "cleanup"
)

const (
	BackendTransitionIssueIllegalV0       = "codex_app_server_backend_transition_illegal"
	BackendTransitionIssueSessionDiedV0   = "codex_app_server_backend_session_died"
	BackendTransitionIssueOwnerInvalidV0  = "codex_app_server_backend_owner_invalid"
	BackendTransitionIssueSocketMissingV0 = "codex_app_server_backend_socket_missing"
)

type ObservacionBackendV0 struct {
	ActionRequested          string
	SessionObserved          bool
	OwnerMarkerObserved      bool
	OwnerMarkerValid         bool
	SocketObserved           bool
	PreflightOK              bool
	PanePIDLive              bool
	ProcessBySocketObserved  bool
	ProcessByRuntimeObserved bool
	IssueCode                string
}

func TransicionBackendV0(
	actual EstadoBackendAppServerV0,
	obs ObservacionBackendV0,
) (EstadoBackendAppServerV0, []string) {
	actual = NormalizarEstadoBackendV0(actual)
	obs.ActionRequested = strings.TrimSpace(obs.ActionRequested)
	obs.IssueCode = strings.TrimSpace(obs.IssueCode)
	issues := []string{}
	if obs.IssueCode != "" {
		issues = append(issues, obs.IssueCode)
	}
	if obs.OwnerMarkerObserved && !obs.OwnerMarkerValid {
		return BackendHuerfanoV0, compactServerStackStringsV0(append(issues, BackendTransitionIssueOwnerInvalidV0))
	}
	if accionBackendApagadoV0(obs.ActionRequested) {
		if backendTieneTrabajoVivoV0(obs) {
			return BackendApagandoV0, compactServerStackStringsV0(issues)
		}
		return BackendApagadoV0, compactServerStackStringsV0(issues)
	}
	if actual == BackendApagandoV0 {
		if backendTieneTrabajoVivoV0(obs) {
			return BackendApagandoV0, compactServerStackStringsV0(issues)
		}
		return BackendApagadoV0, compactServerStackStringsV0(issues)
	}
	if obs.IssueCode != "" {
		return BackendDegradadoV0, compactServerStackStringsV0(issues)
	}
	if obs.SocketObserved && obs.PreflightOK && (obs.OwnerMarkerValid || obs.SessionObserved || backendTieneProcesoV0(obs)) {
		return BackendListoV0, compactServerStackStringsV0(issues)
	}
	if actual == BackendListoV0 && !backendTieneTrabajoVivoV0(obs) {
		return BackendDegradadoV0, compactServerStackStringsV0(append(issues, BackendTransitionIssueSessionDiedV0))
	}
	if actual == BackendSocketPendienteV0 {
		if !obs.SessionObserved && !backendTieneProcesoV0(obs) {
			return BackendDegradadoV0, compactServerStackStringsV0(append(issues, BackendTransitionIssueSessionDiedV0))
		}
		return BackendSocketPendienteV0, compactServerStackStringsV0(append(issues, BackendTransitionIssueSocketMissingV0))
	}
	if actual == BackendPreparandoV0 || obs.ActionRequested == AccionBackendEnsureV0 {
		if obs.SessionObserved || obs.OwnerMarkerValid || backendTieneProcesoV0(obs) {
			return BackendSocketPendienteV0, compactServerStackStringsV0(issues)
		}
		return BackendPreparandoV0, compactServerStackStringsV0(issues)
	}
	if actual == BackendApagadoV0 && backendTieneTrabajoVivoV0(obs) {
		return BackendHuerfanoV0, compactServerStackStringsV0(append(issues, BackendTransitionIssueIllegalV0))
	}
	if actual == BackendInexistenteV0 && !backendTieneTrabajoVivoV0(obs) {
		return BackendInexistenteV0, compactServerStackStringsV0(issues)
	}
	if backendTieneTrabajoVivoV0(obs) {
		return BackendSocketPendienteV0, compactServerStackStringsV0(issues)
	}
	return actual, compactServerStackStringsV0(append(issues, BackendTransitionIssueIllegalV0))
}

func NormalizarEstadoBackendV0(state EstadoBackendAppServerV0) EstadoBackendAppServerV0 {
	switch state {
	case BackendInexistenteV0,
		BackendPreparandoV0,
		BackendSocketPendienteV0,
		BackendListoV0,
		BackendDegradadoV0,
		BackendApagandoV0,
		BackendApagadoV0,
		BackendHuerfanoV0:
		return state
	default:
		return BackendInexistenteV0
	}
}

func backendTieneTrabajoVivoV0(obs ObservacionBackendV0) bool {
	return obs.SessionObserved ||
		obs.PanePIDLive ||
		obs.SocketObserved ||
		backendTieneProcesoV0(obs)
}

func backendTieneProcesoV0(obs ObservacionBackendV0) bool {
	return obs.ProcessBySocketObserved || obs.ProcessByRuntimeObserved
}

func accionBackendApagadoV0(action string) bool {
	return action == AccionBackendShutdownV0 || action == AccionBackendCleanupV0
}
