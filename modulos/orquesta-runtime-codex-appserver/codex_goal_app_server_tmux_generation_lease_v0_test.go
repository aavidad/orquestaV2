package orquestaruntimecodexappserver

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

type failingCodexAppServerProbeGenerationV0 struct{}

func (failingCodexAppServerProbeGenerationV0) ProbeV0(context.Context) error {
	return errors.New("probe unavailable")
}

func TestGenerationLeaseUnixServerHelperV0(t *testing.T) {
	if os.Getenv("ORQUESTA_TEST_UNIX_SERVER_HELPER") != "1" {
		return
	}
	args := os.Args
	if len(args) == 0 {
		os.Exit(2)
	}
	socketPath := args[len(args)-1]
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		os.Exit(3)
	}
	defer listener.Close()
	for {
		connection, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		_ = connection.Close()
	}
}

func TestShortUnixSocketTestRootV0AcotaSockaddrV0(t *testing.T) {
	root := shortUnixSocketTestRootV0(t)
	socketPath := filepath.Join(root, "runtime", codexAppServerTmuxDirV0, "codex-app-server.sock")
	if len(socketPath) >= 100 {
		t.Fatalf("ruta Unix de test sigue siendo larga: len=%d path=%s", len(socketPath), socketPath)
	}
	if err := os.MkdirAll(filepath.Dir(socketPath), 0o700); err != nil {
		t.Fatal(err)
	}
}

func TestEnsureV0AdoptaGeneracionExactaCuandoTmuxDesapareceV0(t *testing.T) {
	backend, tmuxLog := newGenerationLeaseBackendForTestV0(t)
	listener := listenUnixForGenerationTestV0(t, backend.SocketPath)
	defer listener.Close()
	marker := generationMarkerForCurrentProcessTestV0(t, backend, "generation-ref-surviving")
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
	if _, err := backend.tmuxSessionIdentityV0(context.Background(), tmuxPath); err != nil {
		t.Fatal(err)
	}
	if err := backend.tmuxKillSessionV0(context.Background(), tmuxPath); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(tmuxLog)
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if strings.Contains(line, "has-session") || strings.Contains(line, "display-message") || strings.Contains(line, "kill-session") {
			if !strings.Contains(line, "-t ="+backend.SessionName) {
				t.Fatalf("target tmux no exacto: %q", line)
			}
		}
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
	raw, _ := os.ReadFile(tmuxLog)
	if strings.Count(string(raw), "new-session") != 1 {
		t.Fatalf("numero de generaciones=%d log=%s", strings.Count(string(raw), "new-session"), raw)
	}
	if err := backend.ShutdownV0(context.Background()); err != nil {
		t.Fatalf("shutdown test-owned generation: %v", err)
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

func shortUnixSocketTestRootV0(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if len(filepath.Join(root, "runtime", codexAppServerTmuxDirV0, "codex-app-server.sock")) < 100 {
		return root
	}
	aliasFile, err := os.CreateTemp("/tmp", "oq-gl-")
	if err != nil {
		t.Fatalf("crear alias corto para socket Unix: %v", err)
	}
	aliasPath := aliasFile.Name()
	if err := aliasFile.Close(); err != nil {
		t.Fatalf("cerrar reserva de alias corto: %v", err)
	}
	if err := os.Remove(aliasPath); err != nil {
		t.Fatalf("retirar reserva de alias corto: %v", err)
	}
	if err := os.Symlink(root, aliasPath); err != nil {
		t.Fatalf("crear alias corto para t.TempDir: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(aliasPath) })
	return aliasPath
}

func fakeGenerationLeaseTmuxScriptV0() string {
	return `#!/bin/sh
set -eu
log="$ORQUESTA_TEST_TMUX_LOG"
printf '%s\n' "$*" >> "$log"
case "${1:-}" in
  has-session) test -f "$log.session" ;;
  display-message)
    if [ "${ORQUESTA_TEST_TMUX_INVALID_IDENTITY:-}" = 1 ]; then printf '\t\t0\n'; exit 0; fi
    printf '%s\t%s\t%s\n' "$(cat "$log.sid")" "$(cat "$log.created")" "$(cat "$log.pid")"
    ;;
  kill-session)
	    if [ -f "$log.helper" ] && [ -f "$log.pid" ]; then kill "$(cat "$log.pid")" 2>/dev/null || true; fi
	    rm -f "$log.session" "$log.helper"
    ;;
  new-session)
	    if [ "${ORQUESTA_TEST_TMUX_INVALID_IDENTITY:-}" = 1 ]; then
	      : > "$log.session"
	      exit 0
	    fi
	    sock=""
    for arg in "$@"; do case "$arg" in *unix://*) sock="${arg#*unix://}"; sock="${sock%%\'*}"; sock="${sock%%\"*}"; sock="${sock%% *}" ;; esac; done
    test -n "$sock"
    ORQUESTA_TEST_UNIX_SERVER_HELPER=1 "$ORQUESTA_TEST_BINARY" -test.run=TestGenerationLeaseUnixServerHelperV0 -- "$sock" </dev/null >/dev/null 2>&1 &
    pid=$!
    printf '%s\n' "$pid" > "$log.pid"
    printf '%s\n' '$test' > "$log.sid"
	    printf '%s\n' '100' > "$log.created"
	    : > "$log.session"
	    : > "$log.helper"
    i=0; while [ ! -S "$sock" ] && [ "$i" -lt 100 ]; do i=$((i+1)); sleep 0.01; done
    test -S "$sock"
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
