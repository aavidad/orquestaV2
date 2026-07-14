package bootstrap

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/config"
)

func TestBuildRejectsAndClosesInjectedNonLoopbackListener(t *testing.T) {
	listener, err := net.Listen("tcp4", "0.0.0.0:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	tcpListener := listener.(*net.TCPListener)
	if err := tcpListener.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set listener deadline: %v", err)
	}

	runtime, err := Build(context.Background(), Options{
		ConfigPath:   writeTestConfig(t, t.TempDir()),
		Listener:     listener,
		AgentFactory: countingFactory(&atomic.Int64{}),
	})
	if runtime != nil {
		t.Fatalf("expected no runtime, got %#v", runtime)
	}
	if err == nil || err.Error() != "bootstrap.listener_must_be_loopback" {
		t.Fatalf("expected loopback rejection, got %v", err)
	}

	connection, acceptErr := listener.Accept()
	if connection != nil {
		_ = connection.Close()
	}
	if !errors.Is(acceptErr, net.ErrClosed) {
		t.Fatalf("expected injected listener to be closed, got %v", acceptErr)
	}
}

func TestRuntimeRejectsStartAfterShutdown(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	runtime, err := Build(context.Background(), Options{
		ConfigPath:   writeTestConfig(t, t.TempDir()),
		Listener:     listener,
		AgentFactory: countingFactory(&atomic.Int64{}),
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = runtime.Shutdown(ctx)
	})

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := runtime.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if err := runtime.Start(context.Background()); err == nil || err.Error() != "bootstrap.runtime_stopped" {
		t.Fatalf("expected stopped runtime rejection, got %v", err)
	}
}

func TestRuntimeWaitReturnsSameServeErrorToMultipleWaiters(t *testing.T) {
	serveFailure := errors.New("runtime_edge.accept_failed")
	listener := &runtimeEdgeFailingListener{err: serveFailure}
	runtime, err := Build(context.Background(), Options{
		ConfigPath:   writeTestConfig(t, t.TempDir()),
		Listener:     listener,
		AgentFactory: countingFactory(&atomic.Int64{}),
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = runtime.Shutdown(ctx)
	})
	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}

	results := make(chan error, 2)
	for range 2 {
		go func() {
			results <- runtime.Wait()
		}()
	}
	for waiter := 0; waiter < 2; waiter++ {
		select {
		case waitErr := <-results:
			if !errors.Is(waitErr, serveFailure) {
				t.Fatalf("waiter %d: expected serve failure, got %v", waiter, waitErr)
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("waiter %d did not return", waiter)
		}
	}
}

func TestBuildRejectsNilAgentFactoryResult(t *testing.T) {
	runtime, err := Build(context.Background(), Options{
		ConfigPath: writeTestConfig(t, t.TempDir()),
		AgentFactory: func(config.Snapshot, application.Clock) (AgentAdapter, error) {
			return nil, nil
		},
	})
	if runtime != nil {
		t.Fatalf("expected no runtime, got %#v", runtime)
	}
	if err == nil || err.Error() != "bootstrap.agent_factory_returned_nil" {
		t.Fatalf("expected nil agent rejection, got %v", err)
	}
}

func TestBuildRejectsServeMuxPatternSyntaxBeforeCreatingRuntimeState(t *testing.T) {
	root := t.TempDir()
	configPath := writeTestConfig(t, root)
	replaceTestConfigValue(t, configPath, "[server]\n", "[server]\nmcp_path = \"/{\"\n")

	runtime, err := Build(context.Background(), Options{
		ConfigPath: configPath, AgentFactory: countingFactory(&atomic.Int64{}),
	})
	if runtime != nil || !config.HasErrorCode(err, config.ErrorCrossValidation) {
		t.Fatalf("expected literal MCP path rejection, runtime=%v err=%v", runtime, err)
	}
	if _, statErr := os.Lstat(filepath.Join(root, "secrets", "local-owner.token")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("invalid MCP path created runtime state: %v", statErr)
	}
}

func TestBuildRejectsEffectiveStateCollisionWithoutOverwriting(t *testing.T) {
	root := t.TempDir()
	configPath := writeTestConfig(t, root)
	statePath := filepath.Join(root, "state", "orquesta.sqlite")
	if err := os.MkdirAll(filepath.Dir(statePath), 0o700); err != nil {
		t.Fatalf("mkdir state: %v", err)
	}
	sentinel := []byte("existing-state-must-survive")
	if err := os.WriteFile(statePath, sentinel, 0o600); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}
	replaceTestConfigValue(t, configPath,
		"effective_path = "+strconv.Quote(filepath.Join(root, "effective_config.json")),
		"effective_path = "+strconv.Quote(statePath),
	)

	runtime, err := Build(context.Background(), Options{
		ConfigPath: configPath, AgentFactory: countingFactory(&atomic.Int64{}),
	})
	if runtime != nil || !config.HasErrorCode(err, config.ErrorCrossValidation) {
		t.Fatalf("expected path collision, runtime=%v err=%v", runtime, err)
	}
	got, readErr := os.ReadFile(statePath)
	if readErr != nil || string(got) != string(sentinel) {
		t.Fatalf("state sentinel changed: content=%q err=%v", got, readErr)
	}
}

func TestBuildRejectsEffectiveConfigSourceCollisionWithoutOverwriting(t *testing.T) {
	root := t.TempDir()
	configPath := writeTestConfig(t, root)
	replaceTestConfigValue(t, configPath,
		"effective_path = "+strconv.Quote(filepath.Join(root, "effective_config.json")),
		"effective_path = "+strconv.Quote(configPath),
	)
	want, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config fixture: %v", err)
	}

	runtime, buildErr := Build(context.Background(), Options{
		ConfigPath: configPath, AgentFactory: countingFactory(&atomic.Int64{}),
	})
	if runtime != nil || !config.HasErrorCode(buildErr, config.ErrorCrossValidation) {
		t.Fatalf("expected source collision, runtime=%v err=%v", runtime, buildErr)
	}
	got, readErr := os.ReadFile(configPath)
	if readErr != nil || string(got) != string(want) {
		t.Fatalf("source config changed: err=%v", readErr)
	}
}

func TestBuildDetectsRuntimeRootAliasesThroughSymlinkParent(t *testing.T) {
	root := t.TempDir()
	configPath := writeTestConfig(t, root)
	shared := filepath.Join(root, "shared")
	if err := os.Mkdir(shared, 0o700); err != nil {
		t.Fatalf("mkdir shared: %v", err)
	}
	alias := filepath.Join(root, "shared-alias")
	if err := os.Symlink(shared, alias); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	replaceTestConfigValue(t, configPath,
		"path = "+strconv.Quote(filepath.Join(root, "state", "orquesta.sqlite")),
		"path = "+strconv.Quote(filepath.Join(shared, "orquesta.sqlite")),
	)
	replaceTestConfigValue(t, configPath,
		"root = "+strconv.Quote(filepath.Join(root, "artifacts")),
		"root = "+strconv.Quote(alias),
	)

	runtime, err := Build(context.Background(), Options{
		ConfigPath: configPath, AgentFactory: countingFactory(&atomic.Int64{}),
	})
	if runtime != nil || err == nil || err.Error() != "bootstrap.runtime_paths_overlap" {
		t.Fatalf("expected canonical alias collision, runtime=%v err=%v", runtime, err)
	}
}

func replaceTestConfigValue(t *testing.T, path, oldValue, newValue string) {
	t.Helper()
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	updated := strings.Replace(string(payload), oldValue, newValue, 1)
	if updated == string(payload) {
		t.Fatalf("config value not found: %s", oldValue)
	}
	if err := os.WriteFile(path, []byte(updated), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

type runtimeEdgeFailingListener struct {
	err error
}

func (listener *runtimeEdgeFailingListener) Accept() (net.Conn, error) {
	return nil, listener.err
}

func (listener *runtimeEdgeFailingListener) Close() error {
	return nil
}

func (listener *runtimeEdgeFailingListener) Addr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 43210}
}
