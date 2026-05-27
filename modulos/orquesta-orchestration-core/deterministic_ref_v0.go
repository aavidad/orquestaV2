package orquestacionnucleoapp

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
)

func deterministicRefDigestV0(namespace string, fields ...string) string {
	sum := sha256.Sum256(deterministicRefPayloadV0(namespace, fields...))
	return hex.EncodeToString(sum[:])
}

func deterministicRefDigestPrefixV0(namespace string, size int, fields ...string) string {
	digest := deterministicRefDigestV0(namespace, fields...)
	if size <= 0 || size > len(digest) {
		return digest
	}
	return digest[:size]
}

func deterministicRefPayloadV0(namespace string, fields ...string) []byte {
	var builder strings.Builder
	builder.WriteString("orquesta-orchestration-core.ref.v0|")
	builder.WriteString(strings.TrimSpace(namespace))
	builder.WriteByte('|')
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
