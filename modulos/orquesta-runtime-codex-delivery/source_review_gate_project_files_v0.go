package orquestaruntimecodexdelivery

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

type CodexReviewGateProjectFileEvidenceV0 struct{}

var _ CodexReviewGateFileEvidenceResultProviderPortV0 = CodexReviewGateProjectFileEvidenceV0{}
var _ CodexReviewGateFileEvidenceProviderPortV0 = CodexReviewGateProjectFileEvidenceV0{}

func (provider CodexReviewGateProjectFileEvidenceV0) BuildCodexReviewGateFileEvidenceV0(
	ctx context.Context,
	descriptor CodexReceiptDescriptorV0,
	ack orquestaruntimecodex.CodexAgentAckV0,
) (CodexReviewGateFileEvidenceV0, error) {
	root, ok := codexReviewGateProjectRootV0(descriptor.ProjectWorkDir)
	if !ok {
		return CodexReviewGateFileEvidenceV0{
			Issues: []orquestacionnucleoapp.AutoprogrammingReviewGateIssueV0{
				codexReviewGateFileIssueV0("project_workdir_invalid", "project_workdir"),
			},
		}, nil
	}
	return provider.fileEvidenceFromRootV0(ctx, root, ack.Files), nil
}

func (provider CodexReviewGateProjectFileEvidenceV0) BuildCodexReviewGateFilesV0(
	ctx context.Context,
	descriptor CodexReceiptDescriptorV0,
	ack orquestaruntimecodex.CodexAgentAckV0,
) ([]orquestacionnucleoapp.AutoprogrammingReviewGateFileV0, error) {
	evidence, err := provider.BuildCodexReviewGateFileEvidenceV0(ctx, descriptor, ack)
	return evidence.Files, err
}

func (provider CodexReviewGateProjectFileEvidenceV0) fileEvidenceFromRootV0(
	ctx context.Context,
	root string,
	files []string,
) CodexReviewGateFileEvidenceV0 {
	evidence := CodexReviewGateFileEvidenceV0{}
	for _, raw := range compactCodexDeliveryRefsV0(files) {
		if ctx != nil {
			if err := ctx.Err(); err != nil {
				evidence.Issues = append(evidence.Issues, codexReviewGateFileIssueV0("project_read_cancelled", "files"))
				return evidence
			}
		}
		file, issues := codexReviewGateProjectFileV0(root, raw)
		evidence.Files = append(evidence.Files, file)
		evidence.Issues = append(evidence.Issues, issues...)
	}
	return evidence
}

func codexReviewGateProjectFileV0(
	root string,
	rawPath string,
) (orquestacionnucleoapp.AutoprogrammingReviewGateFileV0, []orquestacionnucleoapp.AutoprogrammingReviewGateIssueV0) {
	rel, ok := codexReviewGateRelPathV0(rawPath)
	if !ok {
		return orquestacionnucleoapp.AutoprogrammingReviewGateFileV0{Path: strings.TrimSpace(rawPath)},
			[]orquestacionnucleoapp.AutoprogrammingReviewGateIssueV0{
				codexReviewGateFileIssueV0("delivery_path_invalid", "files.path"),
			}
	}
	fullPath := filepath.Join(root, filepath.FromSlash(rel))
	lineCount, issue := codexReviewGateLineCountFromFileV0(fullPath)
	file := orquestacionnucleoapp.AutoprogrammingReviewGateFileV0{
		Path:      rel,
		LineCount: lineCount,
	}
	if issue != nil {
		return file, []orquestacionnucleoapp.AutoprogrammingReviewGateIssueV0{*issue}
	}
	return file, nil
}

func codexReviewGateProjectRootV0(value string) (string, bool) {
	root := strings.TrimSpace(value)
	if root == "" || !filepath.IsAbs(root) {
		return "", false
	}
	return filepath.Clean(root), true
}

func codexReviewGateRelPathV0(value string) (string, bool) {
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

func codexReviewGateLineCountFromFileV0(
	path string,
) (int, *orquestacionnucleoapp.AutoprogrammingReviewGateIssueV0) {
	info, err := os.Lstat(path)
	if err != nil {
		return 0, codexReviewGateFileIssuePtrV0("delivery_file_missing", "files.path")
	}
	if !info.Mode().IsRegular() {
		return 0, codexReviewGateFileIssuePtrV0("delivery_file_unreadable", "files.path")
	}
	file, err := os.Open(path)
	if err != nil {
		return 0, codexReviewGateFileIssuePtrV0("delivery_file_unreadable", "files.path")
	}
	defer file.Close()
	lines, err := codexReviewGateCountLinesV0(file)
	if err != nil {
		return 0, codexReviewGateFileIssuePtrV0("delivery_file_unreadable", "files.path")
	}
	return lines, nil
}

func codexReviewGateCountLinesV0(reader io.Reader) (int, error) {
	buffered := bufio.NewReader(reader)
	lines := 0
	for {
		part, err := buffered.ReadString('\n')
		if part != "" {
			lines++
		}
		if errors.Is(err, io.EOF) {
			return lines, nil
		}
		if err != nil {
			return 0, err
		}
	}
}

func codexReviewGateFileIssuePtrV0(
	code string,
	field string,
) *orquestacionnucleoapp.AutoprogrammingReviewGateIssueV0 {
	issue := codexReviewGateFileIssueV0(code, field)
	return &issue
}

func codexReviewGateFileIssueV0(
	code string,
	field string,
) orquestacionnucleoapp.AutoprogrammingReviewGateIssueV0 {
	return orquestacionnucleoapp.AutoprogrammingReviewGateIssueV0{
		Code:  code,
		Field: field,
	}
}
