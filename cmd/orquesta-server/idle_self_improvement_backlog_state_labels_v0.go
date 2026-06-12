package main

import "strings"

func idleSelfImprovementSectionStateValueV0(lines []string) string {
	for _, label := range []string{"Estado", "Cierre local"} {
		if value := idleSelfImprovementSectionLooseLabelValueV0(lines, label); value != "" {
			return strings.ToLower(value)
		}
	}
	return ""
}

func idleSelfImprovementSectionLooseLabelValueV0(lines []string, label string) string {
	normalizedLabel := strings.ToLower(strings.TrimSpace(label))
	if normalizedLabel == "" {
		return ""
	}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		before, after, ok := strings.Cut(trimmed, ":")
		if !ok {
			continue
		}
		before = strings.ToLower(strings.TrimSpace(before))
		if before == normalizedLabel || strings.HasPrefix(before, normalizedLabel+" ") {
			return strings.TrimSpace(after)
		}
	}
	return ""
}
