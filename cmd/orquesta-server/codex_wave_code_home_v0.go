package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func codexWaveCopyCodeHomeV0(
	sourceCodeHome string,
	agentRuntimeDir string,
	policy codexWaveCredentialProjectionPolicyV0,
) (string, string, *codexWaveCredentialProjectionReceiptV0, error) {
	source := filepath.Clean(sourceCodeHome)
	if source == "" || !filepath.IsAbs(source) {
		return "", "", nil, errors.New("source_code_home_invalid")
	}
	if info, err := os.Stat(source); err != nil || !info.IsDir() {
		return "", "", nil, errors.New("source_code_home_unavailable")
	}
	homeDir := filepath.Join(agentRuntimeDir, "home")
	codeHomeDir := filepath.Join(homeDir, ".codex")
	if err := os.MkdirAll(codeHomeDir, 0o700); err != nil {
		return "", "", nil, err
	}
	receipt := codexWaveProjectionReceiptV0(policy, agentRuntimeDir)
	for _, item := range codexWaveCredentialProjectionItemsV0(policy) {
		exists, err := codexWaveProjectionSourceExistsV0(filepath.Join(source, item.Name), item.Directory)
		if err != nil {
			return "", "", nil, err
		}
		if !item.Allowed {
			if exists {
				receipt.OmittedCategories = codexWaveProjectionRecordCategoryV0(receipt.OmittedCategories, item.Category)
			}
			continue
		}
		if !exists {
			if policy.Strict && item.RequiredInStrict {
				receipt.RequiredMissing = codexWaveProjectionRecordCategoryV0(receipt.RequiredMissing, item.Category)
			}
			continue
		}
		if item.Directory {
			err = codexWaveCopyDirIfExistsV0(filepath.Join(source, item.Name), filepath.Join(codeHomeDir, item.Name))
		} else {
			err = codexWaveCopyFileIfExistsV0(filepath.Join(source, item.Name), filepath.Join(codeHomeDir, item.Name))
		}
		if err != nil {
			return "", "", nil, err
		}
		receipt.Categories = codexWaveProjectionRecordCategoryV0(receipt.Categories, item.Category)
	}
	if len(receipt.RequiredMissing) > 0 {
		return "", "", &receipt, fmt.Errorf("credential_projection_missing_required:%s", receipt.RequiredMissing[0])
	}
	return homeDir, codeHomeDir, &receipt, nil
}

func codexWaveCopyFileIfExistsV0(source string, target string) error {
	info, err := os.Stat(source)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.IsDir() {
		return nil
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return err
	}
	return os.WriteFile(target, data, 0o600)
}

func codexWaveCopyDirIfExistsV0(source string, target string) error {
	info, err := os.Stat(source)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return nil
	}
	if err := os.MkdirAll(target, 0o700); err != nil {
		return err
	}
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		targetPath := filepath.Join(target, rel)
		if entry.IsDir() {
			return os.MkdirAll(targetPath, 0o700)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		mode := info.Mode().Perm()
		if mode == 0 {
			mode = 0o600
		}
		return os.WriteFile(targetPath, data, mode)
	})
}
