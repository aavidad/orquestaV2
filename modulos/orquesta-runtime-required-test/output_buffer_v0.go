package orquestaruntimerequiredtest

import (
	"strings"
	"sync"
)

type outputBufferV0 struct {
	mu        sync.Mutex
	limit     int64
	used      int64
	truncated bool
	builder   strings.Builder
}

func newOutputBufferV0(limit int64) *outputBufferV0 {
	return &outputBufferV0{limit: limit}
}

func (buffer *outputBufferV0) Write(data []byte) (int, error) {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	if buffer.limit <= 0 || buffer.used >= buffer.limit {
		buffer.truncated = true
		return len(data), nil
	}
	remaining := buffer.limit - buffer.used
	writeLen := int64(len(data))
	if writeLen > remaining {
		writeLen = remaining
		buffer.truncated = true
	}
	buffer.builder.Write(data[:int(writeLen)])
	buffer.used += writeLen
	return len(data), nil
}

func (buffer *outputBufferV0) String() string {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.builder.String()
}

func (buffer *outputBufferV0) Truncated() bool {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.truncated
}
