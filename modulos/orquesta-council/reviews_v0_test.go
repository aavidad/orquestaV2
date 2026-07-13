package orquestacouncil_test

import (
	"errors"
	"testing"

	council "orquesta/modulos/orquesta-council"
)

func revisionV0(reviewer, familia string, veredicto council.VoteV0) council.ReviewReceiptV0 {
	return council.ReviewReceiptV0{
		ReviewerRef: reviewer, FamilyRef: familia, Verdict: veredicto, EvidenceRef: "evidencia-" + reviewer,
	}
}

// Cuatro ojos son mejores que dos, y al menos un par debe venir de fuera de la
// familia del autor: los de dentro comparten sus puntos ciegos.
func TestCierreExigeDosRevisionesIndependientesV0(t *testing.T) {
	err := council.ValidateIndependentReviewsV0("autor", "familia-a", []council.ReviewReceiptV0{
		revisionV0("revisor-1", "familia-a", council.VoteApproveV0),
		revisionV0("revisor-2", "familia-b", council.VoteApproveV0),
	})
	if err != nil {
		t.Fatalf("dos revisiones independientes con una de otra familia deben cerrar: %v", err)
	}

	// Una sola no basta.
	err = council.ValidateIndependentReviewsV0("autor", "familia-a", []council.ReviewReceiptV0{
		revisionV0("revisor-1", "familia-b", council.VoteApproveV0),
	})
	if !errors.Is(err, council.ErrRevisionesInsuficientesV0) {
		t.Fatalf("una sola revision no cierra una entrega: %v", err)
	}

	// Dos, pero ambas de la familia del autor: falta el adversario.
	err = council.ValidateIndependentReviewsV0("autor", "familia-a", []council.ReviewReceiptV0{
		revisionV0("revisor-1", "familia-a", council.VoteApproveV0),
		revisionV0("revisor-2", "familia-a", council.VoteApproveV0),
	})
	if !errors.Is(err, council.ErrSinRevisionAdversariaV0) {
		t.Fatalf("todas las revisiones de la familia del autor no acreditan: %v", err)
	}
}

// La palabra del implementador nunca acredita su propio trabajo.
func TestElAutorNoAcreditaSuPropiaEntregaV0(t *testing.T) {
	err := council.ValidateIndependentReviewsV0("autor", "familia-a", []council.ReviewReceiptV0{
		revisionV0("autor", "familia-a", council.VoteApproveV0),
		revisionV0("revisor-2", "familia-b", council.VoteApproveV0),
	})
	if !errors.Is(err, council.ErrRevisionDelAutorV0) {
		t.Fatalf("el autor no puede revisarse a si mismo: %v", err)
	}
}

// Repetir al mismo revisor no crea un segundo par de ojos: es el mismo mirando
// dos veces. Es el mismo fallo del 'consejo de uno', aplicado al cierre.
func TestRevisorRepetidoNoCuentaDosVecesV0(t *testing.T) {
	err := council.ValidateIndependentReviewsV0("autor", "familia-a", []council.ReviewReceiptV0{
		revisionV0("revisor-1", "familia-b", council.VoteApproveV0),
		revisionV0("revisor-1", "familia-b", council.VoteApproveV0),
	})
	if !errors.Is(err, council.ErrRevisionDuplicadaV0) {
		t.Fatalf("el mismo revisor dos veces no son dos revisiones: %v", err)
	}
}

// Un rework o un bloqueo impiden el cierre por muchas aprobaciones que haya: el
// cierre exige acuerdo, no mayoria.
func TestUnReworkImpideElCierreV0(t *testing.T) {
	err := council.ValidateIndependentReviewsV0("autor", "familia-a", []council.ReviewReceiptV0{
		revisionV0("revisor-1", "familia-b", council.VoteApproveV0),
		revisionV0("revisor-2", "familia-b", council.VoteApproveV0),
		revisionV0("revisor-3", "familia-c", council.VoteReworkV0),
	})
	if !errors.Is(err, council.ErrEntregaNoAprobadaV0) {
		t.Fatalf("un rework impide el cierre pese a dos aprobaciones: %v", err)
	}
}

// Una revision sin evidencia es una opinion, no una revision.
func TestRevisionSinEvidenciaNoAcreditaV0(t *testing.T) {
	err := council.ValidateIndependentReviewsV0("autor", "familia-a", []council.ReviewReceiptV0{
		{ReviewerRef: "revisor-1", FamilyRef: "familia-b", Verdict: council.VoteApproveV0},
		revisionV0("revisor-2", "familia-b", council.VoteApproveV0),
	})
	if !errors.Is(err, council.ErrRevisionSinEvidenciaV0) {
		t.Fatalf("una revision sin evidencia no acredita: %v", err)
	}
}
