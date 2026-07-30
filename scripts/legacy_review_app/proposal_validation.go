// Este fichero valida el filtro de admisión expresado por una propuesta.
// No decide su aplicación: solo impide registros incompletos o incompatibles.
package main

import "strings"

var requiredRules = []string{
	"hexagonal",
	"i18n",
	"single_lifecycle",
	"single_writer_scheduler",
	"config_credentials",
	"identity_permissions",
	"governed_effects",
	"no_legacy_dependency",
}

func validateProposalRequest(item inventoryItem, request proposalRequest) error {
	if request.ItemRef != item.ID || request.ItemRevision != item.Revision ||
		request.ExpectedRevision < 0 || len(request.IdempotencyKey) < 32 ||
		strings.TrimSpace(request.ActorRef) == "" || strings.TrimSpace(request.ProjectRef) == "" ||
		len(strings.TrimSpace(request.Reason)) < 10 || len(strings.TrimSpace(request.FoundedSolution)) < 10 ||
		request.Confidence < 0 || request.Confidence > 100 || !validDisposition(request.Disposition) {
		return errInvalidProposal
	}
	allCompliant := true
	if len(request.RuleCompliance) != len(requiredRules) {
		return errInvalidProposal
	}
	for _, rule := range requiredRules {
		compliant, exists := request.RuleCompliance[rule]
		if !exists {
			return errInvalidProposal
		}
		allCompliant = allCompliant && compliant
	}
	if request.Disposition == dispositionAdmitted && !allCompliant {
		return errInvalidProposal
	}
	if !allCompliant && len(strings.TrimSpace(request.RuleNotes)) < 10 {
		return errInvalidProposal
	}
	return nil
}

func validDisposition(value disposition) bool {
	switch value {
	case dispositionAdmitted, dispositionStudy, dispositionRejected, dispositionEvidence, dispositionDuplicate:
		return true
	default:
		return false
	}
}

func cloneRules(source map[string]bool) map[string]bool {
	result := make(map[string]bool, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
