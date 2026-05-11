package orquestamcp

func mcpProjectRoadmapSourcesV0() []mcpProjectRoadmapItemSourceV0 {
	return []mcpProjectRoadmapItemSourceV0{
		{
			ID:          "CORE-ROADMAP-001",
			Area:        "registro_proyecto",
			Status:      "compartido_v0",
			Owner:       "orquesta-core",
			Focus:       "Convertir AppSpecV0 validada y backlog en ProyectoPlanBorradorV0 gobernado.",
			SummaryKey:  "mcp.project.roadmap.core.registro_proyecto.summary.v0",
			ProgressKey: "mcp.project.roadmap.core.registro_proyecto.compartido_v0.v0",
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
				"../CONTRATOS.md#registrarproyectodesdeappspec-v0",
				"orquesta-core/docs/contratos.md",
			},
		},
		{
			ID:          "CORE-ROADMAP-002",
			Area:        "microtareas",
			Status:      "pendiente_schema_harness",
			Owner:       "orquesta-core",
			Focus:       "Cerrar schemas, fixtures y validador de FunctionContract v0 antes de ejecutar agentes.",
			SummaryKey:  "mcp.project.roadmap.core.microtareas.summary.v0",
			ProgressKey: "mcp.project.roadmap.core.microtareas.schema_harness_pendiente.v0",
			Contracts: []string{
				"FunctionContract v0",
			},
			Guardrails: []string{
				"microtarea_pequena",
				"write_set_cerrado",
				"tests_obligatorios",
				"estado_ejecutable_explicito",
			},
			CanonicalRefs: []string{
				"../CONTRATOS.md#functioncontract-v0",
				"orquesta-core/docs/contratos.md",
			},
		},
		{
			ID:          "CORE-ROADMAP-003",
			Area:        "persistencia_y_eventos",
			Status:      "pendiente_adaptadores",
			Owner:       "orquesta-core",
			Focus:       "Conectar persistencia y observability por puertos, con proyecciones compactas para MCP y web.",
			SummaryKey:  "mcp.project.roadmap.core.persistencia_eventos.summary.v0",
			ProgressKey: "mcp.project.roadmap.core.persistencia_eventos.adaptadores_pendientes.v0",
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
				"../CONTRATOS.md#persistencerepository-v0",
				"../CONTRATOS.md#orquestaevent-v0",
				"orquesta-core/docs/contratos.md",
			},
		},
		{
			ID:          "CORE-ROADMAP-004",
			Area:        "runtime_y_capacidad",
			Status:      "pendiente_harness",
			Owner:       "orquesta-core",
			Focus:       "Core entrega FunctionContract activo y CapacityDecision previa; runtime ejecuta sin decidir negocio.",
			SummaryKey:  "mcp.project.roadmap.core.runtime_capacidad.summary.v0",
			ProgressKey: "mcp.project.roadmap.core.runtime_capacidad.harness_pendiente.v0",
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
				"../CONTRATOS.md#capacitydecision-v0",
				"../CONTRATOS.md#runtimelaunchrequest-v0",
				"orquesta-core/docs/contratos.md",
			},
		},
		{
			ID:          "CORE-ROADMAP-005",
			Area:        "gobernanza_y_deploy",
			Status:      "pendiente_harness",
			Owner:       "orquesta-core",
			Focus:       "Consultar reglas efectivas y plan declarativo antes de cualquier adaptador operativo.",
			SummaryKey:  "mcp.project.roadmap.core.gobernanza_deploy.summary.v0",
			ProgressKey: "mcp.project.roadmap.core.gobernanza_deploy.harness_pendiente.v0",
			Contracts: []string{
				"GovernanceCatalog v0",
				"DeploymentPlan v0",
			},
			Guardrails: []string{
				"dbv1_forense_no_canon",
				"effective_requiere_decision",
				"deploy_plan_declarativo",
				"adaptador_real_requiere_smoke",
			},
			CanonicalRefs: []string{
				"../CONTRATOS.md#governancecatalog-v0",
				"../CONTRATOS.md#deploymentplan-v0",
				"orquesta-core/docs/contratos.md",
			},
		},
	}
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
				"../CONTRATOS.md#functioncontract-v0",
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
				"../CONTRATOS.md#registrarproyectodesdeappspec-v0",
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
				"../CONTRATOS.md#reglas-globales",
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
				"../CONTRATOS.md#governancecatalog-v0",
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
				"../CONTRATOS.md#orquestaevent-v0",
				"../CONTRATOS.md#governancecatalog-v0",
			},
		},
	}
}
