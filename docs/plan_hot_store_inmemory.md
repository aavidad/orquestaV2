## Plan: Hot State Hibrido para el Control Plane

### Diagnostico

La medicion real del daemon en reposo no apunta primero a `status` ni a
`runtime_orders` como tabla. El cuello principal hoy es:

- `procesarAutonomiaAgentesBatch`: `8s-20s`
- `procesarRuntimeMailboxBatch`: `2.6s-5.4s`
- `ProcesarHandoffsBatch`: `1.3s-3.5s`
- `ProcesarSupervisionAutonomaBatch`: `1.5s-2.6s`

Eso significa que mover datos a memoria puede ayudar, pero no resuelve por si
solo el problema principal. El primer gasto hoy es la logica de autonomia y las
consultas derivadas por sesion activa.

### Decision arquitectonica

Esta linea de trabajo no se limita a Orquesta. Se convierte en regla general
para cualquier app futura fabricada con Orquesta que realmente necesite un
servicio residente:

- el nucleo residente debe ser minimo
- lo pesado sale a workers, eventos o caminos on-demand
- el estado caliente se distingue del estado durable
- memoria-first solo para lo efimero que no rompa continuidad
- durabilidad inmediata para lo contractual o recuperable tras reinicio

No conviene mover `runtime_orders` ni `runtime_mailbox` a memoria como unica
verdad.

Motivos:

- `runtime_orders` ya esta fijada en doctrina como cola durable formal.
- `runtime_mailbox` ya esta fijada en doctrina como mensajeria persistente
  tipada con `ack` real.
- ambas primitivas deben sobrevivir a reinicios, handoff, bootstrap y caidas
  del daemon sin perder continuidad.

Lo que si conviene hacer es un modelo hibrido:

- `runtime_handles`:
  - autoridad caliente en memoria
  - snapshot periodico a BD
  - flush final en shutdown
- `runtime_orders`:
  - BD como verdad durable
  - indice caliente en memoria solo para ordenes vivas
  - lectura desde memoria en el hot path
  - escritura sincronizada a BD en transiciones durables
- `runtime_mailbox`:
  - BD como verdad durable
  - indice caliente en memoria solo para mensajes pendientes/entregables
  - lectura desde memoria en el hot path
  - escritura sincronizada a BD en `insert`, `delivered`, `consumed`

### Que no hacer

- no hacer `write-behind` ciego para `runtime_orders`
- no hacer `write-behind` ciego para `runtime_mailbox`
- no devolver punteros mutables del store a varios callers
- no cargar toda la historia a memoria
- no convertir SQLite en un mero backup eventual

### Que si hacer

#### 1. `runtime_handles` memoria-first

Esta es la mejor candidata para salir de la BD como hot path:

- solo el daemon la lee/escribe de verdad
- es estado observado, no autoridad de negocio
- cambia con mucha frecuencia (`last_seen_at`, `estado`, metadata compacta)

Modelo:

- store en memoria por `handle_id`, `agente`, `sesion_id`
- `dirty set` por handle
- flush cada `5s-15s`
- flush final en shutdown
- flush inmediato cuando cambia algo semantico:
  - `estado`
  - `runtime_id`
  - `external_session_id`
  - `capabilities`

#### 2. `runtime_orders` con indice caliente

No memoria-only. Si se cae el daemon, una orden viva no puede desaparecer.

Modelo:

- al arrancar:
  - cargar solo ordenes no terminales
- mantener indices en memoria:
  - `byID`
  - `pending`
  - `running`
  - `byAgent`
- al crear/claim/complete/fail:
  - persistir en BD en la misma operacion
  - actualizar cache caliente en memoria
- el runner y los batches leen el indice caliente, no hacen scans SQL

Objetivo:

- quitar scans repetidos de ordenes vivas
- mantener durabilidad inmediata de las transiciones importantes

#### 3. `runtime_mailbox` con indice caliente

Tampoco memoria-only.

Modelo:

- al arrancar:
  - cargar solo mailbox no consumida
- mantener indices en memoria:
  - `pending by agent`
  - `pending by project`
  - `pending by family`
- al insertar/marcar entregado/marcar consumido:
  - escribir en BD en la misma operacion
  - actualizar indice en memoria

Objetivo:

- evitar scans SQL en cada ciclo del runner
- preservar continuidad durable entre reinicios

### Orden correcto de trabajo

#### Fase A. Cortar el mayor hot path

Antes del hot store:

- cache por ciclo de autonomia
- resolver `construirAgenteTickOutput` con un snapshot precalculado por tanda
- dejar de hacer consultas repetidas por sesion activa

Esto es lo que primero debe bajar la CPU.

#### Fase B. `runtime_handles` memoria-first

Esta si merece ir primero porque es la pieza mas segura de extraer de SQLite.

#### Fase C. Indice caliente de `runtime_mailbox`

Solo para pendientes y mensajes vivos.

#### Fase D. Indice caliente de `runtime_orders`

Solo para ordenes vivas y leases activas.

### Modelo de temporizacion eficiente

La politica de produccion queda asi:

- saldo:
  - siempre al inicio de una sesion real de trabajo
  - luego cada `1-2 min`
- status visible:
  - snapshot cacheado `1 min`
- observacion pesada de runtime:
  - no menos de `60s`
- nada de revisar `/proc`, handoffs o mailbox en bucles de milisegundos

### GPU y paralelizacion

La GPU no aporta nada al core del orquestador.

El trabajo caliente del daemon es:

- consultas SQL
- JSON
- scheduling
- agregacion de estado

Eso es CPU generalista e I/O, no GPU.

La mejora profesional aqui es:

- menos scans
- indices calientes en memoria
- menos serializacion
- menos recomputacion por ciclo
- colas e iteradores bounded
- goroutines limitadas con worker pools pequeños
- batch snapshots reutilizables por ciclo

La GPU solo tendria sentido para:

- inferencia local de modelos
- embeddings
- rerankers
- agentes Ollama locales

No para el control plane de Orquesta.

### Referencias utiles absorbibles

- `oh-my-codex`:
  - buen patron de estado caliente local
  - pero no tiene nuestra semantica durable de cola y mailbox
- `claw-code-dev-rust`:
  - buen patron de runtime local y artefactos ligeros
  - pero tampoco sustituye el control plane durable

La conclusion es hibrida:

- copiar su disciplina de estado caliente
- conservar la durabilidad formal de Orquesta

### Tareas a abrir en Orquesta

1. Cache por tanda para `procesarAutonomiaAgentesBatch` y `construirAgenteTickOutput`.
2. Hot store memoria-first para `runtime_handles`.
3. Indice caliente para `runtime_mailbox` pendiente.
4. Indice caliente para `runtime_orders` vivas.
5. Bench y perfil de CPU del runner con la flota real.
6. Soporte de agentes Ollama locales gobernados por Orquesta.
