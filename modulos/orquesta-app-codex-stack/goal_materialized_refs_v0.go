package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

const (
	goalMaterializedRefsMaxFilesV0                 = 64
	goalMaterializedQAScanMaxFilesV0               = 256
	goalMaterializedQAScanMaxBytesV0               = 512 * 1024
	goalMaterializedWorkDeliveryFileV0             = "work_delivery.json"
	goalMaterializedOPESReworkDeliveryFileV0       = "opes_topic_rework_delivery.json"
	goalMaterializedGoalResultFileV0               = "orquesta_goal_result_v0.json"
	goalMaterializedCheckpointFileV0               = "checkpoint_started.txt"
	goalMaterializedMissingTerminalReceiptEvidence = "evidence-ref-goal-materialized-missing-terminal-receipt-after-artifacts-pass"
)

var errGoalMaterializedRefsScanDoneV0 = errors.New("goal_materialized_refs_scan_done")

type stackGoalMaterializedRefsSourceV0 struct {
	Config ConfigV0
}

type goalMaterializedRefsScanV0 struct {
	Result             orquestamcp.MCPDirectorGoalMaterializedRefsV0
	FilesScanned       int
	HasArtifact        bool
	HasQAPass          bool
	HasTerminalReceipt bool
}

func (source stackGoalMaterializedRefsSourceV0) ResolveDirectorGoalMaterializedRefsV0(
	_ context.Context,
	state orquestagoal.GoalWorkStateV0,
) (orquestamcp.MCPDirectorGoalMaterializedRefsV0, bool, error) {
	projectRoot := strings.TrimSpace(source.Config.Codex.ProjectWorkDir)
	if projectRoot == "" {
		return orquestamcp.MCPDirectorGoalMaterializedRefsV0{}, false, nil
	}
	projectRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return orquestamcp.MCPDirectorGoalMaterializedRefsV0{}, false, nil
	}
	scan := goalMaterializedRefsScanV0{}
	for _, scope := range state.Spec.WriteSet {
		if len(scan.Result.DomainReceiptRefs)+len(scan.Result.ArtifactRefs) >= goalMaterializedRefsMaxFilesV0 ||
			scan.FilesScanned >= goalMaterializedQAScanMaxFilesV0 {
			break
		}
		next, err := source.scanGoalWriteScopeForMaterializedRefsV0(projectRoot, state, scope)
		if err != nil {
			return orquestamcp.MCPDirectorGoalMaterializedRefsV0{}, false, err
		}
		scan = mergeGoalMaterializedRefsScanV0(scan, next)
	}
	result := scan.Result
	if !scan.HasTerminalReceipt && scan.HasArtifact && scan.HasQAPass {
		result.IssueCodes = append(result.IssueCodes, orquestamcp.MCPGoalFirstMissingTerminalReceiptAfterArtifactsPassV0)
		result.EvidenceRefs = append(result.EvidenceRefs, goalMaterializedMissingTerminalReceiptEvidence)
		result.ExpectedReceiptRefs = append(result.ExpectedReceiptRefs,
			"expected-terminal-receipt:"+safeGoalMaterializedRefPartV0(goalMaterializedGoalResultFileV0),
			"expected-terminal-receipt:"+safeGoalMaterializedRefPartV0(goalMaterializedOPESReworkDeliveryFileV0),
		)
	}
	if len(result.DomainReceiptRefs) == 0 &&
		len(result.ArtifactRefs) == 0 &&
		len(result.IssueCodes) == 0 &&
		len(result.EvidenceRefs) == 0 {
		return orquestamcp.MCPDirectorGoalMaterializedRefsV0{}, false, nil
	}
	if len(result.DomainReceiptRefs) > 0 {
		result.EvidenceRefs = append(result.EvidenceRefs, "evidence-ref-goal-materialized-work-delivery-detected")
		result.IssueCodes = append(result.IssueCodes, "goal_first_materialized_work_delivery_detected")
	}
	if len(result.ArtifactRefs) > 0 {
		result.EvidenceRefs = append(result.EvidenceRefs, "evidence-ref-goal-materialized-checkpoint-detected")
		result.IssueCodes = append(result.IssueCodes, "goal_first_materialized_checkpoint_detected")
	}
	result.ExpectedReceiptRefs = compactStringsV0(result.ExpectedReceiptRefs)
	result.EvidenceRefs = compactStringsV0(result.EvidenceRefs)
	result.IssueCodes = compactStringsV0(result.IssueCodes)
	return result, true, nil
}

func (source stackGoalMaterializedRefsSourceV0) scanGoalWriteScopeForMaterializedRefsV0(
	projectRoot string,
	state orquestagoal.GoalWorkStateV0,
	scope orquestagoal.GoalWriteScopeV0,
) (goalMaterializedRefsScanV0, error) {
	relScope := filepath.Clean(filepath.FromSlash(strings.TrimSpace(scope.Path)))
	if relScope == "." || relScope == "" || filepath.IsAbs(relScope) || strings.HasPrefix(relScope, ".."+string(filepath.Separator)) || relScope == ".." {
		return goalMaterializedRefsScanV0{}, nil
	}
	target := filepath.Join(projectRoot, relScope)
	if !pathWithinRootV0(projectRoot, target) {
		return goalMaterializedRefsScanV0{}, nil
	}
	info, err := os.Stat(target)
	if err != nil {
		return goalMaterializedRefsScanV0{}, nil
	}
	if !info.IsDir() {
		return source.scanGoalMaterializedFileV0(projectRoot, state, target, info), nil
	}
	scan := goalMaterializedRefsScanV0{}
	walkErr := filepath.WalkDir(target, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if len(scan.Result.DomainReceiptRefs)+len(scan.Result.ArtifactRefs) >= goalMaterializedRefsMaxFilesV0 ||
			scan.FilesScanned >= goalMaterializedQAScanMaxFilesV0 {
			return errGoalMaterializedRefsScanDoneV0
		}
		if entry == nil || entry.IsDir() {
			return nil
		}
		if !pathWithinRootV0(projectRoot, path) {
			return nil
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return nil
		}
		scan = mergeGoalMaterializedRefsScanV0(
			scan,
			source.scanGoalMaterializedFileV0(projectRoot, state, path, info),
		)
		return nil
	})
	if errors.Is(walkErr, errGoalMaterializedRefsScanDoneV0) {
		walkErr = nil
	}
	return scan, walkErr
}

func (source stackGoalMaterializedRefsSourceV0) scanGoalMaterializedFileV0(
	projectRoot string,
	state orquestagoal.GoalWorkStateV0,
	path string,
	info fs.FileInfo,
) goalMaterializedRefsScanV0 {
	scan := goalMaterializedRefsScanV0{FilesScanned: 1}
	base := strings.ToLower(strings.TrimSpace(filepath.Base(path)))
	switch base {
	case goalMaterializedWorkDeliveryFileV0:
		scan.Result.DomainReceiptRefs = []string{goalMaterializedWorkDeliveryRefV0(projectRoot, path, state.RunRef)}
		scan.HasTerminalReceipt = true
	case goalMaterializedOPESReworkDeliveryFileV0, goalMaterializedGoalResultFileV0:
		scan.HasTerminalReceipt = true
	case goalMaterializedCheckpointFileV0:
		scan.Result.ArtifactRefs = []string{goalMaterializedCheckpointRefV0(projectRoot, path, state.RunRef)}
	}
	if goalMaterializedFileLooksLikeArtifactV0(projectRoot, path, base) {
		scan.HasArtifact = true
	}
	if goalMaterializedFileLooksLikeQAReportV0(projectRoot, path, base, info) {
		scan.HasQAPass = true
		scan.Result.EvidenceRefs = append(scan.Result.EvidenceRefs, goalMaterializedQAPassRefV0(projectRoot, path, state.RunRef))
	}
	return scan
}

func mergeGoalMaterializedRefsScanV0(
	current goalMaterializedRefsScanV0,
	next goalMaterializedRefsScanV0,
) goalMaterializedRefsScanV0 {
	current.Result.DomainReceiptRefs = compactStringsV0(append(current.Result.DomainReceiptRefs, next.Result.DomainReceiptRefs...))
	current.Result.ArtifactRefs = compactStringsV0(append(current.Result.ArtifactRefs, next.Result.ArtifactRefs...))
	current.Result.ExpectedReceiptRefs = compactStringsV0(append(current.Result.ExpectedReceiptRefs, next.Result.ExpectedReceiptRefs...))
	current.Result.EvidenceRefs = compactStringsV0(append(current.Result.EvidenceRefs, next.Result.EvidenceRefs...))
	current.Result.IssueCodes = compactStringsV0(append(current.Result.IssueCodes, next.Result.IssueCodes...))
	current.FilesScanned += next.FilesScanned
	current.HasArtifact = current.HasArtifact || next.HasArtifact
	current.HasQAPass = current.HasQAPass || next.HasQAPass
	current.HasTerminalReceipt = current.HasTerminalReceipt || next.HasTerminalReceipt
	return current
}

func goalMaterializedFileLooksLikeArtifactV0(
	projectRoot string,
	path string,
	base string,
) bool {
	if goalMaterializedPathLooksLikeQAReportV0(projectRoot, path, base) {
		return false
	}
	switch strings.ToLower(filepath.Ext(base)) {
	case ".md", ".html", ".json", ".jsonl", ".pdf", ".mp3", ".wav", ".ogg", ".png", ".jpg", ".jpeg", ".webp":
		return true
	default:
		return false
	}
}

func goalMaterializedFileLooksLikeQAReportV0(
	projectRoot string,
	path string,
	base string,
	info fs.FileInfo,
) bool {
	if info == nil || info.IsDir() || info.Size() <= 0 || info.Size() > goalMaterializedQAScanMaxBytesV0 {
		return false
	}
	if !goalMaterializedPathLooksLikeQAReportV0(projectRoot, path, base) {
		return false
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	if strings.EqualFold(filepath.Ext(base), ".json") {
		var payload any
		if json.Unmarshal(raw, &payload) == nil && goalMaterializedJSONLooksLikeQAPassV0(payload) {
			return true
		}
	}
	return false
}

func goalMaterializedPathLooksLikeQAReportV0(
	projectRoot string,
	path string,
	base string,
) bool {
	rel, err := filepath.Rel(filepath.Clean(projectRoot), filepath.Clean(path))
	if err != nil {
		rel = path
	}
	rel = strings.ToLower(filepath.ToSlash(filepath.Clean(rel)))
	base = strings.ToLower(strings.TrimSpace(base))
	return strings.Contains(rel, "09_validacion/") ||
		strings.Contains(rel, "/validacion/") ||
		strings.Contains(base, "qa") ||
		strings.Contains(base, "validacion") ||
		strings.Contains(base, "validation") ||
		strings.Contains(base, "informe")
}

func goalMaterializedJSONLooksLikeQAPassV0(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			key = strings.ToLower(strings.TrimSpace(key))
			switch v := item.(type) {
			case bool:
				if v && (key == "passed" || key == "ok" || strings.HasSuffix(key, "_pass")) {
					return true
				}
			case string:
				text := strings.ToLower(strings.TrimSpace(v))
				if (key == "status" || key == "estado" || strings.HasSuffix(key, "_status")) &&
					(text == "pass" || text == "passed" || text == "ok" || text == "success") {
					return true
				}
				if strings.HasSuffix(key, "_pass") && (text == "true" || text == "ok" || text == "passed") {
					return true
				}
			}
			if goalMaterializedJSONLooksLikeQAPassV0(item) {
				return true
			}
		}
	case []any:
		for _, item := range typed {
			if goalMaterializedJSONLooksLikeQAPassV0(item) {
				return true
			}
		}
	}
	return false
}

func goalMaterializedQAPassRefV0(
	projectRoot string,
	path string,
	runRef string,
) string {
	rel, err := filepath.Rel(filepath.Clean(projectRoot), filepath.Clean(path))
	if err != nil {
		rel = filepath.Base(path)
	}
	rel = filepath.ToSlash(filepath.Clean(rel))
	return "evidence-ref-goal-materialized-qa-pass:" + safeGoalMaterializedRefPartV0(runRef) + ":" + safeGoalMaterializedRefPartV0(rel)
}

func pathWithinRootV0(root string, path string) bool {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func goalMaterializedWorkDeliveryRefV0(
	projectRoot string,
	path string,
	runRef string,
) string {
	rel, err := filepath.Rel(filepath.Clean(projectRoot), filepath.Clean(path))
	if err != nil {
		rel = filepath.Base(path)
	}
	rel = filepath.ToSlash(filepath.Clean(rel))
	return "domain-receipt-ref-work-delivery:" + safeGoalMaterializedRefPartV0(runRef) + ":" + safeGoalMaterializedRefPartV0(rel)
}

func goalMaterializedCheckpointRefV0(
	projectRoot string,
	path string,
	runRef string,
) string {
	rel, err := filepath.Rel(filepath.Clean(projectRoot), filepath.Clean(path))
	if err != nil {
		rel = filepath.Base(path)
	}
	rel = filepath.ToSlash(filepath.Clean(rel))
	return "artifact-ref-checkpoint:" + safeGoalMaterializedRefPartV0(runRef) + ":" + safeGoalMaterializedRefPartV0(rel)
}

func safeGoalMaterializedRefPartV0(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	var builder strings.Builder
	lastSep := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastSep = false
		default:
			if !lastSep {
				builder.WriteByte('-')
				lastSep = true
			}
		}
	}
	out := strings.Trim(builder.String(), "-")
	if out == "" {
		return "unknown"
	}
	return out
}
