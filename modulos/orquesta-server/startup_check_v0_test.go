package orquestaserver

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRuntimeV0PrepareStartupPersisteReadyV0(t *testing.T) {
	store := &memoryStateStoreV0{}
	check := fakeStartupCheckV0{result: StartupCheckResultV0{
		Status:       StartupCheckStatusReadyV0,
		Ready:        true,
		Message:      "director: orquesta preparada",
		EvidenceRefs: []string{" evidence-startup-ready ", "evidence-startup-ready"},
	}}
	runtime := mustRuntimeForStartupCheckTestV0(t, store, check)

	if err := runtime.prepareStartupV0(context.Background()); err != nil {
		t.Fatalf("prepareStartupV0: %v", err)
	}
	if !store.last.StartupReady ||
		store.last.StartupStatus != StartupCheckStatusReadyV0 ||
		store.last.StartupMessage != "director: orquesta preparada" ||
		len(store.last.StartupEvidenceRefs) != 1 ||
		store.last.StartupEvidenceRefs[0] != "evidence-startup-ready" {
		t.Fatalf("state=%+v", store.last)
	}
}

func TestRuntimeV0PrepareStartupBloqueaSiNoEstaListaV0(t *testing.T) {
	store := &memoryStateStoreV0{}
	check := fakeStartupCheckV0{result: StartupCheckResultV0{
		Status:       "startup_waiting_cleanup",
		Ready:        false,
		Message:      "runs transitorios pendientes",
		EvidenceRefs: []string{"evidence-startup-blocked"},
	}}
	runtime := mustRuntimeForStartupCheckTestV0(t, store, check)

	err := runtime.prepareStartupV0(context.Background())
	if err == nil {
		t.Fatalf("expected startup error")
	}
	if store.last.Status != "startup_blocked" ||
		store.last.StartupReady ||
		store.last.StartupStatus != "startup_waiting_cleanup" ||
		store.last.LastError != "runs transitorios pendientes" {
		t.Fatalf("state=%+v err=%v", store.last, err)
	}
}

func TestRuntimeV0PrepareStartupAuditaSummarySinPathsV0(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "state-secreto")
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:       stateDir,
		ProjectWorkDir: "/tmp/proyecto-secreto",
		RuntimeWorkDir: "/tmp/runtime-secreto",
		TickInterval:   time.Hour,
	}, RuntimeDepsV0{
		StartupCheck: fakeStartupCheckV0{result: StartupCheckResultV0{
			Status:       StartupCheckStatusReadyV0,
			Ready:        true,
			Message:      "ready with /tmp/runtime-secreto",
			EvidenceRefs: []string{"evidence-startup-ready"},
		}},
		StateStore: &memoryStateStoreV0{},
		Clock:      fixedClockV0{now: time.Date(2026, 5, 18, 9, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	if err := runtime.prepareStartupV0(context.Background()); err != nil {
		t.Fatalf("prepareStartupV0: %v", err)
	}

	events := readAuditEventsForTestV0(t, AuditPathV0(runtime.config))
	encoded, err := json.Marshal(events)
	if err != nil {
		t.Fatalf("marshal audit events: %v", err)
	}
	for _, forbidden := range []string{"proyecto-secreto", "runtime-secreto", "state-secreto", "project_work_dir", "runtime_work_dir"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("auditoria startup expone dato sensible %q: %s", forbidden, encoded)
		}
	}
	last := events[len(events)-1]
	if last.Event != "startup_check_ready" ||
		last.Payload["command"] != nil ||
		last.Payload["result"] != nil ||
		last.Payload["command_summary"] == nil ||
		last.Payload["result_summary"] == nil {
		t.Fatalf("auditoria startup no compacta: %+v", last)
	}
}

func mustRuntimeForStartupCheckTestV0(
	t *testing.T,
	store *memoryStateStoreV0,
	check StartupCheckPortV0,
) *RuntimeV0 {
	t.Helper()
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:     t.TempDir(),
		TickInterval: time.Hour,
	}, RuntimeDepsV0{
		StartupCheck: check,
		StateStore:   store,
		Clock:        fixedClockV0{now: time.Date(2026, 5, 18, 9, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	return runtime
}

type fakeStartupCheckV0 struct {
	result StartupCheckResultV0
	err    error
}

func (fake fakeStartupCheckV0) PrepareStartupV0(
	context.Context,
	StartupCheckCommandV0,
) (StartupCheckResultV0, error) {
	return fake.result, fake.err
}
