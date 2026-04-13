/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"fmt"
	"strings"

	"orquesta/coordinacion"
	"orquesta/db"
	"orquesta/gitaplicacion"
	"orquesta/gitoperaciones"
	"orquesta/microprogramacionapp"
)

type microprogramacionRecolectorGit struct{}

func (microprogramacionRecolectorGit) CapturarEntregaGit(agente string, proyectoID *int64, proyectoSlug string) (*microprogramacionapp.EntregaGitCapturada, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return nil, fmt.Errorf("agente obligatorio")
	}
	_ = proyectoID
	proyectoSlug = strings.TrimSpace(proyectoSlug)
	worktree, err := gitService.ResolveActiveWorktree(proyectoSlug, agente)
	if err != nil {
		return nil, err
	}
	estado, err := gitoperaciones.InspeccionarTrabajoGit(strings.TrimSpace(worktree.RutaAbs))
	if err != nil {
		return nil, err
	}
	return &microprogramacionapp.EntregaGitCapturada{
		ProyectoSlug:        strings.TrimSpace(worktree.ProyectoSlug),
		WorktreeID:          worktree.ID,
		RutaWorktree:        strings.TrimSpace(worktree.RutaAbs),
		Branch:              strings.TrimSpace(worktree.Branch),
		BaseRef:             strings.TrimSpace(worktree.BaseRef),
		HeadCommit:          strings.TrimSpace(estado.HeadCommit),
		ArchivosModificados: estado.ArchivosModificados,
		Diff:                strings.TrimSpace(estado.Diff),
	}, nil
}

func (microprogramacionRecolectorGit) CapturarEntregaGitPreferente(agente string, proyectoID *int64, proyectoSlug string, entrada microprogramacionapp.EntradaRegistrarEntregaGit) (*microprogramacionapp.EntregaGitCapturada, error) {
	captura, err := capturarEntregaGitPreferente(agente, proyectoID, proyectoSlug, entrada)
	if err != nil || captura != nil {
		return captura, err
	}
	return microprogramacionRecolectorGit{}.CapturarEntregaGit(agente, proyectoID, proyectoSlug)
}

func capturarEntregaGitPreferente(agente string, proyectoID *int64, proyectoSlug string, entrada microprogramacionapp.EntradaRegistrarEntregaGit) (*microprogramacionapp.EntregaGitCapturada, error) {
	if entrada.PreferenciaWorktreeID == nil && strings.TrimSpace(entrada.PreferenciaRutaWorktree) == "" {
		return nil, nil
	}
	worktree, err := resolverWorktreePreferidaEntrega(proyectoSlug, agente, entrada)
	if err != nil {
		return nil, err
	}
	if worktree == nil {
		return nil, nil
	}
	estado, err := gitoperaciones.InspeccionarTrabajoGit(strings.TrimSpace(worktree.RutaAbs))
	if err != nil {
		return nil, err
	}
	return &microprogramacionapp.EntregaGitCapturada{
		ProyectoSlug:        strings.TrimSpace(worktree.ProyectoSlug),
		WorktreeID:          worktree.ID,
		RutaWorktree:        strings.TrimSpace(worktree.RutaAbs),
		Branch:              strings.TrimSpace(worktree.Branch),
		BaseRef:             strings.TrimSpace(worktree.BaseRef),
		HeadCommit:          strings.TrimSpace(estado.HeadCommit),
		ArchivosModificados: estado.ArchivosModificados,
		Diff:                strings.TrimSpace(estado.Diff),
	}, nil
}

func resolverWorktreePreferidaEntrega(proyectoSlug, agente string, entrada microprogramacionapp.EntradaRegistrarEntregaGit) (*db.Worktree, error) {
	if entrada.PreferenciaWorktreeID != nil && *entrada.PreferenciaWorktreeID > 0 {
		item, err := newCoordinationService().GetWorktree(*entrada.PreferenciaWorktreeID)
		if err == nil && item != nil {
			return adaptarWorktreeCoordinacion(item, strings.TrimSpace(proyectoSlug)), nil
		}
	}
	if strings.TrimSpace(entrada.PreferenciaRutaWorktree) == "" {
		return nil, nil
	}
	return &db.Worktree{
		ID:           int64OrZero(entrada.PreferenciaWorktreeID),
		ProyectoSlug: strings.TrimSpace(proyectoSlug),
		RutaAbs:      strings.TrimSpace(entrada.PreferenciaRutaWorktree),
		Branch:       strings.TrimSpace(entrada.PreferenciaBranch),
		BaseRef:      strings.TrimSpace(entrada.PreferenciaBaseRef),
		Agente:       strings.TrimSpace(agente),
	}, nil
}

func adaptarWorktreeCoordinacion(item *coordinacion.Worktree, proyectoSlug string) *db.Worktree {
	if item == nil {
		return nil
	}
	return &db.Worktree{
		ID:           item.ID,
		ProyectoSlug: strings.TrimSpace(proyectoSlug),
		RutaAbs:      strings.TrimSpace(item.Path),
		Branch:       strings.TrimSpace(item.Branch),
		BaseRef:      strings.TrimSpace(item.BaseRef),
		Agente:       strings.TrimSpace(item.Agent),
	}
}

func int64OrZero(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

type microprogramacionIntegradorGit struct{}

func (microprogramacionIntegradorGit) RegistrarSolicitudMerge(entrada microprogramacionapp.SolicitudMergeMicroprogramacion) (int64, error) {
	return gitService.CreateMerge(gitaplicacion.CreateMergeInput{
		ProyectoSlug: strings.TrimSpace(entrada.ProyectoSlug),
		SourceBranch: strings.TrimSpace(entrada.SourceBranch),
		TargetBranch: strings.TrimSpace(entrada.TargetBranch),
		RequestedBy:  strings.TrimSpace(entrada.SolicitadoPor),
		Estado:       "pendiente",
		SourceCommit: strings.TrimSpace(entrada.CommitOrigen),
		Notas:        strings.TrimSpace(entrada.Notas),
		MetadataJSON: strings.TrimSpace(entrada.MetadataJSON),
	})
}
