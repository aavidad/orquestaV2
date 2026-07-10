package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestDegradedIdentityHTTPHandlerV0PublicaSoloDiagnosticoRedactadoV0(t *testing.T) {
	handler := newDegradedIdentityHTTPHandlerV0(orquestaserver.ConfigV0{
		ProjectWorkDir: "/private/raw/worktree",
		RuntimeIdentity: orquestaserver.ServerRuntimeIdentityV0{
			BinaryPath:   "/private/raw/bin/orquesta-server",
			BinarySHA256: strings.Repeat("a", 64),
			CommitRef:    strings.Repeat("b", 40),
		},
	}, serverWorktreeIdentityRemoteV0)

	status := httptest.NewRecorder()
	handler.ServeHTTP(status, httptest.NewRequest(http.MethodGet, "/api/status", nil))
	body := status.Body.String()
	if status.Code != http.StatusOK || !strings.Contains(body, `"status":"degraded_identity"`) ||
		!strings.Contains(body, `"startup_ready":false`) {
		t.Fatalf("status=%d body=%s", status.Code, body)
	}
	for _, secret := range []string{"/private/raw", strings.Repeat("a", 64), strings.Repeat("b", 40), serverCanonicalGitRemoteURLV0} {
		if strings.Contains(body, secret) {
			t.Fatalf("status filtra identidad cruda %q: %s", secret, body)
		}
	}

	readiness := httptest.NewRecorder()
	handler.ServeHTTP(readiness, httptest.NewRequest(http.MethodGet, orquestaserver.ServerReadinessEndpointV0, nil))
	if readiness.Code != http.StatusServiceUnavailable || strings.Contains(readiness.Body.String(), strings.Repeat("a", 64)) {
		t.Fatalf("readiness=%d body=%s", readiness.Code, readiness.Body.String())
	}

	health := httptest.NewRecorder()
	handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/health", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("health=%d", health.Code)
	}
}

func TestDegradedIdentityHTTPHandlerV0NormalizaReasonNoTipadoSinFugasV0(t *testing.T) {
	secret := "/private/worktree?token=secret"
	handler := newDegradedIdentityHTTPHandlerV0(orquestaserver.ConfigV0{}, secret)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v0/apps/director", nil))
	if response.Code != http.StatusServiceUnavailable || strings.Contains(response.Body.String(), secret) ||
		!strings.Contains(response.Body.String(), serverWorktreeIdentityInvalidV0) {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestDegradedIdentityHTTPHandlerV0BloqueaTodasLasEntradasAmpliasConContratoPublicoV0(t *testing.T) {
	handler := newDegradedIdentityHTTPHandlerV0(orquestaserver.ConfigV0{}, serverWorktreeIdentityInvalidV0)
	paths := []string{
		orquestamcp.MCPAutoprogrammingPrepareRunHTTPPathV0,
		"/api/v0/apps/director",
		"/api/v0/apps/director/goal/observe",
		"/api/v0/autoprogramming/supervise",
		"/api/v0/autoprogramming/status",
		"/api/v0/autoprogramming/goal/observe",
		"/api/v0/resident-director/control",
		"/api/v0/runs/supervise",
		"/api/v0/external-work/run",
	}
	for _, path := range paths {
		t.Run(strings.ReplaceAll(strings.Trim(path, "/"), "/", "_"), func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`)))
			body := response.Body.String()
			if response.Code != http.StatusServiceUnavailable ||
				!strings.Contains(body, `"errores_publicos"`) ||
				strings.Contains(body, `"errores"`) {
				t.Fatalf("path=%s status=%d body=%s", path, response.Code, body)
			}
		})
	}
}

func TestValidateServerWorktreeIdentityV0ExigeBinarioRefsYRemoteExactosV0(t *testing.T) {
	repo, identity := canonicalServerIdentityRepoForTestV0(t)
	if issue := validateServerWorktreeIdentityWithRuntimeV0(t.Context(), repo, identity); issue != nil {
		t.Fatalf("identidad canonica rechazada: %+v", issue)
	}

	t.Run("remote local", func(t *testing.T) {
		runServerIdentityGitForTestV0(t, repo, "remote", "set-url", "origin", filepath.Join(t.TempDir(), "repo.bundle"))
		issue := validateServerWorktreeIdentityWithRuntimeV0(t.Context(), repo, identity)
		if issue == nil || issue.Code != serverWorktreeIdentityRemoteV0 {
			t.Fatalf("remote local issue=%+v", issue)
		}
		runServerIdentityGitForTestV0(t, repo, "remote", "set-url", "origin", serverCanonicalGitRemoteURLV0)
	})

	t.Run("build distinto", func(t *testing.T) {
		mismatched := identity
		mismatched.CommitRef = strings.Repeat("c", 40)
		mismatched.BuildRef = serverRuntimeBuildRefV0(mismatched.CommitRef, mismatched.BinarySHA256, false)
		issue := validateServerWorktreeIdentityWithRuntimeV0(t.Context(), repo, mismatched)
		if issue == nil || issue.Code != serverWorktreeIdentityBuildCommitV0 {
			t.Fatalf("build mismatch issue=%+v", issue)
		}
	})

	t.Run("sha distinto", func(t *testing.T) {
		mismatched := identity
		mismatched.BinarySHA256 = strings.Repeat("d", 64)
		mismatched.BuildRef = serverRuntimeBuildRefV0(mismatched.CommitRef, mismatched.BinarySHA256, false)
		issue := validateServerWorktreeIdentityWithRuntimeV0(t.Context(), repo, mismatched)
		if issue == nil || issue.Code != serverWorktreeIdentityBinaryV0 {
			t.Fatalf("sha mismatch issue=%+v", issue)
		}
	})

	t.Run("binary symlink no se publica como path canonico", func(t *testing.T) {
		link := filepath.Join(t.TempDir(), "orquesta-server-current")
		if err := os.Symlink(identity.BinaryPath, link); err != nil {
			t.Fatal(err)
		}
		mismatched := identity
		mismatched.BinaryPath = link
		issue := validateServerWorktreeIdentityWithRuntimeV0(t.Context(), repo, mismatched)
		if issue == nil || issue.Code != serverWorktreeIdentityBinaryV0 {
			t.Fatalf("binary symlink issue=%+v", issue)
		}
	})

	t.Run("symlink", func(t *testing.T) {
		link := filepath.Join(t.TempDir(), "repo-link")
		if err := os.Symlink(repo, link); err != nil {
			t.Fatal(err)
		}
		issue := validateServerWorktreeIdentityWithRuntimeV0(t.Context(), link, identity)
		if issue == nil || issue.Code != serverWorktreeIdentitySymlinkV0 {
			t.Fatalf("symlink issue=%+v", issue)
		}
	})

	t.Run("ref distinta", func(t *testing.T) {
		repo, identity := canonicalServerIdentityRepoForTestV0(t)
		runServerIdentityGitForTestV0(t, repo, "branch", "-m", "otra-rama")
		runServerIdentityGitForTestV0(t, repo, "update-ref", "refs/remotes/origin/otra-rama", "HEAD")
		runServerIdentityGitForTestV0(t, repo, "branch", "--set-upstream-to", "origin/otra-rama")
		issue := validateServerWorktreeIdentityWithRuntimeV0(t.Context(), repo, identity)
		if issue == nil || issue.Code != serverWorktreeIdentityRefV0 {
			t.Fatalf("ref no canonica issue=%+v", issue)
		}
	})
}

func TestValidateServerWorktreeIdentityV0RechazaMarcadoresYHEADNoExactoV0(t *testing.T) {
	for _, marker := range []string{".orquesta-retired", ".orquesta-stale"} {
		t.Run(marker, func(t *testing.T) {
			repo, identity := canonicalServerIdentityRepoForTestV0(t)
			if err := os.WriteFile(filepath.Join(repo, marker), []byte("marked\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if issue := validateServerWorktreeIdentityWithRuntimeV0(t.Context(), repo, identity); issue == nil {
				t.Fatal("identity valida con marcador")
			}
		})
	}

	repo, identity := canonicalServerIdentityRepoForTestV0(t)
	runServerIdentityGitForTestV0(t, repo, "-c", "user.name=Identity Test", "-c", "user.email=identity@example.invalid", "commit", "--allow-empty", "-m", "ahead")
	issue := validateServerWorktreeIdentityWithRuntimeV0(t.Context(), repo, identity)
	if issue == nil || issue.Code != serverWorktreeIdentityNotAlignedV0 {
		t.Fatalf("HEAD ahead debe rechazarse, issue=%+v", issue)
	}
}

func TestServerIdentityPreflightV0NoCreaEstadoNiRuntimeSiWorktreeInvalidoV0(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "not-git")
	stateDir := filepath.Join(root, "must-not-exist-state")
	runtimeDir := filepath.Join(root, "must-not-exist-runtime")
	if err := os.Mkdir(project, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv(envCodexProjectWorkDirV0, project)
	t.Setenv(envServerStateDirV0, stateDir)
	t.Setenv(envCodexRuntimeWorkDirV0, runtimeDir)

	preflight, err := serverIdentityPreflightFromEnvV0("")
	if err != nil || preflight.Issue == nil {
		t.Fatalf("preflight err=%v issue=%+v", err, preflight.Issue)
	}
	for _, path := range []string{stateDir, runtimeDir} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("preflight creo %s: %v", path, err)
		}
	}
}

func TestServerIdentityPreflightV0AceptaProyectoExternoConWorktreeCanonicoSeparadoV0(t *testing.T) {
	worktree, identity := canonicalServerIdentityRepoForTestV0(t)
	project := t.TempDir()

	preflight := serverIdentityPreflightForWorkdirsV0(t.Context(), project, worktree, identity)
	if preflight.Issue != nil {
		t.Fatalf("preflight rechazada: %+v", preflight.Issue)
	}
	if preflight.ProjectWorkDir != project || preflight.WorktreeDir != worktree {
		t.Fatalf("project=%q worktree=%q", preflight.ProjectWorkDir, preflight.WorktreeDir)
	}
}

func TestServerWorktreeDirFromEnvV0PrefiereConfiguracionExplicitaV0(t *testing.T) {
	t.Setenv(envServerWorktreeV0, "/tmp/orquesta-source")
	if got := serverWorktreeDirFromEnvV0("/tmp/external-app"); got != "/tmp/orquesta-source" {
		t.Fatalf("worktree=%q", got)
	}
}

func TestServerYControlScriptCompartenRemoteCanonicoYPoliticaSymlinkV0(t *testing.T) {
	scriptPath := filepath.Join("..", "..", "scripts", "orquesta_server_ctl.sh")
	raw, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read ctl: %v", err)
	}
	text := string(raw)
	if !strings.Contains(text, `CANONICAL_GIT_REMOTE_URL="`+serverCanonicalGitRemoteURLV0+`"`) ||
		!strings.Contains(text, `CANONICAL_GIT_REF="`+serverCanonicalGitRefV0+`"`) ||
		!strings.Contains(text, `[ "$WORKDIR" != "$workdir_physical" ]`) ||
		!strings.Contains(text, `BINARY_REAL="$(readlink -f -- "$BIN"`) {
		t.Fatalf("ctl diverge de remote/symlink canonicos del server")
	}
}

func TestRunMainV0BloqueaLaunchAmplioAntesDeCrearRuntimeV0(t *testing.T) {
	project := filepath.Join(t.TempDir(), "missing-project")
	runtimeDir := filepath.Join(t.TempDir(), "must-not-exist-runtime")
	t.Setenv(envCodexProjectWorkDirV0, project)
	t.Setenv(envCodexRuntimeWorkDirV0, runtimeDir)
	var stdout strings.Builder
	var stderr strings.Builder
	code := runMain([]string{
		"codex-launch-wave",
		"--project-dir", project,
		"--runtime-dir", runtimeDir,
		"--prompt", "must not launch",
	}, &stdout, &stderr)
	if code != 1 || !strings.Contains(stderr.String(), "reason_code="+serverWorkLaunchDegradedIdentityV0) {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(runtimeDir); !os.IsNotExist(err) {
		t.Fatalf("launch amplio creo runtime antes del guard: %v", err)
	}
}

func TestServerCommandRequiresExactIdentityV0CubreEntradasDeLaunchAmplioV0(t *testing.T) {
	for _, command := range []string{
		"codex-launch-wave",
		"codex-launch-director-wave",
	} {
		if !serverCommandRequiresExactIdentityV0(command) {
			t.Fatalf("comando amplio sin guard: %s", command)
		}
	}
	for _, command := range []string{"status", "run-status", "stop", "codex-wave-status", "codex-wave-tail"} {
		if serverCommandRequiresExactIdentityV0(command) {
			t.Fatalf("comando observacional bloqueado como launch: %s", command)
		}
	}
}

func canonicalServerIdentityRepoForTestV0(t *testing.T) (string, orquestaserver.ServerRuntimeIdentityV0) {
	t.Helper()
	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.Mkdir(repo, 0o700); err != nil {
		t.Fatal(err)
	}
	runServerIdentityGitForTestV0(t, repo, "init", "-q")
	runServerIdentityGitForTestV0(t, repo, "-c", "user.name=Identity Test", "-c", "user.email=identity@example.invalid", "commit", "--allow-empty", "-q", "-m", "initial")
	runServerIdentityGitForTestV0(t, repo, "branch", "-M", serverCanonicalGitRefV0)
	branch := serverWorktreeGitOutputV0(context.Background(), repo, "branch", "--show-current")
	runServerIdentityGitForTestV0(t, repo, "remote", "add", "origin", serverCanonicalGitRemoteURLV0)
	runServerIdentityGitForTestV0(t, repo, "update-ref", "refs/remotes/origin/"+branch, "HEAD")
	runServerIdentityGitForTestV0(t, repo, "branch", "--set-upstream-to", "origin/"+branch)

	binaryPath := filepath.Join(t.TempDir(), "orquesta-server")
	if err := os.WriteFile(binaryPath, []byte("server binary fixture\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	head := serverWorktreeGitOutputV0(context.Background(), repo, "rev-parse", "HEAD")
	binarySHA := serverRuntimeBinarySHA256V0(binaryPath)
	return repo, orquestaserver.ServerRuntimeIdentityV0{
		BinaryPath:   binaryPath,
		BinarySHA256: binarySHA,
		CommitRef:    head,
		BuildRef:     serverRuntimeBuildRefV0(head, binarySHA, false),
	}
}

func runServerIdentityGitForTestV0(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}
