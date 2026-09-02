package agentmicrovm

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/credentials"
)

type expiredContinuationAuthorityReaderStub struct {
	version  credentials.Version
	requests []credentials.DescribeUseAuthorityRequest
}

func (reader *expiredContinuationAuthorityReaderStub) DescribeUseAuthority(
	_ context.Context,
	request credentials.DescribeUseAuthorityRequest,
) (credentials.DescribedUseAuthority, error) {
	reader.requests = append(reader.requests, request)
	return credentials.DescribedUseAuthority{
		CredentialRef: request.CredentialRef, OwnerRef: request.OwnerRef,
		ScopeRef: request.ScopeRef, PurposeRef: request.PurposeRef, Version: reader.version,
	}, nil
}

func TestExpiredLaunchContinuationAuthoritySignerBorrowsPinnedExternalKey(t *testing.T) {
	fixture := readExpiredLaunchContinuationFixture(t)
	subject := expiredContinuationSubjectFromFixture(t, fixture)
	manifestDigest, err := ExpiredLaunchContinuationManifestSHA256V1(subject.ExpectedManifest)
	if err != nil {
		t.Fatal(err)
	}
	client := &expiredContinuationClientStub{
		capabilities: validExpiredContinuationCapabilities(),
		prepared:     microvmPreparation(subject.ExpectedManifest, manifestDigest),
	}
	transport, err := NewExpiredLaunchContinuationTransportV41(client)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := transport.PrepareObserved(context.Background(), subject)
	if err != nil {
		t.Fatal(err)
	}
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{9}, ed25519.SeedSize))
	publicDigest := sha256.Sum256(privateKey.Public().(ed25519.PublicKey))
	store := &credentialSignerStoreStub{material: append([]byte(nil), privateKey...)}
	reader := &expiredContinuationAuthorityReaderStub{version: 7}
	signer, err := NewExpiredLaunchContinuationAuthoritySignerV41(
		ExpiredLaunchContinuationAuthoritySignerConfigV41{
			Store: store, AuthorityReader: reader,
			CredentialRef: "credential:microvm-continuation-signing",
			KeyID:         "continuation-run7", KeyEpoch: 4, TrustRevision: 8,
			ExpectedPublicKeySHA256: hex.EncodeToString(publicDigest[:]), Validity: 2 * time.Minute,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	issuedAt := time.UnixMilli(1_788_000_000_000).UTC()
	issued, err := signer.Issue(
		context.Background(), transport, prepared, "actor:owner", "project:run7",
		"request:continuation-sign:run7", "continuation:run7", issuedAt,
	)
	if err != nil {
		t.Fatal(err)
	}
	wantUse := credentials.UseRequest{
		ActorRef: "actor:owner", RequestRef: "request:continuation-sign:run7",
		CredentialRef: "credential:microvm-continuation-signing", OwnerRef: "actor:owner",
		ScopeRef: "project:run7", PurposeRef: ExpiredLaunchContinuationSigningCredentialPurposeV41,
		Version: 7,
	}
	if len(store.calls) != 1 || store.calls[0] != wantUse || len(reader.requests) != 1 ||
		issued.Authority.Content.KeyID != "continuation-run7" || issued.Authority.Content.KeyEpoch != 4 ||
		issued.Authority.Content.TrustRevision != 8 || bytesSHA256(issued.PublicKey) != hex.EncodeToString(publicDigest[:]) {
		t.Fatalf("issued=%+v use=%+v describe=%+v", issued, store.calls, reader.requests)
	}
	if len(store.retained) != 1 || !reflect.DeepEqual(store.retained[0].Bytes(), make([]byte, ed25519.PrivateKeySize)) {
		t.Fatal("borrowed private key survived callback")
	}
}

func TestExpiredLaunchContinuationAuthoritySignerRejectsTrustDivergence(t *testing.T) {
	fixture := readExpiredLaunchContinuationFixture(t)
	subject := expiredContinuationSubjectFromFixture(t, fixture)
	digest, _ := ExpiredLaunchContinuationManifestSHA256V1(subject.ExpectedManifest)
	client := &expiredContinuationClientStub{
		capabilities: validExpiredContinuationCapabilities(),
		prepared:     microvmPreparation(subject.ExpectedManifest, digest),
	}
	transport, _ := NewExpiredLaunchContinuationTransportV41(client)
	prepared, err := transport.PrepareObserved(context.Background(), subject)
	if err != nil {
		t.Fatal(err)
	}
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{3}, ed25519.SeedSize))
	store := &credentialSignerStoreStub{material: append([]byte(nil), privateKey...)}
	signer, err := NewExpiredLaunchContinuationAuthoritySignerV41(
		ExpiredLaunchContinuationAuthoritySignerConfigV41{
			Store: store, AuthorityReader: &expiredContinuationAuthorityReaderStub{version: 1},
			CredentialRef: "credential:microvm-continuation-signing",
			KeyID:         "continuation-run7", KeyEpoch: 1, TrustRevision: 1,
			ExpectedPublicKeySHA256: strings.Repeat("a", 64), Validity: time.Minute,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = signer.Issue(context.Background(), transport, prepared, "actor:owner", "project:run7",
		"request:continuation-sign:run7", "continuation:run7", time.Unix(1_788_000_000, 0).UTC())
	if ErrorCode(err) != CodeSigningFailed || !errors.Is(err, ErrExpiredLaunchContinuationDivergent) {
		t.Fatalf("trust divergence error=%v", err)
	}
}

func microvmPreparation(
	manifest microvm.ExpiredLaunchContinuationManifestV1,
	digest string,
) microvm.RespuestaPreparacionContinuacionLanzamientoCaducadoV1 {
	return microvm.RespuestaPreparacionContinuacionLanzamientoCaducadoV1{
		Manifiesto: manifest, ManifiestoSHA256: digest,
	}
}
