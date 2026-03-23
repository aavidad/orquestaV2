package controlplane

import (
	"testing"

	"orquesta/db"
)

type stubAutomationService struct {
	staleCount       int
	staleOrdersCount int
	processedCount   int
	refinedCount     int
	staleErr         error
	staleOrdersErr   error
	processedErr     error
	refinedErr       error
	audits           []string
}

func (s *stubAutomationService) CheckReanimaciones() ([]*db.Agente, error) { return nil, nil }
func (s *stubAutomationService) ResetReanimacion(nombre string) error      { return nil }
func (s *stubAutomationService) GarantizarSaludAgentes() error             { return nil }
func (s *stubAutomationService) PlanificarTareasAutomaticamente() error    { return nil }
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
func (s *stubAutomationService) Audit(agente, accion, entidad string, entidadID int64, detalle string) {
	s.audits = append(s.audits, accion+"|"+entidad+"|"+detalle)
}

func TestRunnerRunControlPlaneAuditaTrabajoProcesado(t *testing.T) {
	service := &stubAutomationService{
		staleCount:       2,
		staleOrdersCount: 1,
		processedCount:   3,
		refinedCount:     1,
	}
	r := &Runner{Automation: service}
	r.runControlPlane()

	if len(service.audits) != 4 {
		t.Fatalf("esperaba 4 auditorias, got=%d", len(service.audits))
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
		staleErr:       assertErr("fallo handles"),
		staleOrdersErr: assertErr("fallo stale orders"),
		processedErr:   assertErr("fallo batch"),
		refinedErr:     assertErr("fallo refineria"),
	}
	r := &Runner{Automation: service}
	r.runControlPlane()

	if len(service.audits) != 4 {
		t.Fatalf("esperaba 4 auditorias de error, got=%d", len(service.audits))
	}
}

type simpleErr string

func (e simpleErr) Error() string { return string(e) }

func assertErr(msg string) error { return simpleErr(msg) }
