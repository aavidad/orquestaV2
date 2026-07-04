package orquestaweb

import (
	"strconv"
	"strings"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

const wizardMaxQuestionsPerTurnV0 = 4

type wizardCrossGapRuleV0 struct {
	RuleRef string
	Build   func(WebNuevaAppFormV0) *WizardQuestionV0
}

var wizardCrossGapRulesV0 = []wizardCrossGapRuleV0{
	{RuleRef: "R1", Build: wizardRuleR1PersonalCompartidoV0},
	{RuleRef: "R2", Build: wizardRuleR2PlataformasV0},
	{RuleRef: "R4", Build: wizardRuleR4StorageV0},
	{RuleRef: "R5", Build: wizardRuleR5IntegracionGobiernoV0},
	{RuleRef: "R6", Build: wizardRuleR6MovilPlataformasV0},
	{RuleRef: "R7", Build: wizardRuleR7DeployCompatibleV0},
	{RuleRef: "R8", Build: wizardRuleR8UsuariosCompartidosV0},
}

func WebNuevaAppWizardGapQuestionsV0(form WebNuevaAppFormV0) []WizardQuestionV0 {
	return selectWizardTurnQuestionsV0(webNuevaAppWizardAllGapQuestionsV0(form))
}

func webNuevaAppWizardAllGapQuestionsV0(form WebNuevaAppFormV0) []WizardQuestionV0 {
	form = normalizeWizardFormForQuestionsV0(form)
	out := make([]WizardQuestionV0, 0, 16)
	out = append(out, wizardRequiredGapQuestionsV0(form)...)
	for _, rule := range wizardCrossGapRulesV0 {
		if question := rule.Build(form); question != nil {
			out = append(out, *question)
		}
	}
	out = append(out, wizardRuleR3IntegracionesDominioV0(form)...)
	out = append(out, wizardUniversalDimensionQuestionsV0(form)...)
	out = append(out, wizardTechnicalDimensionQuestionsV0(form)...)
	filtered, _ := wizardQuestionsAfterFactExclusionsV0(form, dedupeWizardQuestionsV0(out))
	return filtered
}

func webNuevaAppWizardAllGapQuestionsWithResolvedV0(form WebNuevaAppFormV0) ([]WizardQuestionV0, []WizardDecisionV0) {
	form = normalizeWizardFormForQuestionsV0(form)
	out := make([]WizardQuestionV0, 0, 24)
	out = append(out, wizardRequiredGapQuestionsV0(form)...)
	for _, rule := range wizardCrossGapRulesV0 {
		if question := rule.Build(form); question != nil {
			out = append(out, *question)
		}
	}
	out = append(out, wizardRuleR3IntegracionesDominioV0(form)...)
	out = append(out, wizardUniversalDimensionQuestionsV0(form)...)
	out = append(out, wizardTechnicalDimensionQuestionsV0(form)...)
	return wizardQuestionsAfterFactExclusionsV0(form, dedupeWizardQuestionsV0(out))
}

func wizardUniversalDimensionQuestionsV0(form WebNuevaAppFormV0) []WizardQuestionV0 {
	if trimV0(form.Objetivo) == "" {
		return nil
	}
	if wizardNeedIsKernelCV0(form) {
		return nil
	}
	questions := []WizardQuestionV0{
		wizardQuestionV0("wizard-u1-audiencia", "usuarios_objetivo", WizardTopicUsoV0, WizardImportanceMediaV0, []WizardOptionV0{
			wizardOptionV0("personal", "nueva_app.wizard.option.u1.personal", false, ""),
			wizardOptionV0("equipo", "nueva_app.wizard.option.u1.equipo", true, "nueva_app.wizard.rationale.u1.equipo"),
			wizardOptionV0("publico", "nueva_app.wizard.option.u1.publico", false, ""),
		}),
		wizardQuestionV0("wizard-u2-superficie", "tipo_app", WizardTopicUsoV0, WizardImportanceMediaV0, []WizardOptionV0{
			wizardOptionV0("web", "nueva_app.wizard.option.u2.web", true, "nueva_app.wizard.rationale.u2.web"),
			wizardOptionV0("cli", "nueva_app.wizard.option.u2.cli", false, ""),
			wizardOptionV0("api", "nueva_app.wizard.option.u2.api", false, ""),
			wizardOptionV0("mobile", "nueva_app.wizard.option.u2.mobile", false, ""),
		}),
		wizardQuestionV0("wizard-u3-dominio-flujo", "descripcion", WizardTopicUsoV0, WizardImportanceMediaV0, []WizardOptionV0{
			wizardOptionV0("flujo_operativo", "nueva_app.wizard.option.u3.flujo_operativo", true, "nueva_app.wizard.rationale.u3.flujo_operativo"),
			wizardOptionV0("catalogo_contenido", "nueva_app.wizard.option.u3.catalogo_contenido", false, ""),
			wizardOptionV0("automatizacion", "nueva_app.wizard.option.u3.automatizacion", false, ""),
		}),
		wizardQuestionV0("wizard-u4-datos", "datos.necesidad_funcional", WizardTopicDatosV0, WizardImportanceMediaV0, []WizardOptionV0{
			wizardOptionV0("gestion_con_persistencia", "nueva_app.wizard.option.datos.gestion", true, "nueva_app.wizard.rationale.datos.gestion"),
			wizardOptionV0("solo_consulta", "nueva_app.wizard.option.datos.consulta", false, ""),
			wizardOptionV0("sin_persistencia", "nueva_app.wizard.option.datos.sin_persistencia", false, ""),
		}),
		wizardQuestionV0("wizard-u5-sensibilidad", "datos.sensibilidad", WizardTopicDatosV0, WizardImportanceMediaV0, []WizardOptionV0{
			wizardOptionV0("interna", "nueva_app.wizard.option.u5.interna", true, "nueva_app.wizard.rationale.u5.interna"),
			wizardOptionV0("personal", "nueva_app.wizard.option.u5.personal", false, ""),
			wizardOptionV0("sanitaria", "nueva_app.wizard.option.u5.sanitaria", false, ""),
			wizardOptionV0("financiera", "nueva_app.wizard.option.u5.financiera", false, ""),
		}),
		wizardQuestionV0("wizard-u6-colaboracion", "agentes.autonomia", WizardTopicUsoV0, WizardImportanceMediaV0, []WizardOptionV0{
			wizardOptionV0("baja", "nueva_app.wizard.option.u6.baja", false, ""),
			wizardOptionV0("media", "nueva_app.wizard.option.u6.media", true, "nueva_app.wizard.rationale.u6.media"),
			wizardOptionV0("alta", "nueva_app.wizard.option.u6.alta", false, ""),
		}),
		wizardQuestionV0("wizard-u7-integraciones", "integraciones.0.tipo", WizardTopicDatosV0, WizardImportanceMediaV0, []WizardOptionV0{
			wizardOptionV0("sin_integraciones", "nueva_app.wizard.option.u7.sin_integraciones", false, ""),
			wizardOptionV0("api", "nueva_app.wizard.option.u7.api", true, "nueva_app.wizard.rationale.u7.api"),
			wizardOptionV0("webhook", "nueva_app.wizard.option.u7.webhook", false, ""),
		}),
		wizardQuestionV0("wizard-u8-offline", "restricciones", WizardTopicEntregaV0, WizardImportanceMediaV0, []WizardOptionV0{
			wizardOptionV0("online", "nueva_app.wizard.option.u8.online", true, "nueva_app.wizard.rationale.u8.online"),
			wizardOptionV0("offline_parcial", "nueva_app.wizard.option.u8.offline_parcial", false, ""),
			wizardOptionV0("offline_total", "nueva_app.wizard.option.u8.offline_total", false, ""),
		}),
		wizardQuestionV0("wizard-u9-contenido", "documentacion.profundidad", WizardTopicDatosV0, WizardImportanceMediaV0, []WizardOptionV0{
			wizardOptionV0("normal", "nueva_app.wizard.option.u9.normal", true, "nueva_app.wizard.rationale.u9.normal"),
			wizardOptionV0("profunda", "nueva_app.wizard.option.u9.profunda", false, ""),
			wizardOptionV0("basica", "nueva_app.wizard.option.u9.basica", false, ""),
		}),
		wizardQuestionV0("wizard-u10-entrega", "deploy.target", WizardTopicEntregaV0, WizardImportanceMediaV0, wizardDeployOptionsV0(form.TipoApp, wizardRecommendedDeployTargetV0(form.TipoApp))),
		wizardQuestionV0("wizard-u11-carga", "datos.operacion.criticidad", WizardTopicEntregaV0, WizardImportanceMediaV0, []WizardOptionV0{
			wizardOptionV0("media", "nueva_app.wizard.option.u11.media", true, "nueva_app.wizard.rationale.u11.media"),
			wizardOptionV0("alta", "nueva_app.wizard.option.u11.alta", false, ""),
			wizardOptionV0("critica", "nueva_app.wizard.option.u11.critica", false, ""),
		}),
		wizardQuestionV0("wizard-u12-autonomia", "agentes.autonomia", WizardTopicEntregaV0, WizardImportanceMediaV0, []WizardOptionV0{
			wizardOptionV0("media", "nueva_app.wizard.option.u12.media", true, "nueva_app.wizard.rationale.u12.media"),
			wizardOptionV0("baja", "nueva_app.wizard.option.u12.baja", false, ""),
			wizardOptionV0("alta", "nueva_app.wizard.option.u12.alta", false, ""),
		}),
	}
	return wizardUniversalQuestionsStillOpenV0(form, questions)
}

func wizardRequiredGapQuestionsV0(form WebNuevaAppFormV0) []WizardQuestionV0 {
	var out []WizardQuestionV0
	if trimV0(form.Locale) == "" {
		out = append(out, wizardQuestionV0(
			"wizard-q-locale",
			"locale",
			WizardTopicUsoV0,
			WizardImportanceAltaV0,
			[]WizardOptionV0{
				wizardOptionV0("es-ES", "nueva_app.wizard.option.locale.es", true, "nueva_app.wizard.rationale.locale.es"),
				wizardOptionV0("en-US", "nueva_app.wizard.option.locale.en", false, ""),
			},
		))
	}
	if trimV0(form.Nombre) == "" {
		out = append(out, wizardQuestionV0(
			"wizard-q-nombre",
			"nombre",
			WizardTopicUsoV0,
			WizardImportanceAltaV0,
			[]WizardOptionV0{
				wizardOptionV0(wizardNameFromObjectiveV0(form.Objetivo), "nueva_app.wizard.option.nombre.inferido", true, "nueva_app.wizard.rationale.nombre.inferido"),
				wizardOptionV0("Nueva app", "nueva_app.wizard.option.nombre.generico", false, ""),
			},
		))
	}
	if trimV0(form.Objetivo) == "" {
		out = append(out, wizardQuestionV0(
			"wizard-q-objetivo",
			"objetivo",
			WizardTopicUsoV0,
			WizardImportanceAltaV0,
			[]WizardOptionV0{
				wizardOptionV0("Describir objetivo verificable", "nueva_app.wizard.option.objetivo.describir", true, "nueva_app.wizard.rationale.objetivo.describir"),
			},
		))
	}
	if trimV0(form.TipoApp) == "" {
		recommended := wizardRecommendedTipoAppV0(form)
		question := wizardQuestionV0(
			"wizard-q-tipo-app",
			"tipo_app",
			WizardTopicUsoV0,
			WizardImportanceAltaV0,
			[]WizardOptionV0{
				wizardOptionV0("web", "nueva_app.wizard.option.tipo_app.web", recommended == "web", "nueva_app.wizard.rationale.tipo_app.web"),
				wizardOptionV0("api", "nueva_app.wizard.option.tipo_app.api", recommended == "api", "nueva_app.wizard.rationale.tipo_app.api"),
				wizardOptionV0("mobile", "nueva_app.wizard.option.tipo_app.mobile", recommended == "mobile", "nueva_app.wizard.rationale.tipo_app.mobile"),
				wizardOptionV0("desktop", "nueva_app.wizard.option.tipo_app.desktop", recommended == "desktop", "nueva_app.wizard.rationale.tipo_app.desktop"),
			},
		)
		question.ExcludedByFacts = []WizardFactV0{{Key: "ui", Value: "none"}}
		out = append(out, question)
	}
	if !webNuevaAppIntakeFieldCapturedV0(form, "datos") {
		out = append(out, wizardQuestionV0(
			"wizard-q-datos",
			"datos.necesidad_funcional",
			WizardTopicDatosV0,
			WizardImportanceAltaV0,
			[]WizardOptionV0{
				wizardOptionV0("gestion_con_persistencia", "nueva_app.wizard.option.datos.gestion", true, "nueva_app.wizard.rationale.datos.gestion"),
				wizardOptionV0("solo_consulta", "nueva_app.wizard.option.datos.consulta", false, ""),
				wizardOptionV0("sin_persistencia", "nueva_app.wizard.option.datos.sin_persistencia", false, ""),
			},
		))
	}
	if trimV0(form.Deploy.Target) == "" {
		recommended := wizardRecommendedDeployTargetV0(form.TipoApp)
		out = append(out, wizardQuestionV0(
			"wizard-q-deploy",
			"deploy.target",
			WizardTopicEntregaV0,
			WizardImportanceAltaV0,
			wizardDeployOptionsV0(form.TipoApp, recommended),
		))
	}
	return out
}

func wizardRuleR1PersonalCompartidoV0(form WebNuevaAppFormV0) *WizardQuestionV0 {
	need := normalizeGuidedNeedV0(form.Objetivo + " " + form.Descripcion)
	if len(compactStringsV0(form.UsuariosObjetivo)) > 0 || !wizardObjectiveHasShareAmbiguityV0(need) {
		return nil
	}
	question := wizardQuestionV0(
		"wizard-r1-uso-personal-compartido",
		"usuarios_objetivo",
		WizardTopicUsoV0,
		WizardImportanceAltaV0,
		[]WizardOptionV0{
			wizardOptionV0("compartir_auth_simple", "nueva_app.wizard.option.uso.compartir", true, "nueva_app.wizard.rationale.uso.compartir"),
			wizardOptionV0("personal", "nueva_app.wizard.option.uso.personal", false, ""),
		},
	)
	return &question
}

func wizardTechnicalDimensionQuestionsV0(form WebNuevaAppFormV0) []WizardQuestionV0 {
	facts := wizardFactsForFormV0(form)
	var out []WizardQuestionV0
	nextIntegrationIndex := wizardNextIntegrationIndexV0(form)
	if !wizardHasAccessDecisionV0(form) {
		q := wizardQuestionV0("wizard-t1-control-acceso", "integraciones."+strconv.Itoa(nextIntegrationIndex)+".tipo", WizardTopicEntregaV0, WizardImportanceAltaV0, []WizardOptionV0{
			wizardOptionV0("rbac_simple_mfa", "nueva_app.wizard.option.t1.rbac_simple", true, "nueva_app.wizard.rationale.t1.rbac_simple"),
			wizardOptionV0("acl_fina", "nueva_app.wizard.option.t1.acl_fina", false, ""),
			wizardOptionV0("multi_tenant", "nueva_app.wizard.option.t1.multi_tenant", false, ""),
		})
		q.RequiresFacts = []WizardFactV0{{Key: "audience", Value: "team"}}
		q.ExcludedByFacts = []WizardFactV0{{Key: "audience", Value: "single"}, {Key: "ui", Value: "none"}}
		out = append(out, q)
		if nextIntegrationIndex < nuevaAppMaxIntegrationRowsV0-1 {
			nextIntegrationIndex++
		}
	}
	if !wizardHasIdentityDecisionV0(form) {
		q := wizardQuestionV0("wizard-t2-identidad-corporativa", "integraciones."+strconv.Itoa(nextIntegrationIndex)+".tipo", WizardTopicEntregaV0, WizardImportanceAltaV0, []WizardOptionV0{
			wizardOptionV0("oidc_sso", "nueva_app.wizard.option.t2.oidc_sso", true, "nueva_app.wizard.rationale.t2.oidc_sso"),
			wizardOptionV0("ldap_bind", "nueva_app.wizard.option.t2.ldap_bind", false, ""),
			wizardOptionV0("saml_sso", "nueva_app.wizard.option.t2.saml_sso", false, ""),
			wizardOptionV0("scim_groups", "nueva_app.wizard.option.t2.scim_groups", false, ""),
		})
		q.RequiresFacts = []WizardFactV0{{Key: "corporate_identity", Value: "true"}}
		q.ExcludedByFacts = []WizardFactV0{{Key: "audience", Value: "single"}, {Key: "ui", Value: "none"}}
		out = append(out, q)
		if nextIntegrationIndex < nuevaAppMaxIntegrationRowsV0-1 {
			nextIntegrationIndex++
		}
	}
	if !wizardHasObservabilityDecisionV0(form) {
		q := wizardQuestionV0("wizard-t3-observabilidad", "calidad.observabilidad", WizardTopicEntregaV0, WizardImportanceAltaV0, []WizardOptionV0{
			wizardOptionV0("journald_rotado_healthchecks", "nueva_app.wizard.option.t3.journald", true, "nueva_app.wizard.rationale.t3.journald"),
			wizardOptionV0("fichero_rotado_healthchecks", "nueva_app.wizard.option.t3.fichero", false, ""),
			wizardOptionV0("colector_central", "nueva_app.wizard.option.t3.colector", false, "").withRequiresFactsV0([]WizardFactV0{{Key: "deploy", Value: "server"}}),
			wizardOptionV0("kernel_printk_trace", "nueva_app.wizard.option.t3.kernel", false, "").withRequiresFactsV0([]WizardFactV0{{Key: "runtime", Value: "kernel_c"}}),
		})
		q.RequiresFacts = []WizardFactV0{{Key: "observability_question", Value: "true"}}
		out = append(out, q)
	}
	if !wizardHasPersistenceDecisionV0(form) {
		q := wizardQuestionV0("wizard-t4-persistencia-tecnica", "datos.storage.0.tipo", WizardTopicDatosV0, WizardImportanceAltaV0, []WizardOptionV0{
			wizardOptionV0("sqlite_backups_restore", "nueva_app.wizard.option.t4.sqlite", true, "nueva_app.wizard.rationale.t4.sqlite"),
			wizardOptionV0("postgresql_migraciones_backups_restore", "nueva_app.wizard.option.t4.postgresql", false, ""),
			wizardOptionV0("mysql_migraciones_backups_restore", "nueva_app.wizard.option.t4.mysql", false, ""),
			wizardOptionV0("kv_embebido_backups_restore", "nueva_app.wizard.option.t4.kv_embebido", false, ""),
			wizardOptionV0("redis_cache_jobs_cron", "nueva_app.wizard.option.t4.redis_jobs", false, "").withRequiresFactsV0([]WizardFactV0{{Key: "deploy", Value: "server"}}),
		})
		q.RequiresFacts = []WizardFactV0{{Key: "data", Value: "own"}}
		out = append(out, q)
	}
	if !wizardHasAPIDecisionV0(form) {
		q := wizardQuestionV0("wizard-t5-api-contratos", "integraciones."+strconv.Itoa(nextIntegrationIndex)+".tipo", WizardTopicDatosV0, WizardImportanceAltaV0, []WizardOptionV0{
			wizardOptionV0("rest_versionada_openapi", "nueva_app.wizard.option.t5.rest_openapi", true, "nueva_app.wizard.rationale.t5.rest_openapi"),
			wizardOptionV0("grpc_contracts", "nueva_app.wizard.option.t5.grpc", false, ""),
			wizardOptionV0("graphql_schema", "nueva_app.wizard.option.t5.graphql", false, ""),
			wizardOptionV0("webhooks_firmados_reintentos", "nueva_app.wizard.option.t5.webhooks", false, ""),
		})
		q.RequiresFacts = []WizardFactV0{{Key: "api_contract", Value: "true"}}
		out = append(out, q)
		if nextIntegrationIndex < nuevaAppMaxIntegrationRowsV0-1 {
			nextIntegrationIndex++
		}
	}
	if !wizardHasDeployAdvancedDecisionV0(form) {
		q := wizardQuestionV0("wizard-t6-despliegue-avanzado", "deploy.restricciones", WizardTopicEntregaV0, WizardImportanceAltaV0, []WizardOptionV0{
			wizardOptionV0("contenedor_systemd_ci", "nueva_app.wizard.option.t6.contenedor_systemd", true, "nueva_app.wizard.rationale.t6.contenedor_systemd"),
			wizardOptionV0("kernel_build_ci", "nueva_app.wizard.option.t6.kernel_build_ci", false, "").withRequiresFactsV0([]WizardFactV0{{Key: "runtime", Value: "kernel_c"}}),
			wizardOptionV0("ha_failover", "nueva_app.wizard.option.t6.ha_failover", false, "").withRequiresFactsV0([]WizardFactV0{{Key: "deploy", Value: "server"}}),
		})
		q.RequiresFacts = []WizardFactV0{{Key: "deploy", Value: "server"}}
		out = append(out, q)
	}
	if !wizardHasResilienceDecisionV0(form) {
		q := wizardQuestionV0("wizard-t7-resiliencia-rendimiento", "agentes.preferencias", WizardTopicEntregaV0, WizardImportanceAltaV0, []WizardOptionV0{
			wizardOptionV0("timeouts_reintentos_circuit_breakers", "nueva_app.wizard.option.t7.timeouts", true, "nueva_app.wizard.rationale.t7.timeouts"),
			wizardOptionV0("paginacion_limites_recursos", "nueva_app.wizard.option.t7.paginacion", false, ""),
			wizardOptionV0("presupuesto_latencia", "nueva_app.wizard.option.t7.latencia", false, "").withRequiresFactsV0([]WizardFactV0{{Key: "latency_budget", Value: "true"}}),
		})
		q.RequiresFacts = []WizardFactV0{{Key: "resilience_question", Value: "true"}}
		out = append(out, q)
	}
	if !wizardHasComplianceDecisionV0(form) {
		q := wizardQuestionV0("wizard-t8-cumplimiento-tecnico", "agentes.preferencias", WizardTopicDatosV0, WizardImportanceAltaV0, []WizardOptionV0{
			wizardOptionV0("auditoria_inmutable_retencion_rgpd", "nueva_app.wizard.option.t8.auditoria_rgpd", true, "nueva_app.wizard.rationale.t8.auditoria_rgpd"),
			wizardOptionV0("anonimizacion_pseudonimizacion", "nueva_app.wizard.option.t8.anonimizacion", false, ""),
			wizardOptionV0("borrado_real_bajo_peticion", "nueva_app.wizard.option.t8.borrado_real", false, ""),
		})
		q.RequiresFacts = []WizardFactV0{{Key: "sensitive_data", Value: "true"}}
		out = append(out, q)
	}
	if len(out) == 0 && len(facts) == 0 {
		return []WizardQuestionV0{}
	}
	return out
}

func (option WizardOptionV0) withRequiresFactsV0(facts []WizardFactV0) WizardOptionV0 {
	option.RequiresFacts = facts
	return option
}

func wizardUniversalQuestionsStillOpenV0(form WebNuevaAppFormV0, questions []WizardQuestionV0) []WizardQuestionV0 {
	out := make([]WizardQuestionV0, 0, len(questions))
	for _, question := range questions {
		switch question.QuestionRef {
		case "wizard-u1-audiencia":
			if len(compactStringsV0(form.UsuariosObjetivo)) > 0 {
				continue
			}
		case "wizard-u2-superficie":
			if trimV0(form.TipoApp) != "" {
				continue
			}
		case "wizard-u3-dominio-flujo":
			if trimV0(form.Descripcion) != "" {
				continue
			}
		case "wizard-u4-datos":
			if webNuevaAppIntakeFieldCapturedV0(form, "datos") {
				continue
			}
		case "wizard-u5-sensibilidad":
			if trimV0(form.Datos.Sensibilidad) != "" {
				continue
			}
		case "wizard-u6-colaboracion":
			if trimV0(form.Agentes.Autonomia) != "" {
				continue
			}
		case "wizard-u7-integraciones":
			if len(form.Integraciones) > 0 && trimV0(form.Integraciones[0].Tipo) != "" {
				continue
			}
		case "wizard-u8-offline":
			if len(compactStringsV0(form.Restricciones)) > 0 {
				continue
			}
		case "wizard-u9-contenido":
			if trimV0(form.Documentacion.Profundidad) != "" {
				continue
			}
		case "wizard-u10-entrega":
			if trimV0(form.Deploy.Target) != "" {
				continue
			}
		case "wizard-u11-carga":
			if trimV0(form.Datos.Operacion.Criticidad) != "" {
				continue
			}
		case "wizard-u12-autonomia":
			if trimV0(form.Agentes.Autonomia) != "" {
				continue
			}
		}
		out = append(out, question)
	}
	return out
}

func wizardRuleR2PlataformasV0(form WebNuevaAppFormV0) *WizardQuestionV0 {
	if len(compactStringsV0(form.Plataformas)) > 0 || wizardTipoAppIsMobileLikeV0(form.TipoApp) {
		return nil
	}
	question := wizardQuestionV0(
		"wizard-r2-plataformas",
		"plataformas",
		WizardTopicUsoV0,
		WizardImportanceAltaV0,
		[]WizardOptionV0{
			wizardOptionV0("web_mobile", "nueva_app.wizard.option.plataformas.ambas", true, "nueva_app.wizard.rationale.plataformas.ambas"),
			wizardOptionV0("web", "nueva_app.wizard.option.plataformas.pc", false, ""),
			wizardOptionV0("mobile", "nueva_app.wizard.option.plataformas.movil", false, ""),
		},
	)
	question.ExcludedByFacts = []WizardFactV0{{Key: "ui", Value: "none"}}
	return &question
}

func wizardRuleR3IntegracionesDominioV0(form WebNuevaAppFormV0) []WizardQuestionV0 {
	need := normalizeGuidedNeedV0(form.Objetivo + " " + form.Descripcion)
	nextIntegrationIndex := wizardNextIntegrationIndexV0(form)
	var out []WizardQuestionV0
	for _, pack := range wizardDomainIntegrationPacksV0() {
		if !guidedContainsAnyV0(need, pack.Keywords...) || wizardHasIntegrationTypeV0(form, pack.IntegrationType) {
			continue
		}
		question := wizardQuestionV0(
			pack.QuestionRef,
			"integraciones."+strconv.Itoa(nextIntegrationIndex)+".tipo",
			WizardTopicDatosV0,
			WizardImportanceAltaV0,
			pack.Options,
		)
		out = append(out, question)
		nextIntegrationIndex++
		if nextIntegrationIndex >= nuevaAppMaxIntegrationRowsV0 {
			break
		}
	}
	if len(out) == 0 &&
		trimV0(form.Objetivo) != "" &&
		!wizardObjectiveMatchesKnownDomainPackV0(need) &&
		!wizardHasOpenDomainDescriptionV0(form) {
		out = append(out, wizardQuestionV0(
			"wizard-r3-dominio-abierto",
			"descripcion",
			WizardTopicDatosV0,
			WizardImportanceAltaV0,
			[]WizardOptionV0{
				wizardOptionV0("describir_flujo_diario", "nueva_app.wizard.option.dominio.flujo_diario", true, "nueva_app.wizard.rationale.dominio.flujo_diario"),
				wizardOptionV0("describir_integraciones", "nueva_app.wizard.option.dominio.integraciones", false, ""),
			},
		))
	}
	if out == nil {
		return []WizardQuestionV0{}
	}
	return out
}

func wizardRuleR4StorageV0(form WebNuevaAppFormV0) *WizardQuestionV0 {
	if !form.Datos.DBRequired || wizardHasStorageTypeV0(form) {
		return nil
	}
	recommended := "relacional"
	if len(form.Datos.Fuentes) > 0 || len(form.Integraciones) > 0 {
		recommended = "mixta"
	}
	question := wizardQuestionV0(
		"wizard-r4-storage-db",
		"datos.storage.0.tipo",
		WizardTopicDatosV0,
		WizardImportanceAltaV0,
		[]WizardOptionV0{
			wizardOptionV0("relacional", "nueva_app.wizard.option.storage.relacional", recommended == "relacional", "nueva_app.wizard.rationale.storage.relacional"),
			wizardOptionV0("documental", "nueva_app.wizard.option.storage.documental", recommended == "documental", "nueva_app.wizard.rationale.storage.documental"),
			wizardOptionV0("mixta", "nueva_app.wizard.option.storage.mixta", recommended == "mixta", "nueva_app.wizard.rationale.storage.mixta"),
			wizardOptionV0("sin_preferencia", "nueva_app.wizard.option.storage.sin_preferencia", recommended == "sin_preferencia", "nueva_app.wizard.rationale.storage.sin_preferencia"),
		},
	)
	return &question
}

func wizardRuleR5IntegracionGobiernoV0(form WebNuevaAppFormV0) *WizardQuestionV0 {
	for index, integration := range form.Integraciones {
		if trimV0(integration.Tipo) == "" && trimV0(integration.Nombre) == "" {
			continue
		}
		if trimV0(integration.Auth) != "" && trimV0(integration.Criticidad) != "" {
			continue
		}
		question := wizardQuestionV0(
			"wizard-r5-integracion-gobierno",
			"integraciones."+strconv.Itoa(index)+".auth",
			WizardTopicDatosV0,
			WizardImportanceAltaV0,
			[]WizardOptionV0{
				wizardOptionV0("auth_simple_criticidad_media", "nueva_app.wizard.option.integracion.gobierno.simple", true, "nueva_app.wizard.rationale.integracion.gobierno.simple"),
				wizardOptionV0("publica_criticidad_baja", "nueva_app.wizard.option.integracion.gobierno.publica", false, ""),
				wizardOptionV0("oauth_criticidad_alta", "nueva_app.wizard.option.integracion.gobierno.oauth_alta", false, ""),
			},
		)
		return &question
	}
	return nil
}

func wizardRuleR6MovilPlataformasV0(form WebNuevaAppFormV0) *WizardQuestionV0 {
	if !wizardTipoAppIsMobileLikeV0(form.TipoApp) || wizardHasConcreteMobilePlatformV0(form) {
		return nil
	}
	question := wizardQuestionV0(
		"wizard-r6-movil-plataformas",
		"plataformas",
		WizardTopicEntregaV0,
		WizardImportanceAltaV0,
		[]WizardOptionV0{
			wizardOptionV0("mobile_ios_android", "nueva_app.wizard.option.mobile.ios_android", true, "nueva_app.wizard.rationale.mobile.ios_android"),
			wizardOptionV0("mobile_ios", "nueva_app.wizard.option.mobile.ios", false, ""),
			wizardOptionV0("mobile_android", "nueva_app.wizard.option.mobile.android", false, ""),
			wizardOptionV0("web_mobile", "nueva_app.wizard.option.mobile.web_responsive", false, ""),
		},
	)
	return &question
}

func wizardRuleR7DeployCompatibleV0(form WebNuevaAppFormV0) *WizardQuestionV0 {
	target := trimV0(form.Deploy.Target)
	if target == "" || wizardDeployTargetCompatibleV0(form.TipoApp, target) {
		return nil
	}
	recommended := wizardRecommendedDeployTargetV0(form.TipoApp)
	question := wizardQuestionV0(
		"wizard-r7-deploy-compatible",
		"deploy.target",
		WizardTopicEntregaV0,
		WizardImportanceAltaV0,
		wizardDeployOptionsV0(form.TipoApp, recommended),
	)
	return &question
}

func wizardRuleR8UsuariosCompartidosV0(form WebNuevaAppFormV0) *WizardQuestionV0 {
	if len(compactStringsV0(form.UsuariosObjetivo)) > 0 || !wizardFormSharedUseV0(form) {
		return nil
	}
	question := wizardQuestionV0(
		"wizard-r8-usuarios-compartido",
		"usuarios_objetivo",
		WizardTopicEntregaV0,
		WizardImportanceAltaV0,
		[]WizardOptionV0{
			wizardOptionV0("equipo_pequeno", "nueva_app.wizard.option.usuarios.equipo", true, "nueva_app.wizard.rationale.usuarios.equipo"),
			wizardOptionV0("profesionales_internos", "nueva_app.wizard.option.usuarios.internos", false, ""),
			wizardOptionV0("clientes_invitados", "nueva_app.wizard.option.usuarios.clientes", false, ""),
		},
	)
	return &question
}

type wizardDomainIntegrationPackV0 struct {
	QuestionRef     string
	IntegrationType string
	Keywords        []string
	Options         []WizardOptionV0
}

func wizardDomainIntegrationPacksV0() []wizardDomainIntegrationPackV0 {
	return []wizardDomainIntegrationPackV0{
		{
			QuestionRef:     "wizard-r3-integracion-agenda",
			IntegrationType: "calendar",
			Keywords:        []string{"agenda", "calendario", "calendar", "cita", "citas"},
			Options: []WizardOptionV0{
				wizardOptionV0("calendar_enterprise_neutral", "nueva_app.wizard.option.integracion.calendar.enterprise", true, "nueva_app.wizard.rationale.integracion.calendar.enterprise"),
				wizardOptionV0("calendar_google_workspace", "nueva_app.wizard.option.integracion.calendar.google", false, ""),
				wizardOptionV0("calendar_microsoft_365", "nueva_app.wizard.option.integracion.calendar.microsoft", false, ""),
				wizardOptionV0("calendar_caldav", "nueva_app.wizard.option.integracion.calendar.caldav", false, ""),
			},
		},
		{
			QuestionRef:     "wizard-r3-integracion-tienda",
			IntegrationType: "ecommerce",
			Keywords:        []string{"tienda", "venta", "ventas", "ecommerce", "catalogo", "carrito", "pago", "pagos", "cobro", "checkout"},
			Options: []WizardOptionV0{
				wizardOptionV0("ecommerce_capability", "nueva_app.wizard.option.integracion.ecommerce.capability", true, "nueva_app.wizard.rationale.integracion.ecommerce.capability"),
				wizardOptionV0("ecommerce_payments_sync", "nueva_app.wizard.option.integracion.ecommerce.payments_sync", false, ""),
				wizardOptionV0("payments_deferred", "nueva_app.wizard.option.integracion.deferred", false, ""),
			},
		},
		{
			QuestionRef:     "wizard-r3-integracion-mapa",
			IntegrationType: "maps",
			Keywords:        []string{"mapa", "mapas", "ubicacion", "geo", "cerca"},
			Options: []WizardOptionV0{
				wizardOptionV0("maps_public_sources", "nueva_app.wizard.option.integracion.maps.public_sources", true, "nueva_app.wizard.rationale.integracion.maps.public_sources"),
				wizardOptionV0("maps_deferred", "nueva_app.wizard.option.integracion.deferred", false, ""),
			},
		},
		wizardDomainPackV0(
			"wizard-r3-integracion-inventario",
			"inventory",
			[]string{"inventario", "almacen", "almacenaje", "stock", "qr", "codigo de barras", "proveedores"},
		),
		wizardDomainPackV0(
			"wizard-r3-integracion-notas",
			"documents",
			[]string{"notas", "documentos", "wiki", "markdown", "plantillas", "adjuntos"},
		),
		wizardDomainPackV0(
			"wizard-r3-integracion-tareas",
			"project_tasks",
			[]string{"tareas", "proyectos", "kanban", "subtareas", "dependencias", "prioridades"},
		),
		wizardDomainPackV0(
			"wizard-r3-integracion-finanzas",
			"finance",
			[]string{"finanzas", "gastos", "presupuesto", "presupuestos", "extractos", "ofx", "ahorro"},
		),
		wizardDomainPackV0(
			"wizard-r3-integracion-crm",
			"crm",
			[]string{"contactos", "crm", "clientes", "pipeline", "seguimientos", "vcard"},
		),
		wizardDomainPackV0(
			"wizard-r3-integracion-reservas",
			"booking",
			[]string{"reservas", "turnos", "citas online", "disponibilidad", "lista de espera"},
		),
		wizardDomainPackV0(
			"wizard-r3-integracion-salud",
			"health",
			[]string{"salud", "fitness", "habitos", "metricas", "wearables", "racha"},
		),
		wizardDomainPackV0(
			"wizard-r3-integracion-educacion",
			"education",
			[]string{"educacion", "cursos", "lecciones", "alumnos", "ejercicios", "certificados"},
		),
		wizardDomainPackV0(
			"wizard-r3-integracion-comunidad",
			"community",
			[]string{"comunidad", "foro", "hilos", "votos", "moderacion", "reputacion"},
		),
		wizardDomainPackV0(
			"wizard-r3-integracion-iot",
			"iot",
			[]string{"domotica", "iot", "sensores", "mqtt", "dispositivos", "telemetria"},
		),
		wizardDomainPackV0(
			"wizard-r3-integracion-media",
			"media",
			[]string{"galeria", "media", "fotos", "imagenes", "albumes", "miniaturas", "transcodificacion"},
		),
		wizardDomainPackV0(
			"wizard-r3-integracion-facturacion",
			"billing",
			[]string{"facturacion", "facturas", "legal", "impuestos", "pdf", "numeracion"},
		),
	}
}

func wizardDomainPackV0(questionRef string, integrationType string, keywords []string) wizardDomainIntegrationPackV0 {
	return wizardDomainIntegrationPackV0{
		QuestionRef:     questionRef,
		IntegrationType: integrationType,
		Keywords:        keywords,
		Options: []WizardOptionV0{
			wizardOptionV0(integrationType+"_capability", "nueva_app.wizard.option.integracion.domain.capability", true, "nueva_app.wizard.rationale.integracion.domain.capability"),
			wizardOptionV0(integrationType+"_sync", "nueva_app.wizard.option.integracion.domain.sync", false, ""),
			wizardOptionV0(integrationType+"_deferred", "nueva_app.wizard.option.integracion.deferred", false, ""),
		},
	}
}

func wizardQuestionV0(ref, field, topic, importance string, options []WizardOptionV0) WizardQuestionV0 {
	return WizardQuestionV0{
		QuestionRef: ref,
		Field:       field,
		TopicGroup:  topic,
		Importance:  importance,
		PromptKey:   "nueva_app.wizard.question." + ref + ".prompt",
		WhyKey:      "nueva_app.wizard.question." + ref + ".why",
		HelpKey:     "nueva_app.wizard.question." + ref + ".help",
		Options:     normalizeWizardOptionsV0(options),
	}
}

func wizardOptionV0(value, labelKey string, recommended bool, rationaleKey string) WizardOptionV0 {
	if recommended && trimV0(rationaleKey) == "" {
		rationaleKey = "nueva_app.wizard.rationale.default"
	}
	return WizardOptionV0{
		Value:        trimV0(value),
		LabelKey:     trimV0(labelKey),
		HelpKey:      wizardOptionHelpKeyV0(labelKey),
		ExampleKey:   wizardOptionExampleKeyV0(labelKey),
		Recommended:  recommended,
		RationaleKey: trimV0(rationaleKey),
	}
}

func wizardOptionHelpKeyV0(labelKey string) string {
	return strings.Replace(trimV0(labelKey), "nueva_app.wizard.option.", "nueva_app.wizard.help.option.", 1)
}

func wizardOptionExampleKeyV0(labelKey string) string {
	key := strings.Replace(trimV0(labelKey), "nueva_app.wizard.option.", "nueva_app.wizard.example.option.", 1)
	if _, ok := nuevaAppWizardOptionExampleKeysV0()[key]; !ok {
		return ""
	}
	return key
}

func normalizeWizardOptionsV0(options []WizardOptionV0) []WizardOptionV0 {
	out := make([]WizardOptionV0, 0, len(options))
	recommended := false
	for _, option := range options {
		if trimV0(option.Value) == "" {
			continue
		}
		if option.Recommended {
			if recommended {
				option.Recommended = false
				option.RationaleKey = ""
			}
			recommended = true
		}
		out = append(out, option)
	}
	if !recommended && len(out) > 0 {
		out[0].Recommended = true
		if out[0].RationaleKey == "" {
			out[0].RationaleKey = "nueva_app.wizard.rationale.default"
		}
	}
	if out == nil {
		return []WizardOptionV0{}
	}
	return out
}

func selectWizardTurnQuestionsV0(questions []WizardQuestionV0) []WizardQuestionV0 {
	questions = dedupeWizardQuestionsV0(questions)
	for _, topic := range []string{WizardTopicUsoV0, WizardTopicDatosV0, WizardTopicEntregaV0} {
		var selected []WizardQuestionV0
		for _, question := range questions {
			if question.TopicGroup != topic {
				continue
			}
			selected = append(selected, question)
			if len(selected) == wizardMaxQuestionsPerTurnV0 {
				break
			}
		}
		if selected != nil {
			return selected
		}
	}
	return []WizardQuestionV0{}
}

func dedupeWizardQuestionsV0(questions []WizardQuestionV0) []WizardQuestionV0 {
	seenRef := map[string]bool{}
	out := make([]WizardQuestionV0, 0, len(questions))
	for _, question := range questions {
		if trimV0(question.QuestionRef) == "" || seenRef[question.QuestionRef] {
			continue
		}
		seenRef[question.QuestionRef] = true
		out = append(out, question)
	}
	if out == nil {
		return []WizardQuestionV0{}
	}
	return out
}

func normalizeWizardFormForQuestionsV0(form WebNuevaAppFormV0) WebNuevaAppFormV0 {
	form.Locale = trimV0(form.Locale)
	form.Nombre = trimV0(form.Nombre)
	form.Objetivo = trimV0(form.Objetivo)
	form.TipoApp = trimV0(form.TipoApp)
	return form
}

func wizardRecommendedTipoAppV0(form WebNuevaAppFormV0) string {
	need := normalizeGuidedNeedV0(form.Objetivo + " " + form.Descripcion)
	switch {
	case guidedContainsAnyV0(need, "api", "servicio", "backend"):
		return "api"
	case guidedContainsAnyV0(need, "movil", "mobile", "android", "ios", "nativo", "native"):
		return "mobile"
	case guidedContainsAnyV0(need, "desktop", "escritorio", "pc offline"):
		return "desktop"
	default:
		return "web"
	}
}

func wizardRecommendedDeployTargetV0(tipoApp string) string {
	switch trimV0(tipoApp) {
	case "mobile":
		return "mobile_store"
	case "desktop":
		return "desktop"
	case "api", "web", "mixed", "":
		return "contenedor"
	default:
		return "local"
	}
}

func wizardDeployOptionsV0(tipoApp, recommended string) []WizardOptionV0 {
	values := []string{"local", "contenedor", "paas"}
	switch trimV0(tipoApp) {
	case "mobile":
		values = []string{"mobile_store", "local", "contenedor"}
	case "desktop":
		values = []string{"desktop", "local", "contenedor"}
	case "api":
		values = []string{"contenedor", "paas", "serverless", "kubernetes"}
	}
	out := make([]WizardOptionV0, 0, len(values))
	for _, value := range values {
		out = append(out, wizardOptionV0(
			value,
			"nueva_app.wizard.option.deploy."+value,
			value == recommended,
			"nueva_app.wizard.rationale.deploy."+value,
		))
	}
	return out
}

func wizardDeployTargetCompatibleV0(tipoApp, target string) bool {
	switch trimV0(tipoApp) {
	case "mobile":
		return containsStringV0(target, "mobile_store", "local", "contenedor")
	case "desktop":
		return containsStringV0(target, "desktop", "local", "contenedor")
	case "web", "api", "mixed", "":
		return !containsStringV0(target, "mobile_store", "desktop")
	default:
		return true
	}
}

func wizardObjectiveHasShareAmbiguityV0(need string) bool {
	if guidedContainsAnyV0(need, "personal", "privado", "solo para mi", "individual") {
		return false
	}
	if guidedContainsAnyV0(need, "compartir", "compartida", "colaborativo", "equipo", "empresa") {
		return false
	}
	return guidedContainsAnyV0(need, "agenda", "calendario", "calendar", "tareas", "notas", "proyecto")
}

func wizardFormSharedUseV0(form WebNuevaAppFormV0) bool {
	need := normalizeGuidedNeedV0(form.Objetivo + " " + form.Descripcion + " " + strings.Join(form.Restricciones, " "))
	if guidedContainsAnyV0(need, "compartir", "compartida", "colaborativo", "equipo", "empresa", "clientes", "usuarios") {
		return true
	}
	for _, integration := range form.Integraciones {
		if trimV0(integration.Auth) != "" {
			return true
		}
	}
	return false
}

func wizardTipoAppIsMobileLikeV0(tipoApp string) bool {
	return containsStringV0(trimV0(tipoApp), "mobile", "mixed")
}

func wizardHasConcreteMobilePlatformV0(form WebNuevaAppFormV0) bool {
	joined := normalizeGuidedNeedV0(strings.Join(append(form.Plataformas, form.PreferenciasTecnicas.Preferencias...), " "))
	return guidedContainsAnyV0(joined, "ios", "android", "web responsive", "web_responsive", "mobile_ios", "mobile_android")
}

func wizardHasIntegrationTypeV0(form WebNuevaAppFormV0, integrationType string) bool {
	for _, integration := range form.Integraciones {
		if trimV0(integration.Tipo) == integrationType {
			return true
		}
	}
	return false
}

func wizardHasStorageTypeV0(form WebNuevaAppFormV0) bool {
	for _, storage := range form.Datos.Storage {
		if orquestafactory.DataStorageTypeSupportedV0(trimV0(storage.Tipo)) {
			return true
		}
	}
	return false
}

func wizardHasOpenDomainDescriptionV0(form WebNuevaAppFormV0) bool {
	description := normalizeGuidedNeedV0(form.Descripcion)
	return guidedContainsAnyV0(description, "usuario un dia normal", "flujo diario", "funcion principal", "capacidad principal")
}

func wizardObjectiveMatchesKnownDomainPackV0(need string) bool {
	for _, pack := range wizardDomainIntegrationPacksV0() {
		if guidedContainsAnyV0(need, pack.Keywords...) {
			return true
		}
	}
	return false
}

func wizardNextIntegrationIndexV0(form WebNuevaAppFormV0) int {
	for index, integration := range form.Integraciones {
		if trimV0(integration.Tipo) == "" && trimV0(integration.Nombre) == "" {
			return index
		}
	}
	if len(form.Integraciones) >= nuevaAppMaxIntegrationRowsV0 {
		return nuevaAppMaxIntegrationRowsV0 - 1
	}
	return len(form.Integraciones)
}

func wizardNameFromObjectiveV0(objective string) string {
	name := guidedTitleFromNeedV0(objective)
	if trimV0(name) == "" {
		return "Nueva app"
	}
	return name
}

func wizardQuestionRecommendedOptionV0(question WizardQuestionV0) (WizardOptionV0, bool) {
	for _, option := range question.Options {
		if option.Recommended {
			return option, true
		}
	}
	return WizardOptionV0{}, false
}

func wizardQuestionsAfterFactExclusionsV0(form WebNuevaAppFormV0, questions []WizardQuestionV0) ([]WizardQuestionV0, []WizardDecisionV0) {
	facts := wizardFactsForFormV0(form)
	out := make([]WizardQuestionV0, 0, len(questions))
	var resolved []WizardDecisionV0
	for _, question := range questions {
		if !wizardFactsMatchAllV0(facts, question.RequiresFacts) || wizardFactsMatchAnyV0(facts, question.ExcludedByFacts) {
			continue
		}
		originalOptionCount := len(question.Options)
		options := make([]WizardOptionV0, 0, len(question.Options))
		for _, option := range question.Options {
			if !wizardFactsMatchAllV0(facts, option.RequiresFacts) || wizardFactsMatchAnyV0(facts, option.ExcludedByFacts) {
				continue
			}
			options = append(options, option)
		}
		question.Options = normalizeWizardOptionsV0(options)
		if len(question.Options) == 0 {
			continue
		}
		if originalOptionCount > 1 && len(question.Options) == 1 {
			for _, decision := range wizardDecisionsForAnswerV0(question, question.Options[0].Value, false) {
				resolved = append(resolved, decision)
			}
			continue
		}
		out = append(out, question)
	}
	if out == nil {
		out = []WizardQuestionV0{}
	}
	if resolved == nil {
		resolved = []WizardDecisionV0{}
	}
	return out, resolved
}

func wizardFactsForFormV0(form WebNuevaAppFormV0) map[string]map[string]bool {
	facts := map[string]map[string]bool{}
	add := func(key, value string) {
		key = trimV0(key)
		value = trimV0(value)
		if key == "" || value == "" {
			return
		}
		if facts[key] == nil {
			facts[key] = map[string]bool{}
		}
		facts[key][value] = true
	}
	need := normalizeGuidedNeedV0(form.Objetivo + " " + form.Descripcion + " " + strings.Join(form.Restricciones, " "))
	if wizardNeedIsKernelCV0(form) {
		add("runtime", "kernel_c")
		add("ui", "none")
		add("web", "none")
		add("deploy", "server")
		add("observability_question", "true")
	}
	switch normalizeGuidedNeedV0(form.TipoApp) {
	case "cli":
		add("ui", "none")
	case "api":
		add("ui", "api")
	case "web", "mobile", "desktop":
		add("ui", "human")
	}
	if trimV0(form.Deploy.Target) != "" && !containsStringV0(trimV0(form.Deploy.Target), "local", "desktop", "mobile_store") {
		add("deploy", "server")
		add("observability_question", "true")
		add("resilience_question", "true")
	}
	if wizardAudienceLooksPersonalV0(form) {
		add("audience", "single")
	} else if wizardFormSharedUseV0(form) || len(compactStringsV0(form.UsuariosObjetivo)) > 0 {
		add("audience", "team")
	}
	if guidedContainsAnyV0(need, "empresa", "oficina", "dominio", "active directory", "samba ad", "ldap", "oidc", "saml", "kerberos", "windows") {
		add("corporate_identity", "true")
		add("audience", "team")
	}
	if form.Datos.DBRequired || trimV0(form.Datos.NecesidadFuncional) != "" {
		add("data", "own")
	}
	if len(form.Integraciones) > 0 || trimV0(form.TipoApp) == "api" {
		add("api_contract", "true")
		add("resilience_question", "true")
	}
	if guidedContainsAnyV0(need, "api", "webhook", "graphql", "grpc", "integracion entrante", "api publica", "contrato") {
		add("api_contract", "true")
		add("resilience_question", "true")
	}
	if guidedContainsAnyV0(need, "latencia", "tiempo real", "alto rendimiento", "muchos usuarios", "critica") {
		add("latency_budget", "true")
		add("resilience_question", "true")
	}
	if guidedContainsAnyV0(normalizeGuidedNeedV0(form.Datos.Sensibilidad), "personal", "sanitaria", "salud", "financiera") {
		add("sensitive_data", "true")
	}
	if len(form.Integraciones) > 0 {
		add("integrations", "present")
	}
	return facts
}

func wizardFactsMatchAllV0(facts map[string]map[string]bool, required []WizardFactV0) bool {
	for _, fact := range required {
		if !wizardFactMatchesV0(facts, fact) {
			return false
		}
	}
	return true
}

func wizardFactsMatchAnyV0(facts map[string]map[string]bool, excluded []WizardFactV0) bool {
	for _, fact := range excluded {
		if wizardFactMatchesV0(facts, fact) {
			return true
		}
	}
	return false
}

func wizardFactMatchesV0(facts map[string]map[string]bool, fact WizardFactV0) bool {
	key := trimV0(fact.Key)
	value := trimV0(fact.Value)
	if key == "" || value == "" {
		return false
	}
	return facts[key] != nil && facts[key][value]
}

func wizardNeedIsKernelCV0(form WebNuevaAppFormV0) bool {
	need := normalizeGuidedNeedV0(form.Objetivo + " " + form.Descripcion + " " + strings.Join(form.PreferenciasTecnicas.Preferencias, " "))
	return guidedContainsAnyV0(need, "nucleo de linux", "kernel linux", "linux kernel", "modulo para el nucleo") &&
		guidedContainsAnyV0(need, " c ", " en c", "lenguaje c", "kernel c")
}

func wizardAudienceLooksPersonalV0(form WebNuevaAppFormV0) bool {
	joined := normalizeGuidedNeedV0(strings.Join(append([]string{form.Objetivo, form.Descripcion}, form.UsuariosObjetivo...), " "))
	return guidedContainsAnyV0(joined, "personal", "solo para mi", "una sola persona", "uso privado", "individual")
}

func wizardHasAccessDecisionV0(form WebNuevaAppFormV0) bool {
	for _, integration := range form.Integraciones {
		if trimV0(integration.Tipo) == "auth" {
			return true
		}
	}
	return false
}

func wizardHasIdentityDecisionV0(form WebNuevaAppFormV0) bool {
	var integrationText []string
	for _, integration := range form.Integraciones {
		integrationText = append(integrationText, integration.Tipo, integration.Nombre, integration.Auth, integration.Proposito)
	}
	need := normalizeGuidedNeedV0(strings.Join(form.PreferenciasTecnicas.Preferencias, " ") + " " + strings.Join(integrationText, " "))
	return guidedContainsAnyV0(need, "oidc", "ldap", "saml", "kerberos", "sso", "active directory")
}

func wizardHasObservabilityDecisionV0(form WebNuevaAppFormV0) bool {
	return form.Calidad.Observabilidad != nil
}

func wizardHasDeployAdvancedDecisionV0(form WebNuevaAppFormV0) bool {
	return len(compactStringsV0(form.Deploy.Restricciones)) > 0
}

func wizardHasPersistenceDecisionV0(form WebNuevaAppFormV0) bool {
	return wizardHasStorageTypeV0(form)
}

func wizardHasAPIDecisionV0(form WebNuevaAppFormV0) bool {
	for _, integration := range form.Integraciones {
		if trimV0(integration.Tipo) == "api_contract" {
			return true
		}
	}
	return false
}

func wizardHasResilienceDecisionV0(form WebNuevaAppFormV0) bool {
	for _, value := range form.Agentes.Preferencias {
		if strings.HasPrefix(trimV0(value), "resiliencia: ") {
			return true
		}
	}
	return false
}

func wizardHasComplianceDecisionV0(form WebNuevaAppFormV0) bool {
	for _, value := range form.Agentes.Preferencias {
		if strings.HasPrefix(trimV0(value), "cumplimiento tecnico: ") {
			return true
		}
	}
	return false
}

func containsStringV0(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if value == candidate {
			return true
		}
	}
	return false
}
