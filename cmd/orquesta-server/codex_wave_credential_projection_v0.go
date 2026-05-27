package main

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const codexWaveCredentialProjectionReceiptSchemaV0 = "orquesta_codex_code_home_projection.v0"
const codexWaveProjectionDefaultMaxFilesV0 = 2048
const codexWaveProjectionDefaultMaxFileBytesV0 = 4 * 1024 * 1024
const codexWaveProjectionDefaultMaxTotalBytesV0 = 32 * 1024 * 1024

type codexWaveCredentialProjectionPolicyV0 struct {
	PolicyRef       string
	Strict          bool
	ProjectMemories bool
	MaxFiles        int
	MaxFileBytes    int64
	MaxTotalBytes   int64
}

type codexWaveCredentialProjectionReceiptV0 struct {
	SchemaVersion     string                                    `json:"schema_version"`
	ProjectionRef     string                                    `json:"projection_ref"`
	PolicyRef         string                                    `json:"policy_ref"`
	Mode              string                                    `json:"mode"`
	Categories        []string                                  `json:"categories,omitempty"`
	OmittedCategories []string                                  `json:"omitted_categories,omitempty"`
	RequiredMissing   []string                                  `json:"required_missing,omitempty"`
	FilesCopied       int                                       `json:"files_copied,omitempty"`
	BytesCopied       int64                                     `json:"bytes_copied,omitempty"`
	Omissions         []codexWaveCredentialProjectionOmissionV0 `json:"omissions,omitempty"`
	SourceRef         string                                    `json:"source_ref"`
}

type codexWaveCredentialProjectionOmissionV0 struct {
	Category string `json:"category"`
	Reason   string `json:"reason"`
	Count    int    `json:"count"`
}

type codexWaveCredentialProjectionItemV0 struct {
	Name             string
	Category         string
	Directory        bool
	Allowed          bool
	RequiredInStrict bool
}

func codexWaveCredentialProjectionPolicyV0FromFlags(strict bool, projectMemories bool) codexWaveCredentialProjectionPolicyV0 {
	return codexWaveCredentialProjectionPolicyWithBoundsV0(
		strict,
		projectMemories,
		codexWaveProjectionDefaultMaxFilesV0,
		codexWaveProjectionDefaultMaxFileBytesV0,
		codexWaveProjectionDefaultMaxTotalBytesV0,
	)
}

func codexWaveCredentialProjectionPolicyWithBoundsV0(
	strict bool,
	projectMemories bool,
	maxFiles int,
	maxFileBytes int64,
	maxTotalBytes int64,
) codexWaveCredentialProjectionPolicyV0 {
	return codexWaveCredentialProjectionPolicyV0{
		PolicyRef:       "codex-code-home-projection-policy-v0",
		Strict:          strict,
		ProjectMemories: projectMemories,
		MaxFiles:        codexWaveProjectionPositiveIntV0(maxFiles, codexWaveProjectionDefaultMaxFilesV0),
		MaxFileBytes:    codexWaveProjectionPositiveInt64V0(maxFileBytes, codexWaveProjectionDefaultMaxFileBytesV0),
		MaxTotalBytes:   codexWaveProjectionPositiveInt64V0(maxTotalBytes, codexWaveProjectionDefaultMaxTotalBytesV0),
	}
}

func codexWaveProjectionPositiveIntV0(value int, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func codexWaveProjectionPositiveInt64V0(value int64, fallback int64) int64 {
	if value <= 0 {
		return fallback
	}
	return value
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

func codexWaveProjectionRecordOmissionV0(
	receipt *codexWaveCredentialProjectionReceiptV0,
	category string,
	reason string,
) {
	category = strings.TrimSpace(category)
	reason = strings.TrimSpace(reason)
	if receipt == nil || category == "" || reason == "" {
		return
	}
	receipt.OmittedCategories = codexWaveProjectionRecordCategoryV0(receipt.OmittedCategories, category)
	for i := range receipt.Omissions {
		if receipt.Omissions[i].Category == category && receipt.Omissions[i].Reason == reason {
			receipt.Omissions[i].Count++
			return
		}
	}
	receipt.Omissions = append(receipt.Omissions, codexWaveCredentialProjectionOmissionV0{
		Category: category,
		Reason:   reason,
		Count:    1,
	})
	sort.Slice(receipt.Omissions, func(i, j int) bool {
		if receipt.Omissions[i].Category == receipt.Omissions[j].Category {
			return receipt.Omissions[i].Reason < receipt.Omissions[j].Reason
		}
		return receipt.Omissions[i].Category < receipt.Omissions[j].Category
	})
}

func codexWaveProjectionSourceExistsV0(path string, _ bool) (bool, error) {
	_, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
