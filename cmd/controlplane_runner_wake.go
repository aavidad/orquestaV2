package cmd

import (
	"sync/atomic"

	"orquesta/db"
	"orquesta/planocontrol"
)

var activeControlPlaneRunner atomic.Pointer[planocontrol.Runner]

func init() {
	runtimesService.SetAfterEnqueueRuntimeOrderHook(func(order *db.RuntimeOrder, orderID int64) {
		wakeControlPlaneRuntimeOrders()
	})
	runtimesService.SetAfterCreateRuntimeMailboxHook(func(msg *db.RuntimeMailboxMessage, mailboxID int64) {
		wakeControlPlaneRuntimeMailbox()
	})
	runtimesService.SetAfterRegisterRuntimeTranscriptHook(func(entry *db.RuntimeTranscriptEntry, transcriptID int64) {
		wakeControlPlaneRuntimeOrders()
	})
}

func registerActiveControlPlaneRunner(runner *planocontrol.Runner) func() {
	activeControlPlaneRunner.Store(runner)
	return func() {
		activeControlPlaneRunner.CompareAndSwap(runner, nil)
	}
}

func wakeControlPlaneRuntimeOrders() bool {
	runner := activeControlPlaneRunner.Load()
	if runner == nil {
		return false
	}
	return runner.WakeRuntimeOrders()
}

func wakeControlPlaneRuntimeMailbox() bool {
	runner := activeControlPlaneRunner.Load()
	if runner == nil {
		return false
	}
	return runner.WakeRuntimeMailbox()
}

func wakeControlPlaneWarm() bool {
	runner := activeControlPlaneRunner.Load()
	if runner == nil {
		return false
	}
	return runner.WakeBatch("control_plane_warm")
}
