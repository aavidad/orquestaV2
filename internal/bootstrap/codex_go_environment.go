package bootstrap

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	codexGoCacheInvalid     = "bootstrap.codex_go_cache_invalid"
	codexGoToolchainInvalid = "bootstrap.codex_go_toolchain_invalid"
)

func prepareCodexGoEnvironment(
	inherited map[string]string,
	configuredCacheRoot string,
	configuredToolchainRoot string,
) (map[string]string, error) {
	cacheRoot, err := canonicalCodexGoCacheRoot(configuredCacheRoot)
	if err != nil {
		return nil, errors.New(codexGoCacheInvalid)
	}
	toolchainRoot := ""
	if configuredToolchainRoot != "" {
		toolchainRoot, err = validateCodexGoToolchain(configuredToolchainRoot)
		if err != nil {
			return nil, errors.New(codexGoToolchainInvalid)
		}
	}
	cachePaths := map[string]string{
		"GOCACHE":    filepath.Join(cacheRoot, "build"),
		"GOMODCACHE": filepath.Join(cacheRoot, "modules"),
		"GOPATH":     filepath.Join(cacheRoot, "gopath"),
		"GOTMPDIR":   filepath.Join(cacheRoot, "go-tmp"),
		"TMPDIR":     filepath.Join(cacheRoot, "tmp"),
	}
	if err := ensurePrivateCodexGoCacheDirectory(cacheRoot); err != nil {
		return nil, errors.New(codexGoCacheInvalid)
	}
	for _, path := range cachePaths {
		if err := ensurePrivateCodexGoCacheDirectory(path); err != nil {
			return nil, errors.New(codexGoCacheInvalid)
		}
	}

	environment := make(map[string]string, len(inherited)+len(cachePaths)+3)
	for name, value := range inherited {
		environment[name] = value
	}
	environment["GOENV"] = "off"
	environment["GOTOOLCHAIN"] = "local"
	for name, path := range cachePaths {
		environment[name] = path
	}
	delete(environment, "GOROOT")

	if toolchainRoot == "" {
		return environment, nil
	}
	environment["GOROOT"] = toolchainRoot
	toolchainBin := filepath.Join(toolchainRoot, "bin")
	if current := environment["PATH"]; current != "" {
		environment["PATH"] = toolchainBin + string(filepath.ListSeparator) + current
	} else {
		environment["PATH"] = toolchainBin
	}
	return environment, nil
}

func canonicalCodexGoCacheRoot(configuredRoot string) (string, error) {
	if configuredRoot == "" ||
		strings.TrimSpace(configuredRoot) != configuredRoot ||
		strings.ContainsRune(configuredRoot, '\x00') {
		return "", errors.New(codexGoCacheInvalid)
	}
	absolute, err := filepath.Abs(filepath.Clean(configuredRoot))
	if err != nil {
		return "", err
	}
	resolved, err := canonicalRuntimePath(configuredRoot)
	if err != nil || resolved != absolute {
		return "", errors.New(codexGoCacheInvalid)
	}
	return absolute, nil
}

func ensurePrivateCodexGoCacheDirectory(path string) error {
	info, err := os.Lstat(path)
	created := false
	if errors.Is(err, fs.ErrNotExist) {
		if err := os.MkdirAll(path, 0o700); err != nil {
			return err
		}
		created = true
		info, err = os.Lstat(path)
	}
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New(codexGoCacheInvalid)
	}
	if created {
		if err := os.Chmod(path, 0o700); err != nil {
			return err
		}
		info, err = os.Lstat(path)
		if err != nil {
			return err
		}
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o700 {
		return errors.New(codexGoCacheInvalid)
	}
	return nil
}

func validateCodexGoToolchain(configuredRoot string) (string, error) {
	if configuredRoot == "" ||
		strings.TrimSpace(configuredRoot) != configuredRoot ||
		strings.ContainsRune(configuredRoot, '\x00') ||
		!filepath.IsAbs(configuredRoot) ||
		filepath.Clean(configuredRoot) != configuredRoot {
		return "", errors.New(codexGoToolchainInvalid)
	}
	resolved, err := filepath.EvalSymlinks(configuredRoot)
	if err != nil || resolved != configuredRoot {
		return "", errors.New(codexGoToolchainInvalid)
	}
	rootInfo, err := os.Lstat(configuredRoot)
	if err != nil || rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() ||
		unsafeCodexGoPermissions(rootInfo.Mode()) {
		return "", errors.New(codexGoToolchainInvalid)
	}
	binPath := filepath.Join(configuredRoot, "bin")
	binResolved, err := filepath.EvalSymlinks(binPath)
	if err != nil || binResolved != binPath {
		return "", errors.New(codexGoToolchainInvalid)
	}
	binInfo, err := os.Lstat(binPath)
	if err != nil || binInfo.Mode()&os.ModeSymlink != 0 || !binInfo.IsDir() ||
		unsafeCodexGoPermissions(binInfo.Mode()) {
		return "", errors.New(codexGoToolchainInvalid)
	}
	executablePath := filepath.Join(binPath, goExecutableName())
	executableResolved, err := filepath.EvalSymlinks(executablePath)
	if err != nil || executableResolved != executablePath {
		return "", errors.New(codexGoToolchainInvalid)
	}
	executableInfo, err := os.Lstat(executablePath)
	if err != nil || executableInfo.Mode()&os.ModeSymlink != 0 || !executableInfo.Mode().IsRegular() ||
		!codexGoExecutable(executableInfo.Mode()) ||
		unsafeCodexGoPermissions(executableInfo.Mode()) {
		return "", errors.New(codexGoToolchainInvalid)
	}
	if err := validateCodexGoToolchainTree(configuredRoot); err != nil {
		return "", errors.New(codexGoToolchainInvalid)
	}
	return configuredRoot, nil
}

func validateCodexGoToolchainTree(root string) error {
	return filepath.WalkDir(root, func(_ string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New(codexGoToolchainInvalid)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.IsDir() && !info.Mode().IsRegular() {
			return errors.New(codexGoToolchainInvalid)
		}
		if unsafeCodexGoPermissions(info.Mode()) {
			return errors.New(codexGoToolchainInvalid)
		}
		return nil
	})
}

func unsafeCodexGoPermissions(mode os.FileMode) bool {
	return runtime.GOOS != "windows" && mode.Perm()&0o022 != 0
}

func codexGoExecutable(mode os.FileMode) bool {
	return runtime.GOOS == "windows" || mode.Perm()&0o111 != 0
}

func goExecutableName() string {
	if runtime.GOOS == "windows" {
		return "go.exe"
	}
	return "go"
}
