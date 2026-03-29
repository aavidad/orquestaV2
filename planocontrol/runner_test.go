package planocontrol

import (
	"strings"
	"testing"

	"orquesta/db"
)

type stubAutomationService struct {
	autonomyCount    int
	supervisionCount int
	reviewCount      int
	staleCount       int
	staleOrdersCount int
	transcriptCount  int
	processedCount   int
	mergedCount      int
	refinedCount     int
	handoffCount     int
	staleErr         error
	staleOrdersErr   error
	transcriptErr    error
	processedErr     error
	mergedErr        error
	refinedErr       error
	handoffErr       error
	autonomyErr      error
	supervisionErr   error
	reviewErr        error
	audits           []string
}

func (s *stubAutomationService) CheckReanimaciones() ([]*db.Agente, error) { return nil, nil }
func (s *stubAutomationService) ResetReanimacion(nombre string) error      { return nil }
func (s *stubAutomationService) GarantizarSaludAgentes() error             { return nil }
func (s *stubAutomationService) PlanificarTareasAutomaticamente() error    { return nil }
func (s *stubAutomationService) ProcesarAutonomiaAgentesBatch() (int, error) {
	return s.autonomyCount, s.autonomyErr
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
	return s.transcriptCount, s.transcriptErr
}
func (s *stubAutomationService) ProcesarRuntimeOrdersBatch() (int, error) {
	return s.processedCount, s.processedErr
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
		supervisionCount: 1,
		reviewCount:      1,
		staleCount:       2,
		staleOrdersCount: 1,
		transcriptCount:  1,
		processedCount:   3,
		mergedCount:      1,
		refinedCount:     1,
		handoffCount:     1,
	}
	r := &Runner{Automation: service}
	r.runControlPlane()

	if len(service.audits) != 10 {
		t.Fatalf("esperaba 10 auditorias, got=%d", len(service.audits))
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

func TestRunnerRunControlPlaneAuditaErrores(t *testing.T) {
	service := &stubAutomationService{
		autonomyErr:    assertErr("fallo autonomia"),
		supervisionErr: assertErr("fallo supervision"),
		reviewErr:      assertErr("fallo review"),
		staleErr:       assertErr("fallo handles"),
		staleOrdersErr: assertErr("fallo stale orders"),
		transcriptErr:  assertErr("fallo transcript"),
		processedErr:   assertErr("fallo batch"),
		mergedErr:      assertErr("fallo merges"),
		refinedErr:     assertErr("fallo refineria"),
		handoffErr:     assertErr("fallo handoffs"),
	}
	r := &Runner{Automation: service}
	r.runControlPlane()

	if len(service.audits) != 10 {
		t.Fatalf("esperaba 10 auditorias de error, got=%d", len(service.audits))
	}
}

func TestRunnerRunControlPlaneEmiteDebug(t *testing.T) {
	service := &stubAutomationService{
		autonomyCount:    1,
		supervisionCount: 2,
		reviewCount:      3,
		staleCount:       2,
		staleOrdersCount: 4,
		transcriptCount:  4,
		processedCount:   5,
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
	if !strings.Contains(traces[len(traces)-1], "control_plane autonomia=%d supervision=%d review=%d handles_stale=%d orders_stale=%d transcript=%d runtime_orders=%d git_merges=%d refineria=%d handoffs=%d") {
		t.Fatalf("traza final inesperada: %+v", traces)
	}
}

type simpleErr string

func (e simpleErr) Error() string { return string(e) }

func assertErr(msg string) error { return simpleErr(msg) }
