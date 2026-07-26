// Package stages defines the pure, immutable V23 catalog of plan templates.
// It owns no Goal lifecycle and knows no application writer, provider,
// transport, persistence, workspace implementation or configuration source.
package stages

type CatalogVersion struct{ value string }
type TemplateRef struct{ value string }
type StageRef struct{ value string }
type UnitRef struct{ value string }
type RoleRef struct{ value string }
type TestRef struct{ value string }
type CriterionRef struct{ value string }
type EffectRef struct{ value string }

func NewCatalogVersion(value string) (CatalogVersion, error) {
	if !validMachineValue(value) {
		return CatalogVersion{}, domainError(ErrorInvalidArgument, "catalog_version")
	}
	return CatalogVersion{value: value}, nil
}

func NewTemplateRef(value string) (TemplateRef, error) {
	value, err := validRef("template", value)
	return TemplateRef{value: value}, err
}

func NewStageRef(value string) (StageRef, error) {
	value, err := validRef("stage", value)
	return StageRef{value: value}, err
}

func NewUnitRef(value string) (UnitRef, error) {
	value, err := validRef("unit", value)
	return UnitRef{value: value}, err
}

func NewRoleRef(value string) (RoleRef, error) {
	value, err := validRef("role", value)
	return RoleRef{value: value}, err
}

func NewTestRef(value string) (TestRef, error) {
	value, err := validRef("test", value)
	return TestRef{value: value}, err
}

func NewCriterionRef(value string) (CriterionRef, error) {
	value, err := validRef("criterion", value)
	return CriterionRef{value: value}, err
}

func NewEffectRef(value string) (EffectRef, error) {
	value, err := validRef("effect", value)
	return EffectRef{value: value}, err
}

func (value CatalogVersion) String() string { return value.value }
func (value TemplateRef) String() string    { return value.value }
func (value StageRef) String() string       { return value.value }
func (value UnitRef) String() string        { return value.value }
func (value RoleRef) String() string        { return value.value }
func (value TestRef) String() string        { return value.value }
func (value CriterionRef) String() string   { return value.value }
func (value EffectRef) String() string      { return value.value }

// RoadmapCapabilityRef provides traceability only. Presence never means a
// capability is implemented, wired, exercised, accredited or sealed.
type RoadmapCapabilityRef struct{ value string }

func NewRoadmapCapabilityRef(value string) (RoadmapCapabilityRef, error) {
	if len(value) < 5 || len(value) > 32 || value[:4] != "WIZ-" {
		return RoadmapCapabilityRef{}, domainError(
			ErrorInvalidRef,
			"roadmap_capability_ref",
		)
	}
	for _, char := range value[4:] {
		if char < '0' || char > '9' {
			return RoadmapCapabilityRef{}, domainError(
				ErrorInvalidRef,
				"roadmap_capability_ref",
			)
		}
	}
	return RoadmapCapabilityRef{value: value}, nil
}

func (value RoadmapCapabilityRef) String() string { return value.value }

type PhaseKind string

const (
	PhaseDiscover PhaseKind = "discover"
	PhaseDesign   PhaseKind = "design"
	PhaseProduce  PhaseKind = "produce"
	PhaseVerify   PhaseKind = "verify"
	PhaseRelease  PhaseKind = "release"
)

type TestKind string

const (
	TestUnit          TestKind = "unit"
	TestContract      TestKind = "contract"
	TestIntegration   TestKind = "integration"
	TestSecurity      TestKind = "security"
	TestReview        TestKind = "review"
	TestPostcondition TestKind = "postcondition"
)

type EffectKind string

const (
	EffectReadContext     EffectKind = "read_context"
	EffectWriteArtifact   EffectKind = "write_artifact"
	EffectMutateWorkspace EffectKind = "mutate_workspace"
	EffectRequestExternal EffectKind = "request_external"
	EffectMutateExternal  EffectKind = "mutate_external"
	EffectPublish         EffectKind = "publish"
	EffectMutateSelf      EffectKind = "mutate_self"
)

type EffortLevel string

const (
	EffortRoutine EffortLevel = "routine"
	EffortFocused EffortLevel = "focused"
	EffortDeep    EffortLevel = "deep"
)

type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskModerate RiskLevel = "moderate"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

type ApprovalPolicy string

const (
	ApprovalNone              ApprovalPolicy = "none"
	ApprovalBeforeMutation    ApprovalPolicy = "before_mutation"
	ApprovalBeforePublish     ApprovalPolicy = "before_publish"
	ApprovalIndependentReview ApprovalPolicy = "independent_review"
)

type SecurityPolicy struct {
	Risk            RiskLevel
	Approval        ApprovalPolicy
	LeastPrivilege  bool
	SensitiveInputs bool
}

type ExecutionPolicy struct {
	Effort   EffortLevel
	Security SecurityPolicy
}

type WriteScope struct {
	Path string
}

type RequiredTest struct {
	Ref           TestRef
	Kind          TestKind
	CriterionRefs []CriterionRef
}

type Criterion struct {
	Ref CriterionRef
}

type Effect struct {
	Ref              EffectRef
	Kind             EffectKind
	ApprovalRequired bool
}

type StageInput struct {
	Ref       StageRef
	Sequence  int
	Phase     PhaseKind
	DependsOn []StageRef
}

type Stage struct {
	ref       StageRef
	sequence  int
	phase     PhaseKind
	dependsOn []StageRef
}

func NewStage(input StageInput) (Stage, error) {
	value := Stage{
		ref: input.Ref, sequence: input.Sequence, phase: input.Phase,
		dependsOn: cloneStageRefs(input.DependsOn),
	}
	if err := validateStage(value, "stage"); err != nil {
		return Stage{}, err
	}
	sortStageRefs(value.dependsOn)
	return value, nil
}

func (value Stage) Ref() StageRef         { return value.ref }
func (value Stage) Sequence() int         { return value.sequence }
func (value Stage) Phase() PhaseKind      { return value.phase }
func (value Stage) DependsOn() []StageRef { return cloneStageRefs(value.dependsOn) }

type UnitInput struct {
	Ref                UnitRef
	StageRef           StageRef
	DependsOn          []UnitRef
	Role               RoleRef
	WriteSet           []WriteScope
	RequiredTests      []RequiredTest
	AcceptanceCriteria []Criterion
	Effects            []Effect
	Policy             ExecutionPolicy
}

type Unit struct {
	ref                UnitRef
	stageRef           StageRef
	dependsOn          []UnitRef
	role               RoleRef
	writeSet           []WriteScope
	requiredTests      []RequiredTest
	acceptanceCriteria []Criterion
	effects            []Effect
	policy             ExecutionPolicy
}

func NewUnit(input UnitInput) (Unit, error) {
	value := Unit{
		ref: input.Ref, stageRef: input.StageRef,
		dependsOn: cloneUnitRefs(input.DependsOn), role: input.Role,
		writeSet:           cloneWriteScopes(input.WriteSet),
		requiredTests:      cloneRequiredTests(input.RequiredTests),
		acceptanceCriteria: cloneCriteria(input.AcceptanceCriteria),
		effects:            cloneEffects(input.Effects), policy: input.Policy,
	}
	if err := validateUnit(value, "unit"); err != nil {
		return Unit{}, err
	}
	sortUnitRefs(value.dependsOn)
	sortWriteScopes(value.writeSet)
	sortRequiredTests(value.requiredTests)
	sortCriteria(value.acceptanceCriteria)
	sortEffects(value.effects)
	return value, nil
}

func (value Unit) Ref() UnitRef                    { return value.ref }
func (value Unit) StageRef() StageRef              { return value.stageRef }
func (value Unit) DependsOn() []UnitRef            { return cloneUnitRefs(value.dependsOn) }
func (value Unit) Role() RoleRef                   { return value.role }
func (value Unit) WriteSet() []WriteScope          { return cloneWriteScopes(value.writeSet) }
func (value Unit) RequiredTests() []RequiredTest   { return cloneRequiredTests(value.requiredTests) }
func (value Unit) AcceptanceCriteria() []Criterion { return cloneCriteria(value.acceptanceCriteria) }
func (value Unit) Effects() []Effect               { return cloneEffects(value.effects) }
func (value Unit) Policy() ExecutionPolicy         { return value.policy }

type TemplateInput struct {
	Ref                   TemplateRef
	RoadmapCapabilityRefs []RoadmapCapabilityRef
	Stages                []Stage
	Units                 []Unit
}

type Template struct {
	ref                   TemplateRef
	roadmapCapabilityRefs []RoadmapCapabilityRef
	stages                []Stage
	units                 []Unit
}

func NewTemplate(input TemplateInput) (Template, error) {
	value := Template{
		ref: input.Ref,
		roadmapCapabilityRefs: cloneRoadmapCapabilityRefs(
			input.RoadmapCapabilityRefs,
		),
		stages: cloneStages(input.Stages),
		units:  cloneUnits(input.Units),
	}
	if err := validateTemplate(value); err != nil {
		return Template{}, err
	}
	sortRoadmapCapabilityRefs(value.roadmapCapabilityRefs)
	sortStages(value.stages)
	sortUnits(value.units, value.stages)
	return value, nil
}

func (value Template) Ref() TemplateRef {
	return value.ref
}

// RoadmapCapabilityRefs are traceability metadata, never accreditation.
func (value Template) RoadmapCapabilityRefs() []RoadmapCapabilityRef {
	return cloneRoadmapCapabilityRefs(value.roadmapCapabilityRefs)
}

func (value Template) Stages() []Stage { return cloneStages(value.stages) }
func (value Template) Units() []Unit   { return cloneUnits(value.units) }
