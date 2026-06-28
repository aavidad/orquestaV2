package orquestaappcodexstack

import (
	"encoding/json"
	"strconv"
	"strings"
)

func canonicalDomainWorkAudioAssetPayloadJSONV0(body string) (string, bool) {
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
		if _, exists := payload["source_ref_details"]; !exists {
			payload["source_ref_details"] = append([]byte(nil), rawRefs...)
		}
		if refs, ok := compactAudioAssetSourceRefsV0(rawRefs); ok {
			if encodedRefs, err := json.Marshal(refs); err == nil {
				payload["source_refs"] = encodedRefs
			} else {
				delete(payload, "source_refs")
			}
		} else {
			delete(payload, "source_refs")
		}
		changed = true
	}
	if rawDuration, ok := payload["duration_seconds"]; ok && len(rawDuration) > 0 {
		if encoded, ok := canonicalAudioAssetDurationJSONV0(rawDuration); ok {
			if string(encoded) != string(rawDuration) {
				payload["duration_seconds"] = encoded
				changed = true
			}
		} else {
			if _, exists := payload["duration_seconds_raw"]; !exists {
				payload["duration_seconds_raw"] = append([]byte(nil), rawDuration...)
			}
			delete(payload, "duration_seconds")
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

func canonicalAudioAssetDurationJSONV0(raw json.RawMessage) (json.RawMessage, bool) {
	var value int
	if err := json.Unmarshal(raw, &value); err == nil && value > 0 {
		return json.RawMessage(strconv.Itoa(value)), true
	}
	var numeric float64
	if err := json.Unmarshal(raw, &numeric); err == nil && numeric > 0 {
		return json.RawMessage(strconv.Itoa(int(numeric))), true
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return nil, false
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil || parsed <= 0 {
		return nil, false
	}
	return json.RawMessage(strconv.Itoa(parsed)), true
}

func compactAudioAssetSourceRefsV0(raw json.RawMessage) ([]string, bool) {
	var refs []string
	if err := json.Unmarshal(raw, &refs); err == nil {
		refs = compactCodexStackStringsV0(refs)
		return refs, len(refs) > 0
	}
	if refs, ok := compactSourceRefsFromRichPayloadV0(raw); ok {
		return refs, true
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || len(object) == 0 {
		return nil, false
	}
	refs = make([]string, 0, len(object))
	for _, key := range []string{
		"source_ref",
		"source_artifact_ref",
		"source_content_artifact_ref",
		"source_content_ref",
		"assembled_topic_artifact_ref",
		"assembled_topic_artifact_id",
		"artifact_ref",
		"topic_id",
		"program_id",
		"course_id",
	} {
		if value := stringFromRawJSONV0(object[key]); value != "" {
			refs = append(refs, value)
		}
	}
	return compactCodexStackStringsV0(refs), len(refs) > 0
}
