package orquestaruntimecodexappserver

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func (backend serverCodexAppServerTmuxBackendV0) prepareTmuxCodeHomeV0() error {
	codeHomeDir := strings.TrimSpace(backend.CodeHomeDir)
	if codeHomeDir == "" {
		return nil
	}
	if err := ensureCodexAppServerTmuxCodeHomeTargetSafeV0(codeHomeDir, backend.RuntimeWorkDir); err != nil {
		return err
	}
	allowedFiles, err := backend.readTmuxCodeHomeAllowedFilesV0(codeHomeDir)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(codeHomeDir); err != nil {
		return codexAppServerCallErrorV0{
			Code: "codex_app_server_tmux_code_home_unavailable",
			Err:  err,
		}
	}
	if err := os.MkdirAll(codeHomeDir, 0o700); err != nil {
		return codexAppServerCallErrorV0{
			Code: "codex_app_server_tmux_code_home_unavailable",
			Err:  err,
		}
	}
	for _, name := range codexAppServerTmuxCodeHomeAllowedFileNamesV0() {
		raw, ok := allowedFiles[name]
		if !ok {
			continue
		}
		if name == "config.toml" {
			raw = codexAppServerTmuxProjectScopedConfigV0(raw, backend.ProjectWorkDir)
		}
		if err := os.WriteFile(filepath.Join(codeHomeDir, name), raw, 0o600); err != nil {
			return codexAppServerCallErrorV0{
				Code: "codex_app_server_tmux_code_home_unavailable",
				Err:  err,
			}
		}
	}
	return nil
}

func (backend serverCodexAppServerTmuxBackendV0) readTmuxCodeHomeAllowedFilesV0(codeHomeDir string) (map[string][]byte, error) {
	allowedFiles := map[string][]byte{}
	sourceDir := strings.TrimSpace(backend.SourceCodeHomeDir)
	for _, name := range codexAppServerTmuxCodeHomeAllowedFileNamesV0() {
		if sourceDir != "" {
			raw, found, err := readCodexAppServerTmuxCodeHomeAllowedFileV0(sourceDir, name)
			if err != nil {
				return nil, err
			}
			if found {
				allowedFiles[name] = raw
				continue
			}
		}
		raw, found, err := readCodexAppServerTmuxCodeHomeAllowedFileV0(codeHomeDir, name)
		if err != nil {
			return nil, err
		}
		if found {
			allowedFiles[name] = raw
		}
	}
	return allowedFiles, nil
}

func codexAppServerTmuxProjectScopedConfigV0(raw []byte, projectWorkDir string) []byte {
	projectWorkDir = strings.TrimSpace(projectWorkDir)
	if projectWorkDir == "" {
		return raw
	}
	lines := strings.SplitAfter(string(raw), "\n")
	out := make([]string, 0, len(lines)+4)
	skipProjectSection := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			skipProjectSection = codexAppServerTmuxConfigSectionIsProjectV0(trimmed)
			if skipProjectSection {
				continue
			}
		}
		if skipProjectSection {
			continue
		}
		out = append(out, line)
	}
	config := strings.TrimRight(strings.Join(out, ""), "\n")
	if config != "" {
		config += "\n"
	}
	config += "\n[projects.\"" + codexAppServerTmuxTOMLStringPathV0(projectWorkDir) + "\"]\ntrust_level = \"trusted\"\n"
	return []byte(config)
}

func codexAppServerTmuxConfigSectionIsProjectV0(section string) bool {
	section = strings.TrimSpace(section)
	return strings.HasPrefix(section, "[projects.") || strings.HasPrefix(section, "[[projects.")
}

func codexAppServerTmuxTOMLStringPathV0(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "\\", "\\\\")
	path = strings.ReplaceAll(path, "\"", "\\\"")
	return path
}

func codexAppServerTmuxCodeHomeAllowedFileNamesV0() []string {
	return []string{"auth.json", "config.toml"}
}

func readCodexAppServerTmuxCodeHomeAllowedFileV0(dir string, name string) ([]byte, bool, error) {
	path := filepath.Join(strings.TrimSpace(dir), name)
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, codexAppServerCallErrorV0{
			Code: "codex_app_server_tmux_code_home_source_unavailable",
			Err:  err,
		}
	}
	if info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, false, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, false, codexAppServerCallErrorV0{
			Code: "codex_app_server_tmux_code_home_source_unavailable",
			Err:  err,
		}
	}
	return raw, true, nil
}

func ensureCodexAppServerTmuxCodeHomeTargetSafeV0(codeHomeDir string, runtimeWorkDir string) error {
	cleanTarget, err := filepath.Abs(strings.TrimSpace(codeHomeDir))
	if err != nil {
		return codexAppServerCallErrorV0{
			Code: "codex_app_server_tmux_code_home_unsafe",
			Err:  err,
		}
	}
	cleanTarget = filepath.Clean(cleanTarget)
	if filepath.Base(cleanTarget) != "codex-home" || filepath.Base(filepath.Dir(cleanTarget)) != codexAppServerTmuxDirV0 {
		return codexAppServerCallErrorV0{
			Code: "codex_app_server_tmux_code_home_unsafe",
			Err:  errors.New("codex_app_server_tmux_code_home_not_goal_srv_target"),
		}
	}
	if strings.TrimSpace(runtimeWorkDir) != "" {
		cleanRuntimeDir, err := filepath.Abs(strings.TrimSpace(runtimeWorkDir))
		if err != nil {
			return codexAppServerCallErrorV0{
				Code: "codex_app_server_tmux_code_home_unsafe",
				Err:  err,
			}
		}
		expectedTarget := filepath.Join(filepath.Clean(cleanRuntimeDir), codexAppServerTmuxDirV0, "codex-home")
		if filepath.Clean(expectedTarget) != cleanTarget {
			return codexAppServerCallErrorV0{
				Code: "codex_app_server_tmux_code_home_unsafe",
				Err:  errors.New("codex_app_server_tmux_code_home_outside_runtime_workdir"),
			}
		}
	}
	if err := ensureExistingPathHasNoSymlinkCodexAppServerTmuxV0(cleanTarget); err != nil {
		return err
	}
	parentDir := filepath.Dir(cleanTarget)
	if err := os.MkdirAll(parentDir, 0o700); err != nil {
		return codexAppServerCallErrorV0{
			Code: "codex_app_server_tmux_code_home_unavailable",
			Err:  err,
		}
	}
	if err := ensureExistingPathHasNoSymlinkCodexAppServerTmuxV0(cleanTarget); err != nil {
		return err
	}
	if info, err := os.Lstat(cleanTarget); err == nil && !info.IsDir() {
		return codexAppServerCallErrorV0{
			Code: "codex_app_server_tmux_code_home_unsafe",
			Err:  errors.New("codex_app_server_tmux_code_home_not_directory"),
		}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return codexAppServerCallErrorV0{
			Code: "codex_app_server_tmux_code_home_unavailable",
			Err:  err,
		}
	}
	return nil
}

func ensureExistingPathHasNoSymlinkCodexAppServerTmuxV0(path string) error {
	cleanPath := filepath.Clean(path)
	volumeName := filepath.VolumeName(cleanPath)
	rest := strings.TrimPrefix(cleanPath, volumeName)
	separator := string(os.PathSeparator)
	current := volumeName
	if strings.HasPrefix(rest, separator) {
		current += separator
		rest = strings.TrimPrefix(rest, separator)
	}
	for _, elem := range strings.Split(rest, separator) {
		if elem == "" || elem == "." {
			continue
		}
		if current == "" || current == separator || strings.HasSuffix(current, separator) {
			current = current + elem
		} else {
			current = filepath.Join(current, elem)
		}
		info, err := os.Lstat(current)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return codexAppServerCallErrorV0{
				Code: "codex_app_server_tmux_code_home_unavailable",
				Err:  err,
			}
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return codexAppServerCallErrorV0{
				Code: "codex_app_server_tmux_code_home_unsafe",
				Err:  errors.New("codex_app_server_tmux_code_home_symlink"),
			}
		}
	}
	return nil
}
