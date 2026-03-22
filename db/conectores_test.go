/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import "testing"

func TestRegistrarConectorForgeYSubproyectoRemoto(t *testing.T) {
	abrirDBTemporalMemoria(t)

	id, err := RegistrarConectorForge(&ConectorForge{
		Slug:       "gh-main",
		Tipo:       "github",
		Owner:      "dipgra",
		OwnerKind:  "org",
		APIBaseURL: "https://api.github.test",
		TokenEnv:   "GITHUB_TOKEN",
		Activo:     true,
	})
	if err != nil {
		t.Fatalf("RegistrarConectorForge: %v", err)
	}
	if id == 0 {
		t.Fatalf("id de conector inesperado")
	}

	conector, err := GetConectorForge("gh-main")
	if err != nil {
		t.Fatalf("GetConectorForge: %v", err)
	}
	if conector.Owner != "dipgra" || conector.Tipo != "github" {
		t.Fatalf("conector inesperado: %+v", conector)
	}

	spID, err := RegistrarSubproyectoRemoto(&SubproyectoRemoto{
		Proyecto:      "orquestador",
		ConectorSlug:  "gh-main",
		ForgeTipo:     "github",
		Owner:         "dipgra",
		RepoName:      "subproyecto-demo",
		RepoFullName:  "dipgra/subproyecto-demo",
		Visibility:    "private",
		Descripcion:   "Demo",
		HTMLURL:       "https://github.test/dipgra/subproyecto-demo",
		CloneURL:      "https://github.test/dipgra/subproyecto-demo.git",
		SSHURL:        "git@github.test:dipgra/subproyecto-demo.git",
		DefaultBranch: "main",
		Estado:        "creado",
		RegistradoPor: "Codex3",
	})
	if err != nil {
		t.Fatalf("RegistrarSubproyectoRemoto: %v", err)
	}
	if spID == 0 {
		t.Fatalf("id de subproyecto inesperado")
	}

	items, err := ListarSubproyectosRemotos("orquestador")
	if err != nil {
		t.Fatalf("ListarSubproyectosRemotos: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("subproyectos inesperados: %+v", items)
	}
	if items[0].RepoFullName != "dipgra/subproyecto-demo" {
		t.Fatalf("repo inesperado: %+v", items[0])
	}
}
