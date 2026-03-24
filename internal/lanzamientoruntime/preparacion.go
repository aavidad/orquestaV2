package lanzamientoruntime

import (
	"database/sql"
	"fmt"
	"strings"

	"orquesta/db"
	"orquesta/internal/bootstrapruntime"
	"orquesta/runtimeagente"
)

type Preparacion struct {
	Agente    *db.Agente
	Proyecto  *db.Proyecto
	Conector  *db.Conector
	Ultima    *db.Sesion
	Bootstrap *bootstrapruntime.State
	Plan      *runtimeagente.LaunchPlan
}

func PrepararDesdeRefs(agenteRef, proyectoRef, conectorRef, modelo, razonamiento, perfilTarea string) (*Preparacion, error) {
	agente, err := db.GetAgente(strings.TrimSpace(agenteRef))
	if err != nil {
		return nil, err
	}
	proyecto, err := db.GetProyecto(strings.TrimSpace(proyectoRef))
	if err != nil {
		return nil, err
	}
	ultima, err := db.ObtenerUltimaSesion(strings.TrimSpace(agenteRef), &proyecto.ID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	conector, err := ResolverConector(strings.TrimSpace(agenteRef), conectorRef, ultima)
	if err != nil {
		return nil, err
	}
	return PrepararDesdeDatos(agente, proyecto, conector, ultima, modelo, razonamiento, perfilTarea)
}

func PrepararDesdeDatos(agente *db.Agente, proyecto *db.Proyecto, conector *db.Conector, ultima *db.Sesion, modelo, razonamiento, perfilTarea string) (*Preparacion, error) {
	if agente == nil || proyecto == nil || conector == nil {
		return nil, fmt.Errorf("agente, proyecto y conector son obligatorios")
	}

	resume, bootstrap, err := bootstrapruntime.Preparar(agente.Nombre, proyecto, ultima)
	if err != nil {
		return nil, err
	}
	req := runtimeagente.LaunchRequest{
		Agente:       strings.TrimSpace(agente.Nombre),
		Rol:          strings.TrimSpace(agente.Rol),
		ProyectoSlug: strings.TrimSpace(proyecto.Slug),
		ProyectoRuta: strings.TrimSpace(proyecto.RutaAbs),
		Modelo:       strings.TrimSpace(modelo),
		Razonamiento: strings.TrimSpace(razonamiento),
		PerfilTarea:  strings.TrimSpace(perfilTarea),
		Conector: runtimeagente.ConnectorConfig{
			Slug:         strings.TrimSpace(conector.Slug),
			Nombre:       strings.TrimSpace(conector.Nombre),
			Transporte:   strings.TrimSpace(conector.Transporte),
			Comando:      strings.TrimSpace(conector.Comando),
			ArgsJSON:     strings.TrimSpace(conector.ArgsJSON),
			EnvJSON:      strings.TrimSpace(conector.EnvJSON),
			MetadataJSON: strings.TrimSpace(conector.MetadataJSON),
			Activo:       conector.Activo,
		},
		Resume: resume,
	}
	plan, err := runtimeagente.DefaultRegistry().Prepare(req)
	if err != nil {
		return nil, err
	}

	return &Preparacion{
		Agente:    agente,
		Proyecto:  proyecto,
		Conector:  conector,
		Ultima:    ultima,
		Bootstrap: bootstrap,
		Plan:      plan,
	}, nil
}

func ResolverConector(agente, conectorRef string, ultima *db.Sesion) (*db.Conector, error) {
	ref := strings.TrimSpace(conectorRef)
	if ref == "" && ultima != nil {
		if strings.TrimSpace(ultima.ConectorSlug) != "" {
			ref = strings.TrimSpace(ultima.ConectorSlug)
		} else if ultima.ConectorID != nil && *ultima.ConectorID > 0 {
			ref = fmt.Sprintf("%d", *ultima.ConectorID)
		}
	}
	if ref == "" {
		ref = "codex-cli"
	}
	conector, err := db.GetConector(ref)
	if err != nil {
		if ref != "codex-cli" {
			return nil, fmt.Errorf("no se pudo resolver el conector para %s: %w", strings.TrimSpace(agente), err)
		}
		return nil, err
	}
	return conector, nil
}

func ResumeContextNoVacio(resume runtimeagente.ResumeContext) bool {
	return strings.TrimSpace(resume.ExternalSessionID) != "" ||
		strings.TrimSpace(resume.ResumePayloadJSON) != "" ||
		strings.TrimSpace(resume.ResumenContinuidad) != ""
}
