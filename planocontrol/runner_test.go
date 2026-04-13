package planocontrol

import (
	"context"
	"strings"
	"testing"
	"time"

	"orquesta/db"
	"orquesta/notificaciones"
)

type stubAutomationService struct {
	autonomyCount    int
	pipelineCount    int
	runtimeSupCount  int
	supervisionCount int
	reviewCount      int
	staleCount       int
	staleOrdersCount int
	transcriptCount  int
	mailboxCount     int
	processedCount   int
	hygieneCount     int
	mergedCount      int
	refinedCount     int
	handoffCount     int
	staleErr         error
	staleOrdersErr   error
	transcriptErr    error
	mailboxErr       error
	processedErr     error
	hygieneErr       error
	mergedErr        error
	refinedErr       error
	handoffErr       error
	autonomyErr      error
	pipelineErr      error
	runtimeSupErr    error
	supervisionErr   error
	reviewErr        error
	transcriptPanic  bool
	processedBlock   <-chan struct{}
	processedCalls   int
	mailboxCalls     int
	audits           []string
	reanimar         []*db.Agente
	reanimarErr      error
	resetCalls       []string
	resetErr         error
}

func (s *stubAutomationService) CheckReanimaciones() ([]*db.Agente, error) {
	return s.reanimar, s.reanimarErr
}
func (s *stubAutomationService) ResetReanimacion(nombre string) error {
	s.resetCalls = append(s.resetCalls, nombre)
	return s.resetErr
}
func (s *stubAutomationService) GarantizarSaludAgentes() error          { return nil }
func (s *stubAutomationService) PlanificarTareasAutomaticamente() error { return nil }
func (s *stubAutomationService) ProcesarAutonomiaAgentesBatch() (int, error) {
	return s.autonomyCount, s.autonomyErr
}
func (s *stubAutomationService) ProcesarPipelineLocalBatch() (int, error) {
	return s.pipelineCount, s.pipelineErr
}
func (s *stubAutomationService) ProcesarPresupuestoSesionObservadoBatch() (int, error) {
	return 0, nil
}
func (s *stubAutomationService) ProcesarRuntimeSupervisionBatch() (int, error) {
	return s.runtimeSupCount, s.runtimeSupErr
}
func (s *stubAutomationService) ProcesarSupervisionAutonomaBatch() (int, error) {
	return s.supervisionCount, s.supervisionErr
}
func (s *stubAutomationService) ProcesarReviewGatesBatch() (int, error) {
	return s.reviewCount, s.reviewErr
}
func (s *stubAutomationService) ReconciliarRuntimeHandlesStale() (int, error) {
	return s.staleCount, s.staleErr
}
func (s *stubAutomationService) ReconciliarRuntimeOrdersStale() (int, error) {
	return s.staleOrdersCount, s.staleOrdersErr
}
func (s *stubAutomationService) ProcesarRuntimeTranscriptBatch() (int, error) {
	if s.transcriptPanic {
		panic("boom transcript")
	}
	return s.transcriptCount, s.transcriptErr
}
func (s *stubAutomationService) ProcesarRuntimeMailboxBatch() (int, error) {
	s.mailboxCalls++
	return s.mailboxCount, s.mailboxErr
}
func (s *stubAutomationService) ProcesarRuntimeOrdersBatch() (int, error) {
	s.processedCalls++
	if s.processedBlock != nil {
		<-s.processedBlock
	}
	return s.processedCount, s.processedErr
}
func (s *stubAutomationService) ProcesarRuntimeHygieneBatch() (int, error) {
	return s.hygieneCount, s.hygieneErr
}
func (s *stubAutomationService) ProcesarGitMergesBatch() (int, error) {
	return s.mergedCount, s.mergedErr
}
func (s *stubAutomationService) ProcesarRefineriaBatch() (int, error) {
	return s.refinedCount, s.refinedErr
}
func (s *stubAutomationService) ProcesarHandoffsBatch() (int, error) {
	return s.handoffCount, s.handoffErr
}
func (s *stubAutomationService) Audit(agente, accion, entidad string, entidadID int64, detalle string) {
	s.audits = append(s.audits, accion+"|"+entidad+"|"+detalle)
}

func TestRunnerRunControlPlaneAuditaTrabajoProcesado(t *testing.T) {
	service := &stubAutomationService{
		autonomyCount:    1,
		pipelineCount:    1,
		runtimeSupCount:  1,
		supervisionCount: 1,
		reviewCount:      1,
		staleCount:       2,
		staleOrdersCount: 1,
		transcriptCount:  1,
		mailboxCount:     1,
		processedCount:   3,
		hygieneCount:     1,
		mergedCount:      1,
		refinedCount:     1,
		handoffCount:     1,
	}
	r := &Runner{Automation: service}
	r.runControlPlane()

	if len(service.audits) != 14 {
		t.Fatalf("esperaba 14 auditorias, got=%d", len(service.audits))
	}
}

func TestRunnerRunControlPlaneSinTrabajoNoAudita(t *testing.T) {
	service := &stubAutomationService{}
	r := &Runner{Automation: service}
	r.runControlPlane()

	if len(service.audits) != 0 {
		t.Fatalf("no esperaba auditorias, got=%d", len(service.audits))
	}
}

func TestRunnerRunReanimacionesAuditaSoloExitoReal(t *testing.T) {
	service := &stubAutomationService{
		reanimar: []*db.Agente{{Nombre: "Codex1", MotivoPausa: "cuota"}},
	}
	r := &Runner{Automation: service}

	r.runReanimaciones()

	if len(service.resetCalls) != 1 || service.resetCalls[0] != "Codex1" {
		t.Fatalf("reset calls inesperadas: %+v", service.resetCalls)
	}
	if len(service.audits) != 1 || !strings.Contains(service.audits[0], "reanimar_agente|") {
		t.Fatalf("auditorias inesperadas: %+v", service.audits)
	}
}

func TestRunnerRunReanimacionesAuditaErrorSinMentirExito(t *testing.T) {
	service := &stubAutomationService{
		reanimar: []*db.Agente{{Nombre: "Codex1", MotivoPausa: "cuota"}},
		resetErr: context.DeadlineExceeded,
	}
	r := &Runner{Automation: service}

	r.runReanimaciones()

	if len(service.resetCalls) != 1 || service.resetCalls[0] != "Codex1" {
		t.Fatalf("reset calls inesperadas: %+v", service.resetCalls)
	}
	if len(service.audits) != 1 {
		t.Fatalf("esperaba una auditoria de error, got=%d", len(service.audits))
	}
	if !strings.Contains(service.audits[0], "reanimar_agente_error|") {
		t.Fatalf("auditoria inesperada: %+v", service.audits)
	}
	if strings.Contains(service.audits[0], "reanimar_agente|") {
		t.Fatalf("no deberia auditar exito si falla la reanimacion: %+v", service.audits)
	}
}

func TestRunnerRunControlPlaneAuditaErrores(t *testing.T) {
	service := &stubAutomationService{
		autonomyErr:    assertErr("fallo autonomia"),
		pipelineErr:    assertErr("fallo pipeline"),
		runtimeSupErr:  assertErr("fallo runtime supervision"),
		supervisionErr: assertErr("fallo supervision"),
		reviewErr:      assertErr("fallo review"),
		staleErr:       assertErr("fallo handles"),
		staleOrdersErr: assertErr("fallo stale orders"),
		transcriptErr:  assertErr("fallo transcript"),
		mailboxErr:     assertErr("fallo mailbox"),
		processedErr:   assertErr("fallo batch"),
		hygieneErr:     assertErr("fallo hygiene"),
		mergedErr:      assertErr("fallo merges"),
		refinedErr:     assertErr("fallo refineria"),
		handoffErr:     assertErr("fallo handoffs"),
	}
	r := &Runner{Automation: service}
	r.runControlPlane()

	if len(service.audits) != 14 {
		t.Fatalf("esperaba 14 auditorias de error, got=%d", len(service.audits))
	}
}

func TestRunnerRunControlPlaneEmiteDebug(t *testing.T) {
	service := &stubAutomationService{
		autonomyCount:    1,
		pipelineCount:    2,
		runtimeSupCount:  2,
		supervisionCount: 2,
		reviewCount:      3,
		staleCount:       2,
		staleOrdersCount: 4,
		transcriptCount:  4,
		mailboxCount:     4,
		processedCount:   5,
		hygieneCount:     5,
		mergedCount:      6,
		refinedCount:     6,
		handoffCount:     7,
	}
	var traces []string
	r := &Runner{
		Automation: service,
		Debugf: func(format string, args ...any) {
			traces = append(traces, format)
		},
	}
	r.runControlPlane()

	if len(traces) == 0 {
		t.Fatalf("esperaba trazas de debug")
	}
	hasHot := false
	hasWarm := false
	hasCold := false
	for _, tr := range traces {
		if strings.Contains(tr, "control_plane_hot transcript=%d mailbox=%d runtime_orders=%d") {
			hasHot = true
		}
		if strings.Contains(tr, "control_plane_warm autonomia=%d pipeline_local=%d runtime_supervision=%d supervision=%d review=%d") {
			hasWarm = true
		}
		if strings.Contains(tr, "control_plane_cold handles_stale=%d orders_stale=%d runtime_hygiene=%d") {
			hasCold = true
		}
	}
	if !hasHot || !hasWarm || !hasCold {
		t.Fatalf("faltan trazas de carriles hot=%v warm=%v cold=%v: %+v", hasHot, hasWarm, hasCold, traces)
	}
}

func TestRunnerRunControlPlaneWarmDrenaCarrilHotSiGeneraTrabajo(t *testing.T) {
	service := &stubAutomationService{
		autonomyCount:  1,
		pipelineCount:  1,
		mailboxCount:   2,
		processedCount: 3,
	}
	r := &Runner{Automation: service}

	r.runControlPlaneWarm()

	if ch := r.batchWakeChannel("control_plane_runtime_mailbox"); ch == nil {
		t.Fatalf("faltaba canal de wake para runtime_mailbox")
	} else {
		select {
		case <-ch:
		default:
			t.Fatalf("warm deberia despertar runtime_mailbox cuando genera trabajo")
		}
	}
	if ch := r.batchWakeChannel("control_plane_runtime_orders"); ch == nil {
		t.Fatalf("faltaba canal de wake para runtime_orders")
	} else {
		select {
		case <-ch:
		default:
			t.Fatalf("warm deberia despertar runtime_orders cuando genera trabajo")
		}
	}
}

func TestRunnerStartRespetaStartupGrace(t *testing.T) {
	service := &stubAutomationService{autonomyCount: 1}
	r := &Runner{
		Automation:        service,
		StartupGrace:      80 * time.Millisecond,
		ControlPlaneCada:  time.Hour,
		ReanimacionCada:   time.Hour,
		SaludCada:         time.Hour,
		PlanificacionCada: time.Hour,
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.Start(ctx)

	time.Sleep(30 * time.Millisecond)
	if len(service.audits) != 0 {
		t.Fatalf("no deberia ejecutar batches antes de startup grace: %+v", service.audits)
	}

	time.Sleep(90 * time.Millisecond)
	if len(service.audits) == 0 {
		t.Fatalf("deberia ejecutar batches tras startup grace")
	}
	cancel()
	r.Wait()
}

func TestRunnerStartResidentCoreNoLanzaTrabajoNoResidente(t *testing.T) {
	service := &stubAutomationService{
		processedCount: 1,
		autonomyCount:  1,
		staleCount:     1,
	}
	r := &Runner{
		Automation:        service,
		StartupGrace:      5 * time.Millisecond,
		ReanimacionCada:   time.Hour,
		SaludCada:         time.Hour,
		PlanificacionCada: time.Hour,
		RuntimeOrdersCada: 10 * time.Millisecond,
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.StartResidentCore(ctx)
	time.Sleep(30 * time.Millisecond)
	cancel()
	r.Wait()

	audits := strings.Join(service.audits, "\n")
	if !strings.Contains(audits, "runtime_orders_batch|runtime_order|Órdenes procesadas en batch: 1") {
		t.Fatalf("faltaba el trabajo caliente del núcleo residente: %s", audits)
	}
	if strings.Contains(audits, "autonomia_agentes_batch|") || strings.Contains(audits, "runtime_handles_stale|") {
		t.Fatalf("el núcleo residente no debería lanzar trabajo no residente: %s", audits)
	}
}

func TestRunnerWakeBatchDespiertaRuntimeOrdersSinEsperarIntervaloNiGrace(t *testing.T) {
	service := &stubAutomationService{processedCount: 1}
	r := &Runner{
		Automation:            service,
		StartupGrace:          time.Hour,
		ReanimacionCada:       time.Hour,
		SaludCada:             time.Hour,
		PlanificacionCada:     time.Hour,
		RuntimeTranscriptCada: time.Hour,
		RuntimeMailboxCada:    time.Hour,
		RuntimeBudgetCada:     time.Hour,
		RuntimeOrdersCada:     time.Hour,
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.StartResidentCore(ctx)
	defer func() {
		cancel()
		r.Wait()
	}()

	time.Sleep(20 * time.Millisecond)
	if service.processedCalls != 0 {
		t.Fatalf("runtime_orders no debería ejecutarse antes del wake explícito: calls=%d", service.processedCalls)
	}
	if !r.WakeRuntimeOrders() {
		t.Fatalf("el wake explícito debería aceptarse")
	}
	deadline := time.Now().Add(300 * time.Millisecond)
	for time.Now().Before(deadline) {
		if service.processedCalls > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("runtime_orders no se despertó con wake explícito: calls=%d audits=%v", service.processedCalls, service.audits)
}

func TestRunnerWakeBatchDespiertaRuntimeMailboxSinEsperarIntervaloNiGrace(t *testing.T) {
	service := &stubAutomationService{mailboxCount: 1}
	r := &Runner{
		Automation:            service,
		StartupGrace:          time.Hour,
		ReanimacionCada:       time.Hour,
		SaludCada:             time.Hour,
		PlanificacionCada:     time.Hour,
		RuntimeTranscriptCada: time.Hour,
		RuntimeMailboxCada:    time.Hour,
		RuntimeBudgetCada:     time.Hour,
		RuntimeOrdersCada:     time.Hour,
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.StartResidentCore(ctx)
	defer func() {
		cancel()
		r.Wait()
	}()

	time.Sleep(20 * time.Millisecond)
	if service.mailboxCalls != 0 {
		t.Fatalf("runtime_mailbox no debería ejecutarse antes del wake explícito: calls=%d", service.mailboxCalls)
	}
	if !r.WakeRuntimeMailbox() {
		t.Fatalf("el wake explícito debería aceptarse")
	}
	deadline := time.Now().Add(300 * time.Millisecond)
	for time.Now().Before(deadline) {
		if service.mailboxCalls > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("runtime_mailbox no se despertó con wake explícito: calls=%d audits=%v", service.mailboxCalls, service.audits)
}

func TestRunnerStartNonResidentWorkerNoLanzaNucleoResidente(t *testing.T) {
	service := &stubAutomationService{
		processedCount: 1,
		autonomyCount:  1,
	}
	r := &Runner{
		Automation:            service,
		StartupGrace:          5 * time.Millisecond,
		ControlPlaneWarmCada:  10 * time.Millisecond,
		ControlPlaneColdCada:  time.Hour,
		NotificationRetryCada: time.Hour,
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.StartNonResidentWorker(ctx)
	time.Sleep(30 * time.Millisecond)
	cancel()
	r.Wait()

	audits := strings.Join(service.audits, "\n")
	if !strings.Contains(audits, "autonomia_agentes_batch|agente|Decisiones autónomas procesadas: 1") {
		t.Fatalf("faltaba el worker no residente: %s", audits)
	}
	if strings.Contains(audits, "runtime_orders_batch|") {
		t.Fatalf("el worker no residente no debería lanzar el núcleo caliente: %s", audits)
	}
}

func TestRunnerRunControlPlaneRecuperaPanicDeBatchYSigue(t *testing.T) {
	service := &stubAutomationService{
		transcriptPanic: true,
		processedCount:  3,
		handoffCount:    1,
	}
	r := &Runner{Automation: service}

	r.runControlPlane()

	audits := strings.Join(service.audits, "\n")
	if !strings.Contains(audits, "runtime_transcript_batch_panic|runtime_transcript|") {
		t.Fatalf("faltaba auditoria de panic: %s", audits)
	}
	if !strings.Contains(audits, "runtime_orders_batch|runtime_order|Órdenes procesadas en batch: 3") {
		t.Fatalf("el control plane deberia seguir tras el panic: %s", audits)
	}
	if !strings.Contains(audits, "handoff_batch|agente|Handoffs automáticos procesados: 1") {
		t.Fatalf("los batches posteriores deberian ejecutarse: %s", audits)
	}
}

func TestRunnerRunControlPlaneTimeoutDeBatchNoCongelaElResto(t *testing.T) {
	service := &stubAutomationService{
		processedBlock: make(chan struct{}),
		handoffCount:   1,
	}
	r := &Runner{
		Automation:   service,
		BatchTimeout: 10 * time.Millisecond,
	}

	r.runControlPlane()

	audits := strings.Join(service.audits, "\n")
	if !strings.Contains(audits, "runtime_orders_batch_error|runtime_order|batch=runtime_orders timeout=10ms") {
		t.Fatalf("faltaba auditoria de timeout del batch runtime_orders: %s", audits)
	}
	if !strings.Contains(audits, "handoff_batch|agente|Handoffs automáticos procesados: 1") {
		t.Fatalf("los batches posteriores deberian seguir ejecutandose tras timeout: %s", audits)
	}
}

func TestRunnerRunControlPlaneNoReiniciaBatchExpiradoMientrasSigueVivo(t *testing.T) {
	block := make(chan struct{})
	service := &stubAutomationService{
		processedBlock: block,
	}
	r := &Runner{
		Automation:   service,
		BatchTimeout: 10 * time.Millisecond,
	}

	r.runControlPlane()
	r.runControlPlane()
	audits := strings.Join(service.audits, "\n")
	if !strings.Contains(audits, "runtime_orders_batch_error|runtime_order|batch=runtime_orders timeout=10ms") {
		t.Fatalf("faltaba timeout inicial del batch runtime_orders: %s", audits)
	}
	if strings.Contains(audits, "overdue timeout=10ms restarting") {
		t.Fatalf("no deberia reiniciar en paralelo un batch ya timeout: %s", audits)
	}
	if service.processedCalls != 1 {
		t.Fatalf("runtime_orders no deberia duplicarse mientras el batch viejo sigue vivo: calls=%d audits=%s", service.processedCalls, audits)
	}

	close(block)
	time.Sleep(20 * time.Millisecond)
	r.runControlPlane()
	if service.processedCalls != 2 {
		t.Fatalf("runtime_orders deberia reanudarse solo tras terminar el batch original: calls=%d audits=%s", service.processedCalls, strings.Join(service.audits, "\n"))
	}
}

func TestRunnerSafeLoopCallRecuperaPanic(t *testing.T) {
	service := &stubAutomationService{}
	r := &Runner{Automation: service}

	r.safeLoopCall("salud", func() { panic("boom salud") })
	r.safeLoopCall("salud", func() {})

	audits := strings.Join(service.audits, "\n")
	if !strings.Contains(audits, "runner_loop_panic|runner|loop=salud panic=boom salud") {
		t.Fatalf("faltaba auditoria de panic en loop: %s", audits)
	}
}

func TestRunnerLoopNotificacionesPrefiereNotificadorEventAware(t *testing.T) {
	service := &stubAutomationService{}
	feed := make(chan db.EventoNotificacion, 1)
	feed <- db.EventoNotificacion{
		Tipo:   "runtime_auto_guidance",
		Agente: "Codex1",
		Texto:  "guía automática",
	}
	close(feed)

	notifier := &stubEventAwareNotifier{}
	r := &Runner{
		Automation:       service,
		NotificationFeed: feed,
		Notifier: func() notificaciones.Notificador {
			return notifier
		},
	}

	r.loopNotificaciones(context.Background())

	if len(notifier.events) != 1 || notifier.events[0].Tipo != "runtime_auto_guidance" {
		t.Fatalf("eventAware no recibió el evento esperado: %+v", notifier.events)
	}
	if len(notifier.messages) != 0 {
		t.Fatalf("no debería caer al path legacy cuando existe EventAware: %+v", notifier.messages)
	}
}

func TestRunnerRunNotificationRetryAuditaReintentos(t *testing.T) {
	service := &stubAutomationService{}
	r := &Runner{
		Automation: service,
		Notifier: func() notificaciones.Notificador {
			return nil
		},
	}
	r.runNotificationRetry()
	if len(service.audits) != 0 {
		t.Fatalf("no deberia auditar nada sin notificador: %+v", service.audits)
	}
}

type simpleErr string

func (e simpleErr) Error() string { return string(e) }

func assertErr(msg string) error { return simpleErr(msg) }

type stubEventAwareNotifier struct {
	events   []db.EventoNotificacion
	messages []string
}

func (s *stubEventAwareNotifier) EnviarMensaje(texto string) error {
	s.messages = append(s.messages, texto)
	return nil
}

func (s *stubEventAwareNotifier) EnviarAlertaBloqueo(tareaID int64, agente, motivo string) error {
	return nil
}

func (s *stubEventAwareNotifier) EnviarPropuestaVotacion(codigo, titulo string) error {
	return nil
}

func (s *stubEventAwareNotifier) EnviarAvisoFinProyecto(proyectoID int64, nombre string) error {
	return nil
}

func (s *stubEventAwareNotifier) EnviarEvento(ev db.EventoNotificacion) error {
	s.events = append(s.events, ev)
	return nil
}
