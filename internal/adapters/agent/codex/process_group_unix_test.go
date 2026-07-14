//go:build unix

package codex

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"orquesta/internal/ports"
)

const processTreePIDFile = "grandchild.pid"

func TestAdapterCancellationKillsProcessGroupDescendants(t *testing.T) {
	config := processTreeTestConfig(t)
	adapter := openTestAdapter(t, config)
	request := testRequest(t, "process-group-cancel", "process tree cancellation", 1024)
	launchCtx, cancel := context.WithCancel(context.Background())
	if _, err := adapter.Launch(launchCtx, request); err != nil {
		cancel()
		t.Fatalf("Launch() error = %v", err)
	}
	grandchildPID := awaitGrandchildPID(t, config, request)
	cancel()

	observation := awaitTerminal(t, adapter, request.ExecutionRef)
	if observation.Status != ports.AgentFailed || observation.ErrorCode != CodeExecutionCanceled {
		t.Fatalf("canceled observation = %+v", observation)
	}
	assertProcessGoneWithESRCH(t, grandchildPID)
}

func TestAdapterShutdownKillsProcessGroupDescendants(t *testing.T) {
	config := processTreeTestConfig(t)
	adapter, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = adapter.Shutdown(ctx)
	})
	request := testRequest(t, "process-group-shutdown", "process tree shutdown", 1024)
	if _, err := adapter.Launch(context.Background(), request); err != nil {
		t.Fatalf("Launch() error = %v", err)
	}
	grandchildPID := awaitGrandchildPID(t, config, request)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := adapter.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	assertProcessGoneWithESRCH(t, grandchildPID)

	reopened := openTestAdapter(t, config)
	observation, err := reopened.Observe(context.Background(), request.ExecutionRef)
	if err != nil {
		t.Fatalf("Observe(reopened) error = %v", err)
	}
	if observation.Status != ports.AgentFailed || observation.ErrorCode != CodeExecutionCanceled {
		t.Fatalf("shutdown observation = %+v", observation)
	}
}

func TestAdapterSuccessfulExecutionKillsBackgroundGroupDescendants(t *testing.T) {
	if info, err := os.Stat("/bin/sleep"); err != nil || !info.Mode().IsRegular() {
		t.Skip("background success helper requires /bin/sleep")
	}
	config := testConfig(t)
	adapter := openTestAdapter(t, config)
	request := testRequest(t, "process-group-success", "helper:background-success", 1024)
	if _, err := adapter.Launch(context.Background(), request); err != nil {
		t.Fatalf("Launch() error = %v", err)
	}
	observation := awaitTerminal(t, adapter, request.ExecutionRef)
	if observation.Status != ports.AgentCompleted || string(observation.Content) != "artifact:background-success" {
		t.Fatalf("successful observation = %+v", observation)
	}
	pidPath := filepath.Join(config.WorkRoot, filepath.FromSlash(executionPath(request.ExecutionRef)), backgroundSuccessPIDFile)
	payload, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatalf("ReadFile(background pid) error = %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(payload)))
	if err != nil || pid <= 0 {
		t.Fatalf("background pid %q is invalid: %v", payload, err)
	}
	assertProcessGoneWithESRCH(t, pid)
}

func processTreeTestConfig(t *testing.T) Config {
	t.Helper()
	for _, required := range []string{"/bin/sh", "/bin/sleep"} {
		if info, err := os.Stat(required); err != nil || !info.Mode().IsRegular() {
			t.Skipf("process tree helper requires %s", required)
		}
	}
	helperPath := filepath.Join(t.TempDir(), "process-tree-helper.sh")
	helper := "#!/bin/sh\n" +
		"/bin/sleep 30 &\n" +
		"grandchild=$!\n" +
		"printf '%s\\n' \"$grandchild\" > " + processTreePIDFile + "\n" +
		"wait \"$grandchild\"\n"
	if err := os.WriteFile(helperPath, []byte(helper), 0o700); err != nil {
		t.Fatalf("WriteFile(process tree helper) error = %v", err)
	}
	config := testConfig(t)
	config.Command = helperPath
	config.Timeout = 10 * time.Second
	return config
}

func awaitGrandchildPID(t *testing.T, config Config, request ports.AgentLaunchRequest) int {
	t.Helper()
	pidPath := filepath.Join(config.WorkRoot, filepath.FromSlash(executionPath(request.ExecutionRef)), processTreePIDFile)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		payload, err := os.ReadFile(pidPath)
		if err == nil {
			pid, parseErr := strconv.Atoi(strings.TrimSpace(string(payload)))
			if parseErr != nil || pid <= 0 {
				t.Fatalf("grandchild pid %q is invalid: %v", payload, parseErr)
			}
			return pid
		}
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("ReadFile(grandchild pid) error = %v", err)
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("grandchild pid was not published")
	return 0
}

func assertProcessGoneWithESRCH(t *testing.T, pid int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		err := syscall.Kill(pid, 0)
		if errors.Is(err, syscall.ESRCH) {
			return
		}
		if err != nil {
			t.Fatalf("Kill(%d, 0) error = %v, want ESRCH", pid, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("Kill(%d, 0) did not return ESRCH", pid)
}
