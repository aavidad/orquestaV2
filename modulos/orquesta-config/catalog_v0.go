package orquestaconfig

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

type RestartBehaviorV0 string

const (
	RestartBehaviorNoneV0    RestartBehaviorV0 = "none"
	RestartBehaviorReloadV0  RestartBehaviorV0 = "reload"
	RestartBehaviorRestartV0 RestartBehaviorV0 = "restart"
)

// PresenceV0 makes omission semantics visible to editors. A pointer is optional;
// non-pointer scalar/container values are always materialized after decoding.
type PresenceV0 string

const (
	PresenceOptionalV0 PresenceV0 = "optional"
	PresenceRequiredV0 PresenceV0 = "required"
)

type ConstraintsV0 struct {
	Minimum   *float64 `json:"minimum,omitempty"`
	Maximum   *float64 `json:"maximum,omitempty"`
	MinLength *int     `json:"min_length,omitempty"`
	MaxLength *int     `json:"max_length,omitempty"`
	Pattern   string   `json:"pattern,omitempty"`
	Enum      []string `json:"enum,omitempty"`
	MinItems  *int     `json:"min_items,omitempty"`
	MaxItems  *int     `json:"max_items,omitempty"`
}

// LeafPolicyV0 is required exactly once per JSON leaf, including leaves that
// are not currently present in a decoded document.
type LeafPolicyV0 struct {
	Pointer           string            `json:"pointer"`
	Presence          PresenceV0        `json:"presence"`
	Constraints       ConstraintsV0     `json:"constraints"`
	Sensitive         bool              `json:"sensitive"`
	Editable          bool              `json:"editable"`
	EditabilityReason string            `json:"editability_reason"`
	RestartBehavior   RestartBehaviorV0 `json:"restart_behavior"`
	SetterRef         string            `json:"setter_ref"`
}

type CatalogEntryV0 struct {
	Pointer string `json:"pointer"`
	Type    string `json:"type"`
	LeafPolicyV0
}
type JSONLeafV0 struct {
	Pointer  string
	Type     string
	Presence PresenceV0
}

func CloneCatalogV0(entries []CatalogEntryV0) []CatalogEntryV0 {
	out := append([]CatalogEntryV0(nil), entries...)
	for index := range out {
		out[index].Constraints.Enum = append([]string(nil), out[index].Constraints.Enum...)
		out[index].Constraints.Minimum = cloneFloat64PointerV0(out[index].Constraints.Minimum)
		out[index].Constraints.Maximum = cloneFloat64PointerV0(out[index].Constraints.Maximum)
		out[index].Constraints.MinLength = cloneIntPointerV0(out[index].Constraints.MinLength)
		out[index].Constraints.MaxLength = cloneIntPointerV0(out[index].Constraints.MaxLength)
		out[index].Constraints.MinItems = cloneIntPointerV0(out[index].Constraints.MinItems)
		out[index].Constraints.MaxItems = cloneIntPointerV0(out[index].Constraints.MaxItems)
	}
	return out
}

func cloneFloat64PointerV0(value *float64) *float64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneIntPointerV0(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

type catalogFieldV0 struct {
	name     string
	tagged   bool
	optional bool
	typ      reflect.Type
	index    []int
}

// BuildCatalogV0 mirrors encoding/json's exported-field, anonymous-field and
// collision rules before requiring a complete, one-to-one policy set.
func BuildCatalogV0(schema any, policies []LeafPolicyV0) ([]CatalogEntryV0, error) {
	leaves, err := reflectedLeavesV0(schema)
	if err != nil {
		return nil, err
	}
	byPointer := make(map[string]LeafPolicyV0, len(policies))
	for _, policy := range policies {
		if err := validatePolicyV0(policy); err != nil {
			return nil, err
		}
		if _, exists := byPointer[policy.Pointer]; exists {
			return nil, fmt.Errorf("orquesta-config: duplicate policy for %s", policy.Pointer)
		}
		byPointer[policy.Pointer] = policy
	}
	entries := make([]CatalogEntryV0, 0, len(leaves))
	for pointer, leaf := range leaves {
		policy, ok := byPointer[pointer]
		if !ok {
			return nil, fmt.Errorf("orquesta-config: missing explicit policy for %s", pointer)
		}
		if policy.Presence != leaf.presence {
			return nil, fmt.Errorf("orquesta-config: policy presence does not match JSON leaf %s", pointer)
		}
		entries = append(entries, CatalogEntryV0{Pointer: pointer, Type: jsonTypeV0(leaf.typ), LeafPolicyV0: policy})
		delete(byPointer, pointer)
	}
	for pointer := range byPointer {
		return nil, fmt.Errorf("orquesta-config: policy does not match a JSON leaf: %s", pointer)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Pointer < entries[j].Pointer })
	return entries, nil
}

// ReflectJSONLeavesV0 exposes the exact JSON leaves selected by the same
// reflection logic as BuildCatalogV0, including anonymous-field collisions.
func ReflectJSONLeavesV0(schema any) ([]JSONLeafV0, error) {
	leaves, err := reflectedLeavesV0(schema)
	if err != nil {
		return nil, err
	}
	out := make([]JSONLeafV0, 0, len(leaves))
	for pointer, leaf := range leaves {
		out = append(out, JSONLeafV0{Pointer: pointer, Type: jsonTypeV0(leaf.typ), Presence: leaf.presence})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Pointer < out[j].Pointer })
	return out, nil
}

type reflectedLeafV0 struct {
	typ      reflect.Type
	presence PresenceV0
}

func reflectedLeavesV0(schema any) (map[string]reflectedLeafV0, error) {
	t := reflect.TypeOf(schema)
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == nil || t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("orquesta-config: catalog schema must be a struct")
	}
	leaves := map[string]reflectedLeafV0{}
	if err := collectObjectLeavesV0(t, "", leaves, false, map[reflect.Type]bool{}); err != nil {
		return nil, err
	}
	return leaves, nil
}

func collectObjectLeavesV0(t reflect.Type, prefix string, leaves map[string]reflectedLeafV0, inheritedOptional bool, ancestors map[reflect.Type]bool) error {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if ancestors[t] {
		return fmt.Errorf("orquesta-config: recursive JSON schema at %s", prefix)
	}
	ancestors[t] = true
	defer delete(ancestors, t)
	fields := visibleFieldsV0(t)
	for _, field := range fields {
		fieldType := field.typ
		optional := inheritedOptional || field.optional || typeIsPointerV0(fieldType)
		base := fieldType
		for base.Kind() == reflect.Pointer {
			base = base.Elem()
		}
		pointer := prefix + "/" + escapePointerTokenV0(field.name)
		if base.Kind() == reflect.Struct {
			if err := collectObjectLeavesV0(base, pointer, leaves, optional, ancestors); err != nil {
				return err
			}
			continue
		}
		if _, exists := leaves[pointer]; exists {
			return fmt.Errorf("orquesta-config: duplicate JSON pointer %s", pointer)
		}
		presence := PresenceRequiredV0
		if optional {
			presence = PresenceOptionalV0
		}
		leaves[pointer] = reflectedLeafV0{typ: fieldType, presence: presence}
	}
	return nil
}

// visibleFieldsV0 follows the dominance rule used by encoding/json: at a
// depth, a sole tagged field wins; otherwise ambiguity omits all colliding fields.
func visibleFieldsV0(t reflect.Type) []catalogFieldV0 {
	var candidates []catalogFieldV0
	var walk func(reflect.Type, []int, bool, map[reflect.Type]bool)
	walk = func(current reflect.Type, index []int, inheritedOptional bool, seen map[reflect.Type]bool) {
		for current.Kind() == reflect.Pointer {
			current = current.Elem()
		}
		if seen[current] {
			return
		}
		seen[current] = true
		defer delete(seen, current)
		for i := 0; i < current.NumField(); i++ {
			f := current.Field(i)
			if f.PkgPath != "" && !f.Anonymous {
				continue
			}
			tag, tagOptions := splitJSONTagV0(f.Tag.Get("json"))
			if tag == "-" {
				continue
			}
			ft := f.Type
			base := ft
			for base.Kind() == reflect.Pointer {
				base = base.Elem()
			}
			if f.Anonymous && tag == "" && base.Kind() == reflect.Struct {
				walk(base, append(append([]int(nil), index...), i), inheritedOptional || tagOptions["omitempty"] || typeIsPointerV0(ft), seen)
				continue
			}
			if f.PkgPath != "" {
				continue
			}
			name, tagged := tag, tag != ""
			if name == "" {
				name = f.Name
			}
			candidates = append(candidates, catalogFieldV0{name: name, tagged: tagged, optional: inheritedOptional || tagOptions["omitempty"] || typeIsPointerV0(ft), typ: ft, index: append(append([]int(nil), index...), i)})
		}
	}
	walk(t, nil, false, map[reflect.Type]bool{})
	byName := map[string][]catalogFieldV0{}
	for _, field := range candidates {
		byName[field.name] = append(byName[field.name], field)
	}
	var out []catalogFieldV0
	for _, sameName := range byName {
		sort.Slice(sameName, func(i, j int) bool {
			if len(sameName[i].index) != len(sameName[j].index) {
				return len(sameName[i].index) < len(sameName[j].index)
			}
			return sameName[i].tagged && !sameName[j].tagged
		})
		first := sameName[0]
		if len(sameName) > 1 && len(sameName[1].index) == len(first.index) && sameName[1].tagged == first.tagged {
			continue
		}
		out = append(out, first)
	}
	return out
}

func validatePolicyV0(p LeafPolicyV0) error {
	if !strings.HasPrefix(p.Pointer, "/") || strings.TrimSpace(p.Pointer) == "/" {
		return fmt.Errorf("orquesta-config: invalid JSON pointer %q", p.Pointer)
	}
	if p.Presence != PresenceOptionalV0 && p.Presence != PresenceRequiredV0 {
		return fmt.Errorf("orquesta-config: policy %s has invalid presence", p.Pointer)
	}
	if strings.TrimSpace(p.EditabilityReason) == "" {
		return fmt.Errorf("orquesta-config: policy %s requires editability_reason", p.Pointer)
	}
	if p.Editable && strings.TrimSpace(p.SetterRef) == "" {
		return fmt.Errorf("orquesta-config: editable policy %s requires a real setter_ref", p.Pointer)
	}
	if !p.Editable && strings.TrimSpace(p.SetterRef) != "" {
		return fmt.Errorf("orquesta-config: non-editable policy %s must not claim a setter_ref", p.Pointer)
	}
	switch p.RestartBehavior {
	case RestartBehaviorNoneV0, RestartBehaviorReloadV0, RestartBehaviorRestartV0:
		return nil
	default:
		return fmt.Errorf("orquesta-config: policy %s has invalid restart_behavior", p.Pointer)
	}
}
func splitJSONTagV0(tag string) (string, map[string]bool) {
	parts := strings.Split(tag, ",")
	options := make(map[string]bool, len(parts)-1)
	for _, option := range parts[1:] {
		options[option] = true
	}
	return parts[0], options
}
func typeIsPointerV0(t reflect.Type) bool { return t.Kind() == reflect.Pointer }
func escapePointerTokenV0(v string) string {
	return strings.NewReplacer("~", "~0", "/", "~1").Replace(v)
}
func jsonTypeV0(t reflect.Type) string {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Map:
		return "object"
	default:
		return t.Kind().String()
	}
}

// PresenceForTypeV0 exposes the convention used by composition policy tables.
func PresenceForTypeV0(t reflect.Type) PresenceV0 {
	for t.Kind() == reflect.Pointer {
		return PresenceOptionalV0
	}
	return PresenceRequiredV0
}
