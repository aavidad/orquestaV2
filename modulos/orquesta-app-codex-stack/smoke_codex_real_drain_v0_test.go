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
	result, err := codexStackRealSmokeDrainUntilProgrammingDeliveredResultV0(
		t,
		ctx,
		stack,
		stores,
		runRef,
		maxExternalWaits,
	)
	if err != nil {
		t.Fatalf("%v\n%s", err, codexStackRealSmokeDiagnosticsV0(runtimeWorkDir))
	}
	return result
}

func codexStackRealSmokeDrainUntilProgrammingDeliveredResultV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	stores codexStackRealSmokeStoresV0,
	runRef string,
	maxExternalWaits int,
) (codexStackRealSmokeProgrammingDrainV0, error) {
	t.Helper()
	var last codexStackRealSmokeProgrammingDrainV0
	previousSequence := mustLoadCodexStackRunForTestV0(t, stack, runRef).LastSequence
	previousFingerprint := codexStackRealSmokeDrainFingerprintV0(t, stores, stack, runRef)
	for cycle := 1; cycle <= 8; cycle++ {
		drain, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
			RunRef:               runRef,
			CorrelationID:        fmt.Sprintf("corr-app-stack-real-programming-drain-%03d", cycle),
			MaxBursts:            16,
			MaxStepsPerBurst:     12,
			MaxDispatchesPerWait: 8,
			MaxCommands:          20,
			MaxOutboxPerCycle:    8,
			MaxExternalWaits:     codexStackRealSmokeDrainCycleMaxExternalWaitsV0(maxExternalWaits),
		})
		if err != nil {
			return last, fmt.Errorf("DrainRunV0 programacion ciclo=%d: %w", cycle, err)
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
		fingerprint := codexStackRealSmokeDrainFingerprintFromRunV0(last.Run, last.Descriptors)
		progressed := run.LastSequence != previousSequence || fingerprint != previousFingerprint
		previousSequence = run.LastSequence
		previousFingerprint = fingerprint
		if codexStackRealSmokeAllRunTasksDeliveredV0(run) {
			return last, nil
		}
		if cause := codexStackRealSmokeProgrammingCausalIssueV0(run, last.Descriptors); cause != "" {
			return last, fmt.Errorf(
				"programacion incompleta con causa observable ciclo=%d: %s status=%s attempts=%d waits=%d",
				cycle,
				cause,
				last.Drain.Status,
				last.Drain.Attempts,
				last.Drain.Waits,
			)
		}
		if !drainRunHasPendingExternalAgentsV0(run) && !progressed {
			return last, fmt.Errorf(
				"run sin progreso de programacion ciclo=%d tasks=%v delivered=%v sequence=%d",
				cycle,
				run.Tasks,
				run.DeliveredTasks,
				run.LastSequence,
			)
		}
	}
	return last, fmt.Errorf(
		"programacion incompleta tras ciclos: tasks=%v delivered=%v started=%v status=%s attempts=%d waits=%d",
		last.Run.Tasks,
		last.Run.DeliveredTasks,
		last.Run.StartedAgents,
		last.Drain.Status,
		last.Drain.Attempts,
		last.Drain.Waits,
	)
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
