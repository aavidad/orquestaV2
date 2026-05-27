package orquestaappcodexstack

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func codexStackAgentRuntimeDetailFileKindV0(name string) string {
	switch name {
	case orquestaruntimecodex.CodexAgentPromptFileNameV0:
		return "prompt"
	case orquestaruntimecodex.CodexAgentPacketFileNameV0,
		orquestaruntimecodex.CodexAgentAckFileNameV0,
		orquestaruntimecodex.CodexDirectorDecisionsFileNameV0,
		orquestaruntimecodex.CodexShutdownRequestFileNameV0,
		orquestaruntimecodex.CodexShutdownCheckpointAckFileNameV0,
		orquestaruntimecodex.CodexUsageAccountingFileNameV0:
		return "control_json"
	case orquestaruntimecodex.CodexWrapperFileNameV0:
		return "wrapper"
	case orquestaruntimecodex.CodexStdoutFileNameV0,
		orquestaruntimecodex.CodexStderrFileNameV0,
		orquestaruntimecodex.CodexLastMessageFileNameV0:
		return "log"
	default:
		return "runtime_file"
	}
}

func codexStackAgentRuntimeDetailRedactionLevelV0(name string) string {
	if codexStackAgentRuntimeDetailIsLogFileV0(name) {
		return "redacted_tail"
	}
	return "metadata_extracts"
}

func codexStackAgentRuntimeDetailEnvelopeV0(
	file CodexStackAgentRuntimeDetailFileV0,
	data []byte,
) CodexStackAgentRuntimeDetailFileV0 {
	sum := sha256.Sum256(data)
	file.SHA256 = fmt.Sprintf("%x", sum[:])
	file.ContentRef = "runtime-detail-ref-" + file.SHA256[:20]
	file.Content = string(data)
	if file.Truncated {
		file.ReasonCodes = append(file.ReasonCodes, "runtime_detail_too_large")
	}
	if codexStackAgentRuntimeDetailIsLogFileV0(file.Name) {
		file.Preview = codexStackAgentRuntimeDetailRedactedPreviewV0(string(data))
		file.ReasonCodes = append(file.ReasonCodes, "runtime_detail_redacted")
		return file
	}
	file.Extracts = codexStackAgentRuntimeDetailExtractsV0(file.Name, data)
	file.ReasonCodes = append(file.ReasonCodes, "runtime_detail_redacted")
	return file
}

func codexStackAgentRuntimeDetailExtractsV0(name string, data []byte) map[string]any {
	if name == orquestaruntimecodex.CodexAgentPromptFileNameV0 ||
		name == orquestaruntimecodex.CodexWrapperFileNameV0 {
		return nil
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return map[string]any{"parse_status": "unavailable"}
	}
	out := map[string]any{}
	for _, key := range []string{"schema_version", "request_id", "correlation_id", "ack_ref",
		"task_ref", "status", "checkpoint_ref", "run_ref", "agent_ref"} {
		if value, ok := payload[key]; ok {
			out[key] = value
		}
	}
	if tests, ok := payload["tests"].([]any); ok {
		out["tests_count"] = len(tests)
	}
	if files, ok := payload["files"].([]any); ok {
		out["files_count"] = len(files)
	}
	if decisions, ok := payload["decisions"].([]any); ok {
		out["decisions_count"] = len(decisions)
	}
	return out
}

func codexStackAgentRuntimeDetailRedactedPreviewV0(value string) string {
	if !utf8.ValidString(value) {
		return "[runtime_detail_binary_redacted]"
	}
	replacements := []string{"authorization:", "bearer ", "api_key=", "access_token=",
		"refresh_token=", "client_secret=", "password=", "/home/", "/Users/", `C:\Users\`}
	out := value
	lower := strings.ToLower(out)
	for _, marker := range replacements {
		index := strings.Index(lower, strings.ToLower(marker))
		if index >= 0 {
			out = out[:index] + "[runtime_detail_redacted]"
			break
		}
	}
	if len(out) > 4096 {
		out = out[len(out)-4096:]
	}
	return out
}
