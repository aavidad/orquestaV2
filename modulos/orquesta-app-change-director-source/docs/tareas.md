# Tareas

## ACDS-007

Objetivo: evitar que OPES 1+6 complete subroles utiles sin padre capaz de
consolidar el producto canonico por estrechamiento de write-set.

Estado: hecho.

Incidencia: un padre OPES con seis subroles recibia solo `base/coordinacion`.
Cuando los subroles generaban material valido bajo `base/subroles/<rol>`, el
padre no podia escribir el producto canonico bajo `base` aunque OPES hubiese
autorizado ese `allowed_write_set`.

Solucion:

- conservar el write-set de producto en el padre integrador;
- mantener los hijos acotados a `base/subroles/<rol>`;
- ordenar la materializacion como seis hijos y padre posterior;
- declarar `depends_on` del padre hacia los seis hijos para evitar solape vivo
  de write-sets.

Validacion:

- `TestAppChangeDirectorDecisionSourceV0MaterializaSeisSubrolesOPES`;
- `TestAppChangeDirectorDecisionSourceV0OPESSubrolesPadreConservaWriteSetProductoAutorizado`;
- `go test -count=1 ./modulos/orquesta-app-change-director-source`;
- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'ExternalWorkRunOPESSubroles'`.

## ACDS-006

Objetivo: evitar que un `external_work` aceptado quede bloqueado o invisible
antes de crear microtarea por `required_tests` largos o por criterios
estructurados fuera de `acceptance_criteria`.

Estado: hecho.

Incidencia: OPES lanzo trabajos de ampliacion de temas Tractorista AP con
`external_work/run`; las runs quedaron en `programacion`, con
`director_questions=1`, `tasks_total=0` y blocker
`app-director-decision-source-error`. La causa fue transportar un comando
Python largo como `required_tests` de la microtarea del director. Ademas, se
detecto que un contrato externo accionable sin criterios explicitos podia
quedar aceptado por transporte pero sin microtarea.

Solucion:

- compactar `required_tests` de microtarea al limite del DTO del director;
- sustituir comandos inline largos por el marcador verificable
  `validar required_tests externos declarados en paquete de dominio`;
- permitir autoplanning de `external_work` accionable aunque no traiga
  `acceptance_criteria`, usando criterios derivados del contrato externo;
- conservar la regla de que cambios internos sin criterios explicitos no crean
  microtarea.

Validacion:

- `TestAppChangeDirectorDecisionSourceV0ExpansionOPESConRequiredTestLargoNoBloquea`;
- `TestAppChangeDirectorDecisionSourceV0ExternalWorkAccionableSinCriteriosCreaMicrotarea`;
- `TestAppChangeDirectorDecisionSourceV0NoInventaMicrotareaSinCriterios`;
- `go test -count=1 ./modulos/orquesta-app-change-director-source`;
- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'ExternalWorkRun|AppChangeConProgramacionPendiente' -v`.

## ACDS-005

Objetivo: proyectar `AppChangeRequestV0.metadata_refs` como `context_refs`
opacas en la microtarea del director.

Estado: hecho.

Validacion:

- `TestAppChangeDirectorDecisionSourceV0ProyectaMetadataRefsComoContextRefs`;
- `go test -count=1 ./modulos/orquesta-app-change-director-source`.

## ACDS-001

Objetivo: generar decisiones del director desde `AppChangeRecordV0` concreto.

Estado: hecho.

Validacion:

- `go test -count=1 ./modulos/orquesta-app-change-director-source`.
- `TestAppChangeDirectorDecisionSourceV0ConsumeCambioRecibidoComoEvento`.

## ACDS-002

Objetivo: proyectar trabajo externo de dominio como contrato/microtarea sin
acoplar la fuente a OPES, REST, MCP ni runtime.

Estado: hecho.

Validacion:

- `TestAppChangeDirectorDecisionSourceV0ProyectaTrabajoExterno`;
- `go test -count=1 ./modulos/orquesta-app-change-director-source`.

## ACDS-003

Objetivo: abrir revision cuando un cambio de app ya esta entregado.

Estado: hecho.

Validacion:

- `TestAppChangeDirectorDecisionSourceV0AbreRevisionTrasEntrega`;
- `TestCodexStackV0CambioProgresivoPasaReviewGate`;
- `go test -count=1 ./modulos/orquesta-app-change-director-source ./modulos/orquesta-app-codex-stack`.

## ACDS-004

Objetivo: proyectar trabajos documentales externos, especialmente
`draft_content_block` de OPES, como microtareas que declaran paquete de dominio
suficiente y no contexto minimo.

Estado: hecho.

Validacion:

- `TestAppChangeDirectorDecisionSourceV0ProyectaTrabajoExterno`;
- `TestAppChangeDirectorDecisionSourceV0ProyectaTrabajoExternoSinWriteSetLocal`;
- `go test -count=1 ./modulos/orquesta-app-change-director-source`.
