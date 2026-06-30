package orquestaappcodexstack

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

const (
	goalMaterializedRefsMaxFilesV0     = 64
	goalMaterializedWorkDeliveryFileV0 = "work_delivery.json"
)

var errGoalMaterializedRefsScanDoneV0 = errors.New("goal_materialized_refs_scan_done")

type stackGoalMaterializedRefsSourceV0 struct {
	Config ConfigV0
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
	result := orquestamcp.MCPDirectorGoalMaterializedRefsV0{}
	for _, scope := range state.Spec.WriteSet {
		if len(result.DomainReceiptRefs) >= goalMaterializedRefsMaxFilesV0 {
			break
		}
		refs, err := source.scanGoalWriteScopeForWorkDeliveriesV0(projectRoot, state, scope)
		if err != nil {
			return orquestamcp.MCPDirectorGoalMaterializedRefsV0{}, false, err
		}
		result.DomainReceiptRefs = compactStringsV0(append(result.DomainReceiptRefs, refs...))
	}
	if len(result.DomainReceiptRefs) == 0 {
		return orquestamcp.MCPDirectorGoalMaterializedRefsV0{}, false, nil
	}
	result.EvidenceRefs = []string{"evidence-ref-goal-materialized-work-delivery-detected"}
	result.IssueCodes = []string{"goal_first_materialized_work_delivery_detected"}
	return result, true, nil
}

func (source stackGoalMaterializedRefsSourceV0) scanGoalWriteScopeForWorkDeliveriesV0(
	projectRoot string,
	state orquestagoal.GoalWorkStateV0,
	scope orquestagoal.GoalWriteScopeV0,
) ([]string, error) {
	relScope := filepath.Clean(filepath.FromSlash(strings.TrimSpace(scope.Path)))
	if relScope == "." || relScope == "" || filepath.IsAbs(relScope) || strings.HasPrefix(relScope, ".."+string(filepath.Separator)) || relScope == ".." {
		return nil, nil
	}
	target := filepath.Join(projectRoot, relScope)
	if !pathWithinRootV0(projectRoot, target) {
		return nil, nil
	}
	info, err := os.Stat(target)
	if err != nil {
		return nil, nil
	}
	if !info.IsDir() {
		if strings.EqualFold(filepath.Base(target), goalMaterializedWorkDeliveryFileV0) {
			return []string{goalMaterializedWorkDeliveryRefV0(projectRoot, target, state.RunRef)}, nil
		}
		return nil, nil
	}
	refs := []string{}
	walkErr := filepath.WalkDir(target, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if len(refs) >= goalMaterializedRefsMaxFilesV0 {
			return errGoalMaterializedRefsScanDoneV0
		}
		if entry == nil || entry.IsDir() || !strings.EqualFold(entry.Name(), goalMaterializedWorkDeliveryFileV0) {
			return nil
		}
		if !pathWithinRootV0(projectRoot, path) {
			return nil
		}
		refs = append(refs, goalMaterializedWorkDeliveryRefV0(projectRoot, path, state.RunRef))
		return nil
	})
	if errors.Is(walkErr, errGoalMaterializedRefsScanDoneV0) {
		walkErr = nil
	}
	return refs, walkErr
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
