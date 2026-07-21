package bootstrap

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func v16LoadFixture(t *testing.T) v16E2EFixture {
	t.Helper()
	directory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		candidate := filepath.Join(directory, "acceptance", "fixtures", "v16_workspace_git.json")
		content, readErr := os.ReadFile(candidate)
		if readErr == nil {
			var fixture v16E2EFixture
			if err := json.Unmarshal(content, &fixture); err != nil {
				t.Fatalf("decode V16 fixture: %v", err)
			}
			if fixture.GitFixture.TargetRef == "" || len(fixture.GitFixture.BaseFiles) == 0 ||
				len(fixture.CrashFrontiers) != 7 || len(fixture.Actors) < 2 {
				t.Fatalf("incomplete V16 fixture: %+v", fixture)
			}
			return fixture
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			t.Fatalf("locate V16 fixture: %v", readErr)
		}
		directory = parent
	}
}

func v16FixtureWrites(values []v16FixtureFile) v16Write {
	result := make(v16Write, len(values))
	for _, value := range values {
		result[value.Path] = value.Content
	}
	return result
}

func v16WriteConfig(t *testing.T, root, seed, workspaceRoot, targetRef string) string {
	t.Helper()
	path := filepath.Join(root, "orquesta.toml")
	content := fmt.Sprintf(`[server]
listen = "127.0.0.1:0"
shutdown_timeout = "2s"

[state.sqlite]
path = %s
busy_timeout = "1s"
max_open_connections = 4

[artifact.filesystem]
root = %s

[credentials.local]
path = %s
max_document_bytes = 1048576

[runtime]
max_output_bytes = 65536

[runtime.codex]
timeout = "1s"
max_concurrent_executions = 4
work_root = %s
env_allowlist = ["PATH"]

[workspace.local]
root = %s

[repository.local]
seed_path = %s
target_ref = %s

[identity]
local_actor = "actor:v16-alice"
local_token_path = %s

[project]
default = "project:v16-alpha"

[scheduler]
poll_interval = "10ms"
observation_interval = "10ms"
claim_lease = "10s"
max_execution_attempts = 3
execution_timeout = "10s"

[config]
effective_path = %s
`, strconv.Quote(filepath.Join(root, "state", "orquesta.sqlite")),
		strconv.Quote(filepath.Join(root, "artifacts")),
		strconv.Quote(filepath.Join(root, "secrets", "credentials.json")),
		strconv.Quote(filepath.Join(root, "agent-work")), strconv.Quote(workspaceRoot),
		strconv.Quote(seed), strconv.Quote(targetRef),
		strconv.Quote(filepath.Join(root, "secrets", "local-owner.token")),
		strconv.Quote(filepath.Join(root, "effective-config.json")))
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func v16Git(t *testing.T, git, directory string, arguments ...string) string {
	t.Helper()
	return v16GitInput(t, git, directory, nil, nil, arguments...)
}

func v16GitInput(
	t *testing.T,
	git, directory string,
	input []byte,
	environment []string,
	arguments ...string,
) string {
	t.Helper()
	command := exec.Command(git, append([]string{
		"-C", directory, "-c", "core.hooksPath=/dev/null", "-c", "commit.gpgSign=false",
	}, arguments...)...)
	command.Env = append([]string{
		"PATH=" + filepath.Dir(git) + string(os.PathListSeparator) + "/usr/bin:/bin",
		"HOME=" + directory, "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0",
	}, environment...)
	if input != nil {
		command.Stdin = strings.NewReader(string(input))
	}
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", arguments, err, output)
	}
	return strings.TrimSpace(string(output))
}

func v16GitCommit(t *testing.T, git, directory, name, email string, at time.Time, message string) {
	t.Helper()
	stamp := at.UTC().Format(time.RFC3339)
	v16GitInput(t, git, directory, nil, []string{
		"GIT_AUTHOR_NAME=" + name, "GIT_AUTHOR_EMAIL=" + email, "GIT_AUTHOR_DATE=" + stamp,
		"GIT_COMMITTER_NAME=" + name, "GIT_COMMITTER_EMAIL=" + email, "GIT_COMMITTER_DATE=" + stamp,
	}, "commit", "-m", message)
}

func v16RequireGitVersion(t *testing.T, output, minimum string) {
	t.Helper()
	actual := strings.TrimPrefix(strings.TrimSpace(output), "git version ")
	parse := func(value string) [3]int {
		var result [3]int
		parts := strings.Split(value, ".")
		for index := 0; index < len(result) && index < len(parts); index++ {
			number := strings.TrimRightFunc(parts[index], func(r rune) bool { return r < '0' || r > '9' })
			result[index], _ = strconv.Atoi(number)
		}
		return result
	}
	got, want := parse(actual), parse(minimum)
	for index := range got {
		if got[index] > want[index] {
			return
		}
		if got[index] < want[index] {
			t.Fatalf("Git version %s below fixture minimum %s", actual, minimum)
		}
	}
}

func v16WriteFile(t *testing.T, root, relative, content string) {
	t.Helper()
	if filepath.IsAbs(relative) || strings.Contains(relative, "..") {
		t.Fatalf("unsafe fixture path %q", relative)
	}
	target := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func v16ApplyFixtureTarget(t *testing.T, git, seed string, fixture v16E2EFixture) {
	t.Helper()
	for _, file := range fixture.GitFixture.TargetChanges {
		v16WriteFile(t, seed, file.Path, file.Content)
	}
	v16Git(t, git, seed, "add", "--all")
	v16GitCommit(t, git, seed, fixture.GitFixture.AuthorName, fixture.GitFixture.AuthorEmail,
		time.Unix(fixture.GitFixture.TargetUnixTime, 0).UTC(), "v16 fixture target")
	commit := v16Git(t, git, seed, "rev-parse", "HEAD")
	tree := v16Git(t, git, seed, "rev-parse", "HEAD^{tree}")
	if commit != fixture.GitFixture.TargetOID || tree != fixture.GitFixture.TargetTreeOID {
		t.Fatalf("sealed target fixture mismatch: commit=%s/%s tree=%s/%s",
			commit, fixture.GitFixture.TargetOID, tree, fixture.GitFixture.TargetTreeOID)
	}
}
