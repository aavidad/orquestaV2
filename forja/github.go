/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package forja

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type GitHubConfig struct {
	APIBaseURL string
	Owner      string
	OwnerKind  string
	Token      string
}

type GitHubCreateRepoRequest struct {
	Name        string
	Description string
	Visibility  string
	Homepage    string
}

type GitHubRepoInfo struct {
	Owner         string
	Name          string
	FullName      string
	HTMLURL       string
	CloneURL      string
	SSHURL        string
	DefaultBranch string
	Visibility    string
	Private       bool
	RawJSON       string
}

type GitHubClient struct {
	HTTPClient *http.Client
}

func (c *GitHubClient) CreateRepo(cfg GitHubConfig, req GitHubCreateRepoRequest) (*GitHubRepoInfo, error) {
	cfg.APIBaseURL = strings.TrimRight(strings.TrimSpace(cfg.APIBaseURL), "/")
	cfg.Owner = strings.TrimSpace(cfg.Owner)
	cfg.OwnerKind = strings.TrimSpace(cfg.OwnerKind)
	cfg.Token = strings.TrimSpace(cfg.Token)
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	req.Visibility = strings.TrimSpace(req.Visibility)
	req.Homepage = strings.TrimSpace(req.Homepage)

	if cfg.APIBaseURL == "" {
		cfg.APIBaseURL = "https://api.github.com"
	}
	if cfg.Owner == "" || cfg.OwnerKind == "" || cfg.Token == "" {
		return nil, fmt.Errorf("api_base_url, owner, owner_kind y token son obligatorios")
	}
	if req.Name == "" {
		return nil, fmt.Errorf("name es obligatorio")
	}
	if req.Visibility == "" {
		req.Visibility = "private"
	}

	endpoint := cfg.APIBaseURL + "/user/repos"
	if cfg.OwnerKind == "org" {
		endpoint = cfg.APIBaseURL + "/orgs/" + cfg.Owner + "/repos"
	}

	payload := map[string]any{
		"name":        req.Name,
		"description": req.Description,
		"homepage":    req.Homepage,
		"private":     req.Visibility != "public",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	httpReq, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+cfg.Token)
	httpReq.Header.Set("Accept", "application/vnd.github+json")
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusCreated {
		var apiErr struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(raw, &apiErr)
		msg := strings.TrimSpace(apiErr.Message)
		if msg == "" {
			msg = strings.TrimSpace(string(raw))
		}
		if msg == "" {
			msg = resp.Status
		}
		return nil, fmt.Errorf("github create repo: %s", msg)
	}

	var out struct {
		Name          string `json:"name"`
		FullName      string `json:"full_name"`
		HTMLURL       string `json:"html_url"`
		CloneURL      string `json:"clone_url"`
		SSHURL        string `json:"ssh_url"`
		DefaultBranch string `json:"default_branch"`
		Private       bool   `json:"private"`
		Owner         struct {
			Login string `json:"login"`
		} `json:"owner"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}

	visibility := "public"
	if out.Private {
		visibility = "private"
	}
	return &GitHubRepoInfo{
		Owner:         out.Owner.Login,
		Name:          out.Name,
		FullName:      out.FullName,
		HTMLURL:       out.HTMLURL,
		CloneURL:      out.CloneURL,
		SSHURL:        out.SSHURL,
		DefaultBranch: out.DefaultBranch,
		Visibility:    visibility,
		Private:       out.Private,
		RawJSON:       string(raw),
	}, nil
}
