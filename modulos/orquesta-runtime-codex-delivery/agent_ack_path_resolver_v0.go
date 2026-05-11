package orquestaruntimecodexdelivery

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

type AgentScopedCodexReceiptAckPathResolverV0 struct {
	BaseDir        string
	ProjectWorkDir string
	FileName       string
}

func (resolver AgentScopedCodexReceiptAckPathResolverV0) ResolveCodexReceiptAckPathV0(
	_ context.Context,
	request CodexReceiptAckPathRequestV0,
) (CodexReceiptAckPathResolutionV0, error) {
	dir, err := CodexReceiptAgentRuntimeDirV0(resolver.BaseDir, request.RunID, request.AgentRef)
	if err != nil {
		return CodexReceiptAckPathResolutionV0{}, err
	}
	fileName := codexReceiptAckFileNameV0(resolver.FileName)
	if fileName == "" {
		return CodexReceiptAckPathResolutionV0{}, fmt.Errorf("codex_receipt_ack_path: file_name invalido")
	}
	return CodexReceiptAckPathResolutionV0{
		AckPath:        filepath.Join(dir, fileName),
		ProjectWorkDir: strings.TrimSpace(resolver.ProjectWorkDir),
	}, nil
}

func CodexReceiptAgentRuntimeDirV0(baseDir string, runID string, agentRef string) (string, error) {
	baseDir = strings.TrimSpace(baseDir)
	if baseDir == "" || !filepath.IsAbs(baseDir) {
		return "", fmt.Errorf("codex_receipt_agent_runtime_dir: base_dir invalido")
	}
	agentPart := codexReceiptSafePathPartV0(agentRef)
	if agentPart == "" {
		return "", fmt.Errorf("codex_receipt_agent_runtime_dir: agent_ref invalido")
	}
	runPart := codexReceiptSafePathPartV0(runID)
	if runPart == "" {
		return filepath.Join(baseDir, agentPart), nil
	}
	return filepath.Join(baseDir, runPart, agentPart), nil
}

func codexReceiptAckFileNameV0(fileName string) string {
	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		fileName = orquestaruntimecodex.CodexAgentAckFileNameV0
	}
	if filepath.Base(fileName) != fileName || strings.Contains(fileName, `\`) {
		return ""
	}
	return fileName
}

func codexReceiptSafePathPartV0(value string) string {
	value = strings.TrimSpace(value)
	var builder strings.Builder
	for i := 0; i < len(value); i++ {
		ch := value[i]
		switch {
		case ch >= 'a' && ch <= 'z':
			builder.WriteByte(ch)
		case ch >= 'A' && ch <= 'Z':
			builder.WriteByte(ch + ('a' - 'A'))
		case ch >= '0' && ch <= '9':
			builder.WriteByte(ch)
		case ch == '-' || ch == '_':
			builder.WriteByte(ch)
		}
	}
	return builder.String()
}
