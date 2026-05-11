package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"

	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

type QueuedArrancarDirectorConfigV0 struct {
	Inner  orquestamcp.MCPTransportArrancarDirectorAppExecutorV0
	Writer orquestarunqueue.RunQueuePriorityWriterPortV0
	Queue  RunQueueConfigV0
	Clock  orquestafactory.AppSpecHTTPClockV0
	Source string
	Reason string
}

type QueuedArrancarDirectorExecutorV0 struct {
	config QueuedArrancarDirectorConfigV0
}

func NewQueuedArrancarDirectorExecutorV0(
	config QueuedArrancarDirectorConfigV0,
) QueuedArrancarDirectorExecutorV0 {
	config.Queue = normalizeRunQueueConfigV0(config.Queue)
	return QueuedArrancarDirectorExecutorV0{config: config}
}

func (executor QueuedArrancarDirectorExecutorV0) Execute(
	ctx context.Context,
	input orquestamcp.MCPArrancarDirectorAppToolInputV0,
) (orquestamcp.MCPArrancarDirectorAppToolResultV0, error) {
	if executor.config.Inner == nil {
		return orquestamcp.MCPArrancarDirectorAppToolResultV0{}, fmt.Errorf("arrancar_director.inner requerido")
	}
	result, err := executor.config.Inner.Execute(ctx, input)
	if err != nil || result.Estado != orquestamcp.MCPArrancarDirectorAppEstadoOKV0 {
		return result, err
	}
	if executor.config.Writer == nil {
		return result, fmt.Errorf("run_queue.writer requerido")
	}
	_, err = executor.config.Writer.SetRunPriorityV0(ctx, executor.queueCommandV0(result))
	return result, err
}

func (executor QueuedArrancarDirectorExecutorV0) queueCommandV0(
	result orquestamcp.MCPArrancarDirectorAppToolResultV0,
) orquestarunqueue.RunQueuePriorityCommandV0 {
	runRef := strings.TrimSpace(result.RunRef)
	return orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:         runRef,
		QueueRef:       executor.config.Queue.QueueRef,
		AppRef:         appRefFromStartedDirectorV0(result),
		PriorityScore:  executor.config.Queue.DefaultPriorityScore,
		UpdatedAt:      stackNowV0(executor.config.Clock),
		RequestedBy:    firstNonEmptyQueuedSourceV0(executor.config.Source, "orquesta-app-codex-stack"),
		Reason:         firstNonEmptyQueuedSourceV0(executor.config.Reason, "run_created"),
		IdempotencyKey: "idem-run-queue-start-" + runRef,
		EvidenceRefs:   []string{"evidence-ref-run-queue-enqueued"},
	}
}

func appRefFromStartedDirectorV0(
	result orquestamcp.MCPArrancarDirectorAppToolResultV0,
) string {
	return firstNonEmptyQueuedSourceV0(
		result.AppSpec.Slug,
		result.AppSpec.SpecID,
		result.AppSpec.Nombre,
		result.RunRef,
	)
}

func firstNonEmptyQueuedSourceV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
