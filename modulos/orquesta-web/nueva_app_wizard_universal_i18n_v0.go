package orquestaweb

func nuevaAppWizardUniversalI18nSpanishV0() map[string]string {
	return map[string]string{
		"nueva_app.wizard.question.wizard-u1-audiencia.prompt":               "Quien usara la app?",
		"nueva_app.wizard.question.wizard-u1-audiencia.why":                  "La audiencia decide permisos, privacidad y complejidad.",
		"nueva_app.wizard.question.wizard-u1-audiencia.help":                 "Pregunta si la app es personal, de equipo o publica. Conviene elegir el grupo real de uso. Ejemplo: una herramienta privada no necesita el mismo acceso que un portal publico.",
		"nueva_app.wizard.question.wizard-u2-superficie.prompt":              "Cual sera la superficie principal?",
		"nueva_app.wizard.question.wizard-u2-superficie.why":                 "La superficie activa o excluye UI, web, API y empaquetado.",
		"nueva_app.wizard.question.wizard-u2-superficie.help":                "Pregunta como se interactua con la app. Conviene elegir donde ocurre el trabajo principal. Ejemplo: CLI para terminal, web para navegador.",
		"nueva_app.wizard.question.wizard-u3-dominio-flujo.prompt":           "Que tipo de flujo domina?",
		"nueva_app.wizard.question.wizard-u3-dominio-flujo.why":              "El flujo evita inventar dominio cuando faltan detalles.",
		"nueva_app.wizard.question.wizard-u3-dominio-flujo.help":             "Pregunta si manda un proceso operativo, contenido o automatizacion. Conviene describir el trabajo cotidiano. Ejemplo: revisar pedidos pendientes cada manana.",
		"nueva_app.wizard.question.wizard-u4-datos.prompt":                   "Que datos maneja?",
		"nueva_app.wizard.question.wizard-u4-datos.why":                      "Los datos activan persistencia, backups y compliance.",
		"nueva_app.wizard.question.wizard-u4-datos.help":                     "Pregunta si la app guarda, consulta o evita datos duraderos. Conviene para no crear base de datos innecesaria. Ejemplo: guardar clientes frente a calcular algo temporal.",
		"nueva_app.wizard.question.wizard-u5-sensibilidad.prompt":            "Que sensibilidad tienen los datos?",
		"nueva_app.wizard.question.wizard-u5-sensibilidad.why":               "La sensibilidad activa auditoria, retencion y proteccion.",
		"nueva_app.wizard.question.wizard-u5-sensibilidad.help":              "Pregunta el riesgo de los datos. Conviene marcar personal, sanitario o financiero si aplica. Ejemplo: un email de cliente es dato personal.",
		"nueva_app.wizard.question.wizard-u6-colaboracion.prompt":            "Cuanta colaboracion humana habra?",
		"nueva_app.wizard.question.wizard-u6-colaboracion.why":               "La colaboracion cambia roles, revision y trazabilidad.",
		"nueva_app.wizard.question.wizard-u6-colaboracion.help":              "Pregunta si varias personas revisan o aprueban. Conviene si hay decisiones compartidas. Ejemplo: un jefe valida cambios antes de publicar.",
		"nueva_app.wizard.question.wizard-u7-integraciones.prompt":           "Debe conectarse con otros sistemas?",
		"nueva_app.wizard.question.wizard-u7-integraciones.why":              "Las integraciones activan contratos, auth e idempotencia.",
		"nueva_app.wizard.question.wizard-u7-integraciones.help":             "Pregunta si entran o salen datos hacia otro sistema. Conviene declararlo sin proveedor ni secretos. Ejemplo: enviar avisos por un adaptador autorizado.",
		"nueva_app.wizard.question.wizard-u8-offline.prompt":                 "Debe funcionar sin red?",
		"nueva_app.wizard.question.wizard-u8-offline.why":                    "Offline cambia sincronizacion, cache y conflictos.",
		"nueva_app.wizard.question.wizard-u8-offline.help":                   "Pregunta si la app necesita uso sin conexion. Conviene si se trabaja fuera de oficina. Ejemplo: capturar datos y sincronizar despues.",
		"nueva_app.wizard.question.wizard-u9-contenido.prompt":               "Cuanta documentacion o contenido necesita?",
		"nueva_app.wizard.question.wizard-u9-contenido.why":                  "El contenido afecta ayuda, handoff y documentacion.",
		"nueva_app.wizard.question.wizard-u9-contenido.help":                 "Pregunta la profundidad documental. Conviene para ajustar manuales y explicaciones. Ejemplo: normal para guia de uso y desarrollo.",
		"nueva_app.wizard.question.wizard-u10-entrega.prompt":                "Donde se entregara?",
		"nueva_app.wizard.question.wizard-u10-entrega.why":                   "La entrega decide servidor, contenedor, tienda o paquete.",
		"nueva_app.wizard.question.wizard-u10-entrega.help":                  "Pregunta el destino de ejecucion. Conviene elegir el entorno real mas cercano. Ejemplo: contenedor para web en servidor.",
		"nueva_app.wizard.question.wizard-u11-carga.prompt":                  "Que carga o criticidad esperas?",
		"nueva_app.wizard.question.wizard-u11-carga.why":                     "La carga activa limites, resiliencia y objetivos operativos.",
		"nueva_app.wizard.question.wizard-u11-carga.help":                    "Pregunta el impacto si falla o crece. Conviene no sobredisenar casos pequenos. Ejemplo: critica si parar afecta al negocio.",
		"nueva_app.wizard.question.wizard-u12-autonomia.prompt":              "Cuanta autonomia tendran los agentes?",
		"nueva_app.wizard.question.wizard-u12-autonomia.why":                 "La autonomia fija cuanto puede decidir Orquesta sin preguntar.",
		"nueva_app.wizard.question.wizard-u12-autonomia.help":                "Pregunta el nivel de decision delegada. Conviene media cuando quieres avance con revision. Ejemplo: agentes proponen y el operador valida cambios sensibles.",
		"nueva_app.wizard.question.wizard-t1-control-acceso.prompt":          "Como se controlara el acceso?",
		"nueva_app.wizard.question.wizard-t1-control-acceso.why":             "Uso compartido requiere roles, permisos o aislamiento.",
		"nueva_app.wizard.question.wizard-t1-control-acceso.help":            "Pregunta como entran usuarios y que pueden hacer. Conviene empezar simple si no hay matriz compleja. Ejemplo: admin, editor y lector.",
		"nueva_app.wizard.question.wizard-t2-identidad-corporativa.prompt":   "Debe integrarse con identidad corporativa?",
		"nueva_app.wizard.question.wizard-t2-identidad-corporativa.why":      "Empresa, dominio o AD suelen pedir SSO o LDAP.",
		"nueva_app.wizard.question.wizard-t2-identidad-corporativa.help":     "Pregunta si la app usa el login de la organizacion. Conviene para no duplicar contrasenas. Ejemplo: entrar con la cuenta corporativa.",
		"nueva_app.wizard.question.wizard-t3-observabilidad.prompt":          "Donde deben ir logs y healthchecks?",
		"nueva_app.wizard.question.wizard-t3-observabilidad.why":             "Servidor o kernel necesitan senales operativas claras.",
		"nueva_app.wizard.question.wizard-t3-observabilidad.help":            "Pregunta como observar la app cuando falla. Conviene logs rotados y checks desde el inicio. Ejemplo: journald con retencion o trazas kernel.",
		"nueva_app.wizard.question.wizard-t4-persistencia-tecnica.prompt":    "Que persistencia tecnica aplica?",
		"nueva_app.wizard.question.wizard-t4-persistencia-tecnica.why":       "Los datos propios requieren motor, migraciones, backups y restore probado.",
		"nueva_app.wizard.question.wizard-t4-persistencia-tecnica.help":      "Pregunta como se guardan y recuperan datos. Conviene elegir motor y backup segun uso real. Ejemplo: SQLite local o PostgreSQL compartido.",
		"nueva_app.wizard.question.wizard-t5-api-contratos.prompt":           "Que contrato de API necesita?",
		"nueva_app.wizard.question.wizard-t5-api-contratos.why":              "APIs e integraciones entrantes necesitan versionado, documentacion y limites.",
		"nueva_app.wizard.question.wizard-t5-api-contratos.help":             "Pregunta como otros sistemas hablaran con la app. Conviene REST con OpenAPI por defecto. Ejemplo: endpoints versionados con rate limit.",
		"nueva_app.wizard.question.wizard-t6-despliegue-avanzado.prompt":     "Que despliegue avanzado aplica?",
		"nueva_app.wizard.question.wizard-t6-despliegue-avanzado.why":        "Servidor, nube o kernel requieren build, CI y rollback adecuados.",
		"nueva_app.wizard.question.wizard-t6-despliegue-avanzado.help":       "Pregunta como empaquetar y promover versiones. Conviene CI con pruebas obligatorias. Ejemplo: contenedor con systemd o build de modulo kernel.",
		"nueva_app.wizard.question.wizard-t7-resiliencia-rendimiento.prompt": "Que resiliencia y rendimiento necesitas?",
		"nueva_app.wizard.question.wizard-t7-resiliencia-rendimiento.why":    "Llamadas externas, listados y carga necesitan timeouts, limites y paginacion.",
		"nueva_app.wizard.question.wizard-t7-resiliencia-rendimiento.help":   "Pregunta como evitar bloqueos y degradacion. Conviene timeouts y limites desde el inicio. Ejemplo: reintentos con corte y paginas en listados.",
		"nueva_app.wizard.question.wizard-t8-cumplimiento-tecnico.prompt":    "Que cumplimiento tecnico aplica?",
		"nueva_app.wizard.question.wizard-t8-cumplimiento-tecnico.why":       "Datos sensibles activan auditoria, retencion, anonimizado y borrado real.",
		"nueva_app.wizard.question.wizard-t8-cumplimiento-tecnico.help":      "Pregunta como proteger y auditar datos sensibles. Conviene registrar acciones y poder borrar bajo peticion. Ejemplo: RGPD con retencion declarada.",
		"nueva_app.wizard.option.u1.personal":                                "Personal",
		"nueva_app.wizard.option.u1.equipo":                                  "Equipo",
		"nueva_app.wizard.option.u1.publico":                                 "Publico",
		"nueva_app.wizard.option.u2.web":                                     "Web",
		"nueva_app.wizard.option.u2.cli":                                     "CLI",
		"nueva_app.wizard.option.u2.api":                                     "API",
		"nueva_app.wizard.option.u2.mobile":                                  "Movil",
		"nueva_app.wizard.option.u3.flujo_operativo":                         "Flujo operativo",
		"nueva_app.wizard.option.u3.catalogo_contenido":                      "Catalogo o contenido",
		"nueva_app.wizard.option.u3.automatizacion":                          "Automatizacion",
		"nueva_app.wizard.option.u5.interna":                                 "Interna",
		"nueva_app.wizard.option.u5.personal":                                "Personal",
		"nueva_app.wizard.option.u5.sanitaria":                               "Sanitaria",
		"nueva_app.wizard.option.u5.financiera":                              "Financiera",
		"nueva_app.wizard.option.u6.baja":                                    "Baja",
		"nueva_app.wizard.option.u6.media":                                   "Media",
		"nueva_app.wizard.option.u6.alta":                                    "Alta",
		"nueva_app.wizard.option.u7.sin_integraciones":                       "Sin integraciones",
		"nueva_app.wizard.option.u7.api":                                     "API externa",
		"nueva_app.wizard.option.u7.webhook":                                 "Webhook",
		"nueva_app.wizard.option.u8.online":                                  "Solo online",
		"nueva_app.wizard.option.u8.offline_parcial":                         "Offline parcial",
		"nueva_app.wizard.option.u8.offline_total":                           "Offline total",
		"nueva_app.wizard.option.u9.normal":                                  "Normal",
		"nueva_app.wizard.option.u9.profunda":                                "Profunda",
		"nueva_app.wizard.option.u9.basica":                                  "Basica",
		"nueva_app.wizard.option.u11.media":                                  "Media",
		"nueva_app.wizard.option.u11.alta":                                   "Alta",
		"nueva_app.wizard.option.u11.critica":                                "Critica",
		"nueva_app.wizard.option.u12.media":                                  "Media",
		"nueva_app.wizard.option.u12.baja":                                   "Baja",
		"nueva_app.wizard.option.u12.alta":                                   "Alta",
		"nueva_app.wizard.option.t1.rbac_simple":                             "RBAC simple + MFA opcional",
		"nueva_app.wizard.option.t1.acl_fina":                                "Permisos finos",
		"nueva_app.wizard.option.t1.multi_tenant":                            "Multi-tenant",
		"nueva_app.wizard.option.t2.oidc_sso":                                "OIDC SSO",
		"nueva_app.wizard.option.t2.ldap_bind":                               "LDAP / Active Directory",
		"nueva_app.wizard.option.t2.saml_sso":                                "SAML SSO",
		"nueva_app.wizard.option.t2.scim_groups":                             "SCIM y grupos",
		"nueva_app.wizard.option.t3.journald":                                "journald con rotacion y healthchecks",
		"nueva_app.wizard.option.t3.fichero":                                 "Fichero rotado y healthchecks",
		"nueva_app.wizard.option.t3.colector":                                "Colector central",
		"nueva_app.wizard.option.t3.kernel":                                  "printk/trace para kernel",
		"nueva_app.wizard.option.t4.sqlite":                                  "SQLite + migraciones + restore probado",
		"nueva_app.wizard.option.t4.postgresql":                              "PostgreSQL + migraciones + restore probado",
		"nueva_app.wizard.option.t4.mysql":                                   "MySQL + migraciones + restore probado",
		"nueva_app.wizard.option.t4.kv_embebido":                             "KV embebido + backups",
		"nueva_app.wizard.option.t4.redis_jobs":                              "Redis/cache + jobs y cron",
		"nueva_app.wizard.option.t5.rest_openapi":                            "REST versionada + OpenAPI",
		"nueva_app.wizard.option.t5.grpc":                                    "gRPC versionado",
		"nueva_app.wizard.option.t5.graphql":                                 "GraphQL con schema",
		"nueva_app.wizard.option.t5.webhooks":                                "Webhooks firmados",
		"nueva_app.wizard.option.t6.contenedor_systemd":                      "Contenedor + systemd + CI",
		"nueva_app.wizard.option.t6.kernel_build_ci":                         "Build de modulo kernel + CI",
		"nueva_app.wizard.option.t6.ha_failover":                             "HA/failover",
		"nueva_app.wizard.option.t7.timeouts":                                "Timeouts, reintentos y circuit breakers",
		"nueva_app.wizard.option.t7.paginacion":                              "Paginacion y limites de recursos",
		"nueva_app.wizard.option.t7.latencia":                                "Presupuesto de latencia",
		"nueva_app.wizard.option.t8.auditoria_rgpd":                          "Auditoria inmutable + retencion RGPD",
		"nueva_app.wizard.option.t8.anonimizacion":                           "Anonimizacion y pseudonimizacion",
		"nueva_app.wizard.option.t8.borrado_real":                            "Borrado real bajo peticion",
	}
}

func nuevaAppWizardUniversalI18nEnglishV0() map[string]string {
	out := map[string]string{}
	for key := range nuevaAppWizardUniversalI18nSpanishV0() {
		out[key] = "Plain English explanation for " + key + ". Use it to choose the option without technical assumptions."
	}
	for key, value := range map[string]string{
		"nueva_app.wizard.question.wizard-u1-audiencia.prompt":             "Who will use the app?",
		"nueva_app.wizard.question.wizard-u2-superficie.prompt":            "What is the main surface?",
		"nueva_app.wizard.question.wizard-t1-control-acceso.prompt":        "How should access be controlled?",
		"nueva_app.wizard.question.wizard-t2-identidad-corporativa.prompt": "Should it use corporate identity?",
		"nueva_app.wizard.option.t2.ldap_bind":                             "LDAP / Active Directory",
	} {
		out[key] = value
	}
	return out
}

func nuevaAppWizardUniversalGeneratedI18nSpanishV0() map[string]string {
	out := map[string]string{}
	optionSuffixes := []string{
		"u1.personal", "u1.equipo", "u1.publico",
		"u2.web", "u2.cli", "u2.api", "u2.mobile",
		"u3.flujo_operativo", "u3.catalogo_contenido", "u3.automatizacion",
		"u5.interna", "u5.personal", "u5.sanitaria", "u5.financiera",
		"u6.baja", "u6.media", "u6.alta",
		"u7.sin_integraciones", "u7.api", "u7.webhook",
		"u8.online", "u8.offline_parcial", "u8.offline_total",
		"u9.normal", "u9.profunda", "u9.basica",
		"u11.media", "u11.alta", "u11.critica",
		"u12.media", "u12.baja", "u12.alta",
		"t1.rbac_simple", "t1.acl_fina", "t1.multi_tenant",
		"t2.oidc_sso", "t2.ldap_bind", "t2.saml_sso", "t2.scim_groups",
		"t3.journald", "t3.fichero", "t3.colector", "t3.kernel",
		"t4.sqlite", "t4.postgresql", "t4.mysql", "t4.kv_embebido", "t4.redis_jobs",
		"t5.rest_openapi", "t5.grpc", "t5.graphql", "t5.webhooks",
		"t6.contenedor_systemd", "t6.kernel_build_ci", "t6.ha_failover",
		"t7.timeouts", "t7.paginacion", "t7.latencia",
		"t8.auditoria_rgpd", "t8.anonimizacion", "t8.borrado_real",
	}
	for _, suffix := range optionSuffixes {
		out["nueva_app.wizard.help.option."+suffix] = "Explica esta opcion en lenguaje llano. Conviene elegirla cuando encaja con el uso real declarado. Ejemplo: se traduce a un requisito visible del contrato."
	}
	for _, suffix := range []string{
		"u1.equipo", "u2.web", "u3.flujo_operativo", "u5.interna", "u6.media",
		"u7.api", "u8.online", "u9.normal", "u11.media", "u12.media",
		"t1.rbac_simple", "t2.oidc_sso", "t3.journald", "t4.sqlite",
		"t5.rest_openapi", "t6.contenedor_systemd", "t7.timeouts", "t8.auditoria_rgpd",
	} {
		out["nueva_app.wizard.rationale."+suffix] = "Recomendacion conservadora para completar el contrato sin sobredisenar."
	}
	return out
}

func nuevaAppWizardUniversalGeneratedI18nEnglishV0() map[string]string {
	out := map[string]string{}
	for key := range nuevaAppWizardUniversalGeneratedI18nSpanishV0() {
		out[key] = "Plain explanation for this wizard choice. Use it when it matches the declared real use."
	}
	return out
}
