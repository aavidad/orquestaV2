package acceptance_test

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"orquesta/internal/application"
)

const v13FixturePath = "acceptance/fixtures/v13_mailbox.json"
const v13TrustedBaseGitCommitOID = "ed2375ec0731864514a8e6f94938c18bfbaedadf"
const v13ProductDeltaBaseGitCommitOID = "b2af1e78f7a024d29a747e4485e619464b5dd12e"
const v13ProductDeltaSealedGitCommitOID = "a8bf28a2acb3a246d8c1a3c1cedea38d240f370a"

type v13Fixture struct {
	SchemaVersion                  int                     `json:"schema_version"`
	ReceiptSchemaVersion           int                     `json:"receipt_schema_version"`
	ContractID                     string                  `json:"contract_id"`
	TrustedBaseGitCommitOID        string                  `json:"trusted_base_git_commit_oid"`
	ProductDeltaBaseGitCommitOID   string                  `json:"product_delta_base_git_commit_oid"`
	ProductDeltaSealedGitCommitOID string                  `json:"product_delta_sealed_git_commit_oid"`
	Command                        string                  `json:"command"`
	ExecutionArgv                  []string                `json:"execution_argv"`
	OutputPath                     string                  `json:"output_path"`
	ReceiptPath                    string                  `json:"receipt_path"`
	CandidateSubjects              []string                `json:"candidate_subjects"`
	OwnedCapabilityIDs             []string                `json:"owned_capability_ids"`
	DeferredCapabilities           []v13DeferredCapability `json:"deferred_capabilities"`
	DeferredSurfaces               []string                `json:"deferred_surfaces"`
	RequiredUseCases               []string                `json:"required_use_cases"`
	RequiredRepositoryMethods      []string                `json:"required_repository_methods"`
	ForbiddenPrivateAuthorities    []string                `json:"forbidden_private_authorities"`
	Scenario                       v13Scenario             `json:"scenario"`
	Assertions                     []string                `json:"assertions"`
}

type v13DeferredCapability struct {
	ID                 string `json:"id"`
	Owner              string `json:"owner"`
	AcceptanceContract string `json:"acceptance_contract"`
}

type v13Scenario struct {
	BaseTime              string   `json:"base_time"`
	ClaimLease            string   `json:"claim_lease"`
	ReclaimAfter          string   `json:"reclaim_after"`
	ProjectRef            string   `json:"project_ref"`
	GoalRef               string   `json:"goal_ref"`
	PlanGeneration        uint64   `json:"plan_generation"`
	ParentWorkItemRef     string   `json:"parent_work_item_ref"`
	ChildWorkItemRefs     []string `json:"child_work_item_refs"`
	SourcePrincipalRef    string   `json:"source_principal_ref"`
	SourceWorkItemRef     string   `json:"source_work_item_ref"`
	SourceExecutionRef    string   `json:"source_execution_ref"`
	RecipientPrincipalRef string   `json:"recipient_principal_ref"`
	RecipientWorkItemRef  string   `json:"recipient_work_item_ref"`
	RecipientExecutionRef string   `json:"recipient_execution_ref"`
	SuccessorExecutionRef string   `json:"successor_execution_ref"`
	MessageKinds          []string `json:"message_kinds"`
	Lifecycle             []string `json:"lifecycle"`
	TerminalResolutions   []string `json:"terminal_resolutions"`
	Summary               string   `json:"summary"`
	ArtifactRefs          []string `json:"artifact_refs"`
	ClaimContenders       int      `json:"claim_contenders"`
}

func TestAcceptanceV13Mailbox(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v13Fixture](t, filepath.Join(repositoryRoot, filepath.FromSlash(v13FixturePath)))
	v13AssertFixtureHeader(t, repositoryRoot, fixture)

	t.Run("exact_envelope_and_recipient_contract", func(t *testing.T) {
		applicationDirectory := filepath.Join(repositoryRoot, "internal", "application")
		v13RequireProductionTypeFields(t, applicationDirectory, "MailboxEndpoint", []string{
			"PrincipalRef", "WorkItemRef", "ExecutionRef",
		})
		v13RequireProductionTypeFields(t, applicationDirectory, "MailboxEnvelope", []string{
			"Ref", "Kind", "GoalRef", "ProjectRef", "TargetPlanGeneration", "ParentWorkItemRef",
			"ChildWorkItemRef", "Source", "Recipient", "Summary", "ArtifactRefs", "ContentHash", "AdmittedAt",
		})
	})

	t.Run("typed_use_cases_and_single_state_repository", func(t *testing.T) {
		orchestrator := reflect.TypeOf((*application.Orchestrator)(nil))
		for _, name := range fixture.RequiredUseCases {
			v13RequireUseCase(t, orchestrator, name)
		}
		statePort := reflect.TypeOf((*application.StateRepository)(nil)).Elem()
		for _, name := range fixture.RequiredRepositoryMethods {
			if _, ok := statePort.MethodByName(name); !ok {
				t.Errorf("V13_RED StateRepository lacks %s", name)
			}
		}
		dependencies := reflect.TypeOf(application.Dependencies{})
		stateFields := 0
		for index := 0; index < dependencies.NumField(); index++ {
			if dependencies.Field(index).Type == statePort {
				stateFields++
			}
		}
		if stateFields != 1 {
			t.Errorf("V13_RED application.Dependencies has %d StateRepository values, want one", stateFields)
		}
	})

	t.Run("prior_vertical_ratchets_remain_green", func(t *testing.T) {
		authorityFixture := evidenceDecodeStrictJSON[v02Fixture](
			t, filepath.Join(repositoryRoot, filepath.FromSlash(v02FixturePath)),
		)
		v02AssertSingleWriterAndScheduler(
			t, v02LoadSources(t, repositoryRoot, authorityFixture.ProductModule, authorityFixture.ProductRoots),
			authorityFixture.Lifecycle,
		)
		v05AssertContractualLineageDoesNotCloseGoalEarly(t)
		for _, relative := range []string{
			"acceptance/v06_atomic_state_outbox_test.go",
			"acceptance/v09_recovery_backup_test.go",
		} {
			content, err := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(relative)))
			if err != nil {
				t.Fatal(err)
			}
			text := string(content)
			if strings.Count(text, "application.New(application.Dependencies{") != 1 ||
				strings.Count(text, "MaxMailboxEnvelopeBytes:") != 1 {
				t.Errorf("%s does not wire the one canonical mailbox envelope limit", relative)
			}
		}
	})

	t.Run("causal_states_use_existing_outbox_and_trusted_fencing", func(t *testing.T) {
		applicationSource := v10ReadProductionGo(t, filepath.Join(repositoryRoot, "internal", "application"))
		for _, required := range []string{
			`"deliver_mailbox"`, `"admitted"`, `"claimed"`, `"delivered"`, `"consumed"`,
			`"acknowledged"`, `"blocked"`, `"retired"`, "MailboxRetirement",
			"Fence", "LeaseUntil",
		} {
			if !strings.Contains(applicationSource, required) {
				t.Errorf("V13_RED mailbox causal contract lacks %q", required)
			}
		}
		for _, forbidden := range fixture.ForbiddenPrivateAuthorities {
			if strings.Contains(strings.ToLower(applicationSource), strings.ToLower(forbidden)) {
				t.Errorf("V13 mailbox adds private authority %q", forbidden)
			}
		}
		for _, typeName := range []string{
			"MarkMailboxDeliveredRequest", "ConsumeMailboxRequest", "ResolveMailboxRequest",
			"MailboxDeliveryAttempt", "MailboxAcknowledgement",
		} {
			fields, found := v13ProductionTypeFields(t, filepath.Join(repositoryRoot, "internal", "application"), typeName)
			if !found {
				t.Errorf("V13_RED production type %s missing", typeName)
			} else if fields["DeliveryAttempt"] {
				t.Errorf("V13 mailbox duplicates Fence as %s.DeliveryAttempt", typeName)
			}
		}
	})

	t.Run("sqlite_persists_mailbox_in_same_authority_and_recovers_each_frontier", func(t *testing.T) {
		migrationPath := filepath.Join(repositoryRoot, "internal", "adapters", "state", "sqlite", "migrations", "008_mailbox.sql")
		migration, err := os.ReadFile(migrationPath)
		if err != nil {
			t.Errorf("V13_RED SQLite migration 008 missing: %v", err)
		} else {
			text := strings.ToLower(string(migration))
			for _, required := range []string{
				"mailbox_envelopes", "mailbox_admission_receipts", "mailbox_delivery_acks",
				"mailbox_delivery_attempts", "mailbox_retirements", "outbox",
			} {
				if !strings.Contains(text, required) {
					t.Errorf("V13_RED migration lacks %s", required)
				}
			}
			if strings.Contains(text, "mailbox_outbox") {
				t.Error("V13 migration creates private mailbox_outbox")
			}
			if strings.Contains(text, "create table mailbox_fences") {
				t.Error("V13 migration creates a second fence authority")
			}
		}
		recovery := v10ReadProductionGo(t, filepath.Join(repositoryRoot, "internal", "adapters", "state", "sqlite"))
		for _, required := range []string{
			"mailbox_envelopes", "mailbox_delivery_attempts", "mailbox_delivery_acks", "mailbox_retirements",
		} {
			if !strings.Contains(recovery, required) {
				t.Errorf("V13_RED SQLite recovery does not validate %s", required)
			}
		}
	})

	t.Run("behavioral_negatives_and_restart_are_executable_tests", func(t *testing.T) {
		testSource := v13ReadGoTests(t,
			filepath.Join(repositoryRoot, "internal", "goal"),
			filepath.Join(repositoryRoot, "internal", "application"),
			filepath.Join(repositoryRoot, "internal", "adapters", "state", "sqlite"),
			filepath.Join(repositoryRoot, "internal", "bootstrap"),
		)
		for _, required := range []string{
			"TestMailboxAdmissionIsAtomicAndRequestIdempotent",
			"TestMailboxRejectsDuplicateChildDeliveryRelation",
			"TestMailboxEnvelopeLimitRejectsBeforeIdentityOrStateEffects",
			"TestMailboxConcurrentClaimHasOneExactRecipientWinner",
			"TestMailboxClaimReplayPreservesHistoricalFrontierWithoutRenewal",
			"TestMailboxMutationReplayPreservesHistoricalFrontiersAfterTerminalState",
			"TestMailboxListUsesDeterministicFIFOOrder",
			"TestMailboxRejectsNonContractualChildEdgeBeforeEffects",
			"TestV05ParentMetadataDoesNotCreateHandoffBarrier",
			"TestMCPParentMetadataClosesWithoutMailboxOrRequeue",
			"TestMailboxRejectsWrongRecipientAndSuccessorExecution",
			"TestMailboxExpiredClaimReclaimsAfterCrash",
			"TestMailboxAcknowledgementReplayNeverRedelivers",
			"TestMailboxHandoffRequirementSurvivesRestartAndGatesAdmission",
			"TestMailboxRestartPreservesEveryCausalFrontier",
			"TestMailboxParentClosureRequiresEveryChildResolution",
			"TestMailboxAcknowledgedAndBlockedReceiptsAreImmutable",
			"TestMailboxRecipientFailureRetiresWithoutReplacementOrForgedResolution",
		} {
			if !strings.Contains(testSource, "func "+required+"(") {
				t.Errorf("V13_RED executable behavior test missing: %s", required)
			}
		}
	})

	t.Run("parent_barrier_is_goal_causal_not_terminality_inference", func(t *testing.T) {
		goalSource := v10ReadProductionGo(t, filepath.Join(repositoryRoot, "internal", "goal"))
		applicationSource := v10ReadProductionGo(t, filepath.Join(repositoryRoot, "internal", "application"))
		for _, required := range []string{"ChildHandoffResolution", "ResolveChildHandoff"} {
			if !strings.Contains(goalSource, required) {
				t.Errorf("V13_RED Goal lacks contractual child resolution %q", required)
			}
		}
		if !strings.Contains(applicationSource, "ResolveChildHandoff") {
			t.Error("V13_RED application does not atomically apply mailbox resolution to Goal")
		}
	})
}

func TestAcceptanceV13MailboxReceipt(t *testing.T) {
	evidenceAssertReceiptV3(t, evidenceRepositoryRoot(t), evidenceReceiptV3Expectation{
		Contract: "AC-V13-MAILBOX", FixturePath: v13FixturePath,
		ReceiptPath:       "product/evidence/v13_mailbox.json",
		ExecutedNotBefore: "2026-07-16T00:00:00Z", TrustedBaseGitCommitOID: v13TrustedBaseGitCommitOID,
	})
}

func TestV13CandidateSubjectsCoverCommittedDelta(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v13Fixture](t, filepath.Join(repositoryRoot, filepath.FromSlash(v13FixturePath)))
	if err := evidenceValidateSealedCommit(
		repositoryRoot, fixture.ProductDeltaBaseGitCommitOID, fixture.ProductDeltaSealedGitCommitOID,
	); err != nil {
		t.Fatal(err)
	}
	output, err := evidenceGit(
		repositoryRoot, "diff", "--name-only",
		fixture.ProductDeltaBaseGitCommitOID, fixture.ProductDeltaSealedGitCommitOID, "--",
	)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.TrimSpace(string(output))
	var changed []string
	if text != "" {
		changed = strings.Split(text, "\n")
	}
	sort.Strings(changed)
	if !reflect.DeepEqual(changed, fixture.CandidateSubjects) {
		t.Fatalf("V13 candidate subjects differ from sealed product delta:\nchanged=%v\nfixture=%v", changed, fixture.CandidateSubjects)
	}
}

func TestV13FixtureIsStrictJSON(t *testing.T) {
	fixture := evidenceDecodeStrictJSON[v13Fixture](t, filepath.Join(evidenceRepositoryRoot(t), filepath.FromSlash(v13FixturePath)))
	if fixture.ContractID != "AC-V13-MAILBOX" {
		t.Fatalf("unexpected V13 contract: %q", fixture.ContractID)
	}
}

func v13AssertFixtureHeader(t *testing.T, repositoryRoot string, fixture v13Fixture) {
	t.Helper()
	wantDeferred := []v13DeferredCapability{
		{ID: "ORC-15", Owner: "context_rag_evals", AcceptanceContract: "AC-V27-CONTEXT-RAG-EVALS"},
		{ID: "ORC-29", Owner: "provider_adapters", AcceptanceContract: "AC-V25-PROVIDER-ADAPTERS"},
		{ID: "CTX-02", Owner: "context_rag_evals", AcceptanceContract: "AC-V27-CONTEXT-RAG-EVALS"},
		{ID: "ORC-22", Owner: "context_rag_evals", AcceptanceContract: "AC-V27-CONTEXT-RAG-EVALS"},
	}
	if fixture.SchemaVersion != 1 || fixture.ReceiptSchemaVersion != 3 ||
		fixture.ContractID != "AC-V13-MAILBOX" ||
		fixture.TrustedBaseGitCommitOID != v13TrustedBaseGitCommitOID ||
		fixture.ProductDeltaBaseGitCommitOID != v13ProductDeltaBaseGitCommitOID ||
		fixture.ProductDeltaSealedGitCommitOID != v13ProductDeltaSealedGitCommitOID ||
		fixture.OutputPath != "product/evidence/v13_mailbox.output.txt" ||
		fixture.ReceiptPath != "product/evidence/v13_mailbox.json" ||
		!reflect.DeepEqual(fixture.OwnedCapabilityIDs, []string{"ORC-04", "ORC-05", "ORC-14"}) ||
		!reflect.DeepEqual(fixture.DeferredCapabilities, wantDeferred) ||
		!reflect.DeepEqual(fixture.RequiredUseCases, []string{
			"AdmitMailbox", "ClaimMailbox", "MarkMailboxDelivered", "ConsumeMailbox",
			"AcknowledgeMailbox", "BlockMailbox", "GetMailbox", "ListMailbox",
		}) || !reflect.DeepEqual(fixture.RequiredRepositoryMethods, []string{
		"MailboxReplay", "AdmitMailbox", "ClaimMailbox", "MarkMailboxDelivered", "ConsumeMailbox",
		"AcknowledgeMailbox", "BlockMailbox", "GetMailbox", "ListMailbox",
	}) || !reflect.DeepEqual(fixture.Assertions, v13ExpectedAssertions()) {
		t.Fatalf("invalid V13 fixture header: %+v", fixture)
	}
	if _, err := time.Parse(time.RFC3339Nano, fixture.Scenario.BaseTime); err != nil {
		t.Fatalf("invalid V13 base time: %v", err)
	}
	lease := v13PositiveDuration(t, "claim_lease", fixture.Scenario.ClaimLease)
	reclaim := v13PositiveDuration(t, "reclaim_after", fixture.Scenario.ReclaimAfter)
	if reclaim <= lease || fixture.Scenario.PlanGeneration == 0 || fixture.Scenario.ClaimContenders < 2 ||
		len(fixture.Scenario.ChildWorkItemRefs) != 2 ||
		!reflect.DeepEqual(fixture.Scenario.MessageKinds, []string{"child_delivery"}) ||
		!reflect.DeepEqual(fixture.Scenario.Lifecycle, []string{"admitted", "claimed", "delivered", "consumed", "retired"}) ||
		!reflect.DeepEqual(fixture.Scenario.TerminalResolutions, []string{"acknowledged", "blocked"}) ||
		fixture.Scenario.RecipientExecutionRef == fixture.Scenario.SuccessorExecutionRef {
		t.Fatalf("invalid V13 scenario: %+v", fixture.Scenario)
	}
	if fixture.Command != "sh -c '"+v13ValidationShellBody()+"'" ||
		!reflect.DeepEqual(fixture.ExecutionArgv, []string{"sh", "-c", v13ValidationShellBody()}) {
		t.Fatalf("invalid V13 command/argv: %q %#v", fixture.Command, fixture.ExecutionArgv)
	}
	if err := evidenceValidateCandidateSubjects(fixture.CandidateSubjects, fixture.ReceiptPath, fixture.OutputPath); err != nil {
		t.Fatal(err)
	}
	for _, relative := range fixture.CandidateSubjects {
		if _, err := os.Stat(filepath.Join(repositoryRoot, filepath.FromSlash(relative))); err != nil {
			t.Errorf("candidate subject %q is not readable: %v", relative, err)
		}
	}
}

func v13ValidationShellBody() string {
	return "go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV13ScopeAndExecutableContract|TestV13EvidenceBelongsOnlyToMailboxCapabilities|TestV13AcceptanceCommandRunsMailboxConsumers|TestRebuildArchitecture|TestTraceabilityRebuildBugLessons|TestAcceptanceV13Mailbox|TestV13CandidateSubjectsCoverCommittedDelta)$\"" +
		" && go test -mod=vendor -count=1 ./internal/goal ./internal/identity ./internal/config ./internal/application ./internal/ports ./internal/adapters/state/sqlite ./internal/bootstrap ./cmd/orquesta"
}

func v13ExpectedAssertions() []string {
	return []string{
		"admission is request-idempotent and atomically binds one immutable envelope to one action in the existing outbox without treating admission as delivery",
		"the envelope binds exact project Goal plan generation parent and child WorkItems plus source and recipient principal WorkItem and execution identities",
		"admitted claimed delivered consumed and acknowledged or blocked are separate causal facts with trusted timestamps and no text-derived lifecycle",
		"concurrent exact-recipient claims have one winner and use transaction-clock lease opaque token and one monotonic fence as the delivery-attempt ordinal rather than caller time",
		"wrong project Goal generation principal WorkItem execution sibling or successor cannot claim deliver consume acknowledge block or enumerate the message",
		"expired lease wrong token and stale fence cannot mutate while a post-expiry reclaim preserves the envelope and increments the single fence exactly once",
		"a crash after claim and before terminal recipient resolution permits safe reclaim after restart without message loss or partial acknowledgement",
		"exact claim replay returns the original claimed frontier without renewing its lease after expiry delivery or consumption and conflicts after a superseding fence or terminal mailbox",
		"exact delivery and consumption replay returns the original historical frontier after terminal state or a later fence without authorizing another mutation",
		"recipient mailbox listing is deterministic causal FIFO by admitted_at then message ref and applies its limit after exact recipient scope",
		"delivery consumption and recipient acknowledgement have distinct immutable receipts and an outbox consumption receipt alone is not recipient evidence",
		"exact acknowledgement replay returns the same receipt and acknowledged or blocked messages are never redelivered by outbox replay restart or another execution",
		"a successor execution cannot acknowledge a message addressed to its predecessor even when both executions use the same principal",
		"if the exact recipient execution fails before resolution the mailbox becomes retired in the same Goal failure transaction without replacement readdress ACK authorization or ChildHandoffResolution",
		"a contractual parent cannot succeed until every successful direct child has one acknowledged delivery or explicit recipient block; failed and dependency_failed skipped children are durable causal blocks",
		"an unrelated child terminal WorkItem admission ACK or delivery without recipient acknowledgement does not satisfy the parent closure barrier",
		"parent lineage is noncontractual by default and only an explicit HandoffRequired true edge activates the mailbox barrier so public V05 DAGs remain operable while public mailbox bindings are deferred",
		"child_delivery is the only V13 causal envelope; canonical mailbox.max_envelope_bytes bounds its summary and artifact refs while generic messages rich context resumable sessions and provider handoff stay deferred",
		"the existing outbox fence is the single mailbox attempt ordinal; mailbox adds no second delivery counter or private fence store",
		"V13 preserves the V02 application-only Goal writer the V05 contractual lineage gate and the V06 V09 V10 acceptance harness wiring",
		"Goal and application remain the only lifecycle authority and mailbox adds no private store database queue scheduler loop goroutine daemon provider policy or parallel lifecycle",
	}
}

func v13RequireUseCase(t *testing.T, owner reflect.Type, name string) {
	t.Helper()
	method, ok := owner.MethodByName(name)
	if !ok {
		t.Errorf("V13_RED Orchestrator lacks %s", name)
		return
	}
	contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	if method.Type.NumIn() != 4 || method.Type.In(1) != contextType ||
		method.Type.In(2) != reflect.TypeOf(application.Access{}) ||
		method.Type.In(3).Name() != name+"Request" || method.Type.NumOut() != 2 ||
		method.Type.Out(0).Name() == "" || method.Type.Out(1) != errorType {
		t.Errorf("V13_RED %s signature=%s", name, method.Type)
	}
}

func v13RequireProductionTypeFields(t *testing.T, directory, typeName string, required []string) {
	t.Helper()
	fields, found := v13ProductionTypeFields(t, directory, typeName)
	if !found {
		t.Errorf("V13_RED production type %s missing", typeName)
		return
	}
	for _, name := range required {
		if !fields[name] {
			t.Errorf("V13_RED %s lacks %s", typeName, name)
		}
	}
}

func v13ProductionTypeFields(t *testing.T, directory, typeName string) (map[string]bool, bool) {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("read production package %s: %v", directory, err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		filename := filepath.Join(directory, entry.Name())
		parsed, err := parser.ParseFile(token.NewFileSet(), filename, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", filename, err)
		}
		for _, declaration := range parsed.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.TYPE {
				continue
			}
			for _, specification := range general.Specs {
				typeSpecification, ok := specification.(*ast.TypeSpec)
				if !ok || typeSpecification.Name.Name != typeName {
					continue
				}
				structure, ok := typeSpecification.Type.(*ast.StructType)
				if !ok {
					return nil, true
				}
				fields := make(map[string]bool)
				for _, field := range structure.Fields.List {
					for _, name := range field.Names {
						fields[name.Name] = true
					}
				}
				return fields, true
			}
		}
	}
	return nil, false
}

func v13ReadGoTests(t *testing.T, directories ...string) string {
	t.Helper()
	var source strings.Builder
	for _, directory := range directories {
		entries, err := os.ReadDir(directory)
		if err != nil {
			t.Fatalf("read test package %s: %v", directory, err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			content, err := os.ReadFile(filepath.Join(directory, entry.Name()))
			if err != nil {
				t.Fatalf("read %s: %v", entry.Name(), err)
			}
			source.Write(content)
			source.WriteByte('\n')
		}
	}
	return source.String()
}

func v13PositiveDuration(t *testing.T, name, raw string) time.Duration {
	t.Helper()
	duration, err := time.ParseDuration(raw)
	if err != nil || duration <= 0 {
		t.Fatalf("invalid V13 %s %q: %v", name, raw, err)
	}
	return duration
}
