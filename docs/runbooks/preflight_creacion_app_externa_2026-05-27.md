# Preflight creacion app externa 2026-05-27

Objetivo: fijar que necesita Orquesta para iniciar una prueba real de creacion
de una app externa que no es Orquesta ni OPES.

## Estado bloqueante

No hay bloqueos conocidos para una prueba controlada de app externa nueva.

- Servidor local verificado: `/healthz` responde `ok`.
- Cola residente verificada: `orquesta.autoprogramming.status.v0` devuelve
  `queue.count=0`.
- No hay agentes Codex vivos fuera del servidor residente durante este corte.
- El bucle de reconciliacion documental repetida queda corregido: una tarea con
  intento base, retry y reconciliacion cerrada por ACK estructurado no vuelve a
  generar otra reconciliacion equivalente.

## Superficies necesarias

- Intake/AppSpec: `POST /api/v0/apps/spec`.
- Arranque de director: `POST /api/v0/apps/director`.
- Cola global de runs: run queue residente, con cierre terminal propagado.
- Runtime Codex: adaptador externo por composicion, no en el nucleo.
- Estado operativo: `autoprogramming/status`, `director/stats`, `/ops` y
  control de run por puertos/adaptadores.
- Cierre: review gate, evidencias de tests requeridos y cierre causal por
  `ContinueAppDirectorV0` cuando hay fuente de cierre inyectada.

## Evidencia ejecutada

```sh
go test -count=1 \
  ./modulos/orquesta-factory \
  ./modulos/orquesta-factory-http \
  ./modulos/orquesta-app-gateway \
  ./modulos/orquesta-web \
  ./modulos/orquesta-app-director-intake \
  ./modulos/orquesta-app-director-service \
  ./modulos/orquesta-app-codex-stack \
  ./modulos/orquesta-server \
  ./cmd/orquesta-server
```

Resultado: OK.

Smoke API real de AppSpec contra el servidor residente:

```sh
POST /api/v0/apps/spec
```

Resultado: HTTP 200 con `AppSpecV0` valido y `BacklogInicialPropuestoV0`.

## No bloqueante para esta prueba

- OPES temporal real hasta `generate_html_site -> local_html_site`: pertenece al
  conector OPES, no a una app externa generica.
- Smoke manual largo de shutdown cooperativo Codex real: importante para
  operaciones, no prerequisito de crear una app nueva controlada.
- Politica productiva fina de rechazo/replan por entregas invalidas: el review
  gate y los reworks ya existen; la politica final puede endurecerse despues.

## Regla de prueba

La prueba de app externa debe arrancarse por el servidor Orquesta/API, no por
scripts ni CLI que lancen agentes fuera del control residente.

Antes de arrancar:

1. Confirmar `/healthz`.
2. Confirmar `queue.count=0`.
3. Confirmar sin procesos Codex vivos fuera del PID del servidor.
4. Usar una app pequena con alcance cerrado y verificable.
5. Vigilar cola, agentes, ACKs y cierre desde los endpoints publicos.
