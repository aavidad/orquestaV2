package main

import (
	"context"
	"errors"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
	orquestaservershutdown "orquesta/modulos/orquesta-server-shutdown"
)

func serverGoalActiveShutdownWorkReaderFromBackendV0(
	backend serverCodexGoalBackendV0,
) orquestaservershutdown.ActiveShutdownWorkReaderPortV0 {
	for _, candidate := range []interface{}{backend.ShutdownHook, backend.Observer, backend.Starter} {
		if active, ok := candidate.(orquestaservershutdown.ActiveShutdownWorkReaderPortV0); ok {
			return active
		}
	}
	return nil
}

func serverGoalActiveShutdownWorkCleanerFromBackendV0(
	backend serverCodexGoalBackendV0,
) orquestaservershutdown.ActiveShutdownWorkCleanerPortV0 {
	for _, candidate := range []interface{}{backend.ShutdownHook, backend.Observer, backend.Starter} {
		if cleaner, ok := candidate.(orquestaservershutdown.ActiveShutdownWorkCleanerPortV0); ok {
			return cleaner
		}
	}
	return nil
}

type serverGoalWorkLauncherWithActiveShutdownWorkV0 struct {
	Inner         orquestagoal.GoalWorkLauncherPortV0
	ActiveWork    orquestaservershutdown.ActiveShutdownWorkReaderPortV0
	ActiveCleaner orquestaservershutdown.ActiveShutdownWorkCleanerPortV0
}

func (launcher serverGoalWorkLauncherWithActiveShutdownWorkV0) LaunchGoalWorkV0(
	ctx context.Context,
	spec orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalLaunchReceiptV0, error) {
	if launcher.Inner == nil {
		return orquestagoal.GoalLaunchReceiptV0{}, errors.New(orquestaruntimecodexgoal.ErrCodexGoalStarterMissingV0)
	}
	return launcher.Inner.LaunchGoalWorkV0(ctx, spec)
}

func (launcher serverGoalWorkLauncherWithActiveShutdownWorkV0) ReadActiveShutdownWorkV0(
	ctx context.Context,
	request orquestaservershutdown.ActiveShutdownWorkRequestV0,
) (orquestaservershutdown.ActiveShutdownWorkResultV0, error) {
	if launcher.ActiveWork == nil {
		return orquestaservershutdown.ActiveShutdownWorkResultV0{}, nil
	}
	return launcher.ActiveWork.ReadActiveShutdownWorkV0(ctx, request)
}

func (launcher serverGoalWorkLauncherWithActiveShutdownWorkV0) CleanupActiveShutdownWorkV0(
	ctx context.Context,
	command orquestaservershutdown.ActiveShutdownWorkCleanupCommandV0,
) (orquestaservershutdown.ActiveShutdownWorkCleanupResultV0, error) {
	if launcher.ActiveCleaner == nil {
		return orquestaservershutdown.ActiveShutdownWorkCleanupResultV0{}, nil
	}
	return launcher.ActiveCleaner.CleanupActiveShutdownWorkV0(ctx, command)
}

type serverGoalWorkObserverWithActiveShutdownWorkV0 struct {
	Inner         orquestagoal.GoalWorkObservationPortV0
	ActiveWork    orquestaservershutdown.ActiveShutdownWorkReaderPortV0
	ActiveCleaner orquestaservershutdown.ActiveShutdownWorkCleanerPortV0
}

func (observer serverGoalWorkObserverWithActiveShutdownWorkV0) ObserveGoalWorkV0(
	ctx context.Context,
	request orquestagoal.GoalObservationRequestV0,
) (orquestagoal.GoalWorkResultV0, error) {
	if observer.Inner == nil {
		return orquestagoal.GoalWorkResultV0{}, errors.New(orquestaruntimecodexgoal.ErrCodexGoalObserverMissingV0)
	}
	return observer.Inner.ObserveGoalWorkV0(ctx, request)
}

func (observer serverGoalWorkObserverWithActiveShutdownWorkV0) ReadActiveShutdownWorkV0(
	ctx context.Context,
	request orquestaservershutdown.ActiveShutdownWorkRequestV0,
) (orquestaservershutdown.ActiveShutdownWorkResultV0, error) {
	if observer.ActiveWork == nil {
		return orquestaservershutdown.ActiveShutdownWorkResultV0{}, nil
	}
	return observer.ActiveWork.ReadActiveShutdownWorkV0(ctx, request)
}

func (observer serverGoalWorkObserverWithActiveShutdownWorkV0) CleanupActiveShutdownWorkV0(
	ctx context.Context,
	command orquestaservershutdown.ActiveShutdownWorkCleanupCommandV0,
) (orquestaservershutdown.ActiveShutdownWorkCleanupResultV0, error) {
	if observer.ActiveCleaner == nil {
		return orquestaservershutdown.ActiveShutdownWorkCleanupResultV0{}, nil
	}
	return observer.ActiveCleaner.CleanupActiveShutdownWorkV0(ctx, command)
}
