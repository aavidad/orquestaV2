package orquestacoreworkflow

import (
	"errors"
	"fmt"
	"strings"
)

func ValidateOrchestrationCommandV0(command OrchestrationCommandV0) error {
	if err := validateCommandEnvelopeV0(command); err != nil {
		return err
	}
	if err := validateCommandPayloadBudgetV0(command); err != nil {
		return err
	}
	return validateCommandPayloadV0(command)
}

func validateCommandEnvelopeV0(command OrchestrationCommandV0) error {
	if strings.TrimSpace(command.CommandID) == "" {
		return commandErrorV0(ErrComandoInvalidoV0, "command_id")
	}
	if !isSupportedCommandTypeV0(command.CommandType) {
		return commandErrorV0(ErrComandoNoSoportadoV0, "command_type")
	}
	if strings.TrimSpace(command.RunID) == "" {
		return commandErrorV0(ErrComandoInvalidoV0, "run_id")
	}
	if strings.TrimSpace(command.IdempotencyKey) == "" {
		return commandErrorV0(ErrIdempotencyKeyRequeridaV0, "idempotency_key")
	}
	if strings.TrimSpace(command.OccurredAt) == "" {
		return commandErrorV0(ErrComandoInvalidoV0, "occurred_at")
	}
	if command.PayloadVersion != OrchestrationCommandPayloadVersionV0 {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload_version")
	}
	if len(command.Payload) == 0 {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func validateCommandPayloadV0(command OrchestrationCommandV0) error {
	switch command.CommandType {
	case OrchestrationCommandStartRunV0:
		payload, err := decodeStartRunCommandPayloadV0(command.Payload)
		if err != nil {
			return err
		}
		return requireCommandPayloadFieldsV0(map[string]string{
			"project_ref":  payload.ProjectRef,
			"app_spec_ref": payload.AppSpecRef,
		})
	case OrchestrationCommandOpenPhaseV0:
		payload, err := decodeOpenPhaseCommandPayloadV0(command.Payload)
		if err != nil {
			return err
		}
		return ValidateOrchestrationPhaseIDV0(OrchestrationPhaseIDV0(payload.PhaseID))
	case OrchestrationCommandClosePhaseV0:
		_, err := decodeClosePhaseCommandPayloadV0(command.Payload)
		return err
	case OrchestrationCommandBlockRunV0:
		payload, err := decodeBlockRunCommandPayloadV0(command.Payload)
		if err != nil {
			return err
		}
		return requireCommandPayloadFieldsV0(map[string]string{
			"blocker_id":  payload.BlockerID,
			"reason_code": payload.ReasonCode,
			"summary":     payload.Summary,
		})
	default:
		return validateExtendedCommandPayloadV0(command)
	}
}

func validateExtendedCommandPayloadV0(command OrchestrationCommandV0) error {
	switch command.CommandType {
	case OrchestrationCommandAskDirectorV0:
		return commandPayloadErrorV0(validateAskDirectorPayloadForCommandV0(command))
	case OrchestrationCommandAnswerDirectorQuestionV0:
		return commandPayloadErrorV0(validateAnswerDirectorQuestionPayloadForCommandV0(command))
	case OrchestrationCommandRequestBrainstormV0:
		_, err := decodeRequestBrainstormCommandPayloadV0(command.Payload)
		return err
	case OrchestrationCommandRequestVoteV0:
		_, err := decodeRequestVoteCommandPayloadV0(command.Payload)
		return err
	case OrchestrationCommandAcceptDecisionV0:
		_, err := decodeAcceptDecisionCommandPayloadV0(command.Payload)
		return err
	case OrchestrationCommandPublishFunctionContractV0:
		_, err := decodePublishFunctionContractCommandPayloadV0(command.Payload)
		return err
	case OrchestrationCommandCreateMicrotaskV0:
		_, err := decodeCreateMicrotaskCommandPayloadV0(command.Payload)
		return err
	case OrchestrationCommandRequestCapacityV0:
		_, err := decodeRequestCapacityCommandPayloadV0(command.Payload)
		return err
	case OrchestrationCommandRegisterCapacityDecisionV0:
		_, err := decodeRegisterCapacityDecisionCommandPayloadV0(command.Payload)
		return err
	case OrchestrationCommandRequestAgentV0:
		_, err := decodeRequestAgentCommandPayloadV0(command.Payload)
		return err
	case OrchestrationCommandRegisterAgentStartedV0:
		_, err := decodeRegisterAgentStartedCommandPayloadV0(command.Payload)
		return err
	case OrchestrationCommandRegisterAgentFailedV0:
		_, err := decodeRegisterAgentFailedCommandPayloadV0(command.Payload)
		return err
	case OrchestrationCommandRegisterAgentLostV0:
		_, err := decodeRegisterAgentLostCommandPayloadV0(command.Payload)
		return err
	case OrchestrationCommandRegisterAgentLeaseExpiredV0:
		_, err := decodeRegisterAgentLeaseExpiredCommandPayloadV0(command.Payload)
		return err
	case OrchestrationCommandStopAgentV0:
		_, err := decodeStopAgentCommandPayloadV0(command.Payload)
		return err
	case OrchestrationCommandRegisterAgentStopConfirmedV0:
		_, err := decodeRegisterAgentStopConfirmedCommandPayloadV0(command.Payload)
		return err
	case OrchestrationCommandAssessAgentWorkV0:
		_, err := decodeAssessAgentWorkCommandPayloadV0(command.Payload)
		return err
	case OrchestrationCommandRecordConcurrencyGateV0:
		_, err := decodeRecordConcurrencyGateCommandPayloadV0(command.Payload)
		return err
	case OrchestrationCommandRecordQualityGateV0:
		_, err := decodeRecordQualityGateCommandPayloadV0(command.Payload)
		return err
	case OrchestrationCommandRegisterPhaseArtifactV0:
		_, err := decodeRegisterPhaseArtifactCommandPayloadV0(command.Payload)
		return err
	case OrchestrationCommandRegisterDeliveryV0:
		_, err := decodeRegisterDeliveryCommandPayloadV0(command.Payload)
		return err
	case OrchestrationCommandRequestReviewV0:
		_, err := decodeRequestReviewCommandPayloadV0(command.Payload)
		return err
	case OrchestrationCommandAcceptReviewV0:
		_, err := decodeAcceptReviewCommandPayloadV0(command.Payload)
		return err
	case OrchestrationCommandRecordReviewResultV0:
		_, err := decodeRecordReviewResultCommandPayloadV0(command.Payload)
		return err
	case OrchestrationCommandRequestReworkV0:
		_, err := decodeRequestReworkCommandPayloadV0(command.Payload)
		return err
	case OrchestrationCommandRecordReplanDecisionV0:
		_, err := decodeRecordReplanDecisionCommandPayloadV0(command.Payload)
		return err
	case OrchestrationCommandCloseTaskV0:
		_, err := decodeCloseTaskCommandPayloadV0(command.Payload)
		return err
	case OrchestrationCommandRegisterFinalValidationV0:
		_, err := decodeRegisterFinalValidationCommandPayloadV0(command.Payload)
		return err
	case OrchestrationCommandCloseRunV0:
		_, err := decodeCloseRunCommandPayloadV0(command.Payload)
		return err
	default:
		return commandErrorV0(ErrComandoNoSoportadoV0, "command_type")
	}
}

func commandPayloadErrorV0(err error) error {
	if err == nil {
		return nil
	}
	var commandErr OrchestrationCommandErrorV0
	if errors.As(err, &commandErr) {
		return commandErr
	}
	var questionErr DirectorQuestionErrorV0
	if errors.As(err, &questionErr) {
		field := strings.TrimSpace(questionErr.Field)
		if field == "" {
			return commandErrorV0(ErrPayloadInvalidoV0, "payload")
		}
		return commandErrorV0(ErrPayloadInvalidoV0, "payload."+field)
	}
	var issue OrchestrationValidationIssueV0
	if errors.As(err, &issue) {
		return issue
	}
	var taskErr WorkflowTaskErrorV0
	if errors.As(err, &taskErr) {
		field := strings.TrimSpace(taskErr.Field)
		if field == "" {
			return commandErrorV0(ErrPayloadInvalidoV0, "payload")
		}
		if taskErr.Code == ErrDetalleProhibidoV0 {
			return commandErrorV0(ErrDetalleProhibidoV0, "payload.task."+field)
		}
		return commandErrorV0(ErrPayloadInvalidoV0, "payload.task."+field)
	}
	return commandErrorV0(ErrPayloadInvalidoV0, "payload")
}

func requireCommandPayloadFieldsV0(fields map[string]string) error {
	for field, value := range fields {
		if strings.TrimSpace(value) == "" {
			return commandErrorV0(ErrPayloadInvalidoV0, fmt.Sprintf("payload.%s", field))
		}
	}
	return nil
}

func isSupportedCommandTypeV0(commandType string) bool {
	needle := strings.TrimSpace(commandType)
	for _, supported := range orchestrationCommandTypeCatalogV0 {
		if supported == needle {
			return true
		}
	}
	return false
}
