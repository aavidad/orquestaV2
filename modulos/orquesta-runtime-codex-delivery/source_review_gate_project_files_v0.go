package orquestaruntimecodexdelivery

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
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
			Issues: []orquestaautoprogramming.AutoprogrammingReviewGateIssueV0{
				codexReviewGateFileIssueV0("project_workdir_invalid", "project_workdir"),
			},
		}, nil
	}
	evidence := provider.fileEvidenceFromRootV0(ctx, root, ack.Files)
	evidence.Issues = append(
		evidence.Issues,
		codexReviewGateProjectWriteSetIssuesV0(ctx, root, descriptor.Spec.AgentPacket.Task.WriteSet)...,
	)
	return evidence, nil
}

func (provider CodexReviewGateProjectFileEvidenceV0) BuildCodexReviewGateFilesV0(
	ctx context.Context,
	descriptor CodexReceiptDescriptorV0,
	ack orquestaruntimecodex.CodexAgentAckV0,
) ([]orquestaautoprogramming.AutoprogrammingReviewGateFileV0, error) {
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
) (orquestaautoprogramming.AutoprogrammingReviewGateFileV0, []orquestaautoprogramming.AutoprogrammingReviewGateIssueV0) {
	rel, ok := codexReviewGateRelPathV0(rawPath)
	if !ok {
		return orquestaautoprogramming.AutoprogrammingReviewGateFileV0{Path: strings.TrimSpace(rawPath)},
			[]orquestaautoprogramming.AutoprogrammingReviewGateIssueV0{
				codexReviewGateFileIssueV0("delivery_path_invalid", "files.path"),
			}
	}
	fullPath := filepath.Join(root, filepath.FromSlash(rel))
	lineCount, issue := codexReviewGateLineCountFromFileV0(fullPath)
	file := orquestaautoprogramming.AutoprogrammingReviewGateFileV0{
		Path:      rel,
		LineCount: lineCount,
	}
	if issue != nil {
		return file, []orquestaautoprogramming.AutoprogrammingReviewGateIssueV0{*issue}
	}
	return file, nil
}

func codexReviewGateProjectWriteSetIssuesV0(
	ctx context.Context,
	root string,
	writeSet []string,
) []orquestaautoprogramming.AutoprogrammingReviewGateIssueV0 {
	issues := make([]orquestaautoprogramming.AutoprogrammingReviewGateIssueV0, 0)
	for _, target := range compactCodexDeliveryRefsV0(writeSet) {
		if ctx != nil {
			if err := ctx.Err(); err != nil {
				return append(issues, codexReviewGateFileIssueV0("project_read_cancelled", "write_set"))
			}
		}
		if codexReviewGateProjectTargetExistsV0(root, target) {
			continue
		}
		issues = append(issues, codexReviewGateFileIssueV0(
			"write_set_target_missing:"+codexReviewGateIssueTargetV0(target),
			"write_set",
		))
	}
	return issues
}

func codexReviewGateProjectTargetExistsV0(root string, rawTarget string) bool {
	target, ok := codexReviewGateRelPathV0(rawTarget)
	if !ok {
		return false
	}
	if target == "web" && codexReviewGateProjectTargetExistsV0(root, "internal/webadmin") {
		return true
	}
	if codexReviewGateHasGlobV0(target) {
		return codexReviewGateProjectGlobHasFileV0(root, target)
	}
	fullPath := filepath.Join(root, filepath.FromSlash(target))
	info, err := os.Stat(fullPath)
	if err != nil {
		return false
	}
	if !info.IsDir() {
		return info.Size() > 0
	}
	return codexReviewGateDirHasFileV0(fullPath)
}

func codexReviewGateProjectGlobHasFileV0(root string, pattern string) bool {
	re, err := codexReviewGateGlobRegexpV0(pattern)
	if err != nil {
		return false
	}
	found := false
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry == nil || found {
			return nil
		}
		if entry.IsDir() {
			if codexReviewGateSkipProjectDirV0(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil || info.Size() == 0 {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err == nil && re.MatchString(filepath.ToSlash(rel)) {
			found = true
		}
		return nil
	})
	return found
}

func codexReviewGateDirHasFileV0(dir string) bool {
	found := false
	_ = filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry == nil || found {
			return nil
		}
		if entry.IsDir() {
			if path != dir && codexReviewGateSkipProjectDirV0(entry.Name()) {
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

func codexReviewGateSkipProjectDirV0(name string) bool {
	switch name {
	case ".git", ".orquesta-runtime", ".orquesta-codex-runtime":
		return true
	default:
		return false
	}
}

func codexReviewGateHasGlobV0(value string) bool {
	return strings.ContainsAny(value, "*?[")
}

func codexReviewGateGlobRegexpV0(pattern string) (*regexp.Regexp, error) {
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

func codexReviewGateIssueTargetV0(value string) string {
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
) (int, *orquestaautoprogramming.AutoprogrammingReviewGateIssueV0) {
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
) *orquestaautoprogramming.AutoprogrammingReviewGateIssueV0 {
	issue := codexReviewGateFileIssueV0(code, field)
	return &issue
}

func codexReviewGateFileIssueV0(
	code string,
	field string,
) orquestaautoprogramming.AutoprogrammingReviewGateIssueV0 {
	return orquestaautoprogramming.AutoprogrammingReviewGateIssueV0{
		Code:  code,
		Field: field,
	}
}
