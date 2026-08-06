//go:build linux

package agentmicrovm

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"
	"golang.org/x/sys/unix"
)

func TestCredentialBrokerServerPublishesPrivateSocketAndServesOneExchange(t *testing.T) {
	server, fixture, config := newCredentialBrokerServerFixture(t, 2)
	if err := server.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	assertPrivateBrokerSocket(t, config.SocketPath, config.OwnerUID)

	connection := dialCredentialBroker(t, config.SocketPath)
	writeAperture(t, connection, fixture.aperture)
	writeRequest(t, connection, fixture.request)
	header, material := readBrokerResponse(t, connection)
	if header.Estado != microvm.EstadoRespuestaAuthJSONCodexDisponible || string(material) != "secret-marker" {
		t.Fatalf("response=%#v material=%q", header, material)
	}
	writeAck(t, connection, fixture.ack(microvm.EtapaAcuseAuthJSONCodexProyectada, true))
	writeAck(t, connection, fixture.ack(microvm.EtapaAcuseAuthJSONCodexPurgada, true))
	if err := connection.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	var trailing [1]byte
	if count, err := connection.Read(trailing[:]); count != 0 || !errors.Is(err, io.EOF) {
		t.Fatalf("connection accepted more than one exchange: count=%d err=%v", count, err)
	}
	_ = connection.Close()
	if fixture.store.callbackCalls() != 1 {
		t.Fatalf("callback calls=%d", fixture.store.callbackCalls())
	}
	shutdownCredentialBrokerServer(t, server)
	if _, err := os.Lstat(config.SocketPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("socket remains after shutdown: %v", err)
	}
	if err := server.Wait(); err != nil {
		t.Fatalf("Wait after Shutdown: %v", err)
	}
}

func TestCredentialBrokerServerRejectsWrongPeerUIDBeforeCredentialAuthority(t *testing.T) {
	fixture := newCredentialBrokerFixture(t, []byte("secret-marker"))
	config := credentialBrokerServerConfigForTest(t, 1)
	config.PeerUID ^= 1
	server, err := NewCredentialBrokerServer(config, fixture.broker(t, time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	connection := dialCredentialBroker(t, config.SocketPath)
	writeApertureBestEffort(connection, fixture.aperture)
	if err := connection.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	var response [1]byte
	if count, readErr := connection.Read(response[:]); count != 0 ||
		!errors.Is(readErr, io.EOF) && !errors.Is(readErr, unix.ECONNRESET) {
		t.Fatalf("unauthorized peer response count=%d err=%v", count, readErr)
	}
	_ = connection.Close()
	if fixture.store.callbackCalls() != 0 {
		t.Fatalf("unauthorized peer crossed credential authority: callback=%d", fixture.store.callbackCalls())
	}
	shutdownCredentialBrokerServer(t, server)
}

func TestCredentialBrokerServerBoundsConcurrentConnections(t *testing.T) {
	server, fixture, config := newCredentialBrokerServerFixture(t, 1)
	if err := server.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	first := dialCredentialBroker(t, config.SocketPath)
	writeAperture(t, first, fixture.aperture)

	second := dialCredentialBroker(t, config.SocketPath)
	writeAperture(t, second, fixture.aperture)
	writeRequest(t, second, fixture.request)
	if err := second.SetReadDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if _, err := microvm.DecodificarCabeceraRespuestaAuthJSONCodexV1(second); err == nil {
		t.Fatal("second connection bypassed MaxConcurrent")
	}
	_ = first.Close()
	if err := second.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	header, material := readBrokerResponse(t, second)
	if header.Estado != microvm.EstadoRespuestaAuthJSONCodexDisponible || string(material) != "secret-marker" {
		t.Fatalf("second response=%#v material=%q", header, material)
	}
	writeAck(t, second, fixture.ack(microvm.EtapaAcuseAuthJSONCodexProyectada, true))
	writeAck(t, second, fixture.ack(microvm.EtapaAcuseAuthJSONCodexPurgada, true))
	_ = second.Close()
	shutdownCredentialBrokerServer(t, server)
}

func TestCredentialBrokerServerConnectionTimeoutIsBoundedAndServerRemainsAvailable(t *testing.T) {
	fixture := newCredentialBrokerFixture(t, []byte("secret-marker"))
	config := credentialBrokerServerConfigForTest(t, 1)
	config.ConnectionTimeout = 40 * time.Millisecond
	server, err := NewCredentialBrokerServer(config, fixture.broker(t, time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	idle := dialCredentialBroker(t, config.SocketPath)
	if err := idle.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	var response [1]byte
	if count, readErr := idle.Read(response[:]); count != 0 ||
		!errors.Is(readErr, io.EOF) && !errors.Is(readErr, unix.ECONNRESET) {
		t.Fatalf("idle connection count=%d err=%v", count, readErr)
	}
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("connection timeout took %s", elapsed)
	}
	_ = idle.Close()

	connection := dialCredentialBroker(t, config.SocketPath)
	writeAperture(t, connection, fixture.aperture)
	writeRequest(t, connection, fixture.request)
	header, material := readBrokerResponse(t, connection)
	if header.Estado != microvm.EstadoRespuestaAuthJSONCodexDisponible || string(material) != "secret-marker" {
		t.Fatalf("response=%#v material=%q", header, material)
	}
	writeAck(t, connection, fixture.ack(microvm.EtapaAcuseAuthJSONCodexProyectada, true))
	writeAck(t, connection, fixture.ack(microvm.EtapaAcuseAuthJSONCodexPurgada, true))
	_ = connection.Close()
	shutdownCredentialBrokerServer(t, server)
}

func TestCredentialBrokerPeerCredentialsReturnExactKernelUIDAndPID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "peer.sock")
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	client := dialCredentialBroker(t, path)
	defer client.Close()
	server, err := listener.AcceptUnix()
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	credential, err := credentialBrokerPeerCredentials(server)
	if err != nil {
		t.Fatal(err)
	}
	if credential.Uid != uint32(os.Geteuid()) || credential.Pid != int32(os.Getpid()) {
		t.Fatalf("peer uid=%d pid=%d, want uid=%d pid=%d", credential.Uid, credential.Pid, os.Geteuid(), os.Getpid())
	}
}

func TestCredentialBrokerServerParentCancellationClosesActiveConnectionAndCleansSocket(t *testing.T) {
	server, fixture, config := newCredentialBrokerServerFixture(t, 1)
	ctx, cancel := context.WithCancel(context.Background())
	if err := server.Start(ctx); err != nil {
		t.Fatal(err)
	}
	connection := dialCredentialBroker(t, config.SocketPath)
	writeAperture(t, connection, fixture.aperture)
	cancel()
	wait := make(chan error, 1)
	go func() { wait <- server.Wait() }()
	select {
	case err := <-wait:
		if err != nil {
			t.Fatalf("Wait: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("parent cancellation did not stop broker")
	}
	if _, err := os.Lstat(config.SocketPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("socket remains: %v", err)
	}
	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("idempotent Shutdown: %v", err)
	}
	_ = connection.Close()
}

func TestCredentialBrokerServerLifecycleRejectsWaitBeforeStartAndSecondStart(t *testing.T) {
	server, _, _ := newCredentialBrokerServerFixture(t, 1)
	if err := server.Wait(); ErrorCode(err) != CodeCredentialBrokerServerInvalid {
		t.Fatalf("Wait before Start error=%v code=%q", err, ErrorCode(err))
	}
	if err := server.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := server.Start(context.Background()); ErrorCode(err) != CodeCredentialBrokerServerInvalid {
		t.Fatalf("second Start error=%v code=%q", err, ErrorCode(err))
	}
	shutdownCredentialBrokerServer(t, server)
	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("second Shutdown: %v", err)
	}
}

func TestCredentialBrokerServerNeverRemovesReplacedSocket(t *testing.T) {
	server, _, config := newCredentialBrokerServerFixture(t, 1)
	if err := server.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(config.SocketPath); err != nil {
		t.Fatal(err)
	}
	const replacement = "replacement-marker"
	if err := os.WriteFile(config.SocketPath, []byte(replacement), 0o600); err != nil {
		t.Fatal(err)
	}
	err := server.Shutdown(context.Background())
	if ErrorCode(err) != CodeCredentialBrokerCleanupFailed {
		t.Fatalf("Shutdown error=%v code=%q", err, ErrorCode(err))
	}
	content, readErr := os.ReadFile(config.SocketPath)
	if readErr != nil || string(content) != replacement {
		t.Fatalf("replacement changed: content=%q err=%v", content, readErr)
	}
	if ErrorCode(server.Wait()) != CodeCredentialBrokerCleanupFailed {
		t.Fatal("Wait lost cleanup failure")
	}
}

func TestCredentialBrokerSocketOwnershipProbeRejectsReplacementBetweenListenAndPublication(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*credentialBrokerSocketHooks, func(string) error)
	}{
		{"before first identity", func(hooks *credentialBrokerSocketHooks, replace func(string) error) {
			hooks.afterBindBeforeIdentity = replace
		}},
		{"after candidate identity", func(hooks *credentialBrokerSocketHooks, replace func(string) error) {
			hooks.afterCandidateBeforeProbe = replace
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := credentialBrokerServerConfigForTest(t, 1)
			var replacement *net.UnixListener
			var replacementIdentity brokerSocketIdentity
			replace := func(path string) error {
				if err := os.Remove(path); err != nil {
					return err
				}
				created, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
				if err != nil {
					return err
				}
				created.SetUnlinkOnClose(false)
				replacement = created
				if err := os.Chmod(path, 0o600); err != nil {
					return err
				}
				var stat unix.Stat_t
				if err := unix.Lstat(path, &stat); err != nil {
					return err
				}
				replacementIdentity = brokerSocketIdentity{device: uint64(stat.Dev), inode: stat.Ino}
				return nil
			}
			hooks := credentialBrokerSocketHooks{}
			test.configure(&hooks, replace)
			listener, root, identity, err := openCredentialBrokerSocketWithHooks(config, hooks)
			if listener != nil || root != nil || identity != (brokerSocketIdentity{}) ||
				ErrorCode(err) != CodeCredentialBrokerSocketUnsafe {
				t.Fatalf("listener=%v root=%v identity=%+v error=%v", listener, root, identity, err)
			}
			if replacement == nil || replacementIdentity == (brokerSocketIdentity{}) {
				t.Fatal("fault seam did not publish replacement")
			}
			defer func() {
				_ = replacement.Close()
				_ = os.Remove(config.SocketPath)
			}()
			var after unix.Stat_t
			if err := unix.Lstat(config.SocketPath, &after); err != nil ||
				(brokerSocketIdentity{device: uint64(after.Dev), inode: after.Ino}) != replacementIdentity {
				t.Fatalf("replacement removed or crossed: stat=%+v err=%v", after, err)
			}
		})
	}
}

func TestCredentialBrokerServerRejectsUnsafeSocketPublicationWithoutReplacingIt(t *testing.T) {
	fixture := newCredentialBrokerFixture(t, []byte("secret-marker"))
	t.Run("existing file", func(t *testing.T) {
		config := credentialBrokerServerConfigForTest(t, 1)
		const marker = "existing-marker"
		if err := os.WriteFile(config.SocketPath, []byte(marker), 0o600); err != nil {
			t.Fatal(err)
		}
		server, err := NewCredentialBrokerServer(config, fixture.broker(t, time.Second))
		if err != nil {
			t.Fatal(err)
		}
		if err := server.Start(context.Background()); ErrorCode(err) != CodeCredentialBrokerSocketUnsafe {
			t.Fatalf("Start error=%v code=%q", err, ErrorCode(err))
		}
		content, err := os.ReadFile(config.SocketPath)
		if err != nil || string(content) != marker {
			t.Fatalf("existing path changed: content=%q err=%v", content, err)
		}
	})
	t.Run("public root", func(t *testing.T) {
		config := credentialBrokerServerConfigForTest(t, 1)
		if err := os.Chmod(filepath.Dir(config.SocketPath), 0o755); err != nil {
			t.Fatal(err)
		}
		server, err := NewCredentialBrokerServer(config, fixture.broker(t, time.Second))
		if err != nil {
			t.Fatal(err)
		}
		if err := server.Start(context.Background()); ErrorCode(err) != CodeCredentialBrokerSocketUnsafe {
			t.Fatalf("Start error=%v code=%q", err, ErrorCode(err))
		}
	})
	t.Run("symlink root", func(t *testing.T) {
		realRoot := t.TempDir()
		linkRoot := filepath.Join(filepath.Dir(realRoot), filepath.Base(realRoot)+"-link")
		if err := os.Symlink(realRoot, linkRoot); err != nil {
			t.Fatal(err)
		}
		config := credentialBrokerServerConfigForTest(t, 1)
		config.SocketPath = filepath.Join(linkRoot, "broker.sock")
		server, err := NewCredentialBrokerServer(config, fixture.broker(t, time.Second))
		if err != nil {
			t.Fatal(err)
		}
		if err := server.Start(context.Background()); ErrorCode(err) != CodeCredentialBrokerSocketUnsafe {
			t.Fatalf("Start error=%v code=%q", err, ErrorCode(err))
		}
	})
}

func TestNewCredentialBrokerServerRequiresEveryExplicitBoundary(t *testing.T) {
	fixture := newCredentialBrokerFixture(t, []byte("secret-marker"))
	valid := credentialBrokerServerConfigForTest(t, 1)
	tests := map[string]func(*CredentialBrokerServerConfig){
		"relative path": func(config *CredentialBrokerServerConfig) { config.SocketPath = "broker.sock" },
		"noncanonical path": func(config *CredentialBrokerServerConfig) {
			config.SocketPath = filepath.Dir(config.SocketPath) + "/./broker.sock"
		},
		"wrong owner":      func(config *CredentialBrokerServerConfig) { config.OwnerUID ^= 1 },
		"zero timeout":     func(config *CredentialBrokerServerConfig) { config.ConnectionTimeout = 0 },
		"negative timeout": func(config *CredentialBrokerServerConfig) { config.ConnectionTimeout = -time.Second },
		"zero concurrency": func(config *CredentialBrokerServerConfig) { config.MaxConcurrent = 0 },
		"too much concurrency": func(config *CredentialBrokerServerConfig) {
			config.MaxConcurrent = maximumCredentialBrokerHandlers + 1
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			config := valid
			mutate(&config)
			server, err := NewCredentialBrokerServer(config, fixture.broker(t, time.Second))
			if server != nil || ErrorCode(err) != CodeCredentialBrokerServerInvalid {
				t.Fatalf("server=%v error=%v", server, err)
			}
		})
	}
	if server, err := NewCredentialBrokerServer(valid, nil); server != nil || ErrorCode(err) != CodeCredentialBrokerServerInvalid {
		t.Fatalf("nil broker server=%v error=%v", server, err)
	}
	server, err := NewCredentialBrokerServer(valid, fixture.broker(t, time.Second))
	if err != nil {
		t.Fatal(err)
	}
	var typedNil *credentialBrokerNilContext
	if err := server.Start(typedNil); ErrorCode(err) != CodeCredentialBrokerServerInvalid {
		t.Fatalf("typed-nil context error=%v code=%q", err, ErrorCode(err))
	}
	if err := server.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := server.Shutdown(typedNil); ErrorCode(err) != CodeCredentialBrokerServerInvalid {
		t.Fatalf("typed-nil shutdown error=%v code=%q", err, ErrorCode(err))
	}
	shutdownCredentialBrokerServer(t, server)
}

func newCredentialBrokerServerFixture(
	t *testing.T,
	maxConcurrent uint32,
) (*CredentialBrokerServer, credentialBrokerFixture, CredentialBrokerServerConfig) {
	t.Helper()
	fixture := newCredentialBrokerFixture(t, []byte("secret-marker"))
	config := credentialBrokerServerConfigForTest(t, maxConcurrent)
	server, err := NewCredentialBrokerServer(config, fixture.broker(t, time.Second))
	if err != nil {
		t.Fatal(err)
	}
	return server, fixture, config
}

func credentialBrokerServerConfigForTest(t *testing.T, maxConcurrent uint32) CredentialBrokerServerConfig {
	t.Helper()
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	return CredentialBrokerServerConfig{
		SocketPath: filepath.Join(root, "broker.sock"), OwnerUID: uint32(os.Geteuid()),
		PeerUID: uint32(os.Geteuid()), ConnectionTimeout: time.Second, MaxConcurrent: maxConcurrent,
	}
}

func assertPrivateBrokerSocket(t *testing.T, path string, owner uint32) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	var stat unix.Stat_t
	statErr := unix.Lstat(path, &stat)
	if statErr != nil || info.Mode()&os.ModeSocket == 0 || info.Mode().Perm() != 0o600 || stat.Uid != owner {
		t.Fatalf("unsafe socket mode=%v owner=%d stat_error=%v", info.Mode(), stat.Uid, statErr)
	}
}

func dialCredentialBroker(t *testing.T, path string) *net.UnixConn {
	t.Helper()
	connection, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	return connection
}

func shutdownCredentialBrokerServer(t *testing.T, server *CredentialBrokerServer) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}

func writeApertureBestEffort(connection net.Conn, aperture microvm.AperturaServicioHostV1) {
	frame, err := microvm.CodificarAperturaServicioHostV1(aperture)
	if err == nil {
		_, _ = connection.Write(frame)
	}
}

type credentialBrokerNilContext struct{}

func (*credentialBrokerNilContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (*credentialBrokerNilContext) Done() <-chan struct{}       { return nil }
func (*credentialBrokerNilContext) Err() error                  { return nil }
func (*credentialBrokerNilContext) Value(any) any               { return nil }
