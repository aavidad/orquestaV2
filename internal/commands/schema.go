package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
)

type schemaNode struct {
	Type                 string                `json:"type"`
	Properties           map[string]schemaNode `json:"properties,omitempty"`
	Required             []string              `json:"required,omitempty"`
	AdditionalProperties *bool                 `json:"additionalProperties,omitempty"`
	Items                *schemaNode           `json:"items,omitempty"`
	Enum                 []string              `json:"enum,omitempty"`
	Minimum              *json.Number          `json:"minimum,omitempty"`
	Maximum              *json.Number          `json:"maximum,omitempty"`
	MinItems             *int                  `json:"minItems,omitempty"`
	MaxItems             *int                  `json:"maxItems,omitempty"`
	MinLength            *int                  `json:"minLength,omitempty"`
	MaxLength            *int                  `json:"maxLength,omitempty"`
}

func validatePayload(schemaBytes, payload json.RawMessage) (json.RawMessage, error) {
	schema, err := parseSchema(schemaBytes)
	if err != nil {
		return nil, errContract
	}
	if len(payload) == 0 {
		payload = json.RawMessage("{}")
	}
	if err := validateJSONValue(schema, payload); err != nil {
		return nil, err
	}
	var value any
	if err := decodeSingle(payload, &value, true); err != nil || value == nil {
		return nil, errors.New("commands.payload_invalid")
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return nil, errors.New("commands.payload_invalid")
	}
	return canonical, nil
}

func parseSchema(encoded json.RawMessage) (schemaNode, error) {
	var schema schemaNode
	if err := decodeSingle(encoded, &schema, true); err != nil {
		return schemaNode{}, err
	}
	if err := validateSchemaNode(schema, true); err != nil {
		return schemaNode{}, err
	}
	return schema, nil
}

// ValidateSchemaContract is shared with commandgen so generation and runtime
// reject the same unsupported or open-world schema forms.
func ValidateSchemaContract(encoded json.RawMessage) error {
	_, err := parseSchema(encoded)
	return err
}

func validateSchemaNode(schema schemaNode, root bool) error {
	switch schema.Type {
	case "object":
		if schema.Properties == nil || schema.AdditionalProperties == nil || *schema.AdditionalProperties ||
			schema.Items != nil || len(schema.Enum) != 0 || hasNumericOrLengthConstraints(schema) {
			return errContract
		}
		seen := make(map[string]struct{}, len(schema.Required))
		for _, name := range schema.Required {
			if _, duplicate := seen[name]; duplicate {
				return errContract
			}
			if _, ok := schema.Properties[name]; !ok {
				return errContract
			}
			seen[name] = struct{}{}
		}
		for _, property := range schema.Properties {
			if err := validateSchemaNode(property, false); err != nil {
				return err
			}
		}
	case "array":
		if root || schema.Items == nil || schema.Properties != nil || len(schema.Required) != 0 ||
			schema.AdditionalProperties != nil || len(schema.Enum) != 0 || schema.Minimum != nil ||
			schema.Maximum != nil || schema.MinLength != nil || schema.MaxLength != nil ||
			invalidRange(schema.MinItems, schema.MaxItems) {
			return errContract
		}
		return validateSchemaNode(*schema.Items, false)
	case "string":
		if root || schema.Properties != nil || len(schema.Required) != 0 || schema.AdditionalProperties != nil ||
			schema.Items != nil || schema.Minimum != nil || schema.Maximum != nil || schema.MinItems != nil ||
			schema.MaxItems != nil || invalidRange(schema.MinLength, schema.MaxLength) {
			return errContract
		}
	case "integer":
		if root || schema.Properties != nil || len(schema.Required) != 0 || schema.AdditionalProperties != nil ||
			schema.Items != nil || len(schema.Enum) != 0 || schema.MinItems != nil || schema.MaxItems != nil ||
			schema.MinLength != nil || schema.MaxLength != nil || invalidNumericRange(schema.Minimum, schema.Maximum) {
			return errContract
		}
	case "boolean":
		if root || schema.Properties != nil || len(schema.Required) != 0 || schema.AdditionalProperties != nil ||
			schema.Items != nil || len(schema.Enum) != 0 || hasNumericOrLengthConstraints(schema) {
			return errContract
		}
	default:
		return errContract
	}
	return nil
}

func invalidRange(minimum, maximum *int) bool {
	return minimum != nil && *minimum < 0 || maximum != nil && *maximum < 0 ||
		minimum != nil && maximum != nil && *maximum < *minimum
}

func invalidNumericRange(minimum, maximum *json.Number) bool {
	var low, high int64
	var err error
	if minimum != nil {
		low, err = minimum.Int64()
		if err != nil {
			return true
		}
	}
	if maximum != nil {
		high, err = maximum.Int64()
		if err != nil {
			return true
		}
	}
	return minimum != nil && maximum != nil && high < low
}

func hasNumericOrLengthConstraints(schema schemaNode) bool {
	return schema.Minimum != nil || schema.Maximum != nil || schema.MinItems != nil || schema.MaxItems != nil ||
		schema.MinLength != nil || schema.MaxLength != nil
}

func validateJSONValue(schema schemaNode, raw json.RawMessage) error {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return errors.New("commands.payload_null")
	}
	switch schema.Type {
	case "object":
		var value map[string]json.RawMessage
		if err := decodeSingle(raw, &value, false); err != nil || value == nil {
			return errors.New("commands.payload_object_invalid")
		}
		for _, name := range schema.Required {
			if child, ok := value[name]; !ok || bytes.Equal(bytes.TrimSpace(child), []byte("null")) {
				return fmt.Errorf("commands.payload_required:%s", name)
			}
		}
		for name, child := range value {
			property, ok := schema.Properties[name]
			if !ok {
				return fmt.Errorf("commands.payload_unknown:%s", name)
			}
			if err := validateJSONValue(property, child); err != nil {
				return fmt.Errorf("commands.payload_field:%s:%w", name, err)
			}
		}
	case "array":
		var values []json.RawMessage
		if err := decodeSingle(raw, &values, false); err != nil || values == nil ||
			(schema.MinItems != nil && len(values) < *schema.MinItems) ||
			(schema.MaxItems != nil && len(values) > *schema.MaxItems) {
			return errors.New("commands.payload_array_invalid")
		}
		for _, value := range values {
			if err := validateJSONValue(*schema.Items, value); err != nil {
				return err
			}
		}
	case "string":
		var value string
		if err := decodeSingle(raw, &value, false); err != nil {
			return errors.New("commands.payload_string_invalid")
		}
		length := utf8.RuneCountInString(value)
		if schema.MinLength != nil && length < *schema.MinLength || schema.MaxLength != nil && length > *schema.MaxLength {
			return errors.New("commands.payload_string_range")
		}
		if len(schema.Enum) > 0 && !containsString(schema.Enum, value) {
			return errors.New("commands.payload_enum_invalid")
		}
	case "integer":
		var value json.Number
		if err := decodeSingle(raw, &value, true); err != nil {
			return errors.New("commands.payload_integer_invalid")
		}
		integer, err := value.Int64()
		if err != nil || belowMinimum(integer, schema.Minimum) || aboveMaximum(integer, schema.Maximum) {
			return errors.New("commands.payload_integer_range")
		}
	case "boolean":
		var value bool
		if err := decodeSingle(raw, &value, false); err != nil {
			return errors.New("commands.payload_boolean_invalid")
		}
	default:
		return errContract
	}
	return nil
}

func belowMinimum(value int64, minimum *json.Number) bool {
	if minimum == nil {
		return false
	}
	want, err := minimum.Int64()
	return err != nil || value < want
}

func aboveMaximum(value int64, maximum *json.Number) bool {
	if maximum == nil {
		return false
	}
	want, err := maximum.Int64()
	return err != nil || value > want
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func decodeSingle(encoded []byte, target any, useNumber bool) error {
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if useNumber {
		decoder.UseNumber()
	}
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("commands.json_trailing_data")
	}
	return nil
}

func decodePayload(payload json.RawMessage, target any) error {
	if err := decodeSingle(payload, target, true); err != nil {
		return commandError{code: CodeInvalidRequest, cause: err}
	}
	return nil
}
