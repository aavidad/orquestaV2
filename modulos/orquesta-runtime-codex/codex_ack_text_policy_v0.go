package orquestaruntimecodex

import (
	"strings"

	orquestarails "orquesta/modulos/orquesta-rails"
)

const CodexAgentAckPendingRailEvidenceRefV0 = "ack-pending-rail:operational-detail-marker"

func codexAckContainsTrimmedV0(values []string, want string) bool {
	want = strings.TrimSpace(want)
	if want == "" {
		return true
	}
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}

func codexAckContainsNotePrefixV0(values []string, want string) bool {
	want = strings.ToLower(strings.TrimSpace(want))
	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == want ||
			strings.HasPrefix(normalized, want+":") ||
			strings.HasPrefix(normalized, want+" ") {
			return true
		}
	}
	return false
}

func codexAckBytesContainForbiddenDetailV0(data []byte) bool {
	text := string(data)
	return codexTextContainsOperationalDetailMarkerV0(text) ||
		codexTextHasExplicitSoftRailMarkerV0(text) ||
		codexTextHasPendingRailMarkerV0(text)
}

func codexTextContainsOperationalDetailMarkerV0(value string) bool {
	lower := strings.ToLower(strings.ReplaceAll(value, `\/`, "/"))
	for _, marker := range orquestarails.OperationalDetailMarkersV0 {
		if strings.Contains(lower, strings.ToLower(marker)) {
			return true
		}
	}
	return false
}

func codexTextContainsSensitiveDetailV0(value string) bool {
	return codexTextContainsSensitiveDetailIgnoringRailEnvV0(value)
}

func codexAckContainsSensitiveDetailV0(ack CodexAgentAckV0) bool {
	return codexAckValuesContainSensitiveDetailV0([]string{
		ack.RequestID,
		ack.CorrelationID,
		ack.AckRef,
		ack.TargetModule,
		ack.TaskRef,
		ack.Status,
	}) ||
		codexAckValuesContainSensitiveDetailV0(ack.Files) ||
		codexAckValuesContainSensitiveDetailV0(ack.Tests) ||
		codexAckNotesContainSensitiveDetailV0(ack.Notes)
}

func codexAckValuesContainSensitiveDetailV0(values []string) bool {
	for _, value := range values {
		if codexAckTextContainsEffectiveSensitiveDetailV0(value) {
			return true
		}
	}
	return false
}

func CodexAgentAckHasPendingRailV0(ack CodexAgentAckV0) bool {
	return codexAckValuesHaveFieldPendingRailV0(ack.Files) ||
		codexAckValuesHaveFieldPendingRailV0(ack.Tests) ||
		codexAckValuesHavePendingRailV0(ack.Notes)
}

func codexAckValuesHavePendingRailV0(values []string) bool {
	for _, value := range values {
		if codexAckValueHasPendingRailV0(value) {
			return true
		}
	}
	return false
}

func codexAckValueHasPendingRailV0(value string) bool {
	if codexAckTextContainsEffectiveSensitiveDetailV0(value) {
		return false
	}
	return codexTextContainsOperationalDetailMarkerV0(value) ||
		codexTextHasExplicitSoftRailMarkerV0(value) ||
		codexTextHasLegacyPendingMarkerV0(value)
}

func codexAckValuesHaveFieldPendingRailV0(values []string) bool {
	for _, value := range values {
		if codexAckFieldValueHasPendingRailV0(value) {
			return true
		}
	}
	return false
}

func codexAckFieldValueHasPendingRailV0(value string) bool {
	if codexAckTextContainsEffectiveSensitiveDetailV0(value) {
		return false
	}
	return codexTextContainsOperationalDetailMarkerV0(value) ||
		codexTextHasExplicitSoftRailMarkerV0(value) ||
		codexTextHasPendingRailMarkerV0(value)
}

func codexAckNotesContainSensitiveDetailV0(notes []string) bool {
	for _, note := range notes {
		if codexAckNoteContainsSensitiveDetailV0(note) {
			return true
		}
	}
	return false
}

func codexAckNoteContainsSensitiveDetailV0(note string) bool {
	return codexAckTextContainsEffectiveSensitiveDetailV0(note)
}

func codexAckTextContainsEffectiveSensitiveDetailV0(value string) bool {
	if !codexTextContainsSensitiveDetailV0(value) {
		return false
	}
	return codexAckNoteHasEffectiveSensitiveValueV0(value)
}

func codexAckNoteHasEffectiveSensitiveValueV0(note string) bool {
	lower := strings.ToLower(strings.ReplaceAll(note, `\/`, "/"))
	for _, fragment := range orquestarails.OperationalSensitiveFragmentsV0 {
		marker := strings.ToLower(strings.TrimSpace(fragment))
		if marker == "" || !strings.Contains(lower, marker) {
			continue
		}
		if marker == "-----begin" {
			return true
		}
		if codexAckSensitiveMarkerHasEffectiveValueV0(lower, marker) {
			return true
		}
	}
	return false
}

func codexAckSensitiveMarkerHasEffectiveValueV0(value string, marker string) bool {
	start := 0
	for {
		index := strings.Index(value[start:], marker)
		if index < 0 {
			return false
		}
		absolute := start + index
		if codexAckSensitiveMarkerBoundaryV0(value, marker, absolute) &&
			codexAckSensitiveTailHasEffectiveValueV0(value[absolute+len(marker):]) {
			return true
		}
		start = absolute + len(marker)
	}
}

func codexAckSensitiveMarkerBoundaryV0(value string, marker string, start int) bool {
	if marker == "" {
		return true
	}
	before := true
	if isCodexAckASCIIAlnumV0(marker[0]) {
		before = start == 0 || !isCodexAckASCIIAlnumV0(value[start-1])
	}
	end := start + len(marker)
	after := true
	if isCodexAckASCIIAlnumV0(marker[len(marker)-1]) {
		after = end >= len(value) || !isCodexAckASCIIAlnumV0(value[end])
	}
	return before && after
}

func codexAckSensitiveTailHasEffectiveValueV0(tail string) bool {
	value := codexAckSensitiveTailValueV0(tail)
	if strings.HasPrefix(value, "bearer ") {
		value = codexAckSensitiveTailValueV0(strings.TrimPrefix(value, "bearer "))
	}
	return value != "" && !codexAckSensitiveValueIsSoftRailV0(value)
}

func codexAckSensitiveTailValueV0(tail string) string {
	value := strings.TrimLeft(strings.TrimSpace(tail), `"'[]{}()<>`)
	for _, separator := range []string{"\n", "\r", ",", ";", `"`, `'`, "`", "]", "}", ")"} {
		if index := strings.Index(value, separator); index >= 0 {
			value = value[:index]
		}
	}
	return strings.TrimSpace(strings.Trim(value, `"'[]{}()<>`))
}

func codexAckSensitiveValueIsSoftRailV0(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || strings.Contains(value, "sin valor") || strings.Contains(value, "no value") {
		return true
	}
	for _, prefix := range []string{
		"redacted", "redactado", "masked", "oculto", "hidden",
		"omitted", "omitido", "placeholder", "policy", "politica",
		"none", "null", "n/a", "***", "xxx",
	} {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func isCodexAckASCIIAlnumV0(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
}

func codexTextHasLegacyPendingMarkerV0(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	for _, fragment := range []string{
		"db",
		"database",
		"sql",
		"dsn",
		"runtime",
		"provider",
		"proveedor",
		"model",
		"modelo",
		"home",
		"oauth", "authorization",
		"codex",
		"claude",
		"ollama",
		"vllm",
		"adapter",
		"adaptador",
		"filesystem",
		"git",
		"docker",
		"tmux",
		"prompt",
		"completion",
		"transcript",
		"secret",
		"secreto",
		"token",
		"password",
		"credential",
		"credencial",
		"api_key", "private key",
	} {
		if codexDeliveryObservationContainsFragmentV0(lower, fragment) {
			return true
		}
	}
	return false
}

func codexTextHasPendingRailMarkerV0(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	for _, fragment := range []string{
		"provider",
		"proveedor",
		"model",
		"modelo",
		"home",
		"oauth", "authorization",
		"prompt",
		"completion",
		"transcript",
		"secret",
		"secreto",
		"token",
		"password",
		"credential",
		"credencial",
		"api_key", "private key",
	} {
		if codexDeliveryObservationContainsFragmentV0(lower, fragment) {
			return true
		}
	}
	return false
}
