package gitlocal

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"orquesta/internal/ports"
)

func TestPinnedGitRejectsSymlinkSpecialWritableForeignAndCaptureRaces(t *testing.T) {
	owner := uint32(os.Geteuid())
	cases := map[string]func(*testing.T) (string, gitPinPolicy){
		"symlink": func(t *testing.T) (string, gitPinPolicy) {
			link := filepath.Join(t.TempDir(), "git-link")
			gitTestNoError(t, os.Symlink(testGitExecutable(t), link))
			return link, gitPinPolicy{ownerUID: owner}
		},
		"special": func(t *testing.T) (string, gitPinPolicy) {
			path := filepath.Join(t.TempDir(), "git-fifo")
			gitTestNoError(t, unix.Mkfifo(path, 0o500))
			return path, gitPinPolicy{ownerUID: owner}
		},
		"writable": func(t *testing.T) (string, gitPinPolicy) {
			return chmodPinnedTestGit(t, 0o520), gitPinPolicy{ownerUID: owner}
		},
		"not_executable": func(t *testing.T) (string, gitPinPolicy) {
			return chmodPinnedTestGit(t, 0o400), gitPinPolicy{ownerUID: owner}
		},
		"foreign_owner": func(t *testing.T) (string, gitPinPolicy) {
			return testGitExecutable(t), gitPinPolicy{ownerUID: owner + 1}
		},
		"rewrite_during_capture": func(t *testing.T) (string, gitPinPolicy) {
			path := testGitExecutable(t)
			return path, gitPinPolicy{ownerUID: owner, afterHash: func() {
				err := os.Chmod(path, 0o700)
				if err == nil {
					err = os.WriteFile(path, []byte("rewritten"), 0o500)
				}
				if err != nil {
					t.Error(err)
				}
			}}
		},
		"swap_during_capture": func(t *testing.T) (string, gitPinPolicy) {
			path := testGitExecutable(t)
			return path, gitPinPolicy{ownerUID: owner, afterHash: func() {
				err := os.Rename(path, path+".old")
				if err == nil {
					err = os.WriteFile(path, []byte("#!/bin/sh\nexit 73\n"), 0o500)
				}
				if err != nil {
					t.Error(err)
				}
			}}
		},
	}
	for name, prepare := range cases {
		t.Run(name, func(t *testing.T) {
			path, policy := prepare(t)
			if file, _, err := pinGitExecutable(path, policy); err == nil {
				_ = file.Close()
				t.Fatal("unsafe executable accepted")
			} else if ErrorCodeOf(err) != CodeConfigInvalid {
				t.Fatalf("error code=%q err=%v", ErrorCodeOf(err), err)
			}
		})
	}
}

func chmodPinnedTestGit(t *testing.T, mode os.FileMode) string {
	t.Helper()
	path := testGitExecutable(t)
	gitTestNoError(t, os.Chmod(path, mode))
	return path
}

func TestPinnedGitExecutesDescriptorAfterPathSwapAndRemoval(t *testing.T) {
	for _, mutation := range []string{"swap", "remove"} {
		t.Run(mutation, func(t *testing.T) {
			path := testGitExecutable(t)
			adapter := pinnedTestAdapter(t, path)
			old := path + ".pinned"
			gitTestNoError(t, os.Rename(path, old))
			if mutation == "swap" {
				gitTestNoError(t, os.WriteFile(path, []byte("#!/bin/sh\nexit 79\n"), 0o500))
			} else {
				gitTestNoError(t, os.Remove(old))
			}
			output, err := adapter.gitRun(context.Background(), "", nil, "--version")
			if err != nil || !strings.HasPrefix(string(output), "git version ") {
				t.Fatalf("pinned command output=%q err=%v", output, err)
			}
		})
	}
}

func TestPinnedGitSealedMemfdSurvivesInPlaceRewriteAfterValidation(t *testing.T) {
	path := testGitExecutable(t)
	adapter := pinnedTestAdapter(t, path)
	beforeRef, beforeDigest := adapter.SnapshotIdentity()
	gitTestNoError(t, os.Chmod(path, 0o700))
	gitTestNoError(t, os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o500))
	output, err := adapter.gitRun(context.Background(), "", nil, "--version")
	if err != nil || !strings.HasPrefix(string(output), "git version ") {
		t.Fatalf("sealed executable output=%q err=%v", output, err)
	}
	if ref, digest := adapter.SnapshotIdentity(); ref != beforeRef || digest != beforeDigest {
		t.Fatalf("source rewrite changed sealed identity: (%q,%q)", ref, digest)
	}
}

func TestPinnedGitMemfdHasMandatoryExecutableSeals(t *testing.T) {
	adapter := pinnedTestAdapter(t, testGitExecutable(t))
	seals, err := unix.FcntlInt(adapter.gitFile.Fd(), unix.F_GET_SEALS, 0)
	if err != nil || seals&pinnedGitSeals != pinnedGitSeals {
		t.Fatalf("seals=%#x want=%#x err=%v", seals, pinnedGitSeals, err)
	}
	if _, err := unix.Write(int(adapter.gitFile.Fd()), []byte("tamper")); !errors.Is(err, unix.EPERM) {
		t.Fatalf("sealed memfd write err=%v", err)
	}
	stat, err := adapter.gitFile.Stat()
	gitTestNoError(t, err)
	if err := unix.Ftruncate(int(adapter.gitFile.Fd()), stat.Size()+1); !errors.Is(err, unix.EPERM) {
		t.Fatalf("sealed memfd grow err=%v", err)
	}
	if err := unix.Fchmod(int(adapter.gitFile.Fd()), 0o600); !errors.Is(err, unix.EPERM) {
		t.Fatalf("sealed memfd exec chmod err=%v", err)
	}
}

func TestPinnedGitRejectsNoexecMountAndMemfdFailureWithoutPathFallback(t *testing.T) {
	path := testGitExecutable(t)
	owner := uint32(os.Geteuid())
	if _, _, err := pinGitExecutable(path, gitPinPolicy{
		ownerUID:   owner,
		mountFlags: func(int) (int64, error) { return unix.MS_NOEXEC, nil },
	}); ErrorCodeOf(err) != CodeConfigInvalid {
		t.Fatalf("noexec source err=%v", err)
	}
	if _, _, err := pinGitExecutable(path, gitPinPolicy{
		ownerUID:    owner,
		memfdCreate: func(string, int) (int, error) { return -1, unix.ENOSYS },
	}); ErrorCodeOf(err) != CodeConfigInvalid {
		t.Fatalf("memfd unavailable err=%v", err)
	}
}

func TestProductionPinAcceptsRootTrustedSystemGit(t *testing.T) {
	root := t.TempDir()
	gitTestNoError(t, os.Chmod(root, 0o700))
	adapter, err := New(Config{
		Root: root, GitCommand: "/usr/bin/git", Locator: testLocator{}, Now: time.Now,
	})
	gitTestNoError(t, err)
	output, err := adapter.gitRun(context.Background(), "", nil, "--version")
	if err != nil || !strings.HasPrefix(string(output), "git version ") {
		t.Fatalf("production sealed Git output=%q err=%v", output, err)
	}
	gitTestNoError(t, adapter.Close())
}

func TestSnapshotIdentityIsPathIndependentAndContractBound(t *testing.T) {
	first := pinnedTestAdapter(t, testGitExecutable(t))
	second := pinnedTestAdapter(t, testGitExecutable(t))
	firstRef, firstDigest := first.SnapshotIdentity()
	secondRef, secondDigest := second.SnapshotIdentity()
	if firstRef != snapshotSourceRef || firstRef != secondRef || firstDigest == "" || firstDigest != secondDigest {
		t.Fatalf("identity first=(%q,%q) second=(%q,%q)", firstRef, firstDigest, secondRef, secondDigest)
	}
	if !strings.HasSuffix(firstRef, ":v2") || !strings.Contains(snapshotSourceContract, "/v2\n") {
		t.Fatalf("snapshot identity contract is not v2: ref=%q", firstRef)
	}
	_, originalContract := snapshotSourceIdentity(snapshotSourceContract, "same-git")
	_, changedContract := snapshotSourceIdentity(snapshotSourceContract+"drift", "same-git")
	driftedPath := testGitExecutable(t)
	gitTestNoError(t, os.Chmod(driftedPath, 0o700))
	drifted, err := os.OpenFile(driftedPath, os.O_WRONLY, 0)
	gitTestNoError(t, err)
	if _, err := drifted.WriteAt([]byte{0}, 0); err != nil {
		_ = drifted.Close()
		t.Fatal(err)
	}
	gitTestNoError(t, drifted.Close())
	gitTestNoError(t, os.Chmod(driftedPath, 0o500))
	driftedAdapter := pinnedTestAdapter(t, driftedPath)
	_, changedBytes := driftedAdapter.SnapshotIdentity()
	if changedContract == originalContract || changedBytes == firstDigest {
		t.Fatal("contract or executable drift preserved snapshot identity")
	}
}

func TestAdapterCloseIsIdempotentConcurrentAndFailsClosed(t *testing.T) {
	path := testGitExecutable(t)
	adapter := pinnedTestAdapter(t, path)
	wantRef, wantDigest := adapter.SnapshotIdentity()
	fd := int(adapter.gitFile.Fd())
	const callers = 16
	var wait sync.WaitGroup
	errs := make(chan error, callers)
	for range callers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			errs <- adapter.Close()
		}()
	}
	wait.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent Close: %v", err)
		}
	}
	if _, err := unix.FcntlInt(uintptr(fd), unix.F_GETFD, 0); !errors.Is(err, unix.EBADF) {
		t.Fatalf("pinned descriptor remained open: %v", err)
	}
	if _, err := adapter.gitRun(context.Background(), "", nil, "--version"); ErrorCodeOf(err) != CodeUnavailable {
		t.Fatalf("closed command err=%v", err)
	}
	if _, err := adapter.OpenSnapshotStream(context.Background(), ports.SnapshotVerificationRequest{}); ErrorCodeOf(err) != CodeUnavailable {
		t.Fatalf("closed snapshot err=%v", err)
	}
	if ref, digest := adapter.SnapshotIdentity(); ref == "" || digest == "" {
		t.Fatal("Close discarded precomputed identity")
	}
	replayed := pinnedTestAdapter(t, path)
	if ref, digest := replayed.SnapshotIdentity(); ref != wantRef || digest != wantDigest {
		t.Fatalf("replayed identity=(%q,%q) want=(%q,%q)", ref, digest, wantRef, wantDigest)
	}
	if _, err := replayed.gitRun(context.Background(), "", nil, "--version"); err != nil {
		t.Fatalf("fresh replay after Close: %v", err)
	}
}

func TestPinnedGitConcurrentCommandsSurviveCloseRace(t *testing.T) {
	adapter := pinnedTestAdapter(t, testGitExecutable(t))
	const callers = 12
	start := make(chan struct{})
	errs := make(chan error, callers)
	var wait sync.WaitGroup
	for range callers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			output, err := adapter.gitRun(context.Background(), "", nil, "--version")
			if err == nil && !strings.HasPrefix(string(output), "git version ") {
				err = errors.New("unexpected pinned git output")
			}
			errs <- err
		}()
	}
	close(start)
	gitTestNoError(t, adapter.Close())
	wait.Wait()
	close(errs)
	for err := range errs {
		if err != nil && ErrorCodeOf(err) != CodeUnavailable {
			t.Fatalf("command/Close race err=%v", err)
		}
	}
}

func TestPinnedGitCommandsDoNotLeakDuplicatedDescriptors(t *testing.T) {
	adapter := pinnedTestAdapter(t, testGitExecutable(t))
	if _, err := adapter.gitRun(context.Background(), "", nil, "--version"); err != nil {
		t.Fatal(err)
	}
	before := openDescriptorCount(t)
	for range 20 {
		if _, err := adapter.gitRun(context.Background(), "", nil, "--version"); err != nil {
			t.Fatal(err)
		}
	}
	if after := openDescriptorCount(t); after != before {
		t.Fatalf("descriptor count before=%d after=%d", before, after)
	}
}

func openDescriptorCount(t *testing.T) int {
	t.Helper()
	entries, err := os.ReadDir("/proc/self/fd")
	gitTestNoError(t, err)
	return len(entries)
}

func pinnedTestAdapter(t *testing.T, path string) *Adapter {
	t.Helper()
	file, digest, err := pinGitExecutable(path, gitPinPolicy{ownerUID: uint32(os.Geteuid())})
	gitTestNoError(t, err)
	_, identityDigest := snapshotSourceIdentity(snapshotSourceContract, digest)
	adapter := &Adapter{
		root: t.TempDir(), gitFile: file,
		snapshotDigest: identityDigest,
	}
	t.Cleanup(func() { _ = adapter.Close() })
	return adapter
}
