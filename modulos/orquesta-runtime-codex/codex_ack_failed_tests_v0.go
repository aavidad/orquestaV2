package orquestaruntimecodex

import (
	"encoding/json"
	"strings"
)

func codexAckHasFailedTestEvidenceV0(ack CodexAgentAckV0) bool {
	return codexAckEvidenceStringsHaveFailureV0(ack.Tests, true) ||
		codexRequiredTestReceiptsHaveFailureV0(ack.TestReceipts)
}

func CodexAgentAckDeclaresIncompleteRequiredEvidenceV0(ack CodexAgentAckV0) bool {
	return codexAckEvidenceStringsDeclareIncompleteV0(ack.Tests, true) ||
		codexRequiredTestReceiptsDeclareIncompleteV0(ack.TestReceipts) ||
		codexAckEvidenceStringsDeclareIncompleteV0(ack.Notes, false)
}

func codexAckBytesHaveFailedTestEvidenceV0(data []byte) bool {
	var raw struct {
		Tests        json.RawMessage `json:"tests"`
		TestReceipts json.RawMessage `json:"test_receipts"`
		Notes        json.RawMessage `json:"notes"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return false
	}
	return codexAckRawEvidenceHasFailureV0(raw.Tests, true) ||
		codexAckRawEvidenceHasFailureV0(raw.TestReceipts, true)
}

func codexRequiredTestReceiptsHaveFailureV0(receipts []CodexRequiredTestReceiptV0) bool {
	for _, receipt := range receipts {
		if strings.TrimSpace(receipt.Status) != "" &&
			strings.TrimSpace(receipt.Status) != "passed" {
			return true
		}
		if receipt.ExitCode != nil && *receipt.ExitCode != 0 {
			return true
		}
	}
	return false
}

func codexRequiredTestReceiptsDeclareIncompleteV0(receipts []CodexRequiredTestReceiptV0) bool {
	for _, receipt := range receipts {
		if codexAckTextDeclaresIncompleteRequiredEvidenceV0(receipt.Command, true) ||
			codexAckTextDeclaresIncompleteRequiredEvidenceV0(receipt.Status, true) {
			return true
		}
		for _, ref := range receipt.EvidenceRefs {
			if codexAckTextDeclaresIncompleteRequiredEvidenceV0(ref, true) {
				return true
			}
		}
	}
	return false
}

func codexAckEvidenceStringsHaveFailureV0(values []string, inTests bool) bool {
	for _, value := range values {
		if codexAckTextDeclaresFailedTestV0(value, inTests) {
			return true
		}
	}
	return false
}

func codexAckEvidenceStringsDeclareIncompleteV0(values []string, inTests bool) bool {
	for _, value := range values {
		if codexAckTextDeclaresIncompleteRequiredEvidenceV0(value, inTests) {
			return true
		}
	}
	return false
}

func codexAckRawEvidenceHasFailureV0(raw json.RawMessage, inTests bool) bool {
	if len(raw) == 0 {
		return false
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return codexAckTextDeclaresFailedTestV0(string(raw), inTests)
	}
	return codexAckValueDeclaresFailedTestV0(value, inTests)
}

func codexAckValueDeclaresFailedTestV0(value any, inTests bool) bool {
	switch typed := value.(type) {
	case string:
		return codexAckTextDeclaresFailedTestV0(typed, inTests)
	case []any:
		for _, item := range typed {
			if codexAckValueDeclaresFailedTestV0(item, inTests) {
				return true
			}
		}
	case map[string]any:
		for key, item := range typed {
			key = strings.ToLower(strings.TrimSpace(key))
			nextInTests := inTests || strings.Contains(key, "test")
			if nextInTests && codexAckFailureFieldV0(key, item) {
				return true
			}
			if nextInTests && codexAckValueDeclaresFailedTestV0(item, nextInTests) {
				return true
			}
		}
	}
	return false
}

func codexAckFailureFieldV0(key string, value any) bool {
	if strings.Contains(key, "status") ||
		strings.Contains(key, "result") ||
		strings.Contains(key, "outcome") ||
		strings.Contains(key, "conclusion") {
		text, ok := value.(string)
		return ok && codexAckFailureWordV0(text)
	}
	if strings.Contains(key, "passed") ||
		strings.Contains(key, "success") ||
		key == "ok" {
		flag, ok := value.(bool)
		return ok && !flag
	}
	if key == "exit_code" || key == "exit_status" {
		return codexAckNonZeroNumberV0(value)
	}
	return strings.Contains(key, "fail") && codexAckPositiveNumberV0(value)
}

func codexAckTextDeclaresFailedTestV0(text string, inTests bool) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return false
	}
	if strings.Contains(lower, "--- fail:") || strings.Contains(lower, "fail\t") {
		return true
	}
	compact := codexAckCompactEvidenceTextV0(lower)
	if strings.Contains(compact, "no tests failed") ||
		strings.Contains(compact, "no test failed") {
		return false
	}
	if inTests && codexAckFailurePhraseV0(compact) {
		return true
	}
	return strings.Contains(compact, "test") && codexAckFailurePhraseV0(compact)
}

func codexAckTextDeclaresIncompleteRequiredEvidenceV0(text string, inTests bool) bool {
	compact := codexAckCompactEvidenceTextV0(strings.ToLower(strings.TrimSpace(text)))
	if compact == "" {
		return false
	}
	if strings.Contains(compact, "no tests failed") ||
		strings.Contains(compact, "no test failed") {
		return false
	}
	mentionsTestOrEvidence := inTests ||
		strings.Contains(compact, "test") ||
		strings.Contains(compact, "smoke") ||
		strings.Contains(compact, "evidencia") ||
		strings.Contains(compact, "evidence") ||
		strings.Contains(compact, "entorno") ||
		strings.Contains(compact, "env") ||
		strings.Contains(compact, "confirmacion") ||
		strings.Contains(compact, "confirmación")
	if !mentionsTestOrEvidence {
		return false
	}
	if codexAckMissingEvidencePhraseV0(compact) {
		return true
	}
	for _, phrase := range []string{
		"no ejecutado",
		"no ejecutada",
		"no se ejecuto",
		"no se ejecutó",
		"sin ejecutar",
		"not executed",
		"not run",
		"was not run",
		"skipped",
		"omitido",
		"omitida",
		"falta entorno",
		"falta env",
		"missing env",
		"falta confirmacion",
		"falta confirmación",
		"missing confirmation",
	} {
		if strings.Contains(compact, phrase) {
			return true
		}
	}
	return false
}

func codexAckMissingEvidencePhraseV0(text string) bool {
	if strings.Contains(text, "no falta") ||
		strings.Contains(text, "no faltan") ||
		strings.Contains(text, "sin faltantes") ||
		strings.Contains(text, "faltantes resueltos") ||
		strings.Contains(text, "resuelto") ||
		strings.Contains(text, "resuelta") ||
		strings.Contains(text, "resolved") {
		return false
	}
	for _, token := range []string{
		"falta ",
		"faltan ",
		"missing ",
		"lacks ",
		"lack ",
	} {
		if strings.Contains(text, token) {
			return true
		}
	}
	return false
}

func codexAckFailurePhraseV0(text string) bool {
	if strings.Contains(text, "test") && codexAckFailureTokenV0(text) {
		return true
	}
	for _, phrase := range []string{
		"status failed",
		"result failed",
		"outcome failed",
		"conclusion failed",
		"passed false",
		"success false",
		"ok false",
		"exit status 1",
		"exit code 1",
		"non zero",
		"nonzero",
		"test failed",
		"tests failed",
		"failing test",
	} {
		if strings.Contains(text, phrase) {
			return true
		}
	}
	return false
}

func codexAckFailureTokenV0(text string) bool {
	fields := strings.Fields(text)
	for index, field := range fields {
		field = strings.Trim(field, ".()[]{}")
		switch field {
		case "failed", "fail", "failure", "failing", "error", "errored", "nonzero":
			return true
		}
		if field == "non" && index+1 < len(fields) && strings.Trim(fields[index+1], ".()[]{}") == "zero" {
			return true
		}
		if field == "exit" && index+2 < len(fields) {
			next := strings.Trim(fields[index+1], ".()[]{}")
			code := strings.Trim(fields[index+2], ".()[]{}")
			if (next == "status" || next == "code") && code == "1" {
				return true
			}
		}
	}
	return false
}

func codexAckFailureWordV0(text string) bool {
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "fail", "failed", "failure", "failing", "error", "errored":
		return true
	default:
		return false
	}
}

func codexAckNonZeroNumberV0(value any) bool {
	if number, ok := value.(float64); ok {
		return number != 0
	}
	return false
}

func codexAckPositiveNumberV0(value any) bool {
	if number, ok := value.(float64); ok {
		return number > 0
	}
	return false
}

func codexAckCompactEvidenceTextV0(text string) string {
	replacer := strings.NewReplacer(":", " ", "=", " ", `"`, " ", "'", " ", ",", " ", ";", " ")
	return strings.Join(strings.Fields(replacer.Replace(text)), " ")
}
