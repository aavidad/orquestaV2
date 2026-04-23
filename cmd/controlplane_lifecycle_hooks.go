package cmd

import "orquesta/db"

func init() {
	db.RegisterLifecycleHookSubscriber(handleControlPlaneLifecycleHook)
}

func handleControlPlaneLifecycleHook(ev db.LifecycleHookEvent) {
	switch ev.Evento {
	case db.HookProjectBlocked,
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
