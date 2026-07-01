# Incidencia: backlog narrativo lanzado como autoprogramacion ejecutable

Fecha: 2026-07-02

## Resumen

El agente remoto de autoprogramacion genero cambios tras interpretar secciones
narrativas del backlog como tareas ejecutables. El parser aceptaba cualquier
heading que empezara por `## T`, por lo que encabezados como `## Tareas...` o
`## Trazas...` podian entrar en el flujo de self-improvement. Ademas, la rama
`backlog-scan` podia aceptarse antes de validar que sus `context_refs` apuntaban
a secciones ejecutables permitidas.

## Impacto

- Orquesta podia abrir goals de autoprogramacion desde contexto documental no
  accionable.
- El sistema consumia cuota en tareas sin contrato de cierre claro.
- Los estados posteriores aparecian como `blocked` por falta de resultado de
  goal, aunque la causa inicial era una mala frontera entre documentacion y
  backlog ejecutable.

## Hipotesis arquitectonica

No es solo un bug puntual de expresion regular. La frontera entre inventario
documental y cola ejecutable debe ser explicita: una composicion puede escanear
documentos, pero solo debe materializar tareas con identificador ejecutable
canonico y seccion admitida por filtro.

## Mitigacion aplicada

- `cmd/orquesta-server` solo reconoce headings ejecutables con forma
  `## T<num>`.
- `backlog-scan` ya no salta el filtro de `context_refs` de seccion.
- Se anade cobertura focal para secciones narrativas, scanner filtrado y
  proyecciones publicas de espera/proceso externo.

## Verificacion

- `go test -count=1 ./cmd/orquesta-server -run 'TestIdleSelfImprovementBacklogPlannerV0Ignora|TestServerStackIdleSelfImprovementFiltra'`
- `go test -count=1 ./modulos/orquesta-server -run 'TestSupervisorProjectionV0Distingue'`
- `go test -count=1 ./modulos/orquesta-runtime-codex -run 'TestCodexWrapperV0RetiraStartupLockObsoletoVacio|TestCodexWrapperV0StaleLock'`
