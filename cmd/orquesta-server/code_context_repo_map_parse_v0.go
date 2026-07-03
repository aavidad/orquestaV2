package main

import (
	"path/filepath"
	"strconv"
	"strings"

	orquestacontext "orquesta/modulos/orquesta-context"
)

func serverRGRepoMapContextHitsV0(output string, query orquestacontext.CodeContextQueryV0) []orquestacontext.CodeContextHitV0 {
	lines := strings.Split(output, "\n")
	candidates := make([]orquestacontext.CodeContextHitV0, 0, len(lines))
	for _, line := range lines {
		path, lineNo, snippet, ok := parseRGCodeContextLineV0(line)
		if !ok {
			continue
		}
		kind, symbol, summary, ok := parseServerRepoMapDeclarationV0(snippet)
		if !ok {
			continue
		}
		candidates = append(candidates, orquestacontext.CodeContextHitV0{
			Kind:    kind,
			Path:    filepath.ToSlash(filepath.Clean(path)),
			Line:    lineNo,
			Symbol:  symbol,
			Summary: summary,
			Snippet: serverGoCodeContextSnippetV0(snippet),
			Score:   1,
		})
	}
	return serverRepoMapSelectHitsV0(candidates, query)
}

func serverRepoMapSelectHitsV0(
	candidates []orquestacontext.CodeContextHitV0,
	query orquestacontext.CodeContextQueryV0,
) []orquestacontext.CodeContextHitV0 {
	maxResults := query.MaxResults
	if maxResults <= 0 {
		maxResults = 8
	}
	tokens := serverRepoMapQueryTokensV0(query.Query)
	matches := make([]orquestacontext.CodeContextHitV0, 0, maxResults)
	fallback := make([]orquestacontext.CodeContextHitV0, 0, maxResults)
	seen := map[string]bool{}
	for _, hit := range candidates {
		key := hit.Path + ":" + strconv.Itoa(hit.Line) + ":" + hit.Symbol
		if seen[key] {
			continue
		}
		seen[key] = true
		if len(fallback) < maxResults {
			fallback = append(fallback, hit)
		}
		if serverRepoMapHitMatchesTokensV0(hit, tokens) && len(matches) < maxResults {
			matches = append(matches, hit)
		}
	}
	if len(matches) > 0 {
		return serverRepoMapNormalizeHitRefsV0(matches)
	}
	return serverRepoMapNormalizeHitRefsV0(fallback)
}

func parseServerRepoMapDeclarationV0(line string) (string, string, string, bool) {
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "func ") {
		tail := strings.TrimSpace(strings.TrimPrefix(line, "func "))
		receiver := ""
		if strings.HasPrefix(tail, "(") {
			if closeIdx := strings.Index(tail, ")"); closeIdx >= 0 {
				receiver = serverRepoMapReceiverTypeV0(tail[1:closeIdx])
				tail = strings.TrimSpace(tail[closeIdx+1:])
			}
		}
		symbol := serverRepoMapLeadingIdentifierV0(tail)
		if symbol == "" {
			return "", "", "", false
		}
		if receiver != "" {
			return "function", symbol, "metodo " + receiver + "." + symbol + " en mapa compacto", true
		}
		return "function", symbol, "funcion " + symbol + " en mapa compacto", true
	}
	if strings.HasPrefix(line, "type ") {
		symbol := serverRepoMapLeadingIdentifierV0(strings.TrimSpace(strings.TrimPrefix(line, "type ")))
		if symbol == "" {
			return "", "", "", false
		}
		return "type", symbol, "tipo " + symbol + " en mapa compacto", true
	}
	return "", "", "", false
}

func serverRepoMapReceiverTypeV0(receiver string) string {
	fields := strings.Fields(strings.TrimSpace(receiver))
	if len(fields) == 0 {
		return ""
	}
	value := fields[len(fields)-1]
	value = strings.TrimLeft(value, "*[]")
	value = strings.TrimSpace(value)
	if idx := strings.LastIndex(value, "."); idx >= 0 {
		value = value[idx+1:]
	}
	return serverRepoMapLeadingIdentifierV0(value)
}

func serverRepoMapLeadingIdentifierV0(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	end := 0
	for end < len(value) {
		ch := value[end]
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' {
			end++
			continue
		}
		break
	}
	return strings.TrimSpace(value[:end])
}

func serverRepoMapQueryTokensV0(query string) []string {
	normalized := strings.NewReplacer("/", " ", "\\", " ", "-", " ", "_", " ", ".", " ", ":", " ", ",", " ").Replace(strings.TrimSpace(query))
	raw := strings.Fields(normalized)
	tokens := make([]string, 0, len(raw))
	for _, token := range raw {
		token = strings.TrimSpace(token)
		switch strings.ToLower(token) {
		case "", "*", "repo", "map", "repo_map", "codigo", "code":
			continue
		default:
			tokens = append(tokens, token)
		}
	}
	return tokens
}

func serverRepoMapHitMatchesTokensV0(hit orquestacontext.CodeContextHitV0, tokens []string) bool {
	if len(tokens) == 0 {
		return true
	}
	haystack := strings.Join([]string{hit.Path, hit.Symbol, hit.Summary, hit.Snippet}, "\n")
	for _, token := range tokens {
		if serverCodeContextSmartCaseMatchV0(haystack, token) {
			return true
		}
	}
	return false
}

func serverRepoMapNormalizeHitRefsV0(
	hits []orquestacontext.CodeContextHitV0,
) []orquestacontext.CodeContextHitV0 {
	for idx := range hits {
		hits[idx].HitRef = "code-context-repo-map-hit-" + strconv.Itoa(idx+1)
	}
	return hits
}
