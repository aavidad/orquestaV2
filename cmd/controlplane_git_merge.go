package cmd

import (
	"encoding/json"
	"strconv"
	"strings"

	"orquesta/coordinacion"
	"orquesta/db"
	"orquesta/gitgobernanza"
	"orquesta/gitoperaciones"
	"orquesta/progresoapp"
)

func (dbAutomationService) ProcesarGitMergesBatch() (int, error) {
	return procesarGitMergesBatch()
}

type autonomiaGitMergeMetadata struct {
	AutoCreated bool   `json:"auto_created"`
	Source      string `json:"source"`
	ReviewGate  int64  `json:"review_gate"`
	WorktreeID  *int64 `json:"worktree_id,omitempty"`
	TareaID     *int64 `json:"tarea_id,omitempty"`
}

func procesarGitMergesBatch() (int, error) {
	svc := gitgobernanza.NewService(gitgobernanza.Repository{})
	processed := map[int64]struct{}{}
	total := 0
	for _, estado := range []string{"aprobado", "ejecutando"} {
		merges, err := svc.ListRequests("", estado)
		if err != nil {
			return total, err
		}
		for _, merge := range merges {
			if merge == nil {
				continue
			}
			if _, ok := processed[merge.ID]; ok {
				continue
			}
			processed[merge.ID] = struct{}{}
			ok, err := procesarGitMergeAutonomo(svc, merge)
			if err != nil {
				return total, err
			}
			if ok {
				total++
			}
		}
	}
	return total, nil
}

func procesarGitMergeAutonomo(svc *gitgobernanza.Service, merge *gitgobernanza.GitMerge) (bool, error) {
	if svc == nil || merge == nil || merge.ProyectoID == 0 {
		return false, nil
	}
	meta := autonomiaGitMergeMetadata{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(merge.MetadataJSON)), &meta); err != nil {
		return false, nil
	}
	if !meta.AutoCreated || strings.TrimSpace(meta.Source) != "review_gate_approved" {
		return false, nil
	}
	proyecto, err := db.GetProyecto(strconv.FormatInt(merge.ProyectoID, 10))
	if err != nil || proyecto == nil {
		return false, err
	}
	policy, err := supervisionService.GetProjectPolicy(proyecto.Slug)
	if err != nil {
		return false, err
	}
	if policy == nil || !policy.Enabled {
		return false, nil
	}
	if strings.TrimSpace(merge.Estado) != "ejecutando" {
		if err := guardarEstadoGitMerge(svc, proyecto.Slug, merge, "ejecutando", merge.CommitOrigen, merge.CommitMerge, appendNotaGitMerge(merge.Notas, "Orquesta: iniciando integración automática.")); err != nil {
			return false, err
		}
		merge.Estado = "ejecutando"
	}
	result, err := gitoperaciones.MergeBranchIsolated(strings.TrimSpace(proyecto.RutaAbs), strings.TrimSpace(merge.SourceBranch), strings.TrimSpace(merge.TargetBranch))
	if err != nil {
		if saveErr := guardarEstadoGitMerge(svc, proyecto.Slug, merge, "fallido", merge.CommitOrigen, merge.CommitMerge, appendNotaGitMerge(merge.Notas, "Orquesta: integración automática fallida: "+err.Error())); saveErr != nil {
			return false, saveErr
		}
		return true, nil
	}
	if err := guardarEstadoGitMerge(svc, proyecto.Slug, merge, "fusionado", result.SourceCommit, result.MergeCommit, appendNotaGitMerge(merge.Notas, "Orquesta: integración automática completada.")); err != nil {
		return false, err
	}
	if err := completarTareaTrasMerge(meta.TareaID, merge.RequestedBy, result.MergeCommit); err != nil {
		return false, err
	}
	if err := cerrarWorktreeTrasMerge(meta.WorktreeID, "merge_fusionado_automaticamente"); err != nil {
		return false, err
	}
	if err := reflejarCierreIntegracionTrasMerge(proyecto, policy); err != nil {
		return false, err
	}
	return true, nil
}

func guardarEstadoGitMerge(svc *gitgobernanza.Service, proyectoSlug string, merge *gitgobernanza.GitMerge, estado, commitOrigen, commitMerge, notas string) error {
	if svc == nil || merge == nil {
		return nil
	}
	_, err := svc.SaveRequest(gitgobernanza.SaveMergeRequestInput{
		ID:           merge.ID,
		ProyectoSlug: proyectoSlug,
		SourceBranch: merge.SourceBranch,
		TargetBranch: merge.TargetBranch,
		RequestedBy:  valorConFallback(strings.TrimSpace(merge.RequestedBy), "orquesta"),
		Estado:       strings.TrimSpace(estado),
		CommitOrigen: strings.TrimSpace(commitOrigen),
		CommitMerge:  strings.TrimSpace(commitMerge),
		Notas:        strings.TrimSpace(notas),
		MetadataJSON: strings.TrimSpace(merge.MetadataJSON),
	})
	return err
}

func completarTareaTrasMerge(tareaID *int64, agente, mergeCommit string) error {
	if tareaID == nil || *tareaID <= 0 {
		return nil
	}
	tarea, err := tareasService.Get(*tareaID)
	if err != nil || tarea == nil {
		return err
	}
	switch tarea.Estado {
	case db.TareaCompletada, db.TareaCancelada:
		return nil
	}
	return tareasService.Complete(*tareaID, valorConFallback(strings.TrimSpace(agente), "orquesta"), strings.TrimSpace(mergeCommit))
}

func cerrarWorktreeTrasMerge(worktreeID *int64, motivo string) error {
	if worktreeID == nil || *worktreeID <= 0 {
		return nil
	}
	svc := newCoordinationService()
	worktree, err := svc.GetWorktree(*worktreeID)
	if err != nil || worktree == nil {
		return err
	}
	if worktree.LockID != nil && *worktree.LockID > 0 {
		lock, err := svc.GetLock(*worktree.LockID)
		if err != nil {
			return err
		}
		if lock != nil && lock.State == coordinacion.LockActive {
			if _, err := svc.ReleaseLock(coordinacion.ReleaseLockInput{
				ID:         lock.ID,
				Agent:      lock.Agent,
				LeaseToken: lock.LeaseToken,
				Reason:     motivo,
			}); err != nil {
				return err
			}
		}
	}
	if worktree.State == coordinacion.WorktreeClosed {
		return nil
	}
	_, err = svc.CloseWorktree(*worktreeID, true, motivo)
	return err
}

func reflejarCierreIntegracionTrasMerge(proyecto *db.Proyecto, policy *db.ProyectoAutonomia) error {
	if proyecto == nil {
		return nil
	}
	fases, err := progresoService.ListPhases(strings.TrimSpace(proyecto.Slug))
	if err != nil {
		return err
	}
	for _, fase := range fases {
		if fase == nil || !strings.EqualFold(strings.TrimSpace(fase.Nombre), "integracion") {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(fase.Estado), "completada") {
			break
		}
		estado := "completada"
		if _, err := progresoService.UpdatePhase(progresoapp.UpdatePhaseInput{
			ID:     fase.ID,
			Estado: &estado,
		}); err != nil {
			return err
		}
		break
	}
	if policy == nil || !policy.Enabled || !policy.AutoCloseProject {
		return nil
	}
	terminado, _, err := proyectoTerminadoAutonomamente(proyecto)
	if err != nil || !terminado {
		return err
	}
	return persistirEstadoProyectoAutonomia(policy, db.AutonomiaProyectoCerrando)
}

func appendNotaGitMerge(prev, next string) string {
	prev = strings.TrimSpace(prev)
	next = strings.TrimSpace(next)
	switch {
	case prev == "":
		return next
	case next == "":
		return prev
	default:
		return prev + "\n" + next
	}
}
