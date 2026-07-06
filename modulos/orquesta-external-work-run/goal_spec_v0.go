package orquestaexternalworkrun

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

const (
	ExternalWorkGoalWorkProfileKindV0 = "domain_work"

	externalWorkGoalSpecEvidenceRefV0 = "evidence-ref-external-work-goal-spec-v0"

	externalWorkGoalInputFieldMaxInlineFieldsV0 = 16
	externalWorkGoalInputFieldMaxPurposeBytesV0 = 700
	externalWorkGoalInputFieldMaxTotalBytesV0   = 5000
	externalWorkGoalInputFieldMaxValuesV0       = 8
	externalWorkGoalInputFieldMaxRawBytesV0     = 900
	externalWorkGoalInputFieldMaxPayloadRefsV0  = 4
)

type externalWorkGoalInputFieldBudgetV0 struct {
	Fields int
	Bytes  int
}

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
			ArtifactType: externalWorkGoalArtifactTypeV0(domainRequest),
			Required:     true,
			EvidenceRefs: []string{externalWorkGoalSpecEvidenceRefV0},
		}},
		EvidenceRefs: externalWorkGoalEvidenceRefsV0(request, domainRequest, token),
		ClosurePolicy: orquestagoal.GoalClosurePolicyV0{
			RequireRequiredTests:                 len(requiredTests) > 0,
			RequireArtifacts:                     true,
			RequireArtifactPaths:                 true,
			RequireMaterializedArtifacts:         true,
			RequireChecklist:                     true,
			RequireReworkPlanForPartialArtifacts: true,
			RequireDomainReceipt:                 true,
			RequiredEvidenceRefs:                 []string{externalWorkGoalSpecEvidenceRefV0},
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
		(2*len(work.InputFields)))
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
	refs = append(refs, externalWorkGoalInputFieldContextRefsV0(request, work.InputFields)...)
	return externalWorkGoalCompactContextRefsV0(refs)
}

func externalWorkGoalInputFieldContextRefsV0(
	request StartExternalWorkRunRequestV0,
	fields []orquestadomainwork.DomainWorkFieldV0,
) []orquestagoal.GoalContextRefV0 {
	if len(fields) == 0 {
		return []orquestagoal.GoalContextRefV0{}
	}
	ordered := externalWorkGoalPrioritizedInputFieldsV0(fields)
	refs := make([]orquestagoal.GoalContextRefV0, 0, len(ordered)+1)
	var inputBudget externalWorkGoalInputFieldBudgetV0
	inlineCount := 0
	payloadCount := 0
	omittedCount := 0
	appendRef := func(kind, ref, purpose string) {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			return
		}
		refs = append(refs, orquestagoal.GoalContextRefV0{
			Kind:    kind,
			Ref:     ref,
			Purpose: purpose,
		})
	}
	for _, field := range ordered {
		name := strings.TrimSpace(field.Name)
		if name == "" {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(name), "opes_subrole_task_refs") {
			for _, taskRef := range externalWorkGoalSanitizeInputValuesV0(name, field.Values) {
				appendRef("opes_subrole_task_ref", taskRef, "Task ref causal esperada para contrato OPES 1+6.")
			}
		}
		fieldRef := "input-field-" + externalWorkGoalSafeInputFieldRefPartV0(name)
		payloadRef, payloadOK := externalWorkGoalInputFieldPayloadRefV0(request, name)
		if purpose, ok := externalWorkGoalInputFieldPurposeV0(field, payloadRef, &inputBudget); ok {
			valueRef := fieldRef + "-value-" + shortExternalWorkGoalHashV0(purpose)
			appendRef("input_field_value", valueRef, purpose)
			inlineCount++
			continue
		}
		if payloadOK &&
			payloadCount < externalWorkGoalInputFieldMaxPayloadRefsV0 &&
			externalWorkGoalInputFieldShouldExposePayloadRefV0(name) {
			appendRef("input_field_payload", payloadRef, externalWorkGoalInputFieldPayloadCompactPurposeV0(name))
			payloadCount++
			continue
		}
		omittedCount++
	}
	appendRef(
		"input_fields_summary",
		"input-fields-summary-"+shortExternalWorkGoalHashV0(externalWorkGoalInputFieldSummarySeedV0(fields)),
		externalWorkGoalInputFieldsSummaryPurposeV0(len(fields), inlineCount, payloadCount, omittedCount),
	)
	return externalWorkGoalCompactContextRefsV0(refs)
}

func externalWorkGoalPrioritizedInputFieldsV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) []orquestadomainwork.DomainWorkFieldV0 {
	out := append([]orquestadomainwork.DomainWorkFieldV0(nil), fields...)
	sort.SliceStable(out, func(i, j int) bool {
		return externalWorkGoalInputFieldPriorityV0(out[i].Name) <
			externalWorkGoalInputFieldPriorityV0(out[j].Name)
	})
	return out
}

func externalWorkGoalInputFieldPriorityV0(name string) int {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "course_id", "program_id", "topic_id", "work_kind", "transport_job_type",
		"expected_artifact_type", "official_order", "probe_ref",
		"opes_subroles_materialization_status", "opes_subroles_blocking_reason",
		"opes_subrole_task_refs", "opes_subrole_roles", "opes_subrole_write_sets",
		"product_write_set_status", "product_write_set_rework_action",
		"current_phase", "opes_current_phase", "audio_current_phase",
		"retry_from_phase", "recommended_retry_phase", "audio_counters", "domain_counters",
		"provider_status", "provider_reason", "provider_timeout", "running_no_recent_progress",
		"superseded_by_local_evidence", "superseded_by_local_evidence_ref",
		"local_evidence_ref", "local_artifact_ref":
		return 10
	case "document_plan_artifact_id", "assembled_topic_artifact_id", "audio_profile_ref":
		return 20
	case "required_read_refs", "required_outputs", "output_contract",
		"course_root_abs", "topic_dir_abs", "topic_manifest_abs", "program_json_abs":
		return 30
	default:
		return 100
	}
}

func externalWorkGoalInputFieldShouldExposePayloadRefV0(name string) bool {
	if strings.TrimSpace(name) == "" || externalWorkGoalFieldNameSensitiveV0(name) {
		return false
	}
	return externalWorkGoalInputFieldPriorityV0(name) <= 30 ||
		externalWorkGoalFieldAllowsOperationalPathV0(name)
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
	if len(domainRequest.InputFields) > 0 {
		criteria = append(criteria, "Usar los input_fields inlineados en context_refs[input_field_value] como contrato operativo compacto; los local_path_ref estructurados son refs opacas no ejecutables y no deben tratarse como paths de filesystem. Los campos omitidos por redaccion o presupuesto quedan durables en AppChange/DomainWork y solo deben resolverse si son imprescindibles para el artefacto. Bloquear con rework de dominio solo si falta un input imprescindible, no por un campo accesorio omitido.")
	}
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

func externalWorkGoalInputFieldPurposeV0(
	field orquestadomainwork.DomainWorkFieldV0,
	payloadRef string,
	budget *externalWorkGoalInputFieldBudgetV0,
) (string, bool) {
	name := strings.TrimSpace(field.Name)
	if name == "" || budget == nil {
		return "", false
	}
	if budget.Fields >= externalWorkGoalInputFieldMaxInlineFieldsV0 {
		return "", false
	}
	summary, ok := externalWorkGoalInputFieldSummaryV0(field, payloadRef)
	if !ok {
		return "", false
	}
	raw, err := json.Marshal(summary)
	if err != nil {
		return "", false
	}
	purpose := "Campo input_fields." + name + " inlineado de forma acotada: " + string(raw)
	purpose, _ = boundedExternalWorkGoalTextV0(purpose, externalWorkGoalInputFieldMaxPurposeBytesV0)
	if budget.Bytes+len(purpose) > externalWorkGoalInputFieldMaxTotalBytesV0 {
		return "", false
	}
	budget.Fields++
	budget.Bytes += len(purpose)
	return purpose, true
}

func externalWorkGoalInputFieldSummaryV0(
	field orquestadomainwork.DomainWorkFieldV0,
	payloadRef string,
) (map[string]any, bool) {
	name := strings.TrimSpace(field.Name)
	if externalWorkGoalFieldNameSensitiveV0(name) {
		return nil, false
	}
	if externalWorkGoalInputFieldTooLargeForInlineV0(field) {
		return nil, false
	}
	summary := map[string]any{"name": name}
	if value, ok := externalWorkGoalInputFieldSummaryScalarV0(name, field.Value); ok {
		summary["value"] = value
	}
	values := externalWorkGoalInputFieldSummaryValuesV0(name, field.Values)
	if len(values) > 0 {
		summary["values"] = values
	}
	if valueJSON := externalWorkGoalSanitizeInputJSONV0(name, field.ValueJSON); valueJSON != "" {
		summary["value_json"] = externalWorkGoalInputJSONSummaryValueV0(valueJSON)
	}
	if len(summary) == 1 {
		return nil, false
	}
	if payloadRef != "" {
		summary["payload_ref"] = payloadRef
	}
	return summary, true
}

func externalWorkGoalInputFieldTooLargeForInlineV0(field orquestadomainwork.DomainWorkFieldV0) bool {
	total := len(strings.TrimSpace(field.Value)) + len(strings.TrimSpace(string(field.ValueJSON)))
	for _, value := range field.Values {
		total += len(strings.TrimSpace(value))
	}
	return len(field.Values) > externalWorkGoalInputFieldMaxValuesV0 ||
		total > externalWorkGoalInputFieldMaxRawBytesV0
}

func externalWorkGoalInputFieldSummaryScalarV0(name string, value string) (any, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, false
	}
	if externalWorkGoalFieldAllowsOperationalPathV0(name) &&
		externalWorkGoalValueLooksAbsoluteLocalPathV0(value) {
		return externalWorkGoalLocalPathRefSummaryObjectV0(value), true
	}
	sanitized := externalWorkGoalSanitizeInputValueV0(name, value)
	if sanitized == "" {
		return nil, false
	}
	return sanitized, true
}

func externalWorkGoalInputFieldSummaryValuesV0(name string, values []string) []any {
	out := make([]any, 0, len(values))
	for _, value := range values {
		if sanitized, ok := externalWorkGoalInputFieldSummaryScalarV0(name, value); ok {
			out = append(out, sanitized)
		}
	}
	if out == nil {
		return []any{}
	}
	return out
}

func externalWorkGoalInputJSONSummaryValueV0(value string) any {
	var decoded any
	if err := json.Unmarshal([]byte(value), &decoded); err == nil {
		return decoded
	}
	return value
}

func externalWorkGoalInputFieldsSummaryPurposeV0(total, inlineCount, payloadCount, omittedCount int) string {
	return fmt.Sprintf(
		"Contrato DomainWork trae %d input_fields; inlineados=%d, payload_refs=%d, omitidos=%d por presupuesto/sensibilidad. Los payloads completos siguen en AppChange/DomainWork; no bloquear por campos accesorios omitidos.",
		total,
		inlineCount,
		payloadCount,
		omittedCount,
	)
}

func externalWorkGoalInputFieldPayloadCompactPurposeV0(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || externalWorkGoalFieldNameSensitiveV0(name) {
		return "Payload de input_field sensible conservado como ref opaca en AppChange/DomainWork."
	}
	return "Payload de input_fields." + name + " conservado como ref opaca en AppChange/DomainWork; resolver solo si es imprescindible para el artefacto."
}

func externalWorkGoalInputFieldSummarySeedV0(fields []orquestadomainwork.DomainWorkFieldV0) string {
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		name := strings.TrimSpace(field.Name)
		if name == "" || externalWorkGoalFieldNameSensitiveV0(name) {
			continue
		}
		parts = append(parts, name)
	}
	return strings.Join(parts, ",")
}

func externalWorkGoalInputFieldPayloadRefV0(
	request StartExternalWorkRunRequestV0,
	name string,
) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" || externalWorkGoalFieldNameSensitiveV0(name) {
		return "", false
	}
	runRef := compactExternalWorkRunRefV0(request.RunRef)
	changeRef := compactExternalWorkRunRefV0(request.AppChangeRequest.ChangeRef)
	fieldRef := externalWorkGoalSafeInputFieldRefPartV0(name)
	return "app_change_payload:" + runRef + ":" + changeRef + ":external_work.input_fields." + fieldRef, true
}

func externalWorkGoalSafeInputFieldRefPartV0(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || externalWorkGoalFieldNameSensitiveV0(name) {
		return "redacted-" + shortExternalWorkGoalHashV0(name)
	}
	return compactExternalWorkRunRefV0(name)
}

func externalWorkGoalSanitizeInputValuesV0(name string, values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if sanitized := externalWorkGoalSanitizeInputValueV0(name, value); sanitized != "" {
			out = append(out, sanitized)
		}
	}
	if out == nil {
		return []string{}
	}
	return out
}

func externalWorkGoalSanitizeInputJSONV0(name string, value json.RawMessage) string {
	text := strings.TrimSpace(string(value))
	if text == "" {
		return ""
	}
	return externalWorkGoalSanitizeInputValueV0(name, text)
}

func externalWorkGoalSanitizeInputValueV0(name string, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if externalWorkGoalFieldNameSensitiveV0(name) ||
		(!externalWorkGoalFieldAllowsOpaqueRefsV0(name) && externalWorkGoalValueLooksSensitiveV0(value)) {
		return "redacted-sensitive-value"
	}
	if externalWorkGoalFieldAllowsOperationalPathV0(name) {
		value = externalWorkGoalSanitizeOperationalValueV0(value)
	} else {
		value = redactExternalWorkGoalLocalPathV0(value)
	}
	value, _ = boundedExternalWorkGoalTextV0(value, externalWorkGoalInputFieldMaxPurposeBytesV0/2)
	return value
}

func externalWorkGoalSanitizeOperationalValueV0(value string) string {
	value = strings.TrimSpace(value)
	if externalWorkGoalValueLooksAbsoluteLocalPathV0(value) {
		return externalWorkGoalLocalPathRefSummaryV0(value)
	}
	return redactExternalWorkGoalLocalPathV0(value)
}

func externalWorkGoalValueLooksAbsoluteLocalPathV0(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	if strings.HasPrefix(value, "/") ||
		strings.HasPrefix(value, "~/") ||
		strings.HasPrefix(value, "$HOME/") ||
		strings.HasPrefix(strings.ToLower(value), "%userprofile%\\") {
		return true
	}
	if len(value) >= 3 && value[1] == ':' && (value[2] == '\\' || value[2] == '/') {
		return true
	}
	return false
}

func externalWorkGoalLocalPathRefSummaryV0(value string) string {
	summary := externalWorkGoalLocalPathRefSummaryObjectV0(value)
	parts := []string{"local_path_ref", "ref=" + summary["ref"].(string), "executable_path=false"}
	if base, ok := summary["basename_hint"].(string); ok && base != "" {
		parts = append(parts, "basename_hint="+base)
	}
	return strings.Join(parts, " ")
}

func externalWorkGoalLocalPathRefSummaryObjectV0(value string) map[string]any {
	summary := map[string]any{
		"kind":            "local_path_ref",
		"ref":             "local-path-ref-" + shortExternalWorkGoalHashV0(value),
		"executable_path": false,
		"resolution":      "opaque_input_field_payload_ref_only",
	}
	if base := externalWorkGoalPathBaseV0(value); base != "" {
		summary["basename_hint"] = base
	}
	return summary
}

func externalWorkGoalPathBaseV0(value string) string {
	value = strings.TrimRight(strings.TrimSpace(value), `/\`)
	if value == "" {
		return ""
	}
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '/' || r == '\\'
	})
	if len(parts) == 0 {
		return ""
	}
	return compactExternalWorkRunRefV0(parts[len(parts)-1])
}

func externalWorkGoalFieldAllowsOpaqueRefsV0(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "opes_subrole_task_refs", "opes_subrole_roles", "opes_subrole_write_sets":
		return true
	default:
		return false
	}
}

func externalWorkGoalFieldAllowsOperationalPathV0(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	if strings.HasSuffix(name, "_abs") ||
		strings.HasSuffix(name, "_path") ||
		strings.HasSuffix(name, "_paths") ||
		strings.HasSuffix(name, "_dir") ||
		strings.HasSuffix(name, "_dirs") ||
		strings.HasSuffix(name, "_root") {
		return true
	}
	switch name {
	case "course_root_abs",
		"topic_dir_abs",
		"topic_manifest_abs",
		"program_json_abs",
		"required_read_refs",
		"required_outputs",
		"output_contract":
		return true
	default:
		return false
	}
}

func externalWorkGoalFieldNameSensitiveV0(name string) bool {
	name = strings.ReplaceAll(strings.ToLower(strings.TrimSpace(name)), "_", "-")
	for _, marker := range []string{
		"access-token",
		"refresh-token",
		"api-key",
		"secret",
		"secreto",
		"password",
		"credential",
		"credencial",
		"private",
		"token",
		"prompt",
		"transcript",
		"completion",
	} {
		if strings.Contains(name, marker) {
			return true
		}
	}
	return false
}

func externalWorkGoalValueLooksSensitiveV0(value string) bool {
	lower := strings.ToLower(value)
	for _, marker := range []string{
		"access_token=",
		"refresh_token=",
		"api_key=",
		"api-key:",
		"authorization:",
		"bearer ",
		"client_secret=",
		"password=",
		"secret=",
		"-----begin ",
		"sk-",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func redactExternalWorkGoalLocalPathV0(value string) string {
	replacer := strings.NewReplacer(
		"/home/", "/home-redacted/",
		"/Users/", "/users-redacted/",
		"/users/", "/users-redacted/",
		"$HOME", "HOME_REF",
		"~/", "HOME_REF/",
	)
	return replacer.Replace(value)
}

func boundedExternalWorkGoalTextV0(value string, maxBytes int) (string, bool) {
	value = strings.TrimSpace(value)
	if maxBytes <= 0 {
		return "", true
	}
	if len(value) <= maxBytes {
		return value, false
	}
	for maxBytes > 0 && !utf8.ValidString(value[:maxBytes]) {
		maxBytes--
	}
	return value[:maxBytes] + "...[truncated]", true
}

func externalWorkGoalArtifactTypeV0(
	domainRequest orquestadomainwork.DomainWorkJobRequestV0,
) string {
	if expected := externalWorkGoalExpectedArtifactTypeFieldV0(domainRequest.InputFields); expected != "" {
		return expected
	}
	return orquestadomainwork.ExpectedDomainWorkArtifactTypeForWorkKindV0(domainRequest.WorkKind)
}

func externalWorkGoalExpectedArtifactTypeFieldV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) string {
	for _, field := range fields {
		if strings.TrimSpace(field.Name) != "expected_artifact_type" {
			continue
		}
		for _, value := range append([]string{field.Value}, field.Values...) {
			if artifactType := externalWorkGoalExpectedArtifactTypeValueV0(value); artifactType != "" {
				return artifactType
			}
		}
		for _, value := range externalWorkRunJSONFieldValuesV0(field.ValueJSON) {
			if artifactType := externalWorkGoalExpectedArtifactTypeValueV0(value); artifactType != "" {
				return artifactType
			}
		}
	}
	return ""
}

func externalWorkGoalExpectedArtifactTypeValueV0(value string) string {
	artifactType := compactExternalWorkRunRefV0(value)
	if artifactType == "" || artifactType == "external-work" {
		return ""
	}
	switch artifactType {
	case "completed_syllabus_package", "finalize_temario_package", "finalize_syllabus_package":
		return orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0
	case orquestadomainwork.DomainWorkArtifactTypeContentBlockV0,
		orquestadomainwork.DomainWorkArtifactTypeVisualAssetV0,
		orquestadomainwork.DomainWorkArtifactTypeBlockRevisionV0,
		orquestadomainwork.DomainWorkArtifactTypeSourceV0,
		orquestadomainwork.DomainWorkArtifactTypeTopicStructureV0,
		orquestadomainwork.DomainWorkArtifactTypeTopicOutlineV0,
		orquestadomainwork.DomainWorkArtifactTypeTopicSummaryV0,
		orquestadomainwork.DomainWorkArtifactTypeTopicExpansionPackageV0,
		orquestadomainwork.DomainWorkArtifactTypeAssembledTopicV0,
		orquestadomainwork.DomainWorkArtifactTypeAudioAssetV0,
		orquestadomainwork.DomainWorkArtifactTypeExamResearchReportV0,
		orquestadomainwork.DomainWorkArtifactTypeQuestionBankV0,
		orquestadomainwork.DomainWorkArtifactTypePracticalCasesV0,
		orquestadomainwork.DomainWorkArtifactTypeLocalHTMLSiteV0,
		orquestadomainwork.DomainWorkArtifactTypeTutorBotPackageV0,
		orquestadomainwork.DomainWorkArtifactTypeInteractivePracticeV0,
		orquestadomainwork.DomainWorkArtifactTypeHelpPackageV0,
		orquestadomainwork.DomainWorkArtifactTypeAgentReviewReportV0,
		orquestadomainwork.DomainWorkArtifactTypeAgentPairReviewReportV0,
		orquestadomainwork.DomainWorkArtifactTypeAgentCandidateV0,
		orquestadomainwork.DomainWorkArtifactTypeAgentCandidateVoteV0,
		orquestadomainwork.DomainWorkArtifactTypeAgentCandidateSelectV0,
		orquestadomainwork.DomainWorkArtifactTypeDirectorReviewMatrixV0,
		orquestadomainwork.DomainWorkArtifactTypeTopicRegistryUpdateV0,
		orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0,
		orquestadomainwork.DomainWorkArtifactTypeGenericWorkDeliveryV0:
		return artifactType
	default:
		return artifactType
	}
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
