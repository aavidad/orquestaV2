//go:build linux

package firecrackerattestor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestWaitLauncherReadyRetriesOnlyExplicitNotReadyState(t *testing.T) {
	config := validSupervisorConfig()
	config.PollInterval = time.Millisecond
	baseline := readyTestIdentity(config)
	probes := 0
	identity, err := waitLauncherReady(
		context.Background(),
		config,
		baseline,
		func(context.Context, string) (UnitIdentity, error) {
			return baseline, nil
		},
		func(Config, UnitIdentity) (bool, error) {
			probes++
			return probes == 2, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if probes != 2 || identity != baseline {
		t.Fatalf("probes=%d identity=%+v", probes, identity)
	}
}

func TestWaitLauncherReadyTreatsIdentityAndProbeErrorsAsTerminal(t *testing.T) {
	config := validSupervisorConfig()
	config.PollInterval = time.Hour
	baseline := readyTestIdentity(config)
	tests := map[string]struct {
		observe func(context.Context, string) (UnitIdentity, error)
		probe   func(Config, UnitIdentity) (bool, error)
	}{
		"identity": {
			observe: func(context.Context, string) (UnitIdentity, error) {
				drifted := baseline
				drifted.InvocationID = "cccccccccccccccccccccccccccccccc"
				return drifted, nil
			},
			probe: func(Config, UnitIdentity) (bool, error) {
				t.Fatal("probe called after identity drift")
				return false, nil
			},
		},
		"probe": {
			observe: func(context.Context, string) (UnitIdentity, error) {
				return baseline, nil
			},
			probe: func(Config, UnitIdentity) (bool, error) {
				return false, errors.New("unsafe socket")
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			started := time.Now()
			if _, err := waitLauncherReady(
				context.Background(),
				config,
				baseline,
				test.observe,
				test.probe,
			); !errors.Is(err, errReadinessUnsafe) {
				t.Fatalf("terminal error=%v", err)
			}
			if elapsed := time.Since(started); elapsed > 100*time.Millisecond {
				t.Fatalf("terminal readiness retried: %s", elapsed)
			}
		})
	}
}

func TestProbeLauncherReadinessRequiresRunsAndExactListeningSocket(t *testing.T) {
	config, identity, root, trustedUID, trustedGID := readinessFixture(t)
	ready, err := probeLauncherReadiness(config, identity, trustedUID, trustedGID)
	if err != nil || ready {
		t.Fatalf("missing runs must be transient: ready=%t err=%v", ready, err)
	}
	if err := os.Mkdir(filepath.Join(root, "runs"), 0o700); err != nil {
		t.Fatal(err)
	}
	ready, err = probeLauncherReadiness(config, identity, trustedUID, trustedGID)
	if err != nil || ready {
		t.Fatalf("missing socket must be transient: ready=%t err=%v", ready, err)
	}
	socket := listenReadinessSocket(t, config.Candidate.LauncherSocketPath)
	defer unix.Close(socket)
	ready, err = probeLauncherReadiness(config, identity, trustedUID, trustedGID)
	if err != nil || !ready {
		t.Fatalf("exact listening socket rejected: ready=%t err=%v", ready, err)
	}
}

func TestProbeLauncherReadinessRejectsUnsafeObjectsWithoutRetry(t *testing.T) {
	t.Run("runs_symlink", func(t *testing.T) {
		config, identity, root, trustedUID, trustedGID := readinessFixture(t)
		if err := os.Symlink(t.TempDir(), filepath.Join(root, "runs")); err != nil {
			t.Fatal(err)
		}
		if ready, err := probeLauncherReadiness(
			config,
			identity,
			trustedUID,
			trustedGID,
		); ready || !errors.Is(err, errReadinessUnsafe) {
			t.Fatalf("unsafe runs result ready=%t err=%v", ready, err)
		}
	})
	t.Run("socket_wrong_type", func(t *testing.T) {
		config, identity, root, trustedUID, trustedGID := readinessFixture(t)
		if err := os.Mkdir(filepath.Join(root, "runs"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(
			config.Candidate.LauncherSocketPath,
			[]byte("not a socket"),
			0o600,
		); err != nil {
			t.Fatal(err)
		}
		if ready, err := probeLauncherReadiness(
			config,
			identity,
			trustedUID,
			trustedGID,
		); ready || !errors.Is(err, errReadinessUnsafe) {
			t.Fatalf("unsafe socket result ready=%t err=%v", ready, err)
		}
	})
	t.Run("peer_pid_mismatch", func(t *testing.T) {
		config, identity, root, trustedUID, trustedGID := readinessFixture(t)
		if err := os.Mkdir(filepath.Join(root, "runs"), 0o700); err != nil {
			t.Fatal(err)
		}
		socket := listenReadinessSocket(t, config.Candidate.LauncherSocketPath)
		defer unix.Close(socket)
		identity.MainPID++
		if ready, err := probeLauncherReadiness(
			config,
			identity,
			trustedUID,
			trustedGID,
		); ready || !errors.Is(err, errReadinessUnsafe) {
			t.Fatalf("foreign peer result ready=%t err=%v", ready, err)
		}
	})
}

func TestTransientLauncherReadinessErrorsAreAnExactAllowlist(t *testing.T) {
	for _, err := range []error{unix.ECONNREFUSED, unix.EAGAIN, unix.EINPROGRESS} {
		if !transientLauncherReadinessError(err) {
			t.Fatalf("expected transient readiness error: %v", err)
		}
	}
	for _, err := range []error{nil, unix.EACCES, unix.ELOOP, unix.ENOTSOCK} {
		if transientLauncherReadinessError(err) {
			t.Fatalf("unsafe readiness error became retryable: %v", err)
		}
	}
}

func readinessFixture(t *testing.T) (Config, UnitIdentity, string, uint32, uint32) {
	t.Helper()
	root := t.TempDir()
	if err := os.Chmod(root, 0o750); err != nil {
		t.Fatal(err)
	}
	config := validSupervisorConfig()
	config.Candidate.RuntimeRoot = root
	config.Candidate.LauncherSocketPath = filepath.Join(root, "launcher.sock")
	config.ChildGID = uint32(os.Getegid())
	config.WorkloadPayload = []byte(
		`{"socket_path":"` + config.Candidate.LauncherSocketPath +
			`","expected_asset_digest":"` + testDigest + `"}`,
	)
	return config, readyTestIdentity(config), root,
		uint32(os.Geteuid()), uint32(os.Getegid())
}

func readyTestIdentity(config Config) UnitIdentity {
	return UnitIdentity{
		UnitName:     config.Candidate.UnitName,
		MainPID:      os.Getpid(),
		InvocationID: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Active:       true, FragmentPath: config.Candidate.UnitPath, Loaded: true,
	}
}

func listenReadinessSocket(t *testing.T, path string) int {
	t.Helper()
	socket, err := unix.Socket(
		unix.AF_UNIX,
		unix.SOCK_SEQPACKET|unix.SOCK_CLOEXEC|unix.SOCK_NONBLOCK,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = unix.Close(socket)
		_ = os.Remove(path)
	})
	if err := unix.Bind(socket, &unix.SockaddrUnix{Name: path}); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o660); err != nil {
		t.Fatal(err)
	}
	if err := unix.Listen(socket, 1); err != nil {
		t.Fatal(err)
	}
	return socket
}
