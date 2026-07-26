package codex

import (
	"context"
	"errors"
	"os"
	"path"
	"path/filepath"
	"testing"

	"orquesta/internal/credentials"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

func TestV7LaunchHashSeparatesReasoningEffortOmittedByV6(t *testing.T) {
	high := testRequest(t, "reasoning-hash", "helper:success", 1024)
	high.ReasoningEffort = governance.ReasoningEffortHigh
	xhigh := high
	xhigh.ReasoningEffort = governance.ReasoningEffortXHigh

	highV6, err := hashV6LaunchRequest(high, "")
	if err != nil {
		t.Fatal(err)
	}
	xhighV6, err := hashV6LaunchRequest(xhigh, "")
	if err != nil {
		t.Fatal(err)
	}
	if highV6 != xhighV6 {
		t.Fatal("historical V6 hash unexpectedly included reasoning effort")
	}
	highV7, err := hashLaunchRequest(high)
	if err != nil {
		t.Fatal(err)
	}
	xhighV7, err := hashLaunchRequest(xhigh)
	if err != nil {
		t.Fatal(err)
	}
	if highV7 == xhighV7 {
		t.Fatal("V7 collapsed high and xhigh into one launch identity")
	}
}

func TestV7LaunchRecordRejectsMissingOrInvalidReasoningEffort(t *testing.T) {
	config := testConfig(t)
	adapter := openTestAdapter(t, config)
	request := testRequest(t, "reasoning-record-validation", "helper:success", 1024)
	record, _, created, err := adapter.ensureLaunchRecord(request, mustRequestHash(t, request))
	if err != nil || !created {
		t.Fatalf("ensure V7 launch created=%v error=%v", created, err)
	}
	for _, effort := range []governance.ReasoningEffort{"", "auto"} {
		t.Run(string(effort), func(t *testing.T) {
			invalid := record
			invalid.ReasoningEffort = effort
			if ErrorCode(validateLaunchRecordV7(invalid)) != CodeStateInvalid {
				t.Fatalf("invalid V7 reasoning effort accepted: %q", effort)
			}
		})
	}
}

func TestV6WithoutPhysicalEffortNeverBindsV7(t *testing.T) {
	config := testConfig(t)
	config.ReasoningEffort = string(governance.ReasoningEffortHigh)
	config.CredentialStore = &credentialTestStore{material: helperCredentialInitial, version: 1}
	config.CredentialRef = credentials.CredentialRef("credential:codex-primary")
	high := testRequest(t, "reasoning-v6-replay", "helper:success", 1024)
	high.ReasoningEffort = governance.ReasoningEffortHigh
	runPath := seedPersistedV6Launch(t, config, high)

	config.ReasoningEffort = string(governance.ReasoningEffortXHigh)
	first := openTestAdapter(t, config)
	xhigh := high
	xhigh.ReasoningEffort = governance.ReasoningEffortXHigh
	if _, err := first.Launch(context.Background(), xhigh); ErrorCode(err) != CodeLegacyExecutionRequiresNewAttempt {
		t.Fatalf("V6 xhigh replay error=%v code=%q", err, ErrorCode(err))
	}
	if _, err := os.Stat(filepath.Join(
		config.WorkRoot, filepath.FromSlash(runPath), launchUpgradeFileName,
	)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("V6 replay persisted V7 binding: %v", err)
	}
	if _, err := first.Launch(context.Background(), high); ErrorCode(err) != CodeLegacyExecutionRequiresNewAttempt {
		t.Fatalf("V6 high replay error=%v code=%q", err, ErrorCode(err))
	}
	observation := awaitTerminal(t, first, high.ExecutionRef)
	if observation.Status != ports.AgentFailed ||
		observation.ErrorCode != CodeExecutionInterrupted {
		t.Fatalf("V6 terminal compatibility=%+v", observation)
	}
	terminal := readPersistedTerminal(t, config, runPath)
	v6Hash, err := hashV6LaunchRequest(high, first.accountProfileRef())
	if err != nil {
		t.Fatal(err)
	}
	if terminal.RequestHash != v6Hash {
		t.Fatalf("V6 terminal changed causal hash: %+v", terminal)
	}
	retry := newV7RetryRequest(t, high, "reasoning-v7-retry", governance.ReasoningEffortHigh)
	retry.Objective = "helper:credential-initial helper:reasoning:high"
	if _, err := first.Launch(context.Background(), retry); err != nil {
		t.Fatalf("Launch(V7 retry): %v", err)
	}
	if retryObservation := awaitTerminal(t, first, retry.ExecutionRef); retryObservation.Status != ports.AgentCompleted {
		t.Fatalf(
			"V7 retry observation=%+v terminal=%+v",
			retryObservation, readPersistedTerminal(t, config, executionPath(retry.ExecutionRef)),
		)
	}
	if _, err := os.Stat(filepath.Join(config.WorkRoot, filepath.FromSlash(runPath), "helper-invocations")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy V6 journal was relaunched: %v", err)
	}
}

func newV7RetryRequest(
	t *testing.T,
	legacy ports.AgentLaunchRequest,
	suffix string,
	effort governance.ReasoningEffort,
) ports.AgentLaunchRequest {
	t.Helper()
	retry := testRequest(
		t, suffix, "helper:reasoning:"+string(effort)+" helper:success", legacy.MaxOutputBytes,
	)
	retry.GoalRef = legacy.GoalRef
	retry.WorkItemRef = legacy.WorkItemRef
	retry.PlanGeneration = legacy.PlanGeneration
	retry.AppSpecGeneration = legacy.AppSpecGeneration
	retry.ExecutionAttempt = legacy.ExecutionAttempt + 1
	retry.SpecHash = legacy.SpecHash
	retry.ActorRef = legacy.ActorRef
	retry.ProjectRef = legacy.ProjectRef
	retry.ReasoningEffort = effort
	return retry
}

func seedPersistedV6Launch(t *testing.T, config Config, request ports.AgentLaunchRequest) string {
	t.Helper()
	adapter, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	runPath := executionPath(request.ExecutionRef)
	if err := adapter.ensurePrivateDirectory("executions"); err != nil {
		t.Fatal(err)
	}
	if err := adapter.ensurePrivateDirectory(runPath); err != nil {
		t.Fatal(err)
	}
	requestHash, err := hashV6LaunchRequest(request, adapter.accountProfileRef())
	if err != nil {
		t.Fatal(err)
	}
	record := launchRecord{
		SchemaVersion:         profileStateSchemaVersion,
		RequestHash:           requestHash,
		ExecutionRef:          request.ExecutionRef.String(),
		ExecutionSessionRef:   request.SessionRef.String(),
		ExecutionWorkspaceRef: request.ExecutionWorkspaceRef.String(),
		AccountProfileRef:     adapter.accountProfileRef(),
		ActorRef:              request.ActorRef.String(),
		ProjectRef:            request.ProjectRef.String(),
		GoalRef:               request.GoalRef.String(),
		WorkItemRef:           request.WorkItemRef.String(),
		PlanGeneration:        request.PlanGeneration,
		AppSpecGeneration:     request.AppSpecGeneration,
		ExecutionAttempt:      request.ExecutionAttempt,
		SpecHash:              request.SpecHash,
		ProviderRef:           ProviderRef,
		ModelRef:              adapter.modelRef(),
		AgentRef:              AgentRef,
		ExternalRef:           "codex:" + path.Base(runPath),
		IdempotencyKey:        request.IdempotencyKey,
		AcceptedAt:            config.Now().UTC(),
		MaxOutputBytes:        request.MaxOutputBytes,
	}
	created, err := adapter.publishJSON(runPath, requestFileName, record)
	if err != nil || !created {
		t.Fatalf("publish V6 request created=%v error=%v", created, err)
	}
	if err := adapter.Close(); err != nil {
		t.Fatal(err)
	}
	return runPath
}

// seed7401V7Upgrade reproduces the complete sidecar format emitted by 7401dcf2:
// it has no provenance beyond the legacy source identity and the copied V7
// replay fields.
func seed7401V7Upgrade(
	t *testing.T,
	config Config,
	request ports.AgentLaunchRequest,
) (string, launchRecord, string) {
	t.Helper()
	runPath := seedPersistedV6Launch(t, config, request)
	adapter, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	source, _, found, err := adapter.loadLaunchRecord(request.ExecutionRef)
	if err != nil || !found {
		t.Fatalf("load V6 source found=%v error=%v", found, err)
	}
	requestHash, err := adapter.hashLaunchRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	bound := source
	bound.SchemaVersion = stateSchemaVersion
	bound.RequestHash = requestHash
	bound.ReasoningEffort = request.ReasoningEffort
	upgrade := launchUpgradeRecord{
		SchemaVersion:       stateSchemaVersion,
		SourceSchemaVersion: source.SchemaVersion,
		SourceRequestHash:   source.RequestHash,
		Launch:              bound,
	}
	if created, err := adapter.publishJSON(
		runPath, launchUpgradeFileName, upgrade,
	); err != nil || !created {
		t.Fatalf("publish 7401 V7 upgrade created=%v error=%v", created, err)
	}
	if err := adapter.Close(); err != nil {
		t.Fatal(err)
	}
	return runPath, source, requestHash
}
