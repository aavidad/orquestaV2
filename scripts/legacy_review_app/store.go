// Este fichero es la única autoridad escritora: conserva propuestas JSONL.
// No modifica el inventario, el catálogo del producto ni crea tareas.
package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
		Reason:           strings.TrimSpace(request.Reason),
		FoundedSolution:  strings.TrimSpace(request.FoundedSolution),
		Confidence:       request.Confidence,
		RuleCompliance:   cloneRules(request.RuleCompliance),
		RuleNotes:        strings.TrimSpace(request.RuleNotes),
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
	if err := rejectSymlink(lockPath); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(store.path), 0o700); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		file.Close()
		return nil, err
	}
	return file, nil
}

func unlockFile(file *os.File) {
	_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	_ = file.Close()
}

func readProposals(filePath string) ([]proposal, error) {
	if err := rejectSymlink(filePath); err != nil {
		return nil, err
	}
	file, err := os.Open(filePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if info.Mode().Perm()&0o077 != 0 {
		return nil, errors.New("el historial de propuestas no es privado")
	}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1<<20)
	var result []proposal
	line := 0
	for scanner.Scan() {
		line++
		if len(bytes.TrimSpace(scanner.Bytes())) == 0 {
			continue
		}
		var item proposal
		if err := json.Unmarshal(scanner.Bytes(), &item); err != nil {
			return nil, fmt.Errorf("propuestas línea %d: %w", line, err)
		}
		result = append(result, item)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return result, validateStoredProposals(result)
}

func writeProposals(filePath string, proposals []proposal) error {
	if err := rejectSymlink(filePath); err != nil {
		return err
	}
	directory := filepath.Dir(filePath)
	temporary, err := os.CreateTemp(directory, "."+filepath.Base(filePath)+".tmp-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	encoder := json.NewEncoder(temporary)
	encoder.SetEscapeHTML(false)
	for _, item := range proposals {
		if err := encoder.Encode(item); err != nil {
			temporary.Close()
			return err
		}
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(name, filePath)
}

func validateStoredProposals(proposals []proposal) error {
	latest := map[string]int{}
	seenKeys := map[string]struct{}{}
	for _, item := range proposals {
		request := proposalRequest{
			ItemRef:          item.ItemRef,
			ItemRevision:     item.ItemRevision,
			ExpectedRevision: item.ExpectedRevision,
			Disposition:      item.Disposition,
			Reason:           item.Reason,
			FoundedSolution:  item.FoundedSolution,
			Confidence:       item.Confidence,
			RuleCompliance:   item.RuleCompliance,
			RuleNotes:        item.RuleNotes,
			ActorRef:         item.ActorRef,
			ProjectRef:       item.ProjectRef,
			IdempotencyKey:   item.IdempotencyKey,
		}
		if item.SchemaVersion != 1 || item.ProposalRevision != latest[item.ItemRef]+1 ||
			item.ExpectedRevision != latest[item.ItemRef] ||
			validateProposalRequest(inventoryItem{ID: item.ItemRef, Revision: item.ItemRevision}, request) != nil ||
			digestRequest(request) != item.RequestDigest ||
			proposalReference(item.ItemRef, item.ProposalRevision, item.RequestDigest) != item.ProposalRef {
			return errors.New("historial de propuestas no canónico")
		}
		if _, err := time.Parse(time.RFC3339Nano, item.CreatedAt); err != nil {
			return errors.New("fecha de propuesta no canónica")
		}
		if _, exists := seenKeys[item.IdempotencyKey]; exists {
			return errors.New("clave idempotente duplicada")
		}
		seenKeys[item.IdempotencyKey] = struct{}{}
		latest[item.ItemRef] = item.ProposalRevision
	}
	return nil
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

func digestRequest(request proposalRequest) string {
	content, _ := json.Marshal(request)
	digest := sha256.Sum256(content)
	return hex.EncodeToString(digest[:])
}

func proposalReference(itemID string, revision int, requestDigest string) string {
	digest := sha256.Sum256([]byte(itemID + "\x00" + fmt.Sprint(revision) + "\x00" + requestDigest))
	return "propuesta:sha256:" + hex.EncodeToString(digest[:])
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
