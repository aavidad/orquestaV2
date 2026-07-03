package main

import "strings"

func idleSelfImprovementSectionTestsV0(lines []string) ([]string, []string) {
	values := idleSelfImprovementSectionTestValuesV0(lines)
	auto := []string{}
	manual := []string{}
	for _, value := range values {
		executable, blockers := idleSelfImprovementSplitTestValueV0(value)
		auto = append(auto, executable...)
		manual = append(manual, blockers...)
	}
	return compactServerStackStringsV0(auto), compactServerStackStringsV0(manual)
}

func idleSelfImprovementSectionTestValuesV0(lines []string) []string {
	values := []string{}
	inBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if label, index, ok := idleSelfImprovementSectionTestLabelInLineV0(trimmed); ok {
			value := strings.TrimSpace(trimmed[index+len(label):])
			if value != "" {
				values = append(values, value)
			}
			inBlock = value == ""
			continue
		}
		if inBlock && strings.HasSuffix(trimmed, ":") && !strings.HasPrefix(trimmed, "- ") {
			break
		}
		if inBlock && strings.HasPrefix(trimmed, "- ") {
			values = append(values, strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
		}
	}
	return values
}

func idleSelfImprovementSectionTestLabelInLineV0(line string) (string, int, bool) {
	selectedLabel := ""
	selectedIndex := -1
	for _, label := range []string{"Tests:", "Validacion:", "Validación:"} {
		index := strings.Index(line, label)
		if index < 0 {
			continue
		}
		if selectedIndex < 0 || index < selectedIndex {
			selectedLabel = label
			selectedIndex = index
		}
	}
	if selectedIndex < 0 {
		return "", -1, false
	}
	return selectedLabel, selectedIndex, true
}

func idleSelfImprovementSplitTestValueV0(value string) ([]string, []string) {
	if strings.Contains(value, "`") {
		return idleSelfImprovementSplitBacktickTestValueV0(value)
	}
	return idleSelfImprovementClassifyTestPartsV0(
		idleSelfImprovementSplitPlainTestValueV0(value),
	)
}

func idleSelfImprovementSplitBacktickTestValueV0(value string) ([]string, []string) {
	auto := []string{}
	manual := []string{}
	for {
		before, rest, ok := strings.Cut(value, "`")
		manual = append(manual, idleSelfImprovementManualTestPartsV0(before)...)
		if !ok {
			break
		}
		code, after, ok := strings.Cut(rest, "`")
		if !ok {
			manual = append(manual, idleSelfImprovementManualTestPartsV0(rest)...)
			break
		}
		part := idleSelfImprovementCleanTestPartV0(code)
		if idleSelfImprovementExecutableTestCommandV0(part) {
			auto = append(auto, part)
		} else if part != "" {
			manual = append(manual, part)
		}
		value = after
	}
	return compactServerStackStringsV0(auto), compactServerStackStringsV0(manual)
}

func idleSelfImprovementClassifyTestPartsV0(parts []string) ([]string, []string) {
	auto := []string{}
	manual := []string{}
	for _, part := range parts {
		part = idleSelfImprovementCleanTestPartV0(part)
		if part == "" {
			continue
		}
		if idleSelfImprovementExecutableTestCommandV0(part) {
			auto = append(auto, part)
			continue
		}
		manual = append(manual, part)
	}
	return compactServerStackStringsV0(auto), compactServerStackStringsV0(manual)
}

func idleSelfImprovementManualTestPartsV0(value string) []string {
	parts := idleSelfImprovementSplitPlainTestValueV0(value)
	out := []string{}
	for _, part := range parts {
		part = idleSelfImprovementCleanTestPartV0(part)
		if part != "" && !idleSelfImprovementIsConnectorTextV0(part) {
			out = append(out, part)
		}
	}
	return out
}

func idleSelfImprovementSplitPlainTestValueV0(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, " y ")
	return append([]string(nil), parts...)
}

func idleSelfImprovementCleanTestPartV0(value string) string {
	value = strings.TrimSpace(strings.Trim(strings.TrimSpace(value), "`"))
	value = strings.TrimPrefix(value, "- ")
	value = strings.TrimSpace(value)
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "y ") || strings.HasPrefix(lower, "e ") {
		value = strings.TrimSpace(value[2:])
	}
	for len(value) > 1 &&
		strings.ContainsRune(".,;:", rune(value[len(value)-1])) &&
		!strings.HasSuffix(value, "...") {
		value = strings.TrimSpace(value[:len(value)-1])
	}
	return value
}

func idleSelfImprovementExecutableTestCommandV0(value string) bool {
	value = strings.TrimSpace(value)
	return strings.HasPrefix(value, "go test ") ||
		value == "git diff --check" ||
		strings.HasPrefix(value, "bash ") ||
		strings.HasPrefix(value, "sh ") ||
		strings.HasPrefix(value, "./scripts/") ||
		strings.HasPrefix(value, "scripts/") ||
		strings.HasPrefix(value, "make ")
}

func idleSelfImprovementIsConnectorTextV0(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	return value == "" || value == "y" || value == "e"
}
