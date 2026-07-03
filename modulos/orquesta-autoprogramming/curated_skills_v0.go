package orquestaautoprogramming

import (
	"strings"
	"unicode"
)

const (
	AutoprogrammingCuratedSkillContextRefPrefixV0 = "skill_ref:"
	AutoprogrammingSkillDistillationReviewV0      = "skill_distillation_review"
)

type AutoprogrammingCuratedSkillV0 struct {
	SkillRef    string   `json:"skill_ref"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

type AutoprogrammingCuratedSkillMatchRequestV0 struct {
	Area               string   `json:"area,omitempty"`
	Title              string   `json:"title,omitempty"`
	Objective          string   `json:"objective,omitempty"`
	FailureSummary     string   `json:"failure_summary,omitempty"`
	WriteSet           []string `json:"write_set,omitempty"`
	RequiredTests      []string `json:"required_tests,omitempty"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	ContextRefs        []string `json:"context_refs,omitempty"`
}

type AutoprogrammingCuratedSkillMatchResultV0 struct {
	SkillRefs   []string                        `json:"skill_refs,omitempty"`
	ContextRefs []string                        `json:"context_refs,omitempty"`
	Issues      []AutoprogrammingRequestIssueV0 `json:"issues,omitempty"`
}

type AutoprogrammingSkillDistillationCandidateV0 struct {
	SourceResultRef string   `json:"source_result_ref,omitempty"`
	SourceRunRef    string   `json:"source_run_ref,omitempty"`
	SkillName       string   `json:"skill_name,omitempty"`
	Summary         string   `json:"summary,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	EvidenceRefs    []string `json:"evidence_refs,omitempty"`
}

type AutoprogrammingSkillDistillationReviewProposalV0 struct {
	Accepted            bool                            `json:"accepted"`
	ReviewRequired      bool                            `json:"review_required"`
	AutoCommitSkills    bool                            `json:"auto_commit_skills"`
	BacklogSectionTitle string                          `json:"backlog_section_title,omitempty"`
	SuggestedWriteSet   []string                        `json:"suggested_write_set,omitempty"`
	AcceptanceCriteria  []string                        `json:"acceptance_criteria,omitempty"`
	ContextRefs         []string                        `json:"context_refs,omitempty"`
	EvidenceRefs        []string                        `json:"evidence_refs,omitempty"`
	Issues              []AutoprogrammingRequestIssueV0 `json:"issues,omitempty"`
}

func AutoprogrammingCuratedSkillRefV0(name string) string {
	name = normalizeAutoprogrammingSkillSlugV0(name)
	if name == "" {
		return ""
	}
	return "skill-ref-" + name + "-v0"
}

func MatchAutoprogrammingCuratedSkillRefsV0(
	catalog []AutoprogrammingCuratedSkillV0,
	request AutoprogrammingCuratedSkillMatchRequestV0,
) AutoprogrammingCuratedSkillMatchResultV0 {
	result := AutoprogrammingCuratedSkillMatchResultV0{}
	if len(catalog) == 0 {
		return result
	}
	requestTokens := autoprogrammingSkillMatchTokensV0(request)
	for i, skill := range catalog {
		skill = normalizeAutoprogrammingCuratedSkillV0(skill)
		issues := autoprogrammingCuratedSkillIssuesV0(skill, i)
		if len(issues) > 0 {
			result.Issues = append(result.Issues, issues...)
			continue
		}
		if !autoprogrammingCuratedSkillMatchesV0(skill, requestTokens) {
			continue
		}
		result.SkillRefs = append(result.SkillRefs, skill.SkillRef)
		result.ContextRefs = append(result.ContextRefs, AutoprogrammingCuratedSkillContextRefPrefixV0+skill.SkillRef)
	}
	result.SkillRefs = compactStringsV0(result.SkillRefs)
	result.ContextRefs = compactStringsV0(result.ContextRefs)
	return result
}

func ValidateAutoprogrammingCuratedSkillCatalogV0(
	catalog []AutoprogrammingCuratedSkillV0,
) []AutoprogrammingRequestIssueV0 {
	var issues []AutoprogrammingRequestIssueV0
	for i, skill := range catalog {
		issues = append(issues, autoprogrammingCuratedSkillIssuesV0(
			normalizeAutoprogrammingCuratedSkillV0(skill),
			i,
		)...)
	}
	return issues
}

func BuildAutoprogrammingSkillDistillationReviewProposalV0(
	candidate AutoprogrammingSkillDistillationCandidateV0,
) AutoprogrammingSkillDistillationReviewProposalV0 {
	candidate.SkillName = strings.TrimSpace(candidate.SkillName)
	candidate.Summary = strings.TrimSpace(candidate.Summary)
	candidate.SourceResultRef = strings.TrimSpace(candidate.SourceResultRef)
	candidate.SourceRunRef = strings.TrimSpace(candidate.SourceRunRef)
	candidate.Tags = compactStringsV0(candidate.Tags)
	candidate.EvidenceRefs = compactStringsV0(candidate.EvidenceRefs)

	proposal := AutoprogrammingSkillDistillationReviewProposalV0{
		ReviewRequired:   true,
		AutoCommitSkills: false,
		SuggestedWriteSet: []string{
			"skills/docs",
		},
		AcceptanceCriteria: []string{
			"destilacion revisada antes de crear o modificar una skill",
			"sin secretos ni rutas absolutas en la propuesta de habilidad",
			"contrato de uso, entrada, salida y validacion declarado",
		},
		EvidenceRefs: candidate.EvidenceRefs,
	}
	var issues []AutoprogrammingRequestIssueV0
	if candidate.SkillName == "" {
		issues = append(issues, autoprogrammingRequestIssueV0(
			"skill_name_missing",
			"skill_name",
			"nombre de habilidad requerido para proponer destilacion",
		))
	}
	if candidate.Summary == "" {
		issues = append(issues, autoprogrammingRequestIssueV0(
			"summary_missing",
			"summary",
			"resumen requerido para proponer destilacion",
		))
	}
	if autoprogrammingContainsUnsafeSkillDetailV0(strings.Join([]string{
		candidate.SkillName,
		candidate.Summary,
		strings.Join(candidate.Tags, " "),
	}, " ")) {
		issues = append(issues, autoprogrammingRequestIssueV0(
			"skill_distillation_sensitive_detail",
			"summary",
			"la propuesta de habilidad contiene secreto o ruta absoluta",
		))
	}
	if len(issues) > 0 {
		proposal.Issues = issues
		return proposal
	}
	proposal.Accepted = true
	proposal.BacklogSectionTitle = "Propuesta habilidad: " + candidate.SkillName
	proposal.ContextRefs = compactStringsV0([]string{
		"review_gate:" + AutoprogrammingSkillDistillationReviewV0,
		"skill_ref:" + AutoprogrammingCuratedSkillRefV0(candidate.SkillName),
		"source_result_ref:" + candidate.SourceResultRef,
		"source_run_ref:" + candidate.SourceRunRef,
	})
	return proposal
}

func normalizeAutoprogrammingCuratedSkillV0(
	skill AutoprogrammingCuratedSkillV0,
) AutoprogrammingCuratedSkillV0 {
	skill.Name = strings.TrimSpace(skill.Name)
	skill.Description = strings.TrimSpace(skill.Description)
	skill.Tags = compactStringsV0(skill.Tags)
	skill.SkillRef = strings.TrimSpace(skill.SkillRef)
	if skill.SkillRef == "" {
		skill.SkillRef = AutoprogrammingCuratedSkillRefV0(skill.Name)
	}
	return skill
}

func autoprogrammingCuratedSkillIssuesV0(
	skill AutoprogrammingCuratedSkillV0,
	index int,
) []AutoprogrammingRequestIssueV0 {
	field := "curated_skills"
	if index >= 0 {
		field = field + "[]"
	}
	var issues []AutoprogrammingRequestIssueV0
	if skill.SkillRef == "" || !strings.HasPrefix(skill.SkillRef, "skill-ref-") {
		issues = append(issues, autoprogrammingRequestIssueV0(
			"curated_skill_ref_invalid",
			field+".skill_ref",
			"skill_ref opaca requerida",
		))
	}
	if skill.Name == "" {
		issues = append(issues, autoprogrammingRequestIssueV0(
			"curated_skill_name_missing",
			field+".name",
			"name requerido",
		))
	}
	if autoprogrammingContainsUnsafeSkillDetailV0(strings.Join([]string{
		skill.SkillRef,
		skill.Name,
		skill.Description,
		strings.Join(skill.Tags, " "),
	}, " ")) {
		issues = append(issues, autoprogrammingRequestIssueV0(
			"curated_skill_sensitive_detail",
			field,
			"habilidad curada contiene secreto o ruta absoluta",
		))
	}
	return issues
}

func autoprogrammingCuratedSkillMatchesV0(
	skill AutoprogrammingCuratedSkillV0,
	requestTokens map[string]bool,
) bool {
	if len(requestTokens) == 0 {
		return false
	}
	for token := range autoprogrammingSkillTokensFromStringsV0(
		skill.Name,
		skill.Description,
		strings.Join(skill.Tags, " "),
	) {
		if requestTokens[token] {
			return true
		}
	}
	return false
}

func autoprogrammingSkillMatchTokensV0(
	request AutoprogrammingCuratedSkillMatchRequestV0,
) map[string]bool {
	tokens := autoprogrammingSkillTokensFromStringsV0(
		request.Area,
		request.Title,
		request.Objective,
		request.FailureSummary,
		strings.Join(request.WriteSet, " "),
		strings.Join(request.RequiredTests, " "),
		strings.Join(request.AcceptanceCriteria, " "),
		strings.Join(request.ContextRefs, " "),
	)
	for _, test := range request.RequiredTests {
		if strings.Contains(strings.ToLower(test), "go test") {
			tokens["test"] = true
			tokens["tests"] = true
		}
	}
	return tokens
}

func autoprogrammingSkillTokensFromStringsV0(values ...string) map[string]bool {
	tokens := map[string]bool{}
	for _, value := range values {
		for _, raw := range strings.FieldsFunc(normalizeAutoprogrammingSkillTextV0(value), func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r)
		}) {
			token := normalizeAutoprogrammingSkillTokenV0(raw)
			if token == "" || autoprogrammingSkillStopTokenV0(token) {
				continue
			}
			tokens[token] = true
		}
	}
	return tokens
}

func normalizeAutoprogrammingSkillSlugV0(value string) string {
	var ordered []string
	for _, raw := range strings.FieldsFunc(normalizeAutoprogrammingSkillTextV0(value), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		token := strings.TrimSpace(strings.ToLower(raw))
		if len(token) < 2 {
			continue
		}
		ordered = append(ordered, token)
	}
	return strings.Join(ordered, "-")
}

func normalizeAutoprogrammingSkillTextV0(value string) string {
	replacer := strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n",
		"Á", "a", "É", "e", "Í", "i", "Ó", "o", "Ú", "u", "Ü", "u", "Ñ", "n",
	)
	return strings.ToLower(replacer.Replace(value))
}

func normalizeAutoprogrammingSkillTokenV0(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if len(value) > 4 && strings.HasSuffix(value, "s") {
		value = strings.TrimSuffix(value, "s")
	}
	return value
}

func autoprogrammingSkillStopTokenV0(token string) bool {
	if len(token) < 3 {
		return true
	}
	switch token {
	case "orquesta", "programacion", "programar", "codigo", "skill", "ref", "para",
		"con", "por", "del", "los", "las", "una", "uno", "que", "como",
		"cuando", "usar", "usa", "esta", "este", "desde", "hacia", "sobre":
		return true
	default:
		return false
	}
}

func autoprogrammingContainsUnsafeSkillDetailV0(value string) bool {
	lower := strings.ToLower(value)
	for _, marker := range []string{
		"/home/", "/users/", "/srv/", "/tmp/", "/var/", "/etc/", "/root/", "/mnt/", "/opt/",
		"api_key=", "apikey=", "token=", "password=", "passwd=", "secret=", "bearer ",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	for _, field := range strings.Fields(value) {
		if strings.HasPrefix(field, "sk-") && len(field) > 12 {
			return true
		}
	}
	return false
}
