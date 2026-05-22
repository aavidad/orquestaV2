# Contexto para agentes: Orquesta

Lee este archivo antes de trabajar en el repo. Luego lee solo los documentos y
`AGENTS.md` locales necesarios para tu modulo.

## Foto vigente

La direccion actual no es "app de programacion con Codex". Orquesta es un
nucleo reutilizable de orquestacion de agentes para apps externas. Programacion
con Codex y OPES son composiciones consumidoras, no el nucleo.
El nucleo generico tampoco es producto OPES: OPES entra solo por conectores de
dominio sobre puertos/refs opacas, igual que cualquier otra app externa.

Tras los cortes del 2026-05-17, el estado real es:

- `modulos/orquesta-director-operativo` existe como contrato puro del Director
  Operativo V1: plan, olas, guardas, contexto insuficiente y delegacion
  recursiva gobernada. No ejecuta runtime.
- `modulos/orquesta-orchestration-core/operational_director_materializer_v0.go`
  materializa solo items `launch_subagents` listos como `WorkflowTaskV0` y
  comandos `CreateMicrotask` con outbox causal. Aun no materializa todo el ciclo
  de espera/review/replan/cierre.
- `WaitAgentRefs` ya existe en los loops progresivos y en el stack Codex para
  esperar agentes concretos. `app-director-service` puede derivarlos desde
  `wait_cohort_ref`, `wait_wave_ref` o `wait_parent_task_ref` usando las
  `WorkflowTaskV0` persistidas. `WorkflowTaskWaitStateV0` registra esa espera
  con causa, tasks, agentes objetivo y pendientes; `orquesta-state-file` lo
  persiste. La regla nueva es usar refs de cohorte/ola cuando esten disponibles,
  no esperar todos los agentes vivos del run.
- `ContinueAppDirectorV0` ya puede recibir un `OperationalDirectorPlanV0` listo,
  materializar `launch_subagents` antes del loop, derivar la ola y reentrar al
  loop con scope acotado.
- El cierre operativo generico ya empieza desde `ContinueAppDirectorV0` tras un
  loop quiescent: si la composicion inyecta `OperationalClosureSource`,
  `app-director-service` construye un `OperationalDirectorClosureRequestV0` y
  `OperationalDirectorClosureV0` cierra task, validacion final y run con review
  aceptada y evidencia durable de tests requeridos. El stack Codex ya inyecta
  una fuente real acotada a microtareas materializadas por el Director
  Operativo; deriva la peticion desde `WorkflowTaskStore` + eventos persistidos
  y no cierra tareas legacy de app-change. Si otra composicion no tiene fuente
  real, ese wiring sigue pendiente y debe documentarse como tal.
- P1 WaitAgentRefs queda cerrado para el stack Codex: `WaitAgentRefs` no vacio
  acota pending, wait e ingesta de ACK/deliveries. `DrainRunRequestV0`
  transporta el scope a `AgentDeliveryObservationRequestV0`, las fuentes de
  observacion quedan filtradas y `DrainRunV0` ignora ACK/delivery de agentes
  fuera del scope. `CodexDeliveryObservationSourceV0` no lee ACKs fuera de scope
  y `domain_work` no hace submit/recovery fuera del scope. `WaitAgentRefs` vacio
  conserva compatibilidad legacy de run completo.
- `app-director-service` ya inyecta `DirectorTaskStore` en review/replan para
  que un `split_task` pueda persistir nuevas `WorkflowTaskV0`; sigue pendiente
  convertir esa pieza en tramo durable del Director Operativo.
- El siguiente tramo no es mas scope de espera ni source basico del stack: la
  salida positiva de `review_deliveries` por ola ya avanza el `PlanState` por
  cadena causal de eventos, y `run_required_tests` ya consume
  `RequiredTestEvidenceV0` durable para pasar a `replan_or_close` o bloquear por
  `required-tests-failed`. Sigue pendiente un runner/adaptador real que genere
  esas evidencias, la rama negativa review/replan, `close` completo,
  replay/idempotencia y estado vivo posterior.
- El nuevo handoff de cierre es
  `docs/corte_cierre_generico_director_operativo_2026-05-17.md`: P1
  `WaitAgentRefs` no se reabre salvo regresion; el foco es cierre causal
  offline generico del Director Operativo. Si no hay codigo/prueba integrada,
  documentalo como pendiente verificable, no como hecho.
- OPES es consumidor, no producto base del nucleo. Ya paso el primer corte
  `plan_tema -> document_plan -> derivados` y el
  expander neutral `DomainDocumentPlanV0 -> DomainWorkJobRequestV0[]` ya existe.
  El conector REST OPES opt-in ya existe para `GET /api/jobs`,
  `GET /api/topics/{id}/blocks`, `POST /api/jobs` y
  `POST /api/jobs/{id}/artifacts`; `plan_temario` local queda cubierto como
  `document_plan` con politica editorial OPES y `xhigh`. La superficie para IA
  es `orquesta.domain_work.v0` en `orquesta-mcp`; REST OPES es solo el
  adaptador inyectado hoy, y MCPO/servidor MCP real debe quedar como transporte
  opt-in. Falta cerrar smoke real de `plan_temario` contra instancia OPES
  aislada, no tocar OPES productivo ni drenar colas amplias.
- La recursion Codex completa sigue pendiente: subagentes de subagentes con
  parent/child refs, presupuesto global, profundidad/fanout y review del
  director antes de cerrar.
- La metadata completa de `WorkflowTaskV0` vive en `WorkflowTaskStore`. Un
  replay solo desde eventos compactos reconstruye refs de tareas, no
  `wave_ref`, `cohort_ref`, parent/child refs, criterios completos ni el estado
  vivo de espera; cualquier recuperacion debe restaurar o rematerializar ese
  store y el `WorkflowTaskWaitStateV0`.

Documentos de entrada obligatorios para cambios transversales:

- `docs/estado_actual_2026-05-17.md`
- `docs/guia_nucleo_orquestacion_2026-05-17.md`
- `docs/principio_orquesta_piensa_director.md`
- `docs/director_operativo_v1_2026-05-17.md` si cambias Director Operativo,
  plan vivo, workflow tasks, candidatos/outbox, cohortes/oleadas o delegacion
  recursiva.
- `docs/corte_director_funcionando_tarde_2026-05-17.md` si el objetivo es dejar
  Orquesta operativa hoy con el Director, waits, review/rework o recursion.
- `docs/corte_cierre_generico_director_operativo_2026-05-17.md` si cambias
  cierre causal, review por ola, tests requeridos durables, replan/close o
  estado vivo del plan.
- `docs/corte_opes_como_consumidor_orquesta_2026-05-18.md` y
  `docs/runbooks/smoke_opes_plan_temario_operadores_2026-05-18.md` si tocas
  OPES, `orquesta-opes-*`, `DomainWork` aplicado a OPES, bridge/drain OPES o
  reglas editoriales de temarios.
- `docs/matriz_pruebas_reales_y_smoke_2026-05-17.md` si cambias smokes,
  runtime, OPES, shutdown o pruebas reales.

## Capas

- Core puro: `modulos/orquesta-core-workflow`.
- Loop de aplicacion del nucleo: `modulos/orquesta-orchestration-core`.
- Plan operativo del director: `modulos/orquesta-director-operativo`.
- Trabajo externo neutral: `modulos/orquesta-domain-work`,
  `modulos/orquesta-app-change`, `modulos/orquesta-external-work-run`.
- Adaptadores de dominio: `modulos/orquesta-opes-*`.
- Runtime neutral: `modulos/orquesta-runtime`.
- Adaptador Codex: `modulos/orquesta-runtime-codex*`.
- Composicion real actual: `modulos/orquesta-app-codex-stack` y
  `cmd/orquesta-server`.
- Persistencia concreta actual: `modulos/orquesta-state-file` y
  `modulos/orquesta-run-file`.
- Adaptador SQL externo de referencia: `modulos/orquesta-domain-work-sql`, solo
  para `domain_work` y sin bundle productivo, driver, DSN, migraciones ni wiring
  de servidor.

## Reglas

- No reintroduzcas `cmd/db/internal`, `ensureLocalDB` ni control-plane legacy.
- No metas Codex, OPES, web, MCP, DB concreta, HOME, OAuth, tokens, paths
  locales ni runtime real dentro de core/workflow/domain-work.
- No trates `orquesta-domain-work-sql` como persistencia global del sistema: una
  DB real debe entrar por composicion opt-in con driver, schema y pruebas propias.
- Toda app externa debe entrar por contratos y refs opacas. No compartir DB ni
  filesystem interno con Orquesta.
- El dominio externo aporta datos, reglas, validadores y ensamblado. Orquesta
  aporta juicio mediante director/agentes.
- Si una decision requiere producto, runtime, modelo, cuota o proveedor,
  documenta la frontera y dejala en adaptador/composicion.
- No borres documentos o codigo antiguo sin revisar. Si algo es historico,
  marcalo como historico y apunta al documento vigente.
- No borres archivos, docs, tests ni piezas "legacy" solo porque parezcan
  obsoletas: revisa primero referencias, historial y uso actual; si no puedes
  cerrarlo con evidencia, deja nota de pendiente.
- Economia de tokens: al lanzar agentes o subagentes, pide comunicacion compacta
  y tecnicas de ahorro como `caveman` si estan disponibles. Para exploracion y
  pruebas usa razonamiento `medium` por defecto; no uses `xhigh` salvo orden
  explicita o riesgo tecnico justificado y documentado. Acota contexto con `rg`
  y lecturas parciales, resume documentos largos en vez de pegarlos completos y
  pide finales breves con solo cambios, pruebas y bloqueos.

## Checklist operativo para futuros agentes

1. Declara write-set antes de editar y mantenlo estrecho.
2. Lee `AGENTS.md` local del modulo y los documentos vigentes listados arriba.
3. Si tocas Director Operativo, comprueba la cadena:
   `orquesta-director-operativo` -> `BuildOperationalDirectorWaveWorkV0` ->
   `OperationalDirectorPlanMaterializerV0` -> `WorkflowTaskStore` ->
   `WorkflowTaskCandidateProviderV0` -> wait refs de `app-director-service` ->
   providers de review/replan -> outbox.
4. Si tocas waits, usa `WaitAgentRefs`/cohorte/ola; no conviertas la espera en
   "todos los agentes del run". Conserva la regla cerrada: `WaitAgentRefs` no
   vacio limita pending, wait e ingesta de ACK/deliveries; `WaitAgentRefs`
   vacio conserva compatibilidad legacy.
5. Si tocas OPES, exige instancia temporal, guarda de confirmacion y filtro por
   tipo de job.
6. Si tocas recursion Codex, conserva parent/child refs, limites de
   profundidad/fanout, presupuesto y review causal.
7. El tramo `OperationalDirectorPlanMaterializerV0 -> WorkflowTaskStore ->
   WorkflowTaskWaitAgentRefsV0 -> WorkflowTaskWaitStateV0 ->
   ContinueAppDirectorV0 -> DrainRunV0 con ingesta acotada por WaitAgentRefs`
   esta verificado offline. El cierre desde `ContinueAppDirectorV0` solo es real
   cuando el loop queda quiescent y hay `OperationalClosureSource` inyectado con
   refs/evidencias causales. `run_required_tests` ya valida
   `RequiredTestEvidenceV0` causal desde el state y entrega esas refs al cierre;
   no ejecuta comandos shell por si mismo. El siguiente foco esta en
   `docs/corte_cierre_generico_director_operativo_2026-05-17.md`:
   runner/adaptador real de tests, rama negativa de review/replan,
   `replan_or_close`/`close` y replay/idempotencia, siempre como cierre causal
   offline antes de Codex/OPES real.

## Verificacion minima

Para cambios transversales:

```bash
git diff --check
go test -count=1 ./...
```

Para cambios de frontera del nucleo, la prueba raiz
`TestNeutralOrchestrationPackagesDoNotImportProductAdapters` debe seguir verde.
