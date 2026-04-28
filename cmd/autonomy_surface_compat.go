package cmd

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

type autonomyProjectSurface struct {
	Project    string                 `json:"project"`
	Events     int                    `json:"events"`
	ByKind     map[string]int         `json:"by_kind,omitempty"`
	LastAt     *time.Time             `json:"last_at,omitempty"`
	Recent     []autonomyEventSummary `json:"recent,omitempty"`
	Highlights []string               `json:"highlights,omitempty"`
}

type autonomySurfaceRecentItem struct {
	Project string `json:"project"`
	autonomyEventSummary
}

type autonomySurface struct {
	Events     int                         `json:"events"`
	ByKind     map[string]int              `json:"by_kind,omitempty"`
	LastAt     *time.Time                  `json:"last_at,omitempty"`
	Recent     []autonomySurfaceRecentItem `json:"recent,omitempty"`
	Highlights []string                    `json:"highlights,omitempty"`
	Projects   []autonomyProjectSurface    `json:"projects,omitempty"`
}

func buildVisibleProjectAutonomySurfaceLocal() (*autonomySurface, error) {
	active := true
	proyectos, err := listarProyectosFiltrados("", &active)
	if err != nil {
		return nil, err
	}
	cockpits := make([]*apiProyectoCockpit, 0, len(proyectos))
	for _, proyecto := range proyectos {
		slug := strings.TrimSpace(fmt.Sprint(proyecto["slug"]))
		if slug == "" {
			continue
		}
		cockpit, err := buildProyectoCockpit(slug)
		if err != nil || cockpit == nil {
			continue
		}
		cockpits = append(cockpits, cockpit)
	}
	return buildAutonomySurfaceFromCockpits(cockpits, 8), nil
}

func fetchServerProjectAutonomySurface(baseURL string) (*autonomySurface, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, nil
	}
	var proyectosResp struct {
		Proyectos []*struct {
			Slug   string `json:"slug"`
			Activo bool   `json:"activo"`
		} `json:"proyectos"`
	}
	if err := fetchServerJSON(baseURL+"/api/proyectos?activa=1", &proyectosResp); err != nil {
		return nil, nil
	}
	cockpits := make([]*apiProyectoCockpit, 0, len(proyectosResp.Proyectos))
	for _, proyecto := range proyectosResp.Proyectos {
		if proyecto == nil || strings.TrimSpace(proyecto.Slug) == "" || !proyecto.Activo {
			continue
		}
		var resp apiProyectoCockpitResponse
		path := fmt.Sprintf("%s/api/proyectos/%s/cockpit", baseURL, url.PathEscape(strings.TrimSpace(proyecto.Slug)))
		if err := fetchServerJSON(path, &resp); err != nil || resp.Cockpit == nil {
			continue
		}
		cockpits = append(cockpits, resp.Cockpit)
	}
	return buildAutonomySurfaceFromCockpits(cockpits, 8), nil
}

func buildAutonomySurfaceFromCockpits(cockpits []*apiProyectoCockpit, recentLimit int) *autonomySurface {
	if len(cockpits) == 0 {
		return nil
	}
	out := &autonomySurface{
		ByKind:   map[string]int{},
		Projects: make([]autonomyProjectSurface, 0, len(cockpits)),
	}
	recent := make([]autonomySurfaceRecentItem, 0)
	for _, cockpit := range cockpits {
		if cockpit == nil || cockpit.AutonomyEvents <= 0 {
			continue
		}
		project := ""
		if cockpit.Proyecto != nil {
			project = strings.TrimSpace(cockpit.Proyecto.Slug)
		}
		projectByKind := map[string]int{}
		for kind, count := range cockpit.AutonomyByKind {
			key := strings.TrimSpace(kind)
			if key == "" || count <= 0 {
				continue
			}
			projectByKind[key] += count
			out.ByKind[key] += count
		}
		projectRecent := append([]autonomyEventSummary(nil), cockpit.Autonomy...)
		out.Events += cockpit.AutonomyEvents
		out.LastAt = maxTimePtr(out.LastAt, cockpit.AutonomyLastAt)
		out.Projects = append(out.Projects, autonomyProjectSurface{
			Project:    project,
			Events:     cockpit.AutonomyEvents,
			ByKind:     projectByKind,
			LastAt:     cockpit.AutonomyLastAt,
			Recent:     projectRecent,
			Highlights: buildAutonomyHighlights(projectByKind, projectRecent, cockpit.AutonomyLastAt, 3),
		})
		for _, item := range projectRecent {
			recent = append(recent, autonomySurfaceRecentItem{
				Project:              project,
				autonomyEventSummary: item,
			})
		}
	}
	if out.Events <= 0 {
		return nil
	}
	sort.SliceStable(out.Projects, func(i, j int) bool {
		left := out.Projects[i]
		right := out.Projects[j]
		switch {
		case left.LastAt == nil && right.LastAt == nil:
			return left.Project < right.Project
		case left.LastAt == nil:
			return false
		case right.LastAt == nil:
			return true
		case left.LastAt.Equal(*right.LastAt):
			return left.Project < right.Project
		default:
			return left.LastAt.After(*right.LastAt)
		}
	})
	sort.SliceStable(recent, func(i, j int) bool {
		if recent[i].CreatedAt.Equal(recent[j].CreatedAt) {
			if recent[i].Project == recent[j].Project {
				return recent[i].Kind < recent[j].Kind
			}
			return recent[i].Project < recent[j].Project
		}
		return recent[i].CreatedAt.After(recent[j].CreatedAt)
	})
	if recentLimit > 0 && len(recent) > recentLimit {
		recent = recent[:recentLimit]
	}
	out.Recent = recent
	surfaceRecent := make([]autonomyEventSummary, 0, len(recent))
	for _, item := range recent {
		surfaceRecent = append(surfaceRecent, item.autonomyEventSummary)
	}
	out.Highlights = buildAutonomyHighlights(out.ByKind, surfaceRecent, out.LastAt, 4)
	return out
}

func formatAutonomySurfaceSummary(surface *autonomySurface) string {
	if surface == nil || surface.Events <= 0 {
		return ""
	}
	partes := []string{fmt.Sprintf("%d evento(s)", surface.Events)}
	if surface.LastAt != nil && !surface.LastAt.IsZero() {
		partes = append(partes, "último "+surface.LastAt.UTC().Format("2006-01-02 15:04:05"))
	}
	top := topCountPairs(surface.ByKind, 3)
	if len(top) > 0 {
		kinds := make([]string, 0, len(top))
		for _, item := range top {
			kinds = append(kinds, fmt.Sprintf("%s=%d", item.Key, item.Count))
		}
		partes = append(partes, strings.Join(kinds, ", "))
	}
	return strings.Join(partes, " · ")
}
