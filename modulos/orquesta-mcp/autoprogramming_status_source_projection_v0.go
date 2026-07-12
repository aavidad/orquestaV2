package orquestamcp

import "strings"

const mcpAutoprogrammingStatusSampleRefsLimitV0 = 3

func projectMCPAutoprogrammingStatusListsV0(
	result MCPAutoprogrammingStatusToolResultV0,
) MCPAutoprogrammingStatusToolResultV0 {
	result.StaleRunningTotal = len(result.StaleRunning)
	result.DiagnosticsTotal = len(result.Diagnostics)
	result.StaleRunning = aggregateMCPAutoprogrammingActionableRunsByCodeV0(result.StaleRunning)
	result.Diagnostics = aggregateMCPAutoprogrammingDiagnosticsByCodeV0(result.Diagnostics)
	result.StaleRunning = limitMCPAutoprogrammingStatusSourceItemsV0(
		result.StaleRunning,
		mcpAutoprogrammingStatusDefaultListLimitV0,
	)
	return result
}

func aggregateMCPAutoprogrammingActionableRunsByCodeV0(
	items []MCPAutoprogrammingActionableRunV0,
) []MCPAutoprogrammingActionableRunV0 {
	if len(items) == 0 {
		return nil
	}
	out := make([]MCPAutoprogrammingActionableRunV0, 0, len(items))
	indexByCode := make(map[string]int, len(items))
	for _, item := range items {
		code := strings.TrimSpace(item.Code)
		index, exists := indexByCode[code]
		if !exists {
			item.Code = code
			out = append(out, item)
			indexByCode[code] = len(out) - 1
			continue
		}
		aggregate := &out[index]
		if aggregate.Count == 0 {
			aggregate.Count = 1
		}
		aggregate.Count++
		aggregate.SampleRefs = mcpAutoprogrammingStatusSampleRefsV0(
			aggregate.RunRef,
			append(append(aggregate.SampleRefs, item.RunRef), item.SampleRefs...)...,
		)
		aggregate.EvidenceRefs = limitMCPAutoprogrammingStatusSourceStringsV0(
			append(aggregate.EvidenceRefs, item.EvidenceRefs...),
			mcpAutoprogrammingStatusSampleRefsLimitV0,
		)
		aggregate.RunRef = ""
		aggregate.AppRef = ""
		aggregate.GoalRef = ""
		aggregate.ExternalGoalRef = ""
	}
	return out
}

func aggregateMCPAutoprogrammingDiagnosticsByCodeV0(
	items []MCPAutoprogrammingDiagnosticV0,
) []MCPAutoprogrammingDiagnosticV0 {
	if len(items) == 0 {
		return nil
	}
	out := make([]MCPAutoprogrammingDiagnosticV0, 0, len(items))
	indexByCode := make(map[string]int, len(items))
	for _, item := range items {
		code := strings.TrimSpace(item.Code)
		index, exists := indexByCode[code]
		if !exists {
			item.Code = code
			out = append(out, item)
			indexByCode[code] = len(out) - 1
			continue
		}
		aggregate := &out[index]
		if aggregate.Count == 0 {
			aggregate.Count = 1
		}
		aggregate.Count++
		sampleRefs := append([]string(nil), aggregate.SampleRefs...)
		if len(sampleRefs) == 0 {
			sampleRefs = append(sampleRefs, aggregate.Scope)
		}
		aggregate.SampleRefs = limitMCPAutoprogrammingStatusSourceStringsV0(
			append(append(sampleRefs, item.Scope), item.SampleRefs...),
			mcpAutoprogrammingStatusSampleRefsLimitV0,
		)
		aggregate.EvidenceRefs = limitMCPAutoprogrammingStatusSourceStringsV0(
			append(aggregate.EvidenceRefs, item.EvidenceRefs...),
			mcpAutoprogrammingStatusSampleRefsLimitV0,
		)
		aggregate.Scope = "aggregate:" + code
	}
	return out
}

func mcpAutoprogrammingStatusSampleRefsV0(primary string, refs ...string) []string {
	return limitMCPAutoprogrammingStatusSourceStringsV0(
		append([]string{strings.TrimSpace(primary)}, refs...),
		mcpAutoprogrammingStatusSampleRefsLimitV0,
	)
}

func limitMCPAutoprogrammingStatusSourceStringsV0(items []string, limit int) []string {
	return limitMCPAutoprogrammingStatusSourceItemsV0(compactStringsMCPV0(items), limit)
}

func limitMCPAutoprogrammingStatusSourceItemsV0[T any](items []T, limit int) []T {
	if len(items) == 0 || limit <= 0 {
		return nil
	}
	if len(items) < limit {
		limit = len(items)
	}
	return append([]T(nil), items[:limit]...)
}
