package orquestaserver

import (
	"context"
	"sync"
	"sync/atomic"
)

type runtimeAsyncWorkGroupV0 struct {
	wg     sync.WaitGroup
	active int32
}

func (runtime *RuntimeV0) runAsyncWorkV0(_ string, fn func()) {
	if runtime == nil || fn == nil {
		return
	}
	runtime.asyncWork.wg.Add(1)
	atomic.AddInt32(&runtime.asyncWork.active, 1)
	go func() {
		defer runtime.asyncWork.wg.Done()
		defer atomic.AddInt32(&runtime.asyncWork.active, -1)
		fn()
	}()
}

func (runtime *RuntimeV0) asyncWorkActiveV0() int {
	if runtime == nil {
		return 0
	}
	active := atomic.LoadInt32(&runtime.asyncWork.active)
	if active < 0 {
		return 0
	}
	return int(active)
}

func (runtime *RuntimeV0) waitAsyncWorkV0(ctx context.Context) bool {
	if runtime == nil {
		return true
	}
	done := make(chan struct{})
	go func() {
		runtime.asyncWork.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-ctx.Done():
		return false
	}
}

func (runtime *RuntimeV0) runBackgroundWorkersV0(ctx context.Context) {
	if runtime == nil || len(runtime.backgroundWorkers) == 0 {
		return
	}
	for _, worker := range runtime.backgroundWorkers {
		worker := worker
		runtime.runAsyncWorkV0("background_worker", func() {
			worker.RunBackgroundV0(ctx)
		})
	}
}
