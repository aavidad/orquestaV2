package db

import "strings"

func aparcarSesionActiva(agente string, proyectoID *int64) error {
	var err error
	agente, err = CanonicalizeAgentName(agente)
	if err != nil {
		return err
	}
	q := `UPDATE sesiones SET activa=0, estado='pausada', heartbeat_at=CURRENT_TIMESTAMP WHERE agente=? AND activa=1`
	args := []any{agente}
	if proyectoID != nil {
		q += ` AND proyecto_id = ?`
		args = append(args, *proyectoID)
	}
	if _, err := DB.Exec(q, args...); err != nil {
		return err
	}
	if _, err := DB.Exec(`UPDATE agentes SET activo=0, estado_sesion='disponible' WHERE nombre=?`, agente); err != nil {
		return err
	}
	Audit(agente, "aparcar_sesion", "sesion", 0, "")
	if proyectoID != nil {
		EmitirHookCicloVida(agente, HookSessionPark, *proyectoID, "sesion", 0, agente, "aparcar_sesion")
	}
	return nil
}

func AparcarSesionActiva(agente string, proyectoID *int64) error {
	if err := aparcarSesionActiva(agente, proyectoID); err != nil {
		return err
	}
	agenteCanonico, err := CanonicalizeAgentName(agente)
	if err != nil {
		return err
	}
	return MarcarRuntimeHandlesCerradosPorAgente(agenteCanonico)
}

func actualizarEstadoSesionParaOrden(order *RuntimeOrder, estado string) error {
	sesion, err := resolverSesionParaOrden(order)
	if err != nil {
		return err
	}
	if sesion == nil || sesion.ID == 0 {
		return nil
	}
	if _, err := DB.Exec(`UPDATE sesiones SET estado=? WHERE id=?`, strings.TrimSpace(estado), sesion.ID); err != nil {
		return err
	}
	return nil
}
