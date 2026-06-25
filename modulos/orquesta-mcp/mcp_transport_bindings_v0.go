package orquestamcp

import (
	"context"

	orquestaobservability "orquesta/modulos/orquesta-observability"
	operator "orquesta/modulos/orquesta-operator-mcp"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

type MCPTransportBindingsV0 struct {
	NuevaApp                   MCPTransportNuevaAppExecutorV0
	ArrancarDirector           MCPTransportArrancarDirectorAppExecutorV0
	ObserveDirectorGoal        MCPTransportObserveAppDirectorGoalExecutorV0
	RequestAppChange           MCPTransportRequestAppChangeExecutorV0
	EjecutarOrquestacion       MCPTransportEjecutarOrquestacionAppExecutorV0
	DirectorDecision           MCPTransportDirectorAgentDecisionExecutorV0
	DirectorSupervisorBriefing MCPTransportDirectorSupervisorBriefingExecutorV0
	DirectorStats              MCPTransportDirectorStatsExecutorV0
	RunControl                 MCPTransportRunControlExecutorV0
	RuntimeModels              orquestaruntime.RuntimeModelManagerPortV0
	RunQueuePriority           MCPTransportRunQueuePriorityExecutorV0
	RunSupervisor              MCPTransportRunSupervisorExecutorV0
	WorkspaceTimeline          orquestaobservability.WorkspaceTimelineSourcePortV0
	AutoprogrammingPrepareRun  MCPTransportAutoprogrammingPrepareRunExecutorV0
	ServerShutdown             MCPTransportServerShutdownExecutorV0
	DomainWork                 MCPDomainWorkExecutorPortV0
	ExternalWorkRun            MCPTransportExternalWorkRunExecutorV0
	AppVCS                     MCPAppVCSExecutorPortV0
	OperatorConnector          operator.OperatorMCPConnectorV0
	OperatorStatus             operator.OperatorMCPStatusPortV0
	OperatorBurst              operator.OperatorMCPBurstPortV0
	OperatorOutbox             operator.OperatorMCPOutboxPortV0
	OperatorQuery              operator.OperatorMCPDirectedQueryPortV0
}

type MCPTransportNuevaAppExecutorV0 interface {
	Execute(context.Context, MCPNuevaAppToolInputV0) (MCPNuevaAppToolResultV0, error)
}

type MCPTransportArrancarDirectorAppExecutorV0 interface {
	Execute(context.Context, MCPArrancarDirectorAppToolInputV0) (MCPArrancarDirectorAppToolResultV0, error)
}

type MCPTransportObserveAppDirectorGoalExecutorV0 interface {
	Execute(context.Context, MCPObserveAppDirectorGoalToolInputV0) (MCPObserveAppDirectorGoalToolResultV0, error)
}

type MCPTransportRequestAppChangeExecutorV0 interface {
	Execute(context.Context, MCPRequestAppChangeToolInputV0) (MCPRequestAppChangeToolResultV0, error)
}

type MCPTransportEjecutarOrquestacionAppExecutorV0 interface {
	Execute(context.Context, MCPEjecutarOrquestacionAppToolInputV0) (MCPEjecutarOrquestacionAppToolResultV0, error)
}

type MCPTransportDirectorStatsExecutorV0 interface {
	Execute(context.Context, MCPDirectorStatsToolInputV0) (MCPDirectorStatsToolResultV0, error)
}

type MCPTransportRunControlExecutorV0 interface {
	Execute(context.Context, MCPRunControlToolInputV0) (MCPRunControlToolResultV0, error)
}

type MCPTransportRunQueuePriorityExecutorV0 interface {
	Execute(context.Context, MCPRunQueuePriorityToolInputV0) (MCPRunQueuePriorityToolResultV0, error)
}

type MCPTransportRunSupervisorExecutorV0 interface {
	Execute(context.Context, MCPRunSupervisorToolInputV0) (MCPRunSupervisorToolResultV0, error)
}

type MCPTransportAutoprogrammingPrepareRunExecutorV0 interface {
	Execute(context.Context, MCPAutoprogrammingPrepareRunToolInputV0) (MCPAutoprogrammingPrepareRunToolResultV0, error)
}

type MCPTransportAutoprogrammingStatusExecutorV0 interface {
	Execute(context.Context, MCPAutoprogrammingStatusToolInputV0) (MCPAutoprogrammingStatusToolResultV0, error)
}

type MCPTransportServerShutdownExecutorV0 interface {
	Execute(context.Context, MCPServerShutdownToolInputV0) (MCPServerShutdownToolResultV0, error)
}

type MCPTransportExternalWorkRunExecutorV0 interface {
	Execute(context.Context, MCPExternalWorkRunToolInputV0) (MCPExternalWorkRunToolResultV0, error)
}

type MCPTransportToolErrorV0 struct {
	Estado    string `json:"estado"`
	Tool      string `json:"tool"`
	ErrorCode string `json:"error_code"`
}
