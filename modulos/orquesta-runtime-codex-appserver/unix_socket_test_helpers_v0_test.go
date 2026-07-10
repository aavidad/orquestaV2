package orquestaruntimecodexappserver

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

const unixSocketAliasRootEnvV0 = "ORQUESTA_TEST_UNIX_SOCKET_ALIAS_ROOT"

var errUnixSocketTestPathTooLongV0 = errors.New("ruta Unix de test excede sockaddr_un")

type unixSocketTestPathIdentityV0 struct {
	device uint64
	inode  uint64
	mode   uint32
	uid    uint32
}

type unixSocketTestAliasV0 struct {
	path              string
	directory         string
	aliasIdentity     unixSocketTestPathIdentityV0
	directoryIdentity unixSocketTestPathIdentityV0
}

func shortUnixSocketTestRootV0(t *testing.T) string {
	t.Helper()
	target := t.TempDir()
	suffix := filepath.Join("runtime", codexAppServerTmuxDirV0, "codex-app-server.sock")
	alias, err := createShortUnixSocketTestAliasV0(target, suffix)
	if err != nil {
		t.Fatalf("crear alias corto para socket Unix: %v", err)
	}
	t.Cleanup(func() {
		if err := alias.cleanupV0(); err != nil {
			t.Errorf("limpiar alias corto para socket Unix: %v", err)
		}
	})
	return alias.path
}

func createShortUnixSocketTestAliasV0(target, suffix string) (unixSocketTestAliasV0, error) {
	if configured := strings.TrimSpace(os.Getenv(unixSocketAliasRootEnvV0)); configured != "" {
		return createUnixSocketTestAliasAtV0(configured, target, suffix, true)
	}

	var attempts []error
	if xdgRuntimeDir := strings.TrimSpace(os.Getenv("XDG_RUNTIME_DIR")); xdgRuntimeDir != "" {
		alias, err := createUnixSocketTestAliasAtV0(xdgRuntimeDir, target, suffix, true)
		if err == nil {
			return alias, nil
		}
		attempts = append(attempts, fmt.Errorf("XDG_RUNTIME_DIR: %w", err))
	}
	for _, sharedRoot := range []string{"/dev/shm", "/tmp"} {
		alias, err := createUnixSocketTestAliasAtV0(sharedRoot, target, suffix, false)
		if err == nil {
			return alias, nil
		}
		attempts = append(attempts, fmt.Errorf("%s: %w", sharedRoot, err))
	}
	return unixSocketTestAliasV0{}, errors.Join(attempts...)
}

func createUnixSocketTestAliasAtV0(base, target, suffix string, requirePrivateBase bool) (unixSocketTestAliasV0, error) {
	baseIdentity, err := validateUnixSocketAliasBaseV0(base, os.Geteuid(), requirePrivateBase)
	if err != nil {
		return unixSocketTestAliasV0{}, err
	}
	directory, err := os.MkdirTemp(filepath.Clean(base), "oq-")
	if err != nil {
		return unixSocketTestAliasV0{}, fmt.Errorf("crear subdirectorio privado: %w", err)
	}
	removeDirectory := true
	defer func() {
		if removeDirectory {
			_ = os.Remove(directory)
		}
	}()

	currentBaseIdentity, err := validateUnixSocketAliasBaseV0(base, os.Geteuid(), requirePrivateBase)
	if err != nil || currentBaseIdentity != baseIdentity {
		return unixSocketTestAliasV0{}, fmt.Errorf("raiz sustituida durante la creacion")
	}
	directoryIdentity, err := validateUnixSocketAliasBaseV0(directory, os.Geteuid(), true)
	if err != nil {
		return unixSocketTestAliasV0{}, fmt.Errorf("subdirectorio privado invalido: %w", err)
	}
	aliasPath := filepath.Join(directory, "r")
	if suffix != "" {
		if err := validateUnixSocketPathLengthV0(filepath.Join(aliasPath, suffix)); err != nil {
			return unixSocketTestAliasV0{}, err
		}
	}
	if err := os.Symlink(target, aliasPath); err != nil {
		return unixSocketTestAliasV0{}, fmt.Errorf("crear symlink a t.TempDir: %w", err)
	}
	aliasIdentity, err := unixSocketTestLstatIdentityV0(aliasPath, true)
	if err != nil {
		_ = os.Remove(aliasPath)
		return unixSocketTestAliasV0{}, fmt.Errorf("verificar alias creado: %w", err)
	}
	linkedTarget, err := os.Readlink(aliasPath)
	if err != nil || linkedTarget != target {
		_ = os.Remove(aliasPath)
		return unixSocketTestAliasV0{}, fmt.Errorf("alias no apunta al t.TempDir esperado")
	}
	removeDirectory = false
	return unixSocketTestAliasV0{
		path:              aliasPath,
		directory:         directory,
		aliasIdentity:     aliasIdentity,
		directoryIdentity: directoryIdentity,
	}, nil
}

func validateUnixSocketAliasBaseV0(path string, expectedEUID int, requirePrivate bool) (unixSocketTestPathIdentityV0, error) {
	clean := filepath.Clean(path)
	if !filepath.IsAbs(clean) {
		return unixSocketTestPathIdentityV0{}, fmt.Errorf("raiz no absoluta: %q", path)
	}
	info, err := unixSocketTestLstatPathV0(clean)
	if err != nil {
		return unixSocketTestPathIdentityV0{}, err
	}
	if !info.IsDir() {
		return unixSocketTestPathIdentityV0{}, fmt.Errorf("raiz no es directorio: %q", clean)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return unixSocketTestPathIdentityV0{}, fmt.Errorf("stat Unix no disponible: %q", clean)
	}
	if requirePrivate && int(stat.Uid) != expectedEUID {
		return unixSocketTestPathIdentityV0{}, fmt.Errorf("raiz con owner ajeno: uid=%d euid=%d", stat.Uid, expectedEUID)
	}
	if requirePrivate && info.Mode().Perm()&0o077 != 0 {
		return unixSocketTestPathIdentityV0{}, fmt.Errorf("raiz no privada: mode=%#o", info.Mode().Perm())
	}
	if requirePrivate && info.Mode().Perm()&0o300 != 0o300 {
		return unixSocketTestPathIdentityV0{}, fmt.Errorf("raiz no escribible/atravesable por owner: mode=%#o", info.Mode().Perm())
	}
	return unixSocketTestPathIdentityV0{device: stat.Dev, inode: stat.Ino, mode: stat.Mode, uid: stat.Uid}, nil
}

func unixSocketTestLstatPathV0(path string) (os.FileInfo, error) {
	current := string(filepath.Separator)
	parts := strings.Split(strings.TrimPrefix(filepath.Clean(path), current), current)
	var info os.FileInfo
	for _, part := range parts {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		var err error
		info, err = os.Lstat(current)
		if err != nil {
			return nil, fmt.Errorf("lstat %q: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("componente symlink rechazado: %q", current)
		}
	}
	if info == nil {
		return os.Lstat(current)
	}
	return info, nil
}

func unixSocketTestLstatIdentityV0(path string, requireSymlink bool) (unixSocketTestPathIdentityV0, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return unixSocketTestPathIdentityV0{}, err
	}
	if requireSymlink && info.Mode()&os.ModeSymlink == 0 {
		return unixSocketTestPathIdentityV0{}, fmt.Errorf("no es symlink: %q", path)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return unixSocketTestPathIdentityV0{}, fmt.Errorf("stat Unix no disponible: %q", path)
	}
	return unixSocketTestPathIdentityV0{device: stat.Dev, inode: stat.Ino, mode: stat.Mode, uid: stat.Uid}, nil
}

func (alias unixSocketTestAliasV0) cleanupV0() error {
	if alias.path == "" || alias.directory == "" {
		return nil
	}
	identity, err := unixSocketTestLstatIdentityV0(alias.path, false)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err == nil {
		if identity != alias.aliasIdentity {
			return fmt.Errorf("alias sustituido; no se elimina %q", alias.path)
		}
		if err := os.Remove(alias.path); err != nil {
			return err
		}
	}
	directoryIdentity, err := unixSocketTestLstatIdentityV0(alias.directory, false)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if directoryIdentity != alias.directoryIdentity {
		return fmt.Errorf("subdirectorio sustituido; no se elimina %q", alias.directory)
	}
	if err := os.Remove(alias.directory); err != nil {
		return fmt.Errorf("retirar subdirectorio propio: %w", err)
	}
	return nil
}

func validateUnixSocketPathLengthV0(socketPath string) error {
	capacity := len(syscall.RawSockaddrUnix{}.Path)
	if strings.IndexByte(socketPath, 0) >= 0 || len(socketPath) >= capacity {
		return fmt.Errorf("%w: len=%d max=%d path=%q", errUnixSocketTestPathTooLongV0, len(socketPath), capacity-1, socketPath)
	}
	return nil
}

func listenUnixFDForGenerationTestV0(socketPath string) (int, error) {
	if err := validateUnixSocketPathLengthV0(socketPath); err != nil {
		return -1, err
	}
	fd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_STREAM|syscall.SOCK_CLOEXEC, 0)
	if err != nil {
		return -1, err
	}
	if err := syscall.Bind(fd, &syscall.SockaddrUnix{Name: socketPath}); err != nil {
		_ = syscall.Close(fd)
		return -1, err
	}
	if err := syscall.Listen(fd, 16); err != nil {
		_ = syscall.Close(fd)
		return -1, err
	}
	return fd, nil
}

func TestShortUnixSocketTestRootV0AcotaSockaddrV0(t *testing.T) {
	root := shortUnixSocketTestRootV0(t)
	socketPath := filepath.Join(root, "runtime", codexAppServerTmuxDirV0, "codex-app-server.sock")
	if err := validateUnixSocketPathLengthV0(socketPath); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(socketPath), 0o700); err != nil {
		t.Fatal(err)
	}
}

func TestUnixSocketAliasRootV0ConfiguradaNoRequiereTmpYSeLimpiaV0(t *testing.T) {
	configuredRoot := t.TempDir()
	if err := os.Chmod(configuredRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv(unixSocketAliasRootEnvV0, configuredRoot)
	t.Setenv("XDG_RUNTIME_DIR", filepath.Join(t.TempDir(), "inexistente"))
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "tmp-inexistente"))
	target := t.TempDir()
	alias, err := createShortUnixSocketTestAliasV0(target, "")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(alias.directory) != configuredRoot {
		t.Fatalf("alias fuera de root configurada: alias=%s root=%s", alias.path, configuredRoot)
	}
	linkedTarget, err := os.Readlink(alias.path)
	if err != nil || linkedTarget != target {
		t.Fatalf("alias target=%q err=%v want=%q", linkedTarget, err, target)
	}
	if err := alias.cleanupV0(); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(configuredRoot)
	if err != nil || len(entries) != 0 {
		t.Fatalf("cleanup dejo artefactos propios: entries=%v err=%v", entries, err)
	}
}

func TestUnixSocketAliasRootV0PrefiereXDGPrivadoSinTmpV0(t *testing.T) {
	xdgRuntimeDir := t.TempDir()
	if err := os.Chmod(xdgRuntimeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv(unixSocketAliasRootEnvV0, "")
	t.Setenv("XDG_RUNTIME_DIR", xdgRuntimeDir)
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "tmp-inexistente"))
	alias, err := createShortUnixSocketTestAliasV0(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(alias.directory) != xdgRuntimeDir {
		t.Fatalf("alias no uso XDG_RUNTIME_DIR: alias=%s xdg=%s", alias.path, xdgRuntimeDir)
	}
	if err := alias.cleanupV0(); err != nil {
		t.Fatal(err)
	}
}

func TestUnixSocketAliasCleanupV0NoEliminaSustitutoV0(t *testing.T) {
	configuredRoot := t.TempDir()
	if err := os.Chmod(configuredRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	alias, err := createUnixSocketTestAliasAtV0(configuredRoot, t.TempDir(), "", true)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(alias.path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(alias.path, []byte("sustituto\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := alias.cleanupV0(); err == nil || !strings.Contains(err.Error(), "alias sustituido") {
		t.Fatalf("cleanup acepto sustituto: %v", err)
	}
	if raw, err := os.ReadFile(alias.path); err != nil || string(raw) != "sustituto\n" {
		t.Fatalf("cleanup elimino/muto sustituto: raw=%q err=%v", raw, err)
	}
	if err := os.Remove(alias.path); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(alias.directory); err != nil {
		t.Fatal(err)
	}
}

func TestUnixSocketAliasRootV0RechazaPermisosOwnerYSymlinkV0(t *testing.T) {
	t.Run("permisos", func(t *testing.T) {
		root := t.TempDir()
		if err := os.Chmod(root, 0o750); err != nil {
			t.Fatal(err)
		}
		if _, err := validateUnixSocketAliasBaseV0(root, os.Geteuid(), true); err == nil || !strings.Contains(err.Error(), "no privada") {
			t.Fatalf("root con permisos amplios aceptada: %v", err)
		}
	})
	t.Run("no-escribible", func(t *testing.T) {
		root := t.TempDir()
		if err := os.Chmod(root, 0o500); err != nil {
			t.Fatal(err)
		}
		if _, err := validateUnixSocketAliasBaseV0(root, os.Geteuid(), true); err == nil || !strings.Contains(err.Error(), "no escribible") {
			t.Fatalf("root no escribible aceptada: %v", err)
		}
	})
	t.Run("owner", func(t *testing.T) {
		root := t.TempDir()
		if _, err := validateUnixSocketAliasBaseV0(root, os.Geteuid()+1, true); err == nil || !strings.Contains(err.Error(), "owner ajeno") {
			t.Fatalf("root con owner ajeno aceptada: %v", err)
		}
	})
	t.Run("symlink", func(t *testing.T) {
		target := t.TempDir()
		link := filepath.Join(t.TempDir(), "root-link")
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}
		if _, err := validateUnixSocketAliasBaseV0(link, os.Geteuid(), true); err == nil || !strings.Contains(err.Error(), "symlink rechazado") {
			t.Fatalf("root symlink aceptada: %v", err)
		}
	})
}

func TestListenUnixFDForGenerationTestV0RechazaLongitudAntesDeSocketV0(t *testing.T) {
	tooLong := "/" + strings.Repeat("x", len(syscall.RawSockaddrUnix{}.Path))
	fd, err := listenUnixFDForGenerationTestV0(tooLong)
	if fd != -1 || !errors.Is(err, errUnixSocketTestPathTooLongV0) {
		t.Fatalf("fd=%d err=%v", fd, err)
	}
}
