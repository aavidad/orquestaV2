//go:build !linux

package filesystem

import (
	"os"
	"testing"

	"orquesta/internal/ports"
)

func TestOpenFailsClosedWhenFilesystemSecurityIsUnsupported(t *testing.T) {
	root := os.TempDir()
	store, err := Open(root)
	if store != nil {
		_ = store.Close()
		t.Fatal("unsupported filesystem store opened")
	}
	if got := ports.ArtifactContractErrorCode(err); got != ports.ArtifactErrorFilesystemUnsupported {
		t.Fatalf("Open() error=%v code=%q want=%q", err, got, ports.ArtifactErrorFilesystemUnsupported)
	}
}
