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

	configtoml "orquesta/internal/adapters/config/toml"
	credentiallocal "orquesta/internal/adapters/credentials/local"
	"orquesta/internal/application"
	"orquesta/internal/config"
)

func TestBuildRejectsAndClosesInjectedNonLoopbackListener(t *testing.T) {
	root := t.TempDir()
	listener, err := net.Listen("tcp4", "0.0.0.0:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	tcpListener := listener.(*net.TCPListener)
	if err := tcpListener.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set listener deadline: %v", err)
	}

	var factoryCalls atomic.Int64
	runtime, err := Build(context.Background(), Options{
		ConfigPath: writeTestConfig(t, root),
		Listener:   listener,
		AgentFactory: func(snapshot config.Snapshot, clock application.Clock) (AgentAdapter, error) {
			factoryCalls.Add(1)
			return countingFactory(&atomic.Int64{})(snapshot, clock)
		},
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
	if factoryCalls.Load() != 0 {
		t.Fatalf("invalid listener reached agent factory %d times", factoryCalls.Load())
	}
	assertNoCompositionState(t, root)
}

func TestBuildAgentFactoryFailurePrecedesDurableCompositionState(t *testing.T) {
	root := t.TempDir()
	want := errors.New("factory_preflight_failed")
	runtime, err := Build(context.Background(), Options{
		ConfigPath: writeTestConfig(t, root),
		AgentFactory: func(config.Snapshot, application.Clock) (AgentAdapter, error) {
			return nil, want
		},
	})
	if runtime != nil || !errors.Is(err, want) {
		t.Fatalf("factory failure = %v, %v", runtime, err)
	}
	assertNoCompositionState(t, root)
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

func TestBuildRequiresExplicitConfigWithoutCreatingFallbackSidecars(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "missing.toml")
	runtime, err := Build(context.Background(), Options{
		ConfigPath: path, AgentFactory: countingFactory(&atomic.Int64{}),
	})
	if runtime != nil || !config.IsDocumentStoreError(err, config.DocumentStoreSourceRequired) {
		t.Fatalf("explicit missing config = %v, %v", runtime, err)
	}
	for _, sidecar := range append([]string{path}, configtoml.ReservedPaths(path)...) {
		if _, statErr := os.Lstat(sidecar); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("missing config created %s: %v", sidecar, statErr)
		}
	}
}

func TestBuildRejectsConfigStoreReservedPathCollisionsBeforeEffects(t *testing.T) {
	for _, suffix := range []string{".lock", ".next", ".receipts"} {
		t.Run(suffix, func(t *testing.T) {
			root := t.TempDir()
			configPath := writeTestConfig(t, root)
			replaceTestConfigValue(t, configPath,
				"effective_path = "+strconv.Quote(filepath.Join(root, "effective_config.json")),
				"effective_path = "+strconv.Quote(configPath+suffix),
			)
			runtime, err := Build(context.Background(), Options{
				ConfigPath: configPath, AgentFactory: countingFactory(&atomic.Int64{}),
			})
			if runtime != nil || err == nil || err.Error() != "bootstrap.runtime_paths_overlap" {
				t.Fatalf("reserved collision = %v, %v", runtime, err)
			}
			if _, statErr := os.Lstat(filepath.Join(root, "secrets", "local-owner.token")); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("reserved collision created token: %v", statErr)
			}
		})
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

func TestBuildRejectsCredentialRecoveryNamespaceCollisionsBeforeEffects(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*testing.T, string, string, string) string
	}{
		{name: "state directory", configure: func(t *testing.T, root, configPath, reservedPath string) string {
			replaceTestConfigValue(t, configPath,
				"path = "+strconv.Quote(filepath.Join(root, "state", "orquesta.sqlite")),
				"path = "+strconv.Quote(filepath.Join(reservedPath, "orquesta.sqlite")),
			)
			return configPath
		}},
		{name: "artifact root", configure: func(t *testing.T, root, configPath, reservedPath string) string {
			replaceTestConfigValue(t, configPath,
				"root = "+strconv.Quote(filepath.Join(root, "artifacts")),
				"root = "+strconv.Quote(reservedPath),
			)
			return configPath
		}},
		{name: "Codex work root", configure: func(t *testing.T, root, configPath, reservedPath string) string {
			replaceTestConfigValue(t, configPath,
				"work_root = "+strconv.Quote(filepath.Join(root, "work")),
				"work_root = "+strconv.Quote(reservedPath),
			)
			return configPath
		}},
		{name: "effective config", configure: func(t *testing.T, root, configPath, reservedPath string) string {
			replaceTestConfigValue(t, configPath,
				"effective_path = "+strconv.Quote(filepath.Join(root, "effective_config.json")),
				"effective_path = "+strconv.Quote(reservedPath),
			)
			return configPath
		}},
		{name: "local token", configure: func(t *testing.T, root, configPath, reservedPath string) string {
			replaceTestConfigValue(t, configPath,
				"local_token_path = "+strconv.Quote(filepath.Join(root, "secrets", "local-owner.token")),
				"local_token_path = "+strconv.Quote(reservedPath),
			)
			return configPath
		}},
		{name: "config source", configure: func(t *testing.T, _ string, configPath, reservedPath string) string {
			if err := os.MkdirAll(filepath.Dir(reservedPath), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Rename(configPath, reservedPath); err != nil {
				t.Fatal(err)
			}
			return reservedPath
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			configPath := writeTestConfig(t, root)
			credentialPath := filepath.Join(root, "credential-store", "credentials.json")
			reservedPath := credentiallocal.ReservedPaths(credentialPath)[0]
			replaceTestConfigValue(t, configPath,
				"path = "+strconv.Quote(filepath.Join(root, "secrets", "credentials.json")),
				"path = "+strconv.Quote(credentialPath),
			)
			configPath = test.configure(t, root, configPath, reservedPath)

			var factoryCalls atomic.Int64
			runtime, err := Build(context.Background(), Options{
				ConfigPath: configPath,
				AgentFactory: func(snapshot config.Snapshot, clock application.Clock) (AgentAdapter, error) {
					factoryCalls.Add(1)
					return countingFactory(&atomic.Int64{})(snapshot, clock)
				},
			})
			if runtime != nil || err == nil || err.Error() != "bootstrap.runtime_paths_overlap" {
				t.Fatalf("credential reserved path collision = %v, %v", runtime, err)
			}
			if factoryCalls.Load() != 0 {
				t.Fatalf("credential reserved path collision reached agent factory %d times", factoryCalls.Load())
			}
			for _, path := range []string{
				filepath.Join(root, "state"), filepath.Join(root, "artifacts"), filepath.Join(root, "work"),
				filepath.Join(root, "effective_config.json"), filepath.Join(root, "secrets", "local-owner.token"),
			} {
				if _, statErr := os.Lstat(path); !errors.Is(statErr, os.ErrNotExist) {
					t.Fatalf("credential reserved path collision created %s: %v", path, statErr)
				}
			}
		})
	}
}

func TestBuildRejectsCredentialRecoveryNamespaceAliasBeforeEffects(t *testing.T) {
	root := t.TempDir()
	configPath := writeTestConfig(t, root)
	credentialPath := filepath.Join(root, "credential-store", "credentials.json")
	reservedPath := credentiallocal.ReservedPaths(credentialPath)[0]
	replaceTestConfigValue(t, configPath,
		"path = "+strconv.Quote(filepath.Join(root, "secrets", "credentials.json")),
		"path = "+strconv.Quote(credentialPath),
	)
	artifactRoot := filepath.Join(root, "artifacts")
	if err := os.MkdirAll(filepath.Dir(reservedPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(artifactRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(artifactRoot, reservedPath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	var factoryCalls atomic.Int64
	runtime, err := Build(context.Background(), Options{
		ConfigPath: configPath,
		AgentFactory: func(snapshot config.Snapshot, clock application.Clock) (AgentAdapter, error) {
			factoryCalls.Add(1)
			return countingFactory(&atomic.Int64{})(snapshot, clock)
		},
	})
	if runtime != nil || err == nil || err.Error() != "bootstrap.runtime_paths_overlap" {
		t.Fatalf("credential reserved path alias = %v, %v", runtime, err)
	}
	if factoryCalls.Load() != 0 {
		t.Fatalf("credential reserved path alias reached agent factory %d times", factoryCalls.Load())
	}
	for _, path := range []string{
		filepath.Join(root, "state"), filepath.Join(root, "work"), filepath.Join(root, "effective_config.json"),
		filepath.Join(root, "secrets", "local-owner.token"),
	} {
		if _, statErr := os.Lstat(path); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("credential reserved path alias created %s: %v", path, statErr)
		}
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

func assertNoCompositionState(t *testing.T, root string) {
	t.Helper()
	for _, relative := range []string{
		"state", "artifacts", "work", "secrets", "effective_config.json",
	} {
		path := filepath.Join(root, relative)
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("failed preflight created %s: %v", path, err)
		}
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
