package cmd

import (
	"errors"
	"time"

	"orquesta/agentesapp"
)

var controlPlanePanelRowsTimeout = 500 * time.Millisecond

func buildPanelRowsForControlPlane() ([]agentesapp.Row, error) {
	if controlPlanePanelRowsTimeout <= 0 {
		return agentesService.BuildPanelRows()
	}
	rows, err := agentRowsForStatusWithinTimeout(controlPlanePanelRowsTimeout)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func controlPlaneRowsTimedOut(err error) bool {
	return errors.Is(err, errStatusFetchTimeout)
}
