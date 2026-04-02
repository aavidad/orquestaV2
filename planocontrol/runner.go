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
	ProcesarPresupuestoSesionObservadoBatch() (int, error)
	ProcesarAutonomiaAgentesBatch() (int, error)
	ProcesarRuntimeSupervisionBatch() (int, error)
	ProcesarSupervisionAutonomaBatch() (int, error)
	ProcesarReviewGatesBatch() (int, error)
	ReconciliarRuntimeHandlesStale() (int, error)
	ReconciliarRuntimeOrdersStale() (int, error)
	ProcesarRuntimeTranscriptBatch() (int, error)
	ProcesarRuntimeMailboxBatch() (int, error)
	ProcesarRuntimeOrdersBatch() (int, error)
	ProcesarRuntimeHygieneBatch() (int, error)
	ProcesarGitMergesBatch() (int, error)
	ProcesarRefineriaBatch() (int, error)
	ProcesarHandoffsBatch() (int, error)
	Audit(agente, accion, entidad string, entidadID int64, detalle string)
}

type Runner struct {
	Automation            AutomationService
	NotificationFeed      <-chan db.EventoNotificacion
	InitNotifications     func()
	Notifier              func() notificaciones.Notificador
	Debugf                func(format string, args ...any)
	StartupGrace          time.Duration
	ReanimacionCada       time.Duration
	SaludCada             time.Duration
	PlanificacionCada     time.Duration
	ControlPlaneCada      time.Duration
	ControlPlaneWarmCada  time.Duration
	ControlPlaneColdCada  time.Duration
	RuntimeTranscriptCada time.Duration
	RuntimeMailboxCada    time.Duration
	RuntimeOrdersCada     time.Duration
	RuntimeBudgetCada     time.Duration
	NotificationRetryCada time.Duration
	BatchTimeout          time.Duration
	mu                    sync.Mutex
	runningBatches        map[string]runningBatchState
	batchIntervals        map[string]time.Time
	nextBatchToken        uint64
	wg                    sync.WaitGroup
}

type runningBatchState struct {
	token     uint64
	startedAt time.Time
	expiresAt time.Time
}

func (r *Runner) Start(ctx context.Context) {
	if r == nil || r.Automation == nil {
		return
	}
	r.debugf("runner start reanimacion=%s salud=%s planificacion=%s control_plane_hot=%s warm=%s cold=%s",
		r.reanimacionCada(), r.saludCada(), r.planificacionCada(),
		r.controlPlaneCada(), r.controlPlaneWarmCada(), r.controlPlaneColdCada())
	r.debugf("runner hot_intervals transcript=%s mailbox=%s runtime_orders=%s budget=%s",
		r.runtimeTranscriptCada(), r.runtimeMailboxCada(), r.runtimeOrdersCada(), r.runtimeBudgetCada())
	if r.InitNotifications != nil {
		r.InitNotifications()
	}

	r.startLoop(ctx, "reanimaciones", r.reanimacionCada(), r.runReanimaciones)
	r.startLoop(ctx, "salud", r.saludCada(), r.runSalud)
	r.startLoop(ctx, "planificacion", r.planificacionCada(), r.runPlanificacion)
	r.startLoopAfter(ctx, "control_plane_runtime_orders", r.runtimeOrdersCada(), 0, r.runControlPlaneRuntimeOrders)
	r.startLoopAfter(ctx, "control_plane_runtime_transcript", r.runtimeTranscriptCada(), 10*time.Second, r.runControlPlaneRuntimeTranscript)
	r.startLoopAfter(ctx, "control_plane_runtime_mailbox", r.runtimeMailboxCada(), 20*time.Second, r.runControlPlaneRuntimeMailbox)
	r.startLoopAfter(ctx, "control_plane_runtime_budget", r.runtimeBudgetCada(), 30*time.Second, r.runControlPlaneBudget)
	r.startLoop(ctx, "control_plane_warm", r.controlPlaneWarmCada(), r.runControlPlaneWarm)
	r.startLoop(ctx, "control_plane_cold", r.controlPlaneColdCada(), r.runControlPlaneCold)
	r.startLoop(ctx, "notification_retry", r.notificationRetryCada(), r.runNotificationRetry)
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.loopNotificaciones(ctx)
	}()
}

func (r *Runner) startLoop(ctx context.Context, name string, each time.Duration, fn func()) {
	r.startLoopAfter(ctx, name, each, 0, fn)
}

func (r *Runner) startLoopAfter(ctx context.Context, name string, each time.Duration, extraDelay time.Duration, fn func()) {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.loop(ctx, name, each, extraDelay, fn)
	}()
}

func (r *Runner) Wait() {
	if r == nil {
		return
	}
	r.wg.Wait()
}

func (r *Runner) loop(ctx context.Context, name string, each time.Duration, extraDelay time.Duration, fn func()) {
	if delay := r.startupGrace() + extraDelay; delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
	} else {
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
	r.safeLoopCall(name, fn)
	ticker := time.NewTicker(each)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			start := time.Now()
			r.safeLoopCall(name, fn)
			elapsed := time.Since(start)
			if elapsed > each/2 {
				backoff := elapsed
				if backoff > 2*each {
					backoff = 2 * each
				}
				r.debugf("runner loop=%s backpressure=%s (elapsed=%s interval=%s)", name, backoff, elapsed, each)
				cooldown := time.NewTimer(backoff)
				select {
				case <-ctx.Done():
					cooldown.Stop()
					return
				case <-cooldown.C:
				}
			}
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
		if err := r.Automation.ResetReanimacion(a.Nombre); err != nil {
			r.debugf("reanimacion agente=%s error=%v", a.Nombre, err)
			r.Automation.Audit("server", "reanimar_agente_error", "agente", 0, fmt.Sprintf("Agente %s no pudo reanimarse tras pausa: %v", a.Nombre, err))
			continue
		}
		r.Automation.Audit("server", "reanimar_agente", "agente", 0, fmt.Sprintf("Agente %s reanimado tras pausa: %s", a.Nombre, a.MotivoPausa))
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

// runControlPlane ejecuta todos los batches (compat para tests existentes).
func (r *Runner) runControlPlane() {
	r.runControlPlaneHotNow()
	r.runControlPlaneBudget()
	r.runControlPlaneWarm()
	r.runControlPlaneCold()
}

// runControlPlaneHot: path caliente — runtime orders, mailbox, transcript.
func (r *Runner) runControlPlaneHot() {
	r.runControlPlaneHotInternal(false)
}

func (r *Runner) runControlPlaneHotNow() {
	r.runControlPlaneHotInternal(true)
}

func (r *Runner) runControlPlaneHotInternal(force bool) {
	now := time.Now()
	transcript := 0
	if force || r.allowIntervalBatch("runtime_transcript_interval", r.runtimeTranscriptCada(), now) {
		transcript = r.runControlPlaneBatch(
			"runtime_transcript",
			"runtime_transcript",
			"runtime_transcript_batch",
			"runtime_transcript_batch_error",
			"runtime_transcript_batch_panic",
			"Conversación/runtime transcript procesado: %d",
			r.Automation.ProcesarRuntimeTranscriptBatch,
		)
	}
	mailbox := 0
	if force || r.allowIntervalBatch("runtime_mailbox_interval", r.runtimeMailboxCada(), now) {
		mailbox = r.runControlPlaneBatch(
			"runtime_mailbox",
			"runtime_mailbox",
			"runtime_mailbox_batch",
			"runtime_mailbox_batch_error",
			"runtime_mailbox_batch_panic",
			"Mailbox runtime procesado: %d",
			r.Automation.ProcesarRuntimeMailboxBatch,
		)
	}
	processed := 0
	if force || r.allowIntervalBatch("runtime_orders_interval", r.runtimeOrdersCada(), now) {
		processed = r.runControlPlaneBatch(
			"runtime_orders",
			"runtime_order",
			"runtime_orders_batch",
			"runtime_orders_batch_error",
			"runtime_orders_batch_panic",
			"Órdenes procesadas en batch: %d",
			r.Automation.ProcesarRuntimeOrdersBatch,
		)
	}
	r.debugf("control_plane_hot transcript=%d mailbox=%d runtime_orders=%d",
		transcript, mailbox, processed)
}

func (r *Runner) runControlPlaneRuntimeTranscript() {
	count := r.runControlPlaneBatch(
		"runtime_transcript",
		"runtime_transcript",
		"runtime_transcript_batch",
		"runtime_transcript_batch_error",
		"runtime_transcript_batch_panic",
		"Conversación/runtime transcript procesado: %d",
		r.Automation.ProcesarRuntimeTranscriptBatch,
	)
	r.debugf("control_plane_runtime_transcript count=%d", count)
}

func (r *Runner) runControlPlaneRuntimeMailbox() {
	count := r.runControlPlaneBatch(
		"runtime_mailbox",
		"runtime_mailbox",
		"runtime_mailbox_batch",
		"runtime_mailbox_batch_error",
		"runtime_mailbox_batch_panic",
		"Mailbox runtime procesado: %d",
		r.Automation.ProcesarRuntimeMailboxBatch,
	)
	r.debugf("control_plane_runtime_mailbox count=%d", count)
}

func (r *Runner) runControlPlaneRuntimeOrders() {
	count := r.runControlPlaneBatch(
		"runtime_orders",
		"runtime_order",
		"runtime_orders_batch",
		"runtime_orders_batch_error",
		"runtime_orders_batch_panic",
		"Órdenes procesadas en batch: %d",
		r.Automation.ProcesarRuntimeOrdersBatch,
	)
	r.debugf("control_plane_runtime_orders count=%d", count)
}

func (r *Runner) runControlPlaneBudget() {
	count := r.runControlPlaneBatch(
		"runtime_budget_observation",
		"presupuesto_sesion",
		"runtime_budget_observation_batch",
		"runtime_budget_observation_batch_error",
		"runtime_budget_observation_batch_panic",
		"Presupuestos observados procesados: %d",
		r.Automation.ProcesarPresupuestoSesionObservadoBatch,
	)
	r.debugf("control_plane_runtime_budget count=%d", count)
}

// runControlPlaneWarm: gestión — autonomia, supervision, review, handoffs, merges, refineria.
func (r *Runner) runControlPlaneWarm() {
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
	r.debugf("control_plane_warm autonomia=%d runtime_supervision=%d supervision=%d review=%d git_merges=%d refineria=%d handoffs=%d",
		autonomia, supervision, supervisionAutonoma, review, merged, refined, handoffs)
}

// runControlPlaneCold: limpieza — stale handles, stale orders, hygiene.
func (r *Runner) runControlPlaneCold() {
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
	hygiene := r.runControlPlaneBatch(
		"runtime_hygiene",
		"runtime",
		"runtime_hygiene_batch",
		"runtime_hygiene_batch_error",
		"runtime_hygiene_batch_panic",
		"Deuda historica de runtime purgada: %d",
		r.Automation.ProcesarRuntimeHygieneBatch,
	)
	r.debugf("control_plane_cold handles_stale=%d orders_stale=%d runtime_hygiene=%d",
		stale, recovered, hygiene)
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
	timeout := r.controlPlaneBatchTimeout()
	token, running, overdue := r.beginControlPlaneBatch(name, timeout)
	if token == 0 {
		if overdue {
			detail := fmt.Sprintf("batch=%s overdue timeout=%s", name, timeout)
			r.Automation.Audit("server", auditErr, entity, 0, detail)
			r.debugf("control_plane %s overdue=still_running timeout=%s", name, timeout)
			return 0
		}
		if running {
			r.debugf("control_plane %s skipped=already_running", name)
			return 0
		}
		r.debugf("control_plane %s skipped=unavailable", name)
		return 0
	}
	start := time.Now()
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
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case outcome := <-outcomeCh:
		duration := time.Since(start).Round(time.Millisecond)
		if outcome.panicDetail != "" {
			r.Automation.Audit("server", auditPanic, entity, 0, outcome.panicDetail)
			r.debugf("control_plane %s panic=%s duration=%s", name, outcome.panicDetail, duration)
			return 0
		}
		if outcome.err != nil {
			r.Automation.Audit("server", auditErr, entity, 0, outcome.err.Error())
			r.debugf("control_plane %s error=%v duration=%s", name, outcome.err, duration)
			return 0
		}
		if outcome.count > 0 {
			r.Automation.Audit("server", auditOK, entity, 0, fmt.Sprintf(successFmt, outcome.count))
		}
		r.debugf("control_plane %s ok count=%d duration=%s", name, outcome.count, duration)
		return outcome.count
	case <-timer.C:
		detail := fmt.Sprintf("batch=%s timeout=%s", name, timeout)
		r.Automation.Audit("server", auditErr, entity, 0, detail)
		r.debugf("control_plane %s timeout=%s", name, timeout)
		return 0
	}
}

func (r *Runner) notifier() notificaciones.Notificador {
	if r.Notifier == nil {
		return nil
	}
	return r.Notifier()
}

func (r *Runner) runNotificationRetry() {
	n := r.notifier()
	if n == nil {
		return
	}
	count, err := notificaciones.RetryDueGatewayDeliveries(n, 10)
	if err != nil {
		r.debugf("notification_retry error=%v", err)
		if r.Automation != nil {
			r.Automation.Audit("server", "notification_retry_error", "notificacion", 0, err.Error())
		}
		return
	}
	if count <= 0 {
		return
	}
	r.debugf("notification_retry retried=%d", count)
	if r.Automation != nil {
		r.Automation.Audit("server", "notification_retry", "notificacion", 0, fmt.Sprintf("Entregas OpenClaw reintentadas: %d", count))
	}
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
		return time.Minute
	}
	return r.PlanificacionCada
}

func (r *Runner) controlPlaneCada() time.Duration {
	if r.ControlPlaneCada <= 0 {
		return 30 * time.Second
	}
	return r.ControlPlaneCada
}

func (r *Runner) controlPlaneWarmCada() time.Duration {
	if r.ControlPlaneWarmCada <= 0 && r.ControlPlaneCada > 0 {
		return r.ControlPlaneCada
	}
	if r.ControlPlaneWarmCada <= 0 {
		return 2 * time.Minute
	}
	return r.ControlPlaneWarmCada
}

func (r *Runner) controlPlaneColdCada() time.Duration {
	if r.ControlPlaneColdCada <= 0 && r.ControlPlaneCada > 0 {
		return r.ControlPlaneCada
	}
	if r.ControlPlaneColdCada <= 0 {
		return 5 * time.Minute
	}
	return r.ControlPlaneColdCada
}

func (r *Runner) runtimeTranscriptCada() time.Duration {
	if r.RuntimeTranscriptCada <= 0 {
		return time.Minute
	}
	return r.RuntimeTranscriptCada
}

func (r *Runner) runtimeMailboxCada() time.Duration {
	if r.RuntimeMailboxCada <= 0 {
		return time.Minute
	}
	return r.RuntimeMailboxCada
}

func (r *Runner) runtimeOrdersCada() time.Duration {
	if r.RuntimeOrdersCada <= 0 {
		return r.controlPlaneCada()
	}
	return r.RuntimeOrdersCada
}

func (r *Runner) runtimeBudgetCada() time.Duration {
	if r.RuntimeBudgetCada <= 0 {
		return 2 * time.Minute
	}
	return r.RuntimeBudgetCada
}

func (r *Runner) allowIntervalBatch(name string, each time.Duration, now time.Time) bool {
	if each <= 0 {
		return true
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.batchIntervals == nil {
		r.batchIntervals = map[string]time.Time{}
	}
	last := r.batchIntervals[name]
	if !last.IsZero() && now.Sub(last) < each {
		return false
	}
	r.batchIntervals[name] = now
	return true
}

func (r *Runner) notificationRetryCada() time.Duration {
	if r.NotificationRetryCada <= 0 {
		return time.Minute
	}
	return r.NotificationRetryCada
}

func (r *Runner) controlPlaneBatchTimeout() time.Duration {
	if r.BatchTimeout <= 0 {
		return 45 * time.Second
	}
	return r.BatchTimeout
}

func (r *Runner) startupGrace() time.Duration {
	if r == nil || r.StartupGrace <= 0 {
		return 0
	}
	return r.StartupGrace
}

func (r *Runner) beginControlPlaneBatch(name string, timeout time.Duration) (token uint64, running bool, overdue bool) {
	if r == nil {
		return 0, false, false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.runningBatches == nil {
		r.runningBatches = map[string]runningBatchState{}
	}
	now := time.Now()
	if state, exists := r.runningBatches[name]; exists {
		if state.expiresAt.IsZero() || now.Before(state.expiresAt) {
			return 0, true, false
		}
		state.expiresAt = now.Add(timeout)
		r.runningBatches[name] = state
		return 0, true, true
	}
	r.nextBatchToken++
	r.runningBatches[name] = runningBatchState{
		token:     r.nextBatchToken,
		startedAt: now,
		expiresAt: now.Add(timeout),
	}
	return r.nextBatchToken, false, false
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
