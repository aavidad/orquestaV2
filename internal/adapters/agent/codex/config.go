package codex

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"orquesta/internal/credentials"
)

const (
	codexAPIKeyEnvironment  = "CODEX_API_KEY"
	openAIAPIKeyEnvironment = "OPENAI_API_KEY"
)

func prepareConfig(config Config) (Config, string, []string, string, *os.Root, error) {
	if config.Command == "" || strings.TrimSpace(config.Command) != config.Command {
		return Config{}, "", nil, "", nil, &Error{Code: CodeCommandRequired}
	}
	if config.WorkRoot == "" || strings.TrimSpace(config.WorkRoot) != config.WorkRoot {
		return Config{}, "", nil, "", nil, &Error{Code: CodeWorkRootRequired}
	}
	if strings.TrimSpace(config.RuntimeScope) != config.RuntimeScope || strings.ContainsRune(config.RuntimeScope, '\x00') {
		return Config{}, "", nil, "", nil, &Error{Code: CodeRuntimeScopeInvalid}
	}
	if config.Model != "" && strings.TrimSpace(config.Model) != config.Model {
		return Config{}, "", nil, "", nil, &Error{Code: CodeCommandInvalid}
	}
	if !validReasoningEffort(config.ReasoningEffort) {
		return Config{}, "", nil, "", nil, &Error{Code: CodeReasoningEffortInvalid}
	}
	if config.Timeout <= 0 {
		return Config{}, "", nil, "", nil, &Error{Code: CodeTimeoutInvalid}
	}
	if config.ProcessPipeDrainDelay <= 0 {
		return Config{}, "", nil, "", nil, &Error{Code: CodeProcessPipeDrainInvalid}
	}
	if config.MaxDiagnosticBytes <= 0 {
		return Config{}, "", nil, "", nil, &Error{Code: CodeDiagnosticLimitInvalid}
	}
	if config.MaxConcurrentExecutions <= 0 {
		return Config{}, "", nil, "", nil, &Error{Code: CodeMaxConcurrentInvalid}
	}
	if config.PromptRenderer == nil {
		return Config{}, "", nil, "", nil, &Error{Code: CodePromptRendererInvalid}
	}
	if config.Now == nil || config.Now().IsZero() {
		return Config{}, "", nil, "", nil, &Error{Code: CodeClockInvalid}
	}
	if !validPrivateEnvironmentName(config.MCPBearerTokenEnvVar) {
		return Config{}, "", nil, "", nil, &Error{Code: CodeEnvironmentInvalid}
	}
	if _, found := config.Environment[codexAPIKeyEnvironment]; found {
		return Config{}, "", nil, "", nil, &Error{Code: CodeEnvironmentInvalid}
	}
	if _, found := config.Environment[openAIAPIKeyEnvironment]; found {
		return Config{}, "", nil, "", nil, &Error{Code: CodeEnvironmentInvalid}
	}
	// This bearer is injected only for the short-lived Codex core process by
	// SessionResolver. A public composition environment must never shadow it,
	// otherwise it could be projected into the Codex tool-shell policy.
	if _, found := config.Environment[config.MCPBearerTokenEnvVar]; found {
		return Config{}, "", nil, "", nil, &Error{Code: CodeEnvironmentInvalid}
	}
	if (config.CredentialStore == nil) != (config.CredentialRef == "") {
		return Config{}, "", nil, "", nil, &Error{Code: CodeCredentialInvalid}
	}
	if config.CredentialRef != "" {
		if err := credentials.ValidateCredentialRef(config.CredentialRef); err != nil {
			return Config{}, "", nil, "", nil, &Error{Code: CodeCredentialInvalid, Cause: err}
		}
	}

	environment, err := exactEnvironment(config.Environment)
	if err != nil {
		return Config{}, "", nil, "", nil, err
	}
	command, err := resolveCommand(config.Command, config.Environment)
	if err != nil {
		return Config{}, "", nil, "", nil, err
	}
	rootPath, root, err := openPrivateRoot(config.WorkRoot)
	if err != nil {
		return Config{}, "", nil, "", nil, err
	}
	config.Environment = cloneEnvironment(config.Environment)
	return config, command, environment, rootPath, root, nil
}

func validPrivateEnvironmentName(name string) bool {
	if name == "" || name == codexAPIKeyEnvironment || name == openAIAPIKeyEnvironment || name[0] < 'A' || name[0] > 'Z' {
		return false
	}
	for _, character := range name {
		if (character < 'A' || character > 'Z') && (character < '0' || character > '9') && character != '_' {
			return false
		}
	}
	return true
}

func validReasoningEffort(value string) bool {
	switch value {
	case "low", "medium", "high", "xhigh", "ultra":
		return true
	default:
		return false
	}
}

func exactEnvironment(environment map[string]string) ([]string, error) {
	keys := make([]string, 0, len(environment))
	for key, value := range environment {
		if key == "" || strings.ContainsAny(key, "=\x00") || strings.ContainsRune(value, '\x00') {
			return nil, &Error{Code: CodeEnvironmentInvalid}
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, key+"="+environment[key])
	}
	if result == nil {
		result = []string{}
	}
	return result, nil
}

func cloneEnvironment(environment map[string]string) map[string]string {
	result := make(map[string]string, len(environment))
	for key, value := range environment {
		result[key] = value
	}
	return result
}

func resolveCommand(command string, environment map[string]string) (string, error) {
	if filepath.IsAbs(command) {
		return validateExecutable(command)
	}
	if strings.ContainsRune(command, filepath.Separator) {
		return "", &Error{Code: CodeCommandInvalid}
	}
	searchPath, found := environment["PATH"]
	if !found || searchPath == "" {
		return "", &Error{Code: CodeCommandNotFound}
	}
	for _, directory := range filepath.SplitList(searchPath) {
		if directory == "" || !filepath.IsAbs(directory) {
			continue
		}
		candidate := filepath.Join(directory, command)
		if resolved, err := validateExecutable(candidate); err == nil {
			return resolved, nil
		}
	}
	return "", &Error{Code: CodeCommandNotFound}
}

func validateExecutable(command string) (string, error) {
	info, err := os.Stat(command)
	if err != nil {
		return "", &Error{Code: CodeCommandNotFound, Cause: err}
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return "", &Error{Code: CodeCommandInvalid}
	}
	return command, nil
}

func openPrivateRoot(configuredPath string) (string, *os.Root, error) {
	rootPath, err := filepath.Abs(configuredPath)
	if err != nil {
		return "", nil, &Error{Code: CodeWorkRootInvalid, Cause: err}
	}
	if err := ensureWorkRootPath(rootPath, syncFilesystemDirectory); err != nil {
		return "", nil, err
	}
	info, err := os.Lstat(rootPath)
	if err != nil {
		return "", nil, &Error{Code: CodeWorkRootInvalid, Cause: err}
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", nil, &Error{Code: CodeWorkRootInvalid}
	}
	if info.Mode().Perm()&0o077 != 0 {
		return "", nil, &Error{Code: CodeWorkRootPermissions}
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return "", nil, &Error{Code: CodeWorkRootOpenFailed, Cause: err}
	}
	return rootPath, root, nil
}

func ensureWorkRootPath(directory string, syncDirectory func(string) error) error {
	info, err := os.Lstat(directory)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return &Error{Code: CodeWorkRootInvalid}
		}
		return nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return &Error{Code: CodeWorkRootInvalid, Cause: err}
	}
	parent := filepath.Dir(directory)
	if parent == directory {
		return &Error{Code: CodeWorkRootInvalid}
	}
	if err := ensureWorkRootPath(parent, syncDirectory); err != nil {
		return err
	}
	if err := os.Mkdir(directory, 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
		return &Error{Code: CodeWorkRootCreateFailed, Cause: err}
	}
	info, err = os.Lstat(directory)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return &Error{Code: CodeWorkRootInvalid, Cause: err}
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return &Error{Code: CodeWorkRootCreateFailed, Cause: err}
	}
	if syncDirectory == nil {
		return &Error{Code: CodeWorkRootCreateFailed}
	}
	if err := syncDirectory(parent); err != nil {
		return &Error{Code: CodeWorkRootCreateFailed, Cause: err}
	}
	return nil
}

func syncFilesystemDirectory(directory string) error {
	handle, err := os.Open(directory)
	if err != nil {
		return err
	}
	defer handle.Close()
	return handle.Sync()
}
