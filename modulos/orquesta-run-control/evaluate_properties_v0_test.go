package orquestaruncontrol

import (
	"reflect"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

func TestEvaluateRunControlV0PropDeterministaEInmutableV0(t *testing.T) {
	// Invariant: a pure evaluation is repeatable and leaves its input untouched.
	rapid.Check(t, func(rt *rapid.T) {
		state := runControlStateRapidV0().Draw(rt, "state")
		before := cloneRunControlStateForTestV0(state)
		first := EvaluateRunControlV0(state)
		second := EvaluateRunControlV0(state)
		if first != second || !reflect.DeepEqual(state, before) {
			rt.Fatalf("state=%+v before=%+v first=%+v second=%+v", state, before, first, second)
		}
	})
}

func TestEvaluateRunControlV0PropNormalizaEstadoRelevanteV0(t *testing.T) {
	// Invariant: casing and surrounding whitespace do not change the decision.
	rapid.Check(t, func(rt *rapid.T) {
		state := runControlStateRapidV0().Draw(rt, "state")
		canonical := NormalizeRunControlStatusV0(state.Status)
		variant := state
		variant.Status = RunControlStatusV0(
			rapid.SampledFrom([]string{" ", "\t", "\n"}).Draw(rt, "left") +
				strings.ToUpper(string(canonical)) +
				rapid.SampledFrom([]string{" ", "\t", "\n"}).Draw(rt, "right"),
		)
		if got, want := EvaluateRunControlV0(variant), EvaluateRunControlV0(RunControlStateV0{
			Status:             canonical,
			CheckpointRecorded: state.CheckpointRecorded,
			Forced:             state.Forced,
		}); got != want {
			rt.Fatalf("state=%+v variant=%+v got=%+v want=%+v", state, variant, got, want)
		}
	})
}

func TestEvaluateRunControlV0PropForcedPermiteParadaSolicitadaV0(t *testing.T) {
	// Invariant: forced requests bypass the checkpoint, but do not reopen execution.
	rapid.Check(t, func(rt *rapid.T) {
		state := RunControlStateV0{
			Status:             rapid.SampledFrom([]RunControlStatusV0{RunControlStatusStopRequestedV0, RunControlStatusCancelRequestedV0}).Draw(rt, "status"),
			CheckpointRecorded: rapid.Bool().Draw(rt, "checkpoint"),
			Forced:             true,
		}
		got := EvaluateRunControlV0(state)
		if got != (RunControlEvaluationV0{StopAgentsAllowed: true}) {
			rt.Fatalf("forced requested state=%+v evaluation=%+v", state, got)
		}
	})
}

func TestEvaluateRunControlV0PropTerminalNoAutorizaAccionesV0(t *testing.T) {
	// Invariant: terminal states remain terminal regardless of stop-related flags.
	rapid.Check(t, func(rt *rapid.T) {
		state := RunControlStateV0{
			Status: rapid.SampledFrom([]RunControlStatusV0{
				RunControlStatusStoppedV0,
				RunControlStatusCanceledV0,
			}).Draw(rt, "status"),
			CheckpointRecorded: rapid.Bool().Draw(rt, "checkpoint"),
			Forced:             rapid.Bool().Draw(rt, "forced"),
		}
		got := EvaluateRunControlV0(state)
		if got != (RunControlEvaluationV0{Terminal: true}) || !IsTerminalRunControlStatusV0(state.Status) {
			rt.Fatalf("terminal state=%+v evaluation=%+v", state, got)
		}
	})
}

func TestEvaluateRunControlV0PropConservaDecisionSeguraV0(t *testing.T) {
	// Invariant: irrelevant metadata cannot grant permissions or alter a decision.
	rapid.Check(t, func(rt *rapid.T) {
		state := runControlStateRapidV0().Draw(rt, "state")
		baseline := EvaluateRunControlV0(state)
		metadataVariant := RunControlStateV0{
			RunRef:             rapid.SampledFrom([]string{"", "run-other", " run-other "}).Draw(rt, "run_ref"),
			Status:             state.Status,
			CheckpointRecorded: state.CheckpointRecorded,
			Forced:             state.Forced,
			EvidenceRefs:       []string{"other-evidence", "other-evidence"},
			Meta: RunControlMetaV0{
				RequestedBy:    "other-actor",
				Reason:         "other-reason",
				IdempotencyKey: "other-key",
			},
		}
		got := EvaluateRunControlV0(metadataVariant)
		if got != baseline || got.SchedulingAllowed != got.DispatchAllowed ||
			(got.CheckpointRequired && got.StopAgentsAllowed) ||
			(got.Terminal && (got.SchedulingAllowed || got.DispatchAllowed || got.CheckpointRequired || got.StopAgentsAllowed)) {
			rt.Fatalf("state=%+v variant=%+v baseline=%+v got=%+v", state, metadataVariant, baseline, got)
		}
	})
}

func runControlStateRapidV0() *rapid.Generator[RunControlStateV0] {
	return rapid.Custom(func(rt *rapid.T) RunControlStateV0 {
		return RunControlStateV0{
			RunRef: rapid.SampledFrom([]string{"", "run-a", " run-b "}).Draw(rt, "run_ref"),
			Status: rapid.SampledFrom([]RunControlStatusV0{
				"",
				RunControlStatusRunningV0,
				RunControlStatusPausedV0,
				RunControlStatusStopRequestedV0,
				RunControlStatusCancelRequestedV0,
				RunControlStatusStoppedV0,
				RunControlStatusCanceledV0,
				"unknown",
				" RUNNING ",
			}).Draw(rt, "status"),
			CheckpointRecorded: rapid.Bool().Draw(rt, "checkpoint"),
			Forced:             rapid.Bool().Draw(rt, "forced"),
			EvidenceRefs:       rapid.SliceOfN(rapid.SampledFrom([]string{"", "evidence-a", "evidence-b"}), 0, 3).Draw(rt, "evidence_refs"),
			Meta: RunControlMetaV0{
				RequestedBy:    rapid.SampledFrom([]string{"", "actor-a"}).Draw(rt, "requested_by"),
				Reason:         rapid.SampledFrom([]string{"", "reason-a"}).Draw(rt, "reason"),
				IdempotencyKey: rapid.SampledFrom([]string{"", "key-a"}).Draw(rt, "idempotency_key"),
			},
		}
	})
}

func cloneRunControlStateForTestV0(state RunControlStateV0) RunControlStateV0 {
	if state.EvidenceRefs != nil {
		evidenceRefs := state.EvidenceRefs
		state.EvidenceRefs = make([]string, len(state.EvidenceRefs))
		copy(state.EvidenceRefs, evidenceRefs)
	}
	return state
}
