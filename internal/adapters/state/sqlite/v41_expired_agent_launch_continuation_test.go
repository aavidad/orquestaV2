package sqlite

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"
	"orquesta/internal/application"
)

func TestMigrationV41AddsExpiredLaunchContinuationAuthorityToV40(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "expired-launch-continuation-v40.db")
	database := agentCapacityDatabase(t, path, recoverySchemaV38TerminalLaunchReconciliation)
	defer database.Close()

	var before int
	sqliteTestNoError(t, database.QueryRow(`PRAGMA user_version`).Scan(&before))
	if before != 40 {
		t.Fatalf("source schema=%d", before)
	}
	sqliteTestNoError(t, applyMigrations(ctx, database))

	var after, receipt, strictTables, replayTriggers, immutableTriggers int
	sqliteTestNoError(t, database.QueryRow(`PRAGMA user_version`).Scan(&after))
	sqliteTestNoError(t, database.QueryRow(`SELECT COUNT(*) FROM schema_migrations
WHERE version=41 AND name='041_expired_agent_launch_continuation_authority.sql'`).Scan(&receipt))
	sqliteTestNoError(t, database.QueryRow(`SELECT COUNT(*) FROM pragma_table_list
WHERE name LIKE 'agent_launch_expired_continuation_%' AND strict=1`).Scan(&strictTables))
	sqliteTestNoError(t, database.QueryRow(`SELECT
 (SELECT COUNT(*) FROM sqlite_schema WHERE type='trigger'
   AND name LIKE 'agent_launch_expired_continuation_%_exact_replay'),
 (SELECT COUNT(*) FROM sqlite_schema WHERE type='trigger'
   AND name LIKE 'agent_launch_expired_continuation_%_immutable_%')`).Scan(&replayTriggers, &immutableTriggers))
	if after != 41 || receipt != 1 || strictTables != 2 || replayTriggers != 2 || immutableTriggers != 4 {
		t.Fatalf("V41 shape version=%d receipt=%d strict=%d replay=%d immutable=%d",
			after, receipt, strictTables, replayTriggers, immutableTriggers)
	}
	if _, _, err := validateRecoveryDatabase(ctx, database); err != nil {
		t.Fatalf("migrated V41 recovery database invalid: %s", sqliteTestErrorChain(err))
	}
}

func TestV41ExpiredLaunchContinuationPreservesRun7CausalityAndExactReplay(t *testing.T) {
	system, original, physicalAttempt, reconciliationAuthorityRef, reconciliationAttemptRef :=
		seedV41Run7ContinuationSource(t, "exact-replay")
	before := countV41OriginalCausality(t, system, original.Action.Ref)
	fixture := newV41ContinuationFixture(t, system, original, physicalAttempt,
		reconciliationAuthorityRef, reconciliationAttemptRef, "exact-replay")

	insertV41ContinuationSubject(t, system, fixture)
	insertV41ContinuationAuthority(t, system, fixture)
	insertV41ContinuationSubject(t, system, fixture)
	insertV41ContinuationAuthority(t, system, fixture)

	var subjects, authorities int
	sqliteTestNoError(t, system.repository.db.QueryRow(`SELECT
 (SELECT COUNT(*) FROM agent_launch_expired_continuation_subjects WHERE ref=?),
 (SELECT COUNT(*) FROM agent_launch_expired_continuation_authorities WHERE authority_ref=?)`,
		fixture.subjectRef, fixture.authorityRef).Scan(&subjects, &authorities))
	if subjects != 1 || authorities != 1 {
		t.Fatalf("exact replay subjects=%d authorities=%d", subjects, authorities)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("valid V41 run7 continuation: %s", sqliteTestErrorChain(err))
	}
	if after := countV41OriginalCausality(t, system, original.Action.Ref); after != before {
		t.Fatalf("V41 synthesized causal rows before=%+v after=%+v", before, after)
	}
	var completed, quarantined int
	var errorCode string
	sqliteTestNoError(t, system.repository.db.QueryRow(`SELECT
 completed_at IS NOT NULL,quarantined_at=completed_at,last_error_code FROM outbox WHERE ref=?`,
		original.Action.Ref).Scan(&completed, &quarantined, &errorCode))
	if completed != 1 || quarantined != 1 || errorCode != "application.effect_unknown_applied" {
		t.Fatalf("run7 terminal evidence completed=%d quarantined=%d error=%q",
			completed, quarantined, errorCode)
	}

	if err := system.repository.Close(); err != nil {
		t.Fatal(err)
	}
	reopened := openSQLiteV15Repository(t, system.path, system.clock.Now)
	defer reopened.Close()
	var manifestBytes, authorityBytes []byte
	sqliteTestNoError(t, reopened.db.QueryRow(`SELECT subject.manifest_bytes,authority.authority_bytes
FROM agent_launch_expired_continuation_subjects subject
JOIN agent_launch_expired_continuation_authorities authority ON authority.subject_ref=subject.ref
WHERE subject.ref=?`, fixture.subjectRef).Scan(&manifestBytes, &authorityBytes))
	if string(manifestBytes) != string(fixture.manifestBytes) || string(authorityBytes) != string(fixture.authorityBytes) {
		t.Fatal("restart changed exact manifest or authority bytes")
	}
}

func TestV41ExpiredLaunchContinuationSourceSelectsFirstRequeuedAttemptWithoutClaiming(t *testing.T) {
	system, original, physicalAttempt, authorityRef, attemptRef :=
		seedV41Run7ContinuationSource(t, "writer-source")
	defer system.repository.Close()
	source, found, err := system.repository.ExpiredAgentLaunchContinuationSourceV41(
		context.Background(), authorityRef,
	)
	if err != nil || !found {
		t.Fatalf("source found=%t err=%s", found, sqliteTestErrorChain(err))
	}
	if source.Authority.Ref != authorityRef || source.ReconciliationAttempt.Ref != attemptRef ||
		source.ReconciliationAttempt.OriginalEffectAttemptRef != physicalAttempt.Ref ||
		source.Claim.RecoveryEffectAttemptRef != physicalAttempt.Ref ||
		source.Claim.Action.Ref != original.Action.Ref ||
		source.Claim.Fence != source.ReconciliationAttempt.JobFence ||
		source.Claim.Disposition != application.ActionClaimDispositionReconcileTerminalLaunch {
		t.Fatalf("source=%+v original=%+v physical=%+v", source, original, physicalAttempt)
	}
	var state, lastError string
	var claimToken any
	sqliteTestNoError(t, system.repository.db.QueryRow(`SELECT state,last_error_code,claim_token
FROM agent_launch_reconciliation_jobs WHERE authority_ref=?`, authorityRef).Scan(&state, &lastError, &claimToken))
	if state != "pending" || lastError != "agent.launch_reconciliation_pending" || claimToken != nil {
		t.Fatalf("read-only source mutated job state=%q error=%q claim=%v", state, lastError, claimToken)
	}
}

func TestV41ExpiredLaunchContinuationRejectsDivergentReplay(t *testing.T) {
	system, original, physicalAttempt, reconciliationAuthorityRef, reconciliationAttemptRef :=
		seedV41Run7ContinuationSource(t, "divergent-replay")
	fixture := newV41ContinuationFixture(t, system, original, physicalAttempt,
		reconciliationAuthorityRef, reconciliationAttemptRef, "divergent-replay")
	insertV41ContinuationSubject(t, system, fixture)
	insertV41ContinuationAuthority(t, system, fixture)

	divergent := fixture
	divergent.authorityBytes = append([]byte(nil), fixture.authorityBytes...)
	divergent.authorityBytes[len(divergent.authorityBytes)-1] ^= 1
	divergent.authorityBytesSHA256 = v41BytesSHA256(divergent.authorityBytes)
	if _, err := system.repository.db.Exec(v41InsertAuthoritySQL, divergent.authorityArgs()...); err == nil {
		t.Fatal("divergent authority bytes replayed under the same one-use reference")
	}
	var persisted []byte
	sqliteTestNoError(t, system.repository.db.QueryRow(`SELECT authority_bytes
FROM agent_launch_expired_continuation_authorities WHERE authority_ref=?`, fixture.authorityRef).Scan(&persisted))
	if string(persisted) != string(fixture.authorityBytes) {
		t.Fatal("divergent replay replaced admitted authority")
	}
}

func TestRecoveryV41RejectsPartialAuthoritySignatureAndAlteredCausalChain(t *testing.T) {
	for _, mode := range []string{"partial", "signature", "causal-chain"} {
		t.Run(mode, func(t *testing.T) {
			system, original, physicalAttempt, reconciliationAuthorityRef, reconciliationAttemptRef :=
				seedV41Run7ContinuationSource(t, "invalid-"+mode)
			fixture := newV41ContinuationFixture(t, system, original, physicalAttempt,
				reconciliationAuthorityRef, reconciliationAttemptRef, "invalid-"+mode)
			insertV41ContinuationSubject(t, system, fixture)
			if mode != "partial" {
				insertV41ContinuationAuthority(t, system, fixture)
			}
			switch mode {
			case "signature":
				mutateV41IgnoringImmutability(
					t, system, "agent_launch_expired_continuation_authorities_immutable_update",
					`UPDATE agent_launch_expired_continuation_authorities
SET signature=zeroblob(64) WHERE authority_ref=?`, fixture.authorityRef,
				)
			case "causal-chain":
				mutateV41IgnoringImmutability(
					t, system, "agent_launch_expired_continuation_subjects_immutable_update",
					`UPDATE agent_launch_expired_continuation_subjects
SET effect_intent_digest=? WHERE ref=?`, strings.Repeat("f", 64), fixture.subjectRef,
				)
			}
			if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err == nil ||
				!recoveryErrorContains(err, "sqlite.recovery_v41_expired_launch_continuation_invalid") {
				t.Fatalf("V41 %s corruption accepted: %s", mode, sqliteTestErrorChain(err))
			}
		})
	}
}

func TestRecoveryV41RejectsFutureV42(t *testing.T) {
	repository, _ := openTestRepository(t)
	sqliteTestNoError(t, func() error {
		_, err := repository.db.Exec(`PRAGMA user_version=42`)
		return err
	}())
	if _, _, err := validateRecoveryDatabase(context.Background(), repository.db); err == nil ||
		!recoveryErrorContains(err, "sqlite.recovery_schema_version_invalid") {
		t.Fatalf("future V42 accepted: %s", sqliteTestErrorChain(err))
	}
}

type v41OriginalCausality struct {
	goals, workItems, executions, launches, intents, effectAttempts int
}

func countV41OriginalCausality(t *testing.T, system *sqliteV15System, actionRef string) v41OriginalCausality {
	t.Helper()
	var counts v41OriginalCausality
	sqliteTestNoError(t, system.repository.db.QueryRow(`SELECT
 (SELECT COUNT(*) FROM goals),(SELECT COUNT(*) FROM work_items),(SELECT COUNT(*) FROM executions),
 (SELECT COUNT(*) FROM outbox WHERE kind='launch_agent'),
 (SELECT COUNT(*) FROM effect_intents WHERE kind='agent_launch'),
 (SELECT COUNT(*) FROM effect_attempts WHERE action_ref=?)`, actionRef).Scan(
		&counts.goals, &counts.workItems, &counts.executions, &counts.launches,
		&counts.intents, &counts.effectAttempts))
	return counts
}

func seedV41Run7ContinuationSource(
	t *testing.T,
	suffix string,
) (*sqliteV15System, application.ActionClaim, application.EffectAttempt, string, string) {
	t.Helper()
	ctx := context.Background()
	system, original, physicalAttempt := seedV27AmbiguousEffectAttempt(t, "v41-"+suffix)
	quarantinedAt := system.clock.Now().UTC()
	sqliteTestNoError(t, system.repository.QuarantineAction(ctx, application.ActionQuarantinedState{
		Claim: original, ErrorCode: "application.effect_unknown_applied", OperationAt: quarantinedAt,
		Event: application.EventRecord{
			Ref: "event:v41-quarantine:" + suffix, Kind: "action.quarantined",
			GoalRef: original.Action.GoalRef, WorkItemRef: original.Action.WorkItemRef,
			ExecutionRef: original.Action.ExecutionRef, OccurredAt: quarantinedAt,
		},
	}))
	system.clock.Advance(original.LeaseUntil.Sub(system.clock.Now()) + time.Second)
	authorized, err := system.orchestrator.ReconcileTerminalAgentLaunch(ctx, system.access,
		application.ReconcileTerminalAgentLaunchRequest{
			RequestRef: "request:v41-reconciliation:" + suffix, GoalRef: original.Action.GoalRef,
			WorkItemRef: original.Action.WorkItemRef, ExecutionRef: original.Action.ExecutionRef,
			ActionRef: original.Action.Ref, EffectIntentRef: original.Action.EffectIntentRef,
			EffectIntentDigest: original.Action.EffectIntent.Digest, EffectAttemptRef: physicalAttempt.Ref,
			PlanGeneration: original.Action.PlanGeneration, WorkItemGeneration: original.Action.WorkItemGeneration,
			ActionFence: physicalAttempt.ActionFence,
		})
	if err != nil || !authorized.Created {
		t.Fatalf("V41 source authorization=%+v err=%s", authorized, sqliteTestErrorChain(err))
	}
	reconciler := &terminalLaunchReconciler{clock: system.clock, temporaryFirst: true}
	restartSQLiteV15System(t, system)
	system.orchestrator = newRestartRecoveryOrchestrator(t, system, reconciler)
	claim, found, err := system.orchestrator.ClaimNextAction(
		ctx, "worker:v41:"+suffix, application.ActionClaimSelection{ExcludeLaunch: true},
	)
	if err != nil || !found || claim.Disposition != application.ActionClaimDispositionReconcileTerminalLaunch {
		t.Fatalf("V41 reconciliation claim=%+v found=%t err=%s", claim, found, sqliteTestErrorChain(err))
	}
	if _, err := system.orchestrator.ProcessClaim(ctx, claim); err != nil {
		t.Fatalf("V41 temporary reconciliation: %s", sqliteTestErrorChain(err))
	}
	var reconciliationAttemptRef string
	sqliteTestNoError(t, system.repository.db.QueryRow(`SELECT ref
FROM agent_launch_reconciliation_attempts WHERE authority_ref=? ORDER BY started_at DESC LIMIT 1`,
		authorized.Authority.Ref).Scan(&reconciliationAttemptRef))
	return system, original, physicalAttempt, authorized.Authority.Ref, reconciliationAttemptRef
}

type v41ContinuationFixture struct {
	subjectRef, reconciliationAuthorityRef, reconciliationAttemptRef string
	projectRef, goalRef, workItemRef, executionRef, actionRef        string
	effectIntentRef, effectIntentDigest, effectAttemptRef            string
	planGeneration, workItemGeneration, actionFence                  int64
	manifest                                                         microvm.ExpiredLaunchContinuationManifestV1
	profileBytes, planBytes, concessionBytes, manifestBytes          []byte
	profileBytesSHA256, planBytesSHA256, concessionBytesSHA256       string
	manifestBytesSHA256, manifestSHA256                              string
	preparedAt                                                       int64
	authorityRef, authoritySHA256, authorityBytesSHA256              string
	authorityBytes, publicKey, signature                             []byte
	keyID                                                            string
	keyEpoch, trustRevision, issuedUnixMS, expiresUnixMS             int64
	admittedUnixMS                                                   int64
}

func newV41ContinuationFixture(
	t *testing.T,
	system *sqliteV15System,
	original application.ActionClaim,
	physicalAttempt application.EffectAttempt,
	reconciliationAuthorityRef, reconciliationAttemptRef, suffix string,
) v41ContinuationFixture {
	t.Helper()
	signedRequest := strings.Repeat("9", 64)
	manifest := microvm.ExpiredLaunchContinuationManifestV1{
		Schema:               microvm.ExpiredLaunchContinuationManifestSchemaV1,
		Audience:             microvm.ExpiredLaunchContinuationAudienceV1,
		BrokerInstanceSHA256: strings.Repeat("1", 64), SourceDigest: strings.Repeat("2", 64),
		RequestKeySHA256: strings.Repeat("3", 64), OriginalRequestSHA256: strings.Repeat("4", 64),
		LaunchRef: "launch:v41:" + suffix, ExecutionRef: "ejecucion:v41:" + suffix,
		RunRef: "run:v41:" + suffix, Fence: physicalAttempt.ActionFence,
		Generation: uint64(original.Action.PlanGeneration), CID: 41, IdentitySHA256: strings.Repeat("5", 64),
		ValidateLaunch: microvm.ExpiredLaunchContinuationTargetV1{
			Operation: "validate_launch", Schema: 2, Code: 8, IntentRef: "intent:validate:v41:" + suffix,
			IdempotencyKey: "idem:validate:v41:" + suffix, RequestSHA256: strings.Repeat("6", 64),
			CommandSHA256: strings.Repeat("7", 64), InnerAuthorizationSHA256: strings.Repeat("8", 64),
			SignedRequestSHA256: &signedRequest, DaemonState: "ambiguous", RootState: "prepared",
			RootRevision: func() *uint64 { value := uint64(1); return &value }(),
		},
		Launch: microvm.ExpiredLaunchContinuationTargetV1{
			Operation: "launch", Schema: 1, Code: 1, IntentRef: "intent:launch:v41:" + suffix,
			IdempotencyKey: "idem:launch:v41:" + suffix, RequestSHA256: strings.Repeat("a", 64),
			CommandSHA256: strings.Repeat("b", 64), InnerAuthorizationSHA256: strings.Repeat("c", 64),
			DaemonState: "reserved", RootState: "absent",
		},
	}
	manifestSHA256, err := microvm.ExpiredLaunchContinuationManifestSHA256V1(manifest)
	sqliteTestNoError(t, err)
	issued := system.clock.Now().UTC().UnixMilli()
	content := microvm.ExpiredLaunchContinuationAuthorityContentV1{
		Schema:       microvm.ExpiredLaunchContinuationAuthoritySchemaV1,
		Audience:     microvm.ExpiredLaunchContinuationAudienceV1,
		Purpose:      microvm.ExpiredLaunchContinuationPurposeV1,
		AuthorityRef: "continuation:v41:" + suffix, ManifestSHA256: manifestSHA256,
		IssuedUnixMS: uint64(issued), ExpiresUnixMS: uint64(issued + 300_000),
		KeyID: "continuation-key-v41", KeyEpoch: 4, TrustRevision: 9,
		Algorithm: microvm.ExpiredLaunchContinuationAlgorithmV1,
	}
	seed := sha256.Sum256([]byte("orquesta-v41-expired-launch-continuation-test-key"))
	privateKey := ed25519.NewKeyFromSeed(seed[:])
	authority, err := microvm.SignExpiredLaunchContinuationAuthorityV1(privateKey, content, manifest)
	sqliteTestNoError(t, err)
	authoritySHA256, err := microvm.ExpiredLaunchContinuationAuthoritySHA256V1(authority)
	sqliteTestNoError(t, err)
	manifestBytes, err := json.Marshal(manifest)
	sqliteTestNoError(t, err)
	authorityBytes, err := json.Marshal(authority)
	sqliteTestNoError(t, err)
	signature, err := base64.StdEncoding.DecodeString(authority.SignatureBase64)
	sqliteTestNoError(t, err)
	profileBytes := []byte(`{"profile":"original-v41"}`)
	planBytes := []byte(`{"plan":"original-v41"}`)
	concessionBytes := []byte(`{"concession":"original-v41"}`)
	return v41ContinuationFixture{
		subjectRef:                 "expired-launch-continuation-subject:v41:" + suffix,
		reconciliationAuthorityRef: reconciliationAuthorityRef, reconciliationAttemptRef: reconciliationAttemptRef,
		projectRef: system.project.String(), goalRef: original.Action.GoalRef.String(),
		workItemRef: original.Action.WorkItemRef.String(), executionRef: original.Action.ExecutionRef.String(),
		actionRef: original.Action.Ref, effectIntentRef: original.Action.EffectIntentRef,
		effectIntentDigest: original.Action.EffectIntent.Digest, effectAttemptRef: physicalAttempt.Ref,
		planGeneration: int64(original.Action.PlanGeneration), workItemGeneration: int64(original.Action.WorkItemGeneration),
		actionFence: int64(physicalAttempt.ActionFence), manifest: manifest,
		profileBytes: profileBytes, planBytes: planBytes, concessionBytes: concessionBytes,
		manifestBytes: manifestBytes, profileBytesSHA256: v41BytesSHA256(profileBytes),
		planBytesSHA256: v41BytesSHA256(planBytes), concessionBytesSHA256: v41BytesSHA256(concessionBytes),
		manifestBytesSHA256: v41BytesSHA256(manifestBytes), manifestSHA256: manifestSHA256,
		preparedAt: system.clock.Now().UTC().UnixNano(), authorityRef: content.AuthorityRef,
		authoritySHA256: authoritySHA256, authorityBytes: authorityBytes,
		authorityBytesSHA256: v41BytesSHA256(authorityBytes), keyID: content.KeyID,
		keyEpoch: int64(content.KeyEpoch), trustRevision: int64(content.TrustRevision),
		publicKey: privateKey.Public().(ed25519.PublicKey), signature: signature,
		issuedUnixMS: issued, expiresUnixMS: issued + 300_000, admittedUnixMS: issued,
	}
}

const v41InsertSubjectSQL = `INSERT INTO agent_launch_expired_continuation_subjects(
ref,reconciliation_authority_ref,project_ref,goal_ref,work_item_ref,execution_ref,action_ref,
effect_intent_ref,effect_intent_digest,effect_attempt_ref,plan_generation,work_item_generation,
action_fence,request_key_sha256,original_request_sha256,amv_launch_ref,amv_execution_ref,amv_run_ref,
amv_fence,amv_generation,amv_cid,amv_identity_sha256,source_digest,
profile_descriptor_bytes,profile_descriptor_bytes_sha256,plan_bytes,plan_bytes_sha256,
concession_bytes,concession_bytes_sha256,manifest_bytes,manifest_bytes_sha256,manifest_sha256,prepared_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`

func (fixture v41ContinuationFixture) subjectArgs() []any {
	return []any{
		fixture.subjectRef, fixture.reconciliationAuthorityRef, fixture.projectRef, fixture.goalRef,
		fixture.workItemRef, fixture.executionRef, fixture.actionRef, fixture.effectIntentRef,
		fixture.effectIntentDigest, fixture.effectAttemptRef, fixture.planGeneration,
		fixture.workItemGeneration, fixture.actionFence, fixture.manifest.RequestKeySHA256,
		fixture.manifest.OriginalRequestSHA256, fixture.manifest.LaunchRef, fixture.manifest.ExecutionRef,
		fixture.manifest.RunRef, int64(fixture.manifest.Fence), int64(fixture.manifest.Generation),
		int64(fixture.manifest.CID), fixture.manifest.IdentitySHA256, fixture.manifest.SourceDigest,
		fixture.profileBytes, fixture.profileBytesSHA256, fixture.planBytes, fixture.planBytesSHA256,
		fixture.concessionBytes, fixture.concessionBytesSHA256, fixture.manifestBytes,
		fixture.manifestBytesSHA256, fixture.manifestSHA256, fixture.preparedAt,
	}
}

const v41InsertAuthoritySQL = `INSERT INTO agent_launch_expired_continuation_authorities(
authority_ref,subject_ref,reconciliation_attempt_ref,manifest_sha256,authority_sha256,
authority_bytes,authority_bytes_sha256,key_id,key_epoch,trust_revision,public_key,signature,
issued_unix_ms,expires_unix_ms,admitted_unix_ms)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`

func (fixture v41ContinuationFixture) authorityArgs() []any {
	return []any{
		fixture.authorityRef, fixture.subjectRef, fixture.reconciliationAttemptRef,
		fixture.manifestSHA256, fixture.authoritySHA256, fixture.authorityBytes,
		fixture.authorityBytesSHA256, fixture.keyID, fixture.keyEpoch, fixture.trustRevision,
		fixture.publicKey, fixture.signature, fixture.issuedUnixMS, fixture.expiresUnixMS,
		fixture.admittedUnixMS,
	}
}

func insertV41ContinuationSubject(t *testing.T, system *sqliteV15System, fixture v41ContinuationFixture) {
	t.Helper()
	_, err := system.repository.db.Exec(v41InsertSubjectSQL, fixture.subjectArgs()...)
	sqliteTestNoError(t, err)
}

func insertV41ContinuationAuthority(t *testing.T, system *sqliteV15System, fixture v41ContinuationFixture) {
	t.Helper()
	_, err := system.repository.db.Exec(v41InsertAuthoritySQL, fixture.authorityArgs()...)
	sqliteTestNoError(t, err)
}

func mutateV41IgnoringImmutability(
	t *testing.T,
	system *sqliteV15System,
	triggerName, statement string,
	arguments ...any,
) {
	t.Helper()
	var triggerSQL string
	sqliteTestNoError(t, system.repository.db.QueryRow(
		`SELECT sql FROM sqlite_schema WHERE type='trigger' AND name=?`, triggerName,
	).Scan(&triggerSQL))
	_, err := system.repository.db.Exec(`DROP TRIGGER ` + triggerName)
	sqliteTestNoError(t, err)
	_, err = system.repository.db.Exec(statement, arguments...)
	sqliteTestNoError(t, err)
	_, err = system.repository.db.Exec(triggerSQL)
	sqliteTestNoError(t, err)
}

func v41BytesSHA256(value []byte) string {
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}
