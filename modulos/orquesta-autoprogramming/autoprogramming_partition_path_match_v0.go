package orquestaautoprogramming

import (
	"strings"
	"unicode"
)

func autoprogrammingWriteSetMatchingAreasV0(
	path string,
	groups []AutoprogrammingTaskGroupV0,
	aliases []AutoprogrammingAreaAliasV0,
) []string {
	pathTokens := autoprogrammingCanonicalPathTokensV0(path)
	pathSegments := autoprogrammingCanonicalPathSegmentKeysV0(path)
	var matches []string
	for _, group := range groups {
		for _, areaTokens := range autoprogrammingAreaTokenSetsV0(group.Area, aliases) {
			if autoprogrammingPathTokensContainAreaV0(pathTokens, areaTokens) ||
				autoprogrammingPathSegmentsContainAreaV0(pathSegments, areaTokens) {
				matches = append(matches, group.Area)
				break
			}
		}
	}
	return matches
}

func autoprogrammingAreaTokenSetsV0(area string, aliases []AutoprogrammingAreaAliasV0) [][]string {
	normalizedArea := normalizeAutoprogrammingTaskAreaV0(area)
	out := [][]string{autoprogrammingCanonicalPathTokensV0(normalizedArea)}
	for _, alias := range aliases {
		if normalizeAutoprogrammingTaskAreaV0(alias.Area) != normalizedArea {
			continue
		}
		out = append(out, autoprogrammingCanonicalPathTokensV0(alias.Alias))
	}
	return out
}

func autoprogrammingCanonicalPathTokensV0(value string) []string {
	tokens := autoprogrammingNormalizedPathTokensV0(normalizeAutoprogrammingPathForMatchV0(value))
	for i, token := range tokens {
		tokens[i] = autoprogrammingCanonicalPathTokenV0(token)
	}
	return compactPathTokensV0(tokens)
}

func autoprogrammingCanonicalPathTokenV0(token string) string {
	switch token {
	case "application", "applications", "apps":
		return "app"
	case "services", "svc":
		return "service"
	}
	if len(token) > 3 && strings.HasSuffix(token, "s") && !strings.HasSuffix(token, "ss") {
		return strings.TrimSuffix(token, "s")
	}
	return token
}

func autoprogrammingNormalizedPathTokensV0(value string) []string {
	return compactPathTokensV0(strings.Split(value, "-"))
}

func compactPathTokensV0(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func autoprogrammingPathTokensContainAreaV0(pathTokens []string, areaTokens []string) bool {
	if len(pathTokens) == 0 || len(areaTokens) == 0 || len(areaTokens) > len(pathTokens) {
		return false
	}
	for start := 0; start <= len(pathTokens)-len(areaTokens); start++ {
		matched := true
		for offset := range areaTokens {
			if pathTokens[start+offset] != areaTokens[offset] {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

func autoprogrammingPathSegmentsContainAreaV0(pathSegments []string, areaTokens []string) bool {
	areaKey := strings.Join(areaTokens, "")
	if areaKey == "" {
		return false
	}
	for _, segment := range pathSegments {
		if segment == areaKey {
			return true
		}
	}
	return false
}

func autoprogrammingCanonicalPathSegmentKeysV0(path string) []string {
	rawSegments := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(path)), func(r rune) bool {
		return r == '/' || r == '\\'
	})
	segments := make([]string, 0, len(rawSegments))
	for _, segment := range rawSegments {
		key := strings.Join(autoprogrammingCanonicalPathTokensV0(segment), "")
		if key != "" {
			segments = appendUniqueStringV0(segments, key)
		}
	}
	return segments
}

func normalizeAutoprogrammingPathForMatchV0(value string) string {
	return strings.Trim(strings.Map(func(r rune) rune {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			return unicode.ToLower(r)
		default:
			return '-'
		}
	}, value), "-")
}
