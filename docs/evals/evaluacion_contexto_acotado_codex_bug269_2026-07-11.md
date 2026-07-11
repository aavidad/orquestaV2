# Evaluacion de contexto acotado Codex: BUG-269

Fecha: 2026-07-11  
Modelo: `gpt-5.6-terra`, effort `high`  
Ejecucion: cuatro `codex exec` reales en worktrees separados y sesiones tmux;
sin subagentes, commits, push, remoto, OPES ni codebase-memory-mcp.

## Metodo

Los cuatro agentes resolvieron el mismo bug real: dos goals estaban
`complete/accepted` y sus runs/cola `closed`, pero el batch persistia
`goals_running` tras restart. Se aumento progresivamente el contexto:

- L1: sintomas, invariantes y cuatro ficheros objetivo;
- L2: L1 mas traza causal, contratos y documentos locales;
- L3: L2 mas alternativas arquitectonicas y E2E de restart;
- L4: contexto amplio, inventario/incidencia y exploracion libre.

Los contadores son los publicados por `codex exec --json`. `input` incluye
cache; `nuevo` es `input - cached_input`. El tiempo parte del mismo segundo de
creacion de las cuatro sesiones y termina al escribir su `rc`.

| Nivel | Tiempo | Comandos | Input | Cached | Nuevo | Output | Resultado |
|---|---:|---:|---:|---:|---:|---:|---|
| L1 | 267 s | 21 | 2.069.060 | 1.941.248 | 127.812 | 9.809 | 3 ficheros; focal y replay verdes |
| L2 | 334 s | 24 | 3.673.803 | 3.481.856 | 191.947 | 11.737 | 4 ficheros mas inventario; modulo y E2E verdes |
| L3 | 411 s | 59 | 3.094.451 | 2.941.952 | 152.499 | 15.977 | restart real de stores/cola; modulo y E2E verdes |
| L4 | 324 s | 20 | 3.201.358 | 3.040.512 | 160.846 | 13.349 | 5 ficheros; observer y recovery modificados |

## Revision de calidad

- L1 encontro la arquitectura correcta con el menor coste, pero su test
  simulaba restart copiando el stack y no reabria adaptadores durables.
- L2 anadio tolerancia a runs retiradas y limite de lectura, pero el limite
  sobre una lista mezclada puede postergar un `closed` antiguo.
- L3 produjo la mejor evidencia: reabre estado, run store, cola y control; no
  carga integracion/gate sobre el timeout del observador.
- L4 leyo mas contexto que L1 sin mejorar coste ni alcance seguro. Ejecutar la
  reconciliacion desde observacion normal y observacion activa amplia el camino
  critico y duplica puntos de disparo.

## Decision

El punto dulce para este frente es un paquete equivalente a L1 enriquecido con
un criterio E2E concreto de L3: sintomas y evidencia, invariantes, 4-8 simbolos
o ficheros iniciales, contratos locales y un test de restart/replay. No entregar
el repositorio/documentacion completa salvo que la primera exploracion pruebe
que falta una frontera.

Se integra una combinacion revisada: recovery desde cola durable en el tick,
`RunStore` por puerto, filtro de app, cancelacion cooperativa, tolerancia a run
retirada y E2E que reabre adaptadores. Se descarta modificar el observador y se
evita un limite que pueda causar starvation. La prueba demuestra que contexto
minimo sin criterio de integracion puede ahorrar tokens pero empeorar evidencia;
contexto completo tampoco garantiza una solucion mejor.
