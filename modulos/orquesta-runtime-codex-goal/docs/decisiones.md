# Decisiones

## D-001 Codex Goal dirige dentro

El prompt generado dice de forma explicita que Codex Goal es el Director
operativo interno. Orquesta no le pide simular el loop historico.

## D-002 Orquesta gobierna fuera

El prompt exige entregar evidencias, tests y artefactos segun el spec. La
aceptacion final queda fuera del runtime.

## D-003 sin arranque real por defecto

El paquete no arranca Codex por si mismo. La composicion debe inyectar un puerto
real cuando el entorno soporte Codex Goal.

El wiring local disponible vive en `cmd/orquesta-server` y solo se activa con
`ORQUESTA_CODEX_GOAL_BACKEND=app_server_proxy`. Usa `codex app-server proxy`
contra daemon local ya disponible; no convierte `codex exec` en sustituto de
Goal.

## D-004 observacion por puerto

La observacion de Codex Goal tambien entra por puerto inyectado. El adaptador
solo normaliza y valida refs/status antes de devolver `GoalWorkResultV0`; no
acepta cierre ni consulta herramientas concretas.

## D-005 resultado terminal estructurado

El prompt exige que la respuesta final termine con
`ORQUESTA_GOAL_RESULT_V0 { ... }`. La composicion `cmd/orquesta-server` puede
leer ese marcador desde `thread/read` y convertirlo en refs opacas de
artefactos, tests, receipts y evidencias. Un `complete` sin marcador o sin refs
requeridas queda como resultado observable, no como cierre aceptado.
