package orquestaopesdirector

import (
	"encoding/json"
	"strconv"
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func topicRegistryQuestionBankQuestionsForFieldsV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) []OPESQuestionBankQuestionV0 {
	var questions []OPESQuestionBankQuestionV0
	for _, field := range fields {
		if !topicRegistryQuestionBankJSONFieldNameV0(field.Name) {
			continue
		}
		if len(field.ValueJSON) > 0 {
			questions = append(questions, topicRegistryQuestionBankQuestionsFromRawJSONV0(field.ValueJSON)...)
		}
		if value := strings.TrimSpace(field.Value); value != "" {
			questions = append(questions, topicRegistryQuestionBankQuestionsFromRawJSONV0([]byte(value))...)
		}
		for _, value := range field.Values {
			if value = strings.TrimSpace(value); value != "" {
				questions = append(questions, topicRegistryQuestionBankQuestionsFromRawJSONV0([]byte(value))...)
			}
		}
	}
	return topicRegistryQuestionBankCompactQuestionsV0(questions)
}

func topicRegistryQuestionBankJSONFieldNameV0(name string) bool {
	switch strings.TrimSpace(name) {
	case "question_bank", "questions", "preguntas", "items", "banco_preguntas", "tests":
		return true
	default:
		return false
	}
}

func topicRegistryQuestionBankQuestionsFromRawJSONV0(raw []byte) []OPESQuestionBankQuestionV0 {
	if len(raw) == 0 || !json.Valid(raw) {
		return nil
	}
	var list []json.RawMessage
	if err := json.Unmarshal(raw, &list); err == nil {
		return topicRegistryQuestionBankQuestionsFromRawListV0(list)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return nil
	}
	for _, name := range []string{"questions", "preguntas", "items", "tests"} {
		if candidate, ok := object[name]; ok {
			var list []json.RawMessage
			if err := json.Unmarshal(candidate, &list); err == nil {
				return topicRegistryQuestionBankQuestionsFromRawListV0(list)
			}
		}
	}
	return nil
}

func topicRegistryQuestionBankQuestionsFromRawListV0(list []json.RawMessage) []OPESQuestionBankQuestionV0 {
	questions := make([]OPESQuestionBankQuestionV0, 0, len(list))
	for _, raw := range list {
		if question, ok := topicRegistryQuestionBankQuestionFromRawJSONV0(raw); ok {
			questions = append(questions, question)
		}
	}
	return questions
}

func topicRegistryQuestionBankQuestionFromRawJSONV0(raw []byte) (OPESQuestionBankQuestionV0, bool) {
	var object map[string]json.RawMessage
	if len(raw) == 0 || json.Unmarshal(raw, &object) != nil {
		return OPESQuestionBankQuestionV0{}, false
	}
	question := OPESQuestionBankQuestionV0{
		QuestionRef: topicRegistryQuestionBankStringFieldV0(object, "question_ref", "id", "ref"),
		Stem:        topicRegistryQuestionBankStringFieldV0(object, "stem", "question", "enunciado", "text", "prompt"),
		Options:     topicRegistryQuestionBankOptionsFromObjectV0(object),
		CorrectAnswerRefs: compactStringsV0(append(
			topicRegistryQuestionBankStringsFieldV0(object, "correct_answers", "correct_options"),
			topicRegistryQuestionBankStringFieldV0(
				object,
				"correct",
				"correcta",
				"correct_answer",
				"respuesta_correcta",
				"correct_option_id",
				"answer",
				"answer_id",
				"solution",
				"solucion",
			),
			topicRegistryQuestionBankIndexFieldV0(object, "correct_index", "correct_option_index"),
		)),
		Explanation: topicRegistryQuestionBankStringFieldV0(
			object,
			"explanation",
			"explicacion",
			"tutor_explanation",
			"feedback",
			"rationale",
		),
	}
	return question, topicRegistryQuestionBankQuestionHasDataV0(question)
}

func topicRegistryQuestionBankOptionsFromObjectV0(object map[string]json.RawMessage) []OPESQuestionBankOptionV0 {
	for _, name := range []string{"options", "opciones", "answers", "respuestas"} {
		raw, ok := object[name]
		if !ok {
			continue
		}
		var list []json.RawMessage
		if err := json.Unmarshal(raw, &list); err != nil {
			continue
		}
		return topicRegistryQuestionBankOptionsFromRawListV0(list)
	}
	return nil
}

func topicRegistryQuestionBankOptionsFromRawListV0(list []json.RawMessage) []OPESQuestionBankOptionV0 {
	options := make([]OPESQuestionBankOptionV0, 0, len(list))
	for _, raw := range list {
		var text string
		if err := json.Unmarshal(raw, &text); err == nil {
			options = append(options, OPESQuestionBankOptionV0{Text: text})
			continue
		}
		var object map[string]json.RawMessage
		if err := json.Unmarshal(raw, &object); err != nil {
			continue
		}
		options = append(options, OPESQuestionBankOptionV0{
			Ref:     topicRegistryQuestionBankStringFieldV0(object, "ref", "id", "label", "key"),
			Text:    topicRegistryQuestionBankStringFieldV0(object, "text", "value", "option", "respuesta", "label"),
			Correct: topicRegistryQuestionBankBoolFieldV0(object, "correct", "is_correct", "correcta"),
		})
	}
	return opesQuestionBankCompactOptionsV0(options)
}

func topicRegistryQuestionBankStringFieldV0(object map[string]json.RawMessage, names ...string) string {
	for _, name := range names {
		raw, ok := object[name]
		if !ok {
			continue
		}
		var text string
		if json.Unmarshal(raw, &text) == nil {
			return strings.TrimSpace(text)
		}
		var number int
		if json.Unmarshal(raw, &number) == nil {
			return strconv.Itoa(number)
		}
	}
	return ""
}

func topicRegistryQuestionBankStringsFieldV0(object map[string]json.RawMessage, names ...string) []string {
	var out []string
	for _, name := range names {
		raw, ok := object[name]
		if !ok {
			continue
		}
		var texts []string
		if json.Unmarshal(raw, &texts) == nil {
			out = append(out, texts...)
			continue
		}
		if text := topicRegistryQuestionBankStringFieldV0(object, name); text != "" {
			out = append(out, text)
		}
	}
	return compactStringsV0(out)
}

func topicRegistryQuestionBankIndexFieldV0(object map[string]json.RawMessage, names ...string) string {
	for _, name := range names {
		raw, ok := object[name]
		if !ok {
			continue
		}
		var number int
		if json.Unmarshal(raw, &number) == nil {
			return strconv.Itoa(number)
		}
	}
	return ""
}

func topicRegistryQuestionBankBoolFieldV0(object map[string]json.RawMessage, names ...string) bool {
	for _, name := range names {
		raw, ok := object[name]
		if !ok {
			continue
		}
		var value bool
		if json.Unmarshal(raw, &value) == nil {
			return value
		}
		var text string
		if json.Unmarshal(raw, &text) == nil {
			switch strings.ToLower(strings.TrimSpace(text)) {
			case "true", "1", "yes", "si", "correct", "correcta":
				return true
			}
		}
	}
	return false
}

func topicRegistryQuestionBankQuestionHasDataV0(question OPESQuestionBankQuestionV0) bool {
	return strings.TrimSpace(question.QuestionRef) != "" ||
		strings.TrimSpace(question.Stem) != "" ||
		len(question.Options) > 0 ||
		len(question.CorrectAnswerRefs) > 0 ||
		strings.TrimSpace(question.Explanation) != ""
}

func topicRegistryQuestionBankCompactQuestionsV0(
	questions []OPESQuestionBankQuestionV0,
) []OPESQuestionBankQuestionV0 {
	out := make([]OPESQuestionBankQuestionV0, 0, len(questions))
	for _, question := range questions {
		if !topicRegistryQuestionBankQuestionHasDataV0(question) {
			continue
		}
		out = append(out, question)
	}
	return out
}
