//go:build linux

package bubblewrap

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"golang.org/x/sys/unix"

	"orquesta/internal/ports"
)

const snapshotFieldMax, maxSnapshotDepth = 4096, 256

type snapshotEntry struct {
	path, mode, target string
	file               *os.File
}
type sandboxSnapshot struct {
	once        sync.Once
	err         error
	entries     []snapshotEntry
	directories []string
}

func readSnapshotStream(source io.Reader, request ports.TestAttestationRequest, max int64) (*sandboxSnapshot, error) {
	return readSnapshotStreamLimited(source, request, max, SubjectEntryLimit(max))
}

func readSnapshotStreamLimited(source io.Reader, request ports.TestAttestationRequest, max, maxFiles int64) (snapshot *sandboxSnapshot, resultErr error) {
	if maxFiles <= 0 {
		return nil, snapshotLimit()
	}
	snapshot = &sandboxSnapshot{}
	defer func() {
		if resultErr != nil {
			_ = snapshot.Close()
			snapshot = nil
		}
	}()
	limited := &io.LimitedReader{R: source, N: max + 1}
	digest := sha256.New()
	canonical := io.TeeReader(limited, digest)
	magic := make([]byte, len(snapshotStreamMagic))
	if _, err := io.ReadFull(canonical, magic); err != nil || !bytes.Equal(magic, snapshotStreamMagic) {
		return nil, streamError(limited, err)
	}
	var fields [4]string
	if err := readStrings(canonical, fields[:]); err != nil || fields[0] != string(request.Subject.ObjectFormat) || fields[1] != request.SubjectDigest || fields[2] != request.Subject.HeadOID || fields[3] != request.Subject.TreeOID {
		return nil, streamError(limited, err)
	}
	directories := map[string]struct{}{}
	maxEntries, openFiles, previous := SubjectEntryLimit(max), int64(0), ""
	for entries := int64(0); ; entries++ {
		var tag [1]byte
		if _, err := io.ReadFull(limited, tag[:]); err != nil {
			return nil, streamError(limited, err)
		}
		if tag[0] == snapshotEndTag {
			break
		}
		if tag[0] != snapshotEntryTag || entries >= maxEntries {
			return nil, snapshotInvalid()
		}
		_, _ = digest.Write(tag[:])
		var values [3]string
		var sizeBytes [8]byte
		if err := readStrings(canonical, values[:]); err != nil {
			return nil, streamError(limited, err)
		}
		if _, err := io.ReadFull(canonical, sizeBytes[:]); err != nil {
			return nil, streamError(limited, err)
		}
		mode, name, size := values[0], values[1], binary.BigEndian.Uint64(sizeBytes[:])
		if !validSnapshotEntry(mode, name, values[2], request.Subject.ObjectFormat) || previous != "" && name <= previous || size > uint64(limited.N) {
			return nil, snapshotInvalid()
		}
		previous = name
		if err := addParentDirectories(directories, name); err != nil {
			return nil, err
		}
		if mode == "120000" {
			if size < 1 || size > snapshotFieldMax {
				return nil, snapshotLimit()
			}
			content := make([]byte, size)
			if _, err := io.ReadFull(canonical, content); err != nil || !safeSymlink(name, content) {
				return nil, snapshotInvalid()
			}
			snapshot.entries = append(snapshot.entries, snapshotEntry{path: name, mode: mode, target: string(content)})
		} else {
			if openFiles >= maxFiles {
				return nil, snapshotLimit()
			}
			file, fileErr := sealedSubjectFile(canonical, int64(size))
			if fileErr != nil {
				return nil, fileErr
			}
			snapshot.entries = append(snapshot.entries, snapshotEntry{path: name, mode: mode, file: file})
			openFiles++
		}
		if entries+1+int64(len(directories)) > maxEntries {
			return nil, snapshotLimit()
		}
	}
	want, got := digest.Sum(nil), make([]byte, sha256.Size)
	if _, err := io.ReadFull(limited, got); err != nil || !bytes.Equal(got, want) {
		return nil, streamError(limited, err)
	}
	var extra [1]byte
	if count, trailing := limited.Read(extra[:]); count != 0 || trailing != io.EOF {
		return nil, streamError(limited, trailing)
	}
	if limited.N == 0 {
		return nil, snapshotLimit()
	}
	for directory := range directories {
		snapshot.directories = append(snapshot.directories, directory)
	}
	sort.Strings(snapshot.directories)
	return snapshot, nil
}

func addParentDirectories(directories map[string]struct{}, name string) error {
	parent := path.Dir(name)
	for depth := 0; parent != "."; depth++ {
		if depth >= maxSnapshotDepth {
			return snapshotLimit()
		}
		if _, found := directories[parent]; found {
			break
		}
		directories[parent] = struct{}{}
		parent = path.Dir(parent)
	}
	return nil
}

func readStrings(reader io.Reader, values []string) error {
	for index := range values {
		var size [4]byte
		if _, err := io.ReadFull(reader, size[:]); err != nil {
			return err
		}
		length := binary.BigEndian.Uint32(size[:])
		if length > snapshotFieldMax {
			return snapshotLimit()
		}
		content := make([]byte, length)
		if _, err := io.ReadFull(reader, content); err != nil {
			return err
		}
		values[index] = string(content)
	}
	return nil
}

func validSnapshotEntry(mode, name, oid string, format ports.GitObjectFormat) bool {
	want := 40
	if format == ports.GitObjectFormatSHA256 {
		want = sha256.Size * 2
	}
	if mode != "100644" && mode != "100755" && mode != "120000" || name == "" || !utf8.ValidString(name) || path.IsAbs(name) || path.Clean(name) != name || strings.ContainsRune(name, 0) || len(oid) != want || strings.Trim(oid, "0123456789abcdef") != "" {
		return false
	}
	for _, part := range strings.Split(name, "/") {
		if part == "" || part == "." || part == ".." || strings.EqualFold(part, ".git") {
			return false
		}
	}
	return true
}

func safeSymlink(name string, content []byte) bool {
	target := string(content)
	resolved := strings.ToLower(path.Clean(path.Join(path.Dir(name), target)))
	return target != "" && utf8.Valid(content) && !path.IsAbs(target) && path.Clean(target) == target && !strings.ContainsRune(target, 0) && resolved != ".." && !strings.HasPrefix(resolved, "../") && resolved != ".git" && !strings.HasPrefix(resolved, ".git/")
}

func sealedSubjectFile(reader io.Reader, size int64) (*os.File, error) {
	fd, err := unix.MemfdCreate("orquesta-subject", unix.MFD_CLOEXEC|unix.MFD_ALLOW_SEALING)
	if errors.Is(err, unix.EMFILE) || errors.Is(err, unix.ENFILE) {
		return nil, snapshotLimit()
	}
	if err != nil {
		return nil, snapshotInvalid()
	}
	file := os.NewFile(uintptr(fd), "orquesta-subject")
	written, copyErr := io.CopyN(file, reader, size)
	if copyErr != nil || written != size || rewindAndSeal(file, unix.F_SEAL_WRITE|unix.F_SEAL_GROW|unix.F_SEAL_SHRINK|unix.F_SEAL_SEAL) != nil {
		_ = file.Close()
		return nil, snapshotInvalid()
	}
	return file, nil
}

func rewindAndSeal(file *os.File, seals int) error {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	_, err := unix.FcntlInt(file.Fd(), unix.F_ADD_SEALS, seals)
	return err
}

func streamError(reader *io.LimitedReader, err error) error {
	if reader.N <= 0 {
		return snapshotLimit()
	}
	if ErrorCode(err) != "" {
		return err
	}
	return snapshotInvalid()
}
func snapshotInvalid() error { return &Error{Code: CodeSnapshotInvalid} }
func snapshotLimit() error   { return &Error{Code: CodeSnapshotLimit} }

func (snapshot *sandboxSnapshot) Close() error {
	if snapshot == nil {
		return nil
	}
	snapshot.once.Do(func() {
		for _, entry := range snapshot.entries {
			if entry.file != nil {
				snapshot.err = errors.Join(snapshot.err, entry.file.Close())
			}
		}
	})
	return snapshot.err
}
