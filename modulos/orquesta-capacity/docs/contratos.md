# Contratos locales: orquesta-capacity

Registra puertos, DTOs y eventos que `orquesta-capacity` expone o consume.

## CapacityDecisionV0

```text
Nombre: CapacityDecisionV0
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-capacity
Consumidores: orquesta-core; orquesta-runtime cuando exista RuntimeLaunchRequest v0; adaptadores de observabilidad solo como lectores de eventos.
Entrada:
  - task_ref: identificador opaco de tarea o fase; no contiene SQL ni ruta interna obligatoria.
  - perfil_tarea: analisis | implementacion | revision | documentacion | script | handoff | orquestacion | otro.
  - fase: identificador funcional opcional, por ejemplo discovery, arquitectura, ejecucion, revision o cierre.
  - riesgo: low | medium | high | critical.
  - calidad_requerida: normal | alta | critica.
  - contexto_estimado: tokens/mensajes/segundos estimados cuando exista; admite unknown.
  - restricciones: local_only, remote_allowed, paid_allowed, max_coste_relativo, requiere_handoff_preventivo, requiere_privacidad_alta.
  - evidencia: quality_signals, benchmark_signals, telemetry_signals, quota_signals, failure_signals; cada senal declara source y freshness.
Salida correcta:
  - decision_id: idempotente para la evaluacion.
  - nivel_capacidad: low | medium | high | xhigh.
  - reasoning_effort: low | medium | high | xhigh | none.
  - pool: PoolCapacidadV0 recomendado.
  - modelo: identificador opaco o familia abstracta; no es contrato de proveedor.
  - home: AgentHomeV0 opcional si runtime necesita cuenta/HOME concreta.
  - cuota: QuotaSnapshotV0 usado en la decision.
  - politica_escalado: policy_ref de ModelEscalationPolicyV0 usada para justificar nivel_capacidad.
  - handoff: none | preventivo | requerido | bloqueado.
  - motivos: lista corta de razones auditables sin prompts ni secretos.
  - alternativas: otros pools/modelos viables ordenados por prioridad, coste y disponibilidad.
  - degradacion: opcion recomendada si cuota, coste o calidad impiden la preferida.
Invariantes:
  - No existe tabla estatica rol -> modelo.
  - La seleccion se calcula por perfil, fase, riesgo, calidad, coste, cuota y evidencia.
  - El contrato admite proveedores remotos y locales; el proveedor concreto es dato de pool, no tipo del contrato.
  - `xhigh` es excepcional: requiere evidencia explicita de riesgo alto/critico, fallo de calidad, revision fallida, arquitectura/brainstorming critico o solicitud humana justificada.
  - Los modelos locales se habilitan por pruebas, score y confianza; no por deseo ni por estar instalados.
  - La cuota distingue real, estimada y obsoleta mediante QuotaSnapshotV0.
  - Si la ventana disponible no alcanza, la decision debe pedir handoff preventivo o degradar; no puede ignorar la senal.
  - No se copian nombres historicos de agentes, cuentas, HOME, modelos o proveedores como seed canonico.
Errores:
  - perfil_tarea_no_soportado
  - evidencia_insuficiente_para_xhigh
  - cuota_no_disponible
  - cuota_obsoleta
  - pool_no_disponible
  - modelo_no_habilitado
  - proveedor_restringido
  - local_sin_score_suficiente
  - concurrencia_home_agotada
  - politica_escalado_incompleta
  - handoff_requerido
Pruebas de contrato:
  - CAP-CT-001 resuelve perfil/fase sin proveedor hardcodeado.
  - CAP-CT-002 rechaza xhigh sin evidencia.
  - CAP-CT-003 marca cuota obsoleta y fuerza degradacion/handoff.
  - CAP-CT-004 permite pool local solo con score y confianza suficientes.
  - CAP-CT-005/CAP-CT-006 validan AgentHomeV0 multi-HOME/OAuth, cuota y concurrencia.
  - CAP-CT-007 valida local/remoto sin preferencia automatica.
  - CAP-CT-008/CAP-CT-009/CAP-CT-010 validan escalado dinamico y degradacion.
Implementacion local CAP-007:
  - DTOs Go exportados en `capacity_decision_v0.go` para el envelope, request, response, PoolCapacidadV0, ModelCapacityRefV0, AgentHomeSummaryV0, QuotaSnapshotV0, ModelEvidenceScoreV0, alternativas, degradacion y evidencia.
  - `DecodeCapacityDecisionV0` usa JSON estricto (`DisallowUnknownFields`) y no llama a DB, filesystem, runtime ni proveedores.
  - `ValidateCapacityDecisionV0` se divide en `capacity_decision_validator_v0.go`, `capacity_decision_components_v0.go`, `capacity_decision_invariants_v0.go` y `capacity_decision_validator_helpers_v0.go` para evitar un fichero monolitico.
  - El validador es puro y valida referencias opacas, enums, textos seguros, xhigh con gate fresco, cuota obsolete/unavailable con degradacion/handoff, ventana insuficiente, score local probado y modelo seleccionado dentro del pool.
  - No implementa seleccion real de modelos, cuotas reales, proveedores, tablas rol->modelo, Ollama/vLLM/Codex ni HOME reales.
```

## PoolCapacidadV0

```text
Nombre: PoolCapacidadV0
Tipo: dto
Version: v0
Propietario: orquesta-capacity
Consumidores: CapacityDecisionV0; RuntimeLaunchRequest v0 futuro.
Campos:
  - pool_id: identificador opaco estable dentro del modulo.
  - provider_kind: remote | local | hybrid.
  - runtime_kind: cli | api | local_runtime | browser | mobile_runtime | otro.
  - plan_kind: free | paid | enterprise | local | unknown.
  - paid: boolean.
  - active: boolean.
  - total_slots: entero >= 1.
  - reserved_slots: entero >= 0 y <= total_slots.
  - allows_child_agents: boolean.
  - allows_multi_model: boolean.
  - allows_overcost: boolean.
  - handoff_policy: preventivo | estricto | manual | none.
  - telemetry_source: real | estimated | manual | unavailable.
  - credential_modes: lista de oauth | api_key | browser_session | local_runtime | none | unknown admitidos por el pool.
  - quota_scope: pool | account | home | model | provider | unknown.
  - modelos: lista de ModelCapacityRefV0.
Invariantes:
  - `provider_kind` no enumera marcas; las marcas viven en adaptadores/configuracion, no en contratos canonicos.
  - Un pool local no implica disponibilidad: requiere pruebas de modelo y score suficiente.
  - Un pool remoto no implica cuota real: debe aportar QuotaSnapshotV0 o declararse unavailable/estimated.
  - La concurrencia disponible es `total_slots - reserved_slots`, nunca negativa.
  - `credential_modes` declara capacidades de binding, no secretos ni nombres de proveedor.
  - La cuota del pool no sustituye la cuota de cuenta/HOME cuando el adaptador pueda observarla por separado.
Errores:
  - pool_inactivo
  - capacidad_agotada
  - telemetria_no_disponible
```

## ModelCapacityRefV0

```text
Nombre: ModelCapacityRefV0
Tipo: dto
Version: v0
Propietario: orquesta-capacity
Campos:
  - model_ref: identificador opaco o alias interno.
  - enabled: boolean.
  - priority: entero; menor valor significa preferencia mayor.
  - relative_cost: numero positivo.
  - locality: remote | local | unknown.
  - known_limits: estructura opcional con mensajes, tokens, segundos, creditos o ventana.
  - execution_mode: remote_api | remote_cli | local_process | local_service | browser | unknown.
  - enablement_source: benchmark | observed_score | manual_candidate | adapter_declared | unknown.
  - supported_efforts: lista opcional de none | low | medium | high | xhigh.
  - score: ModelEvidenceScoreV0 opcional.
Invariantes:
  - `relative_cost` sirve para comparar dentro de politica, no como precio real.
  - `known_limits` puede faltar; si falta, la decision debe tratar cuota como unknown/estimated.
  - `locality=local` requiere score probado, confianza suficiente y runtime disponible antes de ser preferente.
  - `locality=remote` requiere cuota y credencial/HOME viables; remoto no significa ilimitado ni de mas calidad.
  - `enabled=true` habilita evaluacion, no seleccion obligatoria.
  - Si `supported_efforts` falta, la politica no puede asumir `high` o `xhigh` sin evidencia adicional del adaptador.
```

## AgentHomeV0

```text
Nombre: AgentHomeV0
Tipo: dto
Version: v0 preliminar
Propietario: orquesta-capacity; coordinacion requerida con orquesta-runtime.
Consumidores: RuntimeLaunchRequest v0 futuro.
Campos:
  - logical_agent_ref: identificador opaco de agente logico; no es nombre historico ni rol fijo.
  - home_ref: identificador opaco de HOME/perfil runtime; nunca ruta real de filesystem.
  - pool_id: PoolCapacidadV0 asociado.
  - runtime_ref: identificador opaco de runtime/conector/perfil de ejecucion.
  - provider_ref: identificador opaco de proveedor o familia; no contiene marca canonica obligatoria.
  - account_ref: identificador opaco de cuenta/sujeto; no contiene email, usuario, tenant ni organizacion real.
  - credential_ref: identificador opaco opcional de credencial o grant OAuth.
  - credential_kind: oauth | api_key | browser_session | local_runtime | none | unknown.
  - account_kind: personal | service | pro | premium | enterprise | local | ephemeral | unknown.
  - entitlement_kind: free | pro | premium | enterprise | local | unknown.
  - enabled: boolean.
  - paused_until: timestamp opcional.
  - cooldown_until: timestamp opcional cuando un backoff temporal impide reuso.
  - quota_state: active | limited | exhausted | paused | unknown.
  - quota: QuotaSnapshotV0 opcional observado para esta cuenta/HOME.
  - active_sessions: entero >= 0.
  - reserved_sessions: entero >= 0.
  - max_concurrent_sessions: entero >= 1.
  - daily_limit_seconds: opcional; no canonico si procede de DB historica.
  - weekly_limit_seconds: opcional; no canonico si procede de DB historica.
  - usage_observed: segundos/mensajes/tokens/creditos observados.
  - last_checked_at: timestamp opcional de la ultima observacion operativa.
Invariantes:
  - AgentHomeV0 es un binding operativo, no la identidad del agente ni un rol.
  - Multi-HOME significa varias entradas AgentHomeV0 para el mismo `logical_agent_ref`, `pool_id` o `provider_ref`, cada una con `home_ref`, `account_ref` o `credential_ref` propios.
  - Multi-OAuth se modela como credenciales opacas diferentes; no se exponen access tokens, refresh tokens, scopes sensibles, emails ni tenant IDs.
  - La decision no expone secretos, rutas HOME reales, tokens OAuth, nombres de cuenta ni nombres historicos de agente.
  - La concurrencia disponible de un HOME es `max_concurrent_sessions - active_sessions - reserved_sessions`, nunca negativa.
  - La cuota de HOME/cuenta prevalece sobre una cuota estimada de pool cuando ambas existan y sean frescas.
  - Un HOME `premium`, `pro` o `enterprise` no convierte su cuota en real; siempre requiere QuotaSnapshotV0 con source y freshness.
  - Un HOME `local` puede evitar OAuth, pero sigue necesitando runtime local disponible y modelos locales habilitados por score.
  - Varios HOME del mismo proveedor pueden competir como alternativas; capacity debe ordenar por disponibilidad, cuota, coste relativo y evidencia.
  - Pausa y reanimacion son estado operativo, no identidad del agente.
  - Si un HOME esta pausado, en cooldown, agotado o sin concurrencia, la decision debe elegir alternativa, degradar o pedir handoff.
Errores:
  - home_no_habilitado
  - home_pausado
  - cuota_home_agotada
  - quota_home_obsoleta
  - oauth_no_disponible
  - credential_ref_requerido
  - concurrencia_home_agotada
  - provider_home_incompatible
```

## QuotaSnapshotV0

```text
Nombre: QuotaSnapshotV0
Tipo: dto
Version: v0
Propietario: orquesta-capacity
Campos:
  - source: real | estimated | manual | unavailable.
  - freshness: fresh | stale | obsolete | unknown.
  - checked_at: timestamp opcional.
  - scope: pool | account | home | model | provider | task | unknown.
  - window_kind: primary | short | daily | weekly | provider | unknown.
  - reset_at: timestamp opcional.
  - remaining_seconds: entero opcional.
  - remaining_messages: entero opcional.
  - remaining_tokens: entero opcional.
  - remaining_credits: numero opcional.
  - confidence: numero opcional entre 0 y 1 para estimaciones o fuentes manuales.
  - raw_ref: identificador opaco opcional para auditoria; nunca snapshot crudo con secretos.
Invariantes:
  - `real` requiere telemetria observada por adaptador confiable.
  - `estimated` puede decidir, pero debe quedar reflejado en motivos y degradacion.
  - `manual` es una declaracion operativa, no prueba de cuota real.
  - `unavailable` obliga a alternativa, degradacion o handoff cuando la tarea requiere ventana garantizada.
  - `obsolete` no permite escalar a high/xhigh salvo decision humana o evidencia independiente.
  - Valores ausentes no se convierten en cero.
  - Fuente y frescura son independientes: una cuota real puede estar obsoleta y una estimada puede estar fresca.
  - La cuota se evalua por scope; una cuota fresca de pool no autoriza consumir una cuenta/HOME agotada.
  - Si la ventana restante no cubre `contexto_estimado`, la decision debe marcar `ventana_insuficiente`.
Errores:
  - cuota_obsoleta
  - cuota_desconocida
  - ventana_insuficiente
```

## ModelEvidenceScoreV0

```text
Nombre: ModelEvidenceScoreV0
Tipo: dto
Version: v0
Propietario: orquesta-capacity
Campos:
  - materia: codigo | documentacion | analisis | revision | otro.
  - score_base: numero.
  - score_observado: numero.
  - score_total: numero.
  - confianza: numero entre 0 y 1.
  - muestras: entero >= 0.
  - benchmarks: entero >= 0.
  - last_observed_at: timestamp opcional.
Invariantes:
  - Sin muestras ni benchmarks, el score es semilla no probada y no habilita por si solo un modelo local.
  - La confianza baja impide promover local a preferente cuando hay riesgo alto o calidad critica.
```

## ModelEscalationPolicyV0

```text
Nombre: ModelEscalationPolicyV0
Tipo: dto
Version: v0
Propietario: orquesta-capacity
Consumidores: CapacityDecisionV0.
Campos:
  - policy_ref: identificador opaco y versionado de la politica usada.
  - phase_rules: lista de reglas por perfil_tarea/fase con base_level, max_without_evidence y allowed_levels.
  - risk_quality_rules: ajustes por riesgo low | medium | high | critical y calidad_requerida normal | alta | critica.
  - evidence_rules: senales aceptadas para subir, mantener o bajar capacidad; cada senal declara source, freshness y peso cualitativo.
  - quota_rules: reglas para bloquear, degradar o pedir handoff segun QuotaSnapshotV0.
  - locality_rules: reglas para elegir local/remoto/hibrido segun score, confianza, cuota, privacidad y coste.
  - xhigh_gate: condiciones minimas para permitir xhigh.
  - degradation_order: orden recomendado para bajar esfuerzo, cambiar modelo, cambiar HOME, cambiar pool o pedir handoff.
Matriz inicial:
  - discovery/triage: base low o medium; high solo con ambiguedad material, riesgo alto, contexto insuficiente o senales de fallo; xhigh no procede sin evidencia critica.
  - arquitectura/brainstorming: base medium; high si calidad alta, riesgo high/critical, decision irreversible o diseno cruzado; xhigh solo con riesgo critical mas evidencia explicita o solicitud humana justificada.
  - ejecucion/implementacion: base medium; low si cambio mecanico y riesgo low; high si falla prueba/revision, hay regresion probable, write-set amplio o calidad critica; xhigh solo ante fallo repetido o riesgo critical documentado.
  - revision: base medium; high si revision previa fallo, tests fallan, seguridad/datos/contrato estan afectados o hay desacuerdo entre evidencias; xhigh solo para bloqueo critico con evidencia.
  - documentacion/script/cierre: base low o medium; high si la documentacion es contrato compartido, hay riesgo de ambiguedad contractual o la ventana exige handoff preventivo; xhigh excepcional.
  - handoff/orquestacion: base medium; high si la ventana disponible no alcanza, hay dependencias cruzadas o perdida de contexto probable; xhigh solo con riesgo critical y aprobacion/evidencia.
Reglas de escalado:
  - low -> medium: fase no mecanica, incertidumbre moderada, cuota suficiente y calidad normal/alta.
  - medium -> high: riesgo high/critical, calidad critica, fallo de tests/revision, evidencia de baja calidad, contexto insuficiente, decision arquitectonica o impacto contractual.
  - high -> xhigh: requiere riesgo critical o fase arquitectura/brainstorming/revision critica, mas evidencia fresca de fallo, ambiguedad material, solicitud humana justificada o bloqueo repetido.
Reglas de degradacion:
  - high/xhigh -> medium: cuota stale/estimated sin respaldo, coste relativo excede restriccion, modelo no soporta esfuerzo, HOME sin concurrencia o score local insuficiente.
  - cualquier nivel -> handoff preventivo: ventana insuficiente, cuota obsolete/unavailable para tarea larga, o contexto_estimado supera limites conocidos.
  - remoto -> local: permitido por privacidad/coste solo si el modelo local tiene score probado, confianza suficiente y runtime local disponible.
  - local -> remoto: permitido si privacidad lo permite y el local no alcanza calidad, score, ventana o esfuerzo requerido.
Invariantes:
  - low/medium/high/xhigh son niveles de capacidad, no nombres de modelos, roles, agentes ni proveedores.
  - La matriz por fase es una recomendacion base; la decision final combina riesgo, calidad, cuota, coste, restricciones y evidencia.
  - No existe tabla estatica rol -> modelo ni perfil -> modelo.
  - `xhigh` no puede salir solo de fase, rol, preferencia o cuota sobrante.
  - Cuota obsolete bloquea high/xhigh salvo decision humana explicita o evidencia independiente documentada en motivos.
  - Las senales con freshness stale o unknown pesan menos y deben aparecer en motivos si condicionan la decision.
  - La politica debe devolver motivos auditables y alternativa degradada cuando no pueda aplicar el nivel preferido.
Errores:
  - politica_escalado_incompleta
  - fase_no_soportada
  - evidencia_insuficiente_para_xhigh
  - cuota_bloquea_escalado
  - modelo_no_soporta_esfuerzo
  - local_sin_score_suficiente
  - ventana_insuficiente
Pruebas de contrato:
  - CAP-CT-008 aplica base por fase y sube a high con evidencia fresca de calidad/riesgo.
  - CAP-CT-009 rechaza xhigh cuando falta gate de evidencia.
  - CAP-CT-010 degrada o pide handoff cuando cuota/HOME/modelo bloquean el nivel preferido.
```

## Evidencia DB v1 usada para este contrato

```text
Fuente forense en cuarentena: ref relativo opaco `backups/legacy-sqlite-20260422/orquesta.db` cuando exista localmente.
Estado documental: `forense`/`historico`/`quarantine`; no es prerequisito vivo ni fuente operativa para agentes.
Modo historico: SQLite URI readonly `mode=ro`, solo como evidencia pasada.
Tablas consultadas: politicas_modelo, pools_capacidad, pool_modelos, agentes, presupuestos_sesion, agente_scores_locales.
Hallazgos:
  - politicas_modelo: 52 filas activas; scopes global, perfil, fase y agente; prioridad por fila; reasoning observado vacio, medium, high y xhigh.
  - pools_capacidad: 6 pools activos; runtime/proveedor/plan presentes; paid y free; handoff preventivo; telemetria manual historica.
  - pool_modelos: 12 modelos activos; coste relativo observado entre 0.1 y 1.5; limites conocidos vacios.
  - agentes: cuota activa/pausas historicas y limites diarios/semanales; nombres y estados no se promueven como canon.
  - presupuestos_sesion: 634 snapshots; ventanas primary, short/5h, weekly y provider; fuentes observadas de perfil, conteo estimado/observado y backoff.
  - agente_scores_locales: 10 scores iniciales sin muestras ni benchmarks; no bastan para habilitar local como preferente.
Lectura: se promocionan patrones estructurales, no filas, nombres, cuotas ni proveedores historicos.
```

## CONSULTA AL DIRECTOR

```text
Modulo origen: orquesta-capacity
Modulos afectados: orquesta-core, orquesta-runtime, orquesta-observability, contratos globales.
Bloqueo: CapacityDecisionV0 sera consumido por core y condicionara RuntimeLaunchRequest v0, pero el contrato global aun lo marca como pendiente.
Pregunta concreta: ¿Promovemos CapacityDecisionV0, PoolCapacidadV0, QuotaSnapshotV0 y AgentHomeV0 preliminar a contrato compartido global en ../../CONTRATOS.md, o permanecen locales hasta que runtime formalice RuntimeLaunchRequest v0?
Opcion recomendada: Promover CapacityDecisionV0 minimo como contrato global con detalle canonico local, y mantener AgentHomeV0 como preliminar local hasta la microtarea de runtime.
Impacto: core podra pedir capacidad sin conocer proveedores ni DB; runtime recibira solo referencias opacas y no secretos; observability podra auditar decision_id, motivos y fuente de cuota.
Decision del director: Promover `CapacityDecision v0` a contrato global minimo en `../../CONTRATOS.md`. El detalle queda en este archivo. `AgentHomeV0` permanece preliminar local hasta que `orquesta-runtime` formalice `RuntimeLaunchRequest v0`.
```

## Plantilla

```text
Nombre:
Tipo: puerto_entrada | puerto_salida | dto | evento | error
Version:
Propietario:
Consumidores:
Campos:
Invariantes:
Errores:
Pruebas de contrato:
```
