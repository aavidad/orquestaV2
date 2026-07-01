package orquestaserver

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRuntimeV0PublicaBaseURLFileEnRuntimeWorkDirV0(t *testing.T) {
	root := t.TempDir()
	runtimeDir := filepath.Join(root, "runtime")
	runtime, err := NewRuntimeV0(ConfigV0{
		Addr:           "127.0.0.1:0",
		StateDir:       filepath.Join(runtimeDir, "state"),
		RuntimeWorkDir: runtimeDir,
		TickInterval:   time.Hour,
		AuditDisabled:  true,
	}, RuntimeDepsV0{
		AppHandler: http.NotFoundHandler(),
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runtime.RunV0(ctx) }()
	addr := waitRuntimeAddrForTestV0(t, runtime)

	data, err := os.ReadFile(ServerBaseURLPathV0(runtime.config))
	if err != nil {
		cancel()
		t.Fatalf("ReadFile base_url: %v", err)
	}
	if got, want := strings.TrimSpace(string(data)), "http://"+addr; got != want {
		cancel()
		t.Fatalf("base_url=%q want %q", got, want)
	}
	if info, err := os.Stat(ServerBaseURLPathV0(runtime.config)); err != nil {
		cancel()
		t.Fatalf("Stat base_url: %v", err)
	} else if info.Mode().Perm() != 0o600 {
		cancel()
		t.Fatalf("base_url perm=%v", info.Mode().Perm())
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RunV0: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("runtime no cerro")
	}
}

func TestServerBaseURLPathV0VacioSinRuntimeWorkDirV0(t *testing.T) {
	if got := ServerBaseURLPathV0(ConfigV0{}); got != "" {
		t.Fatalf("path=%q", got)
	}
}
