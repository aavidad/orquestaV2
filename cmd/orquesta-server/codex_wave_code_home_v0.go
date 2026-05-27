package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
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
	if info, err := os.Lstat(source); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
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
				codexWaveProjectionRecordOmissionV0(&receipt, item.Category, "category_not_allowed")
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
			err = codexWaveCopyDirIfExistsV0(filepath.Join(source, item.Name), filepath.Join(codeHomeDir, item.Name), item.Category, &receipt, policy)
		} else {
			err = codexWaveCopyFileIfExistsV0(filepath.Join(source, item.Name), filepath.Join(codeHomeDir, item.Name), item.Category, &receipt, policy)
		}
		if err != nil {
			return "", "", nil, err
		}
		if codexWaveProjectionCategoryCopiedV0(receipt, item.Category) {
			receipt.Categories = codexWaveProjectionRecordCategoryV0(receipt.Categories, item.Category)
		}
		if policy.Strict && item.RequiredInStrict && !codexWaveProjectionCategoryCopiedV0(receipt, item.Category) {
			receipt.RequiredMissing = codexWaveProjectionRecordCategoryV0(receipt.RequiredMissing, item.Category)
		}
	}
	if len(receipt.RequiredMissing) > 0 {
		return "", "", &receipt, fmt.Errorf("credential_projection_missing_required:%s", receipt.RequiredMissing[0])
	}
	return homeDir, codeHomeDir, &receipt, nil
}

func codexWaveCopyFileIfExistsV0(
	source string,
	target string,
	category string,
	receipt *codexWaveCredentialProjectionReceiptV0,
	policy codexWaveCredentialProjectionPolicyV0,
) error {
	info, err := os.Lstat(source)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !codexWaveProjectionFileAllowedV0(info, category, receipt, policy) {
		return nil
	}
	data, err := codexWaveReadRegularNoFollowV0(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(target, data, 0o600); err != nil {
		return err
	}
	codexWaveProjectionRecordCopiedV0(receipt, category, info.Size())
	return nil
}

func codexWaveCopyDirIfExistsV0(
	source string,
	target string,
	category string,
	receipt *codexWaveCredentialProjectionReceiptV0,
	policy codexWaveCredentialProjectionPolicyV0,
) error {
	info, err := os.Lstat(source)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		codexWaveProjectionRecordOmissionV0(receipt, category, "symlink")
		return nil
	}
	if !info.IsDir() {
		codexWaveProjectionRecordOmissionV0(receipt, category, "not_directory")
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
		if !codexWaveProjectionFileAllowedV0(info, category, receipt, policy) {
			return nil
		}
		data, err := codexWaveReadRegularNoFollowV0(path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(targetPath, data, 0o600); err != nil {
			return err
		}
		codexWaveProjectionRecordCopiedV0(receipt, category, info.Size())
		return nil
	})
}

func codexWaveProjectionFileAllowedV0(
	info os.FileInfo,
	category string,
	receipt *codexWaveCredentialProjectionReceiptV0,
	policy codexWaveCredentialProjectionPolicyV0,
) bool {
	mode := info.Mode()
	switch {
	case mode&os.ModeSymlink != 0:
		codexWaveProjectionRecordOmissionV0(receipt, category, "symlink")
	case !mode.IsRegular():
		codexWaveProjectionRecordOmissionV0(receipt, category, "non_regular")
	case codexWaveProjectionHardlinkedV0(info):
		codexWaveProjectionRecordOmissionV0(receipt, category, "hardlink")
	case policy.MaxFiles > 0 && receipt != nil && receipt.FilesCopied >= policy.MaxFiles:
		codexWaveProjectionRecordOmissionV0(receipt, category, "max_files_exceeded")
	case policy.MaxFileBytes > 0 && info.Size() > policy.MaxFileBytes:
		codexWaveProjectionRecordOmissionV0(receipt, category, "max_file_bytes_exceeded")
	case policy.MaxTotalBytes > 0 && receipt != nil && receipt.BytesCopied+info.Size() > policy.MaxTotalBytes:
		codexWaveProjectionRecordOmissionV0(receipt, category, "max_total_bytes_exceeded")
	default:
		return true
	}
	return false
}

func codexWaveProjectionRecordCopiedV0(receipt *codexWaveCredentialProjectionReceiptV0, category string, size int64) {
	if receipt == nil {
		return
	}
	receipt.Categories = codexWaveProjectionRecordCategoryV0(receipt.Categories, category)
	receipt.FilesCopied++
	receipt.BytesCopied += size
}

func codexWaveProjectionCategoryCopiedV0(receipt codexWaveCredentialProjectionReceiptV0, category string) bool {
	for _, value := range receipt.Categories {
		if value == category {
			return true
		}
	}
	return false
}

func codexWaveProjectionHardlinkedV0(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Nlink > 1
}

func codexWaveReadRegularNoFollowV0(path string) ([]byte, error) {
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), filepath.Base(path))
	defer file.Close()
	return io.ReadAll(file)
}
