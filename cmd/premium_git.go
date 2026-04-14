package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"orquesta/db"
	"orquesta/gitaplicacion"
	"orquesta/gitoperaciones"
	"orquesta/microprogramacionapp"
	"orquesta/runtimesapp"
)

type premiumRuntimeGitService struct {
	git *gitaplicacion.Service
}

func (s premiumRuntimeGitService) RegistrarEntregaGitPremium(entrada runtimesapp.EntradaRegistrarEntregaGitPremium) (*runtimesapp.ResultadoRegistrarEntregaGitPremium, error) {
	agente := strings.TrimSpace(entrada.Agente)
	proyectoSlug := strings.TrimSpace(entrada.ProyectoSlug)
	if agente == "" {
		return nil, fmt.Errorf("agente obligatorio")
	}
	if proyectoSlug == "" {
		return nil, fmt.Errorf("proyecto obligatorio")
	}
	svc := s.git
	if svc == nil {
		svc = gitService
	}
	preferencia := microprogramacionapp.EntradaRegistrarEntregaGit{
		Agente:                  agente,
		ProyectoID:              entrada.ProyectoID,
		ProyectoSlug:            proyectoSlug,
		PreferenciaWorktreeID:   entrada.PreferenciaWorktreeID,
		PreferenciaRutaWorktree: strings.TrimSpace(entrada.PreferenciaRutaWorktree),
		PreferenciaBranch:       strings.TrimSpace(entrada.PreferenciaBranch),
		PreferenciaBaseRef:      strings.TrimSpace(entrada.PreferenciaBaseRef),
	}
	worktree, err := capturarWorktreePremiumEntrega(svc, agente, proyectoSlug, preferencia)
	if err != nil {
		return nil, err
	}
	if worktree == nil {
		return nil, fmt.Errorf("no existe worktree activa para agente=%s proyecto=%s", agente, proyectoSlug)
	}
	sourceBranch := firstNonEmpty(strings.TrimSpace(preferencia.PreferenciaBranch), strings.TrimSpace(worktree.Branch))
	targetBranch := firstNonEmpty(strings.TrimSpace(preferencia.PreferenciaBaseRef), strings.TrimSpace(worktree.BaseRef), "main")
	if sourceBranch == "" {
		return nil, fmt.Errorf("branch de entrega obligatoria")
	}
	if existente, err := mergePendientePremiumExistente(svc, proyectoSlug, sourceBranch, targetBranch); err != nil {
		return nil, err
	} else if existente != nil {
		return &runtimesapp.ResultadoRegistrarEntregaGitPremium{
			WorktreeID:   worktree.ID,
			RutaWorktree: strings.TrimSpace(worktree.RutaAbs),
			SourceBranch: sourceBranch,
			TargetBranch: targetBranch,
			GitMergeID:   existente.ID,
		}, nil
	}
	estado, err := gitoperaciones.InspeccionarTrabajoGit(strings.TrimSpace(worktree.RutaAbs))
	if err != nil {
		return nil, err
	}
	if len(estado.ArchivosModificados) == 0 {
		return nil, fmt.Errorf("la entrega git no contiene archivos modificados")
	}
	if err := validarWriteSetPremium(entrada.WriteSet, estado.ArchivosModificados); err != nil {
		return nil, err
	}
	metadataJSON := metadataEntregaGitPremium(entrada, worktree, estado, targetBranch)
	notas := notasEntregaGitPremium(entrada, worktree)
	mergeID, err := svc.CreateMerge(gitaplicacion.CreateMergeInput{
		ProyectoSlug: proyectoSlug,
		SourceBranch: sourceBranch,
		TargetBranch: targetBranch,
		RequestedBy:  firstNonEmpty(strings.TrimSpace(entrada.SolicitadoPor), agente),
		Estado:       "pendiente",
		SourceCommit: strings.TrimSpace(estado.HeadCommit),
		Notas:        notas,
		MetadataJSON: metadataJSON,
	})
	if err != nil {
		return nil, err
	}
	return &runtimesapp.ResultadoRegistrarEntregaGitPremium{
		WorktreeID:         worktree.ID,
		RutaWorktree:       strings.TrimSpace(worktree.RutaAbs),
		SourceBranch:       sourceBranch,
		TargetBranch:       targetBranch,
		HeadCommit:         strings.TrimSpace(estado.HeadCommit),
		ArchivosEntregados: append([]string(nil), estado.ArchivosModificados...),
		GitMergeID:         mergeID,
	}, nil
}

func capturarWorktreePremiumEntrega(svc *gitaplicacion.Service, agente, proyectoSlug string, entrada microprogramacionapp.EntradaRegistrarEntregaGit) (*db.Worktree, error) {
	worktree, err := resolverWorktreePreferidaEntrega(proyectoSlug, agente, entrada)
	if err != nil {
		return nil, err
	}
	if worktree != nil {
		return worktree, nil
	}
	return svc.ResolveActiveWorktree(proyectoSlug, agente)
}

func mergePendientePremiumExistente(svc *gitaplicacion.Service, proyectoSlug, sourceBranch, targetBranch string) (*db.GitMergeRequest, error) {
	pendientes, err := svc.ListMerges(strings.TrimSpace(proyectoSlug), "pendiente")
	if err != nil {
		return nil, err
	}
	for _, item := range pendientes {
		if item == nil {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(item.SourceBranch), strings.TrimSpace(sourceBranch)) &&
			strings.EqualFold(strings.TrimSpace(item.TargetBranch), strings.TrimSpace(targetBranch)) {
			return item, nil
		}
	}
	return nil, nil
}

func notasEntregaGitPremium(entrada runtimesapp.EntradaRegistrarEntregaGitPremium, worktree *db.Worktree) string {
	lineas := []string{
		fmt.Sprintf("Entrega premium registrada desde worktree %s.", strings.TrimSpace(worktree.RutaAbs)),
	}
	if strings.TrimSpace(entrada.Carril) != "" {
		lineas = append(lineas, "Carril: "+strings.TrimSpace(entrada.Carril))
	}
	if entrada.TareaObjetivoID > 0 {
		lineas = append(lineas, fmt.Sprintf("Tarea objetivo: #%d", entrada.TareaObjetivoID))
	}
	if evidencia := strings.TrimSpace(entrada.Evidencia); evidencia != "" {
		lineas = append(lineas, evidencia)
	}
	return strings.Join(lineas, "\n")
}

func metadataEntregaGitPremium(entrada runtimesapp.EntradaRegistrarEntregaGitPremium, worktree *db.Worktree, estado *gitoperaciones.EstadoTrabajoGit, targetBranch string) string {
	payload := map[string]any{
		"source":              "premium_runtime",
		"agente":              strings.TrimSpace(entrada.Agente),
		"carril":              strings.TrimSpace(entrada.Carril),
		"tarea_objetivo_id":   entrada.TareaObjetivoID,
		"worktree_id":         worktree.ID,
		"ruta_worktree":       strings.TrimSpace(worktree.RutaAbs),
		"source_branch":       strings.TrimSpace(worktree.Branch),
		"target_branch":       strings.TrimSpace(targetBranch),
		"head_commit":         strings.TrimSpace(estado.HeadCommit),
		"archivos_entregados": append([]string(nil), estado.ArchivosModificados...),
		"write_set":           append([]string(nil), entrada.WriteSet...),
	}
	raw, _ := json.Marshal(payload)
	return string(raw)
}

func validarWriteSetPremium(writeSet, archivos []string) error {
	writeSetNormalizado := normalizarWriteSetPremium(writeSet)
	if len(writeSetNormalizado) == 0 {
		return fmt.Errorf("write_set premium vacio")
	}
	permitidos := make(map[string]struct{}, len(writeSetNormalizado))
	for _, ruta := range writeSetNormalizado {
		permitidos[ruta] = struct{}{}
	}
	var fuera []string
	for _, ruta := range normalizarWriteSetPremium(archivos) {
		if _, ok := permitidos[ruta]; ok {
			continue
		}
		fuera = append(fuera, ruta)
	}
	if len(fuera) > 0 {
		return fmt.Errorf("la entrega git premium modifica archivos fuera del write_set: %s", strings.Join(fuera, ", "))
	}
	return nil
}

func normalizarWriteSetPremium(rutas []string) []string {
	out := make([]string, 0, len(rutas))
	seen := make(map[string]struct{}, len(rutas))
	for _, ruta := range rutas {
		ruta = strings.TrimSpace(ruta)
		if ruta == "" {
			continue
		}
		if _, ok := seen[ruta]; ok {
			continue
		}
		seen[ruta] = struct{}{}
		out = append(out, ruta)
	}
	return out
}
