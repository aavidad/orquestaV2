package orquestaappcodexstack

import (
	"context"
	"errors"
	"testing"
	"time"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestDrainRunV0CortaCooperativamenteTrasCancelarContextoV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newDelayedAckCodexStackRuntimeV0(time.Hour))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	waiter := cancelingExternalWaiterV0{cancel: cancel}
	stack.Ports.ExternalWaiter = waiter
	director := postDirectorAPIV0(t, stack)

	drain, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:               director.RunRef,
		CorrelationID:        "corr-stack-drain-context-cancel-001",
		MaxBursts:            1,
		MaxStepsPerBurst:     1,
		MaxDispatchesPerWait: 1,
		MaxCommands:          1,
		MaxOutboxPerCycle:    1,
		MaxDecisionCycles:    1,
		MaxExternalWaits:     2,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v want context.Canceled drain=%+v", err, drain)
	}
	if len(drain.Attempts) != 1 || len(drain.ExternalWaits) != 1 {
		t.Fatalf("drain no corto tras cancelacion: attempts=%d waits=%d drain=%+v", len(drain.Attempts), len(drain.ExternalWaits), drain)
	}
}

type cancelingExternalWaiterV0 struct {
	cancel context.CancelFunc
}

func (waiter cancelingExternalWaiterV0) WaitExternalProgressV0(
	context.Context,
	orquestacionnucleoapp.ExternalProgressWaitRequestV0,
) (orquestacionnucleoapp.ExternalProgressWaitResultV0, error) {
	if waiter.cancel != nil {
		waiter.cancel()
	}
	return orquestacionnucleoapp.ExternalProgressWaitResultV0{
		Continue:     true,
		EvidenceRefs: []string{"evidence-ref-test-context-canceled-wait"},
	}, nil
}
