package orquestaruntimecodexappserver

import "testing"

func TestTransicionBackendV0InexistenteSinRecursos(t *testing.T) {
	got, issues := TransicionBackendV0("", ObservacionBackendV0{})
	if got != BackendInexistenteV0 || len(issues) != 0 {
		t.Fatalf("got=%s issues=%v", got, issues)
	}
}

func TestTransicionBackendV0PreparandoASocketPendiente(t *testing.T) {
	got, issues := TransicionBackendV0(BackendPreparandoV0, ObservacionBackendV0{
		SessionObserved:  true,
		OwnerMarkerValid: true,
	})
	if got != BackendSocketPendienteV0 || len(issues) != 0 {
		t.Fatalf("got=%s issues=%v", got, issues)
	}
}

func TestTransicionBackendV0SocketPendienteAListoConSocketYPreflightOK(t *testing.T) {
	got, issues := TransicionBackendV0(BackendSocketPendienteV0, ObservacionBackendV0{
		OwnerMarkerValid: true,
		SocketObserved:   true,
		PreflightOK:      true,
	})
	if got != BackendListoV0 || len(issues) != 0 {
		t.Fatalf("got=%s issues=%v", got, issues)
	}
}

func TestTransicionBackendV0SocketPendienteDegradadoSiSesionMuere(t *testing.T) {
	got, issues := TransicionBackendV0(BackendSocketPendienteV0, ObservacionBackendV0{})
	if got != BackendDegradadoV0 || !contieneIssueBackendTestV0(issues, BackendTransitionIssueSessionDiedV0) {
		t.Fatalf("got=%s issues=%v", got, issues)
	}
}

func TestTransicionBackendV0ListoDegradadoConIssueCode(t *testing.T) {
	got, issues := TransicionBackendV0(BackendListoV0, ObservacionBackendV0{
		IssueCode: "codex_app_server_provider_unauthorized",
	})
	if got != BackendDegradadoV0 || !contieneIssueBackendTestV0(issues, "codex_app_server_provider_unauthorized") {
		t.Fatalf("got=%s issues=%v", got, issues)
	}
}

func TestTransicionBackendV0ListoApagandoConKillPedidoYPIDVivo(t *testing.T) {
	got, issues := TransicionBackendV0(BackendListoV0, ObservacionBackendV0{
		ActionRequested: AccionBackendShutdownV0,
		PanePIDLive:     true,
	})
	if got != BackendApagandoV0 || len(issues) != 0 {
		t.Fatalf("got=%s issues=%v", got, issues)
	}
}

func TestTransicionBackendV0ApagandoApagadoSinSesionNiPID(t *testing.T) {
	got, issues := TransicionBackendV0(BackendApagandoV0, ObservacionBackendV0{})
	if got != BackendApagadoV0 || len(issues) != 0 {
		t.Fatalf("got=%s issues=%v", got, issues)
	}
}

func TestTransicionBackendV0SesionSinOwnerEsHuerfano(t *testing.T) {
	got, issues := TransicionBackendV0(BackendPreparandoV0, ObservacionBackendV0{
		SessionObserved:     true,
		OwnerMarkerObserved: true,
		OwnerMarkerValid:    false,
	})
	if got != BackendHuerfanoV0 || !contieneIssueBackendTestV0(issues, BackendTransitionIssueOwnerInvalidV0) {
		t.Fatalf("got=%s issues=%v", got, issues)
	}
}

func TestTransicionBackendV0RechazaTransicionesIlegales(t *testing.T) {
	got, issues := TransicionBackendV0(BackendApagadoV0, ObservacionBackendV0{
		SessionObserved: true,
	})
	if got != BackendHuerfanoV0 || !contieneIssueBackendTestV0(issues, BackendTransitionIssueIllegalV0) {
		t.Fatalf("got=%s issues=%v", got, issues)
	}
}

func contieneIssueBackendTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
