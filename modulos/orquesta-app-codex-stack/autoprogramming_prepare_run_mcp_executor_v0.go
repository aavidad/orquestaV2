package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestafactoryhttp "orquesta/modulos/orquesta-factory-http"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

type CodexStackAutoprogrammingPrepareRunExecutorV0 struct {
	Stack              *StackV0
	DefaultOccurredAt  string
	DefaultRequestedBy string
	QueueWriter        orquestarunqueue.RunQueuePriorityWriterPortV0
	Queue              RunQueueConfigV0
	Clock              orquestafactoryhttp.AppSpecHTTPClockV0
	RuntimeWorkDir     string
}

var _ orquestamcp.MCPTransportAutoprogrammingPrepareRunExecutorV0 = CodexStackAutoprogrammingPrepareRunExecutorV0{}

func NewCodexStackAutoprogrammingPrepareRunExecutorV0(
	stack *StackV0,
	defaultOccurredAt string,
	defaultRequestedBy string,
	queueWriter orquestarunqueue.RunQueuePriorityWriterPortV0,
	queue RunQueueConfigV0,
	clock orquestafactoryhttp.AppSpecHTTPClockV0,
	runtimeWorkDir string,
) CodexStackAutoprogrammingPrepareRunExecutorV0 {
	return CodexStackAutoprogrammingPrepareRunExecutorV0{
		Stack:              stack,
		DefaultOccurredAt:  strings.TrimSpace(defaultOccurredAt),
		DefaultRequestedBy: strings.TrimSpace(defaultRequestedBy),
		QueueWriter:        queueWriter,
		Queue:              normalizeRunQueueConfigV0(queue),
		Clock:              clock,
		RuntimeWorkDir:     strings.TrimSpace(runtimeWorkDir),
	}
}

func (executor CodexStackAutoprogrammingPrepareRunExecutorV0) Execute(
	ctx context.Context,
	input orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0,
) (orquestamcp.MCPAutoprogrammingPrepareRunToolResultV0, error) {
	if executor.Stack == nil {
		return orquestamcp.NewMCPAutoprogrammingPrepareRunErrorResultV0(
			input,
			"autoprogramming_prepare_run_no_stack",
			"stack",
			"stack requerido",
		), nil
	}
	if issues := configProjectionRequiredSettingIssuesV0(input.RequiredSettings, executor.Stack.ConfigProjectionSettings); len(issues) > 0 {
		return orquestamcp.NewMCPAutoprogrammingPrepareRunIssuesResultV0(input, issues), nil
	}
	bridgeRequest, envelopeIssues := codexStackAutoprogrammingBridgeRequestFromMCPV0(
		input,
		executor.DefaultOccurredAt,
		executor.DefaultRequestedBy,
	)
	if len(envelopeIssues) != 0 {
		return orquestamcp.NewMCPAutoprogrammingPrepareRunIssuesResultV0(input, envelopeIssues), nil
	}
	input = codexStackAutoprogrammingPrepareRunInputFromEnvelopeV0(input, *bridgeRequest.PrepareRunEnvelope)
	retryPlan := executor.planFreshAttemptForStaleAutoprogrammingRunV0(ctx, input)
	if len(retryPlan.Issues) != 0 {
		return orquestamcp.NewMCPAutoprogrammingPrepareRunIssuesResultV0(input, retryPlan.Issues), nil
	}
	input = retryPlan.Input
	if retryPlan.StaleRun != nil {
		bridgeRequest, envelopeIssues = codexStackAutoprogrammingBridgeRequestFromMCPV0(
			input,
			executor.DefaultOccurredAt,
			executor.DefaultRequestedBy,
		)
		if len(envelopeIssues) != 0 {
			return orquestamcp.NewMCPAutoprogrammingPrepareRunIssuesResultV0(input, envelopeIssues), nil
		}
		input = codexStackAutoprogrammingPrepareRunInputFromEnvelopeV0(input, *bridgeRequest.PrepareRunEnvelope)
		if err := validateAutoprogrammingBridgePortsV0(executor.Stack.Ports); err != nil {
			return orquestamcp.NewMCPAutoprogrammingPrepareRunErrorResultV0(input, "autoprogramming_prepare_run_error", "ports", err.Error()), nil
		}
		if validation := orquestaautoprogramming.BuildAutoprogrammingProgrammableWorkV0(bridgeRequest.PrepareRunEnvelope.AutoprogrammingRequest); !validation.Accepted {
			return orquestamcp.NewMCPAutoprogrammingPrepareRunIssuesResultV0(input, codexStackAutoprogrammingIssuesMCPV0(validation.Issues)), nil
		}
		_, issues, err := persistAutoprogrammingPrepareRunAuthorityV0(
			ctx,
			executor.Stack.Stores,
			bridgeRequest.PrepareRunEnvelope.AutoprogrammingRequest,
			bridgeRequest.PrepareRunEnvelope,
		)
		if err != nil {
			return orquestamcp.NewMCPAutoprogrammingPrepareRunErrorResultV0(input, "autoprogramming_prepare_run_error", "intent_authority", err.Error()), nil
		}
		if len(issues) != 0 {
			return orquestamcp.NewMCPAutoprogrammingPrepareRunIssuesResultV0(input, codexStackAutoprogrammingIssuesMCPV0(issues)), nil
		}
		if err := executor.markStaleAutoprogrammingQueueCandidateV0(ctx, *retryPlan.StaleRun); err != nil {
			return orquestamcp.NewMCPAutoprogrammingPrepareRunErrorResultV0(input, "autoprogramming_prepare_run_queue_error", "run_queue", err.Error()), nil
		}
	}
	result, err := PrepareAutoprogrammingRunFromStackV0(ctx, *executor.Stack, bridgeRequest)
	if err != nil {
		return orquestamcp.NewMCPAutoprogrammingPrepareRunErrorResultV0(
			input,
			"autoprogramming_prepare_run_error",
			"executor",
			err.Error(),
		), nil
	}
	if result.Accepted && autoprogrammingBridgeHasPreparedLegacyRunV0(result) {
		if err := executor.enqueuePreparedRunV0(ctx, input, result); err != nil {
			return orquestamcp.NewMCPAutoprogrammingPrepareRunErrorResultV0(
				input,
				"autoprogramming_prepare_run_queue_error",
				"run_queue",
				err.Error(),
			), nil
		}
	}
	return codexStackAutoprogrammingPrepareRunResultMCPV0(input, result), nil
}
