package orquestaruntimerequiredtest

import (
	"fmt"
	"path/filepath"
	"strings"
)

func splitCommandV0(command string) ([]string, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return nil, fmt.Errorf("required_test_command_required")
	}
	tokens := make([]string, 0)
	var current strings.Builder
	var quote rune
	escaped := false
	flush := func() {
		if current.Len() == 0 {
			return
		}
		tokens = append(tokens, current.String())
		current.Reset()
	}
	for _, r := range command {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
				continue
			}
			current.WriteRune(r)
			continue
		}
		switch r {
		case '\'', '"':
			quote = r
		case ' ', '\t':
			flush()
		case ';', '&', '|', '<', '>', '`':
			return nil, fmt.Errorf("required_test_command_shell_syntax_prohibited")
		default:
			current.WriteRune(r)
		}
	}
	if escaped || quote != 0 {
		return nil, fmt.Errorf("required_test_command_unclosed_token")
	}
	flush()
	if len(tokens) == 0 {
		return nil, fmt.Errorf("required_test_command_required")
	}
	return tokens, nil
}

func commandIsShellV0(command string) bool {
	base := strings.ToLower(filepath.Base(strings.TrimSpace(command)))
	switch base {
	case "sh", "bash", "dash", "zsh", "fish", "cmd", "cmd.exe", "powershell", "powershell.exe", "pwsh", "pwsh.exe":
		return true
	default:
		return false
	}
}

func envEntryAllowedV0(item string) bool {
	key, value, ok := strings.Cut(item, "=")
	if !ok ||
		strings.TrimSpace(key) == "" ||
		envKeyProhibitedV0(key) ||
		pathHasCredentialMarkerV0(key) ||
		pathHasCredentialMarkerV0(value) {
		return false
	}
	if strings.ContainsAny(value, `/\`) {
		return envPathValueAllowedV0(key, value)
	}
	return true
}

func envKeyProhibitedV0(key string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(key))
	return strings.Contains(normalized, "HOME") ||
		strings.Contains(normalized, "USERPROFILE")
}

func envPathValueAllowedV0(key string, value string) bool {
	switch strings.ToUpper(strings.TrimSpace(key)) {
	case "PATH":
		return envPathListAllowedV0(value)
	case "GOCACHE", "GOMODCACHE", "GOPATH", "GOTMPDIR", "TMPDIR", "ORQUESTA_ISOLATED_TEST_MODULE_CACHE_SEED":
		return filepath.IsAbs(value) && !pathHasCredentialMarkerV0(value)
	default:
		return false
	}
}

func envPathListAllowedV0(value string) bool {
	if strings.TrimSpace(value) == "" {
		return false
	}
	for _, item := range filepath.SplitList(value) {
		if item == "" || !filepath.IsAbs(item) || pathHasCredentialMarkerV0(item) {
			return false
		}
	}
	return true
}

func pathHasCredentialMarkerV0(value string) bool {
	low := strings.ToLower(strings.TrimSpace(value))
	for _, marker := range []string{"$home", "${home}", "%userprofile%", "oauth", "token", "secret", "api_key", "credential", "password"} {
		if strings.Contains(low, marker) {
			return true
		}
	}
	return false
}
