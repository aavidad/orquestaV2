package orquestaruntimecodex

import (
	"fmt"
	"strings"
	"testing"
)

func TestCodexAckPendingRailExternalMatrixV0(t *testing.T) {
	spec := codexSpecForTestV0()
	terms := []string{
		"access_token", "refresh_token", "oauth", "token", "secret",
		"secreto", "begin private key", "transcript", "prompt=", "completion=",
		"prompt", "completion",
		"/home/", "/users/", "$HOME", "~/", "provider=", "model=",
	}

	for index := 0; index < 240; index++ {
		term := terms[index%len(terms)]
		note := fmt.Sprintf("rail pendiente %03d %s texto operativo", index, term)
		data := []byte(codexAckJSONWithNoteForPendingRailTestV0(note))
		t.Run(fmt.Sprintf("pending-rail-%03d", index), func(t *testing.T) {
			_, issues := ValidateCodexAgentAckBytesForSpecV0(data, spec)
			if len(issues) != 0 {
				t.Fatalf("rail pendiente no debe bloquear issues=%+v note=%q", issues, note)
			}
			if !codexAckBytesContainForbiddenDetailV0(data) {
				t.Fatalf("rail pendiente no detecto note=%q", note)
			}
		})
	}
}

func TestCodexAckPendingRailExternalMatrixV0ConRailsDetalleOff(t *testing.T) {
	t.Setenv("ORQUESTA_DETAIL_PROHIBITED_RAILS", "off")
	spec := codexSpecForTestV0()
	data := []byte(codexAckJSONWithNoteForPendingRailTestV0(
		"rail pendiente provider=policy token texto operativo",
	))

	ack, issues := ValidateCodexAgentAckBytesForSpecV0(data, spec)

	if len(issues) != 0 {
		t.Fatalf("rail pendiente no debe bloquear issues=%+v", issues)
	}
	if !CodexAgentAckHasPendingRailV0(ack) ||
		!codexAckBytesContainForbiddenDetailV0(data) {
		t.Fatalf("rail pendiente no conservado ack=%+v", ack)
	}
}

func TestCodexAckPendingRailExternalMatrixV0ConRailsDetalleOffCortaSecretoEfectivo(t *testing.T) {
	t.Setenv("ORQUESTA_DETAIL_PROHIBITED_RAILS", "off")
	spec := codexSpecForTestV0()
	data := []byte(codexAckJSONWithNoteForPendingRailTestV0("access_token=abc123"))

	_, issues := ValidateCodexAgentAckBytesForSpecV0(data, spec)

	requireCodexIssueV0(t, issues, CodexConnectorAckForbiddenV0)
}

func TestCodexAckPendingRailExternalMatrixV0CamposNoCortanConValoresBlandos(t *testing.T) {
	spec := codexSpecForTestV0()
	cases := []string{
		`"files":["README.md","docs/secret:policy.md"],"tests":["go test ./..."]`,
		`"files":["README.md"],"tests":["go test ./...","rail access_token=redacted sin valor"]`,
		`"files":["README.md"],"tests":["go test ./...","rail access_token=REDACTED sin valor"]`,
		`"files":["README.md"],"tests":["go test ./..."],"notes":["authorization: bearer redacted"]`,
		`"files":["README.md"],"tests":["go test ./..."],"notes":["authorization: bearer REDACTED"]`,
	}
	for _, fragment := range cases {
		data := []byte(codexAckJSONWithFragmentForPendingRailTestV0(fragment))
		_, issues := ValidateCodexAgentAckBytesForSpecV0(data, spec)
		if len(issues) != 0 {
			t.Fatalf("rail blando en campo ACK no debe bloquear issues=%+v fragment=%s", issues, fragment)
		}
	}
}

func TestCodexAckPendingRailExternalMatrixV0CamposCortanConValoresEfectivos(t *testing.T) {
	spec := codexSpecForTestV0()
	cases := []string{
		`"files":["README.md","docs/access_token=abc123.md"],"tests":["go test ./..."]`,
		`"files":["README.md"],"tests":["go test ./...","access_token=abc123"]`,
	}
	for _, fragment := range cases {
		data := []byte(codexAckJSONWithFragmentForPendingRailTestV0(fragment))
		_, issues := ValidateCodexAgentAckBytesForSpecV0(data, spec)
		requireCodexIssueV0(t, issues, CodexConnectorAckForbiddenV0)
	}
}

func codexAckJSONWithNoteForPendingRailTestV0(note string) string {
	note = strings.ReplaceAll(note, `\`, `\\`)
	note = strings.ReplaceAll(note, `"`, `\"`)
	return `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed","files":["README.md"],"tests":["go test ./..."],"notes":["` + note + `"]}`
}

func codexAckJSONWithFragmentForPendingRailTestV0(fragment string) string {
	return `{"schema_version":"codex_agent_ack.v0","request_id":"req-codex-001","correlation_id":"corr-codex-001","ack_ref":"ack-ref-001","target_module":"orquesta-generated-app","task_ref":"task-ref-001","status":"completed",` + fragment + `}`
}
