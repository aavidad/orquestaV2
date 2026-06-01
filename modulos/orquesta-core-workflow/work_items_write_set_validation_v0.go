package orquestacoreworkflow

import "strings"

func normalizeWorkflowTaskWriteSetV0(paths []string) []string {
	if paths == nil {
		return nil
	}
	normalized := make([]string, len(paths))
	for index, repoPath := range paths {
		normalized[index] = normalizeWorkflowTaskWriteSetPathV0(repoPath)
	}
	return normalized
}

func normalizeWorkflowTaskWriteSetPathV0(repoPath string) string {
	path := strings.TrimSpace(repoPath)
	for {
		next := strings.TrimSuffix(path, "/")
		switch {
		case next != path:
			path = next
		case strings.HasSuffix(path, "/**"):
			path = strings.TrimSuffix(path, "/**")
		case strings.HasSuffix(path, "/*"):
			path = strings.TrimSuffix(path, "/*")
		default:
			return path
		}
	}
}

func validateWorkflowTaskWriteSetV0(paths []string) error {
	if err := validateRequiredWorkflowTaskStringsV0(paths, "write_set"); err != nil {
		return err
	}
	for _, repoPath := range paths {
		if !workflowTaskWriteSetPathIsSafeV0(repoPath) {
			return workflowTaskErrorV0(ErrWorkflowTaskInvalidaV0, "write_set")
		}
		if workflowTaskWriteSetPathHasForbiddenDetailV0(repoPath) {
			return workflowTaskErrorV0(ErrDetalleProhibidoV0, "write_set")
		}
	}
	return nil
}

func workflowTaskWriteSetPathIsSafeV0(repoPath string) bool {
	if strings.Contains(repoPath, "\x00") || strings.Contains(repoPath, "\\") {
		return false
	}
	if strings.HasPrefix(repoPath, "/") || strings.HasPrefix(repoPath, "~") {
		return false
	}
	if strings.Contains(repoPath, "://") || strings.Contains(repoPath, ":") || strings.Contains(repoPath, "$") {
		return false
	}
	for _, segment := range strings.Split(repoPath, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

func workflowTaskWriteSetPathHasForbiddenDetailV0(repoPath string) bool {
	return textContainsForbiddenOperationalSensitiveDetailV0(repoPath)
}
