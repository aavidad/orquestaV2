package sqlite

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"reflect"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"
)

func validateRecoveryV41ExpiredAgentLaunchContinuation(ctx context.Context, tx *sql.Tx) error {
	if err := validateRecoveryV17Checks(ctx, tx, []recoveryV17Check{
		{
			"sqlite.recovery_v41_expired_launch_continuation_subject_invalid",
			`SELECT COUNT(*)
FROM agent_launch_expired_continuation_subjects subject
LEFT JOIN agent_launch_reconciliation_authorities authority
 ON authority.ref=subject.reconciliation_authority_ref
LEFT JOIN outbox action ON action.ref=subject.action_ref
LEFT JOIN effect_intents intent ON intent.ref=subject.effect_intent_ref
LEFT JOIN effect_attempts attempt ON attempt.ref=subject.effect_attempt_ref
LEFT JOIN executions execution ON execution.ref=subject.execution_ref
LEFT JOIN agent_launch_expired_continuation_authorities continuation
 ON continuation.subject_ref=subject.ref
WHERE continuation.authority_ref IS NULL
 OR authority.ref IS NULL OR authority.project_ref<>subject.project_ref
 OR authority.goal_ref<>subject.goal_ref OR authority.work_item_ref<>subject.work_item_ref
 OR authority.execution_ref<>subject.execution_ref OR authority.action_ref<>subject.action_ref
 OR authority.effect_intent_ref<>subject.effect_intent_ref
 OR authority.effect_intent_digest<>subject.effect_intent_digest
 OR authority.effect_attempt_ref<>subject.effect_attempt_ref
 OR authority.plan_generation<>subject.plan_generation
 OR authority.work_item_generation<>subject.work_item_generation
 OR authority.action_fence<>subject.action_fence
 OR action.ref IS NULL OR action.kind<>'launch_agent' OR action.completed_at IS NULL
 OR action.quarantined_at<>action.completed_at OR action.last_error_code<>'application.effect_unknown_applied'
 OR intent.ref IS NULL OR intent.kind<>'agent_launch' OR intent.digest<>subject.effect_intent_digest
 OR attempt.ref IS NULL OR attempt.intent_ref<>subject.effect_intent_ref
 OR attempt.intent_digest<>subject.effect_intent_digest OR attempt.action_ref<>subject.action_ref
 OR attempt.action_fence<>subject.action_fence
 OR execution.ref IS NULL OR execution.goal_ref<>subject.goal_ref
 OR execution.work_item_ref<>subject.work_item_ref OR subject.amv_fence<>subject.action_fence`,
		},
		{
			"sqlite.recovery_v41_expired_launch_continuation_authority_invalid",
			`SELECT COUNT(*)
FROM agent_launch_expired_continuation_authorities continuation
LEFT JOIN agent_launch_expired_continuation_subjects subject ON subject.ref=continuation.subject_ref
LEFT JOIN agent_launch_reconciliation_attempts reconciliation_attempt
 ON reconciliation_attempt.ref=continuation.reconciliation_attempt_ref
LEFT JOIN agent_launch_reconciliation_authorities reconciliation_authority
 ON reconciliation_authority.ref=subject.reconciliation_authority_ref
WHERE subject.ref IS NULL OR reconciliation_attempt.ref IS NULL
 OR reconciliation_authority.ref IS NULL
 OR reconciliation_attempt.authority_ref<>reconciliation_authority.ref
 OR reconciliation_attempt.original_effect_attempt_ref<>subject.effect_attempt_ref
 OR reconciliation_attempt.request_fingerprint<>reconciliation_authority.request_fingerprint
 OR continuation.manifest_sha256<>subject.manifest_sha256`,
		},
	}); err != nil {
		return invalidRecoveryV41ExpiredAgentLaunchContinuation(err)
	}
	rows, err := tx.QueryContext(ctx, `SELECT ref,
 profile_descriptor_bytes,profile_descriptor_bytes_sha256,
 plan_bytes,plan_bytes_sha256,concession_bytes,concession_bytes_sha256,
 manifest_bytes,manifest_bytes_sha256,manifest_sha256,
 request_key_sha256,original_request_sha256,amv_launch_ref,amv_execution_ref,amv_run_ref,
 amv_fence,amv_generation,amv_cid,amv_identity_sha256,source_digest
FROM agent_launch_expired_continuation_subjects ORDER BY ref`)
	if err != nil {
		return invalidRecoveryV41ExpiredAgentLaunchContinuation(err)
	}
	for rows.Next() {
		var ref string
		var profile, plan, concession, manifest []byte
		var profileDigest, planDigest, concessionDigest, manifestBytesDigest, manifestDigest string
		var requestKeyDigest, originalRequestDigest, launchRef, executionRef, runRef string
		var fence, generation, cid int64
		var identityDigest, sourceDigest string
		if err := rows.Scan(
			&ref, &profile, &profileDigest, &plan, &planDigest, &concession, &concessionDigest,
			&manifest, &manifestBytesDigest, &manifestDigest, &requestKeyDigest, &originalRequestDigest,
			&launchRef, &executionRef, &runRef, &fence, &generation, &cid, &identityDigest, &sourceDigest,
		); err != nil {
			_ = rows.Close()
			return invalidRecoveryV41ExpiredAgentLaunchContinuation(err)
		}
		if continuationBytesSHA256(profile) != profileDigest || continuationBytesSHA256(plan) != planDigest ||
			continuationBytesSHA256(concession) != concessionDigest || continuationBytesSHA256(manifest) != manifestBytesDigest {
			_ = rows.Close()
			return invalidRecoveryV41ExpiredAgentLaunchContinuation(nil)
		}
		decoded, decodeErr := microvm.DecodeExpiredLaunchContinuationManifestV1(manifest)
		canonicalDigest, digestErr := microvm.ExpiredLaunchContinuationManifestSHA256V1(decoded)
		if decodeErr != nil || digestErr != nil || canonicalDigest != manifestDigest || fence <= 0 || generation <= 0 || cid < 3 ||
			decoded.RequestKeySHA256 != requestKeyDigest || decoded.OriginalRequestSHA256 != originalRequestDigest ||
			decoded.LaunchRef != launchRef || decoded.ExecutionRef != executionRef || decoded.RunRef != runRef ||
			decoded.Fence != uint64(fence) || decoded.Generation != uint64(generation) || decoded.CID != uint32(cid) ||
			decoded.IdentitySHA256 != identityDigest || decoded.SourceDigest != sourceDigest {
			_ = rows.Close()
			return invalidRecoveryV41ExpiredAgentLaunchContinuation(errors.Join(decodeErr, digestErr))
		}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return invalidRecoveryV41ExpiredAgentLaunchContinuation(err)
	}
	if err := rows.Close(); err != nil {
		return invalidRecoveryV41ExpiredAgentLaunchContinuation(err)
	}
	authorityRows, err := tx.QueryContext(ctx, `SELECT
 continuation.authority_ref,continuation.manifest_sha256,continuation.authority_sha256,
 continuation.authority_bytes,continuation.authority_bytes_sha256,
 continuation.key_id,continuation.key_epoch,continuation.trust_revision,
 continuation.public_key,continuation.signature,continuation.issued_unix_ms,
 continuation.expires_unix_ms,continuation.admitted_unix_ms,subject.manifest_bytes
FROM agent_launch_expired_continuation_authorities continuation
JOIN agent_launch_expired_continuation_subjects subject ON subject.ref=continuation.subject_ref
ORDER BY continuation.authority_ref`)
	if err != nil {
		return invalidRecoveryV41ExpiredAgentLaunchContinuation(err)
	}
	defer authorityRows.Close()
	for authorityRows.Next() {
		var authorityRef, manifestDigest, authorityDigest, authorityBytesDigest, keyID string
		var authorityBytes, publicKey, signature, manifestBytes []byte
		var keyEpoch, trustRevision, issued, expires, admitted int64
		if err := authorityRows.Scan(
			&authorityRef, &manifestDigest, &authorityDigest, &authorityBytes, &authorityBytesDigest,
			&keyID, &keyEpoch, &trustRevision, &publicKey, &signature, &issued, &expires, &admitted,
			&manifestBytes,
		); err != nil {
			return invalidRecoveryV41ExpiredAgentLaunchContinuation(err)
		}
		decoded, decodeErr := microvm.DecodeExpiredLaunchContinuationAuthorityV1(authorityBytes)
		decodedManifest, manifestErr := microvm.DecodeExpiredLaunchContinuationManifestV1(manifestBytes)
		canonicalDigest, digestErr := microvm.ExpiredLaunchContinuationAuthoritySHA256V1(decoded)
		decodedSignature, signatureErr := base64.StdEncoding.DecodeString(decoded.SignatureBase64)
		verifyErr := microvm.VerifyExpiredLaunchContinuationAuthorityV1(ed25519.PublicKey(publicKey), decoded)
		if decodeErr != nil || manifestErr != nil || digestErr != nil || signatureErr != nil || verifyErr != nil ||
			continuationBytesSHA256(authorityBytes) != authorityBytesDigest || canonicalDigest != authorityDigest ||
			!reflect.DeepEqual(decoded.Manifest, decodedManifest) || decoded.Content.AuthorityRef != authorityRef ||
			decoded.Content.ManifestSHA256 != manifestDigest || decoded.Content.KeyID != keyID || keyEpoch <= 0 ||
			trustRevision <= 0 || issued <= 0 || expires <= issued || admitted < issued || admitted >= expires ||
			decoded.Content.KeyEpoch != uint64(keyEpoch) || decoded.Content.TrustRevision != uint64(trustRevision) ||
			decoded.Content.IssuedUnixMS != uint64(issued) || decoded.Content.ExpiresUnixMS != uint64(expires) ||
			!bytes.Equal(decodedSignature, signature) {
			return invalidRecoveryV41ExpiredAgentLaunchContinuation(
				errors.Join(decodeErr, manifestErr, digestErr, signatureErr, verifyErr),
			)
		}
	}
	if err := authorityRows.Err(); err != nil {
		return invalidRecoveryV41ExpiredAgentLaunchContinuation(err)
	}
	return nil
}

func continuationBytesSHA256(value []byte) string {
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}

func invalidRecoveryV41ExpiredAgentLaunchContinuation(cause error) error {
	return invalid(errors.Join(cause, errors.New("sqlite.recovery_v41_expired_launch_continuation_invalid")))
}
