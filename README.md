# Orquesta

Orquesta es el nucleo de orquestacion de agentes del workspace. La direccion
vigente es que el nucleo sea reutilizable por aplicaciones externas: programacion
con Codex y OPES son composiciones consumidoras, no la definicion del nucleo.

## Lectura inicial

- `AGENTS.md`: reglas para futuros agentes que trabajen en este repo.
- `docs/estado_actual_2026-05-17.md`: foto vigente del proyecto.
- `docs/guia_nucleo_orquestacion_2026-05-17.md`: mapa de piezas contra el
  nuevo nucleo.
- `docs/principio_orquesta_piensa_director.md`: principio de reparto entre
  Orquesta y apps de dominio.
- `docs/matriz_pruebas_reales_y_smoke_2026-05-17.md`: matriz de pruebas reales,
  opt-in y offline.

## Autoridad documental

La foto vigente vive en `AGENTS.md` y `docs/estado_actual_2026-05-17.md`. La
matriz de smokes decide que casos estan cerrados con evidencia; el backlog de
autoprogramacion enumera trabajo ejecutable y no debe relanzar
`CODEX-WAVE-REAL` ni `CODEX-RECURSION-REAL` salvo regresion demostrada. Los
documentos historicos deben citar una fuente vigente antes de usarse para
planificar.

## Validacion rapida

```bash
git diff --check
go test -count=1 ./...
```

## Frontera vigente

El core puro vive en `modulos/orquesta-core-workflow` y el loop de aplicacion en
`modulos/orquesta-orchestration-core`. Los adaptadores concretos de runtime,
persistencia, deploy, web, MCP, OPES y Codex deben quedar fuera de esas capas.
`modulos/orquesta-deploy` es el owner de `DeploymentPlan v0`: prepara contratos
de despliegue declarativos y dry-run por puerto, sin ejecutar infraestructura ni
tocar secretos.

La espina `DirectorCycleStepV0 -> director-runner -> director-scheduler ->
core-workflow -> director-cycle-outbox` define el tick neutral del Director V2:
prepara input compacto, planifica comandos publicos, aplica workflow por puerto
y deja outbox pendiente para un supervisor externo. No es daemon ni composicion
residente; esos cierres viven en el servidor/adaptadores y requieren evidencia
propia.

`modulos/orquesta-domain-work-sql` es solo un adaptador SQL externo de referencia
para jobs de dominio. No es persistencia global de Orquesta, no abre conexiones,
no registra drivers y no esta cableado al servidor productivo.
