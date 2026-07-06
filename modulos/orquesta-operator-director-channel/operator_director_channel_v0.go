package orquestaoperatordirectorchannel

import (
	"context"
	"errors"
	"strings"
)

const (
	OperatorDirectorChannelSchemaVersionV0 = "operator_director_channel.v0"
	OperatorDirectorMessageToolNameV0      = "orquesta.operator.director.message.v0"
	OperatorDirectorMessageResourceURIV0   = "orquesta://operator/director-message/v0"
	OperatorMessageIntentStatusV0          = "status"
	OperatorMessageIntentObserveRunV0      = "observe_run"
	OperatorMessageIntentInstructionV0     = "instruction"
	OperatorMessageIntentGeneralV0         = "general"
	OperatorMessageMaxBodyRunesV0          = 4000
	ErrOperatorMessageRequiredFieldV0      = "operator_message_required_field"
	ErrOperatorMessageOpaqueRefV0          = "operator_message_opaque_ref_invalid"
	ErrOperatorMessageBodyInvalidV0        = "operator_message_body_invalid"
	ErrOperatorMessagePortUnavailableV0    = "operator_message_port_unavailable"
	ErrOperatorMessageDispatchFailedV0     = "operator_message_dispatch_failed"
	ErrOperatorMessageStoreFailedV0        = "operator_message_store_failed"
)

var (
	ErrOperatorDirectorDispatchPortUnavailableV0 = errors.New(ErrOperatorMessagePortUnavailableV0)
	ErrOperatorDirectorStorePortUnavailableV0    = errors.New(ErrOperatorMessageStoreFailedV0)
)

type OperatorMessageV0 struct {
	SchemaVersion   string   `json:"schema_version,omitempty"`
	RequestRef      string   `json:"request_ref"`
	MessageRef      string   `json:"message_ref,omitempty"`
	ConversationRef string   `json:"conversation_ref,omitempty"`
	AdapterRef      string   `json:"adapter_ref,omitempty"`
	SenderRef       string   `json:"sender_ref,omitempty"`
	TargetRef       string   `json:"target_ref"`
	Intent          string   `json:"intent,omitempty"`
	Body            string   `json:"body"`
	EvidenceRefs    []string `json:"evidence_refs,omitempty"`
}

type OperatorDirectorResponseV0 struct {
	SchemaVersion string   `json:"schema_version,omitempty"`
	AckRef        string   `json:"ack_ref"`
	MessageRef    string   `json:"message_ref"`
	Status        string   `json:"status"`
	Summary       string   `json:"summary,omitempty"`
	ResponseRef   string   `json:"response_ref,omitempty"`
	ResponseText  string   `json:"response_text,omitempty"`
	EvidenceRefs  []string `json:"evidence_refs,omitempty"`
}

type OperatorDirectorExchangeV0 struct {
	SchemaVersion string                     `json:"schema_version"`
	Message       OperatorMessageV0          `json:"message"`
	Response      OperatorDirectorResponseV0 `json:"response"`
}

type OperatorMessageIssueV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

type OperatorDirectorDispatchPortV0 interface {
	DispatchOperatorMessageV0(context.Context, OperatorMessageV0) (OperatorDirectorResponseV0, error)
}

type OperatorDirectorExchangeStorePortV0 interface {
	SaveOperatorDirectorExchangeV0(context.Context, OperatorDirectorExchangeV0) error
}

type OperatorDirectorChannelServiceV0 struct {
	Dispatcher OperatorDirectorDispatchPortV0
	Store      OperatorDirectorExchangeStorePortV0
}

func DispatchOperatorDirectorMessageV0(
	ctx context.Context,
	service OperatorDirectorChannelServiceV0,
	input OperatorMessageV0,
) (OperatorDirectorResponseV0, []OperatorMessageIssueV0, error) {
	message := NormalizeOperatorMessageV0(input)
	if issues := ValidateOperatorMessageV0(message); len(issues) > 0 {
		return OperatorDirectorResponseV0{}, issues, nil
	}
	if service.Dispatcher == nil {
		return OperatorDirectorResponseV0{}, nil, ErrOperatorDirectorDispatchPortUnavailableV0
	}
	if service.Store == nil {
		return OperatorDirectorResponseV0{}, nil, ErrOperatorDirectorStorePortUnavailableV0
	}
	response, err := service.Dispatcher.DispatchOperatorMessageV0(ctx, message)
	if err != nil {
		return OperatorDirectorResponseV0{}, nil, err
	}
	response = NormalizeOperatorDirectorResponseV0(response, message)
	exchange := OperatorDirectorExchangeV0{
		SchemaVersion: OperatorDirectorChannelSchemaVersionV0,
		Message:       message,
		Response:      response,
	}
	if err := service.Store.SaveOperatorDirectorExchangeV0(ctx, exchange); err != nil {
		return OperatorDirectorResponseV0{}, nil, err
	}
	return response, nil, nil
}

func NormalizeOperatorMessageV0(input OperatorMessageV0) OperatorMessageV0 {
	input.SchemaVersion = firstNonEmptyChannelV0(input.SchemaVersion, OperatorDirectorChannelSchemaVersionV0)
	input.RequestRef = strings.TrimSpace(input.RequestRef)
	input.MessageRef = strings.TrimSpace(input.MessageRef)
	if input.MessageRef == "" && input.RequestRef != "" {
		input.MessageRef = "operator-message-ref-" + input.RequestRef
	}
	input.ConversationRef = strings.TrimSpace(input.ConversationRef)
	input.AdapterRef = strings.TrimSpace(input.AdapterRef)
	input.SenderRef = strings.TrimSpace(input.SenderRef)
	input.TargetRef = strings.TrimSpace(input.TargetRef)
	input.Intent = normalizeOperatorMessageIntentV0(input.Intent)
	input.Body = strings.TrimSpace(input.Body)
	input.EvidenceRefs = compactStringsChannelV0(input.EvidenceRefs)
	return input
}

func NormalizeOperatorDirectorResponseV0(
	input OperatorDirectorResponseV0,
	message OperatorMessageV0,
) OperatorDirectorResponseV0 {
	input.SchemaVersion = firstNonEmptyChannelV0(input.SchemaVersion, OperatorDirectorChannelSchemaVersionV0)
	input.MessageRef = firstNonEmptyChannelV0(strings.TrimSpace(input.MessageRef), message.MessageRef)
	input.AckRef = firstNonEmptyChannelV0(strings.TrimSpace(input.AckRef), "operator-director-ack-ref-"+input.MessageRef)
	input.Status = firstNonEmptyChannelV0(strings.TrimSpace(input.Status), "accepted")
	input.Summary = strings.TrimSpace(input.Summary)
	input.ResponseRef = strings.TrimSpace(input.ResponseRef)
	input.ResponseText = compactResponseTextV0(input.ResponseText)
	input.EvidenceRefs = compactStringsChannelV0(input.EvidenceRefs)
	return input
}

func ValidateOperatorMessageV0(input OperatorMessageV0) []OperatorMessageIssueV0 {
	var issues []OperatorMessageIssueV0
	for _, item := range []struct {
		field string
		value string
	}{
		{"request_ref", input.RequestRef},
		{"message_ref", input.MessageRef},
		{"target_ref", input.TargetRef},
	} {
		if strings.TrimSpace(item.value) == "" {
			issues = append(issues, OperatorMessageIssueV0{Code: ErrOperatorMessageRequiredFieldV0, Field: item.field})
		} else if !isOpaqueOperatorDirectorRefV0(item.value) {
			issues = append(issues, OperatorMessageIssueV0{Code: ErrOperatorMessageOpaqueRefV0, Field: item.field})
		}
	}
	for _, item := range []struct {
		field string
		value string
	}{
		{"conversation_ref", input.ConversationRef},
		{"adapter_ref", input.AdapterRef},
		{"sender_ref", input.SenderRef},
	} {
		if strings.TrimSpace(item.value) != "" && !isOpaqueOperatorDirectorRefV0(item.value) {
			issues = append(issues, OperatorMessageIssueV0{Code: ErrOperatorMessageOpaqueRefV0, Field: item.field})
		}
	}
	if input.Body == "" || len([]rune(input.Body)) > OperatorMessageMaxBodyRunesV0 {
		issues = append(issues, OperatorMessageIssueV0{Code: ErrOperatorMessageBodyInvalidV0, Field: "body"})
	}
	return issues
}

func normalizeOperatorMessageIntentV0(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	switch value {
	case "status", "estado", "health":
		return OperatorMessageIntentStatusV0
	case "observe-run", "observe", "observa-run", "run-status":
		return OperatorMessageIntentObserveRunV0
	case "instruction", "instruccion", "command":
		return OperatorMessageIntentInstructionV0
	default:
		return OperatorMessageIntentGeneralV0
	}
}

func isOpaqueOperatorDirectorRefV0(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 180 || strings.ContainsAny(value, "\r\n\t /\\") {
		return false
	}
	lower := strings.ToLower(value)
	for _, forbidden := range []string{"/home/", "token", "secret", "password", "oauth", "bearer", "sk-", "://"} {
		if strings.Contains(lower, forbidden) {
			return false
		}
	}
	return true
}

func compactResponseTextV0(value string) string {
	value = strings.TrimSpace(value)
	const maxRunes = 1200
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes])
}

func compactStringsChannelV0(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func firstNonEmptyChannelV0(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
