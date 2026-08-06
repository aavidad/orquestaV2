package codexwork

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"unicode/utf8"
)

const (
	WorkPacketSchemaV1 = "orquesta.codex-work.v1"
	MaxPacketBytesV1   = 1 << 20
	MaxOutputBytesV1   = 1 << 20
	MaxTokenBudgetV1   = 1_000_000_000
	MaxTimeBudgetMSV1  = 24 * 60 * 60 * 1000

	// ControlledEgressProxyURLV1 is the only proxy endpoint the sealed guest
	// executor may project. It is a protocol fact backed by the signed launch
	// plan, never caller-provided configuration or ambient environment.
	ControlledEgressProxyURLV1 = "http://127.0.0.1:18080"
)

type Code string

const (
	CodePacketTooLarge  Code = "codexwork.packet_too_large"
	CodePacketMalformed Code = "codexwork.packet_malformed"
	CodePacketSchema    Code = "codexwork.packet_schema_invalid"
	CodePacketField     Code = "codexwork.packet_field_invalid"
	CodeFrameTooLarge   Code = "codexwork.frame_too_large"
	CodeFrameMalformed  Code = "codexwork.frame_malformed"
	CodeSequence        Code = "codexwork.sequence_invalid"
	CodeDuplicateFrame  Code = "codexwork.frame_duplicate"
	CodeUnknownID       Code = "codexwork.response_id_unknown"
	CodeMethod          Code = "codexwork.method_not_allowed"
	CodeRemote          Code = "codexwork.remote_error"
	CodeThreadMismatch  Code = "codexwork.thread_mismatch"
	CodeTurnMismatch    Code = "codexwork.turn_mismatch"
	CodeOutputMissing   Code = "codexwork.output_missing"
	CodeOutputInvalid   Code = "codexwork.output_invalid"
	CodeTurnFailed      Code = "codexwork.turn_failed"
	CodeTurnInterrupted Code = "codexwork.turn_interrupted"
)

// Error conserva solo un código estable. Nunca incorpora payload remoto.
type Error struct{ Code Code }

func (err *Error) Error() string {
	if err == nil {
		return ""
	}
	return string(err.Code)
}

func (err *Error) Is(target error) bool {
	var other *Error
	return errors.As(target, &other) && err != nil && err.Code == other.Code
}

func ErrorCode(err error) Code {
	var protocolErr *Error
	if errors.As(err, &protocolErr) {
		return protocolErr.Code
	}
	return ""
}

func protocolError(code Code) error { return &Error{Code: code} }

// WorkPacketV1 es la única entrada del ejecutor alojado. Sus referencias son
// opacas: este paquete comprueba forma, no interpreta prefijos de Orquesta.
type WorkPacketV1 struct {
	Schema           string `json:"schema"`
	Prompt           string `json:"prompt"`
	Model            string `json:"model"`
	Effort           string `json:"effort"`
	TokenBudget      uint64 `json:"token_budget"`
	MaxOutputBytes   uint64 `json:"max_output_bytes"`
	TimeBudgetMS     uint64 `json:"time_budget_ms"`
	GoalRef          string `json:"goal_ref"`
	WorkItemRef      string `json:"work_item_ref"`
	ExecutionRef     string `json:"execution_ref"`
	EffectAttemptRef string `json:"effect_attempt_ref"`
	// ControlledEgressProxy is omitted unless the matching signed launch plan
	// contains both the proxy service and its explicit egress grant.
	ControlledEgressProxy string `json:"controlled_egress_proxy,omitempty"`
}

func DecodeWorkPacketV1(raw []byte) (WorkPacketV1, error) {
	if len(raw) > MaxPacketBytesV1 {
		return WorkPacketV1{}, protocolError(CodePacketTooLarge)
	}
	if len(raw) == 0 || !utf8.Valid(raw) {
		return WorkPacketV1{}, protocolError(CodePacketMalformed)
	}
	if !exactWorkPacketJSONObject(raw) {
		return WorkPacketV1{}, protocolError(CodePacketMalformed)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var packet WorkPacketV1
	if err := decoder.Decode(&packet); err != nil {
		return WorkPacketV1{}, protocolError(CodePacketMalformed)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return WorkPacketV1{}, protocolError(CodePacketMalformed)
	}
	if err := packet.Validate(); err != nil {
		return WorkPacketV1{}, err
	}
	return packet, nil
}

func uniqueJSONObject(raw []byte) bool {
	return strictRootJSONObject(raw, nil)
}

var workPacketJSONFields = map[string]bool{
	"schema": true, "prompt": true, "model": true, "effort": true,
	"token_budget": true, "max_output_bytes": true, "time_budget_ms": true,
	"goal_ref": true, "work_item_ref": true, "execution_ref": true,
	"effect_attempt_ref": true, "controlled_egress_proxy": false,
}

func exactWorkPacketJSONObject(raw []byte) bool {
	return strictRootJSONObject(raw, workPacketJSONFields)
}

func strictRootJSONObject(raw []byte, fields map[string]bool) bool {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('{') {
		return false
	}
	seen := make(map[string]struct{})
	for decoder.More() {
		token, err := decoder.Token()
		key, ok := token.(string)
		if err != nil || !ok {
			return false
		}
		if fields != nil {
			if _, allowed := fields[key]; !allowed {
				return false
			}
		}
		if _, duplicate := seen[key]; duplicate {
			return false
		}
		seen[key] = struct{}{}
		if !uniqueJSONValue(decoder) {
			return false
		}
	}
	closing, err := decoder.Token()
	if err != nil || closing != json.Delim('}') {
		return false
	}
	for field, required := range fields {
		if _, exists := seen[field]; required && !exists {
			return false
		}
	}
	_, err = decoder.Token()
	return errors.Is(err, io.EOF)
}

func uniqueJSONObjectBody(decoder *json.Decoder) bool {
	seen := make(map[string]struct{})
	for decoder.More() {
		token, err := decoder.Token()
		key, ok := token.(string)
		if err != nil || !ok {
			return false
		}
		if _, duplicate := seen[key]; duplicate {
			return false
		}
		seen[key] = struct{}{}
		if !uniqueJSONValue(decoder) {
			return false
		}
	}
	token, err := decoder.Token()
	return err == nil && token == json.Delim('}')
}

func uniqueJSONValue(decoder *json.Decoder) bool {
	token, err := decoder.Token()
	if err != nil {
		return false
	}
	delimiter, composite := token.(json.Delim)
	if !composite {
		return true
	}
	switch delimiter {
	case '{':
		return uniqueJSONObjectBody(decoder)
	case '[':
		for decoder.More() {
			if !uniqueJSONValue(decoder) {
				return false
			}
		}
		closing, err := decoder.Token()
		return err == nil && closing == json.Delim(']')
	default:
		return false
	}
}

func (packet WorkPacketV1) Validate() error {
	if packet.Schema != WorkPacketSchemaV1 {
		return protocolError(CodePacketSchema)
	}
	if packet.Prompt == "" || !utf8.ValidString(packet.Prompt) ||
		!validToken(packet.Model, 128) || !validEffort(packet.Effort) ||
		packet.TokenBudget == 0 || packet.TokenBudget > MaxTokenBudgetV1 ||
		packet.MaxOutputBytes == 0 || packet.MaxOutputBytes > MaxOutputBytesV1 ||
		packet.TimeBudgetMS == 0 || packet.TimeBudgetMS > MaxTimeBudgetMSV1 ||
		!validOpaqueRef(packet.GoalRef) || !validOpaqueRef(packet.WorkItemRef) ||
		!validOpaqueRef(packet.ExecutionRef) || !validOpaqueRef(packet.EffectAttemptRef) {
		return protocolError(CodePacketField)
	}
	if packet.ControlledEgressProxy != "" &&
		packet.ControlledEgressProxy != ControlledEgressProxyURLV1 {
		return protocolError(CodePacketField)
	}
	return nil
}

func validEffort(value string) bool {
	switch value {
	case "low", "medium", "high", "xhigh", "ultra":
		return true
	default:
		return false
	}
}

func validOpaqueRef(value string) bool {
	return validToken(value, 512)
}

func validToken(value string, limit int) bool {
	if value == "" || len(value) > limit || value != strings.TrimSpace(value) || !utf8.ValidString(value) {
		return false
	}
	for _, char := range value {
		if char < 0x20 || char == 0x7f {
			return false
		}
	}
	return true
}
