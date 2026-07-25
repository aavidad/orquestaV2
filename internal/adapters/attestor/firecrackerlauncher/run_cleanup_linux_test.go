//go:build linux

package firecrackerlauncher

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRunCleanupIsIdempotentAndNeverFollowsSymlinks(t *testing.T) {
	runner, _, _, _, request, input, _ := physicalRunnerFixture(t, "success")
	workspace, err := createRunWorkspace(
		context.Background(),
		runner.runs,
		runner.config,
		runner.assets,
		request,
		input,
	)
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("preserve"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(workspace.path, "untrusted-link")); err != nil {
		t.Fatal(err)
	}
	if err := workspace.Cleanup(); err != nil {
		t.Fatal(err)
	}
	if err := workspace.Cleanup(); err != nil {
		t.Fatalf("second cleanup: %v", err)
	}
	content, err := os.ReadFile(outside)
	if err != nil || string(content) != "preserve" {
		t.Fatalf("cleanup followed symlink: content=%q err=%v", content, err)
	}
	assertRunsEmpty(t, runner.runs.path)
	if err := runner.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestRunCleanupRefusesTamperedMarkerAndRunnerCloseReportsResidual(t *testing.T) {
	runner, _, _, _, request, input, _ := physicalRunnerFixture(t, "success")
	workspace, err := createRunWorkspace(
		context.Background(),
		runner.runs,
		runner.config,
		runner.assets,
		request,
		input,
	)
	if err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(workspace.path, runMarkerName)
	if err := os.Chmod(marker, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(marker, []byte("tampered\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if ErrorCode(workspace.Cleanup()) != CodeCleanupFailed {
		t.Fatal("tampered marker cleanup accepted")
	}
	if _, err := os.Stat(workspace.path); err != nil {
		t.Fatalf("unowned run removed: %v", err)
	}
	if ErrorCode(runner.Close()) != CodeCleanupFailed {
		t.Fatal("runner close hid residual run")
	}
}

func TestRunCleanupLimitsPreserveOwnedResidualMarker(t *testing.T) {
	tests := map[string]func(*testing.T, *runWorkspace){
		"entries": func(t *testing.T, workspace *runWorkspace) {
			for index := 0; index < 4; index++ {
				path := filepath.Join(workspace.path, "extra-"+string(rune('a'+index)))
				if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			workspace.cleanupEntries = 2
		},
		"depth": func(t *testing.T, workspace *runWorkspace) {
			if err := os.MkdirAll(
				filepath.Join(workspace.path, "deep", "deeper"),
				0o700,
			); err != nil {
				t.Fatal(err)
			}
			workspace.cleanupDepth = 1
		},
		"deadline": func(_ *testing.T, _ *runWorkspace) {},
	}
	for name, prepare := range tests {
		t.Run(name, func(t *testing.T) {
			runner, _, _, _, request, input, _ := physicalRunnerFixture(t, "success")
			workspace, err := createRunWorkspace(
				context.Background(),
				runner.runs,
				runner.config,
				runner.assets,
				request,
				input,
			)
			if err != nil {
				t.Fatal(err)
			}
			prepare(t, workspace)
			ctx := context.Background()
			if name == "deadline" {
				canceled, cancel := context.WithCancel(context.Background())
				cancel()
				ctx = canceled
			}
			if ErrorCode(workspace.CleanupContext(ctx)) != CodeCleanupFailed {
				t.Fatal("bounded cleanup unexpectedly succeeded")
			}
			if _, err := os.Stat(filepath.Join(workspace.path, runMarkerName)); err != nil {
				t.Fatalf("owned residual marker not preserved: %v", err)
			}
			if ErrorCode(runner.Close()) != CodeCleanupFailed {
				t.Fatal("runner Close hid bounded cleanup residual")
			}
		})
	}
}
