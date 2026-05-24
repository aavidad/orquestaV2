package orquestaappchange

import (
	"path"
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func validateAppChangeRequestV0(request AppChangeRequestV0) []AppChangeIssueV0 {
	var issues []AppChangeIssueV0
	if request.RunRef == "" {
		issues = append(issues, appChangeIssueV0(ErrAppChangeRunRefRequiredV0, "run_ref"))
	} else if !isCompactAppChangeRefV0(request.RunRef) {
		issues = append(issues, appChangeIssueV0(ErrAppChangeRunRefInvalidV0, "run_ref"))
	}
	if request.ChangeRef == "" {
		issues = append(issues, appChangeIssueV0(ErrAppChangeChangeRefRequiredV0, "change_ref"))
	} else if !isCompactAppChangeRefV0(request.ChangeRef) {
		issues = append(issues, appChangeIssueV0(ErrAppChangeChangeRefInvalidV0, "change_ref"))
	}
	if request.UserIntent == "" {
		issues = append(issues, appChangeIssueV0(ErrAppChangeIntentRequiredV0, "user_intent"))
	}
	for _, ref := range request.CurrentStateRefs {
		if !isCompactAppChangeRefV0(ref) {
			issues = append(issues, appChangeIssueV0(ErrAppChangeCurrentStateRefV0, "current_state_refs"))
			break
		}
	}
	for _, ref := range request.MetadataRefs {
		if !isCompactAppChangeRefV0(ref) {
			issues = append(issues, appChangeIssueV0(ErrAppChangeMetadataRefV0, "metadata_refs"))
			break
		}
	}
	issues = append(issues, validateAppChangeExternalWorkV0(request.ExternalWork)...)
	for _, entry := range request.AllowedWriteSet {
		if !isSafeRelativeWriteSetV0(entry) {
			issues = append(issues, appChangeIssueV0(ErrAppChangeWriteSetInvalidV0, "allowed_write_set"))
			break
		}
	}
	return issues
}

func validateAppChangeExternalWorkV0(work *AppChangeExternalWorkV0) []AppChangeIssueV0 {
	if work == nil {
		return nil
	}
	var issues []AppChangeIssueV0
	if work.ProjectRef != "" && !isCompactAppChangeRefV0(work.ProjectRef) {
		issues = append(issues, appChangeIssueV0(ErrAppChangeExternalWorkRefV0, "external_work"))
	}
	if work.JobRef != "" && !isCompactAppChangeRefV0(work.JobRef) {
		issues = append(issues, appChangeIssueV0(ErrAppChangeExternalWorkRefV0, "external_work"))
	}
	if work.WorkKind != "" && !isCompactAppChangeRefV0(work.WorkKind) {
		issues = append(issues, appChangeIssueV0(ErrAppChangeExternalWorkRefV0, "external_work"))
	}
	for _, ref := range work.InterfaceRefs {
		if !isCompactAppChangeRefV0(ref) {
			issues = append(issues, appChangeIssueV0(ErrAppChangeExternalWorkRefV0, "external_work"))
			break
		}
	}
	for _, ref := range work.WorkRefs {
		if !isCompactAppChangeRefV0(ref) {
			issues = append(issues, appChangeIssueV0(ErrAppChangeExternalWorkRefV0, "external_work"))
			break
		}
	}
	for _, field := range work.InputFields {
		if !isCompactAppChangeRefV0(field.Name) {
			issues = append(
				issues,
				appChangeIssueV0(
					ErrAppChangeExternalWorkFieldNameV0,
					"external_work.input_fields",
				),
			)
			break
		}
	}
	if domainIssues := orquestadomainwork.ValidateDomainWorkJobRequestV0(
		orquestadomainwork.DomainWorkJobRequestV0{
			RequestID:     "app-change-required-tests-validation",
			DomainRef:     firstAppChangeExternalWorkValidationRefV0(work.ProjectRef),
			WorkKind:      firstAppChangeExternalWorkValidationRefV0(work.WorkKind),
			Objective:     "validar required_tests externos",
			RequiredTests: work.RequiredTests,
		},
	); len(domainIssues) > 0 && len(work.RequiredTests) > 0 {
		issues = append(issues, appChangeIssueV0(
			ErrAppChangeExternalWorkRefV0,
			"external_work.required_tests",
		))
	}
	return issues
}

func firstAppChangeExternalWorkValidationRefV0(value string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return "domain-work-ref-validation"
}

func validateAppChangePortsV0(ports AppChangePortsV0) []AppChangeIssueV0 {
	var issues []AppChangeIssueV0
	if ports.Store == nil {
		issues = append(issues, appChangeIssueV0(ErrAppChangePortUnavailableV0, "store"))
	}
	if ports.DirectorNotifier == nil {
		issues = append(issues, appChangeIssueV0(ErrAppChangePortUnavailableV0, "director_notifier"))
	}
	return issues
}

func isCompactAppChangeRefV0(ref string) bool {
	if strings.TrimSpace(ref) == "" {
		return false
	}
	return !strings.ContainsAny(ref, " /\\\t\n\r")
}

func isSafeRelativeWriteSetV0(entry string) bool {
	trimmed := strings.TrimSpace(entry)
	if trimmed == "" || strings.HasPrefix(trimmed, "/") || strings.Contains(trimmed, "\\") {
		return false
	}
	clean := path.Clean(trimmed)
	return clean != "." && clean != ".." && !strings.HasPrefix(clean, "../")
}

func appChangeIssueV0(code string, field string) AppChangeIssueV0 {
	return AppChangeIssueV0{Code: code, Field: field}
}
