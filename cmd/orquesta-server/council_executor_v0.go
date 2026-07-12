package main

import (
	"context"
	"fmt"

	council "orquesta/modulos/orquesta-council"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

// councilExecutorV0 cablea el consejo de sabios sobre su dominio. El adaptador no
// decide nada: traduce. Toda la autoridad (roles en caliente, precedencia del
// override, veto de seguridad, umbral) vive en el nucleo.
type councilExecutorV0 struct{}

var _ orquestamcp.MCPCouncilPortV0 = councilExecutorV0{}

func (councilExecutorV0) ConveneCouncilV0(
	_ context.Context,
	input orquestamcp.MCPCouncilToolInputV0,
) (orquestamcp.MCPCouncilToolResultV0, error) {
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
		return result, nil
	default:
		return orquestamcp.MCPCouncilToolResultV0{}, fmt.Errorf("council_action_desconocida: %q", input.Action)
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
