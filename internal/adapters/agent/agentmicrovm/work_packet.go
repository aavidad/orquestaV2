package agentmicrovm

import (
	"encoding/json"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"

	"orquesta/internal/agentprotocol/codexwork"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

const (
	CodeWorkPacketRequestInvalid      = "agentmicrovm.work_packet_request_invalid"
	CodeWorkPacketCompilationMismatch = "agentmicrovm.work_packet_compilation_mismatch"
	CodeWorkPacketModelInvalid        = "agentmicrovm.work_packet_model_invalid"
	CodeWorkPacketEffortInvalid       = "agentmicrovm.work_packet_effort_invalid"
	CodeWorkPacketBudgetInvalid       = "agentmicrovm.work_packet_budget_invalid"
	CodeWorkPacketOutputInvalid       = "agentmicrovm.work_packet_output_invalid"
	CodeWorkPacketRendererInvalid     = "agentmicrovm.work_packet_renderer_invalid"
	CodeWorkPacketRenderFailed        = "agentmicrovm.work_packet_render_failed"
	CodeWorkPacketPromptInvalid       = "agentmicrovm.work_packet_prompt_invalid"
	CodeWorkPacketEncodeFailed        = "agentmicrovm.work_packet_encode_failed"
	CodeWorkPacketTooLarge            = "agentmicrovm.work_packet_too_large"
	CodeWorkPacketRoundTripInvalid    = "agentmicrovm.work_packet_roundtrip_invalid"
)

// PromptRenderer receives the provider-neutral instruction allowlist. Runtime
// identity, credentials and physical authority never enter the prompt surface.
type PromptRenderer interface {
	RenderAgentPrompt(ports.AgentPrompt) (string, error)
}

// BuildWorkPacketV1 creates the sole sealed stdin frame accepted by the guest
// executor. It performs no I/O and accepts only compilation facts causally
// bound to the same launch request.
func BuildWorkPacketV1(
	request ports.AgentLaunchRequest,
	binding ProfileBinding,
	compiled Compilation,
	model string,
	renderer PromptRenderer,
) (codexwork.WorkPacketV1, []byte, error) {
	tokenBudget, timeBudgetMS, err := validateWorkPacketV1(request, model, renderer)
	if err != nil {
		return codexwork.WorkPacketV1{}, nil, err
	}
	expected, compileErr := Compile(request, binding, request.EffectAuthority.ActionFence)
	if compileErr != nil || !reflect.DeepEqual(compiled, expected) {
		return codexwork.WorkPacketV1{}, nil, fail(CodeWorkPacketCompilationMismatch, nil)
	}
	return encodeWorkPacketV1(
		request, model, renderer, tokenBudget, timeBudgetMS, compiled.Plan.Egreso != nil,
	)
}

func buildProviderWorkPacketV1(
	request ports.AgentLaunchRequest,
	model string,
	renderer PromptRenderer,
) (codexwork.WorkPacketV1, []byte, error) {
	tokenBudget, timeBudgetMS, err := validateWorkPacketV1(request, model, renderer)
	if err != nil {
		return codexwork.WorkPacketV1{}, nil, err
	}
	return encodeWorkPacketV1(request, model, renderer, tokenBudget, timeBudgetMS, false)
}

func validateWorkPacketV1(
	request ports.AgentLaunchRequest,
	model string,
	renderer PromptRenderer,
) (uint64, uint64, error) {
	if !validWorkPacketToken(model, 128) {
		return 0, 0, fail(CodeWorkPacketModelInvalid, nil)
	}
	if nilInterface(renderer) {
		return 0, 0, fail(CodeWorkPacketRendererInvalid, nil)
	}
	if governance.ValidateReasoningEffort(request.ReasoningEffort) != nil {
		return 0, 0, fail(CodeWorkPacketEffortInvalid, nil)
	}
	tokenBudget, timeBudgetMS, err := workPacketBudgets(request)
	if err != nil {
		return 0, 0, err
	}
	if request.MaxOutputBytes <= 0 || uint64(request.MaxOutputBytes) > codexwork.MaxOutputBytesV1 {
		return 0, 0, fail(CodeWorkPacketOutputInvalid, nil)
	}
	if err := ports.ValidateAgentLaunchRequest(request); err != nil {
		return 0, 0, fail(CodeWorkPacketRequestInvalid, nil)
	}
	return tokenBudget, timeBudgetMS, nil
}

func encodeWorkPacketV1(
	request ports.AgentLaunchRequest,
	model string,
	renderer PromptRenderer,
	tokenBudget uint64,
	timeBudgetMS uint64,
	controlledEgress bool,
) (codexwork.WorkPacketV1, []byte, error) {
	prompt, renderErr := renderer.RenderAgentPrompt(ports.AgentPromptFromLaunchRequest(request))
	if renderErr != nil {
		return codexwork.WorkPacketV1{}, nil, fail(CodeWorkPacketRenderFailed, nil)
	}
	if strings.TrimSpace(prompt) == "" || !utf8.ValidString(prompt) {
		return codexwork.WorkPacketV1{}, nil, fail(CodeWorkPacketPromptInvalid, nil)
	}

	packet := codexwork.WorkPacketV1{
		Schema:           codexwork.WorkPacketSchemaV1,
		Prompt:           prompt,
		Model:            model,
		Effort:           string(request.ReasoningEffort),
		TokenBudget:      tokenBudget,
		MaxOutputBytes:   uint64(request.MaxOutputBytes),
		TimeBudgetMS:     timeBudgetMS,
		GoalRef:          request.GoalRef.String(),
		WorkItemRef:      request.WorkItemRef.String(),
		ExecutionRef:     request.ExecutionRef.String(),
		EffectAttemptRef: request.EffectAuthority.EffectAttemptRef,
	}
	if controlledEgress {
		packet.ControlledEgressProxy = codexwork.ControlledEgressProxyURLV1
	}
	// Marshal a prompt-free packet first. The final JSON cannot be smaller than
	// this encoding plus len(prompt), even when the prompt needs no escaping.
	// This avoids allocating a second multi-megabyte buffer for impossible input.
	minimum := packet
	minimum.Prompt = ""
	minimumRaw, marshalErr := json.Marshal(minimum)
	if marshalErr != nil {
		return codexwork.WorkPacketV1{}, nil, fail(CodeWorkPacketEncodeFailed, nil)
	}
	if len(minimumRaw) >= codexwork.MaxPacketBytesV1 ||
		len(prompt) > codexwork.MaxPacketBytesV1-len(minimumRaw) {
		return codexwork.WorkPacketV1{}, nil, fail(CodeWorkPacketTooLarge, nil)
	}
	if err := packet.Validate(); err != nil {
		return codexwork.WorkPacketV1{}, nil, fail(CodeWorkPacketRequestInvalid, nil)
	}
	raw, marshalErr := json.Marshal(packet)
	if marshalErr != nil {
		return codexwork.WorkPacketV1{}, nil, fail(CodeWorkPacketEncodeFailed, nil)
	}
	if len(raw) > codexwork.MaxPacketBytesV1 {
		return codexwork.WorkPacketV1{}, nil, fail(CodeWorkPacketTooLarge, nil)
	}
	decoded, decodeErr := codexwork.DecodeWorkPacketV1(raw)
	if decodeErr != nil || decoded != packet {
		return codexwork.WorkPacketV1{}, nil, fail(CodeWorkPacketRoundTripInvalid, nil)
	}
	return packet, append([]byte(nil), raw...), nil
}

func workPacketBudgets(request ports.AgentLaunchRequest) (uint64, uint64, error) {
	resources := request.BudgetDemand.Resources
	if governance.ValidateBudgetDemand(request.BudgetDemand) != nil ||
		resources.Tokens <= 0 || uint64(resources.Tokens) > codexwork.MaxTokenBudgetV1 ||
		resources.ActiveTimeNS <= 0 || resources.ActiveTimeNS%int64(time.Millisecond) != 0 ||
		resources.DiskBytes <= 0 {
		return 0, 0, fail(CodeWorkPacketBudgetInvalid, nil)
	}
	timeBudgetMS := resources.ActiveTimeNS / int64(time.Millisecond)
	if timeBudgetMS <= 0 || uint64(timeBudgetMS) > codexwork.MaxTimeBudgetMSV1 {
		return 0, 0, fail(CodeWorkPacketBudgetInvalid, nil)
	}
	return uint64(resources.Tokens), uint64(timeBudgetMS), nil
}

func validWorkPacketToken(value string, limit int) bool {
	if value == "" || len(value) > limit || value != strings.TrimSpace(value) || !utf8.ValidString(value) {
		return false
	}
	for _, character := range value {
		if character < 0x20 || character == 0x7f {
			return false
		}
	}
	return true
}
