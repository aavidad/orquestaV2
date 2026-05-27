package main

import (
	"strings"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func idleSelfImprovementBacklogSectionCollisionsV0(
	sections []idleSelfImprovementBacklogSectionV0,
	excluded map[string]bool,
) []orquestaserver.BacklogScanCollisionV0 {
	var collisions []orquestaserver.BacklogScanCollisionV0
	seenSections := map[string]idleSelfImprovementBacklogSectionV0{}
	for _, section := range sections {
		ref := idleSelfImprovementNormalizeDependencyRefV0(section.Ref)
		if prior, ok := seenSections[ref]; ok {
			collisions = append(collisions, orquestaserver.BacklogScanCollisionV0{
				Code:       "duplicate_backlog_heading",
				SectionRef: section.Ref,
				Message: "fingerprints=" + idleSelfImprovementBacklogSectionFingerprintV0(prior) +
					"," + idleSelfImprovementBacklogSectionFingerprintV0(section),
				EvidenceRefs: []string{
					"evidence-ref-autoprogramming-backlog-heading-collision",
				},
			})
			continue
		}
		seenSections[ref] = section
	}
	for ref := range excluded {
		if strings.HasPrefix(ref, "request-ref-autoprogramming-backlog-scanner-") {
			collisions = append(collisions, orquestaserver.BacklogScanCollisionV0{
				Code:       "backlog_scanner_already_visible",
				RequestRef: ref,
				Message:    "scanner documental ya visible en cola o ACK runtime",
				EvidenceRefs: []string{
					"evidence-ref-autoprogramming-backlog-scanner-visible",
				},
			})
		}
	}
	return compactBacklogScanCollisionsV0(collisions)
}

func compactBacklogScanCollisionsV0(
	values []orquestaserver.BacklogScanCollisionV0,
) []orquestaserver.BacklogScanCollisionV0 {
	out := make([]orquestaserver.BacklogScanCollisionV0, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value.Code = strings.TrimSpace(value.Code)
		value.RequestRef = strings.TrimSpace(value.RequestRef)
		value.SectionRef = strings.TrimSpace(value.SectionRef)
		value.TaskID = strings.TrimSpace(value.TaskID)
		value.InstanceRefs = compactServerStackStringsV0(value.InstanceRefs)
		value.Message = strings.TrimSpace(value.Message)
		value.EvidenceRefs = compactServerStackStringsV0(value.EvidenceRefs)
		key := strings.Join([]string{
			value.Code,
			value.RequestRef,
			value.SectionRef,
			value.TaskID,
			strings.Join(value.InstanceRefs, ","),
			value.Message,
		}, "|")
		if value.Code == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, value)
	}
	return out
}

func idleSelfImprovementBacklogSectionFingerprintV0(
	section idleSelfImprovementBacklogSectionV0,
) string {
	parts := []string{
		strings.TrimSpace(section.Heading),
		strings.TrimSpace(section.Objective),
		strings.TrimSpace(section.SourcePath),
		strings.TrimSpace(section.LocalAlias),
		strings.TrimSpace(section.LocalEntryHash),
		strings.Join(compactServerStackStringsV0(section.Scope), ","),
		strings.Join(compactServerStackStringsV0(section.StateEvidenceRefs), ","),
	}
	return strings.Join(parts, "|")
}
