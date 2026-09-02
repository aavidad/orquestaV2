package agentmicrovm

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

type expiredLaunchContinuationFixture struct {
	Manifest             ExpiredLaunchContinuationManifestV1         `json:"manifiesto"`
	ManifestSHA256       string                                      `json:"manifiesto_sha256"`
	Content              ExpiredLaunchContinuationAuthorityContentV1 `json:"contenido"`
	PublicKeyBase64      string                                      `json:"clave_publica_base64"`
	SigningMessageSHA256 string                                      `json:"mensaje_firma_sha256"`
	AuthoritySHA256      string                                      `json:"autoridad_sha256"`
	SignatureBase64      string                                      `json:"firma_base64"`
}

func TestExpiredLaunchContinuationMatchesRustGoldenV1(t *testing.T) {
	fixture := readExpiredLaunchContinuationFixture(t)
	authority := ExpiredLaunchContinuationAuthorityV1{
		Content: fixture.Content, Manifest: fixture.Manifest, SignatureBase64: fixture.SignatureBase64,
	}
	manifestDigest, err := ExpiredLaunchContinuationManifestSHA256V1(authority.Manifest)
	if err != nil || manifestDigest != fixture.ManifestSHA256 {
		t.Fatalf("manifest digest=%q err=%v", manifestDigest, err)
	}
	message, err := ExpiredLaunchContinuationSigningMessageV1(authority.Content)
	if err != nil {
		t.Fatal(err)
	}
	messageDigest := sha256.Sum256(message)
	if got := hex.EncodeToString(messageDigest[:]); got != fixture.SigningMessageSHA256 {
		t.Fatalf("signing message digest=%q", got)
	}
	authorityDigest, err := ExpiredLaunchContinuationAuthoritySHA256V1(authority)
	if err != nil || authorityDigest != fixture.AuthoritySHA256 {
		t.Fatalf("authority digest=%q err=%v", authorityDigest, err)
	}
	publicKey, err := base64.StdEncoding.DecodeString(fixture.PublicKeyBase64)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyExpiredLaunchContinuationAuthorityV1(ed25519.PublicKey(publicKey), authority); err != nil {
		t.Fatalf("fixture signature: %v", err)
	}
	encoded, err := json.Marshal(authority)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeExpiredLaunchContinuationAuthorityV1(encoded)
	if err != nil || !reflect.DeepEqual(decoded, authority) {
		t.Fatalf("decoded=%+v err=%v", decoded, err)
	}
}

func TestExpiredLaunchContinuationSignerIsDetachedAndRejectsMutation(t *testing.T) {
	fixture := readExpiredLaunchContinuationFixture(t)
	seed := sha256.Sum256([]byte("orquesta-expired-launch-continuation-test-key-v1"))
	privateKey := ed25519.NewKeyFromSeed(seed[:])
	defer clear(privateKey)
	authority, err := SignExpiredLaunchContinuationAuthorityV1(privateKey, fixture.Content, fixture.Manifest)
	if err != nil {
		t.Fatal(err)
	}
	publicKey := privateKey.Public().(ed25519.PublicKey)
	if err := VerifyExpiredLaunchContinuationAuthorityV1(publicKey, authority); err != nil {
		t.Fatal(err)
	}
	mutated := authority
	mutated.Content.TrustRevision++
	if err := VerifyExpiredLaunchContinuationAuthorityV1(publicKey, mutated); err == nil {
		t.Fatal("mutated authority accepted")
	}
	if authority.Content.TrustRevision != fixture.Content.TrustRevision || !reflect.DeepEqual(authority.Manifest, fixture.Manifest) {
		t.Fatal("signer mutated caller-owned subject")
	}
}

func TestDecodeExpiredLaunchContinuationRejectsUnknownAndOversizedFields(t *testing.T) {
	fixture := readExpiredLaunchContinuationFixture(t)
	authority := ExpiredLaunchContinuationAuthorityV1{
		Content: fixture.Content, Manifest: fixture.Manifest, SignatureBase64: fixture.SignatureBase64,
	}
	encoded, err := json.Marshal(authority)
	if err != nil {
		t.Fatal(err)
	}
	withUnknown := append(encoded[:len(encoded)-1], []byte(`,"desconocido":true}`)...)
	if _, err := DecodeExpiredLaunchContinuationAuthorityV1(withUnknown); err == nil {
		t.Fatal("unknown field accepted")
	}
	authority.Content.KeyID = strings.Repeat("x", 129)
	if _, err := ExpiredLaunchContinuationAuthoritySHA256V1(authority); err == nil {
		t.Fatal("oversized key id accepted")
	}
}

func readExpiredLaunchContinuationFixture(t *testing.T) expiredLaunchContinuationFixture {
	t.Helper()
	data, err := os.ReadFile("testdata/expired_launch_continuation_v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture expiredLaunchContinuationFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	return fixture
}
