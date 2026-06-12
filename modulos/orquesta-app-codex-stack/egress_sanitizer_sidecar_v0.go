package orquestaappcodexstack

import (
	"strings"

	orquestacontext "orquesta/modulos/orquesta-context"
)

type EgressSanitizerContextSanitizerV0 struct {
	Config        EgressSanitizerConfigV0
	Deterministic LocalSensitiveDataSanitizerV0
	Sidecar       PrivacyFilterSidecarPortV0
}

func NewEgressSanitizerContextSanitizerV0(
	config EgressSanitizerConfigV0,
) EgressSanitizerContextSanitizerV0 {
	config = NormalizeEgressSanitizerConfigV0(config)
	return EgressSanitizerContextSanitizerV0{
		Config: config,
		Deterministic: LocalSensitiveDataSanitizerV0{
			SanitizerRef: config.SanitizerRef,
			Egress:       config,
		},
		Sidecar: config.Sidecar.Port,
	}
}

func (sanitizer EgressSanitizerContextSanitizerV0) SanitizeContextEntryV0(
	request orquestacontext.ContextSanitizationRequestV0,
) orquestacontext.ContextSanitizationResultV0 {
	config := NormalizeEgressSanitizerConfigV0(sanitizer.Config)
	sidecar := sanitizer.Sidecar
	if sidecar == nil {
		sidecar = config.Sidecar.Port
	}
	preSanitized := sanitizer.fallbackDeterministicV0(config, request)
	if preSanitized.Status == orquestacontext.ContextSanitizationStatusReviewRequiredV0 ||
		preSanitized.Status == orquestacontext.ContextSanitizationStatusBlockedV0 {
		return preSanitized
	}
	if config.Enabled && config.Sidecar.Enabled && sidecar != nil {
		sidecarRequest := request
		if strings.TrimSpace(preSanitized.Content) != "" {
			sidecarRequest.Content = preSanitized.Content
		}
		return sanitizer.resultFromSidecarV0(config, sidecar, sidecarRequest, preSanitized)
	}
	return preSanitized
}

func (sanitizer EgressSanitizerContextSanitizerV0) resultFromSidecarV0(
	config EgressSanitizerConfigV0,
	sidecar PrivacyFilterSidecarPortV0,
	request orquestacontext.ContextSanitizationRequestV0,
	preSanitized orquestacontext.ContextSanitizationResultV0,
) orquestacontext.ContextSanitizationResultV0 {
	result := sidecar.FilterEgressPayloadV0(PrivacyFilterSidecarRequestV0{
		RequestRef: request.WorkOrderRef,
		EntryRef:   request.EntryRef,
		SourceRef:  request.SourceRef,
		Payload:    request.Content,
	})
	categories := map[string]bool{
		"egress_sanitizer":              true,
		"openai_privacy_filter_local":   true,
		"local_privacy_filter_sidecar":  true,
		"privacy_filter_sidecar_result": true,
	}
	for _, category := range preSanitized.Evidence.Categories {
		category = strings.TrimSpace(category)
		if category != "" {
			categories[category] = true
		}
	}
	for _, category := range result.Categories {
		category = strings.TrimSpace(category)
		if category == "" {
			continue
		}
		if privacyFilterSidecarCategoryHasForbiddenDetailV0(category) {
			category = "privacy-filter-category-" + safeContextSanitizerPartV0(category)
		}
		categories[category] = true
	}
	content := result.Content
	status := orquestacontext.ContextSanitizationStatusCleanV0
	reviewRequired := result.ReviewRequired
	if strings.TrimSpace(content) == "" && result.Sanitized {
		reviewRequired = true
	}
	if strings.TrimSpace(content) == "" && !reviewRequired {
		content = request.Content
	}
	replacementCount := preSanitized.Evidence.ReplacementCount + result.ReplacementCount
	if reviewRequired {
		status = orquestacontext.ContextSanitizationStatusReviewRequiredV0
		content = ""
		categories["privacy_filter_sidecar_review_required"] = true
	} else if preSanitized.Status == orquestacontext.ContextSanitizationStatusSanitizedV0 ||
		result.Sanitized ||
		content != request.Content ||
		replacementCount > 0 {
		status = orquestacontext.ContextSanitizationStatusSanitizedV0
	}
	local := LocalSensitiveDataSanitizerV0{
		SanitizerRef: firstCodexStackStringV0(config.SanitizerRef, sanitizer.Deterministic.SanitizerRef),
		Egress:       config,
	}
	return orquestacontext.ContextSanitizationResultV0{
		Status:  status,
		Content: content,
		Evidence: local.evidenceV0(
			request,
			status,
			categories,
			replacementCount,
			reviewRequired,
		),
	}
}

func (sanitizer EgressSanitizerContextSanitizerV0) fallbackDeterministicV0(
	config EgressSanitizerConfigV0,
	request orquestacontext.ContextSanitizationRequestV0,
) orquestacontext.ContextSanitizationResultV0 {
	local := sanitizer.Deterministic
	if strings.TrimSpace(local.SanitizerRef) == "" {
		local.SanitizerRef = config.SanitizerRef
	}
	local.Egress = config
	return local.SanitizeContextEntryV0(request)
}
