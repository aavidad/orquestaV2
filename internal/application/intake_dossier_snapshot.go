package application

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"

	"orquesta/internal/goal"
	"orquesta/internal/intake"
)

var errIntakeDossierSnapshotInvalid = errors.New("application.intake_dossier_snapshot_invalid")

// IntakeDossierSnapshot is the complete adapter-safe representation of an
// immutable dossier. Restore validates every field and recomputes content
// digests and refs; adapters never receive a blind field constructor.
type IntakeDossierSnapshot struct {
	Schema                 string                 `json:"schema"`
	Ref                    IntakeDossierRef       `json:"ref"`
	ActorRef               string                 `json:"actor_ref"`
	ProjectRef             string                 `json:"project_ref"`
	StateRef               intake.Ref             `json:"state_ref"`
	StateRevision          intake.Revision        `json:"state_revision"`
	StateDigest            string                 `json:"state_digest"`
	SourceIntakeReceiptRef string                 `json:"source_intake_receipt_ref"`
	Statement              string                 `json:"statement"`
	Objective              string                 `json:"objective"`
	Sections               []IntakeDossierSection `json:"sections"`
	Diagrams               []IntakeDossierDiagram `json:"diagrams"`
	Decisions              []intake.Decision      `json:"decisions,omitempty"`
	RiskRefs               []IntakeRiskRef        `json:"risk_refs,omitempty"`
	Plan                   PlanSpec               `json:"plan"`
	PlanDigest             string                 `json:"plan_digest"`
	Digest                 string                 `json:"digest"`
}

func SnapshotIntakeDossier(dossier IntakeDossier) IntakeDossierSnapshot {
	sections := dossier.Sections()
	diagrams := dossier.Diagrams()
	decisions := dossier.Decisions()
	riskRefs := dossier.RiskRefs()
	if len(sections) == 0 {
		sections = nil
	}
	if len(diagrams) == 0 {
		diagrams = nil
	}
	if len(decisions) == 0 {
		decisions = nil
	}
	if len(riskRefs) == 0 {
		riskRefs = nil
	}
	return IntakeDossierSnapshot{
		Schema: IntakeDossierSchema, Ref: dossier.Ref(),
		ActorRef: dossier.ActorRef().String(), ProjectRef: dossier.ProjectRef().String(),
		StateRef: dossier.StateRef(), StateRevision: dossier.StateRevision(),
		StateDigest:            dossier.StateDigest(),
		SourceIntakeReceiptRef: dossier.SourceIntakeReceiptRef(),
		Statement:              dossier.Statement(), Objective: dossier.Objective(),
		Sections: sections, Diagrams: diagrams, Decisions: decisions,
		RiskRefs: riskRefs, Plan: dossier.Plan(),
		PlanDigest: dossier.PlanDigest(), Digest: dossier.Digest(),
	}
}

// RestoreIntakeDossier is the only public persisted-field reconstruction path.
// It accepts no trusted digest: plan digest, dossier digest and ref are all
// derived again and must match the snapshot.
func RestoreIntakeDossier(snapshot IntakeDossierSnapshot) (IntakeDossier, error) {
	snapshot = canonicalIntakeDossierSnapshot(snapshot)
	if snapshot.Schema != IntakeDossierSchema ||
		!validIntakeDossierRef(string(snapshot.Ref), "intake-dossier:") ||
		snapshot.ActorRef == "" || snapshot.ProjectRef == "" ||
		!validIntakeStateRef(snapshot.StateRef) || snapshot.StateRevision == 0 ||
		!validIntakeDossierDigest(snapshot.StateDigest) ||
		!validIntakeDossierRef(snapshot.SourceIntakeReceiptRef, "intake-receipt:") ||
		!validIntakeDossierDigest(snapshot.PlanDigest) ||
		!validIntakeDossierDigest(snapshot.Digest) {
		return IntakeDossier{}, invalidIntakeDossierSnapshot(nil)
	}
	actorRef, err := goal.NewActorRef(snapshot.ActorRef)
	if err != nil {
		return IntakeDossier{}, invalidIntakeDossierSnapshot(err)
	}
	projectRef, err := goal.NewProjectRef(snapshot.ProjectRef)
	if err != nil {
		return IntakeDossier{}, invalidIntakeDossierSnapshot(err)
	}
	input := IntakeDossierInput{
		Statement: snapshot.Statement, Objective: snapshot.Objective,
		Sections: cloneIntakeDossierSections(snapshot.Sections),
		Diagrams: cloneIntakeDossierDiagrams(snapshot.Diagrams),
		RiskRefs: append([]IntakeRiskRef(nil), snapshot.RiskRefs...),
	}
	if err := validateIntakeDossierInput(input); err != nil {
		return IntakeDossier{}, invalidIntakeDossierSnapshot(err)
	}
	if err := validateIntakeDossierPlan(snapshot.Plan); err != nil {
		return IntakeDossier{}, invalidIntakeDossierSnapshot(err)
	}
	if err := validateIntakeDossierSnapshotDecisions(
		snapshot.Decisions, snapshot.StateRevision,
	); err != nil {
		return IntakeDossier{}, invalidIntakeDossierSnapshot(err)
	}
	if calculated := IntakeDossierPlanDigest(snapshot.Plan); calculated != snapshot.PlanDigest {
		return IntakeDossier{}, invalidIntakeDossierSnapshot(nil)
	}

	dossier := IntakeDossier{
		actorRef: actorRef, projectRef: projectRef,
		stateRef: snapshot.StateRef, stateRevision: snapshot.StateRevision,
		stateDigest:            snapshot.StateDigest,
		sourceIntakeReceiptRef: snapshot.SourceIntakeReceiptRef,
		statement:              snapshot.Statement, objective: snapshot.Objective,
		sections:   cloneIntakeDossierSections(snapshot.Sections),
		diagrams:   cloneIntakeDossierDiagrams(snapshot.Diagrams),
		decisions:  append([]intake.Decision(nil), snapshot.Decisions...),
		riskRefs:   append([]IntakeRiskRef(nil), snapshot.RiskRefs...),
		plan:       cloneIntakeDossierPlan(snapshot.Plan),
		planDigest: snapshot.PlanDigest,
	}
	dossier.digest = hashIntakeDossier(dossier)
	dossier.ref = IntakeDossierRef("intake-dossier:" + dossier.digest)
	if dossier.digest != snapshot.Digest || dossier.ref != snapshot.Ref ||
		!reflect.DeepEqual(SnapshotIntakeDossier(dossier), snapshot) {
		return IntakeDossier{}, invalidIntakeDossierSnapshot(nil)
	}
	return dossier, nil
}

func canonicalIntakeDossierSnapshot(snapshot IntakeDossierSnapshot) IntakeDossierSnapshot {
	snapshot.Sections = cloneIntakeDossierSections(snapshot.Sections)
	snapshot.Diagrams = cloneIntakeDossierDiagrams(snapshot.Diagrams)
	snapshot.Decisions = append([]intake.Decision(nil), snapshot.Decisions...)
	snapshot.RiskRefs = append([]IntakeRiskRef(nil), snapshot.RiskRefs...)
	snapshot.Plan = cloneIntakeDossierPlan(snapshot.Plan)
	if len(snapshot.Sections) == 0 {
		snapshot.Sections = nil
	}
	if len(snapshot.Diagrams) == 0 {
		snapshot.Diagrams = nil
	}
	if len(snapshot.Decisions) == 0 {
		snapshot.Decisions = nil
	}
	if len(snapshot.RiskRefs) == 0 {
		snapshot.RiskRefs = nil
	}
	if len(snapshot.Plan.Phases) == 0 {
		snapshot.Plan.Phases = nil
	}
	if len(snapshot.Plan.WorkItems) == 0 {
		snapshot.Plan.WorkItems = nil
	}
	return snapshot
}

func validateIntakeDossierSnapshotDecisions(
	decisions []intake.Decision,
	stateRevision intake.Revision,
) error {
	seen := make(map[intake.QuestionRef]struct{}, len(decisions))
	for _, decision := range decisions {
		if !validIntakeDossierRef(string(decision.QuestionRef), "intake-question:") ||
			!validIntakeDossierRef(string(decision.Choice), "intake-option:") ||
			(decision.AnswerText != "" && !intake.ValidAnswerText(decision.AnswerText)) ||
			!validIntakeDossierRef(string(decision.Recommendation), "intake-option:") ||
			!validIntakeDossierMessageKey(decision.RecommendationRationale) ||
			(decision.Origin != intake.OriginChat && decision.Origin != intake.OriginForm) ||
			decision.Revision == 0 || decision.Revision > stateRevision {
			return errIntakeDossierSnapshotInvalid
		}
		if _, duplicate := seen[decision.QuestionRef]; duplicate {
			return errIntakeDossierSnapshotInvalid
		}
		seen[decision.QuestionRef] = struct{}{}
	}
	return nil
}

func validIntakeDossierDigest(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && hex.EncodeToString(decoded) == value
}

func invalidIntakeDossierSnapshot(cause error) error {
	if cause == nil {
		cause = errIntakeDossierSnapshotInvalid
	} else {
		cause = errors.Join(errIntakeDossierSnapshotInvalid, cause)
	}
	return &StateError{Code: StateInvalid, Cause: cause}
}
