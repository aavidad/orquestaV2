package orquestaappcodexstack

import (
	"context"
	"errors"
	"testing"

	council "orquesta/modulos/orquesta-council"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

type cierreFalsoV0 struct{ accepted bool }

func (fake cierreFalsoV0) ValidateGoalWorkClosureV0(
	context.Context, orquestagoal.GoalWorkSpecV0, orquestagoal.GoalWorkResultV0,
) (orquestagoal.GoalClosureValidationV0, error) {
	return orquestagoal.GoalClosureValidationV0{Status: "complete", Accepted: fake.accepted}, nil
}

type revisionesFalsasV0 struct {
	autor   string
	familia string
	reviews []council.ReviewReceiptV0
	err     error
}

func (fake revisionesFalsasV0) ObserveDeliveryReviewsV0(
	context.Context, string,
) (string, string, []council.ReviewReceiptV0, error) {
	return fake.autor, fake.familia, fake.reviews, fake.err
}

func validar(t *testing.T, validator orquestagoal.GoalWorkClosureValidatorPortV0) orquestagoal.GoalClosureValidationV0 {
	t.Helper()
	validation, err := validator.ValidateGoalWorkClosureV0(
		context.Background(),
		orquestagoal.GoalWorkSpecV0{RunRef: "run-1"},
		orquestagoal.GoalWorkResultV0{},
	)
	if err != nil {
		t.Fatalf("ValidateGoalWorkClosureV0: %v", err)
	}
	return validation
}

// Los tests verdes no sustituyen a dos pares de ojos: una entrega sin doble
// revision independiente NO se cierra, aunque el nucleo la aceptara.
func TestCierreSinDobleRevisionVuelveARework(t *testing.T) {
	validator := newCouncilDoubleReviewClosureValidatorV0(
		cierreFalsoV0{accepted: true},
		CouncilDoubleReviewConfigV0{Required: true, Reviews: revisionesFalsasV0{
			autor: "autor", familia: "familia-a",
			reviews: []council.ReviewReceiptV0{
				{ReviewerRef: "revisor-1", FamilyRef: "familia-b", Verdict: council.VoteApproveV0, EvidenceRef: "e1"},
			},
		}},
	)
	validation := validar(t, validator)
	if validation.Accepted || !validation.NeedsRework {
		t.Fatalf("una sola revision no cierra la entrega: %+v", validation)
	}
	if len(validation.Issues) == 0 || validation.Issues[0].Code != CouncilDoubleReviewMissingCodeV0 {
		t.Fatalf("no delato la falta de doble revision: %+v", validation.Issues)
	}
}

func TestCierreConDosRevisionesIndependientesSiCierra(t *testing.T) {
	validator := newCouncilDoubleReviewClosureValidatorV0(
		cierreFalsoV0{accepted: true},
		CouncilDoubleReviewConfigV0{Required: true, Reviews: revisionesFalsasV0{
			autor: "autor", familia: "familia-a",
			reviews: []council.ReviewReceiptV0{
				{ReviewerRef: "revisor-1", FamilyRef: "familia-a", Verdict: council.VoteApproveV0, EvidenceRef: "e1"},
				{ReviewerRef: "revisor-2", FamilyRef: "familia-b", Verdict: council.VoteApproveV0, EvidenceRef: "e2"},
			},
		}},
	)
	validation := validar(t, validator)
	if !validation.Accepted {
		t.Fatalf("dos revisiones independientes deben cerrar: %+v", validation)
	}
}

// El mismo fail-closed que el gate: exigir la doble revision sin fuente de
// revisiones NO puede dejar pasar la entrega.
func TestDobleRevisionExigidaSinFuenteFallaCerrado(t *testing.T) {
	validator := newCouncilDoubleReviewClosureValidatorV0(
		cierreFalsoV0{accepted: true},
		CouncilDoubleReviewConfigV0{Required: true, Reviews: nil},
	)
	validation := validar(t, validator)
	if validation.Accepted {
		t.Fatal("fail-open: sin fuente de revisiones se cerro la entrega igualmente")
	}
}

// Si el nucleo ya rechaza el cierre, el decorador no lo resucita.
func TestDobleRevisionNoResucitaUnCierreRechazado(t *testing.T) {
	validator := newCouncilDoubleReviewClosureValidatorV0(
		cierreFalsoV0{accepted: false},
		CouncilDoubleReviewConfigV0{Required: true, Reviews: revisionesFalsasV0{err: errors.New("da igual")}},
	)
	if validar(t, validator).Accepted {
		t.Fatal("un cierre rechazado por el nucleo no puede aceptarse aqui")
	}
}
