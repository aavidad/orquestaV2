package orquestaopesbridge

import (
	"encoding/json"
	"sort"
	"strings"

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

type JobContextV0 struct {
	TopicBlocks []orquestaopesconnector.TopicBlockV0
}

func BuildExternalWorkRunRequestV0(
	job orquestaopesconnector.ExternalJobV0,
	config JobRunConfigV0,
) (orquestaexternalworkrun.StartExternalWorkRunRequestV0, bool) {
	return BuildExternalWorkRunRequestWithContextV0(job, config, JobContextV0{})
}

func BuildExternalWorkRunRequestWithContextV0(
	job orquestaopesconnector.ExternalJobV0,
	config JobRunConfigV0,
	jobContext JobContextV0,
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
	fields = appendTopicBlocksFieldV0(fields, jobContext.TopicBlocks)
	fields = withOPESGlobalEditorialPolicyFieldV0(fields)
	fields = withOPESHTMLPublicationPolicyFieldV0(fields)
	fields = withOPESHTMLTopicTemplateFieldV0(fields)
	fields = withOPESTemarioAgentRulesFieldV0(fields)
	fields = appendFieldIfMissingV0(fields, "job_id", job.ID)
	fields = appendFieldIfMissingV0(fields, "job_type", job.Type)
	var opaqueOK bool
	fields, opaqueOK = appendOpaqueExecutionRefFieldsV0(fields, job.ExternalRefs)
	if !opaqueOK {
		return orquestaexternalworkrun.StartExternalWorkRunRequestV0{}, false
	}
	fields = appendFieldIfMissingV0(fields, "expected_artifact_type", expectedArtifactTypeV0(job.Type))
	fields = appendFieldIfMissingV0(fields, "context_budget_profile", contextProfileForJobTypeV0(job.Type))
	fields = appendExpansionDocumentContractFieldsV0(fields, job.Type)
	fields = appendDocumentPlanContractFieldsV0(fields, job.Type)
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
		AllowedWriteSet:    []string{opesJobWriteSetV0(workKind, safeJob)},
		ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
			ProjectRef:    config.ProjectRef,
			JobRef:        job.ID,
			InterfaceRefs: interfaceRefsForJobFieldsV0(fields),
			WorkKind:      job.Type,
			WorkRefs:      workRefs,
			InputFields:   fields,
			RequiredTests: opesRequiredTestsForJobV0(job.Type, job.ID, workRefs),
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

func interfaceRefsForJobFieldsV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) []string {
	refs := []string{"opes-rest-v0", "opes-mcp-v0"}
	if fieldDeclaresSixSubrolesV0(fields) {
		refs = append(refs, "opes.padre-tema-6-subroles.v1")
	}
	return compactStringsV0(refs)
}

func fieldDeclaresSixSubrolesV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) bool {
	for _, field := range fields {
		if strings.TrimSpace(field.Name) != "subroles_required" {
			continue
		}
		if len(field.Values) >= 6 {
			return true
		}
		if strings.TrimSpace(string(field.ValueJSON)) == "6" ||
			strings.EqualFold(strings.TrimSpace(string(field.ValueJSON)), "true") {
			return true
		}
		if strings.TrimSpace(field.Value) == "6" ||
			strings.EqualFold(strings.TrimSpace(field.Value), "true") {
			return true
		}
	}
	return false
}

type topicBlockContextV0 struct {
	ID         string          `json:"id,omitempty"`
	StableID   string          `json:"stable_id,omitempty"`
	ChapterID  string          `json:"chapter_id,omitempty"`
	Type       string          `json:"type,omitempty"`
	Status     string          `json:"status,omitempty"`
	Title      string          `json:"title,omitempty"`
	Markdown   string          `json:"markdown,omitempty"`
	SourceRefs []string        `json:"source_refs,omitempty"`
	Citations  json.RawMessage `json:"citations,omitempty"`
}

func appendTopicBlocksFieldV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	blocks []orquestaopesconnector.TopicBlockV0,
) []orquestadomainwork.DomainWorkFieldV0 {
	if len(blocks) == 0 || fieldHasNameV0(fields, "topic_blocks") {
		return fields
	}
	payload := make([]topicBlockContextV0, 0, len(blocks))
	for _, block := range blocks {
		payload = append(payload, topicBlockContextV0{
			ID:         strings.TrimSpace(block.ID),
			StableID:   strings.TrimSpace(block.StableID),
			ChapterID:  strings.TrimSpace(block.ChapterID),
			Type:       strings.TrimSpace(block.Type),
			Status:     strings.TrimSpace(block.Status),
			Title:      strings.TrimSpace(block.Title),
			Markdown:   strings.TrimSpace(block.Markdown),
			SourceRefs: compactStringsV0(block.SourceRefs),
			Citations:  append([]byte(nil), block.Citations...),
		})
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fields
	}
	return append(fields, orquestadomainwork.DomainWorkFieldV0{
		Name:      "topic_blocks",
		ValueJSON: raw,
	})
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
	for _, name := range opaqueExecutionRefFieldNamesV0() {
		if value := fieldValueV0(fields, name); value != "" {
			refs = append(refs, value)
		}
	}
	return compactStringsV0(refs)
}

func currentStateRefsForJobV0(safeJob string, workRefs []string) []string {
	return compactStringsV0(append([]string{"opes-job-" + safeJob}, workRefs...))
}
