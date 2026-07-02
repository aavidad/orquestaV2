package selfprogramming

import (
	"os"
	"strings"
	"testing"
)

func TestSelfProgrammingComposeV0IsRemoteSafe(t *testing.T) {
	compose := readContractFileV0(t, "docker-compose.yml")

	for _, fragment := range []string{
		"/var/run/docker.sock",
		"/run/docker.sock",
		"/home/berserk/deploy/opes",
		"uso-app",
		"network_mode: host",
		"pid: host",
		"ipc: host",
		"privileged: true",
		"sudo ",
	} {
		if strings.Contains(compose, fragment) {
			t.Fatalf("docker-compose.yml must not contain %q", fragment)
		}
	}

	if got := yamlScalarValueV0(compose, "user"); got != "10001:10001" {
		t.Fatalf("user=%q, want non-root 10001:10001", got)
	}
	if got := yamlScalarValueV0(compose, "read_only"); got != "true" {
		t.Fatalf("read_only=%q, want true", got)
	}

	ports := yamlListValuesV0(compose, "ports")
	if len(ports) != 1 || ports[0] != "127.0.0.1:19039:19039" {
		t.Fatalf("ports=%v, want only loopback 127.0.0.1:19039:19039", ports)
	}

	envFiles := yamlListValuesV0(compose, "env_file")
	if len(envFiles) != 1 || envFiles[0] != "/srv/orquesta-self/orquesta-self.env" {
		t.Fatalf("env_file=%v, want isolated /srv/orquesta-self env", envFiles)
	}

	volumes := yamlListValuesV0(compose, "volumes")
	if len(volumes) == 0 {
		t.Fatalf("expected explicit bind mounts")
	}
	for _, volume := range volumes {
		source := hostBindSourceV0(volume)
		if !strings.HasPrefix(source, "/srv/orquesta-self/") {
			t.Fatalf("volume %q uses host source %q outside /srv/orquesta-self", volume, source)
		}
	}

	if !containsValueV0(yamlListValuesV0(compose, "security_opt"), "no-new-privileges:true") {
		t.Fatalf("security_opt must include no-new-privileges:true")
	}
	if !containsValueV0(yamlListValuesV0(compose, "cap_drop"), "ALL") {
		t.Fatalf("cap_drop must include ALL")
	}
}

func TestSelfProgrammingEnvV0KeepsProductionAndFallbacksOff(t *testing.T) {
	env := parseEnvExampleV0(t, readContractFileV0(t, "orquesta-self.env.example"))

	want := map[string]string{
		"ORQUESTA_SERVER_SELF_PROGRAMMING_ONLY":             "true",
		"ORQUESTA_SERVER_SELF_PROGRAMMING_ROOT":             "/workspace",
		"ORQUESTA_SERVER_ADDR":                              "0.0.0.0:19039",
		"ORQUESTA_SERVER_REMOTE_CONTROL_PLANE_CONFIRM":      "true",
		"ORQUESTA_SERVER_CONTROL_TOKEN":                     "",
		"ORQUESTA_CODEX_PROJECT_WORKDIR":                    "/workspace/project",
		"ORQUESTA_CODEX_RUNTIME_WORKDIR":                    "/workspace/runtime",
		"ORQUESTA_SERVER_STATE_DIR":                         "/workspace/state",
		"ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_ENABLED": "false",
		"ORQUESTA_CODEX_GOAL_BACKEND":                       "app_server_tmux",
		"ORQUESTA_CODEX_SANDBOX":                            "workspace-write",
		"ORQUESTA_CODEX_APPROVAL_POLICY":                    "never",
		"ORQUESTA_OPES_BASE_URL":                            "",
		"OPES_BASE_URL":                                     "",
		"ORQUESTA_OPES_BRIDGE_ENABLED":                      "false",
		"ORQUESTA_OPES_BRIDGE_CONFIRM":                      "false",
		"ORQUESTA_OPES_REGISTRY_FINALPKG_ENABLED":           "false",
		"ORQUESTA_OPES_REGISTRY_FINALPKG_CONFIRM":           "false",
		"ORQUESTA_OPES_REGISTRY_FINALPKG_REGISTRY_PATH":     "",
		"ORQUESTA_OPES_REGISTRY_FINALPKG_COURSE_ROOT":       "",
		"ORQUESTA_OPES_TOPIC_REGISTRY_ENABLED":              "false",
		"ORQUESTA_OPES_TOPIC_REGISTRY_TOOL_PATH":            "",
		"ORQUESTA_DOMAIN_WORK_HTTP_BASE_URL":                "",
		"ORQUESTA_DOMAIN_WORK_FILE_ENABLED":                 "false",
	}
	for key, expected := range want {
		if got, ok := env[key]; !ok || got != expected {
			t.Fatalf("%s=%q, want %q (present=%v)", key, got, expected, ok)
		}
	}

	for key, value := range env {
		for _, fragment := range []string{
			"/home/berserk/deploy/opes",
			"/var/run/docker.sock",
			"uso-app",
		} {
			if strings.Contains(value, fragment) {
				t.Fatalf("%s must not point at %q", key, fragment)
			}
		}
	}
}

func TestSelfProgrammingDocsV0KeepRemoteSafetyRunbook(t *testing.T) {
	readme := readContractFileV0(t, "README.md")
	for _, snippet := range []string{
		"No despliega OPES",
		"/var/run/docker.sock",
		"no publica puertos externos",
		"sudo install -d -o 10001 -g 10001 /srv/orquesta-self/state",
		"127.0.0.1:19039",
		"No se copian credenciales de produccion",
		"go test -count=1 ./deploy/self-programming",
		"ReadonlyRootfs",
	} {
		if !strings.Contains(readme, snippet) {
			t.Fatalf("README.md must document %q", snippet)
		}
	}

	manual := readContractFileV0(t, "manual_codex_context.md")
	for _, snippet := range []string{
		"No uses root dentro del contenedor",
		"No montes `/var/run/docker.sock`",
		"sudo` solo para el contenedor",
	} {
		if !strings.Contains(manual, snippet) {
			t.Fatalf("manual_codex_context.md must document %q", snippet)
		}
	}
}

func readContractFileV0(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func yamlListValuesV0(raw string, key string) []string {
	var values []string
	lines := strings.Split(raw, "\n")
	keyLine := key + ":"
	for index := 0; index < len(lines); index++ {
		if strings.TrimSpace(lines[index]) != keyLine {
			continue
		}
		keyIndent := leadingSpacesV0(lines[index])
		for child := index + 1; child < len(lines); child++ {
			line := lines[child]
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				continue
			}
			if leadingSpacesV0(line) <= keyIndent {
				break
			}
			if strings.HasPrefix(trimmed, "- ") {
				values = append(values, trimYAMLScalarV0(strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))))
			}
		}
	}
	return values
}

func yamlScalarValueV0(raw string, key string) string {
	prefix := key + ":"
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) {
			return trimYAMLScalarV0(strings.TrimSpace(strings.TrimPrefix(trimmed, prefix)))
		}
	}
	return ""
}

func trimYAMLScalarV0(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "\"'")
	return value
}

func leadingSpacesV0(value string) int {
	return len(value) - len(strings.TrimLeft(value, " "))
}

func hostBindSourceV0(value string) string {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) < 2 {
		return ""
	}
	return parts[0]
}

func containsValueV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func parseEnvExampleV0(t *testing.T, raw string) map[string]string {
	t.Helper()
	env := map[string]string{}
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		key, value, ok := strings.Cut(trimmed, "=")
		if !ok {
			t.Fatalf("invalid env line %q", line)
		}
		env[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return env
}
