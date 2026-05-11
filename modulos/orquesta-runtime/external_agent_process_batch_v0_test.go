package orquestaruntime

import (
	"context"
	"testing"
)

func TestExternalAgentProcessBatchV0CorrelacionaTresPerfilesConBloqueoParcial(t *testing.T) {
	specs := []ExternalAgentLaunchSpecV0{
		externalAgentLaunchSpecPerfilV0(t, "a"),
		externalAgentLaunchSpecPerfilV0(t, "b"),
		externalAgentLaunchSpecPerfilV0(t, "c"),
	}
	runtime := newControlledExternalAgentBatchRuntimeV0()
	items := []ExternalAgentProcessBatchItemV0{
		externalAgentBatchItemV0(t, "batch-item-a", specs[0], nil, runtime),
		externalAgentBatchItemV0(t, "batch-item-b", specs[1], []ExternalAgentConnectorErrorV0{
			externalAgentProcessErrorV0(ExternalAgentCommandResolutionInvalidaV0, "command_ref", "", false),
		}, runtime),
		externalAgentBatchItemV0(t, "batch-item-c", specs[2], nil, runtime),
	}

	done := make(chan []ExternalAgentProcessBatchResultV0, 1)
	go func() {
		done <- LaunchExternalAgentProcessBatchV0(context.Background(), items, 2)
	}()
	waitExternalAgentBatchRuntimeEntryV0(t, runtime)
	waitExternalAgentBatchRuntimeEntryV0(t, runtime)
	runtime.releaseLaunchesV0(2)

	results := waitExternalAgentBatchResultsV0(t, done)

	if len(results) != 3 {
		t.Fatalf("resultados=%d, want 3", len(results))
	}
	requireExternalAgentBatchCorrelationV0(t, results[0], 0, "batch-item-a", specs[0])
	requireExternalAgentBatchCorrelationV0(t, results[1], 1, "batch-item-b", specs[1])
	requireExternalAgentBatchCorrelationV0(t, results[2], 2, "batch-item-c", specs[2])
	requireExternalAgentProcessStatusV0(t, results[0].Result, ExternalAgentProcessLaunchStartedV0)
	requireExternalAgentProcessStatusV0(t, results[1].Result, ExternalAgentProcessLaunchBlockedV0)
	requireExternalAgentProcessStatusV0(t, results[2].Result, ExternalAgentProcessLaunchStartedV0)
	requireExternalAgentResultCodeV0(t, results[1].Result, ExternalAgentCommandResolutionInvalidaV0)
	if results[1].Result.Issues[0].CorrelationID != specs[1].CorrelationID {
		t.Fatalf("issue sin correlacion del spec: %+v", results[1].Result.Issues[0])
	}
	if runtime.maxActiveSeenV0() > 2 {
		t.Fatalf("max active=%d, want <=2", runtime.maxActiveSeenV0())
	}
}

func TestExternalAgentProcessBatchV0MaxConcurrencyCeroCuentaComoUno(t *testing.T) {
	runtime := newControlledExternalAgentBatchRuntimeV0()
	items := []ExternalAgentProcessBatchItemV0{
		externalAgentBatchItemV0(t, "batch-item-a", externalAgentLaunchSpecPerfilV0(t, "a"), nil, runtime),
		externalAgentBatchItemV0(t, "batch-item-b", externalAgentLaunchSpecPerfilV0(t, "b"), nil, runtime),
		externalAgentBatchItemV0(t, "batch-item-c", externalAgentLaunchSpecPerfilV0(t, "c"), nil, runtime),
	}

	done := make(chan []ExternalAgentProcessBatchResultV0, 1)
	go func() {
		done <- LaunchExternalAgentProcessBatchV0(context.Background(), items, 0)
	}()
	for range items {
		waitExternalAgentBatchRuntimeEntryV0(t, runtime)
		if runtime.maxActiveSeenV0() != 1 {
			t.Fatalf("max active=%d, want 1", runtime.maxActiveSeenV0())
		}
		runtime.releaseLaunchesV0(1)
	}

	results := waitExternalAgentBatchResultsV0(t, done)

	for _, result := range results {
		requireExternalAgentProcessStatusV0(t, result.Result, ExternalAgentProcessLaunchStartedV0)
	}
}

func TestExternalAgentProcessBatchV0ReintentaLaunchRuntimeRetryable(t *testing.T) {
	runtime := &retryExternalAgentBatchRuntimeV0{}
	spec := externalAgentLaunchSpecPerfilV0(t, "retry")
	items := []ExternalAgentProcessBatchItemV0{
		externalAgentBatchItemV0(t, "batch-item-retry", spec, nil, runtime),
	}

	results := LaunchExternalAgentProcessBatchV0(context.Background(), items, 1)

	if len(results) != 1 {
		t.Fatalf("results=%d", len(results))
	}
	requireExternalAgentProcessStatusV0(t, results[0].Result, ExternalAgentProcessLaunchStartedV0)
	if runtime.callsV0() != 2 {
		t.Fatalf("calls=%d want 2", runtime.callsV0())
	}
}
