package cmd

import (
	"fmt"
	"testing"
	"time"

	"orquesta/db"
)

func TestRuntimeMailboxSyncSupervisedHandleWithCooldownSkipsRepeatedBlockedSync(t *testing.T) {
	resetRuntimeMailboxDeferredSupervisionGate()
	t.Cleanup(resetRuntimeMailboxDeferredSupervisionGate)
	runtimeMailboxDeferredSupervisionIntervalOverride = time.Hour
	t.Cleanup(func() { runtimeMailboxDeferredSupervisionIntervalOverride = 0 })

	prev := runtimeMailboxSyncSupervisedHandleInnerFn
	t.Cleanup(func() { runtimeMailboxSyncSupervisedHandleInnerFn = prev })

	calls := 0
	runtimeMailboxSyncSupervisedHandleInnerFn = func(handle *db.RuntimeHandle, runtime *db.RuntimeInstance, source string) (*db.RuntimeHandle, *db.RuntimeInstance, string, error) {
		calls++
		return handle, runtime, "", fmt.Errorf("no se pueden purgar handles con runtime orders vivas asociadas: #11705(pendiente)")
	}

	handle := &db.RuntimeHandle{ID: 41, Agente: "Codex3"}
	if _, _, _, err := runtimeMailboxSyncSupervisedHandleWithCooldown(handle, nil, "runtime_mailbox_supervisor_local"); err == nil {
		t.Fatal("la primera llamada deberia propagar el bloqueo diferible")
	}
	if calls != 1 {
		t.Fatalf("la primera llamada deberia ejecutar la sincronizacion real una vez, got=%d", calls)
	}
	if _, _, _, err := runtimeMailboxSyncSupervisedHandleWithCooldown(handle, nil, "runtime_mailbox_supervisor_local"); err == nil {
		t.Fatal("la segunda llamada deberia mantener el bloqueo en cooldown")
	}
	if calls != 1 {
		t.Fatalf("el cooldown deberia evitar la segunda sincronizacion cara, got=%d", calls)
	}
}

func TestRuntimeMailboxSyncSupervisedHandleWithCooldownRetriesAfterWindow(t *testing.T) {
	resetRuntimeMailboxDeferredSupervisionGate()
	t.Cleanup(resetRuntimeMailboxDeferredSupervisionGate)
	runtimeMailboxDeferredSupervisionIntervalOverride = time.Hour
	t.Cleanup(func() { runtimeMailboxDeferredSupervisionIntervalOverride = 0 })

	prev := runtimeMailboxSyncSupervisedHandleInnerFn
	t.Cleanup(func() { runtimeMailboxSyncSupervisedHandleInnerFn = prev })

	calls := 0
	runtimeMailboxSyncSupervisedHandleInnerFn = func(handle *db.RuntimeHandle, runtime *db.RuntimeInstance, source string) (*db.RuntimeHandle, *db.RuntimeInstance, string, error) {
		calls++
		if calls == 1 {
			return handle, runtime, "", fmt.Errorf("no se pueden purgar handles con runtime orders vivas asociadas: #11705(pendiente)")
		}
		return handle, runtime, "sess-ok", nil
	}

	handle := &db.RuntimeHandle{ID: 41, Agente: "Codex3"}
	if _, _, _, err := runtimeMailboxSyncSupervisedHandleWithCooldown(handle, nil, "runtime_mailbox_supervisor_local"); err == nil {
		t.Fatal("la primera llamada deberia propagar el bloqueo diferible")
	}
	key := runtimeMailboxDeferredSupervisionKey(handle)
	runtimeMailboxDeferredSupervisionGate.Set(key, time.Now().UTC().Add(-2*time.Hour))

	gotHandle, _, externalSessionID, err := runtimeMailboxSyncSupervisedHandleWithCooldown(handle, nil, "runtime_mailbox_supervisor_local")
	if err != nil {
		t.Fatalf("la segunda llamada tras expirar la ventana deberia reintentar, err=%v", err)
	}
	if calls != 2 {
		t.Fatalf("deberia reintentar la sincronizacion real tras expirar el cooldown, got=%d", calls)
	}
	if gotHandle == nil || gotHandle.ID != handle.ID {
		t.Fatalf("handle devuelto inesperado: %+v", gotHandle)
	}
	if externalSessionID != "sess-ok" {
		t.Fatalf("external_session_id inesperada: %q", externalSessionID)
	}
}
