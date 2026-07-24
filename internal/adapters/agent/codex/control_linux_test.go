//go:build linux

package codex

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"orquesta/internal/credentials"
	"orquesta/internal/ports"
)

func TestReopenedLiveProcessRetainsExactSecretGuards(t *testing.T) {
	for _, test := range []struct {
		name, objective, environment string
		launchReplay                 bool
		provider                     bool
	}{
		{"launch-provider", "helper:restart-provider-leak", codexAPIKeyEnvironment + "=" + helperCredentialInitial, true, true},
		{"observe-session", "helper:restart-session-leak", helperSessionEnvironment + "=" + helperSessionBearer, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := testConfig(t)
			config.SessionResolver = &recoverySessionResolver{material: helperSessionBearer}
			if test.provider {
				config.CredentialStore = &credentialTestStore{material: helperCredentialInitial, version: 1}
				config.CredentialRef = credentials.CredentialRef("credential:codex-primary")
			}
			request := testRequest(t, "recovery-"+test.name, test.objective, 1024)
			request.SessionRef, _ = ports.NewExecutionSessionRef("execution-session:recovery-" + test.name)
			command, _, _ := seedUnownedLiveProcess(t, config, request, test.environment)
			reopened := openTestAdapter(t, config)
			if test.launchReplay {
				if _, err := reopened.Launch(context.Background(), request); err != nil {
					t.Fatalf("Launch(replay): %v", err)
				}
			}
			observation := awaitTerminal(t, reopened, request.ExecutionRef)
			if observation.Status != ports.AgentFailed || observation.ErrorCode != CodeSecretLeak || len(observation.Content) != 0 {
				t.Fatalf("recovered terminal=%+v", observation)
			}
			if err := command.Wait(); err != nil {
				t.Fatalf("Wait: %v", err)
			}
			runRoot := filepath.Join(config.WorkRoot, filepath.FromSlash(executionPath(request.ExecutionRef)))
			if output, err := os.ReadFile(filepath.Join(runRoot, lastMessageFileName)); err != nil || len(output) != 0 {
				t.Fatalf("last message=%q error=%v", output, err)
			}
			if terminal, err := os.ReadFile(filepath.Join(runRoot, terminalFileName)); err != nil ||
				strings.Contains(string(terminal), helperCredentialInitial) ||
				strings.Contains(string(terminal), helperSessionBearer) {
				t.Fatalf("terminal retained secret: %q error=%v", terminal, err)
			}
		})
	}
}

func TestCodexSelectiveStopPreservesSiblingProcessTrees(t *testing.T) {
	config := processTreeTestConfig(t)
	adapter := openTestAdapter(t, config)
	requests := make([]ports.AgentLaunchRequest, 4)
	grandchildren := make([]int, 4)
	for index, suffix := range []string{"control-a", "control-b", "control-c", "control-d"} {
		requests[index] = testRequest(t, suffix, "process tree control "+suffix, 1024)
		if _, err := adapter.Launch(context.Background(), requests[index]); err != nil {
			t.Fatalf("Launch(%s) error = %v", suffix, err)
		}
		grandchildren[index] = awaitGrandchildPID(t, config, requests[index])
	}

	launch := launchReceiptForRequest(t, adapter, requests[1])
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	receipt, err := adapter.Stop(ctx, stopRequestForLaunch(launch, ports.AgentStopForced, "stop:control-b"))
	if err != nil || receipt.Status != ports.AgentStopped {
		t.Fatalf("Stop(B) = %+v, %v", receipt, err)
	}
	assertProcessGoneWithESRCH(t, grandchildren[1])
	for _, index := range []int{0, 2, 3} {
		if err := syscall.Kill(grandchildren[index], 0); err != nil {
			t.Fatalf("sibling %d was stopped: %v", index, err)
		}
	}
}

func TestCodexSelectiveStopAdoptsAfterCrashAndRejectsReusedPID(t *testing.T) {
	t.Run("adopts exact live process", func(t *testing.T) {
		config := processTreeTestConfig(t)
		request := testRequest(t, "adopted-control", "adopted process tree", 1024)
		command, record, launch := seedUnownedLiveProcess(t, config, request)
		grandchild := awaitGrandchildPID(t, config, request)

		adapter := openTestAdapter(t, config)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		receipt, err := adapter.Stop(ctx, stopRequestForLaunch(launch, ports.AgentStopForced, "stop:adopted"))
		if err != nil || receipt.Status != ports.AgentStopped {
			t.Fatalf("adopted Stop() = %+v, %v", receipt, err)
		}
		_ = command.Wait()
		assertProcessGoneWithESRCH(t, grandchild)
		if identity, err := platformInspectProcess(record); err != nil || identity != processIdentityGone {
			t.Fatalf("adopted process identity = %v, %v", identity, err)
		}
	})

	t.Run("retries forced signal after crash between intent and syscall", func(t *testing.T) {
		config := processTreeTestConfig(t)
		request := testRequest(t, "forced-intent-before-syscall", "recover forced intent", 1024)
		command, record, launch := seedUnownedLiveProcess(t, config, request)
		grandchild := awaitGrandchildPID(t, config, request)
		stop := stopRequestForLaunch(launch, ports.AgentStopForced, "stop:forced-intent-before-syscall")
		requestHash, err := hashStopRequest(stop)
		if err != nil {
			t.Fatal(err)
		}

		seeder, err := New(config)
		if err != nil {
			t.Fatal(err)
		}
		seeder.mu.Lock()
		state, runPath, err := seeder.controlTargetLocked(stop)
		if err == nil {
			_, err = seeder.ensureStopRequest(runPath, requestHash, stop)
		}
		if err == nil {
			_, _, err = seeder.ownProcessLocked(state)
		}
		if err == nil {
			var created bool
			_, created, err = seeder.prepareStopSignalIntent(runPath, requestHash, stop)
			if err == nil && !created {
				err = errors.New("forced signal intent was not created")
			}
		}
		// Simulate process crash after intent fsync and before SIGKILL. A real
		// crash releases both owner flock and private root descriptor.
		seeder.releaseProcessOwnershipLocked(state)
		seeder.mu.Unlock()
		if err != nil {
			t.Fatal(err)
		}
		if err := seeder.root.Close(); err != nil {
			t.Fatal(err)
		}
		if identity, err := platformInspectProcess(record); err != nil || identity != processIdentityAlive {
			t.Fatalf("process before forced replay = %v, %v", identity, err)
		}

		reopened := openTestAdapter(t, config)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		receipt, err := reopened.Stop(ctx, stop)
		if err != nil || receipt.Status != ports.AgentStopped {
			t.Fatalf("Stop(recovered forced intent) = %+v, %v", receipt, err)
		}
		if receipt.ReceiptRef == "" {
			t.Fatal("forced stop receipt ref missing")
		}
		_ = command.Wait()
		assertProcessGoneWithESRCH(t, grandchild)
		if _, found, err := reopened.loadStopSignalProof(runPath, requestHash, stop); err != nil || !found {
			t.Fatalf("forced replay proof found=%v err=%v", found, err)
		}
		replay, err := reopened.Stop(context.Background(), stop)
		if err != nil || replay != receipt || replay.ReceiptRef != receipt.ReceiptRef {
			t.Fatalf("forced replay receipt = %+v, %v; want %+v", replay, err, receipt)
		}
	})

	t.Run("request without effect plus natural exit is not stopped", func(t *testing.T) {
		controlRoot := t.TempDir()
		readyPath := filepath.Join(controlRoot, "ready")
		releasePath := filepath.Join(controlRoot, "release")
		helperPath := filepath.Join(controlRoot, "natural-exit.sh")
		helper := "#!/bin/sh\n" +
			"printf ready > \"$CODEX_NATURAL_READY\"\n" +
			"while [ ! -f \"$CODEX_NATURAL_RELEASE\" ]; do /bin/sleep 0.01; done\n"
		if err := os.WriteFile(helperPath, []byte(helper), 0o700); err != nil {
			t.Fatal(err)
		}
		config := testConfig(t)
		config.Command = helperPath
		config.Environment = map[string]string{
			"CODEX_NATURAL_READY": readyPath, "CODEX_NATURAL_RELEASE": releasePath,
		}
		request := testRequest(t, "request-before-natural-exit", "natural exit after request", 1024)
		command, record, launch := seedUnownedLiveProcess(t, config, request)
		awaitPath(t, readyPath)

		stop := stopRequestForLaunch(launch, ports.AgentStopForced, "stop:request-without-effect")
		requestHash, err := hashStopRequest(stop)
		if err != nil {
			t.Fatal(err)
		}
		journal, err := New(config)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := journal.ensureStopRequest(executionPath(request.ExecutionRef), requestHash, stop); err != nil {
			t.Fatal(err)
		}
		// Simulate crash after request fsync and before any signal.
		if err := journal.root.Close(); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(releasePath, []byte("natural\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := command.Wait(); err != nil {
			t.Fatalf("natural process exit = %v", err)
		}
		if identity, err := platformInspectProcess(record); err != nil || identity != processIdentityGone {
			t.Fatalf("natural process identity = %v, %v", identity, err)
		}

		adapter := openTestAdapter(t, config)
		receipt, err := adapter.Stop(context.Background(), stop)
		if err != nil || receipt.Status != ports.AgentStopAlreadyFailed {
			t.Fatalf("Stop(after natural exit) = %+v, %v", receipt, err)
		}
		observation, err := adapter.Observe(context.Background(), request.ExecutionRef)
		if err != nil || observation.ErrorCode != CodeExecutionInterrupted {
			t.Fatalf("Observe(after natural exit) = %+v, %v", observation, err)
		}
	})

	t.Run("durable stopped terminal survives crash before receipt", func(t *testing.T) {
		config := processTreeTestConfig(t)
		request := testRequest(t, "crash-after-stopped-terminal", "crash after stopped terminal", 1024)
		command, _, launch := seedUnownedLiveProcess(t, config, request)
		grandchild := awaitGrandchildPID(t, config, request)
		stop := stopRequestForLaunch(launch, ports.AgentStopForced, "stop:crash-after-terminal")
		requestHash, err := hashStopRequest(stop)
		if err != nil {
			t.Fatal(err)
		}
		seeder, err := New(config)
		if err != nil {
			t.Fatal(err)
		}
		seeder.mu.Lock()
		state, runPath, err := seeder.controlTargetLocked(stop)
		if err == nil {
			_, err = seeder.ensureStopRequest(runPath, requestHash, stop)
		}
		var record processRecord
		var identity processIdentityState
		if err == nil {
			record, identity, err = seeder.ownProcessLocked(state)
		}
		var intent stopSignalIntent
		if err == nil {
			var created bool
			intent, created, err = seeder.prepareStopSignalIntent(runPath, requestHash, stop)
			if err == nil && !created {
				err = errors.New("signal intent was not created")
			}
		}
		if err == nil && identity != processIdentityAlive {
			err = errors.New("process was not alive before stop")
		}
		if err == nil {
			err = signalProcessTree(record, ports.AgentStopForced)
		}
		var proof stopSignalProof
		if err == nil {
			proof, err = seeder.persistStopSignalProof(runPath, intent)
			state.stopProof.Store(&proof)
		}
		seeder.mu.Unlock()
		if err != nil {
			t.Fatal(err)
		}
		if gone, err := waitForExactProcess(context.Background(), record, nil); err != nil || !gone {
			t.Fatalf("waitForExactProcess() = %v, %v", gone, err)
		}
		seeder.mu.Lock()
		err = seeder.finishStoppedProcessLocked(state, proof)
		seeder.mu.Unlock()
		if err != nil {
			t.Fatal(err)
		}
		// Crash after terminal fsync: no stop receipt was written.
		if err := seeder.root.Close(); err != nil {
			t.Fatal(err)
		}
		_ = command.Wait()
		assertProcessGoneWithESRCH(t, grandchild)

		reopened := openTestAdapter(t, config)
		receipt, err := reopened.Stop(context.Background(), stop)
		if err != nil || receipt.Status != ports.AgentStopped {
			t.Fatalf("Stop(after stopped terminal crash) = %+v, %v", receipt, err)
		}
	})

	t.Run("crash after signal effect fences replay without inventing proof", func(t *testing.T) {
		config, termLog := ignoreTermTreeTestConfig(t)
		request := testRequest(t, "crash-after-signal-effect", "ignore TERM across owner crash", 1024)
		command, _, launch := seedUnownedLiveProcess(t, config, request)
		grandchild := awaitGrandchildPID(t, config, request)
		cooperative := stopRequestForLaunch(launch, ports.AgentStopCooperative, "stop:crash-after-effect")
		requestHash, err := hashStopRequest(cooperative)
		if err != nil {
			t.Fatal(err)
		}

		seeder, err := New(config)
		if err != nil {
			t.Fatal(err)
		}
		seeder.mu.Lock()
		state, runPath, err := seeder.controlTargetLocked(cooperative)
		if err == nil {
			_, err = seeder.ensureStopRequest(runPath, requestHash, cooperative)
		}
		var record processRecord
		if err == nil {
			record, _, err = seeder.ownProcessLocked(state)
		}
		if err == nil {
			var created bool
			_, created, err = seeder.prepareStopSignalIntent(runPath, requestHash, cooperative)
			if err == nil && !created {
				err = errors.New("signal intent was not created")
			}
		}
		if err == nil {
			err = signalProcessTree(record, cooperative.Mode)
		}
		// Simulate the adapter process dying after the syscall but before proof
		// or receipt fsync. The OS would release this flock on a real crash.
		seeder.releaseProcessOwnershipLocked(state)
		seeder.mu.Unlock()
		if err != nil {
			t.Fatal(err)
		}
		if err := seeder.root.Close(); err != nil {
			t.Fatal(err)
		}
		awaitFileSize(t, termLog, 1)

		reopened := openTestAdapter(t, config)
		replayCtx, cancelReplay := context.WithTimeout(context.Background(), 50*time.Millisecond)
		replay, err := reopened.Stop(replayCtx, cooperative)
		cancelReplay()
		if err != nil || replay.Status != ports.AgentStopPending {
			t.Fatalf("Stop(replay after uncertain effect) = %+v, %v", replay, err)
		}
		alternateCtx, cancelAlternate := context.WithTimeout(context.Background(), 50*time.Millisecond)
		alternate := stopRequestForLaunch(launch, ports.AgentStopCooperative, "stop:alternate-after-uncertain-effect")
		alternateReceipt, err := reopened.Stop(alternateCtx, alternate)
		cancelAlternate()
		if err != nil || alternateReceipt.Status != ports.AgentStopPending {
			t.Fatalf("Stop(alternate after uncertain effect) = %+v, %v", alternateReceipt, err)
		}
		if payload, err := os.ReadFile(termLog); err != nil || string(payload) != "x" {
			t.Fatalf("signal log after replay = %q, %v; want exactly one signal", payload, err)
		}

		forcedCtx, cancelForced := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelForced()
		forced := stopRequestForLaunch(launch, ports.AgentStopForced, "stop:forced-after-uncertain-effect")
		forcedReceipt, err := reopened.Stop(forcedCtx, forced)
		if err != nil || forcedReceipt.Status != ports.AgentStopped {
			t.Fatalf("Stop(forced after uncertain effect) = %+v, %v", forcedReceipt, err)
		}
		_ = command.Wait()
		assertProcessGoneWithESRCH(t, grandchild)
	})

	t.Run("latest uncertain intent does not erase an older durable proof", func(t *testing.T) {
		controlRoot := t.TempDir()
		readyPath := filepath.Join(controlRoot, "ready")
		releasePath := filepath.Join(controlRoot, "release")
		termLog := filepath.Join(controlRoot, "term.log")
		helperPath := filepath.Join(controlRoot, "natural-after-term.sh")
		helper := "#!/bin/sh\n" +
			"trap 'printf x >> \"$CODEX_TERM_LOG\"' TERM\n" +
			"printf ready > \"$CODEX_NATURAL_READY\"\n" +
			"while [ ! -f \"$CODEX_NATURAL_RELEASE\" ]; do /bin/sleep 0.01; done\n"
		if err := os.WriteFile(helperPath, []byte(helper), 0o700); err != nil {
			t.Fatal(err)
		}
		config := testConfig(t)
		config.Command = helperPath
		config.Timeout = 10 * time.Second
		config.Environment = map[string]string{
			"CODEX_TERM_LOG": termLog, "CODEX_NATURAL_READY": readyPath, "CODEX_NATURAL_RELEASE": releasePath,
		}
		request := testRequest(t, "proof-before-uncertain-intent", "natural exit after durable proof", 1024)
		command, _, launch := seedUnownedLiveProcess(t, config, request)
		awaitPath(t, readyPath)

		seeder, err := New(config)
		if err != nil {
			t.Fatal(err)
		}
		cooperative := stopRequestForLaunch(launch, ports.AgentStopCooperative, "stop:durable-cooperative-proof")
		cooperativeCtx, cancelCooperative := context.WithTimeout(context.Background(), 50*time.Millisecond)
		cooperativeReceipt, err := seeder.Stop(cooperativeCtx, cooperative)
		cancelCooperative()
		if err != nil || cooperativeReceipt.Status != ports.AgentStopPending {
			t.Fatalf("Stop(durable cooperative) = %+v, %v", cooperativeReceipt, err)
		}
		awaitFileSize(t, termLog, 1)

		forced := stopRequestForLaunch(launch, ports.AgentStopForced, "stop:uncertain-forced-intent")
		forcedHash, err := hashStopRequest(forced)
		if err != nil {
			t.Fatal(err)
		}
		seeder.mu.Lock()
		state, runPath, err := seeder.controlTargetLocked(forced)
		if err == nil {
			_, err = seeder.ensureStopRequest(runPath, forcedHash, forced)
		}
		if err == nil {
			var created bool
			_, created, err = seeder.prepareStopSignalIntent(runPath, forcedHash, forced)
			if err == nil && !created {
				err = errors.New("forced signal intent was not created")
			}
		}
		// Simulate crash before the newer forced syscall. The older cooperative
		// proof remains durable evidence even though the newer intent is the
		// at-most-once fence.
		seeder.releaseProcessOwnershipLocked(state)
		seeder.mu.Unlock()
		if err != nil {
			t.Fatal(err)
		}
		if err := seeder.root.Close(); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(releasePath, []byte("natural\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := command.Wait(); err != nil {
			t.Fatalf("natural exit = %v", err)
		}

		reopened := openTestAdapter(t, config)
		observation, err := reopened.Observe(context.Background(), request.ExecutionRef)
		if err != nil || observation.ErrorCode != CodeExecutionStopped {
			t.Fatalf("Observe(highest durable proof) = %+v, %v", observation, err)
		}
		winnerReceipt, err := reopened.Stop(context.Background(), cooperative)
		if err != nil || winnerReceipt.Status != ports.AgentStopped {
			t.Fatalf("Stop(cooperative winner) = %+v, %v", winnerReceipt, err)
		}
		loserReceipt, err := reopened.Stop(context.Background(), forced)
		if err != nil || loserReceipt.Status != ports.AgentStopAlreadyStopped {
			t.Fatalf("Stop(uncertain forced loser) = %+v, %v", loserReceipt, err)
		}
		if payload, err := os.ReadFile(termLog); err != nil || string(payload) != "x" {
			t.Fatalf("cooperative signal log = %q, %v; want one signal", payload, err)
		}
	})

	t.Run("leader gone with live child stays owned until forced", func(t *testing.T) {
		config := testConfig(t)
		config.Timeout = 10 * time.Second
		request := testRequest(t, "leader-gone-child-live", "helper:background-success", 1024)
		command, record, launch := seedUnownedLiveProcess(t, config, request)
		childPath := filepath.Join(config.WorkRoot, filepath.FromSlash(executionPath(request.ExecutionRef)), backgroundSuccessPIDFile)
		child := awaitPIDFile(t, childPath)
		// Do not reap the leader yet: the detached child inherited the test
		// helper's diagnostic descriptor, so exec.Cmd.Wait cannot return until
		// that child exits. A zombie leader is already identity-gone and must not
		// hide the still-live process-group member.
		awaitProcessIdentityGone(t, record)

		adapter := openTestAdapter(t, config)
		observation, err := adapter.Observe(context.Background(), request.ExecutionRef)
		if err != nil || observation.Status != ports.AgentRunning {
			t.Fatalf("Observe(live child) = %+v, %v", observation, err)
		}
		contender, err := New(config)
		if err != nil {
			t.Fatal(err)
		}
		if lock, err := contender.acquireOwnerLock(executionPath(request.ExecutionRef)); lock != nil || !errors.Is(err, errOwnerLockBusy) {
			t.Fatalf("owner lock after leader exit = %v, %v", lock, err)
		}
		_ = contender.root.Close()

		forcedCtx, cancelForced := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelForced()
		forced, err := adapter.Stop(forcedCtx, stopRequestForLaunch(launch, ports.AgentStopForced, "stop:child-forced"))
		if err != nil || forced.Status != ports.AgentStopped {
			t.Fatalf("Stop(live child forced) = %+v, %v", forced, err)
		}
		_ = command.Wait()
		assertProcessGoneWithESRCH(t, child)
	})

	t.Run("forced stop escalates a pending cooperative stop", func(t *testing.T) {
		config, termLog := ignoreTermTreeTestConfig(t)
		adapter := openTestAdapter(t, config)
		request := testRequest(t, "cooperative-then-forced", "ignore TERM until forced", 1024)
		if _, err := adapter.Launch(context.Background(), request); err != nil {
			t.Fatal(err)
		}
		grandchild := awaitGrandchildPID(t, config, request)
		launch := launchReceiptForRequest(t, adapter, request)

		started := time.Now()
		cooperative, err := adapter.Stop(context.Background(), stopRequestForLaunch(launch, ports.AgentStopCooperative, "stop:cooperative-pending"))
		if err != nil || cooperative.Status != ports.AgentStopPending {
			t.Fatalf("cooperative Stop() = %+v, %v", cooperative, err)
		}
		if elapsed := time.Since(started); elapsed > time.Second {
			t.Fatalf("cooperative Stop() blocked resident caller for %s", elapsed)
		}
		if err := syscall.Kill(grandchild, 0); err != nil {
			t.Fatalf("cooperative stop unexpectedly killed tree: %v", err)
		}
		awaitFileSize(t, termLog, 1)

		replay, err := adapter.Stop(context.Background(), stopRequestForLaunch(launch, ports.AgentStopCooperative, "stop:cooperative-pending"))
		if err != nil || replay.Status != ports.AgentStopPending {
			t.Fatalf("cooperative replay = %+v, %v", replay, err)
		}
		alternate, err := adapter.Stop(context.Background(), stopRequestForLaunch(launch, ports.AgentStopCooperative, "stop:cooperative-alternate"))
		if err != nil || alternate.Status != ports.AgentStopPending {
			t.Fatalf("alternate cooperative Stop() = %+v, %v", alternate, err)
		}
		if payload, err := os.ReadFile(termLog); err != nil || string(payload) != "x" {
			t.Fatalf("cooperative signal log = %q, %v; want one signal", payload, err)
		}

		forcedCtx, cancelForced := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelForced()
		forced, err := adapter.Stop(forcedCtx, stopRequestForLaunch(launch, ports.AgentStopForced, "stop:forced-escalation"))
		if err != nil || forced.Status != ports.AgentStopped {
			t.Fatalf("forced Stop() = %+v, %v", forced, err)
		}
		assertProcessGoneWithESRCH(t, grandchild)
		loser, err := adapter.Stop(context.Background(), stopRequestForLaunch(launch, ports.AgentStopCooperative, "stop:cooperative-pending"))
		if err != nil || loser.Status != ports.AgentStopAlreadyStopped {
			t.Fatalf("losing cooperative receipt = %+v, %v", loser, err)
		}
	})

	t.Run("cleanup failure outranks a successful stop signal", func(t *testing.T) {
		config := processTreeTestConfig(t)
		adapter := openTestAdapter(t, config)
		adapter.processCleanup = func(*exec.Cmd) error { return errors.New("test cleanup failed") }
		request := testRequest(t, "stop-cleanup-failure", "cleanup failure after forced stop", 1024)
		if _, err := adapter.Launch(context.Background(), request); err != nil {
			t.Fatal(err)
		}
		grandchild := awaitGrandchildPID(t, config, request)
		launch := launchReceiptForRequest(t, adapter, request)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		receipt, err := adapter.Stop(ctx, stopRequestForLaunch(launch, ports.AgentStopForced, "stop:cleanup-failure"))
		if err != nil || receipt.Status != ports.AgentStopAlreadyFailed {
			t.Fatalf("Stop(cleanup failure) = %+v, %v", receipt, err)
		}
		observation, err := adapter.Observe(context.Background(), request.ExecutionRef)
		if err != nil || observation.ErrorCode != CodeProcessCleanupFailed {
			t.Fatalf("Observe(cleanup failure) = %+v, %v", observation, err)
		}
		assertProcessGoneWithESRCH(t, grandchild)
	})

	t.Run("shutdown discovers and waits for process tree after restart", func(t *testing.T) {
		config := processTreeTestConfig(t)
		request := testRequest(t, "adopted-shutdown", "adopted shutdown process tree", 1024)
		command, record, _ := seedUnownedLiveProcess(t, config, request)
		grandchild := awaitGrandchildPID(t, config, request)
		adapter := openTestAdapter(t, config)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := adapter.Shutdown(ctx); err != nil {
			t.Fatalf("Shutdown(adopted) = %v", err)
		}
		_ = command.Wait()
		assertProcessGoneWithESRCH(t, grandchild)
		if identity, err := platformInspectProcess(record); err != nil || identity != processIdentityGone {
			t.Fatalf("shutdown process identity = %v, %v", identity, err)
		}
	})

	t.Run("shutdown continues after corrupt journal and stops later valid tree", func(t *testing.T) {
		config := processTreeTestConfig(t)
		request := testRequest(t, "shutdown-after-corrupt-journal", "valid tree after corrupt journal", 1024)
		command, record, _ := seedUnownedLiveProcess(t, config, request)
		grandchild := awaitGrandchildPID(t, config, request)
		corruptPath := filepath.Join(config.WorkRoot, "executions", "000-corrupt", processFileName)
		if err := os.MkdirAll(filepath.Dir(corruptPath), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(corruptPath, []byte("{}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		adapter, err := New(config)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := adapter.Shutdown(ctx); err == nil {
			t.Fatal("Shutdown(corrupt plus valid) unexpectedly succeeded")
		}
		_ = command.Wait()
		assertProcessGoneWithESRCH(t, grandchild)
		if identity, err := platformInspectProcess(record); err != nil || identity != processIdentityGone {
			t.Fatalf("valid process identity after best-effort shutdown = %v, %v", identity, err)
		}

		contender, err := New(config)
		if err != nil {
			t.Fatal(err)
		}
		lock, err := contender.acquireOwnerLock(executionPath(request.ExecutionRef))
		if err != nil {
			t.Fatalf("valid owner lock was not released = %v", err)
		}
		releaseOwnerLock(lock)
		_ = contender.root.Close()
	})

	t.Run("shutdown cannot report success after adopted signal failure", func(t *testing.T) {
		config := processTreeTestConfig(t)
		request := testRequest(t, "adopted-shutdown-signal-failure", "adopted shutdown signal failure", 1024)
		command, record, _ := seedUnownedLiveProcess(t, config, request)
		grandchild := awaitGrandchildPID(t, config, request)
		adapter, err := New(config)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := adapter.Observe(context.Background(), request.ExecutionRef); err != nil {
			t.Fatal(err)
		}
		adapter.mu.Lock()
		adapter.executions[request.ExecutionRef.String()].process.BirthMarker += "-stale"
		adapter.mu.Unlock()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := adapter.Shutdown(ctx); ErrorCode(err) != CodeProcessIdentityMismatch {
			t.Fatalf("Shutdown(signal failure) = %v, code = %q", err, ErrorCode(err))
		}
		if err := syscall.Kill(grandchild, 0); err != nil {
			t.Fatalf("failed shutdown unexpectedly stopped target: %v", err)
		}
		if err := platformSignalProcess(record, ports.AgentStopForced); err != nil {
			t.Fatalf("manual cleanup signal = %v", err)
		}
		_ = command.Wait()
		assertProcessGoneWithESRCH(t, grandchild)
		adapter.mu.Lock()
		adapter.releaseProcessOwnershipLocked(adapter.executions[request.ExecutionRef.String()])
		adapter.mu.Unlock()
	})

	t.Run("stop journal read errors surface", func(t *testing.T) {
		adapter, err := New(testConfig(t))
		if err != nil {
			t.Fatal(err)
		}
		if err := adapter.root.Close(); err != nil {
			t.Fatal(err)
		}
		if _, err := adapter.hasDurableStopRequest("executions/unreadable"); ErrorCode(err) != CodeStateInvalid {
			t.Fatalf("hasDurableStopRequest() = %v, code = %q", err, ErrorCode(err))
		}
	})

	t.Run("rejects reused pid marker", func(t *testing.T) {
		config := processTreeTestConfig(t)
		request := testRequest(t, "reused-pid-control", "reused pid process tree", 1024)
		command, record, launch := seedUnownedLiveProcess(t, config, request)
		grandchild := awaitGrandchildPID(t, config, request)
		tampered := record
		tampered.BirthMarker += "-stale"
		processPath := filepath.Join(config.WorkRoot, filepath.FromSlash(executionPath(request.ExecutionRef)), processFileName)
		payload, _ := json.Marshal(tampered)
		if err := os.WriteFile(processPath, append(payload, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}

		adapter := openTestAdapter(t, config)
		_, err := adapter.Stop(context.Background(), stopRequestForLaunch(launch, ports.AgentStopForced, "stop:stale"))
		if ErrorCode(err) != CodeProcessIdentityMismatch {
			t.Fatalf("stale PID Stop() error = %v, code = %q", err, ErrorCode(err))
		}
		if err := syscall.Kill(grandchild, 0); err != nil {
			t.Fatalf("stale marker signaled target: %v", err)
		}
		_ = platformSignalProcess(record, ports.AgentStopForced)
		_ = command.Wait()
		assertProcessGoneWithESRCH(t, grandchild)
	})
}

func awaitPath(t *testing.T, filePath string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(filePath); err == nil {
			return
		} else if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("Stat(%s) error = %v", filePath, err)
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("path was not published: %s", filePath)
}

func awaitFileSize(t *testing.T, filePath string, minimum int64) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if info, err := os.Stat(filePath); err == nil && info.Size() >= minimum {
			return
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("Stat(%s) error = %v", filePath, err)
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("file did not reach %d bytes: %s", minimum, filePath)
}

func awaitPIDFile(t *testing.T, filePath string) int {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		payload, err := os.ReadFile(filePath)
		if err == nil {
			pid, parseErr := strconv.Atoi(strings.TrimSpace(string(payload)))
			if parseErr != nil || pid <= 0 {
				t.Fatalf("pid file %q is invalid: %v", payload, parseErr)
			}
			return pid
		}
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("ReadFile(%s) error = %v", filePath, err)
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("pid was not published: %s", filePath)
	return 0
}

func awaitProcessIdentityGone(t *testing.T, record processRecord) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		identity, err := platformInspectProcess(record)
		if err != nil {
			t.Fatalf("platformInspectProcess() error = %v", err)
		}
		if identity == processIdentityGone {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("process leader did not exit: pid=%d", record.PID)
}

func ignoreTermTreeTestConfig(t *testing.T) (Config, string) {
	t.Helper()
	controlRoot := t.TempDir()
	helperPath := filepath.Join(controlRoot, "ignore-term-tree.sh")
	termLog := filepath.Join(controlRoot, "term.log")
	helper := "#!/bin/sh\n" +
		"trap 'printf x >> \"$CODEX_TERM_LOG\"' TERM\n" +
		"( trap '' TERM; exec /bin/sleep 30 ) &\n" +
		"grandchild=$!\n" +
		"printf '%s\\n' \"$grandchild\" > " + processTreePIDFile + "\n" +
		"while kill -0 \"$grandchild\" 2>/dev/null; do wait \"$grandchild\"; done\n"
	if err := os.WriteFile(helperPath, []byte(helper), 0o700); err != nil {
		t.Fatal(err)
	}
	config := testConfig(t)
	config.Command = helperPath
	config.Timeout = 10 * time.Second
	config.ProcessPipeDrainDelay = 10 * time.Millisecond
	config.Environment = map[string]string{"CODEX_TERM_LOG": termLog}
	return config, termLog
}

func TestCodexLaunchGateCrashNeverOrphansProcess(t *testing.T) {
	for _, afterIdentity := range []bool{false, true} {
		name := "before identity publish"
		if afterIdentity {
			name = "after identity publish"
		}
		t.Run(name, func(t *testing.T) {
			marker := filepath.Join(t.TempDir(), "target-started")
			commandPath := filepath.Join(t.TempDir(), "target.sh")
			if err := os.WriteFile(commandPath, []byte("#!/bin/sh\nprintf started > \"$CODEX_GATE_MARKER\"\n"), 0o700); err != nil {
				t.Fatal(err)
			}
			config := testConfig(t)
			config.Command = commandPath
			config.Environment = map[string]string{"CODEX_GATE_MARKER": marker}
			adapter, err := New(config)
			if err != nil {
				t.Fatal(err)
			}
			runPath := "executions/gate-crash"
			if err := adapter.ensurePrivateDirectory(runPath); err != nil {
				t.Fatal(err)
			}
			command, reader, writer, err := adapter.executionCommand(context.Background(), runPath)
			if err != nil {
				t.Fatal(err)
			}
			command.Env = []string{"CODEX_GATE_MARKER=" + marker}
			command.Stdout, command.Stderr = io.Discard, io.Discard
			configureProcessGroup(command, nil)
			if err := command.Start(); err != nil {
				t.Fatal(err)
			}
			_ = reader.Close()
			if afterIdentity {
				pgid, bootID, birth, err := platformCaptureProcess(command.Process.Pid)
				if err != nil {
					t.Fatal(err)
				}
				record := processRecord{
					SchemaVersion: processSchemaVersion, ExecutionRef: "execution:gate-crash",
					RequestHash: "sha256:gate-crash", RuntimeScope: config.RuntimeScope,
					PID: command.Process.Pid, PGID: pgid, BootID: bootID, BirthMarker: birth,
				}
				if err := adapter.persistProcessRecord(runPath, record); err != nil {
					t.Fatal(err)
				}
			}
			// Simulated crash: writer disappears without release token.
			_ = writer.Close()
			if err := command.Wait(); err != nil {
				t.Fatalf("gate wrapper exit = %v", err)
			}
			if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("target crossed unreleased gate: %v", err)
			}
			if err := adapter.root.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}

	t.Run("exec closes inherited gate fd", func(t *testing.T) {
		marker := filepath.Join(t.TempDir(), "fd3")
		commandPath := filepath.Join(t.TempDir(), "fd3.sh")
		script := "#!/bin/sh\nif [ -e /proc/self/fd/3 ]; then printf open; else printf closed; fi > \"$CODEX_GATE_MARKER\"\n"
		if err := os.WriteFile(commandPath, []byte(script), 0o700); err != nil {
			t.Fatal(err)
		}
		config := testConfig(t)
		config.Command = commandPath
		adapter := openTestAdapter(t, config)
		command, reader, writer, err := adapter.executionCommand(context.Background(), "unused")
		if err != nil {
			t.Fatal(err)
		}
		command.Env = []string{"CODEX_GATE_MARKER=" + marker}
		command.Stdout, command.Stderr = io.Discard, io.Discard
		configureProcessGroup(command, nil)
		if err := command.Start(); err != nil {
			t.Fatal(err)
		}
		_ = reader.Close()
		if _, err := writer.Write([]byte("run\n")); err != nil {
			t.Fatal(err)
		}
		_ = writer.Close()
		if err := command.Wait(); err != nil {
			t.Fatal(err)
		}
		if payload, err := os.ReadFile(marker); err != nil || string(payload) != "closed" {
			t.Fatalf("gate fd marker = %q, %v", payload, err)
		}
	})
}

func TestCodexOwnerLockIsExclusiveAndCLOEXEC(t *testing.T) {
	adapter := openTestAdapter(t, testConfig(t))
	runPath := "executions/owner-lock-test"
	if err := adapter.ensurePrivateDirectory(runPath); err != nil {
		t.Fatal(err)
	}
	first, err := adapter.acquireOwnerLock(runPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { releaseOwnerLock(first) })
	if !platformOwnerLockCLOEXEC(first) {
		t.Fatal("owner lock lacks FD_CLOEXEC")
	}
	if second, err := adapter.acquireOwnerLock(runPath); second != nil || !errors.Is(err, errOwnerLockBusy) {
		t.Fatalf("second owner lock = %v, %v", second, err)
	}
	releaseOwnerLock(first)
	first = nil
	third, err := adapter.acquireOwnerLock(runPath)
	if err != nil {
		t.Fatalf("lock after release = %v", err)
	}
	releaseOwnerLock(third)
}

func TestCodexRestoredDatabaseCannotAdoptSourceProcess(t *testing.T) {
	config := processTreeTestConfig(t)
	config.RuntimeScope = "runtime-scope:source-db-device-inode"
	request := testRequest(t, "restored-db-control", "source process tree", 1024)
	command, record, launch := seedUnownedLiveProcess(t, config, request)
	grandchild := awaitGrandchildPID(t, config, request)

	restored := config
	restored.RuntimeScope = "runtime-scope:restored-db-device-inode"
	adapter, err := New(restored)
	if err != nil {
		t.Fatal(err)
	}
	_, err = adapter.Stop(context.Background(), stopRequestForLaunch(launch, ports.AgentStopForced, "stop:restored"))
	if ErrorCode(err) != CodeProcessOwnershipInvalid {
		t.Fatalf("restored scope Stop() error = %v, code = %q", err, ErrorCode(err))
	}
	if err := syscall.Kill(grandchild, 0); err != nil {
		t.Fatalf("restored scope touched source process: %v", err)
	}
	_ = platformSignalProcess(record, ports.AgentStopForced)
	_ = command.Wait()
	assertProcessGoneWithESRCH(t, grandchild)
	if err := adapter.root.Close(); err != nil {
		t.Fatal(err)
	}
}

func seedUnownedLiveProcess(t *testing.T, config Config, request ports.AgentLaunchRequest, privateEnvironment ...string) (*exec.Cmd, processRecord, ports.AgentLaunchReceipt) {
	t.Helper()
	adapter, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	requestHash := mustRequestHash(t, request)
	launchRecord, runPath, created, err := adapter.ensureLaunchRecord(request, requestHash)
	if err != nil || !created {
		t.Fatalf("ensure launch created=%v err=%v", created, err)
	}
	receipt, err := launchRecord.receipt(request.ExecutionRef)
	if err != nil {
		t.Fatal(err)
	}
	if err := adapter.prepareRuntimeFiles(runPath); err != nil {
		t.Fatal(err)
	}
	command, reader, writer, err := adapter.executionCommand(context.Background(), runPath)
	if err != nil {
		t.Fatal(err)
	}
	command.Dir = filepath.Join(config.WorkRoot, filepath.FromSlash(runPath))
	command.Env = append(append([]string(nil), adapter.environment...), privateEnvironment...)
	prompt, err := adapter.renderAgentPrompt(request)
	if err != nil {
		t.Fatal(err)
	}
	command.Stdin, command.Stdout, command.Stderr = strings.NewReader(prompt), io.Discard, io.Discard
	configureProcessGroup(command, nil)
	owner, err := adapter.acquireOwnerLock(runPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	_ = reader.Close()
	pgid, bootID, birthMarker, err := platformCaptureProcess(command.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	record := processRecord{
		SchemaVersion: processSchemaVersion, ExecutionRef: request.ExecutionRef.String(),
		RequestHash: requestHash, RuntimeScope: config.RuntimeScope,
		PID: command.Process.Pid, PGID: pgid, BootID: bootID, BirthMarker: birthMarker,
	}
	if err := adapter.persistProcessRecord(runPath, record); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("run\n")); err != nil {
		t.Fatal(err)
	}
	_ = writer.Close()
	releaseOwnerLock(owner)
	if err := adapter.root.Close(); err != nil {
		t.Fatal(err)
	}
	return command, record, receipt
}

func launchReceiptForRequest(t *testing.T, adapter *Adapter, request ports.AgentLaunchRequest) ports.AgentLaunchReceipt {
	t.Helper()
	record, _, found, err := adapter.loadLaunchRecord(request.ExecutionRef)
	if err != nil || !found {
		t.Fatalf("load launch found=%v err=%v", found, err)
	}
	receipt, err := record.receipt(request.ExecutionRef)
	if err != nil {
		t.Fatal(err)
	}
	return receipt
}

func stopRequestForLaunch(launch ports.AgentLaunchReceipt, mode ports.AgentStopMode, idempotency string) ports.AgentStopRequest {
	return ports.AgentStopRequest{
		ExecutionRef: launch.ExecutionRef, GoalRef: launch.GoalRef, WorkItemRef: launch.WorkItemRef,
		PlanGeneration: launch.PlanGeneration, AppSpecGeneration: launch.AppSpecGeneration,
		ExecutionAttempt: launch.ExecutionAttempt, SpecHash: launch.SpecHash,
		ProviderRef: launch.ProviderRef, ModelRef: launch.ModelRef, AgentRef: launch.AgentRef,
		ExternalRef: launch.ExternalRef, Mode: mode, IdempotencyKey: idempotency,
	}
}
