package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"orquesta/db"
	"orquesta/microprogramacionapp"
)

func recogerResultadoGitSubagente(item *db.SupervisorSubagent) (map[string]any, error) {
	if item == nil {
		return nil, nil
	}
	agente := strings.TrimSpace(item.SubagentName)
	proyecto := strings.TrimSpace(item.ProyectoSlug)
	if agente == "" || proyecto == "" {
		return nil, nil
	}
	proyectoItem, err := runtimesService.GetProject(proyecto)
	if err != nil || proyectoItem == nil {
		return nil, nil
	}
	var evidencia []string
	if path := strings.TrimSpace(item.OutputPath); path != "" {
		evidencia = append(evidencia, "output_path="+path)
	}
	if path := strings.TrimSpace(item.ManifestPath); path != "" {
		evidencia = append(evidencia, "manifest_path="+path)
	}
	if threadID := strings.TrimSpace(item.ThreadID); threadID != "" {
		evidencia = append(evidencia, "thread_id="+threadID)
	}
	preferencia := preferenciaEntregaGitSubagente(item)
	resultado, err := runtimesService.RegistrarEntregaGitMicroprogramacionActivaPreferente(
		agente,
		&proyectoItem.ID,
		proyecto,
		strings.TrimSpace(strings.Join(evidencia, "\n")),
		strings.TrimSpace(item.Supervisor),
		preferencia,
	)
	if err != nil {
		return nil, err
	}
	if resultado == nil || resultado.Entrega == nil {
		return nil, nil
	}
	return map[string]any{
		"especificacion_id":   resultado.Contexto.EspecificacionID,
		"runtime_order_id":    resultado.Contexto.RuntimeOrderID,
		"git_merge_id":        resultado.Entrega.GitMergeID,
		"worktree_id":         resultado.Entrega.WorktreeID,
		"ruta_worktree":       resultado.Entrega.RutaWorktree,
		"source_branch":       resultado.Entrega.SourceBranch,
		"target_branch":       resultado.Entrega.TargetBranch,
		"head_commit":         resultado.Entrega.HeadCommit,
		"archivos_entregados": resultado.Entrega.ArchivosEntregados,
		"receipt_source":      "git_worktree",
		"supervisor_subagent": fmt.Sprintf("%s:%d", strings.TrimSpace(item.Supervisor), item.ID),
	}, nil
}

func preferenciaEntregaGitSubagente(item *db.SupervisorSubagent) microprogramacionapp.EntradaRegistrarEntregaGit {
	entrada := microprogramacionapp.EntradaRegistrarEntregaGit{}
	if item == nil || strings.TrimSpace(item.MetadataJSON) == "" {
		return entrada
	}
	var metadata map[string]any
	if err := json.Unmarshal([]byte(item.MetadataJSON), &metadata); err != nil || metadata == nil {
		return entrada
	}
	if id := int64SupervisorSubagente(metadata["worktree_id"]); id > 0 {
		entrada.PreferenciaWorktreeID = &id
	}
	entrada.PreferenciaRutaWorktree = strings.TrimSpace(stringSupervisorSubagente(metadata["ruta_worktree"]))
	entrada.PreferenciaBranch = strings.TrimSpace(stringSupervisorSubagente(metadata["branch_worktree"]))
	entrada.PreferenciaBaseRef = strings.TrimSpace(stringSupervisorSubagente(metadata["base_ref_worktree"]))
	return entrada
}

func int64SupervisorSubagente(v any) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case int:
		return int64(x)
	case float64:
		return int64(x)
	case json.Number:
		n, _ := x.Int64()
		return n
	case string:
		var n int64
		fmt.Sscanf(strings.TrimSpace(x), "%d", &n)
		return n
	}
	return 0
}

func stringSupervisorSubagente(v any) string {
	s, _ := v.(string)
	return s
}
