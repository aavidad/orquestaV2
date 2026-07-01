package main

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	orquestacontext "orquesta/modulos/orquesta-context"
	orquestaserver "orquesta/modulos/orquesta-server"
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

func TestServerRGCodeContextProviderV0UsaFallbackGoSiRGAusente(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "modulos", "demo"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "modulos", "demo", "service.go"),
		[]byte("package demo\n\nfunc BrokerCentralSinRG() {}\n"),
		0o644,
	); err != nil {
		t.Fatalf("write: %v", err)
	}
	provider := serverRGCodeContextProviderV0{
		RootDir: root,
		Command: filepath.Join(t.TempDir(), "rg-no-existe"),
	}
	result, err := provider.QueryCodeContextV0(context.Background(), orquestacontext.CodeContextQueryV0{
		SchemaVersion: orquestacontext.CodeContextQuerySchemaVersionV0,
		RepositoryRef: "repo-ref-test",
		QueryKind:     orquestacontext.CodeContextQueryKindSearchV0,
		Query:         "BrokerCentralSinRG",
		Scope:         []string{"modulos"},
		MaxResults:    5,
		MaxBytes:      3000,
	})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if result.Estado != orquestacontext.CodeContextEstadoOKV0 ||
		result.ProviderKind != orquestacontext.CodeContextProviderKindFallbackRGV0 ||
		len(result.Results) != 1 {
		t.Fatalf("result=%+v", result)
	}
	if result.Results[0].Path != "modulos/demo/service.go" ||
		result.Results[0].Line != 3 ||
		!containsStringForTestV0(result.EvidenceRefs, "evidence-ref-code-context-go-fallback-v0") {
		t.Fatalf("result=%+v", result)
	}
}

func TestServerCodebaseMemoryCLIProviderV0ParseaSearchGraphYEscribeOwnerMarker(t *testing.T) {
	stateDir := t.TempDir()
	command := writeCodebaseMemoryFakeCLIForTestV0(t, `#!/bin/sh
echo 'level=info msg=mem.init budget_mb=1'
echo '{"total":1,"results":[{"name":"BrokerCentral","qualified_name":"repo.pkg.BrokerCentral","label":"Function","file_path":"modulos/demo/service.go","start_line":7,"rank":0.75}]}'
`)
	query := orquestacontext.CodeContextQueryV0{
		SchemaVersion:        orquestacontext.CodeContextQuerySchemaVersionV0,
		RequestRef:           "request-ref-codebase-cli",
		RepositoryRef:        "repo-ref-orquesta",
		QueryKind:            orquestacontext.CodeContextQueryKindSearchV0,
		Query:                "BrokerCentral",
		MaxResults:           3,
		MaxBytes:             3000,
		AllowExternalIndexer: true,
	}
	provider := serverCodebaseMemoryCLIProviderV0{
		RootDir:     t.TempDir(),
		ProjectName: "project-ref-orquesta-index",
		Command:     command,
		Registry:    newServerFileCodeContextToolOwnerRegistryV0(stateDir),
		Clock:       func() time.Time { return time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC) },
	}

	result, err := provider.QueryCodeContextV0(context.Background(), query)
	if err != nil {
		t.Fatalf("QueryCodeContextV0: %v", err)
	}
	if result.ProviderKind != orquestacontext.CodeContextProviderKindCodebaseMCPV0 ||
		len(result.Results) != 1 ||
		result.Results[0].Path != "modulos/demo/service.go" ||
		result.Results[0].Symbol != "BrokerCentral" {
		t.Fatalf("result=%+v", result)
	}
	marker, err := provider.Registry.ReadCodeContextToolOwnerMarkerV0(orquestacontext.CodeContextToolOwnerRefV0(query))
	if err != nil {
		t.Fatalf("Read owner marker: %v", err)
	}
	if marker.ProviderKind != orquestacontext.CodeContextProviderKindCodebaseMCPV0 ||
		marker.PID <= 0 ||
		marker.ActiveRequests != 1 {
		t.Fatalf("marker=%+v", marker)
	}
}

func TestCodeContextBrokerWiringFromEnvV0ConfiguraCodebaseMemoryCLIConOptInV0(t *testing.T) {
	stateDir := t.TempDir()
	root := t.TempDir()
	command := writeCodebaseMemoryFakeCLIForTestV0(t, `#!/bin/sh
echo '{"total":1,"results":[{"name":"CodebaseBroker","qualified_name":"repo.pkg.CodebaseBroker","label":"Function","file_path":"cmd/orquesta-server/code_context_broker_v0.go","start_line":21,"rank":1}]}'
`)
	t.Setenv(envCodebaseBrokerProviderKindV0, orquestacontext.CodeContextProviderKindCodebaseMCPV0)
	t.Setenv(envCodebaseBrokerExternalIndexerEnabledV0, "true")
	t.Setenv(envCodebaseBrokerStateDirV0, stateDir)
	t.Setenv(envCodebaseBrokerCommandV0, command)
	t.Setenv(envCodebaseBrokerProjectNameV0, "project-ref-orquesta-index")
	wiring := codeContextBrokerWiringFromEnvV0(orquestaserver.ConfigV0{ProjectWorkDir: root})
	query := orquestacontext.CodeContextQueryV0{
		SchemaVersion:        orquestacontext.CodeContextQuerySchemaVersionV0,
		RequestRef:           "request-ref-codebase-wiring",
		RepositoryRef:        "repo-ref-orquesta",
		QueryKind:            orquestacontext.CodeContextQueryKindSearchV0,
		Query:                "CodebaseBroker",
		MaxResults:           2,
		MaxBytes:             3000,
		AllowExternalIndexer: true,
	}

	result, err := wiring.Query.QueryCodeContextV0(context.Background(), query)
	if err != nil {
		t.Fatalf("QueryCodeContextV0: %v", err)
	}
	if result.Estado != orquestacontext.CodeContextEstadoOKV0 ||
		result.ProviderKind != orquestacontext.CodeContextProviderKindCodebaseMCPV0 ||
		len(result.Results) != 1 {
		t.Fatalf("result=%+v", result)
	}
	completed, err := wiring.ToolLeases.ListCodeContextToolLeasesV0(context.Background(), orquestacontext.CodeContextToolLeaseListFilterV0{
		Status: orquestacontext.CodeContextToolLeaseStatusCompletedV0,
	})
	if err != nil {
		t.Fatalf("List leases: %v", err)
	}
	if len(completed) != 1 || completed[0].OwnerRef != orquestacontext.CodeContextToolOwnerRefV0(query) {
		t.Fatalf("completed=%+v", completed)
	}
}

func TestServerCodebaseMemoryCLIProviderV0RealSmokeOptIn(t *testing.T) {
	if os.Getenv("ORQUESTA_CODEBASE_MEMORY_REAL_SMOKE") != "1" {
		t.Skip("smoke real opt-in")
	}
	command, err := exec.LookPath("codebase-memory-mcp")
	if err != nil {
		t.Skip("codebase-memory-mcp no disponible")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("abs root: %v", err)
	}
	stateDir := t.TempDir()
	leases := newServerFileCodeContextToolLeaseStoreV0(filepath.Join(stateDir, serverCodeContextLeasesFileV0))
	query := orquestacontext.CodeContextQueryV0{
		SchemaVersion:        orquestacontext.CodeContextQuerySchemaVersionV0,
		RequestRef:           "request-ref-codebase-real-smoke",
		RepositoryRef:        "repo-ref-orquesta",
		QueryKind:            orquestacontext.CodeContextQueryKindSearchV0,
		Query:                "CodeContextBrokerV0",
		MaxResults:           2,
		MaxBytes:             4000,
		AllowExternalIndexer: true,
	}
	broker := orquestacontext.NewCodeContextBrokerV0(orquestacontext.CodeContextBrokerConfigV0{
		Provider: serverCodebaseMemoryCLIProviderV0{
			RootDir:     root,
			ProjectName: serverCodebaseMemoryProjectNameV0(root, ""),
			Command:     command,
			Registry:    newServerFileCodeContextToolOwnerRegistryV0(stateDir),
		},
		ProviderRef:            "provider-ref-orquesta-code-context-central",
		ProviderKind:           orquestacontext.CodeContextProviderKindCodebaseMCPV0,
		ExternalIndexerEnabled: true,
		ToolLeasePort:          leases,
		Timeout:                10 * time.Second,
	})

	result, err := broker.QueryCodeContextV0(context.Background(), query)
	if err != nil {
		t.Fatalf("QueryCodeContextV0: %v", err)
	}
	if result.Estado != orquestacontext.CodeContextEstadoOKV0 ||
		result.ProviderKind != orquestacontext.CodeContextProviderKindCodebaseMCPV0 ||
		len(result.Results) == 0 {
		t.Fatalf("result=%+v", result)
	}
	completed, err := leases.ListCodeContextToolLeasesV0(context.Background(), orquestacontext.CodeContextToolLeaseListFilterV0{
		Status: orquestacontext.CodeContextToolLeaseStatusCompletedV0,
	})
	if err != nil {
		t.Fatalf("List leases: %v", err)
	}
	if len(completed) != 1 {
		t.Fatalf("completed=%+v", completed)
	}
	registry := newServerFileCodeContextToolOwnerRegistryV0(stateDir)
	marker, err := registry.ReadCodeContextToolOwnerMarkerV0(orquestacontext.CodeContextToolOwnerRefV0(query))
	if err != nil {
		t.Fatalf("Read marker: %v", err)
	}
	if marker.PID <= 0 || processAliveV0(marker.PID) {
		t.Fatalf("marker pid vivo o invalido: %+v", marker)
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

func TestServerCodeContextToolWatchdogV0UsaOwnerMarkerConPeticionesActivas(t *testing.T) {
	started := time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC)
	stateDir := t.TempDir()
	store := newServerFileCodeContextToolLeaseStoreV0(filepath.Join(stateDir, serverCodeContextLeasesFileV0))
	lease, err := store.BeginCodeContextToolLeaseV0(context.Background(), orquestacontext.CodeContextToolLeaseRequestV0{
		RepositoryRef:   "repo-ref-orquesta",
		QueryHash:       "query-hash-active-owner",
		ToolRef:         "tool-ref-codebase-central",
		ProviderKind:    orquestacontext.CodeContextProviderKindCodebaseMCPV0,
		OwnerRef:        "owner-ref-codebase-active",
		StartedAt:       started.Format(time.RFC3339),
		LeaseTTLSeconds: 1,
	})
	if err != nil {
		t.Fatalf("BeginCodeContextToolLeaseV0: %v", err)
	}
	registry := newServerFileCodeContextToolOwnerRegistryV0(stateDir)
	if err := registry.WriteCodeContextToolOwnerMarkerV0(serverCodeContextToolOwnerMarkerV0{
		OwnerRef:        lease.OwnerRef,
		ToolRef:         lease.ToolRef,
		ProviderKind:    orquestacontext.CodeContextProviderKindCodebaseMCPV0,
		PID:             os.Getpid(),
		StartedAt:       started.Format(time.RFC3339),
		LastHeartbeatAt: started.Add(time.Second).Format(time.RFC3339),
		CPUPercent:      95,
		ActiveRequests:  1,
		EvidenceRefs:    []string{"evidence-ref-codebase-owner-marker"},
	}); err != nil {
		t.Fatalf("WriteCodeContextToolOwnerMarkerV0: %v", err)
	}
	stopper := &fakeCodeContextToolOwnerStopperV0{}
	watchdog := serverCodeContextToolWatchdogV0{
		Leases:   store,
		Finisher: store,
		Stopper:  stopper,
		Observer: serverFileCodeContextToolOwnerObserverV0{Registry: registry},
		Now:      func() time.Time { return started.Add(2 * time.Second) },
	}
	result, err := watchdog.RunOnceV0(context.Background())
	if err != nil {
		t.Fatalf("RunOnceV0: %v", err)
	}
	if result.Observed != 1 || result.Stopped != 0 || len(stopper.owners) != 0 {
		t.Fatalf("result=%+v owners=%+v", result, stopper.owners)
	}
	active, err := store.ListCodeContextToolLeasesV0(context.Background(), orquestacontext.CodeContextToolLeaseListFilterV0{
		Status: orquestacontext.CodeContextToolLeaseStatusActiveV0,
	})
	if err != nil {
		t.Fatalf("List active: %v", err)
	}
	if len(active) != 1 || active[0].LeaseRef != lease.LeaseRef {
		t.Fatalf("active=%+v", active)
	}
}

func TestServerCodeContextToolWatchdogV0OwnerMarkerAusenteNoBloqueaOtrosLeases(t *testing.T) {
	started := time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC)
	stateDir := t.TempDir()
	store := newServerFileCodeContextToolLeaseStoreV0(filepath.Join(stateDir, serverCodeContextLeasesFileV0))
	missingMarkerLease, err := store.BeginCodeContextToolLeaseV0(context.Background(), orquestacontext.CodeContextToolLeaseRequestV0{
		RepositoryRef:   "repo-ref-orquesta",
		QueryHash:       "query-hash-missing-marker",
		ToolRef:         "tool-ref-codebase-central",
		ProviderKind:    orquestacontext.CodeContextProviderKindCodebaseMCPV0,
		OwnerRef:        "owner-ref-codebase-missing-marker",
		StartedAt:       started.Format(time.RFC3339),
		LeaseTTLSeconds: 1,
	})
	if err != nil {
		t.Fatalf("Begin missing marker lease: %v", err)
	}
	validMarkerLease, err := store.BeginCodeContextToolLeaseV0(context.Background(), orquestacontext.CodeContextToolLeaseRequestV0{
		RepositoryRef:   "repo-ref-orquesta",
		QueryHash:       "query-hash-valid-marker",
		ToolRef:         "tool-ref-codebase-central",
		ProviderKind:    orquestacontext.CodeContextProviderKindCodebaseMCPV0,
		OwnerRef:        "owner-ref-codebase-valid-marker",
		StartedAt:       started.Format(time.RFC3339),
		LeaseTTLSeconds: 1,
	})
	if err != nil {
		t.Fatalf("Begin valid marker lease: %v", err)
	}
	registry := newServerFileCodeContextToolOwnerRegistryV0(stateDir)
	if err := registry.WriteCodeContextToolOwnerMarkerV0(serverCodeContextToolOwnerMarkerV0{
		OwnerRef:     validMarkerLease.OwnerRef,
		ToolRef:      validMarkerLease.ToolRef,
		ProviderKind: orquestacontext.CodeContextProviderKindCodebaseMCPV0,
		StartedAt:    started.Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("Write valid marker: %v", err)
	}
	watchdog := serverCodeContextToolWatchdogV0{
		Leases:   store,
		Finisher: store,
		Stopper:  serverFileCodeContextToolOwnerStopperV0{Registry: registry},
		Observer: serverFileCodeContextToolOwnerObserverV0{Registry: registry},
		Now:      func() time.Time { return started.Add(2 * time.Second) },
	}

	result, err := watchdog.RunOnceV0(context.Background())
	if err != nil {
		t.Fatalf("RunOnceV0: %v", err)
	}
	if result.Observed != 2 || result.Stopped != 2 || result.Errors != 1 {
		t.Fatalf("result=%+v", result)
	}
	if !codeContextEvidenceContainsForTestV0(result.Evidence, "evidence-ref-code-context-owner-marker-unavailable") {
		t.Fatalf("evidence=%+v", result.Evidence)
	}
	stopped, err := store.ListCodeContextToolLeasesV0(context.Background(), orquestacontext.CodeContextToolLeaseListFilterV0{
		Status: orquestacontext.CodeContextToolLeaseStatusStoppedV0,
	})
	if err != nil {
		t.Fatalf("List stopped: %v", err)
	}
	if len(stopped) != 2 {
		t.Fatalf("stopped=%+v", stopped)
	}
	if stopped[0].LeaseRef != missingMarkerLease.LeaseRef && stopped[1].LeaseRef != missingMarkerLease.LeaseRef {
		t.Fatalf("missing marker lease no parado: %+v", stopped)
	}
	if stopped[0].LeaseRef != validMarkerLease.LeaseRef && stopped[1].LeaseRef != validMarkerLease.LeaseRef {
		t.Fatalf("valid marker lease no procesado: %+v", stopped)
	}
}

func TestServerCodeContextToolWatchdogLoopConfigV0ExigeStateDirOptIn(t *testing.T) {
	t.Setenv(envCodebaseBrokerWatchdogEnabledV0, "true")
	t.Setenv(envCodebaseBrokerStateDirV0, "")

	_, err := serverCodeContextToolWatchdogLoopConfigFromEnvV0()

	if err == nil || err.Error() != "codebase_broker_state_dir_required" {
		t.Fatalf("err=%v", err)
	}
}

func TestRunServerCodeContextToolWatchdogLoopAsyncV0DesactivadoNoProyectaBridgeV0(t *testing.T) {
	events := 0
	config := serverCodeContextToolWatchdogLoopConfigV0{
		Loop: externalBridgeLoopConfigV0{
			Observer: func(context.Context, externalBridgeLoopEventV0) {
				events++
			},
		},
	}

	done := runServerCodeContextToolWatchdogLoopAsyncV0(context.Background(), config, io.Discard)

	if done != nil {
		t.Fatalf("done=%v, want nil para loop desactivado", done)
	}
	if events != 0 {
		t.Fatalf("events=%d, want 0", events)
	}
}

func TestRunServerCodeContextToolWatchdogLoopAsyncV0ParaLeaseExpiradoSinPIDV0(t *testing.T) {
	started := time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC)
	stateDir := t.TempDir()
	store := newServerFileCodeContextToolLeaseStoreV0(filepath.Join(stateDir, serverCodeContextLeasesFileV0))
	lease, err := store.BeginCodeContextToolLeaseV0(context.Background(), orquestacontext.CodeContextToolLeaseRequestV0{
		RepositoryRef:   "repo-ref-orquesta",
		QueryHash:       "query-hash-loop",
		ToolRef:         "tool-ref-codebase-central",
		ProviderKind:    orquestacontext.CodeContextProviderKindCodebaseMCPV0,
		OwnerRef:        "owner-ref-codebase-loop",
		StartedAt:       started.Format(time.RFC3339),
		LeaseTTLSeconds: 1,
	})
	if err != nil {
		t.Fatalf("BeginCodeContextToolLeaseV0: %v", err)
	}
	registry := newServerFileCodeContextToolOwnerRegistryV0(stateDir)
	if err := registry.WriteCodeContextToolOwnerMarkerV0(serverCodeContextToolOwnerMarkerV0{
		OwnerRef:     "owner-ref-codebase-loop",
		ToolRef:      "tool-ref-codebase-central",
		ProviderKind: orquestacontext.CodeContextProviderKindCodebaseMCPV0,
		StartedAt:    started.Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("WriteCodeContextToolOwnerMarkerV0: %v", err)
	}
	config := serverCodeContextToolWatchdogLoopConfigV0{
		StateDir: stateDir,
		Loop: externalBridgeLoopConfigV0{
			Enabled:       true,
			Component:     codeContextToolWatchdogLoopComponentV0,
			ResultField:   codeContextToolWatchdogLoopResultFieldV0,
			InitialDelay:  0,
			Interval:      time.Hour,
			MaxTicks:      1,
			EffectTimeout: time.Second,
		},
	}

	done := runServerCodeContextToolWatchdogLoopAsyncV0(context.Background(), config, io.Discard)
	if !waitServerCodeContextToolWatchdogLoopDoneV0(context.Background(), config, done) {
		t.Fatalf("watchdog loop no termino")
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

func TestServerFileCodeContextToolOwnerStopperV0UsaMarkerSeguro(t *testing.T) {
	stateDir := t.TempDir()
	started := time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC)
	registry := newServerFileCodeContextToolOwnerRegistryV0(stateDir)
	if err := registry.WriteCodeContextToolOwnerMarkerV0(serverCodeContextToolOwnerMarkerV0{
		OwnerRef:     "owner-ref-codebase-stop",
		ToolRef:      "tool-ref-codebase-central",
		ProviderKind: orquestacontext.CodeContextProviderKindCodebaseMCPV0,
		PID:          os.Getpid(),
		StartedAt:    started.Format(time.RFC3339),
		CPUPercent:   88,
		EvidenceRefs: []string{"evidence-ref-codebase-owner-marker"},
	}); err != nil {
		t.Fatalf("WriteCodeContextToolOwnerMarkerV0: %v", err)
	}
	var gotPID int
	var gotSignal syscall.Signal
	stopped := false
	stopper := serverFileCodeContextToolOwnerStopperV0{
		Registry: registry,
		Signal: func(pid int, signal syscall.Signal) error {
			gotPID = pid
			gotSignal = signal
			stopped = true
			return nil
		},
		Alive:    func(int) bool { return !stopped },
		TermWait: time.Millisecond,
		KillWait: time.Millisecond,
	}
	evidence, err := stopper.StopCodeContextToolOwnerV0(context.Background(), "owner-ref-codebase-stop")
	if err != nil {
		t.Fatalf("StopCodeContextToolOwnerV0: %v", err)
	}
	if gotPID != os.Getpid() || gotSignal != syscall.SIGTERM {
		t.Fatalf("pid=%d signal=%v", gotPID, gotSignal)
	}
	if !codeContextEvidenceContainsForTestV0(evidence, "evidence-ref-code-context-owner-stop-signal-sent") {
		t.Fatalf("evidence=%+v", evidence)
	}
	if !codeContextEvidenceContainsForTestV0(evidence, "evidence-ref-code-context-owner-stopped-after-term") {
		t.Fatalf("evidence=%+v", evidence)
	}
}

func TestServerFileCodeContextToolOwnerStopperV0EscalaSiProcesoSigueVivo(t *testing.T) {
	stateDir := t.TempDir()
	started := time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC)
	registry := newServerFileCodeContextToolOwnerRegistryV0(stateDir)
	if err := registry.WriteCodeContextToolOwnerMarkerV0(serverCodeContextToolOwnerMarkerV0{
		OwnerRef:     "owner-ref-codebase-kill",
		ToolRef:      "tool-ref-codebase-central",
		ProviderKind: orquestacontext.CodeContextProviderKindCodebaseMCPV0,
		PID:          424242,
		StartedAt:    started.Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("WriteCodeContextToolOwnerMarkerV0: %v", err)
	}
	var signals []syscall.Signal
	killed := false
	stopper := serverFileCodeContextToolOwnerStopperV0{
		Registry: registry,
		Signal: func(_ int, signal syscall.Signal) error {
			signals = append(signals, signal)
			if signal == syscall.SIGKILL {
				killed = true
			}
			return nil
		},
		Alive:    func(int) bool { return !killed },
		TermWait: time.Millisecond,
		KillWait: time.Millisecond,
	}

	evidence, err := stopper.StopCodeContextToolOwnerV0(context.Background(), "owner-ref-codebase-kill")
	if err != nil {
		t.Fatalf("StopCodeContextToolOwnerV0: %v", err)
	}
	if len(signals) != 2 || signals[0] != syscall.SIGTERM || signals[1] != syscall.SIGKILL {
		t.Fatalf("signals=%+v", signals)
	}
	if !codeContextEvidenceContainsForTestV0(evidence, "evidence-ref-code-context-owner-kill-signal-sent") ||
		!codeContextEvidenceContainsForTestV0(evidence, "evidence-ref-code-context-owner-stopped-after-kill") {
		t.Fatalf("evidence=%+v", evidence)
	}
}

func TestServerFileCodeContextToolOwnerRegistryV0RechazaOwnerInseguro(t *testing.T) {
	registry := newServerFileCodeContextToolOwnerRegistryV0(t.TempDir())
	err := registry.WriteCodeContextToolOwnerMarkerV0(serverCodeContextToolOwnerMarkerV0{
		OwnerRef:     "../owner-ref-codebase-bad",
		ToolRef:      "tool-ref-codebase-central",
		ProviderKind: orquestacontext.CodeContextProviderKindCodebaseMCPV0,
	})
	if err == nil {
		t.Fatalf("WriteCodeContextToolOwnerMarkerV0 acepto owner_ref inseguro")
	}
}

func codeContextEvidenceContainsForTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func writeCodebaseMemoryFakeCLIForTestV0(t *testing.T, script string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "codebase-memory-mcp-fake")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake cli: %v", err)
	}
	return path
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
