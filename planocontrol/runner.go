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
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"orquesta/db"
	"orquesta/notificaciones"
)

type AutomationService interface {
	CheckReanimaciones() ([]*db.Agente, error)
	ResetReanimacion(nombre string) error
	GarantizarSaludAgentes() error
	PlanificarTareasAutomaticamente() error
	ProcesarAutonomiaAgentesBatch() (int, error)
	ProcesarRuntimeSupervisionBatch() (int, error)
	ProcesarSupervisionAutonomaBatch() (int, error)
	ProcesarReviewGatesBatch() (int, error)
	ReconciliarRuntimeHandlesStale() (int, error)
	ReconciliarRuntimeOrdersStale() (int, error)
	ProcesarRuntimeTranscriptBatch() (int, error)
	ProcesarRuntimeMailboxBatch() (int, error)
	ProcesarRuntimeOrdersBatch() (int, error)
	ProcesarGitMergesBatch() (int, error)
	ProcesarRefineriaBatch() (int, error)
	ProcesarHandoffsBatch() (int, error)
	Audit(agente, accion, entidad string, entidadID int64, detalle string)
}

type Runner struct {
	Automation        AutomationService
	NotificationFeed  <-chan db.EventoNotificacion
	InitNotifications func()
	Notifier          func() notificaciones.Notificador
	Debugf            func(format string, args ...any)
	ReanimacionCada   time.Duration
	SaludCada         time.Duration
	PlanificacionCada time.Duration
	ControlPlaneCada  time.Duration
	BatchTimeout      time.Duration
	mu                sync.Mutex
	runningBatches    map[string]runningBatchState
	nextBatchToken    uint64
	wg                sync.WaitGroup
}

type runningBatchState struct {
	token     uint64
	expiresAt time.Time
}

func (r *Runner) Start(ctx context.Context) {
	if r == nil || r.Automation == nil {
		return
	}
	r.debugf("runner start reanimacion=%s salud=%s planificacion=%s control_plane=%s",
		r.reanimacionCada(), r.saludCada(), r.planificacionCada(), r.controlPlaneCada())
	if r.InitNotifications != nil {
		r.InitNotifications()
	}

	r.startLoop(ctx, "reanimaciones", r.reanimacionCada(), r.runReanimaciones)
	r.startLoop(ctx, "salud", r.saludCada(), r.runSalud)
	r.startLoop(ctx, "planificacion", r.planificacionCada(), r.runPlanificacion)
	r.startLoop(ctx, "control_plane", r.controlPlaneCada(), r.runControlPlane)
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.loopNotificaciones(ctx)
	}()
}

func (r *Runner) startLoop(ctx context.Context, name string, each time.Duration, fn func()) {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.loop(ctx, name, each, fn)
	}()
}

func (r *Runner) Wait() {
	if r == nil {
		return
	}
	r.wg.Wait()
}

func (r *Runner) loop(ctx context.Context, name string, each time.Duration, fn func()) {
	select {
	case <-ctx.Done():
		return
	default:
	}
	r.safeLoopCall(name, fn)
	ticker := time.NewTicker(each)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.safeLoopCall(name, fn)
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
		case ev, ok := <-r.NotificationFeed:
			if !ok {
				return
			}
			r.safeLoopCall("notificaciones", func() {
				n := r.notifier()
				if n == nil {
					return
				}
				if aware, ok := n.(notificaciones.EventAware); ok {
					if err := aware.EnviarEvento(ev); err != nil {
						r.debugf("notificaciones event=%s error=%v", strings.TrimSpace(ev.Tipo), err)
					}
					return
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
				default:
					_ = n.EnviarMensaje(strings.TrimSpace(ev.Texto))
				}
			})
		}
	}
}

func (r *Runner) runReanimaciones() {
	reanimar, err := r.Automation.CheckReanimaciones()
	if err != nil {
		r.debugf("reanimaciones error=%v", err)
		return
	}
	r.debugf("reanimaciones candidatos=%d", len(reanimar))
	for _, a := range reanimar {
		r.Automation.Audit("server", "reanimar_agente", "agente", 0, fmt.Sprintf("Agente %s reanimado tras pausa: %s", a.Nombre, a.MotivoPausa))
		_ = r.Automation.ResetReanimacion(a.Nombre)
	}
}

func (r *Runner) runSalud() {
	err := r.Automation.GarantizarSaludAgentes()
	if err != nil {
		r.debugf("salud error=%v", err)
		return
	}
	r.debugf("salud ok")
}

func (r *Runner) runPlanificacion() {
	err := r.Automation.PlanificarTareasAutomaticamente()
	if err != nil {
		r.debugf("planificacion error=%v", err)
		return
	}
	r.debugf("planificacion ok")
}

func (r *Runner) runControlPlane() {
	autonomia := r.runControlPlaneBatch(
		"autonomia",
		"agente",
		"autonomia_agentes_batch",
		"autonomia_agentes_error",
		"autonomia_agentes_panic",
		"Decisiones autónomas procesadas: %d",
		r.Automation.ProcesarAutonomiaAgentesBatch,
	)
	supervision := r.runControlPlaneBatch(
		"runtime_supervision",
		"runtime_handle",
		"runtime_supervision_batch",
		"runtime_supervision_error",
		"runtime_supervision_panic",
		"Supervisiones de runtime procesadas: %d",
		r.Automation.ProcesarRuntimeSupervisionBatch,
	)
	supervisionAutonoma := r.runControlPlaneBatch(
		"supervision",
		"proyecto",
		"supervision_autonoma_batch",
		"supervision_autonoma_error",
		"supervision_autonoma_panic",
		"Supervisiones autónomas procesadas: %d",
		r.Automation.ProcesarSupervisionAutonomaBatch,
	)
	review := r.runControlPlaneBatch(
		"review",
		"review_gate",
		"review_gates_batch",
		"review_gates_error",
		"review_gates_panic",
		"Review gates procesados: %d",
		r.Automation.ProcesarReviewGatesBatch,
	)
	stale := r.runControlPlaneBatch(
		"handles_stale",
		"runtime_handle",
		"runtime_handles_stale",
		"runtime_handles_error",
		"runtime_handles_panic",
		"Handles reconciliados como stale: %d",
		r.Automation.ReconciliarRuntimeHandlesStale,
	)
	recovered := r.runControlPlaneBatch(
		"runtime_orders_stale",
		"runtime_order",
		"runtime_orders_stale",
		"runtime_orders_stale_error",
		"runtime_orders_stale_panic",
		"Órdenes recuperadas por stale: %d",
		r.Automation.ReconciliarRuntimeOrdersStale,
	)
	transcript := r.runControlPlaneBatch(
		"runtime_transcript",
		"runtime_transcript",
		"runtime_transcript_batch",
		"runtime_transcript_batch_error",
		"runtime_transcript_batch_panic",
		"Conversación/runtime transcript procesado: %d",
		r.Automation.ProcesarRuntimeTranscriptBatch,
	)
	mailbox := r.runControlPlaneBatch(
		"runtime_mailbox",
		"runtime_mailbox",
		"runtime_mailbox_batch",
		"runtime_mailbox_batch_error",
		"runtime_mailbox_batch_panic",
		"Mailbox runtime procesado: %d",
		r.Automation.ProcesarRuntimeMailboxBatch,
	)
	processed := r.runControlPlaneBatch(
		"runtime_orders",
		"runtime_order",
		"runtime_orders_batch",
		"runtime_orders_batch_error",
		"runtime_orders_batch_panic",
		"Órdenes procesadas en batch: %d",
		r.Automation.ProcesarRuntimeOrdersBatch,
	)
	merged := r.runControlPlaneBatch(
		"git_merges",
		"git_merge",
		"git_merges_batch",
		"git_merges_batch_error",
		"git_merges_batch_panic",
		"Solicitudes de merge procesadas: %d",
		r.Automation.ProcesarGitMergesBatch,
	)
	refined := r.runControlPlaneBatch(
		"refineria",
		"refineria_solicitud",
		"refineria_batch",
		"refineria_batch_error",
		"refineria_batch_panic",
		"Solicitudes de refinería procesadas: %d",
		r.Automation.ProcesarRefineriaBatch,
	)
	handoffs := r.runControlPlaneBatch(
		"handoffs",
		"agente",
		"handoff_batch",
		"handoff_batch_error",
		"handoff_batch_panic",
		"Handoffs automáticos procesados: %d",
		r.Automation.ProcesarHandoffsBatch,
	)
	r.debugf("control_plane autonomia=%d runtime_supervision=%d supervision=%d review=%d handles_stale=%d orders_stale=%d transcript=%d mailbox=%d runtime_orders=%d git_merges=%d refineria=%d handoffs=%d",
		autonomia, supervision, supervisionAutonoma, review, stale, recovered, transcript, mailbox, processed, merged, refined, handoffs)
}

func (r *Runner) safeLoopCall(name string, fn func()) {
	if fn == nil {
		return
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			detail := fmt.Sprintf("loop=%s panic=%v\n%s", name, recovered, strings.TrimSpace(string(debug.Stack())))
			if r != nil && r.Automation != nil {
				r.Automation.Audit("server", "runner_loop_panic", "runner", 0, detail)
			}
			r.debugf("runner loop=%s panic=%v", name, recovered)
		}
	}()
	fn()
}

func (r *Runner) runControlPlaneBatch(name, entity, auditOK, auditErr, auditPanic, successFmt string, fn func() (int, error)) (count int) {
	token, reclaimed := r.beginControlPlaneBatch(name, r.controlPlaneBatchTimeout())
	if token == 0 {
		r.debugf("control_plane %s skipped=already_running", name)
		return 0
	}
	if reclaimed {
		r.debugf("control_plane %s reclaimed=expired_lease", name)
	}
	type batchOutcome struct {
		count       int
		err         error
		panicDetail string
	}
	outcomeCh := make(chan batchOutcome, 1)
	go func() {
		defer r.finishControlPlaneBatch(name, token)
		defer func() {
			if recovered := recover(); recovered != nil {
				outcomeCh <- batchOutcome{
					panicDetail: fmt.Sprintf("batch=%s panic=%v\n%s", name, recovered, strings.TrimSpace(string(debug.Stack()))),
				}
			}
		}()
		count, err := fn()
		outcomeCh <- batchOutcome{count: count, err: err}
	}()
	select {
	case outcome := <-outcomeCh:
		if outcome.panicDetail != "" {
			r.Automation.Audit("server", auditPanic, entity, 0, outcome.panicDetail)
			r.debugf("control_plane %s panic=%s", name, outcome.panicDetail)
			return 0
		}
		if outcome.err != nil {
			r.Automation.Audit("server", auditErr, entity, 0, outcome.err.Error())
			r.debugf("control_plane %s error=%v", name, outcome.err)
			return 0
		}
		if outcome.count > 0 {
			r.Automation.Audit("server", auditOK, entity, 0, fmt.Sprintf(successFmt, outcome.count))
		}
		return outcome.count
	case <-time.After(r.controlPlaneBatchTimeout()):
		detail := fmt.Sprintf("batch=%s timeout=%s", name, r.controlPlaneBatchTimeout())
		r.Automation.Audit("server", auditErr, entity, 0, detail)
		r.debugf("control_plane %s timeout=%s", name, r.controlPlaneBatchTimeout())
		return 0
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

func (r *Runner) controlPlaneBatchTimeout() time.Duration {
	if r.BatchTimeout <= 0 {
		return 45 * time.Second
	}
	return r.BatchTimeout
}

func (r *Runner) beginControlPlaneBatch(name string, timeout time.Duration) (uint64, bool) {
	if r == nil {
		return 0, false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.runningBatches == nil {
		r.runningBatches = map[string]runningBatchState{}
	}
	now := time.Now()
	if state, exists := r.runningBatches[name]; exists {
		if state.expiresAt.IsZero() || now.Before(state.expiresAt) {
			return 0, false
		}
	}
	r.nextBatchToken++
	r.runningBatches[name] = runningBatchState{
		token:     r.nextBatchToken,
		expiresAt: now.Add(timeout),
	}
	return r.nextBatchToken, true
}

func (r *Runner) finishControlPlaneBatch(name string, token uint64) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	state, ok := r.runningBatches[name]
	if !ok {
		return
	}
	if state.token != token {
		return
	}
	delete(r.runningBatches, name)
}

func (r *Runner) debugf(format string, args ...any) {
	if r == nil || r.Debugf == nil {
		return
	}
	r.Debugf(format, args...)
}
