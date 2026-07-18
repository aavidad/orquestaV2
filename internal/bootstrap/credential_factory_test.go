package bootstrap

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"orquesta/internal/adapters/agent/codex"
	credentiallocal "orquesta/internal/adapters/credentials/local"
	localruntime "orquesta/internal/adapters/system/local"
	"orquesta/internal/credentials"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

const (
	bootstrapCredentialInitial = "factory-test-secret"
	bootstrapCredentialRotated = "factory-test-secret-v2"
)

func init() {
	if len(os.Args) < 2 || os.Args[1] != "exec" {
		return
	}
	if err := runBootstrapCredentialCodexHelper(os.Args[2:]); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(91)
	}
	os.Exit(0)
}

func TestLocalCredentialStoreIntegratesWithCodexLaunchRotationAndRevocation(t *testing.T) {
	root, configPath := credentialFactoryFixture(t)
	snapshot, err := loadConfigSnapshot(context.Background(), configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	agent, err := productionAgentFactory(snapshot, localruntime.Clock{})
	if err != nil {
		t.Fatalf("productionAgentFactory: %v", err)
	}
	t.Cleanup(func() {
		if err := agent.Shutdown(context.Background()); err != nil {
			t.Errorf("Shutdown credential agent: %v", err)
		}
	})

	launchAndAwaitCredentialArtifact(t, agent, "local-credential-v1", "bootstrap-helper:v1", "artifact:credential-v1")

	store, err := credentiallocal.Open(credentiallocal.Options{
		Path: filepath.Join(root, "secrets", "credentials.json"), OwnerUID: os.Geteuid(),
		MaxStoreBytes: 1 << 20, Now: localruntime.Clock{}.Now,
	})
	if err != nil {
		t.Fatalf("Open local credential store for rotation: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("Close local credential store: %v", err)
		}
	})
	rotatedSecret, _ := credentials.NewSecret([]byte(bootstrapCredentialRotated))
	rotated, err := store.Rotate(context.Background(), credentials.RotateRequest{
		ActorRef: "actor:local-owner", RequestRef: "request:factory-rotate",
		CredentialRef: "credential:codex-primary", OwnerRef: "actor:local-owner", ExpectedVersion: 1, Material: rotatedSecret,
	})
	rotatedSecret.Destroy()
	if err != nil || rotated.Metadata.Version != 2 || rotated.Receipt.Version != 2 {
		t.Fatalf("Rotate local credential: result=%+v err=%v", rotated, err)
	}
	launchAndAwaitCredentialArtifact(t, agent, "local-credential-v2", "bootstrap-helper:v2", "artifact:credential-v2")

	revoked, err := store.Revoke(context.Background(), credentials.RevokeRequest{
		ActorRef: "actor:local-owner", RequestRef: "request:factory-revoke",
		CredentialRef: "credential:codex-primary", OwnerRef: "actor:local-owner", ExpectedVersion: 2, Reason: "test revocation",
	})
	if err != nil || !revoked.Metadata.Revoked || revoked.Receipt.Version != 2 {
		t.Fatalf("Revoke local credential: result=%+v err=%v", revoked, err)
	}
	revokedRequest := bootstrapCredentialAgentRequest(t, "local-credential-revoked", "bootstrap-helper:v2")
	if _, err := agent.Launch(context.Background(), revokedRequest); codex.ErrorCode(err) != codex.CodeCredentialUnavailable {
		t.Fatalf("revoked local credential Launch error=%v code=%q", err, codex.ErrorCode(err))
	}
	assertNoCredentialMaterialInFiles(t, filepath.Join(root, "work"), bootstrapCredentialInitial, bootstrapCredentialRotated)
	assertNoCredentialMaterialInFiles(t, filepath.Join(root, "secrets"), bootstrapCredentialInitial, bootstrapCredentialRotated)
}

func TestProductionCredentialStoreUsesEffectiveUIDAuthority(t *testing.T) {
	source, err := parser.ParseFile(token.NewFileSet(), "runtime.go", nil, 0)
	if err != nil {
		t.Fatalf("parse production bootstrap: %v", err)
	}
	for _, declaration := range source.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != "productionAgentFactory" {
			continue
		}
		validOwnerAuthority := false
		ast.Inspect(function.Body, func(node ast.Node) bool {
			field, ok := node.(*ast.KeyValueExpr)
			if !ok {
				return true
			}
			name, ok := field.Key.(*ast.Ident)
			if !ok || name.Name != "OwnerUID" {
				return true
			}
			call, ok := field.Value.(*ast.CallExpr)
			if !ok {
				return false
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return false
			}
			qualifier, qualifierOK := selector.X.(*ast.Ident)
			validOwnerAuthority = qualifierOK && qualifier.Name == "os" &&
				selector.Sel.Name == "Geteuid" && len(call.Args) == 0
			return false
		})
		if !validOwnerAuthority {
			t.Fatal("production CredentialStore owner authority must be os.Geteuid()")
		}
		return
	}
	t.Fatal("productionAgentFactory not found")
}

func TestProductionAgentFactoryPreservesNoCredentialRefCompatibility(t *testing.T) {
	root := t.TempDir()
	configPath := writeTestConfig(t, root)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	replaceTestConfigValue(t, configPath, "[runtime.codex]\ntimeout = \"1s\"",
		"[runtime.codex]\ncommand = "+strconv.Quote(executable)+"\ntimeout = \"1s\"",
	)
	snapshot, err := loadConfigSnapshot(context.Background(), configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	agent, err := productionAgentFactory(snapshot, localruntime.Clock{})
	if err != nil {
		t.Fatalf("productionAgentFactory without credential ref: %v", err)
	}
	t.Cleanup(func() { _ = agent.Shutdown(context.Background()) })
	if _, wrapped := agent.(*credentialAgent); wrapped {
		t.Fatalf("empty credential ref opened credential lifecycle wrapper: %T", agent)
	}
	if _, statErr := os.Lstat(filepath.Join(root, "secrets")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("empty credential ref touched credential/token directory: %v", statErr)
	}
	if err := agent.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}

func TestProductionAgentFactoryOwnsConfiguredCredentialStore(t *testing.T) {
	root, configPath := credentialFactoryFixture(t)
	snapshot, err := loadConfigSnapshot(context.Background(), configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	agent, err := productionAgentFactory(snapshot, localruntime.Clock{})
	if err != nil {
		t.Fatalf("productionAgentFactory: %v", err)
	}
	if _, ok := agent.(*credentialAgent); !ok {
		t.Fatalf("credential store lifecycle wrapper missing: %T", agent)
	}
	if err := agent.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	if err := agent.Shutdown(context.Background()); err != nil {
		t.Fatalf("second Shutdown: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "secrets", "credentials.json")); err != nil {
		t.Fatalf("credential document lost: %v", err)
	}
}

func TestProductionAgentFactoryClosesCredentialStoreWhenCodexConstructionFails(t *testing.T) {
	root, configPath := credentialFactoryFixture(t)
	badWork := filepath.Join(root, "bad-work")
	if err := os.Mkdir(badWork, 0o755); err != nil {
		t.Fatal(err)
	}
	replaceTestConfigValue(t, configPath,
		"work_root = "+strconv.Quote(filepath.Join(root, "work")),
		"work_root = "+strconv.Quote(badWork),
	)
	snapshot, err := loadConfigSnapshot(context.Background(), configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	before := openDescriptorCountForPath(t, filepath.Join(root, "secrets"))
	agent, err := productionAgentFactory(snapshot, localruntime.Clock{})
	if agent != nil || codex.ErrorCode(err) != codex.CodeWorkRootPermissions {
		t.Fatalf("factory failure agent=%v err=%v", agent, err)
	}
	if after := openDescriptorCountForPath(t, filepath.Join(root, "secrets")); after != before {
		t.Fatalf("credential store descriptors leaked: before=%d after=%d", before, after)
	}
}

func credentialFactoryFixture(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	configPath := writeTestConfig(t, root)
	secretsRoot := filepath.Join(root, "secrets")
	if err := os.Mkdir(secretsRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	store, err := credentiallocal.Open(credentiallocal.Options{
		Path: filepath.Join(secretsRoot, "credentials.json"), OwnerUID: os.Geteuid(),
		MaxStoreBytes: 1 << 20, Now: localruntime.Clock{}.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	secret, _ := credentials.NewSecret([]byte(bootstrapCredentialInitial))
	_, err = store.Create(context.Background(), credentials.CreateRequest{
		ActorRef: "actor:local-owner", RequestRef: "request:factory-create",
		CredentialRef: "credential:codex-primary", OwnerRef: "actor:local-owner",
		ScopeRefs: []credentials.ScopeRef{"project:default"}, PurposeRef: codex.ProviderRef, Material: secret,
	})
	secret.Destroy()
	if closeErr := store.Close(); err != nil || closeErr != nil {
		t.Fatalf("provision credential: create=%v close=%v", err, closeErr)
	}
	executable, _ := os.Executable()
	replaceTestConfigValue(t, configPath, "[runtime.codex]\ntimeout = \"1s\"",
		"[runtime.codex]\ncommand = "+strconv.Quote(executable)+"\ncredential_ref = \"credential:codex-primary\"\ntimeout = \"5s\"",
	)
	return root, configPath
}

func runBootstrapCredentialCodexHelper(arguments []string) error {
	var outputPath string
	for index := 0; index < len(arguments); index++ {
		if arguments[index] == "--output-last-message" && index+1 < len(arguments) {
			outputPath = arguments[index+1]
			break
		}
	}
	if outputPath == "" || !filepath.IsAbs(outputPath) {
		return errors.New("bootstrap credential helper output path missing")
	}
	prompt, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("read credential helper prompt: %w", err)
	}
	if bytes.Contains(prompt, []byte("bootstrap-helper:block")) {
		time.Sleep(30 * time.Second)
		return errors.New("bootstrap control helper was not stopped")
	}
	environment, err := os.ReadFile("/proc/self/environ")
	if err != nil {
		return fmt.Errorf("read credential helper environment: %w", err)
	}
	credential := ""
	for _, entry := range strings.Split(strings.TrimSuffix(string(environment), "\x00"), "\x00") {
		if strings.HasPrefix(entry, "CODEX_API_KEY=") {
			credential = strings.TrimPrefix(entry, "CODEX_API_KEY=")
			break
		}
	}
	var artifact string
	switch {
	case bytes.Contains(prompt, []byte("bootstrap-helper:v1")) && credential == bootstrapCredentialInitial:
		artifact = "artifact:credential-v1"
	case bytes.Contains(prompt, []byte("bootstrap-helper:v2")) && credential == bootstrapCredentialRotated:
		artifact = "artifact:credential-v2"
	default:
		return errors.New("bootstrap credential helper received wrong credential version")
	}
	payload, err := json.Marshal(map[string]string{"artifact": artifact})
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, payload, 0o600)
}

func launchAndAwaitCredentialArtifact(t *testing.T, agent AgentAdapter, suffix, objective, wanted string) {
	t.Helper()
	request := bootstrapCredentialAgentRequest(t, suffix, objective)
	receipt, err := agent.Launch(context.Background(), request)
	if err != nil {
		t.Fatalf("Launch(%s): %v", suffix, err)
	}
	if err := ports.ValidateAgentLaunchReceipt(request, receipt); err != nil {
		t.Fatalf("Launch receipt(%s): %v", suffix, err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		observation, err := agent.Observe(context.Background(), request.ExecutionRef)
		if err == nil && observation.Status == ports.AgentCompleted {
			if string(observation.Content) != wanted {
				t.Fatalf("credential artifact(%s)=%q want=%q", suffix, observation.Content, wanted)
			}
			return
		}
		if err == nil && observation.Status == ports.AgentFailed {
			t.Fatalf("credential execution(%s) failed: %+v", suffix, observation)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("credential execution(%s) did not finish", suffix)
}

func bootstrapCredentialAgentRequest(t *testing.T, suffix, objective string) ports.AgentLaunchRequest {
	t.Helper()
	executionRef, _ := goal.NewExecutionRef("execution:" + suffix)
	goalRef, _ := goal.NewGoalRef("goal:" + suffix)
	workItemRef, _ := goal.NewWorkItemRef("work-item:" + suffix)
	actorRef, _ := goal.NewActorRef("actor:local-owner")
	projectRef, _ := goal.NewProjectRef("project:default")
	return ports.AgentLaunchRequest{
		ExecutionRef: executionRef, GoalRef: goalRef, WorkItemRef: workItemRef,
		PlanGeneration: 1, AppSpecGeneration: 1, ExecutionAttempt: 1,
		SpecHash: strings.Repeat("0", 64), ActorRef: actorRef, ProjectRef: projectRef, Objective: objective,
		PhaseRef: "phase-instance:build", PhaseKey: "phase:build", PhaseTemplateRef: "phase-template:program",
		PhaseInputRefs: []string{"input:app-spec"}, PhaseCriterionRefs: []string{"criterion:tests-green"},
		RoleKey: "role:worker", SkillRefs: []string{"skill:go"}, ToolRefs: []string{"tool:go-test"},
		CapabilityRefs: []string{"capability:patch"}, WriteSet: []string{"internal/bootstrap"},
		OutputContract: string(goal.OutputContractEvidenceBundle), ArtifactMediaType: "text/plain",
		IdempotencyKey: "launch:" + suffix, MaxOutputBytes: 1024,
		BudgetDemand: testBudgetDemand(suffix), SecurityCriticality: governance.SecurityCriticalityNormal,
		ReasoningEffort: governance.ReasoningEffortMedium,
	}
}

func assertNoCredentialMaterialInFiles(t *testing.T, root string, materials ...string) {
	t.Helper()
	if err := filepath.WalkDir(root, func(filePath string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		payload, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}
		for _, material := range materials {
			signatures := [][]byte{
				[]byte(material),
				[]byte(base64.StdEncoding.EncodeToString([]byte(material))),
				[]byte(base64.RawURLEncoding.EncodeToString([]byte(material))),
				[]byte(hex.EncodeToString([]byte(material))),
			}
			for _, signature := range signatures {
				if bytes.Contains(payload, signature) {
					t.Fatalf("credential material persisted in %s", filePath)
				}
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("scan credential files: %v", err)
	}
}

func openDescriptorCountForPath(t *testing.T, root string) int {
	t.Helper()
	entries, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		t.Skipf("descriptor inspection unavailable: %v", err)
	}
	count := 0
	for _, entry := range entries {
		target, err := os.Readlink(filepath.Join("/proc/self/fd", entry.Name()))
		if err == nil && (target == root || filepath.Dir(target) == root) {
			count++
		}
	}
	return count
}
