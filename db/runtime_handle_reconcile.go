package db

import (
	"encoding/json"
	"orquesta/runtimeagente"
	"orquesta/runtimepolicy"
	"strings"
)

func runtimeHandleEsTMUXCanonico(handle *RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	meta := mapFromJSON(handle.MetadataJSON)
	return runtimepolicy.RuntimeHandleEsTMUXCanonico(meta, handle.Transporte)
}

func runtimeHandleTMUXSessionRef(handle *RuntimeHandle) string {
	if handle == nil {
		return ""
	}
	meta := mapFromJSON(handle.MetadataJSON)
	return runtimepolicy.RuntimeHandleTMUXSessionRef(meta, handle.Transporte, handle.HandleKind, handle.HandleRef)
}

func runtimeHandleTMUXSessionRefObserved(handle *RuntimeHandle) string {
	if handle == nil {
		return ""
	}
	if ref := runtimeHandleTMUXSessionRef(handle); ref != "" {
		return ref
	}
	snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(handle.MetadataJSON)
	if err != nil || snap == nil {
		return ""
	}
	if ref := strings.TrimSpace(snap.RuntimeRef()); ref != "" {
		return ref
	}
	return strings.TrimSpace(snap.SessionRef())
}

func reconciliarTransporteSesion(existente *RuntimeHandle, inferido *RuntimeHandle) string {
	if inferido == nil {
		if existente == nil {
			return ""
		}
		return strings.TrimSpace(existente.Transporte)
	}
	inferidoTransporte := strings.TrimSpace(inferido.Transporte)
	if existente == nil {
		return inferidoTransporte
	}
	if runtimeHandleEsTMUXCanonico(existente) {
		return "tmux"
	}
	if inferidoTransporte != "" {
		return inferidoTransporte
	}
	return strings.TrimSpace(existente.Transporte)
}

func reconciliarHandleKindSesion(existente *RuntimeHandle, inferido *RuntimeHandle) string {
	if inferido == nil {
		if existente == nil {
			return ""
		}
		return strings.TrimSpace(existente.HandleKind)
	}
	inferidoKind := strings.TrimSpace(inferido.HandleKind)
	if existente == nil {
		return inferidoKind
	}
	if runtimeHandleEsTMUXCanonico(existente) {
		return "session"
	}
	existenteKind := strings.TrimSpace(existente.HandleKind)
	if inferidoKind == "process" {
		return "process"
	}
	if existenteKind != "" {
		return existenteKind
	}
	return inferidoKind
}

func reconciliarHandleRefSesion(existente *RuntimeHandle, inferido *RuntimeHandle) string {
	if inferido == nil {
		if existente == nil {
			return ""
		}
		return strings.TrimSpace(existente.HandleRef)
	}
	inferidoRef := strings.TrimSpace(inferido.HandleRef)
	if existente == nil {
		return inferidoRef
	}
	if runtimeHandleEsTMUXCanonico(existente) {
		if tmuxRef := runtimeHandleTMUXSessionRef(existente); tmuxRef != "" {
			return tmuxRef
		}
	}
	if strings.TrimSpace(inferido.HandleKind) == "process" && inferidoRef != "" {
		return inferidoRef
	}
	if strings.TrimSpace(inferido.HandleKind) == "session" && strings.HasPrefix(inferidoRef, "runtime:") {
		return inferidoRef
	}
	if existenteRef := strings.TrimSpace(existente.HandleRef); existenteRef != "" {
		return existenteRef
	}
	return inferidoRef
}

func reconciliarMetadataSesion(existenteJSON, inferidoJSON string) string {
	existente := mapFromJSON(existenteJSON)
	if existente == nil {
		existente = map[string]any{}
	}
	inferido := mapFromJSON(inferidoJSON)
	if inferido == nil {
		inferido = map[string]any{}
	}
	for _, key := range []string{"herramienta", "external_session_id", "branch", "cwd"} {
		if val := strings.TrimSpace(stringFromMap(inferido, key, "")); val != "" {
			existente[key] = val
		}
	}
	if cwd := strings.TrimSpace(stringFromMap(inferido, "cwd", "")); cwd != "" {
		existente["working_dir"] = cwd
	}
	if workingDir := strings.TrimSpace(stringFromMap(inferido, "working_dir", "")); workingDir != "" {
		existente["working_dir"] = workingDir
	}
	data, _ := json.Marshal(existente)
	return string(data)
}

func reconciliarCapabilitiesSesion(existenteJSON, inferidoJSON string) string {
	existente := mapFromJSON(existenteJSON)
	if existente == nil {
		existente = map[string]any{}
	}
	inferido := mapFromJSON(inferidoJSON)
	if inferido == nil {
		inferido = map[string]any{}
	}
	for key, value := range inferido {
		if _, ok := existente[key]; ok {
			continue
		}
		existente[key] = value
	}
	data, _ := json.Marshal(existente)
	return string(data)
}
