# Inventario vivo de bugs e incidencias Orquesta

Fecha de apertura: 2026-06-30.

Este documento no sustituye a las incidencias detalladas. Es el indice comun
para analizarlas como conjunto y detectar deuda de arquitectura. Cada bug nuevo
observado durante Orquesta, OPES, web, autoprogramacion, MCP, runtime remoto o
conectores debe quedar aqui como fila o enlazado desde aqui.

Regla operativa:

- No tratar un bug como "caso aislado" hasta clasificar si repite patron.
- Si se arregla con codigo, mantener la fila y marcar evidencia de cierre.
- Si solo se documenta, dejar estado `abierto` o `en investigacion`.
- Si el bug viene de OPES u otro consumidor, distinguir dominio consumidor de
  fallo del nucleo Orquesta.
- No abrir nuevos frentes de programacion solo por existir una fila: primero
  decidir si bloquea el cierre actual o queda como backlog agrupado.

## Ejes de arquitectura a vigilar

| Eje | Sintoma repetido | Riesgo arquitectonico |
| --- | --- | --- |
| Goal-first vs loop legacy | rutas que caen al director antiguo, modos implicitos, tests que no fijan opt-in | migracion incompleta y doble fuente de verdad del ciclo |
| Estado vivo/proyecciones | running_stale, procesos vivos no contados, markers goal missing, ACK sin cierre | estado repartido entre filesystem, stores, observadores y colas |
| Runtime Codex | stdio/proxy/tmux confundidos, timeouts, bwrap, identidad git remota | adaptadores mezclados con politica operativa y entorno |
| OPES/DomainWork | artefactos regenerables tratados como canonicos, RAG/audio/HTML con falsos verdes | contrato de dominio demasiado implicito y validadores divergentes |
| Subagentes/causalidad | padres escriben tabla de subroles sin materializar hijos reales | contrato causal no forzado por runtime/ACK/write-set |
| Web Nueva App | opciones visibles no llegan al contrato, ayuda incompleta, errores en idioma equivocado | catalogos UI, i18n, factory y docs duplican verdad |
| Remoto/produccion aislada | bundle stale, sin GitHub directo, worktrees divergentes | operacion remota no tiene protocolo unico de sync y handoff |

## Bugs inventariados

| ID | Estado | Area | Sintoma | Hipotesis arquitectonica | Evidencia / enlace | Accion |
| --- | --- | --- | --- | --- | --- | --- |
| BUG-ORQ-20260630-001 | cerrado | Goal-first/autoprogramacion | `goal_first_state_missing` o estado goal no recuperado desde marker durable | estado goal y marker durable no estaban reconciliados como fuente de verdad | commits previos `3da7d88b`, `f9e330f1` | Mantener tests de status y no reabrir salvo regresion |
| BUG-ORQ-20260630-002 | cerrado | Runtime Codex | confusion entre `stdio`, `app_server_proxy` y `app_server_tmux` | contrato de backend operativo no estaba separado de diagnostico/proxy historico | docs/runbooks y commits `729e6695`, `ece98200`, `00ef13ac` | Canon: `app_server_tmux`; stdio no backend normal |
| BUG-ORQ-20260630-003 | cerrado | Self-programming remoto | bridge externo `disabled` seguia suprimiendo automejora | estado persistido de dominio interpretaba componente historico como sesion activa | commit remoto/local `e91fe0ebe` reportado por agente | Mantener test de `external_bridge_status=disabled` |
| BUG-ORQ-20260630-004 | cerrado | Readiness | backend Goal tmux degradado no bloqueaba readiness de forma publica | readiness no agregaba diagnosticos de backend goal como gate operativo | commit `00ef13ac` | Mantener readiness bloqueante sin filtrar rutas/tokens |
| BUG-ORQ-20260630-005 | cerrado | Autoprogramacion | fallo de launch Goal dejaba run huerfano o sin state bloqueado | launch y persistencia de `GoalWorkState` no eran transaccionales desde el punto de vista operacional | commit `97e134fd` | Persistir state invalid/reparable con reason accionable |
| BUG-ORQ-20260630-006 | abierto | Supervision/autoprogramacion | `/autoprogramming/status` marcaba `running_stale` aunque habia procesos vivos | observadores de procesos, goals y runs no comparten modelo unico de estado vivo | incidencia OPES `TAREA_OPES_ORQUESTA_EXTERNAL_WORK_VALIDACION_Y_STREAM_2026-06-26.md` | Revisar como conjunto con estados/ACK/procesos |
| BUG-ORQ-20260630-007 | cerrado | Supervision HTTP | `/autoprogramming/supervise` despachaba agentes pero el cliente HTTP quedaba colgado | endpoints de control mezclaban accion larga, streaming/ack y respuesta sin contrato temporal claro | `modulos/orquesta-mcp/run_supervisor_http_v0.go`, `modulos/orquesta-mcp/autoprogramming_supervise_http_v0.go`, `cmd/orquesta-server/autoprogramming_supervise_http_flow_v0_test.go` | Cierre: ambos bridges devuelven `202 accepted_background`, `operation_ref`, dedupe de operacion activa y acciones de polling; tests de cliente real cubren cuerpo sin colgar |
| BUG-ORQ-20260630-008 | abierto | Subagentes/OPES | padres OPES no materializaban 6 subagentes reales aunque declaraban subroles | contrato causal de hijos no se valida en cierre padre/write-set | `docs/incidencia_opes_external_work_no_materializa_6_subagentes_por_padre_2026-06-23.md`, `AGENTS.md` regla subroles | Mantener bloqueo si faltan `child_task_refs`/ACK/delivery |
| BUG-ORQ-20260630-009 | abierto | Write-set/agentes | `agent_packet` estrecha write-set e impide consolidar Markdown canonico | permisos de escritura no distinguen borrador, evidencia y canon consolidado | reportes OPES 2026-06-26/27 | Revisar contrato de write-set por rol y fase |
| BUG-ORQ-20260630-010 | cerrado | OPES RAG | corrector ortografico procesaba `10_tutor_rag/corpus` regenerable | pipeline no marcaba artefactos regenerables vs canonicos | `/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/INCIDENCIA_OPES_CORRECTOR_PROCESA_RAG_REGENERABLE_2026-06-29.md` | Regla: no validar/corregir corpus regenerable como fuente canonica |
| BUG-ORQ-20260630-011 | cerrado | OPES finalpkg | RAG suelto `rag/chunks.jsonl` podia dar falso verde frente a `rag/corpus` canonico | validador y contrato de paquete final divergian | commits `f7859da0`, `7009e7bb`, `9180e70f` | Validar `rag/corpus/{chunks,summary}` y manifest refs |
| BUG-ORQ-20260630-012 | cerrado | OPES audio/finalpkg | `index.html`/portadas/listados contaban como manifiestos o fuentes de audio de tema | contrato de paginas tematicas no estaba codificado en validador | commits `c56d9b79`, `7009e7bb`, `9180e70f` | `audio/manifests/**` solo contra `html_final/tema_*.html` y `html_ampliado/tema_*.html` |
| BUG-ORQ-20260630-013 | cerrado | Nueva App UI/i18n | navegador mostraba `Please fill out this field` y backend exponia mensajes tecnicos/rutas | validacion HTML nativa y viewmodels no estaban localizados/sanitizados | commits `5f3e90e0`, `ffe8c7ba` | Validacion publica localizada; no filtrar HOME/tokens/proveedor |
| BUG-ORQ-20260630-014 | cerrado | Nueva App contrato | opciones expertas visibles no llegaban a factory/docs o se perdian filas 5-6 | catalogos UI, wizard, factory y guia eran fuentes duplicadas | commits `31929dc6`, `8266fd61`, `11165fb2` | Tests de catalogo visible e i18n |
| BUG-ORQ-20260630-015 | cerrado | MCP/external-work | faltaba cobertura explicita de que `goal_first` o modo no-legacy no activa legacy | migracion goal-first dependia de tests indirectos | patch remoto `orquesta-external-work-goal-first-explicit-a7940f5e-2026-06-30.patch` | Integrar/verificar commit remoto `a7940f5e` |
| BUG-ORQ-20260630-016 | cerrado | Operacion remota | agente remoto trabajaba sobre bundle stale y sin GitHub directo | protocolo de sync remoto no era canonico ni visible para agentes | bundle `orquesta-trabajo-plataforma-agentes-9180e70f.bundle`, nota `REMOTE_SYNC_AFTER_9180E70F_2026-06-30.md` | Usar bundles versionados y worktree fresco antes de editar |
| BUG-ORQ-20260630-017 | cerrado | Operacion remota | clone fresco no tenia identidad Git para commitear | bootstrap remoto no normalizaba config local de Git | observado en ciclo remoto `a7940f5e` | Config local `Orquesta Codex <orquesta-codex@local>` en clones frescos |

## Lectura de conjunto inicial

Los bugs no parecen solo errores sueltos. Se agrupan en cuatro problemas de
diseno operativo:

1. Migracion incompleta de ciclo: Goal-first, director legacy, supervisor,
   status y MCP siguen coexistiendo. Cada ruta debe declarar si lanza Goal,
   observa Goal o exige opt-in legacy.
2. Estado repartido: proceso vivo, run, goal marker, ACK, delivery, outbox y
   registro OPES se proyectan desde stores distintos. Los falsos `stale` y los
   cierres incompletos nacen de ahi.
3. Contratos de dominio no ejecutables: OPES y Nueva App tenian reglas visibles
   en docs/UI que no siempre estaban en validadores/tests.
4. Operacion remota no canonica: sin GitHub directo, bundles, tmux, identidad
   Git y worktrees deben ser parte del contrato de trabajo, no memoria manual.

## Pendientes de analisis agrupado

- Unificar diagnostico de estado vivo: goals, procesos, runs, ACK y deliveries.
- Revisar endpoints largos: separar submit/ack/observe de operaciones que
  pueden colgar HTTP.
- Auditar todos los validadores OPES contra artefactos canonicos vs
  regenerables.
- Revisar write-set y consolidacion canonica para padres/subagentes.
- Documentar protocolo remoto unico: bundle, checkout, identidad Git, patch,
  summary y no tocar produccion.
