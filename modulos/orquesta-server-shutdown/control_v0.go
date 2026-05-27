package orquestaservershutdown

import (
	"context"
	"errors"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
)

func requestShutdownStopV0(
	ctx context.Context,
	deps ServerShutdownDepsV0,
	command ServerShutdownCommandV0,
	runRef string,
) (orquestaruncontrol.RunControlStateV0, error) {
	return deps.RunControlWriter.StopRunV0(ctx, orquestaruncontrol.StopRunCommandV0{
		RunRef:         runRef,
		RequestedBy:    command.RequestedBy,
		Reason:         command.Reason,
		Forced:         command.Forced,
		IdempotencyKey: command.IdempotencyKey,
		EvidenceRefs:   serverShutdownRunControlEvidenceRefsV0(command),
	})
}

func serverShutdownRunControlEvidenceRefsV0(command ServerShutdownCommandV0) []string {
	return compactServerShutdownStringsV0(append(
		append([]string(nil), command.EvidenceRefs...),
		orquestaruncontrol.RunControlEvidenceAutoResumeAllowedV0,
		"evidence-ref-server-shutdown-run-auto-resume-safe",
	))
}

func readShutdownControlStateV0(
	ctx context.Context,
	reader orquestaruncontrol.RunControlReaderPortV0,
	runRef string,
) (orquestaruncontrol.RunControlStateV0, error) {
	state, err := reader.ReadRunControlStateV0(ctx, orquestaruncontrol.RunControlReadRequestV0{
		RunRef: runRef,
	})
	if err == nil {
		return state, nil
	}
	var notFound orquestaruncontrol.RunControlStateNotFoundErrorV0
	if errors.As(err, &notFound) {
		return orquestaruncontrol.DefaultRunControlStateV0(runRef), nil
	}
	return orquestaruncontrol.RunControlStateV0{}, err
}
