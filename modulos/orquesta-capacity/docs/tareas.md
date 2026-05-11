# Tareas locales: orquesta-capacity

Cada tarea debe ser pequena y cerrada.

## Backlog inicial desde DB v1

```text
ID: CAP-001
Objetivo: Extraer modelo de pools, politicas de modelo, cuotas y multi-HOME desde DB v1 como contrato v0.
Write-set: docs/tareas.md, docs/decisiones.md, docs/contratos.md, docs/pruebas.md
Simbolo foco: CapacityDecisionV0
Contrato: CapacityDecision v0
Validacion: no hardcodea proveedores ni plataformas historicas; permite remoto/local y escalado por evidencia.
Bloqueos: DBV1-000 completada en docs/reinicio_orquesta_v2/inventario_db_v1.md.
Estado: completada el 2026-05-04.
Resultado: CapacityDecisionV0 documentado con PoolCapacidadV0, ModelCapacityRefV0, AgentHomeV0 preliminar, QuotaSnapshotV0 y ModelEvidenceScoreV0. Incluye invariantes, errores, pruebas previstas y CONSULTA AL DIRECTOR para promocion global.
```

```text
ID: CAP-002
Objetivo: Definir AgentHomeV0 para separar agente logico, cuenta, runtime, proveedor y cuota.
Write-set: docs/contratos.md, docs/pruebas.md
Simbolo foco: AgentHomeV0
Contrato: CapacityDecision v0, RuntimeLaunchRequest v0 futuro
Validacion: soporta multiples HOME/OAuth por proveedor, agentes premium/pro/locales y limite de concurrencia por cuenta.
Bloqueos: CAP-001.
Estado: completada el 2026-05-04.
Resultado: AgentHomeV0 ampliado como binding opaco entre agente logico, HOME, cuenta, runtime, proveedor, credencial y pool. Documenta multi-HOME/OAuth, cuota por scope, concurrencia por cuenta, HOME premium/pro/local y errores de cuota, OAuth y concurrencia.
```

```text
ID: CAP-003
Objetivo: Definir politica de escalado de modelo por fase y evidencia de calidad.
Write-set: docs/decisiones.md, docs/contratos.md
Simbolo foco: ModelEscalationPolicyV0
Contrato: CapacityDecision v0
Validacion: low/medium/high/xhigh es recomendacion dinamica; puede subir si baja calidad, falla revision, hay riesgo alto o fase de arquitectura/brainstorming.
Bloqueos: CAP-001.
Estado: completada el 2026-05-04.
Resultado: ModelEscalationPolicyV0 documenta matriz base por fase, reglas de subida low/medium/high/xhigh por evidencia, gates para xhigh, degradacion por cuota/HOME/modelo y reglas local/remoto por score, privacidad, coste y disponibilidad.
```

## Backlog documental para schemas y fixtures

```text
ID: CAP-004
Objetivo: Crear schema y fixtures de CapacityDecisionV0 sin implementar validador ni codigo.
Write-set: docs/schemas/capacity_decision_v0.schema.json, docs/fixtures/capacity_decision_v0/*.json, docs/pruebas.md
Simbolo foco: CapacityDecisionV0
Contrato: CapacityDecision v0
Validacion: el schema cubre entrada y salida del puerto, exige referencias opacas, source/freshness de cuota y motivos auditables; los fixtures cubren decision minima valida, xhigh con evidencia valida, xhigh sin evidencia invalida, cuota obsoleta con degradacion/handoff, modelo local sin score suficiente y proveedor/modelo hardcodeados.
Bloqueos: CAP-001 y CAP-003 completadas; no requiere cambiar ../CONTRATOS.md.
Estado: completada el 2026-05-04.
Resultado: Se anyadieron schema draft-07 documental y fixtures JSON validos/invalidos en docs/schemas y docs/fixtures. No se implemento Go ni harness ejecutable.
```

```text
ID: CAP-005
Objetivo: Crear schema y fixtures de AgentHomeV0 como binding opaco de HOME/cuenta/runtime.
Write-set: docs/schemas/agent_home_v0.schema.json, docs/fixtures/agent_home_v0/*.json, docs/pruebas.md
Simbolo foco: AgentHomeV0
Contrato: CapacityDecision v0, RuntimeLaunchRequest v0 futuro
Validacion: el schema rechaza rutas HOME reales, emails, tokens, secretos, proveedor hardcodeado y nombres historicos; los fixtures cubren multi-HOME/OAuth valido, HOME local sin OAuth valido, cuota HOME obsoleta, concurrencia agotada y reserved_sessions mayor que max_concurrent_sessions como invalido.
Bloqueos: CAP-002 completada y confirmacion futura de RuntimeLaunchRequest v0 antes de promocionar detalles globales.
Estado: completada el 2026-05-04.
Resultado: Se anyadieron schema draft-07 documental y fixtures JSON validos/invalidos. AgentHomeV0 permanece preliminar local.
```

```text
ID: CAP-006
Objetivo: Crear schema y fixtures de ModelEscalationPolicyV0.
Write-set: docs/schemas/model_escalation_policy_v0.schema.json, docs/fixtures/model_escalation_policy_v0/*.json, docs/pruebas.md
Simbolo foco: ModelEscalationPolicyV0
Contrato: CapacityDecision v0
Validacion: el schema exige phase_rules, risk_quality_rules, evidence_rules, quota_rules, locality_rules, xhigh_gate y degradation_order; los fixtures prueban base por fase, subida a high por evidencia fresca, xhigh bloqueado sin gate, xhigh permitido con evidencia critica, degradacion por cuota/HOME/modelo y ausencia de tabla estatica rol -> modelo.
Bloqueos: CAP-003 completada; no requiere cambiar contratos globales.
Estado: completada el 2026-05-04.
Resultado: Se anyadieron schema draft-07 documental y fixtures JSON validos/invalidos sin tabla rol -> modelo.
```

## Backlog Go puro

```text
ID: CAP-007
Objetivo: Implementar DTOs Go y validador puro de CapacityDecisionV0 apoyado en contratos, schemas y fixtures locales.
Write-set: capacity_decision_v0.go, capacity_decision_validator_v0.go, capacity_decision_components_v0.go, capacity_decision_invariants_v0.go, capacity_decision_validator_helpers_v0.go, capacity_decision_v0_test.go, docs/tareas.md, docs/pruebas.md, docs/contratos.md, docs/decisiones.md
Simbolo foco: CapacityDecisionV0
Contrato: CapacityDecision v0
Validacion: decode JSON estricto, referencias opacas, xhigh con evidencia fresca, cuota obsolete con degradacion/handoff, modelo local solo con score probado y sin proveedor/modelo/HOME real hardcodeado.
Bloqueos: CAP-004 completada; no implementa seleccion real, cuotas reales, DB, filesystem, runtime ni proveedores.
Estado: completada el 2026-05-04.
Resultado: Se anyadieron DTOs exportados, `DecodeCapacityDecisionV0`, `ValidateCapacityDecisionV0`, error estructurado y tests unit/contract contra fixtures validos e invalidos. La implementacion es pura, no llama a adaptadores y queda dividida en ficheros pequenos por responsabilidad.
```

```text
ID: CAP-008
Objetivo: Implementar DTOs Go y validador puro de ModelEscalationPolicyV0 apoyado en schema y fixtures locales.
Write-set: model_escalation_policy_v0.go, model_escalation_policy_validation_v0.go, model_escalation_policy_helpers_v0.go, model_escalation_policy_v0_test.go, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Simbolo foco: ModelEscalationPolicyV0
Contrato: CapacityDecision v0
Validacion: decode JSON estricto, refs opacas, reglas obligatorias de fase/evidencia/cuota/localidad/degradacion, xhigh con gate y evidencia fresca, rechazo de tabla estatica rol -> modelo y de proveedor/modelo/HOME real.
Bloqueos: CAP-006 completada; no implementa seleccion real de modelos, cuotas reales, DB, filesystem, runtime, OAuth ni proveedores.
Estado: completada el 2026-05-05.
Resultado: Se anyadieron DTOs exportados, `DecodeModelEscalationPolicyV0`, `ValidateModelEscalationPolicyV0`, `Valid()`/`Validate()` y tests contra fixtures validos e invalidos de `docs/fixtures/model_escalation_policy_v0/`.
```

## Plantilla

```text
ID:
Objetivo:
Write-set:
Simbolo foco:
Contrato:
Validacion:
Bloqueos:
Estado:
```
