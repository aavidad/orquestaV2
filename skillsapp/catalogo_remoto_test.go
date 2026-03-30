package skillsapp

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestExtraerSkillsRemotasHTMLDeduplicaYConstruyeURL(t *testing.T) {
	html := `
	<html><body>
		<a href="/openai/skills/openai-docs">openai-docs</a>
		<a href="https://skills.sh/openai/skills/openai-docs">openai-docs</a>
		<a href="/vercel-labs/skills/find-skills">find-skills</a>
	</body></html>`

	items := extraerSkillsRemotasHTML(html)
	if len(items) != 2 {
		t.Fatalf("items inesperados: %+v", items)
	}
	if items[0].Repo != "openai/skills" || items[0].Skill != "openai-docs" {
		t.Fatalf("primer item inesperado: %+v", items[0])
	}
	if items[0].URLCanonica != "https://skills.sh/openai/skills/openai-docs" {
		t.Fatalf("url canonica inesperada: %+v", items[0])
	}
}

func TestFiltrarSkillsRemotas(t *testing.T) {
	items := []*SkillRemota{
		{Nombre: "openai-docs", Repo: "openai/skills", Skill: "openai-docs", URLCanonica: "https://skills.sh/openai/skills/openai-docs"},
		{Nombre: "find-skills", Repo: "vercel-labs/skills", Skill: "find-skills", URLCanonica: "https://skills.sh/vercel-labs/skills/find-skills"},
	}

	filtradas := filtrarSkillsRemotas(items, "openai")
	if len(filtradas) != 1 || filtradas[0].Skill != "openai-docs" {
		t.Fatalf("filtro inesperado: %+v", filtradas)
	}
}

func TestSkillsSHCatalogoFetcherListar(t *testing.T) {
	srv := newTestHTTPServerOrSkip(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<a href="/openai/skills/openai-docs">openai-docs</a><a href="/vercel-labs/skills/find-skills">find-skills</a>`))
	}))
	defer srv.Close()

	fetcher := &SkillsSHCatalogoFetcher{
		Client:  srv.Client(),
		BaseURL: srv.URL,
	}
	items, err := fetcher.Listar(context.Background(), "find", 10)
	if err != nil {
		t.Fatalf("Listar: %v", err)
	}
	if len(items) != 1 || !strings.EqualFold(items[0].Skill, "find-skills") {
		t.Fatalf("items inesperados: %+v", items)
	}
}
