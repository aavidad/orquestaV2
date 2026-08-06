//go:build linux

package filesource

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"

	"golang.org/x/sys/unix"

	"orquesta/internal/credentials"
)

type fileIdentity struct {
	device, inode, links uint64
	size                 int64
	mode, uid, gid       uint32
	modifiedSec          int64
	modifiedNsec         int64
	changedSec           int64
	changedNsec          int64
}

// WithSecret securely reopens and compares the source before invoking callback
// exactly once. All adapter-owned plaintext is cleared before return.
func (source *Source) WithSecret(ctx context.Context, callback func(credentials.Secret) error) error {
	if source == nil || nilContext(ctx) || callback == nil {
		return credentials.NewError(credentials.ErrorInvalidRequest, "material_source")
	}
	if err := ctx.Err(); err != nil {
		return credentials.WrapError(credentials.ErrorConsumerFailed, "context", err)
	}

	first, firstIdentity, err := source.readPinned(ctx, source.hooks.afterFirstMetadata)
	if err != nil {
		return err
	}
	defer clear(first)
	if source.hooks.beforeReopen != nil {
		source.hooks.beforeReopen()
	}
	if err := ctx.Err(); err != nil {
		return credentials.WrapError(credentials.ErrorConsumerFailed, "context", err)
	}

	second, secondIdentity, err := source.readPinned(ctx, nil)
	if err != nil {
		return err
	}
	defer clear(second)
	if firstIdentity != secondIdentity || !bytes.Equal(first, second) {
		return credentials.NewError(credentials.ErrorUnsafeFile, "material_source_changed")
	}
	if err := validateJSONObject(first); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return credentials.WrapError(credentials.ErrorConsumerFailed, "context", err)
	}

	secret, err := credentials.NewSecret(first)
	if err != nil {
		return credentials.NewError(credentials.ErrorUnsafeFile, "material_source_json")
	}
	defer secret.Destroy()
	if err := callback(secret); err != nil {
		if contextErr := ctx.Err(); contextErr != nil {
			return credentials.WrapError(credentials.ErrorConsumerFailed, "context", contextErr)
		}
		return credentials.NewError(credentials.ErrorConsumerFailed, "material_source_callback")
	}
	if err := ctx.Err(); err != nil {
		return credentials.WrapError(credentials.ErrorConsumerFailed, "context", err)
	}
	return nil
}

func (source *Source) readPinned(ctx context.Context, afterMetadata func()) ([]byte, fileIdentity, error) {
	file, before, err := source.openPinned()
	if err != nil {
		return nil, fileIdentity{}, err
	}
	if afterMetadata != nil {
		afterMetadata()
	}
	content, readErr := readBounded(ctx, file, source.maxBytes, before.size)
	after, statErr := source.validateOpened(file)
	closeErr := file.Close()
	if readErr != nil || statErr != nil || closeErr != nil || before != after || uint64(len(content)) != uint64(before.size) {
		clear(content)
		if readErr != nil && credentials.HasErrorCode(readErr, credentials.ErrorConsumerFailed) {
			return nil, fileIdentity{}, readErr
		}
		return nil, fileIdentity{}, credentials.NewError(credentials.ErrorUnsafeFile, "material_source_changed")
	}
	return content, before, nil
}

func (source *Source) openPinned() (*os.File, fileIdentity, error) {
	fd, err := unix.Openat2(unix.AT_FDCWD, source.path, &unix.OpenHow{
		Flags:   unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK,
		Resolve: unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS,
	})
	if err != nil {
		return nil, fileIdentity{}, credentials.NewError(credentials.ErrorUnsafeFile, "material_source")
	}
	file := os.NewFile(uintptr(fd), "credential-material")
	identity, err := source.validateOpened(file)
	if err != nil {
		_ = file.Close()
		return nil, fileIdentity{}, err
	}
	return file, identity, nil
}

func (source *Source) validateOpened(file *os.File) (fileIdentity, error) {
	var stat unix.Stat_t
	if file == nil || unix.Fstat(int(file.Fd()), &stat) != nil || stat.Mode&unix.S_IFMT != unix.S_IFREG ||
		stat.Nlink != 1 || stat.Uid != source.ownerUID || (stat.Mode&0o7777 != 0o400 && stat.Mode&0o7777 != 0o600) ||
		stat.Size < 0 || uint64(stat.Size) > source.maxBytes {
		return fileIdentity{}, credentials.NewError(credentials.ErrorUnsafeFile, "material_source")
	}
	return fileIdentity{
		device: uint64(stat.Dev), inode: stat.Ino, links: uint64(stat.Nlink), size: stat.Size,
		mode: stat.Mode, uid: stat.Uid, gid: stat.Gid,
		modifiedSec: stat.Mtim.Sec, modifiedNsec: stat.Mtim.Nsec,
		changedSec: stat.Ctim.Sec, changedNsec: stat.Ctim.Nsec,
	}, nil
}

func readBounded(ctx context.Context, file *os.File, maximum uint64, expected int64) ([]byte, error) {
	if expected < 0 || uint64(expected) > maximum || uint64(expected) > uint64(int(^uint(0)>>1)) {
		return nil, credentials.NewError(credentials.ErrorUnsafeFile, "material_source")
	}
	content := make([]byte, int(expected))
	readTotal := 0
	for readTotal < len(content) {
		if err := ctx.Err(); err != nil {
			clear(content)
			return nil, credentials.WrapError(credentials.ErrorConsumerFailed, "context", err)
		}
		read, err := file.Read(content[readTotal:])
		readTotal += read
		if err != nil {
			clear(content)
			if errors.Is(err, io.EOF) {
				return nil, credentials.NewError(credentials.ErrorUnsafeFile, "material_source_changed")
			}
			return nil, credentials.NewError(credentials.ErrorStoreIO, "material_source")
		}
		if read == 0 {
			clear(content)
			return nil, credentials.NewError(credentials.ErrorStoreIO, "material_source")
		}
	}
	if err := ctx.Err(); err != nil {
		clear(content)
		return nil, credentials.WrapError(credentials.ErrorConsumerFailed, "context", err)
	}
	var extra [1]byte
	read, err := file.Read(extra[:])
	clear(extra[:])
	if read != 0 {
		clear(content)
		return nil, credentials.NewError(credentials.ErrorUnsafeFile, "material_source_size")
	}
	if !errors.Is(err, io.EOF) {
		clear(content)
		return nil, credentials.NewError(credentials.ErrorStoreIO, "material_source")
	}
	return content, nil
}
