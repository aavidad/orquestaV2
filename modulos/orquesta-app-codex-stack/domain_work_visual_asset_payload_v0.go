package orquestaappcodexstack

import (
	"encoding/json"
	"sort"
	"strings"
)

func canonicalDomainWorkVisualAssetPayloadJSONV0(body string) (string, bool) {
	body = strings.TrimSpace(body)
	if body == "" {
		return "", false
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &payload); err != nil || len(payload) == 0 {
		return "", false
	}
	asset := firstVisualAssetPayloadEntryV0(payload)
	inputContract := visualAssetObjectFieldV0(payload, "input_contract")
	sourceRefMap := visualAssetObjectFieldV0(payload, "source_refs")
	blueprint := firstNonEmptyVisualAssetObjectFieldV0(asset, "diagram_blueprint", "diagram_spec")
	generationPrompt := firstNonEmptyVisualAssetObjectFieldV0(
		asset,
		"generation_prompt",
		"prompt",
		"render_prompt",
	)
	if len(generationPrompt) == 0 {
		generationPrompt = visualAssetObjectFieldV0(payload, "prompt")
	}
	promptComposition := visualAssetObjectFieldV0(generationPrompt, "composition")
	placement := visualAssetObjectFieldV0(asset, "recommended_placement")
	promptText := firstNonEmptyVisualAssetPayloadValueV0(
		visualAssetStringFieldV0(asset, "prompt"),
		visualAssetStringFieldV0(asset, "render_prompt"),
		visualAssetStringFieldV0(generationPrompt, "instruction"),
		visualAssetStringFieldV0(generationPrompt, "prompt"),
		visualAssetStringFieldV0(generationPrompt, "image_prompt"),
	)

	changed := false
	changed = visualAssetPreserveObjectFieldV0(payload, "prompt", "visual_prompt_details") || changed
	changed = visualAssetSetStringIfEmptyV0(payload, "topic_id", firstNonEmptyVisualAssetPayloadValueV0(
		visualAssetStringFieldV0(asset, "topic_id"),
		visualAssetStringFieldV0(sourceRefMap, "topic_id"),
		visualAssetStringFieldV0(inputContract, "topic_id"),
	)) || changed
	changed = visualAssetSetStringIfEmptyV0(payload, "chapter_id", firstNonEmptyVisualAssetPayloadValueV0(
		visualAssetStringFieldV0(asset, "chapter_id"),
		visualAssetStringFieldV0(sourceRefMap, "chapter_id"),
		visualAssetStringFieldV0(inputContract, "chapter_id"),
	)) || changed
	changed = visualAssetSetStringIfEmptyV0(payload, "block_id", firstNonEmptyVisualAssetPayloadValueV0(
		visualAssetStringFieldV0(asset, "block_id"),
		visualAssetStringFieldV0(inputContract, "block_id"),
	)) || changed
	changed = visualAssetSetStringIfEmptyV0(payload, "asset_type", firstNonEmptyVisualAssetPayloadValueV0(
		visualAssetStringFieldV0(asset, "asset_type"),
		visualAssetStringFieldV0(inputContract, "asset_type"),
		visualAssetStringFieldV0(asset, "visual_kind"),
	)) || changed
	changed = visualAssetSetStringIfEmptyV0(payload, "title", firstNonEmptyVisualAssetPayloadValueV0(
		visualAssetStringFieldV0(asset, "title"),
		visualAssetStringFieldV0(inputContract, "title"),
	)) || changed
	changed = visualAssetSetStringIfEmptyV0(payload, "caption", firstNonEmptyVisualAssetPayloadValueV0(
		visualAssetStringFieldV0(asset, "caption"),
		visualAssetStringFieldV0(asset, "didactic_objective"),
		visualAssetStringFieldV0(asset, "editorial_rationale"),
		visualAssetStringFieldV0(asset, "editorial_fit"),
	)) || changed
	changed = visualAssetSetStringIfEmptyV0(payload, "alt_text", firstNonEmptyVisualAssetPayloadValueV0(
		visualAssetStringFieldV0(asset, "alt_text"),
		visualAssetStringFieldV0(asset, "accessibility_description"),
	)) || changed
	changed = visualAssetSetStringIfEmptyV0(payload, "placement", firstNonEmptyVisualAssetPayloadValueV0(
		visualAssetStringFieldV0(asset, "placement"),
		visualAssetStringFieldV0(asset, "placement_ref"),
		visualAssetStringFieldV0(placement, "placement_ref"),
		visualAssetStringFieldV0(inputContract, "placement"),
	)) || changed
	changed = visualAssetSetStringIfEmptyV0(payload, "prompt", firstNonEmptyVisualAssetPayloadValueV0(
		promptText,
	)) || changed
	changed = visualAssetSetStringIfEmptyV0(payload, "image_prompt", firstNonEmptyVisualAssetPayloadValueV0(
		visualAssetStringFieldV0(asset, "image_prompt"),
		promptText,
		visualAssetStringFieldV0(generationPrompt, "image_prompt"),
	)) || changed
	changed = visualAssetSetStringIfEmptyV0(payload, "negative_prompt", firstNonEmptyVisualAssetPayloadValueV0(
		visualAssetStringFieldV0(asset, "negative_prompt"),
		visualAssetStringFieldV0(generationPrompt, "negative_prompt"),
	)) || changed
	changed = visualAssetSetStringIfEmptyV0(payload, "style", firstNonEmptyVisualAssetPayloadValueV0(
		visualAssetStringFieldV0(asset, "style"),
		visualAssetStringFieldV0(promptComposition, "style"),
		visualAssetStringFieldV0(inputContract, "style"),
	)) || changed
	changed = visualAssetSetStringIfEmptyV0(payload, "aspect_ratio", firstNonEmptyVisualAssetPayloadValueV0(
		visualAssetStringFieldV0(asset, "aspect_ratio"),
		visualAssetStringFieldV0(inputContract, "aspect_ratio"),
	)) || changed
	changed = visualAssetSetStringIfEmptyV0(payload, "language_code", firstNonEmptyVisualAssetPayloadValueV0(
		visualAssetStringFieldV0(asset, "language_code"),
		visualAssetStringFieldV0(generationPrompt, "language"),
		visualAssetStringFieldV0(inputContract, "language_code"),
	)) || changed

	bodyValue := firstNonEmptyVisualAssetPayloadValueV0(
		visualAssetStringFieldV0(payload, "body"),
		visualAssetStringFieldV0(asset, "body"),
		visualAssetStringFieldV0(asset, "markdown"),
		visualAssetStringFieldV0(blueprint, "svg"),
		visualAssetStringFieldV0(blueprint, "mermaid"),
		visualAssetStringFieldV0(blueprint, "html"),
		visualAssetStringFieldV0(asset, "render_prompt"),
		promptText,
	)
	changed = visualAssetSetStringIfEmptyV0(payload, "body", bodyValue) || changed
	changed = visualAssetSetStringIfEmptyV0(payload, "svg", firstNonEmptyVisualAssetPayloadValueV0(
		visualAssetStringFieldV0(asset, "svg"),
		visualAssetStringFieldV0(blueprint, "svg"),
	)) || changed
	changed = visualAssetSetStringIfEmptyV0(payload, "mermaid", firstNonEmptyVisualAssetPayloadValueV0(
		visualAssetStringFieldV0(asset, "mermaid"),
		visualAssetStringFieldV0(blueprint, "mermaid"),
	)) || changed
	changed = visualAssetSetStringIfEmptyV0(payload, "html", firstNonEmptyVisualAssetPayloadValueV0(
		visualAssetStringFieldV0(asset, "html"),
		visualAssetStringFieldV0(blueprint, "html"),
	)) || changed

	format := visualAssetNormalizedFormatV0(firstNonEmptyVisualAssetPayloadValueV0(
		visualAssetStringFieldV0(payload, "format"),
		visualAssetStringFieldV0(asset, "format"),
		visualAssetStringFieldV0(blueprint, "format"),
		visualAssetStringFieldV0(blueprint, "recommended_format"),
		visualAssetStringFieldV0(inputContract, "format"),
	))
	if format == "" {
		switch {
		case visualAssetStringFieldV0(payload, "svg") != "":
			format = "svg"
		case visualAssetStringFieldV0(payload, "mermaid") != "":
			format = "mermaid"
		case visualAssetStringFieldV0(payload, "html") != "":
			format = "html"
		}
	}
	if format == "" || visualAssetFormatMissingConcreteBodyV0(format, payload) {
		if bodyValue != "" {
			format = "markdown"
		}
	}
	changed = visualAssetSetStringV0(payload, "format", format) || changed

	if refs, normalizeRefs := compactVisualAssetSourceRefsV0(payload, asset, inputContract); normalizeRefs {
		if raw, ok := payload["source_refs"]; ok && len(raw) > 0 && !sourceRefsAlreadyCompactV0(raw) {
			if _, exists := payload["source_ref_details"]; !exists {
				payload["source_ref_details"] = append([]byte(nil), raw...)
			}
		}
		if encodedRefs, err := json.Marshal(refs); err == nil {
			payload["source_refs"] = encodedRefs
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

func firstVisualAssetPayloadEntryV0(payload map[string]json.RawMessage) map[string]json.RawMessage {
	for _, key := range []string{"assets", "visual_assets"} {
		raw, ok := payload[key]
		if !ok || len(raw) == 0 {
			continue
		}
		var entries []map[string]json.RawMessage
		if err := json.Unmarshal(raw, &entries); err != nil || len(entries) == 0 {
			continue
		}
		for _, entry := range entries {
			if len(entry) > 0 {
				return entry
			}
		}
	}
	return nil
}

func firstNonEmptyVisualAssetObjectFieldV0(parent map[string]json.RawMessage, keys ...string) map[string]json.RawMessage {
	for _, key := range keys {
		if value := visualAssetObjectFieldV0(parent, key); len(value) > 0 {
			return value
		}
	}
	return nil
}

func visualAssetObjectFieldV0(parent map[string]json.RawMessage, key string) map[string]json.RawMessage {
	if len(parent) == 0 {
		return nil
	}
	raw, ok := parent[key]
	if !ok || len(raw) == 0 {
		return nil
	}
	var out map[string]json.RawMessage
	if err := json.Unmarshal(raw, &out); err != nil || len(out) == 0 {
		return nil
	}
	return out
}

func visualAssetStringFieldV0(parent map[string]json.RawMessage, keys ...string) string {
	for _, key := range keys {
		if len(parent) == 0 {
			return ""
		}
		if value := stringFromRawJSONV0(parent[key]); value != "" {
			return value
		}
	}
	return ""
}

func firstNonEmptyVisualAssetPayloadValueV0(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func visualAssetSetStringIfEmptyV0(payload map[string]json.RawMessage, key string, value string) bool {
	if stringFromRawJSONV0(payload[key]) != "" {
		return false
	}
	return visualAssetSetStringV0(payload, key, value)
}

func visualAssetSetStringV0(payload map[string]json.RawMessage, key string, value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return false
	}
	if string(payload[key]) == string(encoded) {
		return false
	}
	payload[key] = encoded
	return true
}

func visualAssetPreserveObjectFieldV0(
	payload map[string]json.RawMessage,
	key string,
	detailsKey string,
) bool {
	if len(payload) == 0 || payload[detailsKey] != nil || stringFromRawJSONV0(payload[key]) != "" {
		return false
	}
	value := visualAssetObjectFieldV0(payload, key)
	if len(value) == 0 {
		return false
	}
	payload[detailsKey] = append([]byte(nil), payload[key]...)
	return true
}

func visualAssetNormalizedFormatV0(value string) string {
	key := normalizeDomainWorkDeliveryAliasV0(value)
	switch {
	case key == "":
		return ""
	case strings.Contains(key, "svg"):
		return "svg"
	case strings.Contains(key, "mermaid"):
		return "mermaid"
	case strings.Contains(key, "html"):
		return "html"
	case strings.Contains(key, "image") || strings.Contains(key, "data_uri"):
		return "image"
	default:
		return strings.TrimSpace(value)
	}
}

func visualAssetFormatMissingConcreteBodyV0(format string, payload map[string]json.RawMessage) bool {
	switch format {
	case "svg":
		return visualAssetStringFieldV0(payload, "svg") == ""
	case "mermaid":
		return visualAssetStringFieldV0(payload, "mermaid") == ""
	case "html":
		return visualAssetStringFieldV0(payload, "html") == ""
	case "image":
		return visualAssetStringFieldV0(payload, "image_data_uri") == ""
	default:
		return false
	}
}

func compactVisualAssetSourceRefsV0(
	payload map[string]json.RawMessage,
	asset map[string]json.RawMessage,
	inputContract map[string]json.RawMessage,
) ([]string, bool) {
	refs := []string{}
	refs = appendVisualAssetSourceRefsFromRawV0(refs, payload["source_refs"])
	refs = appendVisualAssetSourceRefsFromObjectV0(refs, asset,
		"source_ref", "source_id", "source_ref_id", "citation_ref")
	refs = appendVisualAssetSourceRefsFromObjectV0(refs, inputContract,
		"source_ref", "source_id", "source_ref_id", "citation_ref")
	_, hadRawSourceRefs := payload["source_refs"]
	return compactCodexStackStringsV0(refs), hadRawSourceRefs || len(refs) > 0
}

func appendVisualAssetSourceRefsFromRawV0(refs []string, raw json.RawMessage) []string {
	if len(raw) == 0 || !json.Valid(raw) {
		return refs
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err == nil {
		return append(refs, values...)
	}
	var entries []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &entries); err == nil {
		for _, entry := range entries {
			if ref := sourceRefFromRichEntryV0(entry); ref != "" {
				refs = append(refs, ref)
			}
		}
		return refs
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err == nil {
		return appendVisualAssetSourceRefsFromObjectV0(refs, object,
			"source_ref", "source_id", "source_ref_id", "citation_ref", "ref")
	}
	return refs
}

func appendVisualAssetSourceRefsFromObjectV0(
	refs []string,
	object map[string]json.RawMessage,
	preferredKeys ...string,
) []string {
	if len(object) == 0 {
		return refs
	}
	for _, key := range preferredKeys {
		if value := stringFromRawJSONV0(object[key]); value != "" {
			refs = append(refs, value)
		}
	}
	keys := make([]string, 0, len(object))
	for key := range object {
		key = strings.TrimSpace(key)
		if key != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		normalized := normalizeDomainWorkDeliveryAliasV0(key)
		if normalized != "source_ref" && normalized != "source_id" &&
			normalized != "source_ref_id" && normalized != "citation_ref" {
			continue
		}
		if value := stringFromRawJSONV0(object[key]); value != "" {
			refs = append(refs, value)
		}
	}
	return refs
}
