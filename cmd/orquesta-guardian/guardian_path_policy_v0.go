package main

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	guardianPathPolicySchemaVersionV0       = "orquesta_guardian_path_policy.v0"
	guardianConfigPathOutsideAllowedRootV0  = "guardian_config_path_outside_allowed_root"
	guardianConfigPathSymlinkBlockedV0      = "guardian_config_path_symlink_blocked"
	guardianConfigProjectRootInvalidV0      = "guardian_config_project_root_invalid"
	guardianPathClassificationProductRootV0 = "product_root"
	guardianPathClassificationControlRootV0 = "control_root"
	guardianPathClassificationBinaryV0      = "binary_artifact"
	guardianPathClassificationDiagnosticV0  = "local_diagnostic_path"
	guardianPathClassificationOpaqueRefV0   = "opaque_ref"
	guardianPathPolicyAccessLocalOnlyRefV0  = "guardian-local-path-access-policy-ref-v0"
)

type guardianPathRootPolicyV0 struct {
	SchemaVersion string                         `json:"schema_version"`
	Items         []guardianPathClassificationV0 `json:"items,omitempty"`
	ReasonCodes   []string                       `json:"reason_codes,omitempty"`
}

type guardianPathClassificationV0 struct {
	Field           string `json:"field"`
	Classification  string `json:"classification"`
	Ref             string `json:"ref"`
	ParentRef       string `json:"parent_ref,omitempty"`
	AccessPolicyRef string `json:"access_policy_ref"`
}

func applyGuardianPathPolicyV0(config guardianConfigV0) (guardianConfigV0, error) {
	if err := validateGuardianPathRootPolicyV0(config); err != nil {
		return guardianConfigV0{}, err
	}
	config.PathPolicy = guardianPathRootPolicyV0{
		SchemaVersion: guardianPathPolicySchemaVersionV0,
		Items:         guardianPathPolicyItemsV0(config),
		ReasonCodes: []string{
			"guardian_path_roots_classified",
			"guardian_public_paths_refs_only",
		},
	}
	return config, nil
}

func validateGuardianPathRootPolicyV0(config guardianConfigV0) error {
	if err := validateGuardianProjectRootV0(config.ProjectDir); err != nil {
		return err
	}
	for _, item := range []struct {
		field string
		path  string
	}{
		{field: "state_dir", path: config.StateDir},
		{field: "repair_runtime_dir", path: config.RepairCodexRuntimeDir},
	} {
		if err := validateGuardianControlRootPathV0(item.field, item.path); err != nil {
			return err
		}
	}
	if config.ArtifactRoot != "" {
		if err := validateGuardianControlRootPathV0("artifact_root", config.ArtifactRoot); err != nil {
			return err
		}
	}
	if err := requireGuardianPathUnderRootV0("repair_runtime_dir", config.RepairCodexRuntimeDir, []string{config.StateDir}); err != nil {
		return err
	}
	roots := guardianAllowedArtifactRootsV0(config)
	for field, path := range map[string]string{
		"current_bin":   config.CurrentBin,
		"candidate_bin": config.CandidateBin,
		"last_good_bin": config.LastGoodBin,
	} {
		if strings.TrimSpace(path) == "" {
			continue
		}
		if err := requireGuardianPathUnderRootV0(field, path, roots); err != nil {
			return err
		}
	}
	return nil
}

func validateGuardianProjectRootV0(path string) error {
	if err := rejectGuardianSymlinkPathV0(path); err != nil {
		return guardianConfigParseErrorV0{Code: guardianConfigPathSymlinkBlockedV0, Field: "project_dir"}
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return guardianConfigParseErrorV0{Code: guardianConfigProjectRootInvalidV0, Field: "project_dir"}
	}
	return nil
}

func validateGuardianControlRootPathV0(field string, path string) error {
	if err := rejectGuardianSymlinkPathV0(path); err != nil {
		return guardianConfigParseErrorV0{Code: guardianConfigPathSymlinkBlockedV0, Field: field}
	}
	return nil
}

func guardianAllowedArtifactRootsV0(config guardianConfigV0) []string {
	roots := []string{config.ProjectDir, config.StateDir}
	if strings.TrimSpace(config.ArtifactRoot) != "" {
		roots = append(roots, config.ArtifactRoot)
	}
	return roots
}

func requireGuardianPathUnderRootV0(field string, path string, roots []string) error {
	for _, root := range roots {
		if guardianPathWithinRootV0(path, root) {
			return nil
		}
	}
	return guardianConfigParseErrorV0{Code: guardianConfigPathOutsideAllowedRootV0, Field: field}
}

func guardianPathWithinRootV0(path string, root string) bool {
	path = filepath.Clean(path)
	root = filepath.Clean(root)
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (!filepath.IsAbs(rel) && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}

func guardianPathPolicyItemsV0(config guardianConfigV0) []guardianPathClassificationV0 {
	items := []guardianPathClassificationV0{
		guardianPathPolicyItemV0("project_dir", guardianPathClassificationProductRootV0, "", config.ProjectDir),
		guardianPathPolicyItemV0("state_dir", guardianPathClassificationControlRootV0, "", config.StateDir),
	}
	if config.ArtifactRoot != "" {
		items = append(items, guardianPathPolicyItemV0("artifact_root", guardianPathClassificationControlRootV0, "", config.ArtifactRoot))
	}
	for _, item := range []struct {
		field string
		path  string
	}{
		{"current_bin", config.CurrentBin},
		{"candidate_bin", config.CandidateBin},
		{"last_good_bin", config.LastGoodBin},
	} {
		if item.path != "" {
			items = append(items, guardianPathPolicyItemV0(item.field, guardianPathClassificationBinaryV0, rootRefForGuardianPathV0(config, item.path), item.path))
		}
	}
	items = append(items,
		guardianPathPolicyItemV0("repair_runtime_dir", guardianPathClassificationDiagnosticV0, guardianPathRefV0("guardian-control-root-ref", config.StateDir), config.RepairCodexRuntimeDir),
		guardianPathPolicyItemV0("logs", guardianPathClassificationDiagnosticV0, guardianPathRefV0("guardian-control-root-ref", config.StateDir), filepath.Join(config.StateDir, "logs")),
		guardianPathPolicyItemV0("manifests", guardianPathClassificationDiagnosticV0, guardianPathRefV0("guardian-control-root-ref", config.StateDir), filepath.Dir(config.ManifestPath)),
		guardianPathPolicyItemV0("worktree_ref", guardianPathClassificationOpaqueRefV0, "", "worktree_ref"),
		guardianPathPolicyItemV0("branch_ref", guardianPathClassificationOpaqueRefV0, "", "branch_ref"),
	)
	return items
}

func guardianPathPolicyItemV0(field string, classification string, parentRef string, path string) guardianPathClassificationV0 {
	prefix := "guardian-local-path-ref"
	if classification == guardianPathClassificationProductRootV0 {
		prefix = "guardian-product-root-ref"
	}
	if classification == guardianPathClassificationControlRootV0 {
		prefix = "guardian-control-root-ref"
	}
	if classification == guardianPathClassificationOpaqueRefV0 {
		prefix = "guardian-opaque-ref"
	}
	return guardianPathClassificationV0{
		Field:           field,
		Classification:  classification,
		Ref:             guardianPathRefV0(prefix, path),
		ParentRef:       parentRef,
		AccessPolicyRef: guardianPathPolicyAccessLocalOnlyRefV0,
	}
}

func rootRefForGuardianPathV0(config guardianConfigV0, path string) string {
	for _, root := range []struct {
		prefix string
		path   string
	}{
		{"guardian-product-root-ref", config.ProjectDir},
		{"guardian-control-root-ref", config.StateDir},
		{"guardian-control-root-ref", config.ArtifactRoot},
	} {
		if root.path != "" && guardianPathWithinRootV0(path, root.path) {
			return guardianPathRefV0(root.prefix, root.path)
		}
	}
	return ""
}
