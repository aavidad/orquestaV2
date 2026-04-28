package db

import (
	"sort"
	"strings"
	"time"
)

// ListarRuntimeHandlesActivosOperativosRecientes devuelve el ultimo handle
// operativo reciente por agente/proyecto sin revalidar el proceso externo.
// Se usa como hot snapshot en batches frecuentes; las rutas criticas pueden
// seguir cayendo al selector fuerte cuando necesiten validacion exhaustiva.
func ListarRuntimeHandlesActivosOperativosRecientes() (map[string]*RuntimeHandle, error) {
	return runtimeHandleHotSnapshotOperativo()
}

func ListarRuntimeHandlesCanonicosRecientes(agente *string) ([]*RuntimeHandle, error) {
	snapshot, err := runtimeHandleHotSnapshotCanonico()
	if err != nil {
		return nil, err
	}
	filtro := ""
	if agente != nil {
		filtro = strings.ToLower(strings.TrimSpace(*agente))
	}
	seen := make(map[int64]struct{}, len(snapshot))
	out := make([]*RuntimeHandle, 0, len(snapshot))
	for _, handle := range snapshot {
		if handle == nil {
			continue
		}
		if filtro != "" && strings.ToLower(strings.TrimSpace(handle.Agente)) != filtro {
			continue
		}
		if handle.ID > 0 {
			if _, ok := seen[handle.ID]; ok {
				continue
			}
			seen[handle.ID] = struct{}{}
		}
		out = append(out, cloneRuntimeHandle(handle))
	}
	sort.SliceStable(out, func(i, j int) bool {
		return runtimeHandleRecency(out[i]).After(runtimeHandleRecency(out[j]))
	})
	if len(out) > 0 {
		return out, nil
	}
	return listarRuntimeHandlesCanonicosRecientesDesdeDB(agente)
}

func listarRuntimeHandlesCanonicosRecientesDesdeDB(agente *string) ([]*RuntimeHandle, error) {
	handles, err := ListarRuntimeHandles(agente)
	if err != nil {
		return nil, err
	}
	porProyecto := map[string]*RuntimeHandle{}
	porAgente := map[string]*RuntimeHandle{}
	now := time.Now().UTC()
	for _, handle := range handles {
		if handle == nil {
			continue
		}
		agenteKey := strings.ToLower(strings.TrimSpace(handle.Agente))
		if agenteKey == "" {
			continue
		}
		if current := porAgente[agenteKey]; current == nil || runtimeHandlePreferible(handle, current, now) {
			porAgente[agenteKey] = cloneRuntimeHandle(handle)
		}
		if key, ok := runtimeHandleAgentProjectKey(agenteKey, handle.ProyectoID); ok {
			if current := porProyecto[key]; current == nil || runtimeHandlePreferible(handle, current, now) {
				porProyecto[key] = cloneRuntimeHandle(handle)
			}
		}
	}
	seen := map[int64]struct{}{}
	out := make([]*RuntimeHandle, 0, len(porProyecto)+len(porAgente))
	for _, handle := range porProyecto {
		if handle == nil {
			continue
		}
		if handle.ID > 0 {
			seen[handle.ID] = struct{}{}
		}
		out = append(out, cloneRuntimeHandle(handle))
	}
	for _, handle := range porAgente {
		if handle == nil {
			continue
		}
		if handle.ID > 0 {
			if _, ok := seen[handle.ID]; ok {
				continue
			}
		}
		out = append(out, cloneRuntimeHandle(handle))
	}
	sort.SliceStable(out, func(i, j int) bool {
		return runtimeHandleRecency(out[i]).After(runtimeHandleRecency(out[j]))
	})
	return out, nil
}

func RuntimeHandleSnapshotIsFresh(handle *RuntimeHandle, maxAge time.Duration) bool {
	if handle == nil {
		return false
	}
	if maxAge <= 0 {
		maxAge = time.Minute
	}
	lastSeen := handle.CreatedAt
	if handle.UpdatedAt.After(lastSeen) {
		lastSeen = handle.UpdatedAt.UTC()
	}
	if handle.LastSeenAt != nil && handle.LastSeenAt.After(lastSeen) {
		lastSeen = handle.LastSeenAt.UTC()
	}
	return time.Since(lastSeen.UTC()) <= maxAge
}

func runtimeHandleSnapshotLookup(snapshot map[string]*RuntimeHandle, agente string, proyectoID *int64) *RuntimeHandle {
	if len(snapshot) == 0 {
		return nil
	}
	agente = strings.ToLower(strings.TrimSpace(agente))
	if agente == "" {
		return nil
	}
	if proyectoID != nil {
		if key, ok := runtimeHandleAgentProjectKey(agente, proyectoID); ok {
			if handle := snapshot[key]; handle != nil {
				return cloneRuntimeHandle(handle)
			}
		}
	}
	if handle := snapshot[agente]; handle != nil {
		return cloneRuntimeHandle(handle)
	}
	return nil
}

func runtimeHandleCanonicoRecienteConFallback(agente string, proyectoID *int64) (*RuntimeHandle, error) {
	snapshot, err := runtimeHandleHotSnapshotCanonico()
	if err != nil {
		return nil, err
	}
	if handle := runtimeHandleSnapshotLookup(snapshot, agente, proyectoID); handle != nil {
		if runtimeHandlePuedeRepresentarActivoCanonico(handle, time.Now().UTC()) {
			return handle, nil
		}
	}
	if proyectoID != nil {
		return GetRuntimeHandleActivoAgenteProyecto(agente, proyectoID)
	}
	return GetRuntimeHandleActivoAgente(agente)
}

func runtimeHandleOperativoRecienteConFallback(agente string, proyectoID *int64) (*RuntimeHandle, error) {
	var err error
	agente, err = CanonicalizeAgentName(agente)
	if err != nil {
		return nil, err
	}
	snapshot, err := runtimeHandleHotSnapshotOperativo()
	if err != nil {
		return nil, err
	}
	if handle := runtimeHandleSnapshotLookup(snapshot, agente, proyectoID); handle != nil {
		return handle, nil
	}
	if proyectoID != nil {
		return GetRuntimeHandleActivoAgenteProyecto(agente, proyectoID)
	}
	return GetRuntimeHandleActivoAgente(agente)
}

func GetRuntimeHandleCanonicoRecienteAgente(agente string) (*RuntimeHandle, error) {
	return runtimeHandleCanonicoRecienteConFallback(agente, nil)
}

func GetRuntimeHandleCanonicoRecienteAgenteProyecto(agente string, proyectoID *int64) (*RuntimeHandle, error) {
	return runtimeHandleCanonicoRecienteConFallback(agente, proyectoID)
}

func GetRuntimeHandleOperativoRecienteAgente(agente string) (*RuntimeHandle, error) {
	return runtimeHandleOperativoRecienteConFallback(agente, nil)
}

func GetRuntimeHandleOperativoRecienteAgenteProyecto(agente string, proyectoID *int64) (*RuntimeHandle, error) {
	return runtimeHandleOperativoRecienteConFallback(agente, proyectoID)
}

func GetRuntimePrincipalAgente(agente string) (*RuntimeInstance, error) {
	return runtimePrincipalAgenteProyecto(agente, nil)
}

func GetRuntimePrincipalAgenteProyecto(agente string, proyectoID *int64) (*RuntimeInstance, error) {
	return runtimePrincipalAgenteProyecto(agente, proyectoID)
}
