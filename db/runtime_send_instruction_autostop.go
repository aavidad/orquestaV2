package db

import (
	"orquesta/internal/controlruntime"
	"orquesta/runtimeagente"
)

func runtimeOrderSendInstructionAutoStopLocalOllama(order *RuntimeOrder, payload map[string]any) {
	if order == nil || !runtimeOrderSendInstructionEsMicroprogramacion(payload) {
		return
	}
	if !runtimeagente.EsConectorFamiliaOllama(runtimeagente.ConectorPorDefectoAgente(order.Agente), "") {
		return
	}
	runtime, err := resolverRuntimeParaOrden(order)
	if err != nil {
		Audit("orquesta", "runtime_send_instruction_autostop_error", "runtime_order", order.ID, err.Error())
		return
	}
	handle, err := resolverHandleParaOrden(order)
	if err != nil {
		Audit("orquesta", "runtime_send_instruction_autostop_error", "runtime_order", order.ID, err.Error())
		return
	}
	runtime, handle, err = resolverDestinoRuntimeOrderSendInstruction(order, runtime, handle)
	if err != nil {
		Audit("orquesta", "runtime_send_instruction_autostop_error", "runtime_order", order.ID, err.Error())
		return
	}
	if handle == nil {
		return
	}
	obj := objetivoProcesoDesdeHandleRuntime(handle, runtime)
	if _, _, err := controlruntime.DetenerProceso(obj); err != nil && !errorDetenerSesionObsoletaIgnorable(err) {
		Audit("orquesta", "runtime_send_instruction_autostop_error", "runtime_order", order.ID, err.Error())
		return
	}
	if runtime != nil && runtime.ID > 0 {
		if _, err := DB.Exec(`
			UPDATE runtime_instances
			SET logical_state='cerrado',
			    process_state='finalizado',
			    last_event_at=CURRENT_TIMESTAMP
			WHERE id=?`, runtime.ID); err != nil {
			Audit("orquesta", "runtime_send_instruction_autostop_error", "runtime_order", order.ID, err.Error())
			return
		}
	}
	agenteCanonico, err := CanonicalizeAgentName(order.Agente)
	if err != nil {
		Audit("orquesta", "runtime_send_instruction_autostop_error", "runtime_order", order.ID, err.Error())
		return
	}
	if _, err := DB.Exec(`
		UPDATE runtime_handles
		SET estado='cerrado',
		    last_seen_at=CURRENT_TIMESTAMP
		WHERE agente=? AND estado IN ('activo','pausado','fallido')`, agenteCanonico); err != nil {
		Audit("orquesta", "runtime_send_instruction_autostop_error", "runtime_order", order.ID, err.Error())
		return
	}
	runtimeHandleHotReset()
}
