package application

import (
	"errors"
	"reflect"
	"testing"

	"orquesta/internal/goal"
	"orquesta/internal/intake"
)

func TestBuildIntakeDossierBindsVerifiedRecordPlanAndCompleteDecisions(t *testing.T) {
	record := validIntakeChain(t)[2]
	plan := validIntakeDossierPlan()
	input := validIntakeDossierInput()

	dossier, err := BuildIntakeDossier(record, plan, input)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := BuildIntakeDossier(record, plan, input)
	if err != nil {
		t.Fatal(err)
	}
	decisions := dossier.Decisions()
	if dossier.Schema() != IntakeDossierSchema ||
		dossier.Ref() != IntakeDossierRef("intake-dossier:"+dossier.Digest()) ||
		dossier.Ref() != replay.Ref() ||
		dossier.ActorRef() != record.ActorRef ||
		dossier.ProjectRef() != record.ProjectRef ||
		dossier.StateRef() != record.State.Ref() ||
		dossier.StateRevision() != record.State.Revision() ||
		dossier.StateDigest() != record.Receipt.StateDigest ||
		dossier.SourceIntakeReceiptRef() != record.Receipt.Ref ||
		dossier.PlanDigest() != IntakeDossierPlanDigest(plan) ||
		len(decisions) != 1 ||
		!reflect.DeepEqual(decisions[0], record.State.Decisions()[0]) {
		t.Fatalf("dossier binding invalid: ref=%q decisions=%+v", dossier.Ref(), decisions)
	}

	changedPlan := cloneIntakeDossierPlan(plan)
	changedPlan.WorkItems[0].Objective += " changed"
	changed, err := BuildIntakeDossier(record, changedPlan, input)
	if err != nil {
		t.Fatal(err)
	}
	if dossier.Ref() == changed.Ref() || dossier.PlanDigest() == changed.PlanDigest() {
		t.Fatal("dossier did not bind the exact PlanSpec")
	}
}

func TestIntakeDossierHashProjectsEveryCurrentDecisionField(t *testing.T) {
	dossier, err := BuildIntakeDossier(
		validIntakeChain(t)[2], validIntakeDossierPlan(), validIntakeDossierInput(),
	)
	if err != nil {
		t.Fatal(err)
	}
	baseDigest := hashIntakeDossier(dossier)
	tests := []struct {
		name   string
		mutate func(*intake.Decision)
	}{
		{name: "choice", mutate: func(value *intake.Decision) {
			value.Choice = "intake-option:audience-team"
		}},
		{name: "recommendation", mutate: func(value *intake.Decision) {
			value.Recommendation = "intake-option:audience-personal"
		}},
		{name: "recommendation rationale", mutate: func(value *intake.Decision) {
			value.RecommendationRationale = "intake.option.audience.changed.rationale"
		}},
		{name: "origin", mutate: func(value *intake.Decision) {
			value.Origin = intake.OriginChat
		}},
		{name: "revision", mutate: func(value *intake.Decision) {
			value.Revision++
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			changed := dossier
			changed.decisions = append([]intake.Decision(nil), dossier.decisions...)
			test.mutate(&changed.decisions[0])
			if hashIntakeDossier(changed) == baseDigest {
				t.Fatalf("decision field %q is absent from dossier hash", test.name)
			}
		})
	}
}

func TestBuildIntakeDossierRejectsTamperedRecordUnresolvedStateAndInvalidPlan(t *testing.T) {
	records := validIntakeChain(t)
	plan := validIntakeDossierPlan()
	input := validIntakeDossierInput()

	tampered := records[2]
	tampered.Receipt.StateDigest = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if _, err := BuildIntakeDossier(tampered, plan, input); !errors.Is(err, ErrIntakeDossierInvalid) {
		t.Fatalf("tampered state digest error = %v", err)
	}
	if _, err := BuildIntakeDossier(records[1], plan, input); !errors.Is(err, ErrIntakeDossierInvalid) {
		t.Fatalf("unanswered intake error = %v", err)
	}
	if _, err := BuildIntakeDossier(records[2], PlanSpec{}, input); !errors.Is(err, ErrIntakeDossierInvalid) {
		t.Fatalf("empty plan error = %v", err)
	}
	unknownPhase := cloneIntakeDossierPlan(plan)
	unknownPhase.WorkItems[0].Phase = "phase.unknown"
	if _, err := BuildIntakeDossier(records[2], unknownPhase, input); !errors.Is(err, ErrIntakeDossierInvalid) {
		t.Fatalf("unknown phase error = %v", err)
	}
}

func TestIntakeDossierReturnsDefensiveCopies(t *testing.T) {
	dossier, err := BuildIntakeDossier(
		validIntakeChain(t)[2], validIntakeDossierPlan(), validIntakeDossierInput(),
	)
	if err != nil {
		t.Fatal(err)
	}
	sections, diagrams := dossier.Sections(), dossier.Diagrams()
	decisions, risks, plan := dossier.Decisions(), dossier.RiskRefs(), dossier.Plan()
	sections[0].Markdown = "mutated"
	diagrams[0].Source = "mutated"
	decisions[0].Choice = "intake-option:mutated"
	risks[0] = "intake-risk:mutated"
	plan.Phases[0].InputRefs[0] = "input:mutated"
	plan.WorkItems[0].RequiredTests[0].Arguments[0] = "mutated"
	if dossier.Sections()[0].Markdown == "mutated" ||
		dossier.Diagrams()[0].Source == "mutated" ||
		dossier.Decisions()[0].Choice == "intake-option:mutated" ||
		dossier.RiskRefs()[0] == "intake-risk:mutated" ||
		dossier.Plan().Phases[0].InputRefs[0] == "input:mutated" ||
		dossier.Plan().WorkItems[0].RequiredTests[0].Arguments[0] == "mutated" {
		t.Fatal("caller mutated immutable dossier")
	}
}

func validIntakeDossierPlan() PlanSpec {
	return PlanSpec{
		Phases: []PhaseSpec{{
			Ref: "phase-instance:intake-dossier", Key: "phase.intake-dossier",
			TemplateRef: "phase-template:intake-dossier",
			InputRefs: []string{
				"input:intake-dossier",
			},
			CriterionRefs: []string{"criterion:intake-dossier-reviewed"},
		}},
		WorkItems: []WorkItemSpec{{
			Key: "work:dossier", Objective: "Crear la aplicación confirmada",
			Phase: "phase.intake-dossier", Role: goal.DefaultRoleKey().String(),
			OutputContract: goal.OutputContractEvidenceBundle,
			RequiredTests: []RequiredTestSpec{{
				Ref: "required-test:intake-dossier", ToolRef: "tool:go-test",
				Arguments: []string{"go", "test", "./..."}, WorkingDirectory: ".",
			}},
		}},
	}
}

func validIntakeDossierInput() IntakeDossierInput {
	kinds := []IntakeDossierSectionKind{
		IntakeDossierSectionProductScope, IntakeDossierSectionUsersRoles,
		IntakeDossierSectionArchitecture, IntakeDossierSectionData,
		IntakeDossierSectionIntegrations, IntakeDossierSectionSecurityPrivacy,
		IntakeDossierSectionUIUX, IntakeDossierSectionI18NL10N,
		IntakeDossierSectionDeployOperations, IntakeDossierSectionOrchestrationPlan,
		IntakeDossierSectionRisksOpenIssues,
	}
	sections := make([]IntakeDossierSection, len(kinds))
	for index, kind := range kinds {
		sections[index] = IntakeDossierSection{
			Ref:  IntakeDossierSectionRef("intake-dossier-section:" + string(kind)),
			Kind: kind, TitleKey: intake.MessageKey("intake.dossier." + string(kind) + ".title"),
			Markdown: "Contenido " + string(kind),
		}
	}
	purposes := []IntakeDossierDiagramPurpose{
		IntakeDossierDiagramArchitecture, IntakeDossierDiagramUserFlow,
		IntakeDossierDiagramDataIntegrations, IntakeDossierDiagramI18N,
		IntakeDossierDiagramDeployment,
	}
	diagrams := make([]IntakeDossierDiagram, len(purposes))
	for index, purpose := range purposes {
		diagrams[index] = IntakeDossierDiagram{
			Ref:     IntakeDossierDiagramRef("intake-dossier-diagram:" + string(purpose)),
			Purpose: purpose, Kind: IntakeDossierDiagramMermaid,
			Source:     "flowchart TD\n  source --> " + string(purpose),
			AltTextKey: intake.MessageKey("intake.dossier.diagram." + string(purpose) + ".alt"),
		}
	}
	return IntakeDossierInput{
		Statement: "Quiero una agenda compartida",
		Objective: "Crear una aplicación de agenda",
		Sections:  sections, Diagrams: diagrams,
		RiskRefs: []IntakeRiskRef{"intake-risk:calendar-provider"},
	}
}
