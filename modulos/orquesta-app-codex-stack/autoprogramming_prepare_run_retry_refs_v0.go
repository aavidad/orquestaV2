package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"
	"time"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func autoprogrammingPrepareRetryAttemptV0(requestRef string) int {
	return strings.Count(strings.TrimSpace(requestRef), "-retry-")
}

func reframeAutoprogrammingRequestAfterRepeatedRetriesV0(
	request orquestaautoprogramming.AutoprogrammingRequestV0,
	previousRunRef string,
	nextAttempt int,
) orquestaautoprogramming.AutoprogrammingRequestV0 {
	if len(request.Tasks) == 0 {
		return request
	}
	out := request
	out.Tasks = append([]orquestaautoprogramming.AutoprogrammingTaskGroupCandidateV0(nil), request.Tasks...)
	previousRef := "previous_run_ref:" + autoprogrammingPrepareRetrySafeRefV0(previousRunRef)
	attemptRef := fmt.Sprintf("retry_attempt:%02d", nextAttempt)
	for index := range out.Tasks {
		out.Tasks[index].ContextRefs = compactStringsV0(append(
			append([]string(nil), out.Tasks[index].ContextRefs...),
			previousRef,
			attemptRef,
		))
	}
	if nextAttempt < 2 {
		return out
	}
	strategyRef := fmt.Sprintf("retry_strategy:reframe-%02d", nextAttempt)
	for index := range out.Tasks {
		out.Tasks[index] = reframeAutoprogrammingTaskAfterRepeatedRetriesV0(
			out.Tasks[index],
			nextAttempt,
			strategyRef,
			previousRef,
		)
	}
	return out
}

func reframeAutoprogrammingTaskAfterRepeatedRetriesV0(
	task orquestaautoprogramming.AutoprogrammingTaskGroupCandidateV0,
	nextAttempt int,
	strategyRef string,
	previousRef string,
) orquestaautoprogramming.AutoprogrammingTaskGroupCandidateV0 {
	task.Context = compactStringsV0(append(
		append([]string(nil), task.Context...),
		fmt.Sprintf("Retry %d: el intento anterior quedo atascado; cambia enfoque antes de reintentar.", nextAttempt),
		"Si el contrato es demasiado amplio, entrega un minimo verificable y deja followups causales.",
	))
	task.ContextRefs = compactStringsV0(append(append([]string(nil), task.ContextRefs...), strategyRef, previousRef))
	task.AcceptanceCriteria = compactStringsV0(append(
		append([]string(nil), task.AcceptanceCriteria...),
		"No repetir literalmente el intento fallido; reencuadrar, dividir o normalizar antes de programar.",
		"El ACK explica que cambio de enfoque se aplico y que pruebas verifican el resultado.",
	))
	task.CompactRules = compactStringsV0(append(
		append([]string(nil), task.CompactRules...),
		fmt.Sprintf("retry_reframed_attempt:%02d", nextAttempt),
		"preferir cambios pequenos verificables si el intento anterior fallo por tamano, payload o contrato",
	))
	if strings.TrimSpace(task.Objective) == "" {
		task.Objective = "Reintento reencuadrado: completar la mejora con el menor cambio verificable."
	}
	return task
}

func (executor CodexStackAutoprogrammingPrepareRunExecutorV0) markStaleAutoprogrammingQueueCandidateV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
) error {
	if executor.QueueWriter == nil {
		return nil
	}
	_, err := executor.QueueWriter.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        strings.TrimSpace(run.RunID),
		QueueRef:      executor.Queue.QueueRef,
		AppRef:        firstNonEmptyAutoprogrammingStackV0(run.ProjectRef, run.AppSpecRef, run.RunID),
		Status:        orquestarunqueue.RunStatusStoppedV0,
		PriorityScore: 0,
		UpdatedAt:     stackNowV0(executor.Clock),
		RequestedBy: firstNonEmptyAutoprogrammingStackV0(
			executor.DefaultRequestedBy,
			"orquesta-app-codex-stack-autoprogramming",
		),
		Reason:         "autoprogramming_stale_run_retried",
		IdempotencyKey: "idem-run-queue-autoprogramming-stale-" + autoprogrammingPrepareRetrySafeRefV0(run.RunID),
		EvidenceRefs: []string{
			"evidence-ref-autoprogramming-stale-run-retried",
			"evidence-ref-autoprogramming-old-run-preserved",
		},
	})
	return err
}

func autoprogrammingPrepareRetryRefV0(requestRef string, occurredAt string, now time.Time) string {
	base := strings.TrimSpace(requestRef)
	if base == "" {
		base = "request-ref-autoprogramming"
	}
	stamp := strings.TrimSpace(occurredAt)
	if stamp == "" && !now.IsZero() {
		stamp = now.UTC().Format("20060102T150405Z")
	}
	hash := autoprogrammingPrepareRetryHashV0(base, stamp)
	return base + "-retry-" + hash
}

func autoprogrammingPrepareRetryHashV0(values ...string) string {
	return codexStackDeterministicDigestV0(values...)
}

func autoprogrammingPrepareRetrySafeRefV0(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 80 {
		return value
	}
	return value[:67] + "-" + autoprogrammingPrepareRetryHashV0(value)
}
