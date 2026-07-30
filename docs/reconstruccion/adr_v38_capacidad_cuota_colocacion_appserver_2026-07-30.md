# ADR V38: capacidad física, cuota, colocación y `codex app-server`

Fecha: 2026-07-30

Estado: aceptado para planificación e implementación de V38; pendiente de
implementación, ejercicio y acreditación. Este ADR no acredita `ORC-28`,
`OPS-11` ni conducta alguna.

## Problema

La primera implementación parcial de V38 mezcló cinco magnitudes —slots,
segundos, mensajes, tokens y créditos— en una sola reserva. También añadió un
lector de informe Codex desde fichero y conservó
`runtime.codex.max_concurrent_executions` como límite del presupuesto y del
despachador global.

Esas decisiones no sirven para un orquestador con varios proveedores y dos
aislamientos:

- un slot físico es excluyente y debe reservarse;
- una cuota de proveedor es una observación compartida de una cuenta/perfil,
  no un saldo que Orquesta pueda debitar o devolver;
- los candidatos de cuenta/perfil deben quedar ordenados antes del claim y la
  colocación exacta debe fijarse atómicamente con la reserva;
- un límite llamado Codex no puede gobernar Firecracker, Claude u otro
  proveedor;
- un fichero producido fuera del contrato no demuestra cuota viva ni permite
  recuperación causal;
- proceso anfitrión y worker huésped no pueden compartir estado operativo de
  `codex app-server`.

Firecracker original resuelve el proceso de la microVM y su API física. No
resuelve el DAG, la elección de agente, la cuota de proveedor, la reserva
durable, la colocación, la recuperación de Orquesta, el protocolo de trabajo,
la preservación de resultados ni el cierre acreditado. La pieza pública
`agentmicrovm/v1` añadirá esas garantías físicas reutilizables alrededor de
Firecracker sin bifurcarlo ni reimplementar su monitor de máquinas virtuales.

## Consulta previa

Se ejecutó el read-model local para `ORC-28` y `OPS-11`, sobre el plan V38 y
las operaciones de capacidad/cuota/colocación y persistencia. No devolvió
coincidencias. El hueco es consultivo: obliga a explicitar contrato y pruebas,
pero no crea autoridad ni bloquea por sí solo.

## Orden causal

Existe una sola secuencia para el binding durable:

```text
A02b contrato de candidatos
  → A04.1 estado de observación/reserva
  → A03.2 migración progresiva de cuota y binding 1:1
  → A03.3 lectura y append CAS por StateRepository
  → A04.2 claim transaccional
```

A03.2 no puede adelantarse a los tipos caracterizados en A04.1; A03.3 depende
de esa migración, y A04.2 no puede abrirse directamente desde A04.1: exige
ambas tareas de persistencia y la
presentación A05.4. Este orden serial prevalece sobre cualquier paralelismo
general de A03/A04.

## Decisión

### 1. Capacidad física reservable

La fuente física observa slots y recursos excluyentes de un pool. Para cada
`launch_agent`, `application` calcula la demanda y el claim global:

1. revalida revisión y frescura;
2. crea una reserva física ligada a proyecto, Goal, WorkItem, Execution,
   generación, action, fence, idempotency key, pool y colocación;
3. liga `reservation.observation_ref` al hecho físico usado;
4. la consume al recibir el comprobante de lanzamiento aceptado;
5. la libera solo en terminal exacto o `definitely_not_applied`;
6. ante `unknown_applied`, la retiene/cuarentena hasta reconciliar la identidad
   externa.

No habrá otro scheduler, cola, store, ledger o ciclo de vida.

La fuente publica capacidad bruta/exclusiva. Al decidir disponibilidad, held se
resta exactamente una vez por `source+pool` para reservas en estado
`reserved`, `consumed` o `quarantined`. No se suma por `observation_ref` y no se
acepta una métrica que ya haya descontado agentes activos, porque ambos casos
producirían doble contabilidad.

La unidad física queda fijada así:

- con varios perfiles Codex, cada perfil es un placement/pool reservable con
  capacidad 1;
- con un único perfil, cada lanzamiento reserva 1 de los N slots configurados
  para ese pool.

Cada launch tiene exactamente una `AgentCapacityReservation`. La capacidad del
anfitrión Firecracker se gobierna mediante
`governance.global_process_slots_budget`, comprobación previa física y leases
CID/VM; ninguno crea una segunda reserva de capacidad del agente.

### 2. Cuota por perfil como compuerta

El controlador del proveedor observa, mediante protocolo estructurado, la
elegibilidad de una cuenta/perfil, estado explícito
`available/exhausted/unknown`, ventana/reset, revisión, expiración y `RetryAt`.
La observación se persiste y el claim conserva `quota_observation_ref` como
binding inmutable.

Orquesta no reserva, debita, consume, libera ni devuelve segundos, mensajes,
tokens o créditos. Varios claims pueden referenciar la misma observación
vigente. El protocolo oficial Codex no ofrece restantes absolutos: un
porcentaje, si existe, se conserva solo como evidencia y nunca se convierte en
unidades. Estado desconocido, observación stale, error o timeout cierran solo
lanzamientos nuevos. Stop, observación, órdenes admitidas y preservación
continúan.

No se añade columna de clase ni otro lifecycle para distinguir ambos papeles.
La semántica queda expresada por los dos enlaces distintos:
`reservation.observation_ref` y `quota_observation_ref`.

### 3. Colocación opaca fijada dentro del claim

El conector de proveedor enumera candidatos opacos. Cada candidato enlaza, sin
exponer cuenta, perfil o ruta:

- `AgentPlacementRef`;
- pool/fuente física;
- observación de cuota;
- capacidades compatibles.

`application` observa, ordena de forma determinista y deduplica los candidatos.
`ClaimRequest` lleva `CapacityCandidates` ordenados, cada uno con presentación
física y binding de cuota. Dentro de la misma transacción, el claim revalida
por CAS en ese orden y fija/reserva el primer candidato todavía disponible. Si
ninguno conserva validez, no reclama la acción. El claim devuelto contiene la
`AgentPlacementRef` exacta y el request/receipt del agente debe hacer eco de
ella; pool y launcher jamás reseleccionan. Un replay conserva la misma
colocación.

La referencia opaca sí atraviesa application; nombre, ruta, cuenta y secreto
del perfil no lo hacen. Bootstrap construye conectores y registra el mapa
opaco. No ordena candidatos, elige colocación, decide admisión o escribe ciclo
de vida.

### 4. Controlador anfitrión limitado a cuota

Se permite un controlador Codex pequeño en el anfitrión exclusivamente para
observar cuota y elegibilidad. Por cada perfil configurado, bootstrap posee:

- un proceso persistente `codex app-server`;
- un lector técnico de protocolo acotado a su conexión.

Ambos se arrancan antes del primer agente y antes del planificador. La secuencia
es exacta:

1. validar configuración, credencial y ownership del perfil;
2. arrancar el hijo;
3. completar `initialize` → `initialized`;
4. solicitar y obtener una lectura inicial válida;
5. publicar la observación durable;
6. solo cuando todos los perfiles han completado lo anterior, arrancar el
   planificador.

El lector procesa respuestas y eventos enmarcados de esa conexión. No sondea,
planifica, elige colocación ni escribe lifecycle. La misma conexión persiste y
se reutiliza para cero agentes y para olas `5/10/20`; nunca nace un controlador
por lanzamiento.

Ante pérdida de conexión, la cuota pasa a desconocida, el controlador cierra y
recolecta el hijo exacto y reconecta con límites. Al rotar credencial, retira la
conexión anterior del mismo modo y no publica la sustituta hasta completar su
lectura inicial. En shutdown cancela/junta el lector, cierra `stdio`, espera el
hijo y, si vence el plazo, mata solo su PID verificado y lo recolecta. Si un
perfil falla durante bootstrap, recoge todos los ya iniciados y el planificador
no arranca.

El controlador:

- no implementa `AgentLauncher`, `AgentObserver` ni `AgentController`;
- no inicia, reanuda, orienta o interrumpe turnos;
- no accede a workspace, mailbox o artefactos;
- no decide admisión, colocación o lifecycle;
- no comparte proceso, `stdio`, socket, `CODEX_HOME`, credenciales, hilo o
  turno con un worker.

No es un daemon de dominio ni un microservicio. Es un adaptador técnico pequeño
dentro del monolito modular. Su lector por conexión es el único bucle técnico
permitido; se prohíben bucles de dominio, sondeo o planificación. Si el
protocolo real de `app-server` no permite observar cuota estructurada con la
versión fijada, A05 queda pendiente y falla cerrado. No se introduce un
sustituto por fichero, tmux, logs o scraping.

### 5. Codec común, procesos aislados

`internal/adapters/agent/codex/appserver/` será el único codec específico de
Codex. Posee únicamente:

- framing JSONL y límite de trama;
- IDs de petición y correlación de respuestas/eventos;
- negociación de versión;
- secuencia `initialize` → `initialized`;
- sesión anfitriona de cuota con capacidades nulas y métodos exactos;
- errores de protocolo estables y descarte del contenido remoto sensible.

Tiene exactamente dos consumidores reales: el controlador anfitrión de cuota y
el enlace huésped B05. No migra ni modifica el worker de proceso actual. Se
comparte código, nunca estado vivo. Cada microVM arranca su propio
`codex app-server` dentro del huésped. El enlace huésped adapta el protocolo
neutral `guestproto` al codec común y permanece fino. Ningún adaptador
Firecracker importa este subpaquete Codex.

### 6. Límites neutrales

Se añade al registro canónico
`governance.global_process_slots_budget`, alineado con
`governance.global_token_budget` y `ResourceVector.ProcessSlots`, para que
`BudgetPolicy` no dependa de Codex. Su default y migración semántica son 70.

La transición respeta el orden causal. A05.1d deja de consultar
`RuntimeCodexMaxConcurrentExecutions`, pero conserva provisionalmente
`maxConcurrentLaunches` y `ExcludeLaunch` alimentados por
`governance.global_process_slots_budget`: A04.2 todavía no ha ligado la reserva
física al claim. A07 elimina ese mecanismo únicamente después de acreditar
A04.2; desde entonces ejecuta solo un lanzamiento que `ClaimNextAction` haya
devuelto con colocación y reserva física durables.
`runtime.codex.max_concurrent_executions` queda únicamente como guardarraíl
privado del conector/pool Codex. No existe alias semántico entre ambas: no son
equivalentes. Cada una conserva su entrada de entorno canónica. La prueba
transitoria demuestra que un presupuesto neutral N nunca supera N y que
cambiar solo el límite Codex no modifica el despacho global.

Se retira `runtime.codex.capacity_report_max_bytes` y se define
`runtime.codex.app_server_max_frame_bytes` con default 1048576 y límites
1024..67108864. No hay alias ni fallback desde la clave retirada porque sus
semánticas son distintas; la clave nueva sí conserva su entrada de entorno
canónica.

### 7. Persistencia sustituible

`application` escribe mediante el mismo contrato `StateRepository`.

- SQLite es el adaptador local, de desarrollo y de pruebas.
- PostgreSQL será el adaptador productivo futuro de V31/`OPS-11`.
- Cada despliegue activa exactamente uno.
- Se prohíben escritura doble, réplica de mando, fallback de base y tipos SQL
  dentro del dominio.

La migración ya aplicada `022_agent_capacity.sql` no se modifica. Q2 usará la
siguiente migración libre y progresiva para dos hechos inmutables:

- `AgentQuotaObservationRecord` completo, con colocación, ventana, estado,
  calidad, tiempos, evidencia opcional, idempotencia y par de revisiones;
- `AgentPlacementBinding` 1:1 con exactamente
  `reservation_ref + placement_ref + quota_observation_ref +
  quota_observation_revision`, FK diferida a la reserva y referencia compuesta
  a la cuota de esa misma colocación y revisión.

No añade clase, saldo ni lifecycle de cuota. Q3 amplía el mismo
`StateRepository` con lectura vigente por colocación y append CAS/idempotente de
observaciones. No existe writer ni store independiente para el binding:
únicamente `ClaimNextAction` lo inserta atómicamente junto con la reserva.

## Modelo de componentes

```text
fuente física ───────────────┐
                             ├─> application ordena/deduplica candidatos
controlador de cuota Codex ──┘                        │
                                                      v
                                        StateRepository.claim
                                        - revalida por CAS
                                        - fija primer candidato válido
                                        - reserva física
                                        - placement_ref
                                        - quota_observation_ref
                                                      │
                                                      v
                                             despachador único
                                                      │
                                                      v
                                           worker dentro de microVM
                                           app-server propio

controlador de cuota host ─── codec común ─── enlace huésped B05
       app-server propio              (solo código; estado separado)
```

El controlador anfitrión de cuota también usa el codec común, con una instancia
propia que solo admite lectura y actualización de cuota; no aparece en la ruta
de ejecución del worker.

## Presupuesto y compensación

No se usa la contingencia de V38. La redistribución retira duplicación y
holgura no consumida, y el subtotal de las tareas afectadas permanece
exactamente igual:

| Tarea | Antes P/V | Ahora P/V | Variación |
|---|---:|---:|---:|
| A01 | 0/250 | 0/138 | 0/-112 |
| A02 | 220/250 | 159/226 | -61/-24 |
| A03 | 150/250 | 334/561 | +184/+311 |
| A04 | 480/450 | 459/447 | -21/-3 |
| A05 | 300/300 | 608/508 | +308/+208 |
| A08 | 100/750 | 0/750 | -100/0 |
| B01 | 150/200 | 0/50 | -150/-150 |
| B05 | 650/550 | 490/320 | -160/-230 |
| **Subtotal** | **2.050/3.000** | **2.050/3.000** | **0/0** |

La compensación es real:

- A01 ya materializó sus cuatro cohortes en `P=0,V=138`;
- A02 atribuye una sola vez `77/120 + 57/65 + 25/41 = 159/226`;
- A03 conserva `P=139,V=291` ya consumidos por `71f4a827` y reserva
  `P=195,V=270` para la migración completa de cuota/binding y el contrato CAS
  del mismo `StateRepository`;
- A04 atribuye `P=114,V=53` a la base física, `P=46,V=57` al binding neutral y
  `P=299,V=337` al reclamo y su cierre;
- A05 mide el codec `app-server` aceptado en `P=278,V=179` y conserva una
  envolvente conjunta `P=163,V=146` para controlador e inyección;
- A08 es una compuerta de aceptación y no recibe presupuesto productivo;
- B01 queda como gate contractual `P=0,V=50`: la selección vive en A04, las
  fuentes/controlador en A05 y la composición Firecracker real en B10.4;
- B05 conserva solo el enlace fino del huésped;
- se elimina el lector de fichero y su configuración;
- A04 no implementa reserva, débito o liberación de cuatro magnitudes de cuota;
- no se añaden daemon, store, scheduler, writer, cola ni bucle de dominio,
  sondeo o planificación; solo el lector técnico acotado por conexión que
  bootstrap inicia y cierra.

El total V38 permanece `P=7.200, V=9.800` y ninguna cifra se descuenta dos
veces. Superar cualquiera de estos techos
requiere otro ADR, compensación concreta y autorización antes de programar. Un
exceso no se oculta como generado, prueba existente o trabajo de otra tarea.

## Alternativas rechazadas

- **Reservar cuota como si fuera capacidad física:** produce sobrecontabilidad,
  devoluciones falsas y carreras con el proveedor, que sigue siendo la
  autoridad del consumo real.
- **Elegir perfil después del claim:** impide reserva causal e idempotencia; un
  replay podría ejecutar en otra cuenta.
- **Fijar el perfil en application antes de la transacción:** obliga a repetir
  ciclos si otro claim gana el último slot. Application ordena candidatos y la
  transacción fija el primero todavía válido.
- **Mantener el lector de fichero como fallback:** crea una segunda autoridad
  sin productor, revisión o entrega causal acreditados.
- **Usar tmux o logs para cuota:** el texto es observación heurística, no
  contrato estructurado ni comprobante.
- **Compartir un `app-server` anfitrión entre cuota y trabajo microVM:** rompe
  aislamiento, scope de credenciales, trazabilidad y la garantía de que el
  agente trabaja dentro del huésped.
- **Crear un servicio de cuotas independiente:** añade despliegue, base, bucle y
  autoridad innecesarios. Un adaptador pequeño satisface la frontera.
- **Usar el límite Codex como límite global:** acopla el núcleo a un proveedor
  y gobierna incorrectamente Firecracker y futuros conectores.
- **Implementar PostgreSQL en V38:** reabre una vertical V31 sin necesidad
  causal. V38 mantiene el contrato y una sola fuente activa.

## Pruebas y condiciones de cierre

La implementación deberá acreditar, como mínimo:

- carrera de dos claims sobre el último slot físico;
- varios claims ligados a una misma cuota sin débito o liberación;
- cuota `available/exhausted/unknown`, porcentaje solo como evidencia, nueva
  ventana, stale, error y timeout, sin restantes inventados;
- replay/reinicio conservando colocación y ambas referencias;
- carrera sobre candidatos donde el claim fija el primero disponible y pool o
  launcher no reseleccionan;
- held restado una sola vez por `source+pool`, incluida métrica neta rechazada;
- multiperfil 1 por placement y perfil único 1 de N, con exactamente una
  `AgentCapacityReservation` aunque el aislamiento sea Firecracker;
- `unknown_applied` reteniendo la reserva física;
- cambio de proveedor/aislamiento sin lectura de `runtime.codex.*` global;
- despachador incapaz de lanzar sin claim reservado;
- controlador de cuota incapaz de usar métodos de worker;
- procesos y scopes separados entre anfitrión y huésped;
- cero agentes: lectura inicial, conexión persistente y shutdown completo;
- olas 5/10/20 reutilizando exactamente las conexiones por perfil, sin proceso
  ni lector por lanzamiento;
- fallo del segundo perfil durante bootstrap que recoge el primero y no arranca
  el planificador;
- actualizaciones, pérdida/reconexión y rotación manteniendo una sola conexión
  vigente por perfil;
- shutdown cooperativo y forzado sin proceso, lector, socket o PID huérfanos;
- ausencia del lector/configuración de fichero y de cualquier fallback;
- misma suite contractual con SQLite y, en V31, PostgreSQL, siempre uno activo.

Hasta superar esas pruebas sobre el mismo candidato y recibir evidencia
independiente, todas las tareas de este ADR permanecen `planned` y no
acreditantes.
