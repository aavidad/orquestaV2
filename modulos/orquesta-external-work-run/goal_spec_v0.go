package orquestaexternalworkrun

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

const (
	ExternalWorkGoalWorkProfileKindV0 = "domain_work"

	externalWorkGoalSpecEvidenceRefV0 = "evidence-ref-external-work-goal-spec-v0"
)

func BuildExternalWorkGoalWorkSpecV0(
	request StartExternalWorkRunRequestV0,
	config StartExternalWorkRunConfigV0,
) (orquestagoal.GoalWorkSpecV0, []ExternalWorkRunIssueV0) {
	request = normalizeStartExternalWorkRunRequestV0(request, config)
	if issues := validateStartExternalWorkRunGoalInputV0(request); len(issues) > 0 {
		return orquestagoal.GoalWorkSpecV0{}, issues
	}
	domainRequest, ok := orquestaappchange.DomainWorkJobRequestFromAppChangeV0(request.AppChangeRequest)
	if !ok {
		return orquestagoal.GoalWorkSpecV0{}, []ExternalWorkRunIssueV0{
			externalWorkRunIssueV0(ErrExternalWorkRunExternalWorkRequiredV0, "app_change_request.external_work"),
		}
	}
	spec := externalWorkGoalWorkSpecFromRequestV0(request, domainRequest)
	if issues := orquestagoal.ValidateGoalWorkSpecV0(spec); len(issues) > 0 {
		return orquestagoal.GoalWorkSpecV0{}, externalWorkRunGoalSpecIssuesV0(issues)
	}
	return spec, nil
}

func validateStartExternalWorkRunGoalInputV0(
	request StartExternalWorkRunRequestV0,
) []ExternalWorkRunIssueV0 {
	var issues []ExternalWorkRunIssueV0
	issues = append(issues, validateStartExternalWorkRunRefsV0(request)...)
	if request.SchemaVersion != StartExternalWorkRunRequestSchemaV0 {
		issues = append(issues, externalWorkRunIssueV0(ErrExternalWorkRunSchemaVersionV0, "schema_version"))
	}
	if strings.TrimSpace(request.OccurredAt) == "" {
		issues = append(issues, externalWorkRunIssueV0(ErrExternalWorkRunOccurredAtRequiredV0, "occurred_at"))
	} else if _, err := time.Parse(time.RFC3339, request.OccurredAt); err != nil {
		issues = append(issues, externalWorkRunIssueV0(ErrExternalWorkRunOccurredAtInvalidV0, "occurred_at"))
	}
	if !hasExternalWorkV0(request.AppChangeRequest) {
		issues = append(issues, externalWorkRunIssueV0(ErrExternalWorkRunExternalWorkRequiredV0, "app_change_request.external_work"))
	}
	issues = append(issues, validateExternalWorkRunContextV0(request.AppChangeRequest.ExternalWork)...)
	for _, issue := range orquestaappchange.ValidateAppChangeRequestV0(request.AppChangeRequest) {
		issues = append(issues, externalWorkRunIssueV0(
			ErrExternalWorkRunAppChangeInvalidV0+":"+issue.Code,
			"app_change_request."+issue.Field,
		))
	}
	return issues
}

func externalWorkGoalWorkSpecFromRequestV0(
	request StartExternalWorkRunRequestV0,
	domainRequest orquestadomainwork.DomainWorkJobRequestV0,
) orquestagoal.GoalWorkSpecV0 {
	change := request.AppChangeRequest
	work := change.ExternalWork
	token := externalWorkGoalTokenV0(request.RunRef, request.RequestID, change.ChangeRef)
	requiredTests := externalWorkGoalRequiredTestsV0(change.RequiredTests, work.RequiredTests, token)
	return orquestagoal.NormalizeGoalWorkSpecV0(orquestagoal.GoalWorkSpecV0{
		GoalRef:            "goal-ref-external-work-" + token,
		RequestRef:         request.RequestID,
		RunRef:             request.RunRef,
		ProjectRef:         request.ProjectRef,
		DomainRef:          firstExternalWorkRunStringV0(domainRequest.DomainRef, request.ProjectRef),
		WorkKind:           firstExternalWorkRunStringV0(domainRequest.WorkKind, "external_work"),
		WorkProfileKind:    ExternalWorkGoalWorkProfileKindV0,
		Objective:          firstExternalWorkRunStringV0(domainRequest.Objective, change.UserIntent, "Resolver trabajo externo de dominio."),
		DirectorKind:       orquestagoal.GoalDirectorKindRuntimeGoalV0,
		ContextRefs:        externalWorkGoalContextRefsV0(request, domainRequest),
		RuleRefs:           externalWorkGoalRuleRefsV0(),
		WriteSet:           externalWorkGoalWriteSetV0(change, domainRequest, token),
		RequiredTests:      requiredTests,
		AcceptanceCriteria: externalWorkGoalAcceptanceCriteriaV0(domainRequest),
		ArtifactContracts: []orquestagoal.GoalArtifactContractV0{{
			ArtifactRef:  "artifact-ref-external-work-" + token + "-domain-work",
			ArtifactType: externalWorkGoalArtifactTypeV0(domainRequest.WorkKind),
			Required:     true,
			EvidenceRefs: []string{externalWorkGoalSpecEvidenceRefV0},
		}},
		EvidenceRefs: externalWorkGoalEvidenceRefsV0(request, domainRequest, token),
		ClosurePolicy: orquestagoal.GoalClosurePolicyV0{
			RequireRequiredTests: len(requiredTests) > 0,
			RequireArtifacts:     true,
			RequireDomainReceipt: true,
			RequiredEvidenceRefs: []string{externalWorkGoalSpecEvidenceRefV0},
		},
		ReworkPolicy: orquestagoal.GoalReworkPolicyV0{
			PreferNewGoal:     true,
			MaxReworkGoals:    1,
			PreserveArtifacts: true,
		},
	})
}

func externalWorkGoalContextRefsV0(
	request StartExternalWorkRunRequestV0,
	domainRequest orquestadomainwork.DomainWorkJobRequestV0,
) []orquestagoal.GoalContextRefV0 {
	change := request.AppChangeRequest
	work := change.ExternalWork
	refs := make([]orquestagoal.GoalContextRefV0, 0, 12+
		len(work.InterfaceRefs)+
		len(domainRequest.WorkRefs)+
		len(domainRequest.InputRefs)+
		len(change.MetadataRefs)+
		len(work.InputFields))
	appendRef := func(kind, ref, purpose string, required bool) {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			return
		}
		refs = append(refs, orquestagoal.GoalContextRefV0{
			Kind:     kind,
			Ref:      ref,
			Purpose:  purpose,
			Required: required,
		})
	}
	appendRef("request", request.RequestID, "Solicitud external-work normalizada.", true)
	appendRef("run", request.RunRef, "Run operativo persistible por Orquesta.", true)
	appendRef("app_spec", request.AppSpecRef, "Spec logica external-work.", true)
	appendRef("app_change", change.ChangeRef, "Cambio propietario registrado por AppChange.", true)
	appendRef("app", change.AppRef, "Aplicacion externa propietaria.", true)
	appendRef("domain", domainRequest.DomainRef, "Dominio propietario del trabajo externo.", true)
	appendRef("work_kind", domainRequest.WorkKind, "Tipo de trabajo externo.", true)
	for _, ref := range work.InterfaceRefs {
		appendRef("domain_interface", ref, "Puerto/interfaz declarada por el dominio externo.", false)
	}
	for _, ref := range domainRequest.WorkRefs {
		appendRef("domain_work_ref", ref, "Referencia opaca del trabajo en la app externa.", false)
	}
	for _, ref := range domainRequest.InputRefs {
		appendRef("input_ref", ref, "Referencia opaca de contexto ya existente.", false)
	}
	for _, ref := range change.MetadataRefs {
		appendRef("metadata", ref, "Evidencia o metadata compacta asociada al cambio.", false)
	}
	for _, field := range work.InputFields {
		appendRef("input_field", "input-field-"+compactExternalWorkRunRefV0(field.Name), "Nombre de campo disponible; el payload queda fuera del spec Goal.", false)
	}
	return externalWorkGoalCompactContextRefsV0(refs)
}

func externalWorkGoalRuleRefsV0() []orquestagoal.GoalRuleRefV0 {
	return []orquestagoal.GoalRuleRefV0{
		{Kind: "repo", Ref: "AGENTS.md", Enforcement: orquestagoal.GoalRuleEnforcementHardV0},
		{Kind: "docs", Ref: "docs/orquesta_goal_first_codex_2026-06-25.md", Enforcement: orquestagoal.GoalRuleEnforcementAdvisoryV0},
		{Kind: "contract", Ref: "modulos/orquesta-external-work-run/docs/contratos.md", Enforcement: orquestagoal.GoalRuleEnforcementHardV0},
	}
}

func externalWorkGoalWriteSetV0(
	change orquestaappchange.AppChangeRequestV0,
	domainRequest orquestadomainwork.DomainWorkJobRequestV0,
	token string,
) []orquestagoal.GoalWriteScopeV0 {
	paths := compactExternalWorkRunStringsV0(change.AllowedWriteSet)
	if len(paths) == 0 {
		paths = []string{
			"domain-work/" +
				compactExternalWorkRunRefV0(firstExternalWorkRunStringV0(domainRequest.DomainRef, "external-work")) + "/" +
				compactExternalWorkRunRefV0(firstExternalWorkRunStringV0(domainRequest.WorkKind, "work")) + "/" +
				token,
		}
	}
	scopes := make([]orquestagoal.GoalWriteScopeV0, 0, len(paths))
	for _, path := range paths {
		scopes = append(scopes, orquestagoal.GoalWriteScopeV0{
			Path:    path,
			Purpose: "Artefactos y evidencias del trabajo externo; no implica filesystem interno compartido.",
		})
	}
	return scopes
}

func externalWorkGoalRequiredTestsV0(
	commands []string,
	domainTests []orquestadomainwork.DomainWorkRequiredTestV0,
	token string,
) []orquestagoal.GoalRequiredTestV0 {
	tests := make([]orquestagoal.GoalRequiredTestV0, 0, len(commands)+len(domainTests))
	seen := map[string]bool{}
	for index, command := range commands {
		command = strings.TrimSpace(command)
		if command == "" {
			continue
		}
		testRef := "test-ref-external-work-" + token + "-" + shortExternalWorkGoalHashV0(fmt.Sprintf("%02d:%s", index+1, command))
		if seen[testRef] {
			continue
		}
		seen[testRef] = true
		tests = append(tests, orquestagoal.GoalRequiredTestV0{
			TestRef: testRef,
			Command: command,
		})
	}
	for _, test := range domainTests {
		testRef := strings.TrimSpace(test.TestRef)
		if testRef == "" || seen[testRef] {
			continue
		}
		seen[testRef] = true
		tests = append(tests, orquestagoal.GoalRequiredTestV0{
			TestRef:                testRef,
			AcceptanceCriteria:     append([]string(nil), test.AcceptanceCriteria...),
			AcceptanceCriteriaRefs: append([]string(nil), test.AcceptanceCriteriaRefs...),
			EvidenceRefs:           append([]string(nil), test.EvidenceRefs...),
		})
	}
	if tests == nil {
		return []orquestagoal.GoalRequiredTestV0{}
	}
	return tests
}

func externalWorkGoalAcceptanceCriteriaV0(
	domainRequest orquestadomainwork.DomainWorkJobRequestV0,
) []string {
	criteria := append([]string(nil), domainRequest.AcceptanceCriteria...)
	for _, constraint := range domainRequest.Constraints {
		constraint = strings.TrimSpace(constraint)
		if constraint != "" {
			criteria = append(criteria, "Restriccion de dominio: "+constraint)
		}
	}
	criteria = compactExternalWorkRunStringsV0(criteria)
	if len(criteria) == 0 {
		return []string{"Entregar un artefacto aceptable por el contrato DomainWork y confirmable con receipt de dominio."}
	}
	return criteria
}

func externalWorkGoalArtifactTypeV0(workKind string) string {
	return orquestadomainwork.ExpectedDomainWorkArtifactTypeForWorkKindV0(workKind)
}

func externalWorkGoalEvidenceRefsV0(
	request StartExternalWorkRunRequestV0,
	domainRequest orquestadomainwork.DomainWorkJobRequestV0,
	token string,
) []string {
	return compactExternalWorkRunStringsV0(append(
		[]string{
			externalWorkGoalSpecEvidenceRefV0,
			"evidence-ref-external-work-" + token,
			"evidence-ref-domain-work-" + compactExternalWorkRunRefV0(firstExternalWorkRunStringV0(domainRequest.DomainRef, request.ProjectRef, "external-work")),
		},
		domainRequest.EvidenceRefs...,
	))
}

func externalWorkGoalCompactContextRefsV0(
	values []orquestagoal.GoalContextRefV0,
) []orquestagoal.GoalContextRefV0 {
	seen := map[string]bool{}
	out := make([]orquestagoal.GoalContextRefV0, 0, len(values))
	for _, value := range values {
		key := strings.TrimSpace(value.Kind) + "\x00" + strings.TrimSpace(value.Ref)
		if strings.TrimSpace(value.Ref) == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, value)
	}
	if out == nil {
		return []orquestagoal.GoalContextRefV0{}
	}
	return out
}

func externalWorkRunGoalSpecIssuesV0(
	issues []orquestagoal.GoalWorkIssueV0,
) []ExternalWorkRunIssueV0 {
	out := make([]ExternalWorkRunIssueV0, 0, len(issues))
	for _, issue := range issues {
		code := ErrExternalWorkRunGoalSpecInvalidV0
		if strings.TrimSpace(issue.Code) != "" {
			code += ":" + strings.TrimSpace(issue.Code)
		}
		field := "goal_spec"
		if strings.TrimSpace(issue.Field) != "" {
			field += "." + strings.TrimSpace(issue.Field)
		}
		out = append(out, externalWorkRunIssueV0(code, field))
	}
	if out == nil {
		return []ExternalWorkRunIssueV0{}
	}
	return out
}

func externalWorkGoalTokenV0(values ...string) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return compactExternalWorkRunRefV0(strings.Join(parts, "-"))
}

func shortExternalWorkGoalHashV0(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])[:12]
}
