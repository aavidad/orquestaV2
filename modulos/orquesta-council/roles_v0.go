package orquestacouncil

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// El rol NO se hereda del modelo. Se asigna en caliente segun el presupuesto
// disponible de cada miembro EN ESE MOMENTO, y el operador puede forzarlo. Cero
// nombres de modelo en el codigo: manana el que hoy tiene mas cuota puede tener
// la menos.
type RoleV0 string

const (
	RoleRevisorV0    RoleV0 = "revisor"
	RoleConsultorV0  RoleV0 = "consultor"
	RoleAdversarioV0 RoleV0 = "adversario"
	RoleSeguridadV0  RoleV0 = "seguridad"
)

// El consultor decide, no lee: recibe material masticado (resumen, opciones y
// pregunta), nunca diffs crudos. Por eso es el rol del miembro con menos
// presupuesto.
type MaterialKindV0 string

const (
	MaterialDiffCrudoV0 MaterialKindV0 = "diff_crudo"
	MaterialMasticadoV0 MaterialKindV0 = "masticado"
)

type MemberV0 struct {
	MemberRef string
	FamilyRef string
	// BudgetRemaining en [0,1]: fraccion de cuota que le queda AHORA.
	BudgetRemaining float64
	// CapabilityRank: cuanto mas alto, mas capaz. Solo decide el rol SEGURIDAD.
	CapabilityRank int
}

type RoleOverrideV0 struct {
	Role      RoleV0
	MemberRef string
	// ForcedBy y Reason quedan en la evidencia: se audita si forzar fue buena idea.
	ForcedBy string
	Reason   string
}

type ConvocationV0 struct {
	CouncilRef string
	AuthorRef  string
	// SecurityCritical convoca al rol SEGURIDAD, que tiene derecho de veto.
	SecurityCritical bool
	Members          []MemberV0
	Overrides        []RoleOverrideV0
}

type SeatV0 struct {
	Role      RoleV0
	MemberRef string
	FamilyRef string
	Material  MaterialKindV0
	// Forced y AutomaticMemberRef permiten auditar el override: quien habria
	// salido por presupuesto si nadie hubiera forzado nada.
	Forced             bool
	ForcedBy           string
	ForcedReason       string
	AutomaticMemberRef string
	BudgetWarning      string
}

type AssignmentV0 struct {
	CouncilRef string
	AuthorRef  string
	Seats      []SeatV0
	Warnings   []string
}

var (
	ErrMiembrosInsuficientesV0      = errors.New("council_miembros_insuficientes")
	ErrAutorNoSeRevisaV0            = errors.New("council_autor_no_se_revisa_a_si_mismo")
	ErrAdversarioMismaFamiliaV0     = errors.New("council_adversario_misma_familia_que_autor")
	ErrOverrideMiembroDesconocidoV0 = errors.New("council_override_miembro_desconocido")
	ErrRolDesconocidoV0             = errors.New("council_rol_desconocido")
	ErrSeguridadSinVetoV0           = errors.New("council_seguridad_no_puede_quedar_sin_asignar")
)

const budgetWarningThresholdV0 = 0.10

// AssignRolesV0 aplica la precedencia acordada con el operador:
//
//  1. override manual   ← gana siempre
//  2. asignacion automatica por presupuesto observado en caliente
//
// El override manda en el REPARTO, nunca en las REGLAS DE INTEGRIDAD: no puede
// poner al autor a revisarse, ni de adversario a alguien de su misma familia, ni
// dejar sin cubrir el rol de seguridad cuando la decision es critica.
func AssignRolesV0(convocation ConvocationV0) (AssignmentV0, error) {
	author := strings.TrimSpace(convocation.AuthorRef)
	candidates := elegiblesV0(convocation.Members, author)
	if len(candidates) < 2 {
		return AssignmentV0{}, fmt.Errorf("%w: hacen falta al menos dos miembros distintos del autor", ErrMiembrosInsuficientesV0)
	}

	roles := []RoleV0{RoleRevisorV0, RoleConsultorV0, RoleAdversarioV0}
	if convocation.SecurityCritical {
		roles = append(roles, RoleSeguridadV0)
	}

	automatic := asignacionAutomaticaV0(candidates, convocation.AuthorFamilyV0(), roles)
	overrides, err := overridesPorRolV0(convocation, roles)
	if err != nil {
		return AssignmentV0{}, err
	}

	assignment := AssignmentV0{CouncilRef: convocation.CouncilRef, AuthorRef: author}
	for _, role := range roles {
		seat := SeatV0{
			Role:               role,
			Material:           materialParaRolV0(role),
			AutomaticMemberRef: automatic[role],
			MemberRef:          automatic[role],
		}
		if override, forced := overrides[role]; forced {
			seat.MemberRef = override.MemberRef
			seat.Forced = true
			seat.ForcedBy = override.ForcedBy
			seat.ForcedReason = override.Reason
		}
		if seat.MemberRef == "" {
			if role == RoleSeguridadV0 {
				return AssignmentV0{}, ErrSeguridadSinVetoV0
			}
			continue
		}
		member, ok := miembroPorRefV0(convocation.Members, seat.MemberRef)
		if !ok {
			return AssignmentV0{}, fmt.Errorf("%w: %s", ErrOverrideMiembroDesconocidoV0, seat.MemberRef)
		}
		seat.FamilyRef = member.FamilyRef

		if err := validarIntegridadV0(role, member, author, convocation.AuthorFamilyV0()); err != nil {
			return AssignmentV0{}, err
		}
		// Forzar a alguien sin cuota AVISA pero OBEDECE: es decision del operador.
		if member.BudgetRemaining < budgetWarningThresholdV0 {
			seat.BudgetWarning = fmt.Sprintf(
				"%s tiene %.0f%% de cuota para el rol %s", member.MemberRef, member.BudgetRemaining*100, role,
			)
			assignment.Warnings = append(assignment.Warnings, seat.BudgetWarning)
		}
		assignment.Seats = append(assignment.Seats, seat)
	}
	return assignment, nil
}

// validarIntegridadV0 son las reglas que el override NO puede saltarse.
func validarIntegridadV0(role RoleV0, member MemberV0, author string, authorFamily string) error {
	if member.MemberRef == author {
		return fmt.Errorf("%w: %s no puede ocupar el rol %s de su propia entrega", ErrAutorNoSeRevisaV0, author, role)
	}
	if role == RoleAdversarioV0 && authorFamily != "" && member.FamilyRef == authorFamily {
		return fmt.Errorf(
			"%w: %s es de la familia %s, la del autor; el adversario existe para romper, no para arropar",
			ErrAdversarioMismaFamiliaV0, member.MemberRef, member.FamilyRef,
		)
	}
	return nil
}

func (convocation ConvocationV0) AuthorFamilyV0() string {
	if member, ok := miembroPorRefV0(convocation.Members, convocation.AuthorRef); ok {
		return member.FamilyRef
	}
	return ""
}

// asignacionAutomaticaV0: revisor = mas presupuesto (traga diffs crudos);
// consultor = menos presupuesto (decide sobre material masticado); adversario =
// familia distinta a la del autor; seguridad = mayor capacidad.
func asignacionAutomaticaV0(candidates []MemberV0, authorFamily string, roles []RoleV0) map[RoleV0]string {
	porPresupuesto := append([]MemberV0(nil), candidates...)
	sort.SliceStable(porPresupuesto, func(i, j int) bool {
		if porPresupuesto[i].BudgetRemaining != porPresupuesto[j].BudgetRemaining {
			return porPresupuesto[i].BudgetRemaining > porPresupuesto[j].BudgetRemaining
		}
		return porPresupuesto[i].MemberRef < porPresupuesto[j].MemberRef
	})

	asignados := map[RoleV0]string{}
	tomados := map[string]bool{}
	for _, role := range roles {
		var elegido string
		switch role {
		case RoleRevisorV0:
			elegido = primeroLibreV0(porPresupuesto, tomados, nil)
		case RoleConsultorV0:
			elegido = ultimoLibreV0(porPresupuesto, tomados)
		case RoleAdversarioV0:
			elegido = primeroLibreV0(porPresupuesto, tomados, func(member MemberV0) bool {
				return authorFamily == "" || member.FamilyRef != authorFamily
			})
		case RoleSeguridadV0:
			// Seguridad es un DERECHO DE VETO, no una silla mas: puede recaer en
			// alguien que ya ocupe otro rol. El operador lo dijo: un sabio puede
			// tener mas de un rol. Lo que no se comparte nunca es revisor y
			// adversario, que son los cuatro ojos.
			elegido = masCapazV0(porPresupuesto)
		}
		if elegido != "" {
			asignados[role] = elegido
			tomados[elegido] = true
		}
	}
	return asignados
}

func materialParaRolV0(role RoleV0) MaterialKindV0 {
	if role == RoleConsultorV0 {
		return MaterialMasticadoV0
	}
	return MaterialDiffCrudoV0
}

func overridesPorRolV0(convocation ConvocationV0, roles []RoleV0) (map[RoleV0]RoleOverrideV0, error) {
	permitidos := map[RoleV0]bool{}
	for _, role := range roles {
		permitidos[role] = true
	}
	overrides := map[RoleV0]RoleOverrideV0{}
	for _, override := range convocation.Overrides {
		role := RoleV0(strings.TrimSpace(string(override.Role)))
		if !permitidos[role] {
			return nil, fmt.Errorf("%w: %s", ErrRolDesconocidoV0, override.Role)
		}
		if _, ok := miembroPorRefV0(convocation.Members, override.MemberRef); !ok {
			return nil, fmt.Errorf("%w: %s", ErrOverrideMiembroDesconocidoV0, override.MemberRef)
		}
		overrides[role] = override
	}
	return overrides, nil
}

func elegiblesV0(members []MemberV0, author string) []MemberV0 {
	elegibles := make([]MemberV0, 0, len(members))
	for _, member := range members {
		if strings.TrimSpace(member.MemberRef) == "" || member.MemberRef == author {
			continue
		}
		elegibles = append(elegibles, member)
	}
	return elegibles
}

func miembroPorRefV0(members []MemberV0, ref string) (MemberV0, bool) {
	for _, member := range members {
		if member.MemberRef == strings.TrimSpace(ref) {
			return member, true
		}
	}
	return MemberV0{}, false
}

func primeroLibreV0(members []MemberV0, tomados map[string]bool, filtro func(MemberV0) bool) string {
	for _, member := range members {
		if tomados[member.MemberRef] {
			continue
		}
		if filtro != nil && !filtro(member) {
			continue
		}
		return member.MemberRef
	}
	return ""
}

func ultimoLibreV0(members []MemberV0, tomados map[string]bool) string {
	for idx := len(members) - 1; idx >= 0; idx-- {
		if !tomados[members[idx].MemberRef] {
			return members[idx].MemberRef
		}
	}
	return ""
}

func masCapazV0(members []MemberV0) string {
	mejor := ""
	rank := 0
	for _, member := range members {
		if mejor == "" || member.CapabilityRank > rank {
			mejor = member.MemberRef
			rank = member.CapabilityRank
		}
	}
	return mejor
}
