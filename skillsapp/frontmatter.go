package skillsapp

import (
	"fmt"
	"os"
	"strings"
)

func readSkillFrontmatter(path string) (string, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	return parseSkillFrontmatter(string(data))
}

func parseSkillFrontmatter(content string) (string, string, error) {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", "", fmt.Errorf("SKILL.md sin frontmatter YAML")
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end <= 1 {
		return "", "", fmt.Errorf("frontmatter YAML invalido")
	}
	block := lines[1:end]
	name, _ := extractFrontmatterField(block, "name")
	description, _ := extractFrontmatterField(block, "description")
	return name, description, nil
}

func extractFrontmatterField(lines []string, key string) (string, bool) {
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}
		if !strings.HasPrefix(trimmed, key+":") {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(trimmed, key+":"))
		switch value {
		case "|", ">":
			var block []string
			for j := i + 1; j < len(lines); j++ {
				next := lines[j]
				if strings.TrimSpace(next) == "" {
					block = append(block, "")
					continue
				}
				if !strings.HasPrefix(next, " ") && !strings.HasPrefix(next, "\t") {
					break
				}
				block = append(block, strings.TrimSpace(next))
			}
			return strings.TrimSpace(strings.Join(block, " ")), true
		default:
			return trimYAMLScalar(value), true
		}
	}
	return "", false
}

func trimYAMLScalar(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 2 {
		if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
			return strings.TrimSpace(v[1 : len(v)-1])
		}
	}
	return v
}
