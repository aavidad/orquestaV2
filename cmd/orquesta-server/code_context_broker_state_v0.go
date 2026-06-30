package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	orquestacontext "orquesta/modulos/orquesta-context"
)

const (
	serverCodeContextLeasesFileV0 = "code-context-leases-v0.json"
	serverCodeContextCacheFileV0  = "code-context-cache-v0.json"
)

type serverFileCodeContextToolLeaseStoreV0 struct {
	path string
	mu   sync.Mutex
}

type serverCodeContextToolLeaseStateFileV0 struct {
	SchemaVersion string                                   `json:"schema_version"`
	Leases        []orquestacontext.CodeContextToolLeaseV0 `json:"leases,omitempty"`
}

func newServerFileCodeContextToolLeaseStoreV0(path string) *serverFileCodeContextToolLeaseStoreV0 {
	return &serverFileCodeContextToolLeaseStoreV0{path: strings.TrimSpace(path)}
}

func (store *serverFileCodeContextToolLeaseStoreV0) BeginCodeContextToolLeaseV0(
	ctx context.Context,
	request orquestacontext.CodeContextToolLeaseRequestV0,
) (orquestacontext.CodeContextToolLeaseV0, error) {
	memory := orquestacontext.NewInMemoryCodeContextToolLeaseStoreV0()
	lease, err := memory.BeginCodeContextToolLeaseV0(ctx, request)
	if err != nil {
		return orquestacontext.CodeContextToolLeaseV0{}, err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	leases, err := store.loadLockedV0()
	if err != nil {
		return orquestacontext.CodeContextToolLeaseV0{}, err
	}
	leases = upsertCodeContextLeaseV0(leases, lease)
	return lease, store.saveLockedV0(leases)
}

func (store *serverFileCodeContextToolLeaseStoreV0) FinishCodeContextToolLeaseV0(
	ctx context.Context,
	completion orquestacontext.CodeContextToolLeaseCompletionV0,
) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	leases, err := store.loadLockedV0()
	if err != nil {
		return err
	}
	memory := orquestacontext.NewInMemoryCodeContextToolLeaseStoreV0()
	for _, lease := range leases {
		_, beginErr := memory.BeginCodeContextToolLeaseV0(ctx, orquestacontext.CodeContextToolLeaseRequestV0{
			RequestRef:      lease.RequestRef,
			CorrelationID:   lease.CorrelationID,
			RepositoryRef:   lease.RepositoryRef,
			WorktreeRef:     lease.WorktreeRef,
			CommitRef:       lease.CommitRef,
			QueryHash:       lease.QueryHash,
			ToolRef:         lease.ToolRef,
			ProviderKind:    lease.ProviderKind,
			OwnerRef:        lease.OwnerRef,
			StartedAt:       lease.StartedAt,
			LeaseTTLSeconds: lease.LeaseTTLSeconds,
			EvidenceRefs:    lease.EvidenceRefs,
		})
		if beginErr != nil {
			return beginErr
		}
		if lease.Status != orquestacontext.CodeContextToolLeaseStatusActiveV0 {
			status := orquestacontext.CodeContextToolLeaseCompletionCompletedV0
			if lease.Status == orquestacontext.CodeContextToolLeaseStatusFailedV0 {
				status = orquestacontext.CodeContextToolLeaseCompletionFailedV0
			}
			if lease.Status == orquestacontext.CodeContextToolLeaseStatusStoppedV0 {
				status = orquestacontext.CodeContextToolLeaseCompletionStoppedV0
			}
			_ = memory.FinishCodeContextToolLeaseV0(ctx, orquestacontext.CodeContextToolLeaseCompletionV0{
				LeaseRef:    lease.LeaseRef,
				ToolRef:     lease.ToolRef,
				CompletedAt: firstNonEmptyEnvlessV0(completion.CompletedAt, time.Now().UTC().Format(time.RFC3339)),
				Status:      status,
			})
		}
	}
	if err := memory.FinishCodeContextToolLeaseV0(ctx, completion); err != nil {
		return err
	}
	leases, err = memory.ListCodeContextToolLeasesV0(ctx, orquestacontext.CodeContextToolLeaseListFilterV0{})
	if err != nil {
		return err
	}
	return store.saveLockedV0(leases)
}

func (store *serverFileCodeContextToolLeaseStoreV0) ListCodeContextToolLeasesV0(
	ctx context.Context,
	filter orquestacontext.CodeContextToolLeaseListFilterV0,
) ([]orquestacontext.CodeContextToolLeaseV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	leases, err := store.loadLockedV0()
	if err != nil {
		return nil, err
	}
	memory := orquestacontext.NewInMemoryCodeContextToolLeaseStoreV0()
	for _, lease := range leases {
		_, beginErr := memory.BeginCodeContextToolLeaseV0(ctx, orquestacontext.CodeContextToolLeaseRequestV0{
			RequestRef:      lease.RequestRef,
			CorrelationID:   lease.CorrelationID,
			RepositoryRef:   lease.RepositoryRef,
			WorktreeRef:     lease.WorktreeRef,
			CommitRef:       lease.CommitRef,
			QueryHash:       lease.QueryHash,
			ToolRef:         lease.ToolRef,
			ProviderKind:    lease.ProviderKind,
			OwnerRef:        lease.OwnerRef,
			StartedAt:       lease.StartedAt,
			LeaseTTLSeconds: lease.LeaseTTLSeconds,
			EvidenceRefs:    lease.EvidenceRefs,
		})
		if beginErr != nil {
			return nil, beginErr
		}
		if lease.Status != orquestacontext.CodeContextToolLeaseStatusActiveV0 {
			status := orquestacontext.CodeContextToolLeaseCompletionCompletedV0
			if lease.Status == orquestacontext.CodeContextToolLeaseStatusFailedV0 {
				status = orquestacontext.CodeContextToolLeaseCompletionFailedV0
			}
			if lease.Status == orquestacontext.CodeContextToolLeaseStatusStoppedV0 {
				status = orquestacontext.CodeContextToolLeaseCompletionStoppedV0
			}
			_ = memory.FinishCodeContextToolLeaseV0(ctx, orquestacontext.CodeContextToolLeaseCompletionV0{
				LeaseRef:    lease.LeaseRef,
				ToolRef:     lease.ToolRef,
				CompletedAt: time.Now().UTC().Format(time.RFC3339),
				Status:      status,
			})
		}
	}
	return memory.ListCodeContextToolLeasesV0(ctx, filter)
}

func (store *serverFileCodeContextToolLeaseStoreV0) loadLockedV0() ([]orquestacontext.CodeContextToolLeaseV0, error) {
	if strings.TrimSpace(store.path) == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(store.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var state serverCodeContextToolLeaseStateFileV0
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, err
	}
	return append([]orquestacontext.CodeContextToolLeaseV0(nil), state.Leases...), nil
}

func (store *serverFileCodeContextToolLeaseStoreV0) saveLockedV0(leases []orquestacontext.CodeContextToolLeaseV0) error {
	if strings.TrimSpace(store.path) == "" {
		return nil
	}
	state := serverCodeContextToolLeaseStateFileV0{
		SchemaVersion: "server_code_context_tool_leases.v0",
		Leases:        leases,
	}
	return writeServerCodeContextJSONFileV0(store.path, state)
}

func upsertCodeContextLeaseV0(
	leases []orquestacontext.CodeContextToolLeaseV0,
	lease orquestacontext.CodeContextToolLeaseV0,
) []orquestacontext.CodeContextToolLeaseV0 {
	for idx := range leases {
		if leases[idx].LeaseRef == lease.LeaseRef {
			leases[idx] = lease
			return leases
		}
	}
	return append(leases, lease)
}

type serverFileCodeContextCacheV0 struct {
	path string
	mu   sync.Mutex
}

type serverCodeContextCacheStateFileV0 struct {
	SchemaVersion string                                         `json:"schema_version"`
	Entries       map[string]orquestacontext.CodeContextResultV0 `json:"entries,omitempty"`
}

func newServerFileCodeContextCacheV0(path string) *serverFileCodeContextCacheV0 {
	return &serverFileCodeContextCacheV0{path: strings.TrimSpace(path)}
}

func (cache *serverFileCodeContextCacheV0) LoadCodeContextResultV0(key string) (orquestacontext.CodeContextResultV0, bool) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	state, err := cache.loadLockedV0()
	if err != nil {
		return orquestacontext.CodeContextResultV0{}, false
	}
	result, ok := state.Entries[key]
	return result, ok
}

func (cache *serverFileCodeContextCacheV0) SaveCodeContextResultV0(key string, result orquestacontext.CodeContextResultV0) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	state, err := cache.loadLockedV0()
	if err != nil {
		state = serverCodeContextCacheStateFileV0{Entries: map[string]orquestacontext.CodeContextResultV0{}}
	}
	if state.Entries == nil {
		state.Entries = map[string]orquestacontext.CodeContextResultV0{}
	}
	state.SchemaVersion = "server_code_context_cache.v0"
	state.Entries[key] = result
	_ = writeServerCodeContextJSONFileV0(cache.path, state)
}

func (cache *serverFileCodeContextCacheV0) loadLockedV0() (serverCodeContextCacheStateFileV0, error) {
	state := serverCodeContextCacheStateFileV0{Entries: map[string]orquestacontext.CodeContextResultV0{}}
	if strings.TrimSpace(cache.path) == "" {
		return state, nil
	}
	raw, err := os.ReadFile(cache.path)
	if os.IsNotExist(err) {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	err = json.Unmarshal(raw, &state)
	if state.Entries == nil {
		state.Entries = map[string]orquestacontext.CodeContextResultV0{}
	}
	return state, err
}

type serverCodeContextToolOwnerStopperV0 interface {
	StopCodeContextToolOwnerV0(context.Context, string) ([]string, error)
}

type serverCodeContextToolOwnerObserverV0 interface {
	ObserveCodeContextToolOwnerV0(context.Context, orquestacontext.CodeContextToolLeaseV0, time.Time) (orquestacontext.CodeContextToolLeaseObservationV0, error)
}

type serverCodeContextToolWatchdogV0 struct {
	Leases   orquestacontext.CodeContextToolLeaseListPortV0
	Finisher orquestacontext.CodeContextToolLeasePortV0
	Stopper  serverCodeContextToolOwnerStopperV0
	Observer serverCodeContextToolOwnerObserverV0
	Now      func() time.Time
}

type serverCodeContextToolWatchdogResultV0 struct {
	Observed int
	Stopped  int
	Errors   int
	Evidence []string
}

func (watchdog serverCodeContextToolWatchdogV0) RunOnceV0(ctx context.Context) (serverCodeContextToolWatchdogResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if watchdog.Leases == nil || watchdog.Finisher == nil || watchdog.Stopper == nil {
		return serverCodeContextToolWatchdogResultV0{}, nil
	}
	now := time.Now().UTC()
	if watchdog.Now != nil {
		now = watchdog.Now().UTC()
	}
	leases, err := watchdog.Leases.ListCodeContextToolLeasesV0(ctx, orquestacontext.CodeContextToolLeaseListFilterV0{
		Status: orquestacontext.CodeContextToolLeaseStatusActiveV0,
	})
	if err != nil {
		return serverCodeContextToolWatchdogResultV0{}, err
	}
	result := serverCodeContextToolWatchdogResultV0{Observed: len(leases)}
	for _, lease := range leases {
		observation := serverCodeContextToolLeaseObservationV0(lease, now)
		if watchdog.Observer != nil {
			observed, observeErr := watchdog.Observer.ObserveCodeContextToolOwnerV0(ctx, lease, now)
			if observeErr != nil {
				if err := ctx.Err(); err != nil {
					return result, err
				}
				evidence := []string{"evidence-ref-code-context-owner-marker-unavailable"}
				if err := watchdog.Finisher.FinishCodeContextToolLeaseV0(ctx, orquestacontext.CodeContextToolLeaseCompletionV0{
					LeaseRef:     lease.LeaseRef,
					ToolRef:      lease.ToolRef,
					CompletedAt:  now.Format(time.RFC3339),
					Status:       orquestacontext.CodeContextToolLeaseCompletionStoppedV0,
					EvidenceRefs: evidence,
				}); err != nil {
					return result, err
				}
				result.Errors++
				result.Stopped++
				result.Evidence = append(result.Evidence, evidence...)
				continue
			}
			if observed.Lease.LeaseRef != "" {
				observation = observed
			}
		}
		assessment, assessErr := orquestacontext.EvaluateCodeContextToolLeaseV0(observation)
		if assessErr != nil || !assessment.ShouldRequestStop || strings.TrimSpace(lease.OwnerRef) == "" {
			continue
		}
		evidence, stopErr := watchdog.Stopper.StopCodeContextToolOwnerV0(ctx, lease.OwnerRef)
		if stopErr != nil {
			if err := ctx.Err(); err != nil {
				return result, err
			}
			result.Errors++
			result.Evidence = append(result.Evidence, "evidence-ref-code-context-owner-stop-error-"+compactExternalBridgeErrorCodeV0(stopErr))
			continue
		}
		if err := watchdog.Finisher.FinishCodeContextToolLeaseV0(ctx, orquestacontext.CodeContextToolLeaseCompletionV0{
			LeaseRef:     lease.LeaseRef,
			ToolRef:      lease.ToolRef,
			CompletedAt:  now.Format(time.RFC3339),
			Status:       orquestacontext.CodeContextToolLeaseCompletionStoppedV0,
			EvidenceRefs: append(evidence, assessment.EvidenceRefs...),
		}); err != nil {
			return result, err
		}
		result.Stopped++
		result.Evidence = append(result.Evidence, evidence...)
	}
	return result, nil
}

func serverCodeContextToolLeaseObservationV0(
	lease orquestacontext.CodeContextToolLeaseV0,
	now time.Time,
) orquestacontext.CodeContextToolLeaseObservationV0 {
	return orquestacontext.CodeContextToolLeaseObservationV0{
		ObservedAt:     now.UTC().Format(time.RFC3339),
		Lease:          lease,
		CPUPercent:     100,
		CPUHighPercent: 75,
	}
}

func writeServerCodeContextJSONFileV0(path string, value any) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
