package db

import (
	"encoding/json"
	"fmt"
	"orquesta/internal/controlruntime"
	"orquesta/runtimeagente"
	"orquesta/runtimepolicy"
	"strings"
)

func runtimeOrderBloqueaFallbackPID(order *RuntimeOrder, runtime *RuntimeInstance, handle *RuntimeHandle) bool {
	if handle != nil {
		meta := mapFromJSON(handle.MetadataJSON)
		driver, transport := runtimeHandleDriverTransportObserved(handle)
		if strings.EqualFold(transport, "tmux") ||
			strings.EqualFold(driver, "tmux_cli_session") ||
			runtimepolicy.RuntimeHandleUsaLegacyCLITMUXPreferred(meta) {
			return true
		}
	}
	if runtime != nil && runtimepolicy.RuntimeOrderUsaCLITMUXPreferred(runtime.Connector) {
		return true
	}
	sesion, err := resolverSesionParaOrden(order)
	if err == nil && sesion != nil && runtimepolicy.RuntimeOrderUsaCLITMUXPreferred(connectorDesdeSesion(sesion), sesion.Herramienta) {
		return true
	}
	return false
}

func runtimeHandleBloqueaFallbackPID(handle *RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	driver, transport := runtimeHandleDriverTransportObserved(handle)
	meta := mapFromJSON(handle.MetadataJSON)
	if strings.EqualFold(transport, "tmux") ||
		strings.EqualFold(strings.TrimSpace(handle.HandleKind), "session") ||
		strings.EqualFold(driver, "tmux_cli_session") ||
		runtimepolicy.RuntimeHandleUsaLegacyProcessPTY(meta) {
		return true
	}
	return false
}

func objetivoProcesoDesdeHandleRuntimeOrden(order *RuntimeOrder, handle *RuntimeHandle, runtime *RuntimeInstance) controlruntime.ObjetivoProceso {
	obj := controlruntime.ObjetivoProceso{}
	blockedPID := runtimeHandleBloqueaFallbackPID(handle)
	if handle != nil {
		driver, transport := runtimeHandleDriverTransportObserved(handle)
		if strings.EqualFold(transport, "tmux") || strings.EqualFold(driver, "tmux_cli_session") {
			obj.HandleKind = "session"
			obj.HandleRef = runtimeHandleTMUXSessionRefObserved(handle)
		} else {
			obj.HandleKind = handle.HandleKind
			obj.HandleRef = handle.HandleRef
		}
		obj.MetadataJSON = handle.MetadataJSON
	}
	if runtime != nil &&
		runtime.PID != nil &&
		*runtime.PID > 0 &&
		!blockedPID &&
		!runtimeOrderBloqueaFallbackPID(order, runtime, handle) &&
		(handle == nil || strings.TrimSpace(obj.HandleKind) == "" || strings.TrimSpace(obj.HandleKind) == "process") {
		obj.PID = runtime.PID
	}
	return obj
}

func objetivoProcesoDesdeHandleRuntime(handle *RuntimeHandle, runtime *RuntimeInstance) controlruntime.ObjetivoProceso {
	return objetivoProcesoDesdeHandleRuntimeOrden(nil, handle, runtime)
}

func runtimeHandleDriverTransportObserved(handle *RuntimeHandle) (string, string) {
	if handle == nil {
		return "", ""
	}
	meta := mapFromJSON(handle.MetadataJSON)
	driver := strings.TrimSpace(runtimepolicy.RuntimeHandleDriver(handle.MetadataJSON))
	transport := strings.TrimSpace(handle.Transporte)
	if transport == "" {
		transport = strings.TrimSpace(stringFromMap(meta, "transport", ""))
	}
	if snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(handle.MetadataJSON); err == nil && snap != nil {
		if value := strings.TrimSpace(snap.Driver()); value != "" {
			driver = value
		}
		if value := strings.TrimSpace(snap.Transport()); value != "" {
			transport = value
		}
	}
	if transport == "" && strings.EqualFold(driver, "tmux_cli_session") {
		transport = "tmux"
	}
	return strings.ToLower(driver), strings.ToLower(transport)
}

func asegurarCheckpointCambioContexto(order *RuntimeOrder, suffix, fallbackSummary string) (int64, error) {
	sesion, err := resolverSesionParaOrden(order)
	if err != nil {
		return 0, err
	}
	if sesion == nil {
		return 0, nil
	}
	runtime, err := resolverRuntimeParaOrden(order)
	if err != nil {
		return 0, err
	}
	payload := map[string]any{}
	_ = json.Unmarshal([]byte(order.PayloadJSON), &payload)
	source := fmt.Sprintf("runtime_order:%d:%s", order.ID, strings.TrimSpace(suffix))
	existente, err := GetRuntimeCheckpointBySource(source)
	if err != nil {
		return 0, err
	}
	if existente != nil {
		return existente.ID, nil
	}
	resumen := stringFromMap(payload, "resumen", "")
	if strings.TrimSpace(resumen) == "" {
		resumen = stringFromMap(payload, "motivo", "")
	}
	if strings.TrimSpace(resumen) == "" {
		resumen = strings.TrimSpace(sesion.ResumenContinuidad)
	}
	if strings.TrimSpace(resumen) == "" {
		resumen = fallbackSummary
	}
	cp := &RuntimeCheckpoint{
		Agente:         order.Agente,
		ProyectoID:     order.ProyectoID,
		SesionID:       &sesion.ID,
		CheckpointKind: suffix,
		Resumen:        resumen,
		Branch:         sesion.Branch,
		CWD:            sesion.CWD,
		PayloadJSON:    order.PayloadJSON,
		ResumeStrategy: "resumen_y_payload",
		Source:         source,
	}
	if runtime != nil {
		cp.RuntimeID = &runtime.ID
	}
	return CrearRuntimeCheckpoint(cp)
}
