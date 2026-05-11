package orquestaruntimecodexdelivery

import (
	"testing"

	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadirectoragentfilesource "orquesta/modulos/orquesta-director-agent-file-source"
)

func TestProgrammingTeamDirectorDecisionsV0SonValidas(t *testing.T) {
	tasks := programmingTeamTasksV0()
	decisions := programmingTeamDirectorDecisionsV0("run-ref-programming-team-real-001", tasks)
	if len(decisions) != len(tasks)+6 {
		t.Fatalf("decisions=%d tasks=%d", len(decisions), len(tasks))
	}
	for _, decision := range decisions {
		if issues := orquestadirectoragent.ValidateDirectorAgentDecisionV0(decision); len(issues) > 0 {
			t.Fatalf("decision invalida %s: %+v", decision.DecisionRef, issues)
		}
	}
	data := []byte(programmingTeamDirectorDecisionsJSONV0("run-ref-programming-team-real-001", tasks))
	decoded, err := orquestadirectoragentfilesource.DecodeDirectorAgentDecisionFileV0(data)
	if err != nil {
		t.Fatalf("DecodeDirectorAgentDecisionFileV0: %v", err)
	}
	if len(decoded) != len(decisions) {
		t.Fatalf("decoded=%d decisions=%d", len(decoded), len(decisions))
	}
	packet := programmingTeamDirectorPacketV0(
		"agent-ref-director",
		"task-ref-director",
		"run-ref-programming-team-real-001",
		"brainstorming_arquitectura",
		"corr-programming-team-001",
		tasks,
	)
	spec := codexDeliveryLoopSpecForTestV0(packet.RequestID, packet.Task.TaskRef)
	spec.AgentPacket = packet
	if !spec.Valid() {
		t.Fatalf("director spec invalido: %+v", spec.Validate())
	}
}
