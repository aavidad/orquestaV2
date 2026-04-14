# Plan De Refactor De `db` Hacia Hexagonalidad

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

