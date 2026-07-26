package application

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestIntakeDossierSnapshotRoundTripLosesNoDataAndRecomputesIdentity(t *testing.T) {
	dossier, err := BuildIntakeDossier(
		validIntakeChain(t)[2], validIntakeDossierPlan(), validIntakeDossierInput(),
	)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := SnapshotIntakeDossier(dossier)
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	var persisted IntakeDossierSnapshot
	if err := json.Unmarshal(encoded, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.ActorRef != dossier.ActorRef().String() ||
		persisted.ProjectRef != dossier.ProjectRef().String() ||
		len(persisted.Plan.Phases[0].InputRefs) == 0 ||
		len(persisted.Plan.WorkItems[0].RequiredTests[0].Arguments) == 0 {
		t.Fatalf("JSON snapshot lost refs or rich plan: %+v", persisted)
	}
	restored, err := RestoreIntakeDossier(persisted)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(SnapshotIntakeDossier(restored), snapshot) ||
		restored.Ref() != IntakeDossierRef("intake-dossier:"+restored.Digest()) ||
		restored.PlanDigest() != IntakeDossierPlanDigest(restored.Plan()) {
		t.Fatalf("lossy dossier restore: before=%+v after=%+v",
			snapshot, SnapshotIntakeDossier(restored))
	}
}

func TestRestoreIntakeDossierRejectsTamperedSnapshot(t *testing.T) {
	dossier, err := BuildIntakeDossier(
		validIntakeChain(t)[2], validIntakeDossierPlan(), validIntakeDossierInput(),
	)
	if err != nil {
		t.Fatal(err)
	}
	base := SnapshotIntakeDossier(dossier)
	tests := []struct {
		name   string
		mutate func(*IntakeDossierSnapshot)
	}{
		{name: "ref", mutate: func(value *IntakeDossierSnapshot) {
			value.Ref = "intake-dossier:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		}},
		{name: "state digest", mutate: func(value *IntakeDossierSnapshot) {
			value.StateDigest = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		}},
		{name: "section content", mutate: func(value *IntakeDossierSnapshot) {
			value.Sections[0].Markdown += " tampered"
		}},
		{name: "decision", mutate: func(value *IntakeDossierSnapshot) {
			value.Decisions[0].Choice = "intake-option:tampered"
		}},
		{name: "plan", mutate: func(value *IntakeDossierSnapshot) {
			value.Plan.WorkItems[0].Objective += " tampered"
		}},
		{name: "plan digest", mutate: func(value *IntakeDossierSnapshot) {
			value.PlanDigest = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		}},
		{name: "digest", mutate: func(value *IntakeDossierSnapshot) {
			value.Digest = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tampered := canonicalIntakeDossierSnapshot(base)
			test.mutate(&tampered)
			if _, err := RestoreIntakeDossier(tampered); !IsStateError(err, StateInvalid) {
				t.Fatalf("tampered snapshot error = %v", err)
			}
		})
	}
}

func TestIntakeDossierSnapshotAndRestoreUseDefensiveCopies(t *testing.T) {
	dossier, err := BuildIntakeDossier(
		validIntakeChain(t)[2], validIntakeDossierPlan(), validIntakeDossierInput(),
	)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := SnapshotIntakeDossier(dossier)
	restored, err := RestoreIntakeDossier(snapshot)
	if err != nil {
		t.Fatal(err)
	}

	snapshot.Sections[0].Markdown = "mutated"
	snapshot.Diagrams[0].Source = "mutated"
	snapshot.Decisions[0].Choice = "intake-option:mutated"
	snapshot.RiskRefs[0] = "intake-risk:mutated"
	snapshot.Plan.Phases[0].InputRefs[0] = "input:mutated"
	snapshot.Plan.WorkItems[0].RequiredTests[0].Arguments[0] = "mutated"
	if restored.Sections()[0].Markdown == "mutated" ||
		restored.Diagrams()[0].Source == "mutated" ||
		restored.Decisions()[0].Choice == "intake-option:mutated" ||
		restored.RiskRefs()[0] == "intake-risk:mutated" ||
		restored.Plan().Phases[0].InputRefs[0] == "input:mutated" ||
		restored.Plan().WorkItems[0].RequiredTests[0].Arguments[0] == "mutated" {
		t.Fatal("restore retained snapshot backing storage")
	}

	fresh := SnapshotIntakeDossier(dossier)
	fresh.Sections[0].Markdown = "snapshot mutation"
	fresh.Plan.WorkItems[0].RequiredTests[0].Arguments[0] = "snapshot mutation"
	if dossier.Sections()[0].Markdown == "snapshot mutation" ||
		dossier.Plan().WorkItems[0].RequiredTests[0].Arguments[0] == "snapshot mutation" {
		t.Fatal("snapshot retained dossier backing storage")
	}
}
