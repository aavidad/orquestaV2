package orquestadirectorcandidates

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func commandMetaV0(
	input SchedulableWorkCandidateInputV0,
	commandID string,
	idempotencyKey string,
) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      strings.TrimSpace(commandID),
		RunID:          input.RunRef,
		IdempotencyKey: strings.TrimSpace(idempotencyKey),
		CorrelationID:  input.Commands.CorrelationID,
		RequestedBy:    input.Commands.RequestedBy,
		OccurredAt:     input.Commands.OccurredAt,
	}
}

func candidateErrorV0(field string) DirectorCandidateErrorV0 {
	return DirectorCandidateErrorV0{Code: ErrDirectorCandidateInvalidoV0, Field: field}
}
