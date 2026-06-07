package main

import (
	"strconv"
	"strings"

	orquestaserver "orquesta/modulos/orquesta-server"
)

const idleSelfImprovementBacklogCanonicalPreflightMaxEntriesV0 = 40

func idleSelfImprovementCanonicalPreflightBacklogSectionsV0(
	sections []idleSelfImprovementBacklogSectionV0,
	excluded map[string]bool,
) ([]idleSelfImprovementBacklogSectionV0, []orquestaserver.BacklogScanCollisionV0, map[string]bool) {
	out := make([]idleSelfImprovementBacklogSectionV0, 0, len(sections))
	var collisions []orquestaserver.BacklogScanCollisionV0
	blocked := map[string]bool{}
	for _, section := range sections {
		canonical, ok := idleSelfImprovementClosedCanonicalOverlapV0(out, section)
		if !ok {
			out = append(out, section)
			continue
		}
		collisions = append(collisions, idleSelfImprovementBacklogCanonicalPreflightCollisionV0(canonical, section))
		if idleSelfImprovementBacklogProposalKnownV0(canonical, excluded) ||
			idleSelfImprovementBacklogProposalKnownV0(section, excluded) {
			blocked[idleSelfImprovementCanonicalSectionRefV0(canonical.Ref)] = true
			blocked[idleSelfImprovementCanonicalSectionRefV0(section.Ref)] = true
		}
	}
	return out, compactBacklogScanCollisionsV0(collisions), blocked
}

func idleSelfImprovementClosedCanonicalOverlapV0(
	candidates []idleSelfImprovementBacklogSectionV0,
	section idleSelfImprovementBacklogSectionV0,
) (idleSelfImprovementBacklogSectionV0, bool) {
	if section.Completed || idleSelfImprovementBacklogIntentionalSubshardV0(section) {
		return idleSelfImprovementBacklogSectionV0{}, false
	}
	for _, candidate := range candidates {
		if !candidate.Completed || idleSelfImprovementBacklogIntentionalSubshardV0(candidate) {
			continue
		}
		if idleSelfImprovementBacklogProposalsOverlapV0(candidate, section) {
			return candidate, true
		}
	}
	return idleSelfImprovementBacklogSectionV0{}, false
}

func idleSelfImprovementBacklogCanonicalPreflightCollisionV0(
	canonical idleSelfImprovementBacklogSectionV0,
	duplicate idleSelfImprovementBacklogSectionV0,
) orquestaserver.BacklogScanCollisionV0 {
	return orquestaserver.BacklogScanCollisionV0{
		Code:       "backlog_scanner_canonical_preflight_required",
		SectionRef: duplicate.Ref,
		Message: "canonical_ref=" + canonical.Ref +
			";proposal_fingerprint=" + idleSelfImprovementBacklogProposalFingerprintV0(canonical),
		EvidenceRefs: []string{
			"evidence-ref-autoprogramming-backlog-scanner-canonical-preflight",
		},
	}
}

func idleSelfImprovementBacklogCanonicalPreflightBlockedV0(
	collisions []orquestaserver.BacklogScanCollisionV0,
) bool {
	for _, collision := range collisions {
		if strings.TrimSpace(collision.Code) == "backlog_scanner_canonical_preflight_required" {
			return true
		}
	}
	return false
}

func (planner idleSelfImprovementBacklogPlannerV0) withBacklogScannerCanonicalPreflightV0(
	request orquestaserver.IdleSelfImprovementRequestV0,
) orquestaserver.IdleSelfImprovementRequestV0 {
	sections, err := planner.loadBacklogSectionsV0()
	if err != nil {
		request.ContextRefs = compactServerStackStringsV0(append(request.ContextRefs,
			"backlog_scanner_canonical_preflight:unavailable",
		))
		return request
	}
	entries := idleSelfImprovementBacklogCanonicalPreflightEntriesV0(sections)
	digest := idleSelfImprovementBacklogHashV0(strings.Join(entries, "\n"))
	request.ContextRefs = compactServerStackStringsV0(append(request.ContextRefs,
		append([]string{
			"backlog_scanner_canonical_preflight:required",
			"backlog_scanner_canonical_preflight_index:" + digest,
		}, entries...)...,
	))
	request.AcceptanceCriteria = compactServerStackStringsV0(append(request.AcceptanceCriteria,
		"backlog_scanner_canonical_preflight_required",
		"antes de anadir un nuevo ## Txx, comparar owner/alcance/frontera/tests/evidencia contra backlog_scanner_canonical_preflight_index",
		"si solapa con canonica cerrada, documentar cobertura o regresion concreta; no crear Txx generico",
		"si solapa con canonica pendiente, declarar alias/fusion aditiva o conservar propuesta como borrador con reason backlog_scanner_canonical_preflight_required",
	))
	request.EvidenceRefs = compactServerStackStringsV0(append(request.EvidenceRefs,
		"evidence-ref-autoprogramming-backlog-scanner-canonical-preflight",
	))
	return request
}

func idleSelfImprovementBacklogCanonicalPreflightEntriesV0(
	sections []idleSelfImprovementBacklogSectionV0,
) []string {
	entries := make([]string, 0, len(sections))
	for _, section := range sections {
		fingerprint := idleSelfImprovementBacklogProposalFingerprintV0(section)
		if fingerprint == "" {
			continue
		}
		entries = append(entries,
			"backlog_canonical_preflight_entry:ref="+section.Ref+
				";state="+idleSelfImprovementBacklogCanonicalPreflightStateV0(section)+
				";line="+strconv.Itoa(section.SourceLine)+
				";proposal_fingerprint="+fingerprint,
		)
		if len(entries) >= idleSelfImprovementBacklogCanonicalPreflightMaxEntriesV0 {
			break
		}
	}
	return compactServerStackStringsV0(entries)
}

func idleSelfImprovementBacklogCanonicalPreflightStateV0(section idleSelfImprovementBacklogSectionV0) string {
	if section.Completed {
		return "closed"
	}
	if section.NeedsDocumentReview {
		return "review"
	}
	return "pending"
}
