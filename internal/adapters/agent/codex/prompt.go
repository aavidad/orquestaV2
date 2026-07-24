package codex

import (
	"fmt"
	"strconv"
	"strings"

	"orquesta/internal/ports"
)

// AgentPrompt contains human-facing data; protocol identity stays provider-stable.
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

// PromptRenderer keeps locale and catalog ownership in composition.
type PromptRenderer interface {
	RenderAgentPrompt(AgentPrompt) (string, error)
}

func (adapter *Adapter) renderAgentPrompt(request ports.AgentLaunchRequest) (string, error) {
	if adapter == nil || adapter.config.PromptRenderer == nil {
		return "", &Error{Code: CodePromptRendererInvalid}
	}
	rendered, err := adapter.config.PromptRenderer.RenderAgentPrompt(AgentPrompt{
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
	})
	if err != nil {
		return "", &Error{Code: CodePromptRenderFailed, Cause: err}
	}
	if strings.TrimSpace(rendered) == "" {
		return "", &Error{Code: CodePromptRenderFailed, Cause: fmt.Errorf("empty rendered prompt")}
	}
	return rendered, nil
}
