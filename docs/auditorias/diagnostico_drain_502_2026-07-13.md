# Diagnóstico independiente — 502 en drain idempotente

Fecha: 2026-07-13  
Test: `TestDrainRunV0IgnoraArtefactoYaRegistradoPorLoopGestionado`

## Resultado

El 502 comunicado no fue reproducible con el código observado:

- comando exacto: verde;
- 20 procesos consecutivos: verdes;
- `-count=200`: verde;
- `-race -count=20`: verde;
- vecinos de reingesta/cleanup: dos pasadas verdes;
- paquete `orquesta-app-codex-stack` completo: verde.

El hash del test es idéntico en `cf172b87c9` y `9deae783a8`; no hay diff del
paquete entre esos commits. Por tanto, la evidencia disponible no demuestra una
rotura determinista de la idempotencia.

## Camino causal del 502

El POST de prueba a `/nueva-app` llega a
`orquesta-web/nueva_app_endpoint_post_page_v0.go`. La página devuelve 502 cuando
`RESTArrancarDirectorAppClientV0.ArrancarDirectorApp` devuelve error de
transporte. El fixture usa un cliente HTTP in-process con timeout de un segundo.
Bajo carga concurrente, superar ese deadline se proyecta como 502 aunque no haya
fallado el contrato de artefacto duplicado.

## Decisión

No modificar la lógica productiva de idempotencia sin una reproducción causal.
Tampoco maquillar el assertion ni añadir retries. Si vuelve a fallar, capturar el
error interno del cliente y la duración para distinguir timeout de error del
caso de uso. Un endurecimiento posterior del harness puede separar el test de
idempotencia de un test dirigido de timeout, pero no se acredita como arreglo de
producto mientras no exista el rojo causal.

