package db

import "strings"

type EventoHook string

const (
	HookProjectBlocked   EventoHook = "project_blocked"
	HookProjectUnblocked EventoHook = "project_unblocked"
	HookSessionPark      EventoHook = "session_park"
	HookSessionResume    EventoHook = "session_resume"
	HookTaskStart        EventoHook = "task_start"
	HookTaskFinish       EventoHook = "task_finish"
)

func EmitirHookCicloVida(actor string, evento EventoHook, proyectoID int64, entidad string, entidadID int64, agente string, detalle string) {
	nombreEvento := strings.TrimSpace(string(evento))
	if nombreEvento == "" {
		return
	}
	Audit(strings.TrimSpace(actor), "hook_"+nombreEvento, strings.TrimSpace(entidad), entidadID, strings.TrimSpace(detalle))
	select {
	case CanalNotificaciones <- EventoNotificacion{
		Tipo:       "hook:" + nombreEvento,
		ID:         entidadID,
		Agente:     strings.TrimSpace(agente),
		Texto:      strings.TrimSpace(detalle),
		ProyectoID: proyectoID,
	}:
	default:
	}
}
