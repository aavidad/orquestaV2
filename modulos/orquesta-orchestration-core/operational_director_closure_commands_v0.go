package orquestacionnucleoapp

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

type operationalDirectorClosureCommandBuilderV0 func(
	OperationalDirectorClosureRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, error)

func operationalDirectorCloseTaskCommandV0(
	request OperationalDirectorClosureRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, error) {
	return orquestacoreworkflow.NewCloseTaskCommandV0(
		operationalDirectorClosureCommandMetaV0(request, "close-task", request.TaskID),
		orquestacoreworkflow.CloseTaskCommandPayloadV0{
			TaskID:            request.TaskID,
			PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			DeliveryRef:       request.DeliveryRef,
			AcceptedReviewRef: request.AcceptedReviewRef,
			Summary:           operationalDirectorClosureSummaryV0(request, "Cierre causal de microtarea del Director Operativo."),
			EvidenceRefs:      operationalDirectorClosureEvidenceRefsV0(request),
		},
	)
}

func operationalDirectorOpenRevisionForCloseTaskCommandV0(
	request OperationalDirectorClosureRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, error) {
	return orquestacoreworkflow.NewOpenPhaseCommandV0(
		operationalDirectorClosureCommandMetaV0(request, "open-revision-for-close-task", request.TaskID),
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			Reason:  "Normalizar fase de review antes de cerrar microtarea.",
		},
	)
}

func operationalDirectorOpenFinalValidationCommandV0(
	request OperationalDirectorClosureRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, error) {
	return orquestacoreworkflow.NewOpenPhaseCommandV0(
		operationalDirectorClosureCommandMetaV0(request, "open-final-validation", request.ValidationRef),
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseValidacionFinalV0),
			Reason:  "Validar cierre causal del Director Operativo.",
		},
	)
}

func operationalDirectorRegisterFinalValidationCommandV0(
	request OperationalDirectorClosureRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, error) {
	return orquestacoreworkflow.NewRegisterFinalValidationCommandV0(
		operationalDirectorClosureCommandMetaV0(request, "register-final-validation", request.ValidationRef),
		orquestacoreworkflow.RegisterFinalValidationCommandPayloadV0{
			ValidationRef: request.ValidationRef,
			PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseValidacionFinalV0),
			ClosedTaskRef: request.TaskID,
			Summary:       operationalDirectorClosureSummaryV0(request, "Validacion final con review y tests requeridos."),
			EvidenceRefs:  operationalDirectorClosureEvidenceRefsV0(request),
		},
	)
}

func operationalDirectorOpenClosureCommandV0(
	request OperationalDirectorClosureRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, error) {
	return orquestacoreworkflow.NewOpenPhaseCommandV0(
		operationalDirectorClosureCommandMetaV0(request, "open-closure", request.ClosureRef),
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseCierreV0),
			Reason:  "Cerrar run del Director Operativo.",
		},
	)
}

func operationalDirectorCloseRunCommandV0(
	request OperationalDirectorClosureRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, error) {
	return orquestacoreworkflow.NewCloseRunCommandV0(
		operationalDirectorClosureCommandMetaV0(request, "close-run", request.ClosureRef),
		orquestacoreworkflow.CloseRunCommandPayloadV0{
			ClosureRef:    request.ClosureRef,
			PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseCierreV0),
			ValidationRef: request.ValidationRef,
			Summary:       operationalDirectorClosureSummaryV0(request, "Run cerrado por Director Operativo."),
			EvidenceRefs:  operationalDirectorClosureEvidenceRefsV0(request),
		},
	)
}

func operationalDirectorClosureCommandMetaV0(
	request OperationalDirectorClosureRequestV0,
	action string,
	ref string,
) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	ref = strings.TrimSpace(ref)
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      "cmd-operational-director-" + action + "-" + ref,
		RunID:          request.RunRef,
		IdempotencyKey: "idem-operational-director-" + action + "-" + ref,
		CorrelationID:  request.CorrelationID,
		RequestedBy:    operationalDirectorClosureRequestedByV0(request),
		OccurredAt:     request.OccurredAt,
	}
}
