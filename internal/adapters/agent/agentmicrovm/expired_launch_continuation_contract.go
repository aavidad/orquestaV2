package agentmicrovm

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"
)

const (
	ExpiredLaunchContinuationManifestSchemaV1  = microvm.ExpiredLaunchContinuationManifestSchemaV1
	ExpiredLaunchContinuationAuthoritySchemaV1 = microvm.ExpiredLaunchContinuationAuthoritySchemaV1
	ExpiredLaunchContinuationAudienceV1        = microvm.ExpiredLaunchContinuationAudienceV1
	ExpiredLaunchContinuationPurposeV1         = microvm.ExpiredLaunchContinuationPurposeV1
	ExpiredLaunchContinuationAlgorithmV1       = microvm.ExpiredLaunchContinuationAlgorithmV1
)

type ExpiredLaunchContinuationTargetV1 = microvm.ExpiredLaunchContinuationTargetV1
type ExpiredLaunchContinuationManifestV1 = microvm.ExpiredLaunchContinuationManifestV1
type ExpiredLaunchContinuationAuthorityContentV1 = microvm.ExpiredLaunchContinuationAuthorityContentV1
type ExpiredLaunchContinuationAuthorityV1 = microvm.ExpiredLaunchContinuationAuthorityV1

func DecodeExpiredLaunchContinuationManifestV1(data []byte) (ExpiredLaunchContinuationManifestV1, error) {
	return microvm.DecodeExpiredLaunchContinuationManifestV1(data)
}

func DecodeExpiredLaunchContinuationAuthorityV1(data []byte) (ExpiredLaunchContinuationAuthorityV1, error) {
	return microvm.DecodeExpiredLaunchContinuationAuthorityV1(data)
}

func ExpiredLaunchContinuationManifestMessageV1(manifest ExpiredLaunchContinuationManifestV1) ([]byte, error) {
	return microvm.ExpiredLaunchContinuationManifestMessageV1(manifest)
}

func ExpiredLaunchContinuationManifestSHA256V1(manifest ExpiredLaunchContinuationManifestV1) (string, error) {
	return microvm.ExpiredLaunchContinuationManifestSHA256V1(manifest)
}

func ExpiredLaunchContinuationSigningMessageV1(content ExpiredLaunchContinuationAuthorityContentV1) ([]byte, error) {
	return microvm.ExpiredLaunchContinuationSigningMessageV1(content)
}

func ExpiredLaunchContinuationAuthoritySHA256V1(authority ExpiredLaunchContinuationAuthorityV1) (string, error) {
	return microvm.ExpiredLaunchContinuationAuthoritySHA256V1(authority)
}

func SignExpiredLaunchContinuationAuthorityV1(
	privateKey ed25519.PrivateKey,
	content ExpiredLaunchContinuationAuthorityContentV1,
	manifest ExpiredLaunchContinuationManifestV1,
) (ExpiredLaunchContinuationAuthorityV1, error) {
	return microvm.SignExpiredLaunchContinuationAuthorityV1(privateKey, content, manifest)
}

func VerifyExpiredLaunchContinuationAuthorityV1(
	publicKey ed25519.PublicKey,
	authority ExpiredLaunchContinuationAuthorityV1,
) error {
	return microvm.VerifyExpiredLaunchContinuationAuthorityV1(publicKey, authority)
}

func validLowerSHA256(value string) bool {
	if len(value) != sha256.Size*2 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
