package main

import "strings"

func idleSelfImprovementBacklogWriteSetEntriesV0(value string) []string {
	value = strings.TrimSpace(value)
	original := value
	var out []string
	for {
		_, rest, ok := strings.Cut(value, "`")
		if !ok {
			break
		}
		entry, next, ok := strings.Cut(rest, "`")
		if !ok {
			break
		}
		out = append(out, idleSelfImprovementCleanWriteSetEntryV0(entry))
		value = next
	}
	if len(out) > 0 {
		return compactServerStackStringsV0(out)
	}
	if entry := idleSelfImprovementCleanWriteSetEntryV0(original); entry != "" {
		return []string{entry}
	}
	return nil
}

func idleSelfImprovementCleanWriteSetEntryV0(value string) string {
	value = strings.TrimSpace(strings.Trim(strings.TrimSpace(value), "`"))
	for len(value) > 1 && strings.ContainsRune(".,;:", rune(value[len(value)-1])) {
		value = strings.TrimSpace(value[:len(value)-1])
	}
	return value
}
