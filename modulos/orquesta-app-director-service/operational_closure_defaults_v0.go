package orquestaappdirectorservice

import (
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func appDirectorClosureRequestWithDefaultsV0(
	request ContinueAppDirectorRequestV0,
	closureRequest orquestacionnucleoapp.OperationalDirectorClosureRequestV0,
) orquestacionnucleoapp.OperationalDirectorClosureRequestV0 {
	if closureRequest.RunRef == "" {
		closureRequest.RunRef = request.RunRef
	}
	if closureRequest.OccurredAt == "" {
		closureRequest.OccurredAt = request.OccurredAt
	}
	if closureRequest.CorrelationID == "" {
		closureRequest.CorrelationID = request.CorrelationID
	}
	if closureRequest.RequestedBy == "" {
		closureRequest.RequestedBy = request.RequestedBy
	}
	return closureRequest
}
