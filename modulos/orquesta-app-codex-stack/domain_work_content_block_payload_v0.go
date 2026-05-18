package orquestaappcodexstack

import (
	"encoding/json"
	"strings"
)

func canonicalDomainWorkContentBlockPayloadJSONV0(body string) (string, bool) {
	body = strings.TrimSpace(body)
	if body == "" {
		return "", false
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &payload); err != nil || len(payload) == 0 {
		return "", false
	}
	rawRefs, ok := payload["source_refs"]
	if !ok || len(rawRefs) == 0 {
		return body, true
	}
	if sourceRefsAlreadyCompactV0(rawRefs) {
		return body, true
	}
	refs, ok := compactSourceRefsFromRichPayloadV0(rawRefs)
	if !ok {
		return body, true
	}
	encodedRefs, err := json.Marshal(refs)
	if err != nil {
		return body, true
	}
	payload["source_refs"] = encodedRefs
	if _, exists := payload["source_ref_details"]; !exists {
		payload["source_ref_details"] = append([]byte(nil), rawRefs...)
	}
	canonical, err := json.Marshal(payload)
	if err != nil {
		return body, true
	}
	return string(canonical), true
}

func sourceRefsAlreadyCompactV0(raw json.RawMessage) bool {
	var refs []string
	return json.Unmarshal(raw, &refs) == nil
}

func compactSourceRefsFromRichPayloadV0(raw json.RawMessage) ([]string, bool) {
	var entries []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil || len(entries) == 0 {
		return nil, false
	}
	refs := make([]string, 0, len(entries))
	seen := map[string]struct{}{}
	for _, entry := range entries {
		ref := sourceRefFromRichEntryV0(entry)
		if ref == "" {
			continue
		}
		if _, exists := seen[ref]; exists {
			continue
		}
		seen[ref] = struct{}{}
		refs = append(refs, ref)
	}
	return refs, len(refs) > 0
}

func sourceRefFromRichEntryV0(entry map[string]json.RawMessage) string {
	for _, key := range []string{"source_ref", "ref", "id"} {
		raw, ok := entry[key]
		if !ok {
			continue
		}
		var value string
		if err := json.Unmarshal(raw, &value); err == nil {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
