package orquestarails

import "strings"

type ArchitectureImportPolicyV0 struct {
	ExactImports    []string
	ImportPrefixes  []string
	ImportFragments []string
}

func ArchitectureImportForbiddenV0(path string, policy ArchitectureImportPolicyV0) bool {
	for _, exact := range policy.ExactImports {
		exact = strings.TrimSpace(exact)
		if exact != "" && (path == exact || strings.HasPrefix(path, exact+"/")) {
			return true
		}
	}
	for _, prefix := range policy.ImportPrefixes {
		prefix = strings.TrimSpace(prefix)
		if prefix != "" && strings.HasPrefix(path, prefix) {
			return true
		}
	}
	for _, fragment := range policy.ImportFragments {
		fragment = strings.TrimSpace(fragment)
		if fragment != "" && strings.Contains(path, fragment) {
			return true
		}
	}
	return false
}

func ArchitectureSourceLiteralForbiddenV0(boundary string, value string) bool {
	if textContainsOperationalSensitiveDetailIgnoringEnvV0(value) {
		return true
	}
	lower := strings.ToLower(strings.ReplaceAll(value, `\/`, "/"))
	for _, fragment := range OperationalRawDetailFragmentsV0 {
		if ContainsFragmentWithBoundaryV0(lower, fragment) {
			return true
		}
	}
	return false
}
