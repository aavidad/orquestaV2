/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package forge

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestGitHubClientCreateRepoOrg(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/orgs/dipgra/repos" {
				t.Fatalf("path inesperado: %s", r.URL.Path)
			}
			if got := r.Header.Get("Authorization"); got != "Bearer secret-token" {
				t.Fatalf("auth inesperada: %q", got)
			}
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), `"name":"demo"`) {
				t.Fatalf("body inesperado: %s", string(body))
			}
			return &http.Response{
				StatusCode: http.StatusCreated,
				Header:     make(http.Header),
				Body: io.NopCloser(bytes.NewBufferString(`{
			"name":"demo",
			"full_name":"dipgra/demo",
			"html_url":"https://github.test/dipgra/demo",
			"clone_url":"https://github.test/dipgra/demo.git",
			"ssh_url":"git@github.test:dipgra/demo.git",
			"default_branch":"main",
			"private":true,
			"owner":{"login":"dipgra"}
		}`)),
			}, nil
		}),
	}

	info, err := (&GitHubClient{HTTPClient: client}).CreateRepo(GitHubConfig{
		APIBaseURL: "https://api.github.test",
		Owner:      "dipgra",
		OwnerKind:  "org",
		Token:      "secret-token",
	}, GitHubCreateRepoRequest{
		Name:       "demo",
		Visibility: "private",
	})
	if err != nil {
		t.Fatalf("CreateRepo: %v", err)
	}
	if info.FullName != "dipgra/demo" || info.Visibility != "private" {
		t.Fatalf("repo inesperado: %+v", info)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}
