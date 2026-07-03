package orquestaruntimecodexappserver

import (
	"testing"

	"pgregory.net/rapid"
)

func TestTransicionBackendV0PropEstadoIdempotenteConObservacionRepetidaV0(t *testing.T) {
	// Invariant: repetir la misma observacion no cambia de nuevo el estado alcanzado.
	rapid.Check(t, func(rt *rapid.T) {
		actual := estadoBackendRapidV0().Draw(rt, "actual")
		obs := observacionBackendRapidV0().Draw(rt, "obs")
		primero, _ := TransicionBackendV0(actual, obs)
		segundo, _ := TransicionBackendV0(primero, obs)
		if segundo != primero {
			rt.Fatalf("actual=%s obs=%+v primero=%s segundo=%s", actual, obs, primero, segundo)
		}
	})
}

func TestTransicionBackendV0PropListoSoloConSocketPreflightYAnclaV0(t *testing.T) {
	// Invariant: el estado listo solo aparece con socket, preflight y una ancla de propiedad/proceso.
	rapid.Check(t, func(rt *rapid.T) {
		actual := estadoBackendRapidV0().Draw(rt, "actual")
		obs := observacionBackendRapidV0().Draw(rt, "obs")
		got, _ := TransicionBackendV0(actual, obs)
		if got != BackendListoV0 {
			return
		}
		if !obs.SocketObserved || !obs.PreflightOK ||
			!(obs.OwnerMarkerValid || obs.SessionObserved || obs.ProcessBySocketObserved || obs.ProcessByRuntimeObserved) {
			rt.Fatalf("listo sin prerequisitos: actual=%s obs=%+v", actual, obs)
		}
	})
}

func TestTransicionBackendV0PropOwnerInvalidoSiempreHuerfanoV0(t *testing.T) {
	// Invariant: un owner marker invalido corta a huerfano con issue explicito.
	rapid.Check(t, func(rt *rapid.T) {
		actual := estadoBackendRapidV0().Draw(rt, "actual")
		obs := observacionBackendRapidV0().Draw(rt, "obs")
		obs.OwnerMarkerObserved = true
		obs.OwnerMarkerValid = false
		got, issues := TransicionBackendV0(actual, obs)
		if got != BackendHuerfanoV0 || !contieneIssueBackendTestV0(issues, BackendTransitionIssueOwnerInvalidV0) {
			rt.Fatalf("actual=%s obs=%+v got=%s issues=%v", actual, obs, got, issues)
		}
	})
}

func TestTransicionBackendV0PropApagadoConTrabajoVivoEsIlegalSinAccionApagadoV0(t *testing.T) {
	// Invariant: desde apagado no se adopta trabajo vivo sin marcar transicion ilegal.
	rapid.Check(t, func(rt *rapid.T) {
		obs := ObservacionBackendV0{
			ActionRequested: rapid.SampledFrom([]string{"", AccionBackendEnsureV0}).Draw(rt, "action"),
			SessionObserved: rapid.Bool().Draw(rt, "session"),
			PanePIDLive:     rapid.Bool().Draw(rt, "pid"),
			SocketObserved:  rapid.Bool().Draw(rt, "socket"),
		}
		if !backendTieneTrabajoVivoV0(obs) {
			obs.SessionObserved = true
		}
		got, issues := TransicionBackendV0(BackendApagadoV0, obs)
		if got != BackendHuerfanoV0 || !contieneIssueBackendTestV0(issues, BackendTransitionIssueIllegalV0) {
			rt.Fatalf("obs=%+v got=%s issues=%v", obs, got, issues)
		}
	})
}

func TestTransicionBackendV0PropShutdownSinTrabajoVivoTerminaApagadoV0(t *testing.T) {
	// Invariant: shutdown o cleanup sin trabajo vivo siempre convergen a apagado.
	rapid.Check(t, func(rt *rapid.T) {
		actual := estadoBackendRapidV0().Draw(rt, "actual")
		obs := ObservacionBackendV0{
			ActionRequested: rapid.SampledFrom([]string{AccionBackendShutdownV0, AccionBackendCleanupV0}).Draw(rt, "action"),
		}
		got, issues := TransicionBackendV0(actual, obs)
		if got != BackendApagadoV0 || contieneIssueBackendTestV0(issues, BackendTransitionIssueIllegalV0) {
			rt.Fatalf("actual=%s got=%s issues=%v", actual, got, issues)
		}
	})
}

func estadoBackendRapidV0() *rapid.Generator[EstadoBackendAppServerV0] {
	return rapid.SampledFrom([]EstadoBackendAppServerV0{
		"",
		BackendInexistenteV0,
		BackendPreparandoV0,
		BackendSocketPendienteV0,
		BackendListoV0,
		BackendDegradadoV0,
		BackendApagandoV0,
		BackendApagadoV0,
		BackendHuerfanoV0,
	})
}

func observacionBackendRapidV0() *rapid.Generator[ObservacionBackendV0] {
	return rapid.Custom(func(rt *rapid.T) ObservacionBackendV0 {
		return ObservacionBackendV0{
			ActionRequested:          rapid.SampledFrom([]string{"", AccionBackendEnsureV0, AccionBackendShutdownV0, AccionBackendCleanupV0}).Draw(rt, "action"),
			SessionObserved:          rapid.Bool().Draw(rt, "session"),
			OwnerMarkerObserved:      rapid.Bool().Draw(rt, "owner_observed"),
			OwnerMarkerValid:         rapid.Bool().Draw(rt, "owner_valid"),
			SocketObserved:           rapid.Bool().Draw(rt, "socket"),
			PreflightOK:              rapid.Bool().Draw(rt, "preflight"),
			PanePIDLive:              rapid.Bool().Draw(rt, "pid"),
			ProcessBySocketObserved:  rapid.Bool().Draw(rt, "process_socket"),
			ProcessByRuntimeObserved: rapid.Bool().Draw(rt, "process_runtime"),
			IssueCode:                rapid.SampledFrom([]string{"", "codex_app_server_provider_unauthorized"}).Draw(rt, "issue"),
		}
	})
}
