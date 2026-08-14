package fake

import (
	"context"
	"reflect"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func launchForControl(t *testing.T, adapter *Adapter, suffix string) ports.AgentLaunchReceipt {
	t.Helper()
	request := validRequest(t)
	request.ExecutionRef, _ = goal.NewExecutionRef("execution:" + suffix)
	request.WorkItemRef, _ = goal.NewWorkItemRef("work:" + suffix)
	request.IdempotencyKey = "launch:" + suffix
	receipt, err := adapter.Launch(context.Background(), request)
	if err != nil {
		t.Fatalf("Launch(%s) error = %v", suffix, err)
	}
	return receipt
}

func stopForControl(launch ports.AgentLaunchReceipt, mode ports.AgentStopMode, key string) ports.AgentStopRequest {
	return ports.AgentStopRequest{
		ExecutionRef: launch.ExecutionRef, GoalRef: launch.GoalRef, WorkItemRef: launch.WorkItemRef,
		PlanGeneration: launch.PlanGeneration, AppSpecGeneration: launch.AppSpecGeneration,
		ExecutionAttempt: launch.ExecutionAttempt, LaunchActionFence: launch.LaunchActionFence, SpecHash: launch.SpecHash,
		ProviderRef: launch.ProviderRef, ModelRef: launch.ModelRef, AgentRef: launch.AgentRef,
		ExternalRef: launch.ExternalRef, Mode: mode, IdempotencyKey: key,
	}
}

func newControlAdapter(t *testing.T, capabilities *ports.AgentControlCapabilities) *Adapter {
	t.Helper()
	adapter, err := New(Config{
		ProviderRef: "provider:fake", MediaType: "text/plain", Content: []byte("done"),
		Now: func() time.Time { return time.Unix(100, 0).UTC() }, ControlCapabilities: capabilities,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return adapter
}

func TestFakeStopIsSelectiveAndIdempotent(t *testing.T) {
	adapter := newControlAdapter(t, nil)
	capabilities, err := adapter.ControlCapabilities(context.Background())
	if err != nil || !capabilities.CooperativeStop || !capabilities.ForcedStop {
		t.Fatalf("ControlCapabilities() = %+v, %v", capabilities, err)
	}
	a := launchForControl(t, adapter, "a")
	b := launchForControl(t, adapter, "b")
	request := stopForControl(b, ports.AgentStopForced, "stop:b")
	crossed := request
	crossed.LaunchActionFence++
	if _, err := adapter.Stop(context.Background(), crossed); ports.AgentContractErrorCode(err) != "agent.stop_launch_action_fence_mismatch" {
		t.Fatalf("crossed launch fence error = %v", err)
	}
	first, err := adapter.Stop(context.Background(), request)
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	second, err := adapter.Stop(context.Background(), request)
	if err != nil || !reflect.DeepEqual(first, second) || first.Status != ports.AgentStopped {
		t.Fatalf("idempotent Stop() = %+v/%+v, %v", first, second, err)
	}
	if first.ReceiptRef == "" || first.ReceiptRef != second.ReceiptRef {
		t.Fatalf("unstable stop receipt ref: first=%q second=%q", first.ReceiptRef, second.ReceiptRef)
	}
	if _, err := adapter.Observe(context.Background(), a.ExecutionRef); err != nil {
		t.Fatalf("sibling execution was affected: %v", err)
	}
	if _, err := adapter.Observe(context.Background(), b.ExecutionRef); err == nil || err.Error() != "fake_agent.execution_stopped" {
		t.Fatalf("stopped execution observation error = %v", err)
	}
	already, err := adapter.Stop(context.Background(), stopForControl(
		b, ports.AgentStopCooperative, "stop:b:second-key",
	))
	if err != nil || already.Status != ports.AgentStopAlreadyStopped {
		t.Fatalf("second-key stopped truth = %+v, %v", already, err)
	}
	conflict := request
	conflict.Mode = ports.AgentStopCooperative
	if _, err := adapter.Stop(context.Background(), conflict); err == nil || err.Error() != "fake_agent.stop_conflict" {
		t.Fatalf("same-key mutation error = %v", err)
	}
}

func TestFakeStopReportsMismatchTerminalAndUnsupportedWithoutFalseStop(t *testing.T) {
	capabilities := ports.AgentControlCapabilities{CooperativeStop: true}
	adapter := newControlAdapter(t, &capabilities)
	launch := launchForControl(t, adapter, "target")

	mismatch := stopForControl(launch, ports.AgentStopCooperative, "stop:mismatch")
	mismatch.SpecHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if _, err := adapter.Stop(context.Background(), mismatch); ports.AgentContractErrorCode(err) != "agent.stop_spec_hash_mismatch" {
		t.Fatalf("target mismatch error = %v", err)
	}

	unsupportedRequest := stopForControl(launch, ports.AgentStopForced, "stop:unsupported")
	unsupported, err := adapter.Stop(context.Background(), unsupportedRequest)
	if err != nil || unsupported.Status != ports.AgentStopUnsupported ||
		unsupported.ReceiptRef != "" || !unsupported.ConfirmedAt.IsZero() {
		t.Fatalf("unsupported Stop() = %+v, %v", unsupported, err)
	}
	cooperativeRequest := stopForControl(launch, ports.AgentStopCooperative, "stop:cooperative")
	confirmed, err := adapter.Stop(context.Background(), cooperativeRequest)
	if err != nil || confirmed.Status != ports.AgentStopped {
		t.Fatalf("supported Stop() = %+v, %v", confirmed, err)
	}
	replayed, err := adapter.Stop(context.Background(), unsupportedRequest)
	if err != nil || !reflect.DeepEqual(replayed, unsupported) {
		t.Fatalf("unsupported replay = %+v, %v", replayed, err)
	}
	laterUnsupported := stopForControl(launch, ports.AgentStopForced, "stop:unsupported-after-terminal")
	if _, err := adapter.Stop(context.Background(), laterUnsupported); err != nil {
		t.Fatalf("later unsupported Stop() error = %v", err)
	}
	if _, err := adapter.Observe(context.Background(), launch.ExecutionRef); err == nil || err.Error() != "fake_agent.execution_stopped" {
		t.Fatalf("unsupported request resurrected stopped execution: %v", err)
	}

	completedAdapter := newControlAdapter(t, nil)
	completedLaunch := launchForControl(t, completedAdapter, "completed")
	if _, err := completedAdapter.Observe(context.Background(), completedLaunch.ExecutionRef); err != nil {
		t.Fatal(err)
	}
	terminal, err := completedAdapter.Stop(context.Background(), stopForControl(completedLaunch, ports.AgentStopCooperative, "stop:completed"))
	if err != nil || terminal.Status != ports.AgentStopAlreadyCompleted {
		t.Fatalf("terminal Stop() = %+v, %v", terminal, err)
	}
}
