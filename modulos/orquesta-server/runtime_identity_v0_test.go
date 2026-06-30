package orquestaserver

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestServerRuntimeIdentityV0SeProyectaSinFiltrarPathV0(t *testing.T) {
	now := time.Date(2026, 6, 30, 18, 45, 0, 0, time.UTC)
	tracker := NewStatusTrackerV0(ConfigV0{
		RuntimeIdentity: ServerRuntimeIdentityV0{
			BinaryPath:   "/tmp/orquesta-secret/bin/orquesta-server",
			BinarySHA256: strings.Repeat("a", 64),
			BuildRef:     "build-ref-orquesta-server-test",
			CommitRef:    "commit-ref-runtime-test",
			EvidenceRefs: []string{"evidence-ref-runtime-test"},
		},
	}, now)
	tracker.MarkServingV0("127.0.0.1:8787", now)
	state := tracker.MarkStartupReadyV0(StartupCheckResultV0{
		Status: StartupCheckStatusReadyV0,
		Ready:  true,
	}, now)

	status := NewServerPublicStatusV0(state)
	readiness := NewServerReadinessV0(state)

	if status.RuntimeIdentity.BinaryPathRef != ServerRuntimeBinaryPathRefV0 ||
		status.RuntimeIdentity.BinaryName != "orquesta-server" ||
		status.RuntimeIdentity.BinarySHA256 != strings.Repeat("a", 64) ||
		status.RuntimeIdentity.BuildRef != "build-ref-orquesta-server-test" ||
		status.RuntimeIdentity.CommitRef != "commit-ref-runtime-test" ||
		status.RuntimeIdentity.StartedAt != "2026-06-30T18:45:00Z" {
		t.Fatalf("status runtime identity=%+v", status.RuntimeIdentity)
	}
	if readiness.RuntimeIdentity.BinarySHA256 != status.RuntimeIdentity.BinarySHA256 ||
		readiness.RuntimeIdentity.BuildRef != status.RuntimeIdentity.BuildRef {
		t.Fatalf("readiness runtime identity=%+v status=%+v", readiness.RuntimeIdentity, status.RuntimeIdentity)
	}
	body, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("marshal status: %v", err)
	}
	if strings.Contains(string(body), "orquesta-secret") ||
		strings.Contains(string(body), `"binary_path"`) {
		t.Fatalf("status filtra path interno: %s", string(body))
	}
}

func TestStatusTrackerRestoreV0ReemplazaRuntimeIdentityDurablePorBinarioVivoV0(t *testing.T) {
	now := time.Date(2026, 6, 30, 19, 0, 0, 0, time.UTC)
	durable := StateV0{
		SchemaVersion: StateSchemaVersionV0,
		RuntimeIdentity: ServerRuntimeIdentityV0{
			BinaryPath:   "/tmp/old/orquesta-server",
			BinarySHA256: strings.Repeat("b", 64),
			BuildRef:     "build-ref-old",
		},
	}
	tracker := NewStatusTrackerFromDurableStateV0(ConfigV0{
		RuntimeIdentity: ServerRuntimeIdentityV0{
			BinaryPath:   "/tmp/new/orquesta-server",
			BinarySHA256: strings.Repeat("c", 64),
			BuildRef:     "build-ref-new",
		},
	}, durable, now)

	state := tracker.SnapshotV0()
	if state.RuntimeIdentity.BinarySHA256 != strings.Repeat("c", 64) ||
		state.RuntimeIdentity.BuildRef != "build-ref-new" ||
		strings.Contains(state.RuntimeIdentity.BinaryPath, "/tmp/old") {
		t.Fatalf("runtime identity restaurada=%+v", state.RuntimeIdentity)
	}
}
