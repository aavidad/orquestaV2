package orquestadirectoroperativo

import "strings"

func validateOperationalDirectorRequestV0(
	request OperationalDirectorRequestV0,
) []OperationalDirectorIssueV0 {
	var issues []OperationalDirectorIssueV0
	for _, required := range []struct {
		field string
		value string
	}{
		{field: "request_ref", value: request.RequestRef},
		{field: "run_ref", value: request.RunRef},
		{field: "project_ref", value: request.ProjectRef},
		{field: "objective", value: request.Objective},
	} {
		if strings.TrimSpace(required.value) == "" {
			issues = append(issues, issueV0(required.field+"_missing", required.field, required.field+" requerido"))
		}
	}
	switch request.Mode {
	case OperationalDirectorModeProgrammingV0:
		issues = append(issues, validateOperationalDirectorProgrammingV0(request)...)
	case OperationalDirectorModeDomainWorkV0:
		issues = append(issues, validateOperationalDirectorDomainWorkV0(request)...)
	default:
		issues = append(issues, issueV0("mode_unsupported", "mode", "modo no soportado"))
	}
	return issues
}

func validateOperationalDirectorProgrammingV0(
	request OperationalDirectorRequestV0,
) []OperationalDirectorIssueV0 {
	var issues []OperationalDirectorIssueV0
	if strings.TrimSpace(request.WorktreeRef) == "" {
		issues = append(issues, issueV0("worktree_ref_missing", "worktree_ref", "worktree requerida"))
	}
	if !request.WorktreeIsolated {
		issues = append(issues, issueV0("worktree_not_isolated", "worktree_isolated", "worktree aislada requerida"))
	}
	if strings.TrimSpace(request.BranchRef) == "" {
		issues = append(issues, issueV0("branch_ref_missing", "branch_ref", "rama requerida"))
	}
	if len(compactStringsV0(request.RequiredTests)) == 0 {
		issues = append(issues, issueV0("required_tests_missing", "required_tests", "tests obligatorios requeridos"))
	}
	issues = append(issues, validateOperationalDirectorWriteSetV0(request.WriteSet)...)
	return issues
}

func validateOperationalDirectorDomainWorkV0(
	request OperationalDirectorRequestV0,
) []OperationalDirectorIssueV0 {
	status := request.ContextStatus
	if status == "" {
		status = OperationalDirectorContextSufficientV0
	}
	if status != OperationalDirectorContextSufficientV0 &&
		status != OperationalDirectorContextInsufficientV0 {
		return []OperationalDirectorIssueV0{issueV0("context_status_unsupported", "context_status", "estado de contexto no soportado")}
	}
	if status == OperationalDirectorContextSufficientV0 &&
		len(compactStringsV0(request.DomainRefs)) == 0 {
		return []OperationalDirectorIssueV0{issueV0("domain_refs_missing", "domain_refs", "refs de dominio requeridas")}
	}
	if status == OperationalDirectorContextInsufficientV0 &&
		len(compactStringsV0(request.MissingContext)) == 0 {
		return []OperationalDirectorIssueV0{issueV0("missing_context_required", "missing_context", "campos faltantes requeridos")}
	}
	return nil
}

func validateOperationalDirectorWriteSetV0(
	writeSet []string,
) []OperationalDirectorIssueV0 {
	writeSet = compactStringsV0(writeSet)
	if len(writeSet) == 0 {
		return []OperationalDirectorIssueV0{issueV0("write_set_missing", "write_set", "write-set requerido")}
	}
	var issues []OperationalDirectorIssueV0
	for _, path := range writeSet {
		if !operationalDirectorWriteSetPathSafeV0(path) {
			issues = append(issues, issueV0("write_set_path_invalid", "write_set", "ruta de write-set no permitida"))
		}
	}
	return issues
}

func operationalDirectorWriteSetPathSafeV0(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" || strings.HasPrefix(path, "/") || strings.HasPrefix(path, "~") {
		return false
	}
	if strings.Contains(path, "\x00") || strings.Contains(path, "\\") ||
		strings.Contains(path, "://") || strings.Contains(path, "$") {
		return false
	}
	for _, segment := range strings.Split(path, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	lower := strings.ToLower(path)
	for _, forbidden := range []string{"secret", "secreto", "token", "password", "credential", "credencial", "api_key"} {
		if strings.Contains(lower, forbidden) {
			return false
		}
	}
	return true
}

func issueV0(code string, field string, message string) OperationalDirectorIssueV0 {
	return OperationalDirectorIssueV0{Code: code, Field: field, Message: message}
}
