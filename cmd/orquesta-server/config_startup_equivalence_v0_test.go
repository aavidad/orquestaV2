package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
)

func TestServerProjectConfigStartupResolutionEquivalenceV0(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, serverProjectConfigFileNameV0)
	raw := []byte(`{"schema_version":"orquesta_config.v0","codex_runtime":{"max_concurrency":2}}`)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	fromDir, ok, err := loadServerProjectConfigFileV0(root)
	if err != nil || !ok {
		t.Fatalf("dir err=%v ok=%v", err, ok)
	}
	fromPathLoad, resolved, err := resolveServerProjectConfigProductV0(root, path)
	if err != nil || resolved != path {
		t.Fatalf("path err=%v resolved=%q", err, resolved)
	}
	if !reflect.DeepEqual(fromDir, fromPathLoad.Config) {
		t.Fatalf("startup resolution differs: %#v %#v", fromDir, fromPathLoad.Config)
	}
}

func TestServerProjectConfigStartupFallsBackForZeroAndNegativeV0(t *testing.T) {
	fallback := codexRuntimeLimitsFromProjectConfigFileV0(serverProjectConfigFileV0{})
	for _, configured := range []int{0, -1} {
		configured := configured
		got := codexRuntimeLimitsFromProjectConfigFileV0(serverProjectConfigFileV0{
			CodexRuntime: serverProjectConfigCodexRuntimeV0{MaxBatchReady: &configured, MaxConcurrency: &configured},
		})
		if got != fallback {
			t.Fatalf("configured=%d startup limits=%+v want fallback=%+v", configured, got, fallback)
		}
	}
}

func TestServerProjectConfigStartupRetainsOneProductLoadForStackV0(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, serverProjectConfigFileNameV0)
	if err := os.WriteFile(path, []byte(`{
		"schema_version":"orquesta_config.v0",
		"server":{"addr":"127.0.0.1:9999"},
		"gemini_runtime":{"extra_args":["stable"]},
		"required_test_runner":{"allowed_commands":{"go":"/usr/bin/go"}}
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(envCodexProjectWorkDirV0, root)
	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatal(err)
	}
	loaded, ok := serverProjectConfigStartupLoadV0(root, config.ProjectConfigFilePath)
	if !ok || len(loaded.CanonicalBytes) == 0 || loaded.Revision == "" || config.ProjectConfigRevision != loaded.Revision || len(loaded.Catalog) == 0 {
		t.Fatalf("startup discarded product load: %#v ok=%v", loaded, ok)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	projectConfig, err := projectConfigForBuildStackV0(config)
	if err != nil || projectConfig.Server.Addr == nil || *projectConfig.Server.Addr != "127.0.0.1:9999" {
		t.Fatalf("stack did not consume retained startup load: config=%#v err=%v", projectConfig, err)
	}
	if fromDir := projectConfigFromProjectDirBestEffortV0(root); fromDir.Server.Addr == nil || *fromDir.Server.Addr != "127.0.0.1:9999" {
		t.Fatalf("project-dir consumer reread instead of using retained load: %#v", fromDir)
	}
	loaded.CanonicalBytes[0] ^= 0xff
	loaded.Catalog[0].Pointer = "/tampered"
	*loaded.Config.Server.Addr = "127.0.0.1:1"
	loaded.Config.GeminiRuntime.ExtraArgs[0] = "tampered"
	loaded.Config.RequiredTestRunner.AllowedCommands["go"] = "/tampered/go"
	stored, ok := serverProjectConfigStartupLoadV0(root, config.ProjectConfigFilePath)
	if !ok || stored.Catalog[0].Pointer == "/tampered" || reflect.DeepEqual(stored.CanonicalBytes, loaded.CanonicalBytes) ||
		*stored.Config.Server.Addr != "127.0.0.1:9999" || stored.Config.GeminiRuntime.ExtraArgs[0] != "stable" ||
		stored.Config.RequiredTestRunner.AllowedCommands["go"] != "/usr/bin/go" {
		t.Fatalf("startup load leaked mutable aliases: %#v", stored)
	}
	forgetServerProjectConfigStartupLoadV0(root, config.ProjectConfigFilePath, config.ProjectConfigRevision)
	if _, ok := serverProjectConfigStartupLoadV0(root, config.ProjectConfigFilePath); ok {
		t.Fatal("startup load survived explicit ownership release")
	}
	if err := os.WriteFile(path, []byte(`{"schema_version":"orquesta_config.v0","server":{"addr":"127.0.0.1:7777"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := projectConfigForBuildStackV0(config); err == nil {
		t.Fatal("composition reread a changed file after losing its revision-bound snapshot")
	}
}

func TestServerProjectConfigEffectiveProjectionDoesNotRereadAfterSnapshotV0(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, serverProjectConfigFileNameV0)
	if err := os.WriteFile(path, []byte(`{"schema_version":"orquesta_config.v0","server":{"addr":"127.0.0.1:9911"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(envCodexProjectWorkDirV0, root)
	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatal(err)
	}
	defer forgetServerProjectConfigStartupLoadV0(root, config.ProjectConfigFilePath, config.ProjectConfigRevision)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	effective := serverEffectiveConfigFromEnvV0(config)
	setting := effectiveSettingForTestV0(effective.Settings, envServerAddrV0)
	if setting.Value != "127.0.0.1:9911" || setting.Source != configSettingSourceConfigFileV0 {
		t.Fatalf("effective projection reread or lost snapshot: %+v", setting)
	}
}

func TestServerProjectConfigStartupReadsCanonicalFileExactlyOnceV0(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, serverProjectConfigFileNameV0)
	if err := os.WriteFile(path, []byte(`{"schema_version":"orquesta_config.v0","server":{"addr":"127.0.0.1:9922"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var reads atomic.Int64
	serverProjectConfigReadObserverV0.Lock()
	previous := serverProjectConfigReadObserverV0.observe
	serverProjectConfigReadObserverV0.observe = func(observed string) {
		if filepath.Clean(observed) == filepath.Clean(path) {
			reads.Add(1)
		}
	}
	serverProjectConfigReadObserverV0.Unlock()
	defer func() {
		serverProjectConfigReadObserverV0.Lock()
		serverProjectConfigReadObserverV0.observe = previous
		serverProjectConfigReadObserverV0.Unlock()
	}()
	t.Setenv(envCodexProjectWorkDirV0, root)
	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatal(err)
	}
	defer forgetServerProjectConfigStartupLoadV0(root, config.ProjectConfigFilePath, config.ProjectConfigRevision)
	if got := reads.Load(); got != 1 {
		t.Fatalf("startup read canonical config %d times; want exactly one", got)
	}
}

func TestServerProjectConfigStartupCacheSeparatesSamePathRevisionsV0(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, serverProjectConfigFileNameV0)
	loads := make([]serverProjectConfigLoadV0, 0, 2)
	for _, addr := range []string{"127.0.0.1:9101", "127.0.0.1:9102"} {
		value := addr
		config := serverProjectConfigFileV0{SchemaVersion: serverProjectConfigSchemaVersionV0}
		config.Server.Addr = &value
		canonical, err := canonicalServerProjectConfigV0(config)
		if err != nil {
			t.Fatal(err)
		}
		catalog, err := serverProjectConfigCatalogV0()
		if err != nil {
			t.Fatal(err)
		}
		loads = append(loads, serverProjectConfigLoadV0{Config: config, CanonicalBytes: canonical.Bytes, Revision: canonical.Revision, Catalog: catalog})
	}
	rememberServerProjectConfigStartupLoadV0(root, path, loads[0])
	rememberServerProjectConfigStartupLoadV0(root, path, loads[1])
	defer forgetServerProjectConfigStartupLoadV0(root, path, loads[0].Revision)
	defer forgetServerProjectConfigStartupLoadV0(root, path, loads[1].Revision)
	for index, load := range loads {
		got, ok := serverProjectConfigStartupLoadV0(root, path, load.Revision)
		if !ok || got.Revision != load.Revision || got.Config.Server.Addr == nil || *got.Config.Server.Addr != *load.Config.Server.Addr {
			t.Fatalf("revision %d crossed cache boundary: got=%#v", index, got)
		}
	}
	if _, ambiguous := serverProjectConfigStartupLoadV0(root, path); ambiguous {
		t.Fatal("revision-less lookup selected one of two concurrent snapshots")
	}
}

func TestServerProjectConfigStartupCacheCleansConcurrentEarlyErrorsV0(t *testing.T) {
	const workers = 8
	var wait sync.WaitGroup
	errs := make(chan error, workers)
	for index := 0; index < workers; index++ {
		root := t.TempDir()
		path := filepath.Join(root, serverProjectConfigFileNameV0)
		if err := os.WriteFile(path, []byte(`{"schema_version":"orquesta_config.v0","hermes_operator":{"enabled":true,"base_url":"ftp://invalid.example"}}`), 0o600); err != nil {
			t.Fatal(err)
		}
		wait.Add(1)
		go func(root, path string) {
			defer wait.Done()
			if _, err := serverConfigFromEnvWithProjectConfigPathV0(path); err == nil {
				errs <- fmt.Errorf("invalid startup accepted: %s", path)
				return
			}
			if _, leaked := serverProjectConfigStartupLoadV0(root, path); leaked {
				errs <- fmt.Errorf("early-error cache leaked: %s", path)
				return
			}
			errs <- nil
		}(root, path)
	}
	wait.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestServerProjectConfigStartupCacheRemainsOwnedAcrossBackendErrorUntilCommandReleaseV0(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, serverProjectConfigFileNameV0)
	if err := os.WriteFile(path, []byte(`{"schema_version":"orquesta_config.v0","goal_backend":{"kind":"unsupported-backend"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(envCodexProjectWorkDirV0, root)
	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := serverProjectConfigStartupLoadV0(root, path, config.ProjectConfigRevision); !ok {
		t.Fatal("startup did not retain snapshot before backend construction")
	}
	if _, err := buildRuntimeFromConfigV0(config); err == nil {
		t.Fatal("unsupported backend unexpectedly built")
	}
	if _, retained := serverProjectConfigStartupLoadV0(root, path, config.ProjectConfigRevision); !retained {
		t.Fatal("builder released command-owned startup snapshot on error")
	}
	releaseServerProjectConfigSnapshotV0(config)
	if _, retained := serverProjectConfigStartupLoadV0(root, path, config.ProjectConfigRevision); retained {
		t.Fatal("command release leaked startup snapshot after backend error")
	}
}

func TestServerProjectConfigSnapshotSurvivesBuildAndFileChangesUntilResidentLoopsV0(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, serverProjectConfigFileNameV0)
	stateDir := filepath.Join(root, "state")
	watchdogStateDir := filepath.Join(root, "broker-state")
	raw := fmt.Sprintf(`{
		"schema_version":"orquesta_config.v0",
		"server":{"state_dir":%q},
		"opes":{"base_url":"http://127.0.0.1:8787"},
		"opes_bridge":{"enabled":true,"confirm":true,"dry_run":true,"job_ref":"job-snapshot","input_ledger_disabled":true},
		"codebase_broker":{"watchdog_enabled":true,"state_dir":%q}
	}`, stateDir, watchdogStateDir)
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(envCodexProjectWorkDirV0, root)
	t.Setenv(envCodexRuntimeWorkDirV0, filepath.Join(root, "runtime"))
	t.Setenv(envCodexCommandV0, filepath.Join(root, "codex-bin"))
	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatal(err)
	}
	stack, err := buildStackFromEnvV0(config)
	if err != nil {
		releaseServerProjectConfigSnapshotV0(config)
		t.Fatalf("buildStackFromEnvV0: %v", err)
	}
	defer func() {
		_ = serverRequiredTestResourceShutdownHookFromStackV0(stack).ShutdownV0(context.Background())
	}()
	if err := os.WriteFile(path, []byte(`{"schema_version":"orquesta_config.v0","opes_bridge":{"enabled":false},"codebase_broker":{"watchdog_enabled":false}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	bridge, err := opesBridgeLoopConfigFromServerConfigV0(config, "http://127.0.0.1:9999")
	if err != nil || !bridge.Loop.Enabled || bridge.DrainConfig.JobRef != "job-snapshot" {
		t.Fatalf("bridge used mutated file: config=%+v err=%v", bridge, err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	watchdog, err := serverCodeContextToolWatchdogLoopConfigFromServerConfigV0(config)
	if err != nil || !watchdog.Loop.Enabled || watchdog.StateDir != watchdogStateDir {
		t.Fatalf("watchdog reread deleted file: config=%+v err=%v", watchdog, err)
	}
	releaseServerProjectConfigSnapshotV0(config)
	if _, retained := serverProjectConfigStartupLoadV0(root, path, config.ProjectConfigRevision); retained {
		t.Fatal("resident-loop lifecycle leaked startup snapshot")
	}
}

func TestServerProjectConfigCommandBoundariesReleaseSnapshotV0(t *testing.T) {
	commands := []struct {
		name string
		run  func(path string) int
	}{
		{name: "status", run: func(path string) int {
			return statusServerCommandV0([]string{"--config", path}, io.Discard, io.Discard)
		}},
		{name: "stop", run: func(path string) int {
			return stopServerCommandV0([]string{"--config", path}, io.Discard, io.Discard)
		}},
		{name: "start_error", run: func(path string) int {
			return startServerCommandConfiguredV0(serverCommandConfigOptionsV0{ConfigPath: path}, io.Discard, io.Discard)
		}},
	}
	for _, command := range commands {
		t.Run(command.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, serverProjectConfigFileNameV0)
			if err := os.WriteFile(path, []byte(`{"schema_version":"orquesta_config.v0"}`), 0o600); err != nil {
				t.Fatal(err)
			}
			t.Setenv(envCodexProjectWorkDirV0, root)
			t.Setenv(envCodexCommandV0, "codex-not-absolute")
			_ = command.run(path)
			if _, retained := serverProjectConfigStartupLoadV0(root, path); retained {
				t.Fatalf("%s leaked startup snapshot", command.name)
			}
		})
	}

	t.Run("run_status", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, serverProjectConfigFileNameV0)
		if err := os.WriteFile(path, []byte(`{"schema_version":"orquesta_config.v0"}`), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Setenv(envCodexProjectWorkDirV0, root)
		var stdout, stderr bytes.Buffer
		_ = runStatusCommandV0([]string{"--run-ref", "run-ref-release"}, &stdout, &stderr)
		if _, retained := serverProjectConfigStartupLoadV0(root, path); retained {
			t.Fatal("run-status leaked startup snapshot")
		}
	})
}

func TestServerProjectConfigRevisionlessDirectBuildDoesNotRetainFallbackV0(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, serverProjectConfigFileNameV0)
	if err := os.WriteFile(path, []byte(`{"schema_version":"orquesta_config.v0","codex_runtime":{"max_concurrency":2}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(envCodexProjectWorkDirV0, root)
	t.Setenv(envCodexCommandV0, filepath.Join(root, "codex-bin"))
	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatal(err)
	}
	releaseServerProjectConfigSnapshotV0(config)
	config.ProjectConfigRevision = ""
	stack, err := buildStackFromEnvV0(config)
	if err != nil {
		t.Fatalf("revision-less direct build: %v", err)
	}
	defer func() {
		_ = serverRequiredTestResourceShutdownHookFromStackV0(stack).ShutdownV0(context.Background())
	}()
	if _, retained := serverProjectConfigStartupLoadV0(root, path); retained {
		t.Fatal("revision-less direct build retained an ownerless fallback snapshot")
	}
}

func TestServerProjectConfigRevisionlessReleaseCannotStealExistingOwnerV0(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, serverProjectConfigFileNameV0)
	if err := os.WriteFile(path, []byte(`{"schema_version":"orquesta_config.v0"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(envCodexProjectWorkDirV0, root)
	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatal(err)
	}
	withoutRevision := config
	withoutRevision.ProjectConfigRevision = ""
	releaseServerProjectConfigSnapshotV0(withoutRevision)
	if _, retained := serverProjectConfigStartupLoadV0(root, path, config.ProjectConfigRevision); !retained {
		t.Fatal("revision-less release stole another command's exact snapshot")
	}
	releaseServerProjectConfigSnapshotV0(config)
	if _, retained := serverProjectConfigStartupLoadV0(root, path, config.ProjectConfigRevision); retained {
		t.Fatal("exact release did not remove snapshot")
	}
}
