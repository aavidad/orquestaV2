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
	"sync/atomic"
	"time"

	"orquesta/db"
	"orquesta/notificaciones"
)

type AutomationService interface {
	CheckReanimaciones() ([]*db.Agente, error)
	ResetReanimacion(nombre string) error
	GarantizarSaludAgentes() error
	PlanificarTareasAutomaticamente() error
	TieneTrabajoOrquestablePendiente() (bool, string, error)
	ProcesarPresupuestoSesionObservadoBatch() (int, error)
	ProcesarAutonomiaAgentesBatch() (int, error)
	ProcesarPipelineLocalBatch() (int, error)
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

type automationResetReanimationReporter interface {
	ResetReanimacionOutcome(nombre string) (string, error)
}

type Runner struct {
	Automation                  AutomationService
	NotificationFeed            <-chan db.EventoNotificacion
	InitNotifications           func()
	Notifier                    func() notificaciones.Notificador
	Debugf                      func(format string, args ...any)
	EnforceSafeFloors           bool
	StartupGrace                time.Duration
	ReanimacionCada             time.Duration
	SaludCada                   time.Duration
	PlanificacionCada           time.Duration
	ControlPlaneCada            time.Duration
	ControlPlaneWarmCada        time.Duration
	ControlPlaneWarmRequeueCada time.Duration
	ControlPlaneColdCada        time.Duration
	RuntimeTranscriptCada       time.Duration
	RuntimeMailboxCada          time.Duration
	RuntimeOrdersCada           time.Duration
	RuntimeHygieneCada          time.Duration
	RuntimeBudgetCada           time.Duration
	NotificationRetryCada       time.Duration
	BatchTimeout                time.Duration
	mu                          sync.Mutex
	runningBatches              map[string]runningBatchState
	batchWake                   map[string]chan struct{}
	activeBatches               map[string]struct{}
	nextBatchToken              uint64
	warmLaneActive              atomic.Bool
	warmPhase                   atomic.Uint32
	warmRequeue                 *Throttler
	wg                          sync.WaitGroup
}

type NonResidentWorkerOptions struct {
	Warm              bool
	Cold              bool
	NotificationRetry bool
}

type runningBatchState struct {
	token      uint64
	startedAt  time.Time
	expiresAt  time.Time
	timedOutAt time.Time
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
	r.StartResidentCore(ctx)
	r.StartNonResidentWorker(ctx)
}

// StartRuntimeCore arranca solo el carril runtime imprescindible para
// orquestación básica: orders, transcript, mailbox, higiene ligera,
// observación de presupuesto y notificaciones. Excluye reanimaciones,
// salud, planificación y warm/cold.
func (r *Runner) StartRuntimeCore(ctx context.Context) {
	if r == nil || r.Automation == nil {
		return
	}
	r.startLoopAfter(ctx, "control_plane_runtime_orders", r.runtimeOrdersCada(), 0, r.runControlPlaneRuntimeOrders)
	r.startLoopAfter(ctx, "control_plane_runtime_transcript", r.runtimeTranscriptCada(), r.runtimeTranscriptStartupDelay(), r.runControlPlaneRuntimeTranscript)
	r.startLoopAfter(ctx, "control_plane_runtime_mailbox", r.runtimeMailboxCada(), 20*time.Second, r.runControlPlaneRuntimeMailbox)
	r.startLoopAfter(ctx, "control_plane_runtime_hygiene", r.runtimeHygieneCada(), r.runtimeHygieneStartupDelay(), r.runControlPlaneRuntimeHygiene)
	r.startLoopAfter(ctx, "control_plane_runtime_budget", r.runtimeBudgetCada(), 30*time.Second, r.runControlPlaneBudget)
	r.startNotificationLoop(ctx)
}

// StartRuntimeControlCore arranca el núcleo mínimo para que las órdenes
// runtime vivas sigan fluyendo sin encender transcript/mailbox/budget.
func (r *Runner) StartRuntimeControlCore(ctx context.Context) {
	if r == nil || r.Automation == nil {
		return
	}
	// En modo core-only el carril de órdenes vive en wake-only: evita escaneos
	// automáticos al arrancar, pero sigue reaccionando en cuanto una nueva orden
	// despierta explícitamente el batch.
	const dormant = 24 * time.Hour
	r.startLoopAfter(ctx, "control_plane_runtime_orders", dormant, dormant, r.runControlPlaneRuntimeOrders)
	r.startNotificationLoop(ctx)
}

// StartResidentCore arranca solo el núcleo residente imprescindible del daemon.
func (r *Runner) StartResidentCore(ctx context.Context) {
	if r == nil || r.Automation == nil {
		return
	}
	r.startLoop(ctx, "reanimaciones", r.reanimacionCada(), r.runReanimaciones)
	r.startLoop(ctx, "salud", r.saludCada(), r.runSalud)
	r.startLoop(ctx, "planificacion", r.planificacionCada(), r.runPlanificacion)
	r.StartRuntimeCore(ctx)
}

func (r *Runner) startNotificationLoop(ctx context.Context) {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.loopNotificaciones(ctx)
	}()
}

// StartNonResidentWorker arranca el trabajo periódico pesado que no forma parte
// del núcleo residente mínimo, pero sigue viviendo bajo el mismo daemon.
func (r *Runner) StartNonResidentWorker(ctx context.Context) {
	r.StartNonResidentWorkerWithOptions(ctx, NonResidentWorkerOptions{
		Warm:              true,
		Cold:              true,
		NotificationRetry: true,
	})
}

func (r *Runner) StartNonResidentWorkerWithOptions(ctx context.Context, opts NonResidentWorkerOptions) {
	if r == nil || r.Automation == nil {
		return
	}
	r.debugf("runner non_resident warm=%s cold=%s notification_retry=%s",
		r.controlPlaneWarmCada(), r.controlPlaneColdCada(), r.notificationRetryCada())
	if opts.Warm {
		r.startLoop(ctx, "control_plane_warm", r.controlPlaneWarmCada(), r.runControlPlaneWarm)
	}
	if opts.Cold {
		r.startLoop(ctx, "control_plane_cold", r.controlPlaneColdCada(), r.runControlPlaneCold)
	}
	if opts.NotificationRetry {
		r.startLoop(ctx, "notification_retry", r.notificationRetryCada(), r.runNotificationRetry)
	}
}

func (r *Runner) startLoop(ctx context.Context, name string, each time.Duration, fn func()) {
	r.startLoopAfter(ctx, name, each, 0, fn)
}

func (r *Runner) startLoopAfter(ctx context.Context, name string, each time.Duration, extraDelay time.Duration, fn func()) {
	r.registerBatch(name)
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
	wakeCh := r.batchWakeChannel(name)
	if delay := r.startupGrace() + extraDelay; delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-wakeCh:
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
		case <-wakeCh:
			start := time.Now()
			r.safeLoopCall(name, fn)
			r.applyLoopBackpressure(ctx, name, each, start)
		case <-ticker.C:
			start := time.Now()
			r.safeLoopCall(name, fn)
			r.applyLoopBackpressure(ctx, name, each, start)
		}
	}
}

func (r *Runner) WakeBatch(name string) bool {
	if r == nil {
		return false
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	if !r.hasBatch(name) {
		return false
	}
	ch := r.batchWakeChannel(name)
	select {
	case ch <- struct{}{}:
		return true
	default:
		return false
	}
}

func (r *Runner) WakeRuntimeOrders() bool {
	return r.WakeBatch("control_plane_runtime_orders")
}

func (r *Runner) WakeRuntimeMailbox() bool {
	return r.WakeBatch("control_plane_runtime_mailbox")
}

func (r *Runner) batchWakeChannel(name string) chan struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.batchWake == nil {
		r.batchWake = map[string]chan struct{}{}
	}
	ch, ok := r.batchWake[name]
	if ok {
		return ch
	}
	ch = make(chan struct{}, 1)
	r.batchWake[name] = ch
	return ch
}

func (r *Runner) registerBatch(name string) {
	if r == nil {
		return
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.activeBatches == nil {
		r.activeBatches = map[string]struct{}{}
	}
	r.activeBatches[name] = struct{}{}
}

func (r *Runner) hasBatch(name string) bool {
	if r == nil {
		return false
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.activeBatches[name]
	return ok
}

func (r *Runner) applyLoopBackpressure(ctx context.Context, name string, each time.Duration, start time.Time) {
	elapsed := time.Since(start)
	if elapsed <= each/2 {
		return
	}
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
		resultado := "reactivado"
		if reporter, ok := r.Automation.(automationResetReanimationReporter); ok {
			valor, err := reporter.ResetReanimacionOutcome(a.Nombre)
			if err != nil {
				r.debugf("reanimacion agente=%s error=%v", a.Nombre, err)
				r.Automation.Audit("server", "reanimar_agente_error", "agente", 0, fmt.Sprintf("Agente %s no pudo reanimarse tras pausa: %v", a.Nombre, err))
				continue
			}
			if strings.TrimSpace(valor) != "" {
				resultado = strings.TrimSpace(valor)
			}
		} else if err := r.Automation.ResetReanimacion(a.Nombre); err != nil {
			r.debugf("reanimacion agente=%s error=%v", a.Nombre, err)
			r.Automation.Audit("server", "reanimar_agente_error", "agente", 0, fmt.Sprintf("Agente %s no pudo reanimarse tras pausa: %v", a.Nombre, err))
			continue
		}
		if strings.EqualFold(resultado, "cooldown_sostenido") {
			r.Automation.Audit("server", "reanimar_agente_cooldown_sostenido", "agente", 0, fmt.Sprintf("Agente %s sigue bloqueado por cuota visible: %s", a.Nombre, a.MotivoPausa))
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
	r.runControlPlaneRuntimeTranscript()
	r.runControlPlaneRuntimeMailbox()
	r.runControlPlaneRuntimeOrders()
	r.runControlPlaneRuntimeHygiene()
	r.runControlPlaneBudget()
	r.runControlPlaneWarm()
	r.runControlPlaneCold()
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
	if count > 0 {
		r.WakeRuntimeMailbox()
	}
	r.debugf("control_plane_runtime_orders count=%d", count)
}

func (r *Runner) runControlPlaneBudget() {
	if r.warmLaneActive.Load() {
		r.debugf("control_plane_runtime_budget skipped=warm_active")
		return
	}
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

func (r *Runner) runControlPlaneRuntimeHygiene() {
	count := r.runControlPlaneBatch(
		"runtime_hygiene",
		"runtime",
		"runtime_hygiene_batch",
		"runtime_hygiene_batch_error",
		"runtime_hygiene_batch_panic",
		"Deuda historica de runtime purgada: %d",
		r.Automation.ProcesarRuntimeHygieneBatch,
	)
	if count > 0 {
		r.WakeRuntimeOrders()
		r.WakeRuntimeMailbox()
	}
	r.debugf("control_plane_runtime_hygiene count=%d", count)
}

// runControlPlaneWarm: gestión — autonomia, supervision, review, handoffs, merges, refineria.
func (r *Runner) runControlPlaneWarm() {
	if !r.warmLaneActive.CompareAndSwap(false, true) {
		r.debugf("control_plane_warm skipped=already_active")
		return
	}
	defer r.warmLaneActive.Store(false)
	type warmBatch struct {
		name       string
		entity     string
		auditOK    string
		auditErr   string
		auditPanic string
		successFmt string
		fn         func() (int, error)
	}
	batches := []warmBatch{
		{
			name:       "autonomia",
			entity:     "agente",
			auditOK:    "autonomia_agentes_batch",
			auditErr:   "autonomia_agentes_error",
			auditPanic: "autonomia_agentes_panic",
			successFmt: "Decisiones autónomas procesadas: %d",
			fn:         r.Automation.ProcesarAutonomiaAgentesBatch,
		},
		{
			name:       "pipeline_local",
			entity:     "proyecto",
			auditOK:    "pipeline_local_batch",
			auditErr:   "pipeline_local_batch_error",
			auditPanic: "pipeline_local_batch_panic",
			successFmt: "Pasos de pipeline local procesados: %d",
			fn:         r.Automation.ProcesarPipelineLocalBatch,
		},
		{
			name:       "runtime_supervision",
			entity:     "runtime_handle",
			auditOK:    "runtime_supervision_batch",
			auditErr:   "runtime_supervision_error",
			auditPanic: "runtime_supervision_panic",
			successFmt: "Supervisiones de runtime procesadas: %d",
			fn:         r.Automation.ProcesarRuntimeSupervisionBatch,
		},
		{
			name:       "supervision",
			entity:     "proyecto",
			auditOK:    "supervision_autonoma_batch",
			auditErr:   "supervision_autonoma_error",
			auditPanic: "supervision_autonoma_panic",
			successFmt: "Supervisiones autónomas procesadas: %d",
			fn:         r.Automation.ProcesarSupervisionAutonomaBatch,
		},
		{
			name:       "review",
			entity:     "review_gate",
			auditOK:    "review_gates_batch",
			auditErr:   "review_gates_error",
			auditPanic: "review_gates_panic",
			successFmt: "Review gates procesados: %d",
			fn:         r.Automation.ProcesarReviewGatesBatch,
		},
		{
			name:       "git_merges",
			entity:     "git_merge",
			auditOK:    "git_merges_batch",
			auditErr:   "git_merges_batch_error",
			auditPanic: "git_merges_batch_panic",
			successFmt: "Solicitudes de merge procesadas: %d",
			fn:         r.Automation.ProcesarGitMergesBatch,
		},
		{
			name:       "refineria",
			entity:     "refineria_solicitud",
			auditOK:    "refineria_batch",
			auditErr:   "refineria_batch_error",
			auditPanic: "refineria_batch_panic",
			successFmt: "Solicitudes de refinería procesadas: %d",
			fn:         r.Automation.ProcesarRefineriaBatch,
		},
		{
			name:       "handoffs",
			entity:     "agente",
			auditOK:    "handoff_batch",
			auditErr:   "handoff_batch_error",
			auditPanic: "handoff_batch_panic",
			successFmt: "Handoffs automáticos procesados: %d",
			fn:         r.Automation.ProcesarHandoffsBatch,
		},
	}
	idx := 0
	if len(batches) > 0 {
		idx = int((r.warmPhase.Add(1) - 1) % uint32(len(batches)))
	}
	selected := batches[idx]
	count := r.runControlPlaneBatch(
		selected.name,
		selected.entity,
		selected.auditOK,
		selected.auditErr,
		selected.auditPanic,
		selected.successFmt,
		selected.fn,
	)
	r.debugf("control_plane_warm phase=%s count=%d", selected.name, count)
	if count > 0 {
		r.WakeRuntimeMailbox()
		r.WakeRuntimeOrders()
	}
	r.scheduleWarmFollowUpIfPending()
}

// runControlPlaneCold: reconciliación pesada — stale handles y stale orders.
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
	r.debugf("control_plane_cold handles_stale=%d orders_stale=%d",
		stale, recovered)
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
	token, running, timedOut := r.beginControlPlaneBatch(name, timeout)
	if token == 0 {
		if running {
			if timedOut {
				r.debugf("control_plane %s skipped=timed_out_still_running timeout=%s", name, timeout)
				return 0
			}
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
		r.markControlPlaneBatchTimedOut(name, token)
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
		return 60 * time.Second
	}
	return r.ControlPlaneCada
}

func (r *Runner) controlPlaneWarmCada() time.Duration {
	if r.ControlPlaneWarmCada <= 0 && r.ControlPlaneCada > 0 {
		return r.applySafeFloor(r.ControlPlaneCada, 30*time.Second)
	}
	if r.ControlPlaneWarmCada <= 0 {
		return 2 * time.Minute
	}
	return r.applySafeFloor(r.ControlPlaneWarmCada, 30*time.Second)
}

func (r *Runner) controlPlaneWarmRequeueCada() time.Duration {
	if r.ControlPlaneWarmRequeueCada <= 0 {
		return 30 * time.Second
	}
	return r.applySafeFloor(r.ControlPlaneWarmRequeueCada, 30*time.Second)
}

func (r *Runner) controlPlaneColdCada() time.Duration {
	if r.ControlPlaneColdCada <= 0 && r.ControlPlaneCada > 0 {
		return r.applySafeFloor(r.ControlPlaneCada, time.Minute)
	}
	if r.ControlPlaneColdCada <= 0 {
		return 5 * time.Minute
	}
	return r.applySafeFloor(r.ControlPlaneColdCada, time.Minute)
}

func (r *Runner) runtimeTranscriptCada() time.Duration {
	if r.RuntimeTranscriptCada <= 0 && r.ControlPlaneCada > 0 {
		return r.applySafeFloor(r.ControlPlaneCada, 30*time.Second)
	}
	if r.RuntimeTranscriptCada <= 0 {
		return time.Minute
	}
	return r.applySafeFloor(r.RuntimeTranscriptCada, 30*time.Second)
}

func (r *Runner) runtimeTranscriptStartupDelay() time.Duration {
	if r.RuntimeTranscriptCada > 0 || r.ControlPlaneCada > 0 {
		return 0
	}
	return 10 * time.Second
}

func (r *Runner) runtimeMailboxCada() time.Duration {
	if r.RuntimeMailboxCada <= 0 {
		return 30 * time.Second
	}
	return r.applySafeFloor(r.RuntimeMailboxCada, 30*time.Second)
}

func (r *Runner) runtimeOrdersCada() time.Duration {
	if r.RuntimeOrdersCada <= 0 {
		return r.applySafeFloor(r.controlPlaneCada(), 30*time.Second)
	}
	return r.applySafeFloor(r.RuntimeOrdersCada, 30*time.Second)
}

func (r *Runner) runtimeHygieneCada() time.Duration {
	if r.RuntimeHygieneCada <= 0 {
		return 10 * time.Minute
	}
	return r.applySafeFloor(r.RuntimeHygieneCada, time.Minute)
}

func (r *Runner) runtimeHygieneStartupDelay() time.Duration {
	if r.RuntimeHygieneCada > 0 || r.ControlPlaneCada > 0 {
		return 0
	}
	return time.Minute
}

func (r *Runner) runtimeBudgetCada() time.Duration {
	if r.RuntimeBudgetCada <= 0 {
		return 2 * time.Minute
	}
	return r.RuntimeBudgetCada
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

func (r *Runner) warmRequeueGate() *Throttler {
	if r == nil {
		return NewThrottler()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.warmRequeue == nil {
		r.warmRequeue = NewThrottler()
	}
	return r.warmRequeue
}

func (r *Runner) scheduleWarmFollowUpIfPending() {
	if r == nil || r.Automation == nil {
		return
	}
	pending, detail, err := r.Automation.TieneTrabajoOrquestablePendiente()
	if err != nil {
		r.debugf("control_plane_warm pending_check_error=%v", err)
		return
	}
	if !pending {
		return
	}
	if !r.warmRequeueGate().Allow("control_plane_warm", r.controlPlaneWarmRequeueCada()) {
		return
	}
	r.debugf("control_plane_warm follow_up_pending=%s deferred_until_next_tick=%s", strings.TrimSpace(detail), r.controlPlaneWarmCada())
}

func (r *Runner) startupGrace() time.Duration {
	if r == nil || r.StartupGrace <= 0 {
		return 0
	}
	return r.StartupGrace
}

func (r *Runner) applySafeFloor(value, floor time.Duration) time.Duration {
	if r == nil || !r.EnforceSafeFloors || floor <= 0 {
		return value
	}
	if value < floor {
		return floor
	}
	return value
}

func (r *Runner) beginControlPlaneBatch(name string, timeout time.Duration) (token uint64, running bool, timedOut bool) {
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
		if state.timedOutAt.IsZero() && !state.expiresAt.IsZero() && !now.Before(state.expiresAt) {
			state.timedOutAt = now
			r.runningBatches[name] = state
		}
		return 0, true, !state.timedOutAt.IsZero()
	}
	r.nextBatchToken++
	r.runningBatches[name] = runningBatchState{
		token:     r.nextBatchToken,
		startedAt: now,
		expiresAt: now.Add(timeout),
	}
	return r.nextBatchToken, false, false
}

func (r *Runner) markControlPlaneBatchTimedOut(name string, token uint64) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	state, ok := r.runningBatches[name]
	if !ok || state.token != token || !state.timedOutAt.IsZero() {
		return
	}
	state.timedOutAt = time.Now()
	r.runningBatches[name] = state
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
