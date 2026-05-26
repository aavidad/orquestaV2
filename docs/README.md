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
- [matriz_pruebas_reales_y_smoke_2026-05-17.md](matriz_pruebas_reales_y_smoke_2026-05-17.md):
  evidencias, smokes offline/opt-in y pendientes verificables.

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
- [runbooks/smoke_opes_plan_temario_operadores_2026-05-18.md](runbooks/smoke_opes_plan_temario_operadores_2026-05-18.md):
  smoke acotado de `plan_temario` contra OPES temporal.

## Backlog vivo

- [autoprogramacion_orquesta_pendientes_2026-05-23.md](autoprogramacion_orquesta_pendientes_2026-05-23.md):
  backlog ejecutable y shard canonico durante la migracion documental.
- [rail_errors_observados_2026-05-23.md](rail_errors_observados_2026-05-23.md):
  rail errors observados.
- [duplicaciones_railes_pendientes_2026-05-24.md](duplicaciones_railes_pendientes_2026-05-24.md):
  duplicaciones, rails y fuentes locales federadas.

## Regla para documentos historicos

Los documentos de vision, analisis y diseno previo permanecen como contexto. No
deben abrir trabajo ni cerrar evidencia por si solos si contradicen la foto
vigente; antes de usarlos como plan hay que enlazarlos a `estado_actual`,
`guia_nucleo`, la matriz o el backlog vivo.

## Validacion transversal

Para cambios transversales, la validacion minima sigue siendo:

```bash
git diff --check
go test -count=1 ./...
```
