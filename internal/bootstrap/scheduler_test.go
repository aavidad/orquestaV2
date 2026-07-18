package bootstrap

import (
	"context"
	"testing"
	"time"

	"orquesta/internal/application"
)

func TestSchedulerBoundsLaunchBurstAndImmediatelyContinuesNonLaunchWork(t *testing.T) {
	results := []application.ProcessResult{
		{Processed: true, Action: application.ActionLaunchAgent},
		{Processed: true, Action: application.ActionLaunchAgent},
		{Processed: true, Action: application.ActionStopAgent},
		{Processed: true, Action: application.ActionObserveAgent},
		{},
	}
	calls := 0
	scheduler := scheduler{
		workerRef: "worker:test", pollInterval: time.Minute, maxLaunchesPerCycle: 2,
		processNextForTesting: func(context.Context, string) (application.ProcessResult, error) {
			result := results[calls]
			calls++
			return result, nil
		},
	}

	delay, running := scheduler.processCycle(context.Background())
	if !running || delay != 0 || calls != 2 {
		t.Fatalf("launch-limited cycle = delay %s running %t calls %d", delay, running, calls)
	}
	delay, running = scheduler.processCycle(context.Background())
	if !running || delay != time.Minute || calls != len(results) {
		t.Fatalf("non-launch continuation = delay %s running %t calls %d", delay, running, calls)
	}
}
