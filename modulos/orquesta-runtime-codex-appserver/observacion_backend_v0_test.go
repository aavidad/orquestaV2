package orquestaruntimecodexappserver

import "testing"

func TestRecolectarObservacionBackendV0NormalizaSnapshotDisponible(t *testing.T) {
	got := RecolectarObservacionBackendV0(SnapshotObservacionBackendV0{
		ActionRequested:          " shutdown ",
		OwnerMarkerValid:         true,
		PreflightOK:              true,
		PanePIDLive:              true,
		ProcessBySocketObserved:  true,
		ProcessByRuntimeObserved: true,
		IssueCode:                " codex_app_server_provider_unauthorized ",
	})
	want := ObservacionBackendV0{
		ActionRequested:          AccionBackendShutdownV0,
		OwnerMarkerObserved:      true,
		OwnerMarkerValid:         true,
		SocketObserved:           true,
		PreflightOK:              true,
		PanePIDLive:              true,
		ProcessBySocketObserved:  true,
		ProcessByRuntimeObserved: true,
		IssueCode:                "codex_app_server_provider_unauthorized",
	}
	if got != want {
		t.Fatalf("got=%+v want=%+v", got, want)
	}
}

func TestRecolectarObservacionBackendV0ConservaOwnerMarkerInvalido(t *testing.T) {
	got := RecolectarObservacionBackendV0(SnapshotObservacionBackendV0{
		ActionRequested:     "ENSURE",
		SessionObserved:     true,
		OwnerMarkerObserved: true,
	})
	want := ObservacionBackendV0{
		ActionRequested:     AccionBackendEnsureV0,
		SessionObserved:     true,
		OwnerMarkerObserved: true,
	}
	if got != want {
		t.Fatalf("got=%+v want=%+v", got, want)
	}
}

func TestRecolectarObservacionBackendV0AlimentaTransicionListo(t *testing.T) {
	obs := RecolectarObservacionBackendV0(SnapshotObservacionBackendV0{
		OwnerMarkerValid: true,
		PreflightOK:      true,
	})
	got, issues := TransicionBackendV0(BackendSocketPendienteV0, obs)
	if got != BackendListoV0 || len(issues) != 0 {
		t.Fatalf("got=%s issues=%v obs=%+v", got, issues, obs)
	}
}
