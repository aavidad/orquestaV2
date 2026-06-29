package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSelfProgrammingDeployContractV0NoMontaProduccionNiExponePuertoPublicoV0(t *testing.T) {
	root := filepath.Join("..", "..")
	compose := mustReadSelfProgrammingDeployFileForTestV0(t, root, "deploy", "self-programming", "docker-compose.yml")
	envExample := mustReadSelfProgrammingDeployFileForTestV0(t, root, "deploy", "self-programming", "orquesta-self.env.example")
	dockerfiles := []string{
		mustReadSelfProgrammingDeployFileForTestV0(t, root, "Dockerfile"),
		mustReadSelfProgrammingDeployFileForTestV0(t, root, "Dockerfile.dev"),
		mustReadSelfProgrammingDeployFileForTestV0(t, root, "Dockerfile.self-programming"),
	}

	for _, forbidden := range []string{
		"/home/berserk/deploy/opes",
		"/var/run/docker.sock",
		"/home/orquesta",
		"uso-app",
		"opes-api",
		"0.0.0.0:19039:19039",
		"ORQUESTA_OPES_BRIDGE_DRY_RUN=",
		"ORQUESTA_OPES_REGISTRY_FINALPKG_DRY_RUN=",
	} {
		if strings.Contains(compose, forbidden) || strings.Contains(envExample, forbidden) {
			t.Fatalf("deploy self-programming contiene referencia prohibida %q", forbidden)
		}
	}
	for _, dockerfile := range dockerfiles {
		if strings.Contains(dockerfile, "/home/orquesta") {
			t.Fatalf("Dockerfile activo contiene HOME no aislado en /workspace")
		}
	}
	for _, required := range []string{
		"127.0.0.1:19039:19039",
		"/srv/orquesta-self/state:/workspace/state",
		"/srv/orquesta-self/runtime:/workspace/runtime",
		"/srv/orquesta-self/worktrees/orquesta:/workspace/project",
		"/srv/orquesta-self/home:/workspace/home",
		"/srv/orquesta-self/codex-home:/workspace/codex-home",
		"/srv/orquesta-self/cache/go:/workspace/cache/go",
		"/srv/orquesta-self/cache/gomod:/workspace/go/pkg/mod",
		"ORQUESTA_SERVER_SELF_PROGRAMMING_ONLY=true",
		"ORQUESTA_SERVER_SELF_PROGRAMMING_ROOT=/workspace",
		"ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux",
		"ORQUESTA_OPES_BASE_URL=",
		"OPES_BASE_URL=",
		"ORQUESTA_DOMAIN_WORK_HTTP_BASE_URL=",
	} {
		if !strings.Contains(compose+"\n"+envExample, required) {
			t.Fatalf("deploy self-programming no contiene requisito %q", required)
		}
	}
}

func mustReadSelfProgrammingDeployFileForTestV0(t *testing.T, parts ...string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(parts...))
	if err != nil {
		t.Fatalf("read deploy file %v: %v", parts, err)
	}
	return string(data)
}
