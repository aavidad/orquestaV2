//go:build linux

package bubblewrap

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/sys/unix"
)

type pinnedMetadata struct {
	dev, ino, size          uint64
	mode, uid, gid          uint32
	changeSeconds, changeNS int64
}
type pinnedInputs struct {
	closeOnce               sync.Once
	closeErr                error
	bubblewrap, root, goBin *os.File
	bubblewrapMeta          pinnedMetadata
	treeDigest              string
	owner                   uint32
	identity                string
}

func openPinnedInputs(config Config, owner uint32) (*pinnedInputs, error) {
	inputs := &pinnedInputs{}
	fail := func(code string) (*pinnedInputs, error) {
		_ = inputs.Close()
		return nil, &Error{Code: code}
	}
	var err error
	inputs.bubblewrap, inputs.bubblewrapMeta, err = openPinned(config.BubblewrapCommand, owner, false, CodeBinaryUnsafe)
	if err != nil {
		return nil, err
	}
	bwrapDigest, err := pinnedDigest(inputs.bubblewrap, inputs.bubblewrapMeta)
	if err != nil {
		return fail(CodeBinaryUnsafe)
	}
	var rootMeta pinnedMetadata
	inputs.root, rootMeta, err = openPinned(config.ToolchainRoot, owner, true, CodeToolchainUnsafe)
	if err != nil {
		return fail(CodeToolchainUnsafe)
	}
	goSource, goMeta, err := openPinnedAt(inputs.root, "bin/go", owner)
	if err != nil {
		return fail(CodeToolchainUnsafe)
	}
	var goDigest string
	inputs.goBin, goDigest, err = sealExecutable(goSource, goMeta, "orquesta-go")
	_ = goSource.Close()
	if err != nil {
		return fail(CodeToolchainUnsafe)
	}
	treeDigest, err := trustedTreeIdentity(inputs.root, owner)
	if err != nil {
		return fail(CodeToolchainUnsafe)
	}
	inputs.treeDigest, inputs.owner = treeDigest, owner
	digest := sha256.New()
	for _, value := range []string{bwrapDigest, goDigest, treeDigest, fmt.Sprint(rootMeta.mode, rootMeta.uid, rootMeta.gid)} {
		writeDigestField(digest, value)
	}
	inputs.identity = hex.EncodeToString(digest.Sum(nil))
	return inputs, nil
}

func (inputs *pinnedInputs) validateToolchain() error {
	if inputs == nil || inputs.root == nil || inputs.treeDigest == "" {
		return &Error{Code: CodeToolchainUnsafe}
	}
	digest, err := trustedTreeIdentity(inputs.root, inputs.owner)
	if err != nil || digest != inputs.treeDigest {
		return &Error{Code: CodeToolchainUnsafe}
	}
	return nil
}

func pinnedDigest(source *os.File, before pinnedMetadata) (string, error) {
	return copyPinned(nil, source, before)
}
func copyPinned(output io.Writer, source *os.File, before pinnedMetadata) (string, error) {
	digest := sha256.New()
	writer := io.Writer(digest)
	if output != nil {
		writer = io.MultiWriter(output, digest)
	}
	written, err := io.Copy(writer, io.NewSectionReader(source, 0, int64(before.size)))
	after, statErr := validatePinned(source, before.uid, false)
	if err != nil || statErr != nil || uint64(written) != before.size || after != before {
		return "", errors.New("pin changed")
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}
func openPinned(name string, owner uint32, directory bool, code string) (*os.File, pinnedMetadata, error) {
	if name == "" || !filepath.IsAbs(name) || filepath.Clean(name) != name {
		return nil, pinnedMetadata{}, &Error{Code: code}
	}
	flags := uint64(unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK)
	if directory {
		flags |= unix.O_DIRECTORY
	}
	fd, err := unix.Openat2(unix.AT_FDCWD, name, &unix.OpenHow{Flags: flags, Resolve: unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS})
	if err != nil {
		return nil, pinnedMetadata{}, &Error{Code: code}
	}
	file := os.NewFile(uintptr(fd), "orquesta-pinned")
	metadata, err := validatePinned(file, owner, directory)
	if err != nil {
		_ = file.Close()
		return nil, pinnedMetadata{}, &Error{Code: code}
	}
	return file, metadata, nil
}
func openPinnedAt(root *os.File, name string, owner uint32) (*os.File, pinnedMetadata, error) {
	fd, err := unix.Openat2(int(root.Fd()), name, &unix.OpenHow{Flags: unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK, Resolve: unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS})
	if err != nil {
		return nil, pinnedMetadata{}, err
	}
	file := os.NewFile(uintptr(fd), "orquesta-pinned")
	metadata, err := validatePinned(file, owner, false)
	if err != nil {
		_ = file.Close()
	}
	return file, metadata, err
}
func validatePinned(file *os.File, owner uint32, directory bool) (pinnedMetadata, error) {
	kind := uint32(unix.S_IFREG)
	required := uint32(0)
	if directory {
		kind = unix.S_IFDIR
	}
	metadata, err := pinnedFileMetadata(file, owner, kind, required)
	if err == nil && !directory && metadata.mode&0o111 == 0 {
		err = errors.New("unsafe pin")
	}
	return metadata, err
}
func pinnedFileMetadata(file *os.File, owner, kind, required uint32) (pinnedMetadata, error) {
	var stat unix.Stat_t
	if file == nil || unix.Fstat(int(file.Fd()), &stat) != nil || stat.Uid != owner || stat.Mode&unix.S_IFMT != kind || stat.Mode&0o022 != 0 || stat.Mode&(unix.S_ISUID|unix.S_ISGID|unix.S_ISVTX) != 0 || stat.Mode&required != required {
		return pinnedMetadata{}, errors.New("unsafe pin")
	}
	return pinnedMetadata{uint64(stat.Dev), stat.Ino, uint64(stat.Size), stat.Mode, stat.Uid, stat.Gid, stat.Ctim.Sec, stat.Ctim.Nsec}, nil
}
func sealExecutable(source *os.File, before pinnedMetadata, name string) (*os.File, string, error) {
	fd, err := unix.MemfdCreate(name, unix.MFD_CLOEXEC|unix.MFD_ALLOW_SEALING|unix.MFD_EXEC)
	if err != nil {
		return nil, "", err
	}
	sealed := os.NewFile(uintptr(fd), name)
	digest, copyErr := copyPinned(sealed, source, before)
	seals := unix.F_SEAL_WRITE | unix.F_SEAL_GROW | unix.F_SEAL_SHRINK | unix.F_SEAL_EXEC | unix.F_SEAL_SEAL
	if copyErr != nil || unix.Fchmod(fd, 0o555) != nil {
		_ = sealed.Close()
		return nil, "", errors.New("pin changed")
	}
	if err = rewindAndSeal(sealed, seals); err != nil {
		_ = sealed.Close()
		return nil, "", err
	}
	return sealed, digest, nil
}

func reopenSealedExecutable(source *os.File) (*os.File, error) {
	if source == nil {
		return nil, errors.New("sealed executable unavailable")
	}
	fd, err := unix.Open(fmt.Sprintf("/proc/self/fd/%d", source.Fd()), unix.O_RDONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	reopened := os.NewFile(uintptr(fd), "orquesta-go-run")
	fail := func() (*os.File, error) {
		_ = reopened.Close()
		return nil, errors.New("sealed executable changed")
	}
	var sourceStat, reopenedStat unix.Stat_t
	if unix.Fstat(int(source.Fd()), &sourceStat) != nil || unix.Fstat(fd, &reopenedStat) != nil ||
		sourceStat.Dev != reopenedStat.Dev || sourceStat.Ino != reopenedStat.Ino ||
		sourceStat.Size != reopenedStat.Size || sourceStat.Mode != reopenedStat.Mode {
		return fail()
	}
	want := unix.F_SEAL_WRITE | unix.F_SEAL_GROW | unix.F_SEAL_SHRINK | unix.F_SEAL_EXEC | unix.F_SEAL_SEAL
	seals, sealErr := unix.FcntlInt(uintptr(fd), unix.F_GET_SEALS, 0)
	if sealErr != nil || seals&want != want {
		return fail()
	}
	return reopened, nil
}
func (inputs *pinnedInputs) Close() error {
	if inputs == nil {
		return nil
	}
	inputs.closeOnce.Do(func() {
		for _, file := range []*os.File{inputs.bubblewrap, inputs.root, inputs.goBin} {
			if file != nil {
				inputs.closeErr = errors.Join(inputs.closeErr, file.Close())
			}
		}
	})
	return inputs.closeErr
}
