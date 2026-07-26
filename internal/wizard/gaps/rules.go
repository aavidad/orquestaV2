package gaps

import (
	"strings"

	"orquesta/internal/wizard/catalog"
)

func (value *evaluation) detectCrossRuleIssues() error {
	value.detectR1()
	value.detectR2()
	if err := value.detectR3(); err != nil {
		return err
	}
	value.detectR4()
	value.detectR5()
	value.detectR6()
	value.detectR7()
	value.detectR8()
	return nil
}

func (value *evaluation) detectR1() {
	audience, selected := value.selections.resolved[DimensionU1]
	contradiction := false
	gap := false
	recommended := OptionRef("")
	switch value.facts.SharingIntent {
	case SharingAmbiguous:
		gap = !selected
	case SharingPersonal:
		gap = !selected
		contradiction = selected && audience.Option != optionRef(DimensionU1, "personal") &&
			audience.Option != optionRef(DimensionU1, "custom")
		recommended = optionRef(DimensionU1, "personal")
	case SharingShared:
		gap = !selected
		contradiction = selected && audience.Option != optionRef(DimensionU1, "team") &&
			audience.Option != optionRef(DimensionU1, "public") &&
			audience.Option != optionRef(DimensionU1, "custom")
		recommended = optionRef(DimensionU1, "team")
	}
	if !gap && !contradiction {
		return
	}
	kind := IssueGap
	if contradiction {
		kind = IssueContradiction
	}
	ref := ruleIssueRef(RuleR1)
	dimension := value.byDimension[DimensionU1]
	value.addIssue(Issue{
		ref: ref, kind: kind, ruleRef: RuleR1, dimension: DimensionU1,
		field:     dimension.slot,
		detailKey: MessageKey("wizard.gaps.rule.r1." + string(kind)),
		dependsOn: []DimensionRef{DimensionU1},
	})
	value.openDimension(DimensionU1, ref)
	if recommended != "" {
		value.recommendation[DimensionU1] = recommended
	}
}

func (value *evaluation) detectR2() {
	if value.facts.Surface == SurfaceUnspecified {
		return
	}
	selection, selected := value.selections.resolved[DimensionU2]
	if !selected {
		value.addRuleDimensionIssue(RuleR2, IssueGap, DimensionU2, nil)
		switch value.facts.Surface {
		case SurfaceNativeMobile:
			value.recommendation[DimensionU2] = optionRef(DimensionU2, "native_mobile")
		case SurfaceNativeDesktop:
			value.recommendation[DimensionU2] = optionRef(DimensionU2, "native_desktop")
		case SurfaceServerService, SurfaceKernelModule:
			value.recommendation[DimensionU2] = optionRef(DimensionU2, "no_ui")
		case SurfaceHumanUI:
			value.recommendation[DimensionU2] = optionRef(DimensionU2, "responsive_web")
		}
		return
	}
	incompatible := false
	recommended := OptionRef("")
	switch value.facts.Surface {
	case SurfaceServerService, SurfaceKernelModule:
		incompatible = selection.Option != optionRef(DimensionU2, "no_ui") &&
			selection.Option != optionRef(DimensionU2, "custom")
		recommended = optionRef(DimensionU2, "no_ui")
	case SurfaceNativeDesktop:
		incompatible = selection.Option != optionRef(DimensionU2, "native_desktop") &&
			selection.Option != optionRef(DimensionU2, "custom")
		recommended = optionRef(DimensionU2, "native_desktop")
	}
	if incompatible {
		value.addRuleDimensionIssue(RuleR2, IssueContradiction, DimensionU2, nil)
		value.recommendation[DimensionU2] = recommended
	}
}

func (value *evaluation) detectR3() error {
	if len(value.explicitPackRefs) == 0 {
		return nil
	}
	selectedRefs := make(map[string]struct{}, len(value.explicitPackRefs))
	for _, ref := range value.explicitPackRefs {
		selectedRefs[ref.String()] = struct{}{}
	}
	seenSlot := make(map[SlotKey]struct{})
	for _, pack := range catalog.BuiltIn().Packs() {
		if _, selected := selectedRefs[pack.Ref().String()]; !selected {
			continue
		}
		for _, source := range pack.Questions() {
			if !strings.HasPrefix(source.Slot().String(), "domains.") {
				continue
			}
			question := packQuestion(pack.Ref(), source)
			if _, duplicate := seenSlot[question.slot]; duplicate {
				continue
			}
			if _, resolved := value.questionSelections.resolved[question.ref]; resolved {
				seenSlot[question.slot] = struct{}{}
				continue
			}
			issueRef := IssueRef(
				"intake-issue:wizard.rule.r3." +
					strings.TrimPrefix(source.Ref().String(), "question:"),
			)
			kind := IssueGap
			if value.questionSelections.conflicts[question.ref] {
				kind = IssueContradiction
			}
			value.addIssue(Issue{
				ref: issueRef, kind: kind, ruleRef: RuleR3,
				field:     question.slot,
				detailKey: MessageKey("wizard.gaps.rule.r3." + string(kind)),
				dependsOn: []DimensionRef{DimensionU7},
			})
			question.derivedFrom = []IssueRef{issueRef}
			question.dependsOn = []QuestionRef{questionRef(DimensionU7)}
			value.packQuestions = append(value.packQuestions, question)
			seenSlot[question.slot] = struct{}{}
		}
	}
	return nil
}

func (value *evaluation) detectR4() {
	if !value.selectionIs(DimensionU4, "persisted_user_data", "external_source") {
		return
	}
	if _, selected := value.selections.resolved[DimensionT4]; selected {
		return
	}
	value.addRuleDimensionIssue(
		RuleR4, IssueGap, DimensionT4, []DimensionRef{DimensionU4},
	)
}

func (value *evaluation) detectR5() {
	question := ruleQuestion(
		RuleR5,
		"product.integration_governance",
		DecisionProduct,
		"service_auth_medium",
		"public_low", "service_auth_medium", "oauth_high",
	)
	if !value.integrationSelected() && len(value.explicitPackRefs) == 0 &&
		!value.questionSelections.conflicts[question.ref] {
		return
	}
	if _, resolved := value.questionSelections.resolved[question.ref]; resolved {
		return
	}
	if value.facts.IntegrationAuth == DeclarationDeclared &&
		value.facts.IntegrationCriticality == DeclarationDeclared &&
		!value.questionSelections.conflicts[question.ref] {
		return
	}
	ref := ruleIssueRef(RuleR5)
	kind := IssueGap
	if value.questionSelections.conflicts[question.ref] {
		kind = IssueContradiction
	}
	value.addIssue(Issue{
		ref: ref, kind: kind, ruleRef: RuleR5,
		field:     "product.integration_governance",
		detailKey: MessageKey("wizard.gaps.rule.r5." + string(kind)),
		dependsOn: []DimensionRef{DimensionU7},
	})
	question.derivedFrom = []IssueRef{ref}
	question.dependsOn = []QuestionRef{questionRef(DimensionU7)}
	value.extraQuestions = append(value.extraQuestions, question)
}

func (value *evaluation) detectR6() {
	if value.facts.Surface != SurfaceNativeMobile {
		return
	}
	selection, selected := value.selections.resolved[DimensionU2]
	if !selected {
		return
	}
	if selection.Option == optionRef(DimensionU2, "native_mobile") ||
		selection.Option == optionRef(DimensionU2, "custom") {
		return
	}
	value.addRuleDimensionIssue(
		RuleR6, IssueContradiction, DimensionU2, []DimensionRef{DimensionU2},
	)
	value.recommendation[DimensionU2] = optionRef(DimensionU2, "native_mobile")
}

func (value *evaluation) detectR7() {
	selection, selected := value.selections.resolved[DimensionU10]
	if !selected || value.deploymentCompatible(selection.Option) {
		return
	}
	value.addRuleDimensionIssue(
		RuleR7, IssueContradiction, DimensionU10, []DimensionRef{DimensionU2},
	)
	value.recommendation[DimensionU10] = value.compatibleDeploymentRecommendation()
}

func (value *evaluation) detectR8() {
	question := ruleQuestion(
		RuleR8,
		"product.target_users",
		DecisionProduct,
		"internal_team",
		"internal_team", "invited_customers", "public_users",
	)
	if !value.audienceShared() && !value.questionSelections.conflicts[question.ref] {
		return
	}
	if _, resolved := value.questionSelections.resolved[question.ref]; resolved {
		return
	}
	if value.facts.TargetUsers == DeclarationDeclared &&
		!value.questionSelections.conflicts[question.ref] {
		return
	}
	ref := ruleIssueRef(RuleR8)
	kind := IssueGap
	if value.questionSelections.conflicts[question.ref] {
		kind = IssueContradiction
	}
	value.addIssue(Issue{
		ref: ref, kind: kind, ruleRef: RuleR8,
		field:     "product.target_users",
		detailKey: MessageKey("wizard.gaps.rule.r8." + string(kind)),
		dependsOn: []DimensionRef{DimensionU1},
	})
	question.derivedFrom = []IssueRef{ref}
	question.dependsOn = []QuestionRef{questionRef(DimensionU1)}
	value.extraQuestions = append(value.extraQuestions, question)
}

func (value *evaluation) addRuleDimensionIssue(
	ruleRef RuleRef,
	kind IssueKind,
	dimensionRef DimensionRef,
	dependsOn []DimensionRef,
) {
	dimension := value.byDimension[dimensionRef]
	ref := ruleIssueRef(ruleRef)
	value.addIssue(Issue{
		ref: ref, kind: kind, ruleRef: ruleRef, dimension: dimensionRef,
		field: dimension.slot,
		detailKey: MessageKey(
			"wizard.gaps.rule." + lowerRuleRef(ruleRef) + "." + string(kind),
		),
		dependsOn: append([]DimensionRef(nil), dependsOn...),
	})
	value.openDimension(dimensionRef, ref)
}

func (value *evaluation) addIssue(issue Issue) {
	if _, duplicate := value.issueRefs[issue.ref]; duplicate {
		return
	}
	issue.dependsOn = uniqueDimensionsExcluding(issue.dependsOn, issue.dimension)
	value.issues = append(value.issues, issue)
	value.issueRefs[issue.ref] = struct{}{}
	if issue.dimension != "" {
		value.dependencies[issue.dimension] = append(
			value.dependencies[issue.dimension],
			issue.dependsOn...,
		)
	}
}

func (value *evaluation) openDimension(ref DimensionRef, cause IssueRef) {
	value.open[ref] = true
	for _, existing := range value.causes[ref] {
		if existing == cause {
			return
		}
	}
	value.causes[ref] = append(value.causes[ref], cause)
}
