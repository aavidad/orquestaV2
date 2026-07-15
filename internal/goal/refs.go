package goal

import "strings"

type ActorRef struct{ value string }
type ProjectRef struct{ value string }
type IntentRef struct{ value string }
type AppSpecRef struct{ value string }
type GoalRef struct{ value string }
type WorkItemRef struct{ value string }
type PhaseRef struct{ value string }
type PhaseTemplateRef struct{ value string }
type InputRef struct{ value string }
type CriterionRef struct{ value string }
type SkillRef struct{ value string }
type ToolRef struct{ value string }
type CapabilityRef struct{ value string }
type ExecutionRef struct{ value string }
type ArtifactRef struct{ value string }
type AttestationRef struct{ value string }

func NewActorRef(value string) (ActorRef, error) {
	value, err := validOpaqueRef("actor_ref", value)
	return ActorRef{value: value}, err
}

func NewProjectRef(value string) (ProjectRef, error) {
	value, err := validOpaqueRef("project_ref", value)
	return ProjectRef{value: value}, err
}

func NewIntentRef(value string) (IntentRef, error) {
	value, err := validOpaqueRef("intent_ref", value)
	return IntentRef{value: value}, err
}

func NewAppSpecRef(value string) (AppSpecRef, error) {
	value, err := validOpaqueRef("app_spec_ref", value)
	return AppSpecRef{value: value}, err
}

func NewGoalRef(value string) (GoalRef, error) {
	value, err := validOpaqueRef("goal_ref", value)
	return GoalRef{value: value}, err
}

func NewWorkItemRef(value string) (WorkItemRef, error) {
	value, err := validOpaqueRef("work_item_ref", value)
	return WorkItemRef{value: value}, err
}

func NewPhaseRef(value string) (PhaseRef, error) {
	value, err := validOpaqueRef("phase_ref", value)
	return PhaseRef{value: value}, err
}

func NewPhaseTemplateRef(value string) (PhaseTemplateRef, error) {
	value, err := validOpaqueRef("phase_template_ref", value)
	return PhaseTemplateRef{value: value}, err
}

func NewInputRef(value string) (InputRef, error) {
	value, err := validOpaqueRef("input_ref", value)
	return InputRef{value: value}, err
}

func NewCriterionRef(value string) (CriterionRef, error) {
	value, err := validOpaqueRef("criterion_ref", value)
	return CriterionRef{value: value}, err
}

func NewSkillRef(value string) (SkillRef, error) {
	value, err := validOpaqueRef("skill_ref", value)
	return SkillRef{value: value}, err
}

func NewToolRef(value string) (ToolRef, error) {
	value, err := validOpaqueRef("tool_ref", value)
	return ToolRef{value: value}, err
}

func NewCapabilityRef(value string) (CapabilityRef, error) {
	value, err := validOpaqueRef("capability_ref", value)
	return CapabilityRef{value: value}, err
}

func NewExecutionRef(value string) (ExecutionRef, error) {
	value, err := validOpaqueRef("execution_ref", value)
	return ExecutionRef{value: value}, err
}

func NewArtifactRef(value string) (ArtifactRef, error) {
	value, err := validOpaqueRef("artifact_ref", value)
	return ArtifactRef{value: value}, err
}

func NewAttestationRef(value string) (AttestationRef, error) {
	value, err := validOpaqueRef("attestation_ref", value)
	return AttestationRef{value: value}, err
}

func (ref ActorRef) String() string         { return ref.value }
func (ref ProjectRef) String() string       { return ref.value }
func (ref IntentRef) String() string        { return ref.value }
func (ref AppSpecRef) String() string       { return ref.value }
func (ref GoalRef) String() string          { return ref.value }
func (ref WorkItemRef) String() string      { return ref.value }
func (ref PhaseRef) String() string         { return ref.value }
func (ref PhaseTemplateRef) String() string { return ref.value }
func (ref InputRef) String() string         { return ref.value }
func (ref CriterionRef) String() string     { return ref.value }
func (ref SkillRef) String() string         { return ref.value }
func (ref ToolRef) String() string          { return ref.value }
func (ref CapabilityRef) String() string    { return ref.value }
func (ref ExecutionRef) String() string     { return ref.value }
func (ref ArtifactRef) String() string      { return ref.value }
func (ref AttestationRef) String() string   { return ref.value }

func validOpaqueRef(field, value string) (string, error) {
	if value == "" || strings.TrimSpace(value) != value || strings.ContainsRune(value, '\x00') {
		return "", domainError(ErrorInvalidRef, field)
	}
	return value, nil
}

func validActorRef(ref ActorRef) bool                 { return ref.value != "" }
func validProjectRef(ref ProjectRef) bool             { return ref.value != "" }
func validIntentRef(ref IntentRef) bool               { return ref.value != "" }
func validAppSpecRef(ref AppSpecRef) bool             { return ref.value != "" }
func validGoalRef(ref GoalRef) bool                   { return ref.value != "" }
func validWorkItemRef(ref WorkItemRef) bool           { return ref.value != "" }
func validPhaseRef(ref PhaseRef) bool                 { return ref.value != "" }
func validPhaseTemplateRef(ref PhaseTemplateRef) bool { return ref.value != "" }
func validInputRef(ref InputRef) bool                 { return ref.value != "" }
func validCriterionRef(ref CriterionRef) bool         { return ref.value != "" }
func validSkillRef(ref SkillRef) bool                 { return ref.value != "" }
func validToolRef(ref ToolRef) bool                   { return ref.value != "" }
func validCapabilityRef(ref CapabilityRef) bool       { return ref.value != "" }
func validExecutionRef(ref ExecutionRef) bool         { return ref.value != "" }
func validArtifactRef(ref ArtifactRef) bool           { return ref.value != "" }
func validAttestationRef(ref AttestationRef) bool     { return ref.value != "" }
