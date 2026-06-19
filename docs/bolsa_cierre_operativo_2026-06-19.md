# Cierre operativo Bolsa - 2026-06-19

## Estado

Bolsa queda cerrada como modulo prototipo operativo funcional para la linea VEC.
No se declara producto final de administracion electronica: quedan pendientes
endurecimiento productivo, persistencia duradera, Kerberos real, firma,
registro, notificacion fehaciente y adaptadores oficiales.

URL local vigente:

```text
http://127.0.0.1:18081/
```

Proceso vivo esperado:

```text
/tmp/bolsa-server-real
```

## Evidencia Orquesta

Ultima ola de cierre:

```text
run_ref: bolsa-vec-audit-candidateid-20260619-001
estado: cerrada
tareas: 3/3 cerradas
agentes: 3 lanzados, 3 entregados, 0 fallidos
reviews: 3 aceptadas
closures: 1
blockers: 0
estado: /tmp/orquesta-bolsa-audit-candidateid-20260619-001/results/status.json
```

Ola previa de notificaciones:

```text
run_ref: bolsa-vec-notif-actions-20260619-001
estado: cerrada
tareas: 4/4 cerradas
agentes: 4 lanzados, 4 entregados, 0 fallidos
reviews: 4 aceptadas
closures: 1
blockers: 0
```

El rework final existio porque la primera ola cerro usando
`/api/audit?scope=candidate:{id}` como flujo principal aunque la especificacion
pedia `GET /api/audit?candidate_id={id}`. La ola final corrigio el adaptador
HTTP y el smoke para que `candidate_id` sea el contrato publico principal y
`scope` quede solo como compatibilidad tecnica.

## Validacion ejecutada

En `/home/alberto/Trabajo/Bolsa_Diputacion_app`:

```bash
GOCACHE=/tmp/orquesta-bolsa-real-gocache-audit-candidateid go test -count=1 ./...
node --check web/static/app.js
python3 -m json.tool locales/es.json
```

Smoke HTTP real contra `http://127.0.0.1:18081/`:

- `GET /healthz`
- `GET /api`
- `GET /api/portal`
- `GET /api/modules/bolsa/manifest`
- `POST /api/candidates`
- `POST /api/candidates/{id}/merits`
- `POST /api/candidates/{id}/baremo`
- `GET /api/candidates/{id}/expediente`
- `POST /api/candidates/{id}/documents`
- `POST /api/candidates/{id}/claims`
- `POST /api/candidates/{id}/notifications`
- `POST /api/notifications`
- `GET /api/notifications?candidate_id={id}`
- `POST /api/notifications/{id}/send`
- `POST /api/notifications/{id}/read`
- `GET /api/audit?candidate_id={id}`
- `GET /api/audit?scope=candidate:{id}` como compatibilidad
- `GET /api/audit` devuelve `400` controlado si falta scope/candidate_id

Resultado funcional observado:

```text
baremo_total: 2.4
notificaciones: Creada -> Enviada -> Leida
auditoria candidate_id: document.registered, claim.presented,
notification.created, notification.sent, notification.read
```

Smoke visual:

```text
/home/alberto/Trabajo/bolsa_smoke_artifacts/bolsa-real-18081.png
```

La captura muestra la web cargada con workspace administrativo, navegacion
lateral, panel operativo, tabla y detalle.

## Comprobacion de concurrencia antes de documentar

Se reviso que no quedaran `orquesta-server`, runs temporales ni agentes Codex de
Orquesta vivos tocando Bolsa. Solo queda el servidor real de Bolsa.

`lsof +D /home/alberto/Trabajo/Bolsa_Diputacion_app` mostro ademas una sesion
Codex antigua con `cwd` en Bolsa (`bash/node/codex`, iniciada a las 15:56), sin
ficheros abiertos de escritura ni modificaciones recientes en la app. Por esa
razon este cierre documenta el estado en Orquesta y no realiza nuevas ediciones
en la app Bolsa.

## Declaracion de cierre

Bolsa esta terminada y operativa como modulo prototipo funcional integrado en la
linea VEC. Queda pendiente el endurecimiento productivo para entorno real.
