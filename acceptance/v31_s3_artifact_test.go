package acceptance_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	artifactS3 "orquesta/internal/adapters/artifact/s3"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

const v31S3ArtifactFixturePath = "acceptance/fixtures/v31_s3_artifact.json"

type v31S3ArtifactFixture struct {
	SchemaVersion        int      `json:"schema_version"`
	FixtureID            string   `json:"fixture_id"`
	ContractID           string   `json:"contract_id"`
	CapabilityIDs        []string `json:"capability_ids"`
	Status               string   `json:"status"`
	AdapterPath          string   `json:"adapter_path"`
	ClientContract       string   `json:"client_contract"`
	ImplementedBehaviors []string `json:"implemented_behaviors"`
	DeferredGates        []string `json:"deferred_gates"`
}

func TestAcceptanceV31S3ArtifactFoundation(t *testing.T) {
	fixture := readV31S3ArtifactFixture(t)
	assertV31S3ArtifactFixture(t, fixture)
	behaviorTests := map[string]func(*testing.T){
		"content_addressed_put_over_create_only_client_contract":               runV31S3ContentAddressedPut,
		"schema_project_digest_artifact_digest_size_and_media_syntax_verified": runV31S3MetadataValidation,
		"media_type_conflict_rejected_on_put_replay":                           runV31S3MediaTypeConflict,
		"raw_project_ref_not_embedded_in_object_key":                           runV31S3OpaqueProjectIsolation,
		"bounded_in_memory_stream_read":                                        runV31S3BoundedStreamRead,
		"bounded_retry_reconciliation_on_same_key":                             runV31S3BoundedRetryReconciliation,
		"concurrent_idempotent_put_via_injected_fake":                          runV31S3ConcurrentPut,
		"stateless_store_reinstantiation_over_same_in_memory_backend":          runV31S3StoreReinstantiation,
	}
	if len(behaviorTests) != len(fixture.ImplementedBehaviors) {
		t.Fatalf("V31 S3 executable behavior count=%d fixture count=%d", len(behaviorTests), len(fixture.ImplementedBehaviors))
	}
	for _, behavior := range fixture.ImplementedBehaviors {
		behaviorTest, found := behaviorTests[behavior]
		if !found {
			t.Fatalf("V31 S3 behavior %q has no executable acceptance", behavior)
		}
		t.Run(behavior, behaviorTest)
	}
}

func runV31S3ContentAddressedPut(t *testing.T) {
	client := newV31S3Client()
	store := newV31S3Store(t, client, "project:v31-content-addressed")
	request := ports.PutArtifactRequest{MediaType: "application/json", Content: []byte(`{"gate":"v31"}`)}
	first, err := store.Put(context.Background(), request)
	v31NoError(t, err)
	if err := ports.ValidateStoredArtifact(request, first); err != nil {
		t.Fatalf("stored artifact violates content address: %v", err)
	}
	second, err := store.Put(context.Background(), request)
	v31NoError(t, err)
	if first != second || client.count() != 1 {
		t.Fatalf("conditional CAS not idempotent: first=%+v second=%+v objects=%d", first, second, client.count())
	}
	content, err := store.Get(context.Background(), first.Ref, first.Size)
	v31NoError(t, err)
	if err := ports.ValidateArtifactContent(content); err != nil || !bytes.Equal(content.Content, request.Content) ||
		content.MediaType != request.MediaType {
		t.Fatalf("content=%+v validation=%v", content, err)
	}
}

func runV31S3MetadataValidation(t *testing.T) {
	type metadataCase struct {
		name   string
		mutate func(*artifactS3.ObjectMetadata)
		want   string
	}
	cases := []metadataCase{
		{name: "schema", mutate: func(metadata *artifactS3.ObjectMetadata) { metadata.Schema = "other.schema" }, want: ports.ArtifactErrorDigestMismatch},
		{name: "project_digest", mutate: func(metadata *artifactS3.ObjectMetadata) { metadata.ProjectDigest = strings.Repeat("0", 64) }, want: ports.ArtifactErrorDigestMismatch},
		{name: "artifact_digest", mutate: func(metadata *artifactS3.ObjectMetadata) { metadata.ArtifactDigest = strings.Repeat("f", 64) }, want: ports.ArtifactErrorDigestMismatch},
		{name: "metadata_size", mutate: func(metadata *artifactS3.ObjectMetadata) { metadata.Size++ }, want: ports.ArtifactErrorSizeMismatch},
		{name: "media_syntax", mutate: func(metadata *artifactS3.ObjectMetadata) { metadata.OriginalMediaType = "not-a-media-type" }, want: ports.ArtifactErrorDigestMismatch},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			client := newV31S3Client()
			store := newV31S3Store(t, client, "project:v31-metadata-"+testCase.name)
			request := ports.PutArtifactRequest{MediaType: "text/plain", Content: []byte("causal metadata " + testCase.name)}
			stored, err := store.Put(context.Background(), request)
			v31NoError(t, err)
			if !client.tamperMetadata(stored.Digest, testCase.mutate) {
				t.Fatal("artifact metadata not found for tamper")
			}
			if _, err := store.Get(context.Background(), stored.Ref, stored.Size); ports.ArtifactContractErrorCode(err) != testCase.want {
				t.Fatalf("metadata %s error=%v want code=%s", testCase.name, err, testCase.want)
			}
		})
	}
	t.Run("request_media_syntax", func(t *testing.T) {
		client := newV31S3Client()
		store := newV31S3Store(t, client, "project:v31-request-media")
		stored, err := store.Put(context.Background(), ports.PutArtifactRequest{
			MediaType: "not-a-media-type", Content: []byte("invalid media request"),
		})
		if ports.ArtifactContractErrorCode(err) != ports.ArtifactErrorMediaTypeInvalid || stored.Ref.String() != "" || client.count() != 0 {
			t.Fatalf("invalid media request stored=%+v error=%v objects=%d", stored, err, client.count())
		}
	})
}

func runV31S3MediaTypeConflict(t *testing.T) {
	client := newV31S3Client()
	store := newV31S3Store(t, client, "project:v31-media-conflict")
	content := []byte("same bytes with conflicting media")
	request := ports.PutArtifactRequest{MediaType: "text/plain", Content: content}
	stored, err := store.Put(context.Background(), request)
	v31NoError(t, err)
	before := client.count()
	conflict, err := store.Put(context.Background(), ports.PutArtifactRequest{
		MediaType: "application/octet-stream", Content: content,
	})
	if ports.ArtifactContractErrorCode(err) != ports.ArtifactErrorFileChanged || conflict.Ref.String() != "" || client.count() != before {
		t.Fatalf("media conflict stored=%+v error=%v objects=%d want=%d", conflict, err, client.count(), before)
	}
	got, err := store.Get(context.Background(), stored.Ref, stored.Size)
	v31NoError(t, err)
	if got.MediaType != request.MediaType || !bytes.Equal(got.Content, request.Content) {
		t.Fatalf("media conflict changed original content: %+v", got)
	}
}

func runV31S3OpaqueProjectIsolation(t *testing.T) {
	client := newV31S3Client()
	alphaValue, betaValue := "project:v31-alpha/private", "project:v31-beta/private"
	alpha := newV31S3Store(t, client, alphaValue)
	beta := newV31S3Store(t, client, betaValue)
	request := ports.PutArtifactRequest{MediaType: "text/plain", Content: []byte("same isolated bytes")}
	alphaStored, err := alpha.Put(context.Background(), request)
	v31NoError(t, err)
	if _, err := beta.Get(context.Background(), alphaStored.Ref, alphaStored.Size); ports.ArtifactContractErrorCode(err) != ports.ArtifactErrorNotFound {
		t.Fatalf("cross-project read error=%v", err)
	}
	betaStored, err := beta.Put(context.Background(), request)
	v31NoError(t, err)
	if betaStored.Ref != alphaStored.Ref || client.count() != 2 {
		t.Fatalf("project isolation alpha=%+v beta=%+v objects=%d", alphaStored, betaStored, client.count())
	}
	for _, locator := range client.locators() {
		if strings.Contains(locator.Key, alphaValue) || strings.Contains(locator.Key, betaValue) {
			t.Fatalf("project ref leaked in object key %q", locator.Key)
		}
	}
}

func runV31S3BoundedStreamRead(t *testing.T) {
	client := newV31S3Client()
	store := newV31S3Store(t, client, "project:v31-bounded-stream")
	request := ports.PutArtifactRequest{MediaType: "text/plain", Content: []byte("bounded")}
	stored, err := store.Put(context.Background(), request)
	v31NoError(t, err)
	client.tamperContent(stored.Digest, bytes.Repeat([]byte("x"), 4096), stored.Size)
	if _, err := store.Get(context.Background(), stored.Ref, stored.Size); ports.ArtifactContractErrorCode(err) != ports.ArtifactErrorSizeMismatch {
		t.Fatalf("oversized object stream error=%v", err)
	}
	if got := client.lastOpenReadCount(); got != stored.Size+1 {
		t.Fatalf("bounded stream read=%d want=%d", got, stored.Size+1)
	}
}

func runV31S3BoundedRetryReconciliation(t *testing.T) {
	client := newV31S3Client()
	client.injectAmbiguousWrite(2)
	store := newV31S3Store(t, client, "project:v31-reconciliation")
	request := ports.PutArtifactRequest{MediaType: "text/plain", Content: []byte("effect before timeout")}
	stored, err := store.Put(context.Background(), request)
	v31NoError(t, err)
	if err := ports.ValidateStoredArtifact(request, stored); err != nil {
		t.Fatalf("reconciled artifact validation=%v", err)
	}
	assertV31SameHeadLocator(t, client.headObservations(), 3)

	boundedClient := newV31S3Client()
	boundedClient.injectAmbiguousWrite(1_000_000)
	boundedStore := newV31S3StoreWithTimeout(t, boundedClient, "project:v31-reconciliation-bound", 10*time.Millisecond)
	started := time.Now()
	failed, err := boundedStore.Put(context.Background(), ports.PutArtifactRequest{
		MediaType: "text/plain", Content: []byte("never observable within bound"),
	})
	if ports.ArtifactContractErrorCode(err) != ports.ArtifactErrorIO || failed.Ref.String() != "" {
		t.Fatalf("bounded reconciliation stored=%+v error=%v", failed, err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("bounded reconciliation elapsed=%s", elapsed)
	}
	assertV31SameHeadLocator(t, boundedClient.headObservations(), 2)
}

func runV31S3ConcurrentPut(t *testing.T) {
	client := newV31S3Client()
	store := newV31S3Store(t, client, "project:v31-concurrent")
	request := ports.PutArtifactRequest{MediaType: "text/plain", Content: []byte("concurrent conditional object")}
	const workers = 24
	results := make([]ports.StoredArtifact, workers)
	errorsFound := make([]error, workers)
	start := make(chan struct{})
	var ready, done sync.WaitGroup
	for index := range workers {
		ready.Add(1)
		done.Add(1)
		go func() {
			defer done.Done()
			ready.Done()
			<-start
			results[index], errorsFound[index] = store.Put(context.Background(), request)
		}()
	}
	ready.Wait()
	before := client.count()
	close(start)
	done.Wait()
	for index := range workers {
		if errorsFound[index] != nil || results[index] != results[0] {
			t.Fatalf("concurrent Put[%d] result=%+v error=%v", index, results[index], errorsFound[index])
		}
	}
	if got := client.count(); got != before+1 {
		t.Fatalf("concurrent CAS object count=%d want=%d", got, before+1)
	}
}

func runV31S3StoreReinstantiation(t *testing.T) {
	client := newV31S3Client()
	store := newV31S3Store(t, client, "project:v31-reinstantiation")
	request := ports.PutArtifactRequest{MediaType: "text/plain", Content: []byte("same in-memory backend")}
	stored, err := store.Put(context.Background(), request)
	v31NoError(t, err)
	reinstantiated := newV31S3Store(t, client, "project:v31-reinstantiation")
	content, err := reinstantiated.Get(context.Background(), stored.Ref, stored.Size)
	v31NoError(t, err)
	if !bytes.Equal(content.Content, request.Content) || content.Digest != stored.Digest {
		t.Fatalf("store reinstantiation read=%+v", content)
	}
}

func assertV31SameHeadLocator(t *testing.T, observations []artifactS3.ObjectLocator, minimum int) {
	t.Helper()
	if len(observations) < minimum {
		t.Fatalf("Head observations=%d want at least %d", len(observations), minimum)
	}
	for _, observation := range observations[1:] {
		if observation != observations[0] {
			t.Fatalf("reconciliation changed object key: first=%+v observed=%+v", observations[0], observation)
		}
	}
}

func TestV31S3ArtifactFoundationDoesNotClaimOPS13Accreditation(t *testing.T) {
	fixture := readV31S3ArtifactFixture(t)
	if fixture.Status != "implemented_foundation_not_accredited" ||
		!reflect.DeepEqual(fixture.DeferredGates, []string{
			"real_s3_compatible_backend",
			"real_process_client_backend_restart_recovery",
			"real_s3_compatible_backend_receipt",
			"production_wiring_and_credentials",
			"postgres_s3_multihost_e2e",
			"sealed_candidate_receipt",
		}) {
		t.Fatalf("V31 S3 foundation overclaims closure: %+v", fixture)
	}
}

func readV31S3ArtifactFixture(t *testing.T) v31S3ArtifactFixture {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(evidenceRepositoryRoot(t), v31S3ArtifactFixturePath))
	v31NoError(t, err)
	var fixture v31S3ArtifactFixture
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	v31NoError(t, decoder.Decode(&fixture))
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		t.Fatalf("trailing fixture data: %v", err)
	}
	canonical, err := json.MarshalIndent(fixture, "", "  ")
	v31NoError(t, err)
	if !bytes.Equal(data, append(canonical, '\n')) {
		t.Fatal("V31 S3 fixture is not canonical strict JSON")
	}
	return fixture
}

func assertV31S3ArtifactFixture(t *testing.T, fixture v31S3ArtifactFixture) {
	t.Helper()
	if fixture.SchemaVersion != 1 || fixture.FixtureID != "v31_s3_artifact_foundation" ||
		fixture.ContractID != "AC-V31-POSTGRES-S3-MULTIHOST" ||
		!reflect.DeepEqual(fixture.CapabilityIDs, []string{"OPS-13"}) ||
		fixture.Status != "implemented_foundation_not_accredited" ||
		fixture.AdapterPath != "internal/adapters/artifact/s3" ||
		fixture.ClientContract != "sdk_neutral_injected_client" {
		t.Fatalf("invalid V31 S3 fixture identity: %+v", fixture)
	}
	wantBehaviors := []string{
		"content_addressed_put_over_create_only_client_contract",
		"schema_project_digest_artifact_digest_size_and_media_syntax_verified",
		"media_type_conflict_rejected_on_put_replay",
		"raw_project_ref_not_embedded_in_object_key",
		"bounded_in_memory_stream_read",
		"bounded_retry_reconciliation_on_same_key",
		"concurrent_idempotent_put_via_injected_fake",
		"stateless_store_reinstantiation_over_same_in_memory_backend",
	}
	if !reflect.DeepEqual(fixture.ImplementedBehaviors, wantBehaviors) {
		t.Fatalf("V31 S3 behavior contract drift: %v", fixture.ImplementedBehaviors)
	}
}

func newV31S3Store(t *testing.T, client artifactS3.Client, projectValue string) *artifactS3.Store {
	return newV31S3StoreWithTimeout(t, client, projectValue, time.Second)
}

func newV31S3StoreWithTimeout(
	t *testing.T,
	client artifactS3.Client,
	projectValue string,
	reconcileTimeout time.Duration,
) *artifactS3.Store {
	t.Helper()
	project, err := goal.NewProjectRef(projectValue)
	v31NoError(t, err)
	store, err := artifactS3.New(client, artifactS3.Options{
		Bucket: "v31-artifacts", ProjectRef: project, MaxObjectBytes: 1024,
		ReconcileTimeout: reconcileTimeout,
	})
	v31NoError(t, err)
	return store
}

type v31S3Object struct {
	content  []byte
	metadata artifactS3.ObjectMetadata
}

type v31S3Client struct {
	mu                    sync.Mutex
	objects               map[artifactS3.ObjectLocator]v31S3Object
	headSizeOverride      map[artifactS3.ObjectLocator]int64
	failAfterNextWrite    bool
	headNotFoundRemaining int
	headLocators          []artifactS3.ObjectLocator
	lastRead              int64
}

func newV31S3Client() *v31S3Client {
	return &v31S3Client{
		objects:          make(map[artifactS3.ObjectLocator]v31S3Object),
		headSizeOverride: make(map[artifactS3.ObjectLocator]int64),
	}
}

func (client *v31S3Client) PutIfAbsent(ctx context.Context, request artifactS3.PutObjectRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	content, err := io.ReadAll(io.LimitReader(request.Body, request.Size+1))
	if err != nil || int64(len(content)) != request.Size {
		return errors.New("invalid conditional body")
	}
	client.mu.Lock()
	defer client.mu.Unlock()
	if _, exists := client.objects[request.ObjectLocator]; exists {
		return artifactS3.ErrObjectAlreadyExists
	}
	client.objects[request.ObjectLocator] = v31S3Object{
		content: append([]byte(nil), content...), metadata: request.Metadata,
	}
	if client.failAfterNextWrite {
		client.failAfterNextWrite = false
		return errors.New("injected timeout after write")
	}
	return nil
}

func (client *v31S3Client) Head(
	ctx context.Context,
	locator artifactS3.ObjectLocator,
) (artifactS3.ObjectInfo, error) {
	if err := ctx.Err(); err != nil {
		return artifactS3.ObjectInfo{}, err
	}
	client.mu.Lock()
	defer client.mu.Unlock()
	client.headLocators = append(client.headLocators, locator)
	if client.headNotFoundRemaining > 0 {
		client.headNotFoundRemaining--
		return artifactS3.ObjectInfo{}, artifactS3.ErrObjectNotFound
	}
	object, found := client.objects[locator]
	if !found {
		return artifactS3.ObjectInfo{}, artifactS3.ErrObjectNotFound
	}
	size := int64(len(object.content))
	if override, found := client.headSizeOverride[locator]; found {
		size = override
	}
	return artifactS3.ObjectInfo{Size: size, Metadata: object.metadata}, nil
}

func (client *v31S3Client) Open(ctx context.Context, locator artifactS3.ObjectLocator) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	client.mu.Lock()
	object, found := client.objects[locator]
	client.lastRead = 0
	client.mu.Unlock()
	if !found {
		return nil, artifactS3.ErrObjectNotFound
	}
	return &v31CountingReadCloser{
		reader: bytes.NewReader(append([]byte(nil), object.content...)),
		onRead: func(count int) {
			client.mu.Lock()
			client.lastRead += int64(count)
			client.mu.Unlock()
		},
	}, nil
}

func (client *v31S3Client) count() int {
	client.mu.Lock()
	defer client.mu.Unlock()
	return len(client.objects)
}

func (client *v31S3Client) locators() []artifactS3.ObjectLocator {
	client.mu.Lock()
	defer client.mu.Unlock()
	locators := make([]artifactS3.ObjectLocator, 0, len(client.objects))
	for locator := range client.objects {
		locators = append(locators, locator)
	}
	return locators
}

func (client *v31S3Client) tamperMetadata(
	digest string,
	mutate func(*artifactS3.ObjectMetadata),
) bool {
	client.mu.Lock()
	defer client.mu.Unlock()
	for locator, object := range client.objects {
		if object.metadata.ArtifactDigest == digest {
			mutate(&object.metadata)
			client.objects[locator] = object
			return true
		}
	}
	return false
}

func (client *v31S3Client) tamperContent(digest string, content []byte, reportedSize int64) {
	client.mu.Lock()
	defer client.mu.Unlock()
	for locator, object := range client.objects {
		if object.metadata.ArtifactDigest == digest {
			object.content = append([]byte(nil), content...)
			client.objects[locator] = object
			client.headSizeOverride[locator] = reportedSize
			return
		}
	}
}

func (client *v31S3Client) lastOpenReadCount() int64 {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.lastRead
}

func (client *v31S3Client) injectAmbiguousWrite(delayedHeads int) {
	client.mu.Lock()
	defer client.mu.Unlock()
	client.failAfterNextWrite = true
	client.headNotFoundRemaining = delayedHeads
}

func (client *v31S3Client) headObservations() []artifactS3.ObjectLocator {
	client.mu.Lock()
	defer client.mu.Unlock()
	return append([]artifactS3.ObjectLocator(nil), client.headLocators...)
}

type v31CountingReadCloser struct {
	reader *bytes.Reader
	onRead func(int)
}

func (reader *v31CountingReadCloser) Read(buffer []byte) (int, error) {
	count, err := reader.reader.Read(buffer)
	reader.onRead(count)
	return count, err
}

func (*v31CountingReadCloser) Close() error { return nil }

func v31NoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
