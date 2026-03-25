package planocontrol

import (
	"strings"
	"testing"

	"orquesta/db"
)

type stubAutomationService struct {
	autonomyCount    int
	staleCount       int
	staleOrdersCount int
	processedCount   int
	refinedCount     int
	handoffCount     int
	staleErr         error
	staleOrdersErr   error
	processedErr     error
	refinedErr       error
	handoffErr       error
	autonomyErr      error
	audits           []string
}

func (s *stubAutomationService) CheckReanimaciones() ([]*db.Agente, error) { return nil, nil }
func (s *stubAutomationService) ResetReanimacion(nombre string) error      { return nil }
func (s *stubAutomationService) GarantizarSaludAgentes() error             { return nil }
func (s *stubAutomationService) PlanificarTareasAutomaticamente() error    { return nil }
func (s *stubAutomationService) ProcesarAutonomiaAgentesBatch() (int, error) {
	return s.autonomyCount, s.autonomyErr
}
func (s *stubAutomationService) ReconciliarRuntimeHandlesStale() (int, error) {
	return s.staleCount, s.staleErr
}
func (s *stubAutomationService) ReconciliarRuntimeOrdersStale() (int, error) {
	return s.staleOrdersCount, s.staleOrdersErr
}
func (s *stubAutomationService) ProcesarRuntimeOrdersBatch() (int, error) {
	return s.processedCount, s.processedErr
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
		staleCount:       2,
		staleOrdersCount: 1,
		processedCount:   3,
		refinedCount:     1,
		handoffCount:     1,
	}
	r := &Runner{Automation: service}
	r.runControlPlane()

	if len(service.audits) != 6 {
		t.Fatalf("esperaba 6 auditorias, got=%d", len(service.audits))
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
		staleErr:       assertErr("fallo handles"),
		staleOrdersErr: assertErr("fallo stale orders"),
		processedErr:   assertErr("fallo batch"),
		refinedErr:     assertErr("fallo refineria"),
		handoffErr:     assertErr("fallo handoffs"),
	}
	r := &Runner{Automation: service}
	r.runControlPlane()

	if len(service.audits) != 6 {
		t.Fatalf("esperaba 6 auditorias de error, got=%d", len(service.audits))
	}
}

func TestRunnerRunControlPlaneEmiteDebug(t *testing.T) {
	service := &stubAutomationService{
		autonomyCount:    1,
		staleCount:       2,
		staleOrdersCount: 3,
		processedCount:   4,
		refinedCount:     5,
		handoffCount:     6,
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
	if !strings.Contains(traces[len(traces)-1], "control_plane autonomia=%d handles_stale=%d orders_stale=%d runtime_orders=%d refineria=%d handoffs=%d") {
		t.Fatalf("traza final inesperada: %+v", traces)
	}
}

type simpleErr string

func (e simpleErr) Error() string { return string(e) }

func assertErr(msg string) error { return simpleErr(msg) }
