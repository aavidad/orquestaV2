package skillsapp

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type GitFetcher struct{}

func NewGitFetcher() GitFetcher {
	return GitFetcher{}
}

type RemoteSkill struct {
	Name         string
	Description  string
	Repo         string
	SkillRef     string
	CanonicalURL string
	SourceKind   string
}

func (GitFetcher) Fetch(ctx context.Context, spec SourceSpec) (*RemoteSkill, error) {
	resolved, err := resolveSourceSpec(spec)
	if err != nil {
		return nil, err
	}
	if _, err := exec.LookPath("git"); err != nil {
		return nil, fmt.Errorf("git no disponible: %w", err)
	}
	dir, err := os.MkdirTemp("", "orquesta-skill-import-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	repoDir := filepath.Join(dir, "repo")
	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", resolved.RepoURL, repoDir) //nolint:gosec
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("clonando repositorio: %v (%s)", err, strings.TrimSpace(string(out)))
	}

	skillDir, err := findSkillDir(repoDir, resolved.Skill)
	if err != nil {
		return nil, err
	}
	name, description, err := readSkillFrontmatter(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		return nil, err
	}
	if name == "" {
		name = strings.TrimSpace(resolved.Skill)
	}
	if name == "" {
		name = filepath.Base(skillDir)
	}
	if description == "" {
		description = "Skill importada desde skills.sh"
	}
	return &RemoteSkill{
		Name:         name,
		Description:  description,
		Repo:         resolved.Repo,
		SkillRef:     resolved.Skill,
		CanonicalURL: resolved.CanonicalURL,
		SourceKind:   resolved.SourceKind,
	}, nil
}

type resolvedSourceSpec struct {
	Repo         string
	RepoURL      string
	Skill        string
	CanonicalURL string
	SourceKind   string
}

func resolveSourceSpec(spec SourceSpec) (*resolvedSourceSpec, error) {
	repo := strings.TrimSpace(spec.Repo)
	skill := strings.TrimSpace(spec.Skill)
	rawURL := strings.TrimSpace(spec.URL)
	if rawURL != "" {
		u, err := url.Parse(rawURL)
		if err != nil {
			return nil, fmt.Errorf("url invalida: %w", err)
		}
		host := strings.ToLower(strings.TrimSpace(u.Host))
		parts := pathParts(u.Path)
		switch host {
		case "skills.sh", "www.skills.sh":
			if len(parts) < 2 {
				return nil, fmt.Errorf("url de skills.sh incompleta")
			}
			repo = parts[0] + "/" + parts[1]
			if skill == "" && len(parts) >= 3 {
				skill = parts[2]
			}
			return &resolvedSourceSpec{
				Repo:         repo,
				RepoURL:      "https://github.com/" + repo,
				Skill:        skill,
				CanonicalURL: rawURL,
				SourceKind:   "skills.sh",
			}, nil
		case "github.com", "www.github.com":
			if len(parts) < 2 {
				return nil, fmt.Errorf("url de github incompleta")
			}
			repo = parts[0] + "/" + parts[1]
			if skill == "" && len(parts) >= 3 {
				skill = parts[2]
			}
			return &resolvedSourceSpec{
				Repo:         repo,
				RepoURL:      "https://github.com/" + repo,
				Skill:        skill,
				CanonicalURL: rawURL,
				SourceKind:   "github",
			}, nil
		default:
			return nil, fmt.Errorf("host no soportado: %s", host)
		}
	}
	if repo == "" {
		return nil, fmt.Errorf("repo o url obligatorios")
	}
	repo = strings.TrimPrefix(repo, "https://github.com/")
	repo = strings.Trim(repo, "/")
	return &resolvedSourceSpec{
		Repo:         repo,
		RepoURL:      "https://github.com/" + repo,
		Skill:        skill,
		CanonicalURL: buildCanonicalURL(repo, skill),
		SourceKind:   "skills.sh",
	}, nil
}

func buildCanonicalURL(repo, skill string) string {
	parts := strings.Split(strings.Trim(repo, "/"), "/")
	if len(parts) == 2 {
		if skill != "" {
			return "https://skills.sh/" + parts[0] + "/" + parts[1] + "/" + skill
		}
		return "https://skills.sh/" + parts[0] + "/" + parts[1]
	}
	if skill != "" {
		return "https://github.com/" + repo + "/" + skill
	}
	return "https://github.com/" + repo
}

func pathParts(path string) []string {
	raw := strings.Split(strings.Trim(path, "/"), "/")
	out := make([]string, 0, len(raw))
	for _, part := range raw {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func findSkillDir(repoDir, skill string) (string, error) {
	if _, err := os.Stat(filepath.Join(repoDir, "SKILL.md")); err == nil && strings.TrimSpace(skill) == "" {
		return repoDir, nil
	}
	if skill != "" {
		candidate := filepath.Join(repoDir, skill)
		if _, err := os.Stat(filepath.Join(candidate, "SKILL.md")); err == nil {
			return candidate, nil
		}
	}
	entries, err := os.ReadDir(repoDir)
	if err != nil {
		return "", err
	}
	matches := []string{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		candidate := filepath.Join(repoDir, entry.Name())
		if _, err := os.Stat(filepath.Join(candidate, "SKILL.md")); err == nil {
			if skill == "" || strings.EqualFold(entry.Name(), skill) {
				matches = append(matches, candidate)
			}
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if skill != "" {
		return "", fmt.Errorf("skill %s no encontrada en el repositorio", skill)
	}
	return "", fmt.Errorf("no se pudo resolver una skill unica en el repositorio")
}
