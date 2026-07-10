package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	orquestaruntimerequiredtest "orquesta/modulos/orquesta-runtime-required-test"
	orquestaserver "orquesta/modulos/orquesta-server"
)

const goalRequiredTestAttestationConfigSchemaV0 = "orquesta_goal_required_test_attestation_config.v0"

type serverGoalRequiredTestAttestationConfigFileV0 struct {
	SchemaVersion     string                                          `json:"schema_version"`
	ProjectWorkDir    string                                          `json:"project_work_dir"`
	RuntimeRoot       string                                          `json:"runtime_root"`
	GitCommandPath    string                                          `json:"git_command_path"`
	AllowedCommands   map[string]string                               `json:"allowed_commands"`
	MaxRuntimeSeconds int                                             `json:"max_runtime_seconds"`
	MaxOutputBytes    int64                                           `json:"max_output_bytes"`
	MaxArtifacts      int                                             `json:"max_artifacts"`
	Identity          serverGoalRequiredTestAttestationIdentityFileV0 `json:"identity"`
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
	path, configured, err := goalRequiredTestAttestationConfigFilePathV0(serverConfig.ProjectWorkDir, projectConfig)
	if err != nil || !configured {
		return nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("goal_required_test_attestation_config_read: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("goal_required_test_attestation_config_not_owner_only")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("goal_required_test_attestation_config_read: %w", err)
	}
	if len(raw) == 0 || len(raw) > 1024*1024 {
		return nil, fmt.Errorf("goal_required_test_attestation_config_size_invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var document serverGoalRequiredTestAttestationConfigFileV0
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("goal_required_test_attestation_config_json_invalid: %w", err)
	}
	if err := rejectGoalRequiredTestAttestationConfigTrailingJSONV0(decoder); err != nil {
		return nil, err
	}
	if strings.TrimSpace(document.SchemaVersion) != goalRequiredTestAttestationConfigSchemaV0 {
		return nil, fmt.Errorf("goal_required_test_attestation_config_schema_invalid")
	}
	configuredProject, err := filepath.Abs(strings.TrimSpace(document.ProjectWorkDir))
	if err != nil {
		return nil, fmt.Errorf("goal_required_test_attestation_project_work_dir_invalid")
	}
	serverProject, err := filepath.Abs(strings.TrimSpace(serverConfig.ProjectWorkDir))
	if err != nil {
		return nil, fmt.Errorf("goal_required_test_attestation_project_work_dir_invalid")
	}
	configuredProject, configuredErr := filepath.EvalSymlinks(configuredProject)
	serverProject, serverErr := filepath.EvalSymlinks(serverProject)
	if configuredErr != nil || serverErr != nil || configuredProject != serverProject {
		return nil, fmt.Errorf("goal_required_test_attestation_project_work_dir_mismatch")
	}
	digest := sha256.Sum256(raw)
	identity := document.Identity
	return orquestaruntimerequiredtest.NewLocalGoalRequiredTestAttestationAdapterV0(
		orquestaruntimerequiredtest.LocalGoalRequiredTestAttestationConfigV0{
			ProjectWorkDir: configuredProject,
			RuntimeRoot:    strings.TrimSpace(document.RuntimeRoot), GitCommandPath: strings.TrimSpace(document.GitCommandPath),
			AllowedCommands: document.AllowedCommands,
			MaxRuntime:      time.Duration(document.MaxRuntimeSeconds) * time.Second,
			MaxOutputBytes:  document.MaxOutputBytes, MaxArtifacts: document.MaxArtifacts,
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
		},
	)
}

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
