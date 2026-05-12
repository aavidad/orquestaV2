package orquestaappcodexstack

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func codexStackRealSmokeDrainCycleMaxExternalWaitsV0(maxExternalWaits int) int {
	if maxExternalWaits <= 0 {
		return 1
	}
	if maxExternalWaits > 30 {
		return 30
	}
	return maxExternalWaits
}

func codexStackRealSmokeDrainFingerprintV0(
	t *testing.T,
	stores codexStackRealSmokeStoresV0,
	stack StackV0,
	runRef string,
) string {
	t.Helper()
	return codexStackRealSmokeDrainFingerprintFromRunV0(
		mustLoadCodexStackRunForTestV0(t, stack, runRef),
		codexStackRealSmokeDescriptorsV0(t, stores.ReceiptStore),
	)
}

func codexStackRealSmokeDrainFingerprintFromRunV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) string {
	return strings.Join([]string{
		fmt.Sprintf("seq=%d", run.LastSequence),
		fmt.Sprintf("tasks=%d", len(compactStringsV0(run.Tasks))),
		fmt.Sprintf("delivered=%d", len(compactStringsV0(run.DeliveredTasks))),
		fmt.Sprintf("closed=%d", len(compactStringsV0(run.ClosedTasks))),
		fmt.Sprintf("deliveries=%d", len(compactStringsV0(run.Deliveries))),
		fmt.Sprintf("artifacts=%d", len(compactStringsV0(run.PhaseArtifacts))),
		fmt.Sprintf("started=%d", len(compactStringsV0(run.StartedAgents))),
		fmt.Sprintf("assessments=%d", len(compactStringsV0(run.AgentAssessments))),
		fmt.Sprintf("descriptors=%d", len(descriptors)),
		fmt.Sprintf("acks=%d", codexStackRealSmokeCompletedAckCountV0(descriptors)),
	}, ";")
}

func codexStackRealSmokeProgrammingCausalIssueV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) string {
	pending := codexStackRealSmokePendingProgrammingACKsWithArtifactsV0(run, descriptors)
	if len(pending) == 0 {
		return ""
	}
	if projectDir := codexStackRealSmokeProjectWorkDirFromDescriptorsV0(descriptors); projectDir != "" {
		if err := codexStackRealSmokeCheckGoAppCompilesV0(projectDir); err == nil {
			return "project_compiles_but_ack_missing " + strings.Join(pending, ",")
		}
	}
	if !drainRunHasPendingExternalAgentsV0(run) {
		return "no_external_agents_but_ack_missing " + strings.Join(pending, ",")
	}
	return ""
}

func codexStackRealSmokePendingProgrammingACKsWithArtifactsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) []string {
	pending := []string{}
	for _, descriptor := range codexStackRealSmokeProgrammingReceiptDescriptorsV0(descriptors) {
		if codexStackRealSmokeContainsProjectionPartV0(run.Deliveries, descriptor.Spec.AgentPacket.DeliveryRefs.AckRef) {
			continue
		}
		if _, ok := codexStackRealSmokeCompletedAckV0(descriptor); ok {
			continue
		}
		if !codexStackRealSmokeDescriptorWriteSetCompleteV0(descriptor) {
			continue
		}
		pending = append(pending, codexStackRealSmokePendingACKSummaryV0(descriptor))
	}
	return compactStringsV0(pending)
}

func codexStackRealSmokePendingACKSummaryV0(
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) string {
	_, issues := orquestaruntimecodex.ReadAndValidateCodexAgentAckFileV0(
		descriptor.AckPath,
		descriptor.Spec,
	)
	issue := "ack_missing"
	if len(issues) > 0 {
		issue = string(issues[0].Code) + ":" + strings.TrimSpace(issues[0].Field)
	}
	return fmt.Sprintf(
		"agent=%s task=%s ack=%s issue=%s",
		descriptor.AgentRef,
		descriptor.Spec.AgentPacket.Task.TaskRef,
		descriptor.Spec.AgentPacket.DeliveryRefs.AckRef,
		issue,
	)
}

func codexStackRealSmokeCompletedAckCountV0(
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) int {
	count := 0
	for _, descriptor := range descriptors {
		if _, ok := codexStackRealSmokeCompletedAckV0(descriptor); ok {
			count++
		}
	}
	return count
}

func codexStackRealSmokeProjectWorkDirFromDescriptorsV0(
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) string {
	for _, descriptor := range descriptors {
		if dir := strings.TrimSpace(descriptor.ProjectWorkDir); dir != "" {
			return dir
		}
	}
	return ""
}

func codexStackRealSmokeDescriptorWriteSetCompleteV0(
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) bool {
	projectDir := strings.TrimSpace(descriptor.ProjectWorkDir)
	if projectDir == "" {
		return false
	}
	writeSet := compactStringsV0(descriptor.Spec.AgentPacket.Task.WriteSet)
	if len(writeSet) == 0 {
		return false
	}
	for _, target := range writeSet {
		if !codexStackRealSmokeProjectTargetExistsV0(projectDir, target) {
			return false
		}
	}
	return true
}

func codexStackRealSmokeProjectTargetExistsV0(projectDir string, target string) bool {
	clean := filepath.ToSlash(filepath.Clean(strings.TrimSpace(target)))
	if clean == "." || clean == "" || strings.HasPrefix(clean, "../") || filepath.IsAbs(clean) {
		return false
	}
	if codexStackRealSmokeHasGlobV0(clean) {
		return len(codexStackRealSmokeGlobMatchesNoFatalV0(projectDir, clean)) > 0
	}
	fullPath := filepath.Join(projectDir, filepath.FromSlash(clean))
	info, err := os.Stat(fullPath)
	if err != nil {
		return false
	}
	if !info.IsDir() {
		return info.Size() > 0
	}
	return codexStackRealSmokeDirHasFileV0(fullPath)
}

func codexStackRealSmokeGlobMatchesNoFatalV0(projectDir string, pattern string) []string {
	re, err := codexStackRealSmokeGlobRegexpNoFatalV0(pattern)
	if err != nil {
		return nil
	}
	matches := []string{}
	_ = filepath.Walk(projectDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info == nil {
			return nil
		}
		if info.IsDir() && info.Name() == ".orquesta-codex-runtime" {
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(projectDir, path)
		if err != nil || rel == "." {
			return nil
		}
		if re.MatchString(filepath.ToSlash(rel)) && info.Size() > 0 {
			matches = append(matches, path)
		}
		return nil
	})
	return matches
}

func codexStackRealSmokeGlobRegexpNoFatalV0(pattern string) (*regexp.Regexp, error) {
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

func codexStackRealSmokeDirHasFileV0(dir string) bool {
	found := false
	_ = filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry == nil || entry.IsDir() {
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
