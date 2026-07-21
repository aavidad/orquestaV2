// Package gitlocal implements the opt-in local Git boundary for execution
// workspaces. Physical paths intentionally remain private to this adapter.
package gitlocal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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
	Root       string
	GitCommand string
	Locator    LocalRepositoryLocator
	Now        func() time.Time
}

type Adapter struct {
	root string
	git  string
	loc  LocalRepositoryLocator
	now  func() time.Time

	integrationCASObserver func(integrationCASStage, ports.IntegrationRequest)

	stateMu     sync.RWMutex
	locks       [256]sync.Mutex
	prepared    map[ports.ExecutionWorkspaceRef]preparedRecord
	prepareKeys map[string]preparedRecord
	commits     map[ports.ChangeSetRef]commitRecord
	commitKeys  map[string]commitRecord
	releases    map[string]releaseRecord
}

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
	return &Adapter{
		root: root, git: git, loc: config.Locator, now: config.Now,
		prepared:    make(map[ports.ExecutionWorkspaceRef]preparedRecord),
		prepareKeys: make(map[string]preparedRecord),
		commits:     make(map[ports.ChangeSetRef]commitRecord),
		commitKeys:  make(map[string]commitRecord),
		releases:    make(map[string]releaseRecord),
	}, nil
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
