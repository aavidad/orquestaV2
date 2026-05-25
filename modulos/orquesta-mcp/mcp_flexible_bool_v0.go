package orquestamcp

import (
	"encoding/json"
	"strings"
)

type MCPFlexibleBoolV0 bool

type mcpFlexibleBoolV0 = MCPFlexibleBoolV0

func (flag *MCPFlexibleBoolV0) UnmarshalJSON(raw []byte) error {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		*flag = false
		return nil
	}
	if trimmed == "true" || trimmed == "false" {
		var value bool
		if err := json.Unmarshal(raw, &value); err != nil {
			return err
		}
		*flag = MCPFlexibleBoolV0(value)
		return nil
	}
	if strings.HasPrefix(trimmed, "\"") {
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return err
		}
		*flag = MCPFlexibleBoolV0(mcpFlexibleBoolTextV0(value))
		return nil
	}
	if strings.HasPrefix(trimmed, "[") {
		var values []any
		if err := json.Unmarshal(raw, &values); err != nil {
			return err
		}
		*flag = MCPFlexibleBoolV0(len(values) > 0)
		return nil
	}
	if strings.HasPrefix(trimmed, "{") {
		var values map[string]any
		if err := json.Unmarshal(raw, &values); err != nil {
			return err
		}
		*flag = MCPFlexibleBoolV0(len(values) > 0)
		return nil
	}
	var number float64
	if err := json.Unmarshal(raw, &number); err != nil {
		return err
	}
	*flag = MCPFlexibleBoolV0(number != 0)
	return nil
}

func mcpFlexibleBoolTextV0(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "0", "false", "no", "off", "none", "null":
		return false
	default:
		return true
	}
}
