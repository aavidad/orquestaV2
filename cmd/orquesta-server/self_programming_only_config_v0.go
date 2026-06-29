package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func serverSelfProgrammingOnlyFromEnvV0() bool {
	return boolEnvOrDefaultV0(envServerSelfProgrammingOnlyV0, false)
}

func validateServerSelfProgrammingOnlyConfigV0(config orquestaserver.ConfigV0) error {
	if !serverSelfProgrammingOnlyFromEnvV0() {
		return nil
	}
	root, err := serverSelfProgrammingRootFromEnvV0()
	if err != nil {
		return err
	}
	if codexGoalBackendFromEnvV0() != codexGoalBackendAppServerTmuxV0 {
		return fmt.Errorf("orquesta_server: self_programming_only_requires_app_server_tmux")
	}
	if config.ResidentDirectorEnabled {
		return fmt.Errorf("orquesta_server: self_programming_only_resident_director_disabled_required")
	}
	for _, scopedPath := range []struct {
		label string
		path  string
	}{
		{label: "project_work_dir", path: config.ProjectWorkDir},
		{label: "runtime_work_dir", path: config.RuntimeWorkDir},
		{label: "state_dir", path: config.StateDir},
		{label: "idle_self_improvement_project_work_dir", path: config.IdleSelfImprovementProjectWorkDir},
	} {
		if !pathInsideSelfProgrammingRootV0(scopedPath.path, root) {
			return fmt.Errorf("orquesta_server: self_programming_only_%s_outside_root", scopedPath.label)
		}
	}
	for _, key := range selfProgrammingOnlyMustBeEmptyEnvV0() {
		if strings.TrimSpace(os.Getenv(key)) != "" {
			return fmt.Errorf("orquesta_server: self_programming_only_env_forbidden:%s", key)
		}
	}
	for _, key := range selfProgrammingOnlyMustBeFalseEnvV0() {
		if boolEnvOrDefaultV0(key, false) {
			return fmt.Errorf("orquesta_server: self_programming_only_env_must_be_false:%s", key)
		}
	}
	if boolEnvOrDefaultV0(envServerAutoprogrammingPromotionEnabledV0, false) {
		archiveDir := strings.TrimSpace(os.Getenv(envServerAutoprogrammingPromotionArchiveDirV0))
		if archiveDir != "" && !pathInsideSelfProgrammingRootV0(archiveDir, root) {
			return fmt.Errorf("orquesta_server: self_programming_only_promotion_archive_dir_outside_root")
		}
	}
	if !boolEnvOrDefaultV0(envOPESBridgeDryRunV0, true) {
		return fmt.Errorf("orquesta_server: self_programming_only_opes_bridge_dry_run_required")
	}
	if !boolEnvOrDefaultV0(envOPESRegistryFinalPkgDryRunV0, true) {
		return fmt.Errorf("orquesta_server: self_programming_only_opes_finalpkg_dry_run_required")
	}
	return nil
}

func serverSelfProgrammingRootFromEnvV0() (string, error) {
	raw := strings.TrimSpace(os.Getenv(envServerSelfProgrammingRootV0))
	if raw == "" {
		return "", fmt.Errorf("orquesta_server: self_programming_root_required")
	}
	abs, err := filepath.Abs(raw)
	if err != nil || !filepath.IsAbs(abs) {
		return "", fmt.Errorf("orquesta_server: self_programming_root_invalid")
	}
	return filepath.Clean(abs), nil
}

func pathInsideSelfProgrammingRootV0(path string, root string) bool {
	cleanPath := filepath.Clean(strings.TrimSpace(path))
	cleanRoot := filepath.Clean(strings.TrimSpace(root))
	if cleanPath == "" || cleanRoot == "" || !filepath.IsAbs(cleanPath) || !filepath.IsAbs(cleanRoot) {
		return false
	}
	rel, err := filepath.Rel(cleanRoot, cleanPath)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func selfProgrammingOnlyMustBeEmptyEnvV0() []string {
	return []string{
		envOPESBaseURLV0,
		envOPESBaseURLLegacyV0,
		envOPESBridgeJobTypeV0,
		envOPESBridgeJobRefV0,
		envOPESBridgeProgramIDV0,
		envOPESBridgeTopicIDV0,
		envOPESBridgeCorrelationIDV0,
		envOPESBridgeJobTypeSequenceV0,
		envOPESBridgeInputLedgerPathV0,
		envOPESBridgeDestinationEvidenceV0,
		envOPESRegistryFinalPkgRegistryV0,
		envOPESRegistryFinalPkgCourseRootV0,
		envOPESTopicRegistryToolPathV0,
		envOPESProjectWorkDirV0,
		envDomainWorkHTTPBaseURLV0,
		envDomainDeliveryLedgerPathV0,
	}
}

func selfProgrammingOnlyMustBeFalseEnvV0() []string {
	return []string{
		envAutoprogrammingLegacyDirectorLoopV0,
		envExternalWorkLegacyDirectorLoopV0,
		envOPESBridgeEnabledV0,
		envOPESBridgeConfirmV0,
		envOPESBridgeAllowUnfilteredV0,
		envOPESBridgeProductiveConfirmV0,
		envOPESRegistryFinalPkgEnabledV0,
		envOPESRegistryFinalPkgConfirmV0,
		envOPESRegistryFinalPkgReconcileV0,
		envOPESTopicRegistryEnabledV0,
		envOPESTopicRegistryForceV0,
		envDomainWorkFileEnabledV0,
	}
}
