package gobernanzapolicy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

type RuleSnapshot struct {
	ID          int64
	Category    string
	Title       string
	Description string
}

type SkillSnapshot struct {
	ID          int64
	Name        string
	Description string
	WhenToUse   string
	Priority    int
}

type WorkflowSnapshot struct {
	ID          int64
	Name        string
	Description string
	Steps       string
}

type CatalogSnapshot struct {
	AgentType  string
	ProjectID  *int64
	Resolution string
	Hash       string
	Rules      []RuleSnapshot
	Skills     []SkillSnapshot
	Workflows  []WorkflowSnapshot
}

func HashCatalog(catalog *CatalogSnapshot) string {
	if catalog == nil {
		return ""
	}
	input := struct {
		AgentType  string   `json:"tipo_agente"`
		Resolution string   `json:"resolucion_actual"`
		Rules      []string `json:"reglas"`
		Skills     []string `json:"skills"`
		Workflows  []string `json:"workflows"`
	}{
		AgentType:  catalog.AgentType,
		Resolution: strings.TrimSpace(catalog.Resolution),
	}
	for _, rule := range catalog.Rules {
		input.Rules = append(input.Rules, fmt.Sprintf("%d|%s|%s|%s", rule.ID, rule.Category, rule.Title, rule.Description))
	}
	for _, skill := range catalog.Skills {
		input.Skills = append(input.Skills, fmt.Sprintf("%d|%s|%s|%s|%d", skill.ID, skill.Name, skill.Description, skill.WhenToUse, skill.Priority))
	}
	for _, workflow := range catalog.Workflows {
		input.Workflows = append(input.Workflows, fmt.Sprintf("%d|%s|%s|%s", workflow.ID, workflow.Name, workflow.Description, workflow.Steps))
	}
	data, _ := json.Marshal(input)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:12])
}

func BuildContextSummaryFromCatalog(catalog *CatalogSnapshot) (map[string]any, string) {
	if catalog == nil {
		return nil, ""
	}
	context := map[string]any{
		"tipo_agente":       strings.TrimSpace(catalog.AgentType),
		"hash":              strings.TrimSpace(catalog.Hash),
		"reglas":            len(catalog.Rules),
		"skills":            len(catalog.Skills),
		"workflows":         len(catalog.Workflows),
		"resolucion_actual": strings.TrimSpace(catalog.Resolution),
	}
	if catalog.ProjectID != nil {
		context["proyecto_id"] = *catalog.ProjectID
	}
	summary := fmt.Sprintf(
		"Catálogo efectivo %s (%d reglas, %d skills, %d workflows)",
		strings.TrimSpace(catalog.Hash),
		len(catalog.Rules),
		len(catalog.Skills),
		len(catalog.Workflows),
	)
	return context, summary
}
