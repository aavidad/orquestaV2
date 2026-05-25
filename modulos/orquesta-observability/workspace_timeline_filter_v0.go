package orquestaobservability

import "strings"

func filterWorkspaceTimelineProjectionForQueryV0(
	projection WorkspaceTimelineProjectionV0,
	query WorkspaceTimelineQueryV0,
) WorkspaceTimelineProjectionV0 {
	filtered := cloneWorkspaceTimelineProjectionV0(projection)
	sources := workspaceTimelineQuerySourcesV0(query)
	filtered.Sources = filterWorkspaceTimelineSourceStatusesBySourceV0(filtered.Sources, sources)
	filtered.Items = filterWorkspaceTimelineItemsBySourceV0(filtered.Items, sources)
	limit := workspaceTimelineQueryLimitV0(query)
	if limit > 0 && len(filtered.Items) > limit {
		filtered.Items = filtered.Items[:limit]
	}
	return filtered
}

func filterWorkspaceTimelineSourceStatusesBySourceV0(
	items []WorkspaceTimelineSourceStatusV0,
	sources []string,
) []WorkspaceTimelineSourceStatusV0 {
	allowed := workspaceTimelineSourceSetV0(sources)
	out := make([]WorkspaceTimelineSourceStatusV0, 0, len(items))
	for _, item := range items {
		if allowed[strings.TrimSpace(item.Source)] {
			out = append(out, item)
		}
	}
	return out
}

func filterWorkspaceTimelineItemsBySourceV0(
	items []WorkspaceTimelineItemV0,
	sources []string,
) []WorkspaceTimelineItemV0 {
	allowed := workspaceTimelineSourceSetV0(sources)
	out := make([]WorkspaceTimelineItemV0, 0, len(items))
	for _, item := range items {
		if allowed[strings.TrimSpace(item.Source)] {
			out = append(out, item)
		}
	}
	return out
}

func workspaceTimelineSourceSetV0(sources []string) map[string]bool {
	allowed := map[string]bool{}
	for _, source := range sources {
		allowed[strings.TrimSpace(source)] = true
	}
	return allowed
}
