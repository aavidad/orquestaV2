package orquestaappdirectorintake

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestafactory "orquesta/modulos/orquesta-factory"
)

func directorTaskFromAppSpecV0(spec orquestafactory.AppSpecV0) AppDirectorTaskV0 {
	appRef := appDirectorTaskRefPrefixV0(spec)
	return directorTaskForAreaV0(appRef, directorTaskAreaV0{
		Suffix:   "director",
		Role:     "director",
		Summary:  directorTaskSummaryV0(spec, "Dirigir solicitud de app con contexto pequeno."),
		Capacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
		WriteSet: []string{"docs/arquitectura.md", "docs/plan_microtareas.md"},
	})
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
