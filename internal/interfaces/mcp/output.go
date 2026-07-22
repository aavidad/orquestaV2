package mcpinterface

import (
	"encoding/base64"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

type GoalView struct {
	RequestRef     string                `json:"request_ref"`
	GoalRef        string                `json:"goal_ref"`
	IntentRef      string                `json:"intent_ref"`
	IntentHash     string                `json:"intent_hash"`
	ActorRef       string                `json:"actor_ref"`
	ProjectRef     string                `json:"project_ref"`
	Statement      string                `json:"statement"`
	AppSpec        AppSpecView           `json:"app_spec"`
	State          string                `json:"state"`
	Revision       uint64                `json:"revision"`
	PlanGeneration uint64                `json:"plan_generation"`
	CreatedAt      time.Time             `json:"created_at"`
	StartedAt      *time.Time            `json:"started_at,omitempty"`
	ClosedAt       *time.Time            `json:"closed_at,omitempty"`
	Phases         []PhaseView           `json:"phases"`
	WorkItems      []WorkItemView        `json:"work_items"`
	Executions     []ExecutionView       `json:"executions"`
	Artifacts      []ArtifactEvidence    `json:"artifacts"`
	Attestations   []AttestationEvidence `json:"attestations"`
}

type AppSpecView struct {
	Ref         string     `json:"ref"`
	Generation  uint64     `json:"generation"`
	Hash        string     `json:"hash"`
	ParentRef   string     `json:"parent_ref,omitempty"`
	ParentHash  string     `json:"parent_hash,omitempty"`
	Objective   string     `json:"objective"`
	Reason      string     `json:"reason"`
	ConfirmedBy string     `json:"confirmed_by"`
	ConfirmedAt time.Time  `json:"confirmed_at"`
	Intent      IntentView `json:"intent"`
}

type IntentView struct {
	Ref         string    `json:"ref"`
	Hash        string    `json:"hash"`
	ActorRef    string    `json:"actor_ref"`
	ProjectRef  string    `json:"project_ref"`
	Statement   string    `json:"statement"`
	SubmittedAt time.Time `json:"submitted_at"`
}

type PhaseView struct {
	PhaseRef      string   `json:"phase_ref"`
	PhaseKey      string   `json:"phase_key"`
	TemplateRef   string   `json:"template_ref"`
	InputRefs     []string `json:"input_refs"`
	CriterionRefs []string `json:"criterion_refs"`
}

type WorkItemView struct {
	WorkItemRef     string             `json:"work_item_ref"`
	Objective       string             `json:"objective"`
	PhaseKey        string             `json:"phase_key"`
	RoleKey         string             `json:"role_key"`
	ParentRef       string             `json:"parent_ref,omitempty"`
	ChildRefs       []string           `json:"child_refs"`
	DependencyRefs  []string           `json:"dependency_refs"`
	WriteSet        []string           `json:"write_set"`
	RequiredTests   []RequiredTestView `json:"required_tests"`
	SkillRefs       []string           `json:"skill_refs"`
	ToolRefs        []string           `json:"tool_refs"`
	CapabilityRefs  []string           `json:"capability_refs"`
	OutputContract  string             `json:"output_contract"`
	SkipReason      string             `json:"skip_reason,omitempty"`
	State           string             `json:"state"`
	Revision        uint64             `json:"revision"`
	CreatedAt       time.Time          `json:"created_at"`
	StartedAt       *time.Time         `json:"started_at,omitempty"`
	FinishedAt      *time.Time         `json:"finished_at,omitempty"`
	ExecutionRef    string             `json:"execution_ref,omitempty"`
	ArtifactRefs    []string           `json:"artifact_refs"`
	AttestationRefs []string           `json:"attestation_refs"`
}

type RequiredTestView struct {
	Ref              string   `json:"ref"`
	ToolRef          string   `json:"tool_ref"`
	Arguments        []string `json:"arguments"`
	WorkingDirectory string   `json:"working_directory"`
	Digest           string   `json:"digest"`
}

type ExecutionView struct {
	ExecutionRef      string     `json:"execution_ref"`
	WorkItemRef       string     `json:"work_item_ref"`
	State             string     `json:"state"`
	ArtifactMediaType string     `json:"artifact_media_type"`
	CreatedAt         time.Time  `json:"created_at"`
	DeadlineAt        *time.Time `json:"deadline_at,omitempty"`
	StartedAt         *time.Time `json:"started_at,omitempty"`
	ObservedAt        *time.Time `json:"observed_at,omitempty"`
	FinishedAt        *time.Time `json:"finished_at,omitempty"`
	FailureCode       string     `json:"failure_code,omitempty"`
}

type ArtifactEvidence struct {
	ArtifactRef string    `json:"artifact_ref"`
	WorkItemRef string    `json:"work_item_ref"`
	Digest      string    `json:"digest"`
	MediaType   string    `json:"media_type"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
}

type AttestationEvidence struct {
	AttestationRef string    `json:"attestation_ref"`
	WorkItemRef    string    `json:"work_item_ref"`
	ExecutionRef   string    `json:"execution_ref"`
	ArtifactRef    string    `json:"artifact_ref"`
	Policy         string    `json:"policy"`
	AcceptedAt     time.Time `json:"accepted_at"`
}

type GoalSummaryView struct {
	GoalRef           string     `json:"goal_ref"`
	IntentRef         string     `json:"intent_ref"`
	AppSpecRef        string     `json:"app_spec_ref"`
	AppSpecGeneration uint64     `json:"app_spec_generation"`
	SpecHash          string     `json:"spec_hash"`
	ActorRef          string     `json:"actor_ref"`
	ProjectRef        string     `json:"project_ref"`
	Statement         string     `json:"statement"`
	State             string     `json:"state"`
	Revision          uint64     `json:"revision"`
	CreatedAt         time.Time  `json:"created_at"`
	ClosedAt          *time.Time `json:"closed_at,omitempty"`
	ArtifactCount     int        `json:"artifact_count"`
}

type ArtifactView struct {
	GoalRef       string `json:"goal_ref"`
	ArtifactRef   string `json:"artifact_ref"`
	Digest        string `json:"digest"`
	MediaType     string `json:"media_type"`
	Size          int64  `json:"size"`
	Encoding      string `json:"encoding"`
	ContentBase64 string `json:"content_base64"`
}

func goalView(record application.GoalRecord) GoalView {
	snapshot := record.Goal.Snapshot()
	intent := IntentView{
		Ref: snapshot.AppSpec.Intent.Ref, Hash: snapshot.AppSpec.Intent.Hash,
		ActorRef: snapshot.AppSpec.Intent.ActorRef, ProjectRef: snapshot.AppSpec.Intent.ProjectRef,
		Statement: snapshot.AppSpec.Intent.Statement, SubmittedAt: snapshot.AppSpec.Intent.SubmittedAt,
	}
	appSpec := AppSpecView{
		Ref: snapshot.AppSpec.Ref, Generation: uint64(snapshot.AppSpec.Generation), Hash: snapshot.AppSpec.Hash,
		ParentRef: snapshot.AppSpec.ParentRef, ParentHash: snapshot.AppSpec.ParentHash,
		Objective: snapshot.AppSpec.Objective, Reason: snapshot.AppSpec.Reason,
		ConfirmedBy: snapshot.AppSpec.ConfirmedBy, ConfirmedAt: snapshot.AppSpec.ConfirmedAt,
		Intent: intent,
	}
	phases := make([]PhaseView, 0, len(snapshot.Phases))
	for _, phase := range snapshot.Phases {
		phases = append(phases, PhaseView{
			PhaseRef: phase.Ref, PhaseKey: phase.Key, TemplateRef: phase.TemplateRef,
			InputRefs: nonNilStrings(phase.InputRefs), CriterionRefs: nonNilStrings(phase.CriterionRefs),
		})
	}
	childrenByParent := make(map[string][]string, len(snapshot.WorkItems))
	for _, item := range snapshot.WorkItems {
		if item.ParentRef != "" {
			childrenByParent[item.ParentRef] = append(childrenByParent[item.ParentRef], item.Ref)
		}
	}
	items := make([]WorkItemView, 0, len(snapshot.WorkItems))
	for _, item := range snapshot.WorkItems {
		requiredTests := make([]RequiredTestView, 0, len(item.RequiredTests))
		for _, testSpec := range item.RequiredTests {
			ref, refErr := goal.NewRequiredTestRef(testSpec.Ref)
			toolRef, toolErr := goal.NewToolRef(testSpec.ToolRef)
			spec, specErr := goal.NewRequiredTestSpec(goal.RequiredTestSpecInput{
				Ref: ref, ToolRef: toolRef, Arguments: testSpec.Arguments,
				WorkingDirectory: testSpec.WorkingDirectory,
			})
			if refErr != nil || toolErr != nil || specErr != nil {
				continue
			}
			requiredTests = append(requiredTests, RequiredTestView{
				Ref: testSpec.Ref, ToolRef: testSpec.ToolRef,
				Arguments: nonNilStrings(testSpec.Arguments), WorkingDirectory: testSpec.WorkingDirectory,
				Digest: spec.Digest(),
			})
		}
		items = append(items, WorkItemView{
			WorkItemRef:     item.Ref,
			Objective:       item.Objective,
			PhaseKey:        item.PhaseKey,
			RoleKey:         item.RoleKey,
			ParentRef:       item.ParentRef,
			ChildRefs:       nonNilStrings(childrenByParent[item.Ref]),
			DependencyRefs:  nonNilStrings(item.DependencyRefs),
			WriteSet:        nonNilStrings(item.WriteSet),
			RequiredTests:   requiredTests,
			SkillRefs:       nonNilStrings(item.SkillRefs),
			ToolRefs:        nonNilStrings(item.ToolRefs),
			CapabilityRefs:  nonNilStrings(item.CapabilityRefs),
			OutputContract:  string(item.OutputContract),
			SkipReason:      string(item.SkipReason),
			State:           string(item.State),
			Revision:        uint64(item.Revision),
			CreatedAt:       item.CreatedAt,
			StartedAt:       optionalTime(item.StartedAt),
			FinishedAt:      optionalTime(item.FinishedAt),
			ExecutionRef:    item.ExecutionRef,
			ArtifactRefs:    nonNilStrings(item.ArtifactRefs),
			AttestationRefs: nonNilStrings(item.AttestationRefs),
		})
	}
	artifacts := make([]ArtifactEvidence, 0, len(record.Artifacts))
	for _, artifact := range record.Artifacts {
		artifacts = append(artifacts, ArtifactEvidence{
			ArtifactRef: artifact.Stored.Ref.String(),
			WorkItemRef: artifact.WorkItemRef.String(),
			Digest:      artifact.Stored.Digest,
			MediaType:   artifact.Stored.MediaType,
			Size:        artifact.Stored.Size,
			CreatedAt:   artifact.CreatedAt,
		})
	}
	attestations := make([]AttestationEvidence, 0, len(record.Attestations))
	for _, attestation := range record.Attestations {
		attestations = append(attestations, AttestationEvidence{
			AttestationRef: attestation.Ref.String(),
			WorkItemRef:    attestation.WorkItemRef.String(),
			ExecutionRef:   attestation.ExecutionRef.String(),
			ArtifactRef:    attestation.ArtifactRef.String(),
			Policy:         attestation.Policy,
			AcceptedAt:     attestation.AcceptedAt,
		})
	}
	executions := make([]ExecutionView, 0, len(record.Executions))
	for _, execution := range record.Executions {
		executions = append(executions, executionView(execution))
	}
	return GoalView{
		RequestRef:     record.RequestRef,
		GoalRef:        snapshot.Ref,
		IntentRef:      intent.Ref,
		IntentHash:     intent.Hash,
		ActorRef:       snapshot.ActorRef,
		ProjectRef:     snapshot.ProjectRef,
		Statement:      intent.Statement,
		AppSpec:        appSpec,
		State:          string(snapshot.State),
		Revision:       uint64(snapshot.Revision),
		PlanGeneration: uint64(snapshot.PlanGeneration),
		CreatedAt:      snapshot.CreatedAt,
		StartedAt:      optionalTime(snapshot.StartedAt),
		ClosedAt:       optionalTime(snapshot.ClosedAt),
		Phases:         phases,
		WorkItems:      items,
		Executions:     executions,
		Artifacts:      artifacts,
		Attestations:   attestations,
	}
}

func executionView(execution application.ExecutionRecord) ExecutionView {
	return ExecutionView{
		ExecutionRef:      execution.Ref.String(),
		WorkItemRef:       execution.WorkItemRef.String(),
		State:             string(execution.State),
		ArtifactMediaType: execution.ArtifactMediaType,
		CreatedAt:         execution.CreatedAt,
		DeadlineAt:        optionalTime(execution.DeadlineAt),
		StartedAt:         optionalTime(execution.StartedAt),
		ObservedAt:        optionalTime(execution.LastObservedAt),
		FinishedAt:        optionalTime(execution.FinishedAt),
		FailureCode:       execution.FailureCode,
	}
}

func goalSummaryView(summary application.GoalSummary) GoalSummaryView {
	return GoalSummaryView{
		GoalRef:           summary.Ref.String(),
		IntentRef:         summary.IntentRef.String(),
		AppSpecRef:        summary.AppSpecRef.String(),
		AppSpecGeneration: uint64(summary.AppSpecGeneration),
		SpecHash:          summary.SpecHash,
		ActorRef:          summary.ActorRef.String(),
		ProjectRef:        summary.ProjectRef.String(),
		Statement:         summary.Statement,
		State:             string(summary.State),
		Revision:          uint64(summary.Revision),
		CreatedAt:         summary.CreatedAt,
		ClosedAt:          optionalTime(summary.ClosedAt),
		ArtifactCount:     summary.ArtifactCount,
	}
}

func artifactView(goalRef goal.GoalRef, artifact ports.ArtifactContent) ArtifactView {
	return ArtifactView{
		GoalRef:       goalRef.String(),
		ArtifactRef:   artifact.Ref.String(),
		Digest:        artifact.Digest,
		MediaType:     artifact.MediaType,
		Size:          artifact.Size,
		Encoding:      "base64",
		ContentBase64: base64.StdEncoding.EncodeToString(artifact.Content),
	}
}

func optionalTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	canonical := value.UTC()
	return &canonical
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return append([]string(nil), values...)
}
