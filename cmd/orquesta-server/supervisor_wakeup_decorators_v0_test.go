package main

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunfile "orquesta/modulos/orquesta-run-file"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func TestServerSupervisorWakeupDecoratorsV0DisparanSoloTrasMutacionesEjecutables(t *testing.T) {
	ctx := context.Background()
	var causes []string
	relay := &serverSupervisorWakeupRelayV0{
		request: func(cause string) bool {
			causes = append(causes, cause)
			return true
		},
	}

	runStore := serverWakeupRunStoreV0{
		inner:  orquestacionnucleoapp.NewInMemoryRunStoreV0(),
		wakeup: relay,
	}
	if err := runStore.SaveRunV0(ctx, orquestacoreworkflow.OrchestrationRunV0{
		RunID: "run-ref-wakeup-store-001",
	}); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}

	fileStore, err := orquestarunfile.NewRunFileStoreV0(t.TempDir())
	if err != nil {
		t.Fatalf("NewRunFileStoreV0: %v", err)
	}
	queue := serverWakeupRunQueueV0{inner: fileStore, wakeup: relay}
	if _, err := queue.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef: "run-ref-wakeup-ready-001", QueueRef: "queue-main", Status: orquestarunqueue.RunStatusReadyV0,
	}); err != nil {
		t.Fatalf("SetRunPriorityV0 ready: %v", err)
	}
	if _, err := queue.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef: "run-ref-wakeup-closed-001", QueueRef: "queue-main", Status: orquestarunqueue.RunStatusClosedV0,
	}); err != nil {
		t.Fatalf("SetRunPriorityV0 closed: %v", err)
	}

	control := serverWakeupRunControlV0{inner: fileStore, wakeup: relay}
	if _, err := control.PauseRunV0(ctx, orquestaruncontrol.PauseRunCommandV0{
		RunRef: "run-ref-wakeup-ready-001",
	}); err != nil {
		t.Fatalf("PauseRunV0: %v", err)
	}
	if _, err := control.ResumeRunV0(ctx, orquestaruncontrol.ResumeRunCommandV0{
		RunRef: "run-ref-wakeup-ready-001",
	}); err != nil {
		t.Fatalf("ResumeRunV0: %v", err)
	}

	want := []string{"run_store_saved", "run_queue_ready", "run_control_resume"}
	if !stringSlicesEqualV0(causes, want) {
		t.Fatalf("causes=%v want=%v", causes, want)
	}
}

func stringSlicesEqualV0(got []string, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}
