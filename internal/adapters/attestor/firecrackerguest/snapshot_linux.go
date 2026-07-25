//go:build linux

package firecrackerguest

import (
	"bytes"
	"context"
	"crypto/sha1" // Git SHA-1 object IDs are protocol data, not a security primitive.
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"golang.org/x/sys/unix"

	firecracker "orquesta/internal/testattestorprotocol/rawdrive"
)

const (
	snapshotFieldMax        = 4096
	maxSnapshotDepth        = 256
	maxSnapshotEntries      = 1_000_000
	snapshotEntryTag   byte = 1
	snapshotEndTag     byte = 255
)

var snapshotStreamMagic = []byte("ORQ-SNAPSHOT-2\x00")

type SnapshotIdentity struct {
	ObjectFormat  string
	SubjectDigest string
	HeadOID       string
	TreeOID       string
	Entries       uint64
	Directories   uint64
}

type snapshotEntry struct {
	mode, name, oid string
	size            uint64
}

type snapshotSink interface {
	regular(snapshotEntry, io.Reader) error
	symlink(snapshotEntry, []byte) error
}

type discardSnapshotSink struct{}

func (discardSnapshotSink) regular(entry snapshotEntry, content io.Reader) error {
	_, err := io.CopyN(io.Discard, content, int64(entry.size))
	if err != nil {
		return snapshotInvalid(err)
	}
	return nil
}
func (discardSnapshotSink) symlink(snapshotEntry, []byte) error { return nil }

// ValidateSnapshot validates the complete ORQ-SNAPSHOT-2 stream without
// materializing it. Memory remains bounded independently of file content.
func ValidateSnapshot(
	ctx context.Context,
	source io.Reader,
	expectedSubjectDigest string,
) (SnapshotIdentity, error) {
	return consumeSnapshot(ctx, source, expectedSubjectDigest, discardSnapshotSink{})
}

// MaterializeSnapshot validates before writing, rewinds, then validates again
// while creating entries below root exclusively through openat/O_NOFOLLOW.
func MaterializeSnapshot(
	ctx context.Context,
	source io.ReadSeeker,
	root string,
	expectedSubjectDigest string,
) (result SnapshotIdentity, resultErr error) {
	identity, err := ValidateSnapshot(ctx, source, expectedSubjectDigest)
	if err != nil {
		return SnapshotIdentity{}, err
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return SnapshotIdentity{}, guestError(CodeSnapshotInvalid, err)
	}
	materializer, err := newSnapshotMaterializer(root)
	if err != nil {
		return SnapshotIdentity{}, err
	}
	defer func() {
		if err := materializer.close(); err != nil {
			result = SnapshotIdentity{}
			resultErr = errors.Join(guestError(CodeCleanupFailed, err), resultErr)
		}
	}()
	replayed, err := consumeSnapshot(ctx, source, expectedSubjectDigest, materializer)
	if err != nil || replayed != identity {
		if err == nil {
			err = guestError(CodeSnapshotInvalid, nil)
		}
		return SnapshotIdentity{}, err
	}
	if err := finalizeDirectoryModes(root); err != nil {
		return SnapshotIdentity{}, guestError(CodeMaterializeFailed, err)
	}
	return identity, nil
}

func consumeSnapshot(
	ctx context.Context,
	source io.Reader,
	expectedSubjectDigest string,
	sink snapshotSink,
) (SnapshotIdentity, error) {
	var identity SnapshotIdentity
	limited := &io.LimitedReader{
		R: contextReader{ctx: ctx, reader: source},
		N: int64(firecracker.MaxSnapshotBytes) + 1,
	}
	digest := sha256.New()
	canonical := io.TeeReader(limited, digest)
	magic := make([]byte, len(snapshotStreamMagic))
	if _, err := io.ReadFull(canonical, magic); err != nil || !bytes.Equal(magic, snapshotStreamMagic) {
		return identity, snapshotInvalid(err)
	}
	fields := [4]string{}
	if err := readSnapshotStrings(canonical, fields[:]); err != nil {
		return identity, err
	}
	identity = SnapshotIdentity{
		ObjectFormat: fields[0], SubjectDigest: fields[1], HeadOID: fields[2], TreeOID: fields[3],
	}
	if identity.SubjectDigest != expectedSubjectDigest ||
		!validDigest(identity.SubjectDigest, sha256.Size*2) ||
		!validObjectFormat(identity.ObjectFormat) ||
		!validOID(identity.HeadOID, identity.ObjectFormat) ||
		!validOID(identity.TreeOID, identity.ObjectFormat) {
		return SnapshotIdentity{}, snapshotInvalid(nil)
	}
	previous := ""
	var previousDirectories []string
	for {
		if err := ctx.Err(); err != nil {
			return SnapshotIdentity{}, snapshotInvalid(err)
		}
		var tag [1]byte
		if _, err := io.ReadFull(limited, tag[:]); err != nil {
			return SnapshotIdentity{}, snapshotInvalid(err)
		}
		if tag[0] == snapshotEndTag {
			break
		}
		if tag[0] != snapshotEntryTag {
			return SnapshotIdentity{}, snapshotInvalid(nil)
		}
		if identity.Entries >= maxSnapshotEntries {
			return SnapshotIdentity{}, snapshotLimit(nil)
		}
		if _, err := digest.Write(tag[:]); err != nil {
			return SnapshotIdentity{}, snapshotInvalid(err)
		}
		values := [3]string{}
		var sizeBytes [8]byte
		if err := readSnapshotStrings(canonical, values[:]); err != nil {
			return SnapshotIdentity{}, err
		}
		if _, err := io.ReadFull(canonical, sizeBytes[:]); err != nil {
			return SnapshotIdentity{}, snapshotInvalid(err)
		}
		entry := snapshotEntry{
			mode: values[0], name: values[1], oid: values[2],
			size: binary.BigEndian.Uint64(sizeBytes[:]),
		}
		if !validSnapshotEntry(entry, identity.ObjectFormat) ||
			previous != "" && entry.name <= previous ||
			previous != "" && strings.HasPrefix(entry.name, previous+"/") ||
			entry.size > uint64(limited.N) {
			return SnapshotIdentity{}, snapshotInvalid(nil)
		}
		directories := strings.Split(entry.name, "/")
		directories = directories[:len(directories)-1]
		if err := countSnapshotPath(&identity, previousDirectories, directories); err != nil {
			return SnapshotIdentity{}, err
		}
		previousDirectories = append(previousDirectories[:0], directories...)
		previous = entry.name
		blobDigest := newGitBlobDigest(identity.ObjectFormat, entry.size)
		content := &io.LimitedReader{R: io.TeeReader(canonical, blobDigest), N: int64(entry.size)}
		if entry.mode == "120000" {
			if entry.size == 0 || entry.size > snapshotFieldMax {
				return SnapshotIdentity{}, snapshotLimit(nil)
			}
			value := make([]byte, entry.size)
			if _, err := io.ReadFull(content, value); err != nil || !safeSymlink(entry.name, value) {
				return SnapshotIdentity{}, snapshotInvalid(err)
			}
			if err := sink.symlink(entry, value); err != nil {
				return SnapshotIdentity{}, materializeError(err)
			}
		} else if err := sink.regular(entry, content); err != nil {
			return SnapshotIdentity{}, materializeError(err)
		}
		if content.N != 0 || hex.EncodeToString(blobDigest.Sum(nil)) != entry.oid {
			return SnapshotIdentity{}, snapshotInvalid(nil)
		}
		identity.Entries++
	}
	want, got := digest.Sum(nil), make([]byte, sha256.Size)
	if _, err := io.ReadFull(limited, got); err != nil {
		return SnapshotIdentity{}, snapshotInvalid(err)
	}
	if !bytes.Equal(got, want) {
		return SnapshotIdentity{}, snapshotInvalid(nil)
	}
	var extra [1]byte
	if count, err := limited.Read(extra[:]); count != 0 || err != io.EOF {
		return SnapshotIdentity{}, snapshotInvalid(err)
	}
	if limited.N == 0 {
		return SnapshotIdentity{}, snapshotLimit(nil)
	}
	return identity, nil
}

func countSnapshotPath(
	identity *SnapshotIdentity,
	previousDirectories, directories []string,
) error {
	common := 0
	for common < len(previousDirectories) && common < len(directories) &&
		previousDirectories[common] == directories[common] {
		common++
	}
	addedDirectories := uint64(len(directories) - common)
	totalRecords, ok := checkedScratchBytes(
		identity.Entries, identity.Directories, addedDirectories, 1,
	)
	if !ok || totalRecords > maxSnapshotEntries {
		return snapshotLimit(nil)
	}
	identity.Directories += addedDirectories
	return nil
}

func readSnapshotStrings(source io.Reader, values []string) error {
	for index := range values {
		var size [4]byte
		if _, err := io.ReadFull(source, size[:]); err != nil {
			return snapshotInvalid(err)
		}
		length := binary.BigEndian.Uint32(size[:])
		if length > snapshotFieldMax {
			return snapshotLimit(nil)
		}
		content := make([]byte, length)
		if _, err := io.ReadFull(source, content); err != nil || !utf8.Valid(content) {
			return snapshotInvalid(err)
		}
		values[index] = string(content)
	}
	return nil
}

func validObjectFormat(format string) bool { return format == "sha1" || format == "sha256" }

func validOID(value, format string) bool {
	length := sha1.Size * 2
	if format == "sha256" {
		length = sha256.Size * 2
	}
	return validDigest(value, length)
}

func validDigest(value string, length int) bool {
	return len(value) == length && strings.Trim(value, "0123456789abcdef") == ""
}

func validSnapshotEntry(entry snapshotEntry, format string) bool {
	if entry.mode != "100644" && entry.mode != "100755" && entry.mode != "120000" ||
		entry.name == "" || !utf8.ValidString(entry.name) || path.IsAbs(entry.name) ||
		path.Clean(entry.name) != entry.name || strings.ContainsRune(entry.name, 0) ||
		!validOID(entry.oid, format) {
		return false
	}
	parts := strings.Split(entry.name, "/")
	if len(parts) > maxSnapshotDepth {
		return false
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || strings.EqualFold(part, ".git") {
			return false
		}
	}
	return true
}

func safeSymlink(name string, content []byte) bool {
	target := string(content)
	resolved := strings.ToLower(path.Clean(path.Join(path.Dir(name), target)))
	return target != "" && utf8.Valid(content) && !path.IsAbs(target) &&
		path.Clean(target) == target && !strings.ContainsRune(target, 0) &&
		resolved != ".." && !strings.HasPrefix(resolved, "../") &&
		resolved != ".git" && !strings.HasPrefix(resolved, ".git/")
}

func newGitBlobDigest(format string, size uint64) hash.Hash {
	var digest hash.Hash
	if format == "sha256" {
		digest = sha256.New()
	} else {
		digest = sha1.New()
	}
	_, _ = fmt.Fprintf(digest, "blob %d\x00", size)
	return digest
}

type snapshotMaterializer struct{ rootFD int }

func newSnapshotMaterializer(root string) (*snapshotMaterializer, error) {
	fd, err := unix.Open(root, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, guestError(CodeMaterializeFailed, err)
	}
	return &snapshotMaterializer{rootFD: fd}, nil
}

func (materializer *snapshotMaterializer) close() error {
	if materializer != nil && materializer.rootFD >= 0 {
		err := unix.Close(materializer.rootFD)
		materializer.rootFD = -1
		return err
	}
	return nil
}

func (materializer *snapshotMaterializer) regular(
	entry snapshotEntry,
	content io.Reader,
) (resultErr error) {
	parentFD, leaf, err := materializer.openParent(entry.name)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, unix.Close(parentFD)) }()
	fd, err := unix.Openat(
		parentFD, leaf,
		unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC,
		0o600,
	)
	if err != nil {
		return err
	}
	file := os.NewFile(uintptr(fd), entry.name)
	if file == nil {
		_ = unix.Close(fd)
		return unix.EBADF
	}
	written, copyErr := io.CopyN(file, content, int64(entry.size))
	mode := os.FileMode(0o444)
	if entry.mode == "100755" {
		mode = 0o555
	}
	if copyErr == nil && written == int64(entry.size) {
		copyErr = file.Chmod(mode)
	}
	return errors.Join(copyErr, file.Close())
}

func (materializer *snapshotMaterializer) symlink(
	entry snapshotEntry,
	content []byte,
) (resultErr error) {
	parentFD, leaf, err := materializer.openParent(entry.name)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, unix.Close(parentFD)) }()
	return unix.Symlinkat(string(content), parentFD, leaf)
}

func (materializer *snapshotMaterializer) openParent(name string) (int, string, error) {
	parts := strings.Split(name, "/")
	current, err := unix.Dup(materializer.rootFD)
	if err != nil {
		return -1, "", err
	}
	for _, component := range parts[:len(parts)-1] {
		if err := unix.Mkdirat(current, component, 0o700); err != nil && err != unix.EEXIST {
			_ = unix.Close(current)
			return -1, "", err
		}
		next, err := unix.Openat(
			current, component,
			unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC,
			0,
		)
		_ = unix.Close(current)
		if err != nil {
			return -1, "", err
		}
		current = next
	}
	return current, parts[len(parts)-1], nil
}

func finalizeDirectoryModes(root string) error {
	err := filepath.WalkDir(root, func(name string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return os.Chmod(name, 0o555)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return os.Chmod(root, 0o555)
}

func snapshotInvalid(cause error) error { return guestError(CodeSnapshotInvalid, cause) }
func snapshotLimit(cause error) error   { return guestError(CodeSnapshotLimit, cause) }

func materializeError(cause error) error {
	if ErrorCode(cause) != "" {
		return cause
	}
	return guestError(CodeMaterializeFailed, cause)
}
