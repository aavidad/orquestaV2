package codex

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"os"
	"path"
	"path/filepath"
	"sync"
	"testing"

	"orquesta/internal/credentials"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestCredentialLaunchRotationRevocationAndLeakGate(t *testing.T) {
	store := &credentialTestStore{material: helperCredentialInitial, version: 1}
	credentialRef := credentials.CredentialRef("credential:codex-primary")

	config := testConfig(t)
	config.CredentialStore, config.CredentialRef = store, credentialRef
	adapter := openTestAdapter(t, config)
	assertCredentialOutcome(t, adapter, "credential-initial", "helper:credential-initial", ports.AgentCompleted, "", "artifact:credential-initial")
	assertCredentialOutcome(t, adapter, "secret-leak", "helper:secret-leak", ports.AgentFailed, CodeSecretLeak, "")
	assertCredentialOutcome(t, adapter, "artifact-secret-leak", "helper:artifact-secret-leak", ports.AgentFailed, CodeSecretLeak, "")
	assertCredentialOutcome(t, adapter, "encoded-secret-leak", "helper:encoded-secret-leak", ports.AgentFailed, CodeSecretLeak, "")
	assertNoMaterialInTree(t, config.WorkRoot, helperCredentialInitial)

	rotated := mustTestSecret(t, helperCredentialRotated)
	if _, err := store.Rotate(context.Background(), credentials.RotateRequest{
		ActorRef: "actor:local-owner", RequestRef: "request:rotate-codex-primary",
		CredentialRef: credentialRef, OwnerRef: "actor:local-owner", ExpectedVersion: 1, Material: rotated,
	}); err != nil {
		t.Fatalf("Rotate credential: %v", err)
	}
	rotated.Destroy()
	assertCredentialOutcome(t, adapter, "credential-rotated", "helper:credential-rotated", ports.AgentCompleted, "", "artifact:credential-rotated")

	if _, err := store.Revoke(context.Background(), credentials.RevokeRequest{
		ActorRef: "actor:local-owner", RequestRef: "request:revoke-codex-primary",
		CredentialRef: credentialRef, OwnerRef: "actor:local-owner", ExpectedVersion: 2, Reason: "test revocation",
	}); err != nil {
		t.Fatalf("Revoke credential: %v", err)
	}
	revoked := testRequest(t, "credential-revoked", "helper:credential-rotated", 1024)
	if _, err := adapter.Launch(context.Background(), revoked); ErrorCode(err) != CodeCredentialUnavailable {
		t.Fatalf("revoked Launch error=%v code=%q", err, ErrorCode(err))
	}
	if _, err := os.Lstat(filepath.Join(config.WorkRoot, filepath.FromSlash(executionPath(revoked.ExecutionRef)))); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("revoked launch created execution path: %v", err)
	}
	assertNoMaterialInTree(t, config.WorkRoot, helperCredentialInitial, helperCredentialRotated)
}

func TestCredentialPreflightRejectsPromptAndArguments(t *testing.T) {
	for _, test := range []struct {
		name        string
		editConfig  func(*Config)
		editRequest func(*ports.AgentLaunchRequest)
	}{
		{name: "prompt", editRequest: func(request *ports.AgentLaunchRequest) { request.Objective = helperCredentialInitial }},
		{name: "arguments", editConfig: func(config *Config) { config.Model = helperCredentialInitial }},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := testConfig(t)
			config.CredentialStore = &credentialTestStore{material: helperCredentialInitial, version: 1}
			config.CredentialRef = "credential:codex-primary"
			if test.editConfig != nil {
				test.editConfig(&config)
			}
			adapter := openTestAdapter(t, config)
			request := testRequest(t, "preflight-"+test.name, "helper:success", 1024)
			if test.editRequest != nil {
				test.editRequest(&request)
			}
			if _, err := adapter.Launch(context.Background(), request); ErrorCode(err) != CodeSecretLeak {
				t.Fatalf("Launch preflight error=%v code=%q", err, ErrorCode(err))
			}
			runDirectory := filepath.Join(config.WorkRoot, filepath.FromSlash(executionPath(request.ExecutionRef)))
			if _, err := os.Lstat(runDirectory); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("preflight leak created execution state: %v", err)
			}
		})
	}
}

func TestCredentialPreflightRejectsEveryDurableRequestField(t *testing.T) {
	tests := []struct {
		name     string
		material func(ports.AgentLaunchRequest) string
		mutate   func(*ports.AgentLaunchRequest, string)
	}{
		{name: "execution_ref", material: func(ports.AgentLaunchRequest) string { return helperCredentialInitial }, mutate: func(request *ports.AgentLaunchRequest, material string) {
			request.ExecutionRef, _ = goal.NewExecutionRef(material)
		}},
		{name: "goal_ref", material: func(ports.AgentLaunchRequest) string { return helperCredentialInitial }, mutate: func(request *ports.AgentLaunchRequest, material string) {
			request.GoalRef, _ = goal.NewGoalRef(material)
		}},
		{name: "work_item_ref", material: func(ports.AgentLaunchRequest) string { return helperCredentialInitial }, mutate: func(request *ports.AgentLaunchRequest, material string) {
			request.WorkItemRef, _ = goal.NewWorkItemRef(material)
		}},
		{name: "spec_hash", material: func(request ports.AgentLaunchRequest) string { return request.SpecHash }},
		{name: "idempotency_key", material: func(ports.AgentLaunchRequest) string { return helperCredentialInitial }, mutate: func(request *ports.AgentLaunchRequest, material string) {
			request.IdempotencyKey = material
		}},
		{name: "actor_ref", material: func(ports.AgentLaunchRequest) string { return helperCredentialInitial }, mutate: func(request *ports.AgentLaunchRequest, material string) {
			request.ActorRef, _ = goal.NewActorRef(material)
		}},
		{name: "project_ref", material: func(ports.AgentLaunchRequest) string { return helperCredentialInitial }, mutate: func(request *ports.AgentLaunchRequest, material string) {
			request.ProjectRef, _ = goal.NewProjectRef(material)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := testRequest(t, "preflight-durable-"+test.name, "helper:success", 1024)
			material := test.material(request)
			if test.mutate != nil {
				test.mutate(&request, material)
			}
			config := testConfig(t)
			config.CredentialStore = &credentialTestStore{material: material, version: 1}
			config.CredentialRef = "credential:codex-primary"
			adapter := openTestAdapter(t, config)
			if _, err := adapter.Launch(context.Background(), request); ErrorCode(err) != CodeSecretLeak {
				t.Fatalf("Launch preflight error=%v code=%q", err, ErrorCode(err))
			}
			runDirectory := filepath.Join(config.WorkRoot, filepath.FromSlash(executionPath(request.ExecutionRef)))
			if _, err := os.Lstat(runDirectory); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("preflight leak created execution state: %v", err)
			}
			assertNoMaterialInTree(t, config.WorkRoot, material)
		})
	}
}

func TestCredentialLeakGateFailsClosedWhenScrubFails(t *testing.T) {
	config := testConfig(t)
	adapter := openTestAdapter(t, config)
	request := testRequest(t, "scrub-failure", "helper:success", 1024)
	_, runPath, created, err := adapter.ensureLaunchRecord(request, mustRequestHash(t, request))
	if err != nil || !created {
		t.Fatalf("seed launch created=%v err=%v", created, err)
	}
	if err := adapter.prepareRuntimeFiles(runPath); err != nil {
		t.Fatal(err)
	}
	if err := adapter.writePrivateRuntimeFile(path.Join(runPath, lastMessageFileName), []byte(helperCredentialInitial)); err != nil {
		t.Fatal(err)
	}
	secret := mustTestSecret(t, helperCredentialInitial)
	guard, err := credentials.NewLeakGuard(secret)
	secret.Destroy()
	if err != nil {
		t.Fatal(err)
	}
	state := &executionState{runPath: runPath, maxOutput: 1024, credentialGuard: guard}
	scrubFailure := errors.New("injected scrub failure")
	realScrub := adapter.credentialOutputScrub
	adapter.credentialOutputScrub = func(string) error { return scrubFailure }
	diagnostic := []byte(helperCredentialInitial)
	terminal, err := adapter.gateCredentialTerminalLocked(state, terminalRecord{
		SchemaVersion: stateSchemaVersion, RequestHash: mustRequestHash(t, request), Status: ports.AgentCompleted,
		Artifact: helperCredentialInitial, Diagnostic: diagnostic, ObservedAt: config.Now(),
	})
	if ErrorCode(err) != CodeStatePersistenceFailed || terminal.ErrorCode != CodeSecretLeak || terminal.Artifact != "" ||
		len(terminal.Diagnostic) != 0 || !bytes.Equal(diagnostic, make([]byte, len(diagnostic))) || state.credentialGuard != nil {
		t.Fatalf("fail-open gate terminal=%+v err=%v guard=%v", terminal, err, state.credentialGuard)
	}
	if _, statErr := os.Lstat(filepath.Join(config.WorkRoot, filepath.FromSlash(runPath), terminalFileName)); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("gate persisted a false-safe terminal: %v", statErr)
	}
	adapter.credentialOutputScrub = realScrub
	if err := adapter.credentialOutputScrub(runPath); err != nil {
		t.Fatalf("cleanup after injected failure: %v", err)
	}
	if err := adapter.writePrivateRuntimeFile(path.Join(runPath, lastMessageFileName), []byte("safe output")); err != nil {
		t.Fatal(err)
	}
	destroyedSecret := mustTestSecret(t, helperCredentialInitial)
	destroyedGuard, err := credentials.NewLeakGuard(destroyedSecret)
	destroyedSecret.Destroy()
	if err != nil {
		t.Fatal(err)
	}
	destroyedGuard.Destroy()
	state.credentialGuard = destroyedGuard
	terminal, err = adapter.gateCredentialTerminalLocked(state, terminalRecord{
		SchemaVersion: stateSchemaVersion, RequestHash: mustRequestHash(t, request), Status: ports.AgentCompleted, ObservedAt: config.Now(),
	})
	if ErrorCode(err) != CodeCredentialUnavailable || terminal.ErrorCode != CodeSecretLeak {
		t.Fatalf("unexpected scan failure did not fail closed: terminal=%+v err=%v", terminal, err)
	}
	payload, readErr := os.ReadFile(filepath.Join(config.WorkRoot, filepath.FromSlash(runPath), lastMessageFileName))
	if readErr != nil || len(payload) != 0 {
		t.Fatalf("unexpected scan failure did not scrub raw output: %q err=%v", payload, readErr)
	}
}

func assertCredentialOutcome(t *testing.T, adapter *Adapter, suffix, objective string, status ports.AgentStatus, code, artifact string) {
	t.Helper()
	request := testRequest(t, suffix, objective, 1024)
	if _, err := adapter.Launch(context.Background(), request); err != nil {
		t.Fatalf("Launch(%s): %v", suffix, err)
	}
	observation := awaitTerminal(t, adapter, request.ExecutionRef)
	if observation.Status != status || observation.ErrorCode != code || string(observation.Content) != artifact {
		t.Fatalf("terminal(%s)=%+v", suffix, observation)
	}
}

func mustTestSecret(t *testing.T, material string) credentials.Secret {
	t.Helper()
	secret, err := credentials.NewSecret([]byte(material))
	if err != nil {
		t.Fatalf("NewSecret: %v", err)
	}
	return secret
}

type credentialTestStore struct {
	mu       sync.Mutex
	material string
	version  credentials.Version
	revoked  bool
}

func (*credentialTestStore) Create(context.Context, credentials.CreateRequest) (credentials.MutationResult, error) {
	return credentials.MutationResult{}, credentials.NewError(credentials.ErrorAlreadyExists, "credential_ref")
}

func (store *credentialTestStore) Use(_ context.Context, request credentials.UseRequest, consume func(credentials.Secret) error) (credentials.Receipt, error) {
	store.mu.Lock()
	if store.revoked {
		store.mu.Unlock()
		return credentials.Receipt{}, credentials.NewError(credentials.ErrorRevoked, "credential_ref")
	}
	if request.Version != 0 && request.Version != store.version {
		store.mu.Unlock()
		return credentials.Receipt{}, credentials.NewError(credentials.ErrorVersionConflict, "version")
	}
	material, version := store.material, store.version
	store.mu.Unlock()
	secret, _ := credentials.NewSecret([]byte(material))
	defer secret.Destroy()
	return credentials.Receipt{Version: version}, consume(secret)
}

func (store *credentialTestStore) Rotate(_ context.Context, request credentials.RotateRequest) (credentials.MutationResult, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if request.ExpectedVersion != store.version {
		return credentials.MutationResult{}, credentials.NewError(credentials.ErrorVersionConflict, "version")
	}
	material := request.Material.Bytes()
	defer clearBytes(material)
	store.material, store.version = string(material), store.version+1
	return credentials.MutationResult{Metadata: credentials.Metadata{Version: store.version}}, nil
}

func (store *credentialTestStore) Revoke(_ context.Context, request credentials.RevokeRequest) (credentials.MutationResult, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if request.ExpectedVersion != store.version {
		return credentials.MutationResult{}, credentials.NewError(credentials.ErrorVersionConflict, "version")
	}
	store.revoked, store.material = true, ""
	return credentials.MutationResult{Metadata: credentials.Metadata{Version: store.version, Revoked: true}}, nil
}

func assertNoMaterialInTree(t *testing.T, root string, materials ...string) {
	t.Helper()
	if err := filepath.WalkDir(root, func(filePath string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		payload, readErr := os.ReadFile(filePath)
		if readErr != nil {
			return readErr
		}
		for _, material := range materials {
			encoded := [][]byte{
				[]byte(material), []byte(base64.StdEncoding.EncodeToString([]byte(material))),
				[]byte(base64.RawURLEncoding.EncodeToString([]byte(material))), []byte(hex.EncodeToString([]byte(material))),
			}
			for _, signature := range encoded {
				if !bytes.Contains(payload, signature) {
					continue
				}
				t.Fatalf("credential material persisted in %s", filePath)
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("scan runtime files: %v", err)
	}
}
