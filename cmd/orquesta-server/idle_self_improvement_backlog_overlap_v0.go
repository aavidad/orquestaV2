package main

import (
	"sort"
	"strconv"
	"strings"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func idleSelfImprovementCanonicalizeOverlappingBacklogSectionsV0(
	sections []idleSelfImprovementBacklogSectionV0,
	excluded map[string]bool,
) ([]idleSelfImprovementBacklogSectionV0, []orquestaserver.BacklogScanCollisionV0, map[string]bool) {
	bySignature := map[string][]int{}
	for index, section := range sections {
		if section.Completed || strings.TrimSpace(section.CanonicalRef) != "" {
			continue
		}
		signature := idleSelfImprovementBacklogOverlapSignatureV0(section)
		if signature == "" {
			continue
		}
		bySignature[signature] = append(bySignature[signature], index)
	}
	skip := map[int]bool{}
	blockedCanonical := map[string]bool{}
	var collisions []orquestaserver.BacklogScanCollisionV0
	for signature, indexes := range bySignature {
		if len(indexes) < 2 {
			continue
		}
		sort.SliceStable(indexes, func(i, j int) bool {
			left, right := sections[indexes[i]], sections[indexes[j]]
			if left.SourcePath != right.SourcePath {
				return left.SourcePath < right.SourcePath
			}
			if left.SourceLine != right.SourceLine {
				return left.SourceLine < right.SourceLine
			}
			return left.Ref < right.Ref
		})
		canonicalIndex := indexes[0]
		canonicalRef := idleSelfImprovementCanonicalSectionRefV0(sections[canonicalIndex].Ref)
		for _, duplicateIndex := range indexes[1:] {
			duplicate := sections[duplicateIndex]
			sections[canonicalIndex] = idleSelfImprovementMergeOverlappingBacklogSectionV0(
				sections[canonicalIndex],
				duplicate,
				signature,
			)
			skip[duplicateIndex] = true
			if idleSelfImprovementDuplicateBacklogKnownV0(duplicate, excluded) {
				blockedCanonical[canonicalRef] = true
			}
			collisions = append(collisions, idleSelfImprovementBacklogOverlapCollisionV0(
				duplicate,
				canonicalRef,
				signature,
			))
		}
	}
	out := make([]idleSelfImprovementBacklogSectionV0, 0, len(sections))
	for index, section := range sections {
		if !skip[index] {
			out = append(out, section)
		}
	}
	return out, compactBacklogScanCollisionsV0(collisions), blockedCanonical
}

func idleSelfImprovementMergeOverlappingBacklogSectionV0(
	canonical idleSelfImprovementBacklogSectionV0,
	duplicate idleSelfImprovementBacklogSectionV0,
	signature string,
) idleSelfImprovementBacklogSectionV0 {
	canonical = idleSelfImprovementMergeCanonicalBacklogSectionV0(canonical, duplicate)
	canonical.Criteria = compactServerStackStringsV0(append(canonical.Criteria,
		"covered_by:"+canonical.Ref+
			";alias:"+duplicate.Ref+
			";reason:backlog_task_overlap_canonicalization_required"+
			";overlap_signature:"+signature,
	))
	canonical.StateEvidenceRefs = compactServerStackStringsV0(append(canonical.StateEvidenceRefs,
		"evidence-ref-autoprogramming-backlog-overlap-canonical-merge:"+duplicate.Ref,
	))
	return canonical
}

func idleSelfImprovementBacklogOverlapCollisionV0(
	section idleSelfImprovementBacklogSectionV0,
	canonicalRef string,
	signature string,
) orquestaserver.BacklogScanCollisionV0 {
	return orquestaserver.BacklogScanCollisionV0{
		Code:       "backlog_task_overlap_canonicalization_required",
		SectionRef: section.Ref,
		Message: "canonical_ref=" + canonicalRef +
			";covered_by=" + canonicalRef +
			";overlap_signature=" + signature +
			";source=" + strings.TrimSpace(section.SourcePath) +
			":" + strconv.Itoa(section.SourceLine),
		EvidenceRefs: []string{
			"evidence-ref-autoprogramming-backlog-overlap-canonical-merge",
		},
	}
}

func idleSelfImprovementBacklogOverlapSignatureV0(
	section idleSelfImprovementBacklogSectionV0,
) string {
	frontier := idleSelfImprovementBacklogOverlapFrontierV0(section)
	if frontier == "" {
		return ""
	}
	seed := strings.Join([]string{
		idleSelfImprovementBacklogOverlapOwnerV0(section),
		strings.Join(idleSelfImprovementNormalizeOverlapTokensV0(section.Scope), ","),
		frontier,
		strings.Join(idleSelfImprovementNormalizeOverlapTokensV0(section.Tests), ","),
		strings.Join(idleSelfImprovementNormalizeOverlapTokensV0(section.StateEvidenceRefs), ","),
	}, "|")
	return "backlog-task-overlap-" + idleSelfImprovementBacklogHashV0(seed)
}

func idleSelfImprovementBacklogOverlapOwnerV0(section idleSelfImprovementBacklogSectionV0) string {
	if owner := strings.TrimSpace(section.Owner); owner != "" {
		return owner
	}
	if len(section.Scope) > 0 {
		return strings.Join(idleSelfImprovementNormalizeOverlapTokensV0(section.Scope), ",")
	}
	return strings.TrimSpace(section.SourcePath)
}

func idleSelfImprovementBacklogOverlapFrontierV0(section idleSelfImprovementBacklogSectionV0) string {
	text := idleSelfImprovementNormalizeOverlapTextV0(strings.Join(append(
		[]string{section.Heading, section.Objective},
		section.Criteria...,
	), " "))
	hasBacklogScan := strings.Contains(text, "backlog") &&
		idleSelfImprovementBacklogTextContainsAnyV0(text, "escaneo", "scanner", "scan")
	hasIdentity := idleSelfImprovementBacklogTextContainsAnyV0(text,
		"identidad", "identity", "scan ref", "scan entry ref", "scan entry")
	hasOrder := idleSelfImprovementBacklogTextContainsAnyV0(text,
		"orden", "sequence", "secuencia", "ordinal", "salto", "monotono", "monotónico")
	hasEntry := idleSelfImprovementBacklogTextContainsAnyV0(text,
		"entrada", "entradas", "bloque", "bloques", "section", "entry")
	if hasBacklogScan && hasIdentity && (hasOrder || hasEntry) {
		return "backlog-scan-entry-identity-order"
	}
	hasTask := idleSelfImprovementBacklogTextContainsAnyV0(text, "tarea", "tareas", "task")
	if hasTask && idleSelfImprovementBacklogTextContainsAnyV0(text, "task id", "txx", "reserva") {
		return "backlog-task-id-allocation"
	}
	if hasTask && idleSelfImprovementBacklogTextContainsAnyV0(text, "solape", "overlap", "canonical", "fusion") {
		return "backlog-task-overlap-canonical"
	}
	if hasTask && idleSelfImprovementBacklogTextContainsAnyV0(text, "propuesta", "proposal", "dedup") {
		return "backlog-proposal-deduplication"
	}
	if strings.Contains(text, "federad") && idleSelfImprovementBacklogTextContainsAnyV0(text, "epoch", "freshness") {
		return "federated-backlog-epoch-scope"
	}
	return ""
}

func idleSelfImprovementNormalizeOverlapTokensV0(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if normalized := idleSelfImprovementNormalizeOverlapTextV0(value); normalized != "" {
			out = append(out, normalized)
		}
	}
	sort.Strings(out)
	return compactServerStackStringsV0(out)
}

func idleSelfImprovementNormalizeOverlapTextV0(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.NewReplacer("_", " ", "-", " ", "`", "", ".", " ", ",", " ", ";", " ").Replace(value)
	return strings.Join(strings.Fields(value), " ")
}
