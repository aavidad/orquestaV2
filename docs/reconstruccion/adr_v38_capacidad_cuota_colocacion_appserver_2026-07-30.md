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
la preservación de resultados ni el cierre acreditado. La aplicación hermana
**Agente MicroVM**, publicada como `aavidad/agente_microvm`, aporta las
garantías físicas reutilizables alrededor de Firecracker sin bifurcarlo ni
reimplementar su monitor de máquinas virtuales. Su núcleo y CLI
`agente-microvm` son Rust; el cliente de Orquesta es Go. Es un proyecto
independiente, no un árbol público dentro del módulo Orquesta.

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

- con uno o varios perfiles Codex ligados a cuenta, cada perfil es un
  placement/pool reservable con capacidad 1;
- un adaptador sin perfil ligado no publica candidato: carece de cuota y
  colocación durables, por lo que cierra solo lanzamientos nuevos.

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
`ClaimRequest` lleva `CapacityCandidates` ordenados, cada uno con una entrega
física todavía sin revisión y una presentación durable de cuota. Dentro de la
misma transacción, el claim repite o materializa por CAS la observación física,
revalida la cuota por referencia y revisión y fija/reserva el primer candidato
todavía disponible. Versionar la observación física antes de ese `BEGIN`
reabriría la carrera entre observación y reserva. Si ninguno conserva validez,
no reclama la acción. El claim devuelto contiene la `AgentPlacementRef` exacta
y el request/receipt del agente debe hacer eco de ella; pool y launcher jamás
reseleccionan. Un replay conserva la misma colocación.

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

### 5. Códec interno, esquema neutral y procesos aislados

`internal/adapters/agent/codex/appserver/` será el único codec específico de
Codex. Posee únicamente:

- framing JSONL y límite de trama;
- IDs de petición y correlación de respuestas/eventos;
- negociación de versión;
- secuencia `initialize` → `initialized`;
- sesión anfitriona de cuota con capacidades nulas y métodos exactos;
- errores de protocolo estables y descarte del contenido remoto sensible.

Lo consumen dentro de Orquesta el controlador anfitrión de cuota y, cuando
corresponda, el agente ejecutor de proceso actual. Orquesta deriva un artefacto
neutral versionado con el esquema necesario para el enlace huésped B05. El proyecto
hermano implementa ese artefacto sin importar el paquete `internal`, copiar
credenciales ni compartir estado vivo. Cada microVM arranca su propio
`codex app-server` dentro del huésped. Ningún paquete de `agentmicrovm` importa
este subpaquete Codex.

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

### 8. Agente MicroVM es una aplicación hermana independiente

La frontera física vive en `https://github.com/aavidad/agente_microvm`, con
crate `agente_microvm`, binario Rust `agente-microvm`, cliente Go,
configuración tipada, pruebas y documentación propios. No es un subárbol,
paquete público ni binario de Orquesta. Esta separación está
justificada por tecnología y aislamiento: posee procesos Firecracker, KVM,
jailer/cgroups, sockets vsock y recuperación física, pero no adquiere
ciclo de vida, planificación ni política de dominio.

Orquesta consumirá una única interfaz local versionada
`agentmicrovm.local.v1`, transportada sobre socket Unix y sin punto de conexión
remoto.
El proceso hermano se ejecutará en modo `serve`, supervisado o activado por
socket durante las ejecuciones que lo necesiten. Un único binario ofrece
lanzamiento, observación, órdenes, eventos, parada, preservación, recuperación
y cierre; no se crean servicios por proveedor, fase o función. Un proceso
de una sola ejecución queda descartado porque no puede recibir órdenes
posteriores ni
reconciliar una ejecución después de reiniciar Orquesta.

La autoridad queda separada:

- Orquesta conserva `Goal`, `WorkItem`, `Execution`, admisión, capacidad y
  cuota, permisos, presupuesto, ciclo de vida y comprobantes de producto;
- `agentmicrovm` conserva solo su registro físico privado, identificadores
  opacos, concesiones temporales de CID, cerca, expiración, recuperación y
  comprobantes físicos;
- cada aplicación elige y migra su propio almacenamiento; nunca comparten base
  de datos, sistema de archivos, esquema, migración, transacción, socket de
  estado ni bloqueo;
- Orquesta persiste únicamente referencias opacas, cercas y comprobantes
  recibidos por el protocolo; nunca abre el almacenamiento privado de
  `agentmicrovm`.

Tampoco comparten rutas ni almacenes de artefactos. Orquesta resuelve su
espacio de trabajo/CAS y transmite contenido acotado, enmarcado y verificado
por referencia/resumen criptográfico a través del socket Unix; para cargas
grandes puede transferir un
descriptor anónimo sellado por `SCM_RIGHTS`, nunca una ruta. El resultado
regresa por el mismo contrato de contenido/referencia y Orquesta lo valida y
persiste en su propio almacén.

La autorización entre proyectos usa una concesión neutral firmada, de un solo
uso, ligada al resumen criptográfico completo del plan físico, `RunRef`, cerca,
caducidad y audiencia `agentmicrovm.local.v1`. Orquesta firma después de aplicar su
autorización; `agentmicrovm` verifica con material público de confianza
configurado localmente. No atraviesan la frontera `CredentialStore`, secretos
compartidos, credenciales de proveedor, tipos de atestador, rutas privadas ni
importaciones entre ambos módulos.

El contrato público neutral de `agentmicrovm` expone referencias, solicitudes,
eventos y comprobantes versionados; no contiene `Goal`, `WorkItem`,
`Execution`, SQLite, PostgreSQL, CAS, `CredentialStore`, Codex ni paquetes
`internal` de Orquesta.
El conector Orquesta adapta sus puertos `AgentLauncher`, `AgentObserver`,
`AgentController` y `AgentShutdown` al protocolo local. El controlador de
cuota y el códec Codex permanecen en Orquesta; el huésped consume un artefacto
de protocolo neutral versionado, sin importar el códec interno anfitrión.

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
                                      conector local agentmicrovm
                                      socket Unix, protocolo v1
                                                      │
                                                      v
                                  aplicación hermana agentmicrovm
                                  estado físico privado + Firecracker
                                                      │
                                                      v
                                           worker dentro de microVM

controlador de cuota anfitrión ─── códec Codex interno
       `app-server` propio

Orquesta ── referencias/contenido/concesión firmada ──> agentmicrovm
         <── eventos/comprobantes/contenido ───
```

El controlador anfitrión de cuota usa el códec interno, con una instancia
propia que solo admite lectura y actualización de cuota; no aparece en la ruta
de ejecución del agente ejecutor. El enlace huésped pertenece al proyecto
hermano y usa el protocolo neutral sellado; no importa ni comparte estado con el códec
interno de Orquesta.

## Presupuesto y compensación

No se usa la contingencia de V38. La redistribución retira duplicación y
holgura no consumida, y el subtotal de las tareas afectadas permanece
exactamente igual:

| Tarea | Antes P/V | Ahora P/V | Variación |
|---|---:|---:|---:|
| A01 | 0/250 | 0/138 | 0/-112 |
| A02 | 220/250 | 170/238 | -50/-12 |
| A03 | 150/250 | 334/537 | +184/+287 |
| A04 | 480/450 | 459/447 | -21/-3 |
| A05 | 300/300 | 682/508 | +382/+208 |
| A07 | 250/400 | 165/695 | -85/+295 |
| A08 | 100/750 | 0/750 | -100/0 |
| B01 | 150/200 | 0/50 | -150/-150 |
| B05 | 650/550 | 490/320 | -160/-230 |
| **Subtotal** | **2.300/3.400** | **2.300/3.683** | **0/+283** |

La compensación es real:

- A01 ya materializó sus cuatro cohortes en `P=0,V=138`;
- A02 atribuye una sola vez `77/120 + 68/77 + 25/41 = 170/238`;
- A03 conserva `P=139,V=270` ya consumidos por `71f4a827` y reserva
  `P=195,V=267` para la migración completa de cuota/vínculo y el contrato CAS
  del mismo `StateRepository`;
- A04 atribuye `P=114,V=53` a la base física, `P=46,V=57` al binding neutral y
  `P=299,V=337` al reclamo y su cierre;
- A05 mide el códec `app-server` aceptado en `P=278,V=179` y conserva una
  envolvente conjunta `P=237,V=146` para traductor, controlador e inyección;
- A07 atribuye sus `P=99,V=695` ya integrados y conserva `P=66` para retirar
  los límites transitorios después de A04.2;
- A08 es una compuerta de aceptación y no recibe presupuesto productivo;
- B01 queda como gate contractual `P=0,V=50`: la selección vive en A04, las
  fuentes/controlador en A05 y la composición Firecracker real en B10.4;
- B05 conserva solo el enlace fino del huésped;
- se elimina el lector de fichero y su configuración;
- A04 no implementa reserva, débito o liberación de cuatro magnitudes de cuota;
- no se añaden planificador, escritor, cola ni bucle de dominio, sondeo o
  planificación; el servicio local `agentmicrovm` posee únicamente su estado y
  bucles técnicos físicos, y el lector Codex queda acotado a su conexión.

El total V38 queda en `P=7.200, V=10.083`: el producto se compensa sin crecer
y la verificación aumenta `V=283` porque las pruebas A07 ya integradas no
cabían en el techo anterior. Ninguna cifra se descuenta dos veces. Estos techos
incluyen ambos repositorios y no contienen contingencia
oculta. Superar cualquiera de estos techos
requiere otro ADR, compensación concreta y autorización antes de programar. Un
exceso no se oculta como generado, prueba existente o trabajo de otra tarea.

## Corrección presupuestaria del 2026-08-01

La auditoría posterior a `f2cb965b` demostró que A05.2/A05.4 solo tenían el
observador estático, la ordenación neutral y el arranque de los controladores de
cuota: faltaban catálogo productivo, observaciones físicas renovables e
inyección en el reclamo. El hueco se registró como
`BUG-ORQ-20260801-613`; A04.2 no puede absorberlo sin superar su techo.

Se trasladan exactamente `P=180,V=120` desde B10 hacia A05. B10 ya no vuelve a
implementar un `client.go`: consume el módulo Go público de Agente MicroVM,
ligado a una revisión, y conserva en Orquesta únicamente traducción hexagonal,
composición y guardas de independencia. Esto elimina una duplicación futura
nombrada y reduce también la composición prevista. A05 queda en
`P=862,V=628`; B10 queda en `P=470,V=480`. El total V38 permanece exactamente
en `P=7.200,V=10.083`; no aparece contingencia ni segundo cliente.

La implementación `e7f9e60e`+`993e3d03` consume esa reasignación completa.
El contrato preparatorio compartido con Q4 consume además `P=44,V=107` de la
bolsa A04.2 ya existente, que queda en `P=116,V=83`. Hasta el cutover, un
catálogo Codex vacío no activa la compuerta y conserva V23; un perfil durable
sí publica capacidad uno y exige cuota vigente. Este puente de compatibilidad
no acredita A05: Q4 debe ser el primer escritor transaccional real.

La revisión viva de Q4 del 2026-08-01 encontró que el escritor no puede
separarse honestamente del replay cercado ni de las transiciones: una reserva
única por acción conserva su cerca original, mientras un reclamo recuperado
recibe una cerca posterior. La validación, el esquema progresivo y el consumo
de esa misma reserva deben evolucionar juntos. Por ello los remanentes de
A04.2, A04.3 y A04.4 se gobiernan como una sola bolsa y un solo gate
`P=299,V=337`; no cambia el techo A04 ni el total V38. La migración nueva no
modifica 022/023: impone una reserva por acción y permite que una transición se
acredite mediante un intento posterior cercado, ligado a su propio receipt.

La medición ejecutable posterior del mismo gate demostró que el escritor, las
transiciones atómicas y la recuperación semántica completa consumen
`P=489,V=387`. Se trasladan `P=190,V=50` desde B10.4/B10.5: B10.4 se compensa
por la retirada de la compuerta provisional y B10.5 es verificación sobre los
adaptadores B10.1/B10.2, no otro producto. A04 queda en `P=649,V=497` y B10 en
`P=280,V=430`; su subtotal conjunto sigue siendo exactamente
`P=929,V=927`. V38 permanece en `P=7.200,V=10.083`.

La ejecución de la suite global encontró además 52 fixtures de bootstrap que
todavía omitían la colocación o reconstruían `Orchestrator` sin sus fuentes de
capacidad. Su migración y el simulador común de cuota `app-server` consumen
verificación de Q4, no producto A08. Se trasladan por ello `V=100` de A08 a la
bolsa indivisible A04.2–A04.4: queda en `P=489,V=487`, A04 en `P=649,V=597` y
A08 en `P=0,V=650`. El subtotal y V38 permanecen invariantes.

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
- **Incrustar `agentmicrovm/v1` dentro del módulo Orquesta:** permite
  importaciones,
  almacenamiento y configuración accidentales entre autoridades y convierte
  una frontera tecnológica sustituible en detalle interno.
- **Compartir base de datos, sistema de archivos, rutas o secretos entre ambos
  proyectos:** rompe
  propiedad, recuperación y rotación independientes; además impide demostrar
  el protocolo como única integración.
- **Ejecutar un binario de una sola ejecución por lanzamiento:** no admite
  órdenes posteriores, eventos, parada o recuperación sin inventar canales y
  estado paralelos.
- **Exponer `agentmicrovm` como servicio remoto:** amplía autenticación, red y
  operación sin necesidad causal; V38 exige solo socket Unix local versionado.
- **Enviar una prueba HMAC con secreto compartido:** acopla almacenes de
  credenciales. La concesión firmada neutral permite verificar con confianza
  pública sin revelar el firmante ni secretos de Orquesta.
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
- uno o varios perfiles ligados con capacidad 1 por placement, con exactamente una
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
- contrato cruzado que levanta el binario hermano, negocia
  `agentmicrovm.local.v1` y prueba incompatibilidad de versión y trama;
- guardas que prohíben importaciones cruzadas, base de datos, sistema de
  archivos, ruta, secreto o
  `CredentialStore` compartidos entre ambos proyectos;
- reinicio independiente de cada proceso y de cada almacenamiento, con
  referencias y cercas reconciliadas solo por el protocolo;
- repetición, audiencia, expiración, resumen criptográfico del plan y uso único
  de la concesión neutral firmada;
- artefactos grandes por tramas acotadas o descriptor anónimo sellado, con
  recorrido de ruta, enlace, truncado y resumen criptográfico incorrecto
  rechazados;
- ausencia del lector/configuración de fichero y de cualquier fallback;
- misma suite contractual con SQLite y, en V31, PostgreSQL, siempre uno activo.

Hasta superar esas pruebas sobre la misma composición candidata y los
resúmenes criptográficos exactos de Orquesta y `agentmicrovm`, y recibir
evidencia independiente, todas las tareas de este ADR permanecen `planned` y
no acreditantes.
