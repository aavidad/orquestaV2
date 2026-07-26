//go:build linux

package firecrackerlauncher

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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
	if got := openDescriptorCountForOwnerForTest(t, owned); got != before[owned] {
		t.Fatalf(
			"canceled sealed input retained owner %+v: before=%d after=%d",
			owned,
			before[owned],
			got,
		)
	}
}

func descriptorOwnerForFileForTest(t *testing.T, file *os.File) descriptorOwnerForTest {
	t.Helper()
	var stat unix.Stat_t
	if file == nil || unix.Fstat(int(file.Fd()), &stat) != nil {
		t.Fatal("could not identify descriptor owner")
	}
	return descriptorOwnerForTest{device: uint64(stat.Dev), inode: stat.Ino}
}

func openDescriptorOwnersForTest(
	t *testing.T,
	linkTarget string,
) map[descriptorOwnerForTest]int {
	t.Helper()
	entries, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		t.Fatal(err)
	}
	owners := make(map[descriptorOwnerForTest]int)
	for _, entry := range entries {
		fd, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		if linkTarget != "" {
			target, err := os.Readlink(filepath.Join("/proc/self/fd", entry.Name()))
			if err != nil || !strings.Contains(target, linkTarget) {
				continue
			}
		}
		var stat unix.Stat_t
		if unix.Fstat(fd, &stat) != nil {
			continue
		}
		owners[descriptorOwnerForTest{
			device: uint64(stat.Dev),
			inode:  stat.Ino,
		}]++
	}
	return owners
}

func openDescriptorCountForOwnerForTest(t *testing.T, owner descriptorOwnerForTest) int {
	t.Helper()
	return openDescriptorOwnersForTest(t, "")[owner]
}
