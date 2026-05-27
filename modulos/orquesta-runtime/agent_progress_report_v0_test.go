package orquestaruntime

import (
	"encoding/json"
	"testing"
)

func TestValidateAgentProgressReportV0AceptaReporteValido(t *testing.T) {
	report := agentProgressReportValidoV0()

	issues := ValidateAgentProgressReportV0(report)
	if len(issues) != 0 {
		t.Fatalf("errores inesperados: %#v", issues)
	}
}

func TestAgentProgressReportV0JSONRoundTripMantieneContrato(t *testing.T) {
	raw, err := json.Marshal(agentProgressReportValidoV0())
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}

	var report AgentProgressReportV0
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("unmarshal report: %v", err)
	}

	if issues := ValidateAgentProgressReportV0(report); len(issues) != 0 {
		t.Fatalf("errores inesperados tras JSON round-trip: %#v", issues)
	}
	if report.Status != AgentStalledV0 {
		t.Fatalf("status = %q", report.Status)
	}
}

func TestValidateAgentProgressReportV0RechazaStatusInvalido(t *testing.T) {
	report := agentProgressReportValidoV0()
	report.Status = "waiting"

	requireAgentProgressReportCodeV0(t, ValidateAgentProgressReportV0(report), AgentProgressStatusInvalidoV0)
}

func TestValidateAgentProgressReportV0AceptaCamposDePresupuesto(t *testing.T) {
	report := agentProgressReportValidoV0()
	report.BudgetStatus = AgentProgressBudgetOverBudgetButActiveV0
	report.BudgetReason = "Presupuesto superado con actividad reciente."
	report.AgeSeconds = 120
	report.SecondsSinceActivity = 4
	report.SecondsSinceAck = 120
	report.MaxExpectedSeconds = 60
	report.NoActivityLimitSeconds = 30
	report.StartedAt = "2026-05-11T12:00:00Z"
	report.LastActivityAt = "2026-05-11T12:01:56Z"
	report.DecisionRequired = true

	if issues := ValidateAgentProgressReportV0(report); len(issues) != 0 {
		t.Fatalf("errores inesperados: %#v", issues)
	}
}

func TestValidateAgentProgressReportV0AceptaCapacidadLimitada(t *testing.T) {
	report := agentProgressReportValidoV0()
	report.Status = AgentStoppedV0
	report.BudgetStatus = AgentProgressBudgetCapacityLimitedV0
	report.BudgetReason = "Capacidad externa limitada antes de ACK."
	report.DecisionRequired = true

	if issues := ValidateAgentProgressReportV0(report); len(issues) != 0 {
		t.Fatalf("errores inesperados: %#v", issues)
	}
}

func TestValidateAgentProgressReportV0RechazaBudgetStatusInvalido(t *testing.T) {
	report := agentProgressReportValidoV0()
	report.BudgetStatus = "invalid"

	requireAgentProgressReportCodeV0(t, ValidateAgentProgressReportV0(report), AgentProgressStatusInvalidoV0)
}

func TestValidateAgentProgressReportV0RechazaContadoresDePresupuestoNegativos(t *testing.T) {
	report := agentProgressReportValidoV0()
	report.AgeSeconds = -1

	requireAgentProgressReportCodeV0(t, ValidateAgentProgressReportV0(report), AgentProgressReportInvalidoV0)
}

func TestValidateAgentProgressReportV0RechazaRefNoOpaca(t *testing.T) {
	report := agentProgressReportValidoV0()
	report.EvidenceRefs = []string{"evidence/ref"}

	requireAgentProgressReportCodeV0(t, ValidateAgentProgressReportV0(report), AgentProgressReferenciaNoOpacaV0)
}

func TestValidateAgentProgressReportV0RechazaSecreto(t *testing.T) {
	report := agentProgressReportValidoV0()
	report.Summary = "access_token=abc123 no debe viajar en el reporte"

	requireAgentProgressReportCodeV0(t, ValidateAgentProgressReportV0(report), AgentProgressSecretoDetectadoV0)
}

func TestValidateAgentProgressReportV0PermiteVocabularioOperativoOpaco(t *testing.T) {
	report := agentProgressReportValidoV0()
	report.Summary = "Refs opacas de runtime provider model db sql home conservadas."
	report.EvidenceRefs = []string{"evidence-ref-runtime-provider-model-db-sql-home-001"}

	if issues := ValidateAgentProgressReportV0(report); len(issues) != 0 {
		t.Fatalf("vocabulario operativo opaco rechazado: %#v", issues)
	}
}

func TestValidateAgentProgressReportV0RechazaDetalleCrudoDeProgreso(t *testing.T) {
	report := agentProgressReportValidoV0()
	report.Summary = "prompt=raw no debe viajar en el reporte"

	requireAgentProgressReportCodeV0(t, ValidateAgentProgressReportV0(report), AgentProgressDetalleProveedorV0)
}

func TestValidateAgentProgressReportV0RechazaLoopDetectedSinContadores(t *testing.T) {
	report := agentProgressReportValidoV0()
	report.Status = AgentLoopDetectedV0
	report.NoProgressTicks = 0
	report.RepeatedActionCount = 0

	requireAgentProgressReportCodeV0(t, ValidateAgentProgressReportV0(report), AgentProgressLoopCounterRequeridoV0)
}

func requireAgentProgressReportCodeV0(
	t *testing.T,
	issues []AgentProgressReportErrorV0,
	code AgentProgressReportErrorCodeV0,
) {
	t.Helper()
	for _, issue := range issues {
		if issue.Code == code {
			return
		}
	}
	t.Fatalf("no se encontro codigo %q en %#v", code, issues)
}

func agentProgressReportValidoV0() AgentProgressReportV0 {
	return AgentProgressReportV0{
		ReportID:            "agent-progress-report-ref-001",
		RunID:               "run-ref-001",
		AgentRequestID:      "agent-request-ref-001",
		Status:              AgentStalledV0,
		NoProgressTicks:     3,
		RepeatedActionCount: 0,
		Summary:             "Sin avance observable durante tres ticks de supervision.",
		EvidenceRefs: []string{
			"supervision-evidence-ref-001",
			"mailbox-snapshot-ref-001",
		},
	}
}
