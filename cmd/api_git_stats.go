package cmd

import (
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"orquesta/db"
	"orquesta/gitestadisticasapp"
)

type gitOperationalStatsCollector interface {
	Collect(cwd string, since time.Time) (*gitestadisticasapp.Stats, error)
}

type gitOperationalStatsService struct {
	collector gitOperationalStatsCollector
}

type gitOperationalStats struct {
	RepoRoot              string   `json:"repo_root,omitempty"`
	CWD                   string   `json:"cwd,omitempty"`
	Branch                string   `json:"branch,omitempty"`
	PendingFiles          []string `json:"pending_files,omitempty"`
	RecentCommitFiles     []string `json:"recent_commit_files,omitempty"`
	TouchedFiles          []string `json:"touched_files,omitempty"`
	FilesChanged          int      `json:"files_changed"`
	PendingAddedLines     int      `json:"pending_added_lines"`
	PendingDeletedLines   int      `json:"pending_deleted_lines"`
	CommittedAddedLines   int      `json:"committed_added_lines"`
	CommittedDeletedLines int      `json:"committed_deleted_lines"`
	Insertions            int      `json:"insertions"`
	Deletions             int      `json:"deletions"`
	LinesNet              int      `json:"lines_net"`
	PendingShortStat      string   `json:"pending_shortstat,omitempty"`
	RecentCommitCount     int      `json:"recent_commit_count"`
}

type apiGitStatsResponse struct {
	OK        bool                 `json:"ok"`
	Scope     string               `json:"scope"`
	Project   string               `json:"project,omitempty"`
	Path      string               `json:"path,omitempty"`
	Since     time.Time            `json:"since"`
	Generated time.Time            `json:"generated"`
	Stats     *gitOperationalStats `json:"stats"`
}

type gitChangeTotals struct {
	FilesChanged int
	Insertions   int
	Deletions    int
	LinesNet     int
}

var (
	apiGitStatsService         = gitOperationalStatsService{collector: gitestadisticasapp.NewService()}
	apiGitStatsProjectLookupFn = func(ref string) (*db.Proyecto, error) {
		return apiGetProyectoConRutaEfectivaTimeboxed(ref, "")
	}
)

func apiHandlerGitStats(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	since, err := parseStatsSince(strings.TrimSpace(r.URL.Query().Get("desde")))
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	scope, project, path, err := resolveGitStatsScope(r)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	stats, err := apiGitStatsService.Collect(path, since)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiGitStatsResponse{
		OK:        true,
		Scope:     scope,
		Project:   project,
		Path:      path,
		Since:     since.UTC(),
		Generated: time.Now().UTC(),
		Stats:     stats,
	})
}

func resolveGitStatsScope(r *http.Request) (string, string, string, error) {
	query := r.URL.Query()
	projectRef := strings.TrimSpace(query.Get("proyecto"))
	path := normalizeGitStatsPath(query.Get("path"))
	if path == "" {
		path = normalizeGitStatsPath(query.Get("cwd"))
	}
	if projectRef != "" && path != "" {
		return "", "", "", fmt.Errorf("usa proyecto o path/cwd, pero no ambos")
	}
	if projectRef != "" {
		project, err := apiGitStatsProjectLookupFn(projectRef)
		if err != nil {
			return "", "", "", err
		}
		if project == nil || strings.TrimSpace(project.RutaAbs) == "" {
			return "", "", "", fmt.Errorf("proyecto sin ruta git: %s", projectRef)
		}
		return "proyecto", strings.TrimSpace(project.Slug), normalizeGitStatsPath(project.RutaAbs), nil
	}
	if path != "" {
		return "path", "", path, nil
	}
	return "", "", "", fmt.Errorf("path/cwd o proyecto es obligatorio")
}

func normalizeGitStatsPath(raw string) string {
	path := strings.TrimSpace(raw)
	if path == "" {
		return ""
	}
	return filepath.Clean(path)
}

func (s gitOperationalStatsService) Collect(path string, since time.Time) (*gitOperationalStats, error) {
	if s.collector == nil {
		return nil, fmt.Errorf("servicio git stats no configurado")
	}
	raw, err := s.collector.Collect(strings.TrimSpace(path), since)
	if err != nil {
		return nil, err
	}
	return newGitOperationalStats(raw), nil
}

func newGitOperationalStats(raw *gitestadisticasapp.Stats) *gitOperationalStats {
	if raw == nil {
		return nil
	}
	pendingFiles := normalizeGitFileList(raw.PendingFiles)
	recentCommitFiles := normalizeGitFileList(raw.RecentCommitFiles)
	touchedFiles := normalizeGitFileList(raw.TouchedFiles)
	totals := buildGitChangeTotals(
		touchedFiles,
		raw.PendingAddedLines,
		raw.PendingDeletedLines,
		raw.CommittedAddedLines,
		raw.CommittedDeletedLines,
	)
	return &gitOperationalStats{
		RepoRoot:              strings.TrimSpace(raw.RepoRoot),
		CWD:                   strings.TrimSpace(raw.CWD),
		Branch:                strings.TrimSpace(raw.Branch),
		PendingFiles:          pendingFiles,
		RecentCommitFiles:     recentCommitFiles,
		TouchedFiles:          touchedFiles,
		FilesChanged:          totals.FilesChanged,
		PendingAddedLines:     raw.PendingAddedLines,
		PendingDeletedLines:   raw.PendingDeletedLines,
		CommittedAddedLines:   raw.CommittedAddedLines,
		CommittedDeletedLines: raw.CommittedDeletedLines,
		Insertions:            totals.Insertions,
		Deletions:             totals.Deletions,
		LinesNet:              totals.LinesNet,
		PendingShortStat:      strings.TrimSpace(raw.PendingShortStat),
		RecentCommitCount:     raw.RecentCommitCount,
	}
}

func buildGitChangeTotals(touchedFiles []string, pendingAddedLines, pendingDeletedLines, committedAddedLines, committedDeletedLines int) gitChangeTotals {
	files := normalizeGitFileList(touchedFiles)
	insertions := pendingAddedLines + committedAddedLines
	deletions := pendingDeletedLines + committedDeletedLines
	return gitChangeTotals{
		FilesChanged: len(files),
		Insertions:   insertions,
		Deletions:    deletions,
		LinesNet:     insertions - deletions,
	}
}

func normalizeGitFileList(files []string) []string {
	fileSet := map[string]struct{}{}
	for _, file := range files {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}
		fileSet[file] = struct{}{}
	}
	if len(fileSet) == 0 {
		return nil
	}
	out := make([]string, 0, len(fileSet))
	for file := range fileSet {
		out = append(out, file)
	}
	sort.Strings(out)
	return out
}
