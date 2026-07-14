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
	ToolGoalsAmend    = "orquesta.goals.amend"
	ToolGoalsCreate   = "orquesta.goals.create"
	ToolGoalsGet      = "orquesta.goals.get"
	ToolGoalsList     = "orquesta.goals.list"
	ToolArtifactsRead = "orquesta.artifacts.read"
	ToolSystemStatus  = "orquesta.system.status"
)

type CreateGoalInput struct {
	RequestRef          string     `json:"request_ref,omitempty" jsonschema:"caller-controlled idempotency reference"`
	Statement           string     `json:"statement,omitempty" jsonschema:"exact operator intent to preserve"`
	NormalizedObjective string     `json:"normalized_objective,omitempty" jsonschema:"confirmed objective used for planning"`
	Confirm             bool       `json:"confirm,omitempty" jsonschema:"explicit confirmation required before durable creation"`
	Plan                *PlanInput `json:"plan,omitempty" jsonschema:"optional typed DAG execution plan"`
}

type PlanInput struct {
	Phases    []string        `json:"phases"`
	WorkItems []WorkItemInput `json:"work_items"`
}

type WorkItemInput struct {
	Key            string   `json:"key"`
	Objective      string   `json:"objective"`
	Phase          string   `json:"phase"`
	Role           string   `json:"role"`
	Dependencies   []string `json:"dependencies"`
	WriteSet       []string `json:"write_set"`
	OutputContract string   `json:"output_contract"`
}

type CreateGoalOutput struct {
	Created bool       `json:"created"`
	Goal    *GoalView  `json:"goal,omitempty"`
	Error   *ToolError `json:"error,omitempty"`
}

type AmendGoalInput struct {
	RequestRef             string `json:"request_ref,omitempty" jsonschema:"caller-controlled idempotency reference"`
	SourceGoalRef          string `json:"source_goal_ref,omitempty" jsonschema:"terminal Goal to amend"`
	ExpectedSourceRevision uint64 `json:"expected_source_revision,omitempty" jsonschema:"source revision compare-and-swap fence"`
	ExpectedSourceSpecHash string `json:"expected_source_spec_hash,omitempty" jsonschema:"source AppSpec hash compare-and-swap fence"`
	Statement              string `json:"statement,omitempty" jsonschema:"exact amended operator intent to preserve"`
	NormalizedObjective    string `json:"normalized_objective,omitempty" jsonschema:"confirmed amended objective"`
	Reason                 string `json:"reason,omitempty" jsonschema:"reason for creating a causal successor"`
	Confirm                bool   `json:"confirm,omitempty" jsonschema:"explicit confirmation required before durable amendment"`
}

type AmendGoalOutput struct {
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
	writeAnnotations := func() *sdkmcp.ToolAnnotations {
		return &sdkmcp.ToolAnnotations{
			DestructiveHint: &nonDestructive,
			IdempotentHint:  true,
			OpenWorldHint:   &closedWorld,
		}
	}
	sdkmcp.AddTool(server.server, &sdkmcp.Tool{
		Name:        ToolGoalsCreate,
		Description: server.catalog.Text(server.locale, "tool.goals.create.description"),
		Annotations: writeAnnotations(),
	}, server.createGoal)
	sdkmcp.AddTool(server.server, &sdkmcp.Tool{
		Name:        ToolGoalsAmend,
		Description: server.catalog.Text(server.locale, "tool.goals.amend.description"),
		Annotations: writeAnnotations(),
	}, server.amendGoal)

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
	if !input.Confirm || strings.TrimSpace(input.RequestRef) == "" || strings.TrimSpace(input.RequestRef) != input.RequestRef || strings.TrimSpace(input.Statement) == "" {
		result, public := server.toolError(publicInvalidRequest)
		return result, CreateGoalOutput{Error: public}, nil
	}
	principal, err := server.principal(ctx)
	if err != nil {
		result, public := server.toolError(publicCode(err))
		return result, CreateGoalOutput{Error: public}, nil
	}
	submitted, err := server.orchestrator.Submit(ctx, application.SubmitRequest{
		RequestRef:          input.RequestRef,
		ActorRef:            principal.ActorRef,
		ProjectRef:          principal.DefaultProjectRef,
		Statement:           input.Statement,
		NormalizedObjective: input.NormalizedObjective,
		Confirm:             input.Confirm,
		Plan:                applicationPlan(input.Plan),
	})
	if err != nil {
		result, public := server.toolError(publicCode(err))
		return result, CreateGoalOutput{Error: public}, nil
	}
	view := goalView(submitted.Record)
	return nil, CreateGoalOutput{Created: submitted.Created, Goal: &view}, nil
}

func (server *Interface) amendGoal(ctx context.Context, _ *sdkmcp.CallToolRequest, input AmendGoalInput) (*sdkmcp.CallToolResult, AmendGoalOutput, error) {
	if !input.Confirm || strings.TrimSpace(input.RequestRef) == "" ||
		strings.TrimSpace(input.RequestRef) != input.RequestRef || strings.TrimSpace(input.Statement) == "" ||
		input.ExpectedSourceRevision == 0 || !goal.IsCanonicalAppSpecHash(input.ExpectedSourceSpecHash) ||
		strings.TrimSpace(input.Reason) == "" {
		result, public := server.toolError(publicInvalidRequest)
		return result, AmendGoalOutput{Error: public}, nil
	}
	sourceGoalRef, err := goal.NewGoalRef(input.SourceGoalRef)
	if err != nil {
		result, public := server.toolError(publicInvalidRequest)
		return result, AmendGoalOutput{Error: public}, nil
	}
	principal, err := server.principal(ctx)
	if err != nil {
		result, public := server.toolError(publicCode(err))
		return result, AmendGoalOutput{Error: public}, nil
	}
	amended, err := server.orchestrator.Amend(ctx, application.AmendRequest{
		RequestRef:             input.RequestRef,
		ActorRef:               principal.ActorRef,
		ProjectRef:             principal.DefaultProjectRef,
		SourceGoalRef:          sourceGoalRef,
		ExpectedSourceRevision: goal.Revision(input.ExpectedSourceRevision),
		ExpectedSourceSpecHash: input.ExpectedSourceSpecHash,
		Statement:              input.Statement,
		NormalizedObjective:    input.NormalizedObjective,
		Reason:                 input.Reason,
		Confirm:                input.Confirm,
	})
	if err != nil {
		result, public := server.toolError(publicCode(err))
		return result, AmendGoalOutput{Error: public}, nil
	}
	view := goalView(amended.Record)
	return nil, AmendGoalOutput{Created: amended.Created, Goal: &view}, nil
}

func applicationPlan(input *PlanInput) *application.PlanSpec {
	if input == nil {
		return nil
	}
	result := &application.PlanSpec{Phases: append([]string(nil), input.Phases...)}
	result.WorkItems = make([]application.WorkItemSpec, 0, len(input.WorkItems))
	for _, item := range input.WorkItems {
		result.WorkItems = append(result.WorkItems, application.WorkItemSpec{
			Key: item.Key, Objective: item.Objective, Phase: item.Phase, Role: item.Role,
			Dependencies:   append([]string(nil), item.Dependencies...),
			WriteSet:       append([]string(nil), item.WriteSet...),
			OutputContract: goal.OutputContractKind(item.OutputContract),
		})
	}
	return result
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
