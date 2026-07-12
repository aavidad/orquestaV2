package orquestaruntimecodexappserver

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestCodexAppServerWebSocketProtocolV0ProbeRealOptInV0(t *testing.T) {
	command := os.Getenv("ORQUESTA_TEST_CODEX_APP_SERVER_COMMAND")
	if command == "" {
		t.Skip("requiere ORQUESTA_TEST_CODEX_APP_SERVER_COMMAND")
	}
	root := t.TempDir()
	socketPath := filepath.Join(root, "app-server.sock")
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, command, "app-server", "--listen", "unix://"+socketPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer stdin.Close()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()
	for !codexAppServerTmuxSocketPresentV0(socketPath) {
		select {
		case <-ctx.Done():
			t.Fatalf("socket no disponible: %v stderr=%s", ctx.Err(), stderr.String())
		case <-time.After(25 * time.Millisecond):
		}
	}
	probeCtx, probeCancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer probeCancel()
	protocol := serverCodexAppServerWebSocketProtocolV0{
		SocketPath:        socketPath,
		Timeout:           5 * time.Second,
		DiagnosticLogPath: filepath.Join(root, "missing-diagnostic.log"),
	}
	if err := protocol.ProbeV0(probeCtx); err != nil {
		t.Fatalf("probe real: %T %v stderr=%s", err, err, stderr.String())
	}
}
