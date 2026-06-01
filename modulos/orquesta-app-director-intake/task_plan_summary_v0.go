package orquestaappdirectorintake

import "strings"

const (
	maxDirectorTaskSummaryBytesV0      = 560
	maxDirectorTaskSummaryValueBytesV0 = 180
)

func directorTaskSummaryFieldV0(label string, value string) string {
	value = truncateDirectorTaskSummaryTextV0(
		compactDirectorTaskSummaryTextV0(value),
		maxDirectorTaskSummaryValueBytesV0,
	)
	if label == "" || value == "" {
		return ""
	}
	return label + "=" + value
}

func directorTaskSummaryListFieldV0(label string, values []string) string {
	values = compactDirectorTaskSummaryPartsV0(values)
	if label == "" || len(values) == 0 {
		return ""
	}
	return label + "=" + strings.Join(values, ",")
}

func compactDirectorTaskSummaryTextV0(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func compactDirectorTaskSummaryPartsV0(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = compactDirectorTaskSummaryTextV0(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func joinDirectorTaskSummaryWithinBudgetV0(values []string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	out := ""
	for _, value := range values {
		value = compactDirectorTaskSummaryTextV0(value)
		if value == "" {
			continue
		}
		candidate := value
		if out != "" {
			candidate = out + " " + value
		}
		if len(candidate) <= maxBytes {
			out = candidate
			continue
		}
		remaining := maxBytes - len(out)
		if out != "" {
			remaining--
		}
		if remaining > 24 {
			tail := truncateDirectorTaskSummaryTextV0(value, remaining)
			if tail != "" {
				if out != "" {
					out += " "
				}
				out += tail
			}
		}
		break
	}
	return strings.TrimSpace(out)
}

func truncateDirectorTaskSummaryTextV0(value string, maxBytes int) string {
	value = compactDirectorTaskSummaryTextV0(value)
	if maxBytes <= 0 || value == "" {
		return ""
	}
	if len(value) <= maxBytes {
		return value
	}
	if maxBytes <= 3 {
		return truncateDirectorTaskSummaryTextBytesV0(value, maxBytes)
	}
	limit := maxBytes - 3
	prefix := truncateDirectorTaskSummaryTextBytesV0(value, limit)
	if cut := strings.LastIndex(prefix, " "); cut >= limit/2 {
		prefix = strings.TrimSpace(prefix[:cut])
	}
	prefix = strings.TrimRight(strings.TrimSpace(prefix), ".,;:")
	if prefix == "" {
		prefix = truncateDirectorTaskSummaryTextBytesV0(value, limit)
	}
	return prefix + "..."
}

func truncateDirectorTaskSummaryTextBytesV0(value string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	var builder strings.Builder
	for _, r := range value {
		if builder.Len()+len(string(r)) > maxBytes {
			break
		}
		builder.WriteRune(r)
	}
	return builder.String()
}
