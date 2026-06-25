<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Documentacion de Orquesta

Orquesta se documenta como nucleo reutilizable de orquestacion de agentes para
apps externas. Programacion con Codex, OPES, web, CLI y MCP son composiciones o
adaptadores, no la definicion del nucleo.

## Entrada recomendada

- [00_INDICE.md](00_INDICE.md): indice de lectura, mapa conceptual y backlog.
- [../AGENTS.md](../AGENTS.md): reglas operativas para agentes.
- [estado_actual_2026-05-17.md](estado_actual_2026-05-17.md): foto vigente y
  orden de autoridad documental.
- [guia_nucleo_orquestacion_2026-05-17.md](guia_nucleo_orquestacion_2026-05-17.md):
  mapa operativo de piezas e invariantes.
- [principio_orquesta_piensa_director.md](principio_orquesta_piensa_director.md):
  reparto entre juicio de Orquesta y dominio externo.
- [uso_actual_app_orquesta.md](uso_actual_app_orquesta.md): uso operativo
  server-first de la composicion actual, rutas `/api/v0/*`, web/CLI como
  clientes finos y cuarentena de rutas legacy.
- [matriz_pruebas_reales_y_smoke_2026-05-17.md](matriz_pruebas_reales_y_smoke_2026-05-17.md):
  evidencias, smokes offline/opt-in y pendientes verificables.
- [orquesta_goal_first_codex_2026-06-25.md](orquesta_goal_first_codex_2026-06-25.md):
  corte goal-first: Codex Goal como Director operativo interno y Orquesta como
  plano de gobierno, contexto, contratos y cierre.

## Handoffs vigentes

- [director_operativo_v1_2026-05-17.md](director_operativo_v1_2026-05-17.md):
  contrato del Director Operativo V1.
- [corte_director_funcionando_tarde_2026-05-17.md](corte_director_funcionando_tarde_2026-05-17.md):
  handoff de waits, review/rework y recursion gobernada.
- [corte_cierre_generico_director_operativo_2026-05-17.md](corte_cierre_generico_director_operativo_2026-05-17.md):
  cierre causal, tests durables, replan/close y plan state.
- [mapa_autoprogramacion_capacidad_10x6_2026-05-26.md](mapa_autoprogramacion_capacidad_10x6_2026-05-26.md):
  mapa operativo para dividir una ola opt-in de autoprogramacion con hasta 10
  agentes padre y 6 subagentes por padre.
- [corte_opes_como_consumidor_orquesta_2026-05-18.md](corte_opes_como_consumidor_orquesta_2026-05-18.md):
  OPES como consumidor por conectores.
- [orquesta_goal_first_codex_2026-06-25.md](orquesta_goal_first_codex_2026-06-25.md):
  adelgazamiento del loop residente cuando el runtime soporte goals persistentes.
- [runbooks/smoke_opes_plan_temario_operadores_2026-05-18.md](runbooks/smoke_opes_plan_temario_operadores_2026-05-18.md):
  smoke acotado de `plan_temario` contra OPES temporal.

## Backlog vivo

- [autoprogramacion_orquesta_pendientes_2026-05-23.md](autoprogramacion_orquesta_pendientes_2026-05-23.md):
  backlog ejecutable y shard canonico durante la migracion documental.
- [rail_errors_observados_2026-05-23.md](rail_errors_observados_2026-05-23.md):
  rail errors observados.
- [duplicaciones_railes_pendientes_2026-05-24.md](duplicaciones_railes_pendientes_2026-05-24.md):
  duplicaciones, rails y fuentes locales federadas.

## Clasificacion documental

- Vigentes: `AGENTS.md`, `README.md`, `00_INDICE.md`, `estado_actual`,
  `guia_nucleo`, la matriz de smokes y los handoffs de corte.
- Historicos/stale: vision, analisis y diseno previo, incluido
  `BIBLIA_APP_ORQUESTA.md`, que se conserva como contexto de Orquesta V1 y debe
  apuntar a sustitutos vigentes desde su cabecera. Si su cuerpo conserva
  secciones V1 que se llamen "doctrina", "canonico" o "fuente de verdad", esas
  frases quedan subordinadas a `doc_estado=historico-stale`.
- Forenses: rail errors, duplicaciones y evidencias de scanner. Sirven para
  explicar por que existe una tarea, no para sustituir el backlog ejecutable.
- Plantillas: documentos reutilizables para generar documentacion; no describen
  estado del repo.

Ningun documento historico o forense debe abrir trabajo ni cerrar evidencia por
si solo si contradice la foto vigente; antes de usarlo como plan hay que
enlazarlo a `estado_actual`, `guia_nucleo`, la matriz o el backlog vivo.

Los manuales de uso con `./orquesta serve`, rutas `/api/*` sin version,
OpenClaw, AP-077 como requisito operativo o snapshots DBV1 se consideran
historicos salvo que esten sincronizados con
`uso_actual_app_orquesta.md` y una ruta `/api/v0/*` vigente.
T119 queda cerrado aqui como regla documental: clientes nuevos deben partir del
manual server-first, y cualquier manual V1 conservado solo aporta trazabilidad o
rescate explicito.

Para T88/T116, los marcadores `doc_no_canonico=true` y
`doc_no_ejecutable_sin_fuente_vigente=true` pesan mas que cualquier texto
historico que se autodenomine canonico dentro del documento.

## Validacion transversal

Para cambios transversales, la validacion minima sigue siendo:

```bash
git diff --check
go test -count=1 ./...
```
