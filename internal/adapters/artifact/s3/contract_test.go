package s3

import (
	"strings"
	"testing"
	"time"

	"orquesta/internal/adapters/artifact/contracttest"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestStorePassesSharedArtifactContractWithInjectedFake(t *testing.T) {
	backend := &s3ContractBackend{client: newMemoryClient()}
	contracttest.Run(t, contracttest.Backend{
		Open:                   backend.open,
		LogicalObjectCount:     backend.logicalObjectCount,
		TamperContent:          backend.tamperContent,
		TamperMetadata:         backend.tamperMetadata,
		TamperOversizedContent: backend.tamperOversizedContent,
		ResetReadCount:         backend.resetReadCount,
		ReadCount:              backend.client.lastOpenReadCount,
		AssertProjectRefOpaque: backend.assertProjectRefOpaque,
	})
}

type s3ContractBackend struct{ client *memoryClient }

func (backend *s3ContractBackend) open(t testing.TB, project goal.ProjectRef) contracttest.Handle {
	t.Helper()
	store, err := New(backend.client, Options{
		Bucket: "contract-artifacts", ProjectRef: project, MaxObjectBytes: 8192,
		ReconcileTimeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	return contracttest.Handle{Store: store}
}

func (backend *s3ContractBackend) logicalObjectCount(testing.TB) int {
	return backend.client.objectCount()
}

func (backend *s3ContractBackend) tamperContent(t testing.TB, handle contracttest.Handle, stored ports.StoredArtifact, content []byte) {
	t.Helper()
	store := handle.Store.(*Store)
	backend.client.tamperContent(store.locator(stored.Digest), content)
}

func (backend *s3ContractBackend) tamperMetadata(t testing.TB, handle contracttest.Handle, stored ports.StoredArtifact) {
	t.Helper()
	store := handle.Store.(*Store)
	key := memoryKey(store.locator(stored.Digest))
	backend.client.mu.Lock()
	defer backend.client.mu.Unlock()
	object, found := backend.client.objects[key]
	if !found {
		t.Fatal("artifact object not found for metadata tamper")
	}
	object.metadata.ProjectDigest = strings.Repeat("f", 64)
	backend.client.objects[key] = object
}

func (backend *s3ContractBackend) tamperOversizedContent(t testing.TB, handle contracttest.Handle, stored ports.StoredArtifact, content []byte) {
	t.Helper()
	store := handle.Store.(*Store)
	locator := store.locator(stored.Digest)
	backend.client.tamperContent(locator, content)
	backend.client.mu.Lock()
	backend.client.headSizeByKey[memoryKey(locator)] = stored.Size
	backend.client.mu.Unlock()
}

func (backend *s3ContractBackend) resetReadCount() {
	backend.client.mu.Lock()
	backend.client.lastRead = 0
	backend.client.mu.Unlock()
}

func (backend *s3ContractBackend) assertProjectRefOpaque(t testing.TB, project goal.ProjectRef) {
	t.Helper()
	for _, key := range backend.client.keys() {
		if strings.Contains(key, project.String()) {
			t.Fatalf("project ref leaked in object key %q", key)
		}
	}
}
