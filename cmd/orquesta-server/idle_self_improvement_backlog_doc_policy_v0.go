package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	idleSelfImprovementBacklogScanMaxDocsV0          = 64
	idleSelfImprovementBacklogScanMaxDocumentBytesV0 = 4 * 1024 * 1024
)

func (planner idleSelfImprovementBacklogPlannerV0) readBacklogScanDocumentTextV0(rel string) (string, error) {
	body, err := planner.readBacklogScanDocumentBytesV0(rel)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (planner idleSelfImprovementBacklogPlannerV0) readBacklogScanDocumentBytesV0(rel string) ([]byte, error) {
	_, path, err := backlogScanSafeDocumentPathV0(planner.ProjectWorkDir, rel)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	body, err := io.ReadAll(io.LimitReader(file, idleSelfImprovementBacklogScanMaxDocumentBytesV0+1))
	if err != nil {
		return nil, err
	}
	if len(body) > idleSelfImprovementBacklogScanMaxDocumentBytesV0 {
		return nil, errors.New("backlog_scan_doc_budget_exceeded")
	}
	return body, nil
}

func backlogScanSafeDocumentPathV0(projectDir string, rel string) (string, string, error) {
	projectDir = strings.TrimSpace(projectDir)
	clean, ok := backlogScanCleanDocumentRelV0(rel)
	if projectDir == "" || !ok {
		return "", "", errors.New("backlog_scan_doc_path_blocked")
	}
	base, err := filepath.Abs(projectDir)
	if err != nil {
		return "", "", err
	}
	path := filepath.Join(base, filepath.FromSlash(clean))
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", "", err
	}
	baseRel, err := filepath.Rel(base, absPath)
	if err != nil || baseRel == ".." || strings.HasPrefix(filepath.ToSlash(baseRel), "../") {
		return "", "", errors.New("backlog_scan_doc_outside_project")
	}
	if err := backlogScanRejectSymlinkPathV0(base, clean); err != nil {
		return "", "", err
	}
	info, err := os.Lstat(absPath)
	if err != nil {
		return "", "", err
	}
	if !info.Mode().IsRegular() {
		return "", "", errors.New("backlog_scan_doc_not_regular")
	}
	if info.Size() > idleSelfImprovementBacklogScanMaxDocumentBytesV0 {
		return "", "", errors.New("backlog_scan_doc_budget_exceeded")
	}
	return clean, absPath, nil
}

func backlogScanCleanDocumentRelV0(rel string) (string, bool) {
	value := filepath.ToSlash(strings.TrimSpace(rel))
	if value == "" || filepath.IsAbs(value) || strings.Contains(value, "\\") {
		return "", false
	}
	parts := strings.Split(value, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || strings.HasPrefix(part, ".") {
			return "", false
		}
	}
	clean := filepath.ToSlash(filepath.Clean(value))
	if clean != value || !strings.HasSuffix(clean, ".md") {
		return "", false
	}
	if !strings.HasPrefix(clean, "docs/") && !strings.Contains(clean, "/docs/") {
		return "", false
	}
	return clean, true
}

func backlogScanRejectSymlinkPathV0(base string, rel string) error {
	current := base
	for _, part := range strings.Split(filepath.FromSlash(rel), string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("backlog_scan_doc_symlink_blocked")
		}
	}
	return nil
}

func backlogScanCanonicalDocSetV0(docs []string) map[string]bool {
	out := map[string]bool{}
	for _, rel := range compactServerStackStringsV0(docs) {
		clean, ok := backlogScanCleanDocumentRelV0(rel)
		if ok {
			out[clean] = true
		}
	}
	return out
}
