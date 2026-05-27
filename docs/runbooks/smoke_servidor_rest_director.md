# Smoke real: servidor Orquesta + REST director + stats

Objetivo: ejecutar una prueba repetible y real que arranca Orquesta como
servidor REST, envia una solicitud `AppSpecRequestV0` de app pequena completa,
arranca la entrada real del director, consulta `/api/v0/director/stats` varias
veces y para limpio.

## Comando

Desde la raiz del repo:

```bash
./scripts/smoke_orquesta_server_rest_director.sh
```

El script compila un binario temporal de `cmd/orquesta-server`, arranca el
servidor en `127.0.0.1:0`, lee el puerto real desde el statefile temporal,
envia `POST /api/v0/apps/director`, extrae `run_ref` y hace tres polls a
`POST /api/v0/director/stats` con progreso, referencias de proceso y uso de
agentes activados. Antes de terminar exige observar al menos un agente arrancado
con stats enriquecidas con `process`, `progress` y `usage`.

Este smoke mide el arranque REST directo. Ese endpoint usa la entrada
`start-only`: crea el run y el agente director inicial. Las olas paralelas de
trabajo deben validarse en un smoke separado del supervisor/drain, no como
precondicion de este endpoint.

## Precondiciones

- `go`, `curl` y `python3` disponibles en `PATH`.
- `codex` disponible si la prueba debe lanzar runtime real de agentes.
- Ejecutar desde un working tree donde se permita que Orquesta cree artefactos
  de runtime temporales. Por defecto se usan directorios bajo `/tmp`.

## Payload cubierto

La solicitud pide una app pequena completa:

- Go API REST;
- web HTML minima;
- persistencia detras de conector/puerto;
- SQLite permitido solo como requisito local de la app generada;
- Orquesta no recibe ni hardcodea proveedor de DB.

El envelope REST usa el contrato real:

- `POST /api/v0/apps/director`;
- campo `app_spec_request`;
- `request_kind=crear_app_completa`;
- `execution_mode=normal`;
- limites acotados para que la prueba no quede abierta indefinidamente.

## Controles

Variables utiles:

```bash
ORQUESTA_SMOKE_STATS_POLLS=3
ORQUESTA_SMOKE_STATS_SLEEP_SECONDS=2
ORQUESTA_SMOKE_REQUEST_TIMEOUT_SECONDS=45
ORQUESTA_SMOKE_STATS_TIMEOUT_SECONDS=10
ORQUESTA_SMOKE_MIN_STARTED_AGENTS=1
ORQUESTA_KEEP_SMOKE_DIR=0
```

Para que las stats incluyan uso real observado desde Codex, el servidor debe
arrancar con opt-in explicito y el runtime debe producir un reporte redactado:

```bash
ORQUESTA_CODEX_USAGE_ACCOUNTING=redacted_report
ORQUESTA_CODEX_USAGE_LOG_MAX_BYTES=65536
```

Ese adaptador solo publica contadores y estado de cuota redactados cuando se
consulta stats con `include_agent_usage=true`; no persiste prompts, transcripts,
cuentas reales, HOME ni payloads de proveedor.
Lee solo `codex_usage_accounting.json` en el runtime del agente con
campos redactados como `usage.input_tokens`, `usage.output_tokens` y
`quota.status`; si el reporte trae proveedor, modelo, coste, cuenta, HOME,
tokens o payload crudo, se descarta entero. `stdout`, `stderr` y
`codex_last_message.txt` no son fuente de uso.
Si la fuente opt-in esta activa pero el reporte falta, esta vacio o se rechaza
por no redactado, el agente queda con `quota_status=unknown` y evidencia compacta
`quota_observed_unavailable`; no se publica `not_configured` ni se bloquea el
director.
Las consultas web/API que no envian ese flag mantienen solo progreso, refs de
proceso y agentes observados; el uso queda ausente aunque haya fuente opt-in.
Si una superficie pide `include_agent_usage=true` y la composicion no inyecto
fuente de uso, la respuesta debe conservar stats basicas y anadir issue publico
`agent_usage_source_not_configured`, sin fabricar coste, proveedor, modelo ni
cuenta.

`ORQUESTA_SMOKE_MIN_PARALLEL_AGENTS` se conserva como alias compatible para
subir el umbral manualmente, pero el valor por defecto del smoke REST directo es
un agente arrancado.

Para conservar logs y payloads temporales en caso de investigacion:

```bash
ORQUESTA_KEEP_SMOKE_DIR=1 ./scripts/smoke_orquesta_server_rest_director.sh
```

## Criterio de exito

La salida esperada contiene:

- `servidor listo: http://127.0.0.1:<puerto>`;
- `POST /api/v0/apps/director -> HTTP 2xx`;
- `run_ref=<valor>`;
- tres lineas `POST /api/v0/director/stats poll=N -> HTTP 2xx`;
- lineas compactas con `control_registered`, `no_signal`, `progress_source` y
  `usage_agents`;
- al menos una linea `stats_verificadas=process_refs,progress,usage,director_start`;
- parada limpia al salir del script.

Si un endpoint devuelve no-2xx, el script imprime el JSON de respuesta en
stderr y sale con error. Si los polls no llegan a mostrar los agentes exigidos
con referencias de proceso, progreso y uso, tambien falla. El `trap` de salida
envia `SIGINT` al servidor y, si no termina, escala a `SIGTERM`.

## No ejecutar como test unitario

Este smoke es opt-in y real: puede arrancar agentes externos mediante Codex. No
debe meterse en `go test` ni en CI rapida sin una decision explicita.
