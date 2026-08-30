package tooling

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"sort"
	"unicode/utf8"
)

func canonicalObjectSchema(encoded json.RawMessage) (json.RawMessage, error) {
	value, err := decodeJSONValue(encoded)
	if err != nil {
		return nil, err
	}
	if err := validateSchemaNode(value, true); err != nil {
		return nil, err
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return canonical, nil
}

func validatePayloadAgainstSchema(schemaBytes, payload json.RawMessage) (json.RawMessage, error) {
	schema, err := decodeJSONValue(schemaBytes)
	if err != nil {
		return nil, err
	}
	value, err := decodeJSONValue(payload)
	if err != nil || value == nil {
		return nil, errors.New("payload_invalid")
	}
	if err := validatePayloadNode(schema, value); err != nil {
		return nil, err
	}
	return json.Marshal(value)
}

func decodeJSONValue(encoded []byte) (any, error) {
	if len(encoded) == 0 || !utf8.Valid(encoded) {
		return nil, errors.New("json_invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		return nil, err
	}
	return value, nil
}

func rejectTrailingJSON(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("schema_multiple_values")
		}
		return err
	}
	return nil
}

func validateSchemaNode(raw any, root bool) error {
	node, ok := raw.(map[string]any)
	if !ok {
		return errors.New("schema_node_invalid")
	}
	typeName, ok := node["type"].(string)
	if !ok {
		return errors.New("schema_type_invalid")
	}
	allowed := map[string]bool{"type": true}
	switch typeName {
	case "object":
		allowed["properties"], allowed["required"], allowed["additionalProperties"] = true, true, true
		properties, ok := node["properties"].(map[string]any)
		closed, closedOK := node["additionalProperties"].(bool)
		if !ok || !closedOK || closed {
			return errors.New("schema_object_open")
		}
		for name, child := range properties {
			if name == "" || validateSchemaNode(child, false) != nil {
				return errors.New("schema_property_invalid")
			}
		}
		if required, exists := node["required"]; exists {
			values, ok := required.([]any)
			if !ok {
				return errors.New("schema_required_invalid")
			}
			seen := make(map[string]struct{}, len(values))
			for _, value := range values {
				name, ok := value.(string)
				_, duplicate := seen[name]
				if !ok || name == "" || duplicate {
					return errors.New("schema_required_invalid")
				}
				if _, exists := properties[name]; !exists {
					return errors.New("schema_required_unknown")
				}
				seen[name] = struct{}{}
			}
			sort.Slice(values, func(left, right int) bool {
				return values[left].(string) < values[right].(string)
			})
		}
	case "array":
		if root {
			return errors.New("schema_root_not_object")
		}
		allowed["items"], allowed["minItems"], allowed["maxItems"] = true, true, true
		if validateSchemaNode(node["items"], false) != nil || invalidIntegerRange(node, "minItems", "maxItems", true) {
			return errors.New("schema_array_invalid")
		}
	case "string":
		if root {
			return errors.New("schema_root_not_object")
		}
		allowed["enum"], allowed["minLength"], allowed["maxLength"] = true, true, true
		_, hasEnum := node["enum"]
		if invalidIntegerRange(node, "minLength", "maxLength", true) || hasEnum && invalidStringEnum(node["enum"]) {
			return errors.New("schema_string_invalid")
		}
	case "integer":
		if root {
			return errors.New("schema_root_not_object")
		}
		allowed["minimum"], allowed["maximum"] = true, true
		if invalidIntegerRange(node, "minimum", "maximum", false) {
			return errors.New("schema_integer_invalid")
		}
	case "number":
		if root {
			return errors.New("schema_root_not_object")
		}
		allowed["minimum"], allowed["maximum"] = true, true
		if invalidNumberRange(node, "minimum", "maximum") {
			return errors.New("schema_number_invalid")
		}
	case "boolean":
		if root {
			return errors.New("schema_root_not_object")
		}
	default:
		return errors.New("schema_type_unsupported")
	}
	for keyword := range node {
		if !allowed[keyword] {
			return errors.New("schema_keyword_unsupported")
		}
	}
	return nil
}

func invalidStringEnum(raw any) bool {
	if raw == nil {
		return true
	}
	values, ok := raw.([]any)
	if !ok || len(values) == 0 {
		return true
	}
	seen := make(map[string]struct{}, len(values))
	for _, rawValue := range values {
		value, ok := rawValue.(string)
		_, duplicate := seen[value]
		if !ok || duplicate {
			return true
		}
		seen[value] = struct{}{}
	}
	sort.Slice(values, func(left, right int) bool {
		return values[left].(string) < values[right].(string)
	})
	return false
}

func invalidIntegerRange(node map[string]any, lowKey, highKey string, nonnegative bool) bool {
	low, hasLow, lowOK := integerKeyword(node, lowKey)
	high, hasHigh, highOK := integerKeyword(node, highKey)
	return !lowOK || !highOK || nonnegative && (hasLow && low < 0 || hasHigh && high < 0) ||
		hasLow && hasHigh && high < low
}

func integerKeyword(node map[string]any, key string) (int64, bool, bool) {
	raw, exists := node[key]
	if !exists {
		return 0, false, true
	}
	number, ok := raw.(json.Number)
	if !ok {
		return 0, true, false
	}
	value, err := number.Int64()
	return value, true, err == nil
}

func invalidNumberRange(node map[string]any, lowKey, highKey string) bool {
	low, hasLow, lowOK := numberKeyword(node, lowKey)
	high, hasHigh, highOK := numberKeyword(node, highKey)
	return !lowOK || !highOK || hasLow && hasHigh && low.Cmp(high) > 0
}

func numberKeyword(node map[string]any, key string) (*big.Rat, bool, bool) {
	raw, exists := node[key]
	if !exists {
		return nil, false, true
	}
	number, ok := raw.(json.Number)
	if !ok {
		return nil, true, false
	}
	value, ok := new(big.Rat).SetString(number.String())
	return value, true, ok
}

func validatePayloadNode(schemaRaw, value any) error {
	schema, ok := schemaRaw.(map[string]any)
	if !ok {
		return errors.New("payload_schema_invalid")
	}
	typeName, _ := schema["type"].(string)
	switch typeName {
	case "object":
		object, ok := value.(map[string]any)
		properties, propertiesOK := schema["properties"].(map[string]any)
		if !ok || !propertiesOK {
			return errors.New("payload_object_invalid")
		}
		if required, exists := schema["required"].([]any); exists {
			for _, rawName := range required {
				name, _ := rawName.(string)
				if child, found := object[name]; !found || child == nil {
					return errors.New("payload_required_missing")
				}
			}
		}
		for name, child := range object {
			childSchema, found := properties[name]
			if !found || child == nil || validatePayloadNode(childSchema, child) != nil {
				return errors.New("payload_property_invalid")
			}
		}
	case "array":
		values, ok := value.([]any)
		if !ok || payloadIntegerRangeInvalid(schema, len(values), "minItems", "maxItems") {
			return errors.New("payload_array_invalid")
		}
		for _, child := range values {
			if child == nil || validatePayloadNode(schema["items"], child) != nil {
				return errors.New("payload_array_item_invalid")
			}
		}
	case "string":
		text, ok := value.(string)
		if !ok || payloadIntegerRangeInvalid(schema, utf8.RuneCountInString(text), "minLength", "maxLength") {
			return errors.New("payload_string_invalid")
		}
		if rawEnum, exists := schema["enum"].([]any); exists {
			found := false
			for _, candidate := range rawEnum {
				found = found || candidate == text
			}
			if !found {
				return errors.New("payload_enum_invalid")
			}
		}
	case "integer":
		number, ok := value.(json.Number)
		integer, err := number.Int64()
		if !ok || err != nil || payloadIntegerBoundsInvalid(schema, integer) {
			return errors.New("payload_integer_invalid")
		}
	case "number":
		number, ok := value.(json.Number)
		decimal, decimalOK := new(big.Rat).SetString(number.String())
		if !ok || !decimalOK || payloadNumberBoundsInvalid(schema, decimal) {
			return errors.New("payload_number_invalid")
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return errors.New("payload_boolean_invalid")
		}
	default:
		return errors.New("payload_type_invalid")
	}
	return nil
}

func payloadIntegerRangeInvalid(schema map[string]any, value int, lowKey, highKey string) bool {
	low, hasLow, lowOK := integerKeyword(schema, lowKey)
	high, hasHigh, highOK := integerKeyword(schema, highKey)
	return !lowOK || !highOK || hasLow && int64(value) < low || hasHigh && int64(value) > high
}

func payloadIntegerBoundsInvalid(schema map[string]any, value int64) bool {
	low, hasLow, lowOK := integerKeyword(schema, "minimum")
	high, hasHigh, highOK := integerKeyword(schema, "maximum")
	return !lowOK || !highOK || hasLow && value < low || hasHigh && value > high
}

func payloadNumberBoundsInvalid(schema map[string]any, value *big.Rat) bool {
	low, hasLow, lowOK := numberKeyword(schema, "minimum")
	high, hasHigh, highOK := numberKeyword(schema, "maximum")
	return !lowOK || !highOK || hasLow && value.Cmp(low) < 0 || hasHigh && value.Cmp(high) > 0
}
