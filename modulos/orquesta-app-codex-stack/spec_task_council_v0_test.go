package orquestaappcodexstack

import (
	"context"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadecisioncouncil "orquesta/modulos/orquesta-decision-council"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestCodexLaunchSpecResolverV0MaterializaPacketConsejoPropuestaCriticaVoto(t *testing.T) {
	cases := []struct {
		name             string
		role             string
		taskRef          string
		phase            orquestacoreworkflow.OrchestrationPhaseIDV0
		expectedArtifact string
		wantObjective    []string
	}{
		{
			name:             "proposal",
			role:             orquestadecisioncouncil.CouncilRoleProposalV0,
			taskRef:          "task-council-p-001",
			phase:            orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0,
			expectedArtifact: orquestadecisioncouncil.CouncilArtifactProposalV0,
			wantObjective:    []string{"Propon una opcion independiente", "No cierres la arquitectura global"},
		},
		{
			name:             "critique",
			role:             orquestadecisioncouncil.CouncilRoleCritiqueV0,
			taskRef:          "task-council-c-001",
			phase:            orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0,
			expectedArtifact: orquestadecisioncouncil.CouncilArtifactCritiqueV0,
			wantObjective:    []string{"Critica la propuesta asignada", "critica cruzada verificable"},
		},
		{
			name:             "vote",
			role:             orquestadecisioncouncil.CouncilRoleVoteV0,
			taskRef:          "task-council-v-001",
			phase:            orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0,
			expectedArtifact: orquestadecisioncouncil.CouncilArtifactVoteV0,
			wantObjective:    []string{"architecture_vote.v0", "task_ref", "vote_ref", "option_ref"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			task := codexStackCouncilWorkflowTaskForTestV0(tc.taskRef, tc.phase, tc.role, tc.expectedArtifact)
			resolver := CodexLaunchSpecResolverV0{
				Config: CodexRuntimeConfigV0{
					RuntimeWorkDir: t.TempDir(),
					ProjectWorkDir: t.TempDir(),
				},
				TaskStore: orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(task),
			}

			spec, err := resolver.ResolveExternalAgentLaunchSpecV0(
				context.Background(),
				orquestaruntime.AgentLauncherInboundV0{
					CorrelationID: "corr-council-packet-" + tc.name,
					Payload: &orquestaruntime.LaunchRuntimeAgentRequestV0{
						RunID:          task.RunID,
						AgentRequestID: "agent-ref-council-" + tc.name,
						PhaseID:        string(tc.phase),
						Role:           tc.role,
						TaskRef:        task.TaskID,
						Summary:        "consejo_residente",
						SkillRefs:      []string{"skill-ref-decision-council-test"},
					},
				},
			)
			if err != nil {
				t.Fatalf("ResolveExternalAgentLaunchSpecV0: %v", err)
			}

			packet := spec.Spec.AgentPacket
			if packet.TargetModule != "orquesta-app-stack-decision_council" ||
				packet.Task.Title != task.Title ||
				packet.Task.CohortRef != task.CohortRef ||
				packet.Task.WaveRef != task.WaveRef ||
				!stringInSetV0(packet.Task.SkillRefs, "skill-ref-decision-council-test") ||
				!stringInSetV0(packet.Task.DoneCriteria, "expected_artifact:"+tc.expectedArtifact) {
				t.Fatalf("packet consejo incompleto: %+v", packet)
			}
			for _, want := range append([]string{
				"Trabajo de consejo residente multiagente",
				"Rol estructurado: " + tc.role,
				"Artefacto esperado: " + tc.expectedArtifact,
			}, tc.wantObjective...) {
				if !strings.Contains(packet.Task.Objective, want) {
					t.Fatalf("objective no contiene %q:\n%s", want, packet.Task.Objective)
				}
			}
			if contextBundleHasRequiredRefOnlyEntryV0(packet.Context) {
				t.Fatalf("contexto de consejo no debe caer en ref_only generico: %+v", packet.Context.Entries)
			}
			for _, want := range []string{
				"decision_council_context.v0",
				`"decision_council_role":"` + tc.role + `"`,
				`"expected_artifact":"` + tc.expectedArtifact + `"`,
				`"agent_ref":"agent-ref-council-role"`,
				`"family_ref":"family-ref-council-role"`,
			} {
				if !codexStackContextContainsForTestV0(packet.Context.Entries, want) {
					t.Fatalf("contexto no contiene %q: %+v", want, packet.Context.Entries)
				}
			}
		})
	}
}

func TestCodexLaunchSpecResolverV0DetectaConsejoPorMetadataSinRolPayloadV0(t *testing.T) {
	task := codexStackCouncilWorkflowTaskForTestV0(
		"task-council-v-002",
		orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0,
		orquestadecisioncouncil.CouncilRoleVoteV0,
		orquestadecisioncouncil.CouncilArtifactVoteV0,
	)
	resolver := CodexLaunchSpecResolverV0{
		Config: CodexRuntimeConfigV0{
			RuntimeWorkDir: t.TempDir(),
			ProjectWorkDir: t.TempDir(),
		},
		TaskStore: orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(task),
	}

	spec, err := resolver.ResolveExternalAgentLaunchSpecV0(
		context.Background(),
		orquestaruntime.AgentLauncherInboundV0{
			CorrelationID: "corr-council-packet-metadata",
			Payload: &orquestaruntime.LaunchRuntimeAgentRequestV0{
				RunID:          task.RunID,
				AgentRequestID: "agent-ref-council-metadata",
				PhaseID:        string(task.PhaseID),
				Role:           "",
				TaskRef:        task.TaskID,
				Summary:        "sin role textual",
			},
		},
	)
	if err != nil {
		t.Fatalf("ResolveExternalAgentLaunchSpecV0: %v", err)
	}
	if spec.Spec.AgentPacket.TargetModule != "orquesta-app-stack-decision_council" ||
		!strings.Contains(spec.Spec.AgentPacket.Task.Objective, "architecture_vote.v0") {
		t.Fatalf("packet no detecto consejo por metadata: %+v", spec.Spec.AgentPacket)
	}
}

func codexStackCouncilWorkflowTaskForTestV0(
	taskRef string,
	phase orquestacoreworkflow.OrchestrationPhaseIDV0,
	role string,
	expectedArtifact string,
) orquestacoreworkflow.WorkflowTaskV0 {
	return orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion: orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:        taskRef,
		RunID:         "run-ref-council-packet-001",
		PhaseID:       phase,
		Title:         "Tarea consejo " + role,
		Summary:       "Resolver parte del consejo sin actuar como director global.",
		WriteSet:      []string{"artifacts/council/" + role},
		AcceptanceCriteria: []string{
			"decision_council_role:" + role,
			"assignment_ref:assignment-ref-council-role",
			"agent_ref:agent-ref-council-role",
			"family_ref:family-ref-council-role",
			"expected_artifact:" + expectedArtifact,
			"gate_ref:gate-ref-council-role",
			"minimum_artifacts:3",
			"minimum_distinct_families:3",
			"context_policy:small_isolated_refs_only",
		},
		ContextRefs: []string{
			"decision-council-role-" + codexStackCouncilRoleScopeForTestV0(role),
			"source-ref-council-topic",
		},
		DependsOn:      []string{"task-council-p-previous"},
		CohortRef:      "cohort-ref-council-packet",
		WaveRef:        "wave-ref-council-packet",
		SkillRefs:      []string{"skill-ref-council-local"},
		MaxChildAgents: 2,
	}
}

func codexStackCouncilRoleScopeForTestV0(role string) string {
	switch role {
	case orquestadecisioncouncil.CouncilRoleCritiqueV0:
		return "c"
	case orquestadecisioncouncil.CouncilRoleVoteV0:
		return "v"
	default:
		return "p"
	}
}
