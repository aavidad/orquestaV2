package skillspolicy

import "testing"

func TestNormalizeSkillAppliesDefaults(t *testing.T) {
	skill := &SkillSnapshot{
		AgentType:   "programador",
		Name:        "rg",
		AliasesJSON: `["RipGrep","ripgrep"]`,
		ToolsJSON:   `["RG","rg"]`,
	}
	if err := Normalize(skill); err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if skill.Priority != DefaultPriority || skill.Origin != OriginBuiltin || skill.RiskLevel != RiskLow {
		t.Fatalf("defaults inesperados: %+v", skill)
	}
	if skill.AliasesJSON != `["ripgrep"]` || skill.ToolsJSON != `["rg"]` {
		t.Fatalf("listas normalizadas inesperadas: %+v", skill)
	}
}

func TestSkillsEquivalentByAliasAndTooling(t *testing.T) {
	a := &SkillSnapshot{
		AgentType:   "programador",
		Name:        "rg",
		AliasesJSON: `["ripgrep"]`,
		ToolsJSON:   `["rg"]`,
		Scenario:    "investigacion",
		WhenToUse:   "buscar texto",
	}
	b := &SkillSnapshot{
		AgentType: "programador",
		Name:      "ripgrep",
		ToolsJSON: `["rg"]`,
		Scenario:  "investigacion",
		WhenToUse: "buscar texto",
	}
	if !SkillsEquivalent(a, b) {
		t.Fatalf("las skills deberian considerarse equivalentes")
	}
}

func TestSameNaturalKeyAndExternalClassification(t *testing.T) {
	a := &SkillSnapshot{AgentType: "programador", Name: "rg"}
	b := &SkillSnapshot{AgentType: "PROGRAMADOR", Name: "RG"}
	if !SameNaturalKey(a, b) {
		t.Fatalf("la natural key deberia ser equivalente")
	}
	if !IsExternal(&SkillSnapshot{Origin: OriginThirdParty}) {
		t.Fatalf("third_party deberia considerarse externo")
	}
}
