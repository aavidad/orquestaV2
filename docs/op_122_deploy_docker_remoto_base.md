# OP-122 — Base de despliegue Docker remoto por SSH

Fecha: 2026-03-25  
Responsable: Codex2  
Tarea: `#385`

## Alcance implementado

Primera iteración funcional del despliegue opcional por Docker remoto:

- contrato `deployapp.DockerRemoteSpec`
- planificación de despliegue remoto con estrategias:
  - `docker_save`
  - `registry`
- pasos explícitos de:
  - build
  - preparación remota
  - transferencia
  - despliegue
  - healthcheck
  - rollback opcional
- ejecución controlada con:
  - `dry-run`
  - `auto-rollback`
- CLI:
  - `orquesta deploy docker-remoto plan`
  - `orquesta deploy docker-remoto ejecutar`
- API:
  - `POST /api/deploy/docker-remoto/plan`
  - `POST /api/deploy/docker-remoto/ejecutar`

## Límites de esta iteración

- no integra todavía gestión centralizada de secretos
- no persiste aún evidencias de despliegue en tablas propias
- no añade UI web
- no hace descubrimiento automático del target; el destino se pasa por flags o JSON

## Verificación

- `go test ./deployapp`
- `go test ./cmd -run 'TestAPIDeployDockerRemotePlanAndExecuteDryRun'`
- `go build ./cmd ./deployapp`
