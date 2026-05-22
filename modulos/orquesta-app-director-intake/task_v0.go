package orquestaappdirectorintake

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func directorTaskFromAppSpecV0(spec AppDirectorInputSpecV0) AppDirectorTaskV0 {
	appRef := appDirectorTaskRefPrefixV0(spec)
	return directorTaskForAreaV0(appRef, primaryDirectorTaskAreaV0(spec))
}

func directorTaskForAreaV0(appRef string, area directorTaskAreaV0) AppDirectorTaskV0 {
	suffix := safeDirectorIntakeRefPartV0(area.Suffix)
	return AppDirectorTaskV0{
		TaskRef:        "task-" + appRef + "-" + suffix,
		TopicRef:       "topic-" + appRef + "-" + suffix,
		BrainstormRef:  "brainstorm-" + appRef + "-" + suffix,
		ClaimRef:       "claim-" + appRef + "-" + suffix,
		CapacityRef:    "capacity-" + appRef + "-" + suffix,
		AgentRequestID: "agent-" + appRef + "-" + suffix,
		PhaseID:        orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0,
		Role:           area.Role,
		Capacity:       area.Capacity,
		Summary:        area.Summary,
		WriteSet:       area.WriteSet,
		EvidenceRefs:   []string{"evidence-ref-" + appRef + "-" + suffix + "-intake"},
	}
}
