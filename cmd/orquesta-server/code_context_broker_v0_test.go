package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	orquestacontext "orquesta/modulos/orquesta-context"
)

func TestServerRGCodeContextProviderV0DevuelveCoincidenciasCompactas(t *testing.T) {
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("rg no disponible")
	}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "modulos", "demo"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "modulos", "demo", "service.go"),
		[]byte("package demo\n\nfunc BrokerCentral() {}\n"),
		0o644,
	); err != nil {
		t.Fatalf("write: %v", err)
	}
	provider := serverRGCodeContextProviderV0{RootDir: root, Command: "rg"}
	result, err := provider.QueryCodeContextV0(context.Background(), orquestacontext.CodeContextQueryV0{
		SchemaVersion: orquestacontext.CodeContextQuerySchemaVersionV0,
		RepositoryRef: "repo-ref-test",
		QueryKind:     orquestacontext.CodeContextQueryKindSearchV0,
		Query:         "BrokerCentral",
		Scope:         []string{"modulos"},
		MaxResults:    5,
		MaxBytes:      3000,
	})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if result.Estado != orquestacontext.CodeContextEstadoOKV0 || len(result.Results) != 1 {
		t.Fatalf("result=%+v", result)
	}
	if result.Results[0].Path != "modulos/demo/service.go" ||
		result.Results[0].Line != 3 {
		t.Fatalf("hit=%+v", result.Results[0])
	}
}

func TestServerCodeContextFileStateV0PersisteLeasesYCache(t *testing.T) {
	stateDir := t.TempDir()
	leasePath := filepath.Join(stateDir, serverCodeContextLeasesFileV0)
	cachePath := filepath.Join(stateDir, serverCodeContextCacheFileV0)
	started := time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC)
	store := newServerFileCodeContextToolLeaseStoreV0(leasePath)
	lease, err := store.BeginCodeContextToolLeaseV0(context.Background(), orquestacontext.CodeContextToolLeaseRequestV0{
		RepositoryRef:   "repo-ref-orquesta",
		QueryHash:       "query-hash-1",
		ToolRef:         "tool-ref-codebase-central",
		ProviderKind:    orquestacontext.CodeContextProviderKindCodebaseMCPV0,
		OwnerRef:        "owner-ref-codebase-1",
		StartedAt:       started.Format(time.RFC3339),
		LeaseTTLSeconds: 30,
	})
	if err != nil {
		t.Fatalf("BeginCodeContextToolLeaseV0: %v", err)
	}
	restartedStore := newServerFileCodeContextToolLeaseStoreV0(leasePath)
	leases, err := restartedStore.ListCodeContextToolLeasesV0(context.Background(), orquestacontext.CodeContextToolLeaseListFilterV0{
		Status: orquestacontext.CodeContextToolLeaseStatusActiveV0,
	})
	if err != nil {
		t.Fatalf("ListCodeContextToolLeasesV0: %v", err)
	}
	if len(leases) != 1 || leases[0].LeaseRef != lease.LeaseRef || leases[0].OwnerRef != "owner-ref-codebase-1" {
		t.Fatalf("leases=%+v", leases)
	}

	cache := newServerFileCodeContextCacheV0(cachePath)
	cache.SaveCodeContextResultV0("cache-key-1", orquestacontext.CodeContextResultV0{
		SchemaVersion: orquestacontext.CodeContextResultSchemaVersionV0,
		Estado:        orquestacontext.CodeContextEstadoOKV0,
		ProviderKind:  orquestacontext.CodeContextProviderKindFallbackRGV0,
		Results: []orquestacontext.CodeContextHitV0{{
			HitRef: "hit-ref-1",
			Path:   "modulos/demo/service.go",
			Line:   7,
		}},
	})
	restartedCache := newServerFileCodeContextCacheV0(cachePath)
	result, ok := restartedCache.LoadCodeContextResultV0("cache-key-1")
	if !ok || len(result.Results) != 1 || result.Results[0].Path != "modulos/demo/service.go" {
		t.Fatalf("cache ok=%v result=%+v", ok, result)
	}
}

func TestServerCodeContextToolWatchdogV0ParaLeaseExpiradoPorOwner(t *testing.T) {
	started := time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC)
	store := newServerFileCodeContextToolLeaseStoreV0(filepath.Join(t.TempDir(), serverCodeContextLeasesFileV0))
	lease, err := store.BeginCodeContextToolLeaseV0(context.Background(), orquestacontext.CodeContextToolLeaseRequestV0{
		RepositoryRef:   "repo-ref-orquesta",
		QueryHash:       "query-hash-1",
		ToolRef:         "tool-ref-codebase-central",
		ProviderKind:    orquestacontext.CodeContextProviderKindCodebaseMCPV0,
		OwnerRef:        "owner-ref-codebase-1",
		StartedAt:       started.Format(time.RFC3339),
		LeaseTTLSeconds: 1,
	})
	if err != nil {
		t.Fatalf("BeginCodeContextToolLeaseV0: %v", err)
	}
	stopper := &fakeCodeContextToolOwnerStopperV0{}
	watchdog := serverCodeContextToolWatchdogV0{
		Leases:   store,
		Finisher: store,
		Stopper:  stopper,
		Now:      func() time.Time { return started.Add(2 * time.Second) },
	}
	result, err := watchdog.RunOnceV0(context.Background())
	if err != nil {
		t.Fatalf("RunOnceV0: %v", err)
	}
	if result.Observed != 1 || result.Stopped != 1 || len(stopper.owners) != 1 || stopper.owners[0] != "owner-ref-codebase-1" {
		t.Fatalf("result=%+v owners=%+v", result, stopper.owners)
	}
	stopped, err := store.ListCodeContextToolLeasesV0(context.Background(), orquestacontext.CodeContextToolLeaseListFilterV0{
		Status: orquestacontext.CodeContextToolLeaseStatusStoppedV0,
	})
	if err != nil {
		t.Fatalf("List stopped: %v", err)
	}
	if len(stopped) != 1 || stopped[0].LeaseRef != lease.LeaseRef {
		t.Fatalf("stopped=%+v", stopped)
	}
}

type fakeCodeContextToolOwnerStopperV0 struct {
	owners []string
}

func (fake *fakeCodeContextToolOwnerStopperV0) StopCodeContextToolOwnerV0(
	ctx context.Context,
	ownerRef string,
) ([]string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	fake.owners = append(fake.owners, ownerRef)
	return []string{"evidence-ref-code-context-owner-stopped"}, nil
}

func TestServerRGCodeContextScopesV0RechazaRutasAbsolutasYPadres(t *testing.T) {
	got := serverRGCodeContextScopesV0([]string{
		"/tmp/no",
		"../no",
		"modulos/orquesta-mcp",
		"docs",
	})
	if len(got) != 2 ||
		got[0] != "modulos/orquesta-mcp" ||
		got[1] != "docs" {
		t.Fatalf("scopes=%+v", got)
	}
}

func TestServerLimitedBufferV0DescartaExcesoSinBloquearWriter(t *testing.T) {
	buffer := &serverLimitedBufferV0{maxBytes: 8}
	written, err := buffer.Write([]byte("1234567890"))
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if written != 10 {
		t.Fatalf("written=%d", written)
	}
	if buffer.Len() != 8 || buffer.String() != "12345678" || !buffer.overflow {
		t.Fatalf("buffer len=%d value=%q overflow=%v", buffer.Len(), buffer.String(), buffer.overflow)
	}
	written, err = buffer.Write([]byte("abcdef"))
	if err != nil {
		t.Fatalf("write2: %v", err)
	}
	if written != 6 || buffer.Len() != 8 {
		t.Fatalf("written=%d len=%d", written, buffer.Len())
	}
}
