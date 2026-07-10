package orquestaruntimecodexappserver

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestCleanupOwnedGenerationAfterStartupFailureV0SinMarkerNoCreaLeaseV0(t *testing.T) {
	root := filepath.Join(t.TempDir(), "runtime-not-created")
	backend := serverCodexAppServerTmuxBackendV0{
		SocketPath:  filepath.Join(root, codexAppServerTmuxDirV0, "s.sock"),
		SessionName: "orquesta-goal-clean-1234567890",
	}
	if err := backend.CleanupOwnedGenerationAfterStartupFailureV0(context.Background()); err != nil {
		t.Fatalf("cleanup sin marker: %v", err)
	}
	if _, err := os.Lstat(filepath.Dir(backend.SocketPath)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cleanup sin marker creo runtime/lease: %v", err)
	}
}

func TestCleanupOwnedGenerationAfterStartupFailureV0PropagaMarkerInvalidoV0(t *testing.T) {
	backend, _ := newGenerationLeaseBackendForTestV0(t)
	if err := os.WriteFile(backend.tmuxOwnerMarkerPathV0(), []byte("{invalid"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := backend.CleanupOwnedGenerationAfterStartupFailureV0(context.Background())
	var callErr codexAppServerCallErrorV0
	if !errors.As(err, &callErr) || callErr.Code != "codex_app_server_tmux_owner_marker_invalid" {
		t.Fatalf("marker invalido ocultado: %v", err)
	}
	if _, statErr := os.Lstat(backend.tmuxOwnerMarkerPathV0()); statErr != nil {
		t.Fatalf("marker invalido mutado: %v", statErr)
	}
}

func TestCleanupOwnedGenerationAfterStartupFailureV0AislaDosRuntimesV0(t *testing.T) {
	first, _ := newGenerationLeaseBackendForTestV0(t)
	secondRoot := shortUnixSocketTestRootV0(t)
	second := first
	second.RuntimeWorkDir = filepath.Join(secondRoot, "runtime")
	second.SocketPath = filepath.Join(second.RuntimeWorkDir, codexAppServerTmuxDirV0, "s.sock")
	second.SessionName = "orquesta-goal-second-1234567890"
	if err := os.MkdirAll(filepath.Dir(second.SocketPath), 0o700); err != nil {
		t.Fatal(err)
	}

	firstListener := listenUnixForGenerationTestV0(t, first.SocketPath)
	secondListener := listenUnixForGenerationTestV0(t, second.SocketPath)
	_ = firstListener.Close()
	defer secondListener.Close()
	firstMarker, err := first.newTmuxOwnerMarkerV0("generation-ref-first", 0)
	if err != nil {
		t.Fatal(err)
	}
	secondMarker, err := second.newTmuxOwnerMarkerV0("generation-ref-second", 0)
	if err != nil {
		t.Fatal(err)
	}
	writeGenerationMarkerTestV0(t, first.tmuxOwnerMarkerPathV0(), firstMarker)
	writeGenerationMarkerTestV0(t, second.tmuxOwnerMarkerPathV0(), secondMarker)

	if err := first.CleanupOwnedGenerationAfterStartupFailureV0(context.Background()); err != nil {
		t.Fatalf("cleanup first: %v", err)
	}
	if _, err := os.Lstat(first.tmuxOwnerMarkerPathV0()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("marker first no eliminado: %v", err)
	}
	if _, err := os.Lstat(second.tmuxOwnerMarkerPathV0()); err != nil {
		t.Fatalf("marker second tocado: %v", err)
	}
	if _, err := os.Lstat(second.SocketPath); err != nil {
		t.Fatalf("socket second tocado: %v", err)
	}
}

func TestCleanupOwnedGenerationAfterStartupFailureV0RechazaOwnerSocketGeneracionAjenosV0(t *testing.T) {
	backend, _ := newGenerationLeaseBackendForTestV0(t)
	foreign := backend
	foreign.SocketPath = filepath.Join(filepath.Dir(backend.SocketPath), "foreign.sock")
	foreign.SessionName = "orquesta-goal-foreign-1234567890"
	marker, err := foreign.newTmuxOwnerMarkerV0("generation-ref-foreign", 0)
	if err != nil {
		t.Fatal(err)
	}
	writeGenerationMarkerTestV0(t, backend.tmuxOwnerMarkerPathV0(), marker)
	err = backend.CleanupOwnedGenerationAfterStartupFailureV0(context.Background())
	var callErr codexAppServerCallErrorV0
	if !errors.As(err, &callErr) || callErr.Code != codexAppServerTmuxGenerationConflictV0 {
		t.Fatalf("owner/socket/generacion ajenos aceptados: %v", err)
	}
	if _, statErr := os.Lstat(backend.tmuxOwnerMarkerPathV0()); statErr != nil {
		t.Fatalf("marker ajeno mutado: %v", statErr)
	}
}

func TestCleanupOwnedGenerationAfterStartupFailureV0NoEscaneaTmpUsoAppAjenoV0(t *testing.T) {
	foreignRuntime, err := os.MkdirTemp(os.TempDir(), "oq-gsrv-foreign-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(foreignRuntime)
	process := exec.Command("bash", "-c", "exec -a 'uso-app codex app-server' sleep 30")
	process.Dir = foreignRuntime
	if err := process.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = process.Process.Kill(); _ = process.Wait() }()

	backend := serverCodexAppServerTmuxBackendV0{
		SocketPath:  filepath.Join(t.TempDir(), "owned", "s.sock"),
		SessionName: "orquesta-goal-owned-1234567890",
		Timeout:     100 * time.Millisecond,
	}
	if err := backend.CleanupOwnedGenerationAfterStartupFailureV0(context.Background()); err != nil {
		t.Fatalf("cleanup sin marker propio: %v", err)
	}
	if err := syscall.Kill(process.Process.Pid, 0); err != nil {
		t.Fatalf("proceso uso-app ajeno fue tocado: %v", err)
	}
}
