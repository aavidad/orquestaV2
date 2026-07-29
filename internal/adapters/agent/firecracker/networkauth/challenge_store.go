package networkauth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"sync"
	"time"
)

const challengeBytes = 32

type ChallengeStoreOptions struct {
	Now       func() time.Time
	Random    io.Reader
	TTL       time.Duration
	MaxActive int
}

type ChallengeIssueRequest struct {
	PolicyDigest        string
	LaunchBindingDigest string
	LaunchPlanDigest    string
	ExecutionRef        string
	AgentRef            string
}

type IssuedChallenge struct {
	Ref       string
	Value     []byte
	ExpiresAt time.Time
}

type ChallengeConsumeRequest struct {
	Ref                 string
	Value               []byte
	PolicyDigest        string
	LaunchBindingDigest string
	LaunchPlanDigest    string
	ExecutionRef        string
	AgentRef            string
}

type ChallengeConsumer interface {
	Consume(context.Context, ChallengeConsumeRequest) error
}

// MemoryChallengeStore is intentionally process-local: restart invalidates
// every outstanding challenge, which fails closed. Consume is atomic, so only
// one concurrent presentation can authorize a launch.
type MemoryChallengeStore struct {
	mu        sync.Mutex
	now       func() time.Time
	random    io.Reader
	ttl       time.Duration
	maxActive int
	active    map[string]challengeRecord
}

type challengeRecord struct {
	value               [challengeBytes]byte
	policyDigest        string
	launchBindingDigest string
	launchPlanDigest    string
	executionRef        string
	agentRef            string
	expiresAt           time.Time
}

func NewMemoryChallengeStore(options ChallengeStoreOptions) (*MemoryChallengeStore, error) {
	if options.Now == nil || options.Random == nil || options.TTL <= 0 || options.MaxActive <= 0 {
		return nil, authError("challenge_store_config_invalid")
	}
	return &MemoryChallengeStore{
		now: options.Now, random: options.Random, ttl: options.TTL,
		maxActive: options.MaxActive, active: make(map[string]challengeRecord),
	}, nil
}

func (store *MemoryChallengeStore) Issue(
	ctx context.Context,
	request ChallengeIssueRequest,
) (IssuedChallenge, error) {
	if store == nil || ctx == nil || ctx.Err() != nil || !validDigest(request.PolicyDigest) ||
		!validDigest(request.LaunchBindingDigest) || !validDigest(request.LaunchPlanDigest) ||
		!validRef(request.ExecutionRef) ||
		!validRef(request.AgentRef) {
		return IssuedChallenge{}, authError("challenge_issue_invalid")
	}
	var value [challengeBytes]byte
	if _, err := io.ReadFull(store.random, value[:]); err != nil {
		clear(value[:])
		return IssuedChallenge{}, authError("challenge_entropy_unavailable")
	}
	digest := sha256.Sum256(value[:])
	ref := "challenge:agent-microvm:" + hex.EncodeToString(digest[:])
	now := store.now().UTC()
	if now.IsZero() {
		clear(value[:])
		return IssuedChallenge{}, authError("challenge_clock_invalid")
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	store.pruneExpired(now)
	if len(store.active) >= store.maxActive {
		clear(value[:])
		return IssuedChallenge{}, authError("challenge_capacity_reached")
	}
	if _, collision := store.active[ref]; collision {
		clear(value[:])
		return IssuedChallenge{}, authError("challenge_collision")
	}
	expiresAt := now.Add(store.ttl)
	store.active[ref] = challengeRecord{
		value: value, policyDigest: request.PolicyDigest,
		launchBindingDigest: request.LaunchBindingDigest,
		launchPlanDigest:    request.LaunchPlanDigest,
		executionRef:        request.ExecutionRef, agentRef: request.AgentRef, expiresAt: expiresAt,
	}
	return IssuedChallenge{Ref: ref, Value: append([]byte(nil), value[:]...), ExpiresAt: expiresAt}, nil
}

func (store *MemoryChallengeStore) Consume(
	ctx context.Context,
	request ChallengeConsumeRequest,
) error {
	if store == nil || ctx == nil || ctx.Err() != nil || !validRef(request.Ref) ||
		len(request.Value) != challengeBytes || !validDigest(request.PolicyDigest) ||
		!validDigest(request.LaunchBindingDigest) || !validDigest(request.LaunchPlanDigest) ||
		!validRef(request.ExecutionRef) ||
		!validRef(request.AgentRef) {
		return authError("challenge_consume_invalid")
	}
	now := store.now().UTC()
	if now.IsZero() {
		return authError("challenge_clock_invalid")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	record, found := store.active[request.Ref]
	if !found {
		return authError("challenge_missing_or_consumed")
	}
	if !now.Before(record.expiresAt) {
		delete(store.active, request.Ref)
		clear(record.value[:])
		return authError("challenge_expired")
	}
	valueDigest := sha256.Sum256(request.Value)
	recordDigest := sha256.Sum256(record.value[:])
	if valueDigest != recordDigest ||
		request.PolicyDigest != record.policyDigest ||
		request.LaunchBindingDigest != record.launchBindingDigest ||
		request.LaunchPlanDigest != record.launchPlanDigest ||
		request.ExecutionRef != record.executionRef || request.AgentRef != record.agentRef {
		return authError("challenge_binding_mismatch")
	}
	delete(store.active, request.Ref)
	clear(record.value[:])
	return nil
}

func (store *MemoryChallengeStore) pruneExpired(now time.Time) {
	for ref, record := range store.active {
		if !now.Before(record.expiresAt) {
			delete(store.active, ref)
			clear(record.value[:])
		}
	}
}

var _ ChallengeConsumer = (*MemoryChallengeStore)(nil)
