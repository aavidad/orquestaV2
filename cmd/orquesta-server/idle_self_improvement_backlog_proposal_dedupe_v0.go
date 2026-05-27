package main

import (
	"sort"
	"strings"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func idleSelfImprovementDedupeBacklogProposalsV0(
	sections []idleSelfImprovementBacklogSectionV0,
	excluded map[string]bool,
) ([]idleSelfImprovementBacklogSectionV0, []orquestaserver.BacklogScanCollisionV0, map[string]bool) {
	var out []idleSelfImprovementBacklogSectionV0
	var collisions []orquestaserver.BacklogScanCollisionV0
	blocked := map[string]bool{}
	for _, section := range sections {
		if section.Completed || idleSelfImprovementBacklogIntentionalSubshardV0(section) {
			out = append(out, section)
			continue
		}
		duplicateIndex := -1
		for index, prior := range out {
			if prior.Completed || idleSelfImprovementBacklogIntentionalSubshardV0(prior) {
				continue
			}
			if idleSelfImprovementBacklogProposalsOverlapV0(prior, section) {
				duplicateIndex = index
				break
			}
		}
		if duplicateIndex < 0 {
			out = append(out, section)
			continue
		}
		canonical := out[duplicateIndex]
		collisions = append(collisions, idleSelfImprovementBacklogProposalCollisionV0(canonical, section))
		if idleSelfImprovementBacklogProposalKnownV0(canonical, excluded) ||
			idleSelfImprovementBacklogProposalKnownV0(section, excluded) {
			blocked[idleSelfImprovementCanonicalSectionRefV0(canonical.Ref)] = true
			blocked[idleSelfImprovementCanonicalSectionRefV0(section.Ref)] = true
		}
	}
	return out, compactBacklogScanCollisionsV0(collisions), blocked
}

func idleSelfImprovementBacklogProposalsOverlapV0(
	left idleSelfImprovementBacklogSectionV0,
	right idleSelfImprovementBacklogSectionV0,
) bool {
	if !idleSelfImprovementBacklogProposalScopeOverlapV0(left.Scope, right.Scope) {
		return false
	}
	if idleSelfImprovementBacklogProposalObjectiveV0(left) != "" &&
		idleSelfImprovementBacklogProposalObjectiveV0(left) == idleSelfImprovementBacklogProposalObjectiveV0(right) {
		return true
	}
	leftTokens := idleSelfImprovementBacklogProposalTokensV0(left)
	rightTokens := idleSelfImprovementBacklogProposalTokensV0(right)
	if len(leftTokens) == 0 || len(rightTokens) == 0 {
		return false
	}
	return idleSelfImprovementBacklogTokenOverlapV0(leftTokens, rightTokens) >= 0.72
}

func idleSelfImprovementBacklogProposalObjectiveV0(section idleSelfImprovementBacklogSectionV0) string {
	return idleSelfImprovementHeadingRefV0(section.Objective)
}

func idleSelfImprovementBacklogProposalContextRefsV0(section idleSelfImprovementBacklogSectionV0) []string {
	return []string{"backlog_proposal_fingerprint:" + idleSelfImprovementBacklogProposalFingerprintV0(section)}
}

func idleSelfImprovementBacklogProposalFingerprintV0(section idleSelfImprovementBacklogSectionV0) string {
	parts := []string{
		strings.Join(idleSelfImprovementBacklogProposalTokensV0(section), ","),
		strings.Join(idleSelfImprovementBacklogNormalizeListV0(section.Scope), ","),
		strings.Join(idleSelfImprovementBacklogNormalizeListV0(section.Tests), ","),
	}
	return idleSelfImprovementBacklogHashV0(strings.Join(parts, "|"))
}

func idleSelfImprovementBacklogProposalTokensV0(section idleSelfImprovementBacklogSectionV0) []string {
	text := strings.Join([]string{section.Heading, section.Objective}, " ")
	seen := map[string]bool{}
	var out []string
	for _, raw := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
	}) {
		token := strings.Trim(raw, "0123456789")
		if len(token) < 4 || idleSelfImprovementBacklogProposalStopWordV0(token) || seen[token] {
			continue
		}
		seen[token] = true
		out = append(out, token)
	}
	sort.Strings(out)
	return out
}

func idleSelfImprovementBacklogProposalScopeOverlapV0(left []string, right []string) bool {
	left = idleSelfImprovementBacklogNormalizeListV0(left)
	right = idleSelfImprovementBacklogNormalizeListV0(right)
	if len(left) == 0 || len(right) == 0 {
		return len(left) == len(right)
	}
	seen := map[string]bool{}
	for _, value := range left {
		seen[value] = true
	}
	intersection := 0
	for _, value := range right {
		if seen[value] {
			intersection++
		}
	}
	minLen := len(left)
	if len(right) < minLen {
		minLen = len(right)
	}
	return minLen > 0 && float64(intersection)/float64(minLen) >= 0.75
}

func idleSelfImprovementBacklogTokenOverlapV0(left []string, right []string) float64 {
	seen := map[string]bool{}
	for _, value := range left {
		seen[value] = true
	}
	intersection := 0
	for _, value := range right {
		if seen[value] {
			intersection++
		}
	}
	minLen := len(left)
	if len(right) < minLen {
		minLen = len(right)
	}
	if minLen == 0 {
		return 0
	}
	return float64(intersection) / float64(minLen)
}

func idleSelfImprovementBacklogNormalizeListV0(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if normalized := idleSelfImprovementHeadingRefV0(value); normalized != "" {
			out = append(out, normalized)
		}
	}
	sort.Strings(out)
	return compactServerStackStringsV0(out)
}

func idleSelfImprovementBacklogIntentionalSubshardV0(section idleSelfImprovementBacklogSectionV0) bool {
	text := strings.ToLower(strings.Join(append(append([]string{section.Heading, section.Objective}, section.Criteria...), section.ContextRefs()...), " "))
	return strings.Contains(text, "subshard") ||
		strings.Contains(text, "sub-shard") ||
		(strings.Contains(text, "frontera reducida") &&
			strings.Contains(text, "criterio no cubierto") &&
			strings.Contains(text, "motivo causal"))
}

func (section idleSelfImprovementBacklogSectionV0) ContextRefs() []string {
	return append(append([]string(nil), section.Inputs...), section.Outputs...)
}

func idleSelfImprovementBacklogProposalKnownV0(
	section idleSelfImprovementBacklogSectionV0,
	excluded map[string]bool,
) bool {
	sectionRef := idleSelfImprovementCanonicalSectionRefV0(section.Ref)
	for ref := range excluded {
		normalized := idleSelfImprovementNormalizeQueuedRequestRefV0(ref)
		if normalized == sectionRef ||
			strings.Contains(normalized, "request-ref-autoprogramming-backlog-"+sectionRef+"-") {
			return true
		}
	}
	return false
}

func idleSelfImprovementBacklogProposalCollisionV0(
	canonical idleSelfImprovementBacklogSectionV0,
	duplicate idleSelfImprovementBacklogSectionV0,
) orquestaserver.BacklogScanCollisionV0 {
	return orquestaserver.BacklogScanCollisionV0{
		Code:       "duplicate_backlog_proposal_fingerprint",
		SectionRef: duplicate.Ref,
		Message: "canonical_ref=" + canonical.Ref +
			";proposal_fingerprint=" + idleSelfImprovementBacklogProposalFingerprintV0(canonical),
		EvidenceRefs: []string{
			"evidence-ref-autoprogramming-backlog-proposal-deduplication-fingerprint",
		},
	}
}

func idleSelfImprovementBacklogProposalStopWordV0(token string) bool {
	switch token {
	case "para", "como", "este", "esta", "esto", "tarea", "tareas", "backlog", "pendiente", "pendientes":
		return true
	default:
		return false
	}
}
