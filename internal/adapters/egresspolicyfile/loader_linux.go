//go:build linux

package egresspolicyfile

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"

	"golang.org/x/sys/unix"
)

type loaderHooks struct {
	afterFirstIdentity func()
	afterFirstRead     func()
}

type policyFileIdentity struct {
	device, inode, links uint64
	size                 int64
	mode, uid, gid       uint32
	modified, changed    unix.Timespec
}

func loadPinnedPolicy(
	path string,
	expectedSHA256 string,
	ownerUID uint32,
	maximumBytes int64,
	hooks loaderHooks,
) ([]byte, error) {
	first, err := openPolicyFile(path)
	if err != nil {
		return nil, &Error{Code: CodeFileUnsafe}
	}
	defer first.Close()
	firstBefore, err := inspectPolicyFile(first, ownerUID, maximumBytes)
	if err != nil {
		return nil, err
	}
	if hooks.afterFirstIdentity != nil {
		hooks.afterFirstIdentity()
	}
	content, digest, firstAfter, err := readPolicyFile(first, ownerUID, maximumBytes)
	if err != nil || firstBefore != firstAfter {
		clear(content)
		return nil, &Error{Code: CodeFileUnsafe}
	}
	if digest != expectedSHA256 {
		clear(content)
		return nil, &Error{Code: CodeDigestMismatch}
	}
	if hooks.afterFirstRead != nil {
		hooks.afterFirstRead()
	}

	// Reopen the pathname after reading. Fstat on the first descriptor alone
	// would not notice an unlink-and-replace race because that descriptor keeps
	// referring to the old inode.
	second, err := openPolicyFile(path)
	if err != nil {
		clear(content)
		return nil, &Error{Code: CodeFileUnsafe}
	}
	defer second.Close()
	secondBefore, err := inspectPolicyFile(second, ownerUID, maximumBytes)
	if err != nil || secondBefore != firstAfter {
		clear(content)
		return nil, &Error{Code: CodeFileUnsafe}
	}
	confirmation, confirmationDigest, secondAfter, err := readPolicyFile(second, ownerUID, maximumBytes)
	if err != nil || secondBefore != secondAfter || confirmationDigest != expectedSHA256 ||
		!bytes.Equal(content, confirmation) {
		clear(content)
		clear(confirmation)
		if err == nil && confirmationDigest != expectedSHA256 {
			return nil, &Error{Code: CodeDigestMismatch}
		}
		return nil, &Error{Code: CodeFileUnsafe}
	}
	clear(confirmation)
	return content, nil
}

func openPolicyFile(path string) (*os.File, error) {
	descriptor, err := unix.Openat2(unix.AT_FDCWD, path, &unix.OpenHow{
		Flags:   unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK,
		Resolve: unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS,
	})
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(descriptor), "orquesta-egress-policy"), nil
}

func inspectPolicyFile(file *os.File, ownerUID uint32, maximumBytes int64) (policyFileIdentity, error) {
	if file == nil {
		return policyFileIdentity{}, &Error{Code: CodeFileUnsafe}
	}
	var stat unix.Stat_t
	if unix.Fstat(int(file.Fd()), &stat) != nil || stat.Mode&unix.S_IFMT != unix.S_IFREG ||
		stat.Uid != ownerUID || stat.Nlink != 1 || stat.Mode&0o022 != 0 ||
		stat.Size <= 0 || stat.Size > maximumBytes {
		return policyFileIdentity{}, &Error{Code: CodeFileUnsafe}
	}
	return policyFileIdentity{
		device: uint64(stat.Dev), inode: uint64(stat.Ino), links: uint64(stat.Nlink),
		size: stat.Size, mode: stat.Mode, uid: stat.Uid, gid: stat.Gid,
		modified: stat.Mtim, changed: stat.Ctim,
	}, nil
}

func readPolicyFile(
	file *os.File,
	ownerUID uint32,
	maximumBytes int64,
) ([]byte, string, policyFileIdentity, error) {
	before, err := inspectPolicyFile(file, ownerUID, maximumBytes)
	if err != nil {
		return nil, "", policyFileIdentity{}, err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, "", policyFileIdentity{}, &Error{Code: CodeFileUnsafe}
	}
	content, err := io.ReadAll(io.LimitReader(file, maximumBytes+1))
	if err != nil || int64(len(content)) != before.size || int64(len(content)) > maximumBytes {
		clear(content)
		return nil, "", policyFileIdentity{}, &Error{Code: CodeFileUnsafe}
	}
	after, err := inspectPolicyFile(file, ownerUID, maximumBytes)
	if err != nil || before != after {
		clear(content)
		return nil, "", policyFileIdentity{}, &Error{Code: CodeFileUnsafe}
	}
	digest := sha256.Sum256(content)
	return content, hex.EncodeToString(digest[:]), after, nil
}
