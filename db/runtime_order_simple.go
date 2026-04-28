package db

import (
	"encoding/json"
	"fmt"
	"strings"
)

func ejecutarRuntimeOrderCheckpoint(order *RuntimeOrder) error {
	sesion, err := resolverSesionParaOrden(order)
	if err != nil {
		return err
	}
	if sesion == nil {
		return fmt.Errorf("no existe sesión activa ni reciente para %s", order.Agente)
	}

	runtime, err := resolverRuntimeParaOrden(order)
	if err != nil {
		return err
	}

	payload := map[string]any{}
	_ = json.Unmarshal([]byte(order.PayloadJSON), &payload)
	checkpointKind := stringFromMap(payload, "checkpoint_kind", "manual")
	resumen := stringFromMap(payload, "resumen", sesion.ResumenContinuidad)
	resumeStrategy := stringFromMap(payload, "resume_strategy", "resumen_y_payload")
	source := fmt.Sprintf("runtime_order:%d", order.ID)

	existente, err := GetRuntimeCheckpointBySource(source)
	if err != nil {
		return err
	}
	if existente != nil {
		result := map[string]any{
			"ok":            true,
			"checkpoint_id": existente.ID,
			"sesion_id":     sesion.ID,
			"reused":        true,
		}
		data, _ := json.Marshal(result)
		return MarcarRuntimeOrderEstado(order.ID, "completada", string(data), "")
	}

	cp := &RuntimeCheckpoint{
		Agente:         order.Agente,
		ProyectoID:     order.ProyectoID,
		SesionID:       &sesion.ID,
		CheckpointKind: checkpointKind,
		Resumen:        resumen,
		Branch:         sesion.Branch,
		CWD:            sesion.CWD,
		PayloadJSON:    order.PayloadJSON,
		ResumeStrategy: resumeStrategy,
		Source:         source,
	}
	if runtime != nil {
		cp.RuntimeID = &runtime.ID
	}
	id, err := CrearRuntimeCheckpoint(cp)
	if err != nil {
		return err
	}
	result := map[string]any{
		"ok":            true,
		"checkpoint_id": id,
		"sesion_id":     sesion.ID,
	}
	data, _ := json.Marshal(result)
	return MarcarRuntimeOrderEstado(order.ID, "completada", string(data), "")
}

func ejecutarRuntimeOrderNudge(order *RuntimeOrder) error {
	return ejecutarRuntimeOrderMailboxSimple(order, "nudge", "nudge sin agente destino")
}

func ejecutarRuntimeOrderDiscordia(order *RuntimeOrder) error {
	return ejecutarRuntimeOrderMailboxSimple(order, "discordia", "discordia sin supervisor destino")
}

func ejecutarRuntimeOrderMailboxSimple(order *RuntimeOrder, defaultKind, missingTargetError string) error {
	payload := map[string]any{}
	_ = json.Unmarshal([]byte(order.PayloadJSON), &payload)

	toAgente := stringFromMap(payload, "to_agente", order.Agente)
	fromAgente := stringFromMap(payload, "from_agente", "server")
	kind := stringFromMap(payload, "kind", defaultKind)
	if strings.TrimSpace(toAgente) == "" {
		return fmt.Errorf("%s", missingTargetError)
	}

	existente, err := GetRuntimeMailboxByRuntimeOrderID(order.ID)
	if err != nil {
		return err
	}
	if existente != nil {
		result := map[string]any{
			"ok":          true,
			"mailbox_id":  existente.ID,
			"to_agente":   existente.ToAgente,
			"from_agente": existente.FromAgente,
			"kind":        existente.Kind,
			"reused":      true,
		}
		data, _ := json.Marshal(result)
		return MarcarRuntimeOrderEstado(order.ID, "completada", string(data), "")
	}

	id, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:     fromAgente,
		ToAgente:       toAgente,
		ProyectoID:     order.ProyectoID,
		RuntimeOrderID: &order.ID,
		Kind:           kind,
		PayloadJSON:    order.PayloadJSON,
	})
	if err != nil {
		return err
	}

	result := map[string]any{
		"ok":          true,
		"mailbox_id":  id,
		"to_agente":   toAgente,
		"from_agente": fromAgente,
		"kind":        kind,
	}
	data, _ := json.Marshal(result)
	return MarcarRuntimeOrderEstado(order.ID, "completada", string(data), "")
}
