/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import (
	"database/sql"
	"sort"
	"strings"
	"sync"
	"time"
)

var runtimeOrdersHotIndex = runtimeOrderHotIndexState{
	byID:           map[int64]*RuntimeOrder{},
	byState:        map[string]map[int64]struct{}{},
	byAgent:        map[string]map[int64]struct{}{},
	byAgentProject: map[string]map[int64]struct{}{},
}

type runtimeOrderHotIndexState struct {
	mu             sync.RWMutex
	loaded         bool
	dirty          bool
	byID           map[int64]*RuntimeOrder
	byState        map[string]map[int64]struct{}
	byAgent        map[string]map[int64]struct{}
	byAgentProject map[string]map[int64]struct{}
}

func runtimeOrderEstadoVivo(estado string) bool {
	switch strings.TrimSpace(estado) {
	case "pendiente", "tomada", "ejecutando":
		return true
	default:
		return false
	}
}

func runtimeOrdersHotIndexMutatingQuery(query string) bool {
	raw := strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(query))), " ")
	if raw == "" || !strings.Contains(raw, "runtime_orders") {
		return false
	}
	return strings.Contains(raw, "insert into runtime_orders") ||
		strings.Contains(raw, "update runtime_orders") ||
		strings.Contains(raw, "delete from runtime_orders")
}

func markRuntimeOrdersHotIndexDirty() {
	runtimeOrdersHotIndex.mu.Lock()
	defer runtimeOrdersHotIndex.mu.Unlock()
	runtimeOrdersHotIndex.dirty = true
}

func resetRuntimeOrdersHotIndex() {
	runtimeOrdersHotIndex.mu.Lock()
	defer runtimeOrdersHotIndex.mu.Unlock()
	runtimeOrdersHotIndex.loaded = false
	runtimeOrdersHotIndex.dirty = false
	runtimeOrdersHotIndex.byID = map[int64]*RuntimeOrder{}
	runtimeOrdersHotIndex.byState = map[string]map[int64]struct{}{}
	runtimeOrdersHotIndex.byAgent = map[string]map[int64]struct{}{}
	runtimeOrdersHotIndex.byAgentProject = map[string]map[int64]struct{}{}
}

func runtimeOrdersHotIndexEnsureLoaded() error {
	runtimeOrdersHotIndex.mu.RLock()
	if runtimeOrdersHotIndex.loaded && !runtimeOrdersHotIndex.dirty {
		runtimeOrdersHotIndex.mu.RUnlock()
		return nil
	}
	runtimeOrdersHotIndex.mu.RUnlock()

	orders, err := listarRuntimeOrdersVivasDesdeDB()
	if err != nil {
		return err
	}
	runtimeOrdersHotIndexReplace(orders)
	return nil
}

func runtimeOrdersHotIndexReplace(orders []*RuntimeOrder) {
	byID := make(map[int64]*RuntimeOrder, len(orders))
	byState := map[string]map[int64]struct{}{}
	byAgent := map[string]map[int64]struct{}{}
	byAgentProject := map[string]map[int64]struct{}{}

	for _, order := range orders {
		if order == nil || !runtimeOrderEstadoVivo(order.Estado) {
			continue
		}
		clone := cloneRuntimeOrder(order)
		byID[clone.ID] = clone
		runtimeOrderHotIndexAddID(byState, strings.TrimSpace(clone.Estado), clone.ID)
		runtimeOrderHotIndexAddID(byAgent, strings.TrimSpace(clone.Agente), clone.ID)
		runtimeOrderHotIndexAddID(byAgentProject, runtimeOrderHotAgentProjectKey(clone.Agente, clone.ProyectoID), clone.ID)
	}

	runtimeOrdersHotIndex.mu.Lock()
	defer runtimeOrdersHotIndex.mu.Unlock()
	runtimeOrdersHotIndex.loaded = true
	runtimeOrdersHotIndex.dirty = false
	runtimeOrdersHotIndex.byID = byID
	runtimeOrdersHotIndex.byState = byState
	runtimeOrdersHotIndex.byAgent = byAgent
	runtimeOrdersHotIndex.byAgentProject = byAgentProject
}

func runtimeOrdersHotIndexSyncByID(id int64) error {
	if id <= 0 {
		return nil
	}
	runtimeOrdersHotIndex.mu.RLock()
	loaded := runtimeOrdersHotIndex.loaded
	dirty := runtimeOrdersHotIndex.dirty
	runtimeOrdersHotIndex.mu.RUnlock()
	if !loaded {
		return nil
	}
	if dirty {
		return runtimeOrdersHotIndexEnsureLoaded()
	}

	order, err := GetRuntimeOrder(id)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	runtimeOrdersHotIndex.mu.Lock()
	if !runtimeOrdersHotIndex.loaded {
		runtimeOrdersHotIndex.mu.Unlock()
		return nil
	}
	if runtimeOrdersHotIndex.dirty {
		runtimeOrdersHotIndex.mu.Unlock()
		return runtimeOrdersHotIndexEnsureLoaded()
	}
	if order == nil {
		runtimeOrdersHotIndexRemoveLocked(id)
	} else {
		runtimeOrdersHotIndexApplyLocked(order)
	}
	runtimeOrdersHotIndex.dirty = false
	runtimeOrdersHotIndex.mu.Unlock()
	return nil
}

func runtimeOrdersHotIndexSyncIDs(ids ...int64) error {
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		if err := runtimeOrdersHotIndexSyncByID(id); err != nil {
			return err
		}
	}
	return nil
}

func runtimeOrdersHotIndexApplyLocked(order *RuntimeOrder) {
	id := int64(0)
	if order != nil {
		id = order.ID
	}
	if id > 0 {
		runtimeOrdersHotIndexRemoveLocked(id)
	}
	if order == nil || !runtimeOrderEstadoVivo(order.Estado) {
		return
	}
	clone := cloneRuntimeOrder(order)
	runtimeOrdersHotIndex.byID[clone.ID] = clone
	runtimeOrderHotIndexAddID(runtimeOrdersHotIndex.byState, strings.TrimSpace(clone.Estado), clone.ID)
	runtimeOrderHotIndexAddID(runtimeOrdersHotIndex.byAgent, strings.TrimSpace(clone.Agente), clone.ID)
	runtimeOrderHotIndexAddID(runtimeOrdersHotIndex.byAgentProject, runtimeOrderHotAgentProjectKey(clone.Agente, clone.ProyectoID), clone.ID)
}

func runtimeOrdersHotIndexRemoveLocked(id int64) {
	if id <= 0 {
		return
	}
	current := runtimeOrdersHotIndex.byID[id]
	if current == nil {
		delete(runtimeOrdersHotIndex.byID, id)
		return
	}
	runtimeOrderHotIndexRemoveID(runtimeOrdersHotIndex.byState, strings.TrimSpace(current.Estado), id)
	runtimeOrderHotIndexRemoveID(runtimeOrdersHotIndex.byAgent, strings.TrimSpace(current.Agente), id)
	runtimeOrderHotIndexRemoveID(runtimeOrdersHotIndex.byAgentProject, runtimeOrderHotAgentProjectKey(current.Agente, current.ProyectoID), id)
	delete(runtimeOrdersHotIndex.byID, id)
}

func runtimeOrderHotIndexAddID(index map[string]map[int64]struct{}, key string, id int64) {
	key = strings.TrimSpace(key)
	if key == "" || id <= 0 {
		return
	}
	if index[key] == nil {
		index[key] = map[int64]struct{}{}
	}
	index[key][id] = struct{}{}
}

func runtimeOrderHotIndexRemoveID(index map[string]map[int64]struct{}, key string, id int64) {
	key = strings.TrimSpace(key)
	if key == "" || id <= 0 {
		return
	}
	items := index[key]
	if len(items) == 0 {
		return
	}
	delete(items, id)
	if len(items) == 0 {
		delete(index, key)
	}
}

func runtimeOrderHotAgentProjectKey(agente string, proyectoID *int64) string {
	agente = strings.TrimSpace(agente)
	if proyectoID == nil || *proyectoID <= 0 {
		return agente + "|-"
	}
	return agente + "|" + jsonNumber(*proyectoID)
}

func listarRuntimeOrdersVivasDesdeDB() ([]*RuntimeOrder, error) {
	if DB == nil || DB.DB == nil {
		return []*RuntimeOrder{}, nil
	}
	rows, err := DB.Query(runtimeOrderSelectBase() + `
		WHERE estado IN ('pendiente','tomada','ejecutando')
		ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*RuntimeOrder
	for rows.Next() {
		order, err := scanRuntimeOrder(rows)
		if err != nil {
			return nil, err
		}
		if order != nil {
			out = append(out, order)
		}
	}
	return out, rows.Err()
}

func ListarRuntimeOrdersVivas(filter FiltroRuntimeOrders) ([]*RuntimeOrder, error) {
	if filter.Estado != nil && !runtimeOrderEstadoVivo(*filter.Estado) {
		return []*RuntimeOrder{}, nil
	}
	if err := runtimeOrdersHotIndexEnsureLoaded(); err != nil {
		return nil, err
	}

	runtimeOrdersHotIndex.mu.RLock()
	defer runtimeOrdersHotIndex.mu.RUnlock()

	candidates := runtimeOrdersHotIndexCandidatesLocked(filter)
	out := make([]*RuntimeOrder, 0, len(candidates))
	for _, order := range candidates {
		if order == nil || !runtimeOrderMatchesLiveFilter(order, filter) {
			continue
		}
		out = append(out, cloneRuntimeOrder(order))
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].ID > out[j].ID
	})
	if filter.Limit > 0 && len(out) > filter.Limit {
		out = out[:filter.Limit]
	}
	return out, nil
}

func runtimeOrdersHotIndexCandidatesLocked(filter FiltroRuntimeOrders) []*RuntimeOrder {
	switch {
	case filter.Agente != nil && filter.ProyectoID != nil:
		return runtimeOrdersFromIDSetLocked(runtimeOrdersHotIndex.byAgentProject[runtimeOrderHotAgentProjectKey(*filter.Agente, filter.ProyectoID)])
	case filter.Agente != nil:
		return runtimeOrdersFromIDSetLocked(runtimeOrdersHotIndex.byAgent[strings.TrimSpace(*filter.Agente)])
	case filter.Estado != nil:
		return runtimeOrdersFromIDSetLocked(runtimeOrdersHotIndex.byState[strings.TrimSpace(*filter.Estado)])
	default:
		out := make([]*RuntimeOrder, 0, len(runtimeOrdersHotIndex.byID))
		for _, order := range runtimeOrdersHotIndex.byID {
			out = append(out, order)
		}
		return out
	}
}

func runtimeOrdersFromIDSetLocked(ids map[int64]struct{}) []*RuntimeOrder {
	if len(ids) == 0 {
		return nil
	}
	out := make([]*RuntimeOrder, 0, len(ids))
	for id := range ids {
		if order := runtimeOrdersHotIndex.byID[id]; order != nil {
			out = append(out, order)
		}
	}
	return out
}

func runtimeOrderMatchesLiveFilter(order *RuntimeOrder, filter FiltroRuntimeOrders) bool {
	if order == nil || !runtimeOrderEstadoVivo(order.Estado) {
		return false
	}
	if filter.Agente != nil && strings.TrimSpace(order.Agente) != strings.TrimSpace(*filter.Agente) {
		return false
	}
	if filter.ProyectoID != nil {
		if order.ProyectoID == nil || *order.ProyectoID != *filter.ProyectoID {
			return false
		}
	}
	if filter.Estado != nil && strings.TrimSpace(order.Estado) != strings.TrimSpace(*filter.Estado) {
		return false
	}
	return true
}

func cloneRuntimeOrder(order *RuntimeOrder) *RuntimeOrder {
	if order == nil {
		return nil
	}
	clone := *order
	if order.ProyectoID != nil {
		proyectoID := *order.ProyectoID
		clone.ProyectoID = &proyectoID
	}
	if order.RuntimeID != nil {
		runtimeID := *order.RuntimeID
		clone.RuntimeID = &runtimeID
	}
	if order.HandleID != nil {
		handleID := *order.HandleID
		clone.HandleID = &handleID
	}
	if order.LeaseExpiresAt != nil {
		leaseExpiresAt := *order.LeaseExpiresAt
		clone.LeaseExpiresAt = &leaseExpiresAt
	}
	if order.StartedAt != nil {
		startedAt := *order.StartedAt
		clone.StartedAt = &startedAt
	}
	if order.FinishedAt != nil {
		finishedAt := *order.FinishedAt
		clone.FinishedAt = &finishedAt
	}
	return &clone
}

func runtimeOrderPendingReady(order *RuntimeOrder, now time.Time) bool {
	if order == nil {
		return false
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return !order.AvailableAt.After(now.UTC())
}

func runtimeOrderBootstrapPriority(order *RuntimeOrder, proyectoID *int64) (int, int) {
	projectPriority := 1
	if proyectoID == nil {
		if order != nil && order.ProyectoID == nil {
			projectPriority = 0
		}
	} else {
		switch {
		case order != nil && order.ProyectoID != nil && *order.ProyectoID == *proyectoID:
			projectPriority = 0
		case order != nil && order.ProyectoID == nil:
			projectPriority = 1
		default:
			projectPriority = 2
		}
	}
	typePriority := 9
	if order != nil {
		switch strings.TrimSpace(order.Tipo) {
		case "handoff":
			typePriority = 0
		case "resume":
			typePriority = 1
		case "start":
			typePriority = 2
		}
	}
	return projectPriority, typePriority
}

func runtimeOrderAttemptReference(order *RuntimeOrder) time.Time {
	if order == nil {
		return time.Time{}
	}
	if order.StartedAt != nil && !order.StartedAt.IsZero() {
		return order.StartedAt.UTC()
	}
	if !order.UpdatedAt.IsZero() {
		return order.UpdatedAt.UTC()
	}
	return order.CreatedAt.UTC()
}

func stringsPtrTrimmed(v string) *string {
	v = strings.TrimSpace(v)
	return &v
}
