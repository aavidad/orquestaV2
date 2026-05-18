package orquestadomainwork

import "encoding/json"

func documentPlanSectionsFromRawV0(raw json.RawMessage) []DomainDocumentPlanSectionV0 {
	var values []map[string]json.RawMessage
	if len(raw) == 0 || json.Unmarshal(raw, &values) != nil {
		return nil
	}
	out := make([]DomainDocumentPlanSectionV0, 0, len(values))
	for index, value := range values {
		plannedPages := documentPlanIntV0(value, "planned_pages")
		section := DomainDocumentPlanSectionV0{
			SectionRef:         firstDocumentPlanStringV0(value, "section_ref", "section_id", "id"),
			ParentRef:          firstDocumentPlanStringV0(value, "parent_ref"),
			Order:              firstPositiveDocumentPlanIntV0(index+1, documentPlanIntV0(value, "order")),
			Title:              firstDocumentPlanStringV0(value, "title"),
			Objective:          firstDocumentPlanStringV0(value, "objective", "description"),
			WorkKind:           documentPlanSectionWorkKindV0(firstDocumentPlanStringV0(value, "work_kind")),
			DependsOn:          firstDocumentPlanRefsV0(value, "depends_on"),
			TargetWordsMin:     firstPositiveDocumentPlanIntV0(plannedPages*300, documentPlanIntV0(value, "target_words_min")),
			TargetWordsMax:     firstPositiveDocumentPlanIntV0(plannedPages*450, documentPlanIntV0(value, "target_words_max")),
			RequiredElements:   firstDocumentPlanStringsV0(value, "required_elements", "required_points"),
			AcceptanceCriteria: firstDocumentPlanStringsV0(value, "acceptance_criteria"),
			SourceRefs:         firstDocumentPlanRefsV0(value, "source_refs"),
		}
		if section.SectionRef == "" {
			section.SectionRef = prefixedDocumentPlanRefV0("section", section.Title)
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
	for _, value := range values {
		visual := DomainDocumentPlanVisualV0{
			VisualRef:          firstDocumentPlanStringV0(value, "visual_ref", "visual_id", "id"),
			VisualType:         compactDocumentPlanRefTextV0(firstDocumentPlanStringV0(value, "visual_type", "asset_type")),
			PlacementRef:       firstDocumentPlanStringV0(value, "placement_ref", "target_section_id"),
			Objective:          firstDocumentPlanStringV0(value, "objective", "brief", "title"),
			WorkKind:           documentPlanVisualWorkKindV0(firstDocumentPlanStringV0(value, "work_kind")),
			AcceptanceCriteria: firstDocumentPlanStringsV0(value, "acceptance_criteria"),
			SourceRefs:         firstDocumentPlanRefsV0(value, "source_refs"),
		}
		if visual.VisualRef == "" {
			visual.VisualRef = prefixedDocumentPlanRefV0("visual", visual.Objective)
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
	for index, value := range values {
		review := DomainDocumentPlanReviewV0{
			ReviewRef:          firstDocumentPlanStringV0(value, "review_ref", "review_id", "id"),
			Order:              firstPositiveDocumentPlanIntV0(index+1, documentPlanIntV0(value, "order")),
			WorkKind:           documentPlanReviewWorkKindV0(firstDocumentPlanStringV0(value, "work_kind")),
			Objective:          firstDocumentPlanStringV0(value, "objective", "scope", "title"),
			AcceptanceCriteria: firstDocumentPlanStringsV0(value, "acceptance_criteria"),
		}
		if review.ReviewRef == "" {
			review.ReviewRef = prefixedDocumentPlanRefV0("review", review.Objective)
		}
		out = append(out, review)
	}
	return out
}

func documentPlanDeliverablesFromRawV0(raw json.RawMessage) []DomainDocumentPlanDeliverableV0 {
	var values []map[string]json.RawMessage
	if len(raw) == 0 || json.Unmarshal(raw, &values) != nil {
		return nil
	}
	out := make([]DomainDocumentPlanDeliverableV0, 0, len(values))
	for _, value := range values {
		title := firstDocumentPlanStringV0(value, "title", "name", "description")
		artifactType := firstDocumentPlanStringV0(value, "artifact_type", "name")
		deliverable := DomainDocumentPlanDeliverableV0{
			DeliverableRef: firstDocumentPlanStringV0(value, "deliverable_ref", "deliverable_id", "id"),
			ArtifactType:   compactDocumentPlanRefTextV0(artifactType),
			Title:          title,
			Required:       true,
		}
		if deliverable.DeliverableRef == "" {
			deliverable.DeliverableRef = prefixedDocumentPlanRefV0("deliverable", title)
		}
		out = append(out, deliverable)
	}
	return out
}
