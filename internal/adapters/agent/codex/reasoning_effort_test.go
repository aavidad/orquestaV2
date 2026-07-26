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

func TestV6ReplayBindsOneExactReasoningEffortDurably(t *testing.T) {
	config := testConfig(t)
	config.ReasoningEffort = string(governance.ReasoningEffortHigh)
	config.CredentialStore = &credentialTestStore{material: helperCredentialInitial, version: 1}
	config.CredentialRef = credentials.CredentialRef("credential:codex-primary")
	high := testRequest(t, "reasoning-v6-replay", "helper:success", 1024)
	high.ReasoningEffort = governance.ReasoningEffortHigh
	runPath := seedPersistedV6Launch(t, config, high)

	first := openTestAdapter(t, config)
	xhigh := high
	xhigh.ReasoningEffort = governance.ReasoningEffortXHigh
	if _, err := first.Launch(context.Background(), xhigh); ErrorCode(err) != CodeExecutionConflict {
		t.Fatalf("V6 replay changed historical effort: error=%v code=%q", err, ErrorCode(err))
	}
	if _, err := os.Stat(filepath.Join(
		config.WorkRoot, filepath.FromSlash(runPath), launchUpgradeFileName,
	)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("conflicting effort persisted V7 binding: %v", err)
	}
	firstReceipt, err := first.Launch(context.Background(), high)
	if err != nil {
		t.Fatalf("Launch(V6 high) error = %v", err)
	}
	firstObservation := awaitTerminal(t, first, high.ExecutionRef)
	if firstObservation.Status != ports.AgentFailed ||
		firstObservation.ErrorCode != CodeCredentialOutputUnverifiable {
		t.Fatalf("V6 prestart recovery = %+v", firstObservation)
	}
	var upgrade launchUpgradeRecord
	found, err := first.readPrivateJSON(path.Join(runPath, launchUpgradeFileName), &upgrade)
	if err != nil || !found {
		t.Fatalf("read V7 upgrade found=%v error=%v", found, err)
	}
	if upgrade.SchemaVersion != stateSchemaVersion ||
		upgrade.SourceSchemaVersion != profileStateSchemaVersion ||
		upgrade.Launch.ReasoningEffort != governance.ReasoningEffortHigh ||
		upgrade.Launch.RequestHash != mustRequestHash(t, high) {
		t.Fatalf("V6 -> V7 reasoning binding = %+v", upgrade)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}

	second := openTestAdapter(t, config)
	replayed, err := second.Launch(context.Background(), high)
	if err != nil || replayed != firstReceipt {
		t.Fatalf("V7 restart replay receipt=%+v error=%v want=%+v", replayed, err, firstReceipt)
	}
	replayedObservation := awaitTerminal(t, second, high.ExecutionRef)
	if replayedObservation.Status != firstObservation.Status ||
		replayedObservation.ErrorCode != firstObservation.ErrorCode {
		t.Fatalf("V7 restart observation=%+v want=%+v", replayedObservation, firstObservation)
	}

	if _, err := second.Launch(context.Background(), xhigh); ErrorCode(err) != CodeExecutionConflict {
		t.Fatalf("conflicting xhigh replay error=%v code=%q", err, ErrorCode(err))
	}
	if _, err := os.Stat(filepath.Join(config.WorkRoot, filepath.FromSlash(runPath), "helper-invocations")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("prestart V6 journal was relaunched: %v", err)
	}
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
