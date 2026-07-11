# Tareas: orquesta-autoprogramming

## Estado actual

- Extraido desde `orquesta-orchestration-core` para separar reglas de
  programacion del nucleo generico.
- MCP usa este modulo como preflight.
- `orquesta-runtime-codex-delivery` usa este modulo para evaluar ACKs de codigo.
- La automejora secundaria ya queda modelada como contrato puro:
  `BuildAutoprogrammingSelfImprovementRequestV0` produce una request de baja
  prioridad para que el adaptador la prepare/encole despues.
- `BuildAutoprogrammingProgrammableWorkV0` compacta textos largos antes de
  `WorkflowTaskV0` y devuelve issues con subcampo causal para reparacion.
- Los limites vigentes de ola admiten 10 padres por defecto y hasta 6
  subagentes por padre en el trabajo programable; limites menores pueden
  declararse explicitamente por request cuando una composicion quiera acotar la
  ola.
- `BuildAutoprogrammingProgrammableWorkV0` emite `goal_migration` para separar
  tareas que siguen en loop legacy de tareas candidatas/cubiertas por Goal,
  usando solo refs opacas en `context_refs`.
- Si `goal_migration=goal_ready`, el trabajo programable incluye
  `goal_specs[]` (`GoalWorkSpecV0`) por grupo, sin lanzar runtime ni depender de
  Codex real.

## Cerrado

- APG-001: contrato `v0` cerrado: OPES o apps grandes pueden declarar
  `max_task_refs`, `max_areas` y `max_write_set_entries` hasta 40 sin
  microfragmentar contenido; texto libre no amplia limites.
- APG-002: `AutoprogrammingRequestV1` permite perfiles de trabajo por tipo de
  app, area o tarea y `BuildAutoprogrammingProgrammableWorkV1` materializa esos
  perfiles sobre `WorkProfileV0`/`WorkflowTaskV0`; `v0` conserva su fallback de
  implementacion.
- APG-003: contrato puro `AutoprogrammingRequestSourceV0` para evidencia de
  entrada real web/MCP/CLI sin importar adaptadores. Las fixtures en
  `docs/fixtures/autoprogramming_request_v0/` son ejemplos versionados y pruebas
  de compatibilidad, no el mecanismo de cierre por si solas.
- APG-004: reconciliacion documental T208, 2026-05-27. El guardian break-glass
  queda tratado como umbrella historico ya cubierto por owners especificos
  T212-T237; este modulo no absorbe runtime, VCS, Codex, servidor ni guardian.
  Si aparece regresion, abrir tarea focal con write-set propio.
- APG-005: clasificacion y specs goal-first, 2026-06-25. El modulo distingue
  `legacy_loop_compatible`, `goal_ready`, `blocked_by_goal_capability`,
  `covered_by_goal_first` y `legacy_loop_required`; cuando hay `goal_ready`,
  compila `GoalWorkSpecV0` neutral con write-set/pruebas/criterios por grupo,
  sin lanzar Codex ni tocar servidor. La composicion sigue siendo responsable de
  crear/observar goals y validar smokes reales.

## APG-006: piloto de rotacion de agentes con contexto compacto

Estado: harness preparado en `scripts/experimentos/context_rotation/` y
`docs/experimentos/context_rotation/` (`9cdc2d586`); piloto real pendiente de
ejecucion por Orquesta. No habilitar como default antes de cerrar la evaluacion
independiente y durable descrita aqui.

Objetivo: comprobar si sustituir sesiones Codex crecientes por workers frescos
en fronteras semanticas reduce el coste total y mejora foco/fiabilidad sin
degradar calidad, tiempo, causalidad ni tasa de cierre. Si el resultado supera
los umbrales de adopcion, Orquesta debe materializar una tarea separada para
convertir la rotacion en politica default goal-first reversible. Si no los
supera, conserva el informe y deja la politica desactivada.

Hipotesis:

- Orquesta conserva estado durable de objetivo, plan, refs, baseline, write-set,
  decisiones, checkpoints, artefactos, tests, evidencias y bloqueos.
- Cada worker recibe un paquete compacto con objetivo exacto, causa actual,
  baseline, write-set, invariantes aplicables, refs relevantes, fallo anterior
  resumido, tests y criterios de cierre; nunca el transcript completo.
- Se rota solo tras checkpoint/commit aceptado, cambio de tarea o write-set,
  rework con causa nueva, o falta material de progreso observada. No se corta un
  write-set con mutacion no entregada ni se pierde parentesco causal.
- Implementador y revisor usan contextos independientes; artefactos grandes se
  persisten y se transportan por refs/hash, no copiandolos por conversaciones.

Diseno minimo del piloto:

1. Usar al menos tres pares de trabajos de programacion representativos, cada
   par ejecutado desde el mismo commit y snapshot en worktrees aislados, sin
   promocion automatica ni efectos productivos.
2. Control: una sesion continuada. Tratamiento: worker fresco por frontera
   semantica con handoff compacto. Mismo modelo, capacidad, herramientas,
   criterios, tests, write-set y presupuesto maximo por par.
3. Incluir en el coste total todos los tokens de Director, workers, handoffs,
   recuperaciones y revisores; separar input, output y cacheados cuando el
   proveedor los exponga. No declarar ahorro usando solo tokens del worker.
4. Registrar tambien tiempo total, tool calls, lecturas repetidas, reworks,
   intentos fallidos, conflictos de write-set, intervencion humana, tests,
   cierre independiente y defectos encontrados tras la entrega.
5. Evaluacion ciega por un revisor fresco que reciba objetivo, diff,
   artefactos, criterios y evidencias, pero no el modo asignado ni razonamientos
   de los implementadores.
6. Persistir dataset, configuracion, receipts, resultados por par y calculo de
   decision. Un resumen narrativo o una afirmacion del agente no cuentan como
   medicion.

Umbral de adopcion:

- todos los tratamientos alcanzan la misma aceptacion independiente y los
  mismos tests requeridos que su control, sin falso verde, perdida causal,
  efecto externo adicional ni defecto severo nuevo;
- reduccion mediana de al menos 20% en tokens/coste total y mejora en al menos
  dos de los tres pares;
- tiempo mediano no superior al control en mas de 10%, y reworks, intentos
  fallidos e intervenciones humanas no aumentan;
- los handoffs pueden ser continuados por un worker fresco usando solo el
  paquete y refs declarados, sin pedir transcript bruto ni releer areas fuera
  del scope;
- el revisor independiente acepta el informe y su reproduccion focal.

Adopcion condicionada:

- con veredicto `adopt`, crear por Orquesta una tarea causal separada, con
  write-set propio, para activar rotacion reversible solo en rutas goal-first de
  programacion/autoprogramacion;
- antes de anadir configuracion, buscar y reutilizar el registro canonico
  existente; queda prohibida una variable o lectura de entorno ad hoc;
- conservar excepcion explicita para tareas indivisibles con dependencias
  causales densas, justificandola en el receipt;
- observar las diez primeras ejecuciones adoptadas y revertir al modo anterior
  si aparece regresion de calidad, causalidad o coste respecto del piloto;
- con veredicto `keep_control` o evidencia insuficiente, no cambiar produccion
  ni crear la tarea de adopcion.

Write-set del piloto:

- un harness nuevo y aislado bajo `scripts/`;
- fixtures/receipts bajo una ruta nueva de `docs/experimentos/`;
- este registro de tarea solo para actualizar estado y enlazar el resultado.

Quedan fuera hasta un veredicto `adopt`: cambios de defaults, runtime
productivo, proveedor, OPES, conectores, configuracion residente y borrado de
compatibilidad.

Pruebas/evidencias requeridas:

- `git diff --check`;
- pruebas focales del harness y validacion de schemas/receipts;
- reproduccion de los pares desde snapshots declarados;
- informe con decision mecanica `adopt|keep_control|inconclusive`, metricas
  completas y refs de evidencia.

Referencias primarias de diseno:

- `https://arxiv.org/abs/2307.03172` (`Lost in the Middle`);
- `https://arxiv.org/abs/2404.06654` (`RULER`);
- `https://www.anthropic.com/engineering/multi-agent-research-system`;
- `https://openai.com/business/guides-and-resources/a-practical-guide-to-building-ai-agents/`;
- `https://docs.langchain.com/oss/python/langchain/multi-agent/subagents`;
- `https://microsoft.github.io/autogen/stable/reference/python/autogen_agentchat.agents.html`.
