// Package gitlocal implements the opt-in local Git boundary for execution
// workspaces. Physical paths intentionally remain private to this adapter.
package gitlocal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

const adapterRef = "workspace:git-local"

// LocalRepositoryLocator is composition-owned. It maps an opaque repository
// reference to an already authorised local checkout; no request supplies a
// path, URL, branch or remote.
type LocalRepositoryLocator interface {
	LocateLocalRepository(context.Context, identity.RepositoryRef) (LocalRepositoryBinding, error)
}

// LocalRepositoryBinding stays adapter-private: it is installed by bootstrap,
// never serialised in a Goal or an effect receipt.
type LocalRepositoryBinding struct {
	Path      string
	TargetRef string
}

type Config struct {
	Root               string
	GitCommand         string
	Locator            LocalRepositoryLocator
	Now                func() time.Time
	MaxSnapshotBytes   int64
	MaxSnapshotEntries int64
}

type Adapter struct {
	root               string
	gitFile            *os.File
	snapshotDigest     string
	loc                LocalRepositoryLocator
	now                func() time.Time
	maxSnapshotBytes   int64
	maxSnapshotEntries int64

	integrationCASObserver func(integrationCASStage, ports.IntegrationRequest)
	snapshotRaceObserver   func(snapshotRaceStage, string)

	gitMu       sync.RWMutex
	closed      bool
	closeErr    error
	stateMu     sync.RWMutex
	locks       [256]sync.Mutex
	prepared    map[ports.ExecutionWorkspaceRef]preparedRecord
	prepareKeys map[string]preparedRecord
	commits     map[ports.ChangeSetRef]commitRecord
	commitKeys  map[string]commitRecord
	releases    map[string]releaseRecord
}

type snapshotRaceStage uint8

const (
	snapshotRacePlanReady snapshotRaceStage = iota + 1
	snapshotRaceBeforeBlobRead
)

type integrationCASStage uint8

const (
	integrationCASReady integrationCASStage = iota + 1
	integrationCASReconcile
)

type preparedRecord struct {
	request ports.WorkspacePrepareRequest
	result  ports.WorkspacePrepared
	path    string
	gitFile string
}

type commitRecord struct {
	request ports.CommitRequest
	result  ports.CommitResult
}

type releaseRecord struct {
	request ports.WorkspaceReleaseRequest
	result  ports.WorkspaceReleaseReceipt
}

func New(config Config) (*Adapter, error) {
	return newAdapter(config, gitPinPolicy{ownerUID: 0})
}

func newAdapter(config Config, pinPolicy gitPinPolicy) (*Adapter, error) {
	if config.Root == "" || config.Locator == nil || config.Now == nil {
		return nil, &Error{Code: CodeConfigInvalid}
	}
	root, err := privateRoot(config.Root)
	if err != nil {
		return nil, err
	}
	git := config.GitCommand
	if git == "" || !filepath.IsAbs(git) {
		return nil, &Error{Code: CodeConfigInvalid}
	}
	maxBytes, maxEntries := config.MaxSnapshotBytes, config.MaxSnapshotEntries
	if maxBytes == 0 {
		maxBytes = 512 * 1024 * 1024
	}
	if maxEntries == 0 {
		maxEntries = 200_000
	}
	if maxBytes < 1 || maxBytes > 8*1024*1024*1024 || maxEntries < 1 || maxEntries > 1_000_000 {
		return nil, &Error{Code: CodeConfigInvalid}
	}
	gitFile, gitDigest, err := pinGitExecutable(git, pinPolicy)
	if err != nil {
		return nil, err
	}
	_, snapshotDigest := snapshotSourceIdentity(snapshotSourceContract, gitDigest)
	return &Adapter{
		root: root, gitFile: gitFile,
		snapshotDigest: snapshotDigest, loc: config.Locator, now: config.Now,
		maxSnapshotBytes: maxBytes, maxSnapshotEntries: maxEntries,
		prepared:    make(map[ports.ExecutionWorkspaceRef]preparedRecord),
		prepareKeys: make(map[string]preparedRecord),
		commits:     make(map[ports.ChangeSetRef]commitRecord),
		commitKeys:  make(map[string]commitRecord),
		releases:    make(map[string]releaseRecord),
	}, nil
}

// SnapshotIdentity binds the path-independent snapshot contract to the exact
// Git executable bytes captured by New.
func (adapter *Adapter) SnapshotIdentity() (ref string, digest string) {
	if adapter == nil {
		return "", ""
	}
	return snapshotSourceRef, adapter.snapshotDigest
}

// Close prevents new commands and releases the retained executable descriptor.
// Commands that already duplicated the descriptor remain independent.
func (adapter *Adapter) Close() error {
	if adapter == nil {
		return nil
	}
	adapter.gitMu.Lock()
	defer adapter.gitMu.Unlock()
	if !adapter.closed {
		adapter.closed = true
		if adapter.gitFile != nil {
			adapter.closeErr = adapter.gitFile.Close()
			adapter.gitFile = nil
		}
	}
	return adapter.closeErr
}

func (adapter *Adapter) ensureAvailable() error {
	if adapter == nil {
		return &Error{Code: CodeUnavailable}
	}
	adapter.gitMu.RLock()
	defer adapter.gitMu.RUnlock()
	if adapter.closed || adapter.gitFile == nil {
		return &Error{Code: CodeUnavailable}
	}
	return nil
}

func equalPrepare(left, right ports.WorkspacePrepareRequest) bool {
	return prepareRequestDigest(left) == prepareRequestDigest(right)
}
func equalCommit(left, right ports.CommitRequest) bool {
	return commitRequestDigest(left) == commitRequestDigest(right)
}

func digestRef(prefix, value string) string {
	digest := sha256.Sum256([]byte(value))
	return prefix + hex.EncodeToString(digest[:])
}

func (adapter *Adapter) workspacePath(ref ports.ExecutionWorkspaceRef) string {
	return filepath.Join(adapter.root, "workspaces", digestRef("", ref.String()))
}

func workspaceBranch(ref ports.ExecutionWorkspaceRef) string {
	return "orquesta/workspaces/" + digestRef("", ref.String())
}

func workspaceBaseRef(ref ports.ExecutionWorkspaceRef) string {
	return "refs/orquesta/workspace-bases/" + digestRef("", ref.String())
}

func markerRef(key string) string { return "refs/orquesta/effects/" + digestRef("", key) }

// lockScopes serialises only colliding effects/resources. Git update-ref stays
// the cross-process CAS authority; unrelated agents never share a global lock.
func (adapter *Adapter) lockScopes(scopes ...string) func() {
	unique := make(map[int]struct{}, len(scopes))
	for _, scope := range scopes {
		digest := sha256.Sum256([]byte(scope))
		unique[int(digest[0])] = struct{}{}
	}
	indices := make([]int, 0, len(unique))
	for index := range unique {
		indices = append(indices, index)
	}
	sort.Ints(indices)
	for _, index := range indices {
		adapter.locks[index].Lock()
	}
	return func() {
		for index := len(indices) - 1; index >= 0; index-- {
			adapter.locks[indices[index]].Unlock()
		}
	}
}

var errUnsafeWorkspace = errors.New("unsafe workspace")
