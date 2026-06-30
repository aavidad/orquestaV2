package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

const codexWaveCodebaseMemoryMCPNameV0 = "codebase-memory-mcp"
const codexWaveAgentToolingLedgerPathV0 = "log/orquesta-agent-tooling.env"
const codexWaveCodebaseMemoryMCPGuardStartV0 = "<!-- orquesta-codebase-memory-mcp-guard:start -->"
const codexWaveCodebaseMemoryMCPGuardEndV0 = "<!-- orquesta-codebase-memory-mcp-guard:end -->"

func codexWaveCopyCodeHomeV0(
	sourceCodeHome string,
	agentRuntimeDir string,
	policy codexWaveCredentialProjectionPolicyV0,
) (string, string, *codexWaveCredentialProjectionReceiptV0, error) {
	source := filepath.Clean(sourceCodeHome)
	if source == "" || !filepath.IsAbs(source) {
		return "", "", nil, errors.New("source_code_home_invalid")
	}
	if info, err := os.Lstat(source); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", "", nil, errors.New("source_code_home_unavailable")
	}
	homeDir := filepath.Join(agentRuntimeDir, "home")
	codeHomeDir := filepath.Join(homeDir, ".codex")
	if err := os.MkdirAll(codeHomeDir, 0o700); err != nil {
		return "", "", nil, err
	}
	receipt := codexWaveProjectionReceiptV0(policy, agentRuntimeDir)
	for _, item := range codexWaveCredentialProjectionItemsV0(policy) {
		exists, err := codexWaveProjectionSourceExistsV0(filepath.Join(source, item.Name), item.Directory)
		if err != nil {
			return "", "", nil, err
		}
		if !item.Allowed {
			if exists {
				codexWaveProjectionRecordOmissionV0(&receipt, item.Category, "category_not_allowed")
			}
			continue
		}
		if !exists {
			if policy.Strict && item.RequiredInStrict {
				receipt.RequiredMissing = codexWaveProjectionRecordCategoryV0(receipt.RequiredMissing, item.Category)
			}
			continue
		}
		if item.Directory {
			err = codexWaveCopyDirIfExistsV0(filepath.Join(source, item.Name), filepath.Join(codeHomeDir, item.Name), item.Category, &receipt, policy)
		} else {
			err = codexWaveCopyFileIfExistsV0(filepath.Join(source, item.Name), filepath.Join(codeHomeDir, item.Name), item.Category, &receipt, policy)
		}
		if err != nil {
			return "", "", nil, err
		}
		if codexWaveProjectionCategoryCopiedV0(receipt, item.Category) {
			receipt.Categories = codexWaveProjectionRecordCategoryV0(receipt.Categories, item.Category)
		}
		if policy.Strict && item.RequiredInStrict && !codexWaveProjectionCategoryCopiedV0(receipt, item.Category) {
			receipt.RequiredMissing = codexWaveProjectionRecordCategoryV0(receipt.RequiredMissing, item.Category)
		}
	}
	if err := codexWaveProtectProjectedCodebaseMemoryMCPV0(source, codeHomeDir, &receipt); err != nil {
		return "", "", nil, err
	}
	if len(receipt.RequiredMissing) > 0 {
		return "", "", &receipt, fmt.Errorf("credential_projection_missing_required:%s", receipt.RequiredMissing[0])
	}
	return homeDir, codeHomeDir, &receipt, nil
}

func codexWaveCopyFileIfExistsV0(
	source string,
	target string,
	category string,
	receipt *codexWaveCredentialProjectionReceiptV0,
	policy codexWaveCredentialProjectionPolicyV0,
) error {
	info, err := os.Lstat(source)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !codexWaveProjectionFileAllowedV0(info, category, receipt, policy) {
		return nil
	}
	data, err := codexWaveReadRegularNoFollowV0(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(target, data, 0o600); err != nil {
		return err
	}
	codexWaveProjectionRecordCopiedV0(receipt, category, info.Size())
	return nil
}

func codexWaveCopyDirIfExistsV0(
	source string,
	target string,
	category string,
	receipt *codexWaveCredentialProjectionReceiptV0,
	policy codexWaveCredentialProjectionPolicyV0,
) error {
	info, err := os.Lstat(source)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		codexWaveProjectionRecordOmissionV0(receipt, category, "symlink")
		return nil
	}
	if !info.IsDir() {
		codexWaveProjectionRecordOmissionV0(receipt, category, "not_directory")
		return nil
	}
	if err := os.MkdirAll(target, 0o700); err != nil {
		return err
	}
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		targetPath := filepath.Join(target, rel)
		if entry.IsDir() {
			return os.MkdirAll(targetPath, 0o700)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !codexWaveProjectionFileAllowedV0(info, category, receipt, policy) {
			return nil
		}
		data, err := codexWaveReadRegularNoFollowV0(path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(targetPath, data, 0o600); err != nil {
			return err
		}
		codexWaveProjectionRecordCopiedV0(receipt, category, info.Size())
		return nil
	})
}

func codexWaveProjectionFileAllowedV0(
	info os.FileInfo,
	category string,
	receipt *codexWaveCredentialProjectionReceiptV0,
	policy codexWaveCredentialProjectionPolicyV0,
) bool {
	mode := info.Mode()
	switch {
	case mode&os.ModeSymlink != 0:
		codexWaveProjectionRecordOmissionV0(receipt, category, "symlink")
	case !mode.IsRegular():
		codexWaveProjectionRecordOmissionV0(receipt, category, "non_regular")
	case codexWaveProjectionHardlinkedV0(info):
		codexWaveProjectionRecordOmissionV0(receipt, category, "hardlink")
	case policy.MaxFiles > 0 && receipt != nil && receipt.FilesCopied >= policy.MaxFiles:
		codexWaveProjectionRecordOmissionV0(receipt, category, "max_files_exceeded")
	case policy.MaxFileBytes > 0 && info.Size() > policy.MaxFileBytes:
		codexWaveProjectionRecordOmissionV0(receipt, category, "max_file_bytes_exceeded")
	case policy.MaxTotalBytes > 0 && receipt != nil && receipt.BytesCopied+info.Size() > policy.MaxTotalBytes:
		codexWaveProjectionRecordOmissionV0(receipt, category, "max_total_bytes_exceeded")
	default:
		return true
	}
	return false
}

func codexWaveProjectionRecordCopiedV0(receipt *codexWaveCredentialProjectionReceiptV0, category string, size int64) {
	if receipt == nil {
		return
	}
	receipt.Categories = codexWaveProjectionRecordCategoryV0(receipt.Categories, category)
	receipt.FilesCopied++
	receipt.BytesCopied += size
}

func codexWaveProjectionCategoryCopiedV0(receipt codexWaveCredentialProjectionReceiptV0, category string) bool {
	for _, value := range receipt.Categories {
		if value == category {
			return true
		}
	}
	return false
}

func codexWaveProjectionHardlinkedV0(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Nlink > 1
}

func codexWaveReadRegularNoFollowV0(path string) ([]byte, error) {
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), filepath.Base(path))
	defer file.Close()
	return io.ReadAll(file)
}

func codexWaveProtectProjectedCodebaseMemoryMCPV0(
	sourceCodeHome string,
	codeHomeDir string,
	receipt *codexWaveCredentialProjectionReceiptV0,
) error {
	if err := codexWaveWriteCodebaseMemoryMCPGuardV0(codeHomeDir); err != nil {
		return err
	}
	if codexWaveCodebaseMemoryMCPOptInV0(sourceCodeHome) {
		return nil
	}
	changed, err := codexWaveDisableCodebaseMemoryMCPConfigV0(filepath.Join(codeHomeDir, "config.toml"))
	if err != nil {
		return err
	}
	if changed {
		codexWaveProjectionRecordOmissionV0(receipt, "config", "codebase_memory_mcp_disabled_by_default")
	}
	return nil
}

func codexWaveCodebaseMemoryMCPOptInV0(sourceCodeHome string) bool {
	raw, err := codexWaveReadRegularNoFollowV0(filepath.Join(sourceCodeHome, codexWaveAgentToolingLedgerPathV0))
	if err != nil {
		return false
	}
	values := map[string]string{}
	for _, line := range strings.Split(string(raw), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		values[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return strings.TrimSpace(values["tool"]) == codexWaveCodebaseMemoryMCPNameV0 &&
		codexWaveTruthyLedgerValueV0(values["enabled"])
}

func codexWaveTruthyLedgerValueV0(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func codexWaveWriteCodebaseMemoryMCPGuardV0(codeHomeDir string) error {
	if strings.TrimSpace(codeHomeDir) == "" {
		return nil
	}
	path := filepath.Join(codeHomeDir, "AGENTS.md")
	raw, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	text := string(raw)
	if strings.Contains(text, codexWaveCodebaseMemoryMCPGuardStartV0) {
		return nil
	}
	if strings.TrimSpace(text) != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	if strings.TrimSpace(text) != "" {
		text += "\n"
	}
	text += codexWaveCodebaseMemoryMCPGuardBlockV0()
	return os.WriteFile(path, []byte(text), 0o600)
}

func codexWaveCodebaseMemoryMCPGuardBlockV0() string {
	return strings.Join([]string{
		codexWaveCodebaseMemoryMCPGuardStartV0,
		"# Orquesta Codebase MCP Guard",
		"",
		"Agentes lanzados por Orquesta: no arranques ni uses codebase-memory-mcp, MCP de codebase o indexadores de grafo salvo opt-in central durable de Orquesta.",
		"Opt-in central valido: log/orquesta-agent-tooling.env con tool=codebase-memory-mcp y enabled=1 presente en el CODEX_HOME fuente durante la proyeccion.",
		"Para strings exactos, Markdown, configs, incidencias y lectura local acotada usa rg/sed/find.",
		codexWaveCodebaseMemoryMCPGuardEndV0,
		"",
	}, "\n")
}

func codexWaveDisableCodebaseMemoryMCPConfigV0(path string) (bool, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	filtered, changed := codexWaveFilterCodebaseMemoryMCPTOMLV0(string(raw))
	if !changed {
		return false, nil
	}
	if strings.TrimSpace(filtered) != "" && !strings.HasSuffix(filtered, "\n") {
		filtered += "\n"
	}
	return true, os.WriteFile(path, []byte(filtered), 0o600)
}

type codexWaveTOMLBlockV0 struct {
	header string
	lines  []string
}

func codexWaveFilterCodebaseMemoryMCPTOMLV0(raw string) (string, bool) {
	blocks := codexWaveTOMLBlocksV0(raw)
	removedServers := map[string]bool{}
	for _, block := range blocks {
		serverName := codexWaveMCPServerNameFromHeaderV0(block.header)
		if serverName == "" {
			continue
		}
		if codexWaveMCPServerNameIsCodebaseMemoryV0(serverName) ||
			codexWaveLinesContainCodebaseMemoryMCPV0(block.lines) {
			removedServers[serverName] = true
		}
	}
	var out strings.Builder
	changed := false
	for _, block := range blocks {
		serverName := codexWaveMCPServerNameFromHeaderV0(block.header)
		if serverName != "" && removedServers[serverName] {
			changed = true
			continue
		}
		lines := block.lines
		if codexWaveIsMCPServersParentHeaderV0(block.header) {
			filtered := lines[:0]
			for _, line := range lines {
				if codexWaveLineContainsCodebaseMemoryMCPV0(line) {
					changed = true
					continue
				}
				filtered = append(filtered, line)
			}
			lines = filtered
		}
		for _, line := range lines {
			out.WriteString(line)
		}
	}
	return out.String(), changed
}

func codexWaveTOMLBlocksV0(raw string) []codexWaveTOMLBlockV0 {
	lines := codexWaveSplitLinesKeepEndV0(raw)
	blocks := []codexWaveTOMLBlockV0{{}}
	for _, line := range lines {
		if header := codexWaveTOMLHeaderV0(line); header != "" {
			blocks = append(blocks, codexWaveTOMLBlockV0{header: header, lines: []string{line}})
			continue
		}
		blocks[len(blocks)-1].lines = append(blocks[len(blocks)-1].lines, line)
	}
	return blocks
}

func codexWaveSplitLinesKeepEndV0(raw string) []string {
	if raw == "" {
		return nil
	}
	lines := []string{}
	for len(raw) > 0 {
		index := strings.IndexByte(raw, '\n')
		if index < 0 {
			lines = append(lines, raw)
			break
		}
		lines = append(lines, raw[:index+1])
		raw = raw[index+1:]
	}
	return lines
}

func codexWaveTOMLHeaderV0(line string) string {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "#") || !strings.HasPrefix(trimmed, "[") {
		return ""
	}
	if strings.HasPrefix(trimmed, "[[") {
		end := strings.Index(trimmed, "]]")
		if end < 0 {
			return ""
		}
		return strings.TrimSpace(trimmed[2:end])
	}
	end := strings.Index(trimmed, "]")
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(trimmed[1:end])
}

func codexWaveMCPServerNameFromHeaderV0(header string) string {
	normalized := codexWaveNormalizeTOMLHeaderV0(header)
	const prefix = "mcp-servers."
	if strings.HasPrefix(normalized, prefix) {
		rest := strings.TrimPrefix(normalized, prefix)
		name, _, _ := strings.Cut(rest, ".")
		return strings.TrimSpace(name)
	}
	return ""
}

func codexWaveIsMCPServersParentHeaderV0(header string) bool {
	return codexWaveNormalizeTOMLHeaderV0(header) == "mcp-servers"
}

func codexWaveNormalizeTOMLHeaderV0(header string) string {
	normalized := strings.ToLower(strings.TrimSpace(header))
	normalized = strings.ReplaceAll(normalized, "_", "-")
	normalized = strings.ReplaceAll(normalized, "\"", "")
	normalized = strings.ReplaceAll(normalized, "'", "")
	normalized = strings.ReplaceAll(normalized, " ", "")
	return normalized
}

func codexWaveMCPServerNameIsCodebaseMemoryV0(name string) bool {
	name = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), "_", "-"))
	return name == "codebase-memory-mcp" || name == "codebase-memory"
}

func codexWaveLinesContainCodebaseMemoryMCPV0(lines []string) bool {
	for _, line := range lines {
		if codexWaveLineContainsCodebaseMemoryMCPV0(line) {
			return true
		}
	}
	return false
}

func codexWaveLineContainsCodebaseMemoryMCPV0(line string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(line, "_", "-"))
	return strings.Contains(normalized, codexWaveCodebaseMemoryMCPNameV0)
}
