package orquestarails

import (
	"path/filepath"
	"strings"
)

func WorkspaceRelativePathAllowedV0(value string, allowRoot bool) bool {
	_, ok := NormalizeWorkspaceRelativePathV0(value, allowRoot)
	return ok
}

func NormalizeWorkspaceRelativePathV0(value string, allowRoot bool) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" ||
		strings.Contains(trimmed, "://") ||
		strings.HasPrefix(trimmed, "~") ||
		strings.Contains(trimmed, "$") ||
		strings.Contains(trimmed, "\\") ||
		strings.ContainsAny(trimmed, "\x00\r\n") ||
		filepath.IsAbs(trimmed) ||
		workspacePathHasDrivePrefixV0(trimmed) {
		return "", false
	}
	if trimmed == "." {
		if allowRoot {
			return trimmed, true
		}
		return "", false
	}
	for _, segment := range strings.Split(trimmed, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return "", false
		}
	}
	cleaned := filepath.ToSlash(filepath.Clean(trimmed))
	if cleaned == "." {
		if allowRoot {
			return cleaned, true
		}
		return "", false
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", false
	}
	for _, segment := range strings.Split(cleaned, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return "", false
		}
	}
	return cleaned, true
}

func PathInsideWorkspaceV0(workspace string, target string) bool {
	workspace = strings.TrimSpace(workspace)
	target = strings.TrimSpace(target)
	if workspace == "" || target == "" {
		return false
	}
	workspaceAbs, err := filepath.Abs(filepath.Clean(workspace))
	if err != nil {
		return false
	}
	targetAbs, err := filepath.Abs(filepath.Clean(target))
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(workspaceAbs, targetAbs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return !filepath.IsAbs(rel)
}

func CommandTextContainsDestructiveFilesystemOperationV0(value string) bool {
	if !RailsEnforcedV0() {
		return false
	}
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(value)))
	for _, field := range fields {
		field = strings.Trim(field, "'\"`();")
		switch field {
		case "rm", "rmdir", "unlink", "shred", "del", "remove-item", "removeall":
			return true
		}
	}
	lower := strings.ToLower(value)
	return strings.Contains(lower, "os.removeall(") ||
		strings.Contains(lower, "os.remove(") ||
		strings.Contains(lower, "filesystem.remove(")
}

func workspacePathHasDrivePrefixV0(value string) bool {
	return len(value) >= 2 &&
		((value[0] >= 'a' && value[0] <= 'z') || (value[0] >= 'A' && value[0] <= 'Z')) &&
		value[1] == ':'
}
