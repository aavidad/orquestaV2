# Auditoria de codigo muerto revalidada - 2026-07-10

## Evidencia

Ejecutada localmente con `scripts/orquesta_auditoria_codigo.sh`, salida aislada
no versionada en `/tmp/orquesta-dead-audit-20260710/`. Fuente de candidatos:
`deadcode_tool`; no hubo lineas sin parsear.

| Metrica | Valor |
| --- | ---: |
| candidatos `deadcode` | 1.249 |
| familias de helpers duplicados | 307 |
| ficheros de mas de 800 lineas | 23 |
| modulos sin importador | 5 |

Los modulos con mas candidatos son runtime (96), web (96), orchestration-core
(94), capacity (89), deploy (87), server (79), MCP (69) y persistence (65).
Los ficheros mas grandes son `nueva_app_html_render_v0.go` (2.315 lineas),
`autoprogramming_status_stale_running_v0.go` (1.763) y dos piezas de stack
Codex de mas de 1.600 lineas.

## Lectura correcta

El conteo no equivale a ficheros borrables: `deadcode` no ve entradas por
reflexion, comandos, ensamblado, plugins ni contratos de adaptadores. Los cinco
modulos sin importador incluyen CSV/JSON/PDF-tool, file-store de tools y SQL de
referencia: son adaptadores opt-in nuevos o de referencia, no basura probada.
No se borro codigo en este corte.

La primera muestra manual confirmo tres falsos positivos del analizador:
`normalizeGuardianConfigV0`, `DefaultGatewayHTTPTransportPolicyV0` y
`EvaluateRunQueueWorksetConcurrencyV0` solo aparecen como inalcanzables porque
sus consumidores son pruebas/contratos publicos. No se retiran wrappers ni sus
tests sin decidir primero si forman parte de API soportada.

## Cola gobernada

1. Clasificar por modulo los candidatos con entrada solo por wiring/CLI antes
   de retirar simbolos.
2. Retirar unicamente familias sin referencias con `rg`, tests focales y
   ratchet que baje en el mismo commit.
3. Trocear los 23 hubs por responsabilidad, no por numero de lineas.
4. Mantener adaptadores opt-in sin importador como `retain_pending_composition`
   hasta que tengan consumidor o una decision de retirada documentada.

La ola Orquesta `wave-dead-code-audit-20260710` no entrego clasificacion y fue
parada por presupuesto de diagnostico; esta auditoria manual es un desbloqueo
documentado, no una aceptacion de esa ola.
