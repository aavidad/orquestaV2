package ports

import (
	"strconv"
	"strings"
)

// AgentPrompt contains only human-facing work instructions. Runtime identity,
// credentials, effect authority and physical placement deliberately stay out.
type AgentPrompt struct {
	ProjectRef, GoalRef, WorkItemRef, ExecutionRef string
	PlanGeneration, AppSpecGeneration              string
	Objective                                      string
	PhaseRef, PhaseKey, PhaseTemplateRef           string
	PhaseInputRefs, PhaseCriterionRefs             string
	RoleKey                                        string
	SkillRefs, ToolRefs, CapabilityRefs            string
	WriteSet, OutputContract, ArtifactMediaType    string
}

// AgentPromptFromLaunchRequest is the single provider-neutral allowlist used
// by prompt renderers. Joining here keeps locale/presentation out of lifecycle.
func AgentPromptFromLaunchRequest(request AgentLaunchRequest) AgentPrompt {
	return AgentPrompt{
		ProjectRef:         request.ProjectRef.String(),
		GoalRef:            request.GoalRef.String(),
		WorkItemRef:        request.WorkItemRef.String(),
		ExecutionRef:       request.ExecutionRef.String(),
		PlanGeneration:     strconv.FormatUint(uint64(request.PlanGeneration), 10),
		AppSpecGeneration:  strconv.FormatUint(uint64(request.AppSpecGeneration), 10),
		Objective:          request.Objective,
		PhaseRef:           request.PhaseRef,
		PhaseKey:           request.PhaseKey,
		PhaseTemplateRef:   request.PhaseTemplateRef,
		PhaseInputRefs:     strings.Join(request.PhaseInputRefs, "\n"),
		PhaseCriterionRefs: strings.Join(request.PhaseCriterionRefs, "\n"),
		RoleKey:            request.RoleKey,
		SkillRefs:          strings.Join(request.SkillRefs, "\n"),
		ToolRefs:           strings.Join(request.ToolRefs, "\n"),
		CapabilityRefs:     strings.Join(request.CapabilityRefs, "\n"),
		WriteSet:           strings.Join(request.WriteSet, "\n"),
		OutputContract:     request.OutputContract,
		ArtifactMediaType:  request.ArtifactMediaType,
	}
}
