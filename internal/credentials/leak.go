package credentials

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
)

type LeakSurface struct {
	Name    string
	Content []byte
}

// LeakGuard scans exact material without ever rendering it in an error.
type LeakGuard struct{ material []byte }

func NewLeakGuard(secret Secret) (*LeakGuard, error) {
	if len(secret.material) == 0 {
		return nil, NewError(ErrorInvalidRequest, "secret")
	}
	return &LeakGuard{material: secret.Bytes()}, nil
}

func (*LeakGuard) String() string               { return "[REDACTED]" }
func (*LeakGuard) GoString() string             { return "[REDACTED]" }
func (*LeakGuard) MarshalJSON() ([]byte, error) { return json.Marshal("[REDACTED]") }

func (guard *LeakGuard) Scan(surfaces []LeakSurface) error {
	if guard == nil || len(guard.material) == 0 {
		return NewError(ErrorInvalidRequest, "leak_guard")
	}
	signatures := leakSignatures(guard.material)
	defer clearDerivedSignatures(signatures)
	for _, surface := range surfaces {
		if containsSignature(surface.Content, signatures) {
			return NewError(ErrorSecretLeak, "surface")
		}
	}
	return nil
}

func (guard *LeakGuard) Redact(content []byte) []byte {
	copyOfContent := append([]byte(nil), content...)
	if guard == nil || len(guard.material) == 0 {
		return copyOfContent
	}
	signatures := leakSignatures(guard.material)
	defer clearDerivedSignatures(signatures)
	replacement := []byte("[REDACTED]")
	if containsSignature(replacement, signatures) {
		replacement = nil
	}
	for _, signature := range signatures {
		if !bytes.Contains(copyOfContent, signature) {
			continue
		}
		next := bytes.ReplaceAll(copyOfContent, signature, replacement)
		clear(copyOfContent)
		copyOfContent = next
	}
	if containsSignature(copyOfContent, signatures) {
		clear(copyOfContent)
		return nil
	}
	return copyOfContent
}

func containsSignature(content []byte, signatures [][]byte) bool {
	for _, signature := range signatures {
		if bytes.Contains(content, signature) {
			return true
		}
	}
	return false
}

// Destroy removes the guard's retained material. It is safe to call repeatedly.
func (guard *LeakGuard) Destroy() {
	if guard == nil {
		return
	}
	clear(guard.material)
	guard.material = nil
}

func leakSignatures(material []byte) [][]byte {
	signatures := [][]byte{material}
	appendUnique := func(candidate []byte) {
		if len(candidate) == 0 {
			clear(candidate)
			return
		}
		for _, existing := range signatures {
			if bytes.Equal(existing, candidate) {
				clear(candidate)
				return
			}
		}
		signatures = append(signatures, candidate)
	}
	for _, encoding := range []*base64.Encoding{
		base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding,
	} {
		encoded := make([]byte, encoding.EncodedLen(len(material)))
		encoding.Encode(encoded, material)
		appendUnique(encoded)
	}
	lowerHex := make([]byte, hex.EncodedLen(len(material)))
	hex.Encode(lowerHex, material)
	upperHex := append([]byte(nil), lowerHex...)
	appendUnique(lowerHex)
	for index, value := range upperHex {
		if value >= 'a' && value <= 'f' {
			upperHex[index] -= 'a' - 'A'
		}
	}
	appendUnique(upperHex)
	return signatures
}

func clearDerivedSignatures(signatures [][]byte) {
	for index := 1; index < len(signatures); index++ {
		clear(signatures[index])
	}
}
