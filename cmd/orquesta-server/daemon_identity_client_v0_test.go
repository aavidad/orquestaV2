package main

import (
	"net/http"
	"strings"
	"testing"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestVerifyLiveDaemonIdentityV0AceptaSnapshotCorrespondiente(t *testing.T) {
	state := orquestaserver.StateV0{
		Addr:           "placeholder",
		StartedAt:      "2026-05-26T10:00:00Z",
		ProcessRef:     "process-ref-orquesta-server-test",
		DaemonEpochRef: "daemon-epoch-ref-orquesta-server-test",
	}
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"schema_version":"orquesta_server_state.v0","status":"running","addr":"` +
			state.Addr + `","started_at":"` + state.StartedAt + `","process_ref":"` +
			state.ProcessRef + `","daemon_epoch_ref":"` + state.DaemonEpochRef + `"}`))
	}))
	defer server.Close()
	state.Addr = strings.TrimPrefix(server.URL, "http://")

	live, err := verifyLiveDaemonIdentityV0(state)

	if err != nil {
		t.Fatalf("verifyLiveDaemonIdentityV0: %v", err)
	}
	if live.ProcessRef != state.ProcessRef {
		t.Fatalf("live=%+v", live)
	}
}

func TestVerifyLiveDaemonIdentityV0BloqueaMismatchSinFiltrarSnapshot(t *testing.T) {
	state := orquestaserver.StateV0{
		Addr:           "placeholder",
		StartedAt:      "2026-05-26T10:00:00Z",
		ProcessRef:     "process-ref-orquesta-server-snapshot",
		DaemonEpochRef: "daemon-epoch-ref-orquesta-server-snapshot",
		ProjectWorkDir: "/tmp/project-secret",
	}
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"schema_version":"orquesta_server_state.v0","status":"running","addr":"` +
			state.Addr + `","started_at":"2026-05-26T10:00:01Z","process_ref":"process-ref-other","daemon_epoch_ref":"daemon-epoch-other"}`))
	}))
	defer server.Close()
	state.Addr = strings.TrimPrefix(server.URL, "http://")

	_, err := verifyLiveDaemonIdentityV0(state)

	if err == nil || err.Error() != "daemon_identity_mismatch" {
		t.Fatalf("error=%v", err)
	}
	if strings.Contains(err.Error(), "project-secret") || strings.Contains(err.Error(), state.ProcessRef) {
		t.Fatalf("error filtra snapshot: %v", err)
	}
}
