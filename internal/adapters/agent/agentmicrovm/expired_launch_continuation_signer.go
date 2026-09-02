package agentmicrovm

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"orquesta/internal/credentials"
)

const ExpiredLaunchContinuationSigningCredentialPurposeV41 = credentials.PurposeRef(
	"orquesta.microvm-expired-launch-continuation-authority.v1",
)

type ExpiredLaunchContinuationAuthoritySignerConfigV41 struct {
	Store                   credentials.Store
	AuthorityReader         credentials.UseAuthorityReader
	CredentialRef           credentials.CredentialRef
	KeyID                   string
	KeyEpoch                uint64
	TrustRevision           uint64
	ExpectedPublicKeySHA256 string
	Validity                time.Duration
}

// ExpiredLaunchContinuationAuthoritySignerV41 borrows the configured private
// key for one callback. The public key and trust tuple are pinned by product
// configuration; no trust value is accepted from a command or database row.
type ExpiredLaunchContinuationAuthoritySignerV41 struct {
	store                   credentials.Store
	authorityReader         credentials.UseAuthorityReader
	credentialRef           credentials.CredentialRef
	keyID                   string
	keyEpoch                uint64
	trustRevision           uint64
	expectedPublicKeySHA256 string
	validity                time.Duration
}

func NewExpiredLaunchContinuationAuthoritySignerV41(
	config ExpiredLaunchContinuationAuthoritySignerConfigV41,
) (*ExpiredLaunchContinuationAuthoritySignerV41, error) {
	if nilInterface(config.Store) || nilInterface(config.AuthorityReader) ||
		credentials.ValidateCredentialRef(config.CredentialRef) != nil ||
		!validExpiredLaunchContinuationKeyIDV41(config.KeyID) ||
		config.KeyEpoch == 0 || config.TrustRevision == 0 ||
		!validLowerSHA256(config.ExpectedPublicKeySHA256) ||
		config.Validity <= 0 || config.Validity > 5*time.Minute {
		return nil, fail(CodeConfigurationInvalid, nil)
	}
	return &ExpiredLaunchContinuationAuthoritySignerV41{
		store: config.Store, authorityReader: config.AuthorityReader,
		credentialRef: config.CredentialRef, keyID: config.KeyID,
		keyEpoch: config.KeyEpoch, trustRevision: config.TrustRevision,
		expectedPublicKeySHA256: config.ExpectedPublicKeySHA256, validity: config.Validity,
	}, nil
}

func validExpiredLaunchContinuationKeyIDV41(value string) bool {
	if !strings.HasPrefix(value, "continuation-") || len(value) > 128 {
		return false
	}
	for _, character := range []byte(strings.TrimPrefix(value, "continuation-")) {
		if character >= 'a' && character <= 'z' || character >= '0' && character <= '9' ||
			character == '-' || character == '_' {
			continue
		}
		return false
	}
	return len(value) > len("continuation-")
}

func (signer *ExpiredLaunchContinuationAuthoritySignerV41) Issue(
	ctx context.Context,
	transport *ExpiredLaunchContinuationTransportV41,
	prepared PreparedExpiredLaunchContinuationV41,
	actorRef string,
	projectRef string,
	credentialRequestRef string,
	authorityRef string,
	issuedAt time.Time,
) (IssuedExpiredLaunchContinuationV41, error) {
	if signer == nil || nilInterface(signer.store) || nilInterface(signer.authorityReader) ||
		transport == nil || ctx == nil || issuedAt.IsZero() || issuedAt.UTC().UnixMilli() <= 0 {
		return IssuedExpiredLaunchContinuationV41{}, fail(CodeConfigurationInvalid, nil)
	}
	if err := ctx.Err(); err != nil {
		return IssuedExpiredLaunchContinuationV41{}, err
	}
	describe := credentials.DescribeUseAuthorityRequest{
		ActorRef: actorRef, RequestRef: credentialRequestRef,
		CredentialRef: signer.credentialRef, OwnerRef: credentials.OwnerRef(actorRef),
		ScopeRef: credentials.ScopeRef(projectRef), PurposeRef: ExpiredLaunchContinuationSigningCredentialPurposeV41,
	}
	if err := credentials.ValidateDescribeUseAuthorityRequest(describe); err != nil {
		return IssuedExpiredLaunchContinuationV41{}, fail(CodeSigningFailed, err)
	}
	described, err := signer.authorityReader.DescribeUseAuthority(ctx, describe)
	if err != nil || credentials.ValidateDescribedUseAuthority(describe, described) != nil {
		return IssuedExpiredLaunchContinuationV41{}, fail(CodeSigningFailed, err)
	}
	use := credentials.UseRequest{
		ActorRef: describe.ActorRef, RequestRef: describe.RequestRef,
		CredentialRef: describe.CredentialRef, OwnerRef: describe.OwnerRef,
		ScopeRef: describe.ScopeRef, PurposeRef: describe.PurposeRef, Version: described.Version,
	}
	var issued IssuedExpiredLaunchContinuationV41
	var issueErr error
	_, useErr := signer.store.Use(ctx, use, func(secret credentials.Secret) error {
		defer secret.Destroy()
		material := secret.Bytes()
		defer clear(material)
		if len(material) != ed25519.PrivateKeySize {
			issueErr = fail(CodeSigningFailed, nil)
			return issueErr
		}
		public := ed25519.PrivateKey(material).Public().(ed25519.PublicKey)
		digest := sha256.Sum256(public)
		if hex.EncodeToString(digest[:]) != signer.expectedPublicKeySHA256 {
			issueErr = fail(CodeSigningFailed, ErrExpiredLaunchContinuationDivergent)
			return issueErr
		}
		issuedUnixMS := uint64(issuedAt.UTC().UnixMilli())
		expiresUnixMS := uint64(issuedAt.UTC().Add(signer.validity).UnixMilli())
		issued, issueErr = transport.Issue(prepared, ExpiredLaunchContinuationIssuanceV41{
			AuthorityRef: authorityRef, IssuedUnixMS: issuedUnixMS, ExpiresUnixMS: expiresUnixMS,
			KeyID: signer.keyID, KeyEpoch: signer.keyEpoch, TrustRevision: signer.trustRevision,
		}, ed25519.PrivateKey(material))
		return issueErr
	})
	if issueErr != nil {
		return IssuedExpiredLaunchContinuationV41{}, issueErr
	}
	if useErr != nil {
		if errors.Is(useErr, context.Canceled) || errors.Is(useErr, context.DeadlineExceeded) {
			return IssuedExpiredLaunchContinuationV41{}, useErr
		}
		return IssuedExpiredLaunchContinuationV41{}, fail(CodeSigningFailed, useErr)
	}
	content := issued.Authority.Content
	if content.KeyID != signer.keyID || content.KeyEpoch != signer.keyEpoch ||
		content.TrustRevision != signer.trustRevision || bytesSHA256(issued.PublicKey) != signer.expectedPublicKeySHA256 {
		return IssuedExpiredLaunchContinuationV41{}, fail(CodeSigningFailed, ErrExpiredLaunchContinuationDivergent)
	}
	return issued, nil
}
