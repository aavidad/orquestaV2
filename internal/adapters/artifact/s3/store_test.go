package s3

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math"
	"strings"
	"sync"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestStorePutGetCASIsIdempotentAcrossStoreReinstantiation(t *testing.T) {
	client := newMemoryClient()
	store := newTestStore(t, client, "project:alpha")
	request := ports.PutArtifactRequest{MediaType: "text/plain", Content: []byte("S3 artifact over shared in-memory backend")}

	first, err := store.Put(context.Background(), request)
	testNoError(t, err)
	second, err := store.Put(context.Background(), request)
	testNoError(t, err)
	if first != second || client.objectCount() != 1 {
		t.Fatalf("idempotent Put: first=%+v second=%+v objects=%d", first, second, client.objectCount())
	}

	reinstantiated := newTestStore(t, client, "project:alpha")
	got, err := reinstantiated.Get(context.Background(), first.Ref, first.Size)
	testNoError(t, err)
	if string(got.Content) != string(request.Content) || got.Digest != first.Digest || got.Size != first.Size {
		t.Fatalf("Get(reinstantiated store) = %+v", got)
	}
}

func TestStoreConcurrentPutUsesOneConditionalObject(t *testing.T) {
	client := newMemoryClient()
	store := newTestStore(t, client, "project:concurrent")
	request := ports.PutArtifactRequest{MediaType: "application/octet-stream", Content: []byte("one immutable object")}
	const workers = 32
	results := make([]ports.StoredArtifact, workers)
	errs := make([]error, workers)
	var group sync.WaitGroup
	for index := range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			results[index], errs[index] = store.Put(context.Background(), request)
		}()
	}
	group.Wait()
	for index := range workers {
		if errs[index] != nil || results[index] != results[0] {
			t.Fatalf("Put[%d] result=%+v error=%v", index, results[index], errs[index])
		}
	}
	if client.objectCount() != 1 || client.createdCount() != 1 {
		t.Fatalf("objects=%d creations=%d", client.objectCount(), client.createdCount())
	}
}

func TestStoreIsolatesProjectNamespaceWithoutLeakingRef(t *testing.T) {
	client := newMemoryClient()
	alpha := newTestStore(t, client, "project:alpha/private")
	beta := newTestStore(t, client, "project:beta/private")
	request := ports.PutArtifactRequest{MediaType: "text/plain", Content: []byte("shared bytes")}
	stored, err := alpha.Put(context.Background(), request)
	testNoError(t, err)
	if _, err := beta.Get(context.Background(), stored.Ref, stored.Size); artifactCode(err) != ports.ArtifactErrorNotFound {
		t.Fatalf("foreign project Get error=%v", err)
	}
	betaStored, err := beta.Put(context.Background(), request)
	testNoError(t, err)
	if betaStored.Ref != stored.Ref || client.objectCount() != 2 {
		t.Fatalf("isolated CAS refs alpha=%+v beta=%+v objects=%d", stored, betaStored, client.objectCount())
	}
	for _, key := range client.keys() {
		if strings.Contains(key, "project:alpha") || strings.Contains(key, "project:beta") {
			t.Fatalf("project ref leaked in object key %q", key)
		}
	}
}

func TestStoreReconcilesUnknownPutAndRejectsTampering(t *testing.T) {
	client := newMemoryClient()
	client.ambiguousAfterWrite = 1
	client.headErrors = []error{ErrObjectNotFound, errors.New("transient head failure")}
	store := newTestStore(t, client, "project:reconcile")
	request := ports.PutArtifactRequest{MediaType: "text/plain", Content: []byte("write then timeout")}
	stored, err := store.Put(context.Background(), request)
	testNoError(t, err)
	if client.objectCount() != 1 || client.headCount() < 3 {
		t.Fatalf("ambiguous Put objects=%d head attempts=%d", client.objectCount(), client.headCount())
	}

	client.tamperContent(store.locator(stored.Digest), []byte("tampered content!!"))
	if _, err := store.Get(context.Background(), stored.Ref, stored.Size); artifactCode(err) != ports.ArtifactErrorDigestMismatch {
		t.Fatalf("tampered Get error=%v", err)
	}

	missing := newTestStore(t, newMemoryClient(), "project:missing")
	missing.client.(*memoryClient).failBeforeWrite = errors.New("backend unavailable")
	if _, err := missing.Put(context.Background(), request); artifactCode(err) != ports.ArtifactErrorIO {
		t.Fatalf("failed Put error=%v", err)
	}
	if missing.client.(*memoryClient).headCount() < 2 {
		t.Fatalf("failed Put reconciliation attempts=%d", missing.client.(*memoryClient).headCount())
	}
}

func TestStoreReconciliationDetachesCallerCancellationWithinBound(t *testing.T) {
	client := newMemoryClient()
	client.ambiguousAfterWrite = 1
	client.headErrors = []error{ErrObjectNotFound}
	ctx, cancel := context.WithCancel(context.Background())
	client.afterWrite = cancel
	store := newTestStore(t, client, "project:detached-reconciliation")
	request := ports.PutArtifactRequest{MediaType: "text/plain", Content: []byte("detached bounded reconciliation")}

	stored, err := store.Put(ctx, request)
	testNoError(t, err)
	if err := ports.ValidateStoredArtifact(request, stored); err != nil || client.headCount() < 2 {
		t.Fatalf("detached reconciliation stored=%+v validation=%v head attempts=%d", stored, err, client.headCount())
	}
}

func TestStoreClosesStreamReturnedWithErrorAndRejectsHeadOpenDivergence(t *testing.T) {
	client := newMemoryClient()
	store := newTestStore(t, client, "project:stream-error")
	request := ports.PutArtifactRequest{MediaType: "text/plain", Content: []byte("stream lifecycle")}
	stored, err := store.Put(context.Background(), request)
	testNoError(t, err)

	closesBeforeFailure := client.closeCount()
	client.setOpenFailure(errors.New("transport failed after stream allocation"), true)
	if _, err := store.Get(context.Background(), stored.Ref, stored.Size); artifactCode(err) != ports.ArtifactErrorIO {
		t.Fatalf("stream allocation error=%v", err)
	}
	if client.closeCount() != closesBeforeFailure+1 {
		t.Fatalf("stream closes=%d want=%d", client.closeCount(), closesBeforeFailure+1)
	}

	client.setOpenFailure(ErrObjectNotFound, false)
	if _, err := store.Get(context.Background(), stored.Ref, stored.Size); artifactCode(err) != ports.ArtifactErrorIO {
		t.Fatalf("Head/Open divergence error=%v", err)
	}
}

func TestStoreRejectsIndividuallyCorruptVerifiableMetadata(t *testing.T) {
	for name, mutate := range map[string]func(*ObjectMetadata){
		"schema":          func(metadata *ObjectMetadata) { metadata.Schema = "other.schema" },
		"project digest":  func(metadata *ObjectMetadata) { metadata.ProjectDigest = strings.Repeat("0", 64) },
		"artifact digest": func(metadata *ObjectMetadata) { metadata.ArtifactDigest = strings.Repeat("f", 64) },
		"metadata size":   func(metadata *ObjectMetadata) { metadata.Size++ },
		"invalid media":   func(metadata *ObjectMetadata) { metadata.OriginalMediaType = "not-a-media-type" },
	} {
		t.Run(name, func(t *testing.T) {
			client := newMemoryClient()
			store := newTestStore(t, client, "project:metadata-"+strings.ReplaceAll(name, " ", "-"))
			stored, err := store.Put(context.Background(), ports.PutArtifactRequest{
				MediaType: "text/plain", Content: []byte("individually verified metadata"),
			})
			testNoError(t, err)
			client.mutateMetadata(store.locator(stored.Digest), mutate)
			_, err = store.Get(context.Background(), stored.Ref, stored.Size)
			want := ports.ArtifactErrorDigestMismatch
			if name == "metadata size" {
				want = ports.ArtifactErrorSizeMismatch
			}
			if artifactCode(err) != want {
				t.Fatalf("metadata error=%v want code=%s", err, want)
			}
		})
	}
}

func TestStoreRejectsReplayWithDifferentMetadataAndNilStream(t *testing.T) {
	client := newMemoryClient()
	store := newTestStore(t, client, "project:metadata")
	content := []byte("same content, different causal metadata")
	stored, err := store.Put(context.Background(), ports.PutArtifactRequest{
		MediaType: "text/plain", Content: content,
	})
	testNoError(t, err)
	if replay, err := store.Put(context.Background(), ports.PutArtifactRequest{
		MediaType: "application/octet-stream", Content: content,
	}); artifactCode(err) != ports.ArtifactErrorFileChanged || replay.Ref.String() != "" {
		t.Fatalf("metadata substitution replay=%+v error=%v", replay, err)
	}
	if client.objectCount() != 1 {
		t.Fatalf("metadata replay created %d objects", client.objectCount())
	}

	client.returnNilOpen = true
	if _, err := store.Get(context.Background(), stored.Ref, stored.Size); artifactCode(err) != ports.ArtifactErrorIO {
		t.Fatalf("nil stream Get error=%v", err)
	}
}

func TestStoreBoundsStreamsAndValidatesRequests(t *testing.T) {
	client := newMemoryClient()
	store := newTestStore(t, client, "project:bounded")
	request := ports.PutArtifactRequest{MediaType: "text/plain", Content: []byte("12345")}
	stored, err := store.Put(context.Background(), request)
	testNoError(t, err)
	client.tamperContent(store.locator(stored.Digest), bytes.Repeat([]byte("x"), 1024))
	client.overrideHeadSize(stored.Size)
	if _, err := store.Get(context.Background(), stored.Ref, stored.Size); artifactCode(err) != ports.ArtifactErrorSizeMismatch {
		t.Fatalf("oversized stream error=%v", err)
	}
	if got := client.lastOpenReadCount(); got != stored.Size+1 {
		t.Fatalf("read bytes=%d want=%d", got, stored.Size+1)
	}

	if _, err := store.Put(context.Background(), ports.PutArtifactRequest{
		MediaType: "text/plain", Content: bytes.Repeat([]byte("x"), 65),
	}); artifactCode(err) != ports.ArtifactErrorSizeMismatch {
		t.Fatalf("oversized Put error=%v", err)
	}
	if _, err := store.Get(context.Background(), stored.Ref, -1); artifactCode(err) != ports.ArtifactErrorExpectedSizeInvalid {
		t.Fatalf("negative size Get error=%v", err)
	}
	badRef, _ := goal.NewArtifactRef("artifact:sha256:not-a-digest")
	if _, err := store.Get(context.Background(), badRef, 0); artifactCode(err) != ports.ArtifactErrorRefInvalid {
		t.Fatalf("malformed ref Get error=%v", err)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := store.Put(canceled, request); artifactCode(err) != ports.ArtifactErrorIO {
		t.Fatalf("canceled Put error=%v", err)
	}
}

func TestStoreRequiresInjectedTypedOptions(t *testing.T) {
	project, _ := goal.NewProjectRef("project:valid")
	valid := Options{Bucket: "artifacts", ProjectRef: project, MaxObjectBytes: 64, ReconcileTimeout: time.Second}
	for name, input := range map[string]struct {
		client  Client
		options Options
	}{
		"nil client":    {nil, valid},
		"empty bucket":  {newMemoryClient(), withBucket(valid, "")},
		"spaced bucket": {newMemoryClient(), withBucket(valid, " artifacts")},
		"empty project": {newMemoryClient(), withProject(valid, goal.ProjectRef{})},
		"zero limit":    {newMemoryClient(), withLimit(valid, 0)},
		"unsafe limit":  {newMemoryClient(), withLimit(valid, math.MaxInt64)},
		"zero timeout":  {newMemoryClient(), withTimeout(valid, 0)},
	} {
		t.Run(name, func(t *testing.T) {
			if store, err := New(input.client, input.options); store != nil || artifactCode(err) != ports.ArtifactErrorStoreUnavailable {
				t.Fatalf("New() store=%v error=%v", store, err)
			}
		})
	}
}

func newTestStore(t *testing.T, client Client, projectValue string) *Store {
	t.Helper()
	project, err := goal.NewProjectRef(projectValue)
	testNoError(t, err)
	store, err := New(client, Options{
		Bucket: "artifacts", ProjectRef: project, MaxObjectBytes: 64, ReconcileTimeout: 50 * time.Millisecond,
	})
	testNoError(t, err)
	return store
}

type memoryObject struct {
	content  []byte
	metadata ObjectMetadata
}

type memoryClient struct {
	mu                  sync.Mutex
	objects             map[string]memoryObject
	creations           int
	ambiguousAfterWrite int
	failBeforeWrite     error
	afterWrite          func()
	returnNilOpen       bool
	headErrors          []error
	headCalls           int
	openError           error
	openStreamWithError bool
	streamCloses        int
	headSizeOverride    *int64
	headSizeByKey       map[string]int64
	lastRead            int64
}

func newMemoryClient() *memoryClient {
	return &memoryClient{objects: make(map[string]memoryObject), headSizeByKey: make(map[string]int64)}
}

func (client *memoryClient) PutIfAbsent(ctx context.Context, request PutObjectRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	content, err := io.ReadAll(io.LimitReader(request.Body, request.Size+1))
	if err != nil {
		return err
	}
	if int64(len(content)) != request.Size {
		return errors.New("request size mismatch")
	}
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.failBeforeWrite != nil {
		return client.failBeforeWrite
	}
	key := memoryKey(request.ObjectLocator)
	if _, exists := client.objects[key]; exists {
		return ErrObjectAlreadyExists
	}
	client.objects[key] = memoryObject{content: append([]byte(nil), content...), metadata: request.Metadata}
	client.creations++
	if client.afterWrite != nil {
		client.afterWrite()
	}
	if client.ambiguousAfterWrite > 0 {
		client.ambiguousAfterWrite--
		return errors.New("ambiguous timeout after write")
	}
	return nil
}

func (client *memoryClient) Head(ctx context.Context, locator ObjectLocator) (ObjectInfo, error) {
	if err := ctx.Err(); err != nil {
		return ObjectInfo{}, err
	}
	client.mu.Lock()
	defer client.mu.Unlock()
	client.headCalls++
	if len(client.headErrors) > 0 {
		err := client.headErrors[0]
		client.headErrors = client.headErrors[1:]
		return ObjectInfo{}, err
	}
	object, found := client.objects[memoryKey(locator)]
	if !found {
		return ObjectInfo{}, ErrObjectNotFound
	}
	size := int64(len(object.content))
	if override, found := client.headSizeByKey[memoryKey(locator)]; found {
		size = override
	} else if client.headSizeOverride != nil {
		size = *client.headSizeOverride
	}
	return ObjectInfo{Size: size, Metadata: object.metadata}, nil
}

func (client *memoryClient) Open(ctx context.Context, locator ObjectLocator) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	client.mu.Lock()
	object, found := client.objects[memoryKey(locator)]
	client.lastRead = 0
	openErr := client.openError
	openStreamWithError := client.openStreamWithError
	client.mu.Unlock()
	if !found {
		return nil, ErrObjectNotFound
	}
	if client.returnNilOpen {
		return nil, nil
	}
	stream := &countingReadCloser{
		reader: bytes.NewReader(append([]byte(nil), object.content...)),
		onRead: func(count int) {
			client.mu.Lock()
			client.lastRead += int64(count)
			client.mu.Unlock()
		},
		onClose: func() {
			client.mu.Lock()
			client.streamCloses++
			client.mu.Unlock()
		},
	}
	if openErr != nil {
		if openStreamWithError {
			return stream, openErr
		}
		return nil, openErr
	}
	return stream, nil
}

func (client *memoryClient) tamperContent(locator ObjectLocator, content []byte) {
	client.mu.Lock()
	defer client.mu.Unlock()
	object := client.objects[memoryKey(locator)]
	object.content = append([]byte(nil), content...)
	client.objects[memoryKey(locator)] = object
}

func (client *memoryClient) mutateMetadata(locator ObjectLocator, mutate func(*ObjectMetadata)) {
	client.mu.Lock()
	defer client.mu.Unlock()
	object := client.objects[memoryKey(locator)]
	mutate(&object.metadata)
	client.objects[memoryKey(locator)] = object
}

func (client *memoryClient) setOpenFailure(err error, withStream bool) {
	client.mu.Lock()
	defer client.mu.Unlock()
	client.openError = err
	client.openStreamWithError = withStream
}

func (client *memoryClient) overrideHeadSize(size int64) {
	client.mu.Lock()
	defer client.mu.Unlock()
	client.headSizeOverride = &size
}

func (client *memoryClient) objectCount() int {
	client.mu.Lock()
	defer client.mu.Unlock()
	return len(client.objects)
}

func (client *memoryClient) createdCount() int {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.creations
}

func (client *memoryClient) headCount() int {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.headCalls
}

func (client *memoryClient) closeCount() int {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.streamCloses
}

func (client *memoryClient) keys() []string {
	client.mu.Lock()
	defer client.mu.Unlock()
	keys := make([]string, 0, len(client.objects))
	for key := range client.objects {
		keys = append(keys, key)
	}
	return keys
}

func (client *memoryClient) lastOpenReadCount() int64 {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.lastRead
}

type countingReadCloser struct {
	reader  *bytes.Reader
	onRead  func(int)
	onClose func()
}

func (reader *countingReadCloser) Read(buffer []byte) (int, error) {
	count, err := reader.reader.Read(buffer)
	reader.onRead(count)
	return count, err
}

func (reader *countingReadCloser) Close() error {
	if reader.onClose != nil {
		reader.onClose()
	}
	return nil
}

func memoryKey(locator ObjectLocator) string { return locator.Bucket + "\x00" + locator.Key }

func artifactCode(err error) string { return ports.ArtifactContractErrorCode(err) }

func testNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func withBucket(options Options, bucket string) Options {
	options.Bucket = bucket
	return options
}

func withProject(options Options, project goal.ProjectRef) Options {
	options.ProjectRef = project
	return options
}

func withLimit(options Options, limit int64) Options {
	options.MaxObjectBytes = limit
	return options
}

func withTimeout(options Options, timeout time.Duration) Options {
	options.ReconcileTimeout = timeout
	return options
}
