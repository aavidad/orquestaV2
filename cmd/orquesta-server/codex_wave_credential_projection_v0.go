package main

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const codexWaveCredentialProjectionReceiptSchemaV0 = "orquesta_codex_code_home_projection.v0"

type codexWaveCredentialProjectionPolicyV0 struct {
	PolicyRef       string
	Strict          bool
	ProjectMemories bool
}

type codexWaveCredentialProjectionReceiptV0 struct {
	SchemaVersion     string   `json:"schema_version"`
	ProjectionRef     string   `json:"projection_ref"`
	PolicyRef         string   `json:"policy_ref"`
	Mode              string   `json:"mode"`
	Categories        []string `json:"categories,omitempty"`
	OmittedCategories []string `json:"omitted_categories,omitempty"`
	RequiredMissing   []string `json:"required_missing,omitempty"`
	SourceRef         string   `json:"source_ref"`
}

type codexWaveCredentialProjectionItemV0 struct {
	Name             string
	Category         string
	Directory        bool
	Allowed          bool
	RequiredInStrict bool
}

func codexWaveCredentialProjectionPolicyV0FromFlags(strict bool, projectMemories bool) codexWaveCredentialProjectionPolicyV0 {
	return codexWaveCredentialProjectionPolicyV0{
		PolicyRef:       "codex-code-home-projection-policy-v0",
		Strict:          strict,
		ProjectMemories: projectMemories,
	}
}

func codexWaveCredentialProjectionItemsV0(policy codexWaveCredentialProjectionPolicyV0) []codexWaveCredentialProjectionItemV0 {
	return []codexWaveCredentialProjectionItemV0{
		{Name: "auth.json", Category: "auth", RequiredInStrict: true, Allowed: true},
		{Name: "config.toml", Category: "config", RequiredInStrict: true, Allowed: true},
		{Name: "instructions.md", Category: "instructions", Allowed: true},
		{Name: "AGENTS.md", Category: "instructions", Allowed: true},
		{Name: "installation_id", Category: "runtime_config", Allowed: true},
		{Name: "version.json", Category: "runtime_config", Allowed: true},
		{Name: "models_cache.json", Category: "runtime_config", Allowed: true},
		{Name: ".personality_migration", Category: "runtime_config", Allowed: true},
		{Name: "skills", Category: "skills", Directory: true, Allowed: true},
		{Name: "plugins", Category: "plugins", Directory: true, Allowed: true},
		{Name: "rules", Category: "rules", Directory: true, Allowed: true},
		{Name: "memories", Category: "memories", Directory: true, Allowed: policy.ProjectMemories},
	}
}

func codexWaveProjectionReceiptV0(policy codexWaveCredentialProjectionPolicyV0, agentRuntimeDir string) codexWaveCredentialProjectionReceiptV0 {
	return codexWaveCredentialProjectionReceiptV0{
		SchemaVersion: codexWaveCredentialProjectionReceiptSchemaV0,
		ProjectionRef: "codex-code-home-projection-ref-" + filepath.Base(filepath.Clean(agentRuntimeDir)),
		PolicyRef:     policy.PolicyRef,
		Mode:          "isolated_agent_home",
		SourceRef:     "codex-code-home-source-ref",
	}
}

func codexWaveValidateProjectionPolicyV0(config codexWaveConfigV0) error {
	if config.CredentialProjectionPolicy.Strict && !config.IsolateHome {
		return errors.New("credential_projection_requires_isolated_home")
	}
	return nil
}

func codexWaveProjectionRecordCategoryV0(values []string, category string) []string {
	category = strings.TrimSpace(category)
	if category == "" {
		return values
	}
	for _, value := range values {
		if value == category {
			return values
		}
	}
	values = append(values, category)
	sort.Strings(values)
	return values
}

func codexWaveProjectionSourceExistsV0(path string, directory bool) (bool, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if directory {
		return info.IsDir(), nil
	}
	return info.Mode().IsRegular(), nil
}
