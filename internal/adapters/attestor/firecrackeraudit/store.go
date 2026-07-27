//go:build linux

package firecrackeraudit

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/sys/unix"
)

const (
	openSuffix  = ".jsonl.open"
	finalSuffix = ".jsonl"
)

// Store owns only the root from which independent stream handles are opened.
// Closing it prevents new streams but does not invalidate handles already
// returned to a launcher or supervisor.
type Store struct {
	mu            sync.Mutex
	root          *os.File
	rootPath      string
	ownerUID      uint32
	maxEventBytes int
	maxEvents     uint64
}

type Stream struct {
	mu            sync.Mutex
	root          *os.File
	file          *os.File
	rootPath      string
	openName      string
	finalName     string
	streamRef     string
	ownerUID      uint32
	maxEventBytes int
	maxEvents     uint64
	state         Verification
	blocked       error
	leased        bool

	afterChmodBeforeRename func() error
	afterRenameBeforeSync  func() error
}

type Finalized struct {
	HeadDigest string
	Path       string
	Outcome    Outcome
	Events     uint64
}

func New(rootPath string, ownerUID uint32, maxEventBytes int, maxEvents uint64) (*Store, error) {
	if rootPath == "" || maxEventBytes < 256 || maxEvents == 0 ||
		maxEvents > uint64((1<<63-2)/int64(maxEventBytes)) {
		return nil, auditError(CodeConfig, nil)
	}
	fd, err := unix.Open(rootPath, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, auditError(CodeSecurity, err)
	}
	root := os.NewFile(uintptr(fd), "")
	if err := validateRoot(root, ownerUID); err != nil {
		_ = root.Close()
		return nil, err
	}
	return &Store{
		root: root, rootPath: rootPath, ownerUID: ownerUID,
		maxEventBytes: maxEventBytes, maxEvents: maxEvents,
	}, nil
}

func (store *Store) Begin(streamRef string) (*Stream, error) {
	if !validHex(streamRef) {
		return nil, auditError(CodeRef, nil)
	}
	stream, err := store.newStream(streamRef)
	if err != nil {
		return nil, err
	}
	if err := stream.begin(); err != nil {
		stream.release()
		return nil, err
	}
	return stream, nil
}

// RecoverOpen validates the named crash-left stream. It durably restores only
// the known pre-rename 0400 mode frontier; content is never repaired/truncated.
func (store *Store) RecoverOpen(streamRef string) (*Stream, error) {
	if !validHex(streamRef) {
		return nil, auditError(CodeRef, nil)
	}
	stream, err := store.newStream(streamRef)
	if err != nil {
		return nil, err
	}
	if err := stream.recoverOpen(); err != nil {
		stream.release()
		return nil, err
	}
	return stream, nil
}

// RecoverFinal completes the durability frontier after a crash between rename
// and directory fsync. It verifies terminal content before syncing the root.
func (store *Store) RecoverFinal(streamRef string) (Finalized, error) {
	if !validHex(streamRef) {
		return Finalized{}, auditError(CodeRef, nil)
	}
	stream, err := store.newStream(streamRef)
	if err != nil {
		return Finalized{}, err
	}
	defer stream.Release()
	return stream.recoverFinal()
}

func (store *Store) Close() error {
	if store == nil {
		return nil
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.root == nil {
		return nil
	}
	err := store.root.Close()
	store.root = nil
	if err != nil {
		return auditError(CodeIO, err)
	}
	return nil
}

func (store *Store) newStream(streamRef string) (*Stream, error) {
	if store == nil {
		return nil, auditError(CodeNotOpen, nil)
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.root == nil {
		return nil, auditError(CodeNotOpen, nil)
	}
	fd, err := unix.Openat(
		int(store.root.Fd()), ".",
		unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0,
	)
	if err != nil {
		return nil, auditError(CodeIO, err)
	}
	root := os.NewFile(uintptr(fd), "")
	if err := validateRoot(root, store.ownerUID); err != nil {
		_ = root.Close()
		return nil, err
	}
	return &Stream{
		root: root, rootPath: store.rootPath,
		openName: streamRef + openSuffix, finalName: streamRef + finalSuffix,
		streamRef: streamRef, ownerUID: store.ownerUID,
		maxEventBytes: store.maxEventBytes, maxEvents: store.maxEvents,
		state: Verification{HeadDigest: EmptyDigest},
	}, nil
}

func (stream *Stream) begin() error {
	if err := stream.lockRoot(); err != nil {
		return stream.fail(err)
	}
	defer stream.unlockRoot()
	exists, err := existsAt(stream.root, stream.finalName)
	if err != nil {
		return stream.fail(err)
	}
	if exists {
		return auditError(CodeFinalExists, nil)
	}
	fd, err := unix.Openat(
		int(stream.root.Fd()), stream.openName,
		unix.O_RDWR|unix.O_APPEND|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK,
		0o600,
	)
	if err != nil {
		if errors.Is(err, unix.EEXIST) {
			return stream.classifyExistingOpen()
		}
		return stream.fail(auditError(CodeIO, err))
	}
	file := os.NewFile(uintptr(fd), "")
	if err := stream.prepareOpenFile(file, 0); err != nil {
		_ = file.Close()
		return stream.fail(err)
	}
	if err := stream.acquireLease(file); err != nil {
		_ = file.Close()
		return stream.fail(err)
	}
	if err := stream.root.Sync(); err != nil {
		stream.releaseLease(file)
		_ = file.Close()
		return stream.fail(auditError(CodeIO, err))
	}
	stream.file = file
	stream.leased = true
	return nil
}

func (stream *Stream) recoverOpen() error {
	if err := stream.lockRoot(); err != nil {
		return stream.fail(err)
	}
	defer stream.unlockRoot()
	exists, err := existsAt(stream.root, stream.openName)
	if err != nil {
		return stream.fail(err)
	}
	if !exists {
		finalExists, finalErr := existsAt(stream.root, stream.finalName)
		if finalErr != nil {
			return stream.fail(finalErr)
		}
		if finalExists {
			return auditError(CodeFinalExists, nil)
		}
		return auditError(CodeNotOpen, nil)
	}
	file, err := stream.openRecoverable()
	if err != nil {
		return stream.fail(err)
	}
	content, err := readBounded(file, stream.maxEventBytes, stream.maxEvents)
	if err != nil {
		stream.releaseLease(file)
		_ = file.Close()
		return stream.fail(err)
	}
	state, err := Verify(
		bytes.NewReader(content), stream.streamRef, stream.maxEventBytes, stream.maxEvents,
	)
	if err != nil {
		stream.releaseLease(file)
		_ = file.Close()
		return stream.fail(err)
	}
	stream.file = file
	stream.leased = true
	stream.state = state
	return nil
}

func (stream *Stream) recoverFinal() (Finalized, error) {
	if err := stream.lockRoot(); err != nil {
		return Finalized{}, stream.fail(err)
	}
	defer stream.unlockRoot()
	file, err := stream.openExisting(stream.finalName, 0o400, false)
	if err != nil {
		return Finalized{}, stream.fail(err)
	}
	if err := stream.acquireLease(file); err != nil {
		_ = file.Close()
		return Finalized{}, stream.fail(err)
	}
	stream.file = file
	stream.leased = true
	content, err := readBounded(file, stream.maxEventBytes, stream.maxEvents)
	if err != nil {
		return Finalized{}, stream.fail(err)
	}
	state, err := Verify(bytes.NewReader(content), stream.streamRef, stream.maxEventBytes, stream.maxEvents)
	if err != nil {
		return Finalized{}, stream.fail(err)
	}
	if state.Events == 0 || state.Pending ||
		(state.LastTransition != Succeeded && state.LastTransition != Failed) {
		return Finalized{}, stream.fail(auditError(CodeTransition, nil))
	}
	if err := stream.root.Sync(); err != nil {
		return Finalized{}, stream.fail(auditError(CodeIO, err))
	}
	outcome := OutcomeSucceeded
	if state.LastTransition == Failed {
		outcome = OutcomeFailed
	}
	return Finalized{
		HeadDigest: state.HeadDigest,
		Path:       filepath.Join(stream.rootPath, stream.finalName),
		Outcome:    outcome,
		Events:     state.Events,
	}, nil
}

func (stream *Stream) Append(event Event) error {
	if stream == nil {
		return auditError(CodeNotOpen, nil)
	}
	stream.mu.Lock()
	defer stream.mu.Unlock()
	if err := stream.ready(); err != nil {
		return err
	}
	if err := validateEvent(event); err != nil {
		return err
	}
	if err := stream.refresh(); err != nil {
		return stream.fail(err)
	}
	if stream.state.Events == stream.maxEvents {
		return stream.fail(auditError(CodeLimit, nil))
	}
	if stream.state.Pending {
		if event.Transition == Started || event.OperationRef != stream.state.PendingRef ||
			event.SubjectRef != stream.state.PendingSubject {
			return auditError(CodeTransition, nil)
		}
	} else if event.Transition != Started {
		return auditError(CodeTransition, nil)
	}
	record := Record{
		SchemaVersion: SchemaVersion,
		Seq:           stream.state.Events + 1,
		PrevDigest:    stream.state.HeadDigest,
		StreamRef:     stream.streamRef,
		OperationRef:  event.OperationRef,
		SubjectRef:    event.SubjectRef,
		Transition:    event.Transition,
		Diagnostic:    event.Diagnostic,
	}
	encoded, err := encodeRecord(record, stream.maxEventBytes)
	if err != nil {
		return stream.fail(err)
	}
	written, err := stream.file.Write(encoded)
	if err != nil || written != len(encoded) {
		return stream.fail(auditError(CodeIO, err))
	}
	if err := stream.file.Sync(); err != nil {
		return stream.fail(auditError(CodeIO, err))
	}
	if err := stream.refresh(); err != nil {
		return stream.fail(err)
	}
	return nil
}

// Close atomically publishes this stream without replacing an existing file.
func (stream *Stream) Close(outcome Outcome) (headDigest string, path string, err error) {
	if stream == nil {
		return "", "", auditError(CodeNotOpen, nil)
	}
	stream.mu.Lock()
	defer stream.mu.Unlock()
	if err := stream.ready(); err != nil {
		return "", "", err
	}
	if outcome != OutcomeSucceeded && outcome != OutcomeFailed {
		return "", "", auditError(CodeTransition, nil)
	}
	if err := stream.lockRoot(); err != nil {
		return "", "", stream.fail(err)
	}
	rootLocked := true
	defer func() {
		if rootLocked {
			stream.unlockRoot()
		}
	}()
	if err := stream.refresh(); err != nil {
		return "", "", stream.fail(err)
	}
	if stream.state.Events == 0 || stream.state.Pending ||
		(outcome == OutcomeSucceeded && stream.state.LastTransition != Succeeded) ||
		(outcome == OutcomeFailed && stream.state.LastTransition != Failed) {
		return "", "", auditError(CodeTransition, nil)
	}
	if err := stream.file.Sync(); err != nil {
		return "", "", stream.fail(auditError(CodeIO, err))
	}
	if unix.Fchmod(int(stream.file.Fd()), 0o400) != nil {
		return "", "", stream.fail(auditError(CodeIO, nil))
	}
	if err := stream.file.Sync(); err != nil {
		return "", "", stream.fail(auditError(CodeIO, err))
	}
	if stream.afterChmodBeforeRename != nil {
		if err := stream.afterChmodBeforeRename(); err != nil {
			return "", "", stream.fail(auditError(CodeIO, err))
		}
	}
	renameErr := unix.Renameat2(
		int(stream.root.Fd()), stream.openName,
		int(stream.root.Fd()), stream.finalName,
		unix.RENAME_NOREPLACE,
	)
	if renameErr != nil {
		_ = unix.Fchmod(int(stream.file.Fd()), 0o600)
		_ = stream.file.Sync()
		if errors.Is(renameErr, unix.EEXIST) {
			return "", "", stream.fail(auditError(CodeFinalExists, renameErr))
		}
		return "", "", stream.fail(auditError(CodeIO, renameErr))
	}
	if stream.afterRenameBeforeSync != nil {
		if err := stream.afterRenameBeforeSync(); err != nil {
			return "", "", stream.fail(auditError(CodeIO, err))
		}
	}
	var finalStat unix.Stat_t
	if unix.Fstat(int(stream.file.Fd()), &finalStat) != nil {
		return "", "", stream.fail(auditError(CodeIO, nil))
	}
	if err := stream.validateFile(stream.file, stream.finalName, 0o400, finalStat.Size); err != nil {
		return "", "", stream.fail(err)
	}
	if err := stream.root.Sync(); err != nil {
		return "", "", stream.fail(auditError(CodeIO, err))
	}
	head := stream.state.HeadDigest
	finalPath := filepath.Join(stream.rootPath, stream.finalName)
	stream.unlockRoot()
	rootLocked = false
	stream.releaseLease(stream.file)
	stream.leased = false
	_ = stream.file.Close()
	_ = stream.root.Close()
	stream.file = nil
	stream.root = nil
	return head, finalPath, nil
}

func (stream *Stream) refresh() error {
	if err := stream.verifyOpenIdentity(); err != nil {
		return err
	}
	content, err := readBounded(stream.file, stream.maxEventBytes, stream.maxEvents)
	if err != nil {
		return err
	}
	state, err := Verify(
		bytes.NewReader(content), stream.streamRef, stream.maxEventBytes, stream.maxEvents,
	)
	if err != nil {
		return err
	}
	stream.state = state
	return nil
}

func (stream *Stream) ready() error {
	if stream.root == nil || stream.file == nil || !stream.leased {
		return auditError(CodeNotOpen, nil)
	}
	if stream.blocked != nil {
		return auditError(CodeBlocked, stream.blocked)
	}
	return nil
}

func (stream *Stream) fail(err error) error {
	if stream.blocked == nil {
		stream.blocked = err
	}
	return err
}

func (stream *Stream) Release() error {
	if stream == nil {
		return nil
	}
	stream.mu.Lock()
	defer stream.mu.Unlock()
	return stream.releaseLocked()
}

func (stream *Stream) release() {
	if stream == nil {
		return
	}
	_ = stream.releaseLocked()
}

func (stream *Stream) releaseLocked() error {
	var releaseErr error
	if stream.file != nil {
		if stream.leased {
			stream.releaseLease(stream.file)
			stream.leased = false
		}
		if err := stream.file.Close(); err != nil {
			releaseErr = err
		}
		stream.file = nil
	}
	if stream.root != nil {
		if err := stream.root.Close(); err != nil && releaseErr == nil {
			releaseErr = err
		}
		stream.root = nil
	}
	if releaseErr != nil {
		return auditError(CodeIO, releaseErr)
	}
	return nil
}

func validateRoot(root *os.File, ownerUID uint32) error {
	var stat unix.Stat_t
	if root == nil || unix.Fstat(int(root.Fd()), &stat) != nil ||
		stat.Mode&unix.S_IFMT != unix.S_IFDIR || stat.Mode&0o777 != 0o700 ||
		stat.Uid != ownerUID {
		return auditError(CodeSecurity, nil)
	}
	return nil
}

func (stream *Stream) lockRoot() error {
	return lockDescriptor(stream.root)
}

func (stream *Stream) unlockRoot() {
	unlockDescriptor(stream.root)
}

func lockDescriptor(file *os.File) error {
	if file == nil {
		return auditError(CodeNotOpen, nil)
	}
	for {
		err := unix.Flock(int(file.Fd()), unix.LOCK_EX)
		if !errors.Is(err, unix.EINTR) {
			if err != nil {
				return auditError(CodeIO, err)
			}
			return nil
		}
	}
}

func (stream *Stream) acquireLease(file *os.File) error {
	if file == nil {
		return auditError(CodeNotOpen, nil)
	}
	for {
		err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
			return auditError(CodeInUse, err)
		}
		if err != nil {
			return auditError(CodeIO, err)
		}
		return nil
	}
}

func (stream *Stream) releaseLease(file *os.File) {
	unlockDescriptor(file)
}

func unlockDescriptor(file *os.File) {
	if file != nil {
		_ = unix.Flock(int(file.Fd()), unix.LOCK_UN)
	}
}

func (stream *Stream) classifyExistingOpen() error {
	file, _, err := stream.openRecoverableRead()
	if err != nil {
		return err
	}
	defer file.Close()
	if err := stream.acquireLease(file); err != nil {
		return err
	}
	stream.releaseLease(file)
	return auditError(CodeRecoveryRequired, nil)
}

func (stream *Stream) openRecoverable() (*os.File, error) {
	readOnly, mode, err := stream.openRecoverableRead()
	if err != nil {
		return nil, err
	}
	if err := stream.acquireLease(readOnly); err != nil {
		_ = readOnly.Close()
		return nil, err
	}
	repaired := mode == 0o400
	if repaired && unix.Fchmod(int(readOnly.Fd()), 0o600) != nil {
		stream.releaseLease(readOnly)
		_ = readOnly.Close()
		return nil, auditError(CodeIO, nil)
	}
	stream.releaseLease(readOnly)
	if err := readOnly.Close(); err != nil {
		return nil, auditError(CodeIO, err)
	}
	file, err := stream.openExisting(stream.openName, 0o600, true)
	if err != nil {
		return nil, err
	}
	if err := stream.acquireLease(file); err != nil {
		_ = file.Close()
		return nil, err
	}
	if repaired {
		if err := file.Sync(); err != nil {
			stream.releaseLease(file)
			_ = file.Close()
			return nil, auditError(CodeIO, err)
		}
		if err := stream.root.Sync(); err != nil {
			stream.releaseLease(file)
			_ = file.Close()
			return nil, auditError(CodeIO, err)
		}
	}
	return file, nil
}

func (stream *Stream) openRecoverableRead() (*os.File, uint32, error) {
	fd, err := unix.Openat(
		int(stream.root.Fd()), stream.openName,
		unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0,
	)
	if err != nil {
		return nil, 0, auditError(CodeSecurity, err)
	}
	file := os.NewFile(uintptr(fd), "")
	mode, err := stream.validateRecoverableOpen(file)
	if err != nil {
		_ = file.Close()
		return nil, 0, err
	}
	return file, mode, nil
}

func (stream *Stream) prepareOpenFile(file *os.File, expectedSize int64) error {
	var stat unix.Stat_t
	if unix.Fstat(int(file.Fd()), &stat) != nil {
		return auditError(CodeIO, nil)
	}
	if stat.Uid != stream.ownerUID {
		if unix.Fchown(int(file.Fd()), int(stream.ownerUID), int(stat.Gid)) != nil {
			return auditError(CodeSecurity, nil)
		}
	}
	if unix.Fchmod(int(file.Fd()), 0o600) != nil {
		return auditError(CodeIO, nil)
	}
	return stream.validateFile(file, stream.openName, 0o600, expectedSize)
}

func (stream *Stream) validateFile(file *os.File, name string, mode uint32, expectedSize int64) error {
	var descriptor, linked unix.Stat_t
	if unix.Fstat(int(file.Fd()), &descriptor) != nil ||
		unix.Fstatat(int(stream.root.Fd()), name, &linked, unix.AT_SYMLINK_NOFOLLOW) != nil {
		return auditError(CodeSecurity, nil)
	}
	if descriptor.Mode&unix.S_IFMT != unix.S_IFREG || descriptor.Mode&0o777 != mode ||
		descriptor.Uid != stream.ownerUID || descriptor.Nlink != 1 ||
		descriptor.Dev != linked.Dev || descriptor.Ino != linked.Ino ||
		linked.Mode&unix.S_IFMT != unix.S_IFREG ||
		(expectedSize >= 0 && descriptor.Size != expectedSize) {
		return auditError(CodeSecurity, nil)
	}
	return nil
}

func (stream *Stream) validateRecoverableOpen(file *os.File) (uint32, error) {
	var descriptor, linked unix.Stat_t
	if unix.Fstat(int(file.Fd()), &descriptor) != nil ||
		unix.Fstatat(int(stream.root.Fd()), stream.openName, &linked, unix.AT_SYMLINK_NOFOLLOW) != nil {
		return 0, auditError(CodeSecurity, nil)
	}
	mode := descriptor.Mode & 0o777
	if descriptor.Mode&unix.S_IFMT != unix.S_IFREG ||
		(mode != 0o600 && mode != 0o400) ||
		descriptor.Uid != stream.ownerUID || descriptor.Nlink != 1 ||
		descriptor.Dev != linked.Dev || descriptor.Ino != linked.Ino ||
		linked.Mode&unix.S_IFMT != unix.S_IFREG || linked.Mode&0o777 != mode {
		return 0, auditError(CodeSecurity, nil)
	}
	return mode, nil
}

func (stream *Stream) verifyOpenIdentity() error {
	var stat unix.Stat_t
	if unix.Fstat(int(stream.file.Fd()), &stat) != nil {
		return auditError(CodeIO, nil)
	}
	return stream.validateFile(stream.file, stream.openName, 0o600, stat.Size)
}

func (stream *Stream) openExisting(name string, mode uint32, writable bool) (*os.File, error) {
	flags := unix.O_RDONLY
	if writable {
		flags = unix.O_RDWR | unix.O_APPEND
	}
	fd, err := unix.Openat(
		int(stream.root.Fd()), name,
		flags|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0,
	)
	if err != nil {
		if errors.Is(err, unix.ENOENT) {
			return nil, auditError(CodeNotOpen, err)
		}
		return nil, auditError(CodeSecurity, err)
	}
	file := os.NewFile(uintptr(fd), "")
	if err := stream.validateFile(file, name, mode, -1); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}

func readBounded(file *os.File, maxEventBytes int, maxEvents uint64) ([]byte, error) {
	limit := int64(maxEventBytes)*int64(maxEvents) + 1
	if limit <= 1 {
		return nil, auditError(CodeLimit, nil)
	}
	reader := io.NewSectionReader(file, 0, limit)
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, auditError(CodeIO, err)
	}
	if int64(len(content)) == limit {
		return nil, auditError(CodeLimit, nil)
	}
	return content, nil
}

func existsAt(root *os.File, name string) (bool, error) {
	var stat unix.Stat_t
	err := unix.Fstatat(int(root.Fd()), name, &stat, unix.AT_SYMLINK_NOFOLLOW)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, unix.ENOENT) {
		return false, nil
	}
	return false, auditError(CodeSecurity, err)
}
