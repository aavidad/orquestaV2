package orquestaexternalworkrun

import (
	"context"
	"time"

	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func enqueueExternalWorkRunV0(
	ctx context.Context,
	request StartExternalWorkRunRequestV0,
	writer orquestarunqueue.RunQueuePriorityWriterPortV0,
) error {
	occurredAt, err := time.Parse(time.RFC3339, request.OccurredAt)
	if err != nil {
		return err
	}
	_, err = writer.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:         request.RunRef,
		QueueRef:       request.QueueRef,
		AppRef:         request.AppChangeRequest.AppRef,
		PriorityScore:  request.PriorityScore,
		UpdatedAt:      occurredAt,
		RequestedBy:    request.RequestedBy,
		Reason:         "external_work_run_created",
		IdempotencyKey: "idem-external-work-run-queue-" + request.RunRef,
		EvidenceRefs: []string{
			"evidence-ref-external-work-run-queued",
			request.AppChangeRequest.ChangeRef,
		},
	})
	return err
}
