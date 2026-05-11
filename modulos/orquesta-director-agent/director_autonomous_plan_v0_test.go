package orquestadirectoragent

import "testing"

func TestValidateDirectorAgentDecisionV0AceptaPlanEquipoAutonomoCompacto(t *testing.T) {
	decision := validDirectorAgentPlanTeamDecisionV0()

	if issues := ValidateDirectorAgentDecisionV0(decision); len(issues) != 0 {
		t.Fatalf("issues inesperados: %+v", issues)
	}
}

func TestValidateDirectorAgentDecisionV0RechazaPlanEquipoConDetalleOperativo(t *testing.T) {
	decision := validDirectorAgentPlanTeamDecisionV0()
	decision.ProposePlanTeam.Plan.WorkUnits[0].Summary = "Elegir runtime concreto."

	requireDirectorAgentIssueV0(t,
		ValidateDirectorAgentDecisionV0(decision),
		"director_agent_texto_invalido",
	)
}

func TestValidateDirectorAgentDecisionV0RechazaPlanEquipoSinEquipo(t *testing.T) {
	decision := validDirectorAgentPlanTeamDecisionV0()
	decision.ProposePlanTeam.Plan.Team = nil

	requireDirectorAgentIssueV0(t,
		ValidateDirectorAgentDecisionV0(decision),
		"director_agent_lista_invalida",
	)
}

func TestValidateDirectorAgentDecisionV0RechazaPlanEquipoConAsignacionDesconocida(t *testing.T) {
	decision := validDirectorAgentPlanTeamDecisionV0()
	decision.ProposePlanTeam.Plan.WorkUnits[0].AssignedMemberRef = "member-ref-missing"

	requireDirectorAgentIssueV0(t,
		ValidateDirectorAgentDecisionV0(decision),
		"director_agent_ref_invalida",
	)
}

func validDirectorAgentPlanTeamDecisionV0() DirectorAgentDecisionV0 {
	return DirectorAgentDecisionV0{
		SchemaVersion: DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-plan-team-001",
		RunID:         "run-ref-001",
		PhaseID:       DirectorAgentPlanningPhaseIDV0,
		CommandType:   DirectorAgentCommandProposePlanTeamV0,
		CommandRef:    "command-ref-plan-team-001",
		Summary:       "Proponer plan y equipo autonomo.",
		EvidenceRefs:  []string{"evidence-ref-plan-team-001"},
		ProposePlanTeam: &DirectorAgentPlanTeamCommandV0{
			Plan: DirectorAgentAutonomousPlanTeamV0{
				SchemaVersion: DirectorAgentAutonomousPlanTeamSchemaVersionV0,
				PlanRef:       "plan-ref-autonomous-team-001",
				RunID:         "run-ref-001",
				PhaseID:       DirectorAgentPlanningPhaseIDV0,
				GoalRef:       "goal-ref-autonomous-build-001",
				Summary:       "Dividir trabajo en unidades coordinadas.",
				Team: []DirectorAgentTeamMemberV0{
					{
						MemberRef:          "member-ref-domain-001",
						Role:               "dominio",
						Capacity:           DirectorAgentCapacityMediumV0,
						ResponsibilityRefs: []string{"responsibility-ref-domain-001"},
					},
				},
				WorkUnits: []DirectorAgentAutonomousWorkUnitV0{
					{
						WorkUnitRef:       "work-unit-ref-domain-001",
						PhaseID:           "programacion",
						Title:             "Implementar dominio",
						Summary:           "Crear unidad de dominio compacta.",
						AssignedMemberRef: "member-ref-domain-001",
						WriteSet:          []string{"internal/domain"},
						AcceptanceCriteria: []string{
							"Compila con pruebas unitarias.",
							"Respeta contrato funcional publicado.",
						},
						FunctionContractRefs: []DirectorAgentFunctionContractRefV0{
							{ContractRef: "contract:function:domain:v0", FunctionName: "DomainUseCases"},
						},
					},
				},
				EvidenceRefs: []string{"evidence-ref-plan-team-002"},
			},
		},
	}
}
