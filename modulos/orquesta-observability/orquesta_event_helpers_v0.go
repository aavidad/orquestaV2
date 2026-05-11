package orquestaobservability

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

func validateOptionalOpaqueIDV0(value string, field string, add func(string, string)) {
	if strings.TrimSpace(value) == "" {
		return
	}
	if !isOpaqueIDV0(value) {
		add(ErrOrquestaEventInvalidoV0, field)
	}
}

func isOpaqueIDV0(value string) bool {
	return validSizedPatternV0(strings.TrimSpace(value), minOpaqueIDRunesV0, maxOpaqueIDRunesV0, opaqueIDPatternV0)
}

func isOccurredAtV0(value string) bool {
	trimmed := strings.TrimSpace(value)
	if !strings.HasSuffix(trimmed, "Z") {
		return false
	}
	_, err := time.Parse(time.RFC3339Nano, trimmed)
	return err == nil
}

func validSizedPatternV0(value string, minRunes, maxRunes int, pattern *regexp.Regexp) bool {
	runes := utf8.RuneCountInString(value)
	return runes >= minRunes && runes <= maxRunes && pattern.MatchString(value)
}

func allowedV0(values map[string]bool, value string) bool {
	return values[value]
}

func setV0(values ...string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}

func containsAnyV0(value string, parts []string) bool {
	for _, part := range parts {
		if strings.Contains(value, part) {
			return true
		}
	}
	return false
}

func fieldV0(prefix, field string) string {
	if prefix == "" {
		return field
	}
	if field == "" {
		return prefix
	}
	return prefix + "." + field
}

func addIssueFuncV0(issues *[]OrquestaEventValidationIssueV0) func(string, string) {
	return func(code, field string) {
		*issues = append(*issues, OrquestaEventValidationIssueV0{Code: code, Field: field})
	}
}

func validationErrorV0(code, field string) OrquestaEventValidationErrorV0 {
	return OrquestaEventValidationErrorV0{Issues: []OrquestaEventValidationIssueV0{{Code: code, Field: field}}}
}

func decodeStrictJSONV0(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}
