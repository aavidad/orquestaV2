package cmd

import "orquesta/db"

func init() {
	db.RegisterLifecycleHookSubscriber(handleControlPlaneLifecycleHook)
}

func handleControlPlaneLifecycleHook(ev db.LifecycleHookEvent) {
	switch ev.Evento {
	case db.HookBeforeTool,
		db.HookAfterTool,
		db.HookOnWorkerFail,
		db.HookOnHandoff,
		db.HookOnStop,
		db.HookOnRecovery,
		db.HookProjectBlocked,
		db.HookProjectUnblocked,
		db.HookTaskStart,
		db.HookTaskFinish,
		db.HookSessionPark,
		db.HookSessionResume:
	default:
		return
	}
	invalidateStatusSnapshotCache()
	wakeControlPlaneWarm()
}
