package orquestaopesdirector

import (
	"strconv"
	"strings"
)

func opesQuestionBankCorrectAnswerKeysV0(question OPESQuestionBankQuestionV0) ([]string, []string) {
	keys := map[string]bool{}
	var unresolved []string
	for index, option := range question.Options {
		if option.Correct {
			keys[opesQuestionBankOptionKeyV0(index)] = true
		}
	}
	for _, ref := range question.CorrectAnswerRefs {
		key := opesQuestionBankCorrectRefKeyV0(ref, question.Options)
		if key == "" {
			unresolved = append(unresolved, ref)
			continue
		}
		keys[key] = true
	}
	out := make([]string, 0, len(keys))
	for key := range keys {
		out = append(out, key)
	}
	return compactStringsV0(out), compactStringsV0(unresolved)
}

func opesQuestionBankCorrectRefKeyV0(ref string, options []OPESQuestionBankOptionV0) string {
	normalized := strings.ToLower(strings.TrimSpace(ref))
	if normalized == "" {
		return ""
	}
	switch normalized {
	case "a", "opcion a", "opción a", "option a":
		return "A"
	case "b", "opcion b", "opción b", "option b":
		return "B"
	case "c", "opcion c", "opción c", "option c":
		return "C"
	case "d", "opcion d", "opción d", "option d":
		return "D"
	}
	if n, err := strconv.Atoi(normalized); err == nil {
		switch n {
		case 0, 1:
			return "A"
		case 2:
			return "B"
		case 3:
			return "C"
		case 4:
			return "D"
		}
	}
	for index, option := range options {
		if normalized == strings.ToLower(strings.TrimSpace(option.Ref)) ||
			normalized == strings.ToLower(strings.TrimSpace(option.Text)) {
			return opesQuestionBankOptionKeyV0(index)
		}
	}
	return ""
}

func opesQuestionBankOptionKeyV0(index int) string {
	if index >= 0 && index < 26 {
		return string(rune('A' + index))
	}
	return strconv.Itoa(index + 1)
}

func opesQuestionBankHasBlankOrDuplicateOptionsV0(options []OPESQuestionBankOptionV0) bool {
	seen := map[string]bool{}
	for _, option := range options {
		text := strings.ToLower(strings.TrimSpace(option.Text))
		if text == "" || seen[text] {
			return true
		}
		seen[text] = true
	}
	return false
}

func opesQuestionBankCompactOptionsV0(options []OPESQuestionBankOptionV0) []OPESQuestionBankOptionV0 {
	out := make([]OPESQuestionBankOptionV0, 0, len(options))
	for _, option := range options {
		option.Ref = strings.TrimSpace(option.Ref)
		option.Text = strings.TrimSpace(option.Text)
		if option.Ref == "" && option.Text == "" && !option.Correct {
			continue
		}
		out = append(out, option)
	}
	return out
}

func opesQuestionBankReportPassesV0(status string, evidenceRefs []string, acceptedRefs ...string) bool {
	if opesTopicQualityReportStatusPassesV0(status) {
		return true
	}
	return topicRegistryEvidenceContainsAnyV0(evidenceRefs, acceptedRefs...)
}

func opesQuestionBankIssueV0(code string, field string, message string) OPESQuestionBankQualityIssueV0 {
	return OPESQuestionBankQualityIssueV0{Code: code, Field: field, Message: message}
}
