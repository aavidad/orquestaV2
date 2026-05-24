package main

import (
	"context"
	"testing"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestIdleSelfImprovementStackV0RechazaCandidatoNoEjecutableV0(t *testing.T) {
	queue := &fakeIdleSelfRunQueueV0{candidates: []orquestarunqueue.RunSchedulingCandidateV0{{
		RunRef: "run-ref-1", Status: orquestarunqueue.RunStatusPausedV0,
	}}}
	stack := &orquestaappcodexstack.StackV0{
		Stores:   orquestaappcodexstack.StoresV0{RunQueue: queue},
		RunQueue: orquestaappcodexstack.RunQueueConfigV0{QueueRef: "queue-main"},
	}
	result := (serverStackSupervisorV0{stack: stack}).ensureIdleSelfImprovementQueueVisibleV0(
		context.Background(),
		orquestaserver.IdleSelfImprovementResultV0{Accepted: true, RunRef: "run-ref-1"},
	)
	if result.Accepted || result.Message != "idle_self_improvement_queue_candidate_not_executable" {
		t.Fatalf("result=%+v", result)
	}
}
