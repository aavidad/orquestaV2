package skillspolicy

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const DefaultPriority = 100

const (
	OriginBuiltin    = "builtin"
	OriginLocal      = "local"
	OriginThirdParty = "third_party"
	RiskLow          = "bajo"
	RiskMedium       = "medio"
	RiskHigh         = "alto"
)

type SkillSnapshot struct {
	AgentType        string
	Name             string
	Description      string
	WhenToUse        string
	Scenario         string
	Priority         int
	AliasesJSON      string
	ToolsJSON        string
	Origin           string
	RiskLevel        string
	RequiresApproval bool
	Active           bool
}

func Normalize(skill *SkillSnapshot) error {
	if skill == nil {
		return fmt.Errorf("skill nula")
	}
	skill.AgentType = strings.TrimSpace(skill.AgentType)
	skill.Name = strings.TrimSpace(skill.Name)
	skill.Description = strings.TrimSpace(skill.Description)
	skill.WhenToUse = strings.TrimSpace(skill.WhenToUse)
	skill.Scenario = strings.ToLower(strings.TrimSpace(skill.Scenario))
	skill.Origin = strings.ToLower(strings.TrimSpace(skill.Origin))
	skill.RiskLevel = strings.ToLower(strings.TrimSpace(skill.RiskLevel))
	if skill.Priority <= 0 {
		skill.Priority = DefaultPriority
	}
	if skill.Origin == "" {
		skill.Origin = OriginBuiltin
	}
	switch skill.Origin {
	case OriginBuiltin, OriginLocal, OriginThirdParty:
	default:
		return fmt.Errorf("origen invalido: %s", skill.Origin)
	}
	if skill.RiskLevel == "" {
		if skill.Origin == OriginBuiltin {
			skill.RiskLevel = RiskLow
		} else {
			skill.RiskLevel = RiskMedium
		}
	}
	switch skill.RiskLevel {
	case RiskLow, RiskMedium, RiskHigh:
	default:
		return fmt.Errorf("nivel_riesgo invalido: %s", skill.RiskLevel)
	}

	var err error
	skill.AliasesJSON, err = NormalizeListJSON(skill.AliasesJSON)
	if err != nil {
		return fmt.Errorf("aliases_json invalido: %w", err)
	}
	skill.ToolsJSON, err = NormalizeListJSON(skill.ToolsJSON)
	if err != nil {
		return fmt.Errorf("herramientas_json invalido: %w", err)
	}
	if skill.AgentType == "" || skill.Name == "" {
		return fmt.Errorf("tipo_agente y nombre son obligatorios")
	}
	return nil
}

func NormalizeListJSON(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "[]", nil
	}
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return "", err
	}
	normalized := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		canonical := strings.ToLower(trimmed)
		if _, ok := seen[canonical]; ok {
			continue
		}
		seen[canonical] = struct{}{}
		normalized = append(normalized, canonical)
	}
	sort.Strings(normalized)
	data, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func ParseListJSON(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			out = append(out, strings.ToLower(trimmed))
		}
	}
	sort.Strings(out)
	return out
}

func SameNaturalKey(a, b *SkillSnapshot) bool {
	if a == nil || b == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(a.AgentType), strings.TrimSpace(b.AgentType)) &&
		strings.EqualFold(strings.TrimSpace(a.Name), strings.TrimSpace(b.Name))
}

func SkillsEquivalent(a, b *SkillSnapshot) bool {
	if a == nil || b == nil {
		return false
	}
	keysA := identityKeys(a)
	for key := range identityKeys(b) {
		if _, ok := keysA[key]; ok && key != "" {
			return true
		}
	}
	if strings.TrimSpace(a.Scenario) != "" &&
		strings.EqualFold(strings.TrimSpace(a.Scenario), strings.TrimSpace(b.Scenario)) &&
		stringListsEqual(ParseListJSON(a.ToolsJSON), ParseListJSON(b.ToolsJSON)) &&
		normalizeComparableText(a.WhenToUse) != "" &&
		normalizeComparableText(a.WhenToUse) == normalizeComparableText(b.WhenToUse) {
		return true
	}
	return false
}

func IsExternal(skill *SkillSnapshot) bool {
	if skill == nil {
		return false
	}
	return skill.Origin == OriginLocal || skill.Origin == OriginThirdParty
}

func identityKeys(skill *SkillSnapshot) map[string]struct{} {
	keys := map[string]struct{}{}
	if skill == nil {
		return keys
	}
	for _, raw := range append([]string{skill.Name}, ParseListJSON(skill.AliasesJSON)...) {
		key := normalizeKey(raw)
		if key == "" {
			continue
		}
		keys[key] = struct{}{}
	}
	return keys
}

func normalizeKey(v string) string {
	parts := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(v)), func(r rune) bool {
		switch r {
		case ' ', '-', '_', '/', '\\', '.':
			return true
		default:
			return false
		}
	})
	return strings.Join(parts, "")
}

func normalizeComparableText(v string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(v))), " ")
}

func stringListsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
