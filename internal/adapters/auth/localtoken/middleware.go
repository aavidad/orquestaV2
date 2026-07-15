package localtoken

import (
	"crypto/sha256"
	"crypto/subtle"
)

func (authenticator *Authenticator) matches(candidate string) bool {
	candidateDigest := sha256.Sum256([]byte(candidate))
	return subtle.ConstantTimeCompare(authenticator.digest[:], candidateDigest[:]) == 1
}
