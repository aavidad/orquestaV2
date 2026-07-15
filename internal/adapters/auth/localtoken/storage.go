package localtoken

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"orquesta/internal/identity"
)

const (
	// AuthenticationMethod is the canonical persisted identifier for this
	// adapter. Compositions use it instead of duplicating a string literal.
	AuthenticationMethod = "local_token"
	tokenByteSize        = 32
)

type Authenticator struct {
	token        string
	digest       [sha256.Size]byte
	principal    identity.Principal
	hasPrincipal bool
}

// Open loads the existing local token or creates it once with private
// permissions. Invalid existing state is never replaced silently.
func Open(configuredPath string) (*Authenticator, error) {
	tokenPath, err := normalizePath(configuredPath)
	if err != nil {
		return nil, err
	}
	directory := filepath.Dir(tokenPath)
	if err := ensurePrivateDirectory(directory); err != nil {
		return nil, err
	}

	token, found, err := loadToken(tokenPath)
	if err != nil {
		return nil, err
	}
	if !found {
		token, err = createToken(tokenPath, directory)
		if err != nil {
			return nil, err
		}
	}
	return &Authenticator{token: token, digest: sha256.Sum256([]byte(token))}, nil
}

// Token returns the credential for explicit local provisioning. Callers must
// not log it or place it in non-secret configuration.
func (authenticator *Authenticator) Token() string {
	if authenticator == nil {
		return ""
	}
	return authenticator.token
}

// ForPrincipal returns an immutable request authenticator for one exact local
// principal. Open deliberately returns credential storage without identity so
// bootstrap must compose both authorities explicitly before serving traffic.
func (authenticator *Authenticator) ForPrincipal(principal identity.Principal) (*Authenticator, error) {
	if authenticator == nil || authenticator.token == "" {
		return nil, &Error{Code: CodePrincipalInvalid}
	}
	if err := identity.ValidatePrincipal(principal); err != nil || principal.Method != AuthenticationMethod {
		return nil, &Error{Code: CodePrincipalInvalid, Cause: err}
	}
	if authenticator.hasPrincipal {
		return nil, &Error{Code: CodePrincipalInvalid}
	}
	return &Authenticator{
		token:        authenticator.token,
		digest:       authenticator.digest,
		principal:    principal,
		hasPrincipal: true,
	}, nil
}

func normalizePath(configuredPath string) (string, error) {
	if configuredPath == "" || strings.TrimSpace(configuredPath) != configuredPath || strings.ContainsRune(configuredPath, '\x00') {
		return "", &Error{Code: CodePathInvalid}
	}
	absolute, err := filepath.Abs(filepath.Clean(configuredPath))
	if err != nil || filepath.Base(absolute) == "." || filepath.Base(absolute) == string(filepath.Separator) {
		return "", &Error{Code: CodePathInvalid, Cause: err}
	}
	return absolute, nil
}

func ensurePrivateDirectory(directory string) error {
	_, initialErr := os.Lstat(directory)
	created := errors.Is(initialErr, fs.ErrNotExist)
	if initialErr != nil && !created {
		return &Error{Code: CodeDirectoryInvalid, Cause: initialErr}
	}
	if created {
		if err := validateExistingDirectoryChain(directory); err != nil {
			return err
		}
		if err := createPrivateDirectoryChain(directory); err != nil {
			return &Error{Code: CodeDirectoryInvalid, Cause: err}
		}
	}
	if err := validateDirectoryChain(directory); err != nil {
		return err
	}
	if created {
		if err := os.Chmod(directory, 0o700); err != nil {
			return &Error{Code: CodeDirectoryPermissions, Cause: err}
		}
	}
	info, err := os.Lstat(directory)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return &Error{Code: CodeDirectoryInvalid, Cause: err}
	}
	if info.Mode().Perm() != 0o700 {
		return &Error{Code: CodeDirectoryPermissions}
	}
	return nil
}

func createPrivateDirectoryChain(directory string) error {
	missing := make([]string, 0)
	for current := filepath.Clean(directory); ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
				return &Error{Code: CodeDirectoryInvalid}
			}
			break
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return &Error{Code: CodeDirectoryInvalid, Cause: err}
		}
		missing = append(missing, current)
		parent := filepath.Dir(current)
		if parent == current {
			return &Error{Code: CodeDirectoryInvalid}
		}
	}
	for index := len(missing) - 1; index >= 0; index-- {
		current := missing[index]
		if err := os.Mkdir(current, 0o700); err != nil {
			return &Error{Code: CodeDirectoryInvalid, Cause: err}
		}
		if err := syncDirectory(filepath.Dir(current)); err != nil {
			return &Error{Code: CodeDirectoryInvalid, Cause: err}
		}
	}
	return nil
}

func validateExistingDirectoryChain(directory string) error {
	for current := filepath.Clean(directory); ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		switch {
		case err == nil:
			if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
				return &Error{Code: CodeDirectoryInvalid}
			}
			return validateDirectoryChain(current)
		case !errors.Is(err, fs.ErrNotExist):
			return &Error{Code: CodeDirectoryInvalid, Cause: err}
		}
		parent := filepath.Dir(current)
		if parent == current {
			return &Error{Code: CodeDirectoryInvalid}
		}
	}
}

func validateDirectoryChain(directory string) error {
	for current := filepath.Clean(directory); ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return &Error{Code: CodeDirectoryInvalid, Cause: err}
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
	}
}

func loadToken(tokenPath string) (string, bool, error) {
	info, err := os.Lstat(tokenPath)
	if errors.Is(err, fs.ErrNotExist) {
		return "", false, nil
	}
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", false, &Error{Code: CodeTokenFileInvalid, Cause: err}
	}
	if info.Mode().Perm() != 0o600 {
		return "", false, &Error{Code: CodeTokenFilePermissions}
	}

	file, err := os.Open(tokenPath)
	if err != nil {
		return "", false, &Error{Code: CodeTokenFileInvalid, Cause: err}
	}
	defer file.Close()
	openedInfo, err := file.Stat()
	if err != nil || !openedInfo.Mode().IsRegular() || !os.SameFile(info, openedInfo) {
		return "", false, &Error{Code: CodeTokenFileInvalid, Cause: err}
	}
	if openedInfo.Mode().Perm() != 0o600 {
		return "", false, &Error{Code: CodeTokenFilePermissions}
	}

	maximum := base64.RawURLEncoding.EncodedLen(tokenByteSize)
	content, err := io.ReadAll(io.LimitReader(file, int64(maximum+1)))
	if err != nil {
		return "", false, &Error{Code: CodeTokenFileInvalid, Cause: err}
	}
	if len(content) != maximum {
		return "", false, &Error{Code: CodeTokenInvalid}
	}
	token := string(content)
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(decoded) != tokenByteSize || base64.RawURLEncoding.EncodeToString(decoded) != token {
		return "", false, &Error{Code: CodeTokenInvalid, Cause: err}
	}
	return token, true, nil
}

func createToken(tokenPath, directory string) (string, error) {
	random := make([]byte, tokenByteSize)
	if _, err := rand.Read(random); err != nil {
		return "", &Error{Code: CodeRandomFailed, Cause: err}
	}
	token := base64.RawURLEncoding.EncodeToString(random)

	temporary, err := os.CreateTemp(directory, ".local-token-*")
	if err != nil {
		return "", &Error{Code: CodePersistFailed, Cause: err}
	}
	temporaryPath := temporary.Name()
	defer func() {
		_ = temporary.Close()
		if temporaryPath != "" {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return "", &Error{Code: CodePersistFailed, Cause: err}
	}
	if _, err := io.WriteString(temporary, token); err != nil {
		return "", &Error{Code: CodePersistFailed, Cause: err}
	}
	if err := temporary.Sync(); err != nil {
		return "", &Error{Code: CodePersistFailed, Cause: err}
	}
	if err := temporary.Close(); err != nil {
		return "", &Error{Code: CodePersistFailed, Cause: err}
	}

	if err := os.Link(temporaryPath, tokenPath); err != nil {
		if errors.Is(err, fs.ErrExist) {
			existing, found, loadErr := loadToken(tokenPath)
			if loadErr != nil {
				return "", loadErr
			}
			if found {
				return existing, nil
			}
		}
		return "", &Error{Code: CodePersistFailed, Cause: err}
	}
	if err := os.Remove(temporaryPath); err != nil {
		return "", &Error{Code: CodePersistFailed, Cause: err}
	}
	temporaryPath = ""
	if err := syncDirectory(directory); err != nil {
		return "", &Error{Code: CodePersistFailed, Cause: err}
	}

	stored, found, err := loadToken(tokenPath)
	if err != nil {
		return "", err
	}
	if !found || stored != token {
		return "", &Error{Code: CodeTokenInvalid}
	}
	return stored, nil
}

func syncDirectory(directory string) error {
	file, err := os.Open(directory)
	if err != nil {
		return err
	}
	defer file.Close()
	return file.Sync()
}
