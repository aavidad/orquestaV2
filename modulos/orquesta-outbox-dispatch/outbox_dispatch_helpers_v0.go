package orquestaoutboxdispatch

import "strings"

func cloneEntriesV0(entries []OutboxPendingEntryV0) []OutboxPendingEntryV0 {
	if len(entries) == 0 {
		return nil
	}
	cloned := make([]OutboxPendingEntryV0, 0, len(entries))
	for _, entry := range entries {
		cloned = append(cloned, normalizeEntryV0(entry))
	}
	return cloned
}

func compactStringsV0(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	compact := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			compact = append(compact, trimmed)
		}
	}
	return compact
}

func cloneBytesV0(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}
	cloned := make([]byte, len(data))
	copy(cloned, data)
	return cloned
}
