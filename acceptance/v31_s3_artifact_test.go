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

	client := newV31S3Client()
	alpha := newV31S3Store(t, client, "project:v31-alpha")
	beta := newV31S3Store(t, client, "project:v31-beta")
	request := ports.PutArtifactRequest{MediaType: "application/json", Content: []byte(`{"gate":"v31"}`)}
	first, err := alpha.Put(context.Background(), request)
	v31NoError(t, err)
	second, err := alpha.Put(context.Background(), request)
	v31NoError(t, err)
	if first != second || client.count() != 1 {
		t.Fatalf("conditional CAS not idempotent: first=%+v second=%+v objects=%d", first, second, client.count())
	}

	restarted := newV31S3Store(t, client, "project:v31-alpha")
	content, err := restarted.Get(context.Background(), first.Ref, first.Size)
	v31NoError(t, err)
	if string(content.Content) != string(request.Content) || content.Digest != first.Digest {
		t.Fatalf("restart read=%+v", content)
	}
	if _, err := beta.Get(context.Background(), first.Ref, first.Size); ports.ArtifactContractErrorCode(err) != ports.ArtifactErrorNotFound {
		t.Fatalf("cross-project read error=%v", err)
	}
	for _, locator := range client.locators() {
		if strings.Contains(locator.Key, "project:v31-alpha") || strings.Contains(locator.Key, "project:v31-beta") {
			t.Fatalf("project ref leaked in object key %q", locator.Key)
		}
	}

	metadataRequest := ports.PutArtifactRequest{MediaType: "text/plain", Content: []byte("causal metadata")}
	metadataStored, err := alpha.Put(context.Background(), metadataRequest)
	v31NoError(t, err)
	client.tamperProjectMetadata(metadataStored.Digest)
	if _, err := alpha.Get(context.Background(), metadataStored.Ref, metadataStored.Size); ports.ArtifactContractErrorCode(err) != ports.ArtifactErrorDigestMismatch {
		t.Fatalf("causal metadata substitution error=%v", err)
	}

	boundedRequest := ports.PutArtifactRequest{MediaType: "text/plain", Content: []byte("bounded")}
	boundedStored, err := alpha.Put(context.Background(), boundedRequest)
	v31NoError(t, err)
	client.tamperContent(boundedStored.Digest, bytes.Repeat([]byte("x"), 4096), boundedStored.Size)
	if _, err := alpha.Get(context.Background(), boundedStored.Ref, boundedStored.Size); ports.ArtifactContractErrorCode(err) != ports.ArtifactErrorSizeMismatch {
		t.Fatalf("unbounded object stream error=%v", err)
	}
	if got := client.lastOpenReadCount(); got != boundedStored.Size+1 {
		t.Fatalf("bounded stream read=%d want=%d", got, boundedStored.Size+1)
	}

	concurrent := ports.PutArtifactRequest{MediaType: "text/plain", Content: []byte("concurrent conditional object")}
	const workers = 24
	results := make([]ports.StoredArtifact, workers)
	errorsFound := make([]error, workers)
	start := make(chan struct{})
	var ready sync.WaitGroup
	var done sync.WaitGroup
	for index := range workers {
		ready.Add(1)
		done.Add(1)
		go func() {
			defer done.Done()
			ready.Done()
			<-start
			results[index], errorsFound[index] = alpha.Put(context.Background(), concurrent)
		}()
	}
	ready.Wait()
	beforeConcurrent := client.count()
	close(start)
	done.Wait()
	for index := range workers {
		if errorsFound[index] != nil || results[index] != results[0] {
			t.Fatalf("concurrent Put[%d] result=%+v error=%v", index, results[index], errorsFound[index])
		}
	}
	if got := client.count(); got != beforeConcurrent+1 {
		t.Fatalf("concurrent CAS object count=%d want=%d", got, beforeConcurrent+1)
	}

	client.failAfterNextWrite = true
	ambiguous := ports.PutArtifactRequest{MediaType: "text/plain", Content: []byte("effect before timeout")}
	if _, err := alpha.Put(context.Background(), ambiguous); err != nil {
		t.Fatalf("unknown outcome was not reconciled on immutable key: %v", err)
	}
}

func TestV31S3ArtifactFoundationDoesNotClaimOPS13Accreditation(t *testing.T) {
	fixture := readV31S3ArtifactFixture(t)
	if fixture.Status != "implemented_foundation_not_accredited" ||
		!reflect.DeepEqual(fixture.DeferredGates, []string{
			"real_s3_compatible_backend",
			"filesystem_and_s3_shared_contract_suite",
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
		"content_addressed_immutable_conditional_put",
		"digest_size_and_causal_metadata_verified_after_write",
		"opaque_project_isolation_without_project_ref_leak",
		"bounded_stream_read",
		"unknown_put_outcome_reconciled_on_same_key",
		"concurrent_idempotent_put",
		"stateless_restart",
	}
	if !reflect.DeepEqual(fixture.ImplementedBehaviors, wantBehaviors) {
		t.Fatalf("V31 S3 behavior contract drift: %v", fixture.ImplementedBehaviors)
	}
}

func newV31S3Store(t *testing.T, client artifactS3.Client, projectValue string) *artifactS3.Store {
	t.Helper()
	project, err := goal.NewProjectRef(projectValue)
	v31NoError(t, err)
	store, err := artifactS3.New(client, artifactS3.Options{
		Bucket: "v31-artifacts", ProjectRef: project, MaxObjectBytes: 1024,
		ReconcileTimeout: time.Second,
	})
	v31NoError(t, err)
	return store
}

type v31S3Object struct {
	content  []byte
	metadata artifactS3.ObjectMetadata
}

type v31S3Client struct {
	mu                 sync.Mutex
	objects            map[artifactS3.ObjectLocator]v31S3Object
	headSizeOverride   map[artifactS3.ObjectLocator]int64
	failAfterNextWrite bool
	lastRead           int64
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

func (client *v31S3Client) tamperProjectMetadata(digest string) {
	client.mu.Lock()
	defer client.mu.Unlock()
	for locator, object := range client.objects {
		if object.metadata.ArtifactDigest == digest {
			object.metadata.ProjectDigest = "substituted-project-digest"
			client.objects[locator] = object
			return
		}
	}
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
