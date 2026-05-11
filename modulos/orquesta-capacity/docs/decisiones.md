# Decisiones locales: orquesta-capacity

Las decisiones de este archivo solo afectan a `orquesta-capacity`. Si afectan a otro modulo, deben elevarse al director y registrarse en `../../CONTRATOS.md`.

## Plantilla

```text
Fecha:
Decision:
Motivo:
Alternativas:
Impacto:
Contratos afectados:
Estado:
```

## Decisiones iniciales desde DB v1

```text
Fecha: 2026-05-04
Decision: Promover el modelo conceptual de pools, politicas por scope, multi-HOME y handoff por cuota; rechazar nombres concretos de agentes y cuotas viejas como seed.
Motivo: La DB v1 demuestra que la capacidad no es estatica y que se necesita separar agente logico, proveedor, cuenta, runtime, modelo, reasoning y presupuesto.
Alternativas: Fijar un proveedor unico; copiar pools historicos completos; dejar capacidad para mas adelante.
Impacto: `orquesta-capacity` debe exponer contratos portables y resolver capacidad sin depender de proveedor ni tabla concreta.
Contratos afectados: CapacityDecision v0, AgentHomeV0, ModelEscalationPolicyV0.
Estado: aceptada_director
```

```text
Fecha: 2026-05-04
Decision: Implementar CapacityDecisionV0 en Go como DTOs y validador puro, no como motor de seleccion.
Motivo: CAP-007 necesita una superficie ejecutable para core/runtime futuros sin introducir DB, proveedores, filesystem, cuotas reales ni tablas rol -> modelo.
Alternativas: Generar Go desde JSON Schema; implementar selector real; validar solo fixtures con AJV.
Impacto: `capacity_decision_v0.go` expone DTOs y decoder estricto; el validador queda dividido en ficheros pequenos por responsabilidad. Los tests locales cubren fixtures validos/invalidos, xhigh, cuota obsolete, ventana insuficiente y referencias opacas.
Contratos afectados: CapacityDecisionV0, PoolCapacidadV0, ModelCapacityRefV0, QuotaSnapshotV0, AgentHomeV0 preliminar.
Estado: aceptada_local
```

```text
Fecha: 2026-05-05
Decision: Implementar ModelEscalationPolicyV0 en Go como DTOs y validador puro, no como selector de modelos.
Motivo: CAP-008 necesita ejecutar las invariantes documentadas para politicas de escalado sin introducir proveedor, runtime, DB, OAuth, HOME real, cuota real ni tabla estatica rol -> modelo.
Alternativas: Generar Go desde JSON Schema; validar solo con AJV; implementar motor de escalado real.
Impacto: `model_escalation_policy_v0.go` expone DTOs, decoder estricto y `Valid()`/`Validate()`; el validador se divide en ficheros pequenos y cubre reglas de fase, evidencia, cuota, localidad, degradacion, gate xhigh y referencias opacas.
Contratos afectados: ModelEscalationPolicyV0, CapacityDecisionV0.
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: Publicar JSON Schema draft-07 documentales para CapacityDecisionV0, AgentHomeV0 y ModelEscalationPolicyV0 con referencias opacas y fixtures validos/invalidos.
Motivo: CAP-004..CAP-006 necesita ejemplos parseables antes de implementar validadores o codigo; los schemas deben bloquear proveedor hardcodeado, secretos, rutas HOME reales, emails/tokens y tabla fija rol -> modelo.
Alternativas: Esperar a harness Go; relajar schemas y validar solo en runtime; duplicar contratos globales.
Impacto: Los contratos locales tienen schemas y fixtures iniciales en docs/schemas y docs/fixtures; la validacion ejecutable queda para microtarea posterior.
Contratos afectados: CapacityDecisionV0, AgentHomeV0, ModelEscalationPolicyV0.
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: Definir CapacityDecisionV0 como seleccion dinamica por perfil, fase, riesgo, calidad, coste, cuota y evidencia.
Motivo: `politicas_modelo` en DB v1 muestra scopes global, perfil, fase y agente con prioridad; esto valida una politica por capas, pero no una tabla fija rol -> modelo.
Alternativas: Resolver solo por rol; resolver solo por coste; delegar seleccion al runtime/proveedor.
Impacto: `orquesta-core` solicitara capacidad por intencion y restricciones; `orquesta-capacity` devolvera pool/modelo/home opacos y motivos auditables.
Contratos afectados: CapacityDecisionV0, PoolCapacidadV0, QuotaSnapshotV0.
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: Mantener `xhigh` como nivel excepcional y exigir evidencia explicita para escalar.
Motivo: DB v1 contiene pocos casos xhigh frente a high/medium/vacio y las reglas locales del modulo exigen evidencia; sin esa barrera se gastaria cuota/coste sin justificacion.
Alternativas: Permitir xhigh por perfil; prohibir xhigh en v0; usar solo low/medium/high.
Impacto: El contrato declara el error `evidencia_insuficiente_para_xhigh` y pruebas de contrato especificas.
Contratos afectados: CapacityDecisionV0, ModelEvidenceScoreV0.
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: Tratar la cuota como snapshot con fuente y frescura, no como numero absoluto.
Motivo: `presupuestos_sesion` mezcla ventanas y fuentes historicas; algunas son observadas, otras derivadas o manuales. Copiar valores como verdad romperia el modelo V2.
Alternativas: Guardar solo porcentaje restante; exigir siempre telemetria real; ignorar cuota en v0.
Impacto: QuotaSnapshotV0 distingue real, estimated, manual y unavailable; tambien fresh, stale, obsolete y unknown.
Contratos afectados: QuotaSnapshotV0, CapacityDecisionV0.
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: AgentHomeV0 queda preliminar y no canoniza nombres, rutas HOME, cuentas ni estados historicos.
Motivo: `agentes` evidencia multi-HOME, pausa y reanimacion por cuota, pero sus nombres/roles/limites son estado operacional viejo.
Alternativas: Copiar agentes historicos como seed; dejar HOME solo en runtime; ocultar HOME a capacity.
Impacto: Capacity puede razonar sobre disponibilidad de HOME/cuenta sin exponer secretos ni acoplarse a runtime.
Contratos afectados: AgentHomeV0, RuntimeLaunchRequest v0 futuro.
Estado: aceptada_local_con_consulta_director
```

```text
Fecha: 2026-05-04
Decision: Definir AgentHomeV0 como binding operativo entre agente logico, HOME, cuenta, runtime, proveedor, credencial y pool.
Motivo: CAP-002 necesita soportar multiples HOME/OAuth por proveedor, cuentas premium/pro/locales y limite de concurrencia sin exponer secretos ni rutas reales.
Alternativas: Modelar HOME como identidad del agente; modelar cuota solo a nivel pool; delegar toda seleccion de HOME al runtime.
Impacto: Capacity puede ordenar HOME alternativos por cuota, concurrencia, coste y evidencia, mientras runtime futuro recibe solo referencias opacas.
Contratos afectados: AgentHomeV0, PoolCapacidadV0, QuotaSnapshotV0, RuntimeLaunchRequest v0 futuro.
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: Separar fuente, frescura y scope de cuota antes de usarla para escalar o degradar capacidad.
Motivo: Una cuota real puede estar obsoleta, una estimacion puede estar fresca y la cuota de pool no prueba que una cuenta/HOME concreta tenga ventana disponible.
Alternativas: Tratar plan premium/pro como cuota fiable; convertir valores ausentes en cero; usar solo cuota global de proveedor.
Impacto: High/xhigh quedan bloqueados por cuota obsoleta salvo evidencia independiente o decision humana; ventanas insuficientes fuerzan degradacion o handoff preventivo.
Contratos afectados: QuotaSnapshotV0, AgentHomeV0, ModelEscalationPolicyV0.
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: Definir ModelEscalationPolicyV0 como politica dinamica por fase, riesgo, calidad, cuota, localidad y evidencia.
Motivo: CAP-003 requiere que low/medium/high/xhigh sean recomendaciones de capacidad y no una tabla fija rol -> modelo; la DB v1 evidencia politicas por scope, pero no justifica copiar filas.
Alternativas: Escalar solo por perfil; fijar modelos por rol; permitir xhigh por arquitectura sin gate; no documentar degradacion.
Impacto: Capacity aplica una base por fase, sube con evidencia fresca de calidad/riesgo/fallo y degrada si cuota, HOME, coste o modelo bloquean el nivel preferido.
Contratos afectados: CapacityDecisionV0, ModelEscalationPolicyV0, ModelCapacityRefV0, QuotaSnapshotV0.
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: Promover `CapacityDecision v0` a contrato compartido minimo y mantener `AgentHomeV0` preliminar local.
Motivo: `orquesta-core` necesita pedir capacidad sin conocer proveedores ni DB; runtime todavia no tiene `RuntimeLaunchRequest v0` y no debe recibir contratos de HOME cerrados antes de tiempo.
Alternativas: Mantener todo local; promover tambien AgentHomeV0 completo; esperar a runtime.
Impacto: `../../CONTRATOS.md` registra el contrato global minimo; los detalles y pruebas permanecen en `orquesta-capacity`.
Contratos afectados: CapacityDecision v0, AgentHomeV0, RuntimeLaunchRequest v0 futuro.
Estado: aceptada_director
```
