package application

import (
	"context"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
)

// ToolExecutor is the single outbound boundary for tool observations.
type ToolExecutor interface {
	InvokeTool(context.Context, ToolExecutionRequest) (ToolObservation, error)
}

type ToolCatalogBinding struct {
	ID, Version, Digest, ReviewRef, ReviewDigest string
}

type ToolExecutionBinding struct {
	ToolRef           goal.ToolRef
	GoalRef           goal.GoalRef
	WorkItemRef       goal.WorkItemRef
	ExecutionRef      goal.ExecutionRef
	PlanGeneration    goal.PlanGeneration
	AppSpecGeneration goal.AppSpecGeneration
	ExecutionAttempt  uint64
	SpecHash          string
}

type ToolExecutionRequest struct {
	ToolID, Version, RequestRef, IdempotencyKey, SpecDigest string
	Catalog                                                 ToolCatalogBinding
	CapabilityRef                                           goal.CapabilityRef
	Execution                                               ToolExecutionBinding
	PrincipalRef                                            identity.PrincipalRef
	ActorRef                                                goal.ActorRef
	ProjectRef                                              goal.ProjectRef
	AuthorizationReceiptRefs                                []string
	Input                                                   []byte
	CostMaximum                                             governance.ResourceVector
	MaxOutputBytes                                          int64
}

type ToolObservation struct {
	ToolID, Version, RequestRef, IdempotencyKey, SpecDigest string
	Catalog                                                 ToolCatalogBinding
	CapabilityRef                                           goal.CapabilityRef
	Execution                                               ToolExecutionBinding
	PrincipalRef                                            identity.PrincipalRef
	ActorRef                                                goal.ActorRef
	ProjectRef                                              goal.ProjectRef
	AuthorizationReceiptRefs                                []string
	Output                                                  []byte
	ReceiptRef                                              string
	ObservedAt                                              time.Time
	Usage                                                   governance.ResourceUsage
}
