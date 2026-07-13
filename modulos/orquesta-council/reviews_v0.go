package orquestacouncil

import (
	"errors"
	"fmt"
	"strings"
)

// ReviewReceiptV0 es la revision de UNA entrega material por UN revisor. No la
// firma el autor: la palabra del implementador nunca acredita su propio trabajo.
type ReviewReceiptV0 struct {
	ReviewerRef string
	FamilyRef   string
	Verdict     VoteV0
	EvidenceRef string
}

var (
	ErrRevisionDelAutorV0        = errors.New("council_revision_del_autor_no_acredita")
	ErrRevisionesInsuficientesV0 = errors.New("council_revisiones_independientes_insuficientes")
	ErrRevisionDuplicadaV0       = errors.New("council_revision_duplicada")
	ErrSinRevisionAdversariaV0   = errors.New("council_sin_revision_de_familia_distinta")
	ErrRevisionSinEvidenciaV0    = errors.New("council_revision_sin_evidencia")
	ErrEntregaNoAprobadaV0       = errors.New("council_entrega_no_aprobada_por_revision")
)

// MinimoRevisionesIndependientesV0: "cuatro ojos son mejores que dos". Dos
// revisores DISTINTOS, ninguno el autor.
const MinimoRevisionesIndependientesV0 = 2

// ValidateIndependentReviewsV0 gobierna el cierre de una entrega material.
//
// No basta con que el codigo pase los tests: una entrega se cierra cuando la han
// mirado dos pares independientes, y al menos uno desde fuera de la familia del
// autor. Un revisor de la misma familia tiende a compartir sus puntos ciegos;
// para eso existe el adversario.
func ValidateIndependentReviewsV0(
	authorRef string,
	authorFamily string,
	reviews []ReviewReceiptV0,
) error {
	author := strings.TrimSpace(authorRef)
	vistos := map[string]bool{}
	aprobaciones := 0
	familiaDistinta := false

	for _, review := range reviews {
		reviewer := strings.TrimSpace(review.ReviewerRef)
		if reviewer == "" {
			continue
		}
		if reviewer == author {
			return fmt.Errorf("%w: %s", ErrRevisionDelAutorV0, reviewer)
		}
		if vistos[reviewer] {
			return fmt.Errorf("%w: %s", ErrRevisionDuplicadaV0, reviewer)
		}
		vistos[reviewer] = true

		if strings.TrimSpace(review.EvidenceRef) == "" {
			return fmt.Errorf("%w: %s", ErrRevisionSinEvidenciaV0, reviewer)
		}
		// Un bloqueo o un rework impiden el cierre, por muchas aprobaciones que
		// haya: el cierre exige acuerdo, no mayoria.
		switch review.Verdict {
		case VoteApproveV0:
			aprobaciones++
			if authorFamily != "" && strings.TrimSpace(review.FamilyRef) != authorFamily {
				familiaDistinta = true
			}
		case VoteReworkV0, VoteBlockV0:
			return fmt.Errorf("%w: %s pidio %s", ErrEntregaNoAprobadaV0, reviewer, review.Verdict)
		default:
			return fmt.Errorf("%w: %q", ErrVotoDesconocidoV0, review.Verdict)
		}
	}

	if aprobaciones < MinimoRevisionesIndependientesV0 {
		return fmt.Errorf(
			"%w: %d revisiones independientes, hacen falta %d",
			ErrRevisionesInsuficientesV0, aprobaciones, MinimoRevisionesIndependientesV0,
		)
	}
	if authorFamily != "" && !familiaDistinta {
		return fmt.Errorf(
			"%w: todas las revisiones son de la familia %s, la del autor",
			ErrSinRevisionAdversariaV0, authorFamily,
		)
	}
	return nil
}
