# Errores de rail observados 2026-05-23

Objetivo: acumular errores reales de rails/validadores para convertirlos en
casos de prueba rapidos antes de compilar servidor o ejecutar Orquesta completa.

Comando rapido:

```bash
./scripts/test_rails_fast.sh
```

```text
ID: FILE-BUDGET-FEDERATED-BACKLOG-SOURCE-SPLIT-T257-20260527
Fecha: 2026-05-27
Sintoma: revalidacion OrquestaV2 del backlog pendiente T257 vuelve a exigir
separar carga y parseo del indice federado, con contexto obligatorio
`ref_only` y write-set cerrado al servidor y shards documentales.
Campo: presupuesto de fichero, backlog federado, ACK estricto y contexto
required ref_only.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-c4dae25a0488-g01-ec8e0c187f3a9ac12eb4a2a1b1343950`
de
`request-ref-autoprogramming-backlog-t257-federated-backlog-source-file-split-14e8590c`,
con `required_ref_action=ack_evidence_required` y prueba obligatoria
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`.
Decision: mantener T257 como cierre local ya implementado: carga federada,
parser de indice, parser local y clasificacion de estado/quarantine quedan
separados en `cmd/orquesta-server`, sin reabrir T249/T250/T251/T252/T254 ni
convertir refs opacas en rutas.
Test futuro: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`.
Estado: revalidado por T257; contexto `ref_only` debe resolverse en ACK.
Revalidacion burst 002:
`agent-ref-assessment-task-autoprogramming-c4dae25a0488-g01-a2fd4b485cb6c9eb3e118699b8c263e8`
confirma el cierre sin abrir rail nuevo.
Revalidacion burst 003:
`agent-ref-assessment-task-autoprogramming-c4dae25a0488-g01-6e9bb2925e8315f5a8e8a4458c5dafec`
confirma el cierre T257; la prueba obligatoria queda bloqueada por symbols de
`modulos/orquesta-app-director-service` fuera del write-set.
Revalidacion burst 002 adicional:
`agent-ref-assessment-task-autoprogramming-c4dae25a0488-g01-d42ec6b2c4feefb41c1a8bd1217aba3a`
confirma que el paquete solo reobserva T257. El contexto `ref_only` se resuelve
por lectura local del paquete y fuentes vigentes; la prueba obligatoria vuelve a
quedar bloqueada por symbols de `modulos/orquesta-app-director-service` fuera
del write-set cerrado.
Revalidacion burst 002 adicional 6dc6:
`agent-ref-assessment-task-autoprogramming-c4dae25a0488-g01-6dc6bdca36a88bfa03d0be4e0ef98dbd`
confirma de nuevo que T257 ya esta cerrado localmente. El paquete conserva
`worktree_ref` y `branch_ref` como refs opacas, resuelve contexto `ref_only` por
lectura local/evidencia ACK y no abre codigo ni owners nuevos.
Revalidacion burst 002 adicional b744:
`agent-ref-assessment-task-autoprogramming-c4dae25a0488-g01-b744e45b4bc69e90339caae12c59ddac`
confirma otra reobservacion del mismo cierre T257. La resolucion sigue siendo
lectura local del paquete y fuentes vigentes, ACK con evidencia `ref_only`, sin
codigo nuevo ni ampliacion de write-set.
Revalidacion burst 002 3519:
`agent-ref-assessment-task-autoprogramming-c4dae25a0488-g01-3519b447ee0e247812f0a2de19e107ff`
confirma otra reobservacion de T257 ya cerrado. La resolucion sigue siendo
lectura local/evidencia ACK, sin codigo nuevo ni ampliacion de write-set.
Revalidacion burst 003 e3e841:
`agent-ref-assessment-task-autoprogramming-c4dae25a0488-g01-e3e8416382f03b021e17fad4f48672c6`
confirma otra reobservacion de T257 ya cerrado. La resolucion sigue siendo
lectura local/evidencia ACK, sin codigo nuevo ni ampliacion de write-set.
Revalidacion burst 002 6ad2:
`agent-ref-assessment-task-autoprogramming-c4dae25a0488-g01-6ad2ce2a420fdd4406624de54709433a`
confirma otra reobservacion de T257 ya cerrado. El contexto `ref_only` se
resuelve por lectura local/evidencia ACK; no se abre codigo nuevo ni se amplia
write-set.
```

```text
ID: BACKLOG-TASK-NUMBER-RESERVATION-20260527-001
Fecha: 2026-05-27
Sintoma: scanners concurrentes podian proponer o consumir `## Txx` usando el
numero humano como identidad suficiente, aunque el backlog ya contenia
duplicados historicos como `## T250`.
Campo: autoprogramacion, backlog planner, merge lease documental.
Payload minimo: request de backlog con `request_ref`, `correlation_id`,
`backlog_scan_epoch`, write-set cerrado y snapshot con al menos dos encabezados
`## T250` no equivalentes.
Decision: cada request de backlog debe transportar `task_id_ref`,
`reservation-ref-backlog-task-id-*` y rango `Txx`; el lector debe conservar
instancias por path, linea y fingerprint, y publicar
`backlog_task_number_collision` o exigir reserva antes de escribir.
Test futuro: `go test -count=1 ./cmd/orquesta-server -run
'TestBacklogTaskIDAllocationLeaseV0'`.
Estado: cubierto 2026-05-27 por T252 backlog-task-number-reservation-policy
```

```text
ID: FILE-BUDGET-RUNTIME-PROCESS-CONNECTOR-T256-20260527
Fecha: 2026-05-27
Sintoma: `process_runtime_connector_v0.go` y
`process_runtime_connector_types_v0.go` superaban 300 lineas y mezclaban
lifecycle de proceso, adopcion, watchers, DTOs, validacion y guardas de
env/rutas/shell.
Campo: presupuesto de fichero, runtime neutral y autoprogramacion T256.
Payload minimo: modulo `modulos/orquesta-runtime` con
`ProcessRuntimeConnectorV0`, `ProcessRuntimeLaunchRequestV0` y tests de proceso
local real controlado.
Decision: cerrar T256 separando DTOs/errores, validacion de launch, politica de
env, guardas de valores, estado interno, adopcion y watchers en ficheros
menores de 300 lineas, sin cambiar refs ni codigos publicos.
Test futuro: `go test -count=1 ./modulos/orquesta-runtime`.
Estado: cerrado por T256
Rework de revision 2026-05-27:
`agent-ref-task-ref-review-rework-task-autoprogramming-d1516ddebab4-g01-736b5902b9f28a1666a612eccb83a9c2`
conserva el cierre, no abre rail nuevo y exige ACK con
`contexto_ref_only_resuelto`.
```

```text
ID: FILE-BUDGET-SERVER-SHUTDOWN-T256-REWORK-20260527-C853
Fecha: 2026-05-27
Sintoma: rework de evaluacion vuelve a observar
`server-shutdown-usecase-file-split-before-growth` con contexto obligatorio
`ref_only`, write-set cerrado y T256 ya fusionado con el owner canonico.
Campo: presupuesto de fichero, shutdown hexagonal, ACK estricto y contexto
required ref_only.
Payload minimo: paquete
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-c853a308c21a89806299b2db6b75edfe`
de
`request-ref-autoprogramming-backlog-t256-server-shutdown-usecase-file-split-before-growth-079d82a9`,
con `required_ref_action=ack_evidence_required` y prueba obligatoria
`go test -count=1 ./modulos/orquesta-server-shutdown ./cmd/orquesta-server`.
Decision: conservar T256 como cierre canonico de `orquesta-server-shutdown`;
no abrir owner nuevo para el alias `before-growth`; resolver `ref_only` por
lectura local/evidencia ACK. El build obligatorio queda bloqueado por simbolos
indefinidos en `modulos/orquesta-app-director-service`, fuera del write-set.
Test futuro: `go test -count=1 ./modulos/orquesta-server-shutdown ./cmd/orquesta-server`.
Estado: registrado; requiere decision/scope separado si se corrige el bloqueo
de `orquesta-app-director-service`.
```

```text
ID: FILE-BUDGET-SERVER-SHUTDOWN-T256-REWORK-20260527-3CF9
Fecha: 2026-05-27
Sintoma: rework de reemplazo vuelve a observar
`server-shutdown-usecase-file-split-before-growth` con contexto obligatorio
`ref_only`, write-set cerrado y T256 ya fusionado con el owner canonico.
Campo: presupuesto de fichero, shutdown hexagonal, ACK estricto y contexto
required ref_only.
Payload minimo: paquete
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-3cf9f50f2a1ee0b94df79560b6eeca53`
de
`request-ref-autoprogramming-backlog-t256-server-shutdown-usecase-file-split-before-growth-079d82a9`,
con `required_ref_action=ack_evidence_required` y prueba obligatoria
`go test -count=1 ./modulos/orquesta-server-shutdown ./cmd/orquesta-server`.
Decision: conservar T256 como cierre canonico de `orquesta-server-shutdown`;
no abrir owner nuevo para el alias `before-growth`; resolver `ref_only` por
lectura local/evidencia ACK.
Test futuro: `go test -count=1 ./modulos/orquesta-server-shutdown ./cmd/orquesta-server`.
Estado: registrado; no abre rail nuevo.
```

```text
ID: FILE-BUDGET-SERVER-SHUTDOWN-T256-REWORK-20260527-A0FA
Fecha: 2026-05-27
Sintoma: rework de evaluacion vuelve a observar
`server-shutdown-usecase-file-split-before-growth` con contexto obligatorio
`ref_only`, write-set cerrado y T256 ya fusionado con el owner canonico.
Campo: presupuesto de fichero, shutdown hexagonal, ACK estricto y contexto
required ref_only.
Payload minimo: paquete
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-a0fa2702f7f819236261be3abc537d86`
de
`request-ref-autoprogramming-backlog-t256-server-shutdown-usecase-file-split-before-growth-079d82a9`,
con `required_ref_action=ack_evidence_required` y prueba obligatoria
`go test -count=1 ./modulos/orquesta-server-shutdown ./cmd/orquesta-server`.
Decision: conservar T256 como cierre canonico de `orquesta-server-shutdown`;
no abrir owner nuevo para el alias `before-growth`; resolver `ref_only` por
lectura local/evidencia ACK.
Test futuro: `go test -count=1 ./modulos/orquesta-server-shutdown ./cmd/orquesta-server`.
Estado: registrado; no abre rail nuevo.
```

```text
ID: FILE-BUDGET-SERVER-SHUTDOWN-T256-REWORK-20260527-2E9
Fecha: 2026-05-27
Sintoma: rework de reemplazo vuelve a observar
`server-shutdown-usecase-file-split-before-growth` con contexto obligatorio
`ref_only`, write-set cerrado y T256 ya fusionado con el owner canonico.
Campo: presupuesto de fichero, shutdown hexagonal, ACK estricto y contexto
required ref_only.
Payload minimo: paquete
`agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-autoprogr-2e9-81707decc46db22fc599c9343d59d3cb`
de
`request-ref-autoprogramming-backlog-t256-server-shutdown-usecase-file-split-before-growth-079d82a9`,
con `required_ref_action=ack_evidence_required` y prueba obligatoria
`go test -count=1 ./modulos/orquesta-server-shutdown ./cmd/orquesta-server`.
Decision: conservar T256 como cierre canonico de `orquesta-server-shutdown`;
no abrir owner nuevo para el alias `before-growth`; resolver `ref_only` por
lectura local/evidencia ACK.
Test futuro: `go test -count=1 ./modulos/orquesta-server-shutdown ./cmd/orquesta-server`.
Estado: registrado; no abre rail nuevo.
```

```text
ID: FILE-BUDGET-SERVER-SHUTDOWN-T256-REVALIDACION-20260611-D18F
Fecha: 2026-06-11
Sintoma: OrquestaV2 vuelve a observar
`server-shutdown-usecase-file-split-before-growth` con contexto obligatorio
`ref_only`, write-set cerrado y T256 ya fusionado con el owner canonico.
Campo: presupuesto de fichero, shutdown hexagonal, ACK estricto y contexto
required ref_only.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-fe9cf6f32e89-g01-d18f387e937eb820ba9d7cf0b788b9a3`
de
`request-ref-autoprogramming-backlog-t256-server-shutdown-usecase-file-split-before-growth-079d82a9-retry-659cd992a1204bf061b34ebb4ebd653f0ec1e531ef16e95b8a72d7bf179bae58`,
con `required_ref_action=ack_evidence_required` y prueba obligatoria
`go test -count=1 ./modulos/orquesta-server-shutdown ./cmd/orquesta-server`.
Decision: conservar T256 como cierre canonico de `orquesta-server-shutdown`,
no abrir owner nuevo para el alias `before-growth`, mantener
`ShutdownServerV0` como fachada y separar validacion de dependencias en fichero
propio del modulo.
Test futuro: `go test -count=1 ./modulos/orquesta-server-shutdown ./cmd/orquesta-server`.
Estado: revalidado por T256; contexto `ref_only` debe resolverse en ACK.
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-043
Fecha: 2026-05-27
Sintoma: retry 4da183 burst 002 del scanner de automejora repite contexto
obligatorio `ref_only`, write-set documental cerrado, prueba global obligatoria
y backlog degradado ya cubierto por owners pendientes antes de programar codigo.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete `agent-ref-task-autoprogramming-3533ffab219a-g01` del
retry
`request-ref-autoprogramming-backlog-scanner-15eeecb9-retry-4da1836e0a02ce563fd804cadbabca472bed472d00f586dc4d7070bb0e408426`,
con
`correlation_id=corr-request-ref-autoprogramming-backlog-scanner-15eeecb9-retry-4da1836e0a02ce563fd804cadbabca472bed472d00f586dc4d7070bb0e408426-burst-002`,
`required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250,
T251, T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx ni programar codigo desde el scanner; registrar
no-op cubierto, resolver `ref_only` mediante lectura local/evidencia ACK y
conservar `worktree_ref`/`branch_ref` como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Evidencia ACK: lectura local del paquete, `AGENTS.md`, `README.md`,
`docs/README.md`, foto vigente, guia del nucleo, principio del director,
matriz de smokes, backlog vivo y shards de rail confirma que el retry no aporta
frontera nueva.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: FILE-BUDGET-SERVER-SHUTDOWN-T256-REWORK-20260611-7A851
Fecha: 2026-06-11
Sintoma: OrquestaV2 relanza el alias
`server-shutdown-usecase-file-split-before-growth` con contexto obligatorio
`ref_only`, write-set cerrado y T256 ya cerrado como owner canonico.
Campo: `modulos/orquesta-server-shutdown/shutdown_v0.go` y presupuesto de
ficheros Go del modulo.
Payload minimo: paquete `agent-ref-task-autoprogramming-7a851a71f125-g01` con
`required_ref_action=ack_evidence_required`, prueba obligatoria focal y refs de
worktree/branch opacas.
Decision: no abrir owner nuevo para el alias `before-growth`; conservar T256
como cierre canonico, resolver `ref_only` por lectura local/evidencia ACK y
anadir guarda de no-crecimiento en `architecture_v0_test.go`.
Test futuro: `go test -count=1 ./modulos/orquesta-server-shutdown ./cmd/orquesta-server`.
Estado: cerrado por revalidacion T256
```

```text
ID: FILE-BUDGET-SERVER-SHUTDOWN-T256-REWORK-20260611-F02E
Fecha: 2026-06-11
Sintoma: retry OrquestaV2 vuelve a materializar
`server-shutdown-usecase-file-split-before-growth` pese a T256 cerrado.
Campo: parser de backlog/automejora en `cmd/orquesta-server` y documentos de
rail que usan estado fechado.
Payload minimo: paquete `agent-ref-task-autoprogramming-f02e03774215-g01` con
contexto `ref_only` obligatorio y write-set cerrado.
Decision: conservar el cierre canonico T256 y hacer que el planner reconozca
`Estado 2026-..:`/`Cierre local 2026-..:` como estado local fechado antes de
generar nuevas peticiones.
Test futuro: `go test -count=1 ./modulos/orquesta-server-shutdown ./cmd/orquesta-server`.
Estado: cerrado por revalidacion T256
```

```text
ID: FILE-BUDGET-SERVER-SHUTDOWN-T256-REWORK-20260611-510467
Fecha: 2026-06-11
Sintoma: retry OrquestaV2 f39e vuelve a materializar el alias
`server-shutdown-usecase-file-split-before-growth` con contexto obligatorio
`ref_only`, aunque T256 ya esta cerrado.
Campo: `orquesta-server-shutdown`, presupuesto de ficheros Go y planner de
automejora que no debe abrir owner nuevo para aliases absorbidos.
Payload minimo: paquete `agent-ref-task-autoprogramming-510467d5758e-g01` con
write-set cerrado, prueba obligatoria focal y refs de worktree/branch opacas.
Decision: conservar T256 como cierre canonico, no tocar codigo ya separado,
resolver `ref_only` por lectura local/evidencia ACK y usar la guarda existente
de `architecture_v0_test.go` como prueba de no-crecimiento.
Test futuro: `go test -count=1 ./modulos/orquesta-server-shutdown ./cmd/orquesta-server`.
Estado: cerrado por revalidacion T256
```

```text
ID: BACKLOG-T257-REWORK-ASSESSMENT-20260527-5A8806
Fecha: 2026-05-27
Sintoma: correccion tras revision de T257 recibe paquete estricto con contexto
obligatorio `ref_only`, write-set cerrado a servidor y shards documentales, y
criterio de conservar entrega valida sin relanzar agente padre.
Campo: ACK estricto, backlog federado, rail errors, duplicaciones y prueba
focal del servidor.
Payload minimo: paquete
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-c4dae25a0488-g01-385-5a8806ef75f49aae3734e7a512136f48`
de `request-ref-autoprogramming-backlog-t257-federated-backlog-source-file-split-14e8590c`
con `required_ref_action=ack_evidence_required`.
Decision: no abrir owner nuevo; mantener T257 cerrado como split del backlog
federado en `cmd/orquesta-server`, resolver `ref_only` por lectura local mas
evidencia ACK y conservar refs de runtime como opacas.
Test futuro: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`.
Estado: registrado, cubierto
```

```text
ID: T257-FEDERATED-BACKLOG-SOURCE-FILE-SPLIT-REVALIDATION-20260527-7CAABB
Fecha: 2026-05-27
Sintoma: OrquestaV2 reemite la tarea T257 con contexto obligatorio `ref_only`
y write-set cerrado, aunque el split federado ya esta implementado.
Campo: planner de automejora, backlog federado y ACK estricto.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-c4dae25a0488-g01-7caabb5af4687f69a91e450ea47d88db`
de
`request-ref-autoprogramming-backlog-t257-federated-backlog-source-file-split-14e8590c`,
con `required_ref_action=ack_evidence_required`, test obligatorio
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server` y refs
opacas de worktree/branch.
Decision: no abrir otro Txx ni modificar codigo; registrar revalidacion de
T257, resolver `ref_only` mediante lectura local/evidencia ACK y preservar
`worktree_ref`/`branch_ref` como refs opacas.
Test futuro: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`.
Estado: registrado, cubierto por T257 cerrado
```

```text
ID: BACKLOG-T257-REVALIDATION-NOOP-20260527-EBEB
Fecha: 2026-05-27
Sintoma: OrquestaV2 reobserva T257 `federated-backlog-source-file-split` con
contexto obligatorio `ref_only`, write-set cerrado y owner ya cerrado.
Campo: autoprogramacion, backlog federado, ACK estricto y cierre documental.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-c4dae25a0488-g01-ebebfa72236082cbc973b4c48b012e70`
de la request
`request-ref-autoprogramming-backlog-t257-federated-backlog-source-file-split-14e8590c`,
con `required_ref_action=ack_evidence_required` y prueba obligatoria focal del
servidor.
Decision: no abrir otro owner ni duplicar politicas; cerrar como revalidacion
de T257 con lectura local/evidencia ACK, conservando `worktree_ref` y
`branch_ref` como refs opacas. La prueba focal puede quedar bloqueada por
compilacion de `modulos/orquesta-app-director-service` fuera del write-set.
Test futuro: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`.
Estado: registrado, cubierto por T257 y owners vecinos T44/T249/T250/T251/T252/T254/T255/T256/T258
```

```text
ID: BACKLOG-T257-REVALIDATION-NOOP-20260527-688572
Fecha: 2026-05-27
Sintoma: OrquestaV2 reobserva T257 `federated-backlog-source-file-split` con
contexto obligatorio `ref_only`, write-set cerrado y owner ya cerrado.
Campo: autoprogramacion, backlog federado, ACK estricto y cierre documental.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-c4dae25a0488-g01-688572c0228e7735f6c5a3d0b137d96f`
de la request
`request-ref-autoprogramming-backlog-t257-federated-backlog-source-file-split-14e8590c`,
con `required_ref_action=ack_evidence_required` y prueba obligatoria focal del
servidor.
Decision: no abrir otro owner ni duplicar politicas; cerrar como revalidacion
de T257 con lectura local/evidencia ACK, conservando `worktree_ref` y
`branch_ref` como refs opacas.
Test futuro: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`.
Estado: registrado, cubierto por T257 y owners vecinos T44/T249/T250/T251/T252/T254/T255/T256/T258
```

```text
ID: FILE-BUDGET-APP-DIRECTOR-SERVICE-T258-20260527
Fecha: 2026-05-27
Sintoma: suite historica de `orquesta-app-director-service` acumulaba tests de
Director Operativo, cierre causal, replay statefile y replan en ficheros Go
por encima de 300 lineas.
Campo: presupuesto de fichero, suite Go y autoprogramacion T258.
Payload minimo: `operational_director_v0_test.go`,
`operational_closure_v0_test.go`,
`operational_director_full_statefile_replay_v0_test.go`,
`director_decision_source_v0_test.go` y vecinos de tests locales.
Decision: cerrar T258 repartiendo tests y helpers de test por escenario dentro
del modulo, sin tocar contratos publicos ni adaptadores de producto.
Test futuro: `go test -count=1 ./modulos/orquesta-app-director-service`.
Estado: cerrado por T258
Rework 2026-05-27: revision de entrega confirma que el cierre pertenece a
T258, no a T253; el modulo queda bajo el rail de 300 lineas por fichero Go.
```

```text
ID: BACKLOG-T254-REWORK-ACK-20260527-001
Fecha: 2026-05-27
Sintoma: correccion tras revision para T254 llega con contexto obligatorio
`ref_only`, `required_ref_action=ack_evidence_required` y entrega valida que
debe cerrarse sin relanzar agente padre ni abrir otro owner solapado.
Campo: backlog task id alias index, ACK estricto y required ref_only context.
Payload minimo: paquete
`agent-ref-task-ref-review-rework-task-autoprogramming-6ed65028eb0b-g01-0154a0e80595eed31f9c3059a95a2f93`
de `request-ref-autoprogramming-backlog-t254-backlog-task-id-collision-alias-index-09624d3f`,
write-set cerrado a servidor/autoprogramacion/MCP y shards documentales, con
prueba obligatoria focal del servidor residente.
Decision: conservar T254 como cierre ya implementado, resolver `ref_only` por
lectura local/evidencia ACK, no crear un Txx nuevo y no convertir refs opacas de
worktree/branch en rutas.
Test futuro: `go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-server ./modulos/orquesta-mcp ./cmd/orquesta-server`.
Estado: registrado, rework cerrado por ACK
```

```text
ID: BACKLOG-TASK-ID-COLLISION-ALIAS-INDEX-20260527
Fecha: 2026-05-27
Sintoma: varias secciones ejecutables `## T250` existen en el backlog vivo y un
cierre, ACK, stats, roadmap o MCP que use solo el numero humano puede apuntar a
la frontera equivocada.
Campo: planner de automejora residente, scanner de backlog y superficies
publicas que reportan tareas `Txx`.
Decision aplicada: mantener reparacion aditiva sin renumerar historico; publicar
`task_instance_ref`/`backlog_task_entry_ref`, alias `Txx#NN`,
`backlog_duplicate_task_id_ambiguous`, `instance_refs` y evidencia de alias
index en las requests y colisiones del planner.
Test: `go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-server ./modulos/orquesta-mcp ./cmd/orquesta-server`.
Backlog: `T254 backlog-task-id-collision-alias-index`.
Estado: cubierto por T254.
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-046
Fecha: 2026-05-27
Sintoma: retry 9607b2 burst 002 del scanner de automejora repite contexto
obligatorio `ref_only`, write-set documental cerrado, prueba global obligatoria
y backlog degradado ya cubierto por owners pendientes antes de programar codigo.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete `agent-ref-task-autoprogramming-1db274df2c82-g01` del
retry
`request-ref-autoprogramming-backlog-scanner-15eeecb9-retry-9607b28f843c98b4bd58ad70bf22a6e18f7636b31d8e71af7b7521ec3c3e4963`,
con
`correlation_id=corr-request-ref-autoprogramming-backlog-scanner-15eeecb9-retry-9607b28f843c98b4bd58ad70bf22a6e18f7636b31d8e71af7b7521ec3c3e4963-burst-002`,
`required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250,
T251, T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx ni programar codigo desde el scanner; registrar
no-op cubierto, resolver `ref_only` mediante lectura local/evidencia ACK y
conservar `worktree_ref`/`branch_ref` como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Evidencia ACK: lectura local del paquete, `AGENTS.md`, `README.md`,
`docs/README.md`, foto vigente, guia del nucleo, principio del director,
backlog vivo y shards de rail confirma que el retry no aporta frontera nueva.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-045
Fecha: 2026-05-27
Sintoma: retry ca7a01 burst 002 del scanner de automejora repite contexto
obligatorio `ref_only`, write-set documental cerrado, prueba global obligatoria
y backlog degradado ya cubierto por owners pendientes antes de programar codigo.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete `agent-ref-task-autoprogramming-b06fa6486ff1-g01` del
retry
`request-ref-autoprogramming-backlog-scanner-15eeecb9-retry-ca7a01129405f7db0785957e06ec98f0f0e537093fdac53f5fd258be2448367e`,
con
`correlation_id=corr-request-ref-autoprogramming-backlog-scanner-15eeecb9-retry-ca7a01129405f7db0785957e06ec98f0f0e537093fdac53f5fd258be2448367e-burst-002`,
`required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250,
T251, T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx ni programar codigo desde el scanner; registrar
no-op cubierto, resolver `ref_only` mediante lectura local/evidencia ACK y
conservar `worktree_ref`/`branch_ref` como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Evidencia ACK: lectura local del paquete, `AGENTS.md`, `README.md`,
`docs/README.md`, foto vigente, guia del nucleo, principio del director,
matriz de smokes, backlog vivo y shards de rail confirma que el retry no aporta
frontera nueva.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-044
Fecha: 2026-05-27
Sintoma: retry 4da183 burst 003 del scanner de automejora repite contexto
obligatorio `ref_only`, write-set documental cerrado, prueba global obligatoria
y backlog degradado ya cubierto por owners pendientes antes de programar codigo.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-3533ffab219a-g01-3b2c0d9af98bdc2180840a7c40a31ccc`
del retry
`request-ref-autoprogramming-backlog-scanner-15eeecb9-retry-4da1836e0a02ce563fd804cadbabca472bed472d00f586dc4d7070bb0e408426`,
con
`correlation_id=corr-request-ref-autoprogramming-backlog-scanner-15eeecb9-retry-4da1836e0a02ce563fd804cadbabca472bed472d00f586dc4d7070bb0e408426-burst-003`,
`required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250,
T251, T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx ni programar codigo desde el scanner; registrar
no-op cubierto, resolver `ref_only` mediante lectura local/evidencia ACK y
conservar `worktree_ref`/`branch_ref` como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-037
Fecha: 2026-05-27
Sintoma: retry 598e4b burst 002 del scanner de automejora repite contexto
obligatorio `ref_only`, write-set documental cerrado, prueba global obligatoria
y backlog degradado ya cubierto por owners pendientes antes de programar codigo.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete `agent-ref-task-autoprogramming-65f3f5860bf2-g01` del
retry
`request-ref-autoprogramming-backlog-scanner-15eeecb9-retry-598e4b7a32f46accd9b3de9546dbf463558dc510b5b5012327fc817ecb7b0e83`,
con
`correlation_id=corr-request-ref-autoprogramming-backlog-scanner-15eeecb9-retry-598e4b7a32f46accd9b3de9546dbf463558dc510b5b5012327fc817ecb7b0e83-burst-002`,
`required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250,
T251, T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx ni programar codigo desde el scanner; registrar
no-op cubierto, resolver `ref_only` mediante lectura local/evidencia ACK y
conservar `worktree_ref`/`branch_ref` como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Evidencia ACK: lectura local del paquete, `AGENTS.md`, `README.md`,
`docs/README.md`, foto vigente, guia del nucleo, principio del director,
backlog vivo y shards de rail confirma que el retry no aporta frontera nueva.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-042
Fecha: 2026-05-27
Sintoma: retry db2473 burst 002 del scanner de automejora repite contexto
obligatorio `ref_only`, write-set documental cerrado, prueba global obligatoria
y backlog degradado ya cubierto por owners pendientes antes de programar codigo.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete `agent-ref-task-autoprogramming-ba3b7ec7b59c-g01` del
retry
`request-ref-autoprogramming-backlog-scanner-15eeecb9-retry-db2473564d775de155beec4b14c9eb02616b2c586a9ba5d52a86784b0ed8f115`,
con
`correlation_id=corr-request-ref-autoprogramming-backlog-scanner-15eeecb9-retry-db2473564d775de155beec4b14c9eb02616b2c586a9ba5d52a86784b0ed8f115-burst-002`,
`required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250,
T251, T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx ni programar codigo desde el scanner; registrar
no-op cubierto, resolver `ref_only` mediante lectura local/evidencia ACK y
conservar `worktree_ref`/`branch_ref` como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-041
Fecha: 2026-05-27
Sintoma: retry debe09 burst 002 del scanner de automejora repite contexto
obligatorio `ref_only`, write-set documental cerrado, prueba global obligatoria
y backlog degradado ya cubierto por owners pendientes antes de programar codigo.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete `agent-ref-task-autoprogramming-62119e82f3e7-g01` del
retry
`request-ref-autoprogramming-backlog-scanner-15eeecb9-retry-debe0920522702b51362a84a261ba018fb91aa97096707501f61c5caafc751c6`,
con
`correlation_id=corr-request-ref-autoprogramming-backlog-scanner-15eeecb9-retry-debe0920522702b51362a84a261ba018fb91aa97096707501f61c5caafc751c6-burst-002`,
`required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250,
T251, T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx ni programar codigo desde el scanner; registrar
no-op cubierto, resolver `ref_only` mediante lectura local/evidencia ACK y
conservar `worktree_ref`/`branch_ref` como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-040
Fecha: 2026-05-27
Sintoma: retry c848ae burst 002 del scanner de automejora repite contexto
obligatorio `ref_only`, write-set documental cerrado, prueba global obligatoria
y backlog degradado ya cubierto por owners pendientes antes de programar codigo.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete `agent-ref-task-autoprogramming-d5b73c0f6568-g01` del
retry
`request-ref-autoprogramming-backlog-scanner-15eeecb9-retry-c848ae18b89a2ec8f6d3236024c8333f9c15cfa95c4ade488f8c3bdd7467f843`,
con
`correlation_id=corr-request-ref-autoprogramming-backlog-scanner-15eeecb9-retry-c848ae18b89a2ec8f6d3236024c8333f9c15cfa95c4ade488f8c3bdd7467f843-burst-002`,
`required_ref_action=ack_evidence_required`, write-set limitado a backlog, rail
errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250, T251,
T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx ni programar codigo desde el scanner; registrar
no-op cubierto, resolver `ref_only` mediante lectura local/evidencia ACK y
conservar `worktree_ref`/`branch_ref` como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-039
Fecha: 2026-05-27
Sintoma: retry 33cf37 burst 002 del scanner de automejora repite contexto
obligatorio `ref_only`, write-set documental cerrado, prueba global obligatoria
y backlog degradado ya cubierto por owners pendientes antes de programar codigo.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete `agent-ref-task-autoprogramming-29c1c237ce8c-g01` del
retry
`request-ref-autoprogramming-backlog-scanner-15eeecb9-retry-33cf37fb0bd7e13be407ed5b412020114079ebc4b8b38a7723bc112729f660e8`,
con
`correlation_id=corr-request-ref-autoprogramming-backlog-scanner-15eeecb9-retry-33cf37fb0bd7e13be407ed5b412020114079ebc4b8b38a7723bc112729f660e8-burst-002`,
`required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250,
T251, T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx ni programar codigo desde el scanner; registrar
no-op cubierto, resolver `ref_only` mediante lectura local/evidencia ACK y
conservar `worktree_ref`/`branch_ref` como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: APP-CODEX-AUTOPROGRAMMING-BRIDGE-SPLIT-20260527
Fecha: 2026-05-27
Sintoma: T247 podia reabrirse aunque el bridge productivo ya estuviera dividido
por request, run, store/replay y continue/wait refs; el test local del bridge
seguia concentrando escenarios y ocultaba el cierre verificable.
Campo: stack Codex, prepare-run de autoprogramacion, replay/idempotencia y
reintentos residentes.
Decision: cerrar T247 con sharding de tests del bridge sin mover runtime,
provider, state-file ni politica pura al nucleo.
Test futuro: `go test -count=1 ./modulos/orquesta-app-codex-stack`.
Estado: cerrado localmente
```

```text
ID: APP-CODEX-AUTOPROGRAMMING-BRIDGE-SPLIT-REWORK-20260527
Fecha: 2026-05-27
Sintoma: la correccion de review/rework de T247 podia interpretarse como
necesidad de relanzar otro agente padre o reabrir el bridge ya dividido, pese a
que la entrega aceptada conservaba owner, tests y sharding bajo el limite local.
Campo: stack Codex, review/rework de autoprogramacion y backlog ejecutable.
Decision: no relanzar agente padre ni tocar el bridge productivo; conservar la
entrega valida y registrar la correccion como acreditacion documental con
contexto `ref_only` resuelto por lectura local/evidencia ACK.
Test futuro: `go test -count=1 ./modulos/orquesta-app-codex-stack`.
Estado: cerrado localmente
```

```text
ID: OPS-RUNTIME-DETAIL-RAW-CONTENT-20260527
Fecha: 2026-05-27
Sintoma: `/ops` leia control files de agentes y renderizaba prompts, packets,
ACKs, decisiones y logs como bloques crudos; para no-log usaba lectura completa
antes de truncar.
Campo: detalle runtime operacional del stack Codex y dashboard web.
Decision: cerrado por T227 con envelopes publicos por fichero, presupuesto de
lectura previo, refs/hashes compactos, extractos JSON permitidos y tail
redactado. No exponer paths locales, HOME, tokens, prompts, transcripts,
completions, stdout/stderr completo ni payloads HTTP.
Test futuro: mantener cobertura en
`TestCodexStackAgentRuntimeDetailHTTPHandlerV0DevuelveEnvelopeRedactado`,
`TestCodexStackAgentRuntimeDetailHTTPHandlerV0LeeLogsPorTailAcotado` y
`TestOpsDashboardWebEndpointV0RenderizaPanelLiveCompleto`.
Estado: cerrado localmente
```

```text
ID: BACKLOG-SCAN-ENTRY-IDENTITY-20260527-001
Fecha: 2026-05-27
Sintoma: titulos `Escaneo backlog 2026-05-27 ...` repetidos o fuera de orden
podian usarse como evidencia humana suficiente para merge/rebase aunque no
tuvieran identidad causal propia.
Campo: merge lease del scanner de backlog en `cmd/orquesta-server`.
Payload minimo: secciones `## Escaneo backlog ...` con ordinal duplicado,
ordinal ausente o salto no monotono antes de tareas `## Txx`.
Decision: cerrado por T249; el lease deriva `scan_entry_ref` por documento,
fecha, ordinal y hash de bloque, transporta digest/issues y publica
`backlog_scan_entry_duplicate` o `backlog_scan_entry_order_ambiguous` para pedir
rebase o `CONSULTA AL DIRECTOR` sin renumerar historico.
Test ejecutado: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`.
Estado: cerrado
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-023
Fecha: 2026-05-27
Sintoma: burst 002 del scanner de automejora repite contexto obligatorio
`ref_only`, write-set documental cerrado, prueba global obligatoria y huecos ya
visibles en backlog.
Campo: backlog scanner, ACK estricto, request base y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-9fa6b01dc86ff93d9228ca93a474de5c`
de `request-ref-autoprogramming-backlog-scanner-7e8a6ae9`, con
`correlation_id=corr-request-ref-autoprogramming-backlog-scanner-7e8a6ae9-burst-002`,
`required_ref_action=ack_evidence_required`, write-set limitado a backlog, rail
errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250, T251,
T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un burst equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-036
Fecha: 2026-05-27
Sintoma: retry 3d4312 burst 002 del scanner de automejora repite contexto
obligatorio `ref_only`, write-set documental cerrado, prueba global obligatoria
y backlog degradado ya cubierto por owners pendientes antes de programar codigo.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete `agent-ref-task-autoprogramming-f924bc43a081-g01` del
retry
`request-ref-autoprogramming-backlog-scanner-15eeecb9-retry-3d4312f622da5f7999d371853b749e8037dbd7c2f12110565410479b0fd44256`,
con
`correlation_id=corr-request-ref-autoprogramming-backlog-scanner-15eeecb9-retry-3d4312f622da5f7999d371853b749e8037dbd7c2f12110565410479b0fd44256-burst-002`,
`required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250,
T251, T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx ni programar codigo desde el scanner; registrar
no-op cubierto, resolver `ref_only` mediante lectura local/evidencia ACK y
conservar `worktree_ref`/`branch_ref` como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-035
Fecha: 2026-05-27
Sintoma: retry d3160 burst 002 del scanner de automejora repite contexto
obligatorio `ref_only`, write-set documental cerrado, prueba global obligatoria
y backlog degradado ya cubierto por owners pendientes antes de programar codigo.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete `agent-ref-task-autoprogramming-205bd061b6ad-g01` del
retry
`request-ref-autoprogramming-backlog-scanner-15eeecb9-retry-d3160b6776bd90b62315a49b62da6fdf2d6aa6be3e51763b7b0685423ed83944`,
con
`correlation_id=corr-request-ref-autoprogramming-backlog-scanner-15eeecb9-retry-d3160b6776bd90b62315a49b62da6fdf2d6aa6be3e51763b7b0685423ed83944-burst-002`,
`required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250,
T251, T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx ni programar codigo desde el scanner; registrar
no-op cubierto, resolver `ref_only` mediante lectura local/evidencia ACK y
conservar `worktree_ref`/`branch_ref` como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-018-ED6089
Fecha: 2026-05-27
Sintoma: retry ed6089 burst 002 del scanner de automejora repite contexto
obligatorio `ref_only`, write-set documental cerrado y backlog degradado ya
cubierto por owners pendientes antes de programar codigo.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete
`request-ref-autoprogramming-backlog-scanner-15eeecb9-retry-ed6089ea4ce716f9991ef353adc9468a5261ba8198a581cdd29360ebe75e5d7f`,
`correlation_id=corr-request-ref-autoprogramming-backlog-scanner-15eeecb9-retry-ed6089ea4ce716f9991ef353adc9468a5261ba8198a581cdd29360ebe75e5d7f-burst-002`,
`required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250,
T251, T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-034
Fecha: 2026-05-27
Sintoma: retry bb17 burst 002 del scanner de automejora repite contexto
obligatorio `ref_only`, write-set documental cerrado, prueba global obligatoria
y backlog degradado ya cubierto por owners pendientes antes de programar codigo.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete `agent-ref-task-autoprogramming-6ae909f7547f-g01` del
retry
`request-ref-autoprogramming-backlog-scanner-15eeecb9-retry-bb17b5de9b3273d237c8f0ea110420b58d301fbd02207ca151b44a350cc51e4e`,
con
`correlation_id=corr-request-ref-autoprogramming-backlog-scanner-15eeecb9-retry-bb17b5de9b3273d237c8f0ea110420b58d301fbd02207ca151b44a350cc51e4e-burst-002`,
`required_ref_action=ack_evidence_required`, write-set limitado a backlog, rail
errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250, T251,
T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx ni programar codigo desde el scanner; registrar
no-op cubierto, resolver `ref_only` mediante lectura local/evidencia ACK y
conservar `worktree_ref`/`branch_ref` como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-033
Fecha: 2026-05-27
Sintoma: retry 4e3533 del scanner de automejora repite contexto obligatorio
`ref_only`, write-set documental cerrado, prueba global obligatoria y backlog
degradado ya cubierto por owners pendientes antes de programar codigo.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete `agent-ref-task-autoprogramming-8e83bd89baf6-g01` del
retry
`request-ref-autoprogramming-backlog-scanner-15eeecb9-retry-4e3533f995121822a0dc32f1de4dc8cddecbe0896538c03c502b3ef2f43adfd3`,
con
`correlation_id=corr-request-ref-autoprogramming-backlog-scanner-15eeecb9-retry-4e3533f995121822a0dc32f1de4dc8cddecbe0896538c03c502b3ef2f43adfd3-burst-002`,
`required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250,
T251, T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx ni programar codigo desde el scanner; registrar
no-op cubierto, resolver `ref_only` mediante lectura local/evidencia ACK y
conservar `worktree_ref`/`branch_ref` como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-014
Fecha: 2026-05-27
Sintoma: retry 15eeecb9 del scanner de automejora repite contexto obligatorio
`ref_only`, write-set documental cerrado y backlog degradado ya cubierto por
owners pendientes antes de programar codigo.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete `agent-ref-task-autoprogramming-7b90ca195eaf-g01` del
retry
`request-ref-autoprogramming-backlog-scanner-15eeecb9-retry-5da8540e9825a1ef36992bec2c9c7fcecc002ca6b869058c1afa214eeb713af7`,
con `required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250,
T251, T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx ni programar codigo desde el scanner; registrar
no-op cubierto, resolver `ref_only` mediante lectura local/evidencia ACK y
conservar `worktree_ref`/`branch_ref` como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-032
Fecha: 2026-05-27
Sintoma: retry a3abed burst 002 del scanner de automejora repite contexto
obligatorio `ref_only`, write-set documental cerrado, prueba global obligatoria
y huecos ya visibles en backlog.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete `agent-ref-task-autoprogramming-30b4d5ecfe9b-g01` del
retry
`request-ref-autoprogramming-backlog-scanner-15eeecb9-retry-a3abed43bb343900425d1d9279118a7b96338a93a54f23067693c3273bf5e52a`,
con
`correlation_id=corr-request-ref-autoprogramming-backlog-scanner-15eeecb9-retry-a3abed43bb343900425d1d9279118a7b96338a93a54f23067693c3273bf5e52a-burst-002`,
`required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250,
T251, T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-FEDERATED-EPOCH-SCOPE-20260527-001
Fecha: 2026-05-27
Sintoma: T250 podia confundirse con una reapertura de scanner aunque el servidor
ya distinguia fuentes federadas ejecutables de fuentes historicas, stale o en
quarantine para calcular epoch.
Campo: scanner de backlog federado en `cmd/orquesta-server`.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-88a96698afb2-g01-6dab0da79e1908567a8bf8e7f3250f9b`
con `required_ref_action=ack_evidence_required` y write-set cerrado al servidor
y shards documentales.
Decision: revalidar T250 sin ampliar scope: `BacklogScanDocs`, epoch y reservas
solo usan fuentes federadas ejecutables; las no ejecutables quedan como contexto
`federated_backlog_source_not_executable` y no fuerzan rebase del backlog vivo.
Test futuro: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`.
Estado: revalidado
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-031
Fecha: 2026-05-27
Sintoma: retry 35a079 burst 002 del scanner de automejora repite contexto
obligatorio `ref_only`, write-set documental cerrado, prueba global obligatoria
y huecos ya visibles en backlog.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete `agent-ref-task-autoprogramming-49af6e594adb-g01` del
retry
`request-ref-autoprogramming-backlog-scanner-7e8a6ae9-retry-35a0799f24f24651bedbf5c6cbd3647aad99eeee61e6c12e0be5ba2a10694e3b`,
con `correlation_id=corr-request-ref-autoprogramming-backlog-scanner-7e8a6ae9-retry-35a0799f24f24651bedbf5c6cbd3647aad99eeee61e6c12e0be5ba2a10694e3b-burst-002`,
`required_ref_action=ack_evidence_required`, write-set limitado a backlog, rail
errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250, T251,
T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-030
Fecha: 2026-05-27
Sintoma: retry d1d80 burst 002 del scanner de automejora repite contexto
obligatorio `ref_only`, write-set documental cerrado, prueba global obligatoria
y huecos ya visibles en backlog.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete `agent-ref-task-autoprogramming-798614d0f80e-g01` del
retry
`request-ref-autoprogramming-backlog-scanner-15eeecb9-retry-d1d80a70704399bf29392dd809a2ff101897ee7d290b6aaa3e9c7bb4c80cad1b`,
con `correlation_id=corr-request-ref-autoprogramming-backlog-scanner-15eeecb9-retry-d1d80a70704399bf29392dd809a2ff101897ee7d290b6aaa3e9c7bb4c80cad1b-burst-002`,
`required_ref_action=ack_evidence_required`, write-set limitado a backlog, rail
errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250, T251,
T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-029
Fecha: 2026-05-27
Sintoma: burst 003 fba55/f7d3 del scanner de automejora repite contexto
obligatorio `ref_only`, write-set documental cerrado, prueba global obligatoria
y huecos ya visibles en backlog.
Campo: backlog scanner, ACK estricto, request burst y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-fba55e3296a0-g01-f7d3f88efa3648cbceb5112f1cf4a040`
de `request-ref-autoprogramming-backlog-scanner-15eeecb9`, con
`correlation_id=corr-request-ref-autoprogramming-backlog-scanner-15eeecb9-burst-003`,
`required_ref_action=ack_evidence_required`, write-set limitado a backlog, rail
errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250, T251,
T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un burst equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-026
Fecha: 2026-05-27
Sintoma: burst 002 fba55 del scanner de automejora repite contexto obligatorio
`ref_only`, write-set documental cerrado, prueba global obligatoria y huecos ya
visibles en backlog.
Campo: backlog scanner, ACK estricto, request burst y shards documentales.
Payload minimo: paquete `agent-ref-task-autoprogramming-fba55e3296a0-g01` de
`request-ref-autoprogramming-backlog-scanner-15eeecb9`, con
`correlation_id=corr-request-ref-autoprogramming-backlog-scanner-15eeecb9-burst-002`,
`required_ref_action=ack_evidence_required`, write-set limitado a backlog, rail
errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250, T251,
T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un burst equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-028
Fecha: 2026-05-27
Sintoma: burst 002 fba55/8a1ece del scanner de automejora repite contexto
obligatorio `ref_only`, write-set documental cerrado, prueba global obligatoria
y huecos ya visibles en backlog.
Campo: backlog scanner, ACK estricto, request burst y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-fba55e3296a0-g01-8a1ecec98ecf3323c5488e327056f703`
de `request-ref-autoprogramming-backlog-scanner-15eeecb9`, con
`correlation_id=corr-request-ref-autoprogramming-backlog-scanner-15eeecb9-burst-002`,
`required_ref_action=ack_evidence_required`, write-set limitado a backlog, rail
errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250, T251,
T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un burst equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-027
Fecha: 2026-05-27
Sintoma: burst 002 fba55/f13a del scanner de automejora repite contexto
obligatorio `ref_only`, write-set documental cerrado, prueba global obligatoria
y huecos ya visibles en backlog.
Campo: backlog scanner, ACK estricto, request burst y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-fba55e3296a0-g01-f13a1353960542f949b1c478324a27ec`
de `request-ref-autoprogramming-backlog-scanner-15eeecb9`, con
`correlation_id=corr-request-ref-autoprogramming-backlog-scanner-15eeecb9-burst-002`,
`required_ref_action=ack_evidence_required`, write-set limitado a backlog, rail
errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250, T251,
T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un burst equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-023
Fecha: 2026-05-27
Sintoma: burst 002 f8e40 del scanner de automejora repite contexto obligatorio
`ref_only`, write-set documental cerrado y huecos ya visibles en backlog.
Campo: backlog scanner, ACK estricto, request burst y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-f8e40a7bedded7d2878462d370a72a30`
de `request-ref-autoprogramming-backlog-scanner-7e8a6ae9`, con
`required_ref_action=ack_evidence_required`, write-set limitado a backlog, rail
errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250, T251,
T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un burst equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-023
Fecha: 2026-05-27
Sintoma: burst 002 da14 del scanner de automejora repite contexto obligatorio
`ref_only`, write-set documental cerrado y huecos ya visibles en backlog.
Campo: backlog scanner, ACK estricto, request burst y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-da14e4762930936ab46118bb39f7aa9a`
de `request-ref-autoprogramming-backlog-scanner-7e8a6ae9`, con
`required_ref_action=ack_evidence_required`, write-set limitado a backlog, rail
errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250, T251,
T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un burst equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-015
Fecha: 2026-05-27
Sintoma: burst 002 17978 del scanner de automejora repite contexto obligatorio
`ref_only`, write-set documental cerrado, prueba global obligatoria y huecos ya
visibles en backlog.
Campo: backlog scanner, ACK estricto, request base y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-17978e79d655060ec0da8977cea2a0ee`
de `request-ref-autoprogramming-backlog-scanner-7e8a6ae9`, con
`correlation_id=corr-request-ref-autoprogramming-backlog-scanner-7e8a6ae9-burst-002`,
`required_ref_action=ack_evidence_required`, write-set limitado a backlog, rail
errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250, T251,
T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un burst equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-014
Fecha: 2026-05-27
Sintoma: burst 002 del scanner de automejora repite contexto obligatorio
`ref_only`, write-set documental cerrado y huecos ya visibles en backlog.
Campo: backlog scanner, ACK estricto, request de burst y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-2048ee92b24ec50625c2242b8eb85463`
de `request-ref-autoprogramming-backlog-scanner-7e8a6ae9`, con
`required_ref_action=ack_evidence_required`, write-set limitado a backlog, rail
errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250, T251,
T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un burst equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-025
Fecha: 2026-05-27
Sintoma: burst 002 52b63 del scanner de automejora repite contexto obligatorio
`ref_only`, write-set documental cerrado y huecos ya visibles en backlog.
Campo: backlog scanner, ACK estricto, request principal y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-52b63d817ae78008276d7d3e084d76df`
de `request-ref-autoprogramming-backlog-scanner-7e8a6ae9`, con
`required_ref_action=ack_evidence_required`, write-set limitado a backlog, rail
errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250, T251,
T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un burst equivalente.
Estado: registrado, cubierto por owners pendientes
```

La matriz `TestRailRecordConcurrencyGateExternalMatrixV0` incluye tambien
terminos historicamente problematicos como falsos positivos. La lista operativa
vigente del core se centraliza en
`modulos/orquesta-rails/text_policy_v0.go`: permite palabras
opacas como `runtime`, `provider`, `model`, `codex`, `git`, `db` o `sql` y solo
corta patrones que parecen llevar valor sensible efectivo, por ejemplo
`api_key=`, `client_secret=`, `authorization:`, `bearer ` o material tipo
`-----BEGIN`.

## Formato

```text
ID:
Fecha:
Sintoma:
Campo:
Payload minimo:
Decision:
Test:
Estado:
```

## Casos

```text
ID: RAIL-20260523-001
Fecha: 2026-05-23
Sintoma: `RecordConcurrencyGate: detalle_prohibido`.
Campo: `payload.evidence_refs` / `payload.summary`.
Payload minimo: `secrets_policy`, `oauth-client-secret-policy-doc`, `credential`.
Decision: abrir filtro global por palabras en `RecordConcurrencyGate`; revisar en futuro con sanitizador/clasificador por campo.
Test: `TestRailRecordConcurrencyGateExternalMatrixV0/secrets-policy-reference`.
Estado: cubierto
```

```text
ID: RAIL-20260523-002
Fecha: 2026-05-23
Sintoma: `RecordConcurrencyGate: detalle_prohibido` al crear evento.
Campo: `ConcurrencyGateRecorded.payload`.
Payload minimo: evento de gate con refs opacas que contienen `secrets_policy`.
Decision: excluir `ConcurrencyGateRecorded` del filtro global de eventos; mantener validacion estructural del evento.
Test: `TestRailRecordConcurrencyGateExternalMatrixV0/secrets-policy-reference`.
Estado: cubierto
```

```text
ID: RAIL-20260523-003
Fecha: 2026-05-23
Sintoma: `director_cycle_step_invalido: step.run.command_effects: run invalido`.
Campo: `run.command_effects` y `run.concurrency_gates`.
Payload minimo: `request-ref-app-completion-loop` dentro de subject refs/proyecciones del gate.
Decision: no pasar `concurrency_gates` ni `command_effects` por filtro global de palabras; son proyecciones/efectos estructurados con refs opacas y hashes.
Test: `TestRailRecordConcurrencyGateExternalMatrixV0/completion-run-ref`.
Estado: cubierto
```

```text
ID: RAIL-20260523-004
Fecha: 2026-05-23
Sintoma: `RequestCapacity: detalle_prohibido` bloquea cola de automejora con tareas listas.
Campo: `payload.task_ref`, `payload.summary`, `payload.evidence_refs` y outbox de capacidad/agente.
Payload minimo: refs o textos operativos con `runtime`, `provider`, `model`, `codex`, `$HOME` u OAuth como politica/ref opaca.
Decision: abrir validacion de RequestCapacity/RequestAgent/outbox/director-agent para permitir detalles operativos opacos y cortar solo patrones sensibles efectivos (`client_secret=`, `api_key=`, `authorization: Bearer`, etc.).
Test: `TestRequestCapacityCommandV0PermiteDetallesOperativosOpacos`, `TestRequestAgentCommandV0PermiteDetallesOperativosOpacos` y rechazos sensibles asociados.
Estado: cubierto
```

```text
ID: RAIL-20260524-005
Fecha: 2026-05-24
Sintoma: `detalle_prohibido` durante ingesta/review/cierre despues de entregar agentes.
Campo: `RegisterDelivery`, `RequestReview`, `ReviewResult`, `AcceptReview`, `CloseTask`, `FinalValidation`, `CloseRun`, gates y eventos globales.
Payload minimo: resumenes o refs de entregas que mencionan `runtime`, `provider`, `model`, `codex`, `git`, `db`, `sql`, `prompt policy` o `transcript policy` como contexto opaco.
Decision: centralizar la lista de patrones sensibles en `modulos/orquesta-rails/text_policy_v0.go` y hacer que los validadores de workflow la usen por helper comun; se eliminan listas locales por comando.
Test: `go test -count=1 ./modulos/orquesta-core-workflow`.
Estado: cubierto
```

```text
ID: RAIL-20260524-006
Fecha: 2026-05-24
Sintoma: `director_supervised_burst_step_input: field=step_input_builder: detalle_prohibido`.
Campo: payload de ciclo del director y datos de runtime persistidos.
Payload minimo: trazas locales con detalles operativos reales o de test
(`access_token=...`, `Bearer ...`, HOME, prompt/transcript policy) dentro de
estado/log de runtime.
Decision: abrir los rails por defecto y, desde 2026-06-02, dejarlos
inoperativos hasta nueva orden. `ORQUESTA_RAILS_MODE` se normaliza a `offline`
y `ORQUESTA_DETAIL_PROHIBITED_RAILS=on` no reactiva bloqueo. No borrar listas
ni evidencias. Reendurecer solo con tarea futura, matrices externas amplias y
preservando detalle crudo local para diagnostico.
Test: `TestTextPolicyV0DetalleProhibidoDesactivadoTemporalmente`,
`TestServerEnvironmentWithDetailRailsDefaultV0AddAperturaSiFalta`.
Estado: cubierto como rails offline hasta nueva orden; pendiente matriz nueva
solo si se decide reintroducir algun rail.
```

```text
ID: FLAKE-20260523-001
Fecha: 2026-05-23
Sintoma: `go test -count=1 ./...` fallo una vez en `cmd/orquesta-server`: stdout no contenia prompt ejecutado para un agente fake recursivo.
Campo: `TestCodexLaunchDirectorWaveCommandV0RecursiveFakeRuntimeEjecutableConLinaje`.
Payload minimo: pendiente de aislar.
Decision: tratar como prioridad de fiabilidad; investigar no determinismo en stdout/orden/concurrencia/estado temporal.
Test: repeticion focal `go test -count=1 ./cmd/orquesta-server -run TestCodexLaunchDirectorWaveCommandV0RecursiveFakeRuntimeEjecutableConLinaje -v`.
Estado: registrado, pendiente de diagnostico
```

```text
ID: DOC-ROUTE-20260526-001
Fecha: 2026-05-26
Sintoma: manual de uso podia mezclar ruta server-first viva con secciones V1.
Campo: docs/uso_actual_app_orquesta.md
Payload minimo: `./orquesta serve`, rutas `/api/*` sin version, OpenClaw o AP-077 leidos como requisito vigente.
Decision: clasificar el documento como manual server-first sincronizado y poner todo el bloque V1 en cuarentena historica con sustitutos `/api/v0/*`.
Test: go test -count=1 ./modulos/orquesta-cli ./modulos/orquesta-web ./cmd/orquesta-server
Estado: cubierto por T119
```

```text
ID: RAIL-REVIEW-GATE-20260526-001
Fecha: 2026-05-26
Sintoma: review gate de Codex podia tratar globs o aliases reparables de
write-set como fuera de alcance, o aplicar presupuesto Go estricto a trabajo
documental.
Campo: ReviewGatePolicy / WorktreeVerifyRequest.write_set
Payload minimo: write_set=`web`, `internal/**/*.go` o policy
`work_profile:documentation`.
Decision: owner unico en `codexStackReviewGatePolicyV0`; el stack expande
aliases locales seguros antes de verificar snapshot y runtime-worktree soporta
globstar en write-set. Perfiles documentales no activan presupuesto Go estricto.
Test: `TestCodexStackReviewGatePolicyToleraAliasYGlobsSegurosV0`,
`TestCodexStackReviewGatePolicyPerfilDocumentalNoActivaGoBudgetV0` y
`TestVerifyWorktreeWriteSetV0PresupuestoLineasGoEstricto/acepta_write-set_con_globstar_seguro`.
Estado: cubierto por T117
```

```text
ID: OPS-RUNTIME-DETAIL-RAW-20260527-001
Fecha: 2026-05-27
Sintoma: `/ops` puede publicar detalle runtime crudo de agentes.
Campo: `runtime_detail.files[].content` y HTML de detalle de run.
Payload minimo: `agent_prompt.txt`, `agent_packet.json`, `agent_ack.json`,
`director_decisions.json`, `codex_stdout.log`, `codex_stderr.log` o
`orquesta_shutdown_request.json` con prompts, paths locales, stdout/stderr,
tokens simulados o payloads operativos.
Decision: abrir backlog T227 para envelope publico redactado, lectura acotada
antes de `os.ReadFile`, tail de logs por presupuesto y reason codes en UI.
Test futuro: matriz focal de detalle runtime en `orquesta-app-codex-stack`,
`orquesta-web`, `cmd/orquesta-server` y runtime Codex delivery.
Estado: resuelto 2026-05-27 con epoch activo acotado a fuentes federadas
ejecutables y reason code `federated_backlog_source_not_executable` para
fuentes historicas/quarantine no bloqueantes.
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-018
Fecha: 2026-05-27
Sintoma: retry 42fbc4 burst 002 del scanner de automejora repite contexto
obligatorio `ref_only`, write-set documental cerrado y huecos ya visibles.
Campo: backlog scanner, ACK estricto, request retry y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-721523601cc1-g01-abd199fbb4db7baa1c08602cd2d3dade`
del retry
`request-ref-autoprogramming-backlog-scanner-7e8a6ae9-retry-42fbc4ded7918354a0ba8d427df92a05da07828f0fbb064fba2a680f461c45f5`,
con `required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250,
T251, T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-024
Fecha: 2026-05-27
Sintoma: burst 002 585742 del scanner de automejora repite contexto obligatorio
`ref_only`, write-set documental cerrado, prueba global obligatoria y owners
pendientes ya visibles.
Campo: backlog scanner, ACK estricto, request principal y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-585742898dfa4be257b5a2282769c9c2`
de `request-ref-autoprogramming-backlog-scanner-7e8a6ae9`, con
`required_ref_action=ack_evidence_required`, write-set limitado a backlog, rail
errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250, T251,
T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un burst equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-022
Fecha: 2026-05-27
Sintoma: retry 69e1ba burst 002 del scanner de automejora repite contexto
obligatorio `ref_only`, write-set documental cerrado y huecos ya visibles en
backlog.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-8f23d8e2a22e-g01-c86d18cc769b57d176d2617c4d0990a6`
del retry
`request-ref-autoprogramming-backlog-scanner-7e8a6ae9-retry-69e1ba62ea01ade8fb67d198484d35f513e28c79077c8a7ac535bc483e60a6c6`,
con `required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250,
T251, T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-021
Fecha: 2026-05-27
Sintoma: retry 7a567340 burst 002 del scanner de automejora repite contexto
obligatorio `ref_only`, write-set documental cerrado, prueba global obligatoria
y huecos ya visibles en backlog.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-3ddbc1385ac2-g01-0d1f79ec31384d00b4994f75b7e11068`
del retry
`request-ref-autoprogramming-backlog-scanner-7e8a6ae9-retry-7a56734034f8d2f7cfecdb944e4e265d06361b422163b501881d49a99fce3508`,
con `required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250,
T251, T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-020
Fecha: 2026-05-27
Sintoma: retry 69e1ba burst 002 del scanner de automejora repite contexto
obligatorio `ref_only`, write-set documental cerrado y huecos ya visibles en
backlog.
Campo: backlog scanner, ACK estricto, request retry y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-8f23d8e2a22e-g01-406b68b132e5ad3f6495920aad6064eb`
del retry
`request-ref-autoprogramming-backlog-scanner-7e8a6ae9-retry-69e1ba62ea01ade8fb67d198484d35f513e28c79077c8a7ac535bc483e60a6c6`,
con `required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250,
T251, T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-019
Fecha: 2026-05-27
Sintoma: retry d64a6a burst 002 del scanner de automejora repite contexto
obligatorio `ref_only`, write-set documental cerrado, prueba global obligatoria
y huecos ya visibles en backlog.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-757fa45b2845-g01-644fcb17f3eb0e2fc3b46dc77aa883b8`
del retry
`request-ref-autoprogramming-backlog-scanner-7e8a6ae9-retry-d64a6a4098bc947a8abac738ecfc29123c24afa6dde953cf9750f22d13b272c6`,
con `required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250,
T251, T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-004
Fecha: 2026-05-27
Sintoma: nueva pasada del scanner documental repite contexto `ref_only`,
write-set cerrado a backlog/rail errors/duplicaciones y owners pendientes ya
visibles.
Campo: backlog scanner, ACK estricto, dedupe de Txx y evidencia documental.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-350b93e475a8-g01-8eae8526b9c0a30f5c6c290c6fc9498b`
con `required_ref_action=ack_evidence_required`, write-set limitado a los tres
shards documentales y busquedas que ya encuentran T44, T249, T250, T251, T252,
T254, T255 y T256.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver contexto por
lectura local/evidencia ACK y dejar la implementacion a los owners existentes.
Test futuro: usar pruebas de T44/T249/T250/T251/T252/T254/T255/T256; esta
entrada solo evita duplicar backlog programable desde el scanner.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-014
Fecha: 2026-05-27
Sintoma: retry 5465c1 burst 002 del scanner de automejora repite contexto
obligatorio `ref_only`, write-set documental cerrado y owners pendientes ya
visibles.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-87d5c800f528-g01-219c350ad3df01efd266321f376e4f7b`
del retry
`request-ref-autoprogramming-backlog-scanner-7e8a6ae9-retry-5465c1fa2a2382d2739d96147eaafe7597f99aa10da08494d40665e47b2446c0`,
con `required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250,
T251, T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-016
Fecha: 2026-05-27
Sintoma: burst 002 del scanner de automejora repite contexto obligatorio
`ref_only`, write-set documental cerrado y huecos ya visibles en backlog.
Campo: backlog scanner, ACK estricto, burst de request y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-988548c1efac02c34d8ab6cf81a53d19`
de `request-ref-autoprogramming-backlog-scanner-7e8a6ae9`, con
`correlation_id=corr-request-ref-autoprogramming-backlog-scanner-7e8a6ae9-burst-002`,
`required_ref_action=ack_evidence_required`, write-set limitado a backlog, rail
errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250, T251,
T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un burst equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-016
Fecha: 2026-05-27
Sintoma: scanner de automejora repite contexto obligatorio `ref_only`,
write-set documental cerrado, prueba global obligatoria y huecos ya visibles en
backlog.
Campo: backlog scanner, ACK estricto, required tests y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-562c496eddcc7c55bd3b622c5e825749`
con `correlation_id=corr-request-ref-autoprogramming-backlog-scanner-7e8a6ae9-burst-002`,
`required_ref_action=ack_evidence_required`, write-set limitado a backlog, rail
errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250, T251,
T252, T253, T254, T255, T256, T257 y T258 como owners pendientes.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258;
esta entrada solo evita duplicar backlog programable desde un scanner
equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-016
Fecha: 2026-05-27
Sintoma: burst 002 70a7 del scanner de automejora repite contexto obligatorio
`ref_only`, write-set documental cerrado, prueba global obligatoria y huecos ya
visibles.
Campo: backlog scanner, ACK estricto, request y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-70a7d8db3b4ebd33b19ae663b484aef6`
de `request-ref-autoprogramming-backlog-scanner-7e8a6ae9`, con
`correlation_id=corr-request-ref-autoprogramming-backlog-scanner-7e8a6ae9-burst-002`,
`required_ref_action=ack_evidence_required`, write-set limitado a backlog, rail
errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250, T251,
T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un burst equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-016
Fecha: 2026-05-27
Sintoma: retry d0cf1e burst 002 del scanner de automejora repite contexto
obligatorio `ref_only`, write-set documental cerrado y huecos ya visibles en
backlog.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-350b93e475a8-g01-0d330c4e1579d03076ce1e175634e42b`
del retry
`request-ref-autoprogramming-backlog-scanner-7e8a6ae9-retry-d0cf1ea77c2b6898418f56ac96139dcb5cb606948a8e3adc89ed26b4a9b1f02c`,
con `required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250,
T251, T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-016
Fecha: 2026-05-27
Sintoma: burst 002 e7040 del scanner de automejora repite contexto obligatorio
`ref_only`, write-set documental cerrado y owners pendientes ya visibles.
Campo: backlog scanner, ACK estricto, request base y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-e7040a9c028b6c9195871408101972c0`
de `request-ref-autoprogramming-backlog-scanner-7e8a6ae9`, con
`correlation_id=corr-request-ref-autoprogramming-backlog-scanner-7e8a6ae9-burst-002`,
`required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250,
T251, T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un burst equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-016
Fecha: 2026-05-27
Sintoma: burst 002 del scanner de automejora repite contexto obligatorio
`ref_only`, write-set documental cerrado y huecos ya visibles en backlog.
Campo: backlog scanner, ACK estricto, request base y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-1db27b63b7ddb14755938b05f1eff32f`
del request
`request-ref-autoprogramming-backlog-scanner-7e8a6ae9`, con
`required_ref_action=ack_evidence_required`, write-set limitado a backlog, rail
errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250, T251,
T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un burst equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-016
Fecha: 2026-05-27
Sintoma: scanner de automejora en burst 002 repite contexto obligatorio
`ref_only`, write-set documental cerrado, prueba global obligatoria y huecos ya
visibles en backlog.
Campo: backlog scanner, ACK estricto, burst de request y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-25ebab4465052369f6e1f66d949cd514`
con `correlation_id=corr-request-ref-autoprogramming-backlog-scanner-7e8a6ae9-burst-002`,
`required_ref_action=ack_evidence_required`, write-set limitado a backlog, rail
errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250, T251,
T252, T253, T254, T255, T256, T257 y T258 como owners pendientes.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un scanner equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-016
Fecha: 2026-05-27
Sintoma: burst 002 del scanner de automejora repite contexto obligatorio
`ref_only`, write-set documental cerrado, prueba global obligatoria y huecos ya
visibles en backlog.
Campo: backlog scanner, ACK estricto, burst de request y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-31b58785a2bc25303ea28ac1a457f2d3`
de `request-ref-autoprogramming-backlog-scanner-7e8a6ae9`, con
`correlation_id=corr-request-ref-autoprogramming-backlog-scanner-7e8a6ae9-burst-002`,
`required_ref_action=ack_evidence_required`, write-set limitado a backlog, rail
errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250, T251,
T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un burst equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-016
Fecha: 2026-05-27
Sintoma: burst 002 del scanner de automejora repite contexto obligatorio
`ref_only`, write-set documental cerrado y owners pendientes ya visibles.
Campo: backlog scanner, ACK estricto, burst de request y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-5b79f9accf52b539a3be229cf4e83bdb`
de `request-ref-autoprogramming-backlog-scanner-7e8a6ae9`, con
`correlation_id=corr-request-ref-autoprogramming-backlog-scanner-7e8a6ae9-burst-002`,
`required_ref_action=ack_evidence_required`, write-set limitado a backlog, rail
errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250, T251,
T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un burst equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-016
Fecha: 2026-05-27
Sintoma: burst 002 c10c del scanner de automejora repite contexto obligatorio
`ref_only`, write-set documental cerrado y huecos ya visibles.
Campo: backlog scanner, ACK estricto, request burst y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-c10cb19de09867b08fe7a7c947dc4467`
de `request-ref-autoprogramming-backlog-scanner-7e8a6ae9`, con
`required_ref_action=ack_evidence_required`, write-set limitado a backlog, rail
errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250, T251,
T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un burst equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-017
Fecha: 2026-05-27
Sintoma: burst 002 c0f755 del scanner de automejora repite contexto obligatorio
`ref_only`, write-set documental cerrado, prueba global obligatoria y owners
pendientes ya visibles.
Campo: backlog scanner, ACK estricto, request principal y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-c0f7550c04cbfe4cddf673cf37c73725`
de la request `request-ref-autoprogramming-backlog-scanner-7e8a6ae9`, con
`required_ref_action=ack_evidence_required`, write-set limitado a backlog, rail
errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250, T251,
T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un burst equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-016
Fecha: 2026-05-27
Sintoma: retry 7a5673 burst 002 del scanner de automejora repite contexto
obligatorio `ref_only`, write-set documental cerrado y huecos ya visibles en
backlog.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-3ddbc1385ac2-g01-7aa072fa3bae532da402f4b748f1904d`
del retry
`request-ref-autoprogramming-backlog-scanner-7e8a6ae9-retry-7a56734034f8d2f7cfecdb944e4e265d06361b422163b501881d49a99fce3508`,
con `required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250,
T251, T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-015
Fecha: 2026-05-27
Sintoma: retry 69e1ba burst 002 del scanner de automejora repite contexto
obligatorio `ref_only`, write-set documental cerrado y huecos ya visibles.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-8f23d8e2a22e-g01-89e006236c585aa247a81664e7b037e1`
del retry
`request-ref-autoprogramming-backlog-scanner-7e8a6ae9-retry-69e1ba62ea01ade8fb67d198484d35f513e28c79077c8a7ac535bc483e60a6c6`,
con `required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250,
T251, T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-014
Fecha: 2026-05-27
Sintoma: retry 5465c1 burst 002 del scanner de automejora repite contexto
obligatorio `ref_only`, write-set documental cerrado, prueba global obligatoria
y owners pendientes ya visibles.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-87d5c800f528-g01-ef66a8153742b6161a7f9836a4c2779e`
del retry
`request-ref-autoprogramming-backlog-scanner-7e8a6ae9-retry-5465c1fa2a2382d2739d96147eaafe7597f99aa10da08494d40665e47b2446c0`,
con `required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250,
T251, T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-014
Fecha: 2026-05-27
Sintoma: retry 42fbc4 burst 002 del scanner de automejora repite contexto
obligatorio `ref_only`, write-set documental cerrado, prueba global obligatoria
y huecos ya visibles en backlog.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-721523601cc1-g01-2e79a5694817170a5384a408d1acb03b`
del retry
`request-ref-autoprogramming-backlog-scanner-7e8a6ae9-retry-42fbc4ded7918354a0ba8d427df92a05da07828f0fbb064fba2a680f461c45f5`,
con `required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250,
T251, T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-008
Fecha: 2026-05-27
Sintoma: retry d64a6a del scanner de automejora repite contexto obligatorio
`ref_only`, write-set documental cerrado, prueba global obligatoria y owners
pendientes ya visibles.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-757fa45b2845-g01-9a0012e071185631d9f7652eb10ef3bb`
del retry
`request-ref-autoprogramming-backlog-scanner-7e8a6ae9-retry-d64a6a4098bc947a8abac738ecfc29123c24afa6dde953cf9750f22d13b272c6`,
con `required_ref_action=ack_evidence_required`, write-set limitado a
backlog/rail errors/duplicaciones y busquedas que ya encuentran T44, T249,
T250, T251, T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-011
Fecha: 2026-05-27
Sintoma: retry burst del scanner de automejora vuelve a traer contexto
obligatorio `ref_only`, write-set documental cerrado, prueba global
obligatoria y huecos ya visibles en backlog.
Campo: backlog scanner, ACK estricto, retry burst y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-87d5c800f528-g01-4fe334168e77c5b5a5a29a19c299e7e8`
del retry
`request-ref-autoprogramming-backlog-scanner-7e8a6ae9-retry-5465c1fa2a2382d2739d96147eaafe7597f99aa10da08494d40665e47b2446c0`,
con `required_ref_action=ack_evidence_required`, write-set limitado a los tres
shards documentales y busquedas que ya encuentran T44, T249, T250, T251, T252,
T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-011
Fecha: 2026-05-27
Sintoma: retry del scanner de automejora repite contexto obligatorio
`ref_only`, write-set documental cerrado, prueba global obligatoria y huecos ya
visibles en backlog.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-721523601cc1-g01-6b02071be67727512a14341842bf4332`
del retry
`request-ref-autoprogramming-backlog-scanner-7e8a6ae9-retry-42fbc4ded7918354a0ba8d427df92a05da07828f0fbb064fba2a680f461c45f5`,
con `required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250,
T251, T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-011
Fecha: 2026-05-27
Sintoma: retry del scanner de automejora repite contexto obligatorio
`ref_only`, write-set cerrado a backlog/rail errors/duplicaciones y owners
pendientes ya visibles.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-3ddbc1385ac2-g01-00ef1096db1ef4935750f733b3fe3ccf`
del retry
`request-ref-autoprogramming-backlog-scanner-7e8a6ae9-retry-7a56734034f8d2f7cfecdb944e4e265d06361b422163b501881d49a99fce3508`,
con `required_ref_action=ack_evidence_required`, write-set limitado a
backlog/rail errors/duplicaciones y busquedas que ya encuentran T44, T249,
T250, T251, T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-012
Fecha: 2026-05-27
Sintoma: retry del scanner de automejora repite contexto obligatorio
`ref_only`, write-set documental cerrado y huecos ya visibles en backlog.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-8f23d8e2a22e-g01-697704c71decfb03744436e559856b7f`
del retry
`request-ref-autoprogramming-backlog-scanner-7e8a6ae9-retry-69e1ba62ea01ade8fb67d198484d35f513e28c79077c8a7ac535bc483e60a6c6`,
con `required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250,
T251, T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258;
esta entrada solo evita duplicar backlog programable desde un retry
equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-010
Fecha: 2026-05-27
Sintoma: retry del scanner de automejora vuelve a traer contexto obligatorio
`ref_only`, write-set documental cerrado y huecos ya visibles en backlog.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-87d5c800f528-g01-d77d5415386c2758420eb50d9de1c982`
del retry
`request-ref-autoprogramming-backlog-scanner-7e8a6ae9-retry-5465c1fa2a2382d2739d96147eaafe7597f99aa10da08494d40665e47b2446c0`,
con `required_ref_action=ack_evidence_required`, write-set limitado a los tres
shards documentales y busquedas que ya encuentran T44, T249, T250, T251, T252,
T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-008
Fecha: 2026-05-27
Sintoma: retry del scanner de automejora repite contexto obligatorio
`ref_only`, write-set documental cerrado, prueba global obligatoria y huecos ya
visibles en backlog.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-757fa45b2845-g01-5ab69cd762c0b3eea5d2721713dd49d0`
con `required_ref_action=ack_evidence_required`, retry
`request-ref-autoprogramming-backlog-scanner-7e8a6ae9-retry-d64a6a4098bc947a8abac738ecfc29123c24afa6dde953cf9750f22d13b272c6`,
write-set limitado a backlog, rail errors y duplicaciones, y busquedas que ya
encuentran T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners T44/T249/T250/T251/T252/T254/T255/T256/T257/T258;
esta entrada solo evita duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-010
Fecha: 2026-05-27
Sintoma: scanner de automejora repite contexto obligatorio `ref_only`,
write-set cerrado a los tres shards documentales y huecos ya asignados a
owners pendientes.
Campo: backlog scanner, ACK estricto, required tests y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-350b93e475a8-g01-9be0de2c97caf766173a451288993ca0`
con `required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, refs opacas de worktree/branch y busquedas que ya
encuentran T44, T249, T250, T251, T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258;
esta entrada solo evita duplicar backlog programable desde un scanner
equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-009
Fecha: 2026-05-27
Sintoma: retry del scanner de automejora vuelve a traer contexto obligatorio
`ref_only`, write-set documental cerrado y huecos ya visibles en backlog.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-8f23d8e2a22e-g01-957608057af975d94befe452b621687e`
del retry
`request-ref-autoprogramming-backlog-scanner-7e8a6ae9-retry-69e1ba62ea01ade8fb67d198484d35f513e28c79077c8a7ac535bc483e60a6c6`,
con `required_ref_action=ack_evidence_required`, write-set limitado a los tres
shards documentales y busquedas que ya encuentran T44, T249, T250, T251, T252,
T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners T44/T249/T250/T251/T252/T254/T255/T256/T257/T258;
esta entrada solo evita duplicar backlog programable desde un retry
equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-009
Fecha: 2026-05-27
Sintoma: scanner de automejora repite contexto obligatorio `ref_only`,
write-set cerrado a backlog/rail errors/duplicaciones y huecos ya visibles en
backlog.
Campo: backlog scanner, ACK estricto, required tests y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-e078c7be6425160a0ccb6bb8f8b78d11`
con `required_ref_action=ack_evidence_required`, write-set limitado a los tres
shards documentales y busquedas que ya encuentran T44, T249, T250, T251, T252,
T254, T255, T256, T257 y T258 como owners pendientes.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners T44/T249/T250/T251/T252/T254/T255/T256/T257/T258;
esta entrada solo evita duplicar backlog programable desde un scanner
equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-009
Fecha: 2026-05-27
Sintoma: scanner de automejora repite contexto obligatorio `ref_only`,
write-set cerrado a backlog/rail errors/duplicaciones y huecos ya visibles en
owners pendientes.
Campo: backlog scanner, ACK estricto, required tests y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-4b676d31f48327f48f51c6ce1cc855b4`
con `required_ref_action=ack_evidence_required`, write-set limitado a los tres
shards documentales y busquedas que ya encuentran T44, T249, T250, T251, T252,
T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners T44/T249/T250/T251/T252/T254/T255/T256/T257/T258;
esta entrada solo evita duplicar backlog programable desde un scanner
equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-005
Fecha: 2026-05-27
Sintoma: scanner de automejora recibe de nuevo contexto obligatorio `ref_only`,
write-set documental cerrado y huecos ya asignados a owners pendientes.
Campo: backlog scanner, ACK estricto, required tests y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-ae7b1168a2882f5580acd71e20ec5732`
con `required_ref_action=ack_evidence_required`, write-set limitado a
backlog/rail errors/duplicaciones y busquedas que ya encuentran T44, T249,
T250, T251, T252, T254, T255, T256, T257 y T258 como owners pendientes.
Decision: no abrir otro Txx; registrar pasada documental de no-op cubierto,
resolver `ref_only` mediante lectura local/evidencia ACK y conservar
`worktree_ref`/`branch_ref` como refs opacas.
Test futuro: usar pruebas de los owners T44/T249/T250/T251/T252/T254/T255/T256/T257/T258;
esta entrada solo evita duplicar backlog programable.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-009
Fecha: 2026-05-27
Sintoma: scanner de automejora repite contexto obligatorio `ref_only`,
write-set documental cerrado, prueba global obligatoria y huecos ya visibles en
backlog.
Campo: backlog scanner, ACK estricto, required tests y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-72e651512c5a9856094dd77d5b947474`
con `correlation_id=corr-request-ref-autoprogramming-backlog-scanner-7e8a6ae9-burst-002`,
`required_ref_action=ack_evidence_required`, write-set limitado a backlog, rail
errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250, T251,
T252, T254, T255, T256, T257 y T258 como owners pendientes.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners T44/T249/T250/T251/T252/T254/T255/T256/T257/T258;
esta entrada solo evita duplicar backlog programable desde un scanner
equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-009
Fecha: 2026-05-27
Sintoma: scanner de automejora repite contexto obligatorio `ref_only`,
write-set documental cerrado y huecos ya visibles en backlog.
Campo: backlog scanner, ACK estricto, required tests y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-1957b876997d6c6e6b87bbb19f48735f`
con `required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250,
T251, T252, T254, T255, T256, T257 y T258 como owners pendientes.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners T44/T249/T250/T251/T252/T254/T255/T256/T257/T258;
esta entrada solo evita duplicar backlog programable desde un scanner
equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-009
Fecha: 2026-05-27
Sintoma: nueva pasada del scanner de automejora repite contexto obligatorio
`ref_only`, write-set cerrado a backlog/rail errors/duplicaciones y huecos ya
visibles en backlog.
Campo: backlog scanner, ACK estricto, required tests y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-41c250bd381a729b6b7a942fa03db578`
con `required_ref_action=ack_evidence_required`, write-set limitado a los tres
shards documentales y busquedas que ya encuentran T44, T249, T250, T251, T252,
T254, T255, T256, T257 y T258 como owners pendientes.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners T44/T249/T250/T251/T252/T254/T255/T256/T257/T258;
esta entrada solo evita duplicar backlog programable desde un scanner
equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-004
Fecha: 2026-05-27
Sintoma: scanner de automejora repite contexto obligatorio `ref_only`,
write-set documental cerrado y huecos ya visibles en backlog.
Campo: backlog scanner, ACK estricto, required tests y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-30cb4ab771802c1a99b591d81d13d88d`
con `required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250,
T251, T252, T253, T254, T255, T256, T257 y T258 como owners pendientes.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258;
esta entrada solo evita duplicar backlog programable desde un scanner
equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-009
Fecha: 2026-05-27
Sintoma: scanner documental repite contexto obligatorio `ref_only`, write-set
cerrado a backlog/rail errors/duplicaciones y huecos ya asignados a owners
pendientes.
Campo: backlog scanner, ACK estricto, required tests y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-55fbe00917266eb9dce3cb7ef06512d2`
con `required_ref_action=ack_evidence_required`, write-set limitado a los tres
shards documentales y busquedas que ya encuentran T44, T249, T250, T251, T252,
T254, T255, T256, T257 y T258 como owners pendientes.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners T44/T249/T250/T251/T252/T254/T255/T256/T257/T258;
esta entrada solo evita duplicar backlog programable desde un scanner
equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-009
Fecha: 2026-05-27
Sintoma: scanner de automejora repite contexto obligatorio `ref_only`,
write-set documental cerrado y huecos ya visibles en backlog.
Campo: backlog scanner, ACK estricto, required tests y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-0baf5a292701de903673675f4731b8c4`
con `required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250,
T251, T252, T254, T255, T256, T257 y T258 como owners pendientes.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners T44/T249/T250/T251/T252/T254/T255/T256/T257/T258;
esta entrada solo evita duplicar backlog programable desde un scanner
equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-009
Fecha: 2026-05-27
Sintoma: scanner de automejora vuelve a recibir contexto obligatorio
`ref_only`, write-set cerrado a los tres shards documentales y huecos ya
asignados a owners pendientes.
Campo: backlog scanner, ACK estricto, required tests y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-ed50a0e5d063ce9367242318b76013ca`
con `required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250,
T251, T252, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de T44/T249/T250/T251/T252/T254/T255/T256/T257/T258;
esta entrada solo evita duplicar backlog programable desde un scanner
equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-009
Fecha: 2026-05-27
Sintoma: retry del scanner de automejora vuelve a traer contexto obligatorio
`ref_only`, write-set documental cerrado, prueba global obligatoria y huecos ya
cubiertos por owners pendientes.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-721523601cc1-g01-a88632e247fc4b53576ed868990269ec`
con `required_ref_action=ack_evidence_required`, retry
`request-ref-autoprogramming-backlog-scanner-7e8a6ae9-retry-42fbc4ded7918354a0ba8d427df92a05da07828f0fbb064fba2a680f461c45f5`,
write-set limitado a backlog, rail errors y duplicaciones, y busquedas que ya
encuentran T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners T44/T249/T250/T251/T252/T254/T255/T256/T257/T258;
esta entrada solo evita duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-008
Fecha: 2026-05-27
Sintoma: scanner de automejora repite contexto obligatorio `ref_only`,
write-set documental cerrado y huecos ya visibles en backlog.
Campo: backlog scanner, ACK estricto, required tests y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-87d5c800f528-g01-7099771d60152a40426c551b9fed8fdb`
con `required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que encuentran T44, T249, T250, T251,
T252, T254, T255, T256, T257 y T258 como owners pendientes.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners T44/T249/T250/T251/T252/T254/T255/T256/T257/T258;
esta entrada solo evita duplicar backlog programable desde un scanner
equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-008
Fecha: 2026-05-27
Sintoma: retry del scanner de automejora vuelve a traer contexto obligatorio
`ref_only`, write-set cerrado a backlog/rail errors/duplicaciones y huecos ya
asignados a owners pendientes.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-3ddbc1385ac2-g01-a39762df8a44697f801b68e1a8195f6b`
con `required_ref_action=ack_evidence_required`, retry
`request-ref-autoprogramming-backlog-scanner-7e8a6ae9-retry-7a56734034f8d2f7cfecdb944e4e265d06361b422163b501881d49a99fce3508`,
write-set limitado a los tres shards documentales y busquedas que ya encuentran
T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners T44/T249/T250/T251/T252/T254/T255/T256/T257/T258;
esta entrada solo evita duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-008
Fecha: 2026-05-27
Sintoma: scanner de automejora repite contexto obligatorio `ref_only`,
write-set documental cerrado y huecos ya visibles en backlog.
Campo: backlog scanner, ACK estricto, required tests y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-757fa45b2845-g01-ec8943db18ad09ba2b17a1035b324e9b`
con `required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250,
T251, T252, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners T44/T249/T250/T251/T252/T254/T255/T256/T257/T258;
esta entrada solo evita duplicar backlog programable desde un scanner
equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-007
Fecha: 2026-05-27
Sintoma: scanner de automejora repite contexto obligatorio `ref_only`,
write-set documental cerrado y huecos ya visibles en backlog.
Campo: backlog scanner, ACK estricto, required tests y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-8f23d8e2a22e-g01-556dde497c2e0ff3e929391ffeb28c93`
con `required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250,
T251, T252, T254, T255, T256 y T257.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners T44/T249/T250/T251/T252/T254/T255/T256/T257;
esta entrada solo evita duplicar backlog programable desde un scanner
equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-005
Fecha: 2026-05-27
Sintoma: scanner de automejora repite contexto obligatorio `ref_only`, write-set
documental cerrado, prueba global obligatoria y huecos ya cubiertos por owners
pendientes del backlog.
Campo: backlog scanner, ACK estricto y shards documentales de backlog.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-87d5c800f528-g01-8049fe783ba558122c38feeec2532ceb`
con `required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que encuentran T44, T249, T250, T251,
T252, T254, T255 y T256 como owners pendientes.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar refs de worktree/branch como
opacas.
Test futuro: usar pruebas de los owners T44/T249/T250/T251/T252/T254/T255/T256;
esta entrada solo evita duplicar backlog programable desde un scanner
equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: FILE-BUDGET-20260527-006
Fecha: 2026-05-27
Sintoma: el parser/cargador federado de backlog supera el limite operativo de
300 lineas y acumula politica de fuentes locales, cuarentena y refs de scanner.
Campo: `cmd/orquesta-server/idle_self_improvement_federated_backlog_v0.go`.
Payload minimo: `wc -l` muestra 311 lineas; el fichero mezcla catalogo de
fuentes, lectura de documentos, parser del indice federado, parser de entradas
locales, clasificacion de estado, construccion de secciones y contexto de
scanner.
Decision: abrir backlog T257 para dividir el fichero por responsabilidad local
sin reabrir identidad, dedupe, preflight, alcance de pruebas ni ids humanos del
backlog.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`.
Estado: cerrado 2026-05-27; split ejecutado en `cmd/orquesta-server` con
carga federada, parser de indice, parser local y clasificacion de estado en
ficheros separados, sin cambiar el contrato documental del backlog.
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-006
Fecha: 2026-05-27
Sintoma: retry del scanner de automejora vuelve a traer contexto obligatorio
`ref_only`, write-set cerrado a backlog/rail errors/duplicaciones y huecos ya
cubiertos por owners pendientes.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-350b93e475a8-g01-a9fca45e81d848b2c2a00b8349e4817b`
con `required_ref_action=ack_evidence_required`, retry
`request-ref-autoprogramming-backlog-scanner-7e8a6ae9-retry-d0cf1ea77c2b6898418f56ac96139dcb5cb606948a8e3adc89ed26b4a9b1f02c`,
write-set limitado a los tres shards documentales y busquedas que ya encuentran
T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners T44/T249/T250/T251/T252/T254/T255/T256/T257/T258;
esta entrada solo evita duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-004
Fecha: 2026-05-27
Sintoma: nueva pasada del scanner de automejora repite contexto obligatorio
`ref_only`, write-set documental cerrado y huecos ya visibles en backlog.
Campo: backlog scanner, ACK estricto, required tests y merge documental.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-721523601cc1-g01-520f979fdde9866f084ddfa1d5cc7425`
con `required_ref_action=ack_evidence_required`, write-set limitado a
`docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`,
`docs/rail_errors_observados_2026-05-23.md` y
`docs/duplicaciones_railes_pendientes_2026-05-24.md`, y busquedas que ya
encuentran T44, T249, T250, T251, T252, T254, T255 y T256.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners T44/T249/T250/T251/T252/T254/T255/T256;
esta entrada solo evita duplicar backlog programable desde el scanner.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: APP-DIRECTOR-SERVICE-TEST-SUITE-FILE-BUDGET-20260527-001
Fecha: 2026-05-27
Sintoma: la suite historica de `orquesta-app-director-service` conserva varios
tests Go muy por encima del limite operativo de 300 lineas.
Campo: `modulos/orquesta-app-director-service/*_test.go`.
Payload minimo: `operational_director_v0_test.go` supera 5200 lineas,
`operational_closure_v0_test.go` supera 2500 lineas y otros tests de replay,
decision source, required tests, retry y replan siguen por encima de 300
lineas desde el baseline de T52.
Decision: abrir backlog T258 para partir la suite por escenario de test, sin
reabrir contratos productivos ni duplicar el split productivo ya cerrado por
T52.
Test futuro:
`go test -count=1 ./modulos/orquesta-app-director-service`.
Estado: cerrado localmente el 2026-05-27 por T258; la suite de
`orquesta-app-director-service` quedo repartida por escenario y la prueba focal
es `go test -count=1 ./modulos/orquesta-app-director-service`.
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-003
Fecha: 2026-05-27
Sintoma: nueva pasada del scanner documental recibe el mismo patron de contexto
`ref_only`, pruebas obligatorias globales y write-set cerrado a shards de
backlog.
Campo: backlog scanner, ACK estricto, required tests y merge documental.
Payload minimo: paquete con `required_ref_action=ack_evidence_required`,
write-set limitado a backlog/rail errors/duplicaciones y busquedas que ya
encuentran owners T44, T249, T250, T251, T252, T254, T255, T227 y T241.
Decision: no abrir otro Txx; registrar no-op cubierto y cerrar con ACK que
declare `contexto_ref_only_resuelto` por lectura local/evidencia explicita.
Test futuro: usar las pruebas de los owners citados; esta entrada solo evita
duplicar backlog programable desde un scanner documental equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: PROCESS-RUNTIME-FILE-BUDGET-20260527-001
Fecha: 2026-05-27
Sintoma: el conector neutral de proceso local supera el limite operativo de 300
lineas en sus dos ficheros principales.
Campo: `modulos/orquesta-runtime/process_runtime_connector_v0.go` y
`modulos/orquesta-runtime/process_runtime_connector_types_v0.go`.
Payload minimo: `wc -l` muestra 313 y 318 lineas; la frontera mezcla DTOs,
errores publicos, validacion de launch/adoption/snapshot, allowlist de env,
guardas de shell/rutas, launch/adopt/wait y stop cooperativo/escalado.
Decision: abrir backlog T256 para split local del conector de proceso sin
cambiar semantica ni reabrir `NEUTRAL-PROCESS-STOP-E2E`.
Test futuro: `go test -count=1 ./modulos/orquesta-runtime`.
Estado: corregido 2026-05-27; el planner omite el default global para scanners
documentales, anade prueba focal documental y conserva la guarda `ref_only` con
procedencia visible en `ContextRefs`.
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-003
Fecha: 2026-05-27
Sintoma: nueva pasada del scanner documental encuentra contexto `ref_only`,
write-set cerrado a backlog/rail errors/duplicaciones y huecos ya abiertos en
owners pendientes.
Campo: backlog scanner, ACK estricto, dedupe de Txx y evidencia documental.
Payload minimo: paquete con `required_ref_action=ack_evidence_required`,
write-set documental, busquedas que encuentran T44, T250, T251, T252, T254 y
T255, y evidencia `FILE-BUDGET-20260527-004` ya registrada.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver contexto por
lectura local/evidencia ACK y dejar la implementacion a los owners existentes.
Test futuro: usar pruebas de T44/T250/T251/T252/T254/T255; esta entrada solo
evita duplicar backlog programable desde el scanner.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-003
Fecha: 2026-05-27
Sintoma: nueva pasada del scanner de automejora repite paquete con contexto
`ref_only`, write-set documental cerrado y huecos ya visibles en backlog.
Campo: backlog scanner, ACK estricto, generacion de tareas Txx y merge
documental.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-0f01e601f615c5bf38893ec9f09b4f35`
con `required_ref_action=ack_evidence_required`, write-set limitado a
`docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`,
`docs/rail_errors_observados_2026-05-23.md` y
`docs/duplicaciones_railes_pendientes_2026-05-24.md`, y busquedas que ya
encuentran T44, T249, T250, T251, T252, T254 y T255.
Decision: no abrir otro Txx; registrar pasada documental de no-op cubierto y
cerrar con ACK que declare `contexto_ref_only_resuelto` mediante lectura local
y evidencia explicita.
Test futuro: usar pruebas de los owners T44/T249/T250/T251/T252/T254/T255; esta
entrada solo evita duplicar backlog programable.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: SERVER-SHUTDOWN-USECASE-FILE-SPLIT-20260527-001
Fecha: 2026-05-27
Sintoma: `modulos/orquesta-server-shutdown/shutdown_v0.go` supera 300 lineas y
mezcla autorizacion, cola, targets, checkpoint, deadline, stop, stats,
supervisor y resumen publico en el mismo fichero.
Campo: `modulos/orquesta-server-shutdown/shutdown_v0.go`.
Payload minimo: una nueva regla de shutdown obliga a tocar el caso de uso y
empuja el fichero a seguir creciendo, con riesgo de duplicar politica de
checkpoint, escalado o readiness ya cubierta por T30/T225/T239.
Decision: abrir backlog T256 para split local por responsabilidades manteniendo
`ShutdownServerV0` como fachada, sin meter runtime/procesos ni adaptadores
concretos en el modulo.
Test futuro:
`go test -count=1 ./modulos/orquesta-server-shutdown ./modulos/orquesta-server ./cmd/orquesta-server`.
Estado: cerrado 2026-05-27 por T256
Rework revision 2026-05-27: cierre sincronizado con la entrada canonica T256;
no programar la variante `server-shutdown-usecase-file-split-before-growth`
como tarea separada.
Rework entrega 2026-05-27:
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-846263ad27220f7ca7d97aeec4d5e7d1`
solo valida el cierre existente: no reabre T256, no crea owner nuevo y resuelve
`ref_only` mediante lectura local/evidencia ACK.
Rework adicional 2026-05-27:
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-40ea96487c3783404f7d6a48b0cb8411`
mantiene el cierre existente y deja la variante `before-growth` absorbida por
T256 canonico.
Rework final 2026-05-27:
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-247e6d495674dee5a1b996468956a6be`
confirma el mismo cierre, resuelve `ref_only` en ACK y no reabre codigo ni
backlog nuevo.
Rework correctivo 2026-05-27:
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-ced6b146bcf5b10e5253ef166816de66`
mantiene T256 cerrado, conserva la variante `before-growth` absorbida y resuelve
el contexto obligatorio por lectura local y evidencia ACK.
Rework de reemplazo 2026-05-27:
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-75ba20e2214f2eaaea16f312c68208f0`
revalida el mismo cierre: T256 sigue cerrado, la variante `before-growth` no se
programa por separado y el contexto `ref_only` se resuelve por lectura local y
evidencia ACK.
Rework de evaluacion 2026-05-27:
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-f878e21cfa5e67e7481d20a5f9bd24c5`
mantiene el cierre: T256 sigue cerrado, la variante `before-growth` no se
programa por separado y el contexto `ref_only` se resuelve por lectura local y
evidencia ACK.
Rework de evaluacion 2026-05-27:
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-3d04cc0bcd1f59b871869ec824d2c996`
mantiene T256 cerrado, conserva `before-growth` absorbido por el owner canonico
y resuelve `ref_only` por lectura local y evidencia ACK.
Rework de evaluacion 2026-05-27:
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-46d0d95f62343cc1e51b9ca398f83f84`
mantiene T256 cerrado, conserva la variante `before-growth` absorbida por el
owner canonico y resuelve `ref_only` por lectura local y evidencia ACK.
Rework de evaluacion 2026-05-27:
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-33ca4ea060412afa902e83e3765c1d23`
mantiene T256 cerrado, conserva la variante `before-growth` absorbida por el
owner canonico y resuelve `ref_only` por lectura local y evidencia ACK.
Rework de reemplazo 2026-05-27:
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-b6686ba67b0d901948d8ce067d8771ec`
mantiene T256 cerrado, conserva la variante `before-growth` absorbida por el
owner canonico y resuelve `ref_only` por lectura local y evidencia ACK.
Rework de evaluacion 2026-05-27:
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-bff7aa189e7e24662071cf55dffcb95f`
mantiene T256 cerrado, conserva la variante `before-growth` absorbida por el
owner canonico y resuelve `ref_only` por lectura local y evidencia ACK.
Rework de reemplazo 2026-05-27:
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-55de335961e2cd74deaeda6d56c611bb`
mantiene T256 cerrado, conserva la variante `before-growth` absorbida por el
owner canonico y resuelve `ref_only` por lectura local y evidencia ACK.
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-003
Fecha: 2026-05-27
Sintoma: nueva pasada del scanner de automejora repite el mismo paquete con
contexto `ref_only`, write-set documental cerrado y huecos ya materializados en
owners pendientes.
Campo: backlog scanner, ACK estricto, required tests y merge documental.
Payload minimo: paquete con `required_ref_action=ack_evidence_required`,
write-set limitado a backlog/rail errors/duplicaciones y busquedas que ya
encuentran T44, T249, T250, T251, T252, T254 y T255 como owners pendientes o
evidencia registrada.
Decision: no abrir otro Txx; registrar no-op cubierto y cerrar con ACK que
declare `contexto_ref_only_resuelto` mediante lectura local y evidencia
explicita.
Test futuro: usar las pruebas declaradas por T44/T249/T250/T251/T252/T254/T255;
esta entrada solo evita duplicar backlog programable.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-002
Fecha: 2026-05-27
Sintoma: nueva pasada del scanner de automejora repite contexto `ref_only`,
write-set documental cerrado y huecos ya visibles en backlog.
Campo: backlog scanner, ACK estricto y merge documental.
Payload minimo: paquete con `required_ref_action=ack_evidence_required`,
write-set limitado a backlog/rail errors/duplicaciones y busquedas que ya
encuentran T44, T249, T250, T251, T252 y T254 como owners pendientes.
Decision: no abrir otro Txx; registrar pasada documental de no-op cubierto y
cerrar con ACK que declare `contexto_ref_only_resuelto` mediante lectura local y
evidencia explicita.
Test futuro: usar pruebas de los owners T44/T249/T250/T251/T252/T254; esta
entrada solo evita duplicar backlog programable.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: FILE-BUDGET-20260527-004
Fecha: 2026-05-27
Sintoma: el contrato documental OPES del bridge supera el limite operativo de
300 lineas y es frontera natural para nuevas reglas editoriales.
Campo: `modulos/orquesta-opes-bridge/document_plan_contract_v0.go`.
Payload minimo: `wc -l` muestra 321 lineas; el fichero mezcla politicas
editoriales, HTML, plantilla, contrato `domain_document_plan.v0`, derivacion por
nivel, metodo de asimilacion y requisitos de calidad.
Decision: abrir backlog T255 para dividir el contrato por responsabilidad local
del bridge, sin mover reglas OPES al nucleo ni reabrir T204/T205/T206.
Test futuro: `go test -count=1 ./modulos/orquesta-opes-bridge`.
Estado 2026-05-27: cerrado. El contrato documental OPES se dividio en owners
locales para contrato `document_plan`, politica editorial global, politica/plantilla
HTML y metodologia/calidad OPES; los ficheros Go del modulo quedan bajo 300
lineas y el test focal del bridge pasa.
Rework 2026-05-27: backlog T255 queda marcado como cerrado de forma explicita
para evitar reprogramacion por estado textual stale.
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-003
Fecha: 2026-05-27
Sintoma: nueva pasada del scanner de automejora repite contexto `ref_only`,
write-set documental cerrado, pruebas globales y huecos ya visibles en backlog.
Campo: backlog scanner, ACK estricto, required tests y merge documental.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-b67a7bf97034fe5d7abbfaf0e628c3c5`
con `required_ref_action=ack_evidence_required`, write-set limitado a
backlog/rail errors/duplicaciones y busquedas que ya encuentran T44, T249,
T250, T251, T252, T254 y T255 como owners pendientes.
Decision: no abrir otro Txx; registrar pasada documental de no-op cubierto,
resolver contexto requerido por lectura local/evidencia ACK y dejar las
mejoras reales a los owners pendientes.
Test futuro: usar pruebas de los owners T44/T249/T250/T251/T252/T254/T255; esta
entrada solo evita duplicar backlog programable.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-NO-NEW-GAP-20260527-001
Fecha: 2026-05-27
Sintoma: scanner de automejora encuentra de nuevo el mismo paquete con contexto
`ref_only`, pruebas globales documentales y colisiones Txx ya registradas.
Campo: backlog scanner, ACK estricto, required tests y read model de tareas Txx.
Payload minimo: paquete con `required_ref_action=ack_evidence_required`,
write-set documental y busquedas que detectan `## T250`, `## T251` y `## T252`
duplicados o vecinos.
Decision: no abrir un Txx adicional; conservar evidencia y remitir a T44, T249,
T250, T251 y T252. La resolucion de esta pasada es lectura local/evidencia ACK,
no programacion de otro owner solapado.
Test futuro: usar las pruebas declaradas por esos Txx owners; esta entrada solo
evita que el scanner genere backlog duplicado.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-TASK-ID-COLLISION-20260527-001
Fecha: 2026-05-27
Sintoma: el backlog vivo contiene varias secciones ejecutables con el mismo id
humano `## T250` para fronteras distintas.
Campo: parser de backlog, planner residente, roadmap/MCP, ACK/cierre por task
ref y merge lease documental.
Payload minimo: `rg '^## T250 ' docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
devuelve mas de una seccion: epoch federado, split de
`orquesta-runtime-codex-delivery`, dedupe de propuestas y canonicalizacion de
solapes.
Decision: abrir backlog T254 para indice aditivo de colisiones/aliases,
`task_entry_ref` estable por seccion y bloqueo publico cuando `Txx` sea ambiguo.
No renumerar historico.
Test futuro:
`go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-server ./modulos/orquesta-mcp ./cmd/orquesta-server`.
Estado: registrado, pendiente
```

```text
ID: BACKLOG-TASK-ID-ALLOC-20260527-001
Fecha: 2026-05-27
Sintoma: scanners concurrentes de backlog pueden insertar tareas `Txx` con
huecos visibles, colisiones potenciales o cierre por numero humano sin reserva
causal explicita.
Campo: `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md` y merge lease
del scanner.
Payload minimo: secuencia documental donde `T244` salta a `T247`, varias
secciones `Escaneo backlog 2026-05-27 decimosexta pasada` preceden tareas
distintas y no hay `task_id_ref` reservado por request/correlation.
Decision: cerrado en servidor residente con `task_id_ref`,
`reservation-ref-backlog-task-id-*`, rango `Txx` derivado del epoch documental y
colision publica `backlog_task_id_collision`; los encabezados de escaneo no
cuentan como ids ejecutables.
Test ejecutado: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`.
Estado: cerrado
```

```text
ID: BACKLOG-TASK-ID-ALLOC-20260527-001-HISTORICO
Decision: abrir backlog T250 para reservar ids ejecutables `Txx` por epoch
documental, detectar `backlog_task_id_collision`/
`backlog_task_id_allocation_gap` y bloquear con rebase o `CONSULTA AL DIRECTOR`
sin renumerar historico.
Test futuro: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`.
Estado: registrado, pendiente
```

```text
ID: FILE-BUDGET-20260527-003
Fecha: 2026-05-27
Sintoma: bridge de autoprogramacion del stack Codex supera el limite operativo
de 300 lineas y concentra varias responsabilidades de composicion.
Campo: `modulos/orquesta-app-codex-stack/autoprogramming_bridge_v0.go`.
Payload minimo: `wc -l` muestra 341 lineas; el fichero mezcla normalizacion de
request, run, task store, wait refs, continue request, replay y plan-state.
Decision: abrir backlog T244 para partir por responsabilidad antes de anadir
mas reglas de automejora residente en el stack Codex.
Test futuro: `go test -count=1 ./modulos/orquesta-app-codex-stack`.
Estado: registrado, pendiente
```

```text
ID: GUARDIAN-STATE-LEASE-20260527-001
Fecha: 2026-05-27
Sintoma: promocion/restauracion del guardian no declara exclusion mutua durable
por `state_dir`.
Campo: `check-promote`, `restore-last-good`, `current_bin`, `last_good_bin` y
manifiestos de guardian.
Payload minimo: dos invocaciones solapadas sobre el mismo `state_dir` y
`current_bin`, una promocion y una restauracion, con manifiestos atomicos
individuales pero sin lease comun visible.
Decision aplicada 2026-05-27: `cmd/orquesta-guardian` reclama un lease durable
bajo `state_dir` para `check-promote`, `restore-last-good` y `shutdown-server`,
publica `guardian_promotion_lease_busy`/`guardian_promotion_lease_lost` con
receipt compacto y usa reloj de config para timestamps testeables. El servidor
trata lease busy/lost como efecto `blocked` retryable y no lo convierte en
promocion exitosa ni en reparacion Codex automatica.
Test: `go test -count=1 ./cmd/orquesta-guardian ./cmd/orquesta-server`.
Estado: cubierto por T237.
```

```text
ID: AUTOPROGRAMMING-FILE-BUDGET-20260527-001
Fecha: 2026-05-27
Sintoma: politicas productivas de autoprogramacion quedan cerca del limite de
300 lineas y el siguiente cambio funcional puede mezclar responsabilidades o
romper el rail de tamano.
Campo: `modulos/orquesta-autoprogramming/autoprogramming_review_gate_policy_v0.go`
y `modulos/orquesta-autoprogramming/autoprogramming_partition_policy_v0.go`.
Payload minimo: `wc -l` muestra ambos ficheros productivos en 298 lineas.
Decision aplicada 2026-05-27: T238 dividio las politicas por responsabilidad:
review gate separa resultado/acciones, clasificacion de issue codes y stems
canonicos; particionado separa plan base, steps/bloqueos por trabajo vivo y
matching de paths/aliases.
Test: `go test -count=1 ./modulos/orquesta-autoprogramming`.
Estado: cubierto por T238.
```

```text
ID: CODEX-SHUTDOWN-CHECKPOINT-ATTEMPT-20260527-001
Fecha: 2026-05-27
Sintoma: ACK de checkpoint Codex puede ser reutilizable entre intentos de
shutdown del mismo run/agente.
Campo: `orquesta_shutdown_request.json` y `agent_shutdown_checkpoint_ack.json`.
Payload minimo: dos llamadas de shutdown cooperativo con mismo `run_ref` y
`agent_ref`, pero distinta razon/evidencia/secuencia; el `checkpoint_ref`
deterministico permite que un ACK previo siga correlando.
Decision: backlog T239 cerrado localmente el 2026-05-27: request y ACK de
checkpoint transportan `shutdown_attempt_ref`; un ACK stale queda pendiente con
evidencia compacta `shutdown_checkpoint_attempt_mismatch`.
Rework 2026-05-27: la correccion conserva el cierre T239 y resuelve el contexto
obligatorio `ref_only` por lectura local del paquete mas evidencia explicita en
ACK, sin relanzar agente padre.
Test futuro: `go test -count=1 ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Estado: cerrado localmente
```

```text
ID: ORCH-CORE-FILE-BUDGET-20260527-001
Fecha: 2026-05-27
Sintoma: provider de replan por quality gate supera el limite operativo de 300
lineas y concentra responsabilidades de aplicacion.
Campo: `modulos/orquesta-orchestration-core/quality_gate_replan_candidate_provider_v0.go`.
Payload minimo: `find modulos cmd -name '*.go' -type f ! -name '*_test.go' -exec wc -l {} + | sort -nr | head -60` muestra 335 lineas.
Decision: abrir backlog T253 para split por responsabilidad dentro de
`orquesta-orchestration-core`, conservando puertos y sin mover runtime/proveedor
al nucleo.
Rework 2026-05-27: T253 queda cerrado localmente; el provider se separo en
entrada/base, proyeccion causal, followups y politica local, y el contexto
obligatorio `ref_only` se resolvio por lectura local del paquete mas evidencia
explicita en ACK.
Test futuro: `go test -count=1 ./modulos/orquesta-orchestration-core`.
Estado: cerrado localmente
```

```text
ID: BACKLOG-PROPOSAL-DUP-20260527-001
Fecha: 2026-05-27
Sintoma: scanners concurrentes pueden anadir Txx solapados para la misma
frontera antes de que exista dedupe estructurado de propuestas.
Campo: secciones Txx del backlog, evidencia de scanner y merge lease documental.
Payload minimo: pares T244/T247 o T248/T249 con owner/write-set/criterios
vecinos y titulos distintos.
Decision: abrir backlog T250 para huella de propuesta por owner, write-set,
objetivo, criterios, tests y evidencia; duplicados deben bloquear/rebasear con
reason code publico en vez de encolarse como tareas independientes.
Test futuro: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`.
Estado: cerrado 2026-05-27; el planner publica
`duplicate_backlog_proposal_fingerprint`, conserva evidence refs compactas y
bloquea propuestas equivalentes vivas sin borrar ni renumerar historico.
```

```text
ID: IDLE-SELF-IMPROVEMENT-PROVIDER-AUTH-20260527-001
Fecha: 2026-05-27
Sintoma: automejora residente puede quedar bloqueada por autenticacion externa
del proveedor con razon `provider_auth_blocked`, pero sin contrato comun de
recuperacion operativa.
Campo: estado publico del servidor, auditoria de idle self-improvement,
lectura de runs persistidas y cola de autoprogramacion.
Payload minimo: run activa con `AgentAssessmentProjection` critica
`ask_director`, agentes perdidos o preguntas de director, y cola idle que vuelve
a evaluar backlog.
Decision: abrir backlog T241 para diagnostico publico compacto, accion de
operador por composicion, reintento idempotente y no duplicacion de scanners
mientras el proveedor siga bloqueado.
Test futuro: `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server ./modulos/orquesta-app-codex-stack`.
Estado 2026-05-27: cerrado. `provider_auth_blocked` queda como contrato
operacional recuperable: estado publico con reason/evidencias/refs acotadas,
accion de operador por adaptador y reintento idempotente tras evidencia compacta
de proveedor recuperado.
Rework 2026-05-27: sincronizado con el cierre del backlog T241; la evidencia
vigente es el contrato publico compacto del servidor residente, sin secretos,
paths locales ni lectura de credenciales desde el nucleo.
```

```text
ID: BACKLOG-SCAN-ENTRY-DUPLICATE-20260527-001
Fecha: 2026-05-27
Sintoma: el backlog contiene titulos de escaneo fuera de orden y un titulo
duplicado, por ejemplo `Escaneo backlog 2026-05-27 decimocuarta pasada`.
Campo: secciones Markdown de evidencia de scanner, merge lease documental y
correlacion de cierres por linea/hash.
Payload minimo: dos secciones `## Escaneo backlog ...` con mismo dia y ordinal,
o una secuencia donde `decimoquinta` aparece antes que `decimocuarta`.
Decision: abrir backlog T249 para identidad estable de entrada de scanner,
deteccion de duplicados/saltos y bloqueo/rebase sin renumerar historico.
Test futuro: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`.
Estado: mitigado 2026-05-27 por T249; el planner deriva `scan_entry_ref`,
digest y reason codes `backlog_scan_entry_duplicate` /
`backlog_scan_entry_order_ambiguous` sin renumerar historico.
Rework 2026-05-27: revision corregida por
`agent-ref-task-ref-review-rework-task-autoprogramming-1c574ac7c424-g01-5cba0f0171e79bd80bbbcf4c49f5bc8a`;
se conserva T249 como owner, se resuelve contexto `ref_only` por lectura
local/evidencia ACK y no se crea backlog duplicado.
```

```text
ID: RUNTIME-CODEX-DELIVERY-FILE-BUDGET-20260527-001
Fecha: 2026-05-27
Sintoma: ficheros productivos de `orquesta-runtime-codex-delivery` siguen por
encima del limite operativo de 300 lineas tras el baseline T90.
Campo: `progress_state_v0.go`, `progress_source_v0.go` y `source_v0.go`.
Payload minimo: medicion focal muestra 310, 331 y 314 lineas respectivamente;
los ficheros mezclan estado de progreso, lectura de fuente Codex, observaciones,
ACK/delivery y replay.
Decision: backlog T252 cerro el split por responsabilidad antes de anadir mas
reglas de observacion, ACK estricto o redaccion en runtime Codex delivery.
Test futuro: `go test -count=1 ./modulos/orquesta-runtime-codex-delivery`.
Estado: mitigado 2026-05-27; los ficheros productivos objetivo quedan por
debajo de 300 lineas y el adaptador sigue sin mover filesystem, Codex, HOME,
proveedor ni rutas locales al nucleo.
```

```text
ID: BACKLOG-TASK-NUMBER-COLLISION-20260527-001
Fecha: 2026-05-27
Sintoma: el backlog contiene varios encabezados `## T250` para tareas distintas,
lo que vuelve ambiguo usar el numero humano como identidad ejecutable.
Campo: secciones `Txx` del backlog, reserva documental del scanner y ACK de
autoprogramacion.
Payload minimo: tres secciones `## T250` con objetivos diferentes
(`runtime-codex-delivery-progress-source-file-split`,
`backlog-proposal-deduplication-fingerprint` y
`backlog-task-overlap-canonical-merge-policy`).
Decision: abrir backlog T252 para reservar `task_id_ref` causal antes de
escribir `Txx`, bloquear colisiones con reason publico y conservar historico
duplicado por linea/hash/fingerprint sin renumerarlo.
Test futuro: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`.
Estado: registrado, pendiente
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-003
Fecha: 2026-05-27
Sintoma: nueva pasada del scanner de automejora vuelve a recibir contexto
`ref_only`, write-set documental cerrado y huecos ya registrados en backlog.
Campo: backlog scanner, ACK estricto, owners T44/T249/T250/T251/T252/T254/T255.
Payload minimo: paquete con `required_ref_action=ack_evidence_required`,
write-set limitado a backlog/rail errors/duplicaciones y busquedas que ya
encuentran owners pendientes para contexto por ref, dedupe/preflight,
colisiones de `Txx` y split OPES T255.
Decision: no abrir otro Txx; registrar pasada documental de no-op cubierto y
cerrar con ACK que declare `contexto_ref_only_resuelto` mediante lectura local
y evidencia explicita.
Test futuro: usar las pruebas declaradas por esos Txx owners; esta entrada solo
evita duplicar backlog programable.
Estado: registrado, cubierto por owners pendientes
```

## Candidatos pendientes de convertir en matriz

```text
ID: RAIL-CAND-CORE-EVENTS-001
Origen: revision paralela 2026-05-23.
Casos: `RunBlocked.summary` con `completion`, `token budget`, `prompt policy ref`
o `transcript policy ref`.
Decision pendiente: permitir refs/politicas opacas pero seguir rechazando
contenido crudo (`access_token=...`, keys `prompt`, `raw_text`, `transcript`).
Test futuro: `TestRailEventPayloadExternalMatrixV0`.
```

```text
ID: RAIL-CAND-WORKFLOW-TASK-001
Origen: revision paralela 2026-05-23.
Casos: `WorkflowTaskV0` y `CreateMicrotask` con write-set/context refs de
`modulos/orquesta-runtime`, `provider interface`, `modelado de dominio`,
`git diff` y `token budget`.
Decision pendiente: permitir referencias opacas de arquitectura/repo; reservar
rechazo para secretos efectivos o paths inseguros.
Test futuro: `TestRailWorkflowTaskExternalMatrixV0`.
```

```text
ID: RAIL-CAND-CONTEXT-001
Origen: revision paralela 2026-05-23.
Casos: `ContextBundleRequestV0` con `task_kind=web_application`,
`phase=programming`, `capacity_level=normal` y
`write_set=modulos/orquesta-runtime`.
Decision pendiente: normalizar alias razonables en adaptador/director; no
rechazar el modulo real `orquesta-runtime` por palabra.
Test futuro: matriz rapida en `orquesta-context`.
```

```text
ID: RAIL-CAND-DIRECTOR-AGENT-001
Origen: revision paralela 2026-05-23.
Casos: `DirectorAgentDecisionV0` con `summary` que menciona `codex adapter`,
`runtime`, `model`, `provider`, y `record_review_result` con
`accepted_with_notes`.
Decision pendiente: distinguir alias reparables y refs opacas de proveedor real;
normalizar estados equivalentes fuera del core.
Test futuro: matriz rapida en `orquesta-director-agent`.
```

```text
ID: RAIL-CAND-RUNTIME-001
Origen: revision paralela 2026-05-23.
Casos: rol con espacios (`frontend engineer`), `runtime_kind=local`,
`capacity=normal`, `provider_ref=openai`, `model_ref=gpt-5`.
Decision pendiente: normalizar alias de rol/runtime/capacidad; mantener
proveedor/modelo concretos como refs opacas o adaptador opt-in.
Test futuro: matriz rapida en `orquesta-runtime`.
```

```text
ID: FLAKE-CAND-SERVER-001
Origen: revision paralela 2026-05-23.
Casos: stress de `TestCodexLaunchDirectorWaveCommandV0RecursiveFakeRuntimeEjecutableConLinaje`
con `-count=50 -shuffle=on`, bloque `codex_wave` con procesos fake y `-race`
en `orquesta-runtime`.
Decision pendiente: crear harness rapido de flakes y esperar cierre real de
stdout/ficheros en vez de asumir que `last_message` implica stdout completo.
Test futuro: `./scripts/test_flakes_fast.sh`.
```

```text
ID: RAIL-CAND-PUBLIC-MUTATION-IDENTITY-001
Origen: scanner backlog 2026-05-27 undecima pasada.
Casos: MCP, web, CLI y servidor tienen reglas locales para `request_id`,
`correlation_id`, `X-Correlation-ID`, `Idempotency-Key`,
`X-Idempotency-Key`, ids generados por reloj y mutaciones de `/ops`.
Decision pendiente: unificar owner de identidad publica de mutaciones y
lecturas read-only para no derivar idempotency/correlacion de forma distinta en
cada adaptador; bloquear solo mutaciones sin idempotency efectiva cuando el
contrato lo exija.
Actualizacion 2026-05-27: owner compartido implementado en
`modulos/orquesta-server/publicidentity`; MCP, web, CLI y servidor consumen
headers/normalizacion compartidos para el alcance T236. Las rutas read-only por
POST conservan correlacion sin exigir idempotency.
Test futuro: `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-cli ./modulos/orquesta-web ./modulos/orquesta-server ./cmd/orquesta-server`.
Backlog: `T236 public-mutation-identity-contract`.
```

```text
ID: RAIL-CAND-DETAIL-REACTIVATION-001
Origen: scanner backlog 2026-05-24.
Casos: `ORQUESTA_DETAIL_PROHIBITED_RAILS=off` quedo como apertura temporal por
defecto tras `director_supervised_burst_step_input: detalle_prohibido`.
Decision pendiente: no reactivar el rail global por env sin matriz externa por
campo. La reactivacion debe permitir refs opacas, vocabulario operativo y
politicas de prompt/transcript, y cortar solo valores sensibles efectivos o
material crudo.
Test futuro: `./scripts/test_rails_fast.sh` mas focos de `orquesta-context`,
`orquesta-director-agent`, `orquesta-core-workflow` y `cmd/orquesta-server`.
Backlog: `T15 detail-rails-reactivation`.
Actualizacion 2026-05-24 cuarta pasada: `cmd/orquesta-server` ya reactiva por
defecto el rail con `ORQUESTA_DETAIL_PROHIBITED_RAILS=on` y scope acotado. El
pendiente ya no es "reactivar todo", sino sincronizar docs/matriz y evitar que
esa reactivacion se use para persistir payloads crudos o bloquear vocabulario
operativo fuera de los scopes probados.
Actualizacion 2026-06-02: esta foto queda superada. Los rails de detalle quedan
offline por defecto mediante `ORQUESTA_RAILS_MODE=offline` y el servidor
proyecta `ORQUESTA_DETAIL_PROHIBITED_RAILS=off`. No hay reactivacion por env;
cualquier vuelta a bloqueo requiere tarea futura y cambio explicito de
codigo/configuracion.
```

```text
ID: RAIL-CAND-AUDIT-PAYLOADS-001
Origen: scanner backlog 2026-05-24 tercera pasada.
Casos: la auditoria JSONL del servidor ya guarda metadata HTTP y diagnostics de
drain, pero el pendiente documentado propone captura opt-in de payloads HTTP
completos. Sin politica por campo, esa captura puede duplicar cuerpos grandes,
transcripts, prompt material, tokens o datos privados en una persistencia
paralela.
Decision pendiente: mantener payload completo apagado por defecto; si se activa,
pasar por sanitizador/rail por campo, retencion corta, evidencia de redaccion y
frontera local de composicion. No usar `ORQUESTA_DETAIL_PROHIBITED_RAILS=off`
como autorizacion para persistir material crudo.
Test futuro: focos de `orquesta-server`, `cmd/orquesta-server` y web/API con
payload sensible simulado y auditoria filtrable.
Backlog: `T19 server-audit-ops-surface`.
```

```text
ID: RAIL-CAND-SQL-REAL-DIALECT-001
Origen: scanner backlog 2026-05-24 tercera pasada.
Casos: el adaptador `orquesta-domain-work-sql` acepta `database/sql` inyectado,
pero el bundle real futuro necesitara DSN, driver, schema y errores de dialecto.
El vocabulario `db`, `sql`, `driver` o `dsn` no debe bloquear refs opacas, pero
valores reales de DSN, passwords o URLs con credenciales no pueden acabar en
eventos, ACKs, audit logs ni docs de smoke.
Decision pendiente: probar dialectos reales solo en composicion opt-in con DSN
externo y redaccion obligatoria; el core/domain-work no importa drivers ni
persistencia global.
Test futuro: contract suite de `DomainWorkJobRecordStorePortV0` contra DB
temporal y caso negativo de secreto en DSN/auditoria.
Backlog: `T20 domain-work-sql-real-dialect-bundle`.
```

```text
ID: RAIL-CAND-REAL-SMOKE-GUARDS-001
Origen: scanner backlog 2026-05-24 tercera pasada.
Casos: la matriz marca smokes reales con distintos niveles de guarda. Algunas
rutas tienen confirm env y OPES temporal obligatorio; `USAGE-METRICS-REAL`
declara que su script no tiene guarda propia aunque puede lanzar Codex real.
Decision pendiente: catalogar smokes por riesgo y exigir confirmacion explicita
para proveedor, OPES, DB, red o efectos externos antes de que el residente los
pueda seleccionar. Un smoke real sin guarda debe quedar excluido de ejecucion
automatica o fallar temprano con bloqueo verificable.
Test futuro: `bash -n scripts/*.sh`, prueba de catalogo en `cmd/orquesta-server`
y caso que rechaza ejecutar smoke real sin confirm env.
Backlog: `T21 real-smoke-guards-and-catalog`.
```

```text
ID: RAIL-CAND-DOMAIN-TESTS-001
Origen: scanner backlog 2026-05-24.
Casos: cierre `domain_work` no-OPES cubierto con app temporal y validador fake;
falta politica productiva por puerto para evidencias/tests de dominio generico.
Decision pendiente: mantener los validadores de dominio fuera del nucleo y
aceptar solo refs/artefactos causales; URL, DB, ruta local o nombre de conector
no son evidencia suficiente de cierre.
Test futuro: focos de `orquesta-domain-work`, `orquesta-app-change`,
`orquesta-app-director-service` y `orquesta-app-codex-stack`.
Backlog: `T14 domain-work-required-tests-productivos`.
```

```text
ID: RAIL-CAND-BACKLOG-STALE-001
Origen: scanner backlog 2026-05-24 segunda pasada.
Casos: `T08`, `T09` y `T10` tenian evidencia local cerrada en docs/codigo, pero
el backlog no incluia `Estado: completada`; si el ACK runtime no esta presente,
el planner puede volver a preparar trabajo ya cerrado.
Decision pendiente: sincronizar backlog con ACKs durables, docs locales de
modulo y cola visible; si la evidencia es ambigua, crear revision documental
acotada en vez de ejecucion amplia.
Test futuro: `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server`
con caso de seccion cerrada por evidencia documental y ACK ausente.
Backlog: `T17 autoprogramming-backlog-state-sync`.
```

```text
ID: RAIL-CAND-LOCAL-LISTS-002
Origen: scanner backlog 2026-05-24 segunda pasada.
Casos: quedan rails/listas locales fuera del inventario inicial en
`orquesta-app-codex-stack` para refs de contexto externo y evidencia de
review/replan, y fronteras de operador/leases que usan reglas propias o wrappers
locales. Pueden ser fronteras legitimas, pero deben quedar clasificadas antes de
reactivar detalle global.
Decision pendiente: registrar propietario por modulo, matriz externa y relacion
con `orquesta-rails`; no endurecer por vocabulario operativo ni relajar fronteras
de efecto externo sin caso reproducible.
Test futuro: `./scripts/test_rails_fast.sh` mas focos de
`orquesta-app-codex-stack`, `orquesta-operator-mcp` y `orquesta-core-leases`.
Backlog: `T15 detail-rails-reactivation`.
```

```text
ID: RAIL-CAND-DETAIL-DOC-STATE-002
Origen: scanner backlog 2026-05-24 cuarta pasada.
Casos: el codigo de `cmd/orquesta-server` fija
`ORQUESTA_DETAIL_PROHIBITED_RAILS=on` por defecto con scope acotado, mientras
docs de rails fuera de este write-set aun describen el servidor como `off` por
defecto. Esa divergencia puede hacer que el planner reabra T15 de forma
incorrecta o que un operador reactive/desactive rails con una foto antigua.
Actualizacion 2026-06-02: el codigo vuelve a default offline y agrega
`ORQUESTA_RAILS_MODE=offline`. Este candidato queda historico salvo regresion
que reactive bloqueos por detalle sin opt-in.
Decision pendiente: mantener sincronizado registro vivo, rail errors,
duplicaciones y matriz rapida; conservar `off` como estado declarado del
servidor hasta nueva orden.
Test futuro: `./scripts/test_rails_fast.sh` y
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-rails`.
Backlog: `T22 detail-rails-doc-state-sync`.
```

```text
ID: RAIL-CAND-MCP-RUN-TRANSPORT-001
Origen: scanner backlog 2026-05-24 cuarta pasada.
Casos: el smoke `mcp-real-smoke` cierra transporte MCP temporal por comando,
pero no decide si el servidor residente debe exponer `/mcp` durante `run`.
Si se activa sin guardas, podria duplicar superficies HTTP/MCP, auditoria y
payloads; si no se activa, debe quedar documentado que el comando temporal es la
frontera suficiente.
Decision pendiente: decidir transporte residente opt-in por composicion, con
bind/loopback explicito, payload compacto, errores publicos y auditoria sin
secretos ni transcripts.
Test futuro:
`go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-operator-mcp-client ./cmd/orquesta-server`.
Backlog: `T23 mcp-run-transport-opt-in`.
```

```text
ID: RAIL-CAND-STARTUP-ACK-TEXT-001
Origen: scanner backlog 2026-05-24 quinta pasada.
Casos: compactacion de arranque en `cmd/orquesta-server` detecta ACK
completado leyendo `codex_last_message.txt` y buscando `ack` + `completed`.
Decision pendiente: usar `agent_ack.json`/receipt estructurado y correlado como
fuente de verdad; dejar `codex_last_message.txt` solo como diagnostico.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack`.
Backlog: `T24 startup-structured-ack-compaction`.
```

```text
ID: RAIL-CAND-MCP-ROADMAP-STALE-001
Origen: scanner backlog 2026-05-24 quinta pasada.
Casos: recursos MCP `orquesta.project.roadmap.v0` y
`orquesta.shared_contracts.v0` exponen estados `pendiente_*` hardcodeados que
pueden divergir del backlog vivo y la foto vigente.
Decision pendiente: sincronizar recursos MCP con backlog/docs vigentes o marcar
entradas historicas con freshness/source_refs para no relanzar trabajo cerrado.
Test futuro: `go test -count=1 ./modulos/orquesta-mcp`.
Backlog: `T25 mcp-roadmap-backlog-state-sync`.
```

```text
ID: RAIL-CAND-DOMAIN-QUALITY-001
Origen: scanner backlog 2026-05-24 quinta pasada.
Casos: `topic_expansion_package` usa strings genericas (`TODO`, `placeholder`,
`pendiente_revision`) y recuento de palabras en el stack Codex para decidir
calidad de dominio.
Decision pendiente: mover la politica a puerto/adaptador de dominio con matriz
por campo, issues estructurados y tolerancia a vocabulario valido.
Test futuro:
`go test -count=1 ./modulos/orquesta-domain-work ./modulos/orquesta-app-codex-stack ./modulos/orquesta-opes-bridge`.
Backlog: `T26 domain-work-quality-policy-port`.
```

```text
ID: RAIL-CAND-DOC-STATE-DRIFT-001
Origen: scanner backlog 2026-05-24 sexta pasada.
Casos: documentos de entrada obligatoria presentan estado contradictorio sobre
`CODEX-WAVE-REAL` y `CODEX-RECURSION-REAL`: unas fuentes los declaran cerrados
con proveedor y otras mantienen recursion Codex productiva como pendiente.
Decision pendiente: fijar orden de autoridad documental, sincronizar estado y
marcar historico cualquier documento que no sea foto vigente. El frente real
abierto debe seguir siendo OPES temporal de derivados/cierre, salvo regresion
demostrada de Codex.
Test futuro: prueba documental focal que detecte contradicciones conocidas entre
`AGENTS.md`, `docs/estado_actual_2026-05-17.md`, guia, matriz y backlog.
Backlog: `T27 docs-source-of-truth-state-sync`.
```

```text
ID: RAIL-CAND-RESTART-LIVE-AGENTS-001
Origen: scanner backlog 2026-05-24 sexta pasada.
Casos: `DAEMON-RESTART` valida cola file-based sin agentes reales y la matriz
declara que no cubre rehidratacion de procesos Codex vivos. El contrato runtime
mantiene `resume` fuera de `launch_mode`; si el arranque archiva o completa por
heuristicas locales puede perder trabajo vivo.
Decision pendiente: definir contrato de resume/reconciliacion por refs opacas,
descriptor de runtime, ACK estructurado, checkpoint y wait scope; no cerrar por
`codex_last_message.txt`, summary textual ni ausencia transitoria de pending.
Test futuro: smoke opt-in con agente Codex temporal, reinicio de servidor y
reconciliacion de ACK/delivery/review/cierre o bloqueo causal.
Backlog: `T28 restart-live-agent-reconciliation`.
```

```text
ID: RAIL-CAND-PROVIDER-USAGE-ACCOUNTING-001
Origen: scanner backlog 2026-05-24 sexta pasada.
Casos: telemetry fake y stats REST ya exponen uso, pero falta conector
productivo de cuota/tokens por proveedor. Sin puerto explicito, coste real,
modelo, cuenta o detalles de proveedor pueden mezclarse con stats, auditoria o
ACKs.
Decision pendiente: separar uso/coste por puerto opt-in, redaccion por campo y
errores publicos cuando no haya fuente productiva. El nucleo conserva refs
opacas; proveedor/modelo/cuenta/coste concreto viven en adaptador.
Test futuro:
`go test -count=1 ./modulos/orquesta-orchestration-core ./modulos/orquesta-runtime ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack ./modulos/orquesta-mcp ./modulos/orquesta-web ./cmd/orquesta-server`.
Backlog: `T29 provider-usage-quota-accounting`.
```

```text
ID: RAIL-CAND-SHUTDOWN-GENERIC-001
Origen: scanner backlog 2026-05-24 septima pasada.
Casos: apagado cooperativo ya valida `codex_shutdown_checkpoint_ack.v0` con
Codex real, pero el protocolo sigue ligado a ficheros/prompt Codex. Al elevarlo
a runtimes genericos, vocabulario como `shutdown`, `checkpoint`, `runtime`,
`provider` o nombres de ficheros de control no debe disparar falsos positivos.
Decision pendiente: definir contrato neutral de checkpoint por puerto y aplicar
rails por campo: permitir refs/ficheros de control y cortar solo secretos
efectivos, rutas privadas, prompts/transcripts o payloads de proveedor.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime ./modulos/orquesta-server-shutdown ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T30 runtime-shutdown-checkpoint-port`.
```

```text
ID: RAIL-CAND-RUN-QUEUE-CLAIM-001
Origen: scanner backlog 2026-05-24 septima pasada.
Casos: la cola global ordena y prioriza, pero no reserva candidatos. Un contrato
de claim/lease puede introducir `owner_ref`, `lease_ref`, `ttl`, `heartbeat` y
razones de skip que deben ser refs/metadata opacas, no datos locales de proceso.
Decision pendiente: permitir metadata compacta de lease en cola y supervisor,
rechazar PID/HOME/rutas/host como verdad del nucleo y probar conflictos de
reserva sin relanzar agentes.
Test futuro:
`go test -count=1 ./modulos/orquesta-run-queue ./modulos/orquesta-run-memory ./modulos/orquesta-run-file ./modulos/orquesta-run-supervisor ./modulos/orquesta-server ./cmd/orquesta-server`.
Backlog: `T31 run-queue-reservation-lease`.
```

```text
ID: RAIL-CAND-RESIDENT-WAKEUP-001
Origen: scanner backlog 2026-05-24 septima pasada.
Casos: la politica event-driven pide actuar por senales reales, pero el loop
residente actual se apoya en ticks. Los wakeups futuros pueden traer causas de
cola, delivery, shutdown o capacidad; si transportan payloads crudos duplicarian
auditoria y rails de detalle.
Decision pendiente: wakeup compacto por refs opacas, causa y contadores; sin
prompts, transcripts, payloads HTTP completos, rutas locales, HOME ni tokens.
Test futuro:
`go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-run-supervisor ./modulos/orquesta-run-queue ./cmd/orquesta-server`.
Backlog: `T32 resident-supervisor-event-driven-wakeup`.
```

```text
ID: RAIL-CAND-BACKLOG-ACK-001
Origen: scanner backlog 2026-05-24 octava pasada.
Casos: el planner de automejora considera completada una request de backlog si
encuentra `agent_ack.json` bajo `.orquesta-runtime` con `status=completed`,
sin validar schema, correlacion, ack_ref, task_ref ni pruebas obligatorias.
Decision pendiente: correlacionar ACK contra spec/packet o marcarlo ambiguo; no
ocultar secciones pendientes por ACK incompleto, antiguo o de otra request.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack`.
Backlog: `T33 autoprogramming-backlog-ack-correlation`.
```

```text
ID: RAIL-CAND-DIRECTOR-WAVE-GUARDS-001
Origen: scanner backlog 2026-05-24 octava pasada.
Casos: `codex-director-wave` deja `--strict-director-guards` desactivado por
defecto; en modo no estricto rellena `write_set=.` y tests placeholder, y el
prompt permite ampliar ficheros fuera del shard por justificacion textual.
Decision pendiente: modo estricto por defecto para ejecucion real, opt-in
explicito para write-set raiz o tests placeholder, y prompts que separen shard
flexible de alcance autorizado total.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-director-operativo ./modulos/orquesta-app-codex-stack`.
Backlog: `T34 director-wave-strict-guards-default`.
```

```text
ID: RAIL-CAND-STARTUP-CLEANUP-001
Origen: scanner backlog 2026-05-24 octava pasada.
Casos: el servidor fija `ORQUESTA_STARTUP_CLEANUP_MODE=forced_stop` si no hay
valor explicito. En un reinicio con agentes vivos o ACK ambiguos, esa politica
puede compactar cola/control como parada logica antes de reconciliar runtime.
Decision pendiente: default conservador de bloqueo/reconciliacion; `forced_stop`
solo opt-in con evidencia, causa publica y contadores de kept/blocked/compacted.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-run-control ./modulos/orquesta-runtime ./modulos/orquesta-runtime-codex ./modulos/orquesta-state-file`.
Backlog: `T35 startup-cleanup-safe-mode`.
```

```text
ID: RAIL-CAND-ACK-STRICT-001
Origen: scanner backlog 2026-05-24 novena pasada.
Casos: `ValidateCodexAgentAckBytesForSpecV0` rellena campos de identidad desde
la spec antes de validar, y `ReadCodexDeliveryObservationFileV0` tiene cobertura
que acepta un ACK minimo con solo `schema_version` y `status=completed` si la
spec aporta defaults.
Decision pendiente: separar modo diagnostico/legacy de modo terminal estricto;
un ACK terminal nuevo debe traer refs explicitas, tests requeridos y correlacion
propia antes de cerrar delivery, startup compaction o backlog.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T36 codex-ack-strict-terminal-validation`.
```

```text
ID: RAIL-CAND-BACKLOG-PARSER-TESTS-001
Origen: scanner backlog 2026-05-24 novena pasada.
Casos: el planner de backlog solo extrae comandos `go test` desde `Tests:` y
marca secciones como completadas por narrativa amplia como `revalidacion final`.
Se pierden verificaciones declaradas como `git diff --check`,
`bash -n scripts/*.sh`, `./scripts/test_rails_fast.sh` o prueba documental
focal.
Decision pendiente: parsear estado canonico y comandos de verificacion completos
o declarar blocker manual; no sustituir pruebas de la seccion por el test base
ni ocultar trabajo por texto narrativo.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`.
Backlog: `T37 autoprogramming-backlog-parser-fidelity`.
```

```text
ID: RAIL-CAND-CODEX-OBSERVATION-OWNER-001
Origen: scanner backlog 2026-05-24 novena pasada.
Casos: `codexDeliveryObservationUnsafeForCoreV0` mantiene una lista local que
detecta `codex`, `runtime`, `provider`, `git`, `db` o `sql` en observaciones,
mientras la politica comun de `orquesta-rails` ya distingue vocabulario opaco de
secretos efectivos.
Decision pendiente: decidir si ese helper se elimina como detector historico o
se convierte en politica comun por campo; no duplicar listas ni bloquear refs
opacas validas.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime-codex ./modulos/orquesta-rails ./modulos/orquesta-app-codex-stack`.
Backlog: `T38 codex-delivery-observation-rail-owner`.
```

```text
ID: RAIL-CAND-APP-VCS-EFFECT-GUARDS-001
Origen: scanner backlog 2026-05-24 decima pasada.
Casos: `orquesta.app_vcs.v0` permite `commit`/`push` por MCP/HTTP y el conector
Git hace `git add -A` cuando `commit_paths` esta vacio. El contrato actual no
transporta `write_set`, cierre/review aceptada, tests requeridos ni evidencia de
solape cero.
Decision pendiente: exigir paths/write-set y evidencia causal para commit/push;
dejar `git add -A` y push remoto como opt-in auditado, con redaccion de salida
Git antes de devolver errores o auditar.
Test futuro:
`go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-app-codex-stack ./modulos/orquesta-runtime-worktree ./cmd/orquesta-server`.
Backlog: `T39 app-vcs-write-set-evidence-guards`.
```

```text
ID: RAIL-CAND-PROMOTION-E2E-STATE-001
Origen: scanner backlog 2026-05-24 decima pasada.
Casos: T13 sigue documentado como pendiente aunque ya existen puerto neutral,
conector Git opt-in, wiring de servidor y pruebas unitarias/fake del stack.
Falta e2e acotado con repo temporal y estado documental que distinga piezas
cerradas, promocion local, archivo idempotente, push pendiente y pendientes
productivos.
Decision pendiente: cerrar T13 con prueba de composicion temporal y recibo
durable; no marcar la cola como terminal promocionada si promocion, push o
archivo queda pendiente/bloqueado.
Test futuro:
`go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-runtime-worktree ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T40 autoprogramming-promotion-real-e2e-and-doc-state`.
```

```text
ID: RAIL-CAND-SERVER-AUDIT-PAYLOAD-002
Origen: scanner backlog 2026-05-24 decima pasada.
Casos: `auditEventV0` recibe `payload any` y eventos del supervisor/automejora
guardan structs completos de request/result. Aunque HTTP solo audita metadata,
los payloads internos pueden crecer hacia prompts, transcripts, rutas privadas,
remotos Git o cuerpos de dominio si no hay contrato por evento.
Decision pendiente: sanitizador/esquema por evento antes de escribir JSONL;
guardar refs, contadores, estados y errores publicos, no material crudo.
Mantener payload HTTP completo apagado salvo opt-in con redaccion y retencion
corta.
Test futuro:
`go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-observability ./cmd/orquesta-server`.
Backlog: `T41 server-audit-payload-redaction-contract`.
```

```text
ID: RAIL-CAND-STRICT-ACK-WRITESET-001
Origen: scanner backlog 2026-05-24 undecima pasada.
Casos: `ValidateCodexAgentAckBytesForSpecV0` hidrata identidad desde la spec y
la cobertura actual acepta `files` fuera del write-set como rail blando. El
paquete OrquestaV2 externo exige modo estricto: no editar fuera del write-set,
no declarar archivos de control y fallar con `CONSULTA AL DIRECTOR` si falta
alcance.
Decision aplicada 2026-05-24: separar modo legacy/advisory de modo terminal
estricto por policy/packet. En estricto, `completed` requiere refs explicitas,
`files` concretos dentro del write-set, sin archivos de control, y `tests`
exactamente iguales a los requeridos. La prueba de snapshot queda en
`orquesta-runtime-worktree`: si el adaptador aporta `ACK.files`, todos deben
corresponder a cambios reales y no pueden faltar cambios del snapshot.
Estado: cubierto 2026-05-25; la observacion terminal Codex ya no degrada a modo
legacy cuando el packet trae policy estricta y falta recibo de test.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-worktree ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T42 codex-ack-strict-write-set-terminal-proof`.
```

```text
ID: RAIL-CAND-BACKLOG-SCANNER-MERGE-001
Origen: scanner backlog 2026-05-24 undecima pasada.
Casos: los scanners de automejora usan el mismo write-set documental
(`autoprogramacion`, `rail_errors`, `duplicaciones`) y pueden ejecutarse en
burst. El planner deduplica por refs conocidas/ACK, pero no transporta epoch,
hash de documentos ni lease de merge para bloquear una entrega basada en foto
obsoleta.
Decision pendiente: crear reserva/epoch de scanner y validacion de merge antes
de aceptar ACK; si hay colision, exponer rebase/merge pendiente al director sin
borrar contenido ni marcar completed silencioso.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-autoprogramming ./modulos/orquesta-run-queue`.
Backlog: `T43 backlog-scanner-doc-merge-lease`.
```

```text
ID: RAIL-CAND-REQUIRED-CONTEXT-REFONLY-001
Origen: scanner backlog 2026-05-24 undecima pasada.
Casos: el paquete del agente puede traer una entrada `required=true` con
`kind=doc_ref`, `mode=ref_only` y `total_bytes=0`. Para tareas que dependen de
docs obligatorios, cerrar con `completed` sin evidencia de lectura/materializacion
puede convertir falta de contexto en backlog inventado o incompleto.
Decision pendiente: distinguir `required ref_only` permitido por diseno de
contexto obligatorio no materializado; exigir evidencia de lectura, resolucion
de contexto truncado o `CONSULTA AL DIRECTOR` antes de cierre terminal.
Test futuro:
`go test -count=1 ./modulos/orquesta-context ./modulos/orquesta-orchestration-core ./modulos/orquesta-runtime ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T44 required-context-ref-only-guard`.
```

```text
ID: RAIL-CAND-GO-FILE-LINE-BUDGET-001
Origen: scanner backlog 2026-05-24 duodecima pasada.
Casos: el prompt exige ficheros Go por debajo de 300 lineas, pero
`file_too_large` es advisory en `orquesta-autoprogramming` y la matriz acepta
301..520 lineas con followup. El repo ya contiene deuda historica por encima de
300 lineas; sin baseline, endurecer bloquea todo, pero sin modo estricto una
entrega nueva puede agrandar controladores enormes y cerrar como completed.
Decision 2026-05-24: `orquesta-runtime-worktree` registra lineas Go en
snapshots y valida crecimiento con `StrictGoLineBudget`;
`orquesta-autoprogramming` conserva advisory legacy y separa el bloqueo
`go_file_line_budget_strict_blocking`; `orquesta-app-codex-stack` puede activar
el rail estricto por puerto/configuracion sin confiar en `ACK.LineCount`.
Decision 2026-05-25: la presencia de `LineCountSource` real ya no activa el rail
estricto por si sola; solo `StrictGoLineBudget` endurece cierre. En modo
estricto, el stack tambien usa el diff snapshot/worktree para detectar ficheros
Go omitidos del ACK.
Rework 2026-05-25: se conserva la entrega util y se alinea el estado documental
de T45 sin relanzar la tarea padre.
Test futuro:
`go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-runtime-worktree ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T45 autoprogramming-go-file-line-budget-baseline`.
```

```text
ID: RAIL-CAND-PACKET-WRITESET-PRECEDENCE-001
Origen: scanner backlog 2026-05-24 decimotercera pasada.
Casos historicos: el paquete OrquestaV2 traia `write_set_closed` y el prompt
Codex ordenaba no editar fuera del write-set, pero `programmingObjectiveV0`
anadia permiso textual para tocar otros ficheros y justificarlo en ACK.
Decision aplicada 2026-05-24: los paquetes estrictos ya declaran alcance
cerrado, el prompt fija precedencia de `write_set_closed`, el modo legacy queda
nombrado como compatibilidad y `BuildAgentStartPacketV0` invalida frases de
ampliacion por simple ACK.
Revalidacion 2026-05-25: esta entrada no reabre T46; cualquier resto pendiente
queda separado en receipts de tests estrictos o prueba de efectos destructivos.
Rework de revision 2026-05-25: el paquete estricto
`agent-ref-task-ref-review-rework-task-autoprogramming-e8eba749c154-g01-c80cde88cf66`
mantiene la entrega dentro del write-set documental, valida contexto
`ref_only` requerido por lectura local/evidencia ACK y no relanza otro padre.
Test:
`go test -count=1 ./modulos/orquesta-app-codex-stack ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime`.
Backlog: `T46 agent-packet-write-set-precedence`.
```

```text
ID: RAIL-CAND-ACK-TEST-RECEIPT-001
Origen: scanner backlog 2026-05-24 decimotercera pasada.
Casos: `ValidateCodexAgentAckForSpecV0` exige que `ACK.tests` contenga los
comandos requeridos y detecta evidencia textual de fallo, pero no prueba por si
solo que el agente externo haya ejecutado el comando con exit code exitoso.
Decision aplicada 2026-05-25: en modo estricto, completar requiere
`RequiredTestEvidenceV0` o recibo estructurado del runtime/adaptador; strings de
ACK quedan como diagnostico o compatibilidad legacy.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-required-test ./modulos/orquesta-app-codex-stack ./modulos/orquesta-orchestration-core ./cmd/orquesta-server`.
Backlog: `T47 external-agent-required-test-receipts`.
```

```text
ID: RAIL-CAND-DESTRUCTIVE-WORKTREE-001
Origen: scanner backlog 2026-05-24 decimotercera pasada.
Casos: `VerifyWorktreeWriteSetV0` detecta paths eliminados y cambios fuera de
write-set, pero la politica terminal no separa truncado fuerte, reemplazo masivo
o rename ambiguo dentro de write-set. El paquete OrquestaV2 prohibe borrar,
mover fuera o truncar archivos existentes sin decision del director.
Decision pendiente: clasificar efectos destructivos desde snapshot/worktree y
bloquear `completed` en modo estricto salvo decision causal explicita.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime-worktree ./modulos/orquesta-autoprogramming ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T48 destructive-worktree-change-proof`.
Estado: cubierto 2026-05-24 para clasificacion offline y bloqueo estricto en
runtime-worktree, review gate de autoprogramacion y stack Codex. Revalidado en
segunda pasada: el stack Codex comparte el store de snapshots entre baseline,
verificador de delivery y review gate, y `cmd/orquesta-server` lo inyecta desde
composicion sin filtrar rutas locales al nucleo.
```

```text
ID: RAIL-CAND-CODEX-SECURITY-PROFILE-001
Origen: scanner backlog 2026-05-24 decimocuarta pasada.
Casos: `cmd/orquesta-server` y `orquesta-app-codex-stack` normalizan sandbox
invalido a `workspace-write` antes de validar, mientras el perfil Codex puro lo
rechaza. Tambien existe `DirectorApprovalPolicy=on-request` en configuracion de
prueba, incompatible con autoprogramacion residente desatendida salvo opt-in de
operador vivo.
Decision pendiente: separar compatibilidad/diagnostico de modo estricto; no
corregir flags de seguridad en silencio y exigir approval no interactivo o
decision explicita del director/operador.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T49 codex-runtime-security-profile-strict-mode`.
Estado: cubierto 2026-05-24 para servidor/stack Codex. La configuracion conserva
sandbox invalido hasta la validacion publica, approval interactivo requiere
opt-in `ORQUESTA_CODEX_ALLOW_INTERACTIVE_APPROVAL=1` y el perfil/prompt declaran
si el runtime de control es interno al proyecto o writable root externo.
Revalidacion: los paquetes OrquestaV2 estrictos deben declarar resolucion de
contexto `ref_only` por evidencia explicita en el ACK antes de cerrar
`completed`.
Revalidacion 2026-05-24 r2: required-tests retry resuelve contexto `ref_only`
por evidencia explicita en ACK y conserva T49 como cubierto.
```

```text
ID: RAIL-CAND-RUNTIME-CONTROL-WORKTREE-001
Origen: scanner backlog 2026-05-24 decimocuarta pasada.
Casos: el runtime de control puede vivir como `.orquesta-runtime` dentro del
proyecto y el write-set puede ser `.`. Sin exclusion comun, snapshots,
verification, staging promotion o AppVCS pueden tratar prompts, packets, ACKs,
logs o checkpoints como cambios de producto.
Decision pendiente: excluir control dirs y control files de snapshots,
promocion, AppVCS, contexto y ACK terminal, incluso con write-set raiz. La
lectura del propio `agent_packet.json` queda como excepcion de control, no como
artefacto exportable.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime-worktree ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T50 runtime-control-files-worktree-exclusion`.
Estado: cubierto 2026-05-25 r3. `orquesta-runtime-worktree` centraliza
exclusiones por defecto y issue `control_path`; snapshots/verificacion ignoran
control files, `ACK.files` los rechaza, staging promotion/AppVCS no los
promocionan ni commitean, y `orquesta-runtime-codex` los mantiene prohibidos
incluso con `write_set=["."]`. La revalidacion r3 incluye runtimes locales con
sufijo temporal como `.orquesta-local-runtime-*`.
```

```text
ID: RAIL-CAND-CAPACITY-DEFAULT-POLICY-001
Origen: scanner backlog 2026-05-24 decimocuarta pasada.
Casos: varias rutas del servidor y stack Codex defaultan o elevan
`ORQUESTA_CODEX_REASONING_EFFORT` a `high`/`xhigh`, mientras las reglas vigentes
ordenan `medium` por defecto para exploracion y pruebas y reservar `xhigh` para
orden explicita o riesgo tecnico justificado.
Decision pendiente: politica unica por work_profile/dominio/riesgo, con
override auditable para OPES/temarios reales o decisiones amplias, y default
medio para scanners/automejora documental.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-app-codex-stack ./modulos/orquesta-runtime-codex ./modulos/orquesta-autoprogramming`.
Backlog: `T51 capacity-reasoning-default-policy`.
Estado: cerrado 2026-05-25. `orquesta-autoprogramming` publica la matriz
compacta, `cmd/orquesta-server` conserva `medium` y defaults `medium`, y los
packets/stats Codex exponen refs de politica/evidencia compactas.
```

```text
ID: RAIL-CAND-GO-LINE-BUDGET-APP-DIRECTOR-001
Origen: scanner backlog 2026-05-24 decimoquinta pasada.
Casos: medicion local encontro ficheros Go muy por encima de 300 lineas en
`modulos/orquesta-app-director-service`, incluyendo ciclo operativo, continue,
cierre y tests. El rail OrquestaV2 exige no seguir creciendo controladores
enormes, pero falta shard concreto de particion para este modulo.
Decision 2026-05-25: partir por responsabilidad sin cambiar comportamiento y
registrar baseline de deuda historica que no pueda cerrarse en una sola tarea.
El codigo productivo del modulo queda bajo 300 lineas por fichero; el residuo
baselinado queda limitado a tests historicos y no debe crecer sin nuevo shard.
Test futuro:
`go test -count=1 ./modulos/orquesta-app-director-service ./modulos/orquesta-orchestration-core ./modulos/orquesta-state-file`.
Backlog: `T52 app-director-service-file-split`.
```

```text
ID: RAIL-CAND-GO-LINE-BUDGET-ORCH-CORE-001
Origen: scanner backlog 2026-05-24 decimoquinta pasada.
Casos: `orquesta-orchestration-core` tiene materializador, cierre, plan-state y
runner de tests por encima del rail de 300 lineas. Al ser nucleo de aplicacion,
un refactor mal delimitado puede mezclar puertos neutrales con adaptadores
concretos.
Decision 2026-05-25: separar los cuatro ficheros objetivo por responsabilidad
neutral, conservar puertos/refs opacas y dejar cada fichero productivo tocado
por debajo de 300 lineas. La particion no cambia contratos publicos; solo
redistribuye materializacion, cierre causal, plan-state y runner de tests en
shards locales.
Test futuro:
`go test -count=1 ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-director-service`.
Backlog: `T53 orchestration-core-file-split`.
```

```text
ID: RAIL-CAND-GO-LINE-BUDGET-CODEX-STACK-001
Origen: scanner backlog 2026-05-24 decimoquinta pasada.
Casos: `modulos/orquesta-app-codex-stack` y `cmd/orquesta-server` acumulan
ficheros largos de composicion, comandos y smokes reales. Sin particion, los
rails de ACK, control files, seguridad Codex, VCS y capacidad pueden volver a
duplicarse dentro de controladores grandes.
Decision 2026-05-25: separar `codex-wave`, `director-wave` y `drain` por flujo
sin cambiar contratos ni mover runtime/proveedor/API al nucleo. Queda baseline
explicita para tests y smokes historicos largos; cualquier crecimiento futuro
requiere shard propio.
Test futuro:
`go test -count=1 ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-codex-delivery`.
Backlog: `T54 codex-stack-server-file-split`.
```

```text
ID: RAIL-CAND-SERVER-CONTROL-EXPOSURE-001
Origen: scanner backlog 2026-05-24 decimosexta pasada.
Casos: `ORQUESTA_SERVER_ADDR` puede cambiar el bind del servidor y la superficie
HTTP/MCP contiene rutas de control mutables: shutdown, runs/control, cola,
autoprogramacion, AppVCS opt-in y transporte MCP real futuro.
Decision pendiente: loopback por defecto; bind no-loopback solo con opt-in,
principal/credencial o mTLS/TLS de composicion, autorizacion por ruta y auditoria
sin tokens/cabeceras completas/payloads crudos.
Decision 2026-05-25: `orquesta-server` queda como propietario de la guarda
residente. `ORQUESTA_SERVER_ADDR` no-loopback exige
`ORQUESTA_SERVER_REMOTE_CONTROL_PLANE_CONFIRM=1` y
`ORQUESTA_SERVER_CONTROL_TOKEN`; las mutaciones HTTP/MCP pasan por token opt-in o
loopback local, y la auditoria guarda solo decision, principal/ref de permiso,
bind, metodo/path y claves de query sin valores.
Test futuro:
`go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway ./modulos/orquesta-mcp ./cmd/orquesta-server`.
Backlog: `T55 server-control-plane-exposure-guards`.
```

```text
ID: RAIL-CAND-CODEX-CODEHOME-CREDENTIAL-001
Origen: scanner backlog 2026-05-24 decimosexta pasada.
Casos: `codex-wave` y `codex-director-wave` copian `auth.json`, `config.toml`,
skills, plugins, rules y memories desde `CODEX_HOME` a homes de agentes. El
vocabulario `auth`, `token`, `credential`, `HOME`, `skills` o `plugins` no debe
bloquear refs opacas, pero los valores reales no pueden acabar en snapshot,
ACK, auditoria, promocion ni contexto de producto.
Decision 2026-05-25: `codex-wave`/`codex-director-wave` usan politica de
proyeccion por categoria. La evidencia durable lista categorias y refs opacas,
no valores ni ruta fuente; `memories` requiere opt-in y el modo estricto exige
home aislado con `auth`/`config` presentes o bloquea con error publico
recuperable.
Test:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-worktree ./modulos/orquesta-app-codex-stack`.
Estado: cubierto 2026-05-25; rework de revision sincroniza el estado cerrado de
T56 con el backlog ejecutable y mantiene T152 como shard mecanico separado.
Backlog: `T56 codex-code-home-credential-projection`.
```

```text
ID: RAIL-CAND-OUTBOX-DISPATCH-ACK-RECOVERY-001
Origen: scanner backlog 2026-05-24 decimosexta pasada.
Casos: outbox dispatch ya tiene contratos puros de claim/lease/batch/ACK, pero
la composicion residente debe demostrar recuperacion durable ante restart,
claim expirado, ACK parcial y retry sin duplicar efectos externos.
Decision pendiente: persistir/reconciliar `message_id`, `run_id`,
`target_port`, `claim_ref`, `lease_ref` y ACK correlacionado; conservar
`pending`/`ack_failed` con causa publica y no cerrar planes con outbox pendiente.
Test futuro:
`go test -count=1 ./modulos/orquesta-outbox-dispatch ./modulos/orquesta-orchestration-core ./modulos/orquesta-persistence ./modulos/orquesta-app-director-service ./modulos/orquesta-server ./cmd/orquesta-server`.
Backlog: `T57 outbox-dispatch-durable-ack-recovery`.
```

```text
ID: RAIL-CAND-CODEX-WAVE-TAIL-RAW-LOG-001
Origen: scanner backlog 2026-05-24 decimoseptima pasada.
Casos: `codex-wave-tail` imprime fragmentos crudos de `codex_stdout.log`,
`codex_stderr.log` o `codex_last_message.txt`. Es diagnostico util, pero puede
exponer prompts, transcripts, HOME, rutas privadas, tokens, payloads HTTP o
diffs completos si se usa fuera de una consola local controlada.
Decision: summary/redaccion por defecto, opt-in de fragmento crudo, limites de
bytes/lineas, permiso por `wave_ref`/`agent_ref` y prohibicion de usar logs
como evidencia terminal de ACK/delivery/cierre.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-rails`.
Backlog: `T58 codex-wave-log-tail-redaction-access`.
Estado 2026-05-25: cubierto; `codex-wave-tail` queda como diagnostico JSON
redactado/acotado, exige razon de diagnostico y valida runtime/agente antes de
leer logs.
Rework de revision 2026-05-25: sincronizado con el cierre documental de T58 en
el backlog y la matriz de duplicaciones.
```

```text
ID: RAIL-CAND-CODEX-WAVE-PURGE-RUNTIME-001
Origen: scanner backlog 2026-05-24 decimoseptima pasada.
Casos: `codex-launch-wave` y `codex-launch-director-wave` aceptan
`--purge-runtime`/`ORQUESTA_CODEX_WAVE_PURGE_RUNTIME` y ejecutan borrado del
runtime resuelto. Las guardas actuales reducen riesgo obvio, pero no prueban
raiz permitida, manifest, agentes vivos, checkpoints, ACKs no reconciliados,
director_decisions pendientes, outbox pendiente ni plan state abierto.
Decision implementada 2026-05-25: purga solo con confirmacion explicita,
runtime bajo raiz permitida, report-only, bloqueo por trabajo vivo y evidencia
compacta sin rutas privadas ni contenido de logs/prompts.
Rework de revision 2026-05-25: la correccion estricta conserva la entrega T59,
no relanza otro padre y valida el contexto `required ref_only` mediante lectura
local de `agent_packet.json` mas recibo explicito en ACK.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-worktree ./modulos/orquesta-server-shutdown ./modulos/orquesta-agent-process-registry`.
Backlog: `T59 codex-wave-runtime-purge-proof`.
```

```text
ID: RAIL-CAND-CODEX-WAVE-STOP-PID-REGISTRY-001
Origen: scanner backlog 2026-05-24 decimoseptima pasada.
Casos: `codex-wave-stop` carga un registro desde `runtime-dir` y senala PIDs
listados. Si el runtime-dir esta mal seleccionado o el registro esta corrupto,
la herramienta puede intentar parar un proceso que Orquesta no lanzo ni sigue
reconociendo como agente de esa ola.
Decision implementada 2026-05-25: validar descriptor de proceso antes de senalar PID:
`run_ref`, `wave_ref`, `agent_ref`, command ref, runtime permitido,
start time/owner opaco y estado vivo; si no hay prueba, bloquear con error
publico `blocked_registry_untrusted`. `codex-wave-stop` exige confirmacion,
razon publica y `--force` explicito para senal directa; sin `--force`, escribe
shutdown cooperativo.
Rework de revision 2026-05-25: la correccion estricta conserva la entrega T60,
no relanza otro padre y valida el contexto `required ref_only` mediante lectura
local de `agent_packet.json` mas recibo explicito en ACK.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-runtime ./modulos/orquesta-runtime-codex ./modulos/orquesta-agent-process-registry ./modulos/orquesta-run-control`.
Backlog: `T60 codex-wave-stop-registry-pid-proof`.
```

```text
ID: RAIL-CAND-REQUIRED-TEST-OUTPUT-001
Origen: scanner backlog 2026-05-24 decimoctava pasada.
Casos: `LocalCommandExecutorV0` escribe stdout/stderr de tests requeridos en
artefactos `required-test-output-v0/*.log` con limite de bytes, pero sin
redaccion por campo ni retencion. Un `go test` real puede imprimir HOME, rutas
privadas, payloads HTTP, variables de entorno, tokens simulados o datos de
dominio.
Decision pendiente: persistir evidencia causal compacta y logs diagnosticos
redactados/retencionados por separado; si aparece material sensible no
redactable, bloquear o fallar sin guardar el material crudo.
Decision aplicada 2026-05-25: `LocalCommandExecutorV0` separa
`RequiredTestEvidenceV0` de los logs, redacta stdout/stderr antes de persistir,
marca `output_redacted=true`, aplica limite de bytes y retencion configurable;
salida no UTF-8/no redactable queda como fallo con causa publica sin guardar el
material crudo.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime-required-test ./modulos/orquesta-orchestration-core ./modulos/orquesta-state-file ./modulos/orquesta-app-director-service ./cmd/orquesta-server`.
Backlog: `T61 required-test-output-redaction-retention`.
```

```text
ID: RAIL-CAND-DIRECTOR-DECISIONS-PROMPT-RAIL-001
Origen: scanner backlog 2026-05-24 decimoctava pasada.
Casos: `directorDecisionInstructionsV0` sigue instruyendo al director a no usar
palabras como `provider`, `model`, `db`, `sql`, `runtime`, `adapter`, `git` o
`home` en `director_decisions.json`, aunque la politica vigente permite
vocabulario operativo opaco y corta solo secretos efectivos o payloads crudos.
Decision pendiente: sustituir lista textual por politica positiva de refs
opacas y datos no publicables, alineada con el helper comun o una matriz local
de frontera.
Resolucion 2026-05-25: cerrado en codigo. El prompt de
`director_decisions.json` usa politica positiva de refs opacas y datos no
publicables; los validadores comparten `orquesta-rails` como owner de detalle
sensible y mantienen el corte para secretos efectivos, rutas privadas,
prompts/transcripts crudos y payloads completos.
Rework de revision 2026-05-25: la evidencia documental de T62 queda acotada a
prompt/validacion focal y no reutiliza smokes de servidor de T65; el contexto
`required ref_only` se resuelve con lectura local y recibo explicito en ACK.
Test futuro:
`go test -count=1 ./modulos/orquesta-app-codex-stack ./modulos/orquesta-director-agent ./modulos/orquesta-director-agent-file-source ./modulos/orquesta-rails ./modulos/orquesta-runtime-codex-delivery`.
Backlog: `T62 director-decisions-prompt-rail-sync`.
```

```text
ID: RAIL-CAND-DIRECTOR-DECISIONS-SIDECAR-001
Origen: scanner backlog 2026-05-24 decimoctava pasada.
Casos: el descriptor de `director_decisions.json` se deriva desde el `AckPath`
del recibo Codex y la fuente filtra por run/ACK reflejado. Cierre 2026-05-25:
el sidecar ya obtiene receipt propio con hash, ACK productor, correlacion y
estado de consumo durable en el store de receipts; el source valida hash/tamano
antes de decodificar y marca `consumed`.
Decision aplicada: convertir el sidecar en decision ejecutable solo con receipt
correlado; omitir receipt ya consumido y bloquear payload cambiado bajo el
mismo receipt con conflicto publico compacto.
Test:
`go test -count=1 ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-director-agent-file-source ./modulos/orquesta-app-codex-stack ./modulos/orquesta-app-director-service ./cmd/orquesta-server`.
Backlog: `T63 director-decisions-sidecar-receipt-correlation`.
```

```text
ID: RAIL-CAND-DIRECTOR-CYCLE-SOT-001
Origen: scanner backlog 2026-05-24 decimonovena pasada.
Casos: la espina `orquesta-director-cycle`/`scheduler`/`runner`/`tick-input` ya
existe y `orquesta-orchestration-core` la importa, pero la foto raiz y la matriz
no la nombran; `orquesta-orchestration-core/AGENTS.md` conserva pendientes
viejos de review/tests/rework/cierre como si el ciclo offline siguiera abierto.
Decision 2026-05-25: sincronizados documentos de autoridad y AGENTS locales con
estados separados por codigo offline, composicion residente, smoke real y
proveedor/OPES real. No se abre rail de validacion nuevo por T64; futuros
"pendiente Director" deben declarar el tipo de hueco.
Test futuro:
`go test -count=1 ./modulos/orquesta-director-cycle ./modulos/orquesta-director-scheduler ./modulos/orquesta-director-runner ./modulos/orquesta-director-tick-input ./modulos/orquesta-orchestration-core`.
Backlog: `T64 director-cycle-source-of-truth-sync`.
```

```text
ID: RAIL-CAND-DIRECTOR-CYCLE-RESIDENT-001
Origen: scanner backlog 2026-05-24 decimonovena pasada.
Casos: existen pruebas locales de cycle, scheduler, runner, state-file/outbox
(referencia historica retirada despues por `ARCH-ORQ-20260630-004`) y dispatch,
pero falta smoke de composicion residente que demuestre
`DirectorCycleStepV0` con `FileOutboxLedgerV0`, ACK parcial, reinicio y reentrada
sin duplicar comandos ni cerrar con outbox pendiente.
Decision 2026-05-25: anadido smoke focal de servidor temporal en
`cmd/orquesta-server`: crea stack con `state_dir` durable, ejecuta
`DirectorCycleStepV0`, reinicia la composicion sobre el mismo ledger, bloquea el
segundo paso por outbox pendiente, despacha por puerto fake con ACK y valida
causa publica de ACK fallido con refs compactas.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-director-service ./modulos/orquesta-app-codex-stack ./modulos/orquesta-state-file ./modulos/orquesta-outbox-dispatch`.
Backlog: `T65 director-cycle-resident-restart-smoke`.
```

```text
ID: RAIL-CAND-NEUTRAL-PROCESS-STOP-001
Origen: scanner backlog 2026-05-24 decimonovena pasada.
Casos: `DIR-P006` ya tiene cierre focal neutral en `cmd/orquesta-server` con
proceso temporal real, `DirectorCycleStepV0`, outbox durable,
`StopRuntimeAgent`, ACK/evidencia e idempotencia de replay tras reinicio.
Decision: cerrado como smoke neutral sin Codex; no cubre escalado avanzado,
politica de launch/env ni shutdown checkpoint.
Test:
`go test -count=1 ./modulos/orquesta-runtime ./modulos/orquesta-director ./modulos/orquesta-director-scheduler ./modulos/orquesta-director-cycle ./modulos/orquesta-orchestration-core ./cmd/orquesta-server`.
Backlog: `T66 neutral-process-stop-e2e`.
```

```text
ID: RAIL-CAND-DIRECTOR-SUPERVISOR-BURST-SOT-001
Origen: scanner backlog 2026-05-24 vigesima pasada.
Casos: `orquesta-orchestration-core` ya importa
`orquesta-director-supervisor` y `orquesta-director-supervised-burst`, pero la
foto raiz, matriz y T64 nombran sobre todo cycle/scheduler/runner/tick-input.
Sin fuente de verdad, futuros agentes pueden duplicar la politica de
repeticion/parada o reabrir pendientes viejos de review/tests/cierre.
Decision pendiente: sincronizar docs de autoridad y AGENTS locales para declarar
la cadena de supervision de pasos, separandola de `run-supervisor` global y sin
dar por cerrado un smoke residente que aun no existe.
Test futuro:
`go test -count=1 ./modulos/orquesta-director-supervisor ./modulos/orquesta-director-supervised-burst ./modulos/orquesta-orchestration-core`.
Backlog: `T67 director-supervisor-burst-source-of-truth-sync`.
```

```text
ID: RAIL-CAND-DIRECTOR-BURST-BUDGET-RESIDENT-001
Origen: scanner backlog 2026-05-24 vigesima pasada.
Casos: el servidor configura presupuestos anidados de supervisor global y drain
del Director (`MaxTicks`, `MaxRunsPerTick`, `MaxExecutions`, `MaxBursts`,
`MaxStepsPerBurst`, `MaxCommands`). Hay pruebas locales, pero falta smoke de
composicion que demuestre corte por `wait_outbox`, `wait_external`,
`stop_max_steps` o `stop_error` sin duplicar comandos ni tratar presupuesto
agotado como exito terminal.
Decision pendiente: probar servidor temporal con estado durable/fake, acciones
finales visibles, outbox pendiente conservado, replay idempotente y distincion
entre falta real de trabajo y trabajo bloqueado por presupuesto.
Test futuro:
`go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server ./modulos/orquesta-run-supervisor ./modulos/orquesta-orchestration-core ./modulos/orquesta-director-supervisor ./modulos/orquesta-director-supervised-burst ./modulos/orquesta-outbox-dispatch`.
Backlog: `T68 director-supervised-burst-resident-budget-smoke`.
```

```text
ID: RAIL-CAND-NESTED-SUPERVISOR-STOP-REASON-001
Origen: scanner backlog 2026-05-24 vigesima pasada.
Casos: `run-supervisor`, `director-supervisor` y
`director-supervised-burst` exponen razones de parada locales. Auditoria,
stats, cola e idle autoprogramming pueden interpretar distinto `no_execution`,
`max_ticks`, `wait_outbox`, `wait_external`, `stop_max_steps` o `stop_error` si
cada capa proyecta su propio vocabulario.
Decision cerrada 2026-05-25: `orquesta_supervisor_stop_reason_projection.v0`
queda como proyeccion publica comun con categoria, reason estable, refs y
contadores compactos. `orquesta-server` ya audita summaries compactos y el idle
autoprogramming consulta la proyeccion `idle_no_execution`, no razones locales
de outbox, espera, presupuesto o error.
Test ejecutado:
`go test -count=1 ./modulos/orquesta-run-supervisor ./modulos/orquesta-director-supervisor ./modulos/orquesta-director-supervised-burst ./modulos/orquesta-server ./cmd/orquesta-server`.
Backlog: `T69 nested-supervisor-stop-reason-contract`.
```

```text
ID: RAIL-CAND-CORE-WORKFLOW-DOC-POLICY-001
Origen: scanner backlog 2026-05-24 vigesimoprimera pasada.
Casos: docs locales de `orquesta-core-workflow` siguen afirmando que comandos,
eventos o JSON rechazan palabras como `runtime`, `provider`, `DB`, `SQL`,
`HOME`, `modelo` o `Codex`, mientras el rail vivo permite vocabulario operativo
opaco y corta solo valores sensibles efectivos.
Decision 2026-05-25: docs/pruebas/decisiones locales sincronizados con
`orquesta-rails`: refs opacas y terminos arquitectonicos pasan; secretos,
rutas privadas, prompts/transcripts crudos y payloads masivos se bloquean. No
se cambian validadores ni adaptadores.
Test:
`go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-rails`.
Backlog: `T70 core-workflow-docs-rail-policy-sync`.
```

```text
ID: RAIL-CAND-DOMAIN-WORK-FILE-BUDGET-001
Origen: scanner backlog 2026-05-24 vigesimoprimera pasada.
Casos: el rail OrquestaV2 de ficheros Go menores de 300 lineas no solo afecta a
stack/director. `orquesta-domain-work-sql/store_v0.go`,
`orquesta-document-plan-expander/expander_v0.go` y los `job_creator_v0.go` de
memory/file superan el limite o se acercan mezclando responsabilidades.
Decision 2026-05-26: partir por responsabilidad local `store`, idempotencia,
jobs, filtros/listado, fingerprints, dialecto SQL, helpers y tests; no queda
baseline vivo >300 en el write-set T71.
Test:
`go test -count=1 ./modulos/orquesta-domain-work ./modulos/orquesta-domain-work-memory ./modulos/orquesta-domain-work-file ./modulos/orquesta-domain-work-sql ./modulos/orquesta-document-plan-expander`.
Backlog: `T71 domain-work-adapters-file-budget-split`.
Estado: cubierto
```

```text
ID: RAIL-CAND-DOMAIN-WORK-ARTIFACT-MAP-001
Origen: scanner backlog 2026-05-24 vigesimoprimera pasada.
Casos: el mapa `work_kind -> expected_artifact_type` aparece duplicado en
`orquesta-document-plan-expander`, `orquesta-app-codex-stack`,
`orquesta-opes-bridge` y tests de `cmd/orquesta-server`.
Decision 2026-05-25: `orquesta-domain-work` es el owner neutral del mapa por
`ExpectedDomainWorkArtifactTypeForWorkKindV0`. Expander, builder de entregas,
bridge OPES y drain de servidor consumen/proban esa correspondencia; unknown
work kinds conservan fallback coherente `work_delivery`.
Test futuro:
`go test -count=1 ./modulos/orquesta-domain-work ./modulos/orquesta-document-plan-expander ./modulos/orquesta-app-codex-stack ./modulos/orquesta-opes-bridge ./cmd/orquesta-server`.
Backlog: `T72 domain-work-artifact-contract-map-owner` cerrado.
```

```text
ID: RAIL-CAND-FACTORY-CONNECTOR-POLICY-001
Origen: scanner backlog 2026-05-24 vigesimosegunda pasada.
Casos: `modulos/orquesta-factory/appspec_request_v0.go` rechaza nombres de
integracion si contienen substrings como `postgres`, `sqlite`, `runtime`,
`filesystem`, `llm`, `queue`, `deploy`, `database` o `db`. Esa lista evita
proveedores concretos en el contrato base, pero tambien puede cortar
capacidades de dominio validas o refs opacas reparables.
Decision pendiente: sustituir el rail textual por politica de capacidad vs
proveedor concreto. Bloquear DSN, credenciales, SDK/proveedor elegido o backend
impuesto; permitir vocabulario operativo si el director/adaptador puede
normalizarlo sin riesgo.
Resolucion 2026-05-25: `orquesta-factory` aplica una politica de
capacidad/proveedor para integraciones. Ya no bloquea `db`, `runtime`, `queue`,
`cola`, `deploy` o `database` por substring cuando expresan capacidad; conserva
rechazo recuperable para proveedor/backend concreto, SDK cloud, credenciales y
DSN. Web/MCP solo proyectan el error publico e i18n.
Revalidacion 2026-05-25: `postgres database` y `cloud sdk` quedan cubiertos
como proveedor/backend impuesto; `database audit`, `runtime metrics` y
`cola de tareas` siguen aceptados como capacidades.
Test futuro:
`go test -count=1 ./modulos/orquesta-factory ./modulos/orquesta-mcp ./modulos/orquesta-web`.
Backlog: `T73 factory-connector-capability-policy-port`.
```

```text
ID: RAIL-CAND-DEPLOY-CONTRACT-ISLAND-001
Origen: scanner backlog 2026-05-24 vigesimosegunda pasada.
Casos: `DeploymentPlan v0` aparece en factory/MCP y `orquesta-deploy`, pero
`orquesta-deploy` no esta en el mapa raiz de capas y no hay consumidores Go
fuera del propio modulo. Las tareas de deploy pueden quedar como docs/write-set
sin invocar el contrato dry-run ni producir evidencia por puerto.
Resolucion 2026-05-25: T74 cerrado con `orquesta-deploy` como owner de
`DeploymentPlan v0`, puerto dry-run de composicion y consumidores en
`orquesta-app-planner`, factory y MCP. Sigue prohibido ejecutar Docker,
Kubernetes, cloud, filesystem productivo o secretos desde el nucleo.
Test futuro:
`go test -count=1 ./modulos/orquesta-deploy ./modulos/orquesta-factory ./modulos/orquesta-app-planner ./modulos/orquesta-mcp`.
Backlog: `T74 deployment-plan-composition-wiring`.
```

```text
ID: RAIL-CAND-I18N-DOCS-CONTRACT-ISLAND-001
Origen: scanner backlog 2026-05-24 vigesimosegunda pasada.
Casos: `orquesta-i18n-docs` define builder/validator de bundles, loader shape
y docs generadas, pero no tiene consumidores Go fuera del modulo. Web/factory
mantienen mapas i18n locales y la foto vigente aun habla de brechas historicas
de i18n.
Decision 2026-05-25: `orquesta-i18n-docs` sigue vivo como owner activo.
Factory/web/MCP consumen contrato o proyeccion por puerto; no debe marcarse
historico ni duplicarse con otro owner local de bundles, loader o docs
generadas.
Test futuro:
`go test -count=1 ./modulos/orquesta-i18n-docs ./modulos/orquesta-factory ./modulos/orquesta-web ./modulos/orquesta-mcp`.
Backlog: `T75 i18n-docs-active-composition-owner` cerrado en corte focal
2026-05-25.
```

```text
ID: RAIL-CAND-APPSPEC-ENTRYPOINTS-001
Origen: scanner backlog 2026-05-24 vigesimotercera pasada.
Casos: MCP expone `orquesta.apps.preparar_orquestacion.v0` por
`orquesta-app-runner`/`AppPlan` y tambien `orquesta.apps.arrancar_director.v0`
por `app-director-service`. Sin estado/freshness claro, una IA puede elegir el
camino historico sin Director V2 para un trabajo que exige juicio, waits,
review/tests y cierre causal.
Decision pendiente: declarar ruta operativa preferente, marcar `app-runner`
como preview/legacy/compatibilidad o enlazarlo con
`OperationalDirectorPlanStateV0` antes de usarlo para ejecucion real.
Test futuro:
`go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-app-runner ./modulos/orquesta-app-planner ./modulos/orquesta-app-director-intake ./modulos/orquesta-app-director-service`.
Backlog: `T76 appspec-entrypoints-director-v2-routing`.
Estado: cubierto 2026-05-25; refrescado 2026-05-26. `route_policy` publica
`arrancar_director` como preferente en resultado y descriptor compacto, y
`app-runner` bloquea con `director_v2_required` si se exige Director V2.
```

```text
ID: RAIL-CAND-APP-CHANGE-READINESS-001
Origen: scanner backlog 2026-05-24 vigesimotercera pasada.
Casos: `orquesta-app-change-director-source/readiness_v0.go` usa
`forbiddenAutoPlanFragmentsV0` con `runtime`, `provider`, `model`, `db`, `sql`,
`codex`, `docker`, `home` y otros fragments. `external_work_v0.go` tambien
reescribe vocabulario operativo en criterios. Puede bloquear refs opacas o
ocultar semantica de dominio fuera del scope de `T15`.
Decision pendiente: mover readiness/sanitizacion a politica por campo alineada
con `orquesta-rails`; permitir vocabulario operativo opaco y cortar solo valores
sensibles efectivos, material crudo o efectos externos no autorizados.
Decision 2026-05-25: resuelto en `orquesta-app-change-director-source`; la
fuente usa `orquesta-rails` por campo para readiness, conserva vocabulario
operativo opaco y bloquea solo detalle sensible/raw efectivo.
Revision 2026-05-26: retry Orquesta V2 mantiene T77 cerrado en codigo de
alcance. El test obligatorio no puede declararse verde porque
`orquesta-app-codex-stack` arrastra una compilacion rota de
`orquesta-observability` fuera del write-set. Se observaron fallos externos por
tipos timeline ausentes y, en un reintento posterior, por
`cloneWorkspaceTimelineProjectionV0` redeclarado entre
`workspace_timeline_filter_v0.go` y `workspace_timeline_clone_v0.go`. Retry
2026-05-26 en esta tarea: se observo un fallo externo transitorio en
`orquesta-web` por deriva de `WorkspaceTimelineItemV0`, pero la reejecucion
posterior del comando requerido paso completa. T77 queda cerrado en codigo de
alcance y no requiere editar modulos fuera del write-set.
Validacion OrquestaV2 2026-05-26: contexto `ref_only` resuelto por evidencia
explicita de ACK; el comando requerido paso completo y no se detecta nueva
regresion del rail.
Test futuro:
`go test -count=1 ./modulos/orquesta-app-change-director-source ./modulos/orquesta-app-change ./modulos/orquesta-rails ./modulos/orquesta-app-codex-stack`.
Backlog: `T77 app-change-director-source-readiness-rail-policy`.
```

```text
ID: RAIL-CAND-DIRECTOR-DECISION-PLANSTATE-MERGE-001
Origen: scanner backlog 2026-05-24 vigesimotercera pasada.
Casos: `docs/corte_plan_state_director_decisions_2026-05-21.md` deja pendiente
fusionar nuevas tasks operativas en un `OperationalDirectorPlanStateV0`
existente. El ensure actual devuelve si el state ya existe, por lo que una
decision tardia que abre otra ola necesita contrato de merge/reentrada y replay.
Decision aplicada 2026-05-25: `app-director-service` fusiona en el plan state
abierto las tasks operativas nuevas marcadas por `director_decision`, reabre
`wait_subagents` con scope acotado por agent refs de la nueva task y mantiene
replay idempotente. Si falta metadata operativa, la task queda fuera de este
merge y no crea wait ambiguo.
Revalidacion OrquestaV2 2026-05-26: el paquete `ref_only` se resolvio por
lectura local y evidencia explicita en ACK; el comando requerido paso completo
sin reabrir rails ni ampliar write-set.
Test futuro:
`go test -count=1 ./modulos/orquesta-director-agent-workflow ./modulos/orquesta-app-director-service ./modulos/orquesta-orchestration-core ./modulos/orquesta-state-file ./modulos/orquesta-app-codex-stack`.
Backlog: `T78 director-decisions-existing-planstate-merge`.
```

```text
ID: RAIL-CAND-BACKLOG-DOC-SHARD-001
Origen: scanner backlog 2026-05-24 vigesimocuarta pasada.
Casos: `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`,
`docs/rail_errors_observados_2026-05-23.md` y
`docs/duplicaciones_railes_pendientes_2026-05-24.md` concentran miles de lineas
append-only. `idle_self_improvement_backlog_planner_v0.go` lee un unico backlog
hardcodeado y no conserva fichero/hash por shard. T43 cubre lease de merge,
pero no reduce el hotspot documental.
Decision pendiente: crear indice/shards compatibles con el parser, preservar
linea/hash/fichero por seccion Txx y no borrar ni truncar historico durante la
migracion.
Decision aplicada 2026-05-25: `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
declara indice vivo de shards, el planner lee indice/shards con fallback al
documento historico y cada request conserva fichero, linea y hash en
`BacklogScanDocumentV0`. La migracion fisica a shards nuevos queda pendiente con
write-set propio; no se borra ni trunca historico.
Revalidacion OrquestaV2 2026-05-25: contexto `ref_only` requerido resuelto por
lectura local/evidencia ACK; se mantiene T79 como contrato inicial cerrado sin
reabrir T43 ni migracion fisica de shards.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-autoprogramming ./modulos/orquesta-server`.
Backlog: `T79 autoprogramming-backlog-doc-sharding`.
```

```text
ID: RAIL-CAND-DOMAIN-WORK-HTTP-EGRESS-001
Origen: scanner backlog 2026-05-24 vigesimocuarta pasada.
Casos: `ORQUESTA_DOMAIN_WORK_HTTP_BASE_URL` activa el adaptador HTTP neutral y
`orquesta-domain-work-http` valida esquema/base/path, pero no tiene politica de
host/egress, allowlist, modo smoke ni redaccion especifica del destino. Un error
de composicion podria apuntar a endpoints no temporales o redes internas.
Decision pendiente: anadir politica opt-in de destino, rechazo de credenciales
en URL, allowlist o modo smoke declarado, errores publicos y auditoria compacta
sin URL sensible ni payload crudo.
Decision aplicada 2026-05-25: `orquesta-domain-work-http` normaliza destino con
politica `smoke_local` o `allowlist`, rechaza credenciales/query/fragment,
clasifica loopback, metadata, red interna y externo desconocido, y bloquea desde
`cmd/orquesta-server` cualquier `ORQUESTA_DOMAIN_WORK_HTTP_BASE_URL` sin
`ORQUESTA_DOMAIN_WORK_HTTP_EGRESS_MODE` explicito. Paths de `create_job` y
`submit_artifact` quedan relativos, sin query ni host override.
Revalidacion 2026-05-26: pasa `go test -count=1 ./...`; el rail queda cerrado
para T80 sin ampliar el contrato puro `orquesta-domain-work`.
Backlog: `T80 domain-work-http-egress-policy`.
```

```text
ID: RAIL-CAND-OBSERVABILITY-GLOBAL-TIMELINE-001
Origen: scanner backlog 2026-05-24 vigesimocuarta pasada.
Casos: `docs/op_096_control_total_estado_proyecto_y_estadisticas.md` declara
operativo el control por agente/proyecto y deja abierto el plano global de
workspace, coste y timeline. `orquesta-observability` y la auditoria del
servidor existen, pero no hay contrato unico para API/MCP/web que agregue
workspace sin shell, transcript crudo o stores internos.
Decision pendiente: definir puerto de lectura global con fuentes declaradas,
paginacion temporal, redaccion por campo y `not_available` para fuentes ausentes
en vez de inferencias ad hoc.
Estado 2026-05-25: puerto inicial `WorkspaceTimelineQueryV0` definido en
`orquesta-observability`; API/MCP/web comparten el contrato y el servidor
residente expone `/api/v0/workspace/timeline` con fuentes ausentes como
`not_available`.
Test futuro:
`go test -count=1 ./modulos/orquesta-observability ./modulos/orquesta-server ./modulos/orquesta-mcp ./modulos/orquesta-web ./cmd/orquesta-server`.
Backlog: `T81 observability-global-workspace-timeline`.
```

```text
ID: RAIL-CAND-GOVERNANCE-PUBLIC-SHAPE-001
Origen: scanner backlog 2026-05-24 vigesimoquinta pasada.
Casos: `GovernanceCatalogQueryHTTPHandlerV0` espera request con `filters` y
responde `result/errors`, mientras `GovernanceCatalogCliReaderV0` envia
`module|role|phase|tags` en raiz y espera `GovernanceCatalogQueryResultV0`
directo con errores `errores/codigo`. La ruta tampoco aparece cableada en
`orquesta-http-gateway`, `orquesta-app-gateway` ni `cmd/orquesta-server`.
Decision pendiente: fijar un unico shape publico para request/response/error,
actualizar CLI/handler/MCP docs y cablear gateway con provider inyectado; si no
hay provider, devolver error publico recuperable.
Test futuro:
`go test -count=1 ./modulos/orquesta-governance ./modulos/orquesta-cli ./modulos/orquesta-mcp ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway ./cmd/orquesta-server`.
Backlog: `T82 governance-catalog-public-route-shape-sync`.
Estado 2026-05-25: cerrado. `GovernanceCatalogQueryHTTPHandlerV0`,
`GovernanceCatalogCliReaderV0`, descriptor MCP compartido, gateway HTTP,
app-gateway y `cmd/orquesta-server` quedan sincronizados en el shape publico
`{request_id, correlation_id, filters}` -> `{request_id, correlation_id,
effective, counters}` y errores `{errors:[{code, field}]}`. La ruta responde
`governance_catalog_source_unavailable` cuando no hay provider y no activa
historicos ni lee DB v1.
```

```text
ID: RAIL-CAND-OPERATIONAL-STATUS-PUBLIC-SOURCE-001
Origen: scanner backlog 2026-05-24 vigesimoquinta pasada.
Casos: CLI, web y MCP anuncian `OperationalStatusQueryV0` y endpoint
`/api/v0/operational-status/query`, pero `orquesta-observability` solo incluye
DTOs/validadores y adapter en memoria. Falta source residente y wiring HTTP que
derive `DiagnosticoCompactoV0` real desde stats/cola/runs/runtime con redaccion.
Decision pendiente: publicar source por puerto y handler comun; fuentes
ausentes deben producir warning/`not_available`, no datos inventados ni lecturas
directas de shell, stores internos, runtime dirs o transcripts.
Test futuro:
`go test -count=1 ./modulos/orquesta-observability ./modulos/orquesta-server ./modulos/orquesta-cli ./modulos/orquesta-web ./modulos/orquesta-mcp ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway ./cmd/orquesta-server`.
Backlog: `T83 operational-status-public-query-source`.
Cierre 2026-05-25: resuelto por source residente `OperationalStatusQueryV0`,
handler HTTP comun y wiring de servidor/gateway. La salida valida
`DiagnosticoCompactoV0`, no expone paths ni runtime dirs y conserva warnings
`not_available` para fuentes ausentes sin inventar datos.
```

```text
ID: RAIL-CAND-FUNCTION-CONTRACT-PUBLIC-ROUTE-001
Origen: scanner backlog 2026-05-24 vigesimoquinta pasada.
Casos: la CLI implementa `POST /api/v0/core/function-contracts/list` y
`/view`, pero `core` conserva esas operaciones como candidatas documentales y
`core-workflow` solo proyecta refs compactas `FunctionContractPublished`.
Faltan store/index read-only, shape de gateway y caso de payload insuficiente.
Decision 2026-05-25: consulta read-only definida por puerto desde eventos/stores
causales. Cuando solo hay refs `FunctionContractPublished` sin payload
contractual completo, `list` expone estado `evidencia_insuficiente` y `view`
bloquea con error publico verificable. `registrar` se mantiene como operacion
no promovida.
Test futuro:
`go test -count=1 ./modulos/orquesta-core ./modulos/orquesta-core-workflow ./modulos/orquesta-orchestration-core ./modulos/orquesta-state-file ./modulos/orquesta-cli ./modulos/orquesta-mcp ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway ./cmd/orquesta-server`.
Backlog: `T84 function-contract-readonly-public-index`.
```

```text
ID: RAIL-CAND-SERVER-STATUS-CONFIG-SNAPSHOT-001
Origen: scanner backlog 2026-05-24 vigesimosexta pasada.
Casos: `/api/v0/server/status` devuelve `StateV0` consumido por CLI, pero el
DTO incluye `project_work_dir` y `runtime_work_dir` crudos y no publica el
snapshot efectivo/redactado de variables residentes que la CLI documenta como
necesario para auditoria de automejora. Eso mezcla estado publico, config local
y paths operativos en una sola respuesta.
Decision pendiente: separar estado publico compacto de snapshot de config
efectiva, redactar paths/HOME/runtime dirs/comandos/proveedor/tokens por campo
y permitir detalle local solo con opt-in de composicion.
Test futuro:
`go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-cli ./modulos/orquesta-web ./modulos/orquesta-mcp ./modulos/orquesta-http-gateway ./cmd/orquesta-server`.
Backlog: `T85 server-status-config-snapshot-redaction`.
```

```text
ID: RAIL-CAND-BOOTSTRAP-APPSPEC-LEGACY-ROUTE-001
Origen: scanner backlog 2026-05-24 vigesimosexta pasada.
Casos: `BootstrapAppSpecCliEndpointV0` apunta a
`/api/v0/director/bootstrap/appspec`, mientras gateway/web usan
`/api/v0/apps/spec` y `/api/v0/apps/director`. No aparece route constant,
handler de app-gateway ni wiring de servidor para esa ruta legacy.
Decision pendiente: decidir si `BootstrapProyectoDesdeAppSpec` sigue vivo como
compatibilidad cableada, queda bloqueado con error publico o migra al flujo
vigente; no dejar cliente fino apuntando a transporte inexistente.
Estado: cerrado 2026-05-26. El contrato puro permanece disponible en
director/web/MCP, pero la ruta HTTP legacy queda en cuarentena: la CLI devuelve
`contrato_no_configurado` con `route_policy`, no llama
`/api/v0/director/bootstrap/appspec` y remite al flujo vigente
`/api/v0/apps/director` / `orquesta.apps.arrancar_director.v0`.
Test futuro:
`go test -count=1 ./modulos/orquesta-cli ./modulos/orquesta-director ./modulos/orquesta-factory ./modulos/orquesta-web ./modulos/orquesta-mcp ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway ./cmd/orquesta-server`.
Backlog: `T86 bootstrap-appspec-legacy-route-quarantine`.
```

```text
ID: RAIL-CAND-SERVER-LIVENESS-READINESS-001
Origen: scanner backlog 2026-05-24 vigesimosexta pasada.
Casos: scripts de smoke y daemon tratan `/healthz` como "servidor listo", pero
el handler solo responde liveness `status=ok`. La readiness operativa vive en
`startup_ready/startup_status` de `/api/v0/server/status`, incluyendo cleanup y
reconciliacion.
Estado: cerrado 2026-05-25. Contrato fijado: `/healthz` es liveness,
`/api/v0/server/readiness` es readiness operativa con `startup_ready`,
`startup_status`, mensaje publico y evidence refs. El endpoint devuelve 503
cuando startup cleanup/reconciliacion no esta listo y no expone paths ni runtime
dirs. Daemon y smokes que preparan runs, drenan OPES, lanzan Codex, ejecutan
Director/domain_work o automejora esperan readiness; los checks de `/healthz`
restantes son liveness de socket o apps externas temporales.
Tests de cierre:
`go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-cli ./cmd/orquesta-server` y `bash -n scripts/*.sh`.
Backlog: `T87 server-liveness-readiness-contract`.
```

```text
ID: RAIL-CAND-SERVER-FIRST-USAGE-DOC-ROUTE-001
Origen: scanner backlog T119.
Casos: `docs/uso_actual_app_orquesta.md` anunciaba `./orquesta serve`,
OpenClaw, AP-077 y rutas `/api/*` sin version como si fueran uso operativo
vigente. Un agente podia preparar clientes o pruebas contra control-plane V1.
Decision 2026-05-26: sincronizar el manual con server-first actual:
`cmd/orquesta-server`, readiness `/api/v0/server/readiness`, status versionado,
web/CLI como clientes finos, API `/api/v0/*`, MCP/toolbelt y cuarentena de
aliases legacy.
Test: `go test -count=1 ./modulos/orquesta-cli ./modulos/orquesta-web ./cmd/orquesta-server`.
Estado: cubierto.
Backlog: `T119 server-first-usage-doc-route-sync`.
```

```text
ID: RAIL-CAND-FEDERATED-MODULE-BACKLOG-001
Origen: scanner backlog 2026-05-24 vigesimoseptima pasada.
Casos: el planner residente lee solo
`docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`, pero modulos
mantienen backlog local en `docs/tareas.md` o README (`APG-*`, `RTDELIVERY-*`,
`SSH-*`, `DEP-*`). Sin indice federado, el residente puede no ver pendientes
locales vigentes o duplicarlos como Txx sin alias, owner, linea/hash ni estado.
Decision pendiente: definir indice federado con source path/line/hash,
freshness, owner, alias local y Txx relacionado; el planner solo debe programar
entradas locales promocionadas o vigentes, y dejar docs historicos como
evidencia no ejecutable.
Resolucion 2026-05-25: el planner incorpora un indice federado declarado en el
backlog global. Las fuentes locales `vigente`/`promocionada` generan entradas
con owner, alias, hash y lease; fuentes sin tests/owner abren revision
documental. La revalidacion obligatoria queda bloqueada por compilacion rota en
`modulos/orquesta-observability`, fuera del write-set de T88.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-autoprogramming ./modulos/orquesta-mcp ./modulos/orquesta-server`.
Backlog: `T88 federated-module-backlog-index`.
```

```text
ID: RAIL-CAND-DIRECTOR-OPERATIVO-DOC-STATE-001
Origen: scanner backlog 2026-05-24 vigesimoseptima pasada.
Casos: `modulos/orquesta-director-operativo/README.md` sigue declarando
pendientes wait por cohorte/ola, review/rework/replan/cierre durable y recursion
Codex real, aunque las fuentes vigentes ya cierran WaitAgentRefs, ciclo offline
del PlanState y `CODEX-WAVE-REAL`/`CODEX-RECURSION-REAL`; el pendiente real
abierto es OPES temporal de derivados/cierre salvo regresion demostrada.
Resolucion 2026-05-26: T89 sincroniza README/docs locales con la foto vigente,
marca `CODEX-WAVE-REAL` y `CODEX-RECURSION-REAL` como cerrados salvo regresion
demostrada y deja el pendiente real en OPES temporal de derivados/cierre. El
check focal vive en
`TestDirectorOperativoLocalDocsAlineadosConFotoVigenteV0`.
Test:
`go test -count=1 ./modulos/orquesta-director-operativo`.
Backlog: `T89 director-operativo-local-doc-state-sync`.
```

```text
ID: RAIL-CAND-RESIDUAL-GO-FILE-BUDGET-001
Origen: scanner backlog 2026-05-24 vigesimoctava pasada.
Casos: el line-count deja ficheros Go >300 lineas fuera de los shards ya
identificados en T52-T54/T71, especialmente en `orquesta-runtime-codex-delivery`,
`orquesta-runtime-required-test`, `orquesta-run-coordinator`,
`orquesta-external-work-run`, `orquesta-app-gateway`, `orquesta-director`,
`orquesta-core-workflow` y tests de `cmd/orquesta-server`. El rail de tamano
queda como advisory generico si no hay shard residual con baseline.
Decision pendiente: medir baseline por fichero, partir por responsabilidad
local y conservar T52-T54/T71 como owners de los focos principales; no cerrar
ACK terminal estricto por `ACK.files` sin snapshot/worktree real.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-runtime-required-test ./modulos/orquesta-run-coordinator ./modulos/orquesta-external-work-run ./modulos/orquesta-app-gateway ./modulos/orquesta-director ./modulos/orquesta-core-workflow ./cmd/orquesta-server`.
Backlog: `T90 residual-go-file-budget-splits`.
```

```text
ID: RAIL-CAND-SMOKE-SCRIPT-OPS-LIB-001
Origen: scanner backlog 2026-05-24 vigesimoctava pasada.
Casos: varios smokes largos duplican prerequisitos, confirmaciones opt-in,
arranque/parada de servidor temporal, espera de `/healthz`, cleanup, parseo JSON
y redaccion de salida. Esa duplicacion puede divergir de T21/T87 y hacer que un
script lance Codex/OPES/red antes de readiness o sin guarda equivalente.
Decision pendiente: extraer helpers en `scripts/lib` para prerequisitos,
confirmacion, liveness/readiness, cleanup y redaccion, manteniendo por script la
politica de riesgo y sin imprimir HOME, rutas privadas, prompts, transcripts,
tokens ni payloads crudos por defecto.
Test futuro:
`bash -n scripts/*.sh scripts/lib/*.sh` y focos de smokes con confirmaciones
fake/temporales cuando existan.
Backlog: `T91 smoke-script-ops-library`.

Resolucion 2026-05-26: primer corte mecanico en `scripts/lib/smoke_common.sh`.
Los smokes OPES afectados reutilizan helpers comunes de prerequisitos,
confirmacion, URL local, HTTP JSON, parseo compacto y readiness operativa por
`/api/v0/server/readiness`; no se usa `/healthz` como readiness ni se relajan
guardas opt-in.
```

```text
ID: RAIL-CAND-CAPACITY-DECISION-POLICY-001
Origen: scanner backlog 2026-05-24 vigesimonovena pasada.
Casos: `RequestCapacityDecision` se despacha en el stack con
`CapacityDecisionExecutorV0`, pero el executor usa configuracion estatica/default
de tier/reasoning y no un puerto de politica que consuma `orquesta-capacity`.
Los docs de `orquesta-capacity` tambien conservan casos iniciales como
`Comando: pendiente` aunque ya existen DTOs, fixtures y tests Go.
Decision pendiente: anadir puerto de politica/capacidad inyectable en
composicion, mantener fallback fake/legacy auditable y sincronizar docs locales
para distinguir contrato cerrado, wiring pendiente y cuota/benchmarks reales.
Resolucion 2026-05-26: `CapacityDecisionExecutorV0` ya acepta
`CapacityDecisionPolicyPortV0`; `orquesta-app-codex-stack` lo inyecta mediante
adapter de `orquesta-capacity`, y el fallback estatico queda marcado como
legacy auditable. Quedan fuera de este rail fuentes reales de cuota/benchmarks.
Test futuro:
`go test -count=1 ./modulos/orquesta-capacity ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-codex-stack ./modulos/orquesta-runtime ./cmd/orquesta-server`.
Backlog: `T92 capacity-decision-policy-port`.
```

```text
ID: RAIL-CAND-CODEX-PROMPT-TOOLBELT-001
Origen: scanner backlog 2026-05-24 vigesimonovena pasada.
Casos: `cmd/orquesta-server/codex_prompt_hints_v0.go` enumera en el toolbelt
MCP `orquesta.operator.operations.v0`, pero otra linea recomienda usar
`orquesta.operator.directed_query.v0`; el tool existe bajo
`orquesta-operator-mcp` y la exposicion real depende de puertos/transportes
inyectados. Listas libres en prompts, packets y runbooks pueden divergir del
registry real.
Estado: cerrado 2026-05-26. Los hints Codex derivan HTTP de constantes MCP
montadas por gateway y MCP de `MCPTransportToolsV0`; `directed_query` aparece
como tool registrado y `operator.operations` como resource con subtools. Los
hints publican estados compactos de transporte vivo, puerto opt-in no
configurado y resource registrado, sin payloads ni datos sensibles.
Test cerrado:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-app-codex-stack ./modulos/orquesta-runtime-codex ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-app-gateway`.
Backlog: `T93 codex-prompt-toolbelt-source-sync`.
Verificacion OrquestaV2 2026-05-26: contexto `ref_only` validado por
evidencia explicita en ACK y matriz requerida reejecutada.
```

```text
ID: RAIL-CAND-SERVER-AUDIT-WRITE-FAIL-001
Origen: scanner backlog 2026-05-24 trigesima pasada.
Casos: `modulos/orquesta-server/audit_v0.go` ignora el error devuelto por
`AppendAuditEventV0`. Un fallo de escritura JSONL por IO, permisos, espacio o
fichero inaccesible puede dejar al servidor operando sin evidencia durable y
sin diagnostico publico, justo en rutas que preparan automejora, supervisor
ticks, startup checks o mutaciones HTTP.
Decision: registrar el fallo de auditoria en una proyeccion compacta no
recursiva `audit_*`, con contador, codigo publico `audit_write_failed`, evento
compacto, severidad `warning` y timestamp. No devolver rutas locales, HOME,
permisos crudos, payloads HTTP, prompts, transcripts ni tokens. Si tambien
falla el state store, el fallo de auditoria queda en memoria y el fallo de
estado se separa como `state_persist_failed`.
Test: `TestRuntimeV0AuditEventNoSilenciaFalloDeSinkV0`,
`TestRuntimeV0AuditFailureNoRecursivoSiStateStoreFallaV0` y
`go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server`.
Backlog: `T94 server-audit-write-failure-visibility`.
Estado: cubierto local 2026-05-26.
```

```text
ID: RAIL-CAND-SERVER-STATE-PERSIST-FAIL-001
Origen: scanner backlog 2026-05-24 trigesimoprimera pasada.
Casos: `modulos/orquesta-server/runtime_v0.go` devuelve error si falla el
guardado inicial de `serving`, pero `persistStateV0` ignora errores posteriores
de `saveStateV0`. Ese helper se usa en startup checks, ticks de supervisor,
automejora idle, shutdown y marca de error. Un state store sin permisos, sin
espacio, corrupto o bloqueado por IO puede dejar `/api/status` vivo en memoria
mientras el store durable queda obsoleto para reinicio, CLI o auditoria de
operacion.
Decision pendiente: registrar fallo de persistencia de estado en una proyeccion
compacta no recursiva, separar severidad por transicion y exponer codigo publico
`state_persist_failed` sin rutas locales, permisos exactos, HOME, payloads HTTP,
prompts, transcripts ni tokens. No depender del sink de auditoria para conocer
este fallo.
Test futuro:
`go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server`.
Backlog: `T95 server-state-persist-failure-visibility`.
```

```text
ID: RAIL-CAND-SERVER-DAEMON-STOP-PID-001
Origen: scanner backlog 2026-05-24 trigesimosegunda pasada.
Casos: `orquesta-server stop` carga `pid` y `addr` desde el state file,
invoca `/api/v0/server/shutdown` con `forced=true` y despues senala ese PID.
Si el state esta stale, el PID fue reutilizado o el addr no corresponde al mismo
daemon, el CLI puede actuar sobre un proceso equivocado o saltarse shutdown
cooperativo como default silencioso.
Decision implementada 2026-05-26: state expone `process_ref` y
`daemon_epoch_ref` opacos; `orquesta-server stop` compara el snapshot durable
con `/api/status` antes de senalar y bloquea mismatch/stale con error publico.
El shutdown CLI es cooperativo por defecto; `--force` exige `--reason`. No
devuelve HOME, rutas locales, argv completos, prompts, transcripts ni tokens.
Test:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-server-shutdown ./modulos/orquesta-agent-process-registry`.
Backlog: `T96 server-daemon-stop-process-identity`.
```

```text
ID: RAIL-CAND-SERVER-DAEMON-LOGS-001
Origen: scanner backlog 2026-05-24 trigesimosegunda pasada.
Casos: `orquesta-server start` redirige stdout/stderr del proceso residente a
`stdout.log` y `stderr.log` bajo `StateDir`. Esos logs no tienen owner de
redaccion, rotacion, retencion ni acceso, y pueden duplicar auditoria JSONL,
tail Codex o salida de tests con material crudo.
Decision: politica de logs operacionales del daemon definida en el servidor:
resumen redactado por defecto, captura cruda solo opt-in local con limite,
retencion corta y owner separado de auditoria JSONL, tail Codex y required-test
output. La proyeccion publica expone politica compacta y no usa esos logs como
evidencia terminal.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-observability`.
Backlog: `T97 server-daemon-log-redaction-retention`.
Estado: cubierto 2026-05-26
```

```text
ID: RAIL-CAND-EXTERNAL-BRIDGE-INPUT-LEDGER-001
Origen: scanner backlog 2026-05-24 trigesimosegunda pasada.
Casos: el bridge OPES consulta el ledger de entrada antes del submit y lo
actualiza despues de recibir `run_ref`. Si el submit funciona pero falla el
ledger, o si dos drains procesan el mismo job externo en paralelo, puede quedar
una run creada sin claim durable o una entrada `submitted` sobrescrita por otro
`run_ref`.
Decision cerrada 2026-05-26: el bridge externo de OPES registra claim durable
`claimed` antes de crear la run, con `claim_ref`, `correlation_id`,
`idempotency_key`, `change_ref` y `run_ref` prevista. Los drains concurrentes
ven `claimed/submitted` y no relanzan el job. La finalizacion rechaza
sobrescribir `submitted` con otro `run_ref`; si falla tras recibir `run_ref`,
el resultado publico queda en `external_bridge_recovery_required` para forzar
reconciliacion por refs antes de reintentar, sin URL sensible, payload OPES
completo, HOME, rutas locales, tokens ni respuestas crudas.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-connector ./modulos/orquesta-run-queue`.
Backlog: `T98 external-bridge-input-ledger-claim-recovery`.
```

```text
ID: RAIL-CAND-SERVER-HTTP-RESOURCE-001
Origen: scanner backlog 2026-05-24 trigesimotercera pasada.
Casos: `modulos/orquesta-server/runtime_v0.go` crea `http.Server` solo con
`Handler`, y rutas MCP/HTTP mutables decodifican `r.Body` sin limite comun ni
trailing-token check. El transporte MCP real si tiene limite local, pero no
gobierna `prepare-run`, `domain_work`, `run_control` ni el resto de puertos
HTTP.
Decision resuelta 2026-05-26 en el alcance T99 residente: timeouts HTTP del
servidor por politica de composicion, helpers de decode JSON por perfil para
MCP/HTTP y web, rechazo de trailing tokens y auditoria compacta sin body,
prompts, transcripts, tokens, HOME ni rutas privadas.
Test futuro:
`go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-web ./cmd/orquesta-server`.
Backlog: `T99 server-http-resource-guardrails`.
```

```text
ID: RAIL-CAND-STARTUP-REVISION-ARCHIVE-001
Origen: scanner backlog 2026-05-24 trigesimotercera pasada.
Casos: la compactacion de startup escribe `queue_v0.before.json`,
`control_v0.before.json`, `queue_removed.json`, `control_removed.json`,
`manifest.json` y runtime archivado bajo un directorio de revision, y proyecta
la ruta de revision en readiness. T35 decide si compactar; falta politica del
artefacto generado.
Decision aplicada: guardar revision con permisos restrictivos, retencion,
tamano maximo, redaccion por campo y `revision_ref` opaco en status. Material
crudo solo opt-in local, con limite y causa publica.
Test:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-run-control ./modulos/orquesta-state-file`.
Backlog: `T100 startup-revision-archive-redaction-retention`.
Estado: cubierto 2026-05-26. La revision de startup usa snapshots redactados,
manifest con retencion/limite/permisos, runtime archivado como metadata
redactada y proyeccion publica `startup_revision` con `revision_ref` opaco.
```

```text
ID: RAIL-CAND-DOMAIN-WORK-SUBMISSION-LEDGER-001
Origen: scanner backlog 2026-05-24 trigesimotercera pasada.
Casos: `domain_work_delivery_bridge_v0.go` comprueba ledger por
`idempotency_key`, ejecuta `submit_artifact` y registra accepted/rejected al
final. El ledger file/memory permite reemplazar el record bajo la misma key sin
contrato de claim ni recovery de fallo post-submit.
Decision aplicada: claim durable `claimed/submitting` antes de
`submit_artifact`, recovery compacto si existe una claim no terminal y conflicto
publico si una key terminal intenta cambiar receipt/payload o refs causales. No
usar URL, DB, ruta local o nombre de conector como evidencia de tests de
dominio.
Rework OrquestaV2 2026-06-25: esta entrada queda acotada al ledger de salida
`domain_work`. Cualquier referencia a `codexStackReviewGatePolicyV0`, snapshot o
presupuesto de review pertenece a T117 y no debe usarse como cierre ni evidencia
de T101.
Assessment/rework OrquestaV2 2026-06-25: el cierre causal valido de T101 incluye
estado `submitted` con `receipt_ref` antes de `accepted` para recovery de fallo
post-efecto. Una claim `claimed/submitting/submitted` sin reconciliacion no debe
reenviar `submit_artifact`; el Director conserva el trabajo y exige recovery
compacto, no review gate ni snapshot de T117.
Test:
`go test -count=1 ./modulos/orquesta-app-codex-stack ./modulos/orquesta-domain-work ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-connector ./cmd/orquesta-server`.
Backlog: `T101 domain-work-artifact-submission-ledger-recovery`.
Estado: cubierto 2026-05-26.
```

```text
ID: RAIL-CAND-LEGACY-HTTP-JSON-001
Origen: scanner backlog 2026-05-24 trigesimocuarta pasada.
Casos: handlers HTTP publicos fuera del primer scope de T99 conservan decoders
locales: `orquesta-factory-http` usa `io.ReadAll(r.Body)`, governance aplica su
propio `DisallowUnknownFields` y trailing-token check, y MCP replica decoders por
ruta sin limite comun.
Decision pendiente: unificar frontera JSON por perfil para limite de body,
content-type, trailing tokens, campos desconocidos y error publico; los modos
legacy deben declararse con test. No devolver body crudo, prompts, transcripts,
rutas locales, HOME, tokens ni internals de adaptador.
Decision 2026-05-26: cerrado por T102. Factory y governance usan frontera JSON
estricta con limite/trailing/content-type; MCP y web declaran perfiles legacy
compatibles donde aceptan campos extra o form; el transporte MCP real del
servidor comparte limite/trailing/content-type y error compacto.
Test futuro:
`go test -count=1 ./modulos/orquesta-factory-http ./modulos/orquesta-governance ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-http-gateway ./modulos/orquesta-web ./cmd/orquesta-server`.
Backlog: `T102 legacy-http-json-boundary-policy`.
```

```text
ID: RAIL-CAND-OUTBOUND-HTTP-RESPONSE-001
Origen: scanner backlog 2026-05-24 trigesimocuarta pasada.
Casos: conectores y clientes salientes decodifican o leen respuestas HTTP con
reglas divergentes: OPES y `domain-work-http` hacen `json.NewDecoder` directo
sobre `response.Body`; CLI/governance y function-contract usan `io.ReadAll`;
comandos de servidor leen cuerpos completos para status/diagnostico.
Decision 2026-05-26: conectores OPES/domain-work, CLI, web y comandos de
servidor aplican limite de respuesta por perfil local, status handling
redactado, `Content-Type` JSON o legacy `text/plain` solo si el body es JSON
parseable, trailing-token check y errores publicos compactos. T80 sigue
gobernando egress/host del HTTP neutral;
este rail queda cerrado para la frontera de respuesta saliente T103.
Revalidacion 2026-05-26: la bateria focal del write-set T103 sigue pasando y no
aparecio nueva superficie saliente fuera de ese alcance.
Revalidacion OrquestaV2 2026-05-26:
`agent-ref-task-autoprogramming-49b26a6e419d-g01` cubre explicitamente
`text/plain` legacy solo cuando el body es JSON parseable, sin abrir salida
cruda ni un rail paralelo frente a T138.
Test futuro:
`go test -count=1 ./modulos/orquesta-opes-connector ./modulos/orquesta-domain-work-http ./modulos/orquesta-cli ./modulos/orquesta-web ./modulos/orquesta-app-gateway ./cmd/orquesta-server`.
Backlog: `T103 outbound-http-response-limit-redaction`.
```

```text
ID: RAIL-CAND-FILE-STORE-DURABLE-WRITE-001
Origen: scanner backlog 2026-05-24 trigesimocuarta pasada.
Casos: stores file-based usan contratos de escritura distintos. `orquesta-run-file`
y `orquesta-domain-work-file` hacen temp unico, sync y sync de directorio;
`orquesta-server` y `orquesta-runtime-codex-delivery` usan `path+".tmp"` sin
fsync/dir sync; el ledger de artefactos crea directorio `0755` y no sincroniza
antes/despues de `rename`.
Decision pendiente: politica comun o documentada por modulo para temp unico,
permisos, lock/claim cuando haya multiproceso, fsync/close/rename/sync dir y
limpieza de temp interrumpido. Coordinar con T95, T57, T98 y T101 sin reemplazar
sus contratos especificos.
Test futuro:
`go test -count=1 ./modulos/orquesta-run-file ./modulos/orquesta-domain-work-file ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-server ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T104 file-store-durable-write-policy`.
Estado: cerrado 2026-05-27 con bateria obligatoria completa; control files
Codex quedan coordinados con `T159 codex-control-file-durable-write-policy`.
```

```text
ID: RAIL-CAND-PROCESS-RUNTIME-LAUNCH-IO-001
Origen: scanner backlog 2026-05-24 trigesimoquinta pasada.
Casos: `ProcessRuntimeConnectorV0` arranca `exec.Cmd` con `CommandPath`, `Args`,
`Env` y `WorkingDir` reales ya resueltos, y descarta stdout/stderr con
`io.Discard`. El contrato externo usa refs opacas (`executable_ref`,
`arg_refs`, `env_refs`, `working_dir_ref`), pero falta recibo redacted que
demuestre como se resolvieron y que politica de IO se aplico.
Decision implementada 2026-05-26: `ProcessRuntimeLaunchReceiptV0` registra
refs opacas de resolucion, policy/hash, causa publica y decision de IO
`io_discarded`; `ProcessRuntimeConnectorV0` aplica env allowlist con PATH
controlado y rechaza HOME, secretos, remotos Git, prompt/transcript y payloads
operativos. Registry/stats propagan solo refs compactas de process/session/
launch y receipt/policy, sin path, args, env, PID ni stdout/stderr crudos.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime ./modulos/orquesta-orchestration-core ./modulos/orquesta-agent-process-registry ./modulos/orquesta-agent-process-registry-memory ./cmd/orquesta-server`.
Backlog: `T105 process-runtime-launch-env-io-receipt`.
```

```text
ID: RAIL-CAND-CODEX-WAVE-SUMMARY-REDACTION-001
Origen: scanner backlog 2026-05-24 trigesimoquinta pasada.
Casos: `codex-wave`/`codex-director-wave` publican summaries con rutas reales de
runtime, registry, prompt, ACK, last message, stdout/stderr, HOME/CODE_HOME y
PID. Tail, purge y stop ya tienen tareas separadas, pero el launch/status puede
filtrar material local antes de que esas guardas apliquen.
Decision pendiente: summary publico por refs opacas, contadores y estados;
diagnostico crudo solo opt-in local con limite fuerte. No usar rutas ni logs
crudos como evidencia terminal.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-worktree ./modulos/orquesta-agent-process-registry`.
Backlog: `T106 codex-wave-public-summary-redaction`.
Estado 2026-05-26: cubierto; stdout de `codex-wave`,
`codex-wave-status`, `codex-wave-stop` y `codex-director-wave` queda separado de
la registry local mediante summaries publicos con refs opacas, contadores,
estados y errores redactados. Paths/PID/HOME/logs/control files siguen solo en
registry local para operaciones opt-in y no como evidencia publica.
```

```text
ID: RAIL-CAND-GO-SMOKE-DIAGNOSTIC-REDACTION-001
Origen: scanner backlog 2026-05-24 trigesimoquinta pasada.
Casos: smokes Go y tests opt-in reales usan `t.Fatalf` con body HTTP,
stdout/stderr, prompts o summaries completos. Si falla una ejecucion con Codex
real, OPES temporal o control plane, el log de test puede persistir rutas
privadas, payloads de dominio, prompts, transcripts o tokens simulados.
Decision aplicada 2026-05-26: los smokes Go reales Codex y el harness Go de
procesos usan helpers de diagnostico con limite, redaccion por campo y resumen
compacto; los dumps crudos quedan solo para directorios temporales opt-in de
operador y no cuentan como evidencia terminal. Coordinar con T21, T58, T61,
T91, T97 y T103 si aparece otra superficie fuera de Go tests/smokes.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-app-codex-stack ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-connector ./modulos/orquesta-server`.
Backlog: `T107 real-smoke-go-diagnostic-redaction`.
```

```text
ID: RAIL-CAND-DECISION-COUNCIL-ROUND-WIRING-001
Origen: scanner backlog 2026-05-24 trigesimosexta pasada.
Casos: `orquesta-decision-council` y `orquesta-director-candidates` construyen
planes de propuesta/critica/voto y candidatos schedulables, pero no hay
composicion que los ejecute como rondas vivas del Director con gates, waits y
aceptacion durable. El riesgo es duplicar deliberacion en prompts libres o
saltar directo a un agente sin quorum ni evidencia de votos.
Decision cerrada offline focal 2026-05-26: materializar rondas de consejo por
`WorkflowTaskV0` con gates/waits de cohorte/ola, deps causales
propuesta->critica->voto y roles preservados por `context_refs` estructuradas.
La aceptacion final se construye como `AcceptDecision` solo cuando
`DecisionCouncilVoteV0` alcanza quorum/evidencia y el `VoteRequested` durable
esta reflejado. No se mete proveedor, modelo, HOME, runtime ni familias reales
en el modulo puro.
Test futuro:
`go test -count=1 ./modulos/orquesta-decision-council ./modulos/orquesta-director-candidates ./modulos/orquesta-director-scheduler ./modulos/orquesta-director-tick-input ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-director-service ./modulos/orquesta-app-codex-stack`.
Backlog: `T108 decision-council-operational-rounds`.
```

```text
ID: RAIL-CAND-LEASE-PROGRESS-POLICY-OWNER-001
Origen: scanner backlog 2026-05-24 trigesimosexta pasada.
Casos: `orquesta-core-leases` define leases/timeouts puros, mientras
`orquesta-runtime` y `orquesta-runtime-codex-delivery` mantienen politica de
heartbeat/progreso separada y el scheduler recibe `lease_action_candidates`
como carril distinto. Sin puente, stalled/loop/stopped puede tener umbrales y
acciones divergentes.
Decision 2026-05-26: puente por puerto desde `AgentProgressReportV0` +
`AgentLeasePolicyV0` a `AgentTimeoutAssessmentV0`/`AgentLeaseExpired`, con
`observed_at` inyectado por adaptador, refs compactas y reuso cacheado de la
misma observacion de progreso para no remuestrear. No persistir PID, HOME,
rutas, stdout/stderr, prompts, transcripts, proveedor/modelo ni payloads de
runtime.
Test futuro:
`go test -count=1 ./modulos/orquesta-core-leases ./modulos/orquesta-runtime ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-director ./modulos/orquesta-director-scheduler ./modulos/orquesta-director-tick-input ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T109 agent-lease-progress-policy-bridge`.
```

```text
ID: RAIL-CAND-SCHEDULER-CYCLE-POLICY-001
Origen: scanner backlog 2026-05-24 trigesimoseptima pasada.
Casos: `DirectorSchedulerTickInputV0` filtra campos operativos con una lista
local que incluye `runtime`, `provider`, `model`, `db`, `sql`, `home`,
`filesystem` y `docker`; el ciclo de dispatch del Director conserva otra lista
local para compactar error codes que contiene `db`, `sql`, `provider`, `home`,
`prompt` y `token`.
Decision pendiente: usar politica comun por campo o wrapper local alineado con
`orquesta-rails`: vocabulario operativo opaco pasa; valores sensibles
efectivos, rutas privadas, prompts/transcripts crudos y payloads masivos
bloquean o se redactan.
Decision aplicada 2026-05-26: el scheduler valida sus campos operativos con
`orquesta-rails` por campo y el ciclo de dispatch valida el codigo crudo antes
de normalizarlo. `provider_timeout`, `db_adapter_unavailable` y
`runtime_backpressure` quedan como error codes publicos validos; `api_key=...`,
rutas HOME y prompts crudos degradan a error publico compacto.
Test futuro:
`go test -count=1 ./modulos/orquesta-director-scheduler ./modulos/orquesta-director ./modulos/orquesta-rails ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-codex-stack`.
Backlog: `T110 director-scheduler-cycle-rail-policy-sync`.
```

```text
ID: CONTEXT-CAND-LOCAL-AGENTS-DOC-001
Origen: scanner backlog 2026-05-24 trigesimoseptima pasada.
Casos: `AGENTS.md` locales de core workflow, core concurrency, core leases,
core replanner y director scheduler apuntan a
`../../docs/reinicio_orquesta_v2/protocolo_anti_bucles.md`, pero el directorio
`docs/reinicio_orquesta_v2` no existe en la foto actual del repo.
Decision resuelta 2026-05-26: las refs obligatorias de `AGENTS.md` del alcance
T111 ya no apuntan a `docs/reinicio_orquesta_v2`. La ruta antigua queda
historica/stale; los sustitutos vivos son `docs/estado_actual_2026-05-17.md`,
`docs/guia_nucleo_orquestacion_2026-05-17.md`, el backlog vivo y este rail. Si
el contexto requerido falta, el agente debe pedir `CONSULTA AL DIRECTOR` o
recibir bundle materializado, no inventar protocolo.
Test futuro:
`go test -count=1 ./modulos/orquesta-context ./modulos/orquesta-core-workflow ./modulos/orquesta-core-concurrency ./modulos/orquesta-core-leases ./modulos/orquesta-core-replanner ./modulos/orquesta-director-scheduler`.
Backlog: `T111 local-agents-required-doc-refs-sync`.
```

```text
ID: RAIL-CAND-RUN-QUEUE-WORKSET-001
Origen: scanner backlog 2026-05-24 trigesimoseptima pasada.
Casos: `orquesta-core-concurrency` calcula claims/read-write sets y el
scheduler gatea agentes dentro de una run, pero `orquesta-run-queue` solo rankea
candidatos por estado/prioridad/aging y declara bloqueos/leases/despacho fuera
de alcance. Dos runs de automejora con write-set solapado pueden llegar al
supervisor global sin gate comun previo.
Decision pendiente: transportar claims compactos en la cola o proyeccion
asociada y evaluar solapes contra runs vivos/reservados antes de launch.
Coordinar con T31 para leases de cola y con T43 para merge documental de
scanners; no leer Git ni filesystem real desde `orquesta-run-queue`.
Test futuro:
`go test -count=1 ./modulos/orquesta-run-queue ./modulos/orquesta-run-supervisor ./modulos/orquesta-run-memory ./modulos/orquesta-run-file ./modulos/orquesta-core-concurrency ./modulos/orquesta-director-scheduler ./modulos/orquesta-server ./cmd/orquesta-server`.
Backlog: `T112 run-queue-workset-concurrency-bridge`.
```

```text
ID: CONTEXT-CAND-MODULE-HISTORICAL-DOC-001
Origen: scanner backlog 2026-05-24 trigesimoctava pasada.
Casos: ademas de los `AGENTS.md` cubiertos por T111, docs locales de
`orquesta-core`, `orquesta-core-workflow`, `orquesta-capacity`,
`orquesta-governance` y `orquesta-observability` siguen apuntando a
`docs/reinicio_orquesta_v2/*`, arbol que no existe en la foto vigente.
Decision aplicada 2026-05-26: docs locales del alcance T113 dejan de tratar
esas rutas y el inventario DB v1 como prerequisito vivo de automejora. Las
fuentes vigentes son `AGENTS.md`, `docs/estado_actual_2026-05-17.md`,
`docs/guia_nucleo_orquestacion_2026-05-17.md` y los README/docs locales
existentes; cualquier DB v1 queda como evidencia forense historica.
Test futuro:
`go test -count=1 ./modulos/orquesta-context ./cmd/orquesta-server`.
Backlog: `T113 module-historical-doc-ref-sync`.
```

```text
ID: RAIL-CAND-RUN-QUEUE-FAIRNESS-001
Origen: scanner backlog 2026-05-24 trigesimoctava pasada.
Casos: `RunSchedulingCandidateV0` transporta `fairness_group_ref` y el ranking
tiene aging determinista, pero la politica actual solo usa aging como desempate
dentro de la misma prioridad. Sin owner de fairness por grupo, automejora idle,
smokes reales o trabajo humano pueden monopolizar la cola o quedar hambrientos.
Decision pendiente: definir politica de fairness por grupo/app con reloj
inyectado, reason codes publicos y coordinacion con lease de T31 y work-set de
T112. No elevar trabajos reales con efectos externos por encima de confirmaciones
opt-in.
Test futuro:
`go test -count=1 ./modulos/orquesta-run-queue ./modulos/orquesta-run-supervisor ./modulos/orquesta-run-memory ./modulos/orquesta-run-file ./modulos/orquesta-server ./cmd/orquesta-server`.
Backlog: `T114 run-queue-fairness-group-policy`.
Estado 2026-05-26: resuelto localmente. La cola expone politica con reloj
inyectado, ventanas, limite por grupo, boost y reason codes
`fairness_group_paused`, `fairness_group_boosted` y `fairness_group_missing`;
el supervisor propaga la politica sin sustituir T31/T112.
```

```text
ID: CONTEXT-CAND-WEB-INTAKE-SESSION-001
Origen: scanner backlog 2026-05-24 trigesimoctava pasada.
Casos: `modulos/orquesta-web/docs/tareas.md` mantiene `WEB-013` pendiente para
reemplazar formulario largo de nueva app por sesion conversacional de intake.
T76 cubre routing AppSpec hacia Director V2, pero no el contrato web de sesion
parcial, pregunta pendiente, AppSpec parcial, i18n y fallback fino.
Estado 2026-05-26: cerrado. `WebNuevaAppIntakeSessionV0` ya modela sesion
parcial, preguntas con claves i18n, decisiones, `AppSpecRequestV0` parcial y
handoff compacto con refs opacas hacia el Director; la web sigue como adaptador
sobre puertos/API de intake/director y conserva fallback documentado a
`SolicitarNuevaApp`.
Test futuro:
`go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-app-director-intake ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-http-gateway ./cmd/orquesta-server`.
Backlog: `T115 web-app-intake-session-contract`.
```

```text
ID: DOC-CAND-MODULE-TASK-INTEGRITY-001
Origen: scanner backlog 2026-05-24 trigesimonovena pasada.
Casos: `modulos/orquesta-app-codex-stack/docs/tareas.md` declara dos secciones
`APP-CODEX-STACK-012`, y `modulos/orquesta-cli/docs/pruebas.md` mantiene casos
`CLI-P001`, `CLI-P007` y `CLI-P008` como pendientes aunque existen tests locales
para ayuda sin red, estado de servidor sin fallback local e inventario V1 en
cuarentena.
Decision pendiente: crear linter/indice que detecte IDs duplicados y estados
stale en docs locales antes de que el planner cierre o relance trabajo por
heading ambiguo.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-context ./modulos/orquesta-cli ./modulos/orquesta-app-codex-stack`.
Backlog: `T116 module-task-doc-integrity-linter`.
```

```text
ID: RAIL-CAND-APP-CODEX-REVIEW-GATE-001
Origen: scanner backlog 2026-05-24 trigesimonovena pasada.
Casos: la documentacion local de `orquesta-app-codex-stack` dejaba como
pendiente separar una politica productiva de rechazo/replanificacion por
entregas invalidas. Coexisten review gate, ACK validator, snapshot/worktree,
prompt de 300 lineas y reglas de tests/write-set como fuentes cercanas, pero no
deben competir como owners de aceptacion/rechazo.
Decision 2026-05-26: el owner ejecutable es `codexStackReviewGatePolicyV0`; el
review gate consume evidencia de worktree y emite issues estructurados, el ACK
valida correlacion/recibos y el snapshot aporta hechos sin decidir severidad.
La politica tolera aliases, rutas hijas y globs seguros, y excluye perfiles
documentales/no-Go del presupuesto Go estricto salvo senal Go explicita.
Test futuro:
`go test -count=1 ./modulos/orquesta-app-codex-stack ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-worktree ./modulos/orquesta-app-director-service`.
Backlog: `T117 app-codex-review-gate-policy-owner`.
Estado 2026-05-26: cubierto. `orquesta-app-codex-stack` tiene owner por puerto
`ReviewGateDeliveryPolicyPortV0`; `orquesta-runtime-worktree` conserva la
evidencia snapshot/write-set y el presupuesto Go, y los perfiles documentales o
no-Go declarados no heredan automaticamente el rail Go.
Revalidacion OrquestaV2 retry 2026-05-26: se conserva `codexStackReviewGatePolicyV0`
como owner efectivo; prompt, ACK y snapshot quedan como transporte/evidencia y
no como decision terminal.
```

```text
ID: CONTEXT-CAND-LEGACY-GENERATED-DOCS-001
Origen: scanner backlog 2026-05-24 trigesimonovena pasada.
Casos: `docs/plan_microtareas.md` y decisiones de
`orquesta-app-director-intake` nombran artefactos de app generada como
`docs/manual_desarrollador.md`, `docs/manual_sistemas_deploy.md`,
`docs/pruebas_documentales.md` y `docs/pendientes.md`, que no existen como docs
vivos del repo Orquesta.
Decision 2026-05-26: planes marcados como historicos/debug y nombres
convertidos en contrato de artefactos del proyecto generado; no crear archivos
raiz vacios ni tratar esos `test -s` como pruebas obligatorias del nucleo.
Test futuro:
`go test -count=1 ./modulos/orquesta-app-director-intake ./modulos/orquesta-context ./cmd/orquesta-server`.
Backlog: `T118 legacy-generated-doc-artifact-contract-sync`.
```

```text
ID: CONTEXT-CAND-SERVER-FIRST-USAGE-DOC-001
Origen: scanner backlog 2026-05-24 cuadragesima pasada.
Casos: `docs/uso_actual_app_orquesta.md` describe comandos y rutas legacy como
`./orquesta serve`, `/api/status`, `/api/agentes`, `/api/runtime-transcript`,
OpenClaw y AP-077 mientras la composicion vigente usa servidor actual y rutas
`/api/v0/*`.
Decision pendiente: clasificar ese manual como historico o sincronizarlo con la
foto server-first vigente; no dejar que el planner genere codigo contra API V1
ni reabra control-plane legacy.
Test futuro:
`go test -count=1 ./modulos/orquesta-cli ./modulos/orquesta-web ./cmd/orquesta-server`.
Backlog: `T119 server-first-usage-doc-route-sync`.
```

```text
ID: CONTEXT-CAND-LEGACY-SQLITE-FORENSIC-DOC-001
Origen: scanner backlog 2026-05-24 cuadragesima pasada.
Casos: docs locales de governance, core y capacity conservan referencias a
snapshots SQLite/DBV1 y comandos `sqlite3` como evidencia forense. Algunas
entradas usan rutas absolutas historicas y pueden parecer prerequisito vivo si
un indice federado no respeta cuarentena/freshness.
Cierre T120 2026-05-26: las refs quedan marcadas como forenses/historicas en
cuarentena, las rutas absolutas dejan de ser fuentes vivas y DBV1 o `sqlite3`
no pueden programarse como trabajo operativo sin decision explicita del
director.
Test futuro:
`go test -count=1 ./modulos/orquesta-governance ./modulos/orquesta-core ./modulos/orquesta-capacity ./modulos/orquesta-context`.
Backlog: `T120 legacy-sqlite-forensic-doc-quarantine`.
```

```text
ID: OPS-CAND-MANUAL-AGENT-COMPAT-001
Origen: scanner backlog 2026-05-24 cuadragesima pasada.
Casos: `docs/operacion_agentes_manuales.md` y el manual de uso mantienen
wrappers `scripts/inicio_agente.sh`, Terminator y sesiones manuales como capa de
compatibilidad, pero no hay contrato/linter que pruebe que siguen subordinados
al daemon y no mutan estado por fuera del control plane.
Decision 2026-05-26: T121 queda cerrado en su write-set. Los scripts manuales
se catalogan como recuperacion asistida, exigen servidor residente/CLI publica
o bloquean con error publico, no mutan stores/worktrees/runtime por fuera del
control plane y documentan correlacion por `run_ref`, `task_ref`, `agent_ref`,
`external_session_id` y `worktree_ref` opacas sin convertir tmux/Terminator en
contrato del nucleo.
Test:
`bash -n scripts/inicio_agente.sh scripts/cargar_agentes.sh scripts/terminator_agentes.sh scripts/agente_console.sh` y `go test -count=1 ./modulos/orquesta-cli ./modulos/orquesta-server ./cmd/orquesta-server`.
Backlog: `T121 manual-agent-ops-compatibility-contract`.
```

```text
ID: CONTEXT-CAND-CANONICAL-DOC-PRECEDENCE-001
Origen: scanner backlog 2026-05-24 cuadragesima primera pasada.
Casos: `docs/BIBLIA_APP_ORQUESTA.md` se declara doctrina canonica y
`docs/00_INDICE.md` la lista como `M00`, aunque `estado_actual` ya la trata
como historica/stale y `docs/README.md` conserva el marco "Orquesta v1".
Decision 2026-05-26: T122 marca `docs/BIBLIA_APP_ORQUESTA.md` desde su
cabecera como `doc_estado=historico-stale`, fija sustitutos vigentes y actualiza
indice/README para separar fuentes vigentes, historicas, forenses y plantillas.
Refuerzo retry 2026-05-26: tambien se reetiquetan las secciones internas V1 que
decian "fuentes de verdad" o "doctrina" para que un scanner por cuerpo no las
promueva sobre la cabecera stale.
OpenClaw, SQLite, rutas `/api/*` antiguas o control-plane V1 nombrados en docs
historicos no se leen como foto vigente sin enlace a `estado_actual`,
`guia_nucleo` o backlog vivo.
Test futuro:
`go test -count=1 ./modulos/orquesta-context ./cmd/orquesta-server`.
Backlog: `T122 canonical-doc-precedence-self-sync`.
```

```text
ID: CONTEXT-CAND-FACTORY-BACKLOG-PREVIEW-001
Origen: scanner backlog 2026-05-24 cuadragesima primera pasada.
Casos: `/api/v0/apps/spec` devuelve `BacklogInicialPropuestoV0` como `backlog`;
la web lo renderiza como preview, pero el contrato HTTP no transporta estado
`preview/no_ejecutable`, freshness ni handoff a Director V2.
Decision pendiente: separar backlog determinista inicial de plan operativo y
exigir refs/estado cuando se convierta en trabajo real.
Estado: cerrado 2026-05-26. `BacklogInicialPropuestoV0` ya transporta
`preview_no_ejecutable`, freshness y `director_handoff` a
`orquesta.apps.arrancar_director.v0`; HTTP, web y MCP lo proyectan como preview
compacta/insumo, no como cola ejecutable.
Test futuro:
`go test -count=1 ./modulos/orquesta-factory ./modulos/orquesta-factory-http ./modulos/orquesta-web ./modulos/orquesta-app-director-intake ./modulos/orquesta-mcp ./cmd/orquesta-server`.
Backlog: `T123 factory-backlog-preview-director-handoff`.
```

```text
ID: CONTEXT-CAND-LEGACY-EXTERNAL-ORCHESTRATOR-DOC-001
Origen: scanner backlog 2026-05-24 cuadragesima primera pasada.
Casos: `BIBLIA_APP_ORQUESTA.md`, `op_088_orquesta_servidor_mcp.md`,
`runtime_worker_contract.md` y OPs cercanas presentan OpenClaw, `tmux`,
`/api/mcp` o servidor MCP como superficies canonicas, aunque la foto vigente
los deja como adaptadores/composiciones opt-in o historia.
Decision pendiente: cuarentenar esos docs para que no alimenten planificacion
automatica ni creen endpoints/runtimes no cableados.
Resolucion 2026-05-26: T124 marca esos documentos como
`legacy_external_orchestrator_doc_quarantine`, conserva OpenClaw/`tmux`/MCP real
solo como historia, rescate o composicion opt-in, y enlaza la superficie viva a
`modulos/orquesta-mcp`, `modulos/orquesta-runtime`,
`modulos/orquesta-runtime-codex` y `/mcp` en `cmd/orquesta-server`.
Test futuro:
`go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-runtime ./modulos/orquesta-runtime-codex ./cmd/orquesta-server`.
Backlog: `T124 legacy-external-orchestrator-doc-quarantine`.
```

```text
ID: CONTEXT-CAND-WORK-PROFILES-EMPTY-MODULE-001
Origen: scanner backlog 2026-05-24 cuadragesima segunda pasada.
Casos: `modulos/orquesta-work-profiles` existe solo como directorio vacio con
`docs/`, mientras `WorkProfileV0` vive en `orquesta-core-workflow` y una
decision local ya descarta crear `orquesta-work-profiles` como owner. Un indice
federado puede tratar ese path como modulo pendiente y duplicar el contrato de
perfiles.
Decision pendiente: marcar el directorio como historico/placeholder no
ejecutable o darle owner explicito; el planner no debe abrir tareas desde
directorios sin contrato efectivo.
Test futuro:
`go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-planner ./modulos/orquesta-context ./cmd/orquesta-server`.
Backlog: `T125 work-profiles-empty-module-quarantine`.
Estado: cubierto 2026-05-26; `modulos/orquesta-work-profiles` queda placeholder
historico no ejecutable con `README.md`/`AGENTS.md`, fuente vigente en
`orquesta-core-workflow`/`orquesta-orchestration-core` y test focal
`TestIdleSelfImprovementFederatedBacklogV0IgnoraDirectoriosSinContratoEfectivoV0`.
```

```text
ID: OPS-CAND-MODULE-CODEX-LAUNCHERS-001
Origen: scanner backlog 2026-05-24 cuadragesima segunda pasada.
Casos: muchos `modulos/*/arrancar_codex.sh` delegan en
`../_comun/arrancar_codex_modulo.sh`, pero `modulos/_comun` no existe en la
foto revisada. Algunas READMEs locales aun recomiendan esos wrappers como
arranque manual, lo que puede saltarse daemon, cola, write-set, ACK y shutdown
gobernado.
Decision pendiente: inventariar wrappers, catalogar compatibilidad vs ruta
vigente y exigir helper comun/proof `bash -n` o marcar bloqueo publico. No
crear una via paralela de runtime Codex fuera de OrquestaV2.
Test futuro:
`bash -n $(find modulos -name arrancar_codex.sh | sort)` y `go test -count=1 ./modulos/orquesta-cli ./modulos/orquesta-server ./cmd/orquesta-server`.
Backlog: `T126 module-local-codex-launcher-compatibility-contract`.
Estado: cubierto 2026-05-26; wrappers con helper ausente devuelven error
publico, runtime-codex exige opt-in manual y READMEs dejan de presentar wrappers
locales como ruta vigente de agentes OrquestaV2.
Revalidacion 2026-05-27:
`agent-ref-task-autoprogramming-089a03f67381-g01` confirma que no hay wrappers
versionados apuntando al helper ausente, `modulos/_comun` sigue sin crearse y
la matriz obligatoria pasa con `bash -n $(find modulos -name arrancar_codex.sh | sort)`
y `go test -count=1 ./modulos/orquesta-cli ./modulos/orquesta-server ./cmd/orquesta-server`.
```

```text
ID: RAIL-CAND-IGNORED-LOCAL-ARTIFACTS-001
Origen: scanner backlog 2026-05-24 cuadragesima segunda pasada.
Casos: `.gitignore` excluye artefactos locales de control/diagnostico como
`.ssl-key.log`, `.orquesta-runtime/`, `.orquesta-server` y
`.orquesta-smoke-work`, y algunos existen en la raiz. Si snapshots, contexto,
AppVCS, promocion o ACK terminal caminan el filesystem sin politica comun,
pueden incluir material local ignorado como si fuera producto.
Decision 2026-05-26: politica ejecutable de exclusion por categoria con recibo
compacto; `.gitignore` ayuda pero no sustituye validacion por puerto ni
redaccion. Artefactos locales ignorados no entran en `ACK.files`, contexto,
commits ni evidencias de cierre.
Decision 2026-05-26: cerrado localmente por politica ejecutable en snapshots,
contexto, AppVCS, promocion y ACK terminal; retry sincroniza `.gitignore` con
runtime/control dirs locales sin convertir ignore en fuente unica de verdad.
Refuerzo puntual 2026-05-26: `.orquesta-logs`, `orquesta.env`, `orquesta.db*`
y `.orquesta-inbox.md` quedan como artefactos locales no exportables.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime-worktree ./modulos/orquesta-context ./modulos/orquesta-app-codex-stack ./modulos/orquesta-runtime-codex ./cmd/orquesta-server`.
Backlog: `T127 ignored-local-artifact-exclusion-policy` cerrado.
```

```text
ID: RAIL-CAND-AGENT-PROGRESS-SUPERVISOR-001
Origen: scanner backlog 2026-05-24 cuadragesima tercera pasada.
Casos: `modulos/orquesta-director/agent_progress_supervisor_helpers_v0.go`
mantiene una lista local de fragmentos prohibidos que incluye vocabulario
operacional como `provider`, `model`, `db`, `sql`, `home` y `runtime`.
Evidencias opacas o refs causales pueden desaparecer antes de que el Director
evalua progreso, aunque no expongan secretos ni payloads crudos.
Decision 2026-05-26: el supervisor delega en la politica comun por campo de
`orquesta-rails`. Refs opacas con vocabulario operacional (`runtime`,
`provider`, `model`, `db`, `sql`, `home`) pasan; se bloquean secretos efectivos,
rutas locales reales, prompts/transcripts crudos y payloads masivos.
Test futuro:
`go test -count=1 ./modulos/orquesta-director ./modulos/orquesta-runtime ./modulos/orquesta-core-leases ./modulos/orquesta-rails ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-codex-stack`.
Backlog: `T128 agent-progress-supervisor-rail-policy-sync`.
```

```text
ID: OPS-CAND-DAEMON-START-ENV-001
Origen: scanner backlog 2026-05-24 cuadragesima tercera pasada.
Casos: `cmd/orquesta-server/daemon.go` arranca el proceso residente con un
entorno derivado de `os.Environ()` y defaults de detail rails. Falta recibo de
categorias permitidas/redactadas y contrato publico para estado, diagnostico y
errores de variables requeridas.
Decision 2026-05-26: `start` proyecta entorno por allowlist en
`cmd/orquesta-server/daemon_start_env_policy_v0.go`; no hereda todo
`os.Environ()`. El recibo publico queda en `effective_config` como perfil,
categorias, conteos por origen e issues accionables, sin valores crudos. La
proyeccion Codex, bootstrap, logs daemon y runtime siguen como categorias
separadas.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack ./modulos/orquesta-observability`.
Backlog: `T129 server-daemon-start-env-policy` cerrado 2026-05-26.
```

```text
ID: RAIL-CAND-ARCHITECTURE-TEST-GUARDS-001
Origen: scanner backlog 2026-05-24 cuadragesima tercera pasada.
Casos: varias pruebas de arquitectura combinan guards de imports/efectos con
listas de terminos por substring como `runtime`, `http`, `mcp` o `sql`. Ese
patron puede bloquear comentarios, refs opacas o nombres de politica validos, y
duplica railes de redaccion/neutralidad fuera de un helper comun.
Decision pendiente: conservar guards estrictos de imports/efectos concretos y
acotar listas textuales por campo o matriz de falsos positivos.
Test futuro:
`go test -count=1 ./modulos/orquesta-run-control ./modulos/orquesta-run-queue ./modulos/orquesta-run-memory ./modulos/orquesta-director-candidates ./modulos/orquesta-app-director-intake ./modulos/orquesta-app-runner ./modulos/orquesta-director-agent-workflow ./modulos/orquesta-app-director-service ./modulos/orquesta-rails`.
Backlog: `T130 architecture-guard-test-rail-policy-owner`.
Estado 2026-05-26: cubierto; `orquesta-rails` aporta helper comun para imports
concretos y literales sensibles, y los tests del alcance dejaron de escanear
vocabulario global en comentarios o refs opacas.
```

```text
ID: RAIL-CAND-DOC-LOCAL-PATH-001
Origen: scanner backlog 2026-05-24 cuadragesima cuarta pasada.
Casos: docs operativos de auditoria y estado pueden incluir rutas absolutas
locales, state dirs o ficheros de control como ejemplo de diagnostico. Si el
contexto materializado o el ACK los transporta sin clasificacion, pasan a
parecer evidencia de producto o write-set valido.
Decision pendiente: permitir variables y refs opacas, pero bloquear o marcar
como no exportables HOME, state dirs, `.orquesta-runtime`, prompts, transcripts,
logs locales y ficheros de control en contexto/ACK/snapshots.
Test futuro:
`go test -count=1 ./modulos/orquesta-context ./modulos/orquesta-runtime-worktree ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T131 documentation-local-path-redaction-linter`.

Cierre T131 2026-05-26: contexto materializado, ACK terminal y linter
documental ya distinguen refs/variables opacas de rutas locales o ficheros de
control usados como evidencia publica. Los reportes del linter citan documento
y linea, no el valor local.
```

```text
ID: RAIL-CAND-OBSERVABILITY-PRIVACY-001
Origen: scanner backlog 2026-05-24 cuadragesima cuarta pasada.
Casos: `modulos/orquesta-observability` usa listas propias para claves y
fragmentos sensibles, separadas de `orquesta-rails` y de la auditoria JSONL.
Ese rail puede divergir: bloquear refs opacas como `token_policy_ref` o dejar
pasar payloads crudos si otro canal usa una lista distinta.
Decision 2026-05-26: `orquesta-rails` es owner de taxonomia/redaccion visible;
`orquesta-observability` valida contra esa politica y conserva DTOs/eventos.
Refs opacas `*_policy_ref`, `*_redaction_ref` y equivalentes no se clasifican
como valores crudos. API/MCP/web consumen `privacy.redaction_level` verificable.
Test futuro:
`go test -count=1 ./modulos/orquesta-observability ./modulos/orquesta-rails ./modulos/orquesta-server ./modulos/orquesta-mcp ./modulos/orquesta-web ./cmd/orquesta-server`.
Backlog: `T132 observability-privacy-taxonomy-rail-sync`.
```

```text
ID: DOC-CAND-MODULE-AGENTS-COVERAGE-001
Origen: scanner backlog 2026-05-24 cuadragesima cuarta pasada.
Casos: modulos de frontera sensible como `orquesta-rails`,
`orquesta-domain-work-http` y `orquesta-factory-http` no tienen guia local
completa para agentes. El fallo no es de runtime, pero si de rail documental:
un agente puede tocar HTTP/rails/factory con solo reglas raiz y sin owner local.
Decision pendiente: exigir `AGENTS.md` local o fuente sustituta explicita para
modulos sensibles antes de que el planner prepare runs automaticas.
Test futuro:
`go test -count=1 ./modulos/orquesta-rails ./modulos/orquesta-domain-work-http ./modulos/orquesta-factory-http ./modulos/orquesta-context ./cmd/orquesta-server`.
Backlog: `T133 module-boundary-local-agent-doc-coverage`.
Estado 2026-05-26: cubierto para el alcance T133. Los modulos sensibles tienen
`AGENTS.md` local y el servidor incluye el linter/test documental
`module_boundary_local_agent_doc_missing`, integrado en la revision de
integridad local del planner antes de preparar runs automaticas.
Revalidacion OrquestaV2 retry 2026-05-26: el paquete
`agent-ref-task-autoprogramming-e347186127f9-g01` no abre una excepcion nueva;
mantiene como rail observable que cualquier modulo sensible sin guia local o
fuente sustituta suficiente produzca `module_boundary_local_agent_doc_missing`.
```

```text
ID: OPS-CAND-SERVER-STATUS-LEGACY-ALIAS-001
Origen: scanner backlog 2026-05-24 cuadragesima quinta pasada.
Casos: `cmd/orquesta-server/commands.go` consultaba `/api/status` para
`status` y espera de apagado, mientras `modulos/orquesta-server/handler_v0.go`
aceptaba tambien `/api/v0/server/status`. La ruta versionada ya era la
superficie publica normal, pero el alias legacy seguia vivo sin contrato de
deprecacion.
Decision 2026-05-26 aplicada localmente: canonizar
`/api/v0/server/status`; dejar `/api/status` solo como alias legacy con headers
`Deprecation`, `Link` canonical, `Warning` y `X-Orquesta-Status-*`, sirviendo
el mismo DTO publico redactado que la ruta versionada. Cierre integrado
revalidado en retry OrquestaV2 2026-05-26 con el test obligatorio pasado.
Retry 2026-05-27 explicita `X-Orquesta-Status-Owner=orquesta-server` y
`X-Orquesta-Status-Sunset-Policy=no_new_use` para que clientes, docs y planners
no promuevan el alias como contrato vigente.
Coordinar con T119/T85/T87 sin duplicar freshness, redaccion ni readiness.
Test:
`go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-cli ./modulos/orquesta-web ./cmd/orquesta-server`.
Backlog: `T134 server-status-legacy-alias-sunset`.
```

```text
ID: AUTOPROG-CAND-PLANNER-FALLBACK-SCOPE-001
Origen: scanner backlog 2026-05-24 cuadragesima quinta pasada.
Casos: si el planner de backlog no puede leer el documento o no detecta tareas
pendientes, puede devolver una request fallback basada en la configuracion
generica de automejora y heredar write-set historico de servidor. Un fallo de
scanner documental queda asi convertido en tarea de codigo amplia.
Decision pendiente: el fallback debe bloquear, pedir revision o emitir scanner
acotado a docs salvo opt-in explicito con write-set, pruebas y causa publica.
`planner_empty`, errores de lectura y ACKs ambiguos deben verse en status.
Test futuro:
`go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-autoprogramming ./cmd/orquesta-server`.
Backlog: `T135 idle-self-improvement-planner-fallback-safety`.

Estado 2026-05-26: cerrado para el patron observado. El fallback degradado ya no
vuelve a la request base generica de codigo: con planner sin documento emite
scanner documental acotado, con `capacity_free` y trabajo conocido devuelve
planner empty con evidencia publica, y un error del puerto planner queda
auditado sin preparar automejora.
```

```text
ID: OPS-CAND-RESIDENT-WAIT-BUDGET-001
Origen: scanner backlog 2026-05-24 cuadragesima quinta pasada.
Casos: el supervisor residente recorta `MaxExternalWaits` a 1, mientras otros
flujos del Director admiten presupuestos mayores para Codex real, OPES temporal
u operador. El residente puede tratar una espera externa viva como idle o
capacidad libre antes de que lleguen ACK/delivery causales.
Decision pendiente: definir politica por modo, publicar el valor efectivo y
evitar clamps silenciosos; una run en `wait_external` con refs/lease/ACK
pendiente debe bloquear nueva automejora idle hasta quedar quiescent.
Test futuro:
`go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-run-supervisor ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T136 resident-drain-external-wait-budget-policy`.
Estado 2026-05-26: implementado en la composicion residente. El valor efectivo
de `MaxExternalWaits` queda publicado en configuracion efectiva, el modo
residente bloquea overrides altos con error publico recuperable y una run
observada en `wait_external`/`candidate_pending` bloquea nueva automejora
idle/capacity hasta quedar quiescent.
```

```text
ID: HTTP-CAND-REQUEST-BODY-BOUNDS-001
Origen: scanner backlog 2026-05-24 cuadragesima sexta pasada.
Casos: muchas rutas HTTP publicas de `orquesta-mcp` y `orquesta-web` usan
`json.NewDecoder(r.Body).Decode` directo, mientras `/mcp` si limita JSON-RPC con
`LimitReader`. La politica de limite, content-type, trailing data y campos
desconocidos queda duplicada o ausente por handler.
Decision cerrada 2026-05-26: helper comun de lectura JSON HTTP por frontera
publica, limites por perfil, errores compactos y rechazo de cuerpo crudo en
respuestas de error. La cobertura focal valida tambien que status/supervise de
autoprogramacion usan el perfil de autoprogramacion, no el limite de control.
Test futuro:
`go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-web ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway ./cmd/orquesta-server`.
Backlog: `T137 public-http-request-body-bounds`.
```

```text
ID: HTTP-CAND-REQUEST-BODY-BOUNDS-002
Fecha: 2026-05-26
Sintoma: `cmd/orquesta-server` mantenia decodificadores separados para `/mcp` y workspace timeline, con limites distintos sin owner local compartido.
Campo: body JSON publico de composicion servidor.
Payload minimo: POST workspace timeline con JSON mayor que control-plane y POST `/mcp` JSON-RPC dentro de su limite compatible.
Decision: compartir helper local de lectura JSON publica en `cmd/orquesta-server`, manteniendo perfiles separados para control-plane y MCP JSON-RPC.
Test: `TestBuildServerAppHandlerV0WorkspaceTimelineAPIAcotaBodyComoControlPlaneV0` y `TestMCPRealTransportV0RechazaBodyTooLargeYTrailingV0`.
Estado: cubierto
```

```text
ID: HTTP-CAND-RESPONSE-BODY-BOUNDS-001
Origen: scanner backlog 2026-05-24 cuadragesima sexta pasada.
Casos: comandos y clientes como `run-status`, `status`, web/MCP y conectores
HTTP leen o decodifican respuestas completas sin limite comun; errores HTTP
pueden incluir bodies grandes o material sensible del peer.
Decision pendiente: helper de respuesta con limite, descarte seguro, resumen
redactado y error publico estable; no propagar HTML, transcripts, rutas locales,
tokens ni payloads de dominio en stderr/stdout/auditoria.
Estado 2026-05-26: cubierto en el alcance T138. Los clientes/comandos usan
helpers de lectura acotada y errores publicos con status/correlation o codigo
estable, sin body crudo; gateway no anade cliente HTTP propio.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-web ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-domain-work-http ./modulos/orquesta-opes-connector`.
Backlog: `T138 outbound-http-response-bounds-redaction`.
```

```text
ID: OPS-CAND-COMMAND-OUTPUT-PUBLIC-SHAPE-001
Origen: scanner backlog 2026-05-24 cuadragesima sexta pasada.
Casos: `status`, `run-status`, `opes-drain-once`, `mcp-real-smoke` y comandos
`codex-wave-*` escriben bodies o summaries propios en stdout. Algunos incluyen
base URLs, state fallback, diagnostico local o datos de runtime que no tienen
shape/redaccion publica unica.
Decision cerrada 2026-05-26: DTO publico versionado por comando con freshness,
`redaction_level`, refs/categorias para URLs y diagnostico crudo solo opt-in
local. `status` no imprime statefile crudo como fallback; `run-status` conserva
stats publicos bajo envelope; `opes-drain-once` mantiene shape top-level para
smokes con URLs como refs; `codex-wave-*` y tail publican fuente canonica no
terminal. ACK/delivery/cierre siguen dependiendo de ACK estructurado, eventos,
evidencias y stores causales, no de stdout.
Test cerrado:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-observability ./modulos/orquesta-rails ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack`.
Backlog: `T139 command-output-public-shape-contract`.
```

```text
ID: DOC-CAND-BACKLOG-OVERLAP-001
Origen: scanner backlog 2026-05-24 cuadragesima septima pasada.
Casos: el backlog contiene pares con solape material ya escrito:
`T102 legacy-http-json-boundary-policy` y
`T137 public-http-request-body-bounds`; `T103 outbound-http-response-limit-redaction`
y `T138 outbound-http-response-bounds-redaction`. Sin canon/alias, el planner
puede lanzar dos tareas equivalentes, dividir criterios o cerrar una mientras
la otra sigue como pendiente.
Decision 2026-05-26: implementado offline focal en T140. T137 declara
`Fusionada_con: T102 legacy-http-json-boundary-policy` y T138 declara
`Fusionada_con: T103 outbound-http-response-limit-redaction`; T102/T103
conservan el canon. El planner residente lee esos aliases, fusiona criterios,
tests y `write_set` no redundantes en la tarea canonica, omite la duplicada con
collision `duplicate_backlog_task` y bloquea la canonica si una request viva del
alias ya esta visible. Cierre completo pendiente hasta que la bateria obligatoria
quede verde; el bloqueo observado fue el smoke OPES fake `run-until-assemble`.
Test futuro:
`go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-server ./cmd/orquesta-server`.
Backlog: `T140 backlog-overlap-canonicalization`.
```

```text
ID: OPS-CAND-NONTEST-PANIC-001
Origen: scanner backlog 2026-05-24 cuadragesima septima pasada.
Casos: `rg` encontro `panic(` en codigo no-test:
`cmd/orquesta-server/mcp_real_smoke_v0.go:mustMarshalMCPRealSmokeV0` y
`modulos/orquesta-core-leases/lease_evaluator_validation_v0.go:mustParseAgentLeaseInstantV0`.
Aunque se llamen tras invariantes previas, una frontera publica, smoke opt-in o
validador neutral no debe tumbar el proceso por marshal/parse/invariante rota.
Cierre 2026-05-26: T141 sustituyo esos helpers por `mcpMarshalMCPRealSmokeV0`
con error publico `internal_invariant:mcp_smoke_arguments_json` y
`parseAgentLeaseInstantV0` con `AgentLeaseValidationErrorV0`; quedan solo
`panic(` en tests dentro de este write-set. El recover del supervisor residente
sigue registrado como fallo operacional observable por state/auditoria, no como
sustituto de validacion.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-core-leases ./modulos/orquesta-server ./modulos/orquesta-observability`.
Backlog: `T141 public-boundary-no-panic-contract`.
```

```text
ID: OPS-CAND-CLI-INPUT-BOUNDS-001
Origen: scanner backlog 2026-05-24 cuadragesima octava pasada.
Casos: `modulos/orquesta-cli/command_flags_v0.go` lee `stdin` con
`io.ReadAll` y `--input` con `os.ReadFile` antes de aplicar limite comun o
clasificar si el origen es operador local, agente/toolbelt, path sensible o
fichero de control.
Decision pendiente: acotar bytes de entrada por comando, devolver error publico
sin body crudo y bloquear o marcar como opt-in local rutas absolutas, HOME,
`.orquesta-runtime`, prompts, transcripts, logs y ficheros de control.
Test futuro:
`go test -count=1 ./modulos/orquesta-cli ./cmd/orquesta-server`.
Backlog: `T142 cli-json-input-bounds-and-source-policy`.
Estado: cubierto 2026-05-26 por limite `--input-max-bytes`, clasificacion
`file_explicit` y bloqueo compacto de rutas sensibles.
```

```text
ID: RUNTIME-CAND-CODEX-CONTROL-FILE-BOUNDS-001
Origen: scanner backlog 2026-05-24 cuadragesima octava pasada.
Casos: `agent_ack.json`, `agent_shutdown_checkpoint_ack.json` y lecturas
compatibles de ACK/sidecars se leen completas con `os.ReadFile` en validadores,
planner u observation source. La validacion estructural existe, pero no hay
limite comun ni reason code especifico de sobrelimite.
Decision pendiente: helper de lectura acotada para ficheros de control Codex,
error publico compacto, sin path local ni contenido crudo, y reutilizacion desde
ACK, decisiones, checkpoint y planner.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T143 codex-control-file-size-and-redaction-policy`.
Estado: cubierto local 2026-05-26. La lectura de ficheros de control Codex usa
helper acotado comun con issue compacto `control_file_too_large` para ACK,
checkpoint, sidecar de decisiones y startup strict; logs/progreso quedan como
tail separado.
```

```text
ID: DOMAIN-CAND-ARTIFACT-INTAKE-BOUNDS-001
Origen: scanner backlog 2026-05-24 cuadragesima novena pasada.
Casos: `readDomainWorkDeliveryBodyV0` abre el primer fichero de `ACK.files` con
`os.ReadFile` y lo transforma en payload `domain_work`. La ruta se valida contra
`ProjectWorkDir`, pero falta limite por artifact type, deteccion de binario y
redaccion antes de `PayloadFields`.
Decision cerrada 2026-05-26: helper de intake por artefacto con limite, reason
code publico y bloqueo por campo antes de `PayloadFields`; refs opacas y
markdown/JSON valido pasan, pero HOME, rutas privadas, prompts/transcripts,
tokens, payloads HTTP crudos, binarios no declarados y cuerpos enormes no salen
al conector de dominio.
Test futuro:
`go test -count=1 ./modulos/orquesta-app-codex-stack ./modulos/orquesta-domain-work ./modulos/orquesta-runtime-codex-delivery`.
Backlog: `T144 domain-work-delivery-artifact-intake-policy`.
```

```text
ID: OPERATOR-CAND-MCP-CLIENT-DEADLINE-001
Origen: scanner backlog 2026-05-24 cuadragesima novena pasada.
Casos: `OperatorMCPClientConnectorV0.callToolV0` invoca `CallToolV0` con
`context.Background()`. Un conector externo colgado puede bloquear consulta,
burst, outbox o directed query sin timeout publico ni presupuesto visible.
Decision resuelta 2026-05-26 en T145: contexto padre opcional y timeout de
composicion por llamada; deadline por defecto si no se configura. Timeout y
cancelacion salen como `operator_mcp_timeout` y `operator_mcp_cancelled`;
ausencia de conector y fallo opaco conservan sus codigos publicos previos.
Test futuro:
`go test -count=1 ./modulos/orquesta-operator-mcp-client ./modulos/orquesta-operator-mcp ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T145 operator-mcp-client-deadline-budget`.
```

```text
ID: HTTP-CAND-FACTORY-JSON-BOUNDARY-001
Origen: scanner backlog 2026-05-24 cuadragesima novena pasada.
Casos: `modulos/orquesta-factory-http/appspec_http_v0.go` usaba lectura local de
`POST /api/v0/apps/spec` y quedaba fuera del alcance explicito de T137, que
lista MCP/web/gateway. Tambien coincidia con T133 porque `factory-http` no tenia
guia local completa.
Decision 2026-05-26: cerrado por T146. `factory-http` queda en la politica JSON
publica con limite de body, content-type documentado, trailing data, campos
desconocidos y errores compactos sin body crudo. La guia local vive en
`modulos/orquesta-factory-http/README.md` y la respuesta sigue siendo preview de
factory, no plan ejecutable.
Test futuro:
`go test -count=1 ./modulos/orquesta-factory-http ./modulos/orquesta-factory ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway`.
Backlog: `T146 factory-http-json-boundary-coverage`.
```

```text
ID: RUNTIME-CAND-CODEX-PROGRESS-FAILURE-LOG-BOUNDS-001
Origen: scanner backlog 2026-05-24 quincuagesima pasada.
Casos: `codexProgressReadFailureLogV0` lee stdout/stderr/last-message con
`os.ReadFile` completo y recorta despues a 64 KiB para clasificar no-ACK,
interrupcion o capacidad. Un log enorme puede cargar memoria y despues acabar
resumido como decision de progreso sin haber pasado por el rail tail/redaccion.
Decision 2026-05-26: cerrado por T147. Los lectores de failure context de
progreso Codex usan tail acotado antes de clasificar, la firma de accion limita
la lectura aunque el log crezca durante el read y los lectores vecinos de
recovery/detalle ops del stack Codex aplican tail a stdout/stderr/last-message.
Se mantienen codigos compactos y no se promueven stdout/stderr, prompts,
transcripts, HOME, rutas privadas, tokens ni salida cruda de proveedor como
evidencia terminal.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T147 codex-progress-failure-log-tail-bounds`.
```

```text
ID: FILE-CAND-LEDGER-SNAPSHOT-READ-BOUNDS-001
Origen: scanner backlog 2026-05-24 quincuagesima pasada.
Casos: ledgers/snapshots JSON file-based como
`cmd/orquesta-server/external_bridge_input_ledger.go` y
`modulos/orquesta-app-codex-stack/domain_work_delivery_file_ledger_v0.go`
leen el fichero completo antes de validar schema, records o corrupcion. Si el
snapshot crece demasiado o queda corrupto, el error no distingue sobrelimite,
corrupcion recuperable ni riesgo de duplicar efectos externos.
Decision 2026-05-26: los lectores file-based del alcance T148 aplican limite de
bytes y `max_records` por tipo antes de usar el snapshot como estado. Los
errores publicos compactos distinguen sobrelimite, corrupcion y schema invalido
sin paths locales ni payload crudo, y un ledger ilegible no se trata como vacio.
T98, T101 y T104 siguen siendo owners de claim/recovery, idempotencia y
escritura durable.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-app-codex-stack ./modulos/orquesta-domain-work-file ./modulos/orquesta-run-file ./modulos/orquesta-state-file ./modulos/orquesta-runtime-codex-delivery`.
Backlog: `T148 file-ledger-snapshot-read-bounds`.
```

```text
ID: WORKTREE-CAND-SNAPSHOT-READ-BUDGET-001
Origen: scanner backlog 2026-05-24 quincuagesima primera pasada.
Casos: `CaptureWorktreeSnapshotV0` recorre el worktree y
`hashWorktreeFileV0` lee cada fichero completo con `os.ReadFile`. El snapshot
ya se usa como base futura para ACK estricto, line budget y efectos
destructivos, pero no aplica `max_files`, `max_file_bytes`, `max_total_bytes`
ni hash streaming.
Decision: presupuesto de snapshot por composicion, hash por streaming,
ignore prefixes comunes y errores publicos compactos sin path absoluto ni
contenido de fichero. Los agotamientos de presupuesto se proyectan como rails de
revision/followup y no se sustituyen por contenido de `ACK.files`.
Test:
`go test -count=1 ./modulos/orquesta-runtime-worktree ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T149 worktree-snapshot-read-budget`.
Estado: cubierto por T149
```

```text
ID: REVIEW-CAND-PROJECT-TREE-SCAN-BUDGET-001
Origen: scanner backlog 2026-05-24 quincuagesima primera pasada.
Casos: los gates `codexReviewGate*`, `reviewRework*` y
`domainWorkRecoveryFilesUnderDirV0` usan `filepath.WalkDir` para comprobar
globs/carpetas o recuperar artefactos, con listas de ignore copiadas y sin
contexto, max entradas, max profundidad ni reason code cuando el scan agota
presupuesto.
Decision 2026-05-26: `ProjectTreeScanHasFileV0` y `ProjectTreeScanFilesV0`
quedan como politica comun de scan de proyecto para existencia y recovery, con
`context`, presupuesto de entradas/profundidad/tamano/resultados, ignore
prefixes y reason codes publicos. Review gate, review/rework y recovery de
domain_work delegan en esa politica y excluyen control files/directorios locales
como evidencia de producto.
Test:
`go test -count=1 ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack ./modulos/orquesta-runtime-worktree`.
Backlog: `T150 project-tree-scan-budget-for-review-recovery`.
```

```text
ID: CODEX-CAND-WAVE-FILE-INPUT-BOUNDS-001
Origen: scanner backlog 2026-05-24 quincuagesima primera pasada.
Casos: `codexWavePromptTextV0`, `codexDirectorObjectiveTextV0` y
`codexDirectorDomainContextBlocksFromFilesV0` leian ficheros de operador con
`os.ReadFile` completo antes de crear prompts/contexto para agentes Codex.
Decision 2026-05-26: lectura acotada por fichero, validacion UTF-8/texto,
politica de origen local/control/ref opaca y errores publicos redactados. Se
bloquean `.orquesta-runtime`, `.orquesta-codex-runtime`, logs, ACKs,
checkpoints, prompts/transcripts previos, ficheros no regulares y symlinks; los
summaries exponen solo categoria y refs/hash compactos.
Test:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-worktree`.
Backlog: `T151 codex-wave-operator-file-input-bounds`.
Estado: cubierto
```

```text
ID: RAIL-CAND-CODEX-CODEHOME-COPY-BOUNDS-001
Origen: scanner backlog 2026-05-24 quincuagesima segunda pasada.
Casos: la proyeccion de `CODEX_HOME` en olas Codex usa allowlist de nombres,
pero copia ficheros/directorios con `os.Stat`, `os.ReadFile` y `WalkDir` sin
presupuesto de bytes/ficheros ni politica explicita de symlinks, modos o
entradas omitidas.
Decision aplicada 2026-05-26: `codex-wave`/`codex-director-wave` aplican
presupuesto de copia, `Lstat`/apertura sin seguir symlinks, rechazo de
hardlinks/entradas no regulares, modos seguros y recibo compacto de categorias,
contadores, bytes y omissions por reason code. No se copian secretos, HOME,
memorias/plugins completos ni rutas privadas como evidencia.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-worktree ./modulos/orquesta-app-codex-stack`.
Backlog: `T152 codex-code-home-copy-bounds-symlink-policy`.
```

```text
ID: RAIL-CAND-WORKFLOW-PAYLOAD-BUDGET-001
Origen: scanner backlog 2026-05-24 quincuagesima segunda pasada.
Casos: outbox valida `maxOutboxPayloadBytesV0`, pero comandos/eventos del
workflow serializan o decodifican `json.RawMessage` sin presupuesto comun antes
de persistir en `orquesta-state-file`.
Estado: cerrado 2026-05-26 en core/state-file con presupuesto comun alineado
con outbox y overrides tipados para microtareas.
Decision aplicada: comandos/eventos validan tamano antes de `json.Unmarshal`,
`orquesta-state-file` valida eventos antes de compactar/persistir y los
payloads grandes o crudos deben entrar por refs de artefacto/evidencia, no como
JSON durable del workflow.
Test futuro:
`go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-state-file ./modulos/orquesta-orchestration-core ./modulos/orquesta-director ./modulos/orquesta-app-director-service`.
Backlog: `T153 workflow-command-event-payload-budget`.
```

```text
ID: RAIL-CAND-EFFECT-DEADLINE-CONTEXT-001
Origen: scanner backlog 2026-05-24 quincuagesima segunda pasada.
Casos: puertos/adaptadores con efectos externos pueden recibir `context nil` o
ser invocados desde `context.Background()` y quedar gobernados solo por timeout
local o por el cliente inyectado.
Decision pendiente: deadline/cancelacion por politica de composicion para
runtime launch/stop, HTTP domain_work, OPES REST, MCP operador, bridge externo y
shutdown; errores publicos compactos para timeout/cancelacion.
Decision aplicada 2026-05-26: T154 cierra la politica general del write-set con
deadline por efecto en HTTP domain_work, OPES REST, runtime process launch/stop
y bridge OPES residente; quedan tareas vecinas especificas para shutdown
avanzado, transporte MCP residente y presupuestos de espera externa.
Reintento 2026-05-26: `agent-ref-task-autoprogramming-c22c7438ddc1-g01`
verifica que el backlog ya no deja T154 como pendiente y mantiene este rail como
cubierto por pruebas requeridas.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime ./modulos/orquesta-domain-work-http ./modulos/orquesta-opes-connector ./modulos/orquesta-operator-mcp-client ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T154 effect-port-deadline-context-policy`.
Estado: cubierto
```

```text
ID: RAIL-CAND-IDLE-BACKLOG-ACK-SCAN-BUDGET-001
Origen: scanner backlog 2026-05-24 quincuagesima tercera pasada.
Casos: `completedBacklogRequestRefsV0` camina toda `.orquesta-runtime` para
encontrar `agent_ack.json` y deduplicar requests de backlog completadas. En
runs con olas Codex, homes de agentes, plugins y children, ese scan puede tocar
material ajeno al planner sin max de dirs/ficheros/profundidad/tiempo.
Decision aplicada 2026-05-26: indice o scan acotado por refs esperadas, limite
de bytes por ACK, schema/correlacion terminal y degradacion observable
`backlog_ack_scan_budget_exhausted`/`backlog_ack_scan_ambiguous`. Coordinar con
T33, T79, T135 y T143.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`.
Backlog: `T155 idle-backlog-runtime-ack-scan-budget`.
Resolucion 2026-05-26: cerrado en `cmd/orquesta-server` con scan acotado a
profundidad `run/agent/agent_ack.json`, filtros de run backlog, limites de
directorios/entradas/agentes/bytes/duracion y degradacion visible
`backlog_ack_scan_budget_exhausted` o `backlog_ack_scan_ambiguous` hacia scanner
documental.
Revalidacion OrquestaV2 2026-05-26: el rework
`agent-ref-task-ref-review-rework-task-autoprogramming-a8f8edf38716-g01-4dce1eb3cf44`
mantiene T155 cerrado, no relanza otro padre y resuelve contexto `ref_only` por
lectura local/evidencia en ACK.
```

```text
ID: RAIL-CAND-APP-VCS-GIT-OUTPUT-BUDGET-001
Origen: scanner backlog 2026-05-24 quincuagesima tercera pasada.
Casos: `GitAppVCSConnectorV0` y promocion de staging ejecutan Git con
`CombinedOutput()` para `status`, `commit`, `push` y `rev-parse`; `git status
--porcelain --untracked-files=all` puede devolver demasiadas rutas antes de que
AppVCS aplique write-set o evidencias de promocion.
Decision pendiente: captura limitada de stdout/stderr Git, `max_changed_paths`,
errores publicos `git_output_too_large`/`git_status_too_many_paths`, redaccion
de remotos/rutas/HOME/tokens y timeout `git_command_timeout`. Coordinar con T39,
T105, T139 y T154.
Decision aplicada 2026-05-26: AppVCS y promocion de staging usan captura Git
con `max_output_bytes`, presupuesto `max_changed_paths`, errores publicos
`git_output_too_large`, `git_status_too_many_paths` y `git_command_timeout`, y
evidencia compacta sin stdout/stderr crudo, rutas locales, remotos ni diff.
Revalidacion OrquestaV2 2026-05-26: el rework
`agent-ref-task-ref-review-rework-task-autoprogramming-a61a87a140a8-g01-14fd8c3348e6`
mantiene T156 cerrado, no relanza otro padre y resuelve `required ref_only`
mediante lectura local/evidencia explicita en ACK.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime-worktree ./modulos/orquesta-app-codex-stack ./modulos/orquesta-mcp ./cmd/orquesta-server`.
Backlog: `T156 app-vcs-git-output-and-path-budget`.
```

```text
ID: RAIL-CAND-CODEX-CONTROL-FILE-ROOT-SYMLINK-001
Origen: scanner backlog 2026-05-24 quincuagesima cuarta pasada.
Casos: `ReadCodexAgentAckFileV0`, `ReadCodexShutdownCheckpointAckFileV0`,
`directorDecisionFileExistsV0` y `codexProgressReadTailV0` leen ficheros de
control/progreso desde paths de descriptor con `os.ReadFile`, `os.Stat` u
`os.Open`. T143 cubre tamano/redaccion, pero no raiz autorizada, symlinks ni
entradas no regulares antes de abrir.
Decision: cubierto localmente el 2026-05-26 para ACK, checkpoint, sidecar de
decision y tails de progreso del write-set. La lectura comun de control files
usa nombre base esperado, raiz no trivial, `Lstat` antes de abrir, rechazo de
symlink/hardlink/no regular, comparacion despues de abrir y limite T143. Los
tails de progreso y `codex-wave-tail` aplican `Lstat`/no symlink/no hardlink
antes de leer fragmentos.
Test:
`go test -count=1 ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T157 codex-control-file-root-and-symlink-policy`.
Revalidacion OrquestaV2 2026-05-26: el rework
`agent-ref-task-ref-review-rework-task-autoprogramming-0149f7cf3c20-g01-935bd05e71ff`
mantiene T157 cerrado, no relanza otro padre y resuelve `required ref_only`
mediante lectura local/evidencia explicita en ACK.
```

```text
ID: RAIL-CAND-OPES-BRIDGE-DESTINATION-SUMMARY-001
Origen: scanner backlog 2026-05-24 quincuagesima cuarta pasada.
Casos: `opesDrainConfigFromEnvV0` acepta `ORQUESTA_OPES_BASE_URL`/`OPES_BASE_URL`
y `ORQUESTA_BASE_URL` como strings de composicion; `opesDrainSummaryV0` expone
esas bases completas en el summary publico. El bridge tiene confirmacion y
filtro por job, pero no politica OPES especifica de destino ni redaccion de
summary.
Decision aplicada 2026-05-26: politica de destino OPES temporal/productivo,
rechazo de credenciales/query en URL, modo productivo opt-in con evidence ref
compacta y summary con refs/categorias/filtros/contadores en vez de URL completa
o payload crudo.
Test:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-opes-connector ./modulos/orquesta-opes-bridge`.
Backlog: `T158 opes-bridge-destination-and-summary-policy`.
Revalidacion OrquestaV2 2026-05-26:
`agent-ref-task-autoprogramming-453f91b133d4-g01` mantiene T158 cerrado y
resuelve `required ref_only` mediante lectura local/evidencia explicita en ACK.
Retry OrquestaV2 2026-05-26:
`agent-ref-task-autoprogramming-985c0ad5e009-g01` revalida T158 sin nuevo rail y
mantiene `required ref_only` resuelto por evidencia explicita en ACK.
Retry OrquestaV2 2026-05-26:
`agent-ref-task-autoprogramming-bf0d81417acc-g01` revalida T158 sin nuevo rail,
sin cambios de codigo y con `required ref_only` resuelto por lectura local y
evidencia explicita en ACK.
Retry OrquestaV2 2026-05-26:
`agent-ref-task-autoprogramming-761a129dc1b4-g01` revalida T158 sin nuevo rail,
sin smoke real y con `required ref_only` resuelto por lectura local/evidencia
explicita en ACK.
```

```text
ID: RAIL-CAND-CODEX-CONTROL-FILE-WRITE-DURABILITY-001
Origen: scanner backlog 2026-05-24 quincuagesima quinta pasada.
Casos: `codex_resolver_v0.go`, `codex_wave_command_v0.go`,
`codex_wave_control_v0.go` y `codex_shutdown_checkpoint_v0.go` escriben
`agent_packet.json`, `agent_prompt.txt`, wrappers, registry o
`orquesta_shutdown_request.json` con `os.WriteFile` directo sobre ruta final.
T143/T157 cubren lectura, pero no escritura atomica, permisos, `fsync`, rechazo
de symlinks ni receipt compacto.
Decision aplicada 2026-05-26: `orquesta-runtime-codex` centraliza
`WriteCodexControlFileBytesV0` con raiz autorizada, nombre esperado,
temp+rename+fsync, permisos cerrados, bloqueo de symlink/no regular/hardlink,
modo idempotente/conflicto y receipt compacto por hash/bytes/tipo sin path local
ni contenido crudo. `codex_resolver_v0.go`, `codex_wave_control_v0.go`,
`codex_wave_launch_v0.go` y `codex_shutdown_checkpoint_v0.go` quedan migrados.
Test de cierre:
`go test -count=1 ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Backlog: `T159 codex-control-file-durable-write-policy`.
```

```text
ID: RAIL-CAND-DIRECTOR-DECISIONS-BATCH-BUDGET-001
Origen: scanner backlog 2026-05-24 quincuagesima quinta pasada.
Casos: `DirectorAgentDecisionFileSourceV0` limita bytes por fichero, pero
`decisionsFromDescriptorsV0` y `compositeDirectorDecisionSourceV0` agregaban
descriptors/fuentes sin presupuesto visible para numero total de decisions,
`create_microtask` o tasks nuevas antes de materializar workflow/outbox.
Decision aplicada 2026-05-26: limite por request de descriptors, decisions,
microtasks, tasks, outbox esperado y bytes acumulados; exceso con reason code
publico `director_decisions_batch_too_large` y contadores compactos sin bodies
JSON, rutas, prompts ni transcripts. Coordinar con T63 y T153.
Test futuro:
`go test -count=1 ./modulos/orquesta-director-agent-file-source ./modulos/orquesta-director-agent-workflow ./modulos/orquesta-app-codex-stack ./modulos/orquesta-orchestration-core`.
Backlog: `T160 director-decisions-batch-budget-and-source-limit`.
Estado: cubierto
```

```text
ID: RAIL-CAND-CLOCK-REF-GENERATION-001
Origen: scanner backlog 2026-05-24 quincuagesima sexta pasada.
Casos: `codexWaveRefV0` genera refs con resolucion de segundo, varios
adaptadores rellenan `OccurredAt`/`RequestedAt` con `time.Now` directo y web/CLI
tienen fallbacks de request id basados en `UnixNano`. En concurrencia o replay,
un timestamp puede parecer identidad causal aunque solo sea evidencia temporal.
Decision pendiente: owner comun de reloj/ref generator por composicion, reloj
inyectable en tests, colision observable y prohibicion de usar timestamps como
unica identidad terminal.
Estado 2026-05-26: resuelto para el alcance T161. Owner compartido:
`modulos/orquesta-runtime/clock_ref_policy_v0.go`; adopcion focal en
`codex-wave`, runtime launch, progreso Codex delivery y fallbacks web/CLI.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-runtime ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack ./modulos/orquesta-web ./modulos/orquesta-cli`.
Backlog: `T161 clock-and-ref-generation-policy`.
```

```text
ID: RAIL-CAND-PUBLIC-CLIENT-IDEMPOTENCY-001
Origen: scanner backlog 2026-05-24 quincuagesima sexta pasada.
Casos: clientes web/CLI/MCP y comandos del servidor propagan `request_id`,
`correlation_id` e `idempotency_key` de forma desigual; algunas mutaciones
generan ID local, otras aceptan idempotency opcional y otras reutilizan
correlacion como identidad. Un retry tras timeout puede duplicar efecto o dejar
traza incompleta.
Decision pendiente: politica comun para mutaciones publicas reintentables:
idempotency estable, correlation transversal, headers coherentes, receipt/estado
observable y errores redactados.
Test futuro:
`go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-cli ./modulos/orquesta-mcp ./cmd/orquesta-server ./modulos/orquesta-app-gateway ./modulos/orquesta-http-gateway`.
Backlog: `T162 public-client-mutation-idempotency-policy`.
Cierre 2026-05-26: politica comun implementada en `orquesta-mcp` y consumida
por web/CLI/MCP/gateways para prepare-run, run-control, queue-priority mutante,
shutdown y AppVCS. Validacion focal:
`go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-cli ./modulos/orquesta-mcp ./cmd/orquesta-server ./modulos/orquesta-app-gateway ./modulos/orquesta-http-gateway`.
```

```text
ID: RAIL-CAND-STATE-FILE-EVENT-LOG-BUDGET-001
Origen: scanner backlog 2026-05-24 quincuagesima septima pasada.
Casos: `AppendRunEventsV0` lee todo el documento de eventos del run, recorre
todos los eventos para deduplicar por `event_id` y reescribe el snapshot JSON
completo en cada append. En runs residentes largas, el coste y el riesgo de
snapshot grande/corrupto crecen antes de que replay o cierre causal puedan
degradar de forma observable.
Decision pendiente: presupuesto por append/run, indice durable por `event_id`,
lectura paginada o ventana causal, compaction compatible y reason codes para
log excesivo/corrupto sin asumir historial vacio.
Decision 2026-05-26: cerrado en `orquesta-state-file` con indice durable,
registros por evento, presupuestos de append/run/payload, lectura paginada
`LoadRunEventsPageV0` y consumidores de cierre/review que declaran presupuesto
al pedir historial completo.
Test futuro:
`go test -count=1 ./modulos/orquesta-state-file ./modulos/orquesta-core-workflow ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-director-service`.
Backlog: `T163 state-file-event-log-index-compaction-budget`.
```

```text
ID: RAIL-CAND-WORKFLOW-TASK-PARENT-INDEX-BUDGET-001
Origen: scanner backlog 2026-05-24 quincuagesima septima pasada.
Casos: `LoadWorkflowTasksByParentV0` hace `os.ReadDir` del directorio de tasks
del run y abre cada JSON para filtrar por `parent_task_ref`. Recursion real,
waits por parent y cierre de arbol pueden depender de un scan sin presupuesto ni
indice, y una ausencia/corrupcion puede parecer "sin hijos" si no se distingue.
Decision pendiente: indice parent/child durable con presupuesto de entradas y
bytes, rebuild acotado con reason code, bloqueo si el indice falta o hay outbox
pendiente, y conservacion de wave/cohort/depth para waits por parentesco.
Decision 2026-05-26: cerrado en `orquesta-state-file` con indice durable por
run `workflow_task_parent_index.v0`, actualizacion en `SaveWorkflowTaskV0`,
lectura por refs indexadas en `LoadWorkflowTasksByParentV0` y rebuild acotado
con `workflow_task_parent_index_rebuild_required` /
`workflow_task_parent_index_budget_exhausted`. La ausencia o corrupcion del
indice no se interpreta como "sin hijos"; si no puede rematerializarse dentro
del presupuesto, la lectura falla y el consumidor debe bloquear recursion/cierre.
Test futuro:
`go test -count=1 ./modulos/orquesta-state-file ./modulos/orquesta-core-workflow ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-director-service ./modulos/orquesta-app-codex-stack`.
Backlog: `T164 workflow-task-store-parent-index-budget`.
```

```text
ID: RAIL-CAND-SERVER-CONFIG-GLOBAL-ENV-DEFAULTS-001
Origen: scanner backlog 2026-05-24 quincuagesima septima pasada.
Casos: `serverConfigFromEnvV0`, `setDefaultStartupCleanupModeV0` y
`ensureServerDetailRailsDefaultV0` fijan defaults con `os.Setenv` en el entorno
global del proceso. Eso mezcla input explicito del operador con defaults de
composicion y puede afectar tests, smokes, comandos hijos o lecturas posteriores
de config.
Decision cerrada 2026-05-26: config efectiva sin mutar entorno global,
proyeccion al daemon marcada por conteos `explicit`/`defaulted`/`derived`,
summary redactado por categorias y tests focales de lecturas repetidas sin
contaminacion global.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-rails ./modulos/orquesta-app-codex-stack`.
Backlog: `T165 server-config-global-env-defaults-policy`.
```

```text
ID: OPS-CAND-RESIDENT-ASYNC-SHUTDOWN-001
Origen: scanner backlog 2026-05-24 quincuagesima octava pasada.
Casos: `RuntimeV0.RunV0` lanza `server.Serve`, supervisor ticks async e idle
self-improvement en goroutines; durante shutdown usa
`server.Shutdown(context.Background())`, persiste `stopped` con background y no
espera las goroutines internas antes de terminar.
Decision pendiente: quiescencia async con deadline, estados `stopping`/
`async_work_draining`/`stop_timeout`, receipts compactos y bloqueo de writes
posteriores a `stopped`.
Test futuro:
`go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server ./modulos/orquesta-observability`.
Backlog: `T166 resident-runtime-async-shutdown-quiescence`.
Estado: cubierto 2026-05-26 por RuntimeV0 con grupo async interno, deadline
`ORQUESTA_SERVER_SHUTDOWN_GRACE_MS`, publicacion `stopping`/
`async_work_draining`, `stopped` solo tras quiescencia y `stop_timeout` durable
con `shutdown_async_work_active`.
```

```text
ID: OPS-CAND-EXTERNAL-BRIDGE-LOOP-LIFECYCLE-001
Origen: scanner backlog 2026-05-24 quincuagesima octava pasada.
Casos: `cmd/orquesta-server run` arranca `runOPESBridgeLoopV0` en una goroutine
paralela al runtime; `writeExternalBridgeTickV0` publica cada tick solo por
stderr. El status/auditoria residente no conserva ultimo tick, ultimo error,
estado `stopping` ni join/cancelacion del bridge.
Decision cerrada 2026-05-26: el loop residente OPES usa observer de lifecycle,
publica estado compacto en status/readiness, audita errores de tick redactados
y coordina join/cancel con `cmd/orquesta-server run`; un timeout de bridge se
reporta como `external_bridge_shutdown_timeout` sin exito silencioso.
Test:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-opes-bridge`.
Estado: cubierto.
Backlog: `T167 external-bridge-resident-loop-lifecycle-state`.
```

```text
ID: STATE-CAND-RUN-EVENT-LOAD-VALIDATION-001
Origen: scanner backlog 2026-05-24 quincuagesima octava pasada.
Casos: `StoreV0.LoadRunV0` valida `schema_version` y refs externas del documento
de run, pero no ejecuta `ValidateOrchestrationRunV0` sobre la proyeccion
cargada. `LoadRunEventsV0` valida schema/ref del documento de eventos, pero no
valida cada `OrchestrationEventV0` cargado ni duplicados antes de entregar el
historial a replay/cierre.
Decision cerrada 2026-05-26: `LoadRunV0` valida la proyeccion con
`ValidateOrchestrationRunV0`; `LoadRunEventsV0` valida eventos con
`ValidateOrchestrationEventV0`, refs de run y duplicados, incluidos documentos
legacy y registros corruptos. El lector devuelve error publico compacto y no
trata documentos invalidos como run vacia, sin eventos o completed.
Test:
`go test -count=1 ./modulos/orquesta-state-file ./modulos/orquesta-core-workflow ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-director-service`.
Estado: cubierto.
Backlog: `T168 state-file-run-event-load-validation`.
```

```text
ID: MCP-CAND-REAL-TRANSPORT-REGISTRATION-COLLISION-001
Origen: scanner backlog 2026-05-24 quincuagesima octava pasada.
Casos: `mcpRealTransportRegistryV0.RegisterResourceV0` y `RegisterToolV0`
guardan envelopes en mapas por nombre/URI y sobrescriben duplicados
silenciosamente. Si dos descriptors colisionan, `/mcp` puede ocultar un
resource/tool sin error de registro.
Decision pendiente: rechazar colisiones de tool/resource en el transporte real,
propagar error desde `RegisterMCPTransportV0` y demostrar que no queda catalogo
parcial tras fallo.
Cierre 2026-05-26: implementado en el transporte MCP real con rechazo de
duplicados de resource por `name`/`uri`, tool por `name`, errores publicos
redactados y pruebas de no mutacion parcial/listado estable. El contrato puro
mantiene prueba de propagacion de errores del puerto.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-mcp`.
Backlog: `T169 mcp-real-transport-registration-collision-guard`.
```

```text
ID: RAIL-CAND-OUTBOUND-REDIRECT-POLICY-001
Origen: scanner backlog 2026-05-24 quincuagesima novena pasada.
Casos: los clientes HTTP revisados en `orquesta-domain-work-http`,
`orquesta-opes-connector`, `orquesta-web`, `orquesta-mcp`,
`orquesta-app-gateway` y `cmd/orquesta-server` declaran timeout, pero no
`CheckRedirect` ni una politica comun de cadena 3xx. Go sigue redirects por
defecto, por lo que la URL inicial puede estar validada y el destino efectivo
terminar fuera de origen/politica.
Decision cerrada 2026-05-26: los clientes HTTP salientes del write-set T170
declaran politica de redirect y revalidan cada salto contra origen, esquema,
puerto y path cuando el conector lo declara. Los saltos cross-origin, esquema no
permitido, credenciales o fragmento quedan bloqueados con reason code compacto;
no se autorizo reenvio de headers sensibles fuera del origen inicial.
Revalidacion OrquestaV2 2026-05-26:
`agent-ref-task-ref-review-rework-task-autoprogramming-74734b6a029e-g01-9160302b840b`
mantiene el cierre sin relanzar otro agente padre y resuelve `required ref_only`
por evidencia explicita en ACK.
Test de cierre:
`go test -count=1 ./modulos/orquesta-domain-work-http ./modulos/orquesta-opes-connector ./modulos/orquesta-web ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./cmd/orquesta-server`.
Backlog: `T170 outbound-http-redirect-policy`.
```

```text
ID: RAIL-CAND-PUBLIC-QUERY-FORM-BOUNDS-001
Origen: scanner backlog 2026-05-24 quincuagesima novena pasada.
Casos: `auditHTTPHandlerV0` conserva `raw_query`; endpoints web/server usan
`ParseForm`, `FormValue` o `URL.Query().Get` para controles publicos. T137
limita cuerpos JSON, pero no impone presupuesto/redaccion comun sobre query
string ni form params antes de auditoria/status.
Decision cerrada 2026-05-26: web aplica limites de query/form antes de
`URL.Query`/`ParseForm`; auditoria residente omite `raw_query`, limita claves y
redacta nombres sensibles con reason codes compactos.
Revalidacion OrquestaV2 2026-05-26:
`agent-ref-task-ref-review-rework-task-autoprogramming-03f6b0927ba2-g01-24a1e410cd44`
no reabre T171; el contexto `ref_only` requerido queda resuelto por lectura
local de paquete/docs y evidencia explicita en ACK.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-web ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-http-gateway`.
Backlog: `T171 public-query-form-parameter-bounds-redaction`.
```

```text
ID: RAIL-CAND-HTTP-RESPONSE-WRITE-VISIBILITY-001
Origen: scanner backlog 2026-05-24 quincuagesima novena pasada.
Casos: muchos handlers publicos en server/web/MCP/gateway usan
`_ = json.NewEncoder(w).Encode(...)` o ignoran errores de `w.Write(...)`. Si la
serializacion o escritura falla, el dominio puede haber tenido exito mientras
la entrega HTTP queda invisible para status, auditoria o smokes.
Decision pendiente: helper o patron comun para distinguir fallo de dominio,
fallo de serializacion y fallo de escritura; registrar `response_write_failed`
compacto cuando el status ya fue emitido y no filtrar payloads internos.
Resolucion 2026-05-26: el servidor residente incorpora helper JSON con
serializacion previa a headers y la auditoria HTTP registra
`response_write_failed` con etapa compacta, separando exito de dominio de fallo
de entrega sin guardar payloads.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-web ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-observability`.
Backlog: `T172 http-response-encode-write-error-visibility`.
```

```text
ID: RAIL-CAND-BROWSER-ORIGIN-CSRF-001
Origen: scanner backlog 2026-05-24 sexagesima pasada.
Casos: rutas web/API mutables aceptan POST por navegador o cliente local sin
owner visible para `Origin`, `Referer`, CSRF token o intent ref. T55 cubre
auth/bind remoto y T102/T137 cubren JSON/form/query, pero un POST de navegador
hacia loopback o bind opt-in puede activar control plane si solo se valida body.
Resolucion 2026-05-26: `orquesta-http-gateway` declara mutabilidad publica y
expone una guarda comun de origen/intencion aplicada por `orquesta-app-gateway`;
browser/form requiere `Origin`/`Referer` same-origin o token de intencion, y
clientes JSON no-browser conservan el flujo local/MCP existente.
Decision tomada: declarar por ruta si acepta navegador, CLI/MCP o gateway;
exigir same-origin o intent token para mutaciones browser y registrar decision
compacta sin cookies, query cruda, URL completa ni payload.
Test futuro:
`go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-app-gateway ./modulos/orquesta-mcp ./modulos/orquesta-http-gateway ./cmd/orquesta-server`.
Backlog: `T173 browser-origin-csrf-intent-guard`.
```

```text
ID: RAIL-CAND-MCP-JSONRPC-STRICTNESS-001
Origen: scanner backlog 2026-05-24 sexagesima pasada.
Casos: `/mcp` decodifica un unico objeto JSON-RPC con `LimitReader`, pero no
valida estrictamente `jsonrpc=2.0`, trailing tokens, tipo/tamano de `id`,
batch/notification ni presupuesto de `params` por metodo. T23/T169 cubren
transporte opt-in y colisiones de registro; falta owner de protocolo.
Decision pendiente: contrato JSON-RPC estricto o modo legacy declarado para
version, id, batch, notifications, params, `Content-Type`/`Accept` y errores,
sin eco de argumentos, resource payloads, rutas, prompts, transcripts ni tokens.
Resolucion 2026-05-26: `/mcp` adopta contrato estricto JSON-RPC 2.0:
batch rechazado, notifications limitadas a `notifications/initialized` sin
`id`, `id` seguro, presupuesto de `params`, `Accept` JSON compatible y params
estrictos para `resources/read` y `tools/call`; los errores usan reason codes
compactos sin eco de payloads.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway`.
Backlog: `T174 mcp-jsonrpc-protocol-strictness`.
```

```text
ID: RAIL-CAND-MCP-OUTPUT-BUDGET-001
Origen: scanner backlog 2026-05-24 sexagesima primera pasada.
Casos: `cmd/orquesta-server/mcp_real_transport_v0.go` envuelve payloads de
`resources/read` y `tools/call` como texto completo sin limite comun de salida,
freshness ni redaccion por resource/tool. Un recurso grande o un resultado de
herramienta con diagnostico amplio puede superar presupuesto o filtrar material
operativo antes de que T174 actue sobre el request.
Decision 2026-05-26: presupuesto de salida MCP comun por resource/tool con
modo, freshness, redaccion publica, diagnostico crudo solo opt-in y bloqueo
compacto antes de serializar JSON-RPC.
Test:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-mcp ./modulos/orquesta-observability`.
Backlog: `T175 mcp-tool-resource-output-budget`.
```

```text
ID: RAIL-CAND-CONTROL-PLANE-SECURITY-HEADERS-001
Origen: scanner backlog 2026-05-24 sexagesima primera pasada.
Casos: handlers web/MCP/gateway fijan `Content-Type`, `Allow` y a veces
`X-Correlation-ID`, pero no comparten `Content-Security-Policy`,
`X-Content-Type-Options`, `Referrer-Policy`, anti-frame ni `Cache-Control`.
T173 cubre origen/CSRF; este rail cubre respuesta/cache.
Decision 2026-05-26: helper comun `NewControlPlaneHTTPHeadersV0` con perfiles
HTML, JSON y MCP, cache explicita `no-store`, `nosniff`, `Referrer-Policy`,
anti-frame y reason code publico sin copiar cookies, query, URLs completas ni
datos privados.
Test futuro:
`go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-http-gateway ./cmd/orquesta-server`.
Backlog: `T176 control-plane-http-security-cache-headers`.
```

```text
ID: RAIL-CAND-REST-BASE-URL-ENDPOINT-001
Origen: scanner backlog 2026-05-24 sexagesima primera pasada.
Casos: clientes REST de web/MCP/CLI y composicion unen base URL y endpoint con
concatenacion o `strings.TrimRight/TrimLeft`; algunos normalizan esquema/host y
otros aceptan `BaseURL` como string ya confiable. Antes de T80/T170/T103 falta
un rail comun para userinfo, query, fragment, base path y endpoint absoluto.
Decision 2026-05-26: politica equivalente por adaptador (`webRESTEndpointURLV0`,
`buildCLIRESTEndpointURLV0`, `joinMCPRESTEndpointV0` y
`commandRESTEndpointURLV0`) para parsear base URL, preservar base path, rechazar
userinfo/query/fragment y endpoints absolutos, `..`, query o paths vacios. El
gateway conserva `http://orquesta.internal` solo como destino interno in-process
cubierto por tests, no como egress real.
Test:
`go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-cli ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./cmd/orquesta-server`.
Backlog: `T177 rest-client-base-url-endpoint-policy`.
```

```text
ID: RAIL-CAND-HTTP-CLIENT-TRANSPORT-PROXY-001
Origen: scanner backlog 2026-05-24 sexagesima segunda pasada.
Casos: clientes salientes de web, CLI, MCP, OPES, domain-work y comandos del
servidor crean `http.Client{Timeout: ...}` o cliente nil con transporte por
defecto. Asi proxy, TLS, keepalive y pool de conexiones quedan como decision
implicita del entorno/proceso, incluso para loopback o control plane interno.
Decision pendiente: factory/perfil de transporte por cliente, proxy deny por
defecto para interno/loopback, proxy explicito o allowlisted para egress, y
auditoria por categoria sin URL completa, userinfo, tokens ni rutas privadas.
Cierre 2026-05-26: T178 queda cerrado localmente con factory/perfil por
adaptador: `domain_egress`, `opes_temporal`, `loopback_control_plane` e
`internal_inprocess`. La politica activa fija `proxy_policy=deny`, dial/TLS/
headers, pool y keepalive para red real; proxy/TLS custom queda pendiente como
opt-in futuro de composicion, sin habilitar ni auditar valores crudos en este
corte.
Test futuro:
`go test -count=1 ./modulos/orquesta-domain-work-http ./modulos/orquesta-opes-connector ./modulos/orquesta-web ./modulos/orquesta-cli ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./cmd/orquesta-server`.
Backlog: `T178 outbound-http-client-transport-proxy-policy`.
```

```text
ID: RAIL-CAND-INPROCESS-HTTP-TRANSPORT-001
Origen: scanner backlog 2026-05-24 sexagesima segunda pasada.
Casos: `InProcessTransportV0` y el roundtripper del smoke MCP llaman al handler
directamente y acumulan la respuesta en memoria. Los tests/gateway pueden pasar
sin ejercer limite de salida, cancelacion observable, panic recovery o fallos de
escritura que aparecerian en HTTP real.
Decision pendiente: recorder acotado, propagacion/verificacion de deadline,
errores publicos `inprocess_timeout`/`inprocess_response_too_large` y paridad de
headers/status/correlacion con frontera HTTP real.
Cierre 2026-05-26: el recorder acotado vive en
`modulos/orquesta-app-gateway/inprocesshttp`; `InProcessTransportV0`, helpers web
de test y smoke MCP in-process del servidor lo reutilizan con errores publicos
compactos por exceso, cancelacion, timeout, panic y escritura cerrada.
Test futuro:
`go test -count=1 ./modulos/orquesta-app-gateway ./modulos/orquesta-web ./modulos/orquesta-mcp ./cmd/orquesta-server ./modulos/orquesta-observability`.
Backlog: `T179 inprocess-http-transport-budget-parity`.
```

```text
ID: RAIL-CAND-COMMAND-STDIO-WRITE-001
Origen: scanner backlog 2026-05-24 sexagesima tercera pasada.
Casos: comandos de `cmd/orquesta-server` y runners CLI ignoran errores de
`json.NewEncoder(stdout).Encode(...)` o `fmt.Fprintf(stdout/stderr, ...)`.
T139 define shape publico de comandos y T172 cubre escritura HTTP, pero no hay
owner visible para pipe roto, stdout cerrado o writer de test que falla.
Decision pendiente: comprobar errores de escritura stdio, devolver exit code o
reason code publico y no usar stdout/stderr textual como evidencia terminal de
cierre.
Estado 2026-05-26: cerrado para la visibilidad base. Servidor y CLI proyectan
fallos de escritura stdout como `command_output_write_failed`; observabilidad
mantiene el contrato compacto/redactado y stderr queda best-effort, no evidencia
unica del cierre.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-cli ./modulos/orquesta-observability`.
Backlog: `T180 command-stdio-write-error-visibility`.
```

```text
ID: RAIL-CAND-DOCPLAN-REF-UNIQUENESS-001
Origen: scanner backlog 2026-05-24 sexagesima cuarta pasada.
Casos: `DomainDocumentPlanV0` valida presencia y forma compacta de secciones,
visuales, reviews y entregables, pero no hay rail visible para refs duplicados.
El expander genera `RequestID`/`IdempotencyKey` desde esos refs, asi que un
duplicado puede producir jobs derivados indistinguibles. Ademas, arrays raw
malformados pueden colapsar a `nil` durante canonicalizacion y perder campo
causal.
Estado 2026-05-26: cerrado. El contrato publica
`domain_document_plan_ref_duplicate` para duplicados por tipo,
`domain_document_plan_array_invalid` para arrays raw invalidos por campo,
deriva refs faltantes con sufijos deterministas y el expander bloquea
`RequestID`/`IdempotencyKey` duplicados antes de emitir jobs.
Test ejecutado:
`go test -count=1 ./modulos/orquesta-domain-work ./modulos/orquesta-document-plan-expander ./modulos/orquesta-opes-bridge`.
Backlog: `T181 domain-document-plan-ref-uniqueness-and-diagnostics`.
```

```text
ID: RAIL-CAND-MCP-TOOL-EXEC-BUDGET-001
Origen: scanner backlog 2026-05-24 sexagesima cuarta pasada.
Casos: T174 limita forma/protocolo JSON-RPC y T175 limita salida de
tools/resources, pero el transporte MCP real invoca handlers con el contexto
del request y sin presupuesto de ejecucion por perfil. Un handler lento o que no
observe cancelacion puede retener `/mcp` sin reason code publico propio.
Decision 2026-05-26: deadline/cancel cause por tool/resource antes de invocar
handler, perfiles separados para lectura, mutacion y autoprogramacion larga, y
observabilidad compacta sin argumentos, payloads, prompts ni rutas privadas.
El transporte real devuelve `mcp_tool_timeout`/`mcp_tool_cancelled` y solo
registra metodo, perfil, reason code, bucket de duracion y correlacion.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-observability`.
Backlog: `T182 mcp-tool-execution-budget-and-cancellation`.
```

```text
ID: RAIL-CAND-WEB-HTML-RENDER-ERROR-001
Origen: scanner backlog 2026-05-24 sexagesima cuarta pasada.
Casos: renderers HTML de `modulos/orquesta-web` ejecutan templates sobre el
writer HTTP y varios caminos ignoran el error de `template.Execute`. T172 cubre
fallos de escritura HTTP genericos, pero falta owner para distinguir template
invalido, socket tardio y fallback localizado en vistas HTML.
Decision pendiente: comprobar errores de render/escritura, publicar reason code
estable, no reejecutar efectos y registrar solo contadores compactos sin
formularios, query cruda, cookies, payloads, prompts, HOME, rutas privadas ni
tokens.
Cierre 2026-05-27: helper comun de `orquesta-web` bufferiza `template.Execute`,
fallback preserva locale posible, gateway observa render/write por headers y
writer wrapper, observability guarda solo contador/reason/stage/status/locale.
Test:
`go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-app-gateway ./modulos/orquesta-observability ./cmd/orquesta-server`.
Backlog: `T183 web-html-render-error-contract`.
Estado: cubierto
```

```text
ID: RAIL-CAND-DETERMINISTIC-REF-HASH-COLLISION-001
Origen: scanner backlog 2026-05-24 sexagesima quinta pasada.
Casos: refs y firmas en app-change, MCP, orchestration-core, runtime Codex
delivery y app Codex stack usan FNV32, SHA1 truncado o `strings.Join` con
separadores textuales. Eso es aceptable para diagnostico/advisory, pero puede
ser ambiguo o colisionable si alimenta identidad causal, idempotencia,
recuperacion o evidencia durable.
Decision pendiente: builder canonico no ambiguo para refs causales, margen de
digest suficiente, distincion explicita entre huella visual y ref causal, y
conflicto reparable cuando una ref existente corresponde a otra fuente.
Decision cerrada 2026-05-27: los owners del write-set T184 usan `sha256` sobre
payload canonico con longitud de campos para refs causales, evidencias durables
y firmas progress/rework. El retry de autoprogramacion ya no concatena
`request_ref` y timestamp con separador textual; el core tiene prueba focal de
namespace y payload no ambiguo. Las huellas solo diagnosticas quedan
documentadas como advisory.
Rework de revision 2026-05-27:
`agent-ref-task-ref-review-rework-task-autoprogramming-298dd18fed1e-g01-c005a1d6953bff0a3fc8364f7d4ac61b`
conserva la decision cerrada sin relanzar otro agente padre; el contexto
`required ref_only` queda resuelto por lectura local del paquete y evidencia
explicita en ACK.
Test futuro:
`go test -count=1 ./modulos/orquesta-app-change-director-source ./modulos/orquesta-mcp ./modulos/orquesta-orchestration-core ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack`.
Backlog: `T184 deterministic-ref-hash-collision-proof`.
```

```text
ID: RAIL-CAND-SMOKE-TEMP-ROOT-DELETION-001
Origen: scanner backlog 2026-05-24 sexagesima quinta pasada.
Casos: smokes y runners aceptan raices por `ORQUESTA_SMOKE_ROOT` o
`ORQUESTA_PARALLEL_TEST_TMP` y luego ejecutan `rm -rf` sobre esa raiz durante
cleanup. Si una variable apunta al proyecto, HOME, `.orquesta-runtime` o una
ruta compartida, el borrado puede ser mucho mas amplio que el smoke.
Decision cerrada 2026-05-26: `scripts/lib/smoke_common.sh` aporta helper comun
de cleanup con prefijo temporal permitido, marcador `.orquesta-smoke-root.v0`,
bloqueo de rutas prohibidas y conservacion de raices no verificadas con reason
code publico. Los scripts de smoke/runners ya no llaman `rm -rf` directamente
para raices temporales.
Test vigente:
`bash -n scripts/*.sh scripts/lib/*.sh` y prueba focal del helper con raiz
valida, raiz sin marcador y ruta prohibida.
Rework de revision 2026-05-26:
`agent-ref-task-ref-review-rework-task-autoprogramming-66353e1f46d9-g01-33cae0148aa014c3f796cbabf72f5114`
mantiene la decision cerrada sin relanzar otro agente padre; el contexto
`required ref_only` queda resuelto por lectura local del paquete y evidencia
explicita en ACK.
Backlog: `T185 smoke-script-temp-root-deletion-guard`.
```

```text
ID: RAIL-CAND-DOMAIN-WORK-JOB-FINGERPRINT-001
Origen: scanner backlog 2026-05-24 sexagesima sexta pasada.
Casos: `orquesta-domain-work-memory`, `orquesta-domain-work-file` y
`orquesta-domain-work-sql` derivan fingerprints/job refs con FNV64 base36 desde
payload canonico local. Es suficiente como huella corta de referencia, pero
queda cerca de idempotencia, replay y conflicto durable entre adaptadores.
Decision 2026-05-26: cerrado por
`agent-ref-task-autoprogramming-5abbd2ab1d6c-g01`. El builder canonico
`BuildDomainWorkJobIdentityV0` usa `sha256`; memory/file/SQL lo comparten para
fingerprint y base de `job_ref`, preservan replay legacy por request guardado y
reparan colisiones de `job_ref` con sufijo explicito.
Test futuro:
`go test -count=1 ./modulos/orquesta-domain-work ./modulos/orquesta-domain-work-memory ./modulos/orquesta-domain-work-file ./modulos/orquesta-domain-work-sql`.
Backlog: `T186 domain-work-job-ref-fingerprint-collision-proof`.
```

```text
ID: RAIL-CAND-HTTP-AUDIT-CLIENT-IDENTITY-001
Origen: scanner backlog 2026-05-24 sexagesima sexta pasada.
Casos: `auditHTTPHandlerV0` persiste `remote_addr` completo en eventos
`http_request`. T171 cubre query/form y T132 privacidad general, pero falta
politica concreta para IP:puerto, loopback, redes privadas y headers
`X-Forwarded-*`.
Decision pendiente: guardar categoria/hash/redaccion, aceptar headers de proxy
solo con perfil confiable y no convertir identidad declarada por cliente en
autorizacion o evidencia durable.
Estado 2026-05-26: cerrado para `orquesta-server`; `http_request` guarda
`client_identity` por categoria y politica `category_only`, ignora valores
`X-Forwarded-*`/`Forwarded` por defecto y conserva autorizacion en control
plane, no en headers de cliente.
Revalidacion 2026-05-27: el caso queda cubierto sin guardar `RemoteAddr`,
IP:puerto ni valores de headers de forwarding en evidencia durable.
Test futuro:
`go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-observability ./cmd/orquesta-server`.
Backlog: `T187 http-audit-client-identity-redaction-policy`.
```

```text
ID: RAIL-CAND-HTTP-METHOD-CONTRACT-001
Estado: cerrado el 2026-05-26 por
`task-autoprogramming-aa85b0fb040c-g01`.
Origen: scanner backlog 2026-05-24 sexagesima sexta pasada.
Casos: handlers HTTP/MCP/web/gobernanza devuelven 405 con patrones locales; en
algunas rutas se espera `Allow` y en otras no hay contrato comun para `OPTIONS`.
Decision aplicada: contrato por perfil para metodo no permitido, header
`Allow`, `OPTIONS` sin efectos y shape de error publico sin payload crudo.
Revalidacion 2026-05-27: contrato verificado con bateria focal de MCP, web,
governance, factory HTTP y transporte MCP real.
Test futuro:
`go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-web ./modulos/orquesta-governance ./modulos/orquesta-factory-http ./cmd/orquesta-server`.
Backlog: `T188 http-method-allow-options-contract`.
```

```text
ID: RAIL-CAND-SERVER-SIGNAL-SHUTDOWN-001
Origen: scanner backlog 2026-05-24 sexagesima septima pasada.
Casos: `cmd/orquesta-server run` crea contexto con `signal.NotifyContext` solo
para `os.Interrupt`; `orquesta-server stop` tambien senala el PID con
`os.Interrupt`. En modo daemon o service manager puede llegar SIGTERM, segunda
senal o timeout de cierre sin contrato publico ni estado observable propio.
Decision implementada 2026-05-26: politica por plataforma para
interrupcion/terminacion, deadline de gracia por
`ORQUESTA_SERVER_SHUTDOWN_GRACE_MS`, segunda senal como `signal_escalated` y
status/auditoria compacta de `stopping_by_signal`/`stop_timeout` sin PID crudo,
HOME, rutas, env ni logs.
Rework de revision 2026-05-26:
`agent-ref-task-ref-review-rework-task-autoprogramming-13b40a6e5fc4-g01-a9146f390bba643a490794bd55481129`
solo corrige la clasificacion documental de T189 y conserva este rail como
decision implementada; no reabre checkpoint de agentes, identidad de proceso ni
quiescencia interna.
Test:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-server-shutdown`.
Backlog: `T189 server-resident-signal-shutdown-policy`.
```

```text
ID: RAIL-CAND-HTTP-GATEWAY-ROUTE-MANIFEST-001
Origen: scanner backlog 2026-05-24 sexagesima septima pasada.
Casos: `orquesta-http-gateway` registra rutas exactas y prefijo
`/api/v0/apps/` en `ServeMux`; `orquesta-app-gateway` monta AppVCS con otro mux
en `/api/v0/apps/vcs`. La precedencia depende del mux y puede cambiar al sumar
rutas bajo prefijos existentes sin rail de colision/shadowing.
Decision pendiente: manifiesto canonico de rutas/prefijos con owner, test de
colisiones exactas, prefijos ambiguos y dispatch de overlays; evidencia solo con
refs compactas de ruta, sin query, cookies, headers, payloads ni rutas privadas.
Estado: cubierto 2026-05-26. `PublicRouteManifestV0` declara inventario,
owners, metodos y perfiles; los tests validan colisiones, shadows declarados,
dispatch de AppVCS frente al prefijo de app-change y overlays `/mcp`/workspace
timeline por refs compactas.
Test futuro:
`go test -count=1 ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway ./modulos/orquesta-mcp ./cmd/orquesta-server`.
Backlog: `T190 http-gateway-route-manifest-collision-guard`.
```

```text
ID: RAIL-CAND-PROCESS-RUNTIME-STOP-ESCALATION-001
Origen: scanner backlog 2026-05-24 sexagesima septima pasada.
Casos: `ProcessRuntimeConnectorV0.signalProcessStopV0` envia `os.Interrupt` y
solo ejecuta `Kill` si `Signal` falla. Un proceso que recibe la senal pero la
ignora queda sin deadline de gracia, escalado, estado `stopping` ni reason code.
Decision 2026-05-26: parada cooperativa con timeout, escalation/kill
observable, codigos para senal no soportada/proceso detenido/timeout/kill
fallido y tests con proceso que sale, ignora senal y ya estaba cerrado.
Estado 2026-05-26: resuelto en `ProcessRuntimeConnectorV0`; el snapshot expone
`stopping`, `stop_grace_deadline` y `stop_reason_code`, y el timeout de gracia
escala a kill con ACK compacto.
Rework de revision 2026-05-26: esta entrada queda cerrada y sincronizada con
T191; no genera rail nuevo ni reabre checkpoint/shutdown de otros owners.
Test futuro:
`go test -count=1 ./modulos/orquesta-runtime ./modulos/orquesta-runtime-required-test ./modulos/orquesta-orchestration-core ./cmd/orquesta-server`.
Backlog: `T191 process-runtime-stop-signal-escalation-policy`.
```

```text
ID: RAIL-CAND-SERVER-STATUS-MESSAGE-001
Origen: scanner backlog 2026-05-24 sexagesima octava pasada.
Casos: `status_tracker_v0.go` construye razones de automejora idle a partir de
mensajes, acciones y evidencias de adaptadores; `supervisor_loop_v0.go` emite
eventos de auditoria con requests/results/plans seleccionados.
Decision pendiente: proyeccion de mensajes operativos con codigo de razon, refs
opacas, limites de bytes y redaccion antes de persistir/exponer estado.
Resolucion 2026-05-26: `StateV0` y `ServerPublicStatusV0` mantienen campos
legacy y publican `*_operational_message` estructurados para startup,
supervisor, automejora idle, bridge externo y ultimo error; `auditEventV0`
compacta payloads operativos completos a `*_summary` con refs opacas y no
persiste errores crudos sensibles.
Rework 2026-05-26: entrega revalidada sin relanzar otro padre; el contexto
`required ref_only` queda resuelto por lectura local y evidencia explicita en
ACK.
Test futuro:
`go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server ./modulos/orquesta-observability`.
Backlog: `T192 server-status-operational-message-projection`.
```

```text
ID: RAIL-CAND-FACTORY-APPSPEC-TIME-001
Origen: scanner backlog 2026-05-24 sexagesima octava pasada.
Casos: `appspec_usecase_v0.go` acepta `now`, pero si llega cero usa
`time.Now().UTC()`; `appspec_http_v0.go` ya inyecta reloj desde adaptador.
Decision pendiente: reloj obligatorio por composicion o fallback convertido en
politica explicita/versionada con pruebas de determinismo.
Decision aplicada 2026-05-26: `SolicitarNuevaAppV0` exige reloj inyectado y
rechaza `now` cero con `app_spec_invalida`/`received_at`/
`reloj_recepcion_utc_requerido`; el adaptador HTTP mantiene reloj UTC inyectable
y tests deterministas.
Rework 2026-05-26: entrega revalidada sin relanzar otro padre; el contexto
`required ref_only` queda resuelto por lectura local y evidencia explicita en
ACK.
Test futuro:
`go test -count=1 ./modulos/orquesta-factory ./modulos/orquesta-factory-http ./modulos/orquesta-web ./modulos/orquesta-mcp`.
Backlog: `T193 factory-appspec-time-source-contract`.
```

```text
ID: RAIL-CAND-GOVERNANCE-CATALOG-BUDGET-001
Origen: scanner backlog 2026-05-24 sexagesima octava pasada.
Casos: `governance_catalog_public_query_v0.go` proyecta entradas efectivas del
catalogo para consultas publicas sin owner visible para presupuesto de salida,
frescura y source refs.
Decision cerrada 2026-05-26: proyeccion publica bounded con `output_budget`
normalizado, version/freshness/source refs, `inactive_summary` por conteo/refs
acotadas y sin payload de estados no publicos por defecto.
Test futuro:
`go test -count=1 ./modulos/orquesta-governance ./modulos/orquesta-cli ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway`.
Backlog: `T194 governance-catalog-output-budget-freshness`.
```

```text
ID: RAIL-CAND-MCP-TOOL-SCHEMA-DESCRIPTOR-001
Origen: scanner backlog 2026-05-24 sexagesima novena pasada.
Casos: tools de `orquesta-mcp` declaran `InputSchema`/`Output` como strings
compactos escritos a mano; el transporte MCP real los reexpone y las
capabilities de operador declaran `OutputShape`/`InputRefs` en otra fuente.
Decision 2026-05-26: descriptor verificable por tool contra DTO y registro
real. `orquesta-mcp` deriva campos desde DTOs Go por tool, el transporte MCP
real consume esa fuente canonica para `inputSchema` y expone
`mcp_transport_schema_stale` solo como fallback detectable. Los subtools de
operador publican shape, refs requeridas y errores publicos; si falta puerto
siguen devolviendo `operator_mcp_port_unavailable` sin ocultar el tool.
Test futuro:
`go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-app-gateway ./cmd/orquesta-server`.
Backlog: `T195 mcp-tool-input-schema-descriptor-sync`.
```

```text
ID: RAIL-CAND-CLI-COMMAND-CATALOG-001
Origen: scanner backlog 2026-05-24 sexagesima novena pasada.
Casos: `modulos/orquesta-cli/command_runner_v0.go` mantiene texto de ayuda
ES/EN, dispatch por `hasCLIPathV0` y `FlagSet` por comando como fuentes
manuales separadas. Un comando nuevo puede quedar ejecutable pero no anunciado,
o anunciado con flags desactualizadas.
Decision cerrada 2026-05-26: `modulos/orquesta-cli` declara
`cliCommandCatalogV0` como catalogo canonico local; ayuda localizada, dispatch y
paridad de flags visibles se prueban contra el catalogo. El error de comando
desconocido usa path normalizado, sugerencias acotadas y redaccion de tokens
sensibles sin volcar argumentos completos.
Test futuro:
`go test -count=1 ./modulos/orquesta-cli ./cmd/orquesta-server`.
Backlog: `T196 cli-command-catalog-help-dispatch-sync`.
```

```text
ID: RAIL-CAND-CLI-RESPONSE-BOUNDS-001
Origen: scanner backlog 2026-05-24 septuagesima pasada.
Casos: clientes REST de `orquesta-cli` para FunctionContract,
OperationalStatus, gobernanza, status/run control y comandos afines leen bodies
con `io.ReadAll` o decoders directos antes de producir envelopes publicos.
Decision 2026-05-27: T197 queda cerrado para `orquesta-cli`; los clientes REST
consumen `transport_rest_response_v0.go` con limite por comando,
content-type/trailing JSON y detalle no-2xx redactado, sin devolver body crudo
ni truncar JSON silenciosamente.
Test:
`go test -count=1 ./modulos/orquesta-cli ./modulos/orquesta-web ./modulos/orquesta-mcp ./cmd/orquesta-server`.
Backlog: `T197 cli-rest-response-bounds-redaction-parity`.
```

```text
ID: RAIL-CAND-MCP-RESOURCE-DESCRIPTOR-001
Origen: scanner backlog 2026-05-24 septuagesima pasada.
Casos: resources MCP como operational-status, shared/core contracts, roadmap,
governance y operator capabilities publican shapes, public errors, refs
canonicas y guardrails como strings estaticos separados de DTOs/validadores.
Resolucion 2026-05-27: los resources registrados en MCP transportan
`descriptor_source` con owner, fuente canonica, freshness, DTO/validador,
fuente de errores publicos y verificacion; `resources/list` del transporte real
lo publica sin datos sensibles ni rutas locales. Los resources opt-in conservan
errores publicos `not_configured`/`unavailable` por puerto sin inventar schema.
Revalidacion burst 002 2026-05-27:
`agent-ref-task-autoprogramming-85571f97bc5e-g01` confirma que T198 sigue
cerrado focalmente. El contexto `ref_only` se resuelve por lectura local y
evidencia ACK; no se abre codigo nuevo ni se amplian owners fuera del write-set.
Revalidacion retry 2026-05-27:
`agent-ref-task-autoprogramming-44e164597ee4-g01` confirma el mismo cierre con
lectura local del contexto `ref_only` y prueba obligatoria focal; no se amplia
write-set ni se reabre T195/T197/T199.
Reconciliacion backlog 2026-05-27:
`agent-ref-task-autoprogramming-974732911968-g01` sincroniza backlog y docs de
owners locales con el cierre focal de T198. No hay rail nuevo ni apertura de
codigo; el seguimiento queda como evidencia documental de que el pendiente
residual era stale.
Reconciliacion adicional 2026-05-27:
`agent-ref-task-autoprogramming-c3678e9bc306-g01` confirma el mismo patron
stale tras intentos cerrados de T198. El contexto obligatorio `ref_only` se
resuelve por lectura local/evidencia ACK; no se amplia write-set, no se abre
rail nuevo y no se reabre implementacion.
Test futuro:
`go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-observability ./modulos/orquesta-governance ./modulos/orquesta-core ./cmd/orquesta-server`.
Backlog: `T198 mcp-resource-descriptor-source-sync`.
```

```text
ID: RAIL-CAND-MCP-PUBLIC-ERROR-CATALOG-001
Origen: scanner backlog 2026-05-24 septuagesima pasada.
Casos: helpers MCP/HTTP y operador usan strings locales de error publico
(`metodo_no_permitido`, `request_body_invalido`, `*_no_configurado`,
`*_error`) con mappings distintos entre HTTP, JSON-RPC, CLI/web y resources.
Resolucion 2026-05-27: T199 cerrado localmente con catalogo comun en
`orquesta-i18n-docs` y pruebas de paridad en MCP, operador, web, CLI y servidor.
Los codigos base incluyen i18n key, retryability, severidad y mapping por
frontera; `err.Error()` queda allowlisted en helpers MCP/operador o cae a
codigo generico sin payload crudo.
Refuerzo 2026-05-27: los mensajes publicos de executor MCP no incluyen detalle
de errores desconocidos aunque sea sanitizado; solo anaden el codigo cuando esta
en el catalogo comun.
Test futuro:
`go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-web ./modulos/orquesta-cli ./modulos/orquesta-i18n-docs ./cmd/orquesta-server`.
Backlog: `T199 mcp-public-error-code-catalog`.
```

```text
ID: RAIL-CAND-CMD-SERVER-REST-CLIENT-001
Origen: scanner backlog 2026-05-24 septuagesima primera pasada.
Casos: comandos locales de `cmd/orquesta-server` (`status`, `run-status`,
`stop`) usan clientes HTTP propios. `getStatusBodyV0` y `postRunStatusBodyV0`
leen respuestas con `io.ReadAll`; `run-status` incorpora body no-2xx completo
en stderr; `requestServerShutdownV0` decodifica JSON sin limite ni trailing-data
check.
Decision pendiente: helper o prueba de paridad para limite de respuesta,
content-type, trailing JSON y redaccion de errores del binario servidor, sin
volcar HTML, URLs con credenciales, rutas privadas, HOME, tokens, prompts,
transcripts ni payloads de dominio.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-observability`.
Backlog: `T200 cmd-server-management-rest-client-policy`.
```

```text
ID: RAIL-CAND-OUTBOUND-RETRY-BACKOFF-001
Origen: scanner backlog 2026-05-24 septuagesima segunda pasada.
Casos: `runExternalBridgeLoopV0` reintenta ticks con intervalo fijo y los
conectores `domain_work-http`/OPES devuelven errores compactos sin politica
comun de `Retry-After`, jitter, presupuesto de reintentos, rate limit ni circuit
breaker por destino/ref.
Decision pendiente: definir retry/backoff/rate por adaptador de composicion,
con reason codes publicos y sin reintentar mutaciones no idempotentes cuando
falte ledger o `idempotency_key` causal.
Resolucion focal 2026-05-27: T201 introduce politica opt-in de retry/backoff en
`domain_work-http` y OPES REST, respeta `Retry-After` solo dentro de deadline y
delay maximo, bloquea mutaciones sin identidad idempotente, y hace que
`runExternalBridgeLoopV0` publique `rate_limited`, `retry_scheduled` o
`retry_budget_exhausted` para errores consecutivos. Revalidacion assessment
OrquestaV2 2026-05-27: la bateria obligatoria de T201 pasa completa en el
write-set declarado.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-domain-work-http ./modulos/orquesta-opes-connector ./modulos/orquesta-opes-bridge ./modulos/orquesta-server`.
Backlog: `T201 outbound-connector-retry-backoff-rate-policy`.
```

```text
ID: RAIL-CAND-OUTBOUND-CORRELATION-HEADERS-001
Origen: scanner backlog 2026-05-24 septuagesima segunda pasada.
Casos: `modulos/orquesta-domain-work-http/client_v0.go` fija solo
`Content-Type` y `modulos/orquesta-opes-connector/http_v0.go` fija
`Accept`/`Content-Type`; `correlation_id` e `idempotency_key` viajan en JSON
pero no como cabeceras compactas para proxies, apps externas o ledgers HTTP.
Decision pendiente: propagar `X-Correlation-ID`, `Idempotency-Key` y/o ref
equivalente desde campos ya validados; bloquear valores no compactos y no poner
prompts, transcripts, rutas privadas, HOME, tokens ni payloads de dominio en
headers.
Test futuro:
`go test -count=1 ./modulos/orquesta-domain-work-http ./modulos/orquesta-opes-connector ./cmd/orquesta-server ./modulos/orquesta-app-codex-stack`.
Backlog: `T202 outbound-domain-correlation-idempotency-headers`.
```

```text
ID: RAIL-CAND-OPES-PAGINATION-WINDOW-001
Origen: scanner backlog 2026-05-24 septuagesima segunda pasada.
Casos: `ListExternalJobsV0` lee una sola respuesta de OPES con `limit` y
`opesBridgeScanLimitV0` capado a 100. Si los primeros jobs ya estan en ledger,
OPES no garantiza orden/cursor o la respuesta ignora limit, el bridge puede
revisar siempre la misma ventana y no llegar a jobs pendientes posteriores.
Decision pendiente: contrato de cursor/orden/ventana para OPES o bloqueo
publico `pagination_not_supported`/`window_exhausted`; la secuencia por tipos no
debe tratar `Seen > 0` como progreso si no hubo envio ni avance real.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-opes-connector ./modulos/orquesta-opes-bridge`.
Backlog: `T203 opes-bridge-pagination-window-policy`.
```

```text
ID: RAIL-CAND-OPS-DASHBOARD-LIVE-CACHE-001
Origen: scanner backlog 2026-05-27 primera pasada.
Casos: `/ops` calcula agregados y medias en JS a partir de `percent_complete`,
`usage_summary` y cache de runs activos/completados. Aunque el DTO de stats ya
pueda reflejar agente vivo, proceso registrado o entrega sin cierre, un fetch
tardio/fallido o un snapshot completado puede devolver el dashboard a 0% o
cuota `-` sin reason code.
Decision cerrada 2026-05-27: `/ops` fija owner visual de frescura/cache en
`modulos/orquesta-web` mediante proyeccion publica con `progress_source`,
`freshness`, `stats_fetch_status` y reason code compacto. La entrega viva no se
mezcla con cierre, pero tampoco se muestra como 0% cuando hay agentes, proceso o
entrega; la cuota sin reporte aparece como `unknown`/`unavailable` con causa
publica.
Test futuro:
`go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-server ./modulos/orquesta-orchestration-core ./cmd/orquesta-server`.
Backlog: `T211 ops-dashboard-live-progress-cache-policy`.
```

```text
ID: RAIL-CAND-GUARDIAN-PUBLIC-RESULT-001
Origen: scanner backlog 2026-05-27 segunda pasada.
Casos: `orquesta-guardian` publica y persiste `project_dir`, `current_bin`,
`candidate_bin`, `last_good_bin`, `output_path` y comandos shell completos en
resultado/manifest/repair packet.
Estado 2026-05-27: cubierto para T212. `orquesta-guardian` separa envelope
publico redactado de manifest local diagnostico; el repair packet para agentes
lleva refs/evidence compactas, no paths absolutos, HOME, comandos expandidos con
secretos ni logs completos.
Test futuro: `go test -count=1 ./cmd/orquesta-guardian ./cmd/orquesta-server ./modulos/orquesta-runtime-worktree ./modulos/orquesta-runtime-codex`.
Backlog: `T212 guardian-public-result-and-repair-packet-redaction`.
```

```text
ID: RAIL-CAND-GUARDIAN-OUTPUT-ENV-001
Origen: scanner backlog 2026-05-27 segunda pasada.
Casos: build/test/healthcheck/repair del guardian acumulan stdout/stderr en
`bytes.Buffer`, escriben logs sin presupuesto y heredan `os.Environ()` para
candidato temporal y repair command.
Decision pendiente: limite/tail/redaccion por comando y entorno minimo por
perfil; no heredar tokens, HOME, proxies con credenciales, DSN, prompts,
transcripts ni payloads de dominio por defecto.
Test futuro: `go test -count=1 ./cmd/orquesta-guardian ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-runtime-codex`.
Backlog: `T213 guardian-command-output-budget-and-env-isolation`.
```

```text
ID: RAIL-CAND-GUARDIAN-BINARY-PROMOTION-001
Origen: scanner backlog 2026-05-27 segunda pasada.
Casos: `copyFileAtomicV0` copia binarios con `os.ReadFile`, tmp por
`UnixNano`, sin presupuesto, hash/manifest fuerte, symlink policy ni fsync
visible.
Estado 2026-05-27: cubierto para T214. `cmd/orquesta-guardian` usa copia
streaming con presupuesto configurable, SHA-256, `Lstat`/open/post-copy guard,
bloqueo de symlinks/hardlinks inseguros, raiz declarada opt-in, tmp no
colisionable por `os.CreateTemp`, fsync y manifest interno
`orquesta_guardian_artifact_manifest.v0`. Los fallos de backup/promocion
publican `last_good_unverified` o `promotion_incomplete` sin declarar
`candidate_promoted`.
Test futuro: `go test -count=1 ./cmd/orquesta-guardian ./cmd/orquesta-server ./modulos/orquesta-runtime-worktree`.
Backlog: `T214 guardian-binary-promotion-artifact-policy`.
```

```text
ID: RAIL-CAND-GUARDIAN-READINESS-001
Origen: scanner backlog 2026-05-27 tercera pasada.
Casos: el runbook del guardian pide `/api/v0/server/readiness` antes de efectos
externos, pero `runGuardianCandidateHealthcheckV0` solo espera `/healthz`.
Un candidato con HTTP vivo y readiness rota podria promocionarse.
Decision pendiente: liveness por `/healthz` mas readiness operativa por
`/api/v0/server/readiness`, con razon publica `candidate_readiness_not_ready` y
respuesta acotada/redactada.
Estado 2026-05-27: cerrado para T215. `runGuardianCandidateHealthcheckV0`
espera liveness y despues readiness `ready=true` del candidato; readiness rota,
JSON no valido o ruta versionada ausente bloquean promocion con reason code
publico `candidate_readiness_not_ready`. La prueba focal cubre candidato HTTP
vivo sin readiness y candidato listo contra estado/runtime temporal.
Test futuro:
`go test -count=1 ./cmd/orquesta-guardian ./cmd/orquesta-server ./modulos/orquesta-server`.
Backlog: `T215 guardian-candidate-readiness-gate`.
```

```text
ID: RAIL-CAND-PROMOTION-GUARDIAN-RESULT-001
Origen: scanner backlog 2026-05-27 tercera pasada.
Casos: `shellAutoprogrammingPromotionGuardianRunnerV0` ejecuta el guardian por
shell y decide por exit code; descarta stdout/stderr y no valida
`orquesta_guardian_result.v0`, `status`, `promoted` ni `evidence_refs`.
Estado 2026-05-27: cerrado para T216. El servidor consume stdout acotado,
parsea `orquesta_guardian_result.v0`, propaga `evidence_refs` compactas y
distingue resultado invalido, timeout, exit no cero y estados publicos del
guardian sin publicar stdout/stderr crudo.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./cmd/orquesta-guardian ./modulos/orquesta-autoprogramming`.
Backlog: `T216 promotion-guardian-result-contract`.
```

```text
ID: RAIL-CAND-GUARDIAN-SHUTDOWN-CLIENT-001
Origen: scanner backlog 2026-05-27 tercera pasada.
Casos: `requestGuardianServerShutdownV0` decodifica respuesta de shutdown sin
limite, sin `Content-Type`, sin trailing-data check y con `idempotency_key`
constante para todas las invocaciones del guardian.
Decision pendiente: cliente HTTP acotado para shutdown del guardian, reason
codes compactos, idempotencia por intento/ref y senal PID solo con estado
publico fiable.
Test futuro:
`go test -count=1 ./cmd/orquesta-guardian ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-server-shutdown`.
Backlog: `T217 guardian-shutdown-http-client-policy`.
```

```text
ID: RAIL-CAND-GUARDIAN-REPAIR-AGENT-001
Origen: scanner backlog 2026-05-27 cuarta pasada.
Casos: `--repair-codex` genera `codex-launch-wave --agents 1` desde el
guardian con sandbox amplio, `approval-policy never`, prompt por path local y
sin contrato explicito de write-set, ACK terminal, pruebas requeridas,
presupuesto ni checkpoint.
Decision 2026-05-27: cerrado para el contrato de lanzamiento Codex del guardian.
`--repair-codex` genera packet versionado
`orquesta_guardian_repair_launch_packet.v0`, exige write-set y pruebas
requeridas, usa `codex-launch-director-wave` con branch/worktree refs y
break-glass unmanaged auditado, y declara `agent_ack.json` estructurado como
contrato terminal. El sandbox default queda en `workspace-write`; sandbox amplio
requiere opt-in y evidence ref. La preferencia por cola normal queda como regla
operativa del runbook cuando el servidor residente este sano, no como cierre por
stdout/exit code.
Test futuro:
`go test -count=1 ./cmd/orquesta-guardian ./cmd/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack`.
Backlog: `T218 guardian-repair-agent-launch-contract`.
```

```text
ID: RAIL-CLOSED-GUARDIAN-OUTPUT-ENV-T213-001
Origen: cierre OrquestaV2 2026-05-27.
Casos: build/test/healthcheck/repair del guardian podian conservar stdout/stderr
sin presupuesto y heredar entorno completo del proceso padre.
Decision: cerrado en T213 con captura tail redactada, reason code publico
`guardian_command_output_budget_exceeded`, redaccion de secretos/HOME/paths y
entorno minimo allowlistado. Los comandos con efectos externos siguen bajo
RAIL-CAND-GUARDIAN-COMMAND-EFFECT-001.
Test:
`go test -count=1 ./cmd/orquesta-guardian ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-runtime-codex`.
```

```text
ID: RAIL-CAND-GUARDIAN-PATH-ROOT-001
Origen: scanner backlog 2026-05-27 cuarta pasada.
Casos: `absPathFromBaseV0` aceptaba rutas absolutas para project/state/current/
candidate/last_good/repair runtime y esas rutas podian cruzar a manifest,
resultado o repair packet sin clasificacion de raiz de producto/control.
Estado 2026-05-27: cerrado para T219. `orquesta-guardian` clasifica raices y
paths criticos con `orquesta_guardian_path_policy.v0`, bloquea binarios fuera de
`project_dir`/`state_dir`/`artifact_root`, runtime de reparacion fuera de
`state_dir` y raices con symlinks; la salida publica, audit y repair packet solo
exponen refs hash, clasificaciones y reason codes compactos. `cmd/orquesta-server`
acepta el campo publico `path_policy` y conserva `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro:
`go test -count=1 ./cmd/orquesta-guardian ./cmd/orquesta-server ./modulos/orquesta-runtime-worktree ./modulos/orquesta-runtime-codex`.
Backlog: `T219 guardian-path-root-and-control-surface-policy`.
```

```text
ID: RAIL-CAND-GUARDIAN-MANIFEST-RETENTION-001
Origen: scanner backlog 2026-05-27 cuarta pasada.
Casos: manifests, repair packets y prompts del guardian se escriben con
`os.WriteFile` directo y timestamp de segundo; `candidate-state`,
`candidate-runtime` y logs quedan en `state_dir` sin inventario ni retencion.
Decision pendiente: escritura tmp+fsync+rename, intento idempotente por ref,
estado de manifest incompleto y retencion/limpieza por categoria sin borrar
evidencia requerida.
Decision aplicada 2026-05-27: `cmd/orquesta-guardian` escribe manifest, repair
packet, prompt y logs redactados con tmp+fsync+rename; indexa por
`attempt_ref`/`promotion_ref`/`shutdown_ref`, publica
`guardian_manifest_incomplete` ante conflicto de payload o tmp incompleto y
expone `retention_status` por categoria sin paths absolutos.
Test futuro:
`go test -count=1 ./cmd/orquesta-guardian ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-runtime-worktree`.
Backlog: `T220 guardian-manifest-retention-and-replay-policy`.
```

```text
ID: RAIL-CAND-GUARDIAN-PROMOTION-LEASE-001
Origen: scanner backlog 2026-05-27 quinta pasada.
Casos: `cmd/orquesta-guardian` puede ejecutar `check-promote`,
`restore-last-good` o `shutdown-server` en paralelo contra el mismo `state_dir`,
`current_bin` y `last_good_bin`. La copia atomica de T214 no impide que dos
intentos crucen manifests, `candidate`, `last_good` y resultado publico.
Decision 2026-05-27: lease durable por `attempt_ref`/`promotion_ref`, bloqueo
publico `guardian_promotion_lease_busy` ante intento activo y verificacion del
lease justo antes de promover/restaurar/senalizar; si el lease se pierde,
`guardian_promotion_lease_lost` bloquea sin declarar efecto.
Test futuro:
`go test -count=1 ./cmd/orquesta-guardian ./cmd/orquesta-server ./modulos/orquesta-autoprogramming ./modulos/orquesta-runtime-worktree`.
Backlog: `T221 guardian-promotion-lease-and-concurrency-policy`.
```

```text
ID: RAIL-CAND-GUARDIAN-COMMAND-EFFECT-001
Origen: scanner backlog 2026-05-27 quinta pasada.
Casos: el servidor lanza el guardian con shell configurable y el guardian
acepta `build-command`, `test-command` y `repair-command` como shell libre. T213
cubre output/env, pero no autorizacion de comandos por perfil de efecto ni
opt-in para red, Git remoto, OPES, proveedor real o acciones destructivas.
Decision aplicada 2026-05-27: `cmd/orquesta-guardian` publica `command_ref`,
perfil (`build`, `required_test`, `healthcheck`, `repair`), policy ref,
autorizacion, efectos externos y evidence refs compactas por comando. El default
solo permite build canonico y `go test`; shell no canonico o efectos de red, Git
remoto, OPES, proveedor real o destructivos bloquean con
`guardian_command_effect_policy_blocked` sin exponer el shell completo, salvo
opt-in/evidence ref de composicion. La expansion de placeholders valida rutas
shell-quoted y bloquea placeholders de ruta sin resolver con
`guardian_command_placeholder_invalid`.
Test futuro:
`go test -count=1 ./cmd/orquesta-guardian ./cmd/orquesta-server ./modulos/orquesta-autoprogramming`.
Backlog: `T222 guardian-command-effect-profile-policy`.
```

```text
ID: RAIL-CAND-PROMOTION-GUARDIAN-RECEIPT-001
Origen: scanner backlog 2026-05-27 quinta pasada.
Casos: el servidor reduce el resultado del guardian a refs genericas
`guardian_passed`/`guardian_failed`. T216 exige parsear resultado estructurado,
pero falta receipt causal que una `promotion_ref`, `run_ref`,
`worktree_ref`/`branch_ref`, manifest, hash del candidato y efecto de staging.
Decision pendiente: `promotion_guardian_receipt.v0` con estados
`candidate_verified`, `candidate_promoted`, `promotion_blocked`,
`last_good_restored`, resultado invalido y retry idempotente; salida publica
solo con refs opacas/hashes compactos.
Estado 2026-05-27: cerrado para `cmd/orquesta-server`. El efecto de promocion
adjunta un receipt causal `promotion_guardian_receipt.v0`, conserva refs opacas,
manifest/evidence refs y hash/tamano compacto del candidato si existe, y bloquea
`--promote=false` como `candidate_verified` para no declarar binario activo.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./cmd/orquesta-guardian ./modulos/orquesta-autoprogramming ./modulos/orquesta-runtime-worktree`.
Backlog: `T223 promotion-guardian-causal-receipt`.
```

```text
ID: RAIL-CAND-GUARDIAN-CANDIDATE-PROCESS-001
Origen: scanner backlog 2026-05-27 sexta pasada.
Casos: el healthcheck del guardian arranca el candidato como proceso hijo,
espera `/healthz` y luego intenta `Interrupt` + `Kill` solo sobre el proceso
padre. No hay receipt de parada del arbol completo ni bloqueo si quedan hijos
vivos antes de promocionar.
Estado 2026-05-27: cerrado para el guardian. El healthcheck publica
`process_policy` y `stop_receipt` compacto; Unix usa grupo de proceso propio con
deadline, escalado y confirmacion antes de promocionar, y Windows declara
alcance de proceso padre. Un stop ambiguo o vivo falla el healthcheck con reason
code publico sin PID crudo, rutas locales, stdout/stderr ni env.
Test:
`go test -count=1 ./cmd/orquesta-guardian ./cmd/orquesta-server ./modulos/orquesta-server`.
Backlog: `T224 guardian-candidate-process-tree-lifecycle`.
```

```text
ID: RAIL-CAND-GUARDIAN-FORCED-SHUTDOWN-001
Origen: scanner backlog 2026-05-27 sexta pasada.
Casos: `force_after_timeout=true` por defecto puede convertir timeout
cooperativo en shutdown forzado y senal a PID si existe `ServerPID`, aunque el
resultado no declare `shutdown_ready`.
Decision implementada el 2026-05-27: `force_after_timeout` ya no escala por
defecto; requiere opt-in explicito y evidence ref break-glass. Sin esa evidencia
el guardian bloquea con `guardian_shutdown_escalation_blocked` y no emite
shutdown forzado ni senal local. El resultado publico distingue
`cooperative_timeout`, `forced_requested`, `forced_ready`, `signal_sent` y
`signal_blocked` sin exponer PID, host, HOME, tokens, URLs ni cuerpos HTTP.
Test futuro:
`go test -count=1 ./cmd/orquesta-guardian ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-server-shutdown`.
Backlog: `T225 guardian-forced-shutdown-escalation-contract`.
```

```text
ID: RAIL-CAND-GUARDIAN-REPAIR-BUDGET-001
Origen: scanner backlog 2026-05-27 sexta pasada.
Casos: cada fallo de build/test/healthcheck puede escribir repair packet y
lanzar `repair-command` o `--repair-codex` otra vez aunque el failure packet sea
el mismo y no exista evidencia nueva.
Decision implementada el 2026-05-27: cada repair publica `repair_attempt_ref`,
`failure_packet_hash` y presupuesto por scope de promocion/intento/run. El
guardian registra intento durable antes de lanzar reparacion, bloquea reentrada
equivalente con `guardian_repair_attempt_duplicate` y bloquea exceso de
presupuesto con `guardian_repair_attempt_budget_exhausted`; el servidor consume
los campos publicos sin rutas locales ni stdout/stderr.
Test ejecutable:
`go test -count=1 ./cmd/orquesta-guardian ./cmd/orquesta-server ./modulos/orquesta-autoprogramming ./modulos/orquesta-runtime-codex`.
Backlog: `T226 guardian-repair-attempt-budget-and-idempotency`.
```

```text
ID: RAIL-CAND-GUARDIAN-CANDIDATE-ADDR-001
Origen: scanner backlog 2026-05-27 octava pasada.
Casos: `freeLocalAddrV0` obtiene `127.0.0.1:0`, cierra el listener y despues
lanza el candidato con ese addr; otro proceso puede ocupar el puerto antes del
bind real. `ORQUESTA_GUARDIAN_CANDIDATE_ADDR` tambien puede aportar un addr no
gobernado sin politica de loopback/ownership.
Decision 2026-05-27: T228 cierra el rail para `cmd/orquesta-guardian`; addr
automatico usa `127.0.0.1:0` y addr real desde statefile del candidato,
`ORQUESTA_GUARDIAN_CANDIDATE_ADDR` solo acepta loopback con puerto concreto y
las formas sin ownership publican `candidate_addr_unowned`.
Retry OrquestaV2 2026-05-27:
`agent-ref-task-autoprogramming-db88e93cffaf-g01` revalida el cierre con la
bateria focal requerida; el contexto `ref_only` requerido queda resuelto por
lectura local y evidencia explicita en ACK.
Test ejecutable:
`go test -count=1 ./cmd/orquesta-guardian ./cmd/orquesta-server ./modulos/orquesta-server`.
Backlog: `T228 guardian-candidate-address-ownership-policy`.
```

```text
ID: RAIL-CAND-PROMOTION-GUARDIAN-RUNNER-ENV-001
Origen: scanner backlog 2026-05-27 octava pasada.
Casos: `shellAutoprogrammingPromotionGuardianRunnerV0` invoca el guardian con
comando shell configurable y `os.Environ()` completo. T213 aisla comandos
internos del guardian, pero la frontera servidor -> guardian puede seguir
heredando HOME, tokens, proxies, DSN o variables de proveedor.
Decision 2026-05-27: `shellAutoprogrammingPromotionGuardianRunnerV0` usa entorno
`minimal_allowlist`, no hereda `os.Environ()` completo y bloquea HOME, Codex,
proxies, Git, proveedor y secretos salvo allowlist explicita con evidence refs.
La salida del proceso guardian sigue acotada y validada por contrato
`orquesta_guardian_result.v0`.
Retry OrquestaV2 2026-05-27:
`agent-ref-task-autoprogramming-22cfff60721c-g01` revalida el cierre con la
bateria focal requerida y contexto `ref_only` resuelto por lectura local mas
evidencia explicita en ACK.
Rework de revision 2026-05-27:
`agent-ref-task-ref-review-rework-task-autoprogramming-22cfff60721c-g01-938062230e08e3a836b342951d5c588d`
cierra el hueco restante de herencia accidental de `ORQUESTA_GUARDIAN_*`: el
runner descarta esas variables del padre y solo publica las derivadas del
request antes de invocar el guardian.
Retry de rework OrquestaV2 2026-05-27:
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-autoprogr-3b1cd0fc9711a3bf097b5db163987e69`
revalida el rail con lectura local, bateria focal requerida y evidencia
explicita en ACK; no cambia el alcance ni reabre T213/T216/T222.
Retry final de rework OrquestaV2 2026-05-27:
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-602dc9edc1ac71b6268bcede5729e588`
revalida el rail y conserva que el runner solo use variables
`ORQUESTA_GUARDIAN_*` derivadas del request, con contexto `ref_only` resuelto
por lectura local mas evidencia explicita en ACK.
Test futuro:
`go test -count=1 ./cmd/orquesta-server ./cmd/orquesta-guardian ./modulos/orquesta-autoprogramming`.
Backlog: `T229 promotion-guardian-runner-env-isolation`.
Estado: cerrado localmente.
```

```text
ID: RAIL-CAND-GUARDIAN-SKIP-HEALTH-001
Origen: scanner backlog 2026-05-27 octava pasada.
Casos: `--skip-health` puede saltar el healthcheck vivo del candidato y permitir
promocion tras build/tests locales. T215 exige readiness cuando se ejecuta el
healthcheck, pero no gobierna la excepcion que lo desactiva.
Decision aplicada 2026-05-27: bloquear skip con `promote=true` salvo evidence
ref break-glass explicita, publicar `guardian_healthcheck_required` si falta la
evidencia y distinguir la promocion excepcional como
`candidate_promoted_breakglass` con `candidate_built`,
`candidate_tests_passed` cuando hay tests y `candidate_not_live_checked`.
Test futuro:
`go test -count=1 ./cmd/orquesta-guardian ./cmd/orquesta-server ./modulos/orquesta-autoprogramming`.
Backlog: `T230 guardian-skip-health-breakglass-policy`.
```

```text
ID: RAIL-CAND-GUARDIAN-CONFIG-ENV-STRICTNESS-001
Origen: scanner backlog 2026-05-27 novena pasada.
Casos: los helpers `envBoolOrDefaultV0`, `envDurationOrDefaultV0`,
`envIntOrDefaultV0`, `envInt64OrDefaultV0` y
`guardianOutputMaxBytesFromEnvV0` devuelven defaults ante valores invalidos.
Una variable mal escrita puede dejar `promote=true`, `force_after_timeout=true`
o budgets/timeouts por defecto sin diagnostico en una frontera break-glass.
Decision aplicada 2026-05-27: parseo estricto con reason codes publicos,
`config_effective` compacto en resultado publico y bloqueo seguro antes de
promocionar, restaurar o senalizar. El servidor distingue config invalida de
fallo de candidato al ejecutar el guardian.
Test:
`go test -count=1 ./cmd/orquesta-guardian ./cmd/orquesta-server ./modulos/orquesta-autoprogramming`.
Estado: cubierto por T231.
```

```text
ID: RAIL-CAND-GUARDIAN-CLI-EXTRA-ARGS-001
Origen: scanner backlog 2026-05-27 novena pasada.
Casos: `parseGuardianConfigForCommandV0` ejecuta `flags.Parse(args)` sin
validar `flags.NArg()`. Argumentos sobrantes o posicionales ambiguos pueden
quedar ignorados mientras `check-promote`, `restore-last-good` o
`shutdown-server` continuan con defaults.
Decision aplicada 2026-05-27: `cmd/orquesta-guardian` rechaza argumentos
sobrantes tras `flag.Parse` con `guardian_config_extra_args` y detalle publico
`positional_args:<n>`, sin imprimir los valores crudos. El runbook documenta que
los comandos shell entran por `--build-command`, `--test-command`,
`--repair-command` o env canonica, no como posicionales.
Test futuro:
`go test -count=1 ./cmd/orquesta-guardian ./cmd/orquesta-server`.
Backlog: `T232 guardian-cli-extra-args-strictness`.
```

```text
ID: RAIL-CAND-GUARDIAN-LOCAL-DIAGNOSTIC-REF-001
Origen: scanner backlog 2026-05-27 decima pasada.
Casos: `guardianPathRefV0` genera `ManifestRef`, `RepairPacketRef`,
`OutputRef` y refs de `LocalDiagnostics` desde el path local limpio. No publica
el path, pero la identidad publica cambia con `state_dir`/worktree temporal y
no queda ligada a `promotion_ref`, `attempt_ref` ni hash de contenido.
Decision 2026-05-27: T233 cerrado en `cmd/orquesta-guardian`; las refs
publicas de manifest, repair packet, repair launch, outputs y diagnosticos
locales se derivan de ref causal, tipo de diagnostico y hash permitido para
outputs. Si falta hash de output no se fabrica `OutputRef`; el path queda solo
en manifest local clasificado.
Revalidacion 2026-05-27:
`agent-ref-task-autoprogramming-a35df67e3b46-g01` confirma el cierre local con
contexto `ref_only` resuelto por lectura/evidencia ACK; no abre otro Txx y
mantiene `worktree_ref` y `branch_ref` como refs opacas.
Test:
`go test -count=1 ./cmd/orquesta-guardian ./cmd/orquesta-server ./modulos/orquesta-autoprogramming`.
Backlog: `T233 guardian-local-diagnostic-ref-stability`.
```

```text
ID: RAIL-CAND-GUARDIAN-COMMAND-REF-TEMPLATE-001
Origen: scanner backlog 2026-05-27 decima pasada.
Casos: `guardianCommandRefV0` calcula `command_ref` con `phase|command`, donde
`command` ya puede contener shell expandido, rutas del candidato, `{state_dir}`
resuelto y comandos configurables. El resultado publico queda acoplado a texto
shell aunque T222 deba gobernarlo como efecto.
Decision aplicada 2026-05-27: `cmd/orquesta-guardian` publica `command_ref`
derivado de `phase`, `command_profile`, `template_ref` y `attempt_ref`; conserva
`command_template_changed` con evidence ref compacta para comandos no canonicos
y no usa shell expandido como identidad publica estable.
Test futuro:
`go test -count=1 ./cmd/orquesta-guardian ./cmd/orquesta-server ./modulos/orquesta-autoprogramming`.
Backlog: `T234 guardian-command-ref-template-profile`.
```

```text
ID: RAIL-CAND-GUARDIAN-REPAIR-PACKET-READ-001
Origen: scanner backlog 2026-05-27 decima pasada.
Casos: `guardianCodexRepairCommandV0` lee el repair packet completo con
`os.ReadFile(packetPath)` y lo mete en el prompt del reparador. Si el packet es
grande, no regular, sustituido por symlink/hardlink o no corresponde al intento
actual, la frontera break-glass puede lanzar Codex con contexto local no
gobernado.
Decision 2026-05-27: cerrado en `cmd/orquesta-guardian`; el launch Codex valida
el repair packet con `lstat/open/read` acotado, rechaza symlink/hardlink/no
regular/TOCTOU/tamano, exige schema, `redaction_level`, hash, attempt ref,
presupuesto y fase causal, y bloquea con `guardian_repair_packet_invalid` antes
de construir prompt o comando si falla.
Test:
`go test -count=1 ./cmd/orquesta-guardian ./cmd/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-codex-delivery`.
Backlog: `T235 guardian-repair-packet-read-budget-and-type`.
```

```text
ID: BACKLOG-SCAN-DOC-PATH-BUDGET-20260527-001
Fecha: 2026-05-27
Sintoma: el merge lease del scanner de backlog hashea documentos con lectura
completa desde rutas parseadas de `backlog_scan_doc`.
Campo: `cmd/orquesta-server/idle_self_improvement_backlog_merge_v0.go`.
Payload minimo: criterio `backlog_scan_doc:/tmp/no-canonico.md:line:1:sha256:x`,
`backlog_scan_doc:../fuera.md:line:1:sha256:x` o fichero canonico enorme/symlink.
Decision: cerrado por T240 con catalogo canonico de docs, path clean relativo,
lectura acotada, rechazo de control files/symlinks/no regulares y validacion de
paquetes que bloquea docs fuera del catalogo sin leer rutas locales.
Test futuro: mantener
`TestIdleSelfImprovementBacklogPlannerV0AcotaHashDocsASegurasV0` y
`TestIdleSelfImprovementBacklogPlannerV0RechazaBacklogScanDocNoCanonicoDelPacketV0`
dentro de `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`.
Estado: cerrado local 2026-05-27
```

```text
ID: FILE-BUDGET-20260527-002
Fecha: 2026-05-27
Sintoma: ficheros productivos de composicion y adaptador publico superan el
limite operativo de 300 lineas y siguen siendo candidatos naturales para nuevas
reglas.
Campo: `modulos/orquesta-server/supervisor_loop_v0.go`,
`modulos/orquesta-mcp/human_director_work_review_plan_tool_v0.go` y
`modulos/orquesta-mcp/autoprogramming_self_improvement_tool_v0.go`.
Payload minimo: `wc -l` muestra aproximadamente 780, 372 y 339 lineas.
Decision: abrir backlog T242 y T243 para partir por responsabilidad antes de
seguir anadiendo supervision residente o tools publicos.
Test futuro: `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server`
y `go test -count=1 ./modulos/orquesta-mcp`.
Estado: registrado, T242 y T243 cerrados localmente 2026-05-27.
Evidencia T242: `supervisor_loop_v0.go` queda acotado al tick residente; la
cobertura monolitica `supervisor_loop_v0_test.go` se reparte en tests de tick,
idle async, capacidad, blockers y helpers, manteniendo esos shards por debajo
de 300 lineas.
Assessment OrquestaV2 2026-05-27: el paquete
`task-autoprogramming-594d493666c4-g01` confirma T242 como cierre vigente con
contexto `ref_only` resuelto por evidencia local y sin nuevo owner programable.
Nota T243: los tools publicos MCP grandes quedan partidos por input flexible,
descriptor/ejecutor fino, advice/proyeccion y builders locales, sin cambiar
nombres publicos ni introducir runtime/proveedor en `orquesta-mcp`.
Evidencia T243: `go test -count=1 ./modulos/orquesta-mcp`.
Rework T252 2026-05-27: el limite de 300 lineas se aplico tambien a tests Go de
`orquesta-runtime-codex-delivery`; los shards de progress, delivery, review gate
y worktree quedan bajo 300 lineas con `go test -count=1
./modulos/orquesta-runtime-codex-delivery`.
```

```text
ID: FILE-BUDGET-20260527-003
Fecha: 2026-05-27
Sintoma: residuo productivo de `orquesta-app-codex-stack` sigue por encima del
limite operativo de 300 lineas tras el cierre parcial de T54.
Campo: `autoprogramming_bridge_v0.go`, `spec_external_context_v0.go`,
`spec_task_v0.go`, `app_change_ports_v0.go`,
`composite_decision_source_policy_v0.go`, `assessment_replan_source_v0.go` y
`run_supervisor_mcp_executor_v0.go`.
Payload minimo: `wc -l` muestra aproximadamente 341, 329, 317, 314, 313, 308 y
305 lineas respectivamente.
Decision: abrir backlog T244 para partir el residuo del stack Codex por
responsabilidad local antes de anadir nuevos puentes, specs, contexto externo o
ejecutores MCP/supervisor. Cierre local 2026-05-27: responsabilidades
separadas y ficheros Go productivos del modulo bajo 300 lineas.
Rework 2026-05-27: la revision
`task-ref-review-rework-task-autoprogramming-bbfaee8d0f40-g01-d8a1fe905f79cb6b486cecd73b06d1df`
solo revalida evidencia y contexto `ref_only`; no abre rail nuevo ni autoriza
crecimiento posterior del stack Codex.
Test futuro: `go test -count=1 ./modulos/orquesta-app-codex-stack`.
Estado: cerrado localmente
```

```text
ID: BACKLOG-SCAN-SECTION-ID-20260527-001
Fecha: 2026-05-27
Sintoma: bloques de `Escaneo backlog 2026-05-27` usan ordinales humanos
duplicados o fuera de orden como identificador visible de la evidencia.
Campo: encabezados de `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
y merge lease del scanner.
Payload minimo: dos bloques distintos con titulo `Escaneo backlog 2026-05-27
decimocuarta pasada` y nuevas tareas T241/T242-T243 bajo evidencias distintas.
Decision: T248 cerrado para servidor residente con `backlog_scan_ref` por
request/epoch documental y metadata `scan_entry_ref` para entradas de scanner
duplicadas o fuera de orden; el historico sigue parseable sin renumeracion.
Test futuro: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`.
Rework 2026-05-27: la correccion
`agent-ref-task-ref-review-rework-task-autoprogramming-ce0100f5ae08-g01-a522429e98460b11fc001393c9f07b9e`
mantiene este rail cerrado, no abre owner nuevo y resuelve `ref_only` mediante
lectura local/evidencia ACK.
Estado: cerrado 2026-05-27
```

```text
ID: FEDERATED-BACKLOG-EPOCH-SCOPE-20260527-001
Fecha: 2026-05-27
Sintoma: el scanner incluye todos los `source_path` del indice federado en
`BacklogScanDocs` aunque algunas fuentes esten en `quarantine` y no sean
ejecutables.
Campo: `cmd/orquesta-server/idle_self_improvement_federated_backlog_v0.go` y
`cmd/orquesta-server/idle_self_improvement_backlog_merge_v0.go`.
Payload minimo: documento legacy declarado con `estado: quarantine` cambia de
hash; el epoch del scanner cambia aunque `loadFederatedBacklogSectionsV0` no
programe tareas desde esa fuente.
Decision: abrir backlog T250 para separar fuentes federadas ejecutables del
epoch activo y conservar historicas/quarantine como evidencia no bloqueante.
Test futuro: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`.
Estado: cerrado 2026-05-27
Evidencia: el planner usa solo fuentes federadas ejecutables para
`BacklogScanDocs`, epoch y reservas; las fuentes `historico`, `stale` o
`quarantine` quedan como `federated_backlog_source_not_executable`.
```

```text
ID: FEDERATED-BACKLOG-EPOCH-SCOPE-RETRY-20260527-002
Fecha: 2026-05-27
Sintoma: nuevo burst de T250 reobserva el mismo rail con contexto obligatorio
`ref_only` y write-set cerrado al servidor y shards documentales.
Campo: scanner de backlog federado en `cmd/orquesta-server`.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-88a96698afb2-g01-b0935e0be19681ea514f9a2acacaff39`
de `request-ref-autoprogramming-backlog-t250-federated-backlog-epoch-scope-policy-acff973f`.
Decision: no abrir owner nuevo; resolver por cierre T250 vigente y evidencia
local de lectura/ACK. Las refs de worktree y branch se conservan opacas.
Test futuro: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`.
Estado: revalidado, cubierto
```

```text
ID: BACKLOG-TASK-OVERLAP-20260527-001
Fecha: 2026-05-27
Sintoma: scanners concurrentes pueden abrir dos Txx pendientes para la misma
frontera antes de que exista canonicalizacion ejecutable.
Campo: `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`, planner de
backlog y merge lease documental.
Payload minimo: T248 y T249 cubren identidad/orden de entradas de scanner;
T248 declara fusion posible con T249, pero ambas quedan como tareas pendientes
programables si el planner no calcula equivalencia de owner/alcance/frontera.
Decision: cerrado en T250 con firma compacta `backlog-task-overlap-*`,
canonical task ref por doc/linea/ref, fusion aditiva de criterios/tests/write-set
y reason code publico `backlog_task_overlap_canonicalization_required`.
Test ejecutado: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`.
Estado: cerrado
```

```text
ID: BACKLOG-TASK-OVERLAP-REWORK-20260527-001
Fecha: 2026-05-27
Sintoma: la correccion tras revision de T250 debe conservar la entrega cerrada
sin relanzar el agente padre ni ampliar el write-set.
Campo: ACK estricto, contexto `ref_only`, backlog, rail errors y duplicaciones.
Payload minimo: paquete
`agent-ref-task-ref-review-rework-task-autoprogramming-283ae6944821-g01-45cb26fe5d469e631710450335aa3a53`
con `required_ref_action=ack_evidence_required` y write-set cerrado.
Decision: no abrir owner nuevo; registrar rework documental acotado y cerrar por
lectura local/evidencia ACK. Los archivos de control no son artefactos de
producto y la evidencia focal sigue siendo la prueba T250 del servidor.
Test futuro: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`.
Estado: registrado, cubierto
```

```text
ID: BACKLOG-SCAN-REQUIRED-TEST-SCOPE-20260527-001
Fecha: 2026-05-27
Sintoma: una tarea documental de scanner recibe como prueba obligatoria
`go test -count=1 ./...` aunque el write-set real queda limitado a backlog,
rail errors y duplicaciones.
Campo: generacion de `required_tests` en paquetes de assessment/backlog scanner
y cierre por ACK estricto.
Payload minimo: `write_set` solo con documentos de backlog y
`required_tests=["go test -count=1 ./...","validar contexto required ref_only mediante lectura local, consulta al director o evidencia explicita"]`.
Decision: abrir backlog T251 para politica de alcance de pruebas obligatorias
en scanners documentales: conservar validacion global solo cuando la seccion o
el director la exijan, y emitir pruebas focales/documentales exactas cuando el
cambio no toca codigo productivo.
Test futuro: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`.
Estado: cerrado 2026-05-27. T251 acota `required_tests` de scanners
documentales, preserva tests declarados por seccion y deja trazabilidad en
`required_test_origin:*`/`required_test_scope_policy:*`. Rework de revision:
sincronizado con backlog y duplicaciones; contexto `ref_only` resuelto por
lectura local/evidencia ACK.
```

```text
ID: BACKLOG-DUPLICATE-TASK-ID-READ-20260527-001
Fecha: 2026-05-27
Sintoma: el backlog vivo contiene varias secciones `## T250` con objetivos y
alcances distintos; un lector que use solo el numero humano puede cerrar o
encolar la instancia equivocada.
Campo: parser del backlog, cola de autoprogramacion, cierre por ACK y merge
lease documental.
Payload minimo: cuatro encabezados `## T250` para fronteras distintas
(`federated-backlog-epoch-scope-policy`,
`runtime-codex-delivery-progress-source-file-split`,
`backlog-proposal-deduplication-fingerprint` y
`backlog-task-overlap-canonical-merge-policy`).
Decision: abrir backlog T252 para read model con `task_instance_ref` estable,
reason publico `backlog_duplicate_task_id_ambiguous` y resolucion aditiva por
alias/cobertura sin renumerar historico.
Test futuro:
`go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-server ./cmd/orquesta-server`.
Estado: cerrado 2026-05-27; planner expone instancias y bloquea Txx ambiguo con
`backlog_duplicate_task_id_ambiguous`.
Rework de revision 2026-05-27:
`agent-ref-task-ref-review-rework-task-autoprogramming-9ad7b5061f27-g01-d2d05e084cc48aa9566f43b01d7d8f5e`
revalida la entrega T252 sin relanzar el agente padre; el contexto `ref_only`
queda resuelto por evidencia explicita en ACK y la prueba requerida es
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`.
```

```text
ID: BACKLOG-SCANNER-CANONICAL-PREFLIGHT-20260527-001
Fecha: 2026-05-27
Sintoma: scanner posterior abre backlog nuevo para una frontera ya cubierta por
canonicalizacion previa.
Campo: planner de backlog, merge lease documental y secciones `## Txx`.
Payload minimo: T140 cerro canonicalizacion de solapes con alias y
`duplicate_backlog_task`; T250 vuelve a abrir una tarea equivalente para
scanners concurrentes sin pasar por un preflight que consulte canon/alias/cierre
antes de escribir el nuevo Txx.
Decision: abrir backlog T251 para preflight canonico del scanner: consultar
tareas cerradas/vigentes/pendientes, declarar cobertura o alias/fusion aditiva,
y bloquear con `backlog_scanner_canonical_preflight_required` si hace falta
rebase o decision del director.
Test futuro:
`go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-server ./cmd/orquesta-server`.
Estado: cerrado 2026-05-27. T251 aplica preflight canonico en el planner,
publica indice/fingerprint y reason `backlog_scanner_canonical_preflight_required`
para evitar otro `## Txx` equivalente. Rework de revision: sincronizado con
backlog y duplicaciones; contexto `ref_only` resuelto por lectura local/evidencia
ACK sin relanzar agente padre.
```

```text
ID: FILE-BUDGET-20260527-005
Fecha: 2026-05-27
Sintoma: el caso de uso productivo de apagado controlado supera el limite
operativo de 300 lineas y es frontera natural para nuevos reason codes de
shutdown.
Campo: `modulos/orquesta-server-shutdown/shutdown_v0.go`.
Payload minimo: `wc -l` muestra 319 lineas; el fichero mezcla autorizacion del
requester, lectura de cola, seleccion de targets, checkpoint no forzado,
escritura de control, refresh/supervision y resumen final.
Decision: registrar candidato y fusionarlo con el owner canonico T256
`server-shutdown-usecase-file-split` cuando esa entrada posterior este visible;
no programar dos splits separados para la misma frontera.
Test futuro:
`go test -count=1 ./modulos/orquesta-server-shutdown ./cmd/orquesta-server`.
Estado: cerrado 2026-05-27 por T256
Rework correctivo 2026-05-27: validado por
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-ced6b146bcf5b10e5253ef166816de66`;
no abrir tarea separada para la variante `before-growth`.
Rework de evaluacion 2026-05-27: validado por
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-f878e21cfa5e67e7481d20a5f9bd24c5`;
mantener la variante `before-growth` fusionada con T256 canonico.
Rework de evaluacion 2026-05-27: validado por
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-3d04cc0bcd1f59b871869ec824d2c996`;
mantener la variante `before-growth` fusionada con T256 canonico.
Rework de evaluacion 2026-05-27: validado por
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-46d0d95f62343cc1e51b9ca398f83f84`;
mantener la variante `before-growth` fusionada con T256 canonico.
Rework de evaluacion 2026-05-27: validado por
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-33ca4ea060412afa902e83e3765c1d23`;
mantener la variante `before-growth` fusionada con T256 canonico.
Rework de reemplazo 2026-05-27: validado por
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-b6686ba67b0d901948d8ce067d8771ec`;
mantener la variante `before-growth` fusionada con T256 canonico.
Rework de evaluacion 2026-05-27: validado por
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-bff7aa189e7e24662071cf55dffcb95f`;
mantener la variante `before-growth` fusionada con T256 canonico.
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-003
Fecha: 2026-05-27
Sintoma: scanner de automejora vuelve a recibir contexto obligatorio `ref_only`
con write-set documental cerrado y huecos ya visibles en el backlog.
Campo: backlog scanner, ACK estricto y shards documentales de backlog.
Payload minimo: paquete con `required_ref_action=ack_evidence_required`,
write-set limitado a backlog/rail errors/duplicaciones y busquedas que ya
encuentran T44, T249, T250, T251, T252, T254 y T255 como owners pendientes.
Decision: no abrir otro Txx; registrar pasada documental de no-op cubierto,
resolver `ref_only` mediante lectura local/evidencia ACK y conservar refs de
worktree/branch como opacas.
Test futuro: usar pruebas de los owners T44/T249/T250/T251/T252/T254/T255; esta
entrada solo evita duplicar backlog programable.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-004
Fecha: 2026-05-27
Sintoma: nueva pasada del scanner de automejora repite contexto obligatorio
`ref_only`, write-set cerrado a los tres shards documentales y huecos ya
registrados.
Campo: backlog scanner, ACK estricto, required tests y shards documentales de
backlog.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-3ddbc1385ac2-g01-b0d62e2841ee6bd1a73887a370464c5e`
con `required_ref_action=ack_evidence_required`, write-set limitado a
backlog/rail errors/duplicaciones y busquedas que ya encuentran T44, T249,
T250, T251, T252, T254, T255 y T256 como owners pendientes.
Decision: no abrir otro Txx; registrar pasada documental de no-op cubierto,
resolver `ref_only` mediante lectura local/evidencia ACK y conservar
`worktree_ref`/`branch_ref` como refs opacas.
Test futuro: usar pruebas de los owners T44/T249/T250/T251/T252/T254/T255/T256;
esta entrada solo evita duplicar backlog programable.
Estado: registrado, cubierto por owners pendientes
```
```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-013
Fecha: 2026-05-27
Sintoma: retry d0cf1e del scanner de automejora repite contexto obligatorio
`ref_only`, write-set documental cerrado y huecos ya visibles en backlog.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete
`agent-ref-assessment-task-autoprogramming-350b93e475a8-g01-7b6283b42d81ab988ec874761f9ce9e0`
del retry
`request-ref-autoprogramming-backlog-scanner-7e8a6ae9-retry-d0cf1ea77c2b6898418f56ac96139dcb5cb606948a8e3adc89ed26b4a9b1f02c`,
con `required_ref_action=ack_evidence_required`, write-set limitado a backlog,
rail errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250,
T251, T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx; registrar no-op cubierto, resolver `ref_only`
mediante lectura local/evidencia ACK y conservar `worktree_ref`/`branch_ref`
como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```

```text
ID: BACKLOG-SCAN-COVERED-NOOP-20260527-038
Fecha: 2026-05-27
Sintoma: retry 6ed415 burst 002 del scanner de automejora repite contexto
obligatorio `ref_only`, write-set documental cerrado, prueba global obligatoria
y backlog degradado ya cubierto por owners pendientes antes de programar codigo.
Campo: backlog scanner, ACK estricto, retry de request y shards documentales.
Payload minimo: paquete `agent-ref-task-autoprogramming-6050b74fc6c4-g01` del
retry
`request-ref-autoprogramming-backlog-scanner-15eeecb9-retry-6ed4157ea15c63315787f6758835ef866624b96c293520bade4d4fac7bbd3f1d`,
con
`correlation_id=corr-request-ref-autoprogramming-backlog-scanner-15eeecb9-retry-6ed4157ea15c63315787f6758835ef866624b96c293520bade4d4fac7bbd3f1d-burst-002`,
`required_ref_action=ack_evidence_required`, write-set limitado a backlog, rail
errors y duplicaciones, y busquedas que ya encuentran T44, T249, T250, T251,
T252, T253, T254, T255, T256, T257 y T258.
Decision: no abrir otro Txx ni programar codigo desde el scanner; registrar
no-op cubierto, resolver `ref_only` mediante lectura local/evidencia ACK y
conservar `worktree_ref`/`branch_ref` como refs opacas.
Test futuro: usar pruebas de los owners
T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258; esta entrada solo evita
duplicar backlog programable desde un retry equivalente.
Estado: registrado, cubierto por owners pendientes
```
