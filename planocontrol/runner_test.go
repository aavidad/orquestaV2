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
	runtimeSupErr    error
	supervisionErr   error
	reviewErr        error
	transcriptPanic  bool
	processedBlock   <-chan struct{}
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
	return s.mailboxCount, s.mailboxErr
}
func (s *stubAutomationService) ProcesarRuntimeOrdersBatch() (int, error) {
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

	if len(service.audits) != 13 {
		t.Fatalf("esperaba 13 auditorias, got=%d", len(service.audits))
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

	if len(service.audits) != 13 {
		t.Fatalf("esperaba 13 auditorias de error, got=%d", len(service.audits))
	}
}

func TestRunnerRunControlPlaneEmiteDebug(t *testing.T) {
	service := &stubAutomationService{
		autonomyCount:    1,
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
	if !strings.Contains(traces[len(traces)-1], "control_plane autonomia=%d runtime_supervision=%d supervision=%d review=%d handles_stale=%d orders_stale=%d transcript=%d mailbox=%d runtime_orders=%d runtime_hygiene=%d git_merges=%d refineria=%d handoffs=%d") {
		t.Fatalf("traza final inesperada: %+v", traces)
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

func TestRunnerRunControlPlaneReclamaBatchExpiradoEnSiguienteCiclo(t *testing.T) {
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
	close(block)

	audits := strings.Join(service.audits, "\n")
	if strings.Count(audits, "runtime_orders_batch_error|runtime_order|batch=runtime_orders timeout=10ms") < 2 {
		t.Fatalf("el batch runtime_orders deberia volver a intentarse tras expirar la lease: %s", audits)
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
