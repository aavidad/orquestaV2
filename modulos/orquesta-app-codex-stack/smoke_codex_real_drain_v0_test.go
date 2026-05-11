package orquestaappcodexstack

import (
	"context"
	"fmt"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

type codexStackRealSmokeProgrammingDrainV0 struct {
	Drain       codexStackRealSmokeDrainSummaryV0
	Run         orquestacoreworkflow.OrchestrationRunV0
	Descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0
}

type codexStackRealSmokeDrainSummaryV0 struct {
	Status   string
	Attempts int
	Waits    int
}

func codexStackRealSmokeDrainUntilProgrammingDeliveredV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	stores codexStackRealSmokeStoresV0,
	runRef string,
	runtimeWorkDir string,
	maxExternalWaits int,
) codexStackRealSmokeProgrammingDrainV0 {
	t.Helper()
	var last codexStackRealSmokeProgrammingDrainV0
	for cycle := 1; cycle <= 8; cycle++ {
		descriptors := codexStackRealSmokeDescriptorsV0(t, stores.ReceiptStore)
		if err := codexStackRealSmokeWaitForDescriptorsAckV0(
			ctx,
			codexStackRealSmokePendingProgrammingDescriptorsV0(t, stack, runRef, descriptors),
		); err != nil {
			t.Fatalf("ack programacion ciclo=%d: %v\n%s", cycle, err, codexStackRealSmokeDiagnosticsV0(runtimeWorkDir))
		}
		drain, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
			RunRef:               runRef,
			CorrelationID:        fmt.Sprintf("corr-app-stack-real-programming-drain-%03d", cycle),
			MaxBursts:            16,
			MaxStepsPerBurst:     12,
			MaxDispatchesPerWait: 8,
			MaxCommands:          20,
			MaxOutboxPerCycle:    8,
			MaxExternalWaits:     maxExternalWaits,
		})
		if err != nil {
			t.Fatalf("DrainRunV0 programacion ciclo=%d: %v", cycle, err)
		}
		run := mustLoadCodexStackRunForTestV0(t, stack, runRef)
		last = codexStackRealSmokeProgrammingDrainV0{
			Drain: codexStackRealSmokeDrainSummaryV0{
				Status:   string(drain.Status),
				Attempts: len(drain.Attempts),
				Waits:    len(drain.ExternalWaits),
			},
			Run:         run,
			Descriptors: codexStackRealSmokeDescriptorsV0(t, stores.ReceiptStore),
		}
		if codexStackRealSmokeAllRunTasksDeliveredV0(run) {
			return last
		}
		if !drainRunHasPendingExternalAgentsV0(run) &&
			len(compactStringsV0(run.Tasks)) > 0 &&
			len(compactStringsV0(run.DeliveredTasks)) == 0 {
			t.Fatalf("run sin progreso de programacion ciclo=%d tasks=%v delivered=%v", cycle, run.Tasks, run.DeliveredTasks)
		}
	}
	t.Fatalf(
		"programacion incompleta tras ciclos: tasks=%v delivered=%v started=%v status=%s attempts=%d waits=%d",
		last.Run.Tasks,
		last.Run.DeliveredTasks,
		last.Run.StartedAgents,
		last.Drain.Status,
		last.Drain.Attempts,
		last.Drain.Waits,
	)
	return last
}

func codexStackRealSmokePendingProgrammingDescriptorsV0(
	t *testing.T,
	stack StackV0,
	runRef string,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0 {
	t.Helper()
	run := mustLoadCodexStackRunForTestV0(t, stack, runRef)
	pending := make([]orquestaruntimecodexdelivery.CodexReceiptDescriptorV0, 0, len(descriptors))
	for _, descriptor := range codexStackRealSmokeProgrammingReceiptDescriptorsV0(descriptors) {
		if codexStackRealSmokeContainsProjectionPartV0(run.Deliveries, descriptor.Spec.AgentPacket.DeliveryRefs.AckRef) {
			continue
		}
		pending = append(pending, descriptor)
	}
	return pending
}

func codexStackRealSmokeAllRunTasksDeliveredV0(run orquestacoreworkflow.OrchestrationRunV0) bool {
	tasks := compactStringsV0(run.Tasks)
	if len(tasks) == 0 {
		return false
	}
	for _, taskRef := range tasks {
		if !codexStackStringInSetForTestV0(run.DeliveredTasks, taskRef) &&
			!codexStackStringInSetForTestV0(run.ClosedTasks, taskRef) {
			return false
		}
	}
	return true
}
