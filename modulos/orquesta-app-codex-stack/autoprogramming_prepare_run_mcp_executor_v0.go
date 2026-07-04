package orquestaappcodexstack

import (
	"context"
	"strings"

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
	input = executor.freshAttemptForStaleAutoprogrammingRunV0(ctx, input)
	bridgeRequest := codexStackAutoprogrammingBridgeRequestFromMCPV0(
		input,
		executor.DefaultOccurredAt,
		executor.DefaultRequestedBy,
	)
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
