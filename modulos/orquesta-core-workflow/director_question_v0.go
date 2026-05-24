package orquestacoreworkflow

import "strings"

const (
	DirectorQuestionTargetDirectorV0 = "director"

	ErrDirectorQuestionInvalidaV0 = "director_question_invalida"
)

const (
	maxDirectorQuestionSummaryLenV0 = 700
	maxDirectorQuestionOptionLenV0  = 180
	maxDirectorQuestionRefLenV0     = 180
	maxDirectorQuestionOptionsV0    = 5
	maxDirectorQuestionRefsV0       = 10
)

type DirectorQuestionV0 struct {
	QuestionID   string   `json:"question_id"`
	RunID        string   `json:"run_id"`
	SourceGroup  string   `json:"source_group"`
	TargetGroup  string   `json:"target_group,omitempty"`
	Summary      string   `json:"summary"`
	Options      []string `json:"options,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
	Blocking     bool     `json:"blocking"`
	RequestedAt  string   `json:"requested_at"`
}

type DirectorQuestionErrorV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

func (err DirectorQuestionErrorV0) Error() string {
	return codeFieldErrorTextV0(err.Code, err.Field)
}

func NewDirectorQuestionV0(question DirectorQuestionV0) (DirectorQuestionV0, error) {
	normalized := normalizeDirectorQuestionV0(question)
	if err := ValidateDirectorQuestionV0(normalized); err != nil {
		return DirectorQuestionV0{}, err
	}
	return normalized, nil
}

func ValidateDirectorQuestionV0(question DirectorQuestionV0) error {
	if strings.TrimSpace(question.QuestionID) == "" {
		return directorQuestionErrorV0("question_id")
	}
	if strings.TrimSpace(question.RunID) == "" {
		return directorQuestionErrorV0("run_id")
	}
	if strings.TrimSpace(question.SourceGroup) == "" {
		return directorQuestionErrorV0("source_group")
	}
	if strings.TrimSpace(question.Summary) == "" {
		return directorQuestionErrorV0("summary")
	}
	if strings.TrimSpace(question.RequestedAt) == "" {
		return directorQuestionErrorV0("requested_at")
	}
	if !directorQuestionIsCompactV0(question) {
		return directorQuestionErrorV0("payload")
	}
	return nil
}

func normalizeDirectorQuestionV0(question DirectorQuestionV0) DirectorQuestionV0 {
	question.QuestionID = strings.TrimSpace(question.QuestionID)
	question.RunID = strings.TrimSpace(question.RunID)
	question.SourceGroup = strings.TrimSpace(question.SourceGroup)
	question.TargetGroup = normalizeDirectorTargetGroupV0(question.TargetGroup)
	question.Summary = strings.TrimSpace(question.Summary)
	question.RequestedAt = strings.TrimSpace(question.RequestedAt)
	question.Options = compactStringsV0(question.Options)
	question.EvidenceRefs = compactStringsV0(question.EvidenceRefs)
	return question
}

func normalizeDirectorTargetGroupV0(target string) string {
	compact := strings.TrimSpace(target)
	if compact == "" {
		return DirectorQuestionTargetDirectorV0
	}
	return compact
}

func directorQuestionIsCompactV0(question DirectorQuestionV0) bool {
	if len(question.Summary) > maxDirectorQuestionSummaryLenV0 {
		return false
	}
	return compactListWithinLimitV0(question.Options, maxDirectorQuestionOptionsV0, maxDirectorQuestionOptionLenV0) &&
		compactListWithinLimitV0(question.EvidenceRefs, maxDirectorQuestionRefsV0, maxDirectorQuestionRefLenV0)
}

func compactListWithinLimitV0(values []string, maxItems int, maxLen int) bool {
	if len(values) > maxItems {
		return false
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" || len(value) > maxLen {
			return false
		}
	}
	return true
}

func compactStringsV0(values []string) []string {
	if values == nil {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		compact := strings.TrimSpace(value)
		if compact != "" {
			result = append(result, compact)
		}
	}
	return result
}

func directorQuestionErrorV0(field string) DirectorQuestionErrorV0 {
	return DirectorQuestionErrorV0{Code: ErrDirectorQuestionInvalidaV0, Field: field}
}
