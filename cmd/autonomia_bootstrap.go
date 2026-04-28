package cmd

import (
	"strings"

	"orquesta/db"
	"orquesta/supervisionapp"
)

func bootstrapPersistentAutonomyProject(projectRef string, policy *supervisionapp.Policy, activationPrefix string) error {
	if policy == nil || !policy.Enabled {
		return nil
	}
	projectRef = strings.TrimSpace(projectRef)
	if projectRef == "" {
		return nil
	}
	proyecto, err := db.GetProyecto(projectRef)
	if err != nil || proyecto == nil {
		return err
	}
	supervisor, _, err := resolverSupervisorAutonomiaOperativo(
		proyecto.ID,
		true,
		canonicalAutonomyCodexName(strings.TrimSpace(policy.SupervisorAgente)),
		activationPrefix+"_supervisor",
		"",
	)
	if err != nil {
		return err
	}
	supervisorNombre := ""
	if supervisor != nil {
		supervisorNombre = strings.TrimSpace(supervisor.Nombre)
		if _, err := asegurarTrabajoAutonomia(policy, proyecto, supervisor); err != nil {
			return err
		}
	}
	reviewerNombre := canonicalAutonomyCodexName(strings.TrimSpace(policy.ReviewerAgente))
	if policy.ReviewRequired {
		reviewer, err := repoPersistentResolveAgent(proyecto.ID, reviewerNombre, []string{supervisorNombre}, activationPrefix+"_reviewer")
		if err != nil {
			return err
		}
		if reviewer != nil {
			reviewerNombre = strings.TrimSpace(reviewer.Nombre)
		}
	}
	if policy != nil {
		persistSupervisor := canonicalAutonomyCodexName(strings.TrimSpace(policy.SupervisorAgente))
		persistReviewer := canonicalAutonomyCodexName(strings.TrimSpace(policy.ReviewerAgente))
		if supervisorNombre != "" && !strings.EqualFold(persistSupervisor, supervisorNombre) {
			persistSupervisor = supervisorNombre
		}
		if reviewerNombre != "" && !strings.EqualFold(persistReviewer, reviewerNombre) {
			persistReviewer = reviewerNombre
		}
		if persistSupervisor != strings.TrimSpace(policy.SupervisorAgente) || persistReviewer != strings.TrimSpace(policy.ReviewerAgente) {
			policyActualizada, err := supervisionService.UpsertProjectPolicy(strings.TrimSpace(proyecto.Slug), supervisionapp.PolicyInput{
				Enabled:              policy.Enabled,
				ObjetivoGeneral:      policy.ObjetivoGeneral,
				DefinitionOfDoneJSON: policy.DefinitionOfDoneJSON,
				MaxWorkers:           policy.MaxWorkers,
				SupervisorAgente:     persistSupervisor,
				ReviewerAgente:       persistReviewer,
				ReserveReviewer:      policy.ReserveReviewer,
				ReserveSupervisor:    policy.ReserveSupervisor,
				ReviewRequired:       policy.ReviewRequired,
				AutoCreateTasks:      policy.AutoCreateTasks,
				AutoCloseProject:     policy.AutoCloseProject,
				EstadoAutonomia:      policy.EstadoAutonomia,
			})
			if err != nil {
				return err
			}
			if policyActualizada != nil {
				policy = policyActualizada
			}
		}
	}
	if _, err := repoPersistentEnsureWorkers(proyecto.ID, policy.MaxWorkers, []string{supervisorNombre, reviewerNombre}); err != nil {
		return err
	}
	return db.PlanificarTareasAutomaticamente()
}
