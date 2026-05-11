# Pruebas locales: orquesta-capacity

Registra pruebas obligatorias del modulo.

## Pruebas previstas para CAP-001

```text
Caso: CAP-CT-001 resolver capacidad sin proveedor hardcodeado
Tipo: contract
Comando: pendiente; validar fixture CapacityDecisionV0 con provider_kind remote/local/hybrid y model_ref opaco.
Evidencia esperada: La decision devuelve nivel_capacidad, pool, modelo, cuota y motivos sin nombres canonicos de proveedores historicos.
Ultima ejecucion: No ejecutada; contrato documental v0.
Riesgos: Requiere fixtures cuando exista schema JSON o tipos del modulo.
```

```text
Caso: CAP-CT-002 rechazar xhigh sin evidencia
Tipo: contract
Comando: pendiente; validar solicitud con nivel xhigh deseado y evidencia vacia.
Evidencia esperada: Error `evidencia_insuficiente_para_xhigh` o degradacion a high/medium con motivo auditable.
Ultima ejecucion: No ejecutada; contrato documental v0.
Riesgos: Requiere fixtures cuando ModelEscalationPolicyV0 tenga schema ejecutable.
```

```text
Caso: CAP-CT-003 cuota obsoleta fuerza degradacion o handoff
Tipo: contract
Comando: pendiente; validar QuotaSnapshotV0 con freshness obsolete y ventana insuficiente.
Evidencia esperada: No escala a high/xhigh; devuelve `cuota_obsoleta`, `ventana_insuficiente`, handoff preventivo o degradacion.
Ultima ejecucion: No ejecutada; contrato documental v0.
Riesgos: La fuente real de cuota dependera de adaptadores runtime/observability.
```

```text
Caso: CAP-CT-004 modelo local requiere score probado
Tipo: contract
Comando: pendiente; validar pool local con ModelEvidenceScoreV0 sin muestras ni benchmarks.
Evidencia esperada: Error `local_sin_score_suficiente` o alternativa remota/local probada; no se habilita local por deseo.
Ultima ejecucion: No ejecutada; contrato documental v0.
Riesgos: Falta decidir umbral minimo de confianza y muestras.
```

## Pruebas previstas para CAP-002

```text
Caso: CAP-CT-005 AgentHomeV0 separa agente logico, HOME, cuenta, runtime y credencial
Tipo: contract
Comando: pendiente; validar fixture con dos AgentHomeV0 para el mismo logical_agent_ref y provider_ref, con home_ref/account_ref/credential_ref distintos.
Evidencia esperada: La decision puede elegir HOME alternativo sin exponer email, token OAuth, ruta HOME real, nombre historico de agente ni proveedor canonico.
Ultima ejecucion: No ejecutada; contrato documental v0.
Riesgos: RuntimeLaunchRequest v0 aun debe confirmar el binding final de referencias opacas.
```

```text
Caso: CAP-CT-006 multi-HOME/OAuth respeta cuota y concurrencia por cuenta
Tipo: contract
Comando: pendiente; validar HOME premium/pro/local con quota fresh/stale/obsolete y max_concurrent_sessions agotado.
Evidencia esperada: HOME sin concurrencia o cuota agotada queda fuera; la decision usa alternativa viable, degrada o devuelve handoff preventivo sin inventar cuota real.
Ultima ejecucion: No ejecutada; contrato documental v0.
Riesgos: La fuente real de cuota y sesiones activas dependera de adaptadores runtime/observability.
```

```text
Caso: CAP-CT-007 local/remoto no implica preferencia automatica
Tipo: contract
Comando: pendiente; validar modelos local y remoto con locality, execution_mode, supported_efforts, score y quota_scope distintos.
Evidencia esperada: Local solo es preferente con score probado, confianza suficiente y runtime disponible; remoto requiere cuota y HOME/credencial viable.
Ultima ejecucion: No ejecutada; contrato documental v0.
Riesgos: Los umbrales numericos de score se definiran cuando existan benchmarks vivos.
```

## Pruebas previstas para CAP-003

```text
Caso: CAP-CT-008 escalado dinamico por fase y evidencia
Tipo: contract
Comando: pendiente; validar ModelEscalationPolicyV0 con fase arquitectura/revision/ejecucion y senales quality_signals/failure_signals frescas.
Evidencia esperada: La base por fase no selecciona modelo fijo; sube de medium a high si hay riesgo alto, calidad critica, revision fallida o fallo de pruebas.
Ultima ejecucion: No ejecutada; contrato documental v0.
Riesgos: Necesita fixtures de fases cuando core consuma CapacityDecision v0.
```

```text
Caso: CAP-CT-009 xhigh exige gate explicito
Tipo: contract
Comando: pendiente; validar fase arquitectura o brainstorming con y sin evidencia critica fresca.
Evidencia esperada: Sin gate devuelve `evidencia_insuficiente_para_xhigh` o degradacion; con riesgo critical mas evidencia justificada puede devolver xhigh y motivos auditables.
Ultima ejecucion: No ejecutada; contrato documental v0.
Riesgos: La aprobacion humana debe representarse como evidencia opaca, no transcript ni prompt.
```

```text
Caso: CAP-CT-010 cuota, HOME o modelo bloquean escalado preferido
Tipo: contract
Comando: pendiente; validar nivel preferido high/xhigh con QuotaSnapshotV0 obsolete, HOME sin concurrencia o modelo sin supported_efforts.
Evidencia esperada: La politica degrada, cambia HOME/pool/modelo o pide handoff preventivo; no ignora ventana insuficiente ni score local bajo.
Ultima ejecucion: No ejecutada; contrato documental v0.
Riesgos: Necesita schema para verificar orden de degradacion sin acoplarse a proveedor.
```

## Criterios de validacion para CAP-004..CAP-006

Estos casos documentan la validacion inicial de schemas y fixtures. La microtarea crea JSON Schema y fixtures, pero no anyade Go, harness ni validador ejecutable.

```text
Caso: CAP-CT-011 CapacityDecisionV0 schema cubre puerto completo
Tipo: contract
Comando: jq empty docs/schemas/capacity_decision_v0.schema.json docs/fixtures/capacity_decision_v0/*.json; npx --yes ajv-cli@5.0.0 validate con fixtures validos e invalidos separados.
Evidencia esperada: El schema exige task_ref, perfil_tarea, riesgo, calidad_requerida, cuota con source/freshness, decision_id, nivel_capacidad, reasoning_effort, pool/modelo opacos, handoff, motivos y degradacion sin proveedores hardcodeados.
Ultima ejecucion: 2026-05-04; sintaxis JSON validada con jq empty y AJV confirma validos/invalidos.
Riesgos: El envelope final de request/response puede ajustarse cuando core conecte CapacityDecision v0; queda pendiente harness Go contra JSON Schema.
```

```text
Caso: CAP-CT-012 CapacityDecisionV0 fixtures cubren gates y degradacion
Tipo: contract
Comando: jq empty docs/fixtures/capacity_decision_v0/*.json; npx --yes ajv-cli@5.0.0 validate con fixtures validos e invalidos separados.
Evidencia esperada: Existen casos para decision minima valida, xhigh con evidencia valida, xhigh sin evidencia invalida, cuota obsolete con degradacion/handoff y modelo local rechazado por score insuficiente.
Ultima ejecucion: 2026-05-04; sintaxis JSON validada con jq empty y AJV confirma validos/invalidos.
Riesgos: Los umbrales de score y ventana quedan documentales hasta tener benchmarks y adaptadores vivos.
```

```text
Caso: CAP-CT-013 AgentHomeV0 schema preserva opacidad y secretos
Tipo: contract
Comando: jq empty docs/schemas/agent_home_v0.schema.json docs/fixtures/agent_home_v0/*.json; npx --yes ajv-cli@5.0.0 validate con fixtures validos e invalidos separados.
Evidencia esperada: El schema exige refs opacas para logical_agent_ref, home_ref, pool_id, runtime_ref, provider_ref, account_ref y credential_ref opcional; rechaza rutas HOME reales, emails, tokens, tenants, nombres historicos y secretos.
Ultima ejecucion: 2026-05-04; sintaxis JSON validada con jq empty y AJV confirma validos/invalidos.
Riesgos: RuntimeLaunchRequest v0 debe confirmar que las mismas referencias bastan para el launch futuro; queda pendiente harness Go contra JSON Schema.
```

```text
Caso: CAP-CT-014 AgentHomeV0 fixtures cubren multi-HOME, cuota y concurrencia
Tipo: contract
Comando: jq empty docs/fixtures/agent_home_v0/*.json; npx --yes ajv-cli@5.0.0 validate con fixtures validos e invalidos separados.
Evidencia esperada: Existen casos para multi-HOME/OAuth valido, HOME local sin OAuth valido, quota_state limited/exhausted, cuota obsolete, concurrencia agotada y reserved_sessions mayor que max_concurrent_sessions como invalido.
Ultima ejecucion: 2026-05-04; sintaxis JSON validada con jq empty y AJV confirma validos/invalidos.
Riesgos: La disponibilidad real de sesiones y cuota dependera de adaptadores runtime/observability; la suma active_sessions + reserved_sessions requiere harness posterior.
```

```text
Caso: CAP-CT-015 ModelEscalationPolicyV0 schema exige reglas completas
Tipo: contract
Comando: jq empty docs/schemas/model_escalation_policy_v0.schema.json docs/fixtures/model_escalation_policy_v0/*.json; npx --yes ajv-cli@5.0.0 validate con fixtures validos e invalidos separados.
Evidencia esperada: El schema exige policy_ref versionado, phase_rules, risk_quality_rules, evidence_rules, quota_rules, locality_rules, xhigh_gate y degradation_order; cada senal declara source, freshness y peso cualitativo.
Ultima ejecucion: 2026-05-04; sintaxis JSON validada con jq empty y AJV confirma validos/invalidos.
Riesgos: La forma exacta de cada regla debe mantenerse portable y no acoplada a proveedor, rol o tabla historica; queda pendiente harness Go contra JSON Schema.
```

```text
Caso: CAP-CT-016 ModelEscalationPolicyV0 fixtures prueban escalado dinamico
Tipo: contract
Comando: jq empty docs/fixtures/model_escalation_policy_v0/*.json; npx --yes ajv-cli@5.0.0 validate con fixtures validos e invalidos separados.
Evidencia esperada: Existen casos para base por fase, subida a high con evidencia fresca, xhigh bloqueado sin gate, xhigh permitido con evidencia critica, degradacion por cuota/HOME/modelo y ausencia de tabla estatica rol -> modelo.
Ultima ejecucion: 2026-05-04; sintaxis JSON validada con jq empty y AJV confirma validos/invalidos.
Riesgos: La aprobacion humana debe ser evidencia opaca, no prompt, transcript ni comentario libre con secretos.
```

```text
Caso: CAP-DBV1-001 extraccion readonly de evidencia CAP-001
Tipo: smoke
Comando: sqlite3 'file:/home/alberto/Trabajo/orquesta/backups/legacy-sqlite-20260422/orquesta.db?mode=ro' con PRAGMA table_info y agregados de tablas capacity.
Evidencia esperada: Se leen esquemas/agregados de politicas_modelo, pools_capacidad, pool_modelos, agentes, presupuestos_sesion y agente_scores_locales sin modificar la DB.
Ultima ejecucion: 2026-05-04; ejecutada manualmente durante CAP-001.
Riesgos: La evidencia es forense; no se importa como configuracion viva.
```

## Pruebas ejecutables para CAP-007

```text
Caso: CAP-CT-017 CapacityDecisionV0 DTO/decoder acepta fixtures validos
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-capacity
Evidencia esperada: `DecodeCapacityDecisionV0` acepta decision_minima_valida, decision_xhigh_evidencia_valida y decision_cuota_obsoleta_handoff_valida con schema_version, pool y modelo opacos.
Ultima ejecucion: 2026-05-04; ok.
Riesgos: El validador no sustituye JSON Schema completo; mantiene las invariantes ejecutables criticas del contrato local.
```

```text
Caso: CAP-CT-018 CapacityDecisionV0 rechaza fixtures invalidos principales
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-capacity
Evidencia esperada: xhigh sin evidencia devuelve `evidencia_insuficiente_para_xhigh`; local sin score probado devuelve `local_sin_score_suficiente`; proveedor/modelo hardcodeado devuelve `referencia_no_opaca`.
Ultima ejecucion: 2026-05-04; ok.
Riesgos: La lista de marcas prohibidas es conservadora y local al contrato; no introduce proveedor canonico.
```

```text
Caso: CAP-CT-019 CapacityDecisionV0 fuerza degradacion o handoff por cuota/ventana
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-capacity
Evidencia esperada: cuota `obsolete` sin handoff/degradacion devuelve `cuota_obsoleta`; ventana restante menor que contexto estimado sin handoff/degradacion devuelve `handoff_requerido`.
Ultima ejecucion: 2026-05-04; ok.
Riesgos: No calcula cuota real ni selecciona alternativa; solo valida coherencia de una decision ya formada.
```

```text
Caso: CAP-CT-020 CapacityDecisionV0 decode estricto
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-capacity
Evidencia esperada: JSON con campo no contratado devuelve `capacity_decision_json_invalido`.
Ultima ejecucion: 2026-05-04; ok.
Riesgos: El decoder estricto aplica a DTOs Go; los schemas documentales siguen siendo fuente de referencia para fixtures JSON.
```

## Pruebas ejecutables para CAP-008

```text
Caso: CAP-CT-021 ModelEscalationPolicyV0 DTO/decoder acepta fixtures validos
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-capacity
Evidencia esperada: `DecodeModelEscalationPolicyV0` acepta politica_base_por_fase_valida, politica_high_por_evidencia_valida, politica_xhigh_gate_valida y politica_degradacion_quota_home_modelo_valida con policy_ref versionado y reglas obligatorias.
Ultima ejecucion: 2026-05-05; ok.
Riesgos: El validador ejecuta invariantes criticas del contrato local; el JSON Schema sigue siendo referencia documental completa.
```

```text
Caso: CAP-CT-022 ModelEscalationPolicyV0 rechaza fixtures invalidos principales
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-capacity
Evidencia esperada: xhigh sin gate devuelve `evidencia_insuficiente_para_xhigh`; tabla estatica rol -> modelo devuelve `tabla_rol_modelo_detectada`.
Ultima ejecucion: 2026-05-05; ok.
Riesgos: La deteccion de tabla rol/modelo es conservadora y se limita al DTO de politica; no crea selector real.
```

```text
Caso: CAP-CT-023 ModelEscalationPolicyV0 exige gate, refs opacas y decode estricto
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-capacity
Evidencia esperada: JSON con campo no contratado devuelve `model_escalation_policy_json_invalido`; policy_ref con proveedor hardcodeado devuelve `referencia_no_opaca`; xhigh sin evidencia fresca devuelve `evidencia_insuficiente_para_xhigh`; texto con HOME real se rechaza.
Ultima ejecucion: 2026-05-05; ok.
Riesgos: No valida disponibilidad real de runtime, cuota ni OAuth; solo coherencia declarativa de la politica.
```

## Plantilla

```text
Caso:
Tipo: unit | contract | integration | smoke
Comando:
Evidencia esperada:
Ultima ejecucion:
Riesgos:
```
