package bootstrap

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	credentiallocal "orquesta/internal/adapters/credentials/local"
	"orquesta/internal/config"
	"orquesta/internal/credentials"
)

func TestProvisionMicroVMContinuationAuthorityCLIBootstrapsAndReconsultsPublicTuple(t *testing.T) {
	root, configPath := continuationCredentialProvisionCLIFixture(t, false)
	arguments := continuationCredentialProvisionCLIArguments(
		configPath, "request:continuation-credential-cli-e2e",
	)
	catalog := loadCredentialProvisionCatalog(t)

	var stdout, stderr bytes.Buffer
	code := RunProvisionMicroVMContinuationAuthority(
		context.Background(), arguments, catalog, &stdout, &stderr,
	)
	first := decodeContinuationCredentialProvisionOutput(t, stdout.Bytes())
	publicKey := decodeContinuationPublicKey(t, first.PublicKeyBase64)
	digest := sha256.Sum256(publicKey)
	if code != 0 || stderr.Len() != 0 || first.SchemaVersion != continuationCredentialProvisionOutputSchema ||
		first.Status != "complete" || first.CredentialState != string(credentials.CredentialProvisionCreated) ||
		first.CredentialVersion != 1 || first.KeyID != "continuation-runtime-isolation" ||
		first.KeyEpoch != 1 || first.TrustRevision != 1 ||
		first.PublicKeySHA256 != hex.EncodeToString(digest[:]) ||
		first.ConfiguredPublicKeySHA256Matches != nil || first.Error != nil {
		t.Fatalf("bootstrap code=%d output=%+v stderr=%q", code, first, stderr.String())
	}

	privateKey := readCredentialProvisionMaterial(
		t, root,
		"credential:microvm-continuation-signing",
		"orquesta.microvm-expired-launch-continuation-authority.v1",
		"request:continuation-credential-cli-test-read",
	)
	defer clear(privateKey)
	if len(privateKey) != ed25519.PrivateKeySize ||
		!bytes.Equal(privateKey[ed25519.SeedSize:], publicKey) {
		t.Fatal("stored continuation key does not match public tuple")
	}
	assertCredentialProvisionOutputExcludes(
		t, append(stdout.Bytes(), stderr.Bytes()...),
		configPath, filepath.Join(root, "secrets", "credentials.json"), privateKey,
	)

	digestText := hex.EncodeToString(digest[:])
	replaceTestConfigValue(
		t, configPath,
		"expired_launch_continuation_authority_trust_revision = 1\n",
		"expired_launch_continuation_authority_trust_revision = 1\n"+
			"expired_launch_continuation_authority_public_key_sha256 = \""+digestText+"\"\n",
	)
	stdout.Reset()
	stderr.Reset()
	code = RunProvisionMicroVMContinuationAuthority(
		context.Background(), arguments, catalog, &stdout, &stderr,
	)
	resumed := decodeContinuationCredentialProvisionOutput(t, stdout.Bytes())
	if code != 0 || stderr.Len() != 0 || resumed.Status != "complete" ||
		resumed.CredentialState != string(credentials.CredentialProvisionResumed) ||
		resumed.PublicKeyBase64 != first.PublicKeyBase64 ||
		resumed.PublicKeySHA256 != digestText ||
		resumed.ConfiguredPublicKeySHA256Matches == nil ||
		!*resumed.ConfiguredPublicKeySHA256Matches || resumed.Error != nil {
		t.Fatalf("resume code=%d output=%+v stderr=%q", code, resumed, stderr.String())
	}
	assertCredentialProvisionOutputExcludes(
		t, append(stdout.Bytes(), stderr.Bytes()...),
		configPath, filepath.Join(root, "secrets", "credentials.json"), privateKey,
	)
}

func TestProvisionMicroVMContinuationAuthorityCLIFailsClosedOnPinnedDigestMismatch(t *testing.T) {
	_, configPath := continuationCredentialProvisionCLIFixture(t, true)
	var stdout, stderr bytes.Buffer
	code := RunProvisionMicroVMContinuationAuthority(
		context.Background(),
		continuationCredentialProvisionCLIArguments(
			configPath, "request:continuation-credential-cli-digest",
		),
		loadCredentialProvisionCatalog(t), &stdout, &stderr,
	)
	output := decodeContinuationCredentialProvisionOutput(t, stdout.Bytes())
	if code != 1 || output.Status != "failed" || output.Error == nil ||
		output.Error.Code != "cli.continuation_credential_public_key_digest_mismatch" ||
		output.ConfiguredPublicKeySHA256Matches == nil ||
		*output.ConfiguredPublicKeySHA256Matches ||
		len(decodeContinuationPublicKey(t, output.PublicKeyBase64)) != ed25519.PublicKeySize ||
		!strings.Contains(stderr.String(), "code=cli.continuation_credential_public_key_digest_mismatch") {
		t.Fatalf("digest mismatch code=%d output=%+v stderr=%q", code, output, stderr.String())
	}
}

func TestProvisionMicroVMContinuationAuthorityCLIComposesOnlyCanonicalAuthority(t *testing.T) {
	root, configPath := continuationCredentialProvisionCLIFixture(t, false)
	snapshot, err := loadContinuationCredentialProvisionConfigSnapshot(context.Background(), configPath)
	if err != nil {
		t.Fatal(err)
	}
	const uid = 4242
	store := &credentialProvisionCloseProbe{}
	var storeOptions credentiallocal.Options
	var received credentials.ProvisionEd25519SigningCredentialRequest
	keySource := &continuationCredentialKeySourceProbe{}
	dependencies := productionContinuationCredentialProvisionDependencies()
	dependencies.loadSnapshot = func(context.Context, string) (config.Snapshot, error) {
		return snapshot, nil
	}
	dependencies.effectiveUID = func() int { return uid }
	dependencies.openStore = func(options credentiallocal.Options) (credentialProvisionOwnedStore, error) {
		storeOptions = options
		return store, nil
	}
	dependencies.newKey = func() (credentials.Ed25519PrivateKeySource, error) {
		return keySource, nil
	}
	publicKey := bytes.Repeat([]byte{0x8a}, ed25519.PublicKeySize)
	dependencies.provision = func(
		_ context.Context,
		gotStore credentials.ProvisionStore,
		gotSource credentials.Ed25519PrivateKeySource,
		request credentials.ProvisionEd25519SigningCredentialRequest,
	) (credentials.ProvisionEd25519SigningCredentialResult, error) {
		if gotStore != store || gotSource != keySource {
			t.Fatal("composition changed store or key source authority")
		}
		received = request
		return continuationCredentialProvisionTestProjection(publicKey), nil
	}

	var stdout, stderr bytes.Buffer
	code := runProvisionMicroVMContinuationAuthority(
		context.Background(),
		continuationCredentialProvisionCLIArguments(
			configPath, "request:continuation-canonical-authority",
		),
		loadCredentialProvisionCatalog(t), &stdout, &stderr, dependencies,
	)
	if code != 0 || stderr.Len() != 0 || store.closes != 1 ||
		storeOptions.Path != snapshot.CredentialsLocalPath() || storeOptions.OwnerUID != uid ||
		storeOptions.MaxStoreBytes != snapshot.CredentialsLocalMaxDocumentBytes() {
		t.Fatalf("composition code=%d close=%d store=%+v stderr=%q", code, store.closes, storeOptions, stderr.String())
	}
	if received.ActorRef != snapshot.IdentityLocalActor() ||
		received.OwnerRef.String() != snapshot.IdentityLocalActor() ||
		received.Signing.CredentialRef.String() != string(
			snapshot.RuntimeMicroVMExpiredLaunchContinuationAuthoritySigningCredentialRef(),
		) || len(received.Signing.ScopeRefs) != 1 ||
		received.Signing.ScopeRefs[0].String() != snapshot.ProjectDefault() ||
		received.Signing.PurposeRef != "orquesta.microvm-expired-launch-continuation-authority.v1" ||
		received.Signing.RequestRef != "request:continuation-canonical-authority:continuation-signing" {
		t.Fatalf("noncanonical continuation request=%+v", received)
	}
	projection := stdout.Bytes()
	for _, forbidden := range []string{
		root, snapshot.CredentialsLocalPath(), received.Signing.CredentialRef.String(),
		received.ActorRef, received.OwnerRef.String(), received.Signing.ScopeRefs[0].String(),
		received.Signing.PurposeRef.String(),
	} {
		if bytes.Contains(projection, []byte(forbidden)) {
			t.Fatalf("public tuple leaked internal authority value")
		}
	}
}

func TestProvisionMicroVMContinuationAuthorityCLIRejectsEverythingExceptMissingDigestBeforeStore(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*testing.T, string) []string
	}{
		{name: "relative config", mutate: func(_ *testing.T, path string) []string {
			return continuationCredentialProvisionCLIArguments(filepath.Base(path), "request:continuation-relative")
		}},
		{name: "invalid request ref", mutate: func(_ *testing.T, path string) []string {
			return continuationCredentialProvisionCLIArguments(path, "other:continuation")
		}},
		{name: "missing key epoch", mutate: func(t *testing.T, path string) []string {
			removeTestConfigLines(t, path, "expired_launch_continuation_authority_key_epoch = ")
			return continuationCredentialProvisionCLIArguments(path, "request:continuation-missing-epoch")
		}},
		{name: "invalid credential ref", mutate: func(t *testing.T, path string) []string {
			replaceTestConfigValue(
				t, path,
				`expired_launch_continuation_authority_signing_credential_ref = "credential:microvm-continuation-signing"`,
				`expired_launch_continuation_authority_signing_credential_ref = "credential:../escape"`,
			)
			return continuationCredentialProvisionCLIArguments(path, "request:continuation-invalid-ref")
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			root, configPath := continuationCredentialProvisionCLIFixture(t, false)
			opened, generated, provisioned := false, false, false
			dependencies := productionContinuationCredentialProvisionDependencies()
			dependencies.openStore = func(credentiallocal.Options) (credentialProvisionOwnedStore, error) {
				opened = true
				return nil, errors.New("must not open")
			}
			dependencies.newKey = func() (credentials.Ed25519PrivateKeySource, error) {
				generated = true
				return nil, errors.New("must not generate")
			}
			dependencies.provision = func(
				context.Context,
				credentials.ProvisionStore,
				credentials.Ed25519PrivateKeySource,
				credentials.ProvisionEd25519SigningCredentialRequest,
			) (credentials.ProvisionEd25519SigningCredentialResult, error) {
				provisioned = true
				return credentials.ProvisionEd25519SigningCredentialResult{}, errors.New("must not provision")
			}
			var stdout, stderr bytes.Buffer
			code := runProvisionMicroVMContinuationAuthority(
				context.Background(), test.mutate(t, configPath),
				loadCredentialProvisionCatalog(t), &stdout, &stderr, dependencies,
			)
			if code != 2 || stdout.Len() != 0 || opened || generated || provisioned ||
				!strings.Contains(stderr.String(), "code=cli.credential_provision_") {
				t.Fatalf("code=%d opened=%t generated=%t provisioned=%t stdout=%q stderr=%q",
					code, opened, generated, provisioned, stdout.String(), stderr.String())
			}
			if _, err := os.Lstat(filepath.Join(root, "secrets")); err != nil {
				t.Fatalf("pre-existing credential directory changed unexpectedly: %v", err)
			}
		})
	}
}

func continuationCredentialProvisionCLIFixture(t *testing.T, keepDigest bool) (string, string) {
	t.Helper()
	root := t.TempDir()
	configPath, _, _ := writeRuntimeIsolationMicroVMConfig(t, root, false)
	if !keepDigest {
		removeTestConfigLines(
			t, configPath,
			"expired_launch_continuation_authority_public_key_sha256 = ",
		)
	}
	if err := os.Mkdir(filepath.Join(root, "secrets"), 0o700); err != nil {
		t.Fatal(err)
	}
	return root, configPath
}

func continuationCredentialProvisionCLIArguments(configPath, requestRef string) []string {
	return []string{"--config", configPath, "--request-ref", requestRef}
}

func decodeContinuationCredentialProvisionOutput(
	t *testing.T,
	content []byte,
) continuationCredentialProvisionOutput {
	t.Helper()
	var output continuationCredentialProvisionOutput
	if !json.Valid(content) {
		t.Fatalf("invalid continuation credential JSON output=%q", content)
	}
	if err := json.Unmarshal(content, &output); err != nil {
		t.Fatal(err)
	}
	return output
}

func decodeContinuationPublicKey(t *testing.T, value string) []byte {
	t.Helper()
	publicKey, err := base64.StdEncoding.DecodeString(value)
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		t.Fatalf("invalid public key projection len=%d err=%v", len(publicKey), err)
	}
	return publicKey
}

func continuationCredentialProvisionTestProjection(
	publicKey []byte,
) credentials.ProvisionEd25519SigningCredentialResult {
	return credentials.ProvisionEd25519SigningCredentialResult{
		Signing: credentials.CredentialProvisionStatus{
			State: credentials.CredentialProvisionCreated,
			RequestedAuthority: credentials.RequestedCredentialAuthority{
				CredentialRef: "credential:internal", OwnerRef: "actor:internal",
				ScopeRefs:  []credentials.ScopeRef{"project:internal"},
				PurposeRef: "purpose:internal", Version: 1,
			},
			Receipts: []credentials.Receipt{{
				CredentialRef: "credential:internal", OwnerRef: "actor:internal",
				PurposeRef: "purpose:internal", Version: 1,
				RequestRef: "request:internal", ActorRef: "actor:internal",
				Operation: "create", OccurredAt: time.Unix(1_700_000_000, 0).UTC(),
			}},
		},
		SigningPublicKeyBase64: base64.StdEncoding.EncodeToString(publicKey),
	}
}

type continuationCredentialKeySourceProbe struct{}

func (*continuationCredentialKeySourceProbe) WithPrivateKey(
	context.Context,
	func(credentials.Secret) error,
) error {
	return errors.New("must not be invoked by composition test")
}
