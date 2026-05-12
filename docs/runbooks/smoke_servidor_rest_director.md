# Smoke real: servidor Orquesta + REST director + stats

Objetivo: ejecutar una prueba repetible y real que arranca Orquesta como
servidor REST, envia una solicitud `AppSpecRequestV0` de app pequena completa,
deja que el director cree agentes Codex en paralelo, consulta
`/api/v0/director/stats` varias veces y para limpio.

## Comando

Desde la raiz del repo:

```bash
./scripts/smoke_orquesta_server_rest_director.sh
```

El script compila un binario temporal de `cmd/orquesta-server`, arranca el
servidor en `127.0.0.1:0`, lee el puerto real desde el statefile temporal,
envia `POST /api/v0/apps/director`, extrae `run_ref` y hace tres polls a
`POST /api/v0/director/stats` con progreso, referencias de proceso y uso de
agentes activados. Antes de terminar exige observar al menos dos agentes
arrancados y stats enriquecidas con `process`, `progress` y `usage`.

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
ORQUESTA_SMOKE_MIN_PARALLEL_AGENTS=2
ORQUESTA_KEEP_SMOKE_DIR=0
```

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
- al menos una linea `stats_verificadas=process_refs,progress,usage,parallel_agents`;
- parada limpia al salir del script.

Si un endpoint devuelve no-2xx, el script imprime el JSON de respuesta en
stderr y sale con error. Si los polls no llegan a mostrar agentes paralelos con
referencias de proceso, progreso y uso, tambien falla. El `trap` de salida envia
`SIGINT` al servidor y, si no termina, escala a `SIGTERM`.

## No ejecutar como test unitario

Este smoke es opt-in y real: puede arrancar agentes externos mediante Codex. No
debe meterse en `go test` ni en CI rapida sin una decision explicita.
