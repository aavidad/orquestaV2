package orquestaruntime

import orquestaagentprogress "orquesta/modulos/orquesta-agent-progress"

type AgentProgressStatusV0 = orquestaagentprogress.AgentProgressStatusV0

const (
	AgentProgressingV0  = orquestaagentprogress.AgentProgressingV0
	AgentStalledV0      = orquestaagentprogress.AgentStalledV0
	AgentLoopDetectedV0 = orquestaagentprogress.AgentLoopDetectedV0
	AgentStoppedV0      = orquestaagentprogress.AgentStoppedV0
)

type AgentProgressBudgetStatusV0 = orquestaagentprogress.AgentProgressBudgetStatusV0

const (
	AgentProgressBudgetWorkingV0              = orquestaagentprogress.AgentProgressBudgetWorkingV0
	AgentProgressBudgetStalledV0              = orquestaagentprogress.AgentProgressBudgetStalledV0
	AgentProgressBudgetOverBudgetButActiveV0  = orquestaagentprogress.AgentProgressBudgetOverBudgetButActiveV0
	AgentProgressBudgetOverBudgetNoActivityV0 = orquestaagentprogress.AgentProgressBudgetOverBudgetNoActivityV0
	AgentProgressBudgetAckCleanupV0           = orquestaagentprogress.AgentProgressBudgetAckCleanupV0
	AgentProgressBudgetCapacityLimitedV0      = orquestaagentprogress.AgentProgressBudgetCapacityLimitedV0
)

type AgentProgressReportErrorCodeV0 = orquestaagentprogress.AgentProgressReportErrorCodeV0

const (
	AgentProgressReportInvalidoV0       = orquestaagentprogress.AgentProgressReportInvalidoV0
	AgentProgressStatusInvalidoV0       = orquestaagentprogress.AgentProgressStatusInvalidoV0
	AgentProgressReferenciaNoOpacaV0    = orquestaagentprogress.AgentProgressReferenciaNoOpacaV0
	AgentProgressSecretoDetectadoV0     = orquestaagentprogress.AgentProgressSecretoDetectadoV0
	AgentProgressDetalleProveedorV0     = orquestaagentprogress.AgentProgressDetalleProveedorV0
	AgentProgressLoopCounterRequeridoV0 = orquestaagentprogress.AgentProgressLoopCounterRequeridoV0
)

type AgentProgressReportV0 = orquestaagentprogress.AgentProgressReportV0
type AgentProgressReportErrorV0 = orquestaagentprogress.AgentProgressReportErrorV0

func ValidateAgentProgressReportV0(
	report AgentProgressReportV0,
) []AgentProgressReportErrorV0 {
	return orquestaagentprogress.ValidateAgentProgressReportV0(report)
}
