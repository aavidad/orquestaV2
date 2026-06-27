package orquestadirectoragent

import "strings"

func directorAgentRefCompactV0(value string) bool {
	if len(value) < 2 || len(value) > 512 {
		return false
	}
	for i := 0; i < len(value); i++ {
		b := value[i]
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') {
			continue
		}
		if b == '.' || b == '_' || b == ':' || b == '-' {
			continue
		}
		return false
	}
	return true
}

func directorAgentContextRefCompactV0(value string) bool {
	if strings.TrimSpace(value) == "" || len(value) > maxDirectorAgentStringV0 {
		return false
	}
	return !strings.ContainsAny(value, " /\\\t\n\r")
}
