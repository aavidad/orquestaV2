# Plan de microtareas: self-observability

Estado: primera entrega accionable del director en modo debug. El plan abre la
ruta hacia programacion documental, pero no emite revision, validacion final ni
cierre.

## Workflow

1. `brainstorming_arquitectura`: contexto inicial recibido.
2. `votacion_y_decision`: solicitar voto sobre arquitectura documental inicial.
3. `planificacion_microtareas`: aceptar decision y publicar contrato funcional.
4. `programacion`: lanzar microtareas de documentacion minima.
5. `revision`: pendiente hasta que existan entregas reales.
6. `validacion_final`: pendiente.
7. `cierre`: pendiente.

## Contrato Funcional

`contract-ref-selfobs-docs-v0`

Funciones cubiertas:

- `DocumentAppManuals`: manual de usuario y manual de desarrollador.
- `DocumentSystemsDeploy`: manual de sistemas y despliegue.
- `DocumentDecisionsTestsPending`: decisiones, pruebas documentales y
  pendientes.

## Microtareas Propuestas

### task-selfobs-doc-manuals-v0

Objetivo: redactar manual de usuario y manual de desarrollador.

Write-set:

- `docs/manual_usuario.md`
- `docs/manual_desarrollador.md`

Criterios:

- `objetivo_actual: documentar_app debug high app web API MCP self-observability arquitectura inicial manual_usuario manual_desarrollador manual_sistemas_deploy decisiones pruebas_documentales pendientes`
- Manual de usuario con flujos visibles y estados esperados.
- Manual de desarrollador con hexagono, puertos, DTOs e i18n.
- CONSULTA AL DIRECTOR si falta alcance funcional concreto.

Pruebas requeridas:

- `test -s docs/manual_usuario.md`
- `test -s docs/manual_desarrollador.md`

### task-selfobs-doc-systems-v0

Objetivo: redactar manual de sistemas y despliegue.

Write-set:

- `docs/manual_sistemas_deploy.md`

Criterios:

- `objetivo_actual: documentar_app debug high app web API MCP self-observability arquitectura inicial manual_usuario manual_desarrollador manual_sistemas_deploy decisiones pruebas_documentales pendientes`
- Arranque opt-in documentado.
- Configuracion y conectores descritos sin fijar persistencia concreta.
- Operacion, seguridad y recuperacion descritas a nivel documental.

Pruebas requeridas:

- `test -s docs/manual_sistemas_deploy.md`

### task-selfobs-doc-governance-v0

Objetivo: redactar decisiones, pruebas documentales y pendientes.

Write-set:

- `docs/decisiones.md`
- `docs/pruebas_documentales.md`
- `docs/pendientes.md`

Criterios:

- `objetivo_actual: documentar_app debug high app web API MCP self-observability arquitectura inicial manual_usuario manual_desarrollador manual_sistemas_deploy decisiones pruebas_documentales pendientes`
- Decisiones con motivacion y alcance.
- Pruebas documentales verificables.
- Pendientes separados por producto, arquitectura, pruebas, seguridad y
  revision final.

Pruebas requeridas:

- `test -s docs/decisiones.md`
- `test -s docs/pruebas_documentales.md`
- `test -s docs/pendientes.md`

## Seguridad y Calidad

- No exponer datos sensibles.
- No incluir transcripciones completas.
- Mantener evidencia compacta.
- Revisar textos visibles por i18n antes de cierre productivo.
- No abrir validacion final hasta tener entregas, revision aceptada y cierre de
  tareas.

## Omisiones Debug

No se omiten los minimos documentales en este plan: manual_usuario,
manual_desarrollador, manual_sistemas_deploy, decisiones,
pruebas_documentales y pendientes quedan cubiertos como trabajo planificado. Lo
omitido por modo debug es el cierre productivo y la ejecucion de revision final
sin evidencias reales.
