package orquestaautoprogramming

import (
	"testing"
	"testing/quick"
)

func TestDecideMaterialProgressV0EscalaDeterministaSinProgreso(t *testing.T) {
	for _, test := range []struct {
		name   string
		tokens int64
		want   MaterialProgressActionV0
	}{
		{name: "antes de warning", tokens: 109, want: MaterialProgressActionContinueV0},
		{name: "warning", tokens: 110, want: MaterialProgressActionWarningV0},
		{name: "replan", tokens: 120, want: MaterialProgressActionReplanRequiredV0},
		{name: "hard stop", tokens: 130, want: MaterialProgressActionHardStopRequiredV0},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := validMaterialProgressInputV0()
			input.Checkpoint.TokensAccumulated = test.tokens

			got := DecideMaterialProgressV0(input)

			if !got.Accepted || got.Action != test.want || got.MaterialProgressed {
				t.Fatalf("decision=%+v", got)
			}
			if got.TokensWithoutMaterial != test.tokens-input.Segment.StartTokensAccumulated {
				t.Fatalf("tokens_without_material=%d", got.TokensWithoutMaterial)
			}
		})
	}
}

func TestDecideMaterialProgressV0RenuevaSoloConClasesMaterialesYEvidenciaNueva(t *testing.T) {
	for _, class := range []MaterialProgressClassV0{
		MaterialProgressClassDiffV0,
		MaterialProgressClassTestV0,
		MaterialProgressClassResultV0,
		MaterialProgressClassReceiptV0,
	} {
		t.Run(string(class), func(t *testing.T) {
			input := validMaterialProgressInputV0()
			input.Checkpoint.Sequence = 9
			input.Checkpoint.TokensAccumulated = 999
			input.Checkpoint.MaterialClass = class
			input.Checkpoint.EvidenceRefs = []string{"evidence-ref-" + string(class)}

			got := DecideMaterialProgressV0(input)

			if !got.Accepted || !got.MaterialProgressed || got.Action != MaterialProgressActionContinueV0 {
				t.Fatalf("decision=%+v", got)
			}
			if got.Segment.StartSequence != 9 || got.Segment.StartTokensAccumulated != 999 {
				t.Fatalf("segment=%+v", got.Segment)
			}
			if got.TokensWithoutMaterial != 0 {
				t.Fatalf("tokens_without_material=%d", got.TokensWithoutMaterial)
			}
		})
	}
}

func TestDecideMaterialProgressV0NoneNuncaRenuevaPorResumenHeartbeatONombre(t *testing.T) {
	for _, evidenceRef := range []string{
		"summary-ref-001",
		"heartbeat-ref-001",
		"checkpoint-name-ref-001",
	} {
		t.Run(evidenceRef, func(t *testing.T) {
			input := validMaterialProgressInputV0()
			input.Checkpoint.TokensAccumulated = 130
			input.Checkpoint.EvidenceRefs = []string{evidenceRef}

			got := DecideMaterialProgressV0(input)

			if !got.Accepted || got.MaterialProgressed || got.Action != MaterialProgressActionHardStopRequiredV0 {
				t.Fatalf("decision=%+v", got)
			}
			if got.Segment.StartSequence != input.Segment.StartSequence ||
				got.Segment.StartTokensAccumulated != input.Segment.StartTokensAccumulated ||
				got.Segment.ContextRevisionRef != input.Segment.ContextRevisionRef ||
				len(got.Segment.EvidenceRefs) != len(input.Segment.EvidenceRefs) {
				t.Fatalf("segment renewed: %+v", got.Segment)
			}
		})
	}
}

func TestDecideMaterialProgressV0NoRenuevaEvidenciaMaterialRepetida(t *testing.T) {
	input := validMaterialProgressInputV0()
	input.Checkpoint.TokensAccumulated = 120
	input.Checkpoint.MaterialClass = MaterialProgressClassDiffV0
	input.Checkpoint.EvidenceRefs = []string{"evidence-ref-existing"}
	input.Segment.EvidenceRefs = []string{"evidence-ref-existing"}

	got := DecideMaterialProgressV0(input)

	if !got.Accepted || got.MaterialProgressed || got.Action != MaterialProgressActionReplanRequiredV0 {
		t.Fatalf("decision=%+v", got)
	}
}

func TestDecideMaterialProgressV0ConservaEvidenciaHistoricaYNoRenuevaReplayABA(t *testing.T) {
	input := validMaterialProgressInputV0()
	input.Segment.EvidenceRefs = []string{"evidence-ref-history"}
	input.Checkpoint.Sequence = 2
	input.Checkpoint.TokensAccumulated = 110
	input.Checkpoint.MaterialClass = MaterialProgressClassDiffV0
	input.Checkpoint.EvidenceRefs = []string{"evidence-ref-a"}

	first := DecideMaterialProgressV0(input)
	if !first.Accepted || !first.MaterialProgressed {
		t.Fatalf("first=%+v", first)
	}
	assertMaterialProgressEvidenceRefsV0(t, first.Segment.EvidenceRefs,
		"evidence-ref-history", "evidence-ref-a")

	input.Segment = first.Segment
	input.Checkpoint.Sequence = 3
	input.Checkpoint.TokensAccumulated = 120
	input.Checkpoint.MaterialClass = MaterialProgressClassTestV0
	input.Checkpoint.EvidenceRefs = []string{"evidence-ref-b"}

	second := DecideMaterialProgressV0(input)
	if !second.Accepted || !second.MaterialProgressed {
		t.Fatalf("second=%+v", second)
	}
	assertMaterialProgressEvidenceRefsV0(t, second.Segment.EvidenceRefs,
		"evidence-ref-history", "evidence-ref-a", "evidence-ref-b")

	input.Segment = second.Segment
	input.Checkpoint.Sequence = 4
	input.Checkpoint.TokensAccumulated = 150
	input.Checkpoint.MaterialClass = MaterialProgressClassDiffV0
	input.Checkpoint.EvidenceRefs = []string{"evidence-ref-a"}

	third := DecideMaterialProgressV0(input)
	if !third.Accepted || third.MaterialProgressed || third.Action != MaterialProgressActionHardStopRequiredV0 {
		t.Fatalf("third=%+v", third)
	}
	assertMaterialProgressEvidenceRefsV0(t, third.Segment.EvidenceRefs,
		"evidence-ref-history", "evidence-ref-a", "evidence-ref-b")
}

func TestValidateMaterialProgressV0NormalizaRefsYRechazaContratoInvalido(t *testing.T) {
	input := validMaterialProgressInputV0()
	input.Segment.ContextRevisionRef = " context-ref-segment "
	input.Checkpoint.ContextRevisionRef = " context-ref-checkpoint "
	input.Checkpoint.MaterialClass = MaterialProgressClassDiffV0
	input.Checkpoint.EvidenceRefs = []string{" evidence-ref-a ", "evidence-ref-a", ""}
	input.Policy.ReplanRequiredAfterTokens = input.Policy.WarningAfterTokens

	got := ValidateMaterialProgressV0(input)

	if got.Accepted {
		t.Fatalf("accepted=true")
	}
	if got.Input.Segment.ContextRevisionRef != "context-ref-segment" ||
		got.Input.Checkpoint.ContextRevisionRef != "context-ref-checkpoint" ||
		len(got.Input.Checkpoint.EvidenceRefs) != 1 ||
		got.Input.Checkpoint.EvidenceRefs[0] != "evidence-ref-a" {
		t.Fatalf("normalized=%+v", got.Input)
	}
	assertMaterialProgressIssueV0(t, got.Issues, "policy_threshold_order_invalid")
}

func TestValidateMaterialProgressV0RechazaClaseOEvidenciaInvalidaYRegresion(t *testing.T) {
	input := validMaterialProgressInputV0()
	input.Checkpoint.Sequence = 0
	input.Checkpoint.TokensAccumulated = 9
	input.Checkpoint.ContextRevisionRef = ""
	input.Checkpoint.MaterialClass = MaterialProgressClassV0("summary")
	input.Segment.StartTokensAccumulated = 10

	got := ValidateMaterialProgressV0(input)

	if got.Accepted {
		t.Fatalf("accepted=true")
	}
	assertMaterialProgressIssueV0(t, got.Issues, "checkpoint_sequence_invalid")
	assertMaterialProgressIssueV0(t, got.Issues, "checkpoint_context_revision_ref_missing")
	assertMaterialProgressIssueV0(t, got.Issues, "checkpoint_material_class_invalid")
	assertMaterialProgressIssueV0(t, got.Issues, "checkpoint_before_segment")
	assertMaterialProgressIssueV0(t, got.Issues, "checkpoint_tokens_regressed")
}

func TestValidateMaterialProgressV0RechazaCambioDeContextoDentroDelTramo(t *testing.T) {
	input := validMaterialProgressInputV0()
	input.Checkpoint.ContextRevisionRef = "context-ref-new"

	got := ValidateMaterialProgressV0(input)

	if got.Accepted {
		t.Fatalf("accepted=true")
	}
	assertMaterialProgressIssueV0(t, got.Issues, "checkpoint_context_revision_ref_changed")
}

func TestDecideMaterialProgressV0PropertySeleccionaMayorUmbralAplicable(t *testing.T) {
	check := func(delta uint8) bool {
		input := validMaterialProgressInputV0()
		input.Checkpoint.TokensAccumulated = input.Segment.StartTokensAccumulated + int64(delta)
		got := DecideMaterialProgressV0(input)
		want := MaterialProgressActionContinueV0
		switch {
		case delta >= 30:
			want = MaterialProgressActionHardStopRequiredV0
		case delta >= 20:
			want = MaterialProgressActionReplanRequiredV0
		case delta >= 10:
			want = MaterialProgressActionWarningV0
		}
		return got.Accepted && !got.MaterialProgressed && got.Action == want &&
			got.TokensWithoutMaterial == int64(delta)
	}
	if err := quick.Check(check, nil); err != nil {
		t.Fatal(err)
	}
}

func validMaterialProgressInputV0() MaterialProgressInputV0 {
	return MaterialProgressInputV0{
		Policy: MaterialProgressPolicyV0{
			WarningAfterTokens:          10,
			ReplanRequiredAfterTokens:   20,
			HardStopRequiredAfterTokens: 30,
		},
		Segment: MaterialProgressSegmentV0{
			StartSequence:          1,
			StartTokensAccumulated: 100,
			ContextRevisionRef:     "context-ref-segment",
		},
		Checkpoint: MaterialProgressCheckpointV0{
			Sequence:           2,
			TokensAccumulated:  100,
			ContextRevisionRef: "context-ref-segment",
			MaterialClass:      MaterialProgressClassNoneV0,
		},
	}
}

func assertMaterialProgressEvidenceRefsV0(t *testing.T, got []string, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("evidence_refs=%q want=%q", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("evidence_refs=%q want=%q", got, want)
		}
	}
}

func assertMaterialProgressIssueV0(t *testing.T, issues []MaterialProgressIssueV0, want string) {
	t.Helper()
	for _, issue := range issues {
		if issue.Code == want {
			return
		}
	}
	t.Fatalf("issue %q absent: %+v", want, issues)
}
