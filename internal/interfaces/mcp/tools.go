package mcpinterface

import (
	"context"
	"strings"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"orquesta/internal/application"
	"orquesta/internal/council"
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
	ProjectRef          string     `json:"project_ref" jsonschema:"explicit opaque project scope"`
	RequestRef          string     `json:"request_ref,omitempty" jsonschema:"caller-controlled idempotency reference"`
	Statement           string     `json:"statement,omitempty" jsonschema:"exact operator intent to preserve"`
	NormalizedObjective string     `json:"normalized_objective,omitempty" jsonschema:"confirmed objective used for planning"`
	Confirm             bool       `json:"confirm,omitempty" jsonschema:"explicit confirmation required before durable creation"`
	Plan                *PlanInput `json:"plan,omitempty" jsonschema:"optional typed DAG execution plan"`
}

type PlanInput struct {
	Phases    []PhaseInput    `json:"phases"`
	WorkItems []WorkItemInput `json:"work_items"`
}

type PhaseInput struct {
	Ref           string   `json:"ref"`
	Key           string   `json:"key"`
	TemplateRef   string   `json:"template_ref"`
	InputRefs     []string `json:"input_refs,omitempty"`
	CriterionRefs []string `json:"criterion_refs,omitempty"`
}

type WorkItemInput struct {
	Key            string              `json:"key"`
	Objective      string              `json:"objective"`
	Phase          string              `json:"phase"`
	Role           string              `json:"role"`
	Parent         string              `json:"parent,omitempty"`
	Dependencies   []string            `json:"dependencies"`
	WriteSet       []string            `json:"write_set"`
	CouncilPolicy  string              `json:"council_policy,omitempty" jsonschema:"required for write-scoped work: auto, required, or skip_by_operator"`
	RequiredTests  []RequiredTestInput `json:"required_tests,omitempty"`
	SkillRefs      []string            `json:"skill_refs,omitempty"`
	ToolRefs       []string            `json:"tool_refs,omitempty"`
	CapabilityRefs []string            `json:"capability_refs,omitempty"`
	OutputContract string              `json:"output_contract"`
}

type RequiredTestInput struct {
	Ref              string   `json:"ref"`
	ToolRef          string   `json:"tool_ref"`
	Arguments        []string `json:"arguments"`
	WorkingDirectory string   `json:"working_directory"`
}

type CreateGoalOutput struct {
	Created bool       `json:"created"`
	Goal    *GoalView  `json:"goal,omitempty"`
	Error   *ToolError `json:"error,omitempty"`
}

type AmendGoalInput struct {
	ProjectRef             string `json:"project_ref" jsonschema:"explicit opaque project scope"`
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
	ProjectRef string `json:"project_ref" jsonschema:"explicit opaque project scope"`
	GoalRef    string `json:"goal_ref,omitempty" jsonschema:"opaque Goal reference"`
}

type GetGoalOutput struct {
	Goal  *GoalView  `json:"goal,omitempty"`
	Error *ToolError `json:"error,omitempty"`
}

type ListGoalsInput struct {
	ProjectRef string `json:"project_ref" jsonschema:"explicit opaque project scope"`
	Limit      int    `json:"limit,omitempty" jsonschema:"maximum number of Goals to return"`
}

type ListGoalsOutput struct {
	Goals []GoalSummaryView `json:"goals"`
	Count int               `json:"count"`
	Error *ToolError        `json:"error,omitempty"`
}

type ReadArtifactInput struct {
	ProjectRef  string `json:"project_ref" jsonschema:"explicit opaque project scope"`
	GoalRef     string `json:"goal_ref,omitempty" jsonschema:"opaque Goal reference owning the artifact"`
	ArtifactRef string `json:"artifact_ref,omitempty" jsonschema:"opaque immutable artifact reference"`
}

type ReadArtifactOutput struct {
	Artifact *ArtifactView `json:"artifact,omitempty"`
	Error    *ToolError    `json:"error,omitempty"`
}

type SystemStatusInput struct {
	ProjectRef string `json:"project_ref" jsonschema:"explicit opaque project scope"`
}

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
	access, err := server.access(ctx, input.ProjectRef)
	if err != nil {
		result, public := server.toolError(publicCode(err))
		return result, CreateGoalOutput{Error: public}, nil
	}
	submitted, err := server.orchestrator.Submit(ctx, access, application.SubmitRequest{
		RequestRef:          input.RequestRef,
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
	access, err := server.access(ctx, input.ProjectRef)
	if err != nil {
		result, public := server.toolError(publicCode(err))
		return result, AmendGoalOutput{Error: public}, nil
	}
	amended, err := server.orchestrator.Amend(ctx, access, application.AmendRequest{
		RequestRef:             input.RequestRef,
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
	result := &application.PlanSpec{Phases: make([]application.PhaseSpec, 0, len(input.Phases))}
	for _, phase := range input.Phases {
		result.Phases = append(result.Phases, application.PhaseSpec{
			Ref: phase.Ref, Key: phase.Key, TemplateRef: phase.TemplateRef,
			InputRefs:     append([]string(nil), phase.InputRefs...),
			CriterionRefs: append([]string(nil), phase.CriterionRefs...),
		})
	}
	result.WorkItems = make([]application.WorkItemSpec, 0, len(input.WorkItems))
	for _, item := range input.WorkItems {
		requiredTests := make([]application.RequiredTestSpec, 0, len(item.RequiredTests))
		for _, testSpec := range item.RequiredTests {
			requiredTests = append(requiredTests, application.RequiredTestSpec{
				Ref: testSpec.Ref, ToolRef: testSpec.ToolRef,
				Arguments:        append([]string(nil), testSpec.Arguments...),
				WorkingDirectory: testSpec.WorkingDirectory,
			})
		}
		result.WorkItems = append(result.WorkItems, application.WorkItemSpec{
			Key: item.Key, Objective: item.Objective, Phase: item.Phase, Role: item.Role, Parent: item.Parent,
			Dependencies:   append([]string(nil), item.Dependencies...),
			WriteSet:       append([]string(nil), item.WriteSet...),
			CouncilPolicy:  council.Policy(item.CouncilPolicy),
			RequiredTests:  requiredTests,
			SkillRefs:      append([]string(nil), item.SkillRefs...),
			ToolRefs:       append([]string(nil), item.ToolRefs...),
			CapabilityRefs: append([]string(nil), item.CapabilityRefs...),
			OutputContract: goal.OutputContractKind(item.OutputContract),
		})
	}
	return result
}

func (server *Interface) getGoal(ctx context.Context, _ *sdkmcp.CallToolRequest, input GetGoalInput) (*sdkmcp.CallToolResult, GetGoalOutput, error) {
	access, goalRef, err := server.accessAndGoalRef(ctx, input.ProjectRef, input.GoalRef)
	if err != nil {
		result, public := server.toolError(publicCode(err))
		return result, GetGoalOutput{Error: public}, nil
	}
	record, err := server.orchestrator.GetGoal(ctx, access, goalRef)
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
	access, err := server.access(ctx, input.ProjectRef)
	if err != nil {
		result, public := server.toolError(publicCode(err))
		return result, ListGoalsOutput{Goals: []GoalSummaryView{}, Error: public}, nil
	}
	summaries, err := server.orchestrator.ListGoals(ctx, access, limit)
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
	access, goalRef, err := server.accessAndGoalRef(ctx, input.ProjectRef, input.GoalRef)
	if err != nil {
		result, public := server.toolError(publicCode(err))
		return result, ReadArtifactOutput{Error: public}, nil
	}
	artifactRef, err := goal.NewArtifactRef(input.ArtifactRef)
	if err != nil {
		result, public := server.toolError(publicInvalidRequest)
		return result, ReadArtifactOutput{Error: public}, nil
	}
	content, err := server.orchestrator.GetArtifact(ctx, access, goalRef, artifactRef)
	if err != nil {
		result, public := server.toolError(publicCode(err))
		return result, ReadArtifactOutput{Error: public}, nil
	}
	view := artifactView(goalRef, content)
	return nil, ReadArtifactOutput{Artifact: &view}, nil
}

func (server *Interface) systemStatus(ctx context.Context, _ *sdkmcp.CallToolRequest, input SystemStatusInput) (*sdkmcp.CallToolResult, SystemStatusOutput, error) {
	access, err := server.access(ctx, input.ProjectRef)
	if err != nil {
		result, public := server.toolError(publicCode(err))
		return result, SystemStatusOutput{Version: server.version, Error: public}, nil
	}
	status, err := server.orchestrator.Status(ctx, access)
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
	if err := identity.ValidatePrincipal(principal); err != nil {
		return identity.Principal{}, err
	}
	return principal, nil
}

func (server *Interface) access(ctx context.Context, rawProjectRef string) (application.Access, error) {
	projectRef, err := goal.NewProjectRef(rawProjectRef)
	if err != nil {
		return application.Access{}, err
	}
	principal, err := server.principal(ctx)
	if err != nil {
		return application.Access{}, err
	}
	return application.NewAccess(principal, projectRef)
}

func (server *Interface) accessAndGoalRef(
	ctx context.Context,
	rawProjectRef string,
	rawGoalRef string,
) (application.Access, goal.GoalRef, error) {
	goalRef, err := goal.NewGoalRef(rawGoalRef)
	if err != nil {
		return application.Access{}, goal.GoalRef{}, err
	}
	access, err := server.access(ctx, rawProjectRef)
	if err != nil {
		return application.Access{}, goal.GoalRef{}, err
	}
	return access, goalRef, nil
}
