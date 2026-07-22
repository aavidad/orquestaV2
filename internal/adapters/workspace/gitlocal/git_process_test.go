package gitlocal

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestPinnedGitCancellationKillsDescendantAndReturnsBoundedly(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "descendant.pid")
	script := "#!/bin/sh\nfor last\ndo :\ndone\nsleep 30 &\necho $! > \"$last\"\nwait\n"
	adapter := scriptTestAdapter(t, script)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, err := adapter.gitRun(ctx, "", nil, pidFile)
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(started) > 2*time.Second {
		t.Fatalf("cancel err=%v duration=%s", err, time.Since(started))
	}
	waitForProcessGone(t, readProcessID(t, pidFile))
}

func TestPinnedGitWaitDelayKillsPipeHoldingDescendantAfterLeaderExit(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "descendant.pid")
	script := "#!/bin/sh\nfor last\ndo :\ndone\nsleep 30 &\necho $! > \"$last\"\nexit 0\n"
	adapter := scriptTestAdapter(t, script)
	started := time.Now()
	_, err := adapter.gitRun(context.Background(), "", nil, pidFile)
	if ErrorCodeOf(err) != CodeGitFailed || time.Since(started) > 2*time.Second {
		t.Fatalf("wait-delay err=%v duration=%s", err, time.Since(started))
	}
	pid := readProcessID(t, pidFile)
	waitForProcessGone(t, pid)
}

func TestPinnedGitSuccessfulLeaderExitKillsDetachedPipeDescendant(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "descendant.pid")
	script := "#!/bin/sh\nfor last\ndo :\ndone\n(sleep 30) >/dev/null 2>&1 &\necho $! > \"$last\"\nexit 0\n"
	adapter := scriptTestAdapter(t, script)
	if _, err := adapter.gitRun(context.Background(), "", nil, pidFile); err != nil {
		t.Fatalf("successful leader err=%v", err)
	}
	waitForProcessGone(t, readProcessID(t, pidFile))
}

func TestPinnedGitOutputAndInputLimitsKillWholeGroup(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "descendant.pid")
	script := "#!/bin/sh\nfor last\ndo :\ndone\nsleep 30 &\necho $! > \"$last\"\nwhile :\ndo echo 012345678901234567890123456789\ndone\n"
	adapter := scriptTestAdapter(t, script)
	adapter.maxSnapshotBytes = maxSnapshotStreamFieldBytes
	if _, err := adapter.gitRun(context.Background(), "", nil, pidFile); ErrorCodeOf(err) != CodeSnapshotLimit {
		t.Fatalf("output limit err=%v", err)
	}
	waitForProcessGone(t, readProcessID(t, pidFile))

	marker := filepath.Join(t.TempDir(), "started")
	inputAdapter := scriptTestAdapter(t, "#!/bin/sh\ntouch \"$1\"\n")
	inputAdapter.maxSnapshotBytes = maxSnapshotStreamFieldBytes
	if _, err := inputAdapter.gitRun(context.Background(), "", make([]byte, maxSnapshotStreamFieldBytes+1), marker); ErrorCodeOf(err) != CodeSnapshotLimit {
		t.Fatalf("input limit err=%v", err)
	}
	if _, err := os.Lstat(marker); !os.IsNotExist(err) {
		t.Fatalf("oversized input started child: %v", err)
	}

	commitMarker := filepath.Join(t.TempDir(), "commit-started")
	commitAdapter := scriptTestAdapter(t, "#!/bin/sh\ntouch \""+commitMarker+"\"\n")
	commitAdapter.maxSnapshotBytes = maxSnapshotStreamFieldBytes
	if _, err := commitAdapter.gitCommitTree(context.Background(), "/", "tree", "", "", time.Now(),
		make([]byte, maxSnapshotStreamFieldBytes+1)); ErrorCodeOf(err) != CodeSnapshotLimit {
		t.Fatalf("commit-tree input limit err=%v", err)
	}
	if _, err := os.Lstat(commitMarker); !os.IsNotExist(err) {
		t.Fatalf("oversized commit-tree input started child: %v", err)
	}
}

func TestPinnedGitChildEnvironmentAndDescriptorsAreClosed(t *testing.T) {
	t.Setenv("LD_PRELOAD", "/tmp/hostile.so")
	t.Setenv("GIT_EXEC_PATH", "/tmp/hostile-git-core")
	t.Setenv("ORQUESTA_SECRET_SENTINEL", "must-not-leak")
	script := "#!/bin/sh\nenv\nfor fd in /proc/self/fd/[4-9]*\ndo [ -e \"$fd\" ] && echo unexpected-fd \"$fd\"\ndone\ntrue\n"
	adapter := scriptTestAdapter(t, script)
	output, err := adapter.gitRun(context.Background(), "", nil, "probe")
	gitTestNoError(t, err)
	text := string(output)
	for _, forbidden := range []string{"LD_PRELOAD=", "GIT_EXEC_PATH=", "ORQUESTA_SECRET_SENTINEL=", "unexpected-fd"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("child inherited %q in %q", forbidden, text)
		}
	}
}

func scriptTestAdapter(t *testing.T, content string) *Adapter {
	t.Helper()
	path := filepath.Join(t.TempDir(), "git-script")
	gitTestNoError(t, os.WriteFile(path, []byte(content), 0o500))
	adapter := pinnedTestAdapter(t, path)
	return adapter
}

func waitForProcessGone(t *testing.T, pid int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		err := unix.Kill(pid, 0)
		if errors.Is(err, unix.ESRCH) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("descendant pid=%d remained alive: %v", pid, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func readProcessID(t *testing.T, path string) int {
	t.Helper()
	raw, err := os.ReadFile(path)
	gitTestNoError(t, err)
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	gitTestNoError(t, err)
	return pid
}
