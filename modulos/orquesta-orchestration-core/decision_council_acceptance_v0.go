package orquestacionnucleoapp

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadecisioncouncil "orquesta/modulos/orquesta-decision-council"
)

type DecisionCouncilAcceptDecisionCommandRequestV0 struct {
	Run               orquestacoreworkflow.OrchestrationRunV0
	VoteRef           string
	DecisionRef       string
	AcceptedOptionRef string
	Summary           string
	VoteResult        orquestadecisioncouncil.DecisionCouncilVoteResultV0
	OccurredAt        string
	CorrelationID     string
	RequestedBy       string
}

func BuildDecisionCouncilAcceptDecisionCommandV0(
	request DecisionCouncilAcceptDecisionCommandRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, bool, error) {
	request.VoteRef = strings.TrimSpace(request.VoteRef)
	request.DecisionRef = strings.TrimSpace(request.DecisionRef)
	request.AcceptedOptionRef = firstNonEmptyV0(request.AcceptedOptionRef, request.VoteResult.AcceptedOptionRef)
	if !request.VoteResult.Accepted {
		return orquestacoreworkflow.OrchestrationCommandV0{}, false, nil
	}
	if request.DecisionRef == "" {
		return orquestacoreworkflow.OrchestrationCommandV0{}, false, errorV0(ErrNucleoOrquestacionInvalidoV0, "decision_ref", "decision_ref requerido")
	}
	if request.VoteRef == "" || !stringInSetV0(request.VoteRef, request.Run.Votes) {
		return orquestacoreworkflow.OrchestrationCommandV0{}, false, errorV0(ErrNucleoOrquestacionInvalidoV0, "vote_ref", "vote_ref no reflejado")
	}
	if stringInSetV0(request.DecisionRef, request.Run.Decisions) {
		return orquestacoreworkflow.OrchestrationCommandV0{}, false, nil
	}
	command, err := orquestacoreworkflow.NewAcceptDecisionCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-accept-" + operationalDirectorSafeRefPartV0(request.DecisionRef),
			RunID:          request.Run.RunID,
			IdempotencyKey: "idem-accept-" + operationalDirectorSafeRefPartV0(request.DecisionRef),
			CorrelationID:  request.CorrelationID,
			RequestedBy:    operationalDirectorRequestedByV0(request.RequestedBy),
			OccurredAt:     request.OccurredAt,
		},
		orquestacoreworkflow.AcceptDecisionCommandPayloadV0{
			DecisionRef:       request.DecisionRef,
			PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0),
			VoteRef:           request.VoteRef,
			AcceptedOptionRef: request.AcceptedOptionRef,
			Summary:           firstNonEmptyV0(request.Summary, "Decision aceptada por consejo multiagente."),
			EvidenceRefs:      request.VoteResult.EvidenceRefs,
		},
	)
	return command, err == nil, err
}
