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
	changed := false
	if rawRefs, ok := payload["source_refs"]; ok && len(rawRefs) > 0 && !sourceRefsAlreadyCompactV0(rawRefs) {
		if refs, ok := compactSourceRefsFromRichPayloadV0(rawRefs); ok {
			encodedRefs, err := json.Marshal(refs)
			if err == nil {
				payload["source_refs"] = encodedRefs
				if _, exists := payload["source_ref_details"]; !exists {
					payload["source_ref_details"] = append([]byte(nil), rawRefs...)
				}
				changed = true
			}
		}
	}
	if rawCitations, ok := payload["citations"]; ok && len(rawCitations) > 0 {
		if !contentBlockPayloadHasSourceRefsV0(payload) {
			if refs, ok := sourceRefsFromContentBlockCitationsV0(rawCitations); ok {
				encodedRefs, err := json.Marshal(refs)
				if err == nil {
					payload["source_refs"] = encodedRefs
					changed = true
				}
			}
		}
		if encodedCitations, ok := compactContentBlockCitationsV0(rawCitations); ok {
			payload["citations"] = encodedCitations
			if _, exists := payload["citation_details"]; !exists {
				payload["citation_details"] = append([]byte(nil), rawCitations...)
			}
			changed = true
		}
	}
	if !changed {
		return body, true
	}
	canonical, err := json.Marshal(payload)
	if err != nil {
		return body, true
	}
	return string(canonical), true
}

func contentBlockPayloadHasSourceRefsV0(payload map[string]json.RawMessage) bool {
	raw, ok := payload["source_refs"]
	if !ok || len(raw) == 0 {
		return false
	}
	var refs []string
	if err := json.Unmarshal(raw, &refs); err == nil {
		return len(compactCodexStackStringsV0(refs)) > 0
	}
	if refs, ok := compactSourceRefsFromRichPayloadV0(raw); ok {
		return len(refs) > 0
	}
	return false
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
	for _, key := range []string{"source_ref", "ref", "source_ref_id", "source_id", "source", "id", "fuente"} {
		raw, ok := entry[key]
		if !ok {
			continue
		}
		if value := stringFromRawJSONV0(raw); value != "" {
			return value
		}
	}
	return ""
}

func sourceRefsFromContentBlockCitationsV0(raw json.RawMessage) ([]string, bool) {
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

func compactContentBlockCitationsV0(raw json.RawMessage) (json.RawMessage, bool) {
	var entries []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil || len(entries) == 0 {
		return nil, false
	}
	normalized := make([]map[string]json.RawMessage, 0, len(entries))
	changed := false
	for _, entry := range entries {
		ref := sourceRefFromRichEntryV0(entry)
		if ref == "" {
			changed = true
			continue
		}
		sourceRef, err := json.Marshal(ref)
		if err != nil {
			continue
		}
		if stringFromRawJSONV0(entry["source_ref"]) != ref {
			entry["source_ref"] = sourceRef
			changed = true
		}
		if strings.TrimSpace(stringFromRawJSONV0(entry["note"])) == "" {
			if note := citationNoteFromRichEntryV0(entry); note != "" {
				encodedNote, err := json.Marshal(note)
				if err == nil {
					entry["note"] = encodedNote
					changed = true
				}
			}
		}
		normalized = append(normalized, entry)
	}
	if len(normalized) == 0 {
		return nil, false
	}
	if !changed {
		return nil, false
	}
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return nil, false
	}
	return encoded, true
}

func citationNoteFromRichEntryV0(entry map[string]json.RawMessage) string {
	var parts []string
	for _, key := range []string{"title", "url", "usage", "claim"} {
		value := stringFromRawJSONV0(entry[key])
		if value == "" {
			continue
		}
		parts = append(parts, key+": "+value)
	}
	return strings.Join(parts, "; ")
}

func stringFromRawJSONV0(raw json.RawMessage) string {
	if len(raw) == 0 || !json.Valid(raw) {
		return ""
	}
	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		return strings.TrimSpace(value)
	}
	return ""
}
