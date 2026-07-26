//go:build linux

package firecrackerlauncher

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

type descriptorBarrierReaderForTest struct {
	entered     chan struct{}
	release     chan struct{}
	enterOnce   sync.Once
	releaseOnce sync.Once
}

type descriptorOwnerForTest struct {
	device uint64
	inode  uint64
}

func (reader *descriptorBarrierReaderForTest) Read([]byte) (int, error) {
	reader.enterOnce.Do(func() { close(reader.entered) })
	<-reader.release
	return 0, errors.New("test reader released")
}

func (reader *descriptorBarrierReaderForTest) Release() {
	reader.releaseOnce.Do(func() { close(reader.release) })
}

func TestCanceledSealedInputReclaimsItsOwnedDescriptor(t *testing.T) {
	const memfdTarget = "memfd:orquesta-firecracker-input"

	before := openDescriptorOwnersForTest(t, memfdTarget)
	reader := &descriptorBarrierReaderForTest{
		entered: make(chan struct{}),
		release: make(chan struct{}),
	}
	defer reader.Release()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan struct {
		file *os.File
		err  error
	}, 1)
	go func() {
		file, _, err := newSealedInputFromReaderContext(
			ctx,
			reader,
			int64(firecrackerSectorBytes),
			1<<20,
		)
		result <- struct {
			file *os.File
			err  error
		}{file: file, err: err}
	}()
	select {
	case <-reader.entered:
	case <-time.After(time.Second):
		t.Fatal("sealed input did not reach ownership barrier")
	}

	during := openDescriptorOwnersForTest(t, memfdTarget)
	var owned descriptorOwnerForTest
	newOwners := 0
	for owner, open := range during {
		if open > before[owner] {
			owned = owner
			newOwners++
		}
	}
	if newOwners != 1 {
		t.Fatalf(
			"ownership barrier found %d new descriptors: before=%v during=%v",
			newOwners,
			before,
			during,
		)
	}
	cancel()
	reader.Release()
	select {
	case got := <-result:
		if got.file != nil {
			_ = got.file.Close()
			t.Fatalf("canceled sealed input returned descriptor for owner %+v", owned)
		}
		if ErrorCode(got.err) != CodeUnavailable {
			t.Fatalf("canceled sealed input err=%v", got.err)
		}
	case <-time.After(time.Second):
		t.Fatal("canceled sealed input did not return")
	}
	waitForDescriptorOwnerCountForTest(t, owned, before[owned])
}

func descriptorOwnerForFileForTest(t *testing.T, file *os.File) descriptorOwnerForTest {
	t.Helper()
	var stat unix.Stat_t
	if file == nil || unix.Fstat(int(file.Fd()), &stat) != nil {
		t.Fatal("could not identify descriptor owner")
	}
	return descriptorOwnerForTest{device: uint64(stat.Dev), inode: stat.Ino}
}

func registeredConnectionOwnerForTest(t *testing.T, server *Server) descriptorOwnerForTest {
	t.Helper()
	server.mu.Lock()
	defer server.mu.Unlock()
	if len(server.connections) != 1 {
		t.Fatalf("expected one registered connection, got %d", len(server.connections))
	}
	for connection := range server.connections {
		var stat unix.Stat_t
		if unix.Fstat(connection, &stat) != nil || stat.Mode&unix.S_IFMT != unix.S_IFSOCK {
			t.Fatalf("could not identify registered connection %d", connection)
		}
		return descriptorOwnerForTest{device: uint64(stat.Dev), inode: stat.Ino}
	}
	t.Fatal("registered connection disappeared")
	return descriptorOwnerForTest{}
}

func openDescriptorOwnersForTest(
	t *testing.T,
	linkTarget string,
) map[descriptorOwnerForTest]int {
	t.Helper()
	procFD, err := os.Open("/proc/self/fd")
	if err != nil {
		t.Fatal(err)
	}
	defer procFD.Close()
	entries, err := procFD.ReadDir(-1)
	if err != nil {
		t.Fatal(err)
	}
	owners := make(map[descriptorOwnerForTest]int)
	for _, entry := range entries {
		fd, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		targetBefore, ok := procDescriptorTargetForTest(int(procFD.Fd()), entry.Name())
		if !ok || linkTarget != "" && !strings.Contains(targetBefore, linkTarget) {
			continue
		}
		var sourceBefore unix.Stat_t
		if unix.Fstat(fd, &sourceBefore) != nil {
			continue
		}
		stable, err := unix.Openat(
			int(procFD.Fd()),
			entry.Name(),
			unix.O_PATH|unix.O_CLOEXEC,
			0,
		)
		if err != nil {
			continue
		}
		var pinned, sourceAfter unix.Stat_t
		targetAfter, targetOK := procDescriptorTargetForTest(int(procFD.Fd()), entry.Name())
		pinnedErr := unix.Fstat(stable, &pinned)
		sourceErr := unix.Fstat(fd, &sourceAfter)
		_ = unix.Close(stable)
		if !targetOK || targetAfter != targetBefore || pinnedErr != nil || sourceErr != nil ||
			!sameDescriptorOwnerForTest(sourceBefore, pinned) ||
			!sameDescriptorOwnerForTest(pinned, sourceAfter) {
			continue
		}
		owners[descriptorOwnerForTest{
			device: uint64(pinned.Dev),
			inode:  pinned.Ino,
		}]++
	}
	return owners
}

func procDescriptorTargetForTest(procFD int, name string) (string, bool) {
	buffer := make([]byte, 4096)
	count, err := unix.Readlinkat(procFD, name, buffer)
	if err != nil || count == len(buffer) {
		return "", false
	}
	return string(buffer[:count]), true
}

func sameDescriptorOwnerForTest(first, second unix.Stat_t) bool {
	return first.Dev == second.Dev && first.Ino == second.Ino
}

func openDescriptorCountForOwnerForTest(t *testing.T, owner descriptorOwnerForTest) int {
	t.Helper()
	return openDescriptorOwnersForTest(t, "")[owner]
}

func waitForDescriptorOwnerCountForTest(
	t *testing.T,
	owner descriptorOwnerForTest,
	want int,
) {
	t.Helper()
	for attempts := 0; attempts < 100; attempts++ {
		if openDescriptorCountForOwnerForTest(t, owner) == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf(
		"descriptor owner %+v did not return to %d; got %d",
		owner,
		want,
		openDescriptorCountForOwnerForTest(t, owner),
	)
}
