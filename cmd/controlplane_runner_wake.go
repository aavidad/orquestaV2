package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"orquesta/db"
	"orquesta/planocontrol"
)

var activeControlPlaneRunner atomic.Pointer[planocontrol.Runner]
var runtimeOrdersWakeFallbackRunning atomic.Bool
var runtimeMailboxWakeFallbackRunning atomic.Bool
var runtimeOrdersWakeFallbackStartedAt atomic.Int64
var runtimeMailboxWakeFallbackStartedAt atomic.Int64
var runtimeTranscriptDirectWakeGate = planocontrol.NewThrottler()

var processRuntimeOrdersWakeFallback func()
var processRuntimeMailboxWakeFallback func()
var wakeRuntimeOrdersAfterTranscriptEntryFn = wakeControlPlaneRuntimeOrders

func init() {
	processRuntimeOrdersWakeFallback = runRuntimeOrdersWakeFallback
	processRuntimeMailboxWakeFallback = runRuntimeMailboxWakeFallback
	runtimesService.SetAfterEnqueueRuntimeOrderHook(func(order *db.RuntimeOrder, orderID int64) {
		if activeControlPlaneRunner.Load() != nil {
			wakeControlPlaneRuntimeOrders()
		}
	})
	runtimesService.SetAfterCreateRuntimeMailboxHook(func(msg *db.RuntimeMailboxMessage, mailboxID int64) {
		wakeControlPlaneRuntimeMailbox()
	})
	runtimesService.SetAfterRegisterRuntimeTranscriptHook(func(entry *db.RuntimeTranscriptEntry, transcriptID int64) {
		wakeRuntimeOrdersAfterTranscriptEntry(entry)
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
	if runner != nil {
		return runner.WakeRuntimeOrders()
	}
	return triggerRuntimeOrdersWakeFallback()
}

func wakeControlPlaneRuntimeMailbox() bool {
	runner := activeControlPlaneRunner.Load()
	if runner != nil {
		if !runner.WakeRuntimeMailbox() {
			return false
		}
		resetRuntimeMailboxReevaluationGate()
		return true
	}
	return triggerRuntimeMailboxWakeFallback()
}

func wakeControlPlaneWarm() bool {
	runner := activeControlPlaneRunner.Load()
	if runner == nil {
		return false
	}
	if !runner.WakeBatch("control_plane_warm") {
		return false
	}
	resetAutonomiaActiveSessionsObservationGate()
	return true
}

func wakeRuntimeOrdersAfterTranscriptEntry(entry *db.RuntimeTranscriptEntry) bool {
	if activeControlPlaneRunner.Load() == nil {
		return wakeRuntimeOrdersAfterTranscriptEntryFn()
	}
	if !runtimeTranscriptDirectWakeGate.Allow(runtimeTranscriptDirectWakeKey(entry), runtimeTranscriptDirectWakeInterval()) {
		return false
	}
	return wakeRuntimeOrdersAfterTranscriptEntryFn()
}

func runtimeTranscriptDirectWakeInterval() time.Duration {
	return 2 * time.Second
}

func runtimeTranscriptDirectWakeKey(entry *db.RuntimeTranscriptEntry) string {
	if entry == nil {
		return "global"
	}
	if entry.HandleID != nil && *entry.HandleID > 0 {
		return "handle:" + strconv.FormatInt(*entry.HandleID, 10)
	}
	if entry.RuntimeID > 0 {
		return "runtime:" + strconv.FormatInt(entry.RuntimeID, 10)
	}
	if agente := strings.TrimSpace(entry.Agente); agente != "" {
		return "agente:" + strings.ToLower(agente)
	}
	return "global"
}

func triggerRuntimeOrdersWakeFallback() bool {
	releaseStuckRuntimeOrdersWakeFallback()
	if !runtimeOrdersWakeFallbackRunning.CompareAndSwap(false, true) {
		return false
	}
	runtimeOrdersWakeFallbackStartedAt.Store(time.Now().UTC().UnixNano())
	go func() {
		defer runtimeOrdersWakeFallbackRunning.Store(false)
		defer runtimeOrdersWakeFallbackStartedAt.Store(0)
		processRuntimeOrdersWakeFallback()
	}()
	return true
}

func triggerRuntimeMailboxWakeFallback() bool {
	releaseStuckRuntimeMailboxWakeFallback()
	if !runtimeMailboxWakeFallbackRunning.CompareAndSwap(false, true) {
		return false
	}
	runtimeMailboxWakeFallbackStartedAt.Store(time.Now().UTC().UnixNano())
	go func() {
		defer runtimeMailboxWakeFallbackRunning.Store(false)
		defer runtimeMailboxWakeFallbackStartedAt.Store(0)
		processRuntimeMailboxWakeFallback()
	}()
	return true
}

func runtimeWakeFallbackTimeout() time.Duration {
	return 15 * time.Second
}

func runtimeWakeFallbackDrainPasses() int {
	return 4
}

func releaseStuckRuntimeOrdersWakeFallback() {
	releaseStuckRuntimeWakeFallback(&runtimeOrdersWakeFallbackRunning, &runtimeOrdersWakeFallbackStartedAt)
}

func releaseStuckRuntimeMailboxWakeFallback() {
	releaseStuckRuntimeWakeFallback(&runtimeMailboxWakeFallbackRunning, &runtimeMailboxWakeFallbackStartedAt)
}

func releaseStuckRuntimeWakeFallback(flag *atomic.Bool, startedAt *atomic.Int64) {
	if flag == nil || startedAt == nil || !flag.Load() {
		return
	}
	raw := startedAt.Load()
	if raw <= 0 {
		return
	}
	if time.Since(time.Unix(0, raw)) < runtimeWakeFallbackTimeout() {
		return
	}
	flag.Store(false)
	startedAt.Store(0)
}

func runRuntimeOrdersWakeFallback() {
	automation := dbAutomationService{}
	if recovered, err := automation.ReconciliarRuntimeOrdersStale(); err != nil {
		automation.Audit("server", "runtime_orders_stale_error", "runtime_order", 0, err.Error())
	} else if recovered > 0 {
		automation.Audit("server", "runtime_orders_stale", "runtime_order", 0, fmt.Sprintf("Órdenes stale reconciliadas en fallback: %d", recovered))
	}
	total := 0
	for pass := 0; pass < runtimeWakeFallbackDrainPasses(); pass++ {
		count, err := automation.ProcesarRuntimeOrdersBatch()
		if err != nil {
			automation.Audit("server", "runtime_orders_batch_error", "runtime_order", 0, err.Error())
			return
		}
		total += count
		if count <= 0 {
			break
		}
	}
	if total > 0 {
		automation.Audit("server", "runtime_orders_batch", "runtime_order", 0, fmt.Sprintf("Órdenes procesadas en fallback: %d", total))
		wakeControlPlaneRuntimeMailbox()
	}
}

func runRuntimeMailboxWakeFallback() {
	resetRuntimeMailboxReevaluationGate()
	automation := dbAutomationService{}
	count, err := automation.ProcesarRuntimeMailboxBatch()
	if err != nil {
		automation.Audit("server", "runtime_mailbox_batch_error", "runtime_mailbox", 0, err.Error())
		return
	}
	if count > 0 {
		automation.Audit("server", "runtime_mailbox_batch", "runtime_mailbox", 0, fmt.Sprintf("Mailbox runtime procesado: %d", count))
	}
}
