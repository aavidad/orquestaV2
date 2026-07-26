package stages

const (
	v1VersionValue      = "orquesta.wizard.stages.v1"
	currentVersionValue = v1VersionValue
)

// frozenV1CatalogDigestValue prevents semantic edits from being published
// under an already durable catalog version. Additive revisions must introduce
// a new version and preserve this builder and digest.
const frozenV1CatalogDigestValue = "087d3ef59a49fdf5553b381d90dde60c8af58e0c9c5bcd099bda09953a913df4"

func CurrentVersion() CatalogVersion {
	return mustCatalogVersion(currentVersionValue)
}

// VersionV1 remains addressable after CurrentVersion advances.
func VersionV1() CatalogVersion {
	return mustCatalogVersion(v1VersionValue)
}

func TemplateRefs() []TemplateRef {
	values := make([]TemplateRef, len(requiredTemplateValues))
	for index, value := range requiredTemplateValues {
		values[index] = mustTemplateRef(value)
	}
	return values
}

// BuiltIn returns the current canonical pure V23 catalog. Invalid or
// version-drifting built-in data is a programming defect and therefore panics.
func BuiltIn() Catalog {
	value, err := BuiltInVersion(CurrentVersion())
	if err != nil {
		panic(err)
	}
	return value
}

// BuiltInVersion resolves immutable built-in catalog history. New versions
// are added as new switch branches; an existing builder and frozen digest are
// never replaced.
func BuiltInVersion(version CatalogVersion) (Catalog, error) {
	var value Catalog
	var frozen string
	switch version.value {
	case v1VersionValue:
		value = builtInV1()
		frozen = frozenV1CatalogDigestValue
	default:
		return Catalog{}, domainError(
			ErrorCatalogVersionUnknown,
			"catalog.version",
		)
	}
	if value.Digest().value != frozen {
		return Catalog{}, domainError(
			ErrorCatalogDigestMismatch,
			"catalog.digest",
		)
	}
	return value, nil
}

func builtInV1() Catalog {
	templates := []Template{
		researchTemplate(),
		buildAppTemplate(),
		changeAppTemplate(),
		domainProductionTemplate(),
		deployTemplate(),
		selfChangeTemplate(),
	}
	value, err := NewCatalog(CatalogInput{
		Version: VersionV1(), Templates: templates,
	})
	if err != nil {
		panic(err)
	}
	return value
}

func researchTemplate() Template {
	key := "research"
	stages := linearStages(key,
		stageDefinition{"scope", PhaseDiscover},
		stageDefinition{"evidence", PhaseProduce},
		stageDefinition{"synthesis", PhaseProduce},
		stageDefinition{"verification", PhaseVerify},
	)
	return mustTemplate(TemplateInput{
		Ref:                   mustTemplateRef("template:" + key),
		RoadmapCapabilityRefs: roadmapRefs(),
		Stages:                stages,
		Units: []Unit{
			builtinUnit(key, "frame_question", "scope", nil, "research_lead",
				[]string{"artifacts/research/brief"}, TestContract,
				[]string{"question_bounded", "sources_policy_declared"},
				EffectReadContext, false, focusedPolicy()),
			builtinUnit(key, "collect_evidence", "evidence",
				[]string{"frame_question"}, "researcher",
				[]string{"artifacts/research/evidence"}, TestContract,
				[]string{"claims_traceable", "evidence_deduplicated"},
				EffectRequestExternal, false, sensitivePolicy()),
			builtinUnit(key, "synthesize_findings", "synthesis",
				[]string{"collect_evidence"}, "analyst",
				[]string{"artifacts/research/findings"}, TestReview,
				[]string{"findings_answer_question", "uncertainty_explicit"},
				EffectWriteArtifact, false, focusedPolicy()),
			builtinUnit(key, "verify_findings", "verification",
				[]string{"synthesize_findings"}, "reviewer",
				[]string{"artifacts/research/review"}, TestReview,
				[]string{"citations_resolve", "contradictions_accounted"},
				EffectWriteArtifact, false, deepReviewPolicy()),
		},
	})
}

func buildAppTemplate() Template {
	key := "build_app"
	stages := linearStages(key,
		stageDefinition{"scope", PhaseDiscover},
		stageDefinition{"architecture", PhaseDesign},
		stageDefinition{"construction", PhaseProduce},
		stageDefinition{"integration", PhaseProduce},
		stageDefinition{"verification", PhaseVerify},
	)
	return mustTemplate(TemplateInput{
		Ref:                   mustTemplateRef("template:" + key),
		RoadmapCapabilityRefs: roadmapRefs(),
		Stages:                stages,
		Units: []Unit{
			builtinUnit(key, "bound_scope", "scope", nil, "product_analyst",
				[]string{"artifacts/build_app/scope"}, TestContract,
				[]string{"scope_confirmed", "constraints_explicit"},
				EffectWriteArtifact, false, focusedPolicy()),
			builtinUnit(key, "design_architecture", "architecture",
				[]string{"bound_scope"}, "architect",
				[]string{"artifacts/build_app/architecture"}, TestContract,
				[]string{"boundaries_declared", "ports_identified"},
				EffectWriteArtifact, false, deepReviewPolicy()),
			builtinUnit(key, "implement_domain", "construction",
				[]string{"design_architecture"}, "implementer",
				[]string{"source/domain", "source/application"}, TestUnit,
				[]string{"domain_rules_pass", "dependency_direction_clean"},
				EffectMutateWorkspace, false, focusedPolicy()),
			builtinUnit(key, "implement_interfaces", "construction",
				[]string{"design_architecture"}, "implementer",
				[]string{"source/interfaces", "source/adapters"}, TestContract,
				[]string{"interfaces_match_ports", "errors_typed"},
				EffectMutateWorkspace, false, focusedPolicy()),
			builtinUnit(key, "integrate_slices", "integration",
				[]string{"implement_domain", "implement_interfaces"}, "integrator",
				[]string{"source/bootstrap", "source/integration"}, TestIntegration,
				[]string{"primary_flow_passes", "composition_isolated"},
				EffectMutateWorkspace, false, deepReviewPolicy()),
			builtinUnit(key, "verify_candidate", "verification",
				[]string{"integrate_slices"}, "reviewer",
				[]string{"artifacts/build_app/review"}, TestSecurity,
				[]string{"required_tests_pass", "architecture_guard_passes", "risks_recorded"},
				EffectWriteArtifact, false, deepReviewPolicy()),
		},
	})
}

func changeAppTemplate() Template {
	key := "change_app"
	stages := linearStages(key,
		stageDefinition{"assessment", PhaseDiscover},
		stageDefinition{"plan", PhaseDesign},
		stageDefinition{"change", PhaseProduce},
		stageDefinition{"verification", PhaseVerify},
	)
	return mustTemplate(TemplateInput{
		Ref:                   mustTemplateRef("template:" + key),
		RoadmapCapabilityRefs: roadmapRefs(),
		Stages:                stages,
		Units: []Unit{
			builtinUnit(key, "assess_baseline", "assessment", nil, "maintainer",
				[]string{"artifacts/change_app/baseline"}, TestContract,
				[]string{"baseline_captured", "impact_bounded"},
				EffectReadContext, false, focusedPolicy()),
			builtinUnit(key, "plan_change", "plan",
				[]string{"assess_baseline"}, "architect",
				[]string{"artifacts/change_app/plan"}, TestReview,
				[]string{"write_set_bounded", "regression_plan_present"},
				EffectWriteArtifact, false, deepReviewPolicy()),
			builtinUnit(key, "apply_change", "change",
				[]string{"plan_change"}, "implementer",
				[]string{"source/change"}, TestUnit,
				[]string{"requested_behavior_present", "unrelated_behavior_preserved"},
				EffectMutateWorkspace, false, focusedPolicy()),
			builtinUnit(key, "review_change", "verification",
				[]string{"apply_change"}, "reviewer",
				[]string{"artifacts/change_app/review"}, TestIntegration,
				[]string{"regressions_absent", "change_traceable", "risks_recorded"},
				EffectWriteArtifact, false, deepReviewPolicy()),
		},
	})
}

func domainProductionTemplate() Template {
	key := "domain_production"
	stages := linearStages(key,
		stageDefinition{"brief", PhaseDiscover},
		stageDefinition{"production", PhaseProduce},
		stageDefinition{"quality", PhaseVerify},
		stageDefinition{"package", PhaseRelease},
	)
	return mustTemplate(TemplateInput{
		Ref:                   mustTemplateRef("template:" + key),
		RoadmapCapabilityRefs: roadmapRefs(),
		Stages:                stages,
		Units: []Unit{
			builtinUnit(key, "prepare_brief", "brief", nil, "domain_lead",
				[]string{"artifacts/domain_production/brief"}, TestContract,
				[]string{"audience_defined", "source_material_scoped"},
				EffectReadContext, false, sensitivePolicy()),
			builtinUnit(key, "produce_artifact", "production",
				[]string{"prepare_brief"}, "domain_author",
				[]string{"artifacts/domain_production/draft"}, TestUnit,
				[]string{"coverage_complete", "format_contract_met"},
				EffectWriteArtifact, false, focusedPolicy()),
			builtinUnit(key, "quality_review", "quality",
				[]string{"produce_artifact"}, "domain_reviewer",
				[]string{"artifacts/domain_production/quality"}, TestReview,
				[]string{"domain_accuracy_accepted", "provenance_preserved"},
				EffectWriteArtifact, false, deepReviewPolicy()),
			builtinUnit(key, "package_candidate", "package",
				[]string{"quality_review"}, "release_manager",
				[]string{"artifacts/domain_production/package"}, TestPostcondition,
				[]string{"package_complete", "publication_scope_explicit"},
				EffectPublish, true, publishPolicy()),
		},
	})
}

func deployTemplate() Template {
	key := "deploy"
	stages := linearStages(key,
		stageDefinition{"preparation", PhaseDiscover},
		stageDefinition{"validation", PhaseVerify},
		stageDefinition{"application", PhaseRelease},
		stageDefinition{"postcondition", PhaseVerify},
	)
	return mustTemplate(TemplateInput{
		Ref:                   mustTemplateRef("template:" + key),
		RoadmapCapabilityRefs: roadmapRefs(),
		Stages:                stages,
		Units: []Unit{
			builtinUnit(key, "prepare_release", "preparation", nil, "operator",
				[]string{"artifacts/deploy/plan"}, TestContract,
				[]string{"target_ref_explicit", "rollback_plan_present"},
				EffectReadContext, false, sensitivePolicy()),
			builtinUnit(key, "validate_release", "validation",
				[]string{"prepare_release"}, "release_reviewer",
				[]string{"artifacts/deploy/validation"}, TestSecurity,
				[]string{"candidate_digest_fixed", "authorization_scope_bounded"},
				EffectWriteArtifact, false, deepReviewPolicy()),
			builtinUnit(key, "apply_release", "application",
				[]string{"validate_release"}, "operator",
				[]string{"artifacts/deploy/attempt"}, TestPostcondition,
				[]string{"attempt_receipt_recorded", "mutation_matches_approval"},
				EffectMutateExternal, true, mutationPolicy()),
			builtinUnit(key, "verify_release", "postcondition",
				[]string{"apply_release"}, "release_reviewer",
				[]string{"artifacts/deploy/postcondition"}, TestPostcondition,
				[]string{"health_evidence_present", "rollback_status_recorded"},
				EffectWriteArtifact, false, deepReviewPolicy()),
		},
	})
}

func selfChangeTemplate() Template {
	key := "self_change"
	stages := linearStages(key,
		stageDefinition{"assessment", PhaseDiscover},
		stageDefinition{"design", PhaseDesign},
		stageDefinition{"implementation", PhaseProduce},
		stageDefinition{"independent_review", PhaseVerify},
		stageDefinition{"closure", PhaseVerify},
	)
	return mustTemplate(TemplateInput{
		Ref:                   mustTemplateRef("template:" + key),
		RoadmapCapabilityRefs: roadmapRefs(),
		Stages:                stages,
		Units: []Unit{
			builtinUnit(key, "assess_invariant", "assessment", nil, "maintainer",
				[]string{"artifacts/self_change/assessment"}, TestContract,
				[]string{"invariant_named", "authority_unchanged"},
				EffectReadContext, false, deepReviewPolicy()),
			builtinUnit(key, "design_change", "design",
				[]string{"assess_invariant"}, "architect",
				[]string{"artifacts/self_change/design"}, TestSecurity,
				[]string{"privileges_not_expanded", "rollback_defined"},
				EffectWriteArtifact, false, deepReviewPolicy()),
			builtinUnit(key, "implement_change", "implementation",
				[]string{"design_change"}, "implementer",
				[]string{"source/self_change"}, TestUnit,
				[]string{"change_bounded", "existing_lifecycle_reused"},
				EffectMutateSelf, true, selfMutationPolicy()),
			builtinUnit(key, "independent_review", "independent_review",
				[]string{"implement_change"}, "independent_reviewer",
				[]string{"artifacts/self_change/review"}, TestReview,
				[]string{"review_independent", "security_invariants_pass"},
				EffectWriteArtifact, false, deepReviewPolicy()),
			builtinUnit(key, "verify_self_change", "closure",
				[]string{"independent_review"}, "maintainer",
				[]string{"artifacts/self_change/verification"}, TestIntegration,
				[]string{"required_tests_pass", "recovery_verified", "evidence_complete"},
				EffectWriteArtifact, false, deepReviewPolicy()),
		},
	})
}

type stageDefinition struct {
	name  string
	phase PhaseKind
}

func linearStages(template string, definitions ...stageDefinition) []Stage {
	stages := make([]Stage, len(definitions))
	for index, definition := range definitions {
		var dependencies []StageRef
		if index > 0 {
			dependencies = []StageRef{
				mustStageRef("stage:" + template + "." + definitions[index-1].name),
			}
		}
		stages[index] = mustStage(StageInput{
			Ref:      mustStageRef("stage:" + template + "." + definition.name),
			Sequence: (index + 1) * 10, Phase: definition.phase,
			DependsOn: dependencies,
		})
	}
	return stages
}

func builtinUnit(
	template string,
	name string,
	stage string,
	dependencies []string,
	role string,
	writeSet []string,
	testKind TestKind,
	criteria []string,
	effectKind EffectKind,
	effectApproval bool,
	policy ExecutionPolicy,
) Unit {
	criterionValues := make([]Criterion, len(criteria))
	criterionRefs := make([]CriterionRef, len(criteria))
	for index, criterion := range criteria {
		ref := mustCriterionRef(
			"criterion:" + template + "." + name + "." + criterion,
		)
		criterionValues[index] = Criterion{Ref: ref}
		criterionRefs[index] = ref
	}
	dependencyRefs := make([]UnitRef, len(dependencies))
	for index, dependency := range dependencies {
		dependencyRefs[index] = mustUnitRef("unit:" + template + "." + dependency)
	}
	scopes := make([]WriteScope, len(writeSet))
	for index, path := range writeSet {
		scopes[index] = WriteScope{Path: path}
	}
	return mustUnit(UnitInput{
		Ref:       mustUnitRef("unit:" + template + "." + name),
		StageRef:  mustStageRef("stage:" + template + "." + stage),
		DependsOn: dependencyRefs,
		Role:      mustRoleRef("role:" + role),
		WriteSet:  scopes,
		RequiredTests: []RequiredTest{{
			Ref:  mustTestRef("test:" + template + "." + name),
			Kind: testKind, CriterionRefs: criterionRefs,
		}},
		AcceptanceCriteria: criterionValues,
		Effects: []Effect{{
			Ref:  mustEffectRef("effect:" + template + "." + name),
			Kind: effectKind, ApprovalRequired: effectApproval,
		}},
		Policy: policy,
	})
}

func roadmapRefs() []RoadmapCapabilityRef {
	return []RoadmapCapabilityRef{
		mustRoadmapCapabilityRef("WIZ-11"),
		mustRoadmapCapabilityRef("WIZ-24"),
	}
}

func focusedPolicy() ExecutionPolicy {
	return ExecutionPolicy{
		Effort: EffortFocused,
		Security: SecurityPolicy{
			Risk: RiskLow, Approval: ApprovalNone, LeastPrivilege: true,
		},
	}
}

func sensitivePolicy() ExecutionPolicy {
	return ExecutionPolicy{
		Effort: EffortFocused,
		Security: SecurityPolicy{
			Risk: RiskModerate, Approval: ApprovalNone,
			LeastPrivilege: true, SensitiveInputs: true,
		},
	}
}

func deepReviewPolicy() ExecutionPolicy {
	return ExecutionPolicy{
		Effort: EffortDeep,
		Security: SecurityPolicy{
			Risk: RiskModerate, Approval: ApprovalIndependentReview,
			LeastPrivilege: true,
		},
	}
}

func mutationPolicy() ExecutionPolicy {
	return ExecutionPolicy{
		Effort: EffortDeep,
		Security: SecurityPolicy{
			Risk: RiskHigh, Approval: ApprovalBeforeMutation,
			LeastPrivilege: true, SensitiveInputs: true,
		},
	}
}

func publishPolicy() ExecutionPolicy {
	return ExecutionPolicy{
		Effort: EffortDeep,
		Security: SecurityPolicy{
			Risk: RiskHigh, Approval: ApprovalBeforePublish,
			LeastPrivilege: true, SensitiveInputs: true,
		},
	}
}

func selfMutationPolicy() ExecutionPolicy {
	return ExecutionPolicy{
		Effort: EffortDeep,
		Security: SecurityPolicy{
			Risk: RiskCritical, Approval: ApprovalIndependentReview,
			LeastPrivilege: true, SensitiveInputs: true,
		},
	}
}

func mustCatalogVersion(value string) CatalogVersion {
	ref, err := NewCatalogVersion(value)
	if err != nil {
		panic(err)
	}
	return ref
}

func mustTemplateRef(value string) TemplateRef {
	ref, err := NewTemplateRef(value)
	if err != nil {
		panic(err)
	}
	return ref
}

func mustStageRef(value string) StageRef {
	ref, err := NewStageRef(value)
	if err != nil {
		panic(err)
	}
	return ref
}

func mustUnitRef(value string) UnitRef {
	ref, err := NewUnitRef(value)
	if err != nil {
		panic(err)
	}
	return ref
}

func mustRoleRef(value string) RoleRef {
	ref, err := NewRoleRef(value)
	if err != nil {
		panic(err)
	}
	return ref
}

func mustTestRef(value string) TestRef {
	ref, err := NewTestRef(value)
	if err != nil {
		panic(err)
	}
	return ref
}

func mustCriterionRef(value string) CriterionRef {
	ref, err := NewCriterionRef(value)
	if err != nil {
		panic(err)
	}
	return ref
}

func mustEffectRef(value string) EffectRef {
	ref, err := NewEffectRef(value)
	if err != nil {
		panic(err)
	}
	return ref
}

func mustRoadmapCapabilityRef(value string) RoadmapCapabilityRef {
	ref, err := NewRoadmapCapabilityRef(value)
	if err != nil {
		panic(err)
	}
	return ref
}

func mustStage(input StageInput) Stage {
	value, err := NewStage(input)
	if err != nil {
		panic(err)
	}
	return value
}

func mustUnit(input UnitInput) Unit {
	value, err := NewUnit(input)
	if err != nil {
		panic(err)
	}
	return value
}

func mustTemplate(input TemplateInput) Template {
	value, err := NewTemplate(input)
	if err != nil {
		panic(err)
	}
	return value
}
