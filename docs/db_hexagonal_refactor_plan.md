# Plan De Refactor De `db` Hacia Hexagonalidad

## Estado 2026-04-16

Progreso ya aterrizado en commits pequeños para no pisarse:

- `c3b3541` mueve la validación y normalización de overrides de gobernanza a `gobernanzapolicy`
- `cbe90e9` mueve la precedencia/capas de overrides (`rol -> proyecto -> agente`) a `gobernanzapolicy`
- `058dba9` mueve la policy de frescura TTL de presupuestos a `sesionesapp`
- `0d1f30b` mueve la detección de snapshots que aportan cuota a `sesionesapp`
- `a8778d4` mueve el scoring puro de candidatos de presupuesto canónico a `sesionesapp`
- último corte de esta tanda: mover la evaluación pura de presupuesto y ratio/handoff a `sesionesapp`
- corte nuevo en planner: mover la policy pura de cupo/carga/selección de proyecto automático a `planificadorpolicy`
- corte adicional en planner: mover la policy de filtro de agentes permitidos en autobootstrap a `planificadorpolicy`
- corte adicional en planner: mover la heurística pura de scoring/selección de tarea libre a `planificadorpolicy`
- corte adicional en planner: mover la priorización de agentes planificables y el consumo de capacidad por pool a `planificadorpolicy`
- corte adicional en planner: mover la resolución pura de proyecto preferente (activo/pausado/automático) a `planificadorpolicy`
- corte adicional en planner: mover la detección de operador manual fuera de flota a `planificadorpolicy`
- corte adicional en planner: mover la detección de pool local compartido a `planificadorpolicy`
- corte adicional en planner: mover la ventana de gracia de recuperación de tarea huérfana a `planificadorpolicy`
- corte adicional en planner: mover la política de preservación de tarea huérfana enfocada (frente acotado) a `planificadorpolicy`
- corte adicional en runtime: mover compactación/estado de prompt bootstrap a `runtimepolicy`
- corte adicional en runtime: mover la compactación de transcript pending a `runtimepolicy`
- corte adicional en runtime: mover compactación/síntesis de metadata de `runtime_handle` en `runtimepolicy` (`CompactRuntimeHandleMetadata`)
- corte adicional en runtime: mover heurísticas `runtimeHandleLooksLikeCodexCLIRef` y `runtimeHandleLooksLikeCodexCommand` a `runtimepolicy`
- corte adicional en runtime: mover scoring de contexto de entrega de `runtime_handle` (`RuntimeHandleDeliveryContextScore`) a `runtimepolicy`
- corte adicional en runtime: mover política canónica TMUX y construcción de ref TMUX de `runtime_handle` (`runtimeHandleEsTMUXCanonico`, `runtimeHandleTMUXSessionRef`) a `runtimepolicy`
- corte adicional en runtime: mover policy legacy de control plane de `runtime_handle` (`RuntimeHandleIsLegacyControlPlane`, llamada desde `runtimeHandleEsCandidatoLegacyATMUX`) a `runtimepolicy`
- corte adicional en autonomia: mover la política pura de score de rol de supervisor operativo a `autonomiapolicy`
- `3b6d513` completo el grupo previo en runtime (`runtimepolicy`) para decisiones de `runtime_handle` de borde
- `de6239a` completa el corte de policy legacy TMUX (`RuntimeHandleUsaLegacyCLITMUXPreferred`, `RuntimeHandleUsaLegacyProcessPTY`) en `runtimepolicy`
- corte adicional en runtime: dejar `runtimesapp` con wrapper fino en `runtimeHandleEsCandidatoLegacyATMUX` delegando en `runtimepolicy.RuntimeHandleIsLegacyControlPlane`
- corte adicional en runtime: mover la regla de estados que requieren `SyncSupervisedRuntimeHandle` fresca a `runtimepolicy.RuntimeHandleNeedsFreshSync`, dejando `runtimesapp` como delegador
- corte adicional en runtime: mover la detección de falta de sesión tmux válida (`TMUXSessionExistsMetadata`) a `runtimepolicy.RuntimeHandleTMUXSessionMissing`
- corte adicional en runtime: mover la comprobación de coincidencia runtime/sesión (`RuntimeHandleMatchesRuntime`) y extracción de `driver` (`RuntimeHandleDriver`) a `runtimepolicy`
- `491c264` mueve el cálculo de capacidad/autonomía supervisor a lógica dedicada desde `planificador`, reduciendo decisión en `db/planificador.go`
- `9f51a8f` mueve resolución de ruta/proyecto/worktree hacia `coordinacion` y reduce ensamblaje en `db`
- `e1aa278` preserva el comportamiento de filtrado de `worktrees` al trasladar parte de la coherencia a `coordinacion`
- `71bdf8e` extrae side effects de transición de tareas/asignaciones a coordinadores internos de `db` para wrappers más finos

Estado del frente seguro:

- `db/controlplane_entities.go` sigue siendo hotspot ajeno y no debe tocarse sin reasignación
- `db/asignaciones.go` fue endurecido con seam de transición más finito, pero aún conserva decisiones funcionales asociadas a transición de tareas
- el frente `presupuestos/governanza` ha permitido sacar policy pura fuera de `db` sin tocar `cmd/`
- en `presupuestos`, `db` ya conserva sobre todo carga/configuración/persistencia; la policy de TTL, cuota, scoring y evaluación efectiva vive fuera

## Inventario Actual

### 1. Ya Saneado

- tests de frontera de `db` y allowlist de imports internos en [db/test_architecture_dependencies_test.go](/home/alberto/Trabajo/orquesta/db/test_architecture_dependencies_test.go)
- selección de ruta/worktree y coherencia operativa movida en buena parte a `coordinacion`
- policy de sesiones extraída a `sesionesapp` para asignación de pool y presupuesto efectivo
- policy de presupuesto movida a `sesionesapp`: TTL, detección de cuota, scoring canónico y evaluación de handoff
- policy de dependencias de tareas movida a `tareaspolicy`
- policy pura de selección de proyecto automático del planner movida a `planificadorpolicy`
- policy de pertenencia al pool de autobootstrap del planner movida a `planificadorpolicy`
- policy pura de scoring/ordenación de tarea libre del planner movida a `planificadorpolicy`
- policy de ranking final de agentes planificables movida a `planificadorpolicy`
- policy de resolución de proyecto preferente del planner movida a `planificadorpolicy` (`activo > pausado > automático`)
- policy de detección de operador manual fuera de flota del planner movida a `planificadorpolicy`
- policy de detección de pool local compartido del planner movida a `planificadorpolicy`
- policy de ventana de gracia de recuperación de tarea huérfana movida a `planificadorpolicy`
- policy de preservación de tarea huérfana para frente acotado movida a `planificadorpolicy`
- policy de gobernanza movida a `gobernanzapolicy` para catálogo, validación de overrides y precedencia por capas
- policy de compactación/síntesis de metadata de `runtime_handle` movida a `runtimepolicy` (`CompactRuntimeHandleMetadata`) para minimizar policy en `db/controlplane_entities.go`
- policy canónica TMUX y ref de sesión de `runtime_handle` movida a `runtimepolicy` (`RuntimeHandleEsTMUXCanonico`, `RuntimeHandleTMUXSessionRef`)
- policy legacy de tmux CLI movida desde `db/controlplane_entities.go` a `runtimepolicy` (`RuntimeHandleUsaLegacyCLITMUXPreferred`, `RuntimeHandleUsaLegacyProcessPTY`)
- policy legacy de control plane (`RuntimeHandleIsLegacyControlPlane`) movida desde `runtimesapp` hacia `runtimepolicy` para candidato legacy TMUX; `runtimeHandleEsCandidatoLegacyATMUX` queda como wrapper delegador
- policy de estado operativo que exige re-sincronización fresca de `runtime_handle` delegada a `runtimepolicy.RuntimeHandleNeedsFreshSync`
- policy de sesión tmux válida ausente (`RuntimeHandleTMUXSessionMissing`) delegada a `runtimepolicy`
- policy de match de runtime/sesión y extracción de `driver` (`RuntimeHandleMatchesRuntime`, `RuntimeHandleDriver`) delegada a `runtimepolicy`
- policy de scoring de contexto de entrega de `runtime_handle` movida a `runtimepolicy` para selector explícito de `runtime_order`
- policy de skills movida a `skillspolicy`
- policy de compactación de prompt de bootstrap movida a `runtimepolicy`
- policy de compactación de texto pendiente de transcript en runtime movida a `runtimepolicy`
- policy de ordenación por rol en selección de supervisores operativos movida a `autonomiapolicy`

### 2. Aceptable Con Wrappers

- [db/presupuestos_sesion.go](/home/alberto/Trabajo/orquesta/db/presupuestos_sesion.go) sigue exponiendo API pública de `db`, pero ya actúa sobre todo como wrapper de carga/configuración y adaptación de tipos
- [db/governance_overrides.go](/home/alberto/Trabajo/orquesta/db/governance_overrides.go) mantiene repositorio, consultas y aplicación de overrides cargados, con validación y capas ya delegadas fuera
- [db/reglas.go](/home/alberto/Trabajo/orquesta/db/reglas.go) conserva wrappers y CRUD de catálogo; la parte puramente funcional ya está bastante fuera, pero aún no es repositorio mínimo
- [db/proyectos.go](/home/alberto/Trabajo/orquesta/db/proyectos.go), [db/worktrees.go](/home/alberto/Trabajo/orquesta/db/worktrees.go) y [db/project_context.go](/home/alberto/Trabajo/orquesta/db/project_context.go) están más delgados, pero siguen combinando queries con algo de ensamblaje operativo

### 3. Todavía No Hexagonal

- [db/controlplane_entities.go](/home/alberto/Trabajo/orquesta/db/controlplane_entities.go) sigue siendo el hotspot principal y mezcla persistencia con policy de runtime/bootstrap/receipt/mailbox
- [db/planificador.go](/home/alberto/Trabajo/orquesta/db/planificador.go) sigue cargando demasiada decisión de trabajo/autonomía, aunque varias heurísticas puras de cupo/carga/priorización de proyecto ya salieron fuera
- [db/autonomia_proyecto.go](/home/alberto/Trabajo/orquesta/db/autonomia_proyecto.go) y [db/autonomia_supervisor_operativo.go](/home/alberto/Trabajo/orquesta/db/autonomia_supervisor_operativo.go) siguen siendo parte del frente no saneado
- [db/runtimes.go](/home/alberto/Trabajo/orquesta/db/runtimes.go), [db/runtime_bootstrap_prompt.go](/home/alberto/Trabajo/orquesta/db/runtime_bootstrap_prompt.go) y [db/runtime_transcript.go](/home/alberto/Trabajo/orquesta/db/runtime_transcript.go) todavía forman parte del frente runtime no vaciado
- [db/tareas.go](/home/alberto/Trabajo/orquesta/db/tareas.go) y [db/asignaciones.go](/home/alberto/Trabajo/orquesta/db/asignaciones.go) han mejorado, pero no están completamente reducidos a persistencia pura

### Conclusión Operativa

- no puede decirse aún que `db` sea completamente hexagonal
- sí puede decirse que varios slices ya están claramente encaminados y protegidos por frontera arquitectónica
- el siguiente salto real ya no es seguir raspando seams pequeños, sino vaciar `runtime/controlplane` y `planner`

## Objetivo

Convertir `db/` en una capa de persistencia y soporte transaccional, no en una capa de decisión de negocio u orquestación.

El objetivo no es "cambiar SQLite por otra base mañana", sino dejar `db` en una forma que:

- permita persistir sobre distintos backends sin reescribir policy,
- reduzca la lógica de negocio incrustada en SQL/repositorios,
- y permita a varios agentes trabajar en paralelo sin pisarse.

## Estado Actual

`db/` hoy mezcla dos responsabilidades:

1. Persistencia:
- schema
- migraciones
- queries
- locking
- hot indexes
- mapeo fila <-> entidad

2. Lógica operativa histórica:
- leases bootstrap/runtime
- mailbox/send_instruction/receipt
- partes del planner/autonomía
- handoff policy
- selección de runtime/session/worktree
- reglas de transición de tareas y sesiones

Esto crea una brecha con la arquitectura hexagonal del proyecto.

## Restricción De Trabajo

Mientras dure esta migración:

- no entra lógica nueva de negocio en `db/`
- solo se permiten fixes mínimos de comportamiento roto donde la lógica ya existe ahí
- toda policy nueva debe vivir fuera de `db/`, idealmente en:
  - `runtimesapp`
  - `capacidadapp`
  - `tareasapp`
  - helpers puros fuera de `db`

## Definición De `db` Sana

`db/` debe quedar limitada a:

- repositorios CRUD y queries
- persistencia de eventos/estado
- transacciones
- locking
- schema y migraciones
- índices/vistas/hot paths
- validaciones estrictamente ligadas al almacenamiento

`db/` no debe decidir:

- a qué agente despachar
- cuándo una evidencia es "útil" por policy de negocio compleja
- qué lease bootstrap priorizar por policy de runtime, salvo criterios mínimos persistentes
- cómo resolver autonomía/planning
- cómo gobernar mailboxes, nudge, session_resume o handoff a nivel de negocio

## Hotspots Reales

Los hotspots productivos más importantes son:

- [controlplane_entities.go](/home/alberto/Trabajo/orquesta/db/controlplane_entities.go)
- [sesiones.go](/home/alberto/Trabajo/orquesta/db/sesiones.go)
- [planificador.go](/home/alberto/Trabajo/orquesta/db/planificador.go)
- [proyectos.go](/home/alberto/Trabajo/orquesta/db/proyectos.go)
- [runtimes.go](/home/alberto/Trabajo/orquesta/db/runtimes.go)
- [tareas.go](/home/alberto/Trabajo/orquesta/db/tareas.go)
- [runtime_bootstrap_prompt.go](/home/alberto/Trabajo/orquesta/db/runtime_bootstrap_prompt.go)
- [runtime_transcript.go](/home/alberto/Trabajo/orquesta/db/runtime_transcript.go)
- [worktrees.go](/home/alberto/Trabajo/orquesta/db/worktrees.go)

## Estrategia General

No hacer un rediseño masivo. Hacer estrangulamiento por seams.

Secuencia:

1. Congelar la frontera
2. Añadir tests de arquitectura
3. Extraer policy runtime/bootstrap fuera de `db`
4. Extraer planner/autonomía fuera de `db`
5. Extraer selección de sesión/proyecto/worktree fuera de `db`
6. Extraer reglas de tareas/asignación fuera de `db`
7. Dejar `db` como repositorio puro

## Principio De Refactor

El patrón correcto es este:

1. mantener la API pública existente de `db` al principio
2. mover la decisión a un servicio/helper puro fuera de `db`
3. dejar el código de `db` como wrapper fino:
   - cargar estado
   - delegar policy
   - persistir resultado
4. cuando el call graph ya use el servicio nuevo, adelgazar o eliminar wrapper

## Tracks Paralelos

Esto sí puede repartirse entre varios agentes, pero con write-set fijo por track.

### Track 1. Runtime Bootstrap / Receipt / Mailbox

Objetivo:
- sacar la policy de leases, mailbox y receipt de `db`
- dejar en `db` solo repositorios y persistencia

Archivos actuales a revisar:
- [controlplane_entities.go](/home/alberto/Trabajo/orquesta/db/controlplane_entities.go)
- [runtime_bootstrap_prompt.go](/home/alberto/Trabajo/orquesta/db/runtime_bootstrap_prompt.go)
- [runtime_transcript.go](/home/alberto/Trabajo/orquesta/db/runtime_transcript.go)
- [runtimes.go](/home/alberto/Trabajo/orquesta/db/runtimes.go)

Destino ideal:
- policy en `runtimesapp`
- `db` solo como repositorio

### Track 2. Sesiones / Proyecto / Worktree

Objetivo:
- dejar la elección de ruta/sesión/worktree fuera de `db`
- mantener en `db` solo carga/persistencia

Archivos:
- [sesiones.go](/home/alberto/Trabajo/orquesta/db/sesiones.go)
- [proyectos.go](/home/alberto/Trabajo/orquesta/db/proyectos.go)
- [worktrees.go](/home/alberto/Trabajo/orquesta/db/worktrees.go)
- [project_query_helpers.go](/home/alberto/Trabajo/orquesta/db/project_query_helpers.go)

Destino ideal:
- queries/repos en `db`
- selection policy fuera

### Track 3. Tareas / Asignación / Reglas

Objetivo:
- sacar reglas de transición de negocio fuera de `db`
- dejar invariantes mínimas y persistencia

Archivos:
- [tareas.go](/home/alberto/Trabajo/orquesta/db/tareas.go)
- [asignaciones.go](/home/alberto/Trabajo/orquesta/db/asignaciones.go)
- [reglas.go](/home/alberto/Trabajo/orquesta/db/reglas.go)

Destino ideal:
- policy en `tareasapp`

### Track 4. Planner / Autonomía

Objetivo:
- que `db` no decida trabajo ni autonomía
- que solo exponga backlog, candidatos, estado y locks

Archivos:
- [planificador.go](/home/alberto/Trabajo/orquesta/db/planificador.go)
- [autonomia_proyecto.go](/home/alberto/Trabajo/orquesta/db/autonomia_proyecto.go)
- [autonomia_supervisor_operativo.go](/home/alberto/Trabajo/orquesta/db/autonomia_supervisor_operativo.go)

Destino ideal:
- policy en `capacidadapp` o servicios del control plane fuera de `db`

### Track 5. Infraestructura Pura

Objetivo:
- estabilizar la base de persistencia sin tocar policy

Archivos:
- [backend.go](/home/alberto/Trabajo/orquesta/db/backend.go)
- [backend_sqlite.go](/home/alberto/Trabajo/orquesta/db/backend_sqlite.go)
- [backend_postgres.go](/home/alberto/Trabajo/orquesta/db/backend_postgres.go)
- [backend_mysql.go](/home/alberto/Trabajo/orquesta/db/backend_mysql.go)
- [sqlwrap.go](/home/alberto/Trabajo/orquesta/db/sqlwrap.go)
- schema y migraciones bajo `db/`

## Regla Para No Pisarse

No se reparte "por lo que haya que tocar". Se reparte por archivo dueño.

Reglas:

- un solo agente puede tocar [controlplane_entities.go](/home/alberto/Trabajo/orquesta/db/controlplane_entities.go)
- un solo agente puede tocar [sesiones.go](/home/alberto/Trabajo/orquesta/db/sesiones.go)
- un solo agente puede tocar [planificador.go](/home/alberto/Trabajo/orquesta/db/planificador.go)
- cualquier cambio transversal debe hacerse creando archivo nuevo fuera de `db` y dejando wrappers finos
- nadie toca un hotspot ajeno sin reasignación explícita

## Orden Recomendado

### Fase 1. Congelación Y Tests De Arquitectura

Hacer:
- endurecer [test_architecture_test.go](/home/alberto/Trabajo/orquesta/db/test_architecture_test.go)
- documentar frontera permitida de `db`

Resultado esperado:
- no puede volver a entrar policy nueva a `db`

### Fase 2. Runtime Bootstrap / Receipt

Hacer:
- extraer decisiones de lease/receipt/mailbox a helpers/servicios fuera de `db`
- dejar `db` con funciones de cargar/guardar estado

Resultado esperado:
- el núcleo de orquestación deja de depender del hotspot más peligroso

### Fase 3. Planner / Autonomía

Hacer:
- quitar asignación/planning de `db`
- dejar solo consultas de candidatos y persistencia de decisiones

Resultado esperado:
- la decisión de trabajo vive fuera de la base

### Fase 4. Sesiones / Worktrees / Proyectos

Hacer:
- mover la selección efectiva fuera de `db`

Resultado esperado:
- `db` deja de decidir contexto operativo

### Fase 5. Tareas / Reglas

Hacer:
- mover transición de negocio de tareas a `tareasapp`

Resultado esperado:
- `db` mantiene solo almacenamiento e invariantes

## Criterios De Aceptación Por Track

### Runtime Bootstrap / Receipt

- `db` no decide por sí sola si una evidencia es útil
- `db` no aplica policy de `session_resume`, `mailbox_only`, `nudge`, `handoff` más allá de persistir
- tests del núcleo siguen verdes
- `db` no decide canonicalidad TMUX ni referencia TMUX final; esas decisiones viven en `runtimepolicy`
- `db` solo delega decisiones de `runtime_handle` puras desde `controlplane_entities.go`; no se deben introducir cambios de estado en estas extracciones

Checklist próximo hito runtime/bootstrap:
- [ ] Auditar `db/controlplane_entities.go` para políticas de bootstrap/start que aún empalmen decisión y persistencia
- [ ] Extraer una sola regla autocontenida por corte (sin tocar estado) a `runtimepolicy` o app de runtime
- [ ] Mantener contrato existente de `db` (carga/escritura + hot paths) y migrar solo policy pura
- [ ] Actualizar frontera de imports/allowlist si aparece un nuevo seam externo permitido en `runtimepolicy`
- [ ] Correr test de frontera `TestDBNoIntroduceDependenciasInternasFueraDeLaFronteraPermitida` tras cada cambio de frontera

### Planner / Autonomía

- `db` no asigna tareas automáticamente por policy
- `db` expone candidatos y persiste decisiones tomadas fuera

### Sesiones / Worktrees / Proyectos

- `db` no decide ruta efectiva por policy compleja
- `db` carga worktree/sesión/proyecto y persiste cambios

### Tareas / Reglas

- `db` no decide transición funcional salvo invariantes mínimas

## Qué No Hacer

- no dividir [controlplane_entities.go](/home/alberto/Trabajo/orquesta/db/controlplane_entities.go) entre varios agentes a la vez
- no meter nuevas heurísticas en `db` “porque es más rápido”
- no rehacer todo en una sola PR
- no romper tests de orquestación para ganar pureza teórica

## Riesgo Principal

El principal riesgo no es técnico sino de coordinación:

- si varios agentes tocan el mismo hotspot, el merge será inestable
- si no se fijan write-sets, se reintroducirá lógica en `db`
- si no se añaden tests de frontera, la deuda volverá

## Reparto Sugerido Para Agentes

### Agente A. Runtime Bootstrap / Receipt

Puede mirar:
- [controlplane_entities.go](/home/alberto/Trabajo/orquesta/db/controlplane_entities.go)
- [runtime_bootstrap_prompt.go](/home/alberto/Trabajo/orquesta/db/runtime_bootstrap_prompt.go)
- [runtime_transcript.go](/home/alberto/Trabajo/orquesta/db/runtime_transcript.go)
- [runtimes.go](/home/alberto/Trabajo/orquesta/db/runtimes.go)

No debe tocar:
- planner
- tareas
- sesiones salvo lectura

### Agente B. Planner / Autonomía

Puede mirar:
- [planificador.go](/home/alberto/Trabajo/orquesta/db/planificador.go)
- [autonomia_proyecto.go](/home/alberto/Trabajo/orquesta/db/autonomia_proyecto.go)
- [autonomia_supervisor_operativo.go](/home/alberto/Trabajo/orquesta/db/autonomia_supervisor_operativo.go)

No debe tocar:
- `controlplane_entities.go`

### Agente C. Sesiones / Proyecto / Worktree

Puede mirar:
- [sesiones.go](/home/alberto/Trabajo/orquesta/db/sesiones.go)
- [proyectos.go](/home/alberto/Trabajo/orquesta/db/proyectos.go)
- [worktrees.go](/home/alberto/Trabajo/orquesta/db/worktrees.go)
- [project_query_helpers.go](/home/alberto/Trabajo/orquesta/db/project_query_helpers.go)

### Agente D. Tareas / Reglas

Puede mirar:
- [tareas.go](/home/alberto/Trabajo/orquesta/db/tareas.go)
- [asignaciones.go](/home/alberto/Trabajo/orquesta/db/asignaciones.go)
- [reglas.go](/home/alberto/Trabajo/orquesta/db/reglas.go)

### Agente E. Infraestructura / Tests De Arquitectura

Puede mirar:
- [test_architecture_test.go](/home/alberto/Trabajo/orquesta/db/test_architecture_test.go)
- [backend.go](/home/alberto/Trabajo/orquesta/db/backend.go)
- [backend_sqlite.go](/home/alberto/Trabajo/orquesta/db/backend_sqlite.go)
- [backend_postgres.go](/home/alberto/Trabajo/orquesta/db/backend_postgres.go)
- [backend_mysql.go](/home/alberto/Trabajo/orquesta/db/backend_mysql.go)
- schema relacionados

## Prompt Para Pasarle A Un Agente De Apoyo

```text
Trabajas en /home/alberto/Trabajo/orquesta.

Objetivo:
Ayudar a refactorizar `db/` hacia una capa más hexagonal sin aumentar la brecha arquitectónica.

Reglas obligatorias:
- No metas lógica nueva de negocio en `db/`.
- Si ves que una policy debe vivir fuera de `db`, propón o crea el seam fuera de `db`.
- No toques archivos fuera de tu write-set.
- No reviertas cambios ajenos.
- Si necesitas editar, usa apply_patch.
- Antes de proponer mover algo, identifica qué parte es persistencia y qué parte es policy.

Tu misión en este track es:
- [AQUÍ PEGAR EL TRACK CONCRETO]

Write-set permitido:
- [AQUÍ PEGAR ARCHIVOS PERMITIDOS]

No puedes tocar:
- /home/alberto/Trabajo/orquesta/db/controlplane_entities.go si no está en tu write-set
- cualquier hotspot asignado a otro agente

Definición de éxito:
- el cambio reduce lógica de decisión dentro de `db`
- no rompe los tests del slice afectado
- no añade nuevas heurísticas de orquestación dentro de `db`

Entrega esperada:
1. Diagnóstico breve del problema del slice
2. Qué parte es persistencia y qué parte es policy
3. Cambio concreto propuesto
4. Archivos modificados
5. Tests ejecutados
6. Riesgos abiertos
```

## Prompt Específico Recomendado Para El Primer Agente

```text
Trabajas en /home/alberto/Trabajo/orquesta.

Quiero que te centres solo en el track Runtime Bootstrap / Receipt para ayudar a vaciar policy de `db`.

Objetivo:
Identificar seams para sacar fuera de `db` la policy de:
- bootstrap lease
- mailbox durable
- receipt útil
- session_resume / handoff / start bootstrap

Archivos permitidos:
- /home/alberto/Trabajo/orquesta/db/controlplane_entities.go
- /home/alberto/Trabajo/orquesta/db/runtime_bootstrap_prompt.go
- /home/alberto/Trabajo/orquesta/db/runtime_transcript.go
- /home/alberto/Trabajo/orquesta/db/runtimes.go
- tests asociados en /home/alberto/Trabajo/orquesta/db/*test.go

No puedes tocar:
- planner
- tareas
- sesiones
- cmd/

Lo que necesito:
1. Lista de funciones que hoy mezclan persistencia y policy
2. Propuesta de seams nuevos fuera de `db`
3. Refactor mínimo de uno de esos seams sin romper tests
4. Tests del slice afectados en verde

Regla:
No metas lógica nueva en `db`. Si tienes que añadir comportamiento, sácalo a helper o servicio fuera de `db` y deja `db` como wrapper fino.
```

## Estado Final Deseado

Se considerará que `db` está razonablemente arreglada cuando:

- no entre policy nueva en `db`
- los hotspots estén troceados por dominio
- la mayor parte de la lógica de orquestación viva fuera
- los tests de arquitectura impidan volver atrás
