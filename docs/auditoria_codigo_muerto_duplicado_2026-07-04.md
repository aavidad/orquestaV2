# Auditoría de código muerto/duplicado/obsoleto — 2026-07-04

Autor: Claude (director). Método: herramientas, no lectura manual —
`deadcode` oficial (golang.org/x/tools) sobre los entrypoints de cmd/,
grafo de imports con `go list`, y consultas agregadas guardadas en SQLite
(scratchpad `auditoria_codigo.sqlite`) para no saturar contexto.
Listado completo de deadcode: `docs/auditoria_codigo_deadcode_2026-07-04.txt`.

## Hallazgos

1. **1.188 funciones inalcanzables desde los binarios** (candidatas a
   muertas; el análisis parte de los main de cmd/, así que una función
   usada solo por tests o por consumidores externos sale como inalcanzable:
   son CANDIDATAS, cada borrado exige verificación). Concentración:
   orquesta-orchestration-core (125), orquesta-runtime (97),
   orquesta-deploy (94), orquesta-capacity (91), orquesta-web (89),
   orquesta-mcp (68). Que deploy y capacity casi enteros sean inalcanzables
   sugiere subsistemas legacy completos sin uso desde los binarios — cuadra
   con las 4 generaciones de director que señaló la pericial.
2. **Módulo huérfano**: `modulos/orquesta-work-profiles` — 0 importadores.
   Decidir: borrar o documentar por qué existe.
3. **Helpers copiados entre módulos**: hay al menos un símbolo definido en
   81 de los 83 módulos y familias de helpers privados (compact*/
   firstNonEmpty*/contains*) copiadas en decenas de módulos. Es el patrón
   "cada módulo se copia sus utilidades": coherente con no compartir todo,
   pero 81 copias de lo mismo es deuda de mantenimiento.
4. **15 ficheros >800 líneas** (no test) — candidatos a partir.
5. **NO es deuda**: `contracts_v0.go`/`errors_v0.go`/`public_error_catalog_
   v0.go` repetidos por módulo son el patrón hexagonal intencional (cada
   módulo declara su contrato); no consolidar.
6. Solo 12 marcadores TODO/FIXME — la deuda no está anotada, está callada:
   por eso hace falta herramienta, no lectura.

## Cómo trabajar esto sin leer todo el código (y sin perder perspectiva)

- El MAPA lo dan: informe pericial + AGENTS.md por módulo + docs/contratos
  de cada módulo + metricas de deuda. La HERRAMIENTA baja al detalle:
  deadcode/go list/rg → SQLite → se lee a mano SOLO el punto señalado.
- Esta auditoría debe SER DE ORQUESTA (orden del operador): ver TAREA-9.4.

## Entregado a Codex: TAREA-9 (en instrucciones del director)
