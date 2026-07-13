package orquestaruntimecodexappserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	orquestaservershutdown "orquesta/modulos/orquesta-server-shutdown"
)

type failingCodexAppServerProbeGenerationV0 struct{}

func (failingCodexAppServerProbeGenerationV0) ProbeV0(context.Context) error {
	return errors.New("probe unavailable")
}

type transientPostCASCodexAppServerProbeV0 struct {
	calls int
}

func (probe *transientPostCASCodexAppServerProbeV0) ProbeV0(context.Context) error {
	probe.calls++
	if probe.calls == 2 {
		return errors.New("transient post-CAS probe observation")
	}
	return nil
}

func TestGenerationLeaseUnixServerHelperV0(t *testing.T) {
	if os.Getenv("ORQUESTA_TEST_UNIX_SERVER_HELPER") != "1" {
		return
	}
	args := os.Args
	if len(args) == 0 {
		os.Exit(2)
	}
	if os.Getenv("ORQUESTA_TEST_UNIX_SERVER_IGNORE_TERM") == "1" {
		signal.Ignore(syscall.SIGTERM)
	}
	socketPath := strings.TrimSpace(os.Getenv("ORQUESTA_TEST_UNIX_SOCKET_PATH"))
	if socketPath == "" {
		socketPath = args[len(args)-1]
	}
	fd, err := listenUnixFDForGenerationTestV0(socketPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "unix helper listen %q: %v\n", socketPath, err)
		os.Exit(3)
	}
	defer syscall.Close(fd)
	for {
		connection, _, acceptErr := syscall.Accept(fd)
		if acceptErr != nil {
			return
		}
		_ = syscall.Close(connection)
	}
}

func TestMarkerCASProcessHelperV0(t *testing.T) {
	if os.Getenv("TEST_MARKER_CAS_HELPER") != "1" {
		return
	}
	expectedRaw, err := os.ReadFile(os.Getenv("TEST_MARKER_EXPECTED"))
	if err != nil {
		t.Fatal(err)
	}
	var expected codexAppServerTmuxOwnerMarkerV0
	if err := json.Unmarshal(expectedRaw, &expected); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(os.Getenv("TEST_MARKER_READY"), []byte("ready\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(os.Getenv("TEST_MARKER_START")); err == nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	backend := serverCodexAppServerTmuxBackendV0{
		SocketPath:  expected.SocketPath,
		SessionName: expected.SessionName,
		Timeout:     2 * time.Second,
	}
	next := expected
	next.SocketRef = os.Getenv("TEST_MARKER_NEXT")
	err = backend.replaceTmuxOwnerMarkerV0(expected, next)
	result := "success\n"
	if err != nil {
		var callErr codexAppServerCallErrorV0
		if !errors.As(err, &callErr) || callErr.Code != codexAppServerTmuxGenerationConflictV0 {
			t.Fatalf("CAS helper: %v", err)
		}
		result = "conflict\n"
	}
	if err := os.WriteFile(os.Getenv("TEST_MARKER_RESULT"), []byte(result), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestSocketOwnerWrapperHelperV0(t *testing.T) {
	if os.Getenv("ORQUESTA_TEST_SOCKET_WRAPPER_HELPER") != "1" {
		return
	}
	listenerFile := os.NewFile(3, "inherited-unix-listener")
	if listenerFile == nil {
		t.Fatal("fd 3 ausente")
	}
	child := exec.Command("sleep", "30")
	child.ExtraFiles = []*os.File{listenerFile}
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	_ = listenerFile.Close()
	if err := os.WriteFile(os.Getenv("ORQUESTA_TEST_SOCKET_OWNER_PATH"), []byte(strconv.Itoa(child.Process.Pid)+"\n"), 0o600); err != nil {
		_ = child.Process.Kill()
		t.Fatal(err)
	}
	if err := child.Wait(); err != nil {
		t.Fatal(err)
	}
}

func TestSocketOwnerV0AceptaDescendienteDeWrapperV0(t *testing.T) {
	root := shortUnixSocketTestRootV0(t)
	socketPath := filepath.Join(root, "wrapped.sock")
	ownerPath := filepath.Join(root, "owner.pid")
	listenerFD, err := listenUnixFDForGenerationTestV0(socketPath)
	if err != nil {
		if errors.Is(err, syscall.EPERM) {
			t.Skipf("sandbox no permite listen Unix: %v", err)
		}
		t.Fatal(err)
	}
	listenerFile := os.NewFile(uintptr(listenerFD), "wrapped-listener")
	command := exec.Command(os.Args[0], "-test.run=^TestSocketOwnerWrapperHelperV0$")
	command.Env = append(os.Environ(),
		"ORQUESTA_TEST_SOCKET_WRAPPER_HELPER=1",
		"ORQUESTA_TEST_SOCKET_PATH="+socketPath,
		"ORQUESTA_TEST_SOCKET_OWNER_PATH="+ownerPath,
	)
	command.ExtraFiles = []*os.File{listenerFile}
	var commandOutput bytes.Buffer
	command.Stdout = &commandOutput
	command.Stderr = &commandOutput
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	_ = listenerFile.Close()
	t.Cleanup(func() {
		_ = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
		_ = command.Wait()
	})
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && !codexAppServerTmuxSocketListenerAliveV0(socketPath) {
		time.Sleep(20 * time.Millisecond)
	}
	if !codexAppServerTmuxSocketListenerAliveV0(socketPath) {
		t.Fatalf("listener wrapper no arranco: output=%s", commandOutput.String())
	}
	var wantPID int
	var ownerHandshakeErr error
	for time.Now().Before(deadline) {
		raw, readErr := os.ReadFile(ownerPath)
		if readErr == nil {
			text := strings.TrimSpace(string(raw))
			if text != "" {
				pid, parseErr := strconv.Atoi(text)
				if parseErr == nil && strconv.Itoa(pid) == text && pid > 0 {
					aliveErr := syscall.Kill(pid, 0)
					if aliveErr == nil || errors.Is(aliveErr, syscall.EPERM) {
						wantPID = pid
						break
					}
					ownerHandshakeErr = fmt.Errorf("owner pid %d no esta vivo: %w", pid, aliveErr)
				} else {
					ownerHandshakeErr = fmt.Errorf("owner.pid incompleto/no parseable %q: %v", text, parseErr)
				}
			}
		} else {
			ownerHandshakeErr = readErr
		}
		time.Sleep(20 * time.Millisecond)
	}
	if wantPID == 0 {
		t.Fatalf("handshake owner.pid no completado: err=%v output=%s", ownerHandshakeErr, commandOutput.String())
	}
	owner, err := codexAppServerTmuxSocketOwnerDescendantV0(socketPath, command.Process.Pid)
	if err != nil {
		t.Fatalf("owner descendiente: %v", err)
	}
	if owner.PID != wantPID || owner.PID == command.Process.Pid || owner.StartRef == "" {
		t.Fatalf("owner=%+v wrapper=%d want_child=%d", owner, command.Process.Pid, wantPID)
	}
}

func TestSocketOwnerV0RechazaListenerNoDescendienteV0(t *testing.T) {
	root := shortUnixSocketTestRootV0(t)
	socketPath := filepath.Join(root, "foreign.sock")
	listenerFD, err := listenUnixFDForGenerationTestV0(socketPath)
	if err != nil {
		if errors.Is(err, syscall.EPERM) {
			t.Skipf("sandbox no permite listen Unix: %v", err)
		}
		t.Fatal(err)
	}
	defer syscall.Close(listenerFD)
	foreignPane := exec.Command("sh", "-c", "sleep 30")
	if err := foreignPane.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = foreignPane.Process.Kill(); _ = foreignPane.Wait() })
	if _, err := codexAppServerTmuxSocketOwnerDescendantV0(socketPath, foreignPane.Process.Pid); err == nil {
		t.Fatal("listener ajeno al arbol fue aceptado")
	}
}

func TestSocketOwnerProcV0ResuelveNativeDescendienteDeWrapperV0(t *testing.T) {
	procRoot := t.TempDir()
	socketPath := "/private/runtime/app owner.sock"
	writeProcUnixFixtureV0(t, procRoot, "9001", socketPath)
	writeProcProcessFixtureV0(t, procRoot, 100, 1, "1000", "")
	writeProcProcessFixtureV0(t, procRoot, 110, 100, "1100", "")
	writeProcProcessFixtureV0(t, procRoot, 120, 110, "1200", "socket:[9001]")
	owner, err := codexAppServerTmuxSocketOwnerDescendantAtV0(procRoot, socketPath, 100)
	if err != nil || owner.PID != 120 || owner.StartRef != "1200" {
		t.Fatalf("owner=%+v err=%v", owner, err)
	}
}

func TestSocketOwnerProcV0IgnoraPeersConectadosDelMismoPathV0(t *testing.T) {
	procRoot := t.TempDir()
	socketPath := "/private/runtime/app.sock"
	writeProcUnixRowsFixtureV0(t, procRoot,
		procUnixRowFixtureV0{inode: "9100", socketPath: socketPath, flags: "00010000", socketType: "0001", state: "01"},
		procUnixRowFixtureV0{inode: "9101", socketPath: socketPath, flags: "00000000", socketType: "0001", state: "03"},
	)
	writeProcProcessFixtureV0(t, procRoot, 100, 1, "1000", "")
	writeProcProcessFixtureV0(t, procRoot, 120, 100, "1200", "socket:[9100]")
	writeProcProcessFixtureV0(t, procRoot, 130, 100, "1300", "socket:[9101]")
	owner, err := codexAppServerTmuxSocketOwnerDescendantAtV0(procRoot, socketPath, 100)
	if err != nil || owner.PID != 120 || owner.StartRef != "1200" {
		t.Fatalf("owner listener=%+v err=%v", owner, err)
	}
}

func TestSocketOwnerProcV0DeduplicaMismoListenerSinOcultarRebindV0(t *testing.T) {
	procRoot := t.TempDir()
	socketPath := "/private/runtime/app.sock"
	row := procUnixRowFixtureV0{inode: "9200", socketPath: socketPath, flags: "00010000", socketType: "0001", state: "01"}
	writeProcUnixRowsFixtureV0(t, procRoot, row, row)
	writeProcProcessFixtureV0(t, procRoot, 100, 1, "1000", "")
	writeProcProcessFixtureV0(t, procRoot, 120, 100, "1200", "socket:[9200]")
	if owner, err := codexAppServerTmuxSocketOwnerDescendantAtV0(procRoot, socketPath, 100); err != nil || owner.PID != 120 {
		t.Fatalf("fila duplicada del mismo listener: owner=%+v err=%v", owner, err)
	}

	writeProcUnixRowsFixtureV0(t, procRoot, row,
		procUnixRowFixtureV0{inode: "9201", socketPath: socketPath, flags: "00010000", socketType: "0001", state: "01"},
	)
	if _, err := codexAppServerTmuxSocketOwnerDescendantAtV0(procRoot, socketPath, 100); err == nil || !strings.Contains(err.Error(), "socket_inode_ambiguous") {
		t.Fatalf("dos listeners distintos no fueron ambiguos: %v", err)
	}
}

func TestSocketOwnerProcV0EligeHojaUnicaDeWrapperQueRetieneFDV0(t *testing.T) {
	procRoot := t.TempDir()
	socketPath := "/private/runtime/app.sock"
	writeProcUnixFixtureV0(t, procRoot, "9300", socketPath)
	writeProcProcessFixtureV0(t, procRoot, 100, 1, "1000", "")
	writeProcProcessFixtureV0(t, procRoot, 110, 100, "1100", "socket:[9300]")
	writeProcProcessFixtureV0(t, procRoot, 120, 110, "1200", "socket:[9300]")
	owner, err := codexAppServerTmuxSocketOwnerDescendantAtV0(procRoot, socketPath, 100)
	if err != nil || owner.PID != 120 {
		t.Fatalf("owner hoja=%+v err=%v", owner, err)
	}
}

func TestSocketOwnerProcV0RechazaHolderAjenoYAceptacionAmbiguaV0(t *testing.T) {
	procRoot := t.TempDir()
	socketPath := "/private/runtime/app.sock"
	writeProcUnixFixtureV0(t, procRoot, "9400", socketPath)
	writeProcProcessFixtureV0(t, procRoot, 100, 1, "1000", "")
	writeProcProcessFixtureV0(t, procRoot, 120, 100, "1200", "socket:[9400]")
	writeProcProcessFixtureV0(t, procRoot, 200, 1, "2000", "socket:[9400]")
	if _, err := codexAppServerTmuxSocketOwnerDescendantAtV0(procRoot, socketPath, 100); err == nil || !strings.Contains(err.Error(), "owner_outside_pane") {
		t.Fatalf("holder ajeno fue aceptado: %v", err)
	}

	procRoot = t.TempDir()
	writeProcUnixFixtureV0(t, procRoot, "9401", socketPath)
	writeProcProcessFixtureV0(t, procRoot, 100, 1, "1000", "")
	writeProcProcessFixtureV0(t, procRoot, 120, 100, "1200", "socket:[9401]")
	writeProcProcessFixtureV0(t, procRoot, 130, 100, "1300", "socket:[9401]")
	if _, err := codexAppServerTmuxSocketOwnerDescendantAtV0(procRoot, socketPath, 100); err == nil || !strings.Contains(err.Error(), "owner_ambiguous") {
		t.Fatalf("holders hermanos fueron aceptados: %v", err)
	}
}

func TestSocketOwnerProcV0RechazaOwnerNoDescendienteV0(t *testing.T) {
	procRoot := t.TempDir()
	socketPath := "/private/runtime/app.sock"
	writeProcUnixFixtureV0(t, procRoot, "9002", socketPath)
	writeProcProcessFixtureV0(t, procRoot, 100, 1, "1000", "")
	writeProcProcessFixtureV0(t, procRoot, 200, 1, "2000", "socket:[9002]")
	if _, err := codexAppServerTmuxSocketOwnerDescendantAtV0(procRoot, socketPath, 100); err == nil {
		t.Fatal("owner no descendiente aceptado")
	}
}

func TestEnsureV0AdoptaGeneracionExactaCuandoTmuxDesapareceV0(t *testing.T) {
	backend, tmuxLog := newGenerationLeaseBackendForTestV0(t)
	listener := listenUnixForGenerationTestV0(t, backend.SocketPath)
	defer listener.Close()
	marker := generationMarkerForCurrentProcessTestV0(t, backend, "generation-ref-surviving")
	marker.TmuxSessionID = "$vanished"
	marker.TmuxSessionCreated = "100"
	writeGenerationMarkerTestV0(t, backend.tmuxOwnerMarkerPathV0(), marker)

	if err := backend.EnsureV0(context.Background(), fakeCodexAppServerProbeV0{}); err != nil {
		t.Fatalf("EnsureV0 debe adoptar la generacion viva: %v", err)
	}
	got, ok := backend.readTmuxOwnerMarkerV0()
	if !ok || got.GenerationRef != marker.GenerationRef || got.AppServerPID != os.Getpid() {
		t.Fatalf("marker sustituido: %+v ok=%v", got, ok)
	}
	assertTmuxLogExcludesV0(t, tmuxLog, "new-session", "kill-session")
}

func TestEnsureV0SinTmuxRechazaMarkerIncompletoOMismatchedV0(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		mutate func(*codexAppServerTmuxOwnerMarkerV0)
	}{
		{name: "identidad tmux incompleta", mutate: func(marker *codexAppServerTmuxOwnerMarkerV0) {
			marker.TmuxSessionID = ""
			marker.TmuxSessionCreated = ""
		}},
		{name: "app server distinto del owner", mutate: func(marker *codexAppServerTmuxOwnerMarkerV0) {
			marker.AppServerStartRef += "-reused"
		}},
		{name: "pane reutilizado", mutate: func(marker *codexAppServerTmuxOwnerMarkerV0) {
			marker.TmuxPaneStartRef += "-reused"
		}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			backend, tmuxLog := newGenerationLeaseBackendForTestV0(t)
			listener := listenUnixForGenerationTestV0(t, backend.SocketPath)
			defer listener.Close()
			marker := generationMarkerForCurrentProcessTestV0(t, backend, "generation-ref-no-takeover")
			marker.TmuxSessionID = "$vanished"
			marker.TmuxSessionCreated = "100"
			testCase.mutate(&marker)
			writeGenerationMarkerTestV0(t, backend.tmuxOwnerMarkerPathV0(), marker)

			assertGenerationConflictTestV0(t, backend.EnsureV0(context.Background(), fakeCodexAppServerProbeV0{}))
			got, ok := backend.readTmuxOwnerMarkerV0()
			if !ok || !reflect.DeepEqual(got, marker) {
				t.Fatalf("marker inseguro mutado: got=%+v want=%+v ok=%v", got, marker, ok)
			}
			assertTmuxLogExcludesV0(t, tmuxLog, "new-session", "kill-session")
		})
	}
}

func TestEnsureV0ConTmuxRechazaOtraGeneracionV0(t *testing.T) {
	backend, tmuxLog := newGenerationLeaseBackendForTestV0(t)
	listener := listenUnixForGenerationTestV0(t, backend.SocketPath)
	defer listener.Close()
	marker := generationMarkerForCurrentProcessTestV0(t, backend, "generation-ref-expected")
	marker.TmuxSessionID = "$present"
	marker.TmuxSessionCreated = "100"
	writeGenerationMarkerTestV0(t, backend.tmuxOwnerMarkerPathV0(), marker)
	writeFakeTmuxStateTestV0(t, tmuxLog, marker.TmuxSessionID, marker.TmuxSessionCreated, marker.TmuxPanePID)
	if err := os.WriteFile(tmuxLog+".generation", []byte("generation-ref-foreign\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	assertGenerationConflictTestV0(t, backend.EnsureV0(context.Background(), fakeCodexAppServerProbeV0{}))
	got, ok := backend.readTmuxOwnerMarkerV0()
	if !ok || !reflect.DeepEqual(got, marker) {
		t.Fatalf("otra generacion muto marker: got=%+v want=%+v ok=%v", got, marker, ok)
	}
	assertTmuxLogExcludesV0(t, tmuxLog, "new-session", "kill-session")
}

func TestEnsureV0NoAdoptaSiAusenciaTmuxNoEsObservableV0(t *testing.T) {
	backend, tmuxLog := newGenerationLeaseBackendForTestV0(t)
	listener := listenUnixForGenerationTestV0(t, backend.SocketPath)
	defer listener.Close()
	marker := generationMarkerForCurrentProcessTestV0(t, backend, "generation-ref-observation-required")
	marker.TmuxSessionID = "$vanished"
	marker.TmuxSessionCreated = "100"
	writeGenerationMarkerTestV0(t, backend.tmuxOwnerMarkerPathV0(), marker)
	tmuxPath := filepath.Join(strings.Split(backend.PathEnv, string(os.PathListSeparator))[0], "tmux")
	if err := os.WriteFile(tmuxPath, []byte("#!/bin/sh\nprintf '%s\\n' 'permission denied by tmux server' >&2\nexit 1\n"), 0o700); err != nil {
		t.Fatal(err)
	}

	err := backend.EnsureV0(context.Background(), fakeCodexAppServerProbeV0{})
	var callErr codexAppServerCallErrorV0
	if !errors.As(err, &callErr) || callErr.Code != "codex_app_server_permission_denied" {
		t.Fatalf("error de observacion convertido en ausencia: %v", err)
	}
	got, ok := backend.readTmuxOwnerMarkerV0()
	if !ok || !reflect.DeepEqual(got, marker) {
		t.Fatalf("fallo de observacion muto marker: got=%+v want=%+v ok=%v", got, marker, ok)
	}
	assertTmuxLogExcludesV0(t, tmuxLog, "new-session", "kill-session")
}

func TestCleanupGeneracionV0RechazaSesionReemplazadaSinKillNiUnlinkV0(t *testing.T) {
	backend, tmuxLog := newGenerationLeaseBackendForTestV0(t)
	listener := listenUnixForGenerationTestV0(t, backend.SocketPath)
	defer listener.Close()
	marker := generationMarkerForCurrentProcessTestV0(t, backend, "generation-ref-replaced")
	marker.AppServerPID = 99999999
	marker.AppServerStartRef = "dead"
	marker.TmuxSessionID = "$old"
	marker.TmuxSessionCreated = "10"
	writeGenerationMarkerTestV0(t, backend.tmuxOwnerMarkerPathV0(), marker)
	writeFakeTmuxStateTestV0(t, tmuxLog, "$new", "20", os.Getpid())

	err := backend.cleanupTmuxGenerationV0(context.Background(), marker, true)
	assertGenerationConflictTestV0(t, err)
	assertTmuxLogExcludesV0(t, tmuxLog, "kill-session")
	if _, err := os.Lstat(backend.SocketPath); err != nil {
		t.Fatalf("socket reemplazado fue borrado: %v", err)
	}
}

func TestCleanupShutdownV0ReportaResiduoGeneracionalSinFalsoFalloNiMutacionV0(t *testing.T) {
	backend, tmuxLog := newGenerationLeaseBackendForTestV0(t)
	listener := listenUnixForGenerationTestV0(t, backend.SocketPath)
	defer listener.Close()
	marker := generationMarkerForCurrentProcessTestV0(t, backend, "generation-ref-cleanup-residual")
	marker.AppServerPID = 99999999
	marker.AppServerStartRef = "dead"
	marker.TmuxSessionID = "$expected"
	marker.TmuxSessionCreated = "10"
	writeGenerationMarkerTestV0(t, backend.tmuxOwnerMarkerPathV0(), marker)
	writeFakeTmuxStateTestV0(t, tmuxLog, "$replacement", "20", os.Getpid())

	result, err := backend.CleanupActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkCleanupCommandV0{
		CleanupGoalBackends: true,
	})
	if err != nil {
		t.Fatalf("cleanup shutdown no debe fallar solo por generation conflict: %v", err)
	}
	if result.CleanedWorkCount != 0 || !containsStringMigratedTestV0(result.EvidenceRefs, "evidence-ref-codex-app-server-tmux-generation-conflict-residual") {
		t.Fatalf("cleanup residual=%+v", result)
	}
	assertTmuxLogExcludesV0(t, tmuxLog, "kill-session")
	if !codexAppServerTmuxSocketListenerAliveV0(backend.SocketPath) {
		t.Fatal("cleanup residual termino un socket no verificado")
	}
	if got, ok := backend.readTmuxOwnerMarkerV0(); !ok || !reflect.DeepEqual(got, marker) {
		t.Fatalf("cleanup residual muto marker: got=%+v ok=%v", got, ok)
	}
	assertGenerationConflictTestV0(t, backend.ShutdownV0(context.Background()))
	assertTmuxLogExcludesV0(t, tmuxLog, "kill-session")
}

func TestCleanupShutdownV0UsaMarkerYTokenSiReobservacionTmuxEsTransitoriaV0(t *testing.T) {
	backend, _ := newGenerationLeaseBackendForTestV0(t)
	t.Setenv("ORQUESTA_TEST_UNIX_SERVER_IGNORE_TERM", "1")
	if err := backend.EnsureV0(context.Background(), fakeCodexAppServerProbeV0{}); err != nil {
		t.Fatalf("EnsureV0: %v", err)
	}
	t.Setenv("ORQUESTA_TEST_TMUX_INVALID_IDENTITY", "1")
	result, err := backend.CleanupActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkCleanupCommandV0{
		CleanupGoalBackends: true,
	})
	if err != nil {
		t.Fatalf("cleanup con reobservacion transitoria: %v", err)
	}
	if result.CleanedWorkCount != 1 || !containsStringMigratedTestV0(result.EvidenceRefs, "evidence-ref-codex-app-server-tmux-configured-cleaned") {
		t.Fatalf("cleanup transitorio=%+v", result)
	}
	if _, ok := backend.readTmuxOwnerMarkerV0(); ok || codexAppServerTmuxSocketPresentV0(backend.SocketPath) {
		t.Fatal("cleanup transitorio dejo marker o socket configurado")
	}
}

func TestTmuxCommandsV0UsanTargetExactoV0(t *testing.T) {
	backend, tmuxLog := newGenerationLeaseBackendForTestV0(t)
	writeFakeTmuxStateTestV0(t, tmuxLog, "$exact", "30", os.Getpid())
	tmuxPath, err := codexAppServerTmuxCommandPathV0(backend.PathEnv)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := backend.tmuxHasSessionV0(context.Background(), tmuxPath); err != nil {
		t.Fatal(err)
	}
	identity, err := backend.tmuxSessionIdentityV0(context.Background(), tmuxPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := backend.tmuxKillSessionV0(context.Background(), tmuxPath, identity); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(tmuxLog)
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if strings.Contains(line, "has-session") {
			if !strings.Contains(line, "-t ="+backend.SessionName+":") {
				t.Fatalf("target tmux no exacto: %q", line)
			}
		}
		if strings.Contains(line, "display-message") &&
			!strings.Contains(line, "-t ="+backend.SessionName+":") && !strings.Contains(line, "-t $exact") {
			t.Fatalf("display target inesperado: %q", line)
		}
		if strings.Contains(line, "kill-session") && !strings.Contains(line, "-t $exact") {
			t.Fatalf("kill no usa session_id inmutable: %q", line)
		}
	}
}

func TestEnsureV0PrimerLanzamientoTmuxRealConSocketRealV0(t *testing.T) {
	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		t.Skip("tmux no disponible")
	}
	root := shortUnixSocketTestRootV0(t)
	tmuxRoot := filepath.Join(root, "tmux")
	if err := os.MkdirAll(tmuxRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMUX_TMPDIR", tmuxRoot)
	t.Setenv("ORQUESTA_TEST_BINARY", os.Args[0])
	t.Setenv("ORQUESTA_TEST_UNIX_SERVER_HELPER", "1")

	commandPath := filepath.Join(root, "codex-test-wrapper")
	wrapper := "#!/bin/sh\nexec \"$ORQUESTA_TEST_BINARY\" -test.run=^TestGenerationLeaseUnixServerHelperV0$\n"
	if err := os.WriteFile(commandPath, []byte(wrapper), 0o700); err != nil {
		t.Fatal(err)
	}
	runtimeDir := filepath.Join(root, "runtime")
	backend := serverCodexAppServerTmuxBackendV0{
		CommandPath:    commandPath,
		PathEnv:        os.Getenv("PATH"),
		SocketPath:     filepath.Join(runtimeDir, codexAppServerTmuxDirV0, "real.sock"),
		SessionName:    fmt.Sprintf("orquesta-goal-real-%d", os.Getpid()),
		HomeDir:        root,
		RuntimeWorkDir: runtimeDir,
		ProjectWorkDir: root,
		Timeout:        5 * time.Second,
	}
	t.Setenv("ORQUESTA_TEST_UNIX_SOCKET_PATH", backend.SocketPath)
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = backend.ShutdownV0(cleanupCtx)
		_, _ = backend.runTmuxCommandV0(cleanupCtx, tmuxPath, "kill-server")
	})

	if err := backend.EnsureV0(context.Background(), fakeCodexAppServerProbeV0{}); err != nil {
		t.Fatalf("primer EnsureV0 con tmux real: %v", err)
	}
	marker, ok := backend.readTmuxOwnerMarkerV0()
	if !ok || !marker.tmuxIdentityV0().completeV0() || !marker.appServerAliveV0(backend.SocketPath) {
		t.Fatalf("primera generacion incompleta: marker=%+v ok=%v", marker, ok)
	}
	identity, err := backend.tmuxSessionIdentityV0(context.Background(), tmuxPath)
	if err != nil || !reflect.DeepEqual(identity, marker.tmuxIdentityV0()) {
		t.Fatalf("identidad tmux real: got=%+v want=%+v err=%v", identity, marker.tmuxIdentityV0(), err)
	}
	if err := backend.ShutdownV0(context.Background()); err != nil {
		t.Fatalf("shutdown primera generacion tmux real: %v", err)
	}
	if _, ok := backend.readTmuxOwnerMarkerV0(); ok || codexAppServerTmuxSocketPresentV0(backend.SocketPath) {
		t.Fatal("shutdown tmux real dejo marker o socket configurado")
	}
}

func TestTmuxSessionIdentityFromOutputV0AceptaSeparadorPortableYLegacyV0(t *testing.T) {
	pid := os.Getpid()
	for _, output := range []string{
		fmt.Sprintf("$7|1700000000|%d", pid),
		fmt.Sprintf("$7\t1700000000\t%d", pid),
	} {
		identity, err := codexAppServerTmuxSessionIdentityFromOutputV0(output)
		if err != nil {
			t.Fatalf("output=%q: %v", output, err)
		}
		if identity.SessionID != "$7" || identity.SessionCreated != "1700000000" || identity.PanePID != pid {
			t.Fatalf("output=%q identity=%+v", output, identity)
		}
	}
}

func TestEnsureV0ReintentaPredicadoPostCASTransitorioV0(t *testing.T) {
	backend, _ := newGenerationLeaseBackendForTestV0(t)
	probe := &transientPostCASCodexAppServerProbeV0{}
	if err := backend.EnsureV0(context.Background(), probe); err != nil {
		t.Fatalf("EnsureV0 con observacion post-CAS transitoria: %v", err)
	}
	if probe.calls < 3 {
		t.Fatalf("probe calls=%d; no reintento tras marker completo", probe.calls)
	}
	marker, ok := backend.readTmuxOwnerMarkerV0()
	if !ok || marker.SocketOwnerPID <= 0 || marker.SocketOwnerStartRef == "" {
		t.Fatalf("marker post-CAS incompleto: %+v ok=%v", marker, ok)
	}
	if err := backend.ShutdownV0(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}

func TestWithVerifiedGenerationV0PostverifyRechazaRotacionAunqueRPCFalleV0(t *testing.T) {
	backend, _ := newGenerationLeaseBackendForTestV0(t)
	if err := backend.EnsureV0(context.Background(), fakeCodexAppServerProbeV0{}); err != nil {
		t.Fatalf("EnsureV0: %v", err)
	}
	marker, ok := backend.readTmuxOwnerMarkerV0()
	if !ok {
		t.Fatal("marker ausente")
	}
	lazy := serverCodexAppServerLazyTmuxProtocolV0{Backend: backend}
	_, err := lazy.withVerifiedGenerationV0(context.Background(), marker.GenerationRef, func(serverCodexAppServerProtocolPortV0, string) error {
		// This hook is the blocked RPC while withVerifiedGenerationV0 owns the
		// lease. A real generation rotation after the RPC must invalidate its
		// thread ID even when the RPC itself failed.
		replacement := marker
		replacement.GenerationRef = "generation-ref-rotated-postverify"
		writeGenerationMarkerTestV0(t, backend.tmuxOwnerMarkerPathV0(), replacement)
		return errors.New("rpc blocked by test hook")
	})
	assertGenerationConflictTestV0(t, err)
	writeGenerationMarkerTestV0(t, backend.tmuxOwnerMarkerPathV0(), marker)
	if err := backend.ShutdownV0(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}

func TestSmokeGoalFirstHandoffPipelinePropagaFalloV0(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(repoRoot, "scripts", "smoke_goal_first_app_server_real.sh")
	logPath := filepath.Join(t.TempDir(), "handoff.log")
	smokeRoot := t.TempDir()
	command := exec.Command("bash", "-o", "pipefail", "-c", fmt.Sprintf("%q | tee %q", script, logPath))
	command.Env = append(os.Environ(),
		"SMOKE_GOAL_FIRST_HANDOFF_FAILURE_SELFTEST=1",
		"ORQUESTA_SMOKE_ROOT="+smokeRoot,
		"ORQUESTA_CODEX_RUNTIME_WORKDIR=",
		"TMPDIR="+t.TempDir(),
	)
	output, runErr := command.CombinedOutput()
	var exitErr *exec.ExitError
	if !errors.As(runErr, &exitErr) || exitErr.ExitCode() != 41 {
		t.Fatalf("handoff pipeline rc=%v output=%s", runErr, output)
	}
	if !strings.Contains(string(output), "smoke_goal_first_handoff_failure_selftest=expected_failure") {
		t.Fatalf("handoff pipeline sin evidencia: %s", output)
	}
	if strings.Contains(string(output), "find_tmux_owner_file: command not found") {
		t.Fatalf("cleanup del handoff ejecuto helper no definido: %s", output)
	}
}

func TestEnsureV0NoQuedaVerdeConPanePIDCeroV0(t *testing.T) {
	backend, tmuxLog := newGenerationLeaseBackendForTestV0(t)
	t.Setenv("ORQUESTA_TEST_TMUX_INVALID_IDENTITY", "1")
	err := backend.EnsureV0(context.Background(), fakeCodexAppServerProbeV0{})
	assertGenerationConflictTestV0(t, err)
	marker, ok := backend.readTmuxOwnerMarkerV0()
	if !ok || marker.AppServerPID != 0 || marker.AppServerStartRef != "" {
		t.Fatalf("startup fallido no preservo su marker incompleto para fencing: %+v ok=%v", marker, ok)
	}
	if err := backend.EnsureV0(context.Background(), fakeCodexAppServerProbeV0{}); err == nil {
		t.Fatal("marker con PID cero fue adoptado como verde")
	}
	raw, _ := os.ReadFile(tmuxLog)
	if strings.Count(string(raw), "new-session") != 1 {
		t.Fatalf("startup duplicado: %s", raw)
	}
}

func TestProvisionalGenerationV0RecuperaSesionPorTokenYCierraExactaV0(t *testing.T) {
	backend, tmuxLog := newGenerationLeaseBackendForTestV0(t)
	marker, err := backend.newTmuxOwnerMarkerV0("generation-ref-provisional-recovery", 0)
	if err != nil {
		t.Fatal(err)
	}
	writeGenerationMarkerTestV0(t, backend.tmuxOwnerMarkerPathV0(), marker)
	writeFakeTmuxStateTestV0(t, tmuxLog, "$provisional", "55", os.Getpid())
	if err := os.WriteFile(tmuxLog+".generation", []byte(marker.GenerationRef+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	guard, err := backend.acquireTmuxLeaseGuardV0(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer guard.releaseV0()
	adopted, err := backend.adoptOrFenceExistingGenerationV0(context.Background(), fakeCodexAppServerProbeV0{}, guard)
	if err != nil || adopted {
		t.Fatalf("recovery provisional adopted=%v err=%v", adopted, err)
	}
	if _, err := os.Lstat(backend.tmuxOwnerMarkerPathV0()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("marker provisional exacto no retirado: %v", err)
	}
	raw, _ := os.ReadFile(tmuxLog)
	if !strings.Contains(string(raw), "kill-session -t $provisional") {
		t.Fatalf("recovery no mato session_id exacta: %s", raw)
	}
}

func TestMarkerCASV0ComparaTodosLosCamposConMismaGeneracionV0(t *testing.T) {
	backend, _ := newGenerationLeaseBackendForTestV0(t)
	marker := generationMarkerForCurrentProcessTestV0(t, backend, "generation-ref-same")
	writeGenerationMarkerTestV0(t, backend.tmuxOwnerMarkerPathV0(), marker)
	stale := marker
	stale.SocketRef = "stale-socket-ref"
	next := stale
	next.LeaseOwnerPID++
	assertGenerationConflictTestV0(t, backend.replaceTmuxOwnerMarkerV0(stale, next))
	got, ok := backend.readTmuxOwnerMarkerV0()
	if !ok || !reflect.DeepEqual(got, marker) {
		t.Fatalf("CAS stale muto marker: got=%+v ok=%v", got, ok)
	}
}

func TestMarkerCASV0ConcurrenteMismaGeneracionTieneUnSoloGanadorV0(t *testing.T) {
	backend, _ := newGenerationLeaseBackendForTestV0(t)
	marker := generationMarkerForCurrentProcessTestV0(t, backend, "generation-ref-concurrent")
	writeGenerationMarkerTestV0(t, backend.tmuxOwnerMarkerPathV0(), marker)
	first := marker
	first.SocketRef = "socket-ref-first"
	second := marker
	second.SocketRef = "socket-ref-second"
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, next := range []codexAppServerTmuxOwnerMarkerV0{first, second} {
		next := next
		go func() {
			<-start
			results <- backend.replaceTmuxOwnerMarkerV0(marker, next)
		}()
	}
	close(start)
	successes := 0
	conflicts := 0
	for index := 0; index < 2; index++ {
		err := <-results
		if err == nil {
			successes++
			continue
		}
		var callErr codexAppServerCallErrorV0
		if errors.As(err, &callErr) && callErr.Code == codexAppServerTmuxGenerationConflictV0 {
			conflicts++
			continue
		}
		t.Fatalf("resultado CAS inesperado: %v", err)
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("CAS concurrente successes=%d conflicts=%d", successes, conflicts)
	}
}

func TestMarkerCASV0MultiprocesoRealTieneUnSoloGanadorV0(t *testing.T) {
	backend, _ := newGenerationLeaseBackendForTestV0(t)
	marker := generationMarkerForCurrentProcessTestV0(t, backend, "generation-ref-multiprocess")
	writeGenerationMarkerTestV0(t, backend.tmuxOwnerMarkerPathV0(), marker)
	root := filepath.Dir(backend.SocketPath)
	expectedPath := filepath.Join(root, "expected.json")
	raw, err := json.Marshal(marker)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(expectedPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	startPath := filepath.Join(root, "cas.start")
	commands := make([]*exec.Cmd, 0, 2)
	results := make([]string, 0, 2)
	readyPaths := make([]string, 0, 2)
	for index := 0; index < 2; index++ {
		resultPath := filepath.Join(root, fmt.Sprintf("cas-%d.result", index))
		readyPath := filepath.Join(root, fmt.Sprintf("cas-%d.ready", index))
		cmd := exec.Command(os.Args[0], "-test.run=^TestMarkerCASProcessHelperV0$")
		cmd.Env = append(os.Environ(),
			"TEST_MARKER_CAS_HELPER=1",
			"TEST_MARKER_EXPECTED="+expectedPath,
			"TEST_MARKER_READY="+readyPath,
			"TEST_MARKER_START="+startPath,
			fmt.Sprintf("TEST_MARKER_NEXT=socket-ref-process-%d", index),
			"TEST_MARKER_RESULT="+resultPath,
		)
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		commands = append(commands, cmd)
		results = append(results, resultPath)
		readyPaths = append(readyPaths, readyPath)
	}
	waitForMigratedTestConditionV0(t, 2*time.Second, func() bool {
		for _, path := range readyPaths {
			if _, err := os.Stat(path); err != nil {
				return false
			}
		}
		return true
	})
	if err := os.WriteFile(startPath, []byte("start\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, cmd := range commands {
		if err := cmd.Wait(); err != nil {
			t.Fatalf("helper CAS: %v", err)
		}
	}
	counts := map[string]int{}
	for _, path := range results {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		counts[strings.TrimSpace(string(raw))]++
	}
	if counts["success"] != 1 || counts["conflict"] != 1 {
		t.Fatalf("resultados CAS multiproceso=%v", counts)
	}
}

func TestMarkerYLeaseV0SonDisjuntosParaDosSocketsV0(t *testing.T) {
	backend, _ := newGenerationLeaseBackendForTestV0(t)
	other := backend
	other.SocketPath = filepath.Join(filepath.Dir(backend.SocketPath), "other.sock")
	if backend.tmuxOwnerMarkerPathV0() == other.tmuxOwnerMarkerPathV0() || backend.tmuxLeasePathV0() == other.tmuxLeasePathV0() {
		t.Fatal("marker o lease siguen siendo globales al directorio")
	}
	first := generationMarkerForCurrentProcessTestV0(t, backend, "generation-ref-first")
	second := generationMarkerForCurrentProcessTestV0(t, other, "generation-ref-second")
	writeGenerationMarkerTestV0(t, backend.tmuxOwnerMarkerPathV0(), first)
	writeGenerationMarkerTestV0(t, other.tmuxOwnerMarkerPathV0(), second)
	guard1, err := backend.acquireTmuxLeaseGuardV0(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer guard1.releaseV0()
	guard2, err := other.acquireTmuxLeaseGuardV0(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	guard2.releaseV0()
}

func TestMarkerScanV0IncluyeSoloCuarentenasDelOwnerJSONConfiguradoV0(t *testing.T) {
	backend, _ := newGenerationLeaseBackendForTestV0(t)
	quarantine := filepath.Join(filepath.Dir(backend.tmuxOwnerMarkerPathV0()), ".orquesta-quarantine-"+filepath.Base(backend.tmuxOwnerMarkerPathV0())+"-test")
	writeGenerationMarkerTestV0(t, quarantine, generationMarkerForCurrentProcessTestV0(t, backend, "generation-ref-scan"))
	if err := os.WriteFile(filepath.Join(filepath.Dir(quarantine), ".orquesta-quarantine-other.json-test"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	paths := backend.tmuxOwnerMarkerScanPathsV0()
	if len(paths) != 2 || filepath.Clean(paths[0]) != filepath.Clean(backend.SocketPath+".owner.json") || filepath.Clean(paths[1]) != quarantine {
		t.Fatalf("marker paths=%v", paths)
	}
}

func TestCleanupV0DescubreYRestauraMarkerCuarentenadoConSesionVerificadaV0(t *testing.T) {
	backend, tmuxLog := newGenerationLeaseBackendForTestV0(t)
	marker := generationMarkerForCurrentProcessTestV0(t, backend, "generation-ref-quarantined-live")
	marker.AppServerPID = 99999999
	marker.AppServerStartRef = ""
	marker.TmuxSessionID = "$quarantined"
	marker.TmuxSessionCreated = "100"
	writeFakeTmuxStateTestV0(t, tmuxLog, "$quarantined", "100", os.Getpid())
	if err := os.WriteFile(tmuxLog+".generation", []byte(marker.GenerationRef+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	quarantine := filepath.Join(filepath.Dir(backend.tmuxOwnerMarkerPathV0()), ".orquesta-quarantine-"+filepath.Base(backend.tmuxOwnerMarkerPathV0())+"-live")
	writeGenerationMarkerTestV0(t, quarantine, marker)

	active, err := backend.ReadActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkRequestV0{})
	if err != nil || len(active.ActiveWorks) != 1 || !containsStringMigratedTestV0(active.EvidenceRefs, "evidence-ref-codex-app-server-tmux-owner-quarantine") {
		t.Fatalf("active=%+v err=%v", active, err)
	}
	result, err := backend.CleanupActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkCleanupCommandV0{CleanupGoalBackends: true})
	if err != nil || result.CleanedWorkCount != 1 || !containsStringMigratedTestV0(result.EvidenceRefs, "evidence-ref-codex-app-server-tmux-owner-quarantine-restored") {
		t.Fatalf("cleanup=%+v err=%v", result, err)
	}
	active, err = backend.ReadActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkRequestV0{})
	if err != nil || len(active.ActiveWorks) != 0 {
		t.Fatalf("active after cleanup=%+v err=%v", active, err)
	}
}

func TestRollbackLegacyV0NuncaBorraMarkerGeneracionalExactoV0(t *testing.T) {
	backend, _ := newGenerationLeaseBackendForTestV0(t)
	current := generationMarkerForCurrentProcessTestV0(t, backend, "generation-ref-preserved")
	writeGenerationMarkerTestV0(t, backend.tmuxOwnerMarkerPathV0(), current)
	legacy := codexAppServerTmuxOwnerMarkerV0{
		SchemaVersion: codexAppServerTmuxOwnerSchemaV0,
		OwnerRef:      codexAppServerTmuxEvidenceOwnedV0,
		SessionName:   backend.SessionName,
	}
	next := current
	next.GenerationRef = "generation-ref-illegal-legacy-rollback"
	assertGenerationConflictTestV0(t, backend.replaceTmuxOwnerMarkerV0(legacy, next))
	got, ok := backend.readTmuxOwnerMarkerV0()
	if !ok || !reflect.DeepEqual(got, current) {
		t.Fatalf("rollback legacy muto marker nuevo: got=%+v ok=%v", got, ok)
	}
}

func TestEnsureV0NoBorraSocketUnixVivoSinMarkerV0(t *testing.T) {
	backend, tmuxLog := newGenerationLeaseBackendForTestV0(t)
	listener := listenUnixForGenerationTestV0(t, backend.SocketPath)
	defer listener.Close()
	err := backend.EnsureV0(context.Background(), failingCodexAppServerProbeGenerationV0{})
	assertGenerationConflictTestV0(t, err)
	if _, err := os.Lstat(backend.SocketPath); err != nil {
		t.Fatalf("socket vivo borrado: %v", err)
	}
	assertTmuxLogExcludesV0(t, tmuxLog, "new-session", "kill-session")
}

func TestEnsureV0RetiraSocketUnixStaleAntesDeUnaUnicaGeneracionV0(t *testing.T) {
	backend, tmuxLog := newGenerationLeaseBackendForTestV0(t)
	listener := listenUnixForGenerationTestV0(t, backend.SocketPath)
	listener.(*net.UnixListener).SetUnlinkOnClose(false)
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	if err := backend.EnsureV0(context.Background(), fakeCodexAppServerProbeV0{}); err != nil {
		t.Fatalf("EnsureV0 stale: %v", err)
	}
	marker, ok := backend.readTmuxOwnerMarkerV0()
	if !ok || marker.AppServerPID <= 0 || marker.AppServerStartRef == "" || !marker.tmuxIdentityV0().completeV0() {
		t.Fatalf("marker startup incompleto: %+v ok=%v", marker, ok)
	}
	generationRaw, err := os.ReadFile(tmuxLog + ".generation")
	if err != nil || strings.TrimSpace(string(generationRaw)) != marker.GenerationRef {
		t.Fatalf("token generacional no observable en sesion: raw=%q err=%v marker=%+v", generationRaw, err, marker)
	}
	raw, _ := os.ReadFile(tmuxLog)
	if strings.Count(string(raw), "new-session") != 1 {
		t.Fatalf("numero de generaciones=%d log=%s", strings.Count(string(raw), "new-session"), raw)
	}
	if err := backend.ShutdownV0(context.Background()); err != nil {
		t.Fatalf("shutdown test-owned generation: %v", err)
	}
}

func TestRemoveStaleSocketV0DetectaSustitucionAntesDeMutarV0(t *testing.T) {
	backend, _ := newGenerationLeaseBackendForTestV0(t)
	listenerFD, err := listenUnixFDForGenerationTestV0(backend.SocketPath)
	if err != nil {
		if errors.Is(err, syscall.EPERM) {
			t.Skipf("sandbox no permite listen Unix: %v", err)
		}
		t.Fatal(err)
	}
	_ = syscall.Close(listenerFD)
	attackerOld := backend.SocketPath + ".attacker-old"
	backend.beforeSocketQuarantineV0 = func() {
		if err := os.Rename(backend.SocketPath, attackerOld); err != nil {
			t.Fatalf("adversary rename: %v", err)
		}
		replacementFD, err := listenUnixFDForGenerationTestV0(backend.SocketPath)
		if err != nil {
			t.Fatalf("adversary listen: %v", err)
		}
		_ = syscall.Close(replacementFD)
	}
	assertGenerationConflictTestV0(t, backend.removeStaleTmuxSocketV0(context.Background(), codexAppServerTmuxOwnerMarkerV0{}))
	if _, err := os.Lstat(backend.SocketPath); err != nil {
		t.Fatalf("socket sustituto fue mutado: %v", err)
	}
	if _, err := os.Lstat(attackerOld); err != nil {
		t.Fatalf("socket original adversarial desaparecio: %v", err)
	}
}

func TestQuarantineMarkerV0DetectaSustitucionAntesDeRenameV0(t *testing.T) {
	backend, _ := newGenerationLeaseBackendForTestV0(t)
	expected := generationMarkerForCurrentProcessTestV0(t, backend, "generation-ref-quarantine-old")
	writeGenerationMarkerTestV0(t, backend.tmuxOwnerMarkerPathV0(), expected)
	replacement := expected
	replacement.GenerationRef = "generation-ref-quarantine-new"
	oldPath := backend.tmuxOwnerMarkerPathV0() + ".attacker-old"
	backend.beforePathQuarantineV0 = func(path string) {
		if err := os.Rename(path, oldPath); err != nil {
			t.Fatalf("attacker rename marker: %v", err)
		}
		writeGenerationMarkerTestV0(t, path, replacement)
	}
	guard, err := backend.acquireTmuxLeaseGuardV0(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer guard.releaseV0()
	assertGenerationConflictTestV0(t, backend.removeTmuxOwnerMarkerExpectedWithLeaseV0(guard, expected))
	got, ok := backend.readTmuxOwnerMarkerV0()
	if !ok || !reflect.DeepEqual(got, replacement) {
		t.Fatalf("marker sustituto fue mutado: got=%+v ok=%v", got, ok)
	}
}

func TestTmuxKillSessionV0SustitucionConservaSesionNuevaV0(t *testing.T) {
	backend, tmuxLog := newGenerationLeaseBackendForTestV0(t)
	writeFakeTmuxStateTestV0(t, tmuxLog, "$old", "40", os.Getpid())
	tmuxPath, err := codexAppServerTmuxCommandPathV0(backend.PathEnv)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := backend.tmuxSessionIdentityV0(context.Background(), tmuxPath)
	if err != nil {
		t.Fatal(err)
	}
	backend.beforeSessionKillV0 = func() {
		writeFakeTmuxStateTestV0(t, tmuxLog, "$replacement", "41", os.Getpid())
	}
	if err := backend.tmuxKillSessionV0(context.Background(), tmuxPath, expected); err == nil {
		t.Fatal("kill de session_id sustituida no fallo")
	}
	if _, err := os.Stat(tmuxLog + ".session"); err != nil {
		t.Fatalf("sesion sustituta fue eliminada: %v", err)
	}
	raw, _ := os.ReadFile(tmuxLog + ".sid")
	if strings.TrimSpace(string(raw)) != "$replacement" {
		t.Fatalf("session id sustituto=%q", raw)
	}
}

func TestTmuxHasSessionV0NoConfundeExitErrorAjenoConNotFoundV0(t *testing.T) {
	backend, _ := newGenerationLeaseBackendForTestV0(t)
	badTmux := filepath.Join(filepath.Dir(strings.Split(backend.PathEnv, string(os.PathListSeparator))[0]), "tmux-bad")
	if err := os.WriteFile(badTmux, []byte("#!/bin/sh\nprintf '%s\\n' 'permission denied by tmux server' >&2\nexit 1\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if found, err := backend.tmuxHasSessionV0(context.Background(), badTmux); err == nil || found {
		t.Fatalf("ExitError ajeno clasificado como not-found: found=%v err=%v", found, err)
	}
}

func TestTmuxHasSessionV0ReconoceSesionAusenteConTargetPaneExactoV0(t *testing.T) {
	backend, _ := newGenerationLeaseBackendForTestV0(t)
	missingTmux := filepath.Join(filepath.Dir(strings.Split(backend.PathEnv, string(os.PathListSeparator))[0]), "tmux-missing-session")
	if err := os.WriteFile(missingTmux, []byte("#!/bin/sh\nprintf '%s\\n' \"can't find session: "+backend.SessionName+"\" >&2\nexit 1\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if found, err := backend.tmuxHasSessionV0(context.Background(), missingTmux); err != nil || found {
		t.Fatalf("sesion tmux ausente no reconocida: found=%v err=%v", found, err)
	}
}

func TestTmuxHasSessionV0ReconoceSocketDeServidorAusenteV0(t *testing.T) {
	backend, _ := newGenerationLeaseBackendForTestV0(t)
	missingTmux := filepath.Join(filepath.Dir(strings.Split(backend.PathEnv, string(os.PathListSeparator))[0]), "tmux-missing-server")
	if err := os.WriteFile(missingTmux, []byte("#!/bin/sh\nprintf '%s\\n' 'error connecting to /tmp/tmux-test/default (No such file or directory)' >&2\nexit 1\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if found, err := backend.tmuxHasSessionV0(context.Background(), missingTmux); err != nil || found {
		t.Fatalf("servidor tmux ausente no reconocido: found=%v err=%v", found, err)
	}
}

func TestEnsureV0LegacyAmbiguoEsConservadorV0(t *testing.T) {
	backend, tmuxLog := newGenerationLeaseBackendForTestV0(t)
	listener := listenUnixForGenerationTestV0(t, backend.SocketPath)
	defer listener.Close()
	legacy := codexAppServerTmuxOwnerMarkerV0{
		SchemaVersion: codexAppServerTmuxOwnerSchemaV0, OwnerRef: codexAppServerTmuxEvidenceOwnedV0,
		SessionName: backend.SessionName, SocketRef: "socket-ref-codex-goal-app-server-tmux",
	}
	writeGenerationMarkerTestV0(t, backend.tmuxLegacyOwnerMarkerPathV0(), legacy)
	err := backend.EnsureV0(context.Background(), fakeCodexAppServerProbeV0{})
	assertGenerationConflictTestV0(t, err)
	if _, err := os.Lstat(backend.tmuxLegacyOwnerMarkerPathV0()); err != nil {
		t.Fatalf("legacy marker borrado: %v", err)
	}
	assertTmuxLogExcludesV0(t, tmuxLog, "new-session", "kill-session")
}

func TestCleanupGeneracionV0CASStaleMismaGenerationNoMutaV0(t *testing.T) {
	backend, tmuxLog := newGenerationLeaseBackendForTestV0(t)
	listener := listenUnixForGenerationTestV0(t, backend.SocketPath)
	defer listener.Close()
	current := generationMarkerForCurrentProcessTestV0(t, backend, "generation-ref-current")
	writeGenerationMarkerTestV0(t, backend.tmuxOwnerMarkerPathV0(), current)
	stale := current
	stale.LeaseOwnerStartRef += "-stale"
	assertGenerationConflictTestV0(t, backend.cleanupTmuxGenerationV0(context.Background(), stale, true))
	assertTmuxLogExcludesV0(t, tmuxLog, "kill-session")
	if _, err := os.Lstat(backend.SocketPath); err != nil {
		t.Fatalf("cleanup CAS stale borro socket: %v", err)
	}
}

func newGenerationLeaseBackendForTestV0(t *testing.T) (serverCodexAppServerTmuxBackendV0, string) {
	t.Helper()
	root := shortUnixSocketTestRootV0(t)
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		t.Fatal(err)
	}
	tmuxLog := filepath.Join(root, "tmux.log")
	tmuxPath := filepath.Join(binDir, "tmux")
	if err := os.WriteFile(tmuxPath, []byte(fakeGenerationLeaseTmuxScriptV0()), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ORQUESTA_TEST_TMUX_LOG", tmuxLog)
	t.Setenv("ORQUESTA_TEST_BINARY", os.Args[0])
	runtimeDir := filepath.Join(root, "runtime")
	socketPath := filepath.Join(runtimeDir, codexAppServerTmuxDirV0, "s.sock")
	if err := os.MkdirAll(filepath.Dir(socketPath), 0o700); err != nil {
		t.Fatal(err)
	}
	return serverCodexAppServerTmuxBackendV0{
		PathEnv: binDir + string(os.PathListSeparator) + os.Getenv("PATH"), SocketPath: socketPath,
		SessionName: "orquesta-goal-generation-1234567890", RuntimeWorkDir: runtimeDir, Timeout: 2 * time.Second,
	}, tmuxLog
}

func fakeGenerationLeaseTmuxScriptV0() string {
	return `#!/bin/sh
set -eu
log="$ORQUESTA_TEST_TMUX_LOG"
printf '%s\n' "$*" >> "$log"
case "${1:-}" in
  has-session)
    if [ -f "$log.session" ]; then exit 0; fi
    target="${3:-}"
    printf "can't find session: %s\n" "${target#=}" >&2
    exit 1
    ;;
  display-message)
    if [ "${ORQUESTA_TEST_TMUX_INVALID_IDENTITY:-}" = 1 ]; then printf '\t\t0\n'; exit 0; fi
    printf '%s\t%s\t%s\n' "$(cat "$log.sid")" "$(cat "$log.created")" "$(cat "$log.pid")"
    ;;
  show-environment)
    printf '%s=%s\n' 'ORQUESTA_CODEX_APP_SERVER_GENERATION_REF' "$(cat "$log.generation")"
    ;;
  kill-session)
	    target="${3:-}"
	    if [ ! -f "$log.sid" ] || [ "$target" != "$(cat "$log.sid")" ]; then
	      printf "can't find session: %s\n" "$target" >&2
	      exit 1
	    fi
	    if [ -f "$log.helper" ] && [ -f "$log.pid" ]; then kill "$(cat "$log.pid")" 2>/dev/null || true; fi
	    rm -f "$log.session" "$log.helper"
    ;;
  new-session)
	    if [ "${ORQUESTA_TEST_TMUX_INVALID_IDENTITY:-}" = 1 ]; then
	      : > "$log.session"
	      exit 0
	    fi
	    sock=""
    generation=""
    for arg in "$@"; do
      case "$arg" in
        *unix://*) sock="${arg#*unix://}"; sock="${sock%%\'*}"; sock="${sock%%\"*}"; sock="${sock%% *}" ;;
        ORQUESTA_CODEX_APP_SERVER_GENERATION_REF=*) generation="${arg#*=}" ;;
      esac
    done
    test -n "$sock"
    ORQUESTA_TEST_UNIX_SERVER_HELPER=1 "$ORQUESTA_TEST_BINARY" -test.run=TestGenerationLeaseUnixServerHelperV0 -- "$sock" </dev/null >/dev/null 2>&1 &
    pid=$!
    printf '%s\n' "$pid" > "$log.pid"
    printf '%s\n' '$test' > "$log.sid"
	    printf '%s\n' '100' > "$log.created"
	    printf '%s\n' "$generation" > "$log.generation"
	    : > "$log.session"
	    : > "$log.helper"
    i=0; while [ ! -S "$sock" ] && [ "$i" -lt 100 ]; do i=$((i+1)); sleep 0.01; done
    test -S "$sock"
	    printf '%s\t%s\t%s\n' '$test' '100' "$pid"
    ;;
  *) exit 2 ;;
esac
`
}

func listenUnixForGenerationTestV0(t *testing.T, socketPath string) net.Listener {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(socketPath), 0o700); err != nil {
		t.Fatalf("crear padre de socket Unix temporal: %v", err)
	}
	if err := validateUnixSocketPathLengthV0(socketPath); err != nil {
		t.Fatalf("ruta socket Unix: %v", err)
	}
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "operation not permitted") {
			t.Skipf("unix sockets no disponibles en este sandbox: %v", err)
		}
		t.Fatalf("listen unix: %v", err)
	}
	return listener
}

func generationMarkerForCurrentProcessTestV0(t *testing.T, backend serverCodexAppServerTmuxBackendV0, generationRef string) codexAppServerTmuxOwnerMarkerV0 {
	t.Helper()
	marker, err := backend.newTmuxOwnerMarkerV0(generationRef, os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	marker.SocketOwnerPID = os.Getpid()
	marker.SocketOwnerStartRef = codexAppServerTmuxProcessStartRefV0(os.Getpid())
	marker.TmuxPanePID = os.Getpid()
	marker.TmuxPaneStartRef = marker.SocketOwnerStartRef
	return marker
}

func writeGenerationMarkerTestV0(t *testing.T, path string, marker codexAppServerTmuxOwnerMarkerV0) {
	t.Helper()
	raw, err := json.Marshal(marker)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeFakeTmuxStateTestV0(t *testing.T, logPath, sessionID, created string, panePID int) {
	t.Helper()
	for suffix, value := range map[string]string{
		".session": "", ".sid": sessionID, ".created": created, ".pid": strconv.Itoa(panePID),
	} {
		if err := os.WriteFile(logPath+suffix, []byte(value), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

type procUnixRowFixtureV0 struct {
	inode      string
	socketPath string
	flags      string
	socketType string
	state      string
}

func writeProcUnixFixtureV0(t *testing.T, procRoot, inode, socketPath string) {
	t.Helper()
	writeProcUnixRowsFixtureV0(t, procRoot, procUnixRowFixtureV0{
		inode: inode, socketPath: socketPath, flags: "00010000", socketType: "0001", state: "01",
	})
}

func writeProcUnixRowsFixtureV0(t *testing.T, procRoot string, rows ...procUnixRowFixtureV0) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(procRoot, "net"), 0o700); err != nil {
		t.Fatal(err)
	}
	body := "Num RefCount Protocol Flags Type St Inode Path\n"
	for _, row := range rows {
		body += fmt.Sprintf("00000000: 00000002 00000000 %s %s %s %s %s\n", row.flags, row.socketType, row.state, row.inode, row.socketPath)
	}
	if err := os.WriteFile(filepath.Join(procRoot, "net", "unix"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeProcProcessFixtureV0(t *testing.T, procRoot string, pid, parent int, startRef, socketLink string) {
	t.Helper()
	dir := filepath.Join(procRoot, strconv.Itoa(pid))
	if err := os.MkdirAll(filepath.Join(dir, "fd"), 0o700); err != nil {
		t.Fatal(err)
	}
	fields := make([]string, 20)
	for index := range fields {
		fields[index] = "0"
	}
	fields[0] = "S"
	fields[1] = strconv.Itoa(parent)
	fields[19] = startRef
	body := fmt.Sprintf("%d (fixture) %s\n", pid, strings.Join(fields, " "))
	if err := os.WriteFile(filepath.Join(dir, "stat"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if socketLink != "" {
		if err := os.Symlink(socketLink, filepath.Join(dir, "fd", "7")); err != nil {
			t.Fatal(err)
		}
	}
}

func assertGenerationConflictTestV0(t *testing.T, err error) {
	t.Helper()
	var callErr codexAppServerCallErrorV0
	if !errors.As(err, &callErr) || callErr.Code != codexAppServerTmuxGenerationConflictV0 {
		t.Fatalf("error=%v; esperado conflicto de generacion", err)
	}
}

func assertTmuxLogExcludesV0(t *testing.T, path string, forbidden ...string) {
	t.Helper()
	raw, _ := os.ReadFile(path)
	for _, value := range forbidden {
		if strings.Contains(string(raw), value) {
			t.Fatalf("log contiene %q: %s", value, raw)
		}
	}
}
