package fake

import (
	"crypto/sha256"
	"encoding/hex"

	"orquesta/internal/goal"
)

func fakeReceiptRef(kind string, executionRef goal.ExecutionRef, idempotencyKey string) string {
	digest := sha256.Sum256([]byte(kind + "\x00" + executionRef.String() + "\x00" + idempotencyKey))
	return "fake-" + kind + ":" + hex.EncodeToString(digest[:])
}
