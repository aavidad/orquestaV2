package codex

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"
)

const (
	accountProfileRefPrefix = "account-profile:v1:sha256:"
	accountAuthFileName     = "auth.json"
)

func (adapter *Adapter) accountProfileRef() string {
	if adapter == nil {
		return ""
	}
	return adapter.accountProfileBindingRef
}

func validAccountProfileRef(value string) bool {
	encoded := strings.TrimPrefix(value, accountProfileRefPrefix)
	if len(encoded) != sha256.Size*2 || accountProfileRefPrefix+encoded != value ||
		encoded != strings.ToLower(encoded) {
		return false
	}
	decoded, err := hex.DecodeString(encoded)
	return err == nil && len(decoded) == sha256.Size
}

func deriveAccountProfileRef(accountHomeRoot, accountProfile string) (string, error) {
	if accountHomeRoot == "" && accountProfile == "" {
		return "", nil
	}
	canonicalRoot, err := filepath.EvalSymlinks(accountHomeRoot)
	if err != nil {
		return "", &Error{Code: CodeAccountProfileUnavailable, Cause: err}
	}
	canonicalRoot, err = filepath.Abs(filepath.Clean(canonicalRoot))
	if err != nil {
		return "", &Error{Code: CodeAccountProfileUnavailable, Cause: err}
	}
	digest := sha256.New()
	_, _ = digest.Write([]byte("orquesta.codex.account-profile.v1"))
	_, _ = digest.Write([]byte{0})
	_, _ = digest.Write([]byte(canonicalRoot))
	_, _ = digest.Write([]byte{0})
	_, _ = digest.Write([]byte(accountProfile))
	return accountProfileRefPrefix + hex.EncodeToString(digest.Sum(nil)), nil
}

func (adapter *Adapter) validateAccountProfileBinding(record launchRecord) error {
	if record.AccountProfileRef == adapter.accountProfileRef() {
		return nil
	}
	// A durable binding never falls back to the daemon HOME and cannot be
	// silently rebound to another configured account.
	return &Error{Code: CodeAccountProfileUnavailable}
}

func (adapter *Adapter) accountExecutionEnvironment(base []string) ([]string, error) {
	if adapter.accountHomePath == "" {
		return append([]string(nil), base...), nil
	}
	values := make(map[string]string, len(base)+2)
	for _, entry := range base {
		name, value, found := strings.Cut(entry, "=")
		if !found || name == "" {
			return nil, &Error{Code: CodeEnvironmentInvalid}
		}
		if _, duplicate := values[name]; duplicate {
			return nil, &Error{Code: CodeEnvironmentInvalid}
		}
		values[name] = value
	}
	values["HOME"] = adapter.accountHomePath
	values["CODEX_HOME"] = adapter.accountHomePath
	return exactEnvironment(values)
}
