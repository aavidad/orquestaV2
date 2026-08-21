// Package artifactcontract defines one reusable ArtifactStore behavior suite
// for concrete adapters. Backend hooks only expose physical fault injection and
// observation; every semantic assertion runs unchanged for every adapter.
package artifactcontract

import (
	"bytes"
	"context"
	"sync"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

type Handle struct {
	Store application.ArtifactStore
	Close func() error
}

type Backend struct {
	Open                   func(testing.TB, goal.ProjectRef) Handle
	LogicalObjectCount     func(testing.TB) int
	TamperContent          func(testing.TB, Handle, ports.StoredArtifact, []byte)
	TamperMetadata         func(testing.TB, Handle, ports.StoredArtifact)
	TamperOversizedContent func(testing.TB, Handle, ports.StoredArtifact, []byte)
	ResetReadCount         func()
	ReadCount              func() int64
	AssertProjectRefOpaque func(testing.TB, goal.ProjectRef)
}

// ParityBackend contains only the observations needed to execute the common
// adapter behaviors without backend-specific fault injection.
type ParityBackend struct {
	Open                   func(testing.TB, goal.ProjectRef) Handle
	LogicalObjectCount     func(testing.TB) int
	AssertProjectRefOpaque func(testing.TB, goal.ProjectRef)
}

func Run(t *testing.T, backend Backend) {
	t.Helper()
	validateBackend(t, backend)
	parity := parityBackend(backend)
	t.Run("content_addressed_put_and_exact_replay", func(t *testing.T) { runPutAndReplay(t, parity) })
	t.Run("tamper_rejection_and_bounded_reads", func(t *testing.T) { runTamperAndBounds(t, backend) })
	t.Run("project_isolation_and_store_reinstantiation", func(t *testing.T) {
		runProjectIsolationAndStoreReinstantiation(t, parity)
	})
	t.Run("concurrent_replay_and_conflict", func(t *testing.T) { runConcurrentReplayAndConflict(t, parity) })
}

// RunParity executes the common observable behaviors directly against an
// adapter. It deliberately makes no durability or process-restart claim: Open
// may return another handle over the same backend instance.
func RunParity(t *testing.T, backend ParityBackend) {
	t.Helper()
	validateParityBackend(t, backend)
	t.Run("content_addressed_put_and_exact_replay", func(t *testing.T) { runPutAndReplay(t, backend) })
	t.Run("project_isolation_and_store_reinstantiation", func(t *testing.T) {
		runProjectIsolationAndStoreReinstantiation(t, backend)
	})
	t.Run("concurrent_replay_and_conflict", func(t *testing.T) { runConcurrentReplayAndConflict(t, backend) })
}

func runPutAndReplay(t *testing.T, backend ParityBackend) {
	handle := backend.Open(t, projectRef(t, "put-replay"))
	defer closeHandle(t, handle)
	request := ports.PutArtifactRequest{MediaType: "text/plain", Content: []byte("shared contract artifact")}
	before := backend.LogicalObjectCount(t)
	first := put(t, handle.Store, request)
	if err := ports.ValidateStoredArtifact(request, first); err != nil {
		t.Fatalf("stored artifact violates content address: %v", err)
	}
	second := put(t, handle.Store, request)
	if first != second || backend.LogicalObjectCount(t) != before+1 {
		t.Fatalf("exact replay first=%+v second=%+v objects=%d want=%d", first, second, backend.LogicalObjectCount(t), before+1)
	}
	assertContent(t, handle.Store, request, first)
	conflicting := request
	conflicting.MediaType = "application/octet-stream"
	stored, err := handle.Store.Put(context.Background(), conflicting)
	assertCode(t, err, ports.ArtifactErrorFileChanged)
	if stored.Ref.String() != "" || backend.LogicalObjectCount(t) != before+1 {
		t.Fatalf("conflicting replay stored=%+v objects=%d", stored, backend.LogicalObjectCount(t))
	}
	assertContent(t, handle.Store, request, first)
}

func runTamperAndBounds(t *testing.T, backend Backend) {
	handle := backend.Open(t, projectRef(t, "tamper-bounds"))
	defer closeHandle(t, handle)
	request := ports.PutArtifactRequest{MediaType: "application/json", Content: []byte(`{"contract":"tamper"}`)}
	stored := put(t, handle.Store, request)
	if _, err := handle.Store.Get(context.Background(), stored.Ref, -1); ports.ArtifactContractErrorCode(err) != ports.ArtifactErrorExpectedSizeInvalid {
		t.Fatalf("negative expected size error=%v", err)
	}
	backend.ResetReadCount()
	if _, err := handle.Store.Get(context.Background(), stored.Ref, stored.Size-1); ports.ArtifactContractErrorCode(err) != ports.ArtifactErrorSizeMismatch {
		t.Fatalf("short expected size error=%v", err)
	}
	assertReadBound(t, backend, stored.Size)
	assertMetadataTamper(t, backend, handle)
	assertContentTamper(t, backend, handle)
	boundedRequest := ports.PutArtifactRequest{MediaType: "text/plain", Content: []byte("bounded")}
	boundedStored := put(t, handle.Store, boundedRequest)
	backend.TamperOversizedContent(t, handle, boundedStored, bytes.Repeat([]byte("x"), 4096))
	backend.ResetReadCount()
	if _, err := handle.Store.Get(context.Background(), boundedStored.Ref, boundedStored.Size); ports.ArtifactContractErrorCode(err) != ports.ArtifactErrorSizeMismatch {
		t.Fatalf("oversized content error=%v", err)
	}
	assertReadBound(t, backend, boundedStored.Size+1)
}

func assertMetadataTamper(t *testing.T, backend Backend, handle Handle) {
	request := ports.PutArtifactRequest{MediaType: "text/plain", Content: []byte("metadata tamper")}
	stored := put(t, handle.Store, request)
	backend.TamperMetadata(t, handle, stored)
	if _, err := handle.Store.Get(context.Background(), stored.Ref, stored.Size); ports.ArtifactContractErrorCode(err) != ports.ArtifactErrorDigestMismatch {
		t.Fatalf("metadata tamper error=%v", err)
	}
}

func assertContentTamper(t *testing.T, backend Backend, handle Handle) {
	request := ports.PutArtifactRequest{MediaType: "text/plain", Content: []byte("content tamper")}
	stored := put(t, handle.Store, request)
	backend.TamperContent(t, handle, stored, []byte("CONTENT TAMPER"))
	if _, err := handle.Store.Get(context.Background(), stored.Ref, stored.Size); ports.ArtifactContractErrorCode(err) != ports.ArtifactErrorDigestMismatch {
		t.Fatalf("content tamper error=%v", err)
	}
}

func runProjectIsolationAndStoreReinstantiation(t *testing.T, backend ParityBackend) {
	alphaRef, betaRef := projectRef(t, "alpha/private"), projectRef(t, "beta/private")
	alpha, beta := backend.Open(t, alphaRef), backend.Open(t, betaRef)
	defer closeHandle(t, beta)
	request := ports.PutArtifactRequest{MediaType: "text/plain", Content: []byte("same bytes in isolated projects")}
	before := backend.LogicalObjectCount(t)
	alphaStored := put(t, alpha.Store, request)
	if _, err := beta.Store.Get(context.Background(), alphaStored.Ref, alphaStored.Size); ports.ArtifactContractErrorCode(err) != ports.ArtifactErrorNotFound {
		t.Fatalf("foreign project read error=%v", err)
	}
	betaStored := put(t, beta.Store, request)
	if betaStored.Ref != alphaStored.Ref || backend.LogicalObjectCount(t) != before+2 {
		t.Fatalf("project isolation alpha=%+v beta=%+v objects=%d", alphaStored, betaStored, backend.LogicalObjectCount(t))
	}
	backend.AssertProjectRefOpaque(t, alphaRef)
	backend.AssertProjectRefOpaque(t, betaRef)
	closeHandle(t, alpha)
	reinstantiated := backend.Open(t, alphaRef)
	defer closeHandle(t, reinstantiated)
	assertContent(t, reinstantiated.Store, request, alphaStored)
}

func runConcurrentReplayAndConflict(t *testing.T, backend ParityBackend) {
	const workers = 24
	handles := openHandles(t, backend, projectRef(t, "concurrent"), workers)
	request := ports.PutArtifactRequest{MediaType: "text/plain", Content: []byte("one concurrent artifact")}
	before := backend.LogicalObjectCount(t)
	results, errs := concurrentPut(handles, func(int) ports.PutArtifactRequest { return request })
	for index := range results {
		if errs[index] != nil || results[index] != results[0] {
			t.Fatalf("concurrent replay[%d] result=%+v error=%v", index, results[index], errs[index])
		}
	}
	if backend.LogicalObjectCount(t) != before+1 {
		t.Fatalf("concurrent replay objects=%d want=%d", backend.LogicalObjectCount(t), before+1)
	}
	runConcurrentConflict(t, backend, handles)
}

func runConcurrentConflict(t *testing.T, backend ParityBackend, handles []Handle) {
	before := backend.LogicalObjectCount(t)
	results, errs := concurrentPut(handles, func(index int) ports.PutArtifactRequest {
		mediaType := "text/plain"
		if index%2 == 1 {
			mediaType = "application/octet-stream"
		}
		return ports.PutArtifactRequest{MediaType: mediaType, Content: []byte("concurrent metadata conflict")}
	})
	winner := ""
	for index := range results {
		if errs[index] == nil {
			if winner == "" {
				winner = results[index].MediaType
			}
			if results[index].MediaType != winner {
				t.Fatalf("two metadata winners: %q and %q", winner, results[index].MediaType)
			}
			continue
		}
		assertCode(t, errs[index], ports.ArtifactErrorFileChanged)
		if results[index].Ref.String() != "" {
			t.Fatalf("conflict[%d] returned ref %s", index, results[index].Ref.String())
		}
	}
	if winner == "" || backend.LogicalObjectCount(t) != before+1 {
		t.Fatalf("concurrent conflict winner=%q objects=%d want=%d", winner, backend.LogicalObjectCount(t), before+1)
	}
}

func openHandles(t *testing.T, backend ParityBackend, project goal.ProjectRef, count int) []Handle {
	handles := make([]Handle, count)
	for index := range handles {
		handles[index] = backend.Open(t, project)
		handle := handles[index]
		t.Cleanup(func() { closeHandle(t, handle) })
	}
	return handles
}

func validateBackend(t *testing.T, backend Backend) {
	t.Helper()
	validateParityBackend(t, parityBackend(backend))
	if backend.TamperContent == nil || backend.TamperMetadata == nil || backend.TamperOversizedContent == nil ||
		backend.ResetReadCount == nil || backend.ReadCount == nil {
		t.Fatal("artifact contract backend is incomplete")
	}
}

func validateParityBackend(t *testing.T, backend ParityBackend) {
	t.Helper()
	if backend.Open == nil || backend.LogicalObjectCount == nil || backend.AssertProjectRefOpaque == nil {
		t.Fatal("artifact parity backend is incomplete")
	}
}

func parityBackend(backend Backend) ParityBackend {
	return ParityBackend{
		Open: backend.Open, LogicalObjectCount: backend.LogicalObjectCount,
		AssertProjectRefOpaque: backend.AssertProjectRefOpaque,
	}
}

func projectRef(t testing.TB, suffix string) goal.ProjectRef {
	t.Helper()
	ref, err := goal.NewProjectRef("project:artifact-contract/" + suffix)
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func put(t testing.TB, store application.ArtifactStore, request ports.PutArtifactRequest) ports.StoredArtifact {
	t.Helper()
	stored, err := store.Put(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	return stored
}

func assertContent(t testing.TB, store application.ArtifactStore, request ports.PutArtifactRequest, stored ports.StoredArtifact) {
	t.Helper()
	content, err := store.Get(context.Background(), stored.Ref, stored.Size)
	if err != nil {
		t.Fatal(err)
	}
	if err := ports.ValidateArtifactContent(content); err != nil || !bytes.Equal(content.Content, request.Content) ||
		content.MediaType != request.MediaType || content.Digest != stored.Digest || content.Size != stored.Size {
		t.Fatalf("artifact content=%+v validation=%v", content, err)
	}
}

func assertReadBound(t testing.TB, backend Backend, maximum int64) {
	t.Helper()
	if read := backend.ReadCount(); read > maximum {
		t.Fatalf("backend read %d bytes, maximum %d", read, maximum)
	}
}

func assertCode(t testing.TB, err error, want string) {
	t.Helper()
	if got := ports.ArtifactContractErrorCode(err); got != want {
		t.Fatalf("error=%v code=%q want=%q", err, got, want)
	}
}

func closeHandle(t testing.TB, handle Handle) {
	t.Helper()
	if handle.Close != nil {
		if err := handle.Close(); err != nil {
			t.Fatalf("close artifact store: %v", err)
		}
	}
}

func concurrentPut(handles []Handle, request func(int) ports.PutArtifactRequest) ([]ports.StoredArtifact, []error) {
	results := make([]ports.StoredArtifact, len(handles))
	errs := make([]error, len(handles))
	start := make(chan struct{})
	var ready, done sync.WaitGroup
	for index := range handles {
		ready.Add(1)
		done.Add(1)
		go func() {
			defer done.Done()
			ready.Done()
			<-start
			results[index], errs[index] = handles[index].Store.Put(context.Background(), request(index))
		}()
	}
	ready.Wait()
	close(start)
	done.Wait()
	return results, errs
}
