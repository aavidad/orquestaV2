package orquestaappcodexstack

import (
	"fmt"
	"strings"
	"unicode"

	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

const compositeObjectiveMarkerPrefixV0 = "objetivo_actual:"

var compositeObjectiveStopwordsV0 = map[string]bool{
	"aplicacion": true,
	"completa":   true,
	"crear":      true,
	"desde":      true,
	"existente":  true,
	"modificar":  true,
	"modulo":     true,
	"nueva":      true,
	"para":       true,
	"programar":  true,
	"request":    true,
	"sistema":    true,
	"solicitud":  true,
}

func validateCompositeProgrammingMicrotaskAnchorsV0(
	requestKind string,
	objectiveHints []string,
	tasks []orquestadirectoragent.DirectorAgentMicrotaskV0,
) error {
	if len(tasks) == 0 || len(compactStringsV0(objectiveHints)) == 0 {
		return nil
	}
	tokens := compositeObjectiveTokensV0(objectiveHints)
	for _, task := range tasks {
		if !compositeTaskHasObjectiveMarkerV0(task) {
			return compositeObjectiveAnchorErrorV0(requestKind, task, "sin objetivo_actual")
		}
		if len(tokens) > 0 && !compositeTaskMentionsObjectiveTokenV0(task, tokens) {
			return compositeObjectiveAnchorErrorV0(requestKind, task, "sin token del objetivo actual")
		}
	}
	return nil
}

func compositeTaskHasObjectiveMarkerV0(
	task orquestadirectoragent.DirectorAgentMicrotaskV0,
) bool {
	for _, criterion := range task.AcceptanceCriteria {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(criterion)), compositeObjectiveMarkerPrefixV0) {
			return true
		}
	}
	return false
}

func compositeTaskMentionsObjectiveTokenV0(
	task orquestadirectoragent.DirectorAgentMicrotaskV0,
	tokens map[string]bool,
) bool {
	taskTokens := compositeObjectiveTokensV0([]string{compositeTaskTextV0(task)})
	for token := range tokens {
		if taskTokens[token] {
			return true
		}
	}
	return false
}

func compositeTaskTextV0(
	task orquestadirectoragent.DirectorAgentMicrotaskV0,
) string {
	values := []string{task.TaskID, task.Title, task.Summary}
	values = append(values, task.AcceptanceCriteria...)
	values = append(values, task.RequiredTests...)
	values = append(values, task.WriteSet...)
	return strings.ToLower(strings.Join(values, "\n"))
}

func compositeObjectiveTokensV0(values []string) map[string]bool {
	tokens := map[string]bool{}
	for _, value := range values {
		for _, token := range strings.FieldsFunc(strings.ToLower(value), compositeObjectiveSeparatorV0) {
			if compositeObjectiveTokenUsefulV0(token) {
				tokens[token] = true
			}
		}
	}
	return tokens
}

func compositeObjectiveSeparatorV0(r rune) bool {
	return !unicode.IsLetter(r) && !unicode.IsDigit(r)
}

func compositeObjectiveTokenUsefulV0(token string) bool {
	if len(token) < 4 {
		return false
	}
	return !compositeObjectiveStopwordsV0[token]
}

func compositeObjectiveAnchorErrorV0(
	requestKind string,
	task orquestadirectoragent.DirectorAgentMicrotaskV0,
	reason string,
) error {
	kind := strings.TrimSpace(requestKind)
	if kind == "" {
		kind = "request"
	}
	taskID := strings.TrimSpace(task.TaskID)
	if taskID == "" {
		taskID = "task"
	}
	return fmt.Errorf(
		"director_decisions invalidas: create_microtask %s para %s %s",
		reason,
		kind,
		taskID,
	)
}
