package orquestaappcodexstack

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func codexStackAgentRuntimeDetailDescriptorAgentRefV0(
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) string {
	if strings.TrimSpace(descriptor.AgentRef) != "" {
		return strings.TrimSpace(descriptor.AgentRef)
	}
	if strings.TrimSpace(descriptor.Spec.RequestID) != "" {
		return strings.TrimSpace(descriptor.Spec.RequestID)
	}
	return strings.TrimSpace(descriptor.Spec.AgentPacket.RequestID)
}

func codexStackAgentRuntimeDetailRuntimeDirsV0(
	runtimeRoot string,
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
	agentRef string,
) []string {
	candidates := []string{}
	if strings.TrimSpace(descriptor.AckPath) != "" {
		candidates = append(candidates, filepath.Dir(descriptor.AckPath))
	}
	runtimeRoot = strings.TrimSpace(runtimeRoot)
	if runtimeRoot != "" {
		runRef := codexStackOperationalClosureSafeRefV0(descriptor.RunID)
		safeAgentRef := codexStackOperationalClosureSafeRefV0(agentRef)
		candidates = append(candidates,
			filepath.Join(runtimeRoot, runRef, safeAgentRef),
			filepath.Join(runtimeRoot, descriptor.RunID, agentRef),
		)
		stateDir := filepath.Join(filepath.Dir(runtimeRoot), "state")
		pattern := filepath.Join(stateDir, "revision", "*", "runtime_archived", runRef, safeAgentRef)
		if matches, err := filepath.Glob(pattern); err == nil {
			candidates = append(candidates, matches...)
		}
	}
	return codexStackAgentRuntimeDetailCompactPathsV0(candidates)
}

func codexStackAgentRuntimeDetailFilesV0(
	dirs []string,
	includeLogs bool,
	maxBytes int64,
) []CodexStackAgentRuntimeDetailFileV0 {
	names := []string{
		orquestaruntimecodex.CodexAgentPacketFileNameV0,
		orquestaruntimecodex.CodexAgentPromptFileNameV0,
		orquestaruntimecodex.CodexAgentAckFileNameV0,
		orquestaruntimecodex.CodexDirectorDecisionsFileNameV0,
		orquestaruntimecodex.CodexShutdownRequestFileNameV0,
		orquestaruntimecodex.CodexShutdownCheckpointAckFileNameV0,
		orquestaruntimecodex.CodexLastMessageFileNameV0,
		orquestaruntimecodex.CodexUsageAccountingFileNameV0,
		orquestaruntimecodex.CodexWrapperFileNameV0,
	}
	if includeLogs {
		names = append(names,
			orquestaruntimecodex.CodexStdoutFileNameV0,
			orquestaruntimecodex.CodexStderrFileNameV0,
		)
	}
	out := make([]CodexStackAgentRuntimeDetailFileV0, 0, len(names))
	for _, name := range names {
		out = append(out, codexStackAgentRuntimeDetailReadFileV0(dirs, name, maxBytes))
	}
	return out
}

func codexStackAgentRuntimeDetailReadFileV0(
	dirs []string,
	name string,
	maxBytes int64,
) CodexStackAgentRuntimeDetailFileV0 {
	for _, dir := range dirs {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		file := CodexStackAgentRuntimeDetailFileV0{Name: name, Path: path, Exists: err == nil}
		if err != nil {
			file.Error = err.Error()
			return file
		}
		file.SizeBytes = info.Size()
		data, err := os.ReadFile(path)
		if err != nil {
			file.Error = err.Error()
			return file
		}
		if maxBytes > 0 && int64(len(data)) > maxBytes {
			file.Truncated = true
			data = data[:maxBytes]
		}
		file.Content = string(data)
		return file
	}
	return CodexStackAgentRuntimeDetailFileV0{Name: name, Exists: false}
}

func codexStackAgentRuntimeDetailContentByNameV0(
	files []CodexStackAgentRuntimeDetailFileV0,
	name string,
) string {
	for _, file := range files {
		if file.Name == name {
			return file.Content
		}
	}
	return ""
}

func codexStackAgentRuntimeDetailJSONByNameV0(
	files []CodexStackAgentRuntimeDetailFileV0,
	name string,
) map[string]any {
	content := codexStackAgentRuntimeDetailContentByNameV0(files, name)
	if strings.TrimSpace(content) == "" {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		return nil
	}
	return out
}

func codexStackAgentRuntimeDetailSkillHintsV0(
	packet orquestaruntime.AgentStartPacketV0,
	prompt string,
) CodexStackAgentRuntimeDetailSkillHintsV0 {
	lowerPrompt := strings.ToLower(prompt)
	return CodexStackAgentRuntimeDetailSkillHintsV0{
		CavemanRequested: strings.Contains(lowerPrompt, "caveman"),
		CompactProtocol:  strings.Contains(lowerPrompt, "compact") || strings.Contains(lowerPrompt, "tokens"),
		Policies:         append([]string(nil), packet.Policies...),
		CapacityLevel:    packet.CapacityLevel,
		Phase:            packet.Phase,
		MaxChildAgents:   packet.Task.MaxChildAgents,
	}
}

func codexStackAgentRuntimeDetailMaxBytesV0(value int64) int64 {
	if value <= 0 {
		return 5 * 1024 * 1024
	}
	if value > 20*1024*1024 {
		return 20 * 1024 * 1024
	}
	return value
}

func codexStackAgentRuntimeDetailDirExistsV0(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func codexStackAgentRuntimeDetailCompactPathsV0(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		value = filepath.Clean(value)
		if seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
