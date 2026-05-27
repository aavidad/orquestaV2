package orquestaappcodexstack

import (
	"context"
	"path/filepath"
	"strings"

	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
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
	result := orquestaruntimeworktree.ProjectTreeScanHasFileV0(context.Background(), orquestaruntimeworktree.ProjectTreeScanRequestV0{
		ProjectRoot:    projectDir,
		Target:         target,
		Mode:           orquestaruntimeworktree.ProjectTreeScanModeTargetV0,
		IgnorePrefixes: orquestaruntimeworktree.DefaultWorktreeControlIgnorePrefixesV0(),
	})
	return result.Found
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
	result := orquestaruntimeworktree.ProjectTreeScanHasFileV0(context.Background(), orquestaruntimeworktree.ProjectTreeScanRequestV0{
		ProjectRoot:    projectDir,
		Target:         pattern,
		Mode:           orquestaruntimeworktree.ProjectTreeScanModeGlobV0,
		IgnorePrefixes: orquestaruntimeworktree.DefaultWorktreeControlIgnorePrefixesV0(),
	})
	return result.Found
}

func reviewReworkDirHasFileV0(dir string) bool {
	result := orquestaruntimeworktree.ProjectTreeScanHasFileV0(context.Background(), orquestaruntimeworktree.ProjectTreeScanRequestV0{
		ProjectRoot:    dir,
		Target:         ".",
		Mode:           orquestaruntimeworktree.ProjectTreeScanModeDirV0,
		IgnorePrefixes: orquestaruntimeworktree.DefaultWorktreeControlIgnorePrefixesV0(),
	})
	return result.Found
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

func reviewReworkSkipProjectDirV0(name string) bool {
	if name == ".git" {
		return true
	}
	return orquestaruntimeworktree.IsWorktreeControlPathV0(name)
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
