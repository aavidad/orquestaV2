package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	council "orquesta/modulos/orquesta-council"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

// councilOverridesFileNameV0 guarda los overrides PERSISTENTES del operador: los
// que valen para todos los consejos hasta que los cambie, no solo para una
// peticion. El operador pidio los dos alcances.
const councilOverridesFileNameV0 = "council_overrides_v0.json"

// Precedencia acordada, de mayor a menor:
//
//  1. override de la peticion concreta
//  2. override persistente del operador
//  3. asignacion automatica por presupuesto
type councilPersistentOverridesV0 struct {
	SchemaVersion string                             `json:"schema_version"`
	Overrides     []orquestamcp.MCPCouncilOverrideV0 `json:"overrides"`
}

// councilPublicErrorClassifierV0 traduce el error TIPADO del dominio a un codigo
// publico. Nunca se vuelca err.Error() a la superficie: seria filtrar detalle
// interno al exterior.
func councilPublicErrorClassifierV0(err error) (string, string, bool) {
	switch {
	case errors.Is(err, ErrCouncilBudgetUnobservableV0):
		return "council_budget_unobservable", "members", true
	case errors.Is(err, ErrCouncilReceiptConflictV0):
		return "council_receipt_conflict", "council_ref", true
	case errors.Is(err, ErrCouncilReceiptCorruptV0):
		return "council_receipt_corrupt", "council_ref", true
	case errors.Is(err, council.ErrAutorNoSeRevisaV0):
		return "council_autor_no_se_revisa_a_si_mismo", "overrides", true
	case errors.Is(err, council.ErrAdversarioMismaFamiliaV0):
		return "council_adversario_misma_familia_que_autor", "overrides", true
	case errors.Is(err, council.ErrOverrideMiembroDesconocidoV0):
		return "council_override_miembro_desconocido", "overrides", true
	case errors.Is(err, council.ErrRolDesconocidoV0):
		return "council_rol_desconocido", "overrides", true
	case errors.Is(err, council.ErrMiembrosInsuficientesV0):
		return "council_miembros_insuficientes", "members", true
	case errors.Is(err, council.ErrSeguridadSinVetoV0):
		return "council_seguridad_no_puede_quedar_sin_asignar", "members", true
	case errors.Is(err, council.ErrVotanteNoConvocadoV0):
		return "council_votante_no_convocado", "ballots", true
	case errors.Is(err, council.ErrVotoDuplicadoV0):
		return "council_voto_duplicado", "ballots", true
	case errors.Is(err, council.ErrVetoSinSeguridadV0):
		return "council_veto_reservado_a_seguridad", "ballots", true
	case errors.Is(err, council.ErrQuorumIncompletoV0):
		return "council_quorum_incompleto", "ballots", true
	case errors.Is(err, council.ErrVotoDesconocidoV0):
		return "council_voto_desconocido", "ballots", true
	default:
		return "", "", false
	}
}

// councilExecutorV0 cablea el consejo de sabios sobre su dominio. El adaptador no
// decide nada: traduce. Toda la autoridad (roles en caliente, precedencia del
// override, veto de seguridad, umbral) vive en el nucleo.
type councilExecutorV0 struct {
	overridesPath string
	receipts      councilReceiptStoreV0
	members       orquestamcp.MCPCouncilMemberSourcePortV0
}

var _ orquestamcp.MCPCouncilPortV0 = councilExecutorV0{}

func newCouncilExecutorV0(stateDir string) (councilExecutorV0, error) {
	receipts, err := newCouncilReceiptStoreV0(stateDir)
	if err != nil {
		return councilExecutorV0{}, err
	}
	return councilExecutorV0{
		overridesPath: filepath.Join(stateDir, councilOverridesFileNameV0),
		receipts:      receipts,
	}, nil
}

func (executor councilExecutorV0) withMemberSourceV0(
	source orquestamcp.MCPCouncilMemberSourcePortV0,
) councilExecutorV0 {
	executor.members = source
	return executor
}

// overridesPersistentesV0 lee los overrides durables del operador. Si no hay
// fichero, no hay overrides: la ausencia no es un error.
func (executor councilExecutorV0) overridesPersistentesV0() []orquestamcp.MCPCouncilOverrideV0 {
	if executor.overridesPath == "" {
		return nil
	}
	bytes, err := os.ReadFile(executor.overridesPath)
	if err != nil {
		return nil
	}
	var persistentes councilPersistentOverridesV0
	if err := json.Unmarshal(bytes, &persistentes); err != nil {
		return nil
	}
	return persistentes.Overrides
}

// mergeOverridesV0 aplica la precedencia: lo que el operador manda en ESTA
// peticion pisa a su ajuste persistente, y ambos pisan al automatico.
func mergeOverridesV0(
	persistentes []orquestamcp.MCPCouncilOverrideV0,
	deLaPeticion []orquestamcp.MCPCouncilOverrideV0,
) []orquestamcp.MCPCouncilOverrideV0 {
	porRol := map[string]orquestamcp.MCPCouncilOverrideV0{}
	for _, override := range persistentes {
		porRol[override.Role] = override
	}
	for _, override := range deLaPeticion {
		porRol[override.Role] = override
	}
	fusionados := make([]orquestamcp.MCPCouncilOverrideV0, 0, len(porRol))
	for _, override := range porRol {
		fusionados = append(fusionados, override)
	}
	// Orden estable: recorrer un map da un orden distinto en cada llamada, y eso
	// bastaba para que la huella de la misma convocatoria cambiara.
	sort.Slice(fusionados, func(i, j int) bool { return fusionados[i].Role < fusionados[j].Role })
	return fusionados
}

func (executor councilExecutorV0) ConveneCouncilV0(
	ctx context.Context,
	input orquestamcp.MCPCouncilToolInputV0,
) (orquestamcp.MCPCouncilToolResultV0, error) {
	input.Overrides = mergeOverridesV0(executor.overridesPersistentesV0(), input.Overrides)

	// Sin miembros en la peticion se OBSERVAN los reales y su cuota en caliente.
	// Que el caller los aporte sirve para probar, pero no demuestra nada: quien
	// llama podria inventarse los presupuestos y, con ellos, el reparto de roles.
	if len(input.Members) == 0 && executor.members != nil {
		observados, err := executor.members.ObserveCouncilMembersV0(ctx)
		if err != nil {
			return orquestamcp.MCPCouncilToolResultV0{}, err
		}
		input.Members = observados
	}

	// Idempotencia: una decision ya tomada no se vuelve a tomar. Reconvocar el
	// mismo council_ref devuelve el recibo durable, no un veredicto nuevo, que
	// podria contradecir al anterior.
	fingerprint := councilInputFingerprintV0(input)
	if input.Action == orquestamcp.MCPCouncilActionDecideV0 {
		receipt, ok, err := executor.receipts.LoadV0(input.CouncilRef)
		if err != nil {
			return orquestamcp.MCPCouncilToolResultV0{}, err
		}
		if ok && receipt.Outcome != "" {
			// Mismo consejo y misma convocatoria: reintento, devuelve lo decidido.
			if receipt.InputFingerprint == fingerprint {
				return resultFromReceiptV0(receipt), nil
			}
			// Mismo nombre, convocatoria distinta: no es un reintento, es un choque.
			return orquestamcp.MCPCouncilToolResultV0{}, fmt.Errorf(
				"%w: %s", ErrCouncilReceiptConflictV0, input.CouncilRef,
			)
		}
	}

	assignment, err := council.AssignRolesV0(convocationFromMCPV0(input))
	if err != nil {
		return orquestamcp.MCPCouncilToolResultV0{}, err
	}
	result := orquestamcp.MCPCouncilToolResultV0{
		CouncilRef: assignment.CouncilRef,
		AuthorRef:  assignment.AuthorRef,
		Seats:      seatsToMCPV0(assignment.Seats),
		Warnings:   assignment.Warnings,
	}

	switch input.Action {
	case orquestamcp.MCPCouncilActionAssignV0:
		return result, nil
	case orquestamcp.MCPCouncilActionDecideV0:
		decision, err := council.DecideV0(assignment, ballotsFromMCPV0(input.Ballots))
		if err != nil {
			return orquestamcp.MCPCouncilToolResultV0{}, err
		}
		result.Outcome = string(decision.Outcome)
		result.Approvals = decision.Approvals
		result.Reworks = decision.Reworks
		result.Blocks = decision.Blocks
		result.Total = decision.Total
		result.Rationale = decision.Rationale
		// La decision se hace durable ANTES de devolverla: si el servidor cae
		// justo despues, el recibo ya esta en disco.
		guardado, err := executor.receipts.SaveV0(councilReceiptV0{
			CouncilRef:       result.CouncilRef,
			AuthorRef:        result.AuthorRef,
			InputFingerprint: fingerprint,
			Seats:            result.Seats,
			Warnings:         result.Warnings,
			Outcome:          result.Outcome,
			Approvals:        result.Approvals,
			Reworks:          result.Reworks,
			Blocks:           result.Blocks,
			Total:            result.Total,
			Rationale:        result.Rationale,
			Overrides:        input.Overrides,
		})
		if err != nil {
			return orquestamcp.MCPCouncilToolResultV0{}, err
		}
		// Si otro escritor gano la carrera con la MISMA convocatoria, el veredicto
		// que vale es el suyo: uno solo, no dos.
		return resultFromReceiptV0(guardado), nil
	default:
		return orquestamcp.MCPCouncilToolResultV0{}, fmt.Errorf("council_action_desconocida: %q", input.Action)
	}
}

func resultFromReceiptV0(receipt councilReceiptV0) orquestamcp.MCPCouncilToolResultV0 {
	return orquestamcp.MCPCouncilToolResultV0{
		CouncilRef: receipt.CouncilRef,
		AuthorRef:  receipt.AuthorRef,
		Seats:      receipt.Seats,
		Warnings:   receipt.Warnings,
		Outcome:    receipt.Outcome,
		Approvals:  receipt.Approvals,
		Reworks:    receipt.Reworks,
		Blocks:     receipt.Blocks,
		Total:      receipt.Total,
		Rationale:  receipt.Rationale,
	}
}

func convocationFromMCPV0(input orquestamcp.MCPCouncilToolInputV0) council.ConvocationV0 {
	members := make([]council.MemberV0, 0, len(input.Members))
	for _, member := range input.Members {
		members = append(members, council.MemberV0{
			MemberRef:       member.MemberRef,
			FamilyRef:       member.FamilyRef,
			BudgetRemaining: member.BudgetRemaining,
			CapabilityRank:  member.CapabilityRank,
		})
	}
	overrides := make([]council.RoleOverrideV0, 0, len(input.Overrides))
	for _, override := range input.Overrides {
		overrides = append(overrides, council.RoleOverrideV0{
			Role:      council.RoleV0(override.Role),
			MemberRef: override.MemberRef,
			ForcedBy:  override.ForcedBy,
			Reason:    override.Reason,
		})
	}
	return council.ConvocationV0{
		CouncilRef:       input.CouncilRef,
		AuthorRef:        input.AuthorRef,
		SecurityCritical: input.SecurityCritical,
		Members:          members,
		Overrides:        overrides,
	}
}

func seatsToMCPV0(seats []council.SeatV0) []orquestamcp.MCPCouncilSeatV0 {
	proyectados := make([]orquestamcp.MCPCouncilSeatV0, 0, len(seats))
	for _, seat := range seats {
		proyectados = append(proyectados, orquestamcp.MCPCouncilSeatV0{
			Role:               string(seat.Role),
			MemberRef:          seat.MemberRef,
			FamilyRef:          seat.FamilyRef,
			Material:           string(seat.Material),
			Forced:             seat.Forced,
			ForcedBy:           seat.ForcedBy,
			ForcedReason:       seat.ForcedReason,
			AutomaticMemberRef: seat.AutomaticMemberRef,
			BudgetWarning:      seat.BudgetWarning,
		})
	}
	return proyectados
}

func ballotsFromMCPV0(ballots []orquestamcp.MCPCouncilBallotV0) []council.BallotV0 {
	traducidas := make([]council.BallotV0, 0, len(ballots))
	for _, ballot := range ballots {
		traducidas = append(traducidas, council.BallotV0{
			MemberRef: ballot.MemberRef,
			Vote:      council.VoteV0(ballot.Vote),
			Reason:    ballot.Reason,
		})
	}
	return traducidas
}
