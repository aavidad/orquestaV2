package main

import (
	"os"
	"path/filepath"
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestaserver "orquesta/modulos/orquesta-server"
)

const serverCuratedSkillMaxBytesV0 = 32 * 1024

func (supervisor serverStackSupervisorV0) withCuratedSkillRefsV0(
	request orquestaserver.IdleSelfImprovementRequestV0,
) orquestaserver.IdleSelfImprovementRequestV0 {
	if len(supervisor.curatedSkills) == 0 {
		return request
	}
	result := orquestaautoprogramming.MatchAutoprogrammingCuratedSkillRefsV0(
		supervisor.curatedSkills,
		orquestaautoprogramming.AutoprogrammingCuratedSkillMatchRequestV0{
			Area:               request.SuggestedArea,
			Objective:          request.FailureSummary,
			FailureSummary:     request.FailureSummary,
			WriteSet:           append([]string(nil), request.WriteSet...),
			RequiredTests:      append([]string(nil), request.RequiredTests...),
			AcceptanceCriteria: append([]string(nil), request.AcceptanceCriteria...),
			ContextRefs:        append([]string(nil), request.ContextRefs...),
		},
	)
	request.SkillRefs = compactServerStackStringsV0(append(request.SkillRefs, result.SkillRefs...))
	request.ContextRefs = compactServerStackStringsV0(append(request.ContextRefs, result.ContextRefs...))
	if len(result.SkillRefs) > 0 {
		request.EvidenceRefs = compactServerStackStringsV0(append(
			request.EvidenceRefs,
			"evidence-ref-autoprogramming-curated-skills-matched",
		))
	}
	return request
}

func serverCuratedSkillsFromProjectV0(
	projectWorkDir string,
) []orquestaautoprogramming.AutoprogrammingCuratedSkillV0 {
	projectWorkDir = strings.TrimSpace(projectWorkDir)
	if projectWorkDir == "" {
		return nil
	}
	entries, err := os.ReadDir(filepath.Join(projectWorkDir, "skills"))
	if err != nil {
		return nil
	}
	skills := make([]orquestaautoprogramming.AutoprogrammingCuratedSkillV0, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(projectWorkDir, "skills", entry.Name(), "SKILL.md")
		info, err := os.Stat(path)
		if err != nil || info.IsDir() || info.Size() > serverCuratedSkillMaxBytesV0 {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if skill, ok := serverCuratedSkillFromMarkdownV0(entry.Name(), string(data)); ok {
			if len(orquestaautoprogramming.ValidateAutoprogrammingCuratedSkillCatalogV0(
				[]orquestaautoprogramming.AutoprogrammingCuratedSkillV0{skill},
			)) == 0 {
				skills = append(skills, skill)
			}
		}
	}
	if len(skills) == 0 {
		return nil
	}
	return skills
}

func serverCuratedSkillFromMarkdownV0(
	dirName string,
	content string,
) (orquestaautoprogramming.AutoprogrammingCuratedSkillV0, bool) {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		name := strings.TrimSpace(dirName)
		if name == "" {
			return orquestaautoprogramming.AutoprogrammingCuratedSkillV0{}, false
		}
		return orquestaautoprogramming.AutoprogrammingCuratedSkillV0{
			Name:     name,
			SkillRef: orquestaautoprogramming.AutoprogrammingCuratedSkillRefV0(name),
		}, true
	}
	values := map[string]string{}
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "---" {
			break
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(strings.ToLower(key))
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		switch key {
		case "name", "description", "tags":
			values[key] = value
		}
	}
	name := strings.TrimSpace(values["name"])
	if name == "" {
		name = strings.TrimSpace(dirName)
	}
	if name == "" {
		return orquestaautoprogramming.AutoprogrammingCuratedSkillV0{}, false
	}
	return orquestaautoprogramming.AutoprogrammingCuratedSkillV0{
		Name:        name,
		SkillRef:    orquestaautoprogramming.AutoprogrammingCuratedSkillRefV0(name),
		Description: values["description"],
		Tags:        serverCuratedSkillTagsV0(values["tags"]),
	}, true
}

func serverCuratedSkillTagsV0(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';'
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.Trim(strings.TrimSpace(part), `[]"'`)
		if part != "" {
			out = append(out, part)
		}
	}
	return compactServerStackStringsV0(out)
}
