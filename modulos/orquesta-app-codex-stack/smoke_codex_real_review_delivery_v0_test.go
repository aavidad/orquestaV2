package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

type codexStackRealSmokeProgrammingDeliveryV0 struct {
	Drain      codexStackRealSmokeDrainSummaryV0
	Run        orquestacoreworkflow.OrchestrationRunV0
	Descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0
}

func codexStackRealSmokeDrainUntilFirstProgrammingDeliveryV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	stores codexStackRealSmokeStoresV0,
	runRef string,
	runtimeWorkDir string,
	maxExternalWaits int,
) codexStackRealSmokeProgrammingDeliveryV0 {
	t.Helper()
	previousSequence := mustLoadCodexStackRunForTestV0(t, stack, runRef).LastSequence
	var lastRun orquestacoreworkflow.OrchestrationRunV0
	for cycle := 1; cycle <= 8; cycle++ {
		drain, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
			RunRef:               runRef,
			CorrelationID:        fmt.Sprintf("corr-app-stack-real-first-delivery-%03d", cycle),
			MaxBursts:            16,
			MaxStepsPerBurst:     12,
			MaxDispatchesPerWait: 8,
			MaxCommands:          20,
			MaxOutboxPerCycle:    8,
			MaxExternalWaits:     maxExternalWaits,
		})
		if err != nil {
			t.Fatalf("DrainRunV0 primera entrega ciclo=%d: %v\n%s", cycle, err, codexStackRealSmokeDiagnosticsV0(runtimeWorkDir))
		}
		run := mustLoadCodexStackRunForTestV0(t, stack, runRef)
		lastRun = run
		descriptors := codexStackRealSmokeDescriptorsV0(t, stores.ReceiptStore)
		if descriptor, ok := codexStackRealSmokeStableProgrammingDescriptorV0(run, descriptors); ok {
			return codexStackRealSmokeProgrammingDeliveryV0{
				Drain: codexStackRealSmokeDrainSummaryV0{
					Status:   string(drain.Status),
					Attempts: len(drain.Attempts),
					Waits:    len(drain.ExternalWaits),
				},
				Run:        run,
				Descriptor: descriptor,
			}
		}
		progressed := run.LastSequence != previousSequence
		previousSequence = run.LastSequence
		if !drainRunHasPendingExternalAgentsV0(run) && !progressed {
			t.Fatalf("sin primera entrega de programacion ciclo=%d phase=%s status=%s agents=%v started=%v failed=%v stopped=%v confirmed=%v artifacts=%v tasks=%v delivered=%v sequence=%d descriptors=%v",
				cycle,
				run.CurrentPhase,
				run.Status,
				run.Agents,
				run.StartedAgents,
				run.FailedAgents,
				run.StoppedAgents,
				run.ConfirmedStoppedAgents,
				run.PhaseArtifacts,
				run.Tasks,
				run.DeliveredTasks,
				run.LastSequence,
				codexStackRealSmokeProgrammingDescriptorsV0(descriptors),
			)
		}
	}
	t.Fatalf("sin primera entrega de programacion tras ciclos: tasks=%v delivered=%v started=%v",
		lastRun.Tasks,
		lastRun.DeliveredTasks,
		lastRun.StartedAgents,
	)
	return codexStackRealSmokeProgrammingDeliveryV0{}
}

func codexStackRealSmokeDeliveredProgrammingDescriptorV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) (orquestaruntimecodexdelivery.CodexReceiptDescriptorV0, bool) {
	for _, descriptor := range codexStackRealSmokeProgrammingReceiptDescriptorsV0(descriptors) {
		ack, ok := codexStackRealSmokeCompletedAckV0(descriptor)
		if !ok || !codexStackRealSmokeContainsProjectionPartV0(run.Deliveries, ack.AckRef) {
			continue
		}
		return descriptor, true
	}
	return orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{}, false
}

func codexStackRealSmokeStableProgrammingDescriptorV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) (orquestaruntimecodexdelivery.CodexReceiptDescriptorV0, bool) {
	if drainRunHasPendingExternalAgentsV0(run) ||
		!codexStackRealSmokeCompletedProgrammingACKsRegisteredV0(run, descriptors) {
		return orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{}, false
	}
	return codexStackRealSmokeDeliveredProgrammingDescriptorV0(run, descriptors)
}

func codexStackRealSmokeCompletedProgrammingACKsRegisteredV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) bool {
	for _, descriptor := range codexStackRealSmokeProgrammingReceiptDescriptorsV0(descriptors) {
		ack, ok := codexStackRealSmokeCompletedAckV0(descriptor)
		if !ok {
			continue
		}
		if !codexStackRealSmokeContainsProjectionPartV0(run.Deliveries, ack.AckRef) {
			return false
		}
	}
	return true
}

func codexStackRealSmokeCompletedAckV0(
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) (orquestaruntimecodex.CodexAgentAckV0, bool) {
	ack, issues := orquestaruntimecodex.ReadAndValidateCodexAgentAckFileV0(descriptor.AckPath, descriptor.Spec)
	if len(issues) > 0 {
		return orquestaruntimecodex.CodexAgentAckV0{}, false
	}
	return ack, strings.TrimSpace(ack.Status) == "completed"
}
