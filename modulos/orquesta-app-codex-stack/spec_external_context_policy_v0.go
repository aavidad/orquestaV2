package orquestaappcodexstack

import (
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
)

const (
	externalWorkContextMaxFieldsV0 = 24

	externalWorkContextCompactFieldBytesV0  = 1800
	externalWorkContextStandardFieldBytesV0 = 4000
	externalWorkContextLargeFieldBytesV0    = 12000

	externalWorkContextCompactTotalBytesV0  = 18000
	externalWorkContextStandardTotalBytesV0 = 36000
	externalWorkContextLargeTotalBytesV0    = 72000
)

type externalWorkContextPolicyV0 struct {
	MaxFields     int
	FieldMaxBytes int
	TotalMaxBytes int
}

func externalWorkContextPolicyForWorkV0(
	work *orquestaappchange.AppChangeExternalWorkV0,
) externalWorkContextPolicyV0 {
	policy := externalWorkContextStandardPolicyV0()
	if externalWorkContextLongformKindV0(work) {
		policy = externalWorkContextLargePolicyV0()
	}
	switch externalWorkContextProfileV0(work) {
	case "compact":
		policy = externalWorkContextCompactPolicyV0()
	case "standard":
		policy = externalWorkContextStandardPolicyV0()
	case "large":
		policy = externalWorkContextLargePolicyV0()
	}
	return policy
}

func externalWorkContextCompactPolicyV0() externalWorkContextPolicyV0 {
	return externalWorkContextPolicyV0{
		MaxFields:     externalWorkContextMaxFieldsV0,
		FieldMaxBytes: externalWorkContextCompactFieldBytesV0,
		TotalMaxBytes: externalWorkContextCompactTotalBytesV0,
	}
}

func externalWorkContextStandardPolicyV0() externalWorkContextPolicyV0 {
	return externalWorkContextPolicyV0{
		MaxFields:     externalWorkContextMaxFieldsV0,
		FieldMaxBytes: externalWorkContextStandardFieldBytesV0,
		TotalMaxBytes: externalWorkContextStandardTotalBytesV0,
	}
}

func externalWorkContextLargePolicyV0() externalWorkContextPolicyV0 {
	return externalWorkContextPolicyV0{
		MaxFields:     externalWorkContextMaxFieldsV0,
		FieldMaxBytes: externalWorkContextLargeFieldBytesV0,
		TotalMaxBytes: externalWorkContextLargeTotalBytesV0,
	}
}

func externalWorkContextLongformKindV0(
	work *orquestaappchange.AppChangeExternalWorkV0,
) bool {
	if work == nil {
		return false
	}
	switch strings.TrimSpace(work.WorkKind) {
	case "draft_content_block",
		"expand_topic_from_summary",
		"review_legal",
		"review_pedagogical",
		"review_quality",
		"validate_topic",
		"assemble_topic",
		"verify_sources":
		return true
	default:
		return false
	}
}

func externalWorkContextProfileV0(
	work *orquestaappchange.AppChangeExternalWorkV0,
) string {
	if work == nil {
		return ""
	}
	for _, field := range work.InputFields {
		name := strings.ToLower(strings.TrimSpace(field.Name))
		if name != "context_budget_profile" &&
			name != "context_profile" &&
			name != "work_granularity" &&
			name != "editorial_granularity" {
			continue
		}
		if profile := normalizeExternalWorkContextProfileV0(field.Value); profile != "" {
			return profile
		}
		for _, value := range field.Values {
			if profile := normalizeExternalWorkContextProfileV0(value); profile != "" {
				return profile
			}
		}
	}
	return ""
}

func normalizeExternalWorkContextProfileV0(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "compact", "small", "summary", "resumen":
		return "compact"
	case "standard", "normal", "default":
		return "standard"
	case "large", "long", "longform", "chapter", "capitulo", "topic", "tema", "expansion":
		return "large"
	default:
		return ""
	}
}
