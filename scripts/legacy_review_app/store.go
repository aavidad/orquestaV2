// Este fichero es la única autoridad semántica del historial de propuestas:
// valida solicitudes, revisiones e idempotencia; delega la materialización.
package main

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"syscall"
	"time"
)

type disposition string

const (
	dispositionAdmitted  disposition = "admitido"
	dispositionStudy     disposition = "en_estudio"
	dispositionRejected  disposition = "rechazado"
	dispositionEvidence  disposition = "evidencia_historica"
	dispositionDuplicate disposition = "duplicado"
)

var (
	errInventoryChanged    = errors.New("inventory_changed")
	errRevisionConflict    = errors.New("revision_conflict")
	errIdempotencyConflict = errors.New("idempotency_conflict")
	errInvalidProposal     = errors.New("invalid_proposal")
)

type proposal struct {
	SchemaVersion    int             `json:"schema_version"`
	ProposalRef      string          `json:"proposal_ref"`
	ItemRef          string          `json:"item_ref"`
	ItemRevision     string          `json:"item_revision"`
	ProposalRevision int             `json:"proposal_revision"`
	ExpectedRevision int             `json:"expected_revision"`
	Disposition      disposition     `json:"disposition"`
	Reason           string          `json:"reason"`
	FoundedSolution  string          `json:"founded_solution"`
	Confidence       int             `json:"confidence"`
	RuleCompliance   map[string]bool `json:"rule_compliance"`
	RuleNotes        string          `json:"rule_notes"`
	ActorRef         string          `json:"actor_ref"`
	ProjectRef       string          `json:"project_ref"`
	IdempotencyKey   string          `json:"idempotency_key"`
	RequestDigest    string          `json:"request_digest"`
	CreatedAt        string          `json:"created_at"`
}

type proposalRequest struct {
	ItemRef          string
	ItemRevision     string
	ExpectedRevision int
	Disposition      disposition
	Reason           string
	FoundedSolution  string
	Confidence       int
	RuleCompliance   map[string]bool
	RuleNotes        string
	ActorRef         string
	ProjectRef       string
	IdempotencyKey   string
}

type proposalResult struct {
	Proposal proposal
	Replay   bool
}

type proposalStore struct {
	path string
	now  func() time.Time
	mu   sync.Mutex
}

func newProposalStore(filePath string) *proposalStore {
	return &proposalStore{path: filePath, now: time.Now}
}

func (store *proposalStore) submit(item inventoryItem, request proposalRequest) (proposalResult, error) {
	request = normalizeProposalRequest(request)
	if err := validateProposalRequest(item, request); err != nil {
		return proposalResult{}, err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	lock, err := store.lock()
	if err != nil {
		return proposalResult{}, err
	}
	defer unlockFile(lock)

	proposals, err := readProposals(store.path)
	if err != nil {
		return proposalResult{}, err
	}
	requestDigest := digestRequest(request)
	for _, existing := range proposals {
		if existing.IdempotencyKey != request.IdempotencyKey {
			continue
		}
		if existing.RequestDigest != requestDigest {
			return proposalResult{}, errIdempotencyConflict
		}
		return proposalResult{Proposal: existing, Replay: true}, nil
	}
	latest := latestRevision(proposals, item.ID)
	if request.ExpectedRevision != latest {
		return proposalResult{}, errRevisionConflict
	}
	next := latest + 1
	created := proposal{
		SchemaVersion:    1,
		ProposalRef:      proposalReference(item.ID, next, requestDigest),
		ItemRef:          item.ID,
		ItemRevision:     item.Revision,
		ProposalRevision: next,
		ExpectedRevision: request.ExpectedRevision,
		Disposition:      request.Disposition,
		Reason:           request.Reason,
		FoundedSolution:  request.FoundedSolution,
		Confidence:       request.Confidence,
		RuleCompliance:   cloneRules(request.RuleCompliance),
		RuleNotes:        request.RuleNotes,
		ActorRef:         request.ActorRef,
		ProjectRef:       request.ProjectRef,
		IdempotencyKey:   request.IdempotencyKey,
		RequestDigest:    requestDigest,
		CreatedAt:        store.now().UTC().Format(time.RFC3339Nano),
	}
	proposals = append(proposals, created)
	if err := writeProposals(store.path, proposals); err != nil {
		return proposalResult{}, err
	}
	return proposalResult{Proposal: created}, nil
}

func (store *proposalStore) list() ([]proposal, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	return readProposals(store.path)
}

func (store *proposalStore) lock() (*os.File, error) {
	lockPath := store.path + ".lock"
	if err := os.MkdirAll(filepath.Dir(store.path), 0o700); err != nil {
		return nil, err
	}
	pathInfo, err := privateProposalLockInfo(lockPath)
	if err != nil {
		return nil, err
	}
	file, err := openPrivateProposalLock(lockPath, pathInfo)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		file.Close()
		return nil, err
	}
	return file, nil
}

func privateProposalLockInfo(lockPath string) (os.FileInfo, error) {
	info, err := os.Lstat(lockPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, errors.New("el candado de propuestas no es un fichero regular")
	}
	if info.Mode().Perm()&0o077 != 0 {
		return nil, errors.New("el candado de propuestas no es privado")
	}
	return info, nil
}

func openPrivateProposalLock(lockPath string, pathInfo os.FileInfo) (*os.File, error) {
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, err
	}
	openedInfo, openedErr := file.Stat()
	currentInfo, currentErr := privateProposalLockInfo(lockPath)
	valid := openedErr == nil && openedInfo.Mode().IsRegular() &&
		openedInfo.Mode().Perm()&0o077 == 0 && currentErr == nil &&
		currentInfo != nil && os.SameFile(openedInfo, currentInfo)
	if pathInfo != nil {
		valid = valid && os.SameFile(pathInfo, openedInfo)
	}
	if !valid {
		file.Close()
		return nil, errors.New("el candado de propuestas cambió al abrirlo")
	}
	return file, nil
}

func unlockFile(file *os.File) {
	_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	_ = file.Close()
}

func latestRevision(proposals []proposal, itemID string) int {
	latest := 0
	for _, item := range proposals {
		if item.ItemRef == itemID && item.ProposalRevision > latest {
			latest = item.ProposalRevision
		}
	}
	return latest
}

func rejectSymlink(filePath string) error {
	info, err := os.Lstat(filePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return errors.New("la ruta no es un fichero regular")
	}
	return nil
}

func sortedProposalsForItem(proposals []proposal, itemID string) []proposal {
	var result []proposal
	for _, item := range proposals {
		if item.ItemRef == itemID {
			result = append(result, item)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ProposalRevision > result[j].ProposalRevision
	})
	return result
}
