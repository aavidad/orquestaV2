package orquestaobservability

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

func validateOperationalAreaV0(area string, field string, add func(string, string)) {
	trimmed := strings.TrimSpace(area)
	if !allowedV0(allowedOperationalAreasV0, trimmed) {
		add(ErrOperationalStatusQueryInvalidaV0, field)
		return
	}
	validateOperationalTokenTextV0(trimmed, field, add)
}

func validateOperationalI18nKeyV0(value string, field string, add func(string, string)) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || !validSizedPatternV0(trimmed, 1, maxOperationalI18nKeyRunesV0, operationalI18nKeyPatternV0) {
		add(ErrOperationalStatusQueryInvalidaV0, field)
		return
	}
	validateOperationalTextV0(trimmed, field, maxOperationalI18nKeyRunesV0, true, add)
}

func validateOperationalTokenTextV0(value string, field string, add func(string, string)) {
	validateOperationalTextV0(value, field, maxOperationalTokenRunesV0, true, add)
	trimmed := strings.TrimSpace(value)
	if trimmed != "" && !validSizedPatternV0(trimmed, 1, maxOperationalTokenRunesV0, operationalTokenPatternV0) {
		add(ErrOperationalStatusQueryInvalidaV0, field)
	}
}

func validateOperationalRefListV0(refs []string, maxItems int, prefix string, add func(string, string)) {
	if len(refs) > maxItems {
		add(ErrConsultaDemasiadoAmpliaV0, prefix)
	}
	for index, ref := range refs {
		validateRequiredOperationalRefV0(ref, fmt.Sprintf("%s[%d]", prefix, index), add)
	}
}

func validateRequiredOperationalRefV0(value string, field string, add func(string, string)) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		add(ErrOperationalStatusQueryInvalidaV0, field)
		return
	}
	validateOperationalRefV0(trimmed, field, add)
}

func validateOptionalOperationalRefV0(value string, field string, add func(string, string)) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return
	}
	validateOperationalRefV0(trimmed, field, add)
}

func validateOperationalRefV0(value string, field string, add func(string, string)) {
	if !isOpaqueIDV0(value) {
		add(ErrReferenciaNoOpacaV0, field)
		return
	}
	if code := forbiddenOperationalTextCodeV0(value); code != "" {
		add(code, field)
	}
}

func validateOperationalTextV0(value string, field string, maxRunes int, required bool, add func(string, string)) {
	trimmed := strings.TrimSpace(value)
	if required && trimmed == "" {
		add(ErrOperationalStatusQueryInvalidaV0, field)
		return
	}
	if trimmed == "" {
		return
	}
	if utf8.RuneCountInString(trimmed) > maxRunes {
		add(ErrConsultaDemasiadoAmpliaV0, field)
	}
	if code := forbiddenOperationalTextCodeV0(trimmed); code != "" {
		add(code, field)
	}
}

func forbiddenOperationalTextCodeV0(value string) string {
	lower := strings.ToLower(value)
	if lower == "home" {
		return ErrOperationalStatusQueryInvalidaV0
	}
	if containsAnyV0(lower, operationalTranscriptTextPartsV0) {
		return ErrTranscriptNoPermitidoV0
	}
	if containsAnyV0(lower, operationalSecretTextPartsV0) {
		return ErrSecretoDetectadoV0
	}
	if containsAnyV0(lower, operationalForbiddenTextPartsV0) {
		return ErrOperationalStatusQueryInvalidaV0
	}
	return ""
}

func validateOperationalJSONSizeV0(value any, maxBytes int, field string, add func(string, string)) {
	data, err := json.Marshal(value)
	if err != nil {
		add(ErrOperationalStatusQueryInvalidaV0, field)
		return
	}
	if len(data) > maxBytes {
		add(ErrConsultaDemasiadoAmpliaV0, field)
	}
}
