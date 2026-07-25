package codex

import (
	"context"
	"errors"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"

	"orquesta/internal/credentials"
	"orquesta/internal/ports"
)

type persistedLaunchRecordV3 struct {
	SchemaVersion  int       `json:"schema_version"`
	RequestHash    string    `json:"request_hash"`
	ExecutionRef   string    `json:"execution_ref"`
	SpecHash       string    `json:"spec_hash"`
	ProviderRef    string    `json:"provider_ref"`
	ExternalRef    string    `json:"external_ref"`
	IdempotencyKey string    `json:"idempotency_key"`
	AcceptedAt     time.Time `json:"accepted_at"`
	MaxOutputBytes int64     `json:"max_output_bytes"`
}

func TestAdapterUpgradeV3HashMatchesHistoricalSchemaExactly(t *testing.T) {
	request := testRequest(t, "upgrade-v3-hash", "helper:success", 1024)
	got, err := hashLegacyLaunchRequest(request)
	if err != nil {
		t.Fatalf("hashLegacyLaunchRequest() error = %v", err)
	}
	const want = "sha256:2c02fb94094b0e4c8fab13718ad3fe71758c664eed64124cb696eae6a17b57f1"
	if got != want {
		t.Fatalf("historical V3 hash = %q, want %q", got, want)
	}
}

func TestAdapterUpgradeV3ObservePreservesDurableTerminalWithoutRelaunch(t *testing.T) {
	config := testConfig(t)
	request := testRequest(t, "upgrade-v3-terminal", "helper:success", 1024)
	requestHash, runPath := seedPersistedV3Execution(t, config, request, &terminalRecord{
		SchemaVersion: legacyStateSchemaVersion,
		Status:        ports.AgentCompleted,
		MediaType:     request.ArtifactMediaType,
		Artifact:      "artifact:v3-terminal",
		ObservedAt:    config.Now().Add(time.Minute),
	})

	first := openTestAdapter(t, config)
	firstObservation, err := first.Observe(context.Background(), request.ExecutionRef)
	if err != nil {
		t.Fatalf("Observe(V3 terminal) error = %v", err)
	}
	if firstObservation.Status != ports.AgentCompleted || string(firstObservation.Content) != "artifact:v3-terminal" ||
		firstObservation.SpecHash != request.SpecHash {
		t.Fatalf("V3 terminal observation = %+v", firstObservation)
	}
	assertExecutionCacheSize(t, first, 0)

	second := openTestAdapter(t, config)
	secondObservation, err := second.Observe(context.Background(), request.ExecutionRef)
	if err != nil {
		t.Fatalf("Observe(V3 terminal replay) error = %v", err)
	}
	if !reflect.DeepEqual(secondObservation, firstObservation) {
		t.Fatalf("V3 terminal changed on replay: got=%+v want=%+v", secondObservation, firstObservation)
	}
	if _, err := os.Stat(filepath.Join(config.WorkRoot, filepath.FromSlash(runPath), "helper-invocations")); !os.IsNotExist(err) {
		t.Fatalf("V3 terminal was relaunched: stat error = %v", err)
	}
	terminal := readPersistedTerminal(t, config, runPath)
	if terminal.SchemaVersion != legacyStateSchemaVersion || terminal.RequestHash != requestHash {
		t.Fatalf("historical terminal was replaced: %+v", terminal)
	}
}

func TestAdapterUpgradeV3TerminalLaunchAuthorizesAndBinds(t *testing.T) {
	config := testConfig(t)
	request := testRequest(t, "upgrade-v3-terminal-launch", "helper:success", 1024)
	request.SessionRef, _ = ports.NewExecutionSessionRef("execution-session:upgrade-v3-terminal-launch")
	_, runPath := seedPersistedV3Execution(t, config, request, &terminalRecord{
		SchemaVersion: legacyStateSchemaVersion,
		Status:        ports.AgentCompleted,
		MediaType:     "text/plain",
		Artifact:      "historical terminal",
		ObservedAt:    config.Now().UTC().Add(time.Second),
	})
	config.SessionResolver = sessionResolverFunc(func(_ context.Context, candidate ports.AgentLaunchRequest) (Session, error) {
		if candidate.SessionRef != request.SessionRef || candidate.PlanGeneration != request.PlanGeneration {
			return Session{}, errors.New("authority unavailable")
		}
		secret, _ := credentials.NewSecret([]byte(helperSessionBearer))
		return Session{Ref: candidate.SessionRef, Endpoint: "http://127.0.0.1:7777/mcp", BearerToken: secret}, nil
	})
	adapter := openTestAdapter(t, config)
	untrusted := request
	untrusted.SessionRef, _ = ports.NewExecutionSessionRef("execution-session:upgrade-v3-terminal-attacker")
	untrusted.PlanGeneration++
	if _, err := adapter.Launch(context.Background(), untrusted); ErrorCode(err) != CodeSessionUnavailable {
		t.Fatalf("Launch(untrusted V3 terminal) error = %v, code = %q", err, ErrorCode(err))
	}
	upgradePath := filepath.Join(config.WorkRoot, filepath.FromSlash(runPath), launchUpgradeFileName)
	if _, err := os.Stat(upgradePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("untrusted terminal replay created upgrade: %v", err)
	}
	receipt, err := adapter.Launch(context.Background(), request)
	if err != nil {
		t.Fatalf("Launch(authorized V3 terminal) error = %v", err)
	}
	if err := ports.ValidateAgentLaunchReceipt(request, receipt); err != nil {
		t.Fatalf("Launch(V3 terminal) receipt = %+v, error = %v", receipt, err)
	}
	observation, err := adapter.Observe(context.Background(), request.ExecutionRef)
	if err != nil || observation.Status != ports.AgentCompleted || string(observation.Content) != "historical terminal" {
		t.Fatalf("Observe(V3 terminal) = %+v, %v", observation, err)
	}
	upgrade := readPersistedV3Upgrade(t, config, runPath)
	if upgrade.Launch.RequestHash != mustRequestHash(t, request) ||
		upgrade.Launch.ExecutionSessionRef != request.SessionRef.String() ||
		upgrade.Launch.PlanGeneration != request.PlanGeneration {
		t.Fatalf("terminal binding = %+v", upgrade)
	}
}

func TestAdapterUpgradeV3ConcurrentTerminalBindingHasSingleWinner(t *testing.T) {
	config := testConfig(t)
	config.SessionResolver = sessionResolverFunc(func(_ context.Context, request ports.AgentLaunchRequest) (Session, error) {
		secret, _ := credentials.NewSecret([]byte(helperSessionBearer))
		return Session{Ref: request.SessionRef, Endpoint: "http://127.0.0.1:7777/mcp", BearerToken: secret}, nil
	})
	firstRequest := testRequest(t, "upgrade-v3-terminal-race", "helper:success", 1024)
	firstRequest.SessionRef, _ = ports.NewExecutionSessionRef("execution-session:upgrade-v3-terminal-race")
	legacyHash, runPath := seedPersistedV3Execution(t, config, firstRequest, &terminalRecord{
		SchemaVersion: legacyStateSchemaVersion,
		Status:        ports.AgentCompleted,
		MediaType:     "text/plain",
		Artifact:      "historical terminal",
		ObservedAt:    config.Now().UTC().Add(time.Second),
	})
	secondRequest := firstRequest
	secondRequest.PlanGeneration++
	requests := []ports.AgentLaunchRequest{firstRequest, secondRequest}
	adapters := make([]*Adapter, len(requests))
	for index := range adapters {
		var err error
		adapters[index], err = New(config)
		if err != nil {
			t.Fatalf("New(adapter %d) error = %v", index, err)
		}
		defer func(adapter *Adapter) { _ = adapter.Close() }(adapters[index])
	}
	type result struct {
		receipt ports.AgentLaunchReceipt
		err     error
	}
	results := make([]result, len(requests))
	start := make(chan struct{})
	var group sync.WaitGroup
	for index := range requests {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			<-start
			results[index].receipt, results[index].err = adapters[index].Launch(context.Background(), requests[index])
		}(index)
	}
	close(start)
	group.Wait()
	winner := -1
	for index, result := range results {
		if result.err == nil {
			if winner != -1 {
				t.Fatalf("multiple terminal bindings won: results=%+v", results)
			}
			winner = index
		} else if ErrorCode(result.err) != CodeExecutionConflict {
			t.Fatalf("loser error=%v code=%q", result.err, ErrorCode(result.err))
		}
	}
	if winner == -1 {
		t.Fatalf("no terminal binding won: results=%+v", results)
	}
	upgrade := readPersistedV3Upgrade(t, config, runPath)
	if upgrade.Launch.RequestHash != mustRequestHash(t, requests[winner]) {
		t.Fatalf("terminal binding winner=%+v", upgrade)
	}
	reopened := openTestAdapter(t, config)
	replayed, err := reopened.Launch(context.Background(), requests[winner])
	if err != nil || replayed != results[winner].receipt {
		t.Fatalf("winner replay receipt=%+v error=%v want=%+v", replayed, err, results[winner].receipt)
	}
	loser := 1 - winner
	if _, err := reopened.Launch(context.Background(), requests[loser]); ErrorCode(err) != CodeExecutionConflict {
		t.Fatalf("loser replay error=%v code=%q", err, ErrorCode(err))
	}
	terminal := readPersistedTerminal(t, config, runPath)
	if terminal.RequestHash != legacyHash || terminal.SchemaVersion != legacyStateSchemaVersion {
		t.Fatalf("terminal causal identity changed: %+v", terminal)
	}
}

func TestAdapterUpgradeV3LaunchWithoutPhysicalAuthorityCreatesNoUpgrade(t *testing.T) {
	config := testConfig(t)
	request := testRequest(t, "upgrade-v3-no-authority", "helper:success", 1024)
	_, runPath := seedPersistedV3Execution(t, config, request, &terminalRecord{
		SchemaVersion: legacyStateSchemaVersion,
		Status:        ports.AgentCompleted,
		MediaType:     "text/plain",
		Artifact:      "historical terminal",
		ObservedAt:    config.Now().UTC().Add(time.Second),
	})
	adapter := openTestAdapter(t, config)
	if _, err := adapter.Launch(context.Background(), request); ErrorCode(err) != CodeSessionUnavailable {
		t.Fatalf("Launch(V3 without authority) error = %v, code = %q", err, ErrorCode(err))
	}
	upgradePath := filepath.Join(config.WorkRoot, filepath.FromSlash(runPath), launchUpgradeFileName)
	if _, err := os.Stat(upgradePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("authority-free replay created upgrade: %v", err)
	}
}

func TestAdapterUpgradeV3ObservePersistsInterruptedWithoutRelaunch(t *testing.T) {
	config := testConfig(t)
	request := testRequest(t, "upgrade-v3-observe-interrupted", "helper:success", 1024)
	requestHash, runPath := seedPersistedV3Execution(t, config, request, nil)

	first := openTestAdapter(t, config)
	firstObservation, err := first.Observe(context.Background(), request.ExecutionRef)
	if err != nil {
		t.Fatalf("Observe(V3 dispatching) error = %v", err)
	}
	if firstObservation.Status != ports.AgentFailed || firstObservation.ErrorCode != CodeExecutionInterrupted ||
		firstObservation.SpecHash != request.SpecHash {
		t.Fatalf("V3 interrupted observation = %+v", firstObservation)
	}
	terminal := readPersistedTerminal(t, config, runPath)
	if terminal.SchemaVersion != stateSchemaVersion || terminal.RequestHash != requestHash {
		t.Fatalf("durable interrupted terminal = %+v", terminal)
	}
	if _, err := os.Stat(filepath.Join(config.WorkRoot, filepath.FromSlash(runPath), launchUpgradeFileName)); !os.IsNotExist(err) {
		t.Fatalf("Observe invented V4 request metadata: stat error = %v", err)
	}

	second := openTestAdapter(t, config)
	secondObservation, err := second.Observe(context.Background(), request.ExecutionRef)
	if err != nil || !reflect.DeepEqual(secondObservation, firstObservation) {
		t.Fatalf("interrupted replay = %+v error=%v want=%+v", secondObservation, err, firstObservation)
	}
	if _, err := os.Stat(filepath.Join(config.WorkRoot, filepath.FromSlash(runPath), "helper-invocations")); !os.IsNotExist(err) {
		t.Fatalf("V3 dispatching execution was relaunched by Observe: stat error = %v", err)
	}
}

func TestAdapterUpgradeV3LaunchBindsV4ReceiptAndInterruptedReplay(t *testing.T) {
	config := testConfig(t)
	config.CredentialStore = &credentialTestStore{material: helperCredentialInitial, version: 1}
	config.CredentialRef = credentials.CredentialRef("credential:codex-primary")
	request := testRequest(t, "upgrade-v3-dispatching", "helper:success", 1024)
	legacyHash, runPath := seedPersistedV3Execution(t, config, request, nil)
	if err := os.WriteFile(filepath.Join(config.WorkRoot, filepath.FromSlash(runPath), lastMessageFileName), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	first := openTestAdapter(t, config)
	firstReceipt, err := first.Launch(context.Background(), request)
	if err != nil {
		t.Fatalf("Launch(V3 dispatching) error = %v", err)
	}
	if err := ports.ValidateAgentLaunchReceipt(request, firstReceipt); err != nil {
		t.Fatalf("synthesized V4 receipt error = %v; receipt=%+v", err, firstReceipt)
	}
	if firstReceipt.ModelRef != DefaultModelRef || firstReceipt.AgentRef != AgentRef {
		t.Fatalf("synthesized provider identity = %+v", firstReceipt)
	}
	firstObservation := awaitTerminal(t, first, request.ExecutionRef)
	if firstObservation.Status != ports.AgentFailed || firstObservation.ErrorCode != CodeExecutionInterrupted {
		t.Fatalf("V3 dispatching recovery = %+v", firstObservation)
	}

	config.Now = func() time.Time { return time.Date(2032, 1, 2, 3, 4, 5, 0, time.UTC) }
	second := openTestAdapter(t, config)
	secondReceipt, err := second.Launch(context.Background(), request)
	if err != nil {
		t.Fatalf("Launch(V3 replay) error = %v", err)
	}
	if secondReceipt != firstReceipt {
		t.Fatalf("V3 synthesized receipt changed: got=%+v want=%+v", secondReceipt, firstReceipt)
	}
	secondObservation := awaitTerminal(t, second, request.ExecutionRef)
	if !reflect.DeepEqual(secondObservation, firstObservation) {
		t.Fatalf("V3 interrupted terminal changed: got=%+v want=%+v", secondObservation, firstObservation)
	}

	conflicting := request
	conflicting.PlanGeneration++
	if _, err := second.Launch(context.Background(), conflicting); ErrorCode(err) != CodeExecutionConflict {
		t.Fatalf("Launch(conflicting V4 binding) error = %v, code = %q", err, ErrorCode(err))
	}
	upgrade := readPersistedV3Upgrade(t, config, runPath)
	currentHash, err := hashLaunchRequest(request)
	if err != nil {
		t.Fatalf("hashLaunchRequest() error = %v", err)
	}
	if upgrade.SourceRequestHash != legacyHash || upgrade.Launch.RequestHash != currentHash ||
		upgrade.Launch.GoalRef != request.GoalRef.String() || upgrade.Launch.PlanGeneration != request.PlanGeneration {
		t.Fatalf("V3->V4 binding = %+v", upgrade)
	}
	terminal := readPersistedTerminal(t, config, runPath)
	if terminal.SchemaVersion != stateSchemaVersion || terminal.RequestHash != legacyHash {
		t.Fatalf("interrupted terminal lost V3 causal hash: %+v", terminal)
	}
	if _, err := os.Stat(filepath.Join(config.WorkRoot, filepath.FromSlash(runPath), "helper-invocations")); !os.IsNotExist(err) {
		t.Fatalf("V3 dispatching execution was relaunched: stat error = %v", err)
	}
}

func TestAdapterUpgradeV3ConcurrentBindingHasSingleDurableWinner(t *testing.T) {
	config := testConfig(t)
	config.CredentialStore = &credentialTestStore{material: helperCredentialInitial, version: 1}
	config.CredentialRef = credentials.CredentialRef("credential:codex-primary")
	firstRequest := testRequest(t, "upgrade-v3-race", "helper:success", 1024)
	_, runPath := seedPersistedV3Execution(t, config, firstRequest, nil)
	if err := os.WriteFile(filepath.Join(config.WorkRoot, filepath.FromSlash(runPath), lastMessageFileName), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	secondRequest := firstRequest
	secondRequest.PlanGeneration++
	requests := []ports.AgentLaunchRequest{firstRequest, secondRequest}

	adapters := make([]*Adapter, len(requests))
	for index := range adapters {
		var err error
		adapters[index], err = New(config)
		if err != nil {
			t.Fatalf("New(adapter %d) error = %v", index, err)
		}
		defer func(adapter *Adapter) { _ = adapter.Close() }(adapters[index])
	}
	type result struct {
		receipt ports.AgentLaunchReceipt
		err     error
	}
	results := make([]result, len(requests))
	start := make(chan struct{})
	var group sync.WaitGroup
	for index := range requests {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			<-start
			results[index].receipt, results[index].err = adapters[index].Launch(context.Background(), requests[index])
		}(index)
	}
	close(start)
	group.Wait()

	winner := -1
	for index, result := range results {
		if result.err == nil {
			if winner != -1 {
				t.Fatalf("multiple V4 bindings won: results=%+v", results)
			}
			winner = index
			continue
		}
		if ErrorCode(result.err) != CodeExecutionConflict {
			t.Fatalf("loser error = %v, code = %q", result.err, ErrorCode(result.err))
		}
	}
	if winner == -1 {
		t.Fatalf("no V4 binding won: results=%+v", results)
	}
	observation := awaitTerminal(t, adapters[winner], requests[winner].ExecutionRef)
	if observation.Status != ports.AgentFailed || observation.ErrorCode != CodeExecutionInterrupted {
		t.Fatalf("winner recovery = %+v", observation)
	}

	reopened := openTestAdapter(t, config)
	replayed, err := reopened.Launch(context.Background(), requests[winner])
	if err != nil || replayed != results[winner].receipt {
		t.Fatalf("winner replay receipt=%+v error=%v want=%+v", replayed, err, results[winner].receipt)
	}
	loser := 1 - winner
	if _, err := reopened.Launch(context.Background(), requests[loser]); ErrorCode(err) != CodeExecutionConflict {
		t.Fatalf("loser replay error = %v, code = %q", err, ErrorCode(err))
	}
}

func TestAdapterUpgradeRejectsTerminalSchemasOutsideV3V4V5AndV6(t *testing.T) {
	for _, schemaVersion := range []int{2, 7} {
		t.Run(strconv.Itoa(schemaVersion), func(t *testing.T) {
			config := testConfig(t)
			request := testRequest(t, "upgrade-terminal-schema", "helper:success", 1024)
			_, _ = seedPersistedV3Execution(t, config, request, &terminalRecord{
				SchemaVersion: schemaVersion,
				Status:        ports.AgentFailed,
				ErrorCode:     CodeExecutionInterrupted,
				ObservedAt:    config.Now(),
			})
			adapter := openTestAdapter(t, config)
			if _, err := adapter.Observe(context.Background(), request.ExecutionRef); ErrorCode(err) != CodeStateInvalid {
				t.Fatalf("Observe(schema %d) error=%v code=%q", schemaVersion, err, ErrorCode(err))
			}
		})
	}
}

func seedPersistedV3Execution(
	t *testing.T,
	config Config,
	request ports.AgentLaunchRequest,
	terminal *terminalRecord,
) (string, string) {
	t.Helper()
	adapter, err := New(config)
	if err != nil {
		t.Fatalf("New(V3 seeder) error = %v", err)
	}
	runPath := executionPath(request.ExecutionRef)
	if err := adapter.ensurePrivateDirectory("executions"); err != nil {
		t.Fatalf("ensure executions directory error = %v", err)
	}
	if err := adapter.ensurePrivateDirectory(runPath); err != nil {
		t.Fatalf("ensure execution directory error = %v", err)
	}
	requestHash, err := hashLegacyLaunchRequest(request)
	if err != nil {
		t.Fatalf("hashLegacyLaunchRequest() error = %v", err)
	}
	record := persistedLaunchRecordV3{
		SchemaVersion:  legacyStateSchemaVersion,
		RequestHash:    requestHash,
		ExecutionRef:   request.ExecutionRef.String(),
		SpecHash:       request.SpecHash,
		ProviderRef:    ProviderRef,
		ExternalRef:    "codex:" + path.Base(runPath),
		IdempotencyKey: request.IdempotencyKey,
		AcceptedAt:     config.Now().UTC(),
		MaxOutputBytes: request.MaxOutputBytes,
	}
	created, err := adapter.publishJSON(runPath, requestFileName, record)
	if err != nil || !created {
		t.Fatalf("publish V3 request created=%v error=%v", created, err)
	}
	if terminal != nil {
		terminal.RequestHash = requestHash
		created, err = adapter.publishJSON(runPath, terminalFileName, *terminal)
		if err != nil || !created {
			t.Fatalf("publish V3 terminal created=%v error=%v", created, err)
		}
	}
	if err := adapter.Close(); err != nil {
		t.Fatalf("Close(V3 seeder) error = %v", err)
	}
	return requestHash, runPath
}

func readPersistedV3Upgrade(t *testing.T, config Config, runPath string) launchUpgradeRecord {
	t.Helper()
	adapter, err := New(config)
	if err != nil {
		t.Fatalf("New(upgrade reader) error = %v", err)
	}
	defer func() { _ = adapter.Close() }()
	var upgrade launchUpgradeRecord
	found, err := adapter.readPrivateJSON(path.Join(runPath, launchUpgradeFileName), &upgrade)
	if err != nil || !found {
		t.Fatalf("read upgrade found=%v error=%v", found, err)
	}
	return upgrade
}

func readPersistedTerminal(t *testing.T, config Config, runPath string) terminalRecord {
	t.Helper()
	adapter, err := New(config)
	if err != nil {
		t.Fatalf("New(terminal reader) error = %v", err)
	}
	defer func() { _ = adapter.Close() }()
	var terminal terminalRecord
	found, err := adapter.readPrivateJSON(path.Join(runPath, terminalFileName), &terminal)
	if err != nil || !found {
		t.Fatalf("read terminal found=%v error=%v", found, err)
	}
	return terminal
}
