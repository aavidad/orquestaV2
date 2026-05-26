package orquestaappcodexstack

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
)

func codexStackDeterministicRefV0(prefix string, fields ...string) string {
	return strings.TrimSpace(prefix) + codexStackDeterministicDigestV0(fields...)
}

func codexStackDeterministicDigestV0(fields ...string) string {
	sum := sha256.Sum256(codexStackDeterministicRefPayloadV0(fields...))
	return hex.EncodeToString(sum[:])
}

func codexStackDeterministicRefPayloadV0(fields ...string) []byte {
	var builder strings.Builder
	builder.WriteString("orquesta-app-codex-stack.ref.v0|")
	builder.WriteString(strconv.Itoa(len(fields)))
	builder.WriteByte('|')
	for _, field := range fields {
		value := strings.TrimSpace(field)
		builder.WriteString(strconv.Itoa(len(value)))
		builder.WriteByte(':')
		builder.WriteString(value)
		builder.WriteByte(';')
	}
	return []byte(builder.String())
}
