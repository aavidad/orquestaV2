package orquestaruntimecodex

import "strings"

var codexAckLocalProductForbiddenFragmentsV0 = []string{
	"/home/",
	"\\home\\",
	"/users/",
	"\\users\\",
	"c:\\users\\",
	"$home",
	"~/",
	".orquesta-runtime",
	".orquesta-server",
	".orquesta-smoke-work",
	".orquesta-codex-runtime",
	".orquesta-local-runtime",
	".orquesta-control",
	".orquesta-runs",
	".orquesta-worktrees",
	".orquesta-logs",
	"agent_ack.json",
	"agent_packet.json",
	"agent_prompt.txt",
	"agent_shutdown_checkpoint_ack.json",
	"director_decisions.json",
	"orquesta_shutdown_request.json",
	"codex_stdout.log",
	"codex_stderr.log",
	"codex_last_message.txt",
	"orquesta.env",
	"orquesta.db",
	".orquesta-inbox.md",
	"server.log",
	"logs/",
	".ssl-key.log",
	"transcript=",
	"raw_transcript=",
}

func codexAckContainsLocalProductDetailV0(ack CodexAgentAckV0) bool {
	return codexAckValuesContainLocalProductDetailV0([]string{
		ack.RequestID,
		ack.CorrelationID,
		ack.AckRef,
		ack.TargetModule,
		ack.TaskRef,
		ack.Status,
	}) ||
		codexAckValuesContainLocalProductDetailV0(ack.Files) ||
		codexAckValuesContainLocalProductDetailV0(ack.Tests) ||
		codexAckValuesContainLocalProductDetailV0(codexRequiredTestReceiptSensitiveValuesV0(ack.TestReceipts)) ||
		codexAckNotesContainLocalProductDetailV0(ack.Notes)
}

func codexAckValuesContainLocalProductDetailV0(values []string) bool {
	for _, value := range values {
		if codexAckTextContainsLocalProductDetailV0(value) {
			return true
		}
	}
	return false
}

func codexAckTextContainsLocalProductDetailV0(value string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(value, `\/`, "/"))
	if strings.TrimSpace(normalized) == "" {
		return false
	}
	for _, fragment := range codexAckLocalProductForbiddenFragmentsV0 {
		if strings.Contains(normalized, fragment) {
			return true
		}
	}
	return false
}

func codexAckNotesContainLocalProductDetailV0(notes []string) bool {
	for _, note := range notes {
		if codexAckNoteClassifiesPendingRailV0(note) {
			continue
		}
		if codexAckNoteIsAllowedRefOnlyEvidenceV0(note) {
			continue
		}
		if codexAckTextContainsLocalProductDetailV0(note) {
			return true
		}
	}
	return false
}

func codexAckNoteClassifiesPendingRailV0(note string) bool {
	normalized := strings.ToLower(strings.TrimSpace(note))
	return strings.Contains(normalized, "rail pendiente") ||
		strings.Contains(normalized, "pending rail") ||
		strings.Contains(normalized, CodexAgentAckPendingRailEvidenceRefV0)
}

func codexAckNoteIsAllowedRefOnlyEvidenceV0(note string) bool {
	normalized := strings.ToLower(strings.TrimSpace(note))
	if !strings.HasPrefix(normalized, "contexto_ref_only_resuelto:") &&
		!strings.HasPrefix(normalized, "contexto_ref_only_resuelto ") {
		return false
	}
	if codexAckTextContainsEffectiveSensitiveDetailV0(note) {
		return false
	}
	return strings.Contains(normalized, "ack_evidence_required") ||
		strings.Contains(normalized, "evidencia explicita") ||
		strings.Contains(normalized, "evidence")
}
