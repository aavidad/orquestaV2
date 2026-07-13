//go:build linux

package orquestasecurefile

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
)

func TestSecureFileV0CreateOnceConcurrentAndRejectHardlinkV0(t *testing.T) {
	root := filepath.Join(privateTempDirV0(t), "private", "nested")
	dir, err := OpenDirectoryV0(root, DirectoryOptionsV0{Create: true, CreateMode: 0o700, FinalMode: 0o700})
	if err != nil {
		t.Fatal(err)
	}
	defer dir.Close()
	options := FileOptionsV0{MaxBytes: 1024, ExactMode: 0o600}
	payloads := [][]byte{[]byte("alpha"), []byte("beta")}
	var wg sync.WaitGroup
	const creators = 128
	errs := make(chan error, creators)
	createdResults := make(chan bool, creators)
	for index := 0; index < creators; index++ {
		payload := payloads[index%len(payloads)]
		wg.Add(1)
		go func() {
			defer wg.Done()
			stored, created, err := CreateFileIfAbsentAtV0(dir, "binding.json", payload, options)
			if err == nil && !bytes.Equal(stored, payloads[0]) && !bytes.Equal(stored, payloads[1]) {
				err = errors.New("partial concurrent payload")
			}
			errs <- err
			createdResults <- created
		}()
	}
	wg.Wait()
	close(errs)
	close(createdResults)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	createdCount := 0
	for created := range createdResults {
		if created {
			createdCount++
		}
	}
	if createdCount != 1 {
		t.Fatalf("create-once winners=%d, want 1", createdCount)
	}
	path := filepath.Join(root, "binding.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatalf("published inode stat type=%T", info.Sys())
	}
	if stat.Nlink != 1 {
		t.Fatalf("published inode nlink=%d, want 1", stat.Nlink)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "binding.json" {
		t.Fatalf("named temporary files leaked: %+v", entries)
	}
	link := filepath.Join(root, "binding-hardlink.json")
	if err := os.Link(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadFileAtV0(dir, "binding.json", options); !errors.Is(err, ErrUnsafeFileV0) {
		t.Fatalf("hardlink accepted: %v", err)
	}
}

func TestSecureFileV0FailureBeforePublishLeavesNoNameV0(t *testing.T) {
	root := privateTempDirV0(t)
	dir, err := OpenDirectoryV0(root, DirectoryOptionsV0{FinalMode: 0o700})
	if err != nil {
		t.Fatal(err)
	}
	defer dir.Close()
	injected := errors.New("injected-before-publish")
	_, created, err := createFileIfAbsentAtV0(dir, "never-visible.json", []byte("complete"), FileOptionsV0{MaxBytes: 32, ExactMode: 0o600}, func() error {
		return injected
	})
	if !errors.Is(err, injected) || created {
		t.Fatalf("injected failure drifted: created=%v err=%v", created, err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("failure before publish leaked names: %+v", entries)
	}
}

func TestSecureFileV0PublishesAsEffectiveNonRootWhenAvailableV0(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("effective user is root")
	}
	root := privateTempDirV0(t)
	dir, err := OpenDirectoryV0(root, DirectoryOptionsV0{FinalMode: 0o700})
	if err != nil {
		t.Fatal(err)
	}
	defer dir.Close()
	if _, created, err := CreateFileIfAbsentAtV0(dir, "non-root.json", []byte("complete"), FileOptionsV0{MaxBytes: 32, ExactMode: 0o600}); err != nil || !created {
		t.Fatalf("non-root anonymous publish failed: created=%v err=%v", created, err)
	}
	info, err := os.Stat(filepath.Join(root, "non-root.json"))
	if err != nil {
		t.Fatal(err)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatalf("non-root inode stat type=%T", info.Sys())
	}
	if stat.Uid != uint32(os.Geteuid()) || stat.Nlink != 1 {
		t.Fatalf("non-root inode invalid: %+v", stat)
	}
}

func TestSecureFileV0RejectsNonCanonicalRelativePathsV0(t *testing.T) {
	root := privateTempDirV0(t)
	for _, rel := range []string{"a/../b", "a//b", "./b", " b", "b "} {
		if _, err := ReadFileBeneathV0(root, rel, FileOptionsV0{MaxBytes: 32}); !errors.Is(err, ErrInvalidPathV0) {
			t.Fatalf("non-canonical rel %q accepted: %v", rel, err)
		}
	}
}

func TestSecureFileV0RejectsUnsafeOrIncoherentDirectoryModesBeforeCreateV0(t *testing.T) {
	base := privateTempDirV0(t)
	cases := []struct {
		name    string
		options DirectoryOptionsV0
	}{
		{name: "group-write-create", options: DirectoryOptionsV0{Create: true, CreateMode: 0o770}},
		{name: "other-write-create", options: DirectoryOptionsV0{Create: true, CreateMode: 0o702}},
		{name: "group-write-final", options: DirectoryOptionsV0{Create: true, CreateMode: 0o700, FinalMode: 0o720}},
		{name: "incoherent-modes", options: DirectoryOptionsV0{Create: true, CreateMode: 0o750, FinalMode: 0o700}},
		{name: "special-mode-bits", options: DirectoryOptionsV0{Create: true, CreateMode: 0o1700}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(base, test.name, "child")
			if _, err := OpenDirectoryV0(path, test.options); !errors.Is(err, ErrInvalidPathV0) {
				t.Fatalf("unsafe options accepted: %v", err)
			}
			if _, err := os.Stat(filepath.Join(base, test.name)); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("directory created before option rejection: %v", err)
			}
		})
	}
}

func TestSecureFileV0CreatesPrivateControlDirectoryUnderPermissiveUmaskV0(t *testing.T) {
	base := privateTempDirV0(t)
	previous := syscall.Umask(0o002)
	defer syscall.Umask(previous)
	path := filepath.Join(base, "control")
	dir, err := OpenDirectoryV0(path, DirectoryOptionsV0{Create: true, CreateMode: 0o700, FinalMode: 0o700})
	if err != nil {
		t.Fatal(err)
	}
	dir.Close()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o700 {
		t.Fatalf("control mode under umask 0002=%04o, want 0700", mode)
	}
}

func TestSecureFileV0RejectsGroupOrOtherWritableFileWithoutExactModeV0(t *testing.T) {
	root := privateTempDirV0(t)
	dir, err := OpenDirectoryV0(root, DirectoryOptionsV0{})
	if err != nil {
		t.Fatal(err)
	}
	defer dir.Close()
	for _, mode := range []os.FileMode{0o620, 0o602} {
		name := "agent-result-" + mode.String() + ".json"
		path := filepath.Join(root, name)
		if err := os.WriteFile(path, []byte("result"), mode); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, mode); err != nil {
			t.Fatal(err)
		}
		if _, err := ReadFileAtV0(dir, name, FileOptionsV0{MaxBytes: 32}); !errors.Is(err, ErrUnsafeFileV0) {
			t.Fatalf("writable file mode %04o accepted: %v", mode, err)
		}
	}
}

func TestSecureFileV0RejectsInvalidExactModesBeforeIOV0(t *testing.T) {
	root := privateTempDirV0(t)
	dir, err := OpenDirectoryV0(root, DirectoryOptionsV0{FinalMode: 0o700})
	if err != nil {
		t.Fatal(err)
	}
	defer dir.Close()
	for _, mode := range []uint32{0o666, 0o620, 0o1600} {
		name := "invalid-mode-" + filepath.Base(os.FileMode(mode).String())
		options := FileOptionsV0{MaxBytes: 32, ExactMode: mode}
		if _, err := ReadFileAtV0(dir, name, options); !errors.Is(err, ErrInvalidPathV0) {
			t.Fatalf("ReadFileAtV0 accepted ExactMode %04o: %v", mode, err)
		}
		if _, err := ReadFileBeneathV0(filepath.Join(root, "missing-root"), name, options); !errors.Is(err, ErrInvalidPathV0) {
			t.Fatalf("ReadFileBeneathV0 opened before rejecting ExactMode %04o: %v", mode, err)
		}
		if _, _, err := CreateFileIfAbsentAtV0(dir, name, []byte("data"), options); !errors.Is(err, ErrInvalidPathV0) {
			t.Fatalf("CreateFileIfAbsentAtV0 accepted ExactMode %04o: %v", mode, err)
		}
		if _, err := os.Stat(filepath.Join(root, name)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("invalid ExactMode %04o created a name: %v", mode, err)
		}
	}
	if _, _, err := CreateFileIfAbsentAtV0(dir, "missing-exact", []byte("data"), FileOptionsV0{MaxBytes: 32}); !errors.Is(err, ErrInvalidPathV0) {
		t.Fatalf("create accepted missing ExactMode: %v", err)
	}
	if _, err := ReadFileAtV0(dir, "invalid-max", FileOptionsV0{}); !errors.Is(err, ErrInvalidPathV0) {
		t.Fatalf("read accepted invalid MaxBytes: %v", err)
	}
}

func TestSecureFileV0RejectsSpecialBitsOnExactFileV0(t *testing.T) {
	root := privateTempDirV0(t)
	dir, err := OpenDirectoryV0(root, DirectoryOptionsV0{FinalMode: 0o700})
	if err != nil {
		t.Fatal(err)
	}
	defer dir.Close()
	path := filepath.Join(root, "setuid-result")
	if err := os.WriteFile(path, []byte("result"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Chmod(path, 0o4600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadFileAtV0(dir, filepath.Base(path), FileOptionsV0{MaxBytes: 32, ExactMode: 0o600}); !errors.Is(err, ErrUnsafeFileV0) {
		t.Fatalf("exact read accepted setuid file: %v", err)
	}
	if _, err := ReadFileAtV0(dir, filepath.Base(path), FileOptionsV0{MaxBytes: 32}); !errors.Is(err, ErrUnsafeFileV0) {
		t.Fatalf("non-exact read accepted setuid file: %v", err)
	}
}

func TestSecureFileV0RootPathCannotBypassFinalModeV0(t *testing.T) {
	info, err := os.Stat(string(filepath.Separator))
	if err != nil {
		t.Fatal(err)
	}
	wrongMode := uint32(info.Mode().Perm()) ^ 0o100
	if _, err := OpenDirectoryV0(string(filepath.Separator), DirectoryOptionsV0{FinalMode: wrongMode}); !errors.Is(err, ErrUnsafeDirectoryV0) {
		t.Fatalf("root path bypassed final mode %04o: %v", wrongMode, err)
	}
}

func TestSecureFileV0RejectsSpecialBitsOnFinalDirectoryV0(t *testing.T) {
	base := privateTempDirV0(t)
	path := filepath.Join(base, "sticky-final")
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Chmod(path, 0o1700); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenDirectoryV0(path, DirectoryOptionsV0{FinalMode: 0o700}); !errors.Is(err, ErrUnsafeDirectoryV0) {
		t.Fatalf("final directory hid sticky bit: %v", err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := ReadFileAtV0(file, "target", FileOptionsV0{MaxBytes: 32}); !errors.Is(err, ErrUnsafeDirectoryV0) {
		t.Fatalf("final directory descriptor hid sticky bit: %v", err)
	}
}

func TestSecureFileV0RejectsSymlinkFIFOAndUnsafeAncestorV0(t *testing.T) {
	base := privateTempDirV0(t)
	target := filepath.Join(base, "target")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenDirectoryV0(filepath.Join(link, "child"), DirectoryOptionsV0{Create: true, FinalMode: 0o700}); !errors.Is(err, ErrUnsafeDirectoryV0) {
		t.Fatalf("symlink ancestor accepted: %v", err)
	}
	unsafe := filepath.Join(base, "unsafe")
	if err := os.Mkdir(unsafe, 0o777); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(unsafe, 0o777); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenDirectoryV0(filepath.Join(unsafe, "child"), DirectoryOptionsV0{Create: true, FinalMode: 0o700}); !errors.Is(err, ErrUnsafeDirectoryV0) {
		t.Fatalf("unsafe ancestor accepted: %v", err)
	}
	groupWritable := filepath.Join(base, "group-writable")
	if err := os.Mkdir(groupWritable, 0o770); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(groupWritable, 0o770); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenDirectoryV0(filepath.Join(groupWritable, "child"), DirectoryOptionsV0{Create: true, FinalMode: 0o700}); !errors.Is(err, ErrUnsafeDirectoryV0) {
		t.Fatalf("group-writable ancestor accepted: %v", err)
	}
	dir, err := OpenDirectoryV0(target, DirectoryOptionsV0{FinalMode: 0o700})
	if err != nil {
		t.Fatal(err)
	}
	defer dir.Close()
	if err := syscallMkfifoForTestV0(filepath.Join(target, "pipe")); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadFileAtV0(dir, "pipe", FileOptionsV0{MaxBytes: 32, ExactMode: 0o600}); !errors.Is(err, ErrUnsafeFileV0) {
		t.Fatalf("FIFO accepted: %v", err)
	}
}

func TestSecureFileV0RejectsInvalidDirectoryDescriptorsV0(t *testing.T) {
	root := privateTempDirV0(t)
	filePath := filepath.Join(root, "regular-file")
	if err := os.WriteFile(filePath, []byte("file"), 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(filePath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	assertSecureFileRejectsDirectoryFDV0(t, file, "regular file")

	closed, err := os.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := closed.Close(); err != nil {
		t.Fatal(err)
	}
	assertSecureFileRejectsDirectoryFDV0(t, closed, "closed directory")

	unsafeDirPath := filepath.Join(root, "unsafe-directory")
	if err := os.Mkdir(unsafeDirPath, 0o770); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(unsafeDirPath, 0o770); err != nil {
		t.Fatal(err)
	}
	unsafeDir, err := os.Open(unsafeDirPath)
	if err != nil {
		t.Fatal(err)
	}
	defer unsafeDir.Close()
	assertSecureFileRejectsDirectoryFDV0(t, unsafeDir, "unsafe directory")
}

func assertSecureFileRejectsDirectoryFDV0(t *testing.T, dir *os.File, label string) {
	t.Helper()
	options := FileOptionsV0{MaxBytes: 32, ExactMode: 0o600}
	if _, err := ReadFileAtV0(dir, "target", options); !errors.Is(err, ErrUnsafeDirectoryV0) {
		t.Fatalf("ReadFileAtV0 accepted %s descriptor: %v", label, err)
	}
	if _, _, err := CreateFileIfAbsentAtV0(dir, "target", []byte("data"), options); !errors.Is(err, ErrUnsafeDirectoryV0) {
		t.Fatalf("CreateFileIfAbsentAtV0 accepted %s descriptor: %v", label, err)
	}
}

func privateTempDirV0(t *testing.T) string {
	t.Helper()
	path := t.TempDir()
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}
