package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func selectOPESRegistryFinalPkgCandidatesV0(
	registry opesRegistryFinalPkgRegistryV0,
	config opesRegistryFinalPkgConfigV0,
	existing map[string]struct{},
	active map[string]opesRegistryFinalPkgRunProjectionV0,
) ([]string, error) {
	course, ok := registry.Courses[strings.TrimSpace(config.CourseID)]
	if !ok {
		return nil, fmt.Errorf("course_not_found")
	}
	topicIDs := make([]string, 0, len(course.Topics))
	for topicID := range course.Topics {
		topicIDs = append(topicIDs, strings.TrimSpace(topicID))
	}
	sort.Strings(topicIDs)
	out := make([]string, 0, len(topicIDs))
	for _, topicID := range topicIDs {
		if opesRegistryFinalPkgTopicBlockedV0(config, course.Topics, existing, active, topicID) {
			continue
		}
		out = append(out, topicID)
	}
	return out, nil
}

func opesRegistryFinalPkgTopicBlockedV0(
	config opesRegistryFinalPkgConfigV0,
	topics map[string]opesRegistryFinalPkgTopicV0,
	existing map[string]struct{},
	active map[string]opesRegistryFinalPkgRunProjectionV0,
	topicID string,
) bool {
	if topicID == "" {
		return true
	}
	runRef := opesRegistryFinalPkgRunRefV0(config, topicID)
	if _, ok := existing[runRef]; ok {
		return true
	}
	if _, ok := active[runRef]; ok {
		return true
	}
	if opesRegistryFinalPkgTopicHasActiveLockV0(topics[topicID]) {
		return true
	}
	return opesRegistryFinalPkgPackageCompleteV0(filepath.Join(config.CourseRoot, "tema_"+topicID, "paquete_final"))
}

func opesRegistryFinalPkgRunCountsAsInFlightV0(
	config opesRegistryFinalPkgConfigV0,
	runRef string,
	projection opesRegistryFinalPkgRunProjectionV0,
) bool {
	topicID, ok := opesRegistryFinalPkgTopicIDFromRunRefV0(config, runRef)
	if !ok {
		return true
	}
	if !opesRegistryFinalPkgPackageCompleteV0(filepath.Join(config.CourseRoot, "tema_"+topicID, "paquete_final")) {
		return true
	}
	if len(projection.Deliveries) > 0 ||
		len(projection.ClosedTasks) > 0 ||
		len(projection.AcceptedReviews) > 0 ||
		len(projection.Blockers) > 0 {
		return false
	}
	return true
}

func opesRegistryFinalPkgTopicHasActiveLockV0(topic opesRegistryFinalPkgTopicV0) bool {
	if len(topic.Lock) == 0 {
		return false
	}
	for _, key := range []string{"agent_id", "owner"} {
		if value, ok := topic.Lock[key]; ok && strings.TrimSpace(fmt.Sprint(value)) != "" {
			return true
		}
	}
	return false
}

func opesRegistryFinalPkgPackageCompleteV0(packageDir string) bool {
	return validateOPESRegistryFinalPkgPackageV0(packageDir).Complete
}

type opesRegistryFinalPkgPackageValidationV0 struct {
	Complete bool
	Issues   []string
}

func validateOPESRegistryFinalPkgPackageV0(packageDir string) opesRegistryFinalPkgPackageValidationV0 {
	var issues []string
	for _, relative := range requiredOPESRegistryFinalPkgFilesV0 {
		info, err := os.Stat(filepath.Join(packageDir, relative))
		if err != nil || info.IsDir() || info.Size() <= 0 {
			issues = append(issues, "missing_or_empty:"+relative)
		}
	}
	data, err := os.ReadFile(filepath.Join(packageDir, "tests.json"))
	if err != nil {
		issues = append(issues, "tests_json_missing")
	} else if !opesRegistryFinalPkgTestsJSONHasPublishableQuestionsV0(data) {
		issues = append(issues, "tests_json_without_questions")
	}
	return opesRegistryFinalPkgPackageValidationV0{
		Complete: len(issues) == 0,
		Issues:   compactOPESRegistryFinalPkgStringsV0(issues),
	}
}

func opesRegistryFinalPkgTestsJSONHasPublishableQuestionsV0(data []byte) bool {
	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return false
	}
	return opesRegistryFinalPkgJSONHasPublishableQuestionsV0(decoded)
}

func opesRegistryFinalPkgJSONHasPublishableQuestionsV0(value any) bool {
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if opesRegistryFinalPkgJSONQuestionPublishableV0(item) {
				return true
			}
		}
	case map[string]any:
		for _, key := range []string{"questions", "preguntas", "items", "tests", "question_bank"} {
			if opesRegistryFinalPkgJSONHasPublishableQuestionsV0(typed[key]) {
				return true
			}
		}
	}
	return false
}

func opesRegistryFinalPkgJSONQuestionPublishableV0(value any) bool {
	typed, ok := value.(map[string]any)
	if !ok {
		return false
	}
	return opesRegistryFinalPkgJSONStringAnyV0(typed, "prompt", "enunciado", "question", "pregunta", "text") != "" &&
		opesRegistryFinalPkgJSONOptionsCountV0(typed) >= 2 &&
		opesRegistryFinalPkgJSONStringAnyV0(typed, "answer", "respuesta", "correct_answer", "correcta", "correct_option") != ""
}

func opesRegistryFinalPkgJSONStringAnyV0(value map[string]any, keys ...string) string {
	for _, key := range keys {
		if text, ok := value[key].(string); ok && strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func opesRegistryFinalPkgJSONOptionsCountV0(value map[string]any) int {
	for _, key := range []string{"options", "opciones", "answers", "respuestas", "choices"} {
		if options, ok := value[key].([]any); ok {
			count := 0
			for _, option := range options {
				if strings.TrimSpace(fmt.Sprint(option)) != "" {
					count++
				}
			}
			return count
		}
	}
	return 0
}

func opesRegistryFinalPkgRunRefV0(config opesRegistryFinalPkgConfigV0, topicID string) string {
	return strings.ReplaceAll(strings.TrimSpace(config.TemplateRunRef), strings.TrimSpace(config.TemplateTopicID), strings.TrimSpace(topicID))
}

func opesRegistryFinalPkgRunRefPartsV0(config opesRegistryFinalPkgConfigV0) (string, string) {
	prefix, suffix, ok := strings.Cut(strings.TrimSpace(config.TemplateRunRef), strings.TrimSpace(config.TemplateTopicID))
	if !ok {
		return strings.TrimSpace(config.TemplateRunRef), ""
	}
	return prefix, suffix
}

func opesRegistryFinalPkgTopicIDFromRunRefV0(
	config opesRegistryFinalPkgConfigV0,
	runRef string,
) (string, bool) {
	prefix, suffix := opesRegistryFinalPkgRunRefPartsV0(config)
	trimmed := strings.TrimSpace(runRef)
	if !strings.HasPrefix(trimmed, prefix) || !strings.HasSuffix(trimmed, suffix) {
		return "", false
	}
	topicID := strings.TrimSuffix(strings.TrimPrefix(trimmed, prefix), suffix)
	return strings.TrimSpace(topicID), strings.TrimSpace(topicID) != ""
}

func compactOPESRegistryFinalPkgStringsV0(values []string) []string {
	out := []string{}
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
