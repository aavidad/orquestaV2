package orquestacouncil

import (
	"errors"
	"fmt"
	"strings"
)

type VoteV0 string

const (
	VoteApproveV0 VoteV0 = "approve"
	VoteReworkV0  VoteV0 = "rework"
	// VoteBlockV0 solo lo puede emitir el rol SEGURIDAD, y VETA: no se compensa
	// con mayoria. Ninguna cantidad de aprobaciones levanta un veto de seguridad.
	VoteBlockV0 VoteV0 = "block"
)

type BallotV0 struct {
	MemberRef string
	Role      RoleV0
	Vote      VoteV0
	Reason    string
}

type OutcomeV0 string

const (
	OutcomeAcceptedV0 OutcomeV0 = "council_decision_accepted"
	OutcomeReworkV0   OutcomeV0 = "council_decision_rework"
	OutcomeBlockedV0  OutcomeV0 = "council_decision_blocked"
)

type DecisionV0 struct {
	CouncilRef string
	Outcome    OutcomeV0
	Approvals  int
	Reworks    int
	Blocks     int
	Total      int
	// ApprovalRatio es informativo; la regla es el umbral, no el redondeo.
	ApprovalRatio float64
	Rationale     string
	Ballots       []BallotV0
}

var (
	ErrVotanteNoConvocadoV0 = errors.New("council_votante_no_convocado")
	ErrVotoDuplicadoV0      = errors.New("council_voto_duplicado")
	ErrVetoSinSeguridadV0   = errors.New("council_veto_reservado_a_seguridad")
	ErrQuorumIncompletoV0   = errors.New("council_quorum_incompleto")
	ErrVotoDesconocidoV0    = errors.New("council_voto_desconocido")
)

// El umbral acordado son DOS TERCIOS, comparados en aritmetica entera para que
// no dependa del redondeo: aprobaciones*3 >= total*2.
//
//	3 convocados -> hacen falta 2      4 convocados -> hacen falta 3
//	6 convocados -> hacen falta 4
//
// Un EMPATE (2-2 de 4, es decir 50%) queda por debajo del umbral, luego es
// rework. No hay desempate arbitrario: si el consejo no se pone de acuerdo, el
// trabajo vuelve; no se acepta por sorteo ni por voto de calidad.
func alcanzaElUmbralV0(approvals int, total int) bool {
	return total > 0 && approvals*3 >= total*2
}

func tieneRolV0(roles []RoleV0, buscado RoleV0) bool {
	for _, role := range roles {
		if role == buscado {
			return true
		}
	}
	return false
}

// Un miembro puede ocupar MAS DE UN ROL (p.ej. adversario y seguridad a la vez).
// En ese caso sigue teniendo UNA sola voz: una persona, un voto, aunque lleve dos
// sombreros. Lo que le dan sus roles es el DERECHO a vetar, no votos extra.
func DecideV0(assignment AssignmentV0, ballots []BallotV0) (DecisionV0, error) {
	rolesPorMiembro := map[string][]RoleV0{}
	for _, seat := range assignment.Seats {
		rolesPorMiembro[seat.MemberRef] = append(rolesPorMiembro[seat.MemberRef], seat.Role)
	}

	decision := DecisionV0{CouncilRef: assignment.CouncilRef, Ballots: ballots}
	vistos := map[string]bool{}
	for _, ballot := range ballots {
		roles, convocado := rolesPorMiembro[strings.TrimSpace(ballot.MemberRef)]
		if !convocado {
			return DecisionV0{}, fmt.Errorf("%w: %s", ErrVotanteNoConvocadoV0, ballot.MemberRef)
		}
		if vistos[ballot.MemberRef] {
			return DecisionV0{}, fmt.Errorf("%w: %s", ErrVotoDuplicadoV0, ballot.MemberRef)
		}
		vistos[ballot.MemberRef] = true

		switch ballot.Vote {
		case VoteApproveV0:
			decision.Approvals++
		case VoteReworkV0:
			decision.Reworks++
		case VoteBlockV0:
			if !tieneRolV0(roles, RoleSeguridadV0) {
				return DecisionV0{}, fmt.Errorf(
					"%w: %s ocupa %v, no seguridad", ErrVetoSinSeguridadV0, ballot.MemberRef, roles,
				)
			}
			decision.Blocks++
		default:
			return DecisionV0{}, fmt.Errorf("%w: %q", ErrVotoDesconocidoV0, ballot.Vote)
		}
	}

	if len(vistos) != len(rolesPorMiembro) {
		return DecisionV0{}, fmt.Errorf(
			"%w: votaron %d de %d convocados", ErrQuorumIncompletoV0, len(vistos), len(rolesPorMiembro),
		)
	}

	decision.Total = len(rolesPorMiembro)
	if decision.Total > 0 {
		decision.ApprovalRatio = float64(decision.Approvals) / float64(decision.Total)
	}

	switch {
	case decision.Blocks > 0:
		decision.Outcome = OutcomeBlockedV0
		decision.Rationale = "veto de seguridad: ninguna mayoria lo levanta"
	case alcanzaElUmbralV0(decision.Approvals, decision.Total):
		decision.Outcome = OutcomeAcceptedV0
		decision.Rationale = fmt.Sprintf(
			"%d de %d aprobaciones alcanzan el umbral de dos tercios", decision.Approvals, decision.Total,
		)
	default:
		decision.Outcome = OutcomeReworkV0
		decision.Rationale = fmt.Sprintf(
			"%d de %d aprobaciones no alcanzan el umbral de dos tercios; el empate es rework, no aceptacion",
			decision.Approvals, decision.Total,
		)
	}
	return decision, nil
}
