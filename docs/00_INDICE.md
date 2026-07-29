> Estado documental: histórico — índice del runtime Orquesta pre-rebuild.
>
> No es autoridad de producto, backlog ejecutable ni evidencia de cierre para
> Orquesta V2. La entrada obligatoria V2 es `AGENTS.md` →
> `docs/reconstruccion/LEEME_AGENTE_ORQUESTAV2.md` → `product/roadmap.json` →
> `product/capabilities.json` y `product/evidence/` →
> `docs/reconstruccion/ruta_total_100.md`. Para el estado operativo, consultar
> `docs/reconstruccion/HANDOFF_PARADA_ORQUESTAV2_2026-07-26.md`.

# Indice de documentacion - Orquesta

Este indice es una puerta de entrada. La foto vigente no vive aqui: se resuelve
por el orden de autoridad documental indicado abajo.

## Autoridad vigente

1. [AGENTS.md](../AGENTS.md) y
   [estado_actual_2026-05-17.md](estado_actual_2026-05-17.md) fijan la frontera
   conceptual: Orquesta es nucleo reutilizable de orquestacion; Codex, OPES,
   web, MCP y CLI son composiciones o adaptadores.
2. [guia_nucleo_orquestacion_2026-05-17.md](guia_nucleo_orquestacion_2026-05-17.md),
   [corte_cierre_generico_director_operativo_2026-05-17.md](corte_cierre_generico_director_operativo_2026-05-17.md)
   y [matriz_pruebas_reales_y_smoke_2026-05-17.md](matriz_pruebas_reales_y_smoke_2026-05-17.md)
   fijan piezas, handoffs, evidencias y smokes.
3. [autoprogramacion_orquesta_pendientes_2026-05-23.md](autoprogramacion_orquesta_pendientes_2026-05-23.md)
   es backlog ejecutable y tambien indice vivo de shards durante la migracion.
4. Los `AGENTS.md` y docs locales de cada modulo gobiernan solo su alcance y
   quedan subordinados a las fuentes anteriores si estan stale.
5. Los documentos historicos sirven como contexto, no como plan operativo, salvo
   que enlacen a una fuente vigente.

## Mapa conceptual

| Capa | Owner principal | Lectura inicial |
| --- | --- | --- |
| Core puro | `modulos/orquesta-core-workflow` | [guia nucleo](guia_nucleo_orquestacion_2026-05-17.md), `modulos/orquesta-core-workflow/AGENTS.md` |
| Loop de aplicacion | `modulos/orquesta-orchestration-core` | [director operativo](director_operativo_v1_2026-05-17.md), [cierre generico](corte_cierre_generico_director_operativo_2026-05-17.md) |
| Director Operativo | `modulos/orquesta-director-operativo`, `modulos/orquesta-app-director-service` | [director operativo](director_operativo_v1_2026-05-17.md), [corte director tarde](corte_director_funcionando_tarde_2026-05-17.md) |
| Ciclo neutral V2 | `modulos/orquesta-director-cycle`, `modulos/orquesta-director-runner`, `modulos/orquesta-director-scheduler`, `modulos/orquesta-director-cycle-outbox` | [guia nucleo](guia_nucleo_orquestacion_2026-05-17.md), [matriz](matriz_pruebas_reales_y_smoke_2026-05-17.md) |
| Goal-first | `modulos/orquesta-goal`, `modulos/orquesta-runtime-codex-goal` | [goal-first Codex](orquesta_goal_first_codex_2026-06-25.md) |
| Trabajo externo neutral | `modulos/orquesta-domain-work`, `modulos/orquesta-app-change`, `modulos/orquesta-external-work-run` | [principio director](principio_orquesta_piensa_director.md), [guia nucleo](guia_nucleo_orquestacion_2026-05-17.md) |
| Runtime y composicion | `modulos/orquesta-runtime*`, `modulos/orquesta-app-codex-stack`, `cmd/orquesta-server`, `modulos/orquesta-server` | [matriz](matriz_pruebas_reales_y_smoke_2026-05-17.md), runbooks en `docs/runbooks/` |
| Web/MCP/API | `modulos/orquesta-web`, `modulos/orquesta-mcp`, `cmd/orquesta-server` | [backlog autoprogramacion](autoprogramacion_orquesta_pendientes_2026-05-23.md), runbooks de API/web/MCP |
| Ola autoprogramacion 10x6 | `docs`, write-set autorizado por Director | [mapa 10x6](mapa_autoprogramacion_capacidad_10x6_2026-05-26.md), [backlog autoprogramacion](autoprogramacion_orquesta_pendientes_2026-05-23.md) |
| OPES como consumidor | `modulos/orquesta-opes-*` | [flujo temario operativo](opes_flujo_temario_operativo_2026-06-02.md), [corte OPES](corte_opes_como_consumidor_orquesta_2026-05-18.md), [runbook plan_temario](runbooks/smoke_opes_plan_temario_operadores_2026-05-18.md) |

## Ruta de lectura para agentes

- Cambio transversal: lee las cinco fuentes de autoridad de arriba y despues el
  `AGENTS.md` local del modulo.
- Director, waits, review, replan o cierre: anade
  [director_operativo_v1_2026-05-17.md](director_operativo_v1_2026-05-17.md),
  [corte_director_funcionando_tarde_2026-05-17.md](corte_director_funcionando_tarde_2026-05-17.md)
  y [corte_cierre_generico_director_operativo_2026-05-17.md](corte_cierre_generico_director_operativo_2026-05-17.md).
- OPES, bridge o `domain_work` aplicado a OPES: anade
  [opes_flujo_temario_operativo_2026-06-02.md](opes_flujo_temario_operativo_2026-06-02.md),
  [corte_opes_como_consumidor_orquesta_2026-05-18.md](corte_opes_como_consumidor_orquesta_2026-05-18.md)
  y su runbook temporal.
- Smokes reales, runtime, shutdown o proveedor: usa la matriz y el runbook
  especifico antes de ejecutar nada opt-in.

## Backlog y shards

- [autoprogramacion_orquesta_pendientes_2026-05-23.md](autoprogramacion_orquesta_pendientes_2026-05-23.md):
  backlog canonico y shard historico principal.
- [rail_errors_observados_2026-05-23.md](rail_errors_observados_2026-05-23.md):
  errores de rails observados.
- [duplicaciones_railes_pendientes_2026-05-24.md](duplicaciones_railes_pendientes_2026-05-24.md):
  matriz de duplicaciones, rails y fuentes locales.

## Documentos vigentes frecuentes

- [README raiz](../README.md)
- [README docs](README.md)
- [principio_orquesta_piensa_director.md](principio_orquesta_piensa_director.md)
- [corte_supervisor_codex_director_2026-05-18.md](corte_supervisor_codex_director_2026-05-18.md)
- [corte_tests_requeridos_y_smoke_programacion_2026-05-21.md](corte_tests_requeridos_y_smoke_programacion_2026-05-21.md)
- [corte_required_tests_failed_replan_2026-05-21.md](corte_required_tests_failed_replan_2026-05-21.md)
- [corte_replan_negativo_followups_split_2026-05-21.md](corte_replan_negativo_followups_split_2026-05-21.md)
- [corte_plan_state_director_decisions_2026-05-21.md](corte_plan_state_director_decisions_2026-05-21.md)
- [informe_operativo_cierre_autoprogramacion_2026-05-25.md](informe_operativo_cierre_autoprogramacion_2026-05-25.md)
- [orquesta_goal_first_codex_2026-06-25.md](orquesta_goal_first_codex_2026-06-25.md)

## Politicas y disenos reutilizables

- [politica_seleccion_lenguaje_es.md](politica_seleccion_lenguaje_es.md) |
  [EN](politica_seleccion_lenguaje_en.md)
- [politica_arquitectura_tipos_proyecto_es.md](politica_arquitectura_tipos_proyecto_es.md) |
  [EN](politica_arquitectura_tipos_proyecto_en.md)
- [gestion_reglas_skills_workflows_es.md](gestion_reglas_skills_workflows_es.md) |
  [EN](gestion_reglas_skills_workflows_en.md)
- [politica_acceso_persistencia_es.md](politica_acceso_persistencia_es.md) |
  [EN](politica_acceso_persistencia_en.md)
- [politica_backups_es.md](politica_backups_es.md) |
  [EN](politica_backups_en.md)
- [politica_supervision_residente_event_driven.md](politica_supervision_residente_event_driven.md)
- [politica_multilenguaje_por_defecto.md](politica_multilenguaje_por_defecto.md)

## Contexto historico

Estos documentos conservan contexto de producto, diagnostico o diseno previo.
Antes de usarlos como evidencia o plan, enlazalos a una fuente vigente.

Historicos/stale con sustituto vigente:

- [BIBLIA_APP_ORQUESTA.md](BIBLIA_APP_ORQUESTA.md): Orquesta V1 y vision
  previa. Marcado internamente como `doc_estado=historico-stale`; sustituido
  para trabajo actual por `AGENTS.md`, `README.md`, `estado_actual`,
  `guia_nucleo` y backlog vivo. Aunque conserve secciones historicas llamadas
  "fuentes de verdad" o "doctrina", esas secciones no gobiernan trabajo vigente
  sin enlace nuevo a la autoridad documental anterior.
- [orquesta_v1_vision.md](orquesta_v1_vision.md)
- [orquesta_v1_roadmap.md](orquesta_v1_roadmap.md)
- [uso_actual_app_orquesta.md](uso_actual_app_orquesta.md)
- [operacion_agentes_manuales.md](operacion_agentes_manuales.md)
- [informe_revision_hexagonal.md](informe_revision_hexagonal.md)
- [informe_ecosistema_orquestacion_2026-03-29.md](informe_ecosistema_orquestacion_2026-03-29.md)
- [analisis_repos_control_agentes_2026-03-23.md](analisis_repos_control_agentes_2026-03-23.md)
- [unificacion_ramas_2026-05-11.md](unificacion_ramas_2026-05-11.md)

Forenses:

- [rail_errors_observados_2026-05-23.md](rail_errors_observados_2026-05-23.md):
  evidencia y diagnostico de rails observados; no sustituye al backlog vivo.
- [duplicaciones_railes_pendientes_2026-05-24.md](duplicaciones_railes_pendientes_2026-05-24.md):
  matriz de duplicaciones y priorizacion; no sustituye al backlog vivo.

Marcadores detectables para T88/T116: `doc_estado=historico-stale`,
`doc_no_canonico=true`, `doc_no_ejecutable_sin_fuente_vigente=true` y
`doc_excluir_planificacion_automatica=true` obligan al indice federado a tratar
la fuente como historica aunque el cuerpo conserve texto V1 con palabras como
"canonico", "doctrina" o "fuente de verdad".

## Plantillas

- [plantillas_documentacion/README_es.md](plantillas_documentacion/README_es.md)
- [plantillas_documentacion/manual_usuario_es.md](plantillas_documentacion/manual_usuario_es.md)
- [plantillas_documentacion/manual_desarrollador_es.md](plantillas_documentacion/manual_desarrollador_es.md)
- [plantillas_documentacion/manual_sysadmin_es.md](plantillas_documentacion/manual_sysadmin_es.md)
- [plantillas_i18n/README_es.md](plantillas_i18n/README_es.md)
