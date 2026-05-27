# Plan de microtareas: self-observability

Estado: historico/debug. Este documento conserva la primera entrega accionable
del director en modo debug para auditoria, pero no gobierna requisitos vivos del
repo Orquesta ni obliga a crear documentos raiz vacios.

Vigencia 2026-05-26: los nombres `manual_usuario`, `manual_desarrollador`,
`manual_sistemas_deploy`, `decisiones`, `pruebas_documentales` y `pendientes`
son artefactos esperados de una app generada o modificada por Orquesta. Si una
composicion los materializa, debe hacerlo dentro del proyecto objetivo o como
refs de artefacto de ese proyecto, no como nuevos documentos obligatorios del
nucleo Orquesta.

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

## Contrato de artefactos generado

Este plan usa rutas `docs/...` como rutas relativas del proyecto generado. Para
el repo Orquesta vigente, la fuente reutilizable de esas piezas vive en
`docs/plantillas_documentacion/`:

- `manual_usuario`: manual funcional de la app generada.
- `manual_desarrollador`: arquitectura, contratos y desarrollo de la app
  generada.
- `manual_sistemas_deploy`: alias historico compuesto por `manual_sysadmin` y
  `guia_despliegue` cuando aplique operacion/despliegue.
- `pruebas_documentales`: evidencia documental de validaciones de la app
  generada.
- `pendientes`: backlog publico y verificable de la app generada.

## Microtareas Propuestas Historicas

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

Pruebas historicas del proyecto generado:

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

Pruebas historicas del proyecto generado:

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

Pruebas historicas del proyecto generado:

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
sin evidencias reales. Esta afirmacion pertenece al proyecto generado de aquel
corte debug; no declara requisitos vivos para la documentacion raiz de Orquesta.
