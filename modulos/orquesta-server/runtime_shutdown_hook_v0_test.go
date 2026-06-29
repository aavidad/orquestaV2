package orquestaserver

import (
	"context"
	"sync/atomic"
	"testing"
)

func TestRuntimeV0CompactaShutdownHooksV0(t *testing.T) {
	hook := &countingRuntimeShutdownHookV0{}
	runtime, err := NewRuntimeV0(ConfigV0{
		Addr:          "127.0.0.1:0",
		StateDir:      t.TempDir(),
		AuditDisabled: true,
	}, RuntimeDepsV0{
		ShutdownHooks: []RuntimeShutdownHookPortV0{nil, hook},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	if len(runtime.shutdownHooks) != 1 {
		t.Fatalf("shutdown hooks=%d", len(runtime.shutdownHooks))
	}
	runtime.runShutdownHooksV0(context.Background())
	if got := atomic.LoadInt32(&hook.calls); got != 1 {
		t.Fatalf("hook calls=%d", got)
	}
}

type countingRuntimeShutdownHookV0 struct {
	calls int32
}

func (hook *countingRuntimeShutdownHookV0) ShutdownV0(context.Context) error {
	atomic.AddInt32(&hook.calls, 1)
	return nil
}
