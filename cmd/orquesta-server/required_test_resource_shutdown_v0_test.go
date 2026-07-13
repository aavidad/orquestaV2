package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimerequiredtest "orquesta/modulos/orquesta-runtime-required-test"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestServerRequiredTestResourceShutdownHookV0ClosesReverseAndOnce(t *testing.T) {
	order := []string{}
	first := &recordingRequiredTestCloserV0{name: "required-runner", order: &order, err: errors.New("required close")}
	second := &recordingRequiredTestCloserV0{name: "attestor", order: &order}
	third := &recordingRequiredTestCloserV0{name: "batch-runner", order: &order, err: errors.New("batch close")}
	hook := newServerRequiredTestResourceShutdownHookV0(first, second, third)

	err := hook.ShutdownV0(context.Background())
	if err == nil || !strings.Contains(err.Error(), "required close") || !strings.Contains(err.Error(), "batch close") {
		t.Fatalf("joined close err=%v", err)
	}
	if got := strings.Join(order, ","); got != "batch-runner,attestor,required-runner" {
		t.Fatalf("close order=%q", got)
	}
	if secondErr := hook.ShutdownV0(context.Background()); secondErr == nil || secondErr.Error() != err.Error() {
		t.Fatalf("second close err=%v want=%v", secondErr, err)
	}
	if first.calls != 1 || second.calls != 1 || third.calls != 1 {
		t.Fatalf("close calls=%d/%d/%d", first.calls, second.calls, third.calls)
	}
}

func TestServerRuntimeDepsV0OwnsRequiredTestResourcesFromStack(t *testing.T) {
	runner := &closingRequiredTestRunnerForShutdownV0{}
	stack := orquestaappcodexstack.StackV0{}
	stack.Ports.RequiredTestRunner = runner

	deps := serverRuntimeDepsFromStackV0(
		serverConfigForRequiredTestShutdownV0(t),
		nil,
		nil,
		nil,
		stack,
		serverCodexGoalBackendsV0{},
	)
	if len(deps.ShutdownHooks) != 1 {
		t.Fatalf("shutdown hooks=%d", len(deps.ShutdownHooks))
	}
	if err := deps.ShutdownHooks[0].ShutdownV0(context.Background()); err != nil {
		t.Fatalf("ShutdownV0: %v", err)
	}
	if err := deps.ShutdownHooks[0].ShutdownV0(context.Background()); err != nil {
		t.Fatalf("ShutdownV0 second: %v", err)
	}
	if runner.closeCalls != 1 {
		t.Fatalf("runner close calls=%d", runner.closeCalls)
	}
}

func TestBuildStackRequiredTestWrappersPreserveDescriptorOwnerUntilShutdownV0(t *testing.T) {
	projectDir := t.TempDir()
	commandPath := requiredTestExecutableForEnvTestV0(t)
	before, err := serverRequiredTestTrackedDescriptorsV0(commandPath)
	if err != nil {
		t.Skipf("/proc fd inspection unavailable: %v", err)
	}
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envServerStateDirV0, t.TempDir())
	t.Setenv(envCodexRuntimeWorkDirV0, filepath.Join(t.TempDir(), "runtime"))
	t.Setenv(envCodexCommandV0, filepath.Join(projectDir, "codex-bin"))
	t.Setenv(envRequiredTestRunnerEnabledV0, "1")
	t.Setenv(envRequiredTestGoCommandV0, commandPath)
	t.Setenv(envRequiredTestOutputDirV0, filepath.Join(t.TempDir(), "required-test-output"))
	t.Setenv(envRequiredTestEnvV0, "")
	t.Setenv(envOPESBaseURLV0, "http://127.0.0.1:18082")
	t.Setenv(envOPESTemporalConfirmV0, "1")
	t.Setenv(envOPESBaseURLLegacyV0, "")
	t.Setenv(envDomainWorkFileEnabledV0, "")
	t.Setenv(envDomainWorkFileDirV0, "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	stack, err := buildStackFromEnvV0(config)
	if err != nil {
		t.Fatalf("buildStackFromEnvV0: %v", err)
	}
	hook := serverRequiredTestResourceShutdownHookFromStackV0(stack)
	if hook == nil {
		t.Fatal("real enabled stack did not transfer required-test resource ownership")
	}
	t.Cleanup(func() { _ = hook.ShutdownV0(context.Background()) })
	domainRunner, ok := stack.Ports.RequiredTestRunner.(orquestaappcodexstack.DomainWorkRequiredTestRunnerV0)
	if !ok {
		t.Fatalf("required-test outer wrapper=%T, want DomainWorkRequiredTestRunnerV0", stack.Ports.RequiredTestRunner)
	}
	if innerType := reflect.TypeOf(domainRunner.Inner); innerType == nil || !strings.Contains(innerType.String(), "codexAckRequiredTestRunnerV0") {
		t.Fatalf("required-test inner wrapper=%T, want codexAckRequiredTestRunnerV0", domainRunner.Inner)
	}

	opened, err := serverRequiredTestTrackedDescriptorsV0(commandPath)
	if err != nil {
		t.Fatal(err)
	}
	for fd := range before {
		delete(opened, fd)
	}
	memfdCount, sourceCount := 0, 0
	for _, target := range opened {
		if strings.Contains(target, "memfd:orquesta-allowed-command-v0") {
			memfdCount++
		}
		if target == commandPath {
			sourceCount++
		}
	}
	if memfdCount == 0 || sourceCount == 0 {
		t.Fatalf("real stack descriptors missing: memfd=%d source=%d opened=%v", memfdCount, sourceCount, opened)
	}
	if err := hook.ShutdownV0(context.Background()); err != nil {
		t.Fatalf("ShutdownV0: %v", err)
	}
	for fd, target := range opened {
		if current, err := os.Readlink(filepath.Join("/proc/self/fd", strconv.Itoa(fd))); err == nil {
			t.Fatalf("descriptor %d remained open after real stack shutdown: admitted=%q current=%q", fd, target, current)
		}
	}
}

func TestMCPStdioV0ClosesRequiredTestResourcesAtEOF(t *testing.T) {
	closer := &recordingRequiredTestCloserV0{name: "stdio"}
	hook := newServerRequiredTestResourceShutdownHookV0(closer)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := serveMCPStdioWithRequiredTestResourcesV0(
		strings.NewReader(`{"jsonrpc":"2.0","method":"notifications/initialized"}`),
		&stdout,
		&stderr,
		orquestaappcodexstack.StackV0{}.MCPTransportBindings,
		hook,
	)
	if code != 0 || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if closer.calls != 1 {
		t.Fatalf("stdio close calls=%d", closer.calls)
	}
}

func TestServerRequiredTestRunnerAndBatchCloseTheirExecutorsV0(t *testing.T) {
	newExecutor := func() orquestaruntimerequiredtest.LocalCommandExecutorV0 {
		t.Helper()
		path, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		path, err = filepath.EvalSymlinks(path)
		if err != nil {
			t.Fatal(err)
		}
		executor, err := orquestaruntimerequiredtest.NewLocalCommandExecutorV0(
			orquestaruntimerequiredtest.LocalCommandExecutorV0{
				ProjectWorkDir:  t.TempDir(),
				OutputDir:       t.TempDir(),
				AllowedCommands: map[string]string{"test-bin": path},
				MaxOutputBytes:  4096,
			},
		)
		if err != nil {
			t.Fatalf("NewLocalCommandExecutorV0: %v", err)
		}
		return executor
	}
	assertClosed := func(executor orquestaruntimerequiredtest.LocalCommandExecutorV0) {
		t.Helper()
		result, err := executor.RunRequiredTestCommandV0(context.Background(), orquestacionnucleoapp.RequiredTestCommandExecutionRequestV0{
			RunRef: "run-ref-close", TaskRef: "task-ref-close", TestCommand: "test-bin", CorrelationID: "correlation-ref-close",
		})
		if err != nil || result.Status != orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0 {
			t.Fatalf("closed executor result=%+v err=%v", result, err)
		}
	}

	requiredExecutor := newExecutor()
	requiredRunner := &serverRequiredTestRunnerV0{
		RequiredTestRunnerV0: orquestacionnucleoapp.RequiredTestRunnerV0{Executor: requiredExecutor},
		executor:             requiredExecutor,
	}
	if err := requiredRunner.Close(); err != nil {
		t.Fatalf("required runner Close: %v", err)
	}
	assertClosed(requiredExecutor)

	batchExecutor := newExecutor()
	batchRunner := &serverAutoprogrammingBatchTestRunnerV0{Executor: batchExecutor}
	if err := batchRunner.Close(); err != nil {
		t.Fatalf("batch runner Close: %v", err)
	}
	assertClosed(batchExecutor)
}

func TestGoalRequiredTestWorkspaceSelectorCloseReleasesCanonicalAdapterV0(t *testing.T) {
	projectDir := t.TempDir()
	configPath := writeCompleteGoalRequiredTestAttestationConfigForTestV0(
		t,
		projectDir,
		filepath.Join(t.TempDir(), "attestation-runtime"),
	)
	t.Setenv(envGoalRequiredTestAttestationConfigFileV0, configPath)
	selector, err := goalRequiredTestAttestationWorkspaceSelectorFromConfigV0(
		orquestaserver.ConfigV0{ProjectWorkDir: projectDir},
		serverProjectConfigFileV0{},
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("selector: %v", err)
	}
	if err := selector.Close(); err != nil {
		t.Fatalf("selector Close: %v", err)
	}
	if err := selector.Canonical.PreflightGoalRequiredTestAttestationV0(context.Background()); err == nil {
		t.Fatal("canonical adapter remained usable after selector Close")
	}
}

type recordingRequiredTestCloserV0 struct {
	name  string
	order *[]string
	calls int
	err   error
}

func (closer *recordingRequiredTestCloserV0) Close() error {
	closer.calls++
	if closer.order != nil {
		*closer.order = append(*closer.order, closer.name)
	}
	return closer.err
}

type closingRequiredTestRunnerForShutdownV0 struct {
	closeCalls int
}

func (runner *closingRequiredTestRunnerForShutdownV0) RunRequiredTestsV0(
	context.Context,
	orquestacionnucleoapp.RequiredTestExecutionRequestV0,
) (orquestacionnucleoapp.RequiredTestExecutionResultV0, error) {
	return orquestacionnucleoapp.RequiredTestExecutionResultV0{}, nil
}

func (runner *closingRequiredTestRunnerForShutdownV0) Close() error {
	runner.closeCalls++
	return nil
}

func serverConfigForRequiredTestShutdownV0(t *testing.T) orquestaserver.ConfigV0 {
	t.Helper()
	return orquestaserver.ConfigV0{
		StateDir:       t.TempDir(),
		ProjectWorkDir: t.TempDir(),
	}
}

func serverRequiredTestTrackedDescriptorsV0(commandPath string) (map[int]string, error) {
	entries, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		return nil, err
	}
	tracked := make(map[int]string)
	for _, entry := range entries {
		fd, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		target, err := os.Readlink(filepath.Join("/proc/self/fd", entry.Name()))
		if err != nil {
			continue
		}
		if target == commandPath || strings.Contains(target, "memfd:orquesta-allowed-command-v0") {
			tracked[fd] = target
		}
	}
	return tracked, nil
}
