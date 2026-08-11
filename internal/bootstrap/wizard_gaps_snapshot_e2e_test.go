package bootstrap

import (
	"bytes"
	"testing"
)

func TestV23WizardGapsPublicReplayReturnsExactHistoricalEvaluation(t *testing.T) {
	root := t.TempDir()
	configPath := writeTestConfig(t, root)
	firstRuntime := buildV23IntakeRuntime(t, configPath)
	principal, hierarchy, err := localIdentityComposition(firstRuntime.config)
	if err != nil {
		t.Fatal(err)
	}
	projectRef := hierarchy.ProjectRef().String()
	const intakeRef = "intake:v23-wizard-snapshot-public"
	dispatchV23IntakeCommand(t, firstRuntime, principal, projectRef,
		"orquesta.intakes.create", "request:v23-wizard-snapshot-public-create",
		map[string]any{"intake_ref": intakeRef})
	const requestRef = "request:v23-wizard-snapshot-public-replay"
	payload := wizardGapsPublicPayload(intakeRef, 1)
	first := dispatchV23IntakeCommand(t, firstRuntime, principal, projectRef,
		"orquesta.intakes.wizard.gaps.apply", requestRef, payload)
	firstView := decodeV23WizardGapsReplayView(t, first)
	if !firstView.EvaluationReplayExact || wizardGapsPublicLeaksSnapshot(first.Data) {
		t.Fatalf("first exact=%t data=%s", firstView.EvaluationReplayExact, first.Data)
	}
	shutdownRuntime(t, firstRuntime)

	restarted := buildV23IntakeRuntime(t, configPath)
	t.Cleanup(func() { shutdownRuntime(t, restarted) })
	replayed := dispatchV23IntakeCommand(t, restarted, principal, projectRef,
		"orquesta.intakes.wizard.gaps.apply", requestRef, payload)
	replayedView := decodeV23WizardGapsReplayView(t, replayed)
	if !replayedView.EvaluationReplayExact || replayed.Failure != nil ||
		replayed.AuditRef != first.AuditRef || !bytes.Equal(replayed.Data, first.Data) ||
		wizardGapsPublicLeaksSnapshot(replayed.Data) {
		t.Fatalf("first=%+v replayed=%+v", first, replayed)
	}
}

func wizardGapsPublicLeaksSnapshot(encoded []byte) bool {
	return bytes.Contains(encoded, []byte("evaluation_snapshot")) ||
		bytes.Contains(encoded, []byte("snapshot_bytes")) ||
		bytes.Contains(encoded, []byte("snapshot_digest"))
}
