package orquestaautoprogramming

import (
	"strings"
	"testing"
	"time"
)

func TestValidateMaterialProgressStateV0AceptaDocumentoPersistible(t *testing.T) {
	state := validMaterialProgressStateV0()

	got := ValidateMaterialProgressStateV0(state)

	if !got.Accepted {
		t.Fatalf("issues=%+v", got.Issues)
	}
	if got.State.ObservedAt.Location() != time.UTC || got.State.EvidenceRefs[0] != "evidence-ref-a" {
		t.Fatalf("state=%+v", got.State)
	}
}

func TestNormalizeMaterialProgressStateV0CompactaRefsHashYEvidencia(t *testing.T) {
	state := validMaterialProgressStateV0()
	state.RunRef = " run-ref-001 "
	state.GoalRef = " goal-ref-001 "
	state.BaselineRef = " baseline-ref-001 "
	state.ContextRevisionRef = " context-ref-001 "
	state.LastCheckpointRef = " checkpoint-ref-001 "
	state.WriteSetSHA256 = strings.ToUpper(state.WriteSetSHA256)
	state.Segment.EvidenceRefs = []string{" evidence-ref-history ", "evidence-ref-history"}
	state.LastCheckpoint.EvidenceRefs = []string{" evidence-ref-a ", "evidence-ref-history"}
	state.EvidenceRefs = []string{" evidence-ref-a ", ""}

	got := NormalizeMaterialProgressStateV0(state)

	if got.RunRef != "run-ref-001" || got.WriteSetSHA256 != strings.ToLower(state.WriteSetSHA256) {
		t.Fatalf("state=%+v", got)
	}
	assertMaterialProgressStateRefsV0(t, got.EvidenceRefs, "evidence-ref-a", "evidence-ref-history")
}

func TestValidateMaterialProgressStateV0RechazaZeroHashRefYDecisionInconsistente(t *testing.T) {
	state := MaterialProgressStateV0{}
	got := ValidateMaterialProgressStateV0(state)
	if got.Accepted {
		t.Fatal("accepted=true")
	}
	for _, code := range []string{
		"state_schema_version_invalid", "state_store_version_invalid", "state_run_ref_invalid",
		"state_write_set_sha256_invalid", "state_observed_at_missing", "state_last_decision_invalid",
	} {
		assertMaterialProgressStateIssueV0(t, got.Issues, code)
	}

	state = validMaterialProgressStateV0()
	state.WriteSetSHA256 = "not-a-sha256"
	state.LastDecision.Action = MaterialProgressActionWarningV0
	got = ValidateMaterialProgressStateV0(state)
	if got.Accepted {
		t.Fatal("accepted=true")
	}
	assertMaterialProgressStateIssueV0(t, got.Issues, "state_write_set_sha256_invalid")
	assertMaterialProgressStateIssueV0(t, got.Issues, "state_last_decision_checkpoint_mismatch")
}

func TestValidateMaterialProgressStateV0RechazaContextoDivergente(t *testing.T) {
	state := validMaterialProgressStateV0()
	state.ContextRevisionRef = "context-ref-other"

	got := ValidateMaterialProgressStateV0(state)

	if got.Accepted {
		t.Fatal("accepted=true")
	}
	assertMaterialProgressStateIssueV0(t, got.Issues, "state_context_revision_ref_mismatch")
}

func TestMaterialProgressActionIdempotencyKeyV0EsDeterministaYSensibleALaAccion(t *testing.T) {
	first := MaterialProgressActionIdempotencyKeyV0(
		"run-ref-001", "goal-ref-001", "checkpoint-ref-001", MaterialProgressActionWarningV0)
	second := MaterialProgressActionIdempotencyKeyV0(
		" run-ref-001 ", "goal-ref-001", "checkpoint-ref-001", MaterialProgressActionWarningV0)
	other := MaterialProgressActionIdempotencyKeyV0(
		"run-ref-001", "goal-ref-001", "checkpoint-ref-001", MaterialProgressActionReplanRequiredV0)

	if first == "" || first != second || first == other {
		t.Fatalf("first=%q second=%q other=%q", first, second, other)
	}
	if got := MaterialProgressActionIdempotencyKeyV0("", "goal", "checkpoint", MaterialProgressActionWarningV0); got != "" {
		t.Fatalf("invalid key=%q", got)
	}
}

func TestMaterialProgressCheckpointRefV0EsDeterministaYSensibleAlContenido(t *testing.T) {
	state := validMaterialProgressStateV0()
	first := MaterialProgressCheckpointRefV0(
		state.RunRef, state.GoalRef, state.BaselineRef, state.WriteSetSHA256, state.LastCheckpoint,
	)
	reordered := state.LastCheckpoint
	reordered.EvidenceRefs = []string{"evidence-ref-b", "evidence-ref-a"}
	withBoth := state.LastCheckpoint
	withBoth.EvidenceRefs = []string{"evidence-ref-a", "evidence-ref-b"}
	if first == "" || MaterialProgressCheckpointRefV0(
		state.RunRef, state.GoalRef, state.BaselineRef, state.WriteSetSHA256, reordered,
	) != MaterialProgressCheckpointRefV0(
		state.RunRef, state.GoalRef, state.BaselineRef, state.WriteSetSHA256, withBoth,
	) {
		t.Fatal("checkpoint ref no determinista por orden de evidencia")
	}
	changed := state.LastCheckpoint
	changed.TokensAccumulated++
	if first == MaterialProgressCheckpointRefV0(
		state.RunRef, state.GoalRef, state.BaselineRef, state.WriteSetSHA256, changed,
	) {
		t.Fatal("checkpoint ref no cambia con tokens")
	}
}

func TestValidateMaterialProgressStateV0RechazaRefsNoCompactasYCheckpointLibre(t *testing.T) {
	state := validMaterialProgressStateV0()
	state.BaselineRef = "/tmp/baseline"
	state.EvidenceRefs = append(state.EvidenceRefs, "evidence ref con espacios")
	state.LastCheckpointRef = "checkpoint-ref-libre"
	got := ValidateMaterialProgressStateV0(state)
	if got.Accepted {
		t.Fatal("accepted=true")
	}
	assertMaterialProgressStateIssueV0(t, got.Issues, "state_baseline_ref_invalid")
	assertMaterialProgressStateIssueV0(t, got.Issues, "state_evidence_ref_invalid")
	assertMaterialProgressStateIssueV0(t, got.Issues, "state_checkpoint_ref_invalid")
}

func TestMaterialProgressStateCASConflictErrorV0(t *testing.T) {
	err := MaterialProgressStateCASConflictErrorV0{
		RunRef: "run-ref-001", GoalRef: "goal-ref-001", ExpectedVersion: 2, ActualVersion: 3,
	}
	if err.Error() != "material_progress_state_cas_conflict" {
		t.Fatalf("error=%q", err.Error())
	}
}

func validMaterialProgressStateV0() MaterialProgressStateV0 {
	segment := MaterialProgressSegmentV0{
		StartSequence: 2, StartTokensAccumulated: 110, ContextRevisionRef: "context-ref-001",
		EvidenceRefs: []string{"evidence-ref-a"},
	}
	checkpoint := MaterialProgressCheckpointV0{
		Sequence: 2, TokensAccumulated: 110, ContextRevisionRef: "context-ref-001",
		MaterialClass: MaterialProgressClassDiffV0, EvidenceRefs: []string{"evidence-ref-a"},
	}
	decision := MaterialProgressDecisionV0{
		Accepted: true, Action: MaterialProgressActionContinueV0, MaterialProgressed: true, Segment: segment,
	}
	state := MaterialProgressStateV0{
		SchemaVersion: MaterialProgressStateSchemaVersionV0,
		StoreVersion:  1,
		RunRef:        "run-ref-001", GoalRef: "goal-ref-001", Policy: MaterialProgressPolicyV0{
			WarningAfterTokens: 10, ReplanRequiredAfterTokens: 20, HardStopRequiredAfterTokens: 30,
		},
		Segment: segment, LastCheckpoint: checkpoint, LastDecision: decision,
		BaselineRef: "baseline-ref-001", WriteSetSHA256: strings.Repeat("a", 64),
		ContextRevisionRef: "context-ref-001", ObservedAt: time.Date(2026, 7, 11, 12, 0, 0, 0, time.FixedZone("CEST", 7200)),
		EvidenceRefs: []string{"evidence-ref-a"},
	}
	state.LastCheckpointRef = MaterialProgressCheckpointRefV0(
		state.RunRef, state.GoalRef, state.BaselineRef, state.WriteSetSHA256, state.LastCheckpoint)
	state.LastActionIdempotencyKey = MaterialProgressActionIdempotencyKeyV0(
		state.RunRef, state.GoalRef, state.LastCheckpointRef, state.LastDecision.Action)
	return state
}

func assertMaterialProgressStateRefsV0(t *testing.T, got []string, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("refs=%q want=%q", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("refs=%q want=%q", got, want)
		}
	}
}

func assertMaterialProgressStateIssueV0(t *testing.T, issues []MaterialProgressIssueV0, want string) {
	t.Helper()
	for _, issue := range issues {
		if issue.Code == want {
			return
		}
	}
	t.Fatalf("issue %q absent: %+v", want, issues)
}
