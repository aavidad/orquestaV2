package orquestaobservability

func cloneWorkspaceTimelineProjectionV0(projection WorkspaceTimelineProjectionV0) WorkspaceTimelineProjectionV0 {
	clone := projection
	clone.Sources = append([]WorkspaceTimelineSourceStatusV0(nil), projection.Sources...)
	for index := range clone.Sources {
		clone.Sources[index].EvidenceRefs = append([]string(nil), projection.Sources[index].EvidenceRefs...)
	}
	clone.Items = append([]WorkspaceTimelineItemV0(nil), projection.Items...)
	for index := range clone.Items {
		clone.Items[index].EvidenceRefs = append([]string(nil), projection.Items[index].EvidenceRefs...)
	}
	if projection.Counters != nil {
		clone.Counters = make(map[string]float64, len(projection.Counters))
		for key, value := range projection.Counters {
			clone.Counters[key] = value
		}
	}
	clone.Warnings = append([]DiagnosticoWarningV0(nil), projection.Warnings...)
	return clone
}
