package cmd

import (
	"errors"
	"sync"
	"time"

	"orquesta/agentesapp"
)

var controlPlanePanelRowsTimeout = 500 * time.Millisecond
var controlPlanePanelRowsCacheTTL = 750 * time.Millisecond

type controlPlanePanelRowsCacheState struct {
	mu      sync.Mutex
	rows    []agentesapp.Row
	expires time.Time
}

var controlPlanePanelRowsCache controlPlanePanelRowsCacheState

func buildPanelRowsForControlPlane() ([]agentesapp.Row, error) {
	if rows, ok := controlPlanePanelRowsCached(); ok {
		return rows, nil
	}
	return refreshPanelRowsForControlPlane()
}

func refreshPanelRowsForControlPlane() ([]agentesapp.Row, error) {
	if controlPlanePanelRowsTimeout <= 0 {
		rows, err := agentesService.BuildPanelRows()
		if err != nil {
			return nil, err
		}
		storeControlPlanePanelRowsCache(rows)
		return rows, nil
	}
	rows, err := agentRowsForStatusWithinTimeout(controlPlanePanelRowsTimeout)
	if err != nil {
		return nil, err
	}
	storeControlPlanePanelRowsCache(rows)
	return rows, nil
}

func controlPlanePanelRowsCached() ([]agentesapp.Row, bool) {
	if controlPlanePanelRowsCacheTTL <= 0 {
		return nil, false
	}
	now := time.Now()
	controlPlanePanelRowsCache.mu.Lock()
	defer controlPlanePanelRowsCache.mu.Unlock()
	if controlPlanePanelRowsCache.expires.IsZero() || now.After(controlPlanePanelRowsCache.expires) || len(controlPlanePanelRowsCache.rows) == 0 {
		return nil, false
	}
	rows := append([]agentesapp.Row(nil), controlPlanePanelRowsCache.rows...)
	return rows, true
}

func storeControlPlanePanelRowsCache(rows []agentesapp.Row) {
	controlPlanePanelRowsCache.mu.Lock()
	defer controlPlanePanelRowsCache.mu.Unlock()
	controlPlanePanelRowsCache.rows = append([]agentesapp.Row(nil), rows...)
	if controlPlanePanelRowsCacheTTL > 0 {
		controlPlanePanelRowsCache.expires = time.Now().Add(controlPlanePanelRowsCacheTTL)
	} else {
		controlPlanePanelRowsCache.expires = time.Time{}
	}
}

func resetControlPlanePanelRowsCache() {
	controlPlanePanelRowsCache.mu.Lock()
	defer controlPlanePanelRowsCache.mu.Unlock()
	controlPlanePanelRowsCache.rows = nil
	controlPlanePanelRowsCache.expires = time.Time{}
}

func controlPlaneRowsTimedOut(err error) bool {
	return errors.Is(err, errStatusFetchTimeout)
}
