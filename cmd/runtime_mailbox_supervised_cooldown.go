package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"orquesta/db"
	"orquesta/planocontrol"
)

var runtimeMailboxDeferredSupervisionGate = planocontrol.NewThrottler()
var runtimeMailboxDeferredSupervisionIntervalOverride time.Duration
var runtimeMailboxSyncSupervisedHandleInnerFn = db.SincronizarRuntimeHandleSupervisado

func init() {
	runtimeMailboxSyncSupervisedHandleFn = runtimeMailboxSyncSupervisedHandleWithCooldown
}

func runtimeMailboxDeferredSupervisionInterval() time.Duration {
	if runtimeMailboxDeferredSupervisionIntervalOverride > 0 {
		return runtimeMailboxDeferredSupervisionIntervalOverride
	}
	seconds := controlPlaneConfigIntOrDefault("runtime_mailbox_supervised_handle_retry_interval_seconds", 30)
	if seconds <= 0 {
		seconds = 30
	}
	return time.Duration(seconds) * time.Second
}

func runtimeMailboxDeferredSupervisionKey(handle *db.RuntimeHandle) string {
	if handle == nil || handle.ID <= 0 {
		return ""
	}
	scope := strings.TrimSpace(db.CurrentStorageDisplayTarget())
	if scope == "" {
		scope = "global"
	}
	return scope + "|" + strconv.FormatInt(handle.ID, 10)
}

func resetRuntimeMailboxDeferredSupervisionGate() {
	runtimeMailboxDeferredSupervisionGate.Reset()
}

func runtimeMailboxSyncSupervisedHandleWithCooldown(handle *db.RuntimeHandle, runtime *db.RuntimeInstance, source string) (*db.RuntimeHandle, *db.RuntimeInstance, string, error) {
	key := runtimeMailboxDeferredSupervisionKey(handle)
	if key != "" && !runtimeMailboxDeferredSupervisionGate.Allow(key, runtimeMailboxDeferredSupervisionInterval()) {
		return handle, runtime, "", fmt.Errorf("no se pueden purgar handles con runtime orders vivas asociadas: cooldown activo handle=%d source=%s", handle.ID, strings.TrimSpace(source))
	}
	syncFn := runtimeMailboxSyncSupervisedHandleInnerFn
	if syncFn == nil {
		syncFn = db.SincronizarRuntimeHandleSupervisado
	}
	refreshedHandle, refreshedRuntime, externalSessionID, err := syncFn(handle, runtime, source)
	if runtimeMailboxCanDeferSupervisedHandleError(err) {
		if key != "" {
			runtimeMailboxDeferredSupervisionGate.Set(key, time.Now().UTC())
		}
		return refreshedHandle, refreshedRuntime, externalSessionID, err
	}
	if key != "" {
		runtimeMailboxDeferredSupervisionGate.Forget(key)
	}
	return refreshedHandle, refreshedRuntime, externalSessionID, err
}
