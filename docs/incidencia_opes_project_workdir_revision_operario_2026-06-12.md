# Incidencia Orquesta: OPES revision ejecutada con ProjectWorkDir incorrecto

Fecha: 2026-06-12  
Origen: revisión del curso OPES `ope-operario` desde `/home/alberto/Trabajo/OPES`.

## Resumen

La primera ola de revisión se lanzó contra la instancia Orquesta ya activa en
`127.0.0.1:8787`. Los seis runs fueron aceptados y los agentes entregaron
informes, pero el `ProjectWorkDir` efectivo de Codex era
`/home/alberto/Trabajo/orquesta`. Como consecuencia, varios informes no pudieron
ver el material real de OPES y emitieron conclusiones incompletas, por ejemplo
`no publicado` o `0 MP3`, aunque el curso Operario tenía evidencia local de
staging, audio y publicación.

La segunda ola se corrigió arrancando una instancia aislada con:

```text
ORQUESTA_CODEX_PROJECT_WORKDIR=/home/alberto/Trabajo/OPES
ORQUESTA_CODEX_RUNTIME_WORKDIR=/home/alberto/Trabajo/OPES/.orquesta-runtime-operario-review-20260612
ORQUESTA_SERVER_ADDR=127.0.0.1:8791
```

Con esa configuración los seis informes reales se completaron correctamente.

## Tarea de mejora

Crear una protección de Orquesta para trabajos OPES:

- Si `project_ref=opes` o `app_ref=opes`, el runtime debe exigir que el
  `ProjectWorkDir` del agente sea `/home/alberto/Trabajo/OPES` o un workspace
  OPES declarado.
- Si el servidor activo no cumple esa condición, `/api/v0/external-work/run`
  debe rechazar el trabajo con una issue clara, no aceptar y producir una
  auditoría parcial.
- La respuesta pública debe indicar el `project_work_dir` efectivo de forma
  opaca/segura para que el director detecte una mala configuración antes de
  consumir tokens.
- Los informes de ACK deberían distinguir `completed_with_scope_limitation`
  cuando las rutas `main_paths` no son accesibles desde el workspace.

## Reparación aplicada

Estado: `resuelta_en_codigo`.

Se ha añadido un guard reutilizable en `orquesta-app-codex-stack` para
`/api/v0/external-work/run`. Si el trabajo declara `project_ref=opes`,
`project-ref-opes`, `app_ref=opes` o `external_work.project_ref=opes`, Orquesta
compara el `ProjectWorkDir` efectivo del servidor con el directorio OPES
obligatorio. Por defecto exige `/home/alberto/Trabajo/OPES`; puede ajustarse con
`ORQUESTA_OPES_PROJECT_WORKDIR`.

Cuando no coincide, el run queda rechazado con:

```text
code=external_work_project_work_dir_mismatch
field=project_work_dir:evidence-ref-opes-project-workdir-required
```

Pruebas ejecutadas:

```bash
go test ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server \
  -run 'TestExternalWorkRunProjectWorkDirGuard|TestExternalWorkRunProjectWorkDirGuardConfig|TestCodexStackV0ExternalWorkRun' \
  -count=1
```

Resultado: OK.

Pendiente no bloqueante: añadir un estado de ACK
`completed_with_scope_limitation` para trabajos que arrancan correctamente pero
no pueden abrir rutas declaradas en `main_paths`.

## Evidencia

- Primera ola: informes en OPES `opes-salidas/coordinacion_temarios/revision_operario_orquesta_2026-06-12/resultados/`.
- Segunda ola válida: informes en OPES `opes-salidas/coordinacion_temarios/revision_operario_orquesta_2026-06-12/resultados_reales/`.
- Diferencia operativa: la segunda ola usó `ORQUESTA_CODEX_PROJECT_WORKDIR=/home/alberto/Trabajo/OPES`.
