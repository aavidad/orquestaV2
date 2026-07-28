package gitlocal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	"golang.org/x/sys/unix"

	"orquesta/internal/ports"
)

const (
	snapshotStreamMagic                    = "ORQ-SNAPSHOT-2\x00"
	maxSnapshotMetadataBytes         int64 = 64 * 1024 * 1024
	maxSnapshotStreamFieldBytes            = 4096
	snapshotEntryTag, snapshotEndTag byte  = 1, 255
	snapshotMemfdSeals                     = unix.F_SEAL_WRITE | unix.F_SEAL_GROW | unix.F_SEAL_SHRINK | unix.F_SEAL_SEAL
)

type snapshotPlan struct {
	request      ports.SnapshotVerificationRequest
	format       ports.GitObjectFormat
	entries      []snapshotEntry
	objects      []snapshotObject
	maxWireBytes int64
}

type snapshotEntry struct{ mode, path, oid string }
type snapshotObject struct{ kind, oid string }
type snapshotBudget struct{ bytes, entries int64 }

func (adapter *Adapter) captureSnapshotPlan(ctx context.Context, repository *snapshotRepository,
	request ports.SnapshotVerificationRequest,
) (snapshotPlan, error) {
	budget := snapshotBudget{adapter.maxSnapshotBytes, adapter.maxSnapshotEntries}
	subject := request.Subject
	raw, err := repository.run(ctx, maxSnapshotMetadataBytes, "rev-parse", "--show-object-format",
		subject.HeadOID+"^{tree}", subject.HeadOID+"^", subject.ParentOID+"^{tree}")
	if err != nil {
		return snapshotPlan{}, &Error{Code: CodeSnapshotHash, Cause: err}
	}
	values := strings.Fields(string(raw))
	if len(values) != 4 || ports.GitObjectFormat(values[0]) != subject.ObjectFormat ||
		values[1] != subject.TreeOID || values[2] != subject.ParentOID || subject.BaseOID != subject.ParentOID ||
		ports.ValidateGitOID(values[3], subject.ObjectFormat) != nil {
		return snapshotPlan{}, &Error{Code: CodeSnapshotInvalid}
	}
	if err := budget.consume(int64(len(raw)), 0); err != nil {
		return snapshotPlan{}, err
	}
	entries, headTrees, err := snapshotTreeEntries(ctx, repository, subject.TreeOID, &budget)
	if err != nil {
		return snapshotPlan{}, err
	}
	_, parentTrees, err := snapshotTreeEntries(ctx, repository, values[3], &budget)
	if err != nil {
		return snapshotPlan{}, err
	}
	if err := verifySnapshotDiff(ctx, repository, request, &budget); err != nil {
		return snapshotPlan{}, err
	}
	objects := []snapshotObject{{"commit", subject.HeadOID}, {"commit", subject.ParentOID}}
	objects = append(objects, headTrees...)
	objects = appendUniqueSnapshotObjects(objects, parentTrees...)
	return snapshotPlan{request, subject.ObjectFormat, entries, objects, budget.bytes}, nil
}

func appendUniqueSnapshotObjects(objects []snapshotObject, candidates ...snapshotObject) []snapshotObject {
	seen := make(map[snapshotObject]struct{}, len(objects)+len(candidates))
	for _, object := range objects {
		seen[object] = struct{}{}
	}
	for _, object := range candidates {
		if _, exists := seen[object]; !exists {
			objects, seen[object] = append(objects, object), struct{}{}
		}
	}
	return objects
}

func snapshotTreeEntries(ctx context.Context, repository *snapshotRepository, tree string, budget *snapshotBudget,
) ([]snapshotEntry, []snapshotObject, error) {
	raw, err := repository.run(ctx, budget.bytes, "ls-tree", "-r", "-t", "-z", "--full-tree", tree)
	if err != nil {
		return nil, nil, &Error{Code: CodeSnapshotHash, Cause: err}
	}
	if err := budget.consume(int64(len(raw)), 0); err != nil {
		return nil, nil, err
	}
	return parseSnapshotTreeEntries(raw, tree, budget)
}

func parseSnapshotTreeEntries(raw []byte, tree string, budget *snapshotBudget,
) ([]snapshotEntry, []snapshotObject, error) {
	var entries []snapshotEntry
	trees := []snapshotObject{{"tree", tree}}
	lastOrderKey := ""
	seenPaths := make(map[string]struct{})
	for _, record := range bytes.Split(raw, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		header, name, found := bytes.Cut(record, []byte{'\t'})
		fields := strings.Fields(string(header))
		entry := snapshotEntry{path: string(name)}
		if len(fields) == 3 {
			entry.mode, entry.oid = fields[0], fields[2]
		}
		isTree, modeErr := snapshotEntryMode(entry.mode)
		orderKey := snapshotTreeOrderKey(entry.path, isTree)
		_, duplicate := seenPaths[entry.path]
		if !found || len(fields) != 3 || modeErr != nil || !validSnapshotPath(entry.path) ||
			duplicate || orderKey <= lastOrderKey {
			if modeErr != nil {
				return nil, nil, modeErr
			}
			return nil, nil, &Error{Code: CodeSnapshotInvalid}
		}
		lastOrderKey = orderKey
		seenPaths[entry.path] = struct{}{}
		if err := budget.consume(0, 1); err != nil {
			return nil, nil, err
		}
		if isTree {
			trees = append(trees, snapshotObject{"tree", entry.oid})
		} else {
			entries = append(entries, entry)
		}
	}
	return entries, trees, nil
}

func snapshotTreeOrderKey(entryPath string, isTree bool) string {
	if isTree {
		return entryPath + "/"
	}
	return entryPath
}

func verifySnapshotDiff(ctx context.Context, repository *snapshotRepository, request ports.SnapshotVerificationRequest,
	budget *snapshotBudget,
) error {
	subject := request.Subject
	raw, err := repository.run(ctx, budget.bytes, "diff-tree", "--no-ext-diff", "--no-textconv", "--raw", "-z",
		"--no-abbrev", "--no-renames", "-r", "--no-commit-id", subject.ParentOID, subject.HeadOID)
	if err != nil || budget.consume(int64(len(raw)), 0) != nil {
		return &Error{Code: CodeSnapshotInvalid, Cause: err}
	}
	digest := sha256.Sum256(raw)
	if hex.EncodeToString(digest[:]) != subject.DiffDigest {
		return &Error{Code: CodeChangeConflict}
	}
	parts := bytes.Split(raw, []byte{0})
	if len(parts)%2 == 0 || len(parts[len(parts)-1]) != 0 {
		return &Error{Code: CodeSnapshotInvalid}
	}
	paths := make([]string, 0, len(parts)/2)
	for index := 0; index < len(parts)-1; index += 2 {
		if !validSnapshotPath(string(parts[index+1])) {
			return &Error{Code: CodeSnapshotInvalid}
		}
		paths = append(paths, string(parts[index+1]))
	}
	want := append([]string(nil), request.ChangedPaths...)
	sort.Strings(paths)
	sort.Strings(want)
	if !slices.Equal(paths, want) || !pathsWithin(paths, request.WriteSet) {
		return &Error{Code: CodeWriteSetViolation}
	}
	return nil
}

func (budget *snapshotBudget) consume(byteCount, entryCount int64) error {
	if byteCount < 0 || entryCount < 0 || byteCount > budget.bytes || entryCount > budget.entries {
		return &Error{Code: CodeSnapshotLimit}
	}
	budget.bytes, budget.entries = budget.bytes-byteCount, budget.entries-entryCount
	return nil
}

func (adapter *Adapter) OpenSnapshotStream(ctx context.Context,
	request ports.SnapshotVerificationRequest,
) (stream io.ReadCloser, resultErr error) {
	if err := adapter.ensureAvailable(); err != nil {
		return nil, err
	}
	request.ChangedPaths = append([]string(nil), request.ChangedPaths...)
	request.WriteSet = append([]string(nil), request.WriteSet...)
	if err := ports.ValidateSnapshotVerificationRequest(request); err != nil {
		return nil, err
	}
	repository, err := adapter.openSnapshotRepository(ctx, request.Subject)
	if err != nil {
		return nil, err
	}
	defer func() {
		resultErr = errors.Join(resultErr, repository.Close())
		if resultErr != nil && stream != nil {
			_ = stream.Close()
			stream = nil
		}
	}()
	plan, err := adapter.captureSnapshotPlan(ctx, repository, request)
	if err != nil {
		return nil, err
	}
	adapter.observeSnapshotRace(snapshotRacePlanReady, "")
	return materializeSnapshot(ctx, repository, plan)
}

func materializeSnapshot(ctx context.Context, repository *snapshotRepository, plan snapshotPlan) (*os.File, error) {
	fd, err := unix.MemfdCreate("orquesta-snapshot", unix.MFD_CLOEXEC|unix.MFD_ALLOW_SEALING)
	if err != nil {
		return nil, &Error{Code: CodeSnapshotStream, Cause: err}
	}
	file := os.NewFile(uintptr(fd), "orquesta-snapshot")
	err = emitSnapshot(ctx, file, repository, plan)
	if err == nil {
		err = sealMemfd(file, 0o400, snapshotMemfdSeals)
	}
	if err == nil {
		_, err = file.Seek(0, io.SeekStart)
	}
	if err != nil {
		_ = file.Close()
		if ErrorCodeOf(err) == "" {
			err = &Error{Code: CodeSnapshotStream, Cause: err}
		}
		return nil, err
	}
	return file, nil
}

func emitSnapshot(ctx context.Context, output io.Writer, repository *snapshotRepository, plan snapshotPlan) (resultErr error) {
	session, err := openGitObjectSession(ctx, repository, plan.format)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, session.Close()) }()
	wire := &snapshotWireWriter{output, plan.maxWireBytes}
	digest := sha256.New()
	canonical := io.MultiWriter(wire, digest)
	if _, err := io.WriteString(canonical, snapshotStreamMagic); err != nil {
		return err
	}
	if err := writeSnapshotFields(canonical, string(plan.format), plan.request.SubjectDigest,
		plan.request.Subject.HeadOID, plan.request.Subject.TreeOID); err != nil {
		return err
	}
	for _, object := range plan.objects {
		err := session.read(object.oid, object.kind, wire.remaining, func(size int64, content io.Reader) error {
			wire.remaining -= size
			_, err := io.Copy(io.Discard, content)
			return err
		})
		if err != nil {
			return err
		}
	}
	for _, entry := range plan.entries {
		repository.adapter.observeSnapshotRace(snapshotRaceBeforeBlobRead, entry.oid)
		err := session.read(entry.oid, "blob", plan.maxWireBytes, func(size int64, content io.Reader) error {
			return writeSnapshotEntry(canonical, entry, size, content)
		})
		if err != nil {
			return err
		}
	}
	if _, err := wire.Write([]byte{snapshotEndTag}); err != nil {
		return err
	}
	if _, err := wire.Write(digest.Sum(nil)); err != nil {
		return err
	}
	return nil
}

func (adapter *Adapter) observeSnapshotRace(stage snapshotRaceStage, oid string) {
	if adapter.snapshotRaceObserver != nil {
		adapter.snapshotRaceObserver(stage, oid)
	}
}

func writeSnapshotEntry(writer io.Writer, entry snapshotEntry, size int64, content io.Reader) error {
	if _, err := writer.Write([]byte{snapshotEntryTag}); err != nil {
		return err
	}
	if err := writeSnapshotFields(writer, entry.mode, entry.path, entry.oid); err != nil {
		return err
	}
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], uint64(size))
	if _, err := writer.Write(encoded[:]); err != nil {
		return err
	}
	if entry.mode != "120000" {
		_, err := io.Copy(writer, content)
		return err
	}
	if size < 1 || size > maxSnapshotStreamFieldBytes {
		return &Error{Code: CodeSnapshotLimit}
	}
	value, err := io.ReadAll(content)
	if err != nil || !validSymlinkTarget(entry.path, value) {
		return &Error{Code: CodeSnapshotPath, Cause: err}
	}
	_, err = writer.Write(value)
	return err
}

func writeSnapshotFields(writer io.Writer, values ...string) error {
	for _, value := range values {
		if len(value) > maxSnapshotStreamFieldBytes {
			return &Error{Code: CodeSnapshotInvalid}
		}
		var length [4]byte
		binary.BigEndian.PutUint32(length[:], uint32(len(value)))
		if _, err := writer.Write(length[:]); err != nil {
			return err
		}
		if _, err := io.WriteString(writer, value); err != nil {
			return err
		}
	}
	return nil
}

type snapshotWireWriter struct {
	io.Writer
	remaining int64
}

func (writer *snapshotWireWriter) Write(value []byte) (int, error) {
	if int64(len(value)) > writer.remaining {
		return 0, &Error{Code: CodeSnapshotLimit}
	}
	count, err := writer.Writer.Write(value)
	writer.remaining -= int64(count)
	return count, err
}

func validSymlinkTarget(entryPath string, value []byte) bool {
	target := string(value)
	if target == "" || !utf8.Valid(value) || strings.IndexByte(target, 0) >= 0 || path.IsAbs(target) || path.Clean(target) != target {
		return false
	}
	resolved := path.Clean(path.Join(path.Dir(entryPath), target))
	return resolved != ".." && !strings.HasPrefix(resolved, "../") && !strings.EqualFold(resolved, ".git") &&
		!strings.HasPrefix(strings.ToLower(resolved), ".git/")
}
