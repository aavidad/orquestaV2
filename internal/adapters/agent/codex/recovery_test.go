package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"orquesta/internal/credentials"
	"orquesta/internal/ports"
)

type recoverySessionResolver struct {
	material    string
	unavailable bool
	mismatch    bool
}

func (resolver *recoverySessionResolver) ResolveCodexSession(_ context.Context, request ports.AgentLaunchRequest) (Session, error) {
	return resolver.session(request.SessionRef)
}

func (resolver *recoverySessionResolver) RecoverCodexSession(_ context.Context, request ports.AgentLaunchRequest) (Session, error) {
	return resolver.session(request.SessionRef)
}

func (resolver *recoverySessionResolver) session(ref ports.ExecutionSessionRef) (Session, error) {
	if resolver.unavailable {
		return Session{}, errors.New("recovery unavailable")
	}
	if resolver.mismatch {
		ref, _ = ports.NewExecutionSessionRef("execution-session:mismatch")
	}
	secret, _ := credentials.NewSecret([]byte(resolver.material))
	return Session{Ref: ref, Endpoint: "http://127.0.0.1:7777/mcp", BearerToken: secret}, nil
}

func TestCredentialRecoveryScrubsUntrustedLastMessage(t *testing.T) {
	config := testConfig(t)
	config.CredentialStore = &credentialTestStore{material: helperCredentialInitial, version: 1}
	config.CredentialRef = credentials.CredentialRef("credential:codex-primary")
	request := testRequest(t, "credential-interrupted-output", "helper:success", 1024)
	_, runPath := seedAcceptedExecution(t, config, request)
	lastMessagePath := filepath.Join(config.WorkRoot, filepath.FromSlash(runPath), lastMessageFileName)
	if err := os.WriteFile(lastMessagePath, []byte(helperCredentialInitial), 0o600); err != nil {
		t.Fatal(err)
	}

	adapter := openTestAdapter(t, config)
	observation, err := adapter.Observe(context.Background(), request.ExecutionRef)
	if ErrorCode(err) != CodeSessionUnavailable || !reflect.DeepEqual(observation, ports.AgentObservation{}) {
		t.Fatalf("recovery observation=%+v err=%v", observation, err)
	}
	payload, err := os.ReadFile(lastMessagePath)
	if err != nil || bytes.Contains(payload, []byte(helperCredentialInitial)) || len(payload) != 0 {
		t.Fatalf("interrupted credential output survived recovery: %q err=%v", payload, err)
	}
}

func TestObserveRecoveryAuthorityFailureScrubsAndRejectsOutput(t *testing.T) {
	for _, test := range []struct {
		name string
		code string
		set  func(*recoverySessionResolver)
	}{
		{"unavailable", CodeSessionUnavailable, func(value *recoverySessionResolver) { value.unavailable = true }},
		{"mismatch", CodeSessionInvalid, func(value *recoverySessionResolver) { value.mismatch = true }},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := testConfig(t)
			resolver := &recoverySessionResolver{material: helperSessionBearer}
			test.set(resolver)
			config.SessionResolver = resolver
			request := testRequest(t, "guard-recovery-"+test.name, "helper:success", 1024)
			request.SessionRef, _ = ports.NewExecutionSessionRef("execution-session:guard-recovery-" + test.name)
			_, runPath := seedAcceptedExecution(t, config, request)
			output := filepath.Join(config.WorkRoot, filepath.FromSlash(runPath), lastMessageFileName)
			if err := os.WriteFile(output, []byte(helperSessionBearer), 0o600); err != nil {
				t.Fatal(err)
			}
			adapter := openTestAdapter(t, config)
			observation, err := adapter.Observe(context.Background(), request.ExecutionRef)
			if ErrorCode(err) != test.code || !reflect.DeepEqual(observation, ports.AgentObservation{}) {
				t.Fatalf("Observe() observation=%+v error=%v", observation, err)
			}
			if payload, readErr := os.ReadFile(output); readErr != nil || len(payload) != 0 {
				t.Fatalf("rejected output=%q error=%v", payload, readErr)
			}
		})
	}
}

func TestAdapterRejectsPersistedSpecHashThatDoesNotEchoRequest(t *testing.T) {
	config := testConfig(t)
	request := testRequest(t, "tampered-spec-hash", "helper:success", 1024)
	_, runPath := seedAcceptedExecution(t, config, request)
	recordPath := filepath.Join(config.WorkRoot, filepath.FromSlash(runPath), requestFileName)
	payload, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatalf("ReadFile(request record) error = %v", err)
	}
	var record launchRecord
	if err := json.Unmarshal(payload, &record); err != nil {
		t.Fatalf("Unmarshal(request record) error = %v", err)
	}
	record.SpecHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	payload, err = json.Marshal(record)
	if err != nil {
		t.Fatalf("Marshal(request record) error = %v", err)
	}
	if err := os.WriteFile(recordPath, payload, 0o600); err != nil {
		t.Fatalf("WriteFile(request record) error = %v", err)
	}

	adapter := openTestAdapter(t, config)
	if _, err := adapter.Launch(context.Background(), request); ErrorCode(err) != CodeStateInvalid {
		t.Fatalf("Launch(tampered spec hash) error = %v, code = %q", err, ErrorCode(err))
	}
}

func TestAdapterLaunchRecoversPersistedExecutionWithoutRelaunch(t *testing.T) {
	config := testConfig(t)
	request := testRequest(t, "interrupted-launch", "helper:success", 1024)
	wantReceipt, runPath := seedAcceptedExecution(t, config, request)
	invocationPath := filepath.Join(config.WorkRoot, filepath.FromSlash(runPath), "helper-invocations")
	if err := os.WriteFile(invocationPath, []byte{'1'}, 0o600); err != nil {
		t.Fatalf("WriteFile(invocation marker) error = %v", err)
	}

	recoveredAt := config.Now().Add(time.Hour)
	config.Now = func() time.Time { return recoveredAt }
	adapter := openTestAdapter(t, config)
	gotReceipt, err := adapter.Launch(context.Background(), request)
	if err != nil {
		t.Fatalf("Launch(restarted) error = %v", err)
	}
	if gotReceipt != wantReceipt {
		t.Fatalf("receipt changed: got=%+v want=%+v", gotReceipt, wantReceipt)
	}
	observation := awaitTerminal(t, adapter, request.ExecutionRef)
	if observation.Status != ports.AgentFailed || observation.ErrorCode != CodeExecutionInterrupted {
		t.Fatalf("recovered observation = %+v", observation)
	}
	if gotReceipt.SpecHash != request.SpecHash || observation.SpecHash != request.SpecHash {
		t.Fatalf("spec hash lost during launch recovery: receipt=%q observation=%q want=%q", gotReceipt.SpecHash, observation.SpecHash, request.SpecHash)
	}

	repeatedReceipt, err := adapter.Launch(context.Background(), request)
	if err != nil {
		t.Fatalf("Launch(repeated) error = %v", err)
	}
	if repeatedReceipt != wantReceipt {
		t.Fatalf("repeated receipt changed: got=%+v want=%+v", repeatedReceipt, wantReceipt)
	}
	repeatedObservation := awaitTerminal(t, adapter, request.ExecutionRef)
	if !reflect.DeepEqual(repeatedObservation, observation) {
		t.Fatalf("repeated observation changed: got=%+v want=%+v", repeatedObservation, observation)
	}
	invocations, err := os.ReadFile(invocationPath)
	if err != nil {
		t.Fatalf("ReadFile(invocation marker) error = %v", err)
	}
	if string(invocations) != "1" {
		t.Fatalf("helper invocations = %q, want one pre-restart invocation", invocations)
	}
}

func TestAdapterObserveRecoversPersistedExecutionAsStableTerminal(t *testing.T) {
	config := testConfig(t)
	request := testRequest(t, "interrupted-observe", "helper:success", 1024)
	_, _ = seedAcceptedExecution(t, config, request)

	recoveredAt := config.Now().Add(2 * time.Hour)
	config.Now = func() time.Time { return recoveredAt }
	first := openTestAdapter(t, config)
	firstObservation, err := first.Observe(context.Background(), request.ExecutionRef)
	if err != nil {
		t.Fatalf("Observe(first restart) error = %v", err)
	}
	if firstObservation.Status != ports.AgentFailed || firstObservation.ErrorCode != CodeExecutionInterrupted {
		t.Fatalf("first recovered observation = %+v", firstObservation)
	}
	if firstObservation.SpecHash != request.SpecHash {
		t.Fatalf("first recovered spec hash = %q, want %q", firstObservation.SpecHash, request.SpecHash)
	}
	assertExecutionCacheSize(t, first, 0)
	if err := first.Close(); err != nil {
		t.Fatalf("Close(first restart) error = %v", err)
	}

	config.Now = func() time.Time { return recoveredAt.Add(24 * time.Hour) }
	second := openTestAdapter(t, config)
	secondObservation, err := second.Observe(context.Background(), request.ExecutionRef)
	if err != nil {
		t.Fatalf("Observe(second restart) error = %v", err)
	}
	if !reflect.DeepEqual(secondObservation, firstObservation) {
		t.Fatalf("persisted interrupted terminal changed: got=%+v want=%+v", secondObservation, firstObservation)
	}
	assertExecutionCacheSize(t, second, 0)
}

func TestAdapterObserveEvictsManyDurableTerminalPayloads(t *testing.T) {
	config := testConfig(t)
	seeder, err := New(config)
	if err != nil {
		t.Fatalf("New(seeder) error = %v", err)
	}
	const executionCount = 40
	payload := strings.Repeat("artifact-payload-", 256)
	requests := make([]ports.AgentLaunchRequest, 0, executionCount)
	for index := 0; index < executionCount; index++ {
		request := testRequest(t, fmt.Sprintf("terminal-eviction-%d", index), "helper:success", 8192)
		requestHash := mustRequestHash(t, request)
		_, runPath, created, err := seeder.ensureLaunchRecord(request, requestHash)
		if err != nil || !created {
			t.Fatalf("ensureLaunchRecord(%d) created=%v error=%v", index, created, err)
		}
		_, err = seeder.persistTerminal(runPath, terminalRecord{
			SchemaVersion: stateSchemaVersion,
			RequestHash:   requestHash,
			Status:        ports.AgentCompleted,
			MediaType:     request.ArtifactMediaType,
			Artifact:      payload,
			ObservedAt:    config.Now(),
		}, request.SpecHash, request.MaxOutputBytes)
		if err != nil {
			t.Fatalf("persistTerminal(%d) error = %v", index, err)
		}
		requests = append(requests, request)
	}
	if err := seeder.Close(); err != nil {
		t.Fatalf("Close(seeder) error = %v", err)
	}

	adapter := openTestAdapter(t, config)
	for index, request := range requests {
		observation, err := adapter.Observe(context.Background(), request.ExecutionRef)
		if err != nil {
			t.Fatalf("Observe(%d) error = %v", index, err)
		}
		if observation.Status != ports.AgentCompleted || string(observation.Content) != payload {
			t.Fatalf("Observe(%d) = %+v", index, observation)
		}
	}
	assertExecutionCacheSize(t, adapter, 0)

	repeated, err := adapter.Observe(context.Background(), requests[0].ExecutionRef)
	if err != nil {
		t.Fatalf("Observe(repeated) error = %v", err)
	}
	if repeated.Status != ports.AgentCompleted || string(repeated.Content) != payload {
		t.Fatalf("repeated observation = %+v", repeated)
	}
	assertExecutionCacheSize(t, adapter, 0)
}

func seedAcceptedExecution(t *testing.T, config Config, request ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, string) {
	t.Helper()
	adapter, err := New(config)
	if err != nil {
		t.Fatalf("New(seed adapter) error = %v", err)
	}
	requestHash := mustRequestHash(t, request)
	record, runPath, created, err := adapter.ensureLaunchRecord(request, requestHash)
	if err != nil || !created {
		t.Fatalf("ensureLaunchRecord() created=%v error=%v", created, err)
	}
	receipt, err := record.receipt(request.ExecutionRef)
	if err != nil {
		t.Fatalf("receipt() error = %v", err)
	}
	if err := adapter.Close(); err != nil {
		t.Fatalf("Close(seed adapter) error = %v", err)
	}
	return receipt, runPath
}

func assertExecutionCacheSize(t *testing.T, adapter *Adapter, want int) {
	t.Helper()
	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	if got := len(adapter.executions); got != want {
		t.Fatalf("cached executions = %d, want %d", got, want)
	}
}
