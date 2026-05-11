package orquestadirector

import (
	"fmt"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

type progressiveRuntimeFakeOutboxDispatchErrorCodeV0 string

const (
	progressiveRuntimeFakeOutboxLenInvalidoV0      progressiveRuntimeFakeOutboxDispatchErrorCodeV0 = "runtime_fake_outbox_len_invalido"
	progressiveRuntimeFakeOutboxTipoNoSoportadoV0  progressiveRuntimeFakeOutboxDispatchErrorCodeV0 = "runtime_fake_outbox_tipo_no_soportado"
	progressiveRuntimeFakeOutboxPayloadInvalidoV0  progressiveRuntimeFakeOutboxDispatchErrorCodeV0 = "runtime_fake_outbox_payload_invalido"
	progressiveRuntimeFakeOutboxContratoInvalidoV0 progressiveRuntimeFakeOutboxDispatchErrorCodeV0 = "runtime_fake_outbox_contrato_invalido"
	progressiveRuntimeFakeOutboxDispatchFailedV0   progressiveRuntimeFakeOutboxDispatchErrorCodeV0 = "runtime_fake_outbox_dispatch_failed"
)

type progressiveRuntimeFakeOutboxDispatchErrorV0 struct {
	Code        progressiveRuntimeFakeOutboxDispatchErrorCodeV0
	Field       string
	MessageType string
	TargetPort  string
	Evidence    []string
	Err         error
}

func (e progressiveRuntimeFakeOutboxDispatchErrorV0) Error() string {
	detail := e.Field
	if e.MessageType != "" || e.TargetPort != "" {
		detail = fmt.Sprintf("%s message_type=%q target_port=%q", detail, e.MessageType, e.TargetPort)
	}
	if len(e.Evidence) != 0 {
		detail = fmt.Sprintf("%s evidence=%v", detail, e.Evidence)
	}
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, detail, e.Err)
	}
	if detail == "" {
		return string(e.Code)
	}
	return fmt.Sprintf("%s: %s", e.Code, detail)
}

func (e progressiveRuntimeFakeOutboxDispatchErrorV0) Unwrap() error {
	return e.Err
}

func validateProgressiveRuntimeOutboxEnvelopeV0(
	message orquestacoreworkflow.OutboxMessageV0,
) error {
	if err := orquestacoreworkflow.ValidateOutboxMessageV0(message); err != nil {
		return progressiveRuntimeFakeOutboxDispatchErrorV0{
			Code:        progressiveRuntimeFakeOutboxContratoInvalidoV0,
			Field:       "outbox",
			MessageType: message.MessageType,
			TargetPort:  message.TargetPort,
			Err:         err,
		}
	}
	return nil
}

func progressiveOutboxLenErrorV0(length int) progressiveRuntimeFakeOutboxDispatchErrorV0 {
	return progressiveRuntimeFakeOutboxDispatchErrorV0{
		Code:     progressiveRuntimeFakeOutboxLenInvalidoV0,
		Field:    "outbox",
		Evidence: []string{fmt.Sprintf("len=%d", length)},
	}
}

func progressivePayloadErrorV0(
	message orquestacoreworkflow.OutboxMessageV0,
	err error,
) progressiveRuntimeFakeOutboxDispatchErrorV0 {
	return progressiveRuntimeFakeOutboxDispatchErrorV0{
		Code:        progressiveRuntimeFakeOutboxPayloadInvalidoV0,
		Field:       "payload",
		MessageType: message.MessageType,
		TargetPort:  message.TargetPort,
		Err:         err,
	}
}

func progressiveLaunchContractErrorV0(
	message orquestacoreworkflow.OutboxMessageV0,
	issues []orquestaruntime.AgentLauncherInboundErrorV0,
) progressiveRuntimeFakeOutboxDispatchErrorV0 {
	evidence := make([]string, 0, len(issues))
	for _, issue := range issues {
		evidence = append(evidence, string(issue.Code))
	}
	return progressiveContractErrorV0(message, "launch", evidence)
}

func progressiveStopContractErrorV0(
	message orquestacoreworkflow.OutboxMessageV0,
	issues []orquestaruntime.AgentStopperInboundErrorV0,
) progressiveRuntimeFakeOutboxDispatchErrorV0 {
	evidence := make([]string, 0, len(issues))
	for _, issue := range issues {
		evidence = append(evidence, string(issue.Code))
	}
	return progressiveContractErrorV0(message, "stop", evidence)
}

func progressiveContractErrorV0(
	message orquestacoreworkflow.OutboxMessageV0,
	field string,
	evidence []string,
) progressiveRuntimeFakeOutboxDispatchErrorV0 {
	return progressiveRuntimeFakeOutboxDispatchErrorV0{
		Code:        progressiveRuntimeFakeOutboxContratoInvalidoV0,
		Field:       field,
		MessageType: message.MessageType,
		TargetPort:  message.TargetPort,
		Evidence:    evidence,
	}
}

func progressiveDispatchFailedErrorV0(
	message orquestacoreworkflow.OutboxMessageV0,
	field string,
	err error,
) progressiveRuntimeFakeOutboxDispatchErrorV0 {
	return progressiveRuntimeFakeOutboxDispatchErrorV0{
		Code:        progressiveRuntimeFakeOutboxDispatchFailedV0,
		Field:       field,
		MessageType: message.MessageType,
		TargetPort:  message.TargetPort,
		Err:         err,
	}
}

func TestProgressiveRuntimeFakeOutboxDispatcherV0RechazaMensajeNoSoportado(t *testing.T) {
	dispatcher := newProgressiveRuntimeFakeOutboxDispatcherV0(t)

	_, err := dispatcher.dispatch(orquestacoreworkflow.OutboxMessageV0{
		MessageType: orquestacoreworkflow.OutboxMessageRequestCapacityDecisionV0,
		TargetPort:  orquestacoreworkflow.OutboxTargetCapacityV0,
	})
	if err == nil {
		t.Fatalf("dispatch unsupported outbox: nil error")
	}
	dispatchErr, ok := err.(progressiveRuntimeFakeOutboxDispatchErrorV0)
	if !ok {
		t.Fatalf("dispatch error type=%T, want progressiveRuntimeFakeOutboxDispatchErrorV0", err)
	}
	if dispatchErr.Code != progressiveRuntimeFakeOutboxTipoNoSoportadoV0 {
		t.Fatalf("dispatch code=%s, want %s", dispatchErr.Code, progressiveRuntimeFakeOutboxTipoNoSoportadoV0)
	}
}
