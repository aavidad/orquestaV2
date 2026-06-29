package orquestamcp

func mcpProjectRoadmapSourcesV0() []mcpProjectRoadmapItemSourceV0 {
	return append([]mcpProjectRoadmapItemSourceV0{
		{
			ID:          "CORE-ROADMAP-001",
			Area:        "registro_proyecto",
			Status:      "historico_compatibilidad_appspec_v0",
			Owner:       "orquesta-core",
			Focus:       "Compatibilidad de programacion desde AppSpec; el nucleo vigente se gobierna por workflow, domain_work y Director Operativo.",
			SummaryKey:  "mcp.project.roadmap.core.registro_proyecto.summary.v0",
			ProgressKey: "mcp.project.roadmap.core.registro_proyecto.historico_compatibilidad.v0",
			Contracts: []string{
				"RegistrarProyectoDesdeAppSpec v0",
				"ProyectoPlanBorradorV0",
			},
			Dependencies: []string{
				"SolicitarNuevaApp v0",
			},
			Guardrails: []string{
				"solo_app_spec_validada",
				"no_persiste_v0",
				"no_arranca_runtime",
				"idempotency_key_requerida",
			},
			CanonicalRefs: []string{
				"modulos/orquesta-core/docs/contratos.md",
				"modulos/orquesta-factory/docs/contratos.md",
			},
			BacklogRefs:  mcpRoadmapBacklogRefsV0(),
			Verification: mcpMCPResourceVerificationV0(),
		},
		{
			ID:          "CORE-ROADMAP-002",
			Area:        "microtareas",
			Status:      "vigente_workflow_task_store",
			Owner:       "orquesta-core-workflow",
			Focus:       "Usar WorkflowTaskV0 y WorkProfileV0 como metadata viva; FunctionContract queda como compatibilidad de launch.",
			SummaryKey:  "mcp.project.roadmap.core.microtareas.summary.v0",
			ProgressKey: "mcp.project.roadmap.core.microtareas.workflow_task_store_vigente.v0",
			Contracts: []string{
				"WorkflowTaskV0",
				"WorkProfileV0",
				"FunctionContract v0",
			},
			Guardrails: []string{
				"microtarea_pequena",
				"write_set_cerrado",
				"tests_obligatorios",
				"estado_ejecutable_explicito",
			},
			CanonicalRefs: []string{
				"modulos/orquesta-core-workflow/docs/contratos.md",
				"modulos/orquesta-orchestration-core/docs/contratos.md",
				"docs/estado_actual_2026-05-17.md",
			},
			BacklogRefs:  mcpRoadmapBacklogRefsV0(),
			Verification: mcpMCPResourceVerificationV0(),
		},
		{
			ID:          "CORE-ROADMAP-003",
			Area:        "persistencia_y_eventos",
			Status:      "vigente_puertos_y_adaptadores_file",
			Owner:       "orquesta-state-file",
			Focus:       "Persistencia real actual file-based por adaptadores; DB global no es canon del nucleo.",
			SummaryKey:  "mcp.project.roadmap.core.persistencia_eventos.summary.v0",
			ProgressKey: "mcp.project.roadmap.core.persistencia_eventos.adaptadores_vigentes.v0",
			Contracts: []string{
				"PersistenceRepository v0",
				"OrquestaEvent v0",
			},
			Dependencies: []string{
				"RegistrarProyectoDesdeAppSpec v0",
				"ProyectoPlanBorradorV0",
			},
			Guardrails: []string{
				"db_es_adaptador",
				"eventos_compactos",
				"mcp_lee_proyecciones_no_sink",
				"sin_reglas_negocio_en_adaptadores",
			},
			CanonicalRefs: []string{
				"modulos/orquesta-persistence/docs/contratos.md",
				"modulos/orquesta-observability/docs/contratos.md",
				"modulos/orquesta-state-file/docs/contratos.md",
				"docs/estado_actual_2026-05-17.md",
			},
			BacklogRefs:  mcpRoadmapBacklogRefsV0(),
			Verification: mcpMCPResourceVerificationV0(),
		},
		{
			ID:          "CORE-ROADMAP-004",
			Area:        "runtime_y_capacidad",
			Status:      "vigente_runtime_opt_in",
			Owner:       "orquesta-runtime",
			Focus:       "Runtime neutral y adaptadores concretos ejecutan por puertos opt-in; el nucleo no conoce proveedor.",
			SummaryKey:  "mcp.project.roadmap.core.runtime_capacidad.summary.v0",
			ProgressKey: "mcp.project.roadmap.core.runtime_capacidad.opt_in_vigente.v0",
			Contracts: []string{
				"CapacityDecision v0",
				"RuntimeLaunchRequest v0",
				"FunctionContract v0",
			},
			Guardrails: []string{
				"runtime_ejecuta_no_decide",
				"capacity_previa_requerida",
				"referencias_opacas",
				"sin_secretos_prompts_transcripts",
			},
			CanonicalRefs: []string{
				"modulos/orquesta-runtime/docs/contratos.md",
				"modulos/orquesta-capacity/docs/contratos.md",
				"docs/estado_actual_2026-05-17.md",
			},
			BacklogRefs:  mcpRoadmapBacklogRefsV0(),
			Verification: mcpMCPResourceVerificationV0(),
		},
		{
			ID:          "CORE-ROADMAP-005",
			Area:        "domain_work_consumidores_externos",
			Status:      "vigente_puerto_neutral_conectores_opt_in",
			Owner:       "orquesta-domain-work",
			Focus:       "Las apps externas consumen Orquesta por DomainWork, refs opacas y conectores de composicion opt-in; las evidencias de cada dominio viven fuera del contrato generico.",
			SummaryKey:  "mcp.project.roadmap.core.domain_work_consumidores.summary.v0",
			ProgressKey: "mcp.project.roadmap.core.domain_work_consumidores.puerto_neutral_vigente.v0",
			Contracts: []string{
				"DomainWorkJobRequestV0",
				"DomainWorkArtifactSubmissionV0",
				"DomainDocumentPlanV0",
			},
			Guardrails: []string{
				"dominio_externo_por_refs_opacas",
				"conectores_por_composicion_opt_in",
				"scope_por_job_type_o_job_ref",
				"sin_compartir_db_filesystem_app_externa",
			},
			CanonicalRefs: []string{
				"modulos/orquesta-domain-work/docs/contratos.md",
				"modulos/orquesta-document-plan-expander/docs/contratos.md",
			},
			BacklogRefs:  mcpRoadmapBacklogRefsV0(),
			Verification: mcpMCPResourceVerificationV0(),
		},
	}, mcpProjectRoadmapDeploymentSourcesV0()...)
}

func mcpProjectDecisionSourcesV0() []mcpProjectDecisionSourceV0 {
	return []mcpProjectDecisionSourceV0{
		{
			ID:          "CORE-DEC-001",
			Status:      "aceptada",
			Decision:    "Toda microtarea ejecutable exige FunctionContract v0 con write_set cerrado.",
			DecisionKey: "mcp.project.decisions.core.function_contract_write_set.v0",
			Motivo:      "Permite validar alcance, pruebas obligatorias y cierre antes de runtime.",
			MotivoKey:   "mcp.project.decisions.core.function_contract_write_set.motivo.v0",
			AppliesTo: []string{
				"FunctionContract v0",
				"RuntimeLaunchRequest v0",
			},
			Guardrails: []string{
				"write_set_es_superficie_autorizada",
				"dependencias_prohibidas_prevalecen",
				"evidencia_verificable_sin_transcripts",
			},
			CanonicalRefs: []string{
				"modulos/orquesta-core/docs/contratos.md",
			},
		},
		{
			ID:          "CORE-DEC-002",
			Status:      "aceptada",
			Decision:    "RegistrarProyectoDesdeAppSpec v0 devuelve borrador y no persiste ni arranca runtime.",
			DecisionKey: "mcp.project.decisions.core.registro_borrador_no_runtime.v0",
			Motivo:      "Core separa decision de dominio de adaptadores persistence, observability y runtime.",
			MotivoKey:   "mcp.project.decisions.core.registro_borrador_no_runtime.motivo.v0",
			AppliesTo: []string{
				"RegistrarProyectoDesdeAppSpec v0",
				"ProyectoPlanBorradorV0",
				"PersistenceRepository v0",
				"RuntimeLaunchRequest v0",
			},
			Guardrails: []string{
				"no_persiste_v0",
				"no_arranca_runtime",
				"idempotency_key_requerida",
			},
			CanonicalRefs: []string{
				"modulos/orquesta-core/docs/contratos.md",
			},
		},
		{
			ID:          "CORE-DEC-003",
			Status:      "aceptada",
			Decision:    "DB, runtime, filesystem, MCP, CLI, HTTP, Docker y proveedores entran al core solo como puertos o conectores.",
			DecisionKey: "mcp.project.decisions.core.hexagonal_adaptadores.v0",
			Motivo:      "Evita que el nucleo dependa de detalles operativos o proveedores concretos.",
			MotivoKey:   "mcp.project.decisions.core.hexagonal_adaptadores.motivo.v0",
			AppliesTo: []string{
				"FunctionContract v0",
				"PersistenceRepository v0",
				"RuntimeLaunchRequest v0",
				"DeploymentPlan v0",
			},
			Guardrails: []string{
				"hexagonal",
				"sin_detalles_operativos",
				"conectores_por_puerto",
			},
			CanonicalRefs: []string{
				"docs/guia_nucleo_orquestacion_2026-05-17.md",
			},
		},
		{
			ID:          "CORE-DEC-004",
			Status:      "aceptada",
			Decision:    "DB v1 es evidencia forense, no canon vivo ni fuente de tareas vivas.",
			DecisionKey: "mcp.project.decisions.core.dbv1_forense.v0",
			Motivo:      "El nucleo V2 no hereda reglas activas ni identificadores canonicos desde historico.",
			MotivoKey:   "mcp.project.decisions.core.dbv1_forense.motivo.v0",
			AppliesTo: []string{
				"GovernanceCatalog v0",
				"FunctionContract v0",
			},
			Guardrails: []string{
				"dbv1_evidencia_no_canon",
				"effective_requiere_decision",
				"ids_heredados_no_publicos",
			},
			CanonicalRefs: []string{
				"modulos/orquesta-governance/docs/contratos.md",
			},
		},
		{
			ID:          "CORE-DEC-005",
			Status:      "aceptada",
			Decision:    "MCP y web leen proyecciones compactas futuras, no adaptadores productivos ni eventos completos.",
			DecisionKey: "mcp.project.decisions.core.proyecciones_compactas.v0",
			Motivo:      "Una IA necesita estado accionable, no dumps ni superficies internas de observability.",
			MotivoKey:   "mcp.project.decisions.core.proyecciones_compactas.motivo.v0",
			AppliesTo: []string{
				"OrquestaEvent v0",
				"GovernanceCatalog v0",
				"orquesta.project.roadmap.v0",
			},
			Guardrails: []string{
				"proyecciones_compactas",
				"sin_transcripts_completos",
				"sin_secretos",
			},
			CanonicalRefs: []string{
				"modulos/orquesta-observability/docs/contratos.md",
				"modulos/orquesta-governance/docs/contratos.md",
			},
		},
	}
}
