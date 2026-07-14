package mcpinterface

import (
	"context"
	"strings"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

const (
	ToolGoalsCreate   = "orquesta.goals.create"
	ToolGoalsGet      = "orquesta.goals.get"
	ToolGoalsList     = "orquesta.goals.list"
	ToolArtifactsRead = "orquesta.artifacts.read"
	ToolSystemStatus  = "orquesta.system.status"
)

type CreateGoalInput struct {
	RequestRef string `json:"request_ref,omitempty" jsonschema:"caller-controlled idempotency reference"`
	Statement  string `json:"statement,omitempty" jsonschema:"objective to coordinate as a durable Goal"`
}

type CreateGoalOutput struct {
	Created bool       `json:"created"`
	Goal    *GoalView  `json:"goal,omitempty"`
	Error   *ToolError `json:"error,omitempty"`
}

type GetGoalInput struct {
	GoalRef string `json:"goal_ref,omitempty" jsonschema:"opaque Goal reference"`
}

type GetGoalOutput struct {
	Goal  *GoalView  `json:"goal,omitempty"`
	Error *ToolError `json:"error,omitempty"`
}

type ListGoalsInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"maximum number of Goals to return"`
}

type ListGoalsOutput struct {
	Goals []GoalSummaryView `json:"goals"`
	Count int               `json:"count"`
	Error *ToolError        `json:"error,omitempty"`
}

type ReadArtifactInput struct {
	GoalRef     string `json:"goal_ref,omitempty" jsonschema:"opaque Goal reference owning the artifact"`
	ArtifactRef string `json:"artifact_ref,omitempty" jsonschema:"opaque immutable artifact reference"`
}

type ReadArtifactOutput struct {
	Artifact *ArtifactView `json:"artifact,omitempty"`
	Error    *ToolError    `json:"error,omitempty"`
}

type SystemStatusInput struct{}

type SystemStatusOutput struct {
	Ready              bool       `json:"ready"`
	Version            string     `json:"version"`
	Goals              int64      `json:"goals"`
	RunningGoals       int64      `json:"running_goals"`
	PendingActions     int64      `json:"pending_actions"`
	QuarantinedActions int64      `json:"quarantined_actions"`
	Error              *ToolError `json:"error,omitempty"`
}

func (server *Interface) registerTools() {
	closedWorld := false
	nonDestructive := false
	sdkmcp.AddTool(server.server, &sdkmcp.Tool{
		Name:        ToolGoalsCreate,
		Description: server.catalog.Text(server.locale, "tool.goals.create.description"),
		Annotations: &sdkmcp.ToolAnnotations{
			DestructiveHint: &nonDestructive,
			IdempotentHint:  true,
			OpenWorldHint:   &closedWorld,
		},
	}, server.createGoal)

	readOnlyAnnotations := func() *sdkmcp.ToolAnnotations {
		return &sdkmcp.ToolAnnotations{
			ReadOnlyHint:   true,
			IdempotentHint: true,
			OpenWorldHint:  &closedWorld,
		}
	}
	sdkmcp.AddTool(server.server, &sdkmcp.Tool{
		Name: ToolGoalsGet, Description: server.catalog.Text(server.locale, "tool.goals.get.description"),
		Annotations: readOnlyAnnotations(),
	}, server.getGoal)
	sdkmcp.AddTool(server.server, &sdkmcp.Tool{
		Name: ToolGoalsList, Description: server.catalog.Text(server.locale, "tool.goals.list.description"),
		Annotations: readOnlyAnnotations(),
	}, server.listGoals)
	sdkmcp.AddTool(server.server, &sdkmcp.Tool{
		Name: ToolArtifactsRead, Description: server.catalog.Text(server.locale, "tool.artifacts.read.description"),
		Annotations: readOnlyAnnotations(),
	}, server.readArtifact)
	sdkmcp.AddTool(server.server, &sdkmcp.Tool{
		Name: ToolSystemStatus, Description: server.catalog.Text(server.locale, "tool.system.status.description"),
		Annotations: readOnlyAnnotations(),
	}, server.systemStatus)
}

func (server *Interface) createGoal(ctx context.Context, _ *sdkmcp.CallToolRequest, input CreateGoalInput) (*sdkmcp.CallToolResult, CreateGoalOutput, error) {
	if strings.TrimSpace(input.RequestRef) == "" || strings.TrimSpace(input.RequestRef) != input.RequestRef || strings.TrimSpace(input.Statement) == "" {
		result, public := server.toolError(publicInvalidRequest)
		return result, CreateGoalOutput{Error: public}, nil
	}
	principal, err := server.principal(ctx)
	if err != nil {
		result, public := server.toolError(publicCode(err))
		return result, CreateGoalOutput{Error: public}, nil
	}
	submitted, err := server.orchestrator.Submit(ctx, application.SubmitRequest{
		RequestRef: input.RequestRef,
		ActorRef:   principal.ActorRef,
		ProjectRef: principal.DefaultProjectRef,
		Statement:  input.Statement,
	})
	if err != nil {
		result, public := server.toolError(publicCode(err))
		return result, CreateGoalOutput{Error: public}, nil
	}
	view := goalView(submitted.Record)
	return nil, CreateGoalOutput{Created: submitted.Created, Goal: &view}, nil
}

func (server *Interface) getGoal(ctx context.Context, _ *sdkmcp.CallToolRequest, input GetGoalInput) (*sdkmcp.CallToolResult, GetGoalOutput, error) {
	principal, goalRef, err := server.principalAndGoalRef(ctx, input.GoalRef)
	if err != nil {
		result, public := server.toolError(publicCode(err))
		return result, GetGoalOutput{Error: public}, nil
	}
	record, err := server.orchestrator.GetGoal(ctx, application.GoalQuery{
		ActorRef: principal.ActorRef, ProjectRef: principal.DefaultProjectRef, GoalRef: goalRef,
	})
	if err != nil {
		result, public := server.toolError(publicCode(err))
		return result, GetGoalOutput{Error: public}, nil
	}
	view := goalView(record)
	return nil, GetGoalOutput{Goal: &view}, nil
}

func (server *Interface) listGoals(ctx context.Context, _ *sdkmcp.CallToolRequest, input ListGoalsInput) (*sdkmcp.CallToolResult, ListGoalsOutput, error) {
	limit := input.Limit
	if limit == 0 {
		limit = server.maxListLimit
	}
	if limit < 1 || limit > server.maxListLimit {
		result, public := server.toolError(publicInvalidRequest)
		return result, ListGoalsOutput{Goals: []GoalSummaryView{}, Error: public}, nil
	}
	principal, err := server.principal(ctx)
	if err != nil {
		result, public := server.toolError(publicCode(err))
		return result, ListGoalsOutput{Goals: []GoalSummaryView{}, Error: public}, nil
	}
	summaries, err := server.orchestrator.ListGoals(ctx, principal.ActorRef, principal.DefaultProjectRef, limit)
	if err != nil {
		result, public := server.toolError(publicCode(err))
		return result, ListGoalsOutput{Goals: []GoalSummaryView{}, Error: public}, nil
	}
	goals := make([]GoalSummaryView, 0, len(summaries))
	for _, summary := range summaries {
		goals = append(goals, goalSummaryView(summary))
	}
	return nil, ListGoalsOutput{Goals: goals, Count: len(goals)}, nil
}

func (server *Interface) readArtifact(ctx context.Context, _ *sdkmcp.CallToolRequest, input ReadArtifactInput) (*sdkmcp.CallToolResult, ReadArtifactOutput, error) {
	principal, goalRef, err := server.principalAndGoalRef(ctx, input.GoalRef)
	if err != nil {
		result, public := server.toolError(publicCode(err))
		return result, ReadArtifactOutput{Error: public}, nil
	}
	artifactRef, err := goal.NewArtifactRef(input.ArtifactRef)
	if err != nil {
		result, public := server.toolError(publicInvalidRequest)
		return result, ReadArtifactOutput{Error: public}, nil
	}
	content, err := server.orchestrator.GetArtifact(ctx, application.ArtifactQuery{
		ActorRef: principal.ActorRef, ProjectRef: principal.DefaultProjectRef,
		GoalRef: goalRef, ArtifactRef: artifactRef,
	})
	if err != nil {
		result, public := server.toolError(publicCode(err))
		return result, ReadArtifactOutput{Error: public}, nil
	}
	view := artifactView(goalRef, content)
	return nil, ReadArtifactOutput{Artifact: &view}, nil
}

func (server *Interface) systemStatus(ctx context.Context, _ *sdkmcp.CallToolRequest, _ SystemStatusInput) (*sdkmcp.CallToolResult, SystemStatusOutput, error) {
	status, err := server.orchestrator.Status(ctx)
	if err != nil {
		result, public := server.toolError(publicCode(err))
		return result, SystemStatusOutput{Version: server.version, Error: public}, nil
	}
	return nil, SystemStatusOutput{
		Ready:              true,
		Version:            server.version,
		Goals:              status.Goals,
		RunningGoals:       status.RunningGoals,
		PendingActions:     status.PendingActions,
		QuarantinedActions: status.QuarantinedActions,
	}, nil
}

func (server *Interface) principal(ctx context.Context) (identity.Principal, error) {
	principal, err := server.identity.Principal(ctx)
	if err != nil {
		return identity.Principal{}, err
	}
	if principal.ActorRef.String() == "" || principal.DefaultProjectRef.String() == "" {
		return identity.Principal{}, &goal.DomainError{Code: goal.ErrorInvalidRef}
	}
	return principal, nil
}

func (server *Interface) principalAndGoalRef(ctx context.Context, rawGoalRef string) (identity.Principal, goal.GoalRef, error) {
	goalRef, err := goal.NewGoalRef(rawGoalRef)
	if err != nil {
		return identity.Principal{}, goal.GoalRef{}, err
	}
	principal, err := server.principal(ctx)
	if err != nil {
		return identity.Principal{}, goal.GoalRef{}, err
	}
	return principal, goalRef, nil
}
