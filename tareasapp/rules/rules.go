package rules

import (
	"fmt"
	"orquesta/db"
)

// Assert interfaces for compile-time safety.
var (
	_ db.TaskTransitionCoordinator       = TaskTransitionCoordinator{}
	_ db.AsignacionTransitionCoordinator = AsignacionTransitionCoordinator{}
	_ db.GovernanceTransitionCoordinator = GovernanceTransitionCoordinator{}
)

type TaskTransitionCoordinator struct{}

func (TaskTransitionCoordinator) AfterTomarTarea(t *db.Tarea, agente string) error {
	if t == nil {
		return nil
	}
	db.Audit(agente, "tomar_tarea", "tarea", t.ID, t.Titulo)
	return nil
}

func (TaskTransitionCoordinator) AfterIniciarTarea(t *db.Tarea, agente string) error {
	if t == nil {
		return nil
	}
	db.Audit(agente, "iniciar_tarea", "tarea", t.ID, t.Titulo)
	db.SetEstadoSesion(agente, "programando")
	if t.ProyectoID != nil {
		db.EmitirHookCicloVida(agente, db.HookTaskStart, *t.ProyectoID, "tarea", t.ID, agente, t.Titulo)
	}
	return nil
}

func (TaskTransitionCoordinator) AfterCompletarTarea(t *db.Tarea, agente, commit string) error {
	if t == nil {
		return nil
	}
	db.Audit(agente, "completar_tarea", "tarea", t.ID, t.Titulo+" commit="+commit)
	db.SetEstadoSesion(agente, "disponible")
	if t.ProyectoID != nil {
		db.EmitirHookCicloVida(agente, db.HookTaskFinish, *t.ProyectoID, "tarea", t.ID, agente, t.Titulo)
	}
	if t.ProyectoID != nil {
		stats, _ := db.GetEstadisticasProyecto(*t.ProyectoID)
		if stats.Total > 0 && stats.Completadas == stats.Total {
			p, _ := db.GetProyecto(fmt.Sprintf("%d", *t.ProyectoID))
			if p != nil {
				db.EmitirNotificacion(db.EventoNotificacion{
					Tipo:       "fin_proyecto",
					ID:         p.ID,
					Texto:      p.Nombre,
					ProyectoID: p.ID,
				})
			}
		}
	}
	return nil
}

func (TaskTransitionCoordinator) AfterCancelarTarea(t *db.Tarea, agente, motivo string) error {
	if t == nil {
		return nil
	}
	db.Audit(agente, "cancelar_tarea", "tarea", t.ID, t.Titulo+": "+motivo)
	db.SetEstadoSesion(agente, "disponible")
	return nil
}

func (TaskTransitionCoordinator) AfterBloquearTarea(t *db.Tarea, agente, motivo string) error {
	if t == nil {
		return nil
	}
	if t.ProyectoID != nil {
		if db.BloqueoProyectoRequiereIntervencionHumana(motivo) {
			if err := db.MarcarProyectoEsperandoHumano(*t.ProyectoID, motivo); err != nil {
				return err
			}
		}
		db.EmitirHookCicloVida(agente, db.HookProjectBlocked, *t.ProyectoID, "tarea", t.ID, agente, motivo)
	}
	db.Audit(agente, "bloquear_tarea", "tarea", t.ID, t.Titulo+": "+motivo)
	db.SetEstadoSesion(agente, "esperando")
	db.EmitirNotificacion(db.EventoNotificacion{
		Tipo:   "bloqueo",
		ID:     t.ID,
		Agente: agente,
		Texto:  motivo,
	})
	return nil
}

func (TaskTransitionCoordinator) AfterDesbloquearTarea(t *db.Tarea, agente, resolucion string) error {
	if t == nil {
		return nil
	}
	if t.ProyectoID != nil {
		op, err := db.GetProyectoOperacion(*t.ProyectoID)
		if err != nil {
			return err
		}
		if op.ResumeAutomatico {
			if err := db.MarcarProyectoActivo(*t.ProyectoID, ""); err != nil {
				return err
			}
		}
		db.EmitirHookCicloVida(agente, db.HookProjectUnblocked, *t.ProyectoID, "tarea", t.ID, agente, resolucion)
	}
	db.Audit(agente, "desbloquear_tarea", "tarea", t.ID, resolucion)
	db.SetEstadoSesion(agente, "disponible")
	db.EmitirNotificacion(db.EventoNotificacion{
		Tipo:  "mensaje",
		Texto: fmt.Sprintf("🔓 *Tarea #%d Desbloqueada*\\n\\nResolución: %s", t.ID, resolucion),
	})
	return nil
}

type AsignacionTransitionCoordinator struct{}

func (AsignacionTransitionCoordinator) AfterActivarAsignacion(agente string, proyectoID int64, nota string) error {
	db.Audit(agente, "activar_asignacion", "proyecto", proyectoID, nota)
	return nil
}

func (AsignacionTransitionCoordinator) AfterPausarAsignacion(agente string, proyectoID int64, nota string) error {
	db.Audit(agente, "pausar_asignacion", "proyecto", proyectoID, nota)
	return nil
}

type GovernanceTransitionCoordinator struct{}

func (GovernanceTransitionCoordinator) AfterCrearRegla(actor string, r *db.Regla, id int64) error {
	if r == nil {
		return nil
	}
	db.Audit(actor, "crear_regla", "regla", id, r.Titulo)
	db.NotificarRefreshGobernanza(actor, r.TipoAgente, "crear_regla", map[string]any{
		"regla_id": id,
		"titulo":   r.Titulo,
	})
	return nil
}

func (GovernanceTransitionCoordinator) AfterActualizarRegla(actor string, r *db.Regla) error {
	if r == nil {
		return nil
	}
	db.Audit(actor, "actualizar_regla", "regla", r.ID, r.Titulo)
	db.NotificarRefreshGobernanza(actor, r.TipoAgente, "actualizar_regla", map[string]any{
		"regla_id": r.ID,
		"titulo":   r.Titulo,
	})
	return nil
}

func (GovernanceTransitionCoordinator) AfterSetReglaActiva(actor string, r *db.Regla, activa bool) error {
	if r == nil {
		return nil
	}
	db.Audit(actor, "set_regla_activa", "regla", r.ID, fmt.Sprintf("activa=%t", activa))
	db.NotificarRefreshGobernanza(actor, r.TipoAgente, "set_regla_activa", map[string]any{
		"regla_id": r.ID,
		"activa":   activa,
		"titulo":   r.Titulo,
	})
	return nil
}

func (GovernanceTransitionCoordinator) AfterCrearSkill(actor string, s *db.Skill, id int64) error {
	if s == nil {
		return nil
	}
	db.Audit(actor, "crear_skill", "skill", id, s.Nombre)
	db.NotificarRefreshSkillCatalogo(actor, s, "crear")
	return nil
}
func (GovernanceTransitionCoordinator) AfterActualizarSkill(actor string, s *db.Skill) error {
	if s == nil {
		return nil
	}
	db.Audit(actor, "actualizar_skill", "skill", s.ID, s.Nombre)
	db.NotificarRefreshSkillCatalogo(actor, s, "actualizar")
	return nil
}

func (GovernanceTransitionCoordinator) AfterSetSkillActiva(actor string, s *db.Skill, activa bool) error {
	if s == nil {
		return nil
	}
	db.Audit(actor, "set_skill_activa", "skill", s.ID, fmt.Sprintf("activa=%t", activa))
	db.NotificarRefreshSkillCatalogo(actor, s, "activar")
	return nil
}

func (GovernanceTransitionCoordinator) AfterCrearWorkflow(actor string, w *db.Workflow, id int64) error {
	if w == nil {
		return nil
	}
	db.Audit(actor, "crear_workflow", "workflow", id, w.Nombre)
	db.NotificarRefreshGobernanza(actor, w.TipoAgente, "crear_workflow", map[string]any{
		"workflow_id": id,
		"nombre":      w.Nombre,
	})
	return nil
}

func (GovernanceTransitionCoordinator) AfterActualizarWorkflow(actor string, w *db.Workflow) error {
	if w == nil {
		return nil
	}
	db.Audit(actor, "actualizar_workflow", "workflow", w.ID, w.Nombre)
	db.NotificarRefreshGobernanza(actor, w.TipoAgente, "actualizar_workflow", map[string]any{
		"workflow_id": w.ID,
		"nombre":      w.Nombre,
	})
	return nil
}

func (GovernanceTransitionCoordinator) AfterSetWorkflowActivo(actor string, w *db.Workflow, activo bool) error {
	if w == nil {
		return nil
	}
	db.Audit(actor, "set_workflow_activo", "workflow", w.ID, fmt.Sprintf("activo=%t", activo))
	db.NotificarRefreshGobernanza(actor, w.TipoAgente, "set_workflow_activo", map[string]any{
		"workflow_id": w.ID,
		"activo":      activo,
		"nombre":      w.Nombre,
	})
	return nil
}

func Install() {
	db.SetTaskTransitionCoordinator(TaskTransitionCoordinator{})
	db.SetAsignacionTransitionCoordinator(AsignacionTransitionCoordinator{})
	db.SetGovernanceTransitionCoordinator(GovernanceTransitionCoordinator{})
}
