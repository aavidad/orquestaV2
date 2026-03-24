/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package planocontrol

import (
	"context"
	"fmt"
	"time"

	"orquesta/db"
	"orquesta/notificaciones"
)

type AutomationService interface {
	CheckReanimaciones() ([]*db.Agente, error)
	ResetReanimacion(nombre string) error
	GarantizarSaludAgentes() error
	PlanificarTareasAutomaticamente() error
	ReconciliarRuntimeHandlesStale() (int, error)
	ReconciliarRuntimeOrdersStale() (int, error)
	ProcesarRuntimeOrdersBatch() (int, error)
	ProcesarRefineriaBatch() (int, error)
	ProcesarHandoffsBatch() (int, error)
	Audit(agente, accion, entidad string, entidadID int64, detalle string)
}

type Runner struct {
	Automation        AutomationService
	NotificationFeed  <-chan db.EventoNotificacion
	InitNotifications func()
	Notifier          func() notificaciones.Notificador
	ReanimacionCada   time.Duration
	SaludCada         time.Duration
	PlanificacionCada time.Duration
	ControlPlaneCada  time.Duration
}

func (r *Runner) Start(ctx context.Context) {
	if r == nil || r.Automation == nil {
		return
	}
	if r.InitNotifications != nil {
		r.InitNotifications()
	}

	go r.loop(ctx, r.reanimacionCada(), r.runReanimaciones)
	go r.loop(ctx, r.saludCada(), r.runSalud)
	go r.loop(ctx, r.planificacionCada(), r.runPlanificacion)
	go r.loop(ctx, r.controlPlaneCada(), r.runControlPlane)
	go r.loopNotificaciones(ctx)
}

func (r *Runner) loop(ctx context.Context, each time.Duration, fn func()) {
	fn()
	ticker := time.NewTicker(each)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fn()
		}
	}
}

func (r *Runner) loopNotificaciones(ctx context.Context) {
	if r.NotificationFeed == nil {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case ev := <-r.NotificationFeed:
			n := r.notifier()
			if n == nil {
				continue
			}
			switch ev.Tipo {
			case "bloqueo":
				_ = n.EnviarAlertaBloqueo(ev.ID, ev.Agente, ev.Texto)
			case "propuesta":
				_ = n.EnviarPropuestaVotacion(ev.Codigo, ev.Texto)
			case "fin_proyecto":
				_ = n.EnviarAvisoFinProyecto(ev.ID, ev.Texto)
			case "mensaje":
				_ = n.EnviarMensaje(ev.Texto)
			}
		}
	}
}

func (r *Runner) runReanimaciones() {
	reanimar, err := r.Automation.CheckReanimaciones()
	if err != nil {
		return
	}
	for _, a := range reanimar {
		r.Automation.Audit("server", "reanimar_agente", "agente", 0, fmt.Sprintf("Agente %s reanimado tras pausa: %s", a.Nombre, a.MotivoPausa))
		_ = r.Automation.ResetReanimacion(a.Nombre)
	}
}

func (r *Runner) runSalud() {
	_ = r.Automation.GarantizarSaludAgentes()
}

func (r *Runner) runPlanificacion() {
	_ = r.Automation.PlanificarTareasAutomaticamente()
}

func (r *Runner) runControlPlane() {
	stale, err := r.Automation.ReconciliarRuntimeHandlesStale()
	if err != nil {
		r.Automation.Audit("server", "runtime_handles_error", "runtime_handle", 0, err.Error())
	} else if stale > 0 {
		r.Automation.Audit("server", "runtime_handles_stale", "runtime_handle", 0, fmt.Sprintf("Handles reconciliados como stale: %d", stale))
	}
	recovered, err := r.Automation.ReconciliarRuntimeOrdersStale()
	if err != nil {
		r.Automation.Audit("server", "runtime_orders_stale_error", "runtime_order", 0, err.Error())
	} else if recovered > 0 {
		r.Automation.Audit("server", "runtime_orders_stale", "runtime_order", 0, fmt.Sprintf("Órdenes recuperadas por stale: %d", recovered))
	}
	processed, err := r.Automation.ProcesarRuntimeOrdersBatch()
	if err != nil {
		r.Automation.Audit("server", "runtime_orders_batch_error", "runtime_order", 0, err.Error())
	} else if processed > 0 {
		r.Automation.Audit("server", "runtime_orders_batch", "runtime_order", 0, fmt.Sprintf("Órdenes procesadas en batch: %d", processed))
	}
	refined, err := r.Automation.ProcesarRefineriaBatch()
	if err != nil {
		r.Automation.Audit("server", "refineria_batch_error", "refineria_solicitud", 0, err.Error())
	} else if refined > 0 {
		r.Automation.Audit("server", "refineria_batch", "refineria_solicitud", 0, fmt.Sprintf("Solicitudes de refinería procesadas: %d", refined))
	}
	handoffs, err := r.Automation.ProcesarHandoffsBatch()
	if err != nil {
		r.Automation.Audit("server", "handoff_batch_error", "agente", 0, err.Error())
	} else if handoffs > 0 {
		r.Automation.Audit("server", "handoff_batch", "agente", 0, fmt.Sprintf("Handoffs automáticos procesados: %d", handoffs))
	}
}

func (r *Runner) notifier() notificaciones.Notificador {
	if r.Notifier == nil {
		return nil
	}
	return r.Notifier()
}

func (r *Runner) reanimacionCada() time.Duration {
	if r.ReanimacionCada <= 0 {
		return time.Minute
	}
	return r.ReanimacionCada
}

func (r *Runner) saludCada() time.Duration {
	if r.SaludCada <= 0 {
		return time.Minute
	}
	return r.SaludCada
}

func (r *Runner) planificacionCada() time.Duration {
	if r.PlanificacionCada <= 0 {
		return 30 * time.Second
	}
	return r.PlanificacionCada
}

func (r *Runner) controlPlaneCada() time.Duration {
	if r.ControlPlaneCada <= 0 {
		return 15 * time.Second
	}
	return r.ControlPlaneCada
}
