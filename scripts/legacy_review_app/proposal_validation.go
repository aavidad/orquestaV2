// Este fichero valida el filtro de admisión expresado por una propuesta.
// No decide su aplicación: solo impide registros incompletos o incompatibles.
package main

import (
	"strings"
	"unicode"
)

const maxProposalReferenceBytes = 200

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
		!validProposalItemRef(request.ItemRef) || !validProposalItemRevision(request.ItemRevision) ||
		request.ExpectedRevision < 0 || !validProposalIdempotencyKey(request.IdempotencyKey) ||
		!validCanonicalOpaqueRef(request.ActorRef) || !validCanonicalOpaqueRef(request.ProjectRef) ||
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

func validProposalItemRef(value string) bool {
	return value != "" && len(value) <= maxProposalReferenceBytes &&
		value == strings.TrimSpace(value) &&
		strings.IndexFunc(value, unicode.IsControl) == -1
}

func validProposalItemRevision(value string) bool {
	const prefix = "sha256:"
	if len(value) != len(prefix)+64 || !strings.HasPrefix(value, prefix) {
		return false
	}
	for _, character := range value[len(prefix):] {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

func validCanonicalOpaqueRef(value string) bool {
	return value == strings.TrimSpace(value) && validOpaqueRef(value) &&
		strings.IndexFunc(value, unicode.IsControl) == -1
}

func validProposalIdempotencyKey(value string) bool {
	if len(value) < 32 || len(value) > maxProposalReferenceBytes ||
		value != strings.TrimSpace(value) {
		return false
	}
	return strings.IndexFunc(value, func(character rune) bool {
		return unicode.IsSpace(character) || unicode.IsControl(character)
	}) == -1
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
