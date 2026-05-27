package orquestaappdirectorservice

import (
	"context"
	orquestacorereplanner "orquesta/modulos/orquesta-core-replanner"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func (source splitReviewReworkReplanSourceV0) BuildReviewReworkReplanPlansV0(
	_ context.Context,
	request orquestacionnucleoapp.ReviewReworkReplanPlanRequestV0,
) ([]orquestacionnucleoapp.ReviewReworkReplanPlanV0, error) {
	return []orquestacionnucleoapp.ReviewReworkReplanPlanV0{{
		CandidateRef:     "candidate-ref-service-review-split-001",
		ReplanRef:        "replan-ref-service-review-split-001",
		SignalRef:        "signal-ref-service-review-split-001",
		ReworkRequestRef: "rework-request-ref-service-review-split-001",
		TaskRef:          "task-ref-service-review-original-001",
		ReasonRef:        "review-split-required",
		RequestedAction:  orquestacorereplanner.ReplanActionSplitTaskV0,
		Summary:          "Dividir retrabajo de revision.",
		EvidenceRefs:     []string{"evidence-ref-service-review-split-001"},
		ReviewResult: orquestacoreworkflow.ReviewResultV0{
			ReviewResultRef: "review-result-ref-service-review-split-001",
			ReviewRequestID: "review-request-ref-service-review-split-001",
			DeliveryRef:     "delivery-ref-service-review-split-001",
			Status:          orquestacoreworkflow.ReviewResultStatusChangesRequestedV0,
			Summary:         "Cambios requeridos.",
			EvidenceRefs:    []string{"evidence-ref-review-result-service-split-001"},
		},
		SplitTasks: []orquestacoreworkflow.WorkflowTaskV0{
			serviceReviewReworkSplitTaskV0(request.Run.RunID),
		},
	}}, nil
}

func serviceReviewReworkSplitTaskV0(runRef string) orquestacoreworkflow.WorkflowTaskV0 {
	return orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion: orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:        "task-ref-service-review-split-a",
		RunID:         runRef,
		PhaseID:       orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Title:         "Microtarea de retrabajo",
		Summary:       "Corregir entrega dividida por revision.",
		WriteSet:      []string{"app/rework_split_a.go"},
		AcceptanceCriteria: []string{
			"Entrega corregida lista para revision.",
		},
		RequiredTests: []string{"go test ./..."},
	}
}

func serviceReviewOriginalTaskV0(runRef string) orquestacoreworkflow.WorkflowTaskV0 {
	return orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion: orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:        "task-ref-service-review-original-001",
		RunID:         runRef,
		PhaseID:       orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Title:         "Microtarea original",
		Summary:       "Trabajo original entregado a revision.",
		WriteSet:      []string{"app/original.go"},
		AcceptanceCriteria: []string{
			"Entrega inicial trazable.",
		},
	}
}

func serviceWorkflowProfileResolverTaskV0(runRef string) orquestacoreworkflow.WorkflowTaskV0 {
	return orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:   orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:          "task-ref-service-profile-resolver-001",
		RunID:           runRef,
		PhaseID:         orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Title:           "Microtarea con perfil inyectado",
		Summary:         "Validar que la composicion puede elegir el perfil.",
		WriteSet:        []string{"app/profile_resolver.go"},
		WorkProfileKind: orquestacoreworkflow.WorkProfileImplementationV0,
		AcceptanceCriteria: []string{
			"El candidato usa el perfil resuelto por la composicion.",
		},
		RequiredTests: []string{"go test ./modulos/orquesta-app-director-service -count=1"},
	}
}

type recordingWorkflowTaskProfileResolverForTestV0 struct {
	called      bool
	lastTaskRef string
	resolution  orquestacionnucleoapp.WorkflowTaskProfileResolutionV0
}

func (resolver *recordingWorkflowTaskProfileResolverForTestV0) ResolveWorkflowTaskProfileV0(
	_ context.Context,
	request orquestacionnucleoapp.WorkflowTaskProfileRequestV0,
) (orquestacionnucleoapp.WorkflowTaskProfileResolutionV0, error) {
	resolver.called = true
	resolver.lastTaskRef = request.Task.TaskID
	return resolver.resolution, nil
}

type recordingReviewGateSourceV0 struct {
	called bool
}

func (source *recordingReviewGateSourceV0) BuildReviewGateObservationsV0(
	_ context.Context,
	_ orquestacionnucleoapp.ReviewGateObservationRequestV0,
) ([]orquestacionnucleoapp.ReviewGateObservationV0, error) {
	source.called = true
	return []orquestacionnucleoapp.ReviewGateObservationV0{{
		ReviewRequestID: "review-request-ref-service-review-gate-001",
		ReviewResultRef: "review-result-ref-service-review-gate-001",
		DeliveryRef:     "delivery-ref-service-review-gate-001",
		PhaseID:         string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
		Status:          orquestacoreworkflow.ReviewResultStatusChangesRequestedV0,
		Summary:         "Revision de composicion.",
		EvidenceRefs:    []string{"evidence-ref-service-review-gate-001"},
	}}, nil
}
