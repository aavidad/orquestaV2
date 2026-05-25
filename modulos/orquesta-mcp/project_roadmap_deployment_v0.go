package orquestamcp

func mcpProjectRoadmapDeploymentSourcesV0() []mcpProjectRoadmapItemSourceV0 {
	return []mcpProjectRoadmapItemSourceV0{
		{
			ID:          "CORE-ROADMAP-006",
			Area:        "deployment_plan",
			Status:      "vigente_dry_run_por_puerto",
			Owner:       "orquesta-deploy",
			Focus:       "DeploymentPlan v0 se consume por composicion dry-run desde AppSpec/microtarea de deploy, sin efectos externos.",
			SummaryKey:  "mcp.project.roadmap.core.deployment_plan.summary.v0",
			ProgressKey: "mcp.project.roadmap.core.deployment_plan.dry_run_port_vigente.v0",
			Contracts: []string{
				"DeploymentPlan v0",
				"DeploymentPlanDryRunReceiptV0",
			},
			Dependencies: []string{
				"AppSpecV0",
				"WorkflowTaskV0",
			},
			Guardrails: []string{
				"dry_run_por_puerto",
				"sin_docker_kubernetes_cloud_secretos",
				"deploy_es_adaptador_no_core",
				"refs_plan_evidencia",
			},
			CanonicalRefs: []string{
				"modulos/orquesta-deploy/docs/contratos.md",
				"modulos/orquesta-deploy/composition_dry_run_v0.go",
				"docs/guia_nucleo_orquestacion_2026-05-17.md",
			},
			BacklogRefs:  mcpRoadmapBacklogRefsV0(mcpBacklogT74RefV0),
			Verification: mcpDeploymentPlanVerificationV0(),
		},
	}
}
