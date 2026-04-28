package cmd

import (
	"fmt"
	"net/http"
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
	path := strings.TrimSpace(query.Get("path"))
	if path == "" {
		path = strings.TrimSpace(query.Get("cwd"))
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
		return "proyecto", strings.TrimSpace(project.Slug), strings.TrimSpace(project.RutaAbs), nil
	}
	if path != "" {
		return "path", "", path, nil
	}
	return "", "", "", fmt.Errorf("path/cwd o proyecto es obligatorio")
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
	insertions := raw.PendingAddedLines + raw.CommittedAddedLines
	deletions := raw.PendingDeletedLines + raw.CommittedDeletedLines
	return &gitOperationalStats{
		RepoRoot:              strings.TrimSpace(raw.RepoRoot),
		CWD:                   strings.TrimSpace(raw.CWD),
		Branch:                strings.TrimSpace(raw.Branch),
		PendingFiles:          append([]string(nil), raw.PendingFiles...),
		RecentCommitFiles:     append([]string(nil), raw.RecentCommitFiles...),
		TouchedFiles:          append([]string(nil), raw.TouchedFiles...),
		FilesChanged:          len(raw.TouchedFiles),
		PendingAddedLines:     raw.PendingAddedLines,
		PendingDeletedLines:   raw.PendingDeletedLines,
		CommittedAddedLines:   raw.CommittedAddedLines,
		CommittedDeletedLines: raw.CommittedDeletedLines,
		Insertions:            insertions,
		Deletions:             deletions,
		LinesNet:              insertions - deletions,
		PendingShortStat:      strings.TrimSpace(raw.PendingShortStat),
		RecentCommitCount:     raw.RecentCommitCount,
	}
}
