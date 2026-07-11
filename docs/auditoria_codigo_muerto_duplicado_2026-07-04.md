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

## Actualización Codex — 2026-07-04 tarde

TAREA-9.4 queda implementada como herramienta reproducible:

- Nuevo `scripts/orquesta_auditoria_codigo.sh`.
- Salidas: JSON y SQLite local como caché/reporte derivado, no verdad
  operativa.
- Métricas: candidatos `deadcode`, módulos huérfanos, familias de helpers
  copiadas y ficheros no-test de más de 800 líneas.
- `scripts/orquesta_smoke_nightly.sh` ejecuta la auditoría por defecto y aplica
  ratchet contra el último nightly verde con métricas de auditoría: falla si
  suben `deadcode_candidates` o `helper_duplicate_definitions`.
- Tests: `scripts/test_orquesta_auditoria_codigo.sh` y
  `scripts/test_orquesta_smoke_nightly.sh`.

Incidencia cerrada durante la integración:

- `BUG-ORQ-20260704-182`: la primera versión caminaba `**/*.go` desde la raíz
  y podía quedarse demasiado tiempo en workspaces grandes. Cierre: recorrido
  acotado desde `cmd/` y `modulos/`.

Medición fresca con `deadcode` instalado en GOPATH tras la primera ola segura:

- `deadcode_source=deadcode_tool`.
- `deadcode_candidates=1230`.
- `helper_duplicate_definitions=288`.
- `orphan_modules=1`.
- `large_files_over_800=17`.
- `modulos/orquesta-deploy=87`, frente a 94 en el snapshot histórico de Claude.

Nota de comparación: el snapshot inicial de Claude marcaba 1188 candidatos, pero
el árbol actual ya incluye cambios posteriores y el comando reproducible fresco
devuelve 1230. Para futuros ratchets usar la métrica fresca de
`scripts/orquesta_auditoria_codigo.sh`, no comparar directamente contra el
snapshot histórico.

Primera ola segura aplicada:

- Eliminados siete helpers `Has*IssueV0` sin consumidores internos en
  `modulos/orquesta-deploy`.
- No se borra `orquesta-deploy` completo: sigue importado por
  `orquesta-app-planner`.
- No se borra `orquesta-work-profiles`: queda como placeholder histórico y
  requiere decisión documental separada si se reabre.

Segunda ola segura aplicada:

- `modulos/orquesta-capacity`: inlinados helpers privados de un solo uso en
  `capacity_policy_v0.go` y `model_escalation_policy_helpers_v0.go`.
- No se tocan APIs exportadas (`DefaultCapacityPolicyV0`, Decode/Validate,
  contratos de decision) ni helpers con consumidores internos.
- Medición fresca posterior:
  `deadcode_candidates=1228`, `helper_duplicate_definitions=288`,
  `orphan_modules=1`, `large_files_over_800=17`,
  `modulos/orquesta-capacity=89`.

Correccion de ratchet aplicada despues de la segunda ola:

- `BUG-ORQ-20260704-184`: el auditor ya no depende solo de `PATH`; localiza
  `deadcode` en `PATH`, `GOBIN` o `GOPATH/bin`, evitando caer al snapshot
  historico cuando la herramienta Go esta instalada de forma canonica.
- `BUG-ORQ-20260704-185`: el nightly rechaza auditorias con schema invalido,
  metricas requeridas ausentes/no enteras o fuente `snapshot_file`,
  `unavailable`/`deadcode_tool_failed`.
- Medicion viva posterior al fix:
  `deadcode_source=deadcode_tool`, `deadcode_candidates=1228`,
  `helper_duplicate_definitions=288`, `orphan_modules=1`,
  `large_files_over_800=17`.

## Actualizacion Codex 2026-07-11: ola privada verificable

La auditoria viva previa dio `deadcode_candidates=1202`, 30 privados sin refs
textuales, 409 exportados/contratos por revisar, 40 privados usados solo por
tests, 8 modulos sin importadores internos, 307 helpers duplicados y 22
ficheros grandes. Estos conteos sustituyen los snapshots 1188/1228 para el
arbol actual; no equivalen a bugs ni autorizan borrados masivos.

Tres commits eliminan solo los 30 privados sin ninguna referencia:
`65616e06a` (10 wrappers, 77 lineas), `9009d8321` (17 wrappers, 100 lineas) y
`6d1781812` (3 helpers de modulos, 18 lineas). Tras la ola y la nueva config
canonica: `deadcode_candidates=1173`, `helper_duplicate_definitions=307`,
`orphan_modules=8`, `large_files_over_800=22`, `functions_indexed=25490`.
Evidencia derivada local:
`/tmp/orquesta-code-audit-20260711-r2/current.json`.

Los ocho modulos sin importadores no se borran: son adaptadores/tools opt-in
recientes, incluido SQL de referencia declarado en AGENTS. Los exportados,
contratos y privados de test quedan fuera de esta ola. Duplicados y ficheros
grandes son deuda de mantenibilidad, no prueba de codigo muerto.
