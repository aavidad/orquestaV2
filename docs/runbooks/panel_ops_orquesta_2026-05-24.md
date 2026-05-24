# Panel Ops Orquesta 2026-05-24

## URL

- Panel live: `http://127.0.0.1:8787/ops`
- Estado servidor: `GET /api/v0/server/status`
- Recursos servidor: `GET /api/v0/server/resources`
- Cola/autoprogramacion: `POST /api/v0/autoprogramming/status`
- Stats por run: `POST /api/v0/director/stats`

## Datos que muestra

- Estado del servidor, heartbeat, ticks y ejecuciones del supervisor.
- Estado de automejora e idle self-improvement.
- Memoria del proceso, goroutines y disco de `project_work_dir` y `runtime_work_dir`.
- Numero de proyectos activos inferido desde `app_ref/project_ref`.
- Numero de tareas en cola y validacion publica derivada de estado/evidencias.
- Runs/proyectos con progreso, fase actual, agentes activos y bloqueos.
- Agentes por run con estado, tarea, senal de progreso y porcentaje operativo.

## Notas

- El panel refresca cada segundo desde el navegador.
- No accede a DB ni filesystem desde la web: consume APIs publicas del servidor.
- El porcentaje por agente se deriva del estado/progreso disponible cuando el
  runtime no publica un porcentaje numerico propio.
- El listado de proyectos activos no tiene endpoint dedicado todavia: se infiere
  desde cola y stats por run. Si hace falta una fuente canonica, debe entrar como
  puerto/adaptador nuevo, no en el nucleo.

## Verificacion focal

```bash
go test -count=1 ./modulos/orquesta-server
go test -count=1 ./modulos/orquesta-http-gateway
go test -count=1 ./modulos/orquesta-web
go test -count=1 ./modulos/orquesta-app-gateway
```
