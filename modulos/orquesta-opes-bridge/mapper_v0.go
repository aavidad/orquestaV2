package orquestaopesbridge

import (
	"encoding/json"
	"sort"
	"strings"
	"unicode"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestaexternalworkrun "orquesta/modulos/orquesta-external-work-run"
	orquestaopesconnector "orquesta/modulos/orquesta-opes-connector"
)

const (
	DefaultProjectRefV0 = "opes"
	DefaultLocaleV0     = "es-ES"
)

type JobRunConfigV0 struct {
	ProjectRef    string
	Locale        string
	QueueRef      string
	PriorityScore int
	OccurredAt    string
	RequestedBy   string
}

func BuildExternalWorkRunRequestV0(
	job orquestaopesconnector.ExternalJobV0,
	config JobRunConfigV0,
) (orquestaexternalworkrun.StartExternalWorkRunRequestV0, bool) {
	job.ID = strings.TrimSpace(job.ID)
	job.Type = strings.TrimSpace(job.Type)
	if job.ID == "" || job.Type == "" {
		return orquestaexternalworkrun.StartExternalWorkRunRequestV0{}, false
	}
	config = normalizeJobRunConfigV0(config)
	safeJob := compactOPESBridgeRefV0(job.ID)
	workKind := compactOPESBridgeRefV0(job.Type)
	fields, ok := payloadFieldsV0(job.PayloadJSON)
	if !ok {
		return orquestaexternalworkrun.StartExternalWorkRunRequestV0{}, false
	}
	fields = appendFieldIfMissingV0(fields, "job_id", job.ID)
	fields = appendFieldIfMissingV0(fields, "job_type", job.Type)
	fields = appendFieldIfMissingV0(fields, "expected_artifact_type", expectedArtifactTypeV0(job.Type))
	fields = appendFieldIfMissingV0(fields, "context_budget_profile", contextProfileForJobTypeV0(job.Type))
	workRefs := workRefsForPayloadV0(job, fields)
	change := orquestaappchange.AppChangeRequestV0{
		SchemaVersion:      orquestaappchange.AppChangeRequestSchemaV0,
		RequestID:          "req-opes-external-" + safeJob,
		CorrelationID:      firstNonEmptyV0(job.CorrelationID, "corr-opes-external-"+safeJob),
		AppRef:             config.ProjectRef,
		ChangeRef:          "opes-job-" + safeJob,
		ActorRef:           "opes",
		Locale:             config.Locale,
		UserIntent:         userIntentForJobV0(job.Type),
		TargetArea:         "domain_work",
		CurrentStateRefs:   currentStateRefsForJobV0(safeJob, workRefs),
		AcceptanceCriteria: acceptanceCriteriaForJobV0(job.Type),
		Constraints:        constraintsForJobV0(),
		AllowedWriteSet:    []string{"external/opes/" + workKind},
		ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
			ProjectRef:    config.ProjectRef,
			JobRef:        job.ID,
			InterfaceRefs: []string{"opes-rest-v0", "opes-mcp-v0"},
			WorkKind:      job.Type,
			WorkRefs:      workRefs,
			InputFields:   fields,
		},
	}
	return orquestaexternalworkrun.StartExternalWorkRunRequestV0{
		SchemaVersion:    orquestaexternalworkrun.StartExternalWorkRunRequestSchemaV0,
		RequestID:        change.RequestID,
		CorrelationID:    change.CorrelationID,
		ProjectRef:       config.ProjectRef,
		AppSpecRef:       "app-spec-external-work-" + config.ProjectRef,
		QueueRef:         config.QueueRef,
		PriorityScore:    config.PriorityScore,
		OccurredAt:       config.OccurredAt,
		RequestedBy:      config.RequestedBy,
		AppChangeRequest: change,
	}, true
}

func normalizeJobRunConfigV0(config JobRunConfigV0) JobRunConfigV0 {
	config.ProjectRef = compactOPESBridgeRefV0(firstNonEmptyV0(config.ProjectRef, DefaultProjectRefV0))
	config.Locale = firstNonEmptyV0(config.Locale, DefaultLocaleV0)
	config.QueueRef = strings.TrimSpace(config.QueueRef)
	config.OccurredAt = strings.TrimSpace(config.OccurredAt)
	config.RequestedBy = strings.TrimSpace(config.RequestedBy)
	return config
}

func payloadFieldsV0(payload string) ([]orquestadomainwork.DomainWorkFieldV0, bool) {
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return []orquestadomainwork.DomainWorkFieldV0{}, true
	}
	var values map[string]json.RawMessage
	if err := json.Unmarshal([]byte(payload), &values); err != nil {
		return nil, false
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fields := make([]orquestadomainwork.DomainWorkFieldV0, 0, len(keys))
	for _, key := range keys {
		field, ok := payloadFieldV0(key, values[key])
		if ok {
			fields = append(fields, field)
		}
	}
	return fields, true
}

func payloadFieldV0(
	name string,
	raw json.RawMessage,
) (orquestadomainwork.DomainWorkFieldV0, bool) {
	name = strings.TrimSpace(name)
	if name == "" || len(raw) == 0 || !json.Valid(raw) {
		return orquestadomainwork.DomainWorkFieldV0{}, false
	}
	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		return orquestadomainwork.DomainWorkFieldV0{Name: name, Value: strings.TrimSpace(value)}, true
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err == nil {
		return orquestadomainwork.DomainWorkFieldV0{Name: name, Values: compactStringsV0(values)}, true
	}
	return orquestadomainwork.DomainWorkFieldV0{Name: name, ValueJSON: append([]byte(nil), raw...)}, true
}

func appendFieldIfMissingV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	name string,
	value string,
) []orquestadomainwork.DomainWorkFieldV0 {
	if strings.TrimSpace(value) == "" || fieldHasNameV0(fields, name) {
		return fields
	}
	return append(fields, orquestadomainwork.DomainWorkFieldV0{Name: name, Value: value})
}

func fieldHasNameV0(fields []orquestadomainwork.DomainWorkFieldV0, name string) bool {
	for _, field := range fields {
		if strings.TrimSpace(field.Name) == name {
			return true
		}
	}
	return false
}

func fieldValueV0(fields []orquestadomainwork.DomainWorkFieldV0, name string) string {
	for _, field := range fields {
		if strings.TrimSpace(field.Name) == name {
			return strings.TrimSpace(field.Value)
		}
	}
	return ""
}

func workRefsForPayloadV0(
	job orquestaopesconnector.ExternalJobV0,
	fields []orquestadomainwork.DomainWorkFieldV0,
) []string {
	refs := []string{"opes-job-" + compactOPESBridgeRefV0(job.ID)}
	for _, name := range []string{
		"program_id",
		"topic_id",
		"chapter_id",
		"summary_job_id",
		"summary_artifact_id",
		"source_version_id",
	} {
		value := fieldValueV0(fields, name)
		if value == "" {
			continue
		}
		refs = append(refs, compactOPESBridgeRefV0("opes-"+name+"-"+value))
	}
	return compactStringsV0(refs)
}

func currentStateRefsForJobV0(safeJob string, workRefs []string) []string {
	return compactStringsV0(append([]string{"opes-job-" + safeJob}, workRefs...))
}

func expectedArtifactTypeV0(jobType string) string {
	switch strings.TrimSpace(jobType) {
	case "draft_content_block", "generate_block", "generate_program_topic_draft":
		return "content_block"
	case "generate_visual_asset":
		return "visual_asset"
	case "review_legal", "review_pedagogical", "review_quality", "validate_topic":
		return "block_revision"
	case "research_sources", "download_source", "verify_sources":
		return "source"
	case "split_syllabus_topic":
		return "topic_structure"
	case "draft_topic_outline", "create_exam_outline":
		return "topic_outline"
	case "summarize_block", "summarize_chapter", "summarize_topic":
		return "topic_summary"
	case "expand_topic_from_summary":
		return "topic_expansion_package"
	default:
		return "work_delivery"
	}
}

func contextProfileForJobTypeV0(jobType string) string {
	switch strings.TrimSpace(jobType) {
	case "expand_topic_from_summary",
		"draft_content_block",
		"review_legal",
		"review_pedagogical",
		"review_quality",
		"validate_topic":
		return "large"
	default:
		return "standard"
	}
}

func userIntentForJobV0(jobType string) string {
	return "Resolver job OPES " + strings.TrimSpace(jobType) +
		" y devolver artefacto " + expectedArtifactTypeV0(jobType) +
		" por el contrato publico OPES."
}

func acceptanceCriteriaForJobV0(jobType string) []string {
	return []string{
		"devolver artifact_type=" + expectedArtifactTypeV0(jobType),
		"payload_json valido y trazable",
		"sin placeholders",
		"sin leer internals de OPES",
		"entrega en fichero unico bajo allowed_write_set",
	}
}

func constraintsForJobV0() []string {
	return []string{
		"no inventar contenido",
		"no leer DB ni ficheros internos de OPES",
		"usar solo el paquete de dominio recibido",
		"si falta contexto obligatorio declarar bloqueo",
	}
}

func compactOPESBridgeRefV0(value string) string {
	value = strings.TrimSpace(value)
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r), r == '_', r == '.', r == '-':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "opes"
	}
	return out
}

func compactStringsV0(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	if out == nil {
		return []string{}
	}
	return out
}

func firstNonEmptyV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
