package sqlite

import (
	"context"
	"reflect"
	"testing"
	"time"

	"orquesta/internal/application"
)

func TestExpiredAgentLaunchContinuationStoreV41IsAtomicAndBoundToOriginalEffectAttempt(t *testing.T) {
	system, original, physicalAttempt, reconciliationAuthorityRef, reconciliationAttemptRef :=
		seedV41Run7ContinuationSource(t, "store-atomic")
	defer system.repository.Close()
	fixture := newV41ContinuationFixture(
		t, system, original, physicalAttempt, reconciliationAuthorityRef,
		reconciliationAttemptRef, "store-atomic",
	)
	record := v41ContinuationRecord(fixture)

	stored, created, err := system.repository.RecordExpiredAgentLaunchContinuationV41(
		context.Background(), record,
	)
	if err != nil || !created || !reflect.DeepEqual(stored, record) {
		t.Fatalf("record=%+v created=%t err=%s", stored, created, sqliteTestErrorChain(err))
	}
	replayed, created, err := system.repository.RecordExpiredAgentLaunchContinuationV41(
		context.Background(), record,
	)
	if err != nil || created || !reflect.DeepEqual(replayed, record) {
		t.Fatalf("replay=%+v created=%t err=%s", replayed, created, sqliteTestErrorChain(err))
	}
	loaded, found, err := system.repository.ExpiredAgentLaunchContinuationV41(
		context.Background(), reconciliationAuthorityRef,
	)
	if err != nil || !found || !reflect.DeepEqual(loaded, record) ||
		loaded.EffectAttemptRef != physicalAttempt.Ref {
		t.Fatalf("loaded=%+v found=%t err=%s", loaded, found, sqliteTestErrorChain(err))
	}

	divergent := record
	divergent.EffectAttemptRef += ":other"
	if _, _, err := system.repository.RecordExpiredAgentLaunchContinuationV41(
		context.Background(), divergent,
	); err == nil {
		t.Fatal("divergent EffectAttempt replay accepted")
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("database after replay rejection: %s", sqliteTestErrorChain(err))
	}
}

func v41ContinuationRecord(fixture v41ContinuationFixture) application.ExpiredAgentLaunchContinuationRecordV41 {
	return application.ExpiredAgentLaunchContinuationRecordV41{
		SubjectRef:                 fixture.subjectRef,
		ReconciliationAuthorityRef: fixture.reconciliationAuthorityRef,
		ReconciliationAttemptRef:   fixture.reconciliationAttemptRef,
		ProjectRef:                 fixture.projectRef, GoalRef: fixture.goalRef, WorkItemRef: fixture.workItemRef,
		ExecutionRef: fixture.executionRef, ActionRef: fixture.actionRef,
		EffectIntentRef: fixture.effectIntentRef, EffectIntentDigest: fixture.effectIntentDigest,
		EffectAttemptRef: fixture.effectAttemptRef, PlanGeneration: uint64(fixture.planGeneration),
		WorkItemGeneration: uint64(fixture.workItemGeneration), ActionFence: uint64(fixture.actionFence),
		RequestKeySHA256:      fixture.manifest.RequestKeySHA256,
		OriginalRequestSHA256: fixture.manifest.OriginalRequestSHA256,
		AMVLaunchRef:          fixture.manifest.LaunchRef, AMVExecutionRef: fixture.manifest.ExecutionRef,
		AMVRunRef: fixture.manifest.RunRef, AMVFence: fixture.manifest.Fence,
		AMVGeneration: fixture.manifest.Generation, AMVCID: fixture.manifest.CID,
		AMVIdentitySHA256: fixture.manifest.IdentitySHA256, SourceDigest: fixture.manifest.SourceDigest,
		ProfileDescriptorBytes:  append([]byte(nil), fixture.profileBytes...),
		ProfileDescriptorSHA256: fixture.profileBytesSHA256,
		PlanBytes:               append([]byte(nil), fixture.planBytes...), PlanSHA256: fixture.planBytesSHA256,
		ConcessionBytes:     append([]byte(nil), fixture.concessionBytes...),
		ConcessionSHA256:    fixture.concessionBytesSHA256,
		ManifestBytes:       append([]byte(nil), fixture.manifestBytes...),
		ManifestBytesSHA256: fixture.manifestBytesSHA256, ManifestSHA256: fixture.manifestSHA256,
		PreparedAt: time.Unix(0, fixture.preparedAt).UTC(), AuthorityRef: fixture.authorityRef,
		AuthoritySHA256: fixture.authoritySHA256, AuthorityBytes: append([]byte(nil), fixture.authorityBytes...),
		AuthorityBytesSHA256: fixture.authorityBytesSHA256, KeyID: fixture.keyID,
		KeyEpoch: uint64(fixture.keyEpoch), TrustRevision: uint64(fixture.trustRevision),
		PublicKey: append([]byte(nil), fixture.publicKey...), Signature: append([]byte(nil), fixture.signature...),
		IssuedUnixMS: uint64(fixture.issuedUnixMS), ExpiresUnixMS: uint64(fixture.expiresUnixMS),
		AdmittedUnixMS: uint64(fixture.admittedUnixMS),
	}
}
