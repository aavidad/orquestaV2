package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
	orquestaruntimerequiredtest "orquesta/modulos/orquesta-runtime-required-test"
	orquestaserver "orquesta/modulos/orquesta-server"
)

const goalRequiredTestAttestationConfigSchemaV0 = "orquesta_goal_required_test_attestation_config.v0"

type serverGoalRequiredTestAttestationConfigFileV0 struct {
	SchemaVersion       string                                          `json:"schema_version"`
	ProjectWorkDir      string                                          `json:"project_work_dir"`
	RuntimeRoot         string                                          `json:"runtime_root"`
	GitCommandPath      string                                          `json:"git_command_path"`
	AllowedCommands     map[string]string                               `json:"allowed_commands"`
	ModuleCacheSnapshot string                                          `json:"module_cache_snapshot"`
	PreflightCommands   []string                                        `json:"preflight_commands"`
	MaxRuntimeSeconds   int                                             `json:"max_runtime_seconds"`
	MaxOutputBytes      int64                                           `json:"max_output_bytes"`
	MaxArtifacts        int                                             `json:"max_artifacts"`
	Identity            serverGoalRequiredTestAttestationIdentityFileV0 `json:"identity"`
}

type serverGoalRequiredTestAttestationIdentityFileV0 struct {
	TrustPolicyRef           string `json:"trust_policy_ref"`
	ImplementerAgentRef      string `json:"implementer_agent_ref"`
	ImplementerCredentialRef string `json:"implementer_credential_ref"`
	ImplementerPrincipalRef  string `json:"implementer_principal_ref"`
	AttestorAgentRef         string `json:"attestor_agent_ref"`
	AttestorCredentialRef    string `json:"attestor_credential_ref"`
	AttestorPrincipalRef     string `json:"attestor_principal_ref"`
}

func goalRequiredTestAttestationAdapterFromConfigV0(
	serverConfig orquestaserver.ConfigV0,
	projectConfig serverProjectConfigFileV0,
) (*orquestaruntimerequiredtest.LocalGoalRequiredTestAttestationAdapterV0, error) {
	config, configured, err := goalRequiredTestAttestationRuntimeConfigFromConfigV0(serverConfig, projectConfig)
	if err != nil || !configured {
		return nil, err
	}
	adapter, err := orquestaruntimerequiredtest.NewLocalGoalRequiredTestAttestationAdapterV0(config)
	if err != nil {
		return nil, err
	}
	if err := adapter.PreflightGoalRequiredTestAttestationV0(context.Background()); err != nil {
		return nil, fmt.Errorf("goal_required_test_attestation_preflight_failed")
	}
	return adapter, nil
}

func goalRequiredTestAttestationRuntimeConfigFromConfigV0(
	serverConfig orquestaserver.ConfigV0,
	projectConfig serverProjectConfigFileV0,
) (orquestaruntimerequiredtest.LocalGoalRequiredTestAttestationConfigV0, bool, error) {
	path, configured, err := goalRequiredTestAttestationConfigFilePathV0(serverConfig.ProjectWorkDir, projectConfig)
	if err != nil || !configured {
		return orquestaruntimerequiredtest.LocalGoalRequiredTestAttestationConfigV0{}, configured, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return orquestaruntimerequiredtest.LocalGoalRequiredTestAttestationConfigV0{}, false, fmt.Errorf("goal_required_test_attestation_config_read: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		return orquestaruntimerequiredtest.LocalGoalRequiredTestAttestationConfigV0{}, false, fmt.Errorf("goal_required_test_attestation_config_not_owner_only")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return orquestaruntimerequiredtest.LocalGoalRequiredTestAttestationConfigV0{}, false, fmt.Errorf("goal_required_test_attestation_config_read: %w", err)
	}
	if len(raw) == 0 || len(raw) > 1024*1024 {
		return orquestaruntimerequiredtest.LocalGoalRequiredTestAttestationConfigV0{}, false, fmt.Errorf("goal_required_test_attestation_config_size_invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var document serverGoalRequiredTestAttestationConfigFileV0
	if err := decoder.Decode(&document); err != nil {
		return orquestaruntimerequiredtest.LocalGoalRequiredTestAttestationConfigV0{}, false, fmt.Errorf("goal_required_test_attestation_config_json_invalid: %w", err)
	}
	if err := rejectGoalRequiredTestAttestationConfigTrailingJSONV0(decoder); err != nil {
		return orquestaruntimerequiredtest.LocalGoalRequiredTestAttestationConfigV0{}, false, err
	}
	if strings.TrimSpace(document.SchemaVersion) != goalRequiredTestAttestationConfigSchemaV0 {
		return orquestaruntimerequiredtest.LocalGoalRequiredTestAttestationConfigV0{}, false, fmt.Errorf("goal_required_test_attestation_config_schema_invalid")
	}
	configuredProject, err := filepath.Abs(strings.TrimSpace(document.ProjectWorkDir))
	if err != nil {
		return orquestaruntimerequiredtest.LocalGoalRequiredTestAttestationConfigV0{}, false, fmt.Errorf("goal_required_test_attestation_project_work_dir_invalid")
	}
	serverProject, err := filepath.Abs(strings.TrimSpace(serverConfig.ProjectWorkDir))
	if err != nil {
		return orquestaruntimerequiredtest.LocalGoalRequiredTestAttestationConfigV0{}, false, fmt.Errorf("goal_required_test_attestation_project_work_dir_invalid")
	}
	configuredProject, configuredErr := filepath.EvalSymlinks(configuredProject)
	serverProject, serverErr := filepath.EvalSymlinks(serverProject)
	if configuredErr != nil || serverErr != nil || configuredProject != serverProject {
		return orquestaruntimerequiredtest.LocalGoalRequiredTestAttestationConfigV0{}, false, fmt.Errorf("goal_required_test_attestation_project_work_dir_mismatch")
	}
	digest := sha256.Sum256(raw)
	identity := document.Identity
	return orquestaruntimerequiredtest.LocalGoalRequiredTestAttestationConfigV0{
		ProjectWorkDir: configuredProject,
		RuntimeRoot:    strings.TrimSpace(document.RuntimeRoot), GitCommandPath: strings.TrimSpace(document.GitCommandPath),
		AllowedCommands:        document.AllowedCommands,
		DependencySnapshotPath: strings.TrimSpace(document.ModuleCacheSnapshot),
		PreflightCommands:      append([]string(nil), document.PreflightCommands...),
		MaxRuntime:             time.Duration(document.MaxRuntimeSeconds) * time.Second,
		MaxOutputBytes:         document.MaxOutputBytes, MaxArtifacts: document.MaxArtifacts,
		Identity: orquestaruntimerequiredtest.LocalTrustedGoalRequiredTestIdentityPolicyV0{
			TrustPolicyRef:           strings.TrimSpace(identity.TrustPolicyRef),
			PolicyEvidenceRef:        "goal-required-test-local-policy-evidence-ref-" + hex.EncodeToString(digest[:]),
			ImplementerAgentRef:      strings.TrimSpace(identity.ImplementerAgentRef),
			ImplementerCredentialRef: strings.TrimSpace(identity.ImplementerCredentialRef),
			ImplementerPrincipalRef:  strings.TrimSpace(identity.ImplementerPrincipalRef),
			AttestorAgentRef:         strings.TrimSpace(identity.AttestorAgentRef),
			AttestorCredentialRef:    strings.TrimSpace(identity.AttestorCredentialRef),
			AttestorPrincipalRef:     strings.TrimSpace(identity.AttestorPrincipalRef),
		},
	}, true, nil
}

type serverGoalRequiredTestAttestationWorkspaceSelectorV0 struct {
	Canonical       *orquestaruntimerequiredtest.LocalGoalRequiredTestAttestationAdapterV0
	RuntimeConfig   orquestaruntimerequiredtest.LocalGoalRequiredTestAttestationConfigV0
	WorkspaceLookup serverCodexGoalWorkspaceBindingLookupV0
}

func goalRequiredTestAttestationWorkspaceSelectorFromConfigV0(
	serverConfig orquestaserver.ConfigV0,
	projectConfig serverProjectConfigFileV0,
	workspaceLookup serverCodexGoalWorkspaceBindingLookupV0,
) (*serverGoalRequiredTestAttestationWorkspaceSelectorV0, error) {
	runtimeConfig, configured, err := goalRequiredTestAttestationRuntimeConfigFromConfigV0(serverConfig, projectConfig)
	if err != nil || !configured {
		return nil, err
	}
	canonical, err := orquestaruntimerequiredtest.NewLocalGoalRequiredTestAttestationAdapterV0(runtimeConfig)
	if err != nil {
		return nil, err
	}
	if err := canonical.PreflightGoalRequiredTestAttestationV0(context.Background()); err != nil {
		return nil, fmt.Errorf("goal_required_test_attestation_preflight_failed")
	}
	return &serverGoalRequiredTestAttestationWorkspaceSelectorV0{
		Canonical:       canonical,
		RuntimeConfig:   runtimeConfig,
		WorkspaceLookup: workspaceLookup,
	}, nil
}

func (selector *serverGoalRequiredTestAttestationWorkspaceSelectorV0) BindGoalRequiredTestSpecV0(
	ctx context.Context,
	spec orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalWorkSpecV0, error) {
	if selector == nil || selector.Canonical == nil {
		return orquestagoal.GoalWorkSpecV0{}, fmt.Errorf("goal_required_test_attestation_adapter_unavailable")
	}
	return selector.Canonical.BindGoalRequiredTestSpecV0(ctx, spec)
}

func (selector *serverGoalRequiredTestAttestationWorkspaceSelectorV0) CaptureGoalRequiredTestFinalSnapshotV0(
	ctx context.Context,
	request orquestagoal.GoalRequiredTestFinalSnapshotRequestV0,
) (orquestagoal.GoalRequiredTestFinalSnapshotV0, error) {
	adapter, err := selector.adapterForGoalV0(ctx, request.GoalRef)
	if err != nil {
		return orquestagoal.GoalRequiredTestFinalSnapshotV0{}, err
	}
	return adapter.CaptureGoalRequiredTestFinalSnapshotV0(ctx, request)
}

func (selector *serverGoalRequiredTestAttestationWorkspaceSelectorV0) AttestGoalRequiredTestsV0(
	ctx context.Context,
	request orquestagoal.GoalRequiredTestAttestationRequestV0,
) ([]orquestagoal.GoalRequiredTestAttestationV0, error) {
	adapter, err := selector.adapterForGoalV0(ctx, request.GoalRef)
	if err != nil {
		return nil, err
	}
	return adapter.AttestGoalRequiredTestsV0(ctx, request)
}

func (selector *serverGoalRequiredTestAttestationWorkspaceSelectorV0) VerifyGoalRequiredTestIdentityV0(
	ctx context.Context,
	request orquestagoal.GoalRequiredTestIdentityVerificationRequestV0,
) (orquestagoal.GoalRequiredTestIdentityVerificationV0, error) {
	if selector == nil || selector.Canonical == nil {
		return orquestagoal.GoalRequiredTestIdentityVerificationV0{}, fmt.Errorf("goal_required_test_attestation_adapter_unavailable")
	}
	return selector.Canonical.VerifyGoalRequiredTestIdentityV0(ctx, request)
}

func (selector *serverGoalRequiredTestAttestationWorkspaceSelectorV0) adapterForGoalV0(
	ctx context.Context,
	goalRef string,
) (*orquestaruntimerequiredtest.LocalGoalRequiredTestAttestationAdapterV0, error) {
	if selector == nil || selector.Canonical == nil {
		return nil, fmt.Errorf("goal_required_test_attestation_adapter_unavailable")
	}
	if selector.WorkspaceLookup == nil {
		return selector.Canonical, nil
	}
	found, err := selector.WorkspaceLookup.HasCodexGoalWorkspaceBindingV0(ctx, goalRef)
	if err != nil {
		return nil, err
	}
	if !found {
		return selector.Canonical, nil
	}
	binding, err := selector.WorkspaceLookup.ResolveCodexGoalWorkspaceV0(ctx, orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		GoalRef: strings.TrimSpace(goalRef),
	})
	if err != nil || strings.TrimSpace(binding.ProjectWorkDir) == "" {
		return nil, fmt.Errorf("autoprogramming_goal_workspace_attestation_unavailable")
	}
	workspaceDir := filepath.Clean(binding.ProjectWorkDir)
	workspaceConfig := selector.RuntimeConfig
	workspaceConfig.ProjectWorkDir = workspaceDir
	adapter, err := orquestaruntimerequiredtest.NewLocalGoalRequiredTestAttestationAdapterV0(workspaceConfig)
	if err != nil {
		return nil, err
	}
	if err := adapter.PreflightGoalRequiredTestAttestationV0(context.Background()); err != nil {
		return nil, fmt.Errorf("autoprogramming_goal_workspace_attestation_preflight_failed")
	}
	return adapter, nil
}

var _ orquestagoal.GoalRequiredTestSpecBinderPortV0 = (*serverGoalRequiredTestAttestationWorkspaceSelectorV0)(nil)
var _ orquestagoal.GoalRequiredTestFinalSnapshotObserverPortV0 = (*serverGoalRequiredTestAttestationWorkspaceSelectorV0)(nil)
var _ orquestagoal.GoalRequiredTestAttestorPortV0 = (*serverGoalRequiredTestAttestationWorkspaceSelectorV0)(nil)
var _ orquestagoal.GoalRequiredTestIdentityVerifierPortV0 = (*serverGoalRequiredTestAttestationWorkspaceSelectorV0)(nil)

func goalRequiredTestAttestationConfigFilePathV0(
	projectWorkDir string,
	projectConfig serverProjectConfigFileV0,
) (string, bool, error) {
	if value, ok := os.LookupEnv(envGoalRequiredTestAttestationConfigFileV0); ok {
		value = strings.TrimSpace(value)
		if value == "" {
			return "", false, fmt.Errorf("goal_required_test_attestation_config_file_required")
		}
		abs, err := filepath.Abs(value)
		if err != nil {
			return "", false, fmt.Errorf("goal_required_test_attestation_config_file_invalid")
		}
		return abs, true, nil
	}
	configured := projectConfig.Autoprogramming.RequiredTestAttestationConfigFile
	if configured == nil {
		return "", false, nil
	}
	value := strings.TrimSpace(*configured)
	if value == "" {
		return "", false, fmt.Errorf("goal_required_test_attestation_config_file_required")
	}
	if !filepath.IsAbs(value) {
		value = filepath.Join(strings.TrimSpace(projectWorkDir), value)
	}
	abs, err := filepath.Abs(value)
	if err != nil {
		return "", false, fmt.Errorf("goal_required_test_attestation_config_file_invalid")
	}
	return abs, true, nil
}

func rejectGoalRequiredTestAttestationConfigTrailingJSONV0(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return fmt.Errorf("goal_required_test_attestation_config_json_invalid: %w", err)
	}
	return fmt.Errorf("goal_required_test_attestation_config_json_multiple_values")
}
