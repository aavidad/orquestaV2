package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"hash"
	"io"
	"os"
	"path/filepath"
)

func copyGuardianArtifactAtomicV0(
	config guardianConfigV0,
	src string,
	dst string,
	mode os.FileMode,
) (guardianArtifactCopyResultV0, error) {
	if err := validateGuardianArtifactPathV0(src); err != nil {
		return guardianArtifactCopyResultV0{}, err
	}
	if err := validateGuardianArtifactPathV0(dst); err != nil {
		return guardianArtifactCopyResultV0{}, err
	}
	if err := validateGuardianArtifactRootV0(config, src); err != nil {
		return guardianArtifactCopyResultV0{}, err
	}
	if err := validateGuardianArtifactRootV0(config, dst); err != nil {
		return guardianArtifactCopyResultV0{}, err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return guardianArtifactCopyResultV0{}, guardianArtifactPolicyErrorV0{Code: "guardian_artifact_mkdir_failed", Err: err}
	}
	if err := rejectGuardianSymlinkPathV0(src); err != nil {
		return guardianArtifactCopyResultV0{}, err
	}
	if err := rejectGuardianSymlinkPathV0(filepath.Dir(dst)); err != nil {
		return guardianArtifactCopyResultV0{}, err
	}
	if err := validateGuardianDestinationV0(dst); err != nil {
		return guardianArtifactCopyResultV0{}, err
	}
	source, err := openGuardianArtifactSourceV0(config, src)
	if err != nil {
		return guardianArtifactCopyResultV0{}, err
	}
	defer source.Close()
	tmp, err := os.CreateTemp(filepath.Dir(dst), filepath.Base(dst)+".tmp-*")
	if err != nil {
		return guardianArtifactCopyResultV0{}, guardianArtifactPolicyErrorV0{Code: "guardian_artifact_tmp_failed", Err: err}
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		_ = tmp.Close()
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()
	if err := tmp.Chmod(mode); err != nil {
		return guardianArtifactCopyResultV0{}, guardianArtifactPolicyErrorV0{Code: "guardian_artifact_chmod_failed", Err: err}
	}
	hashWriter := sha256.New()
	size, err := copyGuardianArtifactStreamV0(tmp, hashWriter, source, config.ArtifactMaxBytes)
	if err != nil {
		return guardianArtifactCopyResultV0{}, err
	}
	if err := source.VerifyStable(); err != nil {
		return guardianArtifactCopyResultV0{}, err
	}
	if err := tmp.Sync(); err != nil {
		return guardianArtifactCopyResultV0{}, guardianArtifactPolicyErrorV0{Code: "guardian_artifact_fsync_failed", Err: err}
	}
	if err := tmp.Close(); err != nil {
		return guardianArtifactCopyResultV0{}, guardianArtifactPolicyErrorV0{Code: "guardian_artifact_close_failed", Err: err}
	}
	if err := os.Rename(tmpPath, dst); err != nil {
		return guardianArtifactCopyResultV0{}, guardianArtifactPolicyErrorV0{Code: "guardian_artifact_rename_failed", Err: err}
	}
	cleanup = false
	_ = fsyncGuardianDirV0(filepath.Dir(dst))
	destDigest, err := inspectGuardianArtifactV0(config, dst)
	if err != nil {
		return guardianArtifactCopyResultV0{}, err
	}
	return guardianArtifactCopyResultV0{
		Source: guardianArtifactDigestV0{
			Ref:       guardianPathRefV0("guardian-artifact-ref", src),
			SizeBytes: size,
			SHA256:    hex.EncodeToString(hashWriter.Sum(nil)),
			Mode:      source.Mode().String(),
		},
		Dest: destDigest,
	}, nil
}

type guardianArtifactSourceV0 struct {
	file *os.File
	info os.FileInfo
}

func (source guardianArtifactSourceV0) Close() error      { return source.file.Close() }
func (source guardianArtifactSourceV0) Mode() os.FileMode { return source.info.Mode() }

func (source guardianArtifactSourceV0) Read(p []byte) (int, error) {
	return source.file.Read(p)
}

func (source guardianArtifactSourceV0) VerifyStable() error {
	after, err := source.file.Stat()
	if err != nil {
		return guardianArtifactPolicyErrorV0{Code: "guardian_artifact_post_stat_failed", Err: err}
	}
	if !sameGuardianFileIdentityV0(source.info, after) {
		return guardianArtifactPolicyErrorV0{Code: "guardian_artifact_changed_during_copy", Err: nil}
	}
	return nil
}

func openGuardianArtifactSourceV0(config guardianConfigV0, path string) (guardianArtifactSourceV0, error) {
	if err := validateGuardianArtifactRootV0(config, path); err != nil {
		return guardianArtifactSourceV0{}, err
	}
	before, err := inspectGuardianArtifactInfoV0(path)
	if err != nil {
		return guardianArtifactSourceV0{}, err
	}
	if before.Size() > config.ArtifactMaxBytes {
		return guardianArtifactSourceV0{}, guardianArtifactPolicyErrorV0{Code: "guardian_artifact_size_exceeded", Err: nil}
	}
	file, err := os.Open(path)
	if err != nil {
		return guardianArtifactSourceV0{}, guardianArtifactPolicyErrorV0{Code: "guardian_artifact_open_failed", Err: err}
	}
	after, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return guardianArtifactSourceV0{}, guardianArtifactPolicyErrorV0{Code: "guardian_artifact_open_stat_failed", Err: err}
	}
	if !sameGuardianFileIdentityV0(before, after) {
		_ = file.Close()
		return guardianArtifactSourceV0{}, guardianArtifactPolicyErrorV0{Code: "guardian_artifact_changed_before_open", Err: nil}
	}
	return guardianArtifactSourceV0{file: file, info: before}, nil
}

func inspectGuardianArtifactV0(config guardianConfigV0, path string) (guardianArtifactDigestV0, error) {
	source, err := openGuardianArtifactSourceV0(config, path)
	if err != nil {
		return guardianArtifactDigestV0{}, err
	}
	defer source.Close()
	hashWriter := sha256.New()
	size, err := copyGuardianArtifactStreamV0(io.Discard, hashWriter, source, config.ArtifactMaxBytes)
	if err != nil {
		return guardianArtifactDigestV0{}, err
	}
	if err := source.VerifyStable(); err != nil {
		return guardianArtifactDigestV0{}, err
	}
	return guardianArtifactDigestV0{
		Ref:       guardianPathRefV0("guardian-artifact-ref", path),
		SizeBytes: size,
		SHA256:    hex.EncodeToString(hashWriter.Sum(nil)),
		Mode:      source.Mode().String(),
	}, nil
}

func copyGuardianArtifactStreamV0(
	dst io.Writer,
	hashWriter hash.Hash,
	src io.Reader,
	maxBytes int64,
) (int64, error) {
	written, err := io.Copy(io.MultiWriter(dst, hashWriter), io.LimitReader(src, maxBytes+1))
	if err != nil {
		return written, guardianArtifactPolicyErrorV0{Code: "guardian_artifact_copy_failed", Err: err}
	}
	if written > maxBytes {
		return written, guardianArtifactPolicyErrorV0{Code: "guardian_artifact_size_exceeded", Err: nil}
	}
	return written, nil
}

func inspectGuardianArtifactInfoV0(path string) (os.FileInfo, error) {
	if err := rejectGuardianSymlinkPathV0(path); err != nil {
		return nil, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, guardianArtifactPolicyErrorV0{Code: "guardian_artifact_symlink_blocked", Err: nil}
	}
	if !info.Mode().IsRegular() {
		return nil, guardianArtifactPolicyErrorV0{Code: "guardian_artifact_not_regular", Err: nil}
	}
	if hasUnsafeHardlinksV0(info) {
		return nil, guardianArtifactPolicyErrorV0{Code: "guardian_artifact_hardlink_blocked", Err: nil}
	}
	return info, nil
}

func validateGuardianDestinationV0(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return guardianArtifactPolicyErrorV0{Code: "guardian_artifact_dest_symlink_blocked", Err: nil}
	}
	if !info.Mode().IsRegular() {
		return guardianArtifactPolicyErrorV0{Code: "guardian_artifact_dest_not_regular", Err: nil}
	}
	if hasUnsafeHardlinksV0(info) {
		return guardianArtifactPolicyErrorV0{Code: "guardian_artifact_dest_hardlink_blocked", Err: nil}
	}
	return nil
}

func sameGuardianFileIdentityV0(a os.FileInfo, b os.FileInfo) bool {
	if a == nil || b == nil {
		return false
	}
	return os.SameFile(a, b) &&
		a.Size() == b.Size() &&
		a.Mode() == b.Mode() &&
		a.ModTime().Equal(b.ModTime())
}

func fsyncGuardianDirV0(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
