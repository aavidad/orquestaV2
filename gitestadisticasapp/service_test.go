package gitestadisticasapp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestCollectResumeRepoYVentanaTemporal(t *testing.T) {
	t.Parallel()

	repo := initGitStatsRepo(t)
	writeGitStatsFile(t, filepath.Join(repo, "cmd", "app.go"), "package main\n\nfunc main() {}\n")
	gitStatsCommit(t, repo, "cmd/app.go")

	writeGitStatsFile(t, filepath.Join(repo, "cmd", "app.go"), "package main\n\nfunc main() {\n\tprintln(\"hola\")\n}\n")
	writeGitStatsFile(t, filepath.Join(repo, "db", "repo.go"), "package db\n")

	svc := NewService()
	stats, err := svc.Collect(repo, time.Now().UTC().Add(-time.Hour))
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if stats == nil {
		t.Fatalf("Collect devolvió nil")
	}
	if stats.RepoRoot != repo {
		t.Fatalf("repo root inesperado: %q", stats.RepoRoot)
	}
	if stats.CWD != repo {
		t.Fatalf("cwd inesperado: %q", stats.CWD)
	}
	if stats.Branch != "main" {
		t.Fatalf("branch inesperada: %q", stats.Branch)
	}
	if stats.PendingShortStat == "" {
		t.Fatalf("pending shortstat vacío")
	}
	if stats.RecentCommitCount != 1 {
		t.Fatalf("commit count inesperado: %d", stats.RecentCommitCount)
	}
	if stats.PendingAddedLines <= 0 {
		t.Fatalf("pending added lines inesperado: %d", stats.PendingAddedLines)
	}
	if stats.CommittedAddedLines <= 0 {
		t.Fatalf("committed added lines inesperado: %d", stats.CommittedAddedLines)
	}
	assertContainsGitStats(t, stats.PendingFiles, "cmd/app.go")
	assertContainsGitStats(t, stats.PendingFiles, "db/repo.go")
	assertContainsGitStats(t, stats.RecentCommitFiles, "cmd/app.go")
	assertContainsGitStats(t, stats.TouchedFiles, "cmd/app.go")
	assertContainsGitStats(t, stats.TouchedFiles, "db/repo.go")
}

func TestCollectFueraDeRepoDevuelveSoloCWD(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	svc := &Service{runner: fakeRunner{err: fmt.Errorf("not a git repo")}}
	stats, err := svc.Collect(dir, time.Now().UTC().Add(-time.Hour))
	if err != nil {
		t.Fatalf("Collect fuera repo: %v", err)
	}
	if stats == nil {
		t.Fatalf("Collect devolvió nil fuera repo")
	}
	if stats.CWD != dir {
		t.Fatalf("cwd inesperado: %q", stats.CWD)
	}
	if stats.RepoRoot != "" {
		t.Fatalf("repo root debería ir vacío: %q", stats.RepoRoot)
	}
}

type fakeRunner struct {
	out string
	err error
}

func (f fakeRunner) Run(dir string, args ...string) (string, error) {
	return f.out, f.err
}

func TestAggregateStatsSumaYMantieneFicherosUnicos(t *testing.T) {
	t.Parallel()

	agg := AggregateStats(
		&Stats{
			RepoRoot:              "/tmp/a",
			TouchedFiles:          []string{"cmd/a.go", "db/x.go"},
			PendingAddedLines:     3,
			PendingDeletedLines:   1,
			CommittedAddedLines:   5,
			CommittedDeletedLines: 2,
			RecentCommitCount:     1,
		},
		&Stats{
			RepoRoot:              "/tmp/b",
			TouchedFiles:          []string{"cmd/a.go", "cmd/b.go"},
			PendingAddedLines:     4,
			PendingDeletedLines:   2,
			CommittedAddedLines:   7,
			CommittedDeletedLines: 3,
			RecentCommitCount:     2,
		},
	)

	if agg.PendingAddedLines != 7 || agg.PendingDeletedLines != 3 {
		t.Fatalf("pending agregado inesperado: %+v", agg)
	}
	if agg.CommittedAddedLines != 12 || agg.CommittedDeletedLines != 5 {
		t.Fatalf("commits agregados inesperados: %+v", agg)
	}
	if agg.RecentCommitCount != 3 {
		t.Fatalf("recent commit count inesperado: %d", agg.RecentCommitCount)
	}
	if len(agg.RepoRoots) != 2 || agg.RepoRoots[0] != "/tmp/a" || agg.RepoRoots[1] != "/tmp/b" {
		t.Fatalf("repo roots inesperados: %+v", agg.RepoRoots)
	}
	if len(agg.TouchedFiles) != 3 {
		t.Fatalf("touched files inesperados: %+v", agg.TouchedFiles)
	}
}

func TestParseGitStatusFilesMantieneRutaCompleta(t *testing.T) {
	files := parseGitStatusFiles(" M cmd/api_test.go\n?? .orquesta-inbox.md\n")
	if len(files) != 2 || files[0] != ".orquesta-inbox.md" || files[1] != "cmd/api_test.go" {
		t.Fatalf("status files inesperados: %+v", files)
	}
}

func TestParseGitNumstatTotalsSumaLineasYFicheros(t *testing.T) {
	added, deleted, files := parseGitNumstatTotals("12\t3\tcmd/app.go\n5\t1\tdb/repo.go\n")
	if added != 17 || deleted != 4 {
		t.Fatalf("totales inesperados: +%d/-%d", added, deleted)
	}
	if len(files) != 2 || files[0] != "cmd/app.go" || files[1] != "db/repo.go" {
		t.Fatalf("files inesperados: %+v", files)
	}
}

func initGitStatsRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runGitStats(t, repo, "init", "-b", "main")
	runGitStats(t, repo, "config", "user.name", "Orquesta Test")
	runGitStats(t, repo, "config", "user.email", "orquesta@example.com")
	return repo
}

func gitStatsCommit(t *testing.T, repo string, paths ...string) {
	t.Helper()
	args := append([]string{"add"}, paths...)
	runGitStats(t, repo, args...)
	runGitStats(t, repo, "commit", "-m", "stats test")
}

func writeGitStatsFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func runGitStats(t *testing.T, repo string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, string(out))
	}
	return string(out)
}

func assertContainsGitStats(t *testing.T, items []string, want string) {
	t.Helper()
	for _, item := range items {
		if item == want {
			return
		}
	}
	t.Fatalf("falta %q en %+v", want, items)
}
