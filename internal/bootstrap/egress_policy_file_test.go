//go:build linux

package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/adapters/egresspolicyfile"
	"orquesta/internal/application"
	"orquesta/internal/config"
	"orquesta/internal/goal"
)

func TestBuildEgressPolicyResolverIsAbsentWithoutExplicitPolicy(t *testing.T) {
	configPath := writeTestConfig(t, t.TempDir())
	snapshot, err := loadConfigSnapshot(context.Background(), configPath)
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := openBuildEgressPolicyResolver(snapshot)
	if err != nil || resolver != nil {
		t.Fatalf("absent egress policy resolver = %T, %v", resolver, err)
	}
}

func TestBuildLoadsAndInjectsExactEgressPolicyResolver(t *testing.T) {
	root := t.TempDir()
	configPath, _, _ := writeRuntimeIsolationMicroVMConfig(t, root, false)
	policyRef, policyPath, raw, digest := writeBootstrapEgressPolicy(t, root, "configured")
	configureBootstrapEgressPolicy(t, configPath, policyRef.String(), policyPath, digest, int64(len(raw)))

	snapshot, err := loadConfigSnapshot(context.Background(), configPath)
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := openBuildEgressPolicyResolver(snapshot)
	if err != nil {
		t.Fatalf("open resolver: %v", err)
	}
	authority, err := resolver.ResolveEgressPolicy(context.Background(), policyRef)
	if err != nil || authority.PolicyRef != policyRef || authority.PayloadSHA256 != digest ||
		authority.CanonicalPayload != string(raw) {
		t.Fatalf("resolved authority = %+v, %v", authority, err)
	}
	runtime, err := Build(context.Background(), Options{
		ConfigPath: configPath, AgentFactory: countingFactory(new(atomic.Int64)),
	})
	if err != nil {
		t.Fatalf("Build(configured egress policy): %v", err)
	}
	if runtime == nil {
		t.Fatal("Build(configured egress policy) returned nil runtime")
	}
	access := testRuntimeAccess(t, runtime)
	submitted, err := runtime.Orchestrator().Submit(context.Background(), access, application.SubmitRequest{
		RequestRef: "request:bootstrap-egress-policy", Statement: "use the exact sealed egress authority", Confirm: true,
		Plan: &application.PlanSpec{
			Phases: []application.PhaseSpec{{
				Ref: "phase-instance:bootstrap-egress", Key: goal.DefaultPhaseKey().String(),
				TemplateRef: "phase-template:default",
			}},
			WorkItems: []application.WorkItemSpec{{
				Key: "work:bootstrap-egress", Objective: "exercise the configured egress policy",
				Phase: goal.DefaultPhaseKey().String(), Role: goal.DefaultRoleKey().String(),
				OutputContract: goal.OutputContractEvidenceBundle, EgressPolicyRef: policyRef.String(),
			}},
		},
	})
	if err != nil {
		t.Fatalf("Submit(configured egress policy): %v", err)
	}
	durable, err := runtime.Orchestrator().GetGoal(context.Background(), access, submitted.Record.Goal.Ref())
	if err != nil || len(durable.WorkItemAuthorities) != 1 ||
		durable.WorkItemAuthorities[0].EgressPolicy != authority {
		t.Fatalf("durable egress authority = %+v, %v; want %+v", durable.WorkItemAuthorities, err, authority)
	}
	if err := runtime.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown(configured egress policy): %v", err)
	}
}

func TestBuildRejectsEgressPolicyProtectedPathCollisionsBeforeEffects(t *testing.T) {
	tests := []struct {
		name      string
		collision func(root, configPath string) string
	}{
		{name: "effective config", collision: func(root, _ string) string { return filepath.Join(root, "effective_config.json") }},
		{name: "state", collision: func(root, _ string) string { return filepath.Join(root, "state", "orquesta.sqlite") }},
		{name: "state sidecar", collision: func(root, _ string) string { return filepath.Join(root, "state", "orquesta.sqlite-wal") }},
		{name: "artifact root", collision: func(root, _ string) string { return filepath.Join(root, "artifacts") }},
		{name: "credential store", collision: func(root, _ string) string { return filepath.Join(root, "secrets", "credentials.json") }},
		{name: "credential sidecar", collision: func(root, _ string) string { return filepath.Join(root, "secrets", "credentials.json.next") }},
		{name: "workspace root", collision: func(root, _ string) string { return filepath.Join(root, "workspaces") }},
		{name: "config sidecar", collision: func(_, configPath string) string { return configPath + ".next" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			configPath, _, _ := writeRuntimeIsolationMicroVMConfig(t, root, false)
			_, policyPath, raw, digest := writeBootstrapEgressPolicy(t, root, "protected-pair")
			configureBootstrapEgressPolicy(t, configPath, "egreso:protected-pair", policyPath, digest, int64(len(raw)))
			replaceTestConfigValue(t, configPath, "[identity]\n", `[workspace.local]
root = `+strconv.Quote(filepath.Join(root, "workspaces"))+`

[identity]
`)
			replaceTestConfigValue(t, configPath, policyPath, test.collision(root, configPath))
			listener := &egressPolicyListenerProbe{}
			var factoryCalls atomic.Int64
			runtime, err := Build(context.Background(), Options{
				ConfigPath: configPath, Listener: listener,
				AgentFactory: func(snapshot config.Snapshot, clock application.Clock) (AgentAdapter, error) {
					factoryCalls.Add(1)
					return countingFactory(new(atomic.Int64))(snapshot, clock)
				},
			})
			validError := config.HasErrorCode(err, config.ErrorCrossValidation) ||
				err != nil && err.Error() == "bootstrap.runtime_paths_overlap"
			if runtime != nil || !validError {
				t.Fatalf("protected collision Build() = %v, %v", runtime, err)
			}
			if listener.closes.Load() != 0 || factoryCalls.Load() != 0 {
				t.Fatalf("protected collision reached listener=%d agent=%d", listener.closes.Load(), factoryCalls.Load())
			}
			assertNoCompositionState(t, root)
		})
	}
}

func TestBuildRejectsEgressPolicySymlinkAliasBeforeEffects(t *testing.T) {
	root := t.TempDir()
	configPath, _, _ := writeRuntimeIsolationMicroVMConfig(t, root, false)
	_, policyPath, raw, digest := writeBootstrapEgressPolicy(t, root, "alias")
	artifactRoot := filepath.Join(root, "artifacts")
	if err := os.MkdirAll(artifactRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(artifactRoot, "preserved-policy.json")
	if err := os.Rename(policyPath, target); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "policy-alias.json")
	if err := os.Symlink(target, alias); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	configureBootstrapEgressPolicy(t, configPath, "egreso:alias", alias, digest, int64(len(raw)))
	listener := &egressPolicyListenerProbe{}
	var factoryCalls atomic.Int64
	runtime, err := Build(context.Background(), Options{
		ConfigPath: configPath, Listener: listener,
		AgentFactory: func(snapshot config.Snapshot, clock application.Clock) (AgentAdapter, error) {
			factoryCalls.Add(1)
			return countingFactory(new(atomic.Int64))(snapshot, clock)
		},
	})
	if runtime != nil || err == nil || err.Error() != "bootstrap.runtime_paths_overlap" {
		t.Fatalf("symlink alias Build() = %v, %v", runtime, err)
	}
	if listener.closes.Load() != 0 || factoryCalls.Load() != 0 {
		t.Fatalf("symlink alias reached listener=%d agent=%d", listener.closes.Load(), factoryCalls.Load())
	}
	for _, path := range []string{
		filepath.Join(root, "state"), filepath.Join(root, "work"), filepath.Join(root, "secrets"),
		filepath.Join(root, "effective_config.json"),
	} {
		if _, statErr := os.Lstat(path); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("symlink alias created %s: %v", path, statErr)
		}
	}
	if preserved, readErr := os.ReadFile(target); readErr != nil || string(preserved) != string(raw) {
		t.Fatalf("symlink alias altered protected input: %q, %v", preserved, readErr)
	}
}

type egressPolicyListenerProbe struct{ closes atomic.Int64 }

func (*egressPolicyListenerProbe) Accept() (net.Conn, error) {
	return nil, errors.New("egress_policy_listener_probe.accept_unexpected")
}
func (listener *egressPolicyListenerProbe) Close() error {
	listener.closes.Add(1)
	return nil
}
func (*egressPolicyListenerProbe) Addr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 43210}
}

func TestBuildRejectsInvalidEgressPolicyBeforeCompositionOutputs(t *testing.T) {
	root := t.TempDir()
	configPath, _, _ := writeRuntimeIsolationMicroVMConfig(t, root, false)
	policyRef, policyPath, raw, _ := writeBootstrapEgressPolicy(t, root, "invalid-digest")
	wrong := sha256.Sum256([]byte("another policy"))
	configureBootstrapEgressPolicy(
		t, configPath, policyRef.String(), policyPath, hex.EncodeToString(wrong[:]), int64(len(raw)),
	)

	runtime, err := Build(context.Background(), Options{
		ConfigPath: configPath, AgentFactory: countingFactory(new(atomic.Int64)),
	})
	if runtime != nil || !egresspolicyfile.IsError(err, egresspolicyfile.CodeDigestMismatch) {
		t.Fatalf("invalid policy Build() = %v, %v", runtime, err)
	}
	assertNoCompositionState(t, root)
	for _, path := range []string{policyPath, configPath} {
		if info, statErr := os.Lstat(path); statErr != nil || !info.Mode().IsRegular() {
			t.Fatalf("invalid Build altered input %s: info=%v err=%v", filepath.Base(path), info, statErr)
		}
	}
}

func writeBootstrapEgressPolicy(
	t *testing.T,
	root, suffix string,
) (application.EgressPolicyRef, string, []byte, string) {
	t.Helper()
	ref, err := application.NewEgressPolicyRef("egreso:" + suffix)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(microvm.ConcesionEgreso{
		Esquema: microvm.EsquemaConcesionEgreso, Referencia: ref.String(),
		Destinos:         []microvm.DestinoEgreso{{Host: "api.openai.com", Puertos: []uint16{443}}},
		MaximoConexiones: 4, LimiteTiempoMS: 60_000,
		LimiteSubidaBytes: 1 << 20, LimiteBajadaBytes: 8 << 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "egress-policy-"+suffix+".json")
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(payload)
	return ref, path, payload, hex.EncodeToString(digest[:])
}

func configureBootstrapEgressPolicy(
	t *testing.T,
	configPath, ref, path, digest string,
	maximumBytes int64,
) {
	t.Helper()
	replaceTestConfigValue(t, configPath, "[runtime.codex]\n", `egress_policy_ref = `+strconv.Quote(ref)+`
egress_policy_path = `+strconv.Quote(path)+`
egress_policy_expected_sha256 = `+strconv.Quote(digest)+`
egress_policy_max_bytes = `+strconv.FormatInt(maximumBytes, 10)+`

[runtime.codex]
`)
}
