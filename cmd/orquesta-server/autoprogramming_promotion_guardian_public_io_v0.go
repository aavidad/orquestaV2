package main

import "bytes"

const autoprogrammingPromotionGuardianOutputMaxBytesV0 = 64 * 1024

type autoprogrammingPromotionGuardianOutputBufferV0 struct {
	buffer    bytes.Buffer
	limit     int
	truncated bool
}

func newAutoprogrammingPromotionGuardianOutputBufferV0() *autoprogrammingPromotionGuardianOutputBufferV0 {
	return &autoprogrammingPromotionGuardianOutputBufferV0{limit: autoprogrammingPromotionGuardianOutputMaxBytesV0}
}

func (buffer *autoprogrammingPromotionGuardianOutputBufferV0) Write(chunk []byte) (int, error) {
	if buffer.limit <= 0 || buffer.buffer.Len() >= buffer.limit {
		buffer.truncated = buffer.truncated || len(chunk) > 0
		return len(chunk), nil
	}
	remaining := buffer.limit - buffer.buffer.Len()
	if len(chunk) > remaining {
		_, _ = buffer.buffer.Write(chunk[:remaining])
		buffer.truncated = true
		return len(chunk), nil
	}
	_, _ = buffer.buffer.Write(chunk)
	return len(chunk), nil
}

func (buffer *autoprogrammingPromotionGuardianOutputBufferV0) String() string {
	return buffer.buffer.String()
}
