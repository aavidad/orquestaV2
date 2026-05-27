package main

import (
	"strings"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func idleSelfImprovementCanonicalizeBacklogSectionsV0(
	sections []idleSelfImprovementBacklogSectionV0,
	excluded map[string]bool,
) ([]idleSelfImprovementBacklogSectionV0, []orquestaserver.BacklogScanCollisionV0, map[string]bool) {
	indexByRef := map[string]int{}
	for index, section := range sections {
		indexByRef[idleSelfImprovementCanonicalSectionRefV0(section.Ref)] = index
	}
	collisions := []orquestaserver.BacklogScanCollisionV0{}
	blockedCanonical := map[string]bool{}
	skip := map[int]bool{}
	for index, section := range sections {
		canonicalRef := idleSelfImprovementCanonicalSectionRefV0(section.CanonicalRef)
		if canonicalRef == "" || canonicalRef == idleSelfImprovementCanonicalSectionRefV0(section.Ref) {
			continue
		}
		canonicalIndex, ok := indexByRef[canonicalRef]
		if !ok {
			collisions = append(collisions, idleSelfImprovementDuplicateBacklogCollisionV0(
				"duplicate_backlog_task_target_missing",
				section,
				canonicalRef,
			))
			continue
		}
		sections[canonicalIndex] = idleSelfImprovementMergeCanonicalBacklogSectionV0(
			sections[canonicalIndex],
			section,
		)
		skip[index] = true
		if idleSelfImprovementDuplicateBacklogKnownV0(section, excluded) {
			blockedCanonical[canonicalRef] = true
		}
		collisions = append(collisions, idleSelfImprovementDuplicateBacklogCollisionV0(
			"duplicate_backlog_task",
			section,
			canonicalRef,
		))
	}
	out := make([]idleSelfImprovementBacklogSectionV0, 0, len(sections))
	for index, section := range sections {
		if skip[index] {
			continue
		}
		out = append(out, section)
	}
	overlapSections, overlapCollisions, overlapBlocked := idleSelfImprovementCanonicalizeOverlappingBacklogSectionsV0(out, excluded)
	for ref := range overlapBlocked {
		blockedCanonical[ref] = true
	}
	collisions = append(collisions, overlapCollisions...)
	out = overlapSections
	return out, compactBacklogScanCollisionsV0(collisions), blockedCanonical
}

func idleSelfImprovementMergeCanonicalBacklogSectionV0(
	canonical idleSelfImprovementBacklogSectionV0,
	duplicate idleSelfImprovementBacklogSectionV0,
) idleSelfImprovementBacklogSectionV0 {
	canonical.Scope = compactServerStackStringsV0(append(canonical.Scope, duplicate.Scope...))
	canonical.Criteria = compactServerStackStringsV0(append(canonical.Criteria, duplicate.Criteria...))
	canonical.Tests = compactServerStackStringsV0(append(canonical.Tests, duplicate.Tests...))
	canonical.ManualVerifications = compactServerStackStringsV0(append(
		canonical.ManualVerifications,
		duplicate.ManualVerifications...,
	))
	canonical.Dependencies = compactServerStackStringsV0(append(canonical.Dependencies, duplicate.Dependencies...))
	canonical.Inputs = compactServerStackStringsV0(append(canonical.Inputs, duplicate.Inputs...))
	canonical.Outputs = compactServerStackStringsV0(append(canonical.Outputs, duplicate.Outputs...))
	canonical.StateEvidenceRefs = compactServerStackStringsV0(append(
		canonical.StateEvidenceRefs,
		"evidence-ref-autoprogramming-backlog-canonical-merge:"+duplicate.Ref,
	))
	return canonical
}

func idleSelfImprovementDuplicateBacklogKnownV0(
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

func idleSelfImprovementDuplicateBacklogCollisionV0(
	code string,
	section idleSelfImprovementBacklogSectionV0,
	canonicalRef string,
) orquestaserver.BacklogScanCollisionV0 {
	return orquestaserver.BacklogScanCollisionV0{
		Code:       code,
		SectionRef: section.Ref,
		Message:    "canonical_ref=" + canonicalRef,
		EvidenceRefs: []string{
			"evidence-ref-autoprogramming-backlog-duplicate-canonical",
		},
	}
}

func idleSelfImprovementSectionCanonicalRefV0(lines []string) string {
	for _, label := range []string{
		"Reemplazada_por:",
		"Fusionada_con:",
		"Fusionada con:",
		"Alias canonico:",
		"Canonica:",
	} {
		if value := idleSelfImprovementSectionValueV0(lines, label); value != "" {
			return idleSelfImprovementCanonicalSectionRefV0(value)
		}
	}
	return ""
}

func idleSelfImprovementCanonicalSectionRefV0(value string) string {
	value = strings.Trim(strings.TrimSpace(value), "`.,; ")
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "request-ref-autoprogramming-backlog-") {
		value = strings.TrimPrefix(value, "request-ref-autoprogramming-backlog-")
		if hashIndex := strings.LastIndex(value, "-"); hashIndex > 0 {
			return value[:hashIndex]
		}
	}
	return idleSelfImprovementHeadingRefV0(value)
}
