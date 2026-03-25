package skillsapp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

type SkillRemota struct {
	Nombre      string `json:"nombre"`
	Repo        string `json:"repo"`
	Skill       string `json:"skill"`
	URLCanonica string `json:"url_canonica"`
	Origen      string `json:"origen"`
}

type CatalogoRemotoFetcher interface {
	Listar(ctx context.Context, filtro string, limite int) ([]*SkillRemota, error)
}

type SkillsSHCatalogoFetcher struct {
	Client  *http.Client
	BaseURL string
}

func NewSkillsSHCatalogoFetcher() *SkillsSHCatalogoFetcher {
	return &SkillsSHCatalogoFetcher{
		Client:  &http.Client{Timeout: 12 * time.Second},
		BaseURL: "https://skills.sh/",
	}
}

var reHrefSkillRemota = regexp.MustCompile(`href=["'](?:https://skills\.sh)?/([A-Za-z0-9_.-]+)/([A-Za-z0-9_.-]+)/([A-Za-z0-9_.-]+)["']`)

func (f *SkillsSHCatalogoFetcher) Listar(ctx context.Context, filtro string, limite int) ([]*SkillRemota, error) {
	if f == nil {
		return nil, fmt.Errorf("fetcher remoto nulo")
	}
	client := f.Client
	if client == nil {
		client = &http.Client{Timeout: 12 * time.Second}
	}
	baseURL := strings.TrimSpace(f.BaseURL)
	if baseURL == "" {
		baseURL = "https://skills.sh/"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("consultando catalogo remoto: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("catalogo remoto devolvio %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	items := extraerSkillsRemotasHTML(string(body))
	items = filtrarSkillsRemotas(items, filtro)
	if limite <= 0 {
		limite = 24
	}
	if len(items) > limite {
		items = items[:limite]
	}
	return items, nil
}

func extraerSkillsRemotasHTML(html string) []*SkillRemota {
	matches := reHrefSkillRemota.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(matches))
	items := make([]*SkillRemota, 0, len(matches))
	for _, match := range matches {
		if len(match) != 4 {
			continue
		}
		repo := strings.TrimSpace(match[1] + "/" + match[2])
		skill := strings.TrimSpace(match[3])
		if repo == "" || skill == "" {
			continue
		}
		key := strings.ToLower(repo + "/" + skill)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		items = append(items, &SkillRemota{
			Nombre:      skill,
			Repo:        repo,
			Skill:       skill,
			URLCanonica: buildCanonicalURL(repo, skill),
			Origen:      "skills.sh",
		})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Repo == items[j].Repo {
			return items[i].Skill < items[j].Skill
		}
		return items[i].Repo < items[j].Repo
	})
	return items
}

func filtrarSkillsRemotas(items []*SkillRemota, filtro string) []*SkillRemota {
	filtro = strings.ToLower(strings.TrimSpace(filtro))
	if filtro == "" {
		return items
	}
	out := make([]*SkillRemota, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		haystack := strings.ToLower(strings.Join([]string{
			item.Nombre,
			item.Repo,
			item.Skill,
			item.URLCanonica,
		}, " "))
		if strings.Contains(haystack, filtro) {
			out = append(out, item)
		}
	}
	return out
}
