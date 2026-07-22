//go:build linux

package bubblewrap

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"orquesta/internal/ports"
)

func TestCgroupEventParserRejectsAmbiguousControls(t *testing.T) {
	for name, content := range map[string]string{
		"empty":            "",
		"odd_tokens":       "oom 0 oom_kill",
		"invalid_name":     "OOM 0 oom_kill 0",
		"invalid_value":    "oom nope oom_kill 0",
		"negative":         "oom -1 oom_kill 0",
		"overflow":         "oom 18446744073709551616 oom_kill 0",
		"duplicate":        "oom 0 oom 1 oom_kill 0",
		"missing_required": "oom 0 max 0",
	} {
		t.Run(name, func(t *testing.T) {
			positive, err := eventPositive(content, "oom", "oom_kill")
			if positive || ErrorCode(err) != CodeResourceUnsafe {
				t.Fatalf("positive=%t code=%q err=%v", positive, ErrorCode(err), err)
			}
		})
	}
	for name, check := range map[string]struct {
		content  string
		required string
	}{
		"pids_missing_max":         {content: "local_max 0", required: "max"},
		"cgroup_missing_populated": {content: "frozen 0", required: "populated"},
	} {
		t.Run(name, func(t *testing.T) {
			positive, err := eventPositive(check.content, check.required)
			if positive || ErrorCode(err) != CodeResourceUnsafe {
				t.Fatalf("positive=%t code=%q err=%v", positive, ErrorCode(err), err)
			}
		})
	}
}

func TestReadControlFailsClosedOnOpenReadAndSize(t *testing.T) {
	root := t.TempDir()
	dir, err := unix.Open(root, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(dir)
	if err := os.Mkdir(filepath.Join(root, "read-error"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "too-large"), bytes.Repeat([]byte{'x'}, 4097), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"missing", "read-error", "too-large"} {
		t.Run(name, func(t *testing.T) {
			content, err := readControl(dir, name)
			if content != "" || ErrorCode(err) != CodeResourceUnsafe {
				t.Fatalf("content=%q code=%q err=%v", content, ErrorCode(err), err)
			}
		})
	}
}

func TestCgroupLeafExceededPropagatesControlFailures(t *testing.T) {
	for name, files := range map[string]map[string]string{
		"memory_parse":   {"memory.events": "oom nope\noom_kill 0", "pids.events": "max 0"},
		"memory_missing": {"memory.events": "oom 0", "pids.events": "max 0"},
		"pids_read":      {"memory.events": "oom 0\noom_kill 0"},
		"pids_parse":     {"memory.events": "oom 0\noom_kill 0", "pids.events": "max 0\nmax 1"},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			for path, content := range files {
				if err := os.WriteFile(filepath.Join(root, path), []byte(content), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			dir, err := unix.Open(root, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
			if err != nil {
				t.Fatal(err)
			}
			defer unix.Close(dir)
			exceeded, err := (&cgroupLeaf{dir: dir}).exceeded()
			if exceeded || ErrorCode(err) != CodeResourceUnsafe {
				t.Fatalf("exceeded=%t code=%q err=%v", exceeded, ErrorCode(err), err)
			}
		})
	}
}

func TestConfiguredResourceEnvelopeFailsClosedBeforeWork(t *testing.T) {
	limits := Limits{
		Timeout: 30 * time.Second, MaxSubjectBytes: 512 << 20, MaxConcurrentRuns: 2,
		MemoryMaxBytes: 2 << 30,
	}
	valid := inheritedResourceEnvelope{
		noFile: unix.Rlimit{Cur: 8192, Max: 8192}, fileSize: unix.Rlimit{Cur: 512 << 20, Max: 512 << 20},
		addressSpace: unix.Rlimit{Cur: 4 << 30, Max: 4 << 30}, cpu: unix.Rlimit{Cur: 30, Max: 30},
	}
	if !resourceEnvelopeAllowed(limits, valid) {
		t.Fatal("finite configured resource envelope rejected")
	}
	for name, mutate := range map[string]func(*inheritedResourceEnvelope){
		"infinite nofile": func(value *inheritedResourceEnvelope) { value.noFile.Max = unix.RLIM_INFINITY },
		"oversized fsize": func(value *inheritedResourceEnvelope) { value.fileSize.Max++ },
		"oversized as":    func(value *inheritedResourceEnvelope) { value.addressSpace.Max++ },
		"oversized cpu":   func(value *inheritedResourceEnvelope) { value.cpu.Max++ },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := valid
			mutate(&candidate)
			if resourceEnvelopeAllowed(limits, candidate) {
				t.Fatal("unsafe inherited resource envelope accepted")
			}
		})
	}
}

func TestRunTestFailsClosedOnCgroupObservationAndCleanupErrors(t *testing.T) {
	policy := PolicyIdentity{PolicyRef, testDigest("policy")}
	run := testRun(t, policy)
	snapshot, err := readSnapshotStream(bytes.NewReader(
		testStream(t, run.Request, "internal/main.go", []byte("package main\n")),
	), run.Request, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	defer snapshot.Close()
	for _, test := range []struct {
		name       string
		observeErr error
		closeErr   error
		want       string
	}{
		{name: "observation", observeErr: errors.New("raw cgroup observation failure"), want: CodeResourceUnsafe},
		{name: "cleanup", closeErr: resourceError(), want: CodeCleanupFailed},
	} {
		t.Run(test.name, func(t *testing.T) {
			adapter := testAdapter(t, &testSnapshotSource{}, policy)
			adapter.inputs = executableTestInputs(t)
			controller := &fakeCgroupController{session: &fakeCgroupSession{
				observationErr: test.observeErr, closeErr: test.closeErr,
			}}
			adapter.cgroups = controller
			outcome, err := adapter.runTest(context.Background(), snapshot, run.Request.RequiredTests[0])
			if outcome != (ports.RequiredTestOutcome{}) || ErrorCode(err) != test.want ||
				controller.active.Load() != 0 || controller.session.closes.Load() != 1 {
				t.Fatalf("outcome=%+v code=%q active=%d closes=%d err=%v", outcome,
					ErrorCode(err), controller.active.Load(), controller.session.closes.Load(), err)
			}
		})
	}
}

func TestRunTestNeverAcceptsOverflowWhenWaitAndLimitRace(t *testing.T) {
	done := make(chan error, 1)
	done <- nil
	overflow := make(chan struct{})
	close(overflow)
	waitErr, failure := waitSandboxCompletion(context.Background(), done, overflow)
	if waitErr != nil || ErrorCode(failure) != CodeOutputLimit {
		t.Fatalf("simultaneously ready Wait/overflow accepted: wait=%v code=%q err=%v",
			waitErr, ErrorCode(failure), failure)
	}
}

func TestStopSandboxKillsWholeCgroupAndReturnsBoundedly(t *testing.T) {
	done := make(chan error, 1)
	done <- nil
	session := &fakeCgroupSession{}
	if err := stopSandbox(nil, session, done, time.Second); err != nil || session.kills.Load() != 1 {
		t.Fatalf("stop sandbox kills=%d err=%v", session.kills.Load(), err)
	}
}

type fakeCgroupController struct {
	session *fakeCgroupSession
	active  atomic.Int64
}

func (controller *fakeCgroupController) newLeaf(Limits) (cgroupSession, error) {
	controller.active.Add(1)
	controller.session.controller = controller
	return controller.session, nil
}
func (*fakeCgroupController) Close() error { return nil }

type fakeCgroupSession struct {
	controller     *fakeCgroupController
	observationErr error
	closeErr       error
	killErr        error
	kills          atomic.Int64
	closes         atomic.Int64
}

func (*fakeCgroupSession) apply(*exec.Cmd) error { return nil }
func (session *fakeCgroupSession) kill() error {
	session.kills.Add(1)
	return session.killErr
}
func (session *fakeCgroupSession) exceeded() (bool, error) {
	return false, session.observationErr
}
func (session *fakeCgroupSession) close(context.Context) error {
	session.closes.Add(1)
	session.controller.active.Add(-1)
	return session.closeErr
}

func executableTestInputs(t *testing.T) *pinnedInputs {
	t.Helper()
	return testPinnedFiles(t)
}
