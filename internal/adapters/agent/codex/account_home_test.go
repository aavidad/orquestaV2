//go:build linux

package codex

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

const (
	accountTestProfile          = "Codex1"
	accountTestAuthMaximumBytes = int64(1024 * 1024)
)

func TestAccountProfileLeaseIsExclusiveForAdapterLifetime(t *testing.T) {
	root, _ := secureAccountFixture(t, accountTestProfile)
	secondProfilePath := filepath.Join(root, "Codex2")
	if err := os.Mkdir(secondProfilePath, 0o700); err != nil {
		t.Fatal(err)
	}
	mustWritePrivate(t, filepath.Join(secondProfilePath, accountAuthFileName), []byte(`{"tokens":"second"}`))
	firstConfig := accountTestConfig(t, root, accountTestProfile)
	first, err := New(firstConfig)
	if err != nil {
		t.Fatalf("New(first) error = %v", err)
	}

	secondConfig := accountTestConfig(t, root, accountTestProfile)
	if second, err := New(secondConfig); second != nil || ErrorCode(err) != CodeAccountProfileUnavailable {
		t.Fatalf("New(second) adapter=%v error=%v code=%q", second, err, ErrorCode(err))
	}
	distinctConfig := accountTestConfig(t, root, "Codex2")
	distinct, err := New(distinctConfig)
	if err != nil {
		t.Fatalf("New(distinct profile) error = %v", err)
	}
	if err := distinct.Close(); err != nil {
		t.Fatalf("Close(distinct profile) error = %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("Close(first) error = %v", err)
	}
	third, err := New(secondConfig)
	if err != nil {
		t.Fatalf("New(after release) error = %v", err)
	}
	if err := third.Close(); err != nil {
		t.Fatalf("Close(after release) error = %v", err)
	}
}

func TestAccountProfileRequiresSingleExecution(t *testing.T) {
	root, _ := secureAccountFixture(t, accountTestProfile)
	config := accountTestConfig(t, root, accountTestProfile)
	config.MaxConcurrentExecutions = 2
	if adapter, err := New(config); adapter != nil || ErrorCode(err) != CodeAccountProfileInvalid {
		t.Fatalf("New() adapter=%v error=%v code=%q", adapter, err, ErrorCode(err))
	}
}

func TestAccountProfileRefIsCanonicalOpaqueAndStrict(t *testing.T) {
	root, _ := secureAccountFixture(t, accountTestProfile)
	derived, err := deriveAccountProfileRef(root, accountTestProfile)
	if err != nil {
		t.Fatal(err)
	}
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := deriveAccountProfileRef(canonicalRoot, accountTestProfile)
	if err != nil || derived != canonical || !validAccountProfileRef(derived) {
		t.Fatalf("canonical ref derived=%q canonical=%q error=%v", derived, canonical, err)
	}
	for _, invalid := range []string{
		"",
		"account-profile:Codex1",
		accountProfileRefPrefix,
		accountProfileRefPrefix + strings.Repeat("0", 63),
		accountProfileRefPrefix + strings.Repeat("0", 65),
		accountProfileRefPrefix + strings.Repeat("A", 64),
		accountProfileRefPrefix + strings.Repeat("G", 64),
	} {
		if validAccountProfileRef(invalid) {
			t.Fatalf("invalid profile ref accepted: %q", invalid)
		}
	}
}

func TestAccountProfileEnvironmentOverridesHomeExactlyOnce(t *testing.T) {
	root, profilePath := secureAccountFixture(t, accountTestProfile)
	config := accountTestConfig(t, root, accountTestProfile)
	adapter := openTestAdapter(t, config)

	environment, err := adapter.accountExecutionEnvironment([]string{
		"PATH=/usr/bin",
		"HOME=/daemon/home",
		"CODEX_HOME=/daemon/codex",
	})
	if err != nil {
		t.Fatalf("accountExecutionEnvironment() error = %v", err)
	}
	defer clearEnvironment(environment)
	want := map[string]string{"PATH": "/usr/bin", "HOME": profilePath, "CODEX_HOME": profilePath}
	if len(environment) != len(want) {
		t.Fatalf("environment = %q", environment)
	}
	for _, entry := range environment {
		name, value, found := strings.Cut(entry, "=")
		if !found || want[name] != value {
			t.Fatalf("unexpected environment entry %q in %q", entry, environment)
		}
		delete(want, name)
	}
	if len(want) != 0 {
		t.Fatalf("missing environment = %v", want)
	}
	if _, err := adapter.accountExecutionEnvironment([]string{"HOME=one", "HOME=two"}); ErrorCode(err) != CodeEnvironmentInvalid {
		t.Fatalf("duplicate HOME error=%v code=%q", err, ErrorCode(err))
	}
}

func TestAccountProfileShellPolicyIncludesHomesOnlyWhenBound(t *testing.T) {
	accountless := openTestAdapter(t, testConfig(t))
	if got := accountless.shellEnvironmentIncludeOnly(); got !=
		`shell_environment_policy.include_only=["CODEX_TEST_EXACT"]` {
		t.Fatalf("accountless shell policy = %s", got)
	}

	root, _ := secureAccountFixture(t, accountTestProfile)
	bound := openTestAdapter(t, accountTestConfig(t, root, accountTestProfile))
	if got := bound.shellEnvironmentIncludeOnly(); got !=
		`shell_environment_policy.include_only=["CODEX_HOME","CODEX_TEST_EXACT","HOME"]` {
		t.Fatalf("account-bound shell policy = %s", got)
	}
}

func TestAccountProfileRejectsUnsafeSources(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(t *testing.T, base, root, profilePath string)
	}{
		{name: "root permissions", mutate: func(t *testing.T, _, root, _ string) {
			t.Helper()
			mustChmod(t, root, 0o770)
		}},
		{name: "profile permissions", mutate: func(t *testing.T, _, _, profilePath string) {
			t.Helper()
			mustChmod(t, profilePath, 0o750)
		}},
		{name: "writable ancestor", mutate: func(t *testing.T, base, _, _ string) {
			t.Helper()
			mustChmod(t, base, 0o770)
		}},
		{name: "auth symlink", mutate: func(t *testing.T, _, _, profilePath string) {
			t.Helper()
			authPath := filepath.Join(profilePath, accountAuthFileName)
			if err := os.Remove(authPath); err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(profilePath, "target.json")
			mustWritePrivate(t, target, []byte(`{"tokens":"present"}`))
			if err := os.Symlink(target, authPath); err != nil {
				t.Skipf("symlink unsupported: %v", err)
			}
		}},
		{name: "auth hardlink", mutate: func(t *testing.T, _, _, profilePath string) {
			t.Helper()
			authPath := filepath.Join(profilePath, accountAuthFileName)
			linkPath := filepath.Join(profilePath, "auth-copy.json")
			if err := os.Link(authPath, linkPath); err != nil {
				t.Skipf("hardlink unsupported: %v", err)
			}
		}},
		{name: "auth permissions", mutate: func(t *testing.T, _, _, profilePath string) {
			t.Helper()
			mustChmod(t, filepath.Join(profilePath, accountAuthFileName), 0o640)
		}},
		{name: "auth malformed", mutate: func(t *testing.T, _, _, profilePath string) {
			t.Helper()
			mustWritePrivate(t, filepath.Join(profilePath, accountAuthFileName), []byte(`{"broken"`))
		}},
		{name: "auth empty object", mutate: func(t *testing.T, _, _, profilePath string) {
			t.Helper()
			mustWritePrivate(t, filepath.Join(profilePath, accountAuthFileName), []byte(`{}`))
		}},
		{name: "auth oversized", mutate: func(t *testing.T, _, _, profilePath string) {
			t.Helper()
			authPath := filepath.Join(profilePath, accountAuthFileName)
			file, err := os.OpenFile(authPath, os.O_WRONLY|os.O_TRUNC, 0o600)
			if err != nil {
				t.Fatal(err)
			}
			if err := file.Truncate(accountTestAuthMaximumBytes + 1); err != nil {
				_ = file.Close()
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "lock symlink", mutate: func(t *testing.T, _, _, profilePath string) {
			t.Helper()
			target := filepath.Join(profilePath, "lock-target")
			mustWritePrivate(t, target, []byte("lock"))
			if err := os.Symlink(target, filepath.Join(profilePath, accountProfileLockName)); err != nil {
				t.Skipf("symlink unsupported: %v", err)
			}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			base, root, profilePath := secureAccountFixtureWithBase(t, accountTestProfile)
			test.mutate(t, base, root, profilePath)
			config := accountTestConfig(t, root, accountTestProfile)
			if adapter, err := New(config); adapter != nil || ErrorCode(err) != CodeAccountProfileUnavailable {
				t.Fatalf("New() adapter=%v error=%v code=%q", adapter, err, ErrorCode(err))
			}
		})
	}
}

func TestAccountProfileRejectsCredentialStoreAuthority(t *testing.T) {
	root, _ := secureAccountFixture(t, accountTestProfile)
	config := accountTestConfig(t, root, accountTestProfile)
	config.CredentialRef = "credential:codex"
	if adapter, err := New(config); adapter != nil || ErrorCode(err) != CodeAccountProfileInvalid {
		t.Fatalf("New() adapter=%v error=%v code=%q", adapter, err, ErrorCode(err))
	}
}

func TestAccountProfileLaunchPersistsOpaqueBindingAndUsesPersistentHome(t *testing.T) {
	root, profilePath := secureAccountFixture(t, accountTestProfile)
	config := accountTestConfig(t, root, accountTestProfile)
	adapter, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	request := testRequest(t, "account-home", "helper:account-home helper:success", 1024)
	if _, err := adapter.Launch(context.Background(), request); err != nil {
		t.Fatalf("Launch() error = %v", err)
	}
	if observation := awaitTerminal(t, adapter, request.ExecutionRef); observation.Status != ports.AgentCompleted ||
		string(observation.Content) != "artifact:account-home-ok" {
		t.Fatalf("observation = %+v", observation)
	}
	runPath := filepath.Join(config.WorkRoot, filepath.FromSlash(executionPath(request.ExecutionRef)))
	journal, err := os.ReadFile(filepath.Join(runPath, requestFileName))
	if err != nil {
		t.Fatalf("ReadFile(request journal) error = %v", err)
	}
	var stored launchRecord
	if err := json.Unmarshal(journal, &stored); err != nil {
		t.Fatal(err)
	}
	if stored.AccountProfileRef != adapter.accountProfileRef() || !validAccountProfileRef(stored.AccountProfileRef) ||
		strings.Contains(string(journal), root) || strings.Contains(string(journal), profilePath) ||
		strings.Contains(string(journal), accountTestProfile) || strings.Contains(string(journal), "tokens") {
		t.Fatalf("journal binding leaked path/profile/secret or missed opaque ref: %s", journal)
	}
	boundHash, err := adapter.hashLaunchRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	if stored.RequestHash != boundHash {
		t.Fatalf("stored request hash=%q want profile-bound %q", stored.RequestHash, boundHash)
	}
	updatedAuth := []byte(`{"tokens":"refreshed-by-codex"}`)
	mustWritePrivate(t, filepath.Join(profilePath, accountAuthFileName), updatedAuth)
	if err := adapter.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	reopened, err := New(config)
	if err != nil {
		t.Fatalf("New(reopen) error = %v", err)
	}
	defer reopened.Close()
	got, err := os.ReadFile(filepath.Join(profilePath, accountAuthFileName))
	if err != nil || string(got) != string(updatedAuth) {
		t.Fatalf("persistent auth=%q error=%v", got, err)
	}
}

func TestAccountProfileRejectsAccountlessLegacyJournals(t *testing.T) {
	for _, schemaVersion := range []int{
		legacyStateSchemaVersion,
		intermediateStateSchemaVersion,
		accountlessStateSchemaVersion,
	} {
		t.Run(string(rune('0'+schemaVersion)), func(t *testing.T) {
			root, _ := secureAccountFixture(t, accountTestProfile)
			config := accountTestConfig(t, root, accountTestProfile)
			request := testRequest(t, "account-legacy", "helper:success", 1024)
			seedAccountlessLaunchRecord(t, config, request, schemaVersion)
			adapter, err := New(config)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			defer adapter.Close()
			if _, err := adapter.Launch(context.Background(), request); ErrorCode(err) != CodeAccountProfileUnavailable {
				t.Fatalf("Launch(V%d) error=%v code=%q", schemaVersion, err, ErrorCode(err))
			}
			if _, err := adapter.Observe(context.Background(), request.ExecutionRef); ErrorCode(err) != CodeAccountProfileUnavailable {
				t.Fatalf("Observe(V%d) error=%v code=%q", schemaVersion, err, ErrorCode(err))
			}
		})
	}
}

func TestAccountProfileV6ReplayRequiresNewV7Attempt(t *testing.T) {
	root, _ := secureAccountFixture(t, accountTestProfile)
	config := accountTestConfig(t, root, accountTestProfile)
	request := testRequest(t, "account-v6-v7-reasoning", "helper:account-home", 1024)
	request.ReasoningEffort = governance.ReasoningEffortHigh
	runPath := seedPersistedV6Launch(t, config, request)

	adapter := openTestAdapter(t, config)
	if _, err := adapter.Launch(context.Background(), request); ErrorCode(err) != CodeLegacyExecutionRequiresNewAttempt {
		t.Fatalf("Launch(V6 account replay) error=%v code=%q", err, ErrorCode(err))
	}
	observation := awaitTerminal(t, adapter, request.ExecutionRef)
	if observation.Status != ports.AgentFailed ||
		observation.ErrorCode != CodeExecutionInterrupted {
		t.Fatalf("V6 account replay observation=%+v", observation)
	}
	if _, err := os.Stat(filepath.Join(
		config.WorkRoot, filepath.FromSlash(runPath), launchUpgradeFileName,
	)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("account V6 replay created V7 binding: %v", err)
	}
}

func TestAccountProfileIdentitySeparatesRootsWithSameProfileName(t *testing.T) {
	firstRoot, _ := secureAccountFixture(t, accountTestProfile)
	secondRoot, _ := secureAccountFixture(t, accountTestProfile)
	firstConfig := accountTestConfig(t, firstRoot, accountTestProfile)
	secondConfig := accountTestConfig(t, secondRoot, accountTestProfile)
	first := openTestAdapter(t, firstConfig)
	second := openTestAdapter(t, secondConfig)
	request := testRequest(t, "account-root-identity", "helper:account-home", 1024)

	firstHash, err := first.hashLaunchRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	secondHash, err := second.hashLaunchRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	if first.accountProfileRef() == second.accountProfileRef() || firstHash == secondHash {
		t.Fatal("different canonical roots collapsed into one account identity")
	}
	for _, candidate := range []string{first.accountProfileRef(), second.accountProfileRef()} {
		if !validAccountProfileRef(candidate) ||
			strings.Contains(candidate, firstRoot) || strings.Contains(candidate, secondRoot) ||
			strings.Contains(candidate, accountTestProfile) {
			t.Fatalf("account profile ref is not opaque and canonical: %q", candidate)
		}
	}
	if _, err := first.Launch(context.Background(), request); err != nil {
		t.Fatalf("Launch(first) error = %v", err)
	}
	if _, err := second.Launch(context.Background(), request); err != nil {
		t.Fatalf("Launch(second) error = %v", err)
	}
	_ = awaitTerminal(t, first, request.ExecutionRef)
	_ = awaitTerminal(t, second, request.ExecutionRef)

	firstRecord := readAccountLaunchRecord(t, firstConfig, request)
	secondRecord := readAccountLaunchRecord(t, secondConfig, request)
	if firstRecord.AccountProfileRef == secondRecord.AccountProfileRef ||
		firstRecord.RequestHash == secondRecord.RequestHash {
		t.Fatalf("different roots produced identical journals: first=%+v second=%+v", firstRecord, secondRecord)
	}
}

func TestAccountProfileBindingRejectsSameNameFromDifferentRootOnLaunchObserveAndStop(t *testing.T) {
	firstRoot, _ := secureAccountFixture(t, accountTestProfile)
	secondRoot, _ := secureAccountFixture(t, accountTestProfile)
	firstConfig := accountTestConfig(t, firstRoot, accountTestProfile)
	request := testRequest(t, "account-binding", "helper:account-home", 1024)
	first, err := New(firstConfig)
	if err != nil {
		t.Fatal(err)
	}
	firstHash, err := first.hashLaunchRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	accountlessHash, err := hashLaunchRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	if firstHash == accountlessHash {
		t.Fatal("account-bound V6 hash equals accountless V6 hash")
	}
	firstReceipt, err := first.Launch(context.Background(), request)
	if err != nil {
		t.Fatalf("Launch(first) error = %v", err)
	}
	_ = awaitTerminal(t, first, request.ExecutionRef)
	if err := first.Close(); err != nil {
		t.Fatalf("Close(first) error = %v", err)
	}

	secondConfig := firstConfig
	secondConfig.AccountHomeRoot = secondRoot
	second, err := New(secondConfig)
	if err != nil {
		t.Fatalf("New(second) error = %v", err)
	}
	defer second.Close()
	secondHash, err := second.hashLaunchRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	if firstHash == secondHash {
		t.Fatal("same profile name under different roots produced the same V6 request hash")
	}
	if _, err := second.Launch(context.Background(), request); ErrorCode(err) != CodeAccountProfileUnavailable {
		t.Fatalf("Launch(second) error=%v code=%q", err, ErrorCode(err))
	}
	if _, err := second.Observe(context.Background(), request.ExecutionRef); ErrorCode(err) != CodeAccountProfileUnavailable {
		t.Fatalf("Observe(second) error=%v code=%q", err, ErrorCode(err))
	}
	stopRequest := stopRequestForLaunch(firstReceipt, ports.AgentStopCooperative, "stop:account-binding")
	if _, err := second.Stop(context.Background(), stopRequest); ErrorCode(err) != CodeAccountProfileUnavailable {
		t.Fatalf("Stop(second) error=%v code=%q", err, ErrorCode(err))
	}
}

func readAccountLaunchRecord(t *testing.T, config Config, request ports.AgentLaunchRequest) launchRecord {
	t.Helper()
	journalPath := filepath.Join(
		config.WorkRoot,
		filepath.FromSlash(executionPath(request.ExecutionRef)),
		requestFileName,
	)
	payload, err := os.ReadFile(journalPath)
	if err != nil {
		t.Fatal(err)
	}
	var record launchRecord
	if err := json.Unmarshal(payload, &record); err != nil {
		t.Fatal(err)
	}
	return record
}

func TestAccountProfileRemovalFailsBeforeReplay(t *testing.T) {
	root, profilePath := secureAccountFixture(t, accountTestProfile)
	config := accountTestConfig(t, root, accountTestProfile)
	adapter, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := adapter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(profilePath); err != nil {
		t.Fatal(err)
	}
	if reopened, err := New(config); reopened != nil || ErrorCode(err) != CodeAccountProfileUnavailable {
		t.Fatalf("New(removed profile) adapter=%v error=%v code=%q", reopened, err, ErrorCode(err))
	}
}

func TestExistingV7UpgradePreservesDurableV6CausalSource(t *testing.T) {
	config := testConfig(t)
	request := testRequest(t, "v6-v7-upgrade-chain", "helper:success", 1024)
	seedAccountlessLaunchRecord(t, config, request, legacyStateSchemaVersion)
	seeder, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	legacy, runPath, found, err := seeder.loadLaunchRecord(request.ExecutionRef)
	if err != nil || !found {
		t.Fatalf("load legacy found=%v error=%v", found, err)
	}
	v5 := accountlessV5Record(t, legacy, request)
	oldUpgrade := launchUpgradeRecord{
		SchemaVersion:       accountlessStateSchemaVersion,
		SourceSchemaVersion: legacy.SchemaVersion,
		SourceRequestHash:   legacy.RequestHash,
		Launch:              v5,
	}
	if created, err := seeder.publishJSON(runPath, legacyLaunchUpgradeFileName, oldUpgrade); err != nil || !created {
		t.Fatalf("publish old upgrade created=%v error=%v", created, err)
	}
	v6 := profileV6Record(t, v5, request)
	v6Upgrade := launchUpgradeRecord{
		SchemaVersion:       profileStateSchemaVersion,
		SourceSchemaVersion: v5.SchemaVersion,
		SourceRequestHash:   v5.RequestHash,
		Launch:              v6,
	}
	if created, err := seeder.publishJSON(runPath, profileLaunchUpgradeFileName, v6Upgrade); err != nil || !created {
		t.Fatalf("publish V6 upgrade created=%v error=%v", created, err)
	}
	currentHash, err := hashLaunchRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	v7 := v6
	v7.SchemaVersion = stateSchemaVersion
	v7.RequestHash = currentHash
	v7.ReasoningEffort = request.ReasoningEffort
	v7Upgrade := launchUpgradeRecord{
		SchemaVersion:       stateSchemaVersion,
		SourceSchemaVersion: v6.SchemaVersion,
		SourceRequestHash:   v6.RequestHash,
		Launch:              v7,
	}
	if created, err := seeder.publishJSON(
		runPath, launchUpgradeFileName, v7Upgrade,
	); err != nil || !created {
		t.Fatalf("publish preexisting V7 upgrade created=%v error=%v", created, err)
	}
	if err := seeder.Close(); err != nil {
		t.Fatal(err)
	}

	adapter := openTestAdapter(t, config)
	var upgrade launchUpgradeRecord
	found, err = adapter.readPrivateJSON(filepath.ToSlash(filepath.Join(runPath, launchUpgradeFileName)), &upgrade)
	if err != nil || !found {
		t.Fatalf("read V7 upgrade found=%v error=%v", found, err)
	}
	if upgrade.SourceSchemaVersion != profileStateSchemaVersion ||
		upgrade.SourceRequestHash != v6.RequestHash ||
		upgrade.Launch.SchemaVersion != stateSchemaVersion ||
		upgrade.Launch.ReasoningEffort != request.ReasoningEffort {
		t.Fatalf("V6 -> V7 causal chain lost: %+v", upgrade)
	}
}

func accountlessV5Record(t *testing.T, legacy launchRecord, request ports.AgentLaunchRequest) launchRecord {
	t.Helper()
	requestHash, err := hashV5LaunchRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	return launchRecord{
		SchemaVersion:         accountlessStateSchemaVersion,
		RequestHash:           requestHash,
		ExecutionRef:          legacy.ExecutionRef,
		ExecutionSessionRef:   request.SessionRef.String(),
		ExecutionWorkspaceRef: request.ExecutionWorkspaceRef.String(),
		ActorRef:              request.ActorRef.String(),
		ProjectRef:            request.ProjectRef.String(),
		GoalRef:               request.GoalRef.String(),
		WorkItemRef:           request.WorkItemRef.String(),
		PlanGeneration:        request.PlanGeneration,
		AppSpecGeneration:     request.AppSpecGeneration,
		ExecutionAttempt:      request.ExecutionAttempt,
		SpecHash:              legacy.SpecHash,
		ProviderRef:           legacy.ProviderRef,
		ModelRef:              DefaultModelRef,
		AgentRef:              AgentRef,
		ExternalRef:           legacy.ExternalRef,
		IdempotencyKey:        legacy.IdempotencyKey,
		AcceptedAt:            legacy.AcceptedAt,
		MaxOutputBytes:        legacy.MaxOutputBytes,
	}
}

func profileV6Record(t *testing.T, legacy launchRecord, request ports.AgentLaunchRequest) launchRecord {
	t.Helper()
	requestHash, err := hashV6LaunchRequest(request, "")
	if err != nil {
		t.Fatal(err)
	}
	record := accountlessV5Record(t, legacy, request)
	record.SchemaVersion = profileStateSchemaVersion
	record.RequestHash = requestHash
	return record
}

func seedAccountlessLaunchRecord(
	t *testing.T,
	config Config,
	request ports.AgentLaunchRequest,
	schemaVersion int,
) {
	t.Helper()
	seederConfig := config
	seederConfig.AccountHomeRoot = ""
	seederConfig.AccountProfile = ""
	seeder, err := New(seederConfig)
	if err != nil {
		t.Fatalf("New(seeder) error = %v", err)
	}
	runPath := executionPath(request.ExecutionRef)
	if err := seeder.ensurePrivateDirectory("executions"); err != nil {
		t.Fatal(err)
	}
	if err := seeder.ensurePrivateDirectory(runPath); err != nil {
		t.Fatal(err)
	}
	record := launchRecord{
		SchemaVersion:     schemaVersion,
		ExecutionRef:      request.ExecutionRef.String(),
		SpecHash:          request.SpecHash,
		ProviderRef:       ProviderRef,
		ExternalRef:       "codex:" + filepath.Base(runPath),
		IdempotencyKey:    request.IdempotencyKey,
		AcceptedAt:        seederConfig.Now(),
		MaxOutputBytes:    request.MaxOutputBytes,
		AccountProfileRef: "",
	}
	switch schemaVersion {
	case legacyStateSchemaVersion:
		record.RequestHash, err = hashLegacyLaunchRequest(request)
	case intermediateStateSchemaVersion:
		record.RequestHash, err = hashV4LaunchRequest(request)
		record.GoalRef = request.GoalRef.String()
		record.WorkItemRef = request.WorkItemRef.String()
		record.PlanGeneration = request.PlanGeneration
		record.AppSpecGeneration = request.AppSpecGeneration
		record.ExecutionAttempt = request.ExecutionAttempt
		record.ModelRef = DefaultModelRef
		record.AgentRef = AgentRef
	case accountlessStateSchemaVersion:
		record.RequestHash, err = hashV5LaunchRequest(request)
		record.ExecutionSessionRef = request.SessionRef.String()
		record.ExecutionWorkspaceRef = request.ExecutionWorkspaceRef.String()
		record.ActorRef = request.ActorRef.String()
		record.ProjectRef = request.ProjectRef.String()
		record.GoalRef = request.GoalRef.String()
		record.WorkItemRef = request.WorkItemRef.String()
		record.PlanGeneration = request.PlanGeneration
		record.AppSpecGeneration = request.AppSpecGeneration
		record.ExecutionAttempt = request.ExecutionAttempt
		record.ModelRef = DefaultModelRef
		record.AgentRef = AgentRef
	default:
		t.Fatalf("unsupported schema %d", schemaVersion)
	}
	if err != nil {
		t.Fatalf("hash(V%d) error = %v", schemaVersion, err)
	}
	if created, err := seeder.publishJSON(runPath, requestFileName, record); err != nil || !created {
		t.Fatalf("publishJSON(V%d) created=%v error=%v", schemaVersion, created, err)
	}
	if err := seeder.Close(); err != nil {
		t.Fatalf("Close(seeder) error = %v", err)
	}
}

func accountTestConfig(t *testing.T, root, profile string) Config {
	t.Helper()
	config := testConfig(t)
	config.AccountHomeRoot = root
	config.AccountProfile = profile
	config.AccountAuthMaxDocumentBytes = accountTestAuthMaximumBytes
	config.MaxConcurrentExecutions = 1
	return config
}

func secureAccountFixture(t *testing.T, profile string) (string, string) {
	t.Helper()
	_, root, profilePath := secureAccountFixtureWithBase(t, profile)
	return root, profilePath
}

func secureAccountFixtureWithBase(t *testing.T, profile string) (string, string, string) {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir() error = %v", err)
	}
	base, err := os.MkdirTemp(home, ".orquesta-account-test-")
	if err != nil {
		t.Fatalf("MkdirTemp(account fixture) error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(base, 0o700)
		if err := os.RemoveAll(base); err != nil {
			t.Errorf("RemoveAll(account fixture) error = %v", err)
		}
	})
	mustChmod(t, base, 0o700)
	root := filepath.Join(base, "homes")
	profilePath := filepath.Join(root, profile)
	if err := os.MkdirAll(profilePath, 0o700); err != nil {
		t.Fatalf("MkdirAll(profile) error = %v", err)
	}
	mustChmod(t, root, 0o700)
	mustChmod(t, profilePath, 0o700)
	mustWritePrivate(t, filepath.Join(profilePath, accountAuthFileName), []byte(`{"tokens":"present"}`))
	return base, root, profilePath
}

func mustChmod(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	if err := os.Chmod(path, mode); err != nil {
		t.Fatalf("Chmod(%s) error = %v", path, err)
	}
}

func mustWritePrivate(t *testing.T, path string, payload []byte) {
	t.Helper()
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

func TestAccountProfileAuthDocumentRemainsJSON(t *testing.T) {
	_, profilePath := secureAccountFixture(t, accountTestProfile)
	payload, err := os.ReadFile(filepath.Join(profilePath, accountAuthFileName))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(payload, &document); err != nil || len(document) == 0 {
		t.Fatalf("auth fixture invalid: %v", err)
	}
}
