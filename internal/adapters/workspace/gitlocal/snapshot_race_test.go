package gitlocal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"orquesta/internal/ports"
)

func TestSnapshotLooseBlobMutationAfterPlanFailsClosed(t *testing.T) {
	for _, mutation := range []string{"delete", "replace"} {
		t.Run(mutation, func(t *testing.T) {
			adapter, request, _ := committedSnapshot(t)
			repository := adapter.loc.(testLocator).binding.Path
			oid := trimOID(gitTestOutput(t, repository, "rev-parse", request.Subject.HeadOID+":allowed.txt"))
			objectPath := filepath.Join(repository, ".git", "objects", oid[:2], oid[2:])
			var mutationErr error
			adapter.snapshotRaceObserver = func(stage snapshotRaceStage, _ string) {
				if stage != snapshotRacePlanReady {
					return
				}
				if mutation == "delete" {
					mutationErr = os.Remove(objectPath)
					return
				}
				original := gitObjectContent(t, repository, "blob", oid)
				original[len(original)-1] ^= 1
				if err := os.Chmod(objectPath, 0o600); err != nil {
					mutationErr = err
					return
				}
				mutationErr = writeLooseGitObject(objectPath, "blob", original)
			}
			stream, openErr := adapter.OpenSnapshotStream(context.Background(), request)
			var readErr, closeErr error
			if stream != nil {
				_, readErr = io.Copy(io.Discard, stream)
				closeErr = stream.Close()
			}
			if mutationErr != nil {
				t.Fatal(mutationErr)
			}
			code := ErrorCodeOf(errors.Join(openErr, readErr, closeErr))
			if code != CodeSnapshotHash && code != CodeSnapshotInvalid {
				t.Fatalf("mutation=%s code=%q read=%v close=%v", mutation, code, readErr, closeErr)
			}
		})
	}
}

func TestOpenedSnapshotStreamSurvivesAdapterClose(t *testing.T) {
	adapter, request, _ := committedSnapshot(t)
	baseline := readSnapshotBytes(t, adapter, request)
	stream, err := adapter.OpenSnapshotStream(context.Background(), request)
	gitTestNoError(t, err)
	gitTestNoError(t, adapter.Close())
	content, readErr := io.ReadAll(stream)
	closeErr := stream.Close()
	if readErr != nil || closeErr != nil || !bytes.Equal(content, baseline) {
		t.Fatalf("admitted stream bytes=%d read=%v close=%v", len(content), readErr, closeErr)
	}
}

func TestSnapshotFixedOIDsIgnoreBranchMovement(t *testing.T) {
	adapter, request, _ := committedSnapshot(t)
	baseline := readSnapshotBytes(t, adapter, request)
	repository := adapter.loc.(testLocator).binding.Path
	var moveErr error
	adapter.snapshotRaceObserver = func(stage snapshotRaceStage, _ string) {
		if stage == snapshotRacePlanReady {
			_, moveErr = adapter.gitRun(context.Background(), repository, nil, "update-ref",
				"refs/heads/"+workspaceBranch(request.Subject.WorkspaceRef), request.Subject.ParentOID, request.Subject.HeadOID)
		}
	}
	captured := readSnapshotBytes(t, adapter, request)
	if moveErr != nil {
		t.Fatal(moveErr)
	}
	if !bytes.Equal(captured, baseline) {
		t.Fatal("branch movement changed fixed-OID snapshot bytes")
	}
}

func TestSnapshotSealedBytesSurvivePostEOFObjectDeletion(t *testing.T) {
	adapter, request, _ := committedSnapshot(t)
	captured := readSnapshotBytes(t, adapter, request)
	assertSealedSnapshot(t, captured)
	before := sha256.Sum256(captured)
	repository := adapter.loc.(testLocator).binding.Path
	oid := trimOID(gitTestOutput(t, repository, "rev-parse", request.Subject.HeadOID+":allowed.txt"))
	gitTestNoError(t, os.Remove(filepath.Join(repository, ".git", "objects", oid[:2], oid[2:])))
	after := sha256.Sum256(captured)
	if before != after {
		t.Fatal("post-EOF object deletion changed sealed capture")
	}
}

func TestSnapshotRereadsDuplicateBlobOIDWithoutHeapCache(t *testing.T) {
	content := []byte("same object bytes\n")
	fixture := newCommittedSnapshot(t, "", snapshotFixtureWrite{"allowed.txt", content}, snapshotFixtureWrite{"copy.txt", content})
	adapter, request := fixture.adapter, fixture.request
	reads := make(map[string]int)
	var mu sync.Mutex
	adapter.snapshotRaceObserver = func(stage snapshotRaceStage, oid string) {
		if stage == snapshotRaceBeforeBlobRead {
			mu.Lock()
			reads[oid]++
			mu.Unlock()
		}
	}
	captured := readSnapshotBytes(t, adapter, request)
	assertSealedSnapshot(t, captured)
	if count := bytes.Count(captured, content); count != 2 {
		t.Fatalf("duplicate blob serialized %d times", count)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(reads) != 1 {
		t.Fatalf("unique blob reads=%v", reads)
	}
	for oid, count := range reads {
		if count != 2 {
			t.Fatalf("blob %s read %d times", oid, count)
		}
	}
}

func TestSnapshotDuplicateBlobsUseNoHeapCache(t *testing.T) {
	content := bytes.Repeat([]byte{'x'}, 4*1024*1024+1)
	fixture := newCommittedSnapshot(t, "", snapshotFixtureWrite{"allowed.txt", content}, snapshotFixtureWrite{"copy.txt", content})
	reads := 0
	fixture.adapter.snapshotRaceObserver = func(stage snapshotRaceStage, _ string) {
		if stage == snapshotRaceBeforeBlobRead {
			reads++
		}
	}
	content = readSnapshotBytes(t, fixture.adapter, fixture.request)
	if reads != 2 || len(content) < 8*1024*1024 {
		t.Fatalf("large duplicate reads=%d stream-bytes=%d", reads, len(content))
	}
}

func TestSnapshotPackedObjectRemovalAfterPlanFailsClosed(t *testing.T) {
	adapter, request, _ := committedSnapshot(t)
	repository := adapter.loc.(testLocator).binding.Path
	gitTestOutput(t, repository, "gc", "--prune=now")
	packs, err := filepath.Glob(filepath.Join(repository, ".git", "objects", "pack", "*.pack"))
	if err != nil || len(packs) == 0 {
		t.Fatalf("packed fixture packs=%v err=%v", packs, err)
	}
	adapter.snapshotRaceObserver = func(stage snapshotRaceStage, _ string) {
		if stage != snapshotRacePlanReady {
			return
		}
		for _, pack := range packs {
			_ = os.Remove(pack)
			_ = os.Remove(strings.TrimSuffix(pack, ".pack") + ".idx")
		}
	}
	stream, openErr := adapter.OpenSnapshotStream(context.Background(), request)
	if openErr != nil {
		return
	}
	_, readErr := io.Copy(io.Discard, stream)
	closeErr := stream.Close()
	if errors.Join(readErr, closeErr) == nil {
		t.Fatal("packed object removal produced a valid stream")
	}
}

func TestSnapshotParentTreeClosureConsumedByDiffFailsClosedAfterPlanTamper(t *testing.T) {
	adapter, prepare := testAdapterAndPrepare(t)
	repository := adapter.loc.(testLocator).binding.Path
	gitTestNoError(t, os.Mkdir(filepath.Join(repository, "legacy"), 0o700))
	gitTestNoError(t, os.WriteFile(filepath.Join(repository, "legacy", "parent-only.txt"), []byte("parent\n"), 0o600))
	gitTestOutput(t, repository, "add", "legacy/parent-only.txt")
	gitTestOutput(t, repository, "commit", "-m", "parent tree closure")
	prepare.WriteSet = []string{"allowed.txt", "legacy"}
	prepare.WriteSetDigest = ports.WorkspaceWriteSetDigest(prepare.WriteSet)
	prepared, err := adapter.Prepare(context.Background(), prepare)
	gitTestNoError(t, err)
	workspace := adapter.workspacePath(prepare.WorkspaceRef)
	gitTestNoError(t, os.RemoveAll(filepath.Join(workspace, "legacy")))
	gitTestNoError(t, os.WriteFile(filepath.Join(workspace, "allowed.txt"), []byte("head\n"), 0o600))
	change, err := adapter.Commit(context.Background(), testCommit(t, prepare, prepared))
	gitTestNoError(t, err)
	request := gitSnapshotRequest(prepare, prepared, change)
	parentTree := trimOID(gitTestOutput(t, repository, "rev-parse", request.Subject.ParentOID+":legacy"))
	objectPath := filepath.Join(repository, ".git", "objects", parentTree[:2], parentTree[2:])
	var mutationErr error
	adapter.snapshotRaceObserver = func(stage snapshotRaceStage, _ string) {
		if stage == snapshotRacePlanReady {
			mutationErr = os.Remove(objectPath)
		}
	}
	stream, openErr := adapter.OpenSnapshotStream(context.Background(), request)
	if mutationErr != nil {
		t.Fatal(mutationErr)
	}
	if stream != nil {
		_, openErr = io.Copy(io.Discard, stream)
		openErr = errors.Join(openErr, stream.Close())
	}
	if code := ErrorCodeOf(openErr); code != CodeSnapshotHash && code != CodeSnapshotInvalid {
		t.Fatalf("parent tree closure tamper code=%q err=%v", code, openErr)
	}
}

func readSnapshotBytes(t *testing.T, adapter *Adapter, request ports.SnapshotVerificationRequest) []byte {
	t.Helper()
	stream, err := adapter.OpenSnapshotStream(context.Background(), request)
	gitTestNoError(t, err)
	content, readErr := io.ReadAll(stream)
	closeErr := stream.Close()
	if readErr != nil || closeErr != nil {
		t.Fatalf("read snapshot: read=%v close=%v", readErr, closeErr)
	}
	return content
}

func assertSealedSnapshot(t *testing.T, content []byte) {
	t.Helper()
	if len(content) < sha256.Size+1 || content[len(content)-sha256.Size-1] != snapshotEndTag {
		t.Fatal("snapshot sealing trailer missing")
	}
	want := sha256.Sum256(content[:len(content)-sha256.Size-1])
	if !bytes.Equal(content[len(content)-sha256.Size:], want[:]) {
		t.Fatal("snapshot sealing digest mismatch")
	}
}
