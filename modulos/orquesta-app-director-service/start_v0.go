package orquestaappdirectorservice

import (
	"context"

	orquestaappdirectorintake "orquesta/modulos/orquesta-app-director-intake"
	orquestafactory "orquesta/modulos/orquesta-factory"
)

func StartAppDirectorV0(
	ctx context.Context,
	request StartAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
) (StartAppDirectorResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	request = normalizeStartAppDirectorRequestV0(request)
	request = startAppDirectorRequestWithOperationalPlanRunRefV0(request)
	if err := validateStartAppDirectorRequestV0(request); err != nil {
		return StartAppDirectorResultV0{}, err
	}
	now, err := startAppDirectorNowV0(request.OccurredAt)
	if err != nil {
		return StartAppDirectorResultV0{}, err
	}
	if request.OccurredAt == "" {
		request.OccurredAt = now.Format("2006-01-02T15:04:05Z07:00")
	}
	spec, issues := orquestafactory.SolicitarNuevaAppV0(request.AppSpecRequest, now)
	if len(issues) > 0 {
		return invalidStartAppDirectorResultV0(request, issues), nil
	}
	if err := validateStartAppDirectorPortsV0(ports); err != nil {
		return StartAppDirectorResultV0{}, err
	}
	prepared, err := orquestaappdirectorintake.PrepareAppDirectorIntakeV0(
		prepareDirectorIntakeRequestV0(request, spec),
	)
	if err != nil {
		return StartAppDirectorResultV0{}, err
	}
	if err := persistPreparedDirectorIntakeV0(ctx, ports, prepared); err != nil {
		return StartAppDirectorResultV0{}, err
	}
	request, err = startAppDirectorWithOperationalDirectorPlanV0(ctx, request, ports, prepared)
	if err != nil {
		return StartAppDirectorResultV0{}, err
	}
	loop, err := runPreparedDirectorAutonomyLoopV0(ctx, request, ports, prepared)
	if err != nil {
		return StartAppDirectorResultV0{}, err
	}
	return startAppDirectorResultV0(request, spec, prepared, loop), nil
}

func prepareDirectorIntakeRequestV0(
	request StartAppDirectorRequestV0,
	spec orquestafactory.AppSpecV0,
) orquestaappdirectorintake.PrepareAppDirectorIntakeRequestV0 {
	return orquestaappdirectorintake.PrepareAppDirectorIntakeRequestV0{
		RunRef:        request.RunRef,
		ProjectRef:    request.ProjectRef,
		OccurredAt:    request.OccurredAt,
		CorrelationID: request.CorrelationID,
		RequestedBy:   request.RequestedBy,
		AppSpec:       spec,
	}
}

func persistPreparedDirectorIntakeV0(
	ctx context.Context,
	ports StartAppDirectorPortsV0,
	prepared orquestaappdirectorintake.AppDirectorIntakePreparedV0,
) error {
	if len(prepared.InitialEvents) > 0 {
		if err := ports.EventSink.AppendRunEventsV0(ctx, prepared.Run.RunID, prepared.InitialEvents); err != nil {
			return err
		}
	}
	return ports.RunStore.SaveRunV0(ctx, prepared.Run)
}
