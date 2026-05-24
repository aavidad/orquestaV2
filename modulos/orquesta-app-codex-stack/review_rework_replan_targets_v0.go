package orquestaappcodexstack

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func reviewReworkMissingWriteSetTargetsV0(
	deliveryRef string,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) []string {
	deliveryRef = strings.TrimSpace(deliveryRef)
	missing := make([]string, 0)
	for _, descriptor := range descriptors {
		if strings.TrimSpace(descriptor.Spec.AgentPacket.DeliveryRefs.AckRef) != deliveryRef {
			continue
		}
		projectDir := strings.TrimSpace(descriptor.ProjectWorkDir)
		if projectDir == "" {
			continue
		}
		for _, target := range compactStringsV0(descriptor.Spec.AgentPacket.Task.WriteSet) {
			if reviewReworkProjectTargetExistsV0(projectDir, target) {
				continue
			}
			missing = append(missing, target)
		}
	}
	return compactStringsV0(missing)
}

func reviewReworkProjectTargetExistsV0(projectDir string, rawTarget string) bool {
	target, ok := reviewReworkRelTargetV0(rawTarget)
	if !ok {
		return false
	}
	for _, alias := range reviewReworkTargetAliasesV0(target) {
		if reviewReworkProjectTargetExistsV0(projectDir, alias) {
			return true
		}
	}
	if reviewReworkTargetHasGlobV0(target) {
		return reviewReworkProjectGlobHasFileV0(projectDir, target)
	}
	path := filepath.Join(projectDir, filepath.FromSlash(target))
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if !info.IsDir() {
		return info.Size() > 0
	}
	return reviewReworkDirHasFileV0(path)
}

func reviewReworkTargetAliasesV0(target string) []string {
	switch strings.Trim(strings.ToLower(filepath.ToSlash(target)), "/") {
	case "web":
		return []string{"internal/webadmin", "internal/web", "frontend", "ui"}
	case "api":
		return []string{"internal/api", "cmd/api", "cmd/server"}
	case "docs":
		return []string{"README.md"}
	default:
		return nil
	}
}

func reviewReworkProjectGlobHasFileV0(projectDir string, pattern string) bool {
	re, err := reviewReworkGlobRegexpV0(pattern)
	if err != nil {
		return false
	}
	found := false
	_ = filepath.WalkDir(projectDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry == nil || found {
			return nil
		}
		if entry.IsDir() {
			if reviewReworkSkipProjectDirV0(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil || info.Size() == 0 {
			return nil
		}
		rel, err := filepath.Rel(projectDir, path)
		if err == nil && re.MatchString(filepath.ToSlash(rel)) {
			found = true
		}
		return nil
	})
	return found
}

func reviewReworkDirHasFileV0(dir string) bool {
	found := false
	_ = filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry == nil || found {
			return nil
		}
		if entry.IsDir() {
			if path != dir && reviewReworkSkipProjectDirV0(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := entry.Info()
		if err == nil && info.Size() > 0 {
			found = true
		}
		return nil
	})
	return found
}

func reviewReworkRelTargetV0(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" ||
		strings.Contains(value, "://") ||
		strings.HasPrefix(value, "~") ||
		strings.Contains(value, "$HOME") ||
		strings.ContainsAny(value, "\x00\r\n") ||
		filepath.IsAbs(value) {
		return "", false
	}
	cleaned := filepath.ToSlash(filepath.Clean(value))
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", false
	}
	return cleaned, true
}

func reviewReworkTargetHasGlobV0(value string) bool {
	return strings.ContainsAny(value, "*?[")
}

func reviewReworkGlobRegexpV0(pattern string) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		if strings.HasPrefix(pattern[i:], "**") {
			b.WriteString(".*")
			i++
			continue
		}
		switch pattern[i] {
		case '*':
			b.WriteString("[^/]*")
		case '?':
			b.WriteString("[^/]")
		default:
			b.WriteString(regexp.QuoteMeta(string(pattern[i])))
		}
	}
	b.WriteString("$")
	return regexp.Compile(b.String())
}

func reviewReworkSkipProjectDirV0(name string) bool {
	switch name {
	case ".git", ".orquesta-runtime", ".orquesta-codex-runtime":
		return true
	default:
		return false
	}
}

func reviewReworkReplanSafeOpaqueTargetV0(value string) string {
	value = strings.TrimSpace(filepath.ToSlash(value))
	replacer := strings.NewReplacer(
		"\\", "-",
		"/", "-",
		" ", "-",
		"*", "star",
		"?", "q",
		"[", "-",
		"]", "-",
	)
	value = strings.Trim(replacer.Replace(value), "-")
	if value == "" {
		return "target"
	}
	if len(value) > 120 {
		return value[:120]
	}
	return value
}
