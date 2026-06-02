package orquestadomainwork

import (
	"encoding/json"
	"strconv"
	"strings"
)

func documentPlanSectionsFromRawV0(raw json.RawMessage) []DomainDocumentPlanSectionV0 {
	var values []map[string]json.RawMessage
	if len(raw) == 0 || json.Unmarshal(raw, &values) != nil {
		return nil
	}
	out := make([]DomainDocumentPlanSectionV0, 0, len(values))
	usedRefs := map[string]struct{}{}
	for index, value := range values {
		plannedPages := documentPlanIntV0(value, "planned_pages")
		sectionRef := firstDocumentPlanStringV0(value, "section_ref", "section_id", "id")
		section := DomainDocumentPlanSectionV0{
			SectionRef:         sectionRef,
			ParentRef:          firstDocumentPlanStringV0(value, "parent_ref"),
			Order:              firstPositiveDocumentPlanIntV0(index+1, documentPlanIntV0(value, "order"), documentPlanIntV0(value, "ordinal")),
			Title:              firstDocumentPlanStringV0(value, "title", "planned_title"),
			Objective:          firstDocumentPlanTextV0(value, "objective", "description", "planned_work", "production_action", "official_topic_ref"),
			WorkKind:           documentPlanSectionWorkKindV0(firstDocumentPlanStringV0(value, "work_kind")),
			DependsOn:          firstDocumentPlanRefsV0(value, "depends_on"),
			TargetWordsMin:     firstPositiveDocumentPlanIntV0(plannedPages*300, documentPlanIntV0(value, "target_words_min")),
			TargetWordsMax:     firstPositiveDocumentPlanIntV0(plannedPages*450, documentPlanIntV0(value, "target_words_max")),
			RequiredElements:   firstDocumentPlanStringsV0(value, "required_elements", "required_points"),
			AcceptanceCriteria: firstDocumentPlanStringsV0(value, "acceptance_criteria"),
			SourceRefs:         documentPlanRefsFromRawKeysV0(value, "source_refs", "primary_source_refs", "official_topic_ref"),
		}
		if sectionRef == "" {
			section.SectionRef = nextGeneratedDocumentPlanRefV0(usedRefs, "section", section.Title)
		} else {
			reserveDocumentPlanRefV0(usedRefs, section.SectionRef)
		}
		out = append(out, section)
	}
	return out
}

func documentPlanVisualsFromRawV0(raw json.RawMessage) []DomainDocumentPlanVisualV0 {
	var values []map[string]json.RawMessage
	if len(raw) == 0 || json.Unmarshal(raw, &values) != nil {
		return nil
	}
	out := make([]DomainDocumentPlanVisualV0, 0, len(values))
	usedRefs := map[string]struct{}{}
	for _, value := range values {
		visualRef := firstDocumentPlanStringV0(value, "visual_ref", "visual_id", "id")
		visual := DomainDocumentPlanVisualV0{
			VisualRef:          visualRef,
			VisualType:         compactDocumentPlanRefTextV0(firstDocumentPlanStringV0(value, "visual_type", "asset_type")),
			PlacementRef:       firstDocumentPlanStringV0(value, "placement_ref", "target_section_id"),
			Objective:          firstDocumentPlanStringV0(value, "objective", "brief", "title"),
			WorkKind:           documentPlanVisualWorkKindV0(firstDocumentPlanStringV0(value, "work_kind")),
			AcceptanceCriteria: firstDocumentPlanStringsV0(value, "acceptance_criteria"),
			SourceRefs:         firstDocumentPlanRefsV0(value, "source_refs"),
		}
		if visualRef == "" {
			visual.VisualRef = nextGeneratedDocumentPlanRefV0(usedRefs, "visual", visual.Objective)
		} else {
			reserveDocumentPlanRefV0(usedRefs, visual.VisualRef)
		}
		out = append(out, visual)
	}
	return out
}

func documentPlanReviewsFromRawV0(raw json.RawMessage) []DomainDocumentPlanReviewV0 {
	var values []map[string]json.RawMessage
	if len(raw) == 0 || json.Unmarshal(raw, &values) != nil {
		return nil
	}
	out := make([]DomainDocumentPlanReviewV0, 0, len(values))
	usedRefs := map[string]struct{}{}
	for index, value := range values {
		reviewRef := firstDocumentPlanStringV0(value, "review_ref", "review_id", "step_id", "id")
		review := DomainDocumentPlanReviewV0{
			ReviewRef:          reviewRef,
			Order:              firstPositiveDocumentPlanIntV0(index+1, documentPlanIntV0(value, "order")),
			WorkKind:           documentPlanReviewWorkKindV0(firstDocumentPlanStringV0(value, "work_kind")),
			Objective:          firstDocumentPlanTextV0(value, "objective", "scope", "title", "description"),
			AcceptanceCriteria: firstDocumentPlanStringsV0(value, "acceptance_criteria"),
		}
		if reviewRef == "" {
			review.ReviewRef = nextGeneratedDocumentPlanRefV0(usedRefs, "review", review.Objective)
		} else {
			reserveDocumentPlanRefV0(usedRefs, review.ReviewRef)
		}
		out = append(out, review)
	}
	return out
}

func firstDocumentPlanTextV0(raw map[string]json.RawMessage, keys ...string) string {
	for _, key := range keys {
		if value := documentPlanStringV0(raw[key]); value != "" {
			return value
		}
		if values := documentPlanStringsV0(raw[key]); len(values) > 0 {
			return strings.Join(values, " ")
		}
	}
	return ""
}

func documentPlanRefsFromRawKeysV0(raw map[string]json.RawMessage, keys ...string) []string {
	values := []string{}
	for _, key := range keys {
		values = append(values, documentPlanStringsV0(raw[key])...)
	}
	return compactDocumentPlanRefsV0(values)
}

func documentPlanDeliverablesFromRawV0(raw json.RawMessage) []DomainDocumentPlanDeliverableV0 {
	var values []map[string]json.RawMessage
	if len(raw) == 0 || json.Unmarshal(raw, &values) != nil {
		return nil
	}
	out := make([]DomainDocumentPlanDeliverableV0, 0, len(values))
	usedRefs := map[string]struct{}{}
	for _, value := range values {
		title := firstDocumentPlanStringV0(value, "title", "name", "description")
		artifactType := firstDocumentPlanStringV0(value, "artifact_type", "name")
		deliverableRef := firstDocumentPlanStringV0(value, "deliverable_ref", "deliverable_id", "id")
		deliverable := DomainDocumentPlanDeliverableV0{
			DeliverableRef: deliverableRef,
			ArtifactType:   compactDocumentPlanRefTextV0(artifactType),
			Title:          title,
			Required:       true,
		}
		if deliverableRef == "" {
			deliverable.DeliverableRef = nextGeneratedDocumentPlanRefV0(usedRefs, "deliverable", title)
		} else {
			reserveDocumentPlanRefV0(usedRefs, deliverable.DeliverableRef)
		}
		out = append(out, deliverable)
	}
	return out
}

func nextGeneratedDocumentPlanRefV0(
	used map[string]struct{},
	prefix string,
	source string,
) string {
	base := prefixedDocumentPlanRefV0(prefix, source)
	if base == "" {
		base = compactDocumentPlanRefTextV0(prefix)
	}
	for suffix := 1; ; suffix++ {
		candidate := base
		if suffix > 1 {
			candidate = base + "-" + strconv.Itoa(suffix)
		}
		if _, ok := used[candidate]; ok {
			continue
		}
		used[candidate] = struct{}{}
		return candidate
	}
}

func reserveDocumentPlanRefV0(used map[string]struct{}, ref string) {
	if ref != "" {
		used[ref] = struct{}{}
	}
}
