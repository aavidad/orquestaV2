package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"orquesta/db"
	"orquesta/gitestadisticasapp"
)

type fakeGitStatsCollector struct {
	gotPath  string
	gotSince time.Time
	stats    *gitestadisticasapp.Stats
	err      error
}

func (f *fakeGitStatsCollector) Collect(cwd string, since time.Time) (*gitestadisticasapp.Stats, error) {
	f.gotPath = cwd
	f.gotSince = since
	return f.stats, f.err
}

func TestGitOperationalStatsServiceExponeMetricasCanonicas(t *testing.T) {
	svc := gitOperationalStatsService{collector: &fakeGitStatsCollector{stats: &gitestadisticasapp.Stats{
		RepoRoot:              "/repo",
		CWD:                   "/repo",
		Branch:                "main",
		TouchedFiles:          []string{"cmd/api.go", "cmd/api_git_stats.go"},
		PendingAddedLines:     7,
		PendingDeletedLines:   2,
		CommittedAddedLines:   11,
		CommittedDeletedLines: 5,
		RecentCommitCount:     2,
	}}}

	stats, err := svc.Collect("/repo", time.Now().UTC().Add(-time.Hour))
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if stats.FilesChanged != 2 {
		t.Fatalf("files_changed inesperado: %d", stats.FilesChanged)
	}
	if stats.Insertions != 18 || stats.Deletions != 7 || stats.LinesNet != 11 {
		t.Fatalf("totales canonicos inesperados: %+v", stats)
	}
}

func TestBuildGitChangeTotalsNormalizaFicherosYTotales(t *testing.T) {
	got := buildGitChangeTotals(
		[]string{" cmd/api.go ", "", "cmd/api.go", "db/repo.go"},
		3,
		1,
		4,
		2,
	)
	if got.FilesChanged != 2 {
		t.Fatalf("files_changed inesperado: %+v", got)
	}
	if got.Insertions != 7 || got.Deletions != 3 || got.LinesNet != 4 {
		t.Fatalf("totales canonicos inesperados: %+v", got)
	}
}

func TestNewGitOperationalStatsNormalizaListasDeFicheros(t *testing.T) {
	got := newGitOperationalStats(&gitestadisticasapp.Stats{
		PendingFiles:          []string{" cmd/z.go ", "", "cmd/a.go", "cmd/a.go"},
		RecentCommitFiles:     []string{"db/repo.go", " db/repo.go "},
		TouchedFiles:          []string{" cmd/z.go ", "db/repo.go", "", "cmd/z.go"},
		PendingAddedLines:     2,
		PendingDeletedLines:   1,
		CommittedAddedLines:   3,
		CommittedDeletedLines: 1,
	})
	if got == nil {
		t.Fatalf("stats nil")
	}
	if !reflect.DeepEqual(got.PendingFiles, []string{"cmd/a.go", "cmd/z.go"}) {
		t.Fatalf("pending files inesperados: %+v", got.PendingFiles)
	}
	if !reflect.DeepEqual(got.RecentCommitFiles, []string{"db/repo.go"}) {
		t.Fatalf("recent_commit_files inesperados: %+v", got.RecentCommitFiles)
	}
	if !reflect.DeepEqual(got.TouchedFiles, []string{"cmd/z.go", "db/repo.go"}) {
		t.Fatalf("touched_files inesperados: %+v", got.TouchedFiles)
	}
	if got.FilesChanged != 2 || got.Insertions != 5 || got.Deletions != 2 || got.LinesNet != 3 {
		t.Fatalf("totales inesperados: %+v", got)
	}
}

func TestAPIGitStatsPathDevuelveContratoCanonico(t *testing.T) {
	fake := &fakeGitStatsCollector{stats: &gitestadisticasapp.Stats{
		RepoRoot:              "/tmp/repo",
		CWD:                   "/tmp/repo",
		Branch:                "main",
		TouchedFiles:          []string{"cmd/app.go"},
		PendingAddedLines:     3,
		PendingDeletedLines:   1,
		CommittedAddedLines:   4,
		CommittedDeletedLines: 2,
	}}
	prevService := apiGitStatsService
	t.Cleanup(func() {
		apiGitStatsService = prevService
	})
	apiGitStatsService = gitOperationalStatsService{collector: fake}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/git/stats?path=/tmp/repo&desde=2026-04-28T10:00:00Z", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp apiGitStatsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !resp.OK || resp.Scope != "path" || resp.Path != "/tmp/repo" {
		t.Fatalf("respuesta base inesperada: %+v", resp)
	}
	if fake.gotPath != "/tmp/repo" {
		t.Fatalf("path pasado al servicio inesperado: %q", fake.gotPath)
	}
	if !fake.gotSince.Equal(time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC)) {
		t.Fatalf("since pasado al servicio inesperado: %s", fake.gotSince)
	}
	if resp.Stats == nil || resp.Stats.FilesChanged != 1 || resp.Stats.Insertions != 7 || resp.Stats.Deletions != 3 || resp.Stats.LinesNet != 4 {
		t.Fatalf("stats canonicas inesperadas: %+v", resp.Stats)
	}
}

func TestAPIGitStatsProyectoResuelveRuta(t *testing.T) {
	fake := &fakeGitStatsCollector{stats: &gitestadisticasapp.Stats{RepoRoot: "/repo/demo", CWD: "/repo/demo"}}
	prevService := apiGitStatsService
	prevProjectLookup := apiGitStatsProjectLookupFn
	t.Cleanup(func() {
		apiGitStatsService = prevService
		apiGitStatsProjectLookupFn = prevProjectLookup
	})
	apiGitStatsService = gitOperationalStatsService{collector: fake}
	apiGitStatsProjectLookupFn = func(ref string) (*db.Proyecto, error) {
		if ref != "demo" {
			t.Fatalf("ref proyecto inesperada: %q", ref)
		}
		return &db.Proyecto{Slug: "demo", RutaAbs: "/repo/demo"}, nil
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/git/stats?proyecto=demo", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp apiGitStatsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Scope != "proyecto" || resp.Project != "demo" || fake.gotPath != "/repo/demo" {
		t.Fatalf("resolucion proyecto inesperada: resp=%+v path=%q", resp, fake.gotPath)
	}
}

func TestAPIGitStatsRechazaScopeAmbiguo(t *testing.T) {
	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/git/stats?proyecto=demo&path=/repo", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
}
