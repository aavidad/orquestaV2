package orquestadirectoragentworkflow

import (
	"testing"

	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

func TestBuildDirectorAgentWorkflowCommandV0RechazaPlanEquipoComoComandoWorkflow(t *testing.T) {
	_, issues := BuildDirectorAgentWorkflowCommandV0(DirectorAgentWorkflowCommandRequestV0{
		Decision:   validDirectorAgentWorkflowPlanTeamDecisionV0(),
		OccurredAt: "2026-05-09T00:00:00Z",
	})

	requireDirectorAgentWorkflowIssueV0(t, issues, "director_agent_command_no_soportado")
}

func validDirectorAgentWorkflowPlanTeamDecisionV0() orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-plan-team-001",
		RunID:         "run-ref-001",
		PhaseID:       orquestadirectoragent.DirectorAgentPlanningPhaseIDV0,
		CommandType:   orquestadirectoragent.DirectorAgentCommandProposePlanTeamV0,
		CommandRef:    "command-ref-plan-team-001",
		Summary:       "Proponer plan y equipo autonomo.",
		EvidenceRefs:  []string{"evidence-ref-plan-team-001"},
		ProposePlanTeam: &orquestadirectoragent.DirectorAgentPlanTeamCommandV0{
			Plan: orquestadirectoragent.DirectorAgentAutonomousPlanTeamV0{
				SchemaVersion: orquestadirectoragent.DirectorAgentAutonomousPlanTeamSchemaVersionV0,
				PlanRef:       "plan-ref-autonomous-team-001",
				RunID:         "run-ref-001",
				PhaseID:       orquestadirectoragent.DirectorAgentPlanningPhaseIDV0,
				GoalRef:       "goal-ref-autonomous-build-001",
				Summary:       "Dividir trabajo en unidades coordinadas.",
				Team: []orquestadirectoragent.DirectorAgentTeamMemberV0{
					{MemberRef: "member-ref-domain-001", Role: "dominio"},
				},
				WorkUnits: []orquestadirectoragent.DirectorAgentAutonomousWorkUnitV0{
					{
						WorkUnitRef:       "work-unit-ref-domain-001",
						PhaseID:           "programacion",
						Title:             "Implementar dominio",
						Summary:           "Crear unidad de dominio compacta.",
						AssignedMemberRef: "member-ref-domain-001",
						WriteSet:          []string{"internal/domain"},
						AcceptanceCriteria: []string{
							"Compila con pruebas unitarias.",
						},
						FunctionContractRefs: []orquestadirectoragent.DirectorAgentFunctionContractRefV0{
							{ContractRef: "contract:function:domain:v0"},
						},
					},
				},
			},
		},
	}
}
