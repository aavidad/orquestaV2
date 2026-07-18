package codex

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"orquesta/internal/ports"
)

func TestAdapterCapacityIsTemporaryAndCreatesNothingUntilSlotIsReleased(t *testing.T) {
	config := testConfig(t)
	config.MaxConcurrentExecutions = 1
	adapter := openTestAdapter(t, config)

	first := testRequest(t, "capacity-first", "helper:block", 1024)
	firstContext, cancelFirst := context.WithCancel(context.Background())
	defer cancelFirst()
	firstReceipt, err := adapter.Launch(firstContext, first)
	if err != nil {
		t.Fatalf("Launch(first) error = %v", err)
	}
	repeatedReceipt, err := adapter.Launch(context.Background(), first)
	if err != nil || repeatedReceipt != firstReceipt {
		t.Fatalf("idempotent Launch(first) receipt=%+v error=%v", repeatedReceipt, err)
	}

	second := testRequest(t, "capacity-second", "helper:success", 1024)
	if _, err := adapter.Launch(context.Background(), second); ErrorCode(err) != CodeCapacityUnavailable ||
		!isTemporaryCodexError(err) || !isDefinitelyNotAppliedCodexError(err) {
		t.Fatalf("Launch(second) error=%v code=%q temporary=%v unapplied=%v",
			err, ErrorCode(err), isTemporaryCodexError(err), isDefinitelyNotAppliedCodexError(err))
	}
	if _, _, found, err := adapter.loadLaunchRecord(second.ExecutionRef); err != nil || found {
		t.Fatalf("capacity rejection durable record: found=%v error=%v", found, err)
	}
	secondRunPath := filepath.Join(config.WorkRoot, filepath.FromSlash(executionPath(second.ExecutionRef)))
	if _, err := os.Lstat(secondRunPath); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("capacity rejection run path error = %v, want not exists", err)
	}

	cancelFirst()
	firstTerminal := awaitTerminal(t, adapter, first.ExecutionRef)
	if firstTerminal.Status != ports.AgentFailed || firstTerminal.ErrorCode != CodeExecutionCanceled {
		t.Fatalf("first terminal = %+v", firstTerminal)
	}
	if _, err := adapter.Launch(context.Background(), second); err != nil {
		t.Fatalf("Launch(second after release) error = %v", err)
	}
	secondTerminal := awaitTerminal(t, adapter, second.ExecutionRef)
	if secondTerminal.Status != ports.AgentCompleted || string(secondTerminal.Content) != "artifact:success" {
		t.Fatalf("second terminal = %+v", secondTerminal)
	}
}

func TestAdapterRejectsMissingCapacityAndErrorTemporaryUsesFlag(t *testing.T) {
	config := testConfig(t)
	config.MaxConcurrentExecutions = 0
	if adapter, err := New(config); adapter != nil || ErrorCode(err) != CodeMaxConcurrentInvalid || isTemporaryCodexError(err) {
		t.Fatalf("New() adapter=%v error=%v code=%q temporary=%v", adapter, err, ErrorCode(err), isTemporaryCodexError(err))
	}
	temporary := &Error{Code: CodeCapacityUnavailable, TemporaryFailure: true}
	permanent := &Error{Code: CodeCapacityUnavailable}
	ambiguous := &Error{Code: CodeUnavailable, TemporaryFailure: true}
	if !temporary.Temporary() || permanent.Temporary() || !temporary.DefinitelyNotApplied() ||
		!permanent.DefinitelyNotApplied() || ambiguous.DefinitelyNotApplied() {
		t.Fatalf("temporary=%v permanent=%v capacity_unapplied=%v ambiguous_unapplied=%v",
			temporary.Temporary(), permanent.Temporary(), temporary.DefinitelyNotApplied(), ambiguous.DefinitelyNotApplied())
	}
}

func isTemporaryCodexError(err error) bool {
	var adapterErr *Error
	return errors.As(err, &adapterErr) && adapterErr.Temporary()
}

func isDefinitelyNotAppliedCodexError(err error) bool {
	var adapterErr interface{ DefinitelyNotApplied() bool }
	return errors.As(err, &adapterErr) && adapterErr.DefinitelyNotApplied()
}
