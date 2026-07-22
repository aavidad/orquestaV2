//go:build linux

package bubblewrap

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"hash"
	"io"
	"os"
	"sort"
	"strconv"

	"golang.org/x/sys/unix"
)

func trustedTreeIdentity(root *os.File, owner uint32) (string, error) {
	if root == nil {
		return "", errors.New("trusted tree unavailable")
	}
	fd, err := unix.Openat(int(root.Fd()), ".", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return "", err
	}
	directory := os.NewFile(uintptr(fd), "orquesta-trusted-tree")
	defer directory.Close()
	digest := sha256.New()
	writeDigestField(digest, "orquesta.trusted-toolchain-tree.v1")
	if err := hashTrustedDirectory(digest, directory, ".", owner); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func hashTrustedDirectory(digest hash.Hash, directory *os.File, prefix string, owner uint32) error {
	before, err := trustedTreeMetadata(directory, owner, true)
	if err != nil {
		return err
	}
	writeTrustedMetadata(digest, prefix, "directory", before)
	entries, err := directory.ReadDir(-1)
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		name := entry.Name()
		if name == "" || name == "." || name == ".." {
			return errors.New("trusted tree name invalid")
		}
		fd, openErr := unix.Openat2(int(directory.Fd()), name, &unix.OpenHow{Flags: unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK, Resolve: unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS})
		if openErr != nil {
			return openErr
		}
		child := os.NewFile(uintptr(fd), "orquesta-trusted-tree-entry")
		childPath := name
		if prefix != "." {
			childPath = prefix + "/" + name
		}
		var stat unix.Stat_t
		if unix.Fstat(fd, &stat) != nil {
			err = errors.New("trusted tree stat failed")
		} else if stat.Mode&unix.S_IFMT == unix.S_IFDIR {
			err = hashTrustedDirectory(digest, child, childPath, owner)
		} else if stat.Mode&unix.S_IFMT == unix.S_IFREG {
			var metadata pinnedMetadata
			metadata, err = trustedTreeMetadata(child, owner, false)
			if err == nil {
				writeTrustedMetadata(digest, childPath, "file", metadata)
				var copied int64
				copied, err = io.CopyN(digest, child, int64(metadata.size))
				if err == nil && copied != int64(metadata.size) {
					err = errors.New("trusted tree read failed")
				}
				if after, checkErr := trustedTreeMetadata(child, owner, false); err == nil && (checkErr != nil || after != metadata) {
					err = errors.New("trusted tree changed")
				}
			}
		} else {
			err = errors.New("trusted tree type invalid")
		}
		err = errors.Join(err, child.Close())
		if err != nil {
			return err
		}
	}
	after, err := trustedTreeMetadata(directory, owner, true)
	if err != nil || after != before {
		return errors.New("trusted tree changed")
	}
	return nil
}

func trustedTreeMetadata(file *os.File, owner uint32, directory bool) (pinnedMetadata, error) {
	kind, required := uint32(unix.S_IFREG), uint32(0o400)
	if directory {
		kind, required = unix.S_IFDIR, 0o500
	}
	return pinnedFileMetadata(file, owner, kind, required)
}

func writeTrustedMetadata(digest hash.Hash, path, kind string, metadata pinnedMetadata) {
	for _, value := range []string{path, kind, strconv.FormatUint(uint64(metadata.mode&0o7777), 8), strconv.FormatUint(uint64(metadata.uid), 10), strconv.FormatUint(uint64(metadata.gid), 10), strconv.FormatUint(metadata.size, 10)} {
		writeDigestField(digest, value)
	}
}
