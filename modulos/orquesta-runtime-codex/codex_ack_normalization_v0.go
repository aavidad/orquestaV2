package orquestaruntimecodex

import (
	"sort"
	"strings"
)

func codexSchemaVersionCompatibleV0(value string, current string) bool {
	return CodexSchemaVersionCompatibleV0(value, current)
}

func CodexSchemaVersionCompatibleV0(value string, current string) bool {
	value = strings.TrimSpace(value)
	current = strings.TrimSpace(current)
	if value == current {
		return true
	}
	if current == "" || !strings.HasSuffix(current, ".v0") {
		return false
	}
	if !strings.HasPrefix(value, current+".") {
		return false
	}
	suffix := strings.TrimPrefix(value, current+".")
	if suffix == "" {
		return false
	}
	segmentHasDigit := false
	for _, r := range suffix {
		switch {
		case r >= '0' && r <= '9':
			segmentHasDigit = true
		case r == '.':
			if !segmentHasDigit {
				return false
			}
			segmentHasDigit = false
		default:
			return false
		}
	}
	return segmentHasDigit
}

func canonicalTestCommandV0(value string) string {
	return CanonicalCodexTestCommandV0(value)
}

func CanonicalCodexTestCommandV0(value string) string {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return ""
	}
	prefixLen := codexTestCommandPrefixLenV0(fields)
	if prefixLen > len(fields) {
		prefixLen = len(fields)
	}
	prefix := append([]string(nil), fields[:prefixLen]...)
	flags := []string{}
	args := []string{}
	for i := prefixLen; i < len(fields); i++ {
		token := strings.TrimSpace(fields[i])
		if token == "" {
			continue
		}
		if strings.HasPrefix(token, "-") {
			if !strings.Contains(token, "=") &&
				codexTestCommandFlagUsuallyHasValueV0(token) &&
				i+1 < len(fields) &&
				!strings.HasPrefix(fields[i+1], "-") {
				flags = append(flags, token+"="+strings.TrimSpace(fields[i+1]))
				i++
				continue
			}
			flags = append(flags, token)
			continue
		}
		args = append(args, token)
	}
	sort.Strings(flags)
	out := append(prefix, flags...)
	out = append(out, args...)
	return strings.Join(out, " ")
}

func codexTestCommandPrefixLenV0(fields []string) int {
	if len(fields) >= 2 {
		first := strings.TrimSpace(fields[0])
		second := strings.TrimSpace(fields[1])
		if first == "go" && second == "test" {
			return 2
		}
		if (first == "npm" || first == "pnpm" || first == "yarn") &&
			(second == "test" || second == "run") {
			return 2
		}
	}
	return 1
}

func codexTestCommandFlagUsuallyHasValueV0(flag string) bool {
	switch strings.TrimSpace(flag) {
	case "-bench",
		"-benchtime",
		"-blockprofile",
		"-blockprofilerate",
		"-count",
		"-covermode",
		"-coverpkg",
		"-coverprofile",
		"-cpu",
		"-cpuprofile",
		"-exec",
		"-list",
		"-memprofile",
		"-memprofilerate",
		"-mutexprofile",
		"-mutexprofilefraction",
		"-o",
		"-outputdir",
		"-parallel",
		"-run",
		"-shuffle",
		"-tags",
		"-timeout",
		"-trace",
		"-vet":
		return true
	default:
		return false
	}
}

func codexTestCommandInSetV0(values []string, want string) bool {
	return CodexTestCommandInSetV0(values, want)
}

func CodexTestCommandInSetV0(values []string, want string) bool {
	want = canonicalTestCommandV0(want)
	if want == "" {
		return false
	}
	for _, value := range values {
		if canonicalTestCommandV0(value) == want {
			return true
		}
	}
	return false
}

func CodexTestCommandMatchesV0(got string, want string) bool {
	want = canonicalTestCommandV0(want)
	return want != "" && canonicalTestCommandV0(got) == want
}
