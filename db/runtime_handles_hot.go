/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import (
	"strconv"
	"strings"
	"sync"
	"time"
)

type runtimeHandleHotState struct {
	mu             sync.RWMutex
	loaded         bool
	byAgent        map[string]*RuntimeHandle
	byAgentProject map[string]*RuntimeHandle
	bySession      map[int64]*RuntimeHandle
	operational    map[string]*RuntimeHandle
}

var runtimeHandlesHot = runtimeHandleHotState{
	byAgent:        map[string]*RuntimeHandle{},
	byAgentProject: map[string]*RuntimeHandle{},
	bySession:      map[int64]*RuntimeHandle{},
	operational:    map[string]*RuntimeHandle{},
}

func runtimeHandleHotReset() {
	runtimeHandlesHot.mu.Lock()
	defer runtimeHandlesHot.mu.Unlock()
	runtimeHandlesHot.loaded = false
	runtimeHandlesHot.byAgent = map[string]*RuntimeHandle{}
	runtimeHandlesHot.byAgentProject = map[string]*RuntimeHandle{}
	runtimeHandlesHot.bySession = map[int64]*RuntimeHandle{}
	runtimeHandlesHot.operational = map[string]*RuntimeHandle{}
}

func ResetRuntimeHandlesHotCache() {
	runtimeHandleHotReset()
}

func runtimeHandleHotResetOnSuccess(err error) error {
	if err == nil {
		runtimeHandleHotReset()
	}
	return err
}

func runtimeHandleHotRemember(handle *RuntimeHandle) {
	if handle == nil {
		return
	}
	estado := strings.TrimSpace(handle.Estado)
	if estado != "activo" && estado != "pausado" {
		return
	}
	agente := strings.ToLower(strings.TrimSpace(handle.Agente))
	if agente == "" {
		return
	}
	clone := cloneRuntimeHandle(handle)
	runtimeHandlesHot.mu.Lock()
	defer runtimeHandlesHot.mu.Unlock()
	if current := runtimeHandlesHot.byAgent[agente]; current == nil || runtimeHandlePreferible(clone, current, time.Now().UTC()) {
		runtimeHandlesHot.byAgent[agente] = clone
	}
	if handle.SesionID != nil && *handle.SesionID > 0 {
		if current := runtimeHandlesHot.bySession[*handle.SesionID]; current == nil || runtimeHandlePreferible(clone, current, time.Now().UTC()) {
			runtimeHandlesHot.bySession[*handle.SesionID] = clone
		}
	}
	if key, ok := runtimeHandleAgentProjectKey(agente, handle.ProyectoID); ok {
		if current := runtimeHandlesHot.byAgentProject[key]; current == nil || runtimeHandlePreferible(clone, current, time.Now().UTC()) {
			runtimeHandlesHot.byAgentProject[key] = clone
		}
		if runtimeHandleSostieneSesionOperativaConCutoff(clone, sessionOperationalCutoff()) {
			if current := runtimeHandlesHot.operational[key]; current == nil || runtimeHandlePreferible(clone, current, time.Now().UTC()) {
				runtimeHandlesHot.operational[key] = clone
			}
		}
	}
}

func runtimeHandleHotLookup(agente string, proyectoID *int64) *RuntimeHandle {
	if err := runtimeHandleHotEnsureLoaded(); err != nil {
		return nil
	}
	agente = strings.ToLower(strings.TrimSpace(agente))
	if agente == "" {
		return nil
	}
	runtimeHandlesHot.mu.RLock()
	defer runtimeHandlesHot.mu.RUnlock()
	if proyectoID != nil {
		if key, ok := runtimeHandleAgentProjectKey(agente, proyectoID); ok {
			return cloneRuntimeHandle(runtimeHandlesHot.byAgentProject[key])
		}
		return nil
	}
	return cloneRuntimeHandle(runtimeHandlesHot.byAgent[agente])
}

func runtimeHandleHotLookupBySessionID(sesionID int64) *RuntimeHandle {
	if sesionID <= 0 {
		return nil
	}
	if err := runtimeHandleHotEnsureLoaded(); err != nil {
		return nil
	}
	runtimeHandlesHot.mu.RLock()
	defer runtimeHandlesHot.mu.RUnlock()
	return cloneRuntimeHandle(runtimeHandlesHot.bySession[sesionID])
}

func runtimeHandleHotSnapshotOperativo() (map[string]*RuntimeHandle, error) {
	if err := runtimeHandleHotEnsureLoaded(); err != nil {
		return nil, err
	}
	runtimeHandlesHot.mu.RLock()
	defer runtimeHandlesHot.mu.RUnlock()
	out := make(map[string]*RuntimeHandle, len(runtimeHandlesHot.operational))
	for key, handle := range runtimeHandlesHot.operational {
		out[key] = cloneRuntimeHandle(handle)
	}
	return out, nil
}

func runtimeHandleHotSnapshotCanonico() (map[string]*RuntimeHandle, error) {
	if err := runtimeHandleHotEnsureLoaded(); err != nil {
		return nil, err
	}
	runtimeHandlesHot.mu.RLock()
	defer runtimeHandlesHot.mu.RUnlock()
	out := make(map[string]*RuntimeHandle, len(runtimeHandlesHot.byAgent)+len(runtimeHandlesHot.byAgentProject))
	for key, handle := range runtimeHandlesHot.byAgentProject {
		out[key] = cloneRuntimeHandle(handle)
	}
	for agente, handle := range runtimeHandlesHot.byAgent {
		if handle == nil {
			continue
		}
		if key, ok := runtimeHandleAgentProjectKey(agente, handle.ProyectoID); ok {
			if _, exists := out[key]; exists {
				continue
			}
		}
		out[agente] = cloneRuntimeHandle(handle)
	}
	return out, nil
}

func runtimeHandleHotEnsureLoaded() error {
	runtimeHandlesHot.mu.RLock()
	if runtimeHandlesHot.loaded {
		runtimeHandlesHot.mu.RUnlock()
		return nil
	}
	runtimeHandlesHot.mu.RUnlock()

	handles, err := runtimeHandleHotLoadCandidates()
	if err != nil {
		return err
	}
	return runtimeHandleHotReplace(handles)
}

func runtimeHandleHotLoadCandidates() ([]*RuntimeHandle, error) {
	return consultarConReintentos(func() ([]*RuntimeHandle, error) {
		rows, err := DB.Query(runtimeHandleSelectBase() + `
			WHERE estado IN ('activo','pausado')
			ORDER BY COALESCE(last_seen_at, updated_at, created_at) DESC, id DESC`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var handles []*RuntimeHandle
		for rows.Next() {
			handle, err := scanRuntimeHandle(rows)
			if err != nil {
				return nil, err
			}
			handles = append(handles, handle)
		}
		return handles, rows.Err()
	})
}

func runtimeHandleHotReplace(handles []*RuntimeHandle) error {
	cutoff := sessionOperationalCutoff()
	groupedByAgent := map[string][]*RuntimeHandle{}
	groupedByAgentProject := map[string][]*RuntimeHandle{}
	groupedBySession := map[int64][]*RuntimeHandle{}
	operational := map[string]*RuntimeHandle{}

	for _, handle := range handles {
		if handle == nil {
			continue
		}
		agente := strings.ToLower(strings.TrimSpace(handle.Agente))
		if agente == "" {
			continue
		}
		groupedByAgent[agente] = append(groupedByAgent[agente], handle)
		if key, ok := runtimeHandleAgentProjectKey(agente, handle.ProyectoID); ok {
			groupedByAgentProject[key] = append(groupedByAgentProject[key], handle)
			if operational[key] == nil && runtimeHandleSostieneSesionOperativaConCutoff(handle, cutoff) {
				operational[key] = cloneRuntimeHandle(handle)
			}
		}
		if handle.SesionID != nil && *handle.SesionID > 0 {
			groupedBySession[*handle.SesionID] = append(groupedBySession[*handle.SesionID], handle)
		}
	}

	byAgent := map[string]*RuntimeHandle{}
	for agente, grouped := range groupedByAgent {
		if handle := runtimeHandleHotSelectCanonical(grouped); handle != nil {
			byAgent[agente] = cloneRuntimeHandle(handle)
		}
	}
	byAgentProject := map[string]*RuntimeHandle{}
	for key, grouped := range groupedByAgentProject {
		if handle := runtimeHandleHotSelectCanonical(grouped); handle != nil {
			byAgentProject[key] = cloneRuntimeHandle(handle)
			if len(grouped) > 1 {
				if err := cerrarRuntimeHandlesActivosSuperseded(handle, grouped); err != nil {
					return err
				}
			}
		}
	}
	bySession := map[int64]*RuntimeHandle{}
	for sesionID, grouped := range groupedBySession {
		if handle := runtimeHandleHotSelectSession(grouped); handle != nil {
			bySession[sesionID] = cloneRuntimeHandle(handle)
		}
	}

	runtimeHandlesHot.mu.Lock()
	defer runtimeHandlesHot.mu.Unlock()
	runtimeHandlesHot.loaded = true
	runtimeHandlesHot.byAgent = byAgent
	runtimeHandlesHot.byAgentProject = byAgentProject
	runtimeHandlesHot.bySession = bySession
	runtimeHandlesHot.operational = operational
	return nil
}

func runtimeHandleHotSelectCanonical(handles []*RuntimeHandle) *RuntimeHandle {
	canonicos := make([]*RuntimeHandle, 0, len(handles))
	for _, handle := range handles {
		if handle == nil || runtimeHandleExcluidoDelActivoCanonico(handle) {
			continue
		}
		canonicos = append(canonicos, handle)
	}
	if len(canonicos) == 0 {
		return nil
	}
	return elegirRuntimeHandleActivoPreferente(canonicos)
}

func runtimeHandleHotSelectSession(handles []*RuntimeHandle) *RuntimeHandle {
	if len(handles) == 0 {
		return nil
	}
	return elegirRuntimeHandleActivoPreferente(handles)
}

func runtimeHandleAgentProjectKey(agente string, proyectoID *int64) (string, bool) {
	if proyectoID == nil {
		return "", false
	}
	agente = strings.ToLower(strings.TrimSpace(agente))
	if agente == "" {
		return "", false
	}
	return agente + "#" + strconv.FormatInt(*proyectoID, 10), true
}

func cloneRuntimeHandle(handle *RuntimeHandle) *RuntimeHandle {
	if handle == nil {
		return nil
	}
	cp := *handle
	cp.ProyectoID = cloneInt64Ptr(handle.ProyectoID)
	cp.SesionID = cloneInt64Ptr(handle.SesionID)
	cp.RuntimeID = cloneInt64Ptr(handle.RuntimeID)
	cp.LastSeenAt = cloneTimePtr(handle.LastSeenAt)
	return &cp
}

func cloneInt64Ptr(v *int64) *int64 {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}

func cloneTimePtr(v *time.Time) *time.Time {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}
