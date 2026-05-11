package orquestafactory

import (
	"strings"
	"time"
)

func assembleAppSpecV0(req AppSpecRequestV0, now time.Time) AppSpecV0 {
	normalizer := appSpecNormalizerV0{req: req}
	return AppSpecV0{
		SchemaVersion:    AppSpecSchemaV0,
		SpecID:           buildSpecIDV0(req),
		RequestID:        strings.TrimSpace(req.RequestID),
		CreatedAt:        now.UTC().Format(time.RFC3339),
		Locale:           strings.TrimSpace(req.Locale),
		RequestKind:      normalizer.requestKind(),
		ExecutionMode:    normalizer.executionMode(),
		App:              normalizer.app(),
		Scope:            normalizer.scope(),
		Architecture:     normalizer.architecture(),
		I18N:             normalizer.i18n(),
		Data:             normalizer.data(),
		Connectors:       normalizer.connectors(),
		Platforms:        normalizer.platforms(),
		Deploy:           normalizer.deploy(),
		Quality:          normalizer.quality(),
		Docs:             normalizer.docs(),
		AgentPreferences: normalizer.agentPreferences(),
		DefaultsApplied:  normalizer.defaultsApplied(),
		Validation:       validAppSpecValidationV0(),
	}
}

func validAppSpecValidationV0() ValidationSummaryV0 {
	return ValidationSummaryV0{
		Estado:   "valida",
		Warnings: []ValidationIssue{},
		Errores:  []ValidationIssue{},
	}
}
