package gitlocal

import (
	"bytes"
	"compress/zlib"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/unix"

	"orquesta/internal/ports"
)

func TestSnapshotObjectTamperFailsGitHashRecalculation(t *testing.T) {
	for _, objectType := range []string{"head", "parent", "tree", "parent_tree", "blob"} {
		t.Run(objectType, func(t *testing.T) {
			adapter, request, _ := committedSnapshot(t)
			repository := adapter.loc.(testLocator).binding.Path
			oid, gitType := request.Subject.HeadOID, "commit"
			switch objectType {
			case "parent":
				oid = request.Subject.ParentOID
			case "tree":
				oid, gitType = request.Subject.TreeOID, "tree"
			case "parent_tree":
				oid = strings.TrimSpace(string(gitTestOutput(t, repository, "rev-parse", request.Subject.ParentOID+"^{tree}")))
				gitType = "tree"
			case "blob":
				oid = strings.TrimSpace(string(gitTestOutput(t, repository, "rev-parse", request.Subject.HeadOID+":allowed.txt")))
				gitType = "blob"
			}
			original := gitObjectContent(t, repository, gitType, oid)
			mutated := append([]byte(nil), original...)
			mutated[len(mutated)-1] ^= 1
			objectPath := filepath.Join(repository, ".git", "objects", oid[:2], oid[2:])
			gitTestNoError(t, os.Chmod(objectPath, 0o600))
			gitTestNoError(t, writeLooseGitObject(objectPath, gitType, mutated))
			err := drainSnapshot(adapter, request)
			code := ErrorCodeOf(err)
			if code != CodeSnapshotHash && (objectType != "parent_tree" || code != CodeChangeConflict) {
				t.Fatalf("tampered %s code=%q err=%v", objectType, code, err)
			}
		})
	}
}

func TestSnapshotStreamUsesSealedMemfd(t *testing.T) {
	adapter, request, _ := committedSnapshot(t)
	stream, err := adapter.OpenSnapshotStream(context.Background(), request)
	gitTestNoError(t, err)
	file, ok := stream.(*os.File)
	if !ok {
		t.Fatalf("snapshot stream type %T is not a memfd", stream)
	}
	seals, err := unix.FcntlInt(file.Fd(), unix.F_GET_SEALS, 0)
	if err != nil || seals&snapshotMemfdSeals != snapshotMemfdSeals {
		t.Fatalf("snapshot seals=%x err=%v", seals, err)
	}
	if _, err := file.WriteAt([]byte{'x'}, 0); !errors.Is(err, unix.EPERM) {
		t.Fatalf("sealed snapshot write err=%v", err)
	}
	gitTestNoError(t, stream.Close())
}

func TestSnapshotStreamEnforcesSubjectByteLimit(t *testing.T) {
	adapter, request, _ := committedSnapshot(t)
	adapter.maxSnapshotBytes = 1
	stream, err := adapter.OpenSnapshotStream(context.Background(), request)
	if err != nil {
		if ErrorCodeOf(err) != CodeSnapshotLimit {
			t.Fatalf("open limit err=%v", err)
		}
		return
	}
	_, readErr := io.Copy(io.Discard, stream)
	closeErr := stream.Close()
	if ErrorCodeOf(errors.Join(readErr, closeErr)) != CodeSnapshotLimit {
		t.Fatalf("limit read=%v close=%v", readErr, closeErr)
	}
}

func gitObjectContent(t *testing.T, repository, objectType, oid string) []byte {
	t.Helper()
	command := []string{"cat-file", objectType, oid}
	output := gitTestOutput(t, repository, command...)
	return output
}

func gitTestOutput(t *testing.T, directory string, args ...string) []byte {
	t.Helper()
	command := exec.Command("/usr/bin/git", append([]string{"-C", directory}, args...)...)
	command.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + directory, "GIT_CONFIG_NOSYSTEM=1"}
	output, err := command.Output()
	gitTestNoError(t, err)
	return output
}

func writeLooseGitObject(path, objectType string, content []byte) error {
	var compressed bytes.Buffer
	writer := zlib.NewWriter(&compressed)
	if _, err := writer.Write([]byte(objectType + " " + decimalInt(len(content)) + "\x00")); err != nil {
		return err
	}
	if _, err := writer.Write(content); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return os.WriteFile(path, compressed.Bytes(), 0o600)
}

func decimalInt(value int) string {
	if value == 0 {
		return "0"
	}
	var result [32]byte
	index := len(result)
	for value > 0 {
		index--
		result[index] = byte('0' + value%10)
		value /= 10
	}
	return string(result[index:])
}

type committedSnapshotFixture struct {
	adapter   *Adapter
	prepare   ports.WorkspacePrepareRequest
	prepared  ports.WorkspacePrepared
	change    ports.CommitResult
	request   ports.SnapshotVerificationRequest
	workspace string
}

type snapshotFixtureWrite struct {
	name    string
	content []byte
}

func newCommittedSnapshot(t *testing.T, format string, writes ...snapshotFixtureWrite) committedSnapshotFixture {
	t.Helper()
	adapter, prepare := testAdapterAndPrepareFormat(t, format)
	if len(writes) == 0 {
		writes = []snapshotFixtureWrite{{"allowed.txt", []byte("readonly snapshot\n")}}
	}
	prepare.WriteSet = make([]string, len(writes))
	for index, write := range writes {
		prepare.WriteSet[index] = write.name
	}
	prepare.WriteSetDigest = ports.WorkspaceWriteSetDigest(prepare.WriteSet)
	prepared, err := adapter.Prepare(context.Background(), prepare)
	gitTestNoError(t, err)
	workspace := adapter.workspacePath(prepare.WorkspaceRef)
	for _, write := range writes {
		gitTestNoError(t, os.WriteFile(filepath.Join(workspace, write.name), write.content, 0o600))
	}
	change, err := adapter.Commit(context.Background(), testCommit(t, prepare, prepared))
	gitTestNoError(t, err)
	return committedSnapshotFixture{adapter, prepare, prepared, change,
		gitSnapshotRequest(prepare, prepared, change), workspace}
}

func committedSnapshot(t *testing.T) (*Adapter, ports.SnapshotVerificationRequest, string) {
	fixture := newCommittedSnapshot(t, "")
	return fixture.adapter, fixture.request, fixture.workspace
}
