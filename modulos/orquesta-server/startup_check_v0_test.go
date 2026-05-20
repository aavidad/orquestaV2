package orquestaserver

import (
	"context"
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
