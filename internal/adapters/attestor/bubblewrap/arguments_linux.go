//go:build linux

package bubblewrap

import (
	"errors"
	"os"
	"path"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"

	"orquesta/internal/goal"
)

type sealedArguments struct {
	*os.File
	auxiliary    []*os.File
	max, written int64
}

func newSealedArguments(max int64) (*sealedArguments, error) {
	fd, err := unix.MemfdCreate("orquesta-bwrap-args", unix.MFD_CLOEXEC|unix.MFD_ALLOW_SEALING)
	if err != nil {
		return nil, snapshotLimit()
	}
	return &sealedArguments{File: os.NewFile(uintptr(fd), "orquesta-bwrap-args"), max: max}, nil
}

func (arguments *sealedArguments) add(values ...string) error {
	for _, value := range values {
		need := int64(len(value)) + 1
		if arguments == nil || arguments.File == nil || strings.ContainsRune(value, 0) ||
			need > arguments.max-arguments.written {
			return snapshotLimit()
		}
		if count, err := arguments.File.Write(append([]byte(value), 0)); err != nil || count != int(need) {
			return snapshotLimit()
		}
		arguments.written += need
	}
	return nil
}

func (arguments *sealedArguments) seal() error {
	seals := unix.F_SEAL_WRITE | unix.F_SEAL_GROW | unix.F_SEAL_SHRINK | unix.F_SEAL_SEAL
	if arguments == nil || arguments.File == nil || arguments.written == 0 ||
		unix.Fchmod(int(arguments.Fd()), 0o400) != nil || rewindAndSeal(arguments.File, seals) != nil {
		return snapshotLimit()
	}
	got, err := unix.FcntlInt(arguments.Fd(), unix.F_GET_SEALS, 0)
	if err != nil || got&seals != seals {
		return snapshotLimit()
	}
	return nil
}

func (arguments *sealedArguments) Close() error {
	if arguments == nil {
		return nil
	}
	var result error
	if arguments.File != nil {
		err := arguments.File.Close()
		arguments.File = nil
		if !errors.Is(err, os.ErrClosed) {
			result = errors.Join(result, err)
		}
	}
	for _, file := range arguments.auxiliary {
		if file != nil {
			result = errors.Join(result, file.Close())
		}
	}
	arguments.auxiliary = nil
	return result
}

func appendSandboxPreamble(arguments *sealedArguments) error {
	return arguments.add("--unshare-all", "--unshare-user", "--disable-userns", "--assert-userns-disabled", "--die-with-parent", "--new-session",
		"--clearenv", "--cap-drop", "ALL", "--uid", "65534", "--gid", "65534", "--dir", toolchainMount,
		"--ro-bind-fd", "4", toolchainMount, "--perms", "0555", "--ro-bind-data", "5", goCommand,
		"--tmpfs", workspaceMount)
}

func appendSandboxTail(arguments *sealedArguments, limits Limits, spec goal.RequiredTestSpec) error {
	if err := arguments.add("--remount-ro", workspaceMount, "--size", strconv.FormatInt(limits.MemoryMaxBytes/2, 10), "--perms", "01777", "--tmpfs", "/tmp", "--proc", "/proc", "--dev", "/dev"); err != nil {
		return err
	}
	for _, item := range [][2]string{{"PATH", "/toolchain/bin"}, {"HOME", "/tmp"}, {"TMPDIR", "/tmp"}, {"GOCACHE", "/tmp/go-cache"}, {"GOPATH", "/tmp/go-path"}, {"GOTMPDIR", "/tmp"}, {"GOENV", "off"}, {"GOTOOLCHAIN", "local"}, {"CGO_ENABLED", "0"}, {"LANG", "C"}, {"LC_ALL", "C"}} {
		if err := arguments.add("--setenv", item[0], item[1]); err != nil {
			return err
		}
	}
	return arguments.add("--chdir", path.Join(workspaceMount, spec.WorkingDirectory()))
}

func modePermissions(mode string) string {
	if mode == "100755" {
		return "0555"
	}
	return "0444"
}
