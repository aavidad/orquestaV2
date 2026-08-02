# Plan de ejecución V38: agentes autónomos sobre Firecracker

Fecha de preparación: 2026-07-30

Estado: plan ejecutable; ninguna tarea de esta lista está acreditada por este
documento.

## Resultado que debe quedar acreditado

V38 acreditará `ORC-28` cuando una misma composición candidata demuestre las
tres compuertas siguientes, en este orden:

- **Compuerta A, núcleo neutral:** Orquesta conserva el `Goal`, deriva todo el
  conjunto listo, representa capacidad externa durable, lanza en paralelo solo
  acciones `launch_agent`, sigue observando y deteniendo cuando el lanzamiento
  está saturado, recupera tras reinicio y preserva el entorno. Esta compuerta no
  ejecuta ni requiere KVM, Firecracker, `rootfs` o sockets vsock.
- **Compuerta B, aplicación hermana `agentmicrovm` y conector explícito:** el
  proyecto independiente crea exactamente una microVM física por ejecución de
  agente; Orquesta lo consume solo por un protocolo local versionado sobre
  socket Unix. El huésped recibe contenido, nunca rutas del anfitrión, usa
  intermediario y proxy controlados, acepta órdenes posteriores al arranque, se observa
  y detiene por identidad exacta y entrega sello e inventario antes de
  desmontar. Bubblewrap continúa siendo el aislamiento del `TestAttestor`.
- **Compuerta C, olas físicas:** los mismos resúmenes criptográficos de ambos
  árboles, binarios, `rootfs`, configuraciones y datos de prueba sellados superan las olas
  `1/5/10/16/20`, sin reconstrucción entre peldaños y sin convertir esos
  límites físicos en un límite lógico del DAG.

Las cohortes lógicas obligatorias son `1/16/70/500`. Los peldaños físicos son
`1/5/10/16/20`. Un `WorkItem` listo cuya ejecución ya fue admitida y no obtiene
capacidad conserva la misma `Execution` y queda esperando; no consume un nuevo
intento de ejecución.

Este trabajo usa la excepción bootstrap descrita en el `AGENTS.md`: Codex
directo puede coordinar write-sets acotados hasta que Orquesta acredite esta
vertical. Cada encargo deberá declarar capability, invariante, write-set,
pruebas, riesgos y siguiente dependencia; no usará ni modificará el motor
antiguo ni convertirá esta coordinación temporal en otra arquitectura.

## Autoridad y alcance

- Capability promovible: `ORC-28`, contrato
  `AC-V38-AGENT-RUNTIME-ELASTIC`.
- Prerrequisitos que se reutilizan, sin reabrirlos:
  `AGT-01`, `AGT-03`, `GOV-21`, `ORC-10` y `EVD-13`.
- Dependencias ya acreditadas que se consumen: configuración y credenciales,
  backup/recuperación, controles y efectos, presupuestos, workspace/Git/CAS,
  TestAttestor y E2E Codex.
- `ORC-15` sigue perteneciendo a V27 y `OPS-16/OPS-17` a V32. V38 solo prueba
  continuidad estrecha de las órdenes/mailbox ya disponibles y el stop exacto
  necesario para `ORC-28`; no promueve esas capabilities futuras.
- Firecracker continúa desligado de V23. Ninguna compuerta del Wizard, SQLite o V23
  puede adquirir una dependencia con V38.
- **Agente MicroVM** se publica como `aavidad/agente_microvm`: núcleo y CLI
  `agente-microvm` en Rust, cliente de Orquesta en Go y protocolo máquina
  `agentmicrovm.local.v1` por HTTP/1.1 sobre socket Unix. No comparten base de
  datos, sistema de archivos, secreto, ruta ni importación.
- Fuentes consultadas antes de preparar el plan:
  `LEEME_AGENTE_ORQUESTAV2.md`, el handoff vigente del 2026-07-30,
  `ruta_total_100.md`, roadmap, datos de prueba y guardas V38, contratos de puertos,
  inventario de herramientas, cortes Firecracker y lecciones legacy.
  Las consultas específicas del 2026-07-30 por `ORC-28` y `OPS-11` no
  devolvieron coincidencias; se registran como huecos consultivos, no como
  permiso para inventar semántica. Las consultas anteriores por `ORC-03`,
  `ORC-10`, `OPS-16`, `EXT-10`, `EVD-13` y `EXT-21` aportaron invariantes de
  caracterización, no autoridad de cierre.
- La decisión de planificación que separa capacidad física reservable, cuota
  por perfil, colocación y protocolo común de `codex app-server` vive en
  `adr_v38_capacidad_cuota_colocacion_appserver_2026-07-30.md`. Está aceptada
  para ejecutar V38, pero no acredita ninguna capability ni conducta.

## Lo que ya existe y no se vuelve a construir

- `Goal.RunnableWorkItems` y `Goal.ReadyWorkItems` derivan el conjunto completo,
  determinista y libre de conflictos. V38 lo caracteriza a escala; no crea
  otro planificador, ready queue, cohort manager ni state machine.
- `application.Orchestrator` es el único escritor del ciclo de vida y decide
  las mutaciones mediante el contrato `StateRepository`; SQLite es hoy el
  conector local, de desarrollo y de pruebas, no parte del dominio ni destino
  productivo definitivo. PostgreSQL será el conector productivo futuro de
  V31/`OPS-11`. Cada despliegue selecciona una sola fuente transaccional activa:
  no hay escritura doble, réplica de mando ni fallback entre bases.
  `agentmicrovm` tendrá un `RuntimeJournal` físico privado con migración y
  recuperación propias; no se compone sobre `StateRepository` ni comparte su
  fuente. Planificador, eventos, `outbox`, `bootstrap` y proyecciones no
  obtienen autoridad propia.
- El registro de efectos ya separa intent, aprobación, intento y comprobante, y ya
  trata `unknown_applied` como cuarentena. V38 lo extiende a capacidad y
  microVM; no crea un ledger paralelo.
- Los presupuestos durables ya existen. El presupuesto genérico de slots de
  despliegue seguirá gobernando política global, pero no se disfrazará de
  capacidad física ni de cuota externa. El actual
  `runtime.codex.max_concurrent_executions` se retirará de `BudgetPolicy` y del
  despachador global; quedará únicamente como guardarraíl local del
  conector/pool Codex.
- `AgentLauncher`, `AgentObserver`, `AgentController`, `AgentShutdown`,
  `ActionStopAgentRequest` y el pool Codex ya conservan binding e idempotencia.
  El conector local `agentmicrovm` implementará esos puertos; no añadirá ciclo
  de vida de proveedor.
- Espacio de trabajo/Git, CAS, integración en anfitrión, sesiones de ejecución,
  MCP y buzón causal ya están acreditados. El conector Orquesta traduce esos almacenes a
  referencias y contenido verificado por el socket local; `agentmicrovm` y el huésped nunca
  reciben una ruta del anfitrión ni abren el CAS de Orquesta.
- Siguen existiendo los contratos `agent_microvm_network` y
  `agent_microvm_launch_auth`, además del adaptador de autenticación que B08
  debe caracterizar y retirar. B06 retiró `agent_microvm_bundle` y su único
  consumidor `launchplan`: solo proyectaban metadata sin mover o verificar
  bytes. B02 ya migró la autoridad
  CID al registro privado del proyecto hermano y el commit `127c0a45` retiró
  de Orquesta `agent_microvm_vsock_cid`, el allocator y el renderer físicos,
  sin consumidores, bridge ni adaptador duplicado «por compatibilidad».
- El launcher de `TestAttestor` puede aportar primitivas pequeñas y comunes de
  path confiable, hashes, FDs, jailer/cgroup y teardown solo cuando dos
  consumidores reales lo justifiquen. Su protocolo one-shot, rootfs y
  comprobantes no son el motor de agentes y su evidencia `1+16` no acredita V38.
- Los datos de prueba V38 actuales prueban que el plan declarado es semánticamente
  coherente. No prueban composición, persistencia, KVM ni olas reales.
- La base neutral de A07 —`ExcludeLaunch`, `ClaimNextAction` y
  `ProcessClaim`— quedó contrarrevisada e integrada en `7ae98026`. El
  despachador concurrente quedó contrarrevisado e integrado en `e4a6a98e`.
  Ninguna de esas piezas acredita por sí sola A07 ni V38: falta enlazar la
  capacidad durable de A04 y superar la compuerta A08.
- Existen una fuente configurada, una observación Codex basada en un fichero y
  una reserva única que mezcla slots con segundos, mensajes, tokens y créditos.
  Son piezas parciales, no acreditadas. V38 conserva la fuente configurada como
  sustituto contractual; reemplaza y retira el lector de fichero sin fallback;
  y migra el estado para reservar solo recursos físicos y tratar la cuota por
  perfil como compuerta durable sin débito ni liberación.

Ya existen el proyecto independiente `agentmicrovm`, su protocolo local
versionado, el registro físico privado, la concesión CID durable y el motor
Firecracker que consume ese CID. También existen la fuente neutral de
capacidad, el controlador de cuota estructurada, la colocación opaca fijada,
el protocolo huésped vivo y la imagen reproducible con SBOM. Aún faltan cerrar
el intermediario/proxy, el retorno sellado y la
evidencia física `1/5/10/16/20`. Tampoco existe aún el conector final Orquesta ni la prueba
cruzada de compatibilidad. Esas son las brechas que siguen.

## Orden causal y write-sets

`A01 + A02`; `A02 → [A03.1 | A05.1 → (A05.1b + A05.1c + A05.1d)]`;
`A02 → A02b → [A04.1 → A03.2 → A03.3 | A05.2 | A05.3a → A05.3b]`;
`[A05.1b + A05.1c + A05.1d + A05.2 + A05.3b] → A05.4`;
`[A03.3 + A05.4] → A04.2 → A04.3 → A04.4 → A06 → A07 → A08 →`
`B03 → [B02 | (B05 + B06 → B07) | B08 | B09]`;
`B02 + B07 + B08 + B09 → B04 → B01`;
`B01 + B04 + B06 → B10 → B11 → B12 →`
`C01 → C02 → C03 → C04 → C05`.

El bloqueo causal es explícito: el censo contiene exactamente 53 literales de
`ClaimRequest` sin observación de capacidad: uno productivo que corrige A05.4 y
52 de pruebas que migra A04.2, repartidos entre los paquetes `acceptance`,
`internal/application`, `internal/adapters/state/sqlite` e
`internal/bootstrap`. Cada paquete de prueba usa como máximo un helper local no
exportado; queda prohibido crear un helper cruzado, una API de pruebas o un
fallback productivo. Por ello A04.2 no puede activar antes el cierre seguro de
lanzamientos ni cerrarse antes de A05.4. Tampoco puede conservar el límite
Codex, inventar una observación, reutilizar una obsoleta o tratar
ausencia/error/timeout como capacidad disponible.

La selección causal tiene una única interpretación: `application` observa,
ordena de forma determinista y deduplica candidatos opacos y entrega
`CapacityCandidates` al claim. Dentro de la misma transacción, el claim
revalida por CAS y fija/reserva el primer candidato todavía disponible. Devuelve
la `AgentPlacementRef` exacta; pool, launcher y replay no reseleccionan.
Bootstrap registra candidatos y conectores, pero nunca selecciona ni decide
admisión.

Existe además una compuerta causal urgente entre A05.1 y B10. Mientras el motor
microVM aún no esté compuesto, A05.1b rechaza `runtime.isolation=microvm` en
bootstrap antes de construir el adaptador Codex de proceso, con código máquina
estable y mensaje de catálogo i18n. `process` continúa funcionando. B10
sustituye esa compuerta —en el mismo write-set serial de composición— por el
motor Firecracker real; nunca la salta ni la convierte en fallback a proceso.

A05.1c explicita una corrección de planificación, no una reatribución:
`87db76eb` consumió las 49 LOC de producto y 49 de verificación de A05.1, pero
el registro no contiene `runtime.capacity.observation_timeout`. El coste mínimo
demostrado de esa clave es `P=14, V=5`: catorce líneas del mismo objeto de
duración que `observation_ttl`, más comprobaciones de valor por defecto, valor
explícito y rechazo de cero. La prueba canónica existente regenera y compara
getter, schema, referencia, ejemplo y superficie web.

A05.1d retira otra dependencia histórica incorrecta:
`runtime.codex.max_concurrent_executions` no puede gobernar `BudgetPolicy` ni
el despachador de proveedores o aislamientos distintos. El registro canónico
añade `governance.global_process_slots_budget`, con default 70. A05.1d
desacopla el despachador de Codex y migra provisionalmente su techo a esa clave
neutral, conservando `maxConcurrentLaunches` y `ExcludeLaunch`: A04.2 todavía
no ha ligado la reserva física al claim. A07 retira ese mecanismo solo después
de acreditar A04.2; desde entonces ejecuta únicamente un `launch_agent` que
`ClaimNextAction` haya devuelto con reserva física durable. La clave Codex se
mantiene exclusivamente dentro de su conector/pool como defensa local y no es
alias de la clave global.

El símbolo `NNN` significa «siguiente migración libre en el momento de
integrar». El número se asigna bajo exclusión mutua después de releer la cadena;
no se reserva anticipadamente `022` mientras haya trabajo SQLite concurrente
de Orquesta. Las migraciones privadas de `agentmicrovm` usan su propia cadena y
no compiten por número ni transacción con esta.
Los write-sets separados por `+` pueden ejecutarse en paralelo. Toda tarea que
toque `internal/application/state.go`, `config/registry.json`, composición o
la cadena de migraciones se serializa aunque aparezca en otra ola.

En la columna LOC, `P` es código no generado de producto/scripts y `V` son
pruebas, datos y arneses. Los techos suman el trabajo de Orquesta y del
repositorio hermano `agentmicrovm`; no existe un presupuesto externo adicional.
Es un techo, no un objetivo.

Toda ejecución física de B05–B12 o C01–C04 debe llevar, antes de arrancar, un
manifiesto inmutable ligado al digest candidato con cuatro enteros positivos:
`wall_time_limit_ms`, `ram_peak_limit_bytes`, `disk_peak_limit_bytes` y
`agent_token_limit`. C01 rechaza valores ausentes, cero o «ilimitado», comprueba
RAM/disco disponibles y reservas de tokens, y registra solicitado/observado.
RAM y disco ya incluyen el margen operativo declarado; no existe otro margen
implícito. Superar cualquiera de los cuatro límites detiene solo los recursos
exactos del run y obliga a replanificar, nunca a rebajar silenciosamente la ola.

| ID | Capability, invariante y autoridad | Write-set propuesto, dependencias y paralelismo | Contrato reutilizado, función preservada y complejidad retirada/evitada | Verificación, criterio de cierre y presupuesto |
|---|---|---|---|---|
| **A01 — cohortes lógicas** | `ORC-28`. El conjunto listo completo pertenece al `Goal`; el límite físico nunca elimina ni recrea `WorkItems`/`Executions`. Autoridad: `Goal` y `application`. | `internal/goal/v38_ready_set_test.go`, `internal/application/v38_schedule_ready_test.go`, dato estructurado en `acceptance/fixtures/`. Sin dependencia; paralela con A02. | Reusa `RunnableWorkItems`, `ReadyWorkItems`, `scheduleReady`. Preserva conflicto/write-set y orden determinista. Evita gestor de cohortes, paginador de DAG y segunda cola. | Pruebas normales y de propiedades para `1/16/70/500`; negativos de dependencia, ciclo/conflicto y presupuesto. La capacidad parcial no crea ni reemplaza una `Execution` o `Attempt` ya admitidos. Cierra cuando las cuatro cohortes se materializan completas sin tope implícito. `P≤0, V≤138`. |
| **A02 — contrato neutral de capacidad** | `ORC-28`. Capacidad y cuota son observaciones externas durables, no presupuesto ni texto de agente. Autoridad de frescura y política: `application`, usando su reloj. | `internal/application/agent_capacity.go`, `internal/application/agent_placement.go` y pruebas. Sin dependencia; paralela con A01. | La fuente física expresa plazas brutas presentes o ausentes por `SourceRef`+`PoolRef`. La compuerta de cuota expresa estado `available/exhausted/unknown`, `WindowRef`, reinicio, expiración y `RetryAt`; un porcentaje es solo evidencia, no saldo. Ambas conservan revisión, calidad y colocación opaca. Evita almacén, servicio o planificador de cuotas. | Pruebas: una plaza cero no se pierde; ausente no equivale a ilimitado; la cuota no inventa restantes; los estados desconocido, obsoleto o no disponible cierran nuevos lanzamientos; una nueva ventana observada puede reabrir; el reloj del proveedor no decide frescura; la respuesta o el registro textual no se hacen autoridad. `P≤170,V≤238`. |
| **A03 — persistencia canónica de capacidad y cuota** | `ORC-28`. Capacidad física y cuota son hechos durables con papeles distintos: solo la primera admite reserva, consumo o liberación; la cuota por perfil es una compuerta versionada sin saldo interno. `application` decide y escribe mediante `StateRepository`. | Conserva intacta `022_agent_capacity.sql`. Una migración progresiva posterior a A04.1 añade la observación completa de cuota y el vínculo 1:1; después el mismo `StateRepository` implementa lectura y adición con CAS. Orden `A02b → A04.1 → A03.2 → A03.3 → A04.2`. | No añade clase ni ciclo de vida de observación: `reservation.observation_ref` identifica el hecho físico reservable y `quota_observation_ref` conserva el vínculo inmutable. No existe almacén ni operación separada para escribir el vínculo: solo `ClaimNextAction` lo inserta con la reserva. SQLite es local, de desarrollo y de pruebas; PostgreSQL es el único adaptador productivo futuro V31; nunca están activos a la vez. | La base consumió 139/270; Q2 dispone de 100/147 y Q3 de 95/120. Prueba migración progresiva, CAS, idempotencia, reversión, clave externa diferida, aislamiento, reapertura, copia y restauración, ausencia de reserva de cuota y fuente única. Techo `P≤334,V≤537`. |
| **A04 — colocación y reserva física durable en el claim** | `ORC-28`. `application` entrega candidatos opacos ordenados/deduplicados; el claim revalida y fija el primero disponible. Solo ese claim crea una `AgentCapacityReservation` física y liga cuota sin debitarla. Stop/observe siguen reclamables. | `internal/application/{agent_capacity,agent_capacity_state,state}.go`, `internal/adapters/state/sqlite/{claim,agent_capacity}.go` y pruebas. Orden único: `A02b → A04.1 → A03.2 → A03.3 → A04.2`; A04.2 depende además de A05.4 y es serial con A06/A07. | Cada perfil Codex ligado es un placement/pool de capacidad 1; un adaptador sin perfil no presenta candidato por carecer de cuota y colocación durables. Dentro del `BEGIN`, CAS fija el primero válido y crea una reserva+binding 1:1. Held se resta una vez por `source+pool`. Firecracker host queda bajo presupuesto/preflight y leases CID/VM, sin segunda reserva. | Base física 114/53 y binding neutral 46/57 ya consumidos; quedan `P≤489,V≤537`. Prueba agotamiento, stale, carreras, replay, doble contabilidad y ausencia de segunda reserva Firecracker; migra los literales afectados. Techo `P≤649,V≤647`. |
| **A05 — fuentes, colocación, configuración e inyección V38** | `ORC-28`. Proveedor y aislamiento son ejes ortogonales. Una fuente física informa recursos reservables; un controlador persistente por perfil informa cuota. `application` decide frescura, orden, eliminación de duplicados y admisión; el reclamo fija colocación. | Registro y salidas, códec, controlador Codex, `internal/bootstrap/`, `cmd/orquesta/`, internacionalización y aplicación. Depende A02b; A05.3a precede A05.3b y A05.4 precede A04.2. | Por perfil de cuota: un `app-server` persistente y un lector de protocolo acotado. La sesión fija capacidades nulas, solo admite métodos de cuota y descarta errores remotos sensibles. El arranque exige lectura inicial antes del planificador; la misma conexión sirve actualizaciones y olas 5/10/20. | Cero agentes y 5/10/20 agentes sin proceso por lanzamiento; fallo inicial que aborta el arranque y cierre sin huérfanos. Falla cerrado ante información obsoleta o espera agotada. Las guardas impiden importaciones de Firecracker y puertos de agente. `P≤862,V≤628`. |
| **A06 — contrato neutral de preservación** | `ORC-28`. Antes de desmontar, el entorno entra en `preserved_pending_review` con sello e inventario; no es un nuevo estado del `Goal`. Autoridad de aceptación: `application`. | `internal/ports/agent_environment.go`, `internal/application/agent_environment.go`, migración `NNN_agent_environment_receipts.sql`, implementación SQLite específica y pruebas. Depende A03; migración y `state.go` seriales con A04. | El comprobante liga execution/external identity/fence, base del workspace, change-set/bundle CAS, inventario, configuración/rootfs y tiempos. Reusa CAS, refs, efectos y revisión. No expone operación delete/GC en V38. | Normal, comprobante duplicado, digest alterado, ref de otro proyecto, sello ausente, desmontaje prematuro, fence obsoleto y reinicio. Cierra cuando terminalizar un proveedor que exige preservación falla cerrado sin comprobante válido y el material sigue direccionable. `P≤350, V≤350`. |
| **A07 — despachador único con `launch_agent` concurrente** | `ORC-28`. Un despachador global; procesamiento secuencial salvo `launch_agent` ya reclamado. Observación y parada progresan aunque el lanzamiento esté saturado. Autoridad de reclamo, admisión y procesamiento: `application`; el arranque solo compone, inicia y drena. | Base integrada en `7ae98026`; despachador y pruebas integrados en `e4a6a98e`. Depende de A04 para cerrar capacidad durable; los archivos compartidos se tratan en serie. | `ClaimNextAction` solo devuelve un lanzamiento con colocación y reserva física durables. El despachador retira `maxConcurrentLaunches` y `ExcludeLaunch` después de A04.2. Cada lanzamiento admitido usa una gorutina corta ligada a su reclamo. Evita un conjunto global de ejecutores, cola por proveedor y segundo bucle. | Pruebas `1/5/10/16/20`, saturación, observación, parada, reclamo cercado, cierre, carrera y ausencia de giro activo. Los negativos cambian proveedor o aislamiento sin leer `runtime.codex.*` y rechazan cualquier lanzamiento sin reserva. A07 cierra con A04 y el comprobante A08. `P≤165,V≤695`. |
| **A08 — compuerta A neutral** | `ORC-28`. Acredita semántica sin KVM: cohortes, capacidad parcial, órdenes/mailbox, observación, parada, preservación y recuperación. Autoridad: contrato de aceptación V38, todavía sin promoción. | `acceptance/v38_agent_runtime_elastic_core_test.go`, datos V38 y sustituto contractual bajo la superficie de prueba existente. Depende A01–A07. | Reusa sustituto Agent, execution sessions/mailbox, CAS, reloj y almacén reales. No introduce sustituto productivo ni interpreta ACK/texto como efecto. | E2E neutral `1/16/70/500`; capacidad menor que conjunto listo; refresco; orden/ACK exactos posteriores al lanzamiento; parada; reinicio en fronteras de intento/efecto/comprobante; `-race`; suite sin `/dev/kvm`, binario Firecracker ni artefacto de huésped. Cierra solo con comprobante de compuerta A ligado a digests. `P≤0, V≤600`. |
| **B01 — selección explícita y frontera de composición** | `ORC-28`. Compuerta que verifica que proveedor, aislamiento, fuente física, cuota y aplicación hermana son ejes distintos. La implementación pertenece a A04/A05/B03/B04/B10; B01 no vuelve a programarla. | Solo prueba contractual compacta después de A05/A08/B03/B04 y antes de B10. | Reutiliza selección A04, fuentes/controlador A05 y contrato cruzado B04. Verifica que la futura composición B10 solo puede elegir el conector local, sin importar el motor hermano. | Negativos prueban que `process` no intenta conectar `agentmicrovm`, que una microVM incompleta no cae a proceso, que el controlador anfitrión solo observa cuota y que no reaparece un límite Codex global. `P≤0,V≤50`. |
| **B02 — estado físico privado y concesiones temporales de CID** | `ORC-28`. CID sirve para encaminamiento, nunca identidad; la concesión temporal durable tiene revisión, cerca, expiración y recuperación antes del enlace físico. `RuntimeJournal`/`CIDLease` pertenecen solo a `agentmicrovm` y no comparten autoridad con `StateRepository`. | Repositorio hermano: migración privada, adaptador del registro físico/CID y pruebas. Depende B03; paralela con B05/B06/B08/B09. Orquesta solo caracteriza y retira sus puertos y asignador físicos solapados al migrar todos los consumidores. | Reimplementa detrás del contrato hermano las lecciones de asignación, cerca, expiración y recuperación; no copia el paquete SQL actual. El binario compone exactamente un conector de estado físico privado. No hay esquema, transacción, migración, base de datos, sistema de archivos ni bloqueo común con Orquesta, ni puente permanente. | Asignación concurrente, vuelta de rango/agotamiento, liberación obsoleta, caída/reapertura independiente, concesión expirada, CID ocupado, recuperación exacta y carreras. Guardas cruzadas prohíben cadena de conexión, esquema, ruta e importación compartidos. Cierra con una fuente física privada activa y solo referencias/comprobantes por protocolo. `P≤200, V≤250`. |
| **B03 — proyecto hermano y contrato local v1** | `ORC-28`. Agente MicroVM es una aplicación independiente y reutilizable; su autoridad termina en recursos microVM y comprobantes físicos. Nunca conoce `Goal`, `WorkItem`, `Execution`, ciclo de vida, permisos o presupuestos. | Caracterizar y completar el candidato publicado `https://github.com/aavidad/agente_microvm`: crate `agente_microvm`, binario Rust `agente-microvm`, cliente Go, configuración, documentación, API `v1`, conjunto contractual y protocolo `agentmicrovm.local.v1` por HTTP/1.1 sobre socket Unix. Depende A08; primera tarea B. El baseline `bdba503` se mide aparte según el [ADR de baseline](adr_v38_baseline_agente_microvm_2026-08-02.md). | El contrato neutral comprende referencias opacas y `Launch/Observe/SendOrder/Events/Stop/Preserve/Recover/Close`, pero capacidades anuncia solo operaciones reales: B03 congela la frontera; B05 activa eventos y B07/B10 recuperación explícita. No existen stubs exitosos ni fallback. Los puertos cohesivos privados cubren registro físico/CID, reloj, confianza y servicios huésped. No recibe `CredentialStore`, atestador, CAS de Orquesta, PostgreSQL ni tipos Orquesta. El modo `serve` es local, supervisable y sin TCP remoto. | Guardas de módulos y dependencias, negociación compatible/incompatible, tramas acotadas, propietario/permisos del socket, arranque/cierre y consumidor mínimo. Cierra la base contractual cuando ambos repositorios compilan por separado y no existe importación, `replace` productivo, ruta, secreto o almacén común; las operaciones diferidas cierran en sus tareas causales. Presupuesto de delta posterior al baseline: `P≤250, V≤300`. |
| **B04 — contrato cruzado entre repositorios** | `ORC-28`. La independencia se demuestra contra un candidato versionado real de `agentmicrovm`, no mediante una extracción futura ni un `replace` al árbol Orquesta. | Prueba de aceptación del conector en Orquesta, conjunto de pruebas del protocolo en el repositorio hermano y manifiesto de compatibilidad de ambos resúmenes criptográficos. Depende B02/B07/B08/B09. | Construye el binario hermano, lo sirve en un socket temporal privado y ejecuta un consumidor Orquesta solo por `agentmicrovm.local.v1`. Producción y evidencia no usan `replace`, importaciones cruzadas ni un módulo temporal que encubra el límite. | Negativos de versión, método, trama, socket, importación/dependencia transitiva, acceso a almacenes/rutas y resumen criptográfico de binario distinto. Cierra con pruebas correctas en ambos repositorios y compatibilidad ligada a sus dos resúmenes criptográficos. `P≤50, V≤350`. |
| **B05 — `rootfs` y protocolo del huésped reproducibles** | `ORC-28`. Protocolo y supervisor del huésped son neutrales y pertenecen a `agentmicrovm`. Cada microVM ejecuta su propio agente ejecutor; nunca comparte proceso, socket, `HOME`, hilo, turno, códec ni estado con el controlador Codex anfitrión de Orquesta. | Repositorio hermano: `v1/guestproto`, supervisor, enlace de proveedor bajo la imagen y `tools/agentmicrovm-rootfs/`, manifiesto/SBOM/licencias y pruebas. Depende B03 y de un artefacto de esquema Codex neutral fijado; paralela con B02/B06/B08/B09. | `guestproto` versiona arranque/control/eventos; el supervisor aporta reconexión, deduplicación y contrapresión. El enlace huésped no importa `internal/adapters/agent/codex/appserver`; implementa el esquema neutral versionado y arranca su instancia propia. | Construcción reproducible, resumen criptográfico, aislamiento de procesos, reinicialización, incompatibilidad, trama parcial/repetición/desorden y ausencia de WebSocket/tmux/secretos. Cierra con núcleo/`rootfs`/`init` sellados y agente ejecutor Codex real independiente. `P≤490,V≤320`. |
| **B06 — artefactos por el protocolo local** | `ORC-28`. Orquesta conserva autoridad sobre espacio de trabajo, base, instantánea, conjunto de escritura, CAS e integración; `agentmicrovm` solo recibe y devuelve contenido por referencia/resumen criptográfico y no interpreta repositorios. | Tramas/flujos de artefactos en `agentmicrovm.local.v1`, conector Orquesta y extensión mínima del productor de espacio de trabajo/Git; pruebas en ambos repositorios. Depende A08/B03; paralela con B02/B05/B08/B09. | Orquesta resuelve su almacén y transmite contenido verificado con tipo de contenido. Para cargas grandes puede pasar un descriptor anónimo sellado mediante `SCM_RIGHTS`; nunca envía rutas. `agentmicrovm` no importa CAS ni abre el sistema de archivos de Orquesta. El resultado vuelve por el mismo contrato. | Paquete válido/corrupto/truncado, base errónea, recorrido de ruta, enlaces/dispositivos, expansión abusiva, referencia entre proyectos, descriptor no sellado, repetición idempotente y conjunto de cambios fuera del conjunto de escritura. Cierra con anfitrión→socket→huésped→socket→integración, sin almacén o ruta compartidos. `P≤400, V≤350`. |
| **B07 — motor físico Firecracker en `agentmicrovm`** | `ORC-28`. El motor hermano crea una microVM por `RunRef`, observa y reconcilia recursos físicos; no conoce el proveedor alojado ni cierra o reabre trabajo de la aplicación llamante. | Repositorio hermano: motor Firecracker, lanzador, observador, recuperación, primitivas neutrales justificadas y conjunto contractual. Depende B02/B03/B05/B06. | Implementa lanzamiento, vsock, observación y desmontaje físicos. Consume su registro físico/CID privado, flujos de protocolo y verificación de concesiones; persiste identificadores opacos en su almacenamiento propio. No importa Orquesta, CAS ni credenciales. Sin NIC/TAP/puente/NAT ni VM compartida. | Dos lanzamientos→dos VMs, fallo por paso, PID reutilizado, fugas, recurso inválido, KVM ausente, agotamiento de memoria, observación ambigua, reinicio y carreras. Cierra con identidad exacta, neutralidad y cero recursos propios huérfanos. `P≤500, V≤450`. |
| **B08 — servicio huésped y concesión neutral firmada** | `ORC-28`. Orquesta autoriza la intención; `agentmicrovm` verifica una concesión neutral firmada, de un solo uso, y enlaza la VM exacta a servicios huésped. Ninguno obtiene autoridad de ciclo de vida ajena. | Caracteriza y retira `agent_microvm_launch_auth` y `firecracker/networkauth` de Orquesta; implementa concesión/verificador e intermediario físico en el repositorio hermano, y firmante/traductor en el conector Orquesta. Depende A08/B03; paralela con B02/B05/B06/B09. | La concesión liga audiencia v1, `RunRef`, cerca, expiración y resumen criptográfico completo del plan. Orquesta firma tras permisos/atestación; `agentmicrovm` verifica con confianza pública configurada. No cruza `CredentialStore`, secreto HMAC, credencial de proveedor ni tipo de atestador. MCP/buzón causal/artefactos viajan como mensajes/referencias autorizados por el conector. | Repetición/concurrencia, firma/audiencia/plan/cerca/expiración erróneos, alcance cruzado, reconexión, confirmación/reversión, reinicio, orden posterior y artefacto por referencia. Guardas prohíben secretos, `CredentialStore` e importaciones cruzadas. Cierra cuando solo la VM exacta consume la concesión una vez. `P≤550, V≤450`. |
| **B09 — salida controlada en `agentmicrovm`** | `ORC-28`. El proyecto hermano aplica una concesión neutral de red y emite comprobantes físicos; Orquesta decide autorización, presupuesto y registro antes/después. Única salida: `controlled_egress_proxy` por vsock. | Repositorio hermano: salida Firecracker y protocolo; conector Orquesta traduce política a `EgressGrant`. Depende B03; paralela con B02/B05/B06/B08. | `EgressGrant` incluye destinos, puertos, tiempo/contenido y referencia opaca. El motor revalida DNS/redirecciones y bloquea bucle local, RFC1918, ULA, enlace local/metadatos. No recibe tipos de permiso/presupuesto Orquesta ni secretos. | Permitir/negar, revinculación DNS, redirección privada, IPv4/IPv6, exceso, tiempo de espera, comprobante perdido, presupuesto agotado, VM vecina y entrada negativa. Cierra sin NIC del huésped y con comprobantes idempotentes. `P≤600, V≤550`. |
| **B10 — conector de agente y composición única de Orquesta** | `ORC-28`. `application` decide ciclo de vida, capacidad, permisos y efectos; un adaptador traduce `AgentLauncher/Observer/Controller/Shutdown` al protocolo local v1 sin ganar autoridad. | `internal/adapters/agent/agentmicrovm/`, módulo Go público de Agente MicroVM, `internal/bootstrap/agent_provider.go`, `cmd/orquesta/` y pruebas. Depende B01/B04/B06; única tarea de composición, serial sobre la compuerta A05.1b. | El adaptador consume el cliente Go independiente ligado a revisión; no vuelve a implementar transporte. Solo recibe punto de conexión, límites y confianza/firma tipados. La composición no compone Firecracker, registro físico, CID, intermediario o almacenes físicos. | Idempotencia, cerca, reinicio independiente, socket perdido, versión incompatible y modo `-race`; `process` sigue igual, `microvm` completo llega al binario hermano y la composición parcial falla antes de construir Codex/proceso. Cierra con Codex real dentro de la microVM. `P≤280, V≤430`. |
| **B11 — parada y cierre exactos por protocolo** | `ORC-28`. Orquesta deja de admitir reclamaciones, detiene el pulso, espera lanzamientos reclamados y después pide `Stop`/`Shutdown`; `agentmicrovm` actúa cooperativa y luego forzadamente sobre el identificador físico opaco exacto. | Métodos v1 de control/parada en el repositorio hermano; traducción en `internal/adapters/agent/agentmicrovm/` y corrección mínima de la composición de arranque Orquesta. Depende B10. | La petición usa `RunRef` y cerca; el servicio privado reconcilia PID/hora de inicio, `cgroup` y CID y devuelve comprobante. Orquesta no lee el registro físico ni termina procesos físicos. Evita `pkill`, terminación por nombre/cohorte y uso de un cliente cerrado. | Cooperativa, tiempo de espera→forzada, PID reutilizado, comprobante perdido, repetición, cerca obsoleta, par intacto, reinicio independiente durante parada, lanzamientos en curso y limpieza exacta. `P≤300, V≤350`. |
| **B12 — sello, inventario y compuerta B** | `ORC-28`. `agentmicrovm` devuelve por protocolo conjunto de cambios, sello e inventario físicos; Orquesta verifica y persiste `preserved_pending_review` antes de confirmar desmontaje/liberación. No borra. | Método v1 de preservación y transmisión en el repositorio hermano; `internal/adapters/agent/agentmicrovm/preserve.go`, integración A06/B06 y aceptación de una VM. Depende B06/B10/B11. | El servicio emite inventario físico neutral y contenido/resúmenes criptográficos; Orquesta añade espacio de trabajo/configuración/herramientas, valida el conjunto de escritura y persiste su comprobante/CAS. No comparten espacio de trabajo, configuración, `rootfs`, evidencia TestAttestor, sistema de archivos ni API de borrado. | Una microVM real, orden posterior, observación, parada, sello, inventario y desmontaje; negativos de resumen criptográfico/conjunto de escritura/referencia/concesión y reinicio a ambos lados antes/después del sello. Cierra con comprobante de compuerta B ligado a ambos candidatos. `P≤400, V≤500`. |
| **C01 — arnés y comprobación previa de olas** | `ORC-28`. El anfitrión declara capacidad real antes de la ola y nunca rebaja silenciosamente el peldaño. Todos los recursos pertenecen a un run aislado. | `acceptance/v38_agent_firecracker_wave_harness_test.go`, script/runbook acotado, manifiesto de presupuesto y datos de recursos. Depende B12. | Reusa perfil de anfitrión solo como lección; mide CPU/RAM/disco/KVM/CID/FD/cgroup y digests. Sella por candidato los cuatro límites exactos `wall_time_limit_ms`, `ram_peak_limit_bytes`, `disk_peak_limit_bytes`, `agent_token_limit`. | Positivos/ausentes/excedidos, reserva de disco/tokens, RAM disponible, nombres/temporales exactos, aborto y limpieza inventariada. Si no caben 20 o falta un límite, la compuerta queda bloqueada. `P≤300, V≤300`. |
| **C02 — olas físicas de los mismos candidatos** | `ORC-28`. Ejecuta `1/5/10/16/20` con una microVM por agente y los mismos resúmenes criptográficos candidatos de Orquesta y `agentmicrovm`; el conjunto listo lógico puede ser mayor. Es compuerta física independiente de B04 y TestAttestor. | Arnés C01 y resultados estructurados bajo `product/evidence/candidates/`; usa sin cambios el manifiesto de tiempo/RAM/disco/tokens y no edita el plan de producto. Depende C01. | Reusa despachador, capacidad, conector local v1, motor hermano, intermediario/proxy, observación, buzón causal, parada y preservación. Evita reconstrucción entre pasos, selección manual y métricas desde registros. | Por ola registra `present`, inicio, fin, duración, `RunRef`, peldaño y ambos resúmenes criptográficos para siete latencias: **solicitud de capacidad, aprovisionamiento, lanzamiento, disponibilidad, parada cooperativa, parada forzada y sello/inventario**. Cada peldaño ejecuta subcasos separados cooperativo y forzado controlado; cero no significa ausencia. Registra también RAM, disco y tokens solicitados/observados. Cierra solo si pasan cinco peldaños dentro de presupuesto. `P≤150, V≤400`. |
| **C03 — matriz adversarial, carreras y recuperación** | `ORC-28`. Seguridad, causalidad e idempotencia se mantienen bajo fallos reales; estados ambiguos nunca disparan reintento ciego ni borrado. | Pruebas de aceptación adversarial V38 y datos de inyección de fallos; no producto salvo corrección causal separada. Depende C02. | Reusa casos A/B: fuente stale/unknown, nueva ventana, caída en intento/efecto/comprobante, replay de credencial, CID ABA, DNS rebinding, VM vecina, comprobante de parada perdido y sello incompleto. Evita una suite feliz como evidencia total. | `go test -race`, reinicio/reapertura, parada controlada de procesos propios, negativos de auth/proyecto/traversal/red y escaneo final. Toda corrección repite C02 con nuevo digest candidato. Cierra sin P0/P1 abierto. `P≤50, V≤650`. |
| **C04 — sello y revisión independiente** | `ORC-28`. Evidencia pertenece a los mismos árboles fuente, binarios, `rootfs` y configuraciones efectivas; los comprobantes quedan fuera de los sujetos y los referencian. Autoridad: flujo de publicación/TestAttestor más revisión primaria y adversarial. | Manifiesto/índice inmutable de ambos candidatos, comprobantes de aceptación y notas de revisión; ningún cambio funcional. Depende C03. | Reusa TestAttestor sin darle red/vsock y el modelo de evidencia actual. Evita el resumen criptográfico autorreferente imposible, el comprobante manual y la mezcla de candidatos. | Compuertas globales en ambos repositorios: `git diff --check`, pruebas unitarias/contractuales/de integración, `go test`, `vet`, carreras relevantes, arquitectura/configuración/autorización, copia de seguridad/restauración/cierre y cero procesos propios. Cierra con dos revisiones y resúmenes criptográficos coincidentes A+B+C. `P≤50, V≤300`. |
| **C05 — promoción canónica atómica** | `ORC-28`. Solo tras C04, el plan de producto y la evidencia de la misma revisión promueven la vertical completa; ningún modelo de lectura derivado puede acreditar por sí solo. | Obligatorios en el conjunto de cambios Orquesta: `product/roadmap.json`, `command`/`fixture`/`test` reales, comprobante/índice de `product/evidence/`, manifiesto y resumen criptográfico inmutables de la versión hermana `agentmicrovm`, decisión `agent_microvm_network`, `product/knowledge/tooling_adoption_v1.json`, `product_roadmap_test.go` y guardas `acceptance/v38_agent_runtime_elastic_plan_*`. `product/capabilities.json` solo se regenera si la composición/publicación canónica lo exige y referencia la misma evidencia. Depende C04; serial final. | En `product/roadmap.json`, `AC-V38-AGENT-RUNTIME-ELASTIC` pasa `planned→executable` con los campos reales `command`, `test_ref`, `fixture` y `receipt`; `ORC-28` pasa `declared→accredited` con `evidence_refs`; `agent_microvm_network` pasa `planned_not_applied→applied`. `tooling_adoption_v1.json` cambia Firecracker de `candidate` al estado acreditado, soportado por la misma evidencia y sin volverse autoridad. | Sustituye las guardas vigentes que exigen contrato V38 `planned` y sin comprobante, ausencia de evidencia V38, `ORC-28` `declared`, decisión `planned_not_applied` y Firecracker `candidate`; las nuevas guardas comprueban referencias/resúmenes criptográficos coincidentes de ambos repositorios. Si falta un campo real, comando ejecutable o comprobante A+B+C, nada se promueve. Cierra V38 en una revisión Orquesta indivisible que referencia el candidato hermano ya sellado. `P≤100, V≤250`. |

## Ejes ortogonales y alcance real de proveedor

Proveedor, aislamiento, transporte, persistencia, artefactos y control son seis
conectores hexagonales ortogonales. El proveedor implementa el contrato neutral
de sesión, turnos y eventos; el aislamiento decide si la ejecución vive en
proceso o en una microVM; transporte mueve mensajes; persistencia conserva el
registro durable; artefactos mueve contenido por referencia y resumen
criptográfico; control
observa, reorienta, interrumpe y detiene. Ninguno decide o incorpora a los
demás, y la composición los ensambla mediante puertos tipados. Para proveedor y
aislamiento elige una opción de cada eje, pero no crea adaptadores
`codex_process`, `codex_microvm`, `claude_process`, `claude_microvm` ni una
matriz N×M equivalente.

`application` decide admisión, autorización y mutaciones mediante puertos.
`StateRepository` pertenece a Orquesta. El `RuntimeJournal` y `CIDLease`
físicos pertenecen a `agentmicrovm`; son autoridades y almacenamientos
separados. La composición de arranque de Orquesta únicamente construye el
cliente del socket y
rechaza dos fuentes de mando propias: no decide política, no escribe ciclo de
vida y no habilita escritura doble. Ninguna composición comparte base de datos,
sistema de archivos, esquema, migración, transacción o ruta alternativa entre
los dos proyectos.

El protocolo y supervisor neutrales viven en el repositorio hermano, separados
del paquete Firecracker para servir a otros aislamientos. La API v1 no exporta
enumeraciones, DTO, configuración o errores propios de Codex, Claude, Gemini,
Ollama o un agente local. El traductor concreto queda en la imagen del
proveedor y se sustituye sin cambiar el motor físico ni el protocolo local
Orquesta↔`agentmicrovm`.

V38 debe ejecutar Codex real porque es el proveedor candidato configurado para
esta acreditación. Esa prueba demuestra únicamente `ORC-28` con la composición
Codex+Firecracker; no implementa ni acredita la matriz de proveedores de V25,
ni permite afirmar soporte de Claude, Gemini, Ollama o agentes locales. Esos
proveedores deberán superar posteriormente su propio contrato y pruebas de V25
sin modificar la API física neutral.

El códec Codex interno de A05 fija explícitamente la versión y el esquema
compatibles de `codex app-server`, las tramas JSONL, los identificadores de
petición, la correlación de eventos, los límites y
`initialize`→`initialized`. Lo consumen el controlador anfitrión de cuota y el
agente ejecutor de proceso de Orquesta cuando corresponda. Para el huésped se
genera o publica un artefacto neutral de esquema/versiones; el repositorio
hermano lo implementa sin importar el paquete `internal`, compartir códec o
compartir
estado vivo. Cada consumidor arranca una instancia separada, con proceso,
`stdio`, socket, `CODEX_HOME`, credenciales, hilo y turno propios. El
controlador anfitrión negocia capacidades nulas y tiene una lista cerrada de
observación de cuota/elegibilidad; no puede usar
métodos de hilo, turno, órdenes posteriores, parada de agente, espacio de
trabajo, buzón causal ni
artefactos. Los errores remotos se validan y su contenido se descarta antes de
salir del códec. Si el protocolo real no ofrece cuota estructurada acreditable, A05
queda bloqueada y cerrada a lanzamientos.

B05 adapta `guestproto` al artefacto neutral de esquema y arranca
`codex app-server` dentro del huésped. Cada arranque y reinicio completa la
inicialización antes de cualquier operación. Una petición iniciada por el
servidor sin manejador/autorización explícitos no se confunde con una respuesta
y falla cerrada. Quedan prohibidos WebSocket, tmux, lectura heurística de
terminal, extracción heurística de registros, fichero de cuota y órdenes
iniciales sin canal
posterior.

## Microencargos operativos de los padres grandes

Estas divisiones son hijos operativos de las tareas de la tabla anterior: no
añaden presupuesto, otra capability ni otro cierre. Cada hijo debe producir un
commit pequeño y una revisión propia. Los hijos que comparten fichero son
seriales; solo los declarados paralelos pueden abrirse a la vez. Cada suma
coincide exactamente con el techo de su padre. Ningún encargo pendiente supera
`P=200, V=200`; las bases ya integradas y A05.3a informan su medida real y no
autorizan otro write-set de ese tamaño.

### A02 — `P=170, V=238`

| Hijo | Write-set y resultado | Orden | Presupuesto |
|---|---|---|---:|
| A02.0 | Base `7e232072`: demanda y admisión multidimensionales sin perder ceros presentes. | Integrada, no acreditante por sí sola. | `P=77, V=120` consumidos |
| A02.1 | Separación neutral entre capacidad física, cuota no reservable y colocación opaca. | Integrada, no acreditante por sí sola. | `P=68, V=77` consumidos |
| A02.2 | Materialización CAS de la observación de cuota con revisión, idempotencia y colocación exactas. | Integrada en `fdfab8c7`; abre A03.2/A03.3. | `P=25, V=41` consumidos |

### A03 — `P=334, V=537`

| Hijo | Write-set y resultado | Orden | Presupuesto |
|---|---|---|---:|
| A03.1 | Base ya integrada en `71f4a827`: observación, reserva física y `022_agent_capacity.sql`, incluida su prueba canónica. Se conserva y se caracteriza; no se vuelve a presupuestar ni se edita la migración aplicada. | Completada parcial, no acreditante. Medición neta del cambio ya integrado. | `P=139, V=270` consumidos |
| A03.2 / Q2 | Migración progresiva con el hecho completo `AgentQuotaObservationRecord` y `AgentPlacementBinding` 1:1 de cuatro columnas, clave externa diferida a reserva y cuota, colocación y revisión exactas; sin clase, saldo ni ciclo de vida de cuota. | Después de A04.1; exclusión sobre migraciones. No añade operación de escritura del vínculo. | `P=100, V=147` |
| A03.3 / Q3 | `StateRepository` obtiene la observación vigente por colocación y hace una adición con CAS e idempotencia. SQLite implementa ese mismo puerto para uso local, desarrollo y pruebas; no aparece otro almacén. | Después de Q2 y antes de A04.2. PostgreSQL implementará el mismo contrato productivo en V31. | `P=95, V=120` |

### A04 — `P=649, V=647`

| Hijo | Write-set y resultado | Orden | Presupuesto |
|---|---|---|---:|
| A04.1 | Base ya integrada en `33802c27`: estado de observación/reserva que ahora se caracteriza y migra al binding 1:1, sin segundo lifecycle. | Después de A02b; habilita A03.2, nunca A04.2 directamente. Completada parcial, no acreditante. | `P=114, V=53` consumidos |
| A04.1b | Binding neutral exacto ya integrado en `4ccfedc0`; no posee writer, store ni lifecycle propios. | Después de A04.1; caracteriza Q2/Q3/A04.2. | `P=46, V=57` consumidos |
| A04.2 / Q4 | `a6c326d7`: `CapacityCandidates` ordenados/deduplicados en `ClaimRequest`; dentro del `BEGIN IMMEDIATE`, CAS revalida en orden y fija/reserva el primero disponible. La misma transacción materializa o repite la observación, crea la única reserva física, liga cuota y devuelve la colocación exacta. | `exercised` junto con A04.3/A04.4; abre A06. No acredita V38 ni `ORC-28` antes de A08 y las compuertas B/C. | Consumido `P=489,V=515` de la bolsa conjunta `P≤489,V≤537` |
| A04.3 | `a6c326d7`: receipt aceptado consume; prueba estructural de no aplicación libera; efecto ambiguo cuarentena; terminalización y reconciliación liquidan solo transiciones permitidas. Held se resta una vez por `source+pool`; cuota, preflight Firecracker y leases CID/VM no crean otra reserva. | `exercised` en el mismo gate indivisible Q4; serial con A06 ya resuelta. | Incluida en el consumo conjunto `P=489,V=515` |
| A04.4 | `a6c326d7`: reinicio, lease expirado con cerca posterior, recuperación ambigua, carrera del último candidato, replay sin reselección, doble contabilidad, aislamiento de proyecto y exactamente una `AgentCapacityReservation` por launch. El pool Codex resuelve solo la colocación fijada. | `exercised`; suites completas y focales `-race` verdes. La siguiente dependencia causal es A06. | Incluida en el consumo conjunto `P=489,V=515` |

### A05 — `P=862, V=628`

| Hijo | Write-set y resultado | Orden | Presupuesto |
|---|---|---|---:|
| A05.1 | `config/registry.json` y salidas generadas: ejes `provider`/`isolation`, fuente, `runtime.capacity.observation_ttl` y máximo de lectura del informe. Ya integrado en `87db76eb`; sus `P=49, V=49` están consumidos, pero el lector de informe y su clave se retirarán en A05.1d/A05.3b. | Primero y serial sobre registro; completado parcial, no acreditante. | `P=49, V=49` |
| A05.1b | `internal/bootstrap/runtime.go`, `cmd/orquesta/main.go`, catálogos `es.json`/`en.json`, manifest y pruebas compactas/reutilizadas: antes de construir renderizador, credenciales o Codex/proceso, `microvm` devuelve exactamente `bootstrap.runtime_isolation_not_composed`, presentado con `error.bootstrap.runtime_isolation_not_composed`; la CLI conserva el código y no lo degrada a `internal`. Cero fallback/recursos; `process` sigue verde. | Después de A05.1; antes de A05.4 y permanece serial con B01/B10 hasta que B10 lo sustituya. | `P=30, V=21` |
| A05.1c | `config/registry.json` y proyecciones generadas: añadir `runtime.capacity.observation_timeout` positivo, separado de `observation_ttl` y de `runtime.codex.*`; comprobar valor por defecto, valor TOML explícito, rechazo de cero y sincronización canónica de todas las proyecciones. | Después de A05.1; serial sobre registro y antes de A05.4. | `P=14, V=5` |
| A05.1d | Registro/bootstrap/scheduler: presupuesto global neutral, separación del límite Codex y tamaño máximo de trama `app-server`; retira la clave de informe sin alias ni fallback. | Integrada; serial sobre registro/bootstrap/scheduler. | `P=21, V=44` consumidos |
| A05.2 | Fuente física configurada y candidatos opacos: slots brutos por `source+pool`, ventanas, ceros presentes, frescura y `AgentPlacementRef`; declara si la medida es bruta y rechaza la doble resta. El observador inmutable fue sustituido por la fuente renovable de A05.2b. | `exercised` por el consumidor transaccional Q4; no acreditada aisladamente antes de la compuerta A. | `P=53, V=64` de la base sustituida y compensada por A05.2b |
| A05.2b | Sustituye la observación inmutable por fuentes renovables de capacidad bruta, cataloga `placement→source+pool` sin cuenta o ruta, obtiene la cuota vigente y entrega a `ClaimRequest` candidatos físicos sin revisión, ordenados y deduplicados. `e7f9e60e`+`993e3d03`; catálogo vacío conserva V23 hasta el cutover. | `exercised` junto con `a6c326d7`; el claim es ya su consumidor real. No acredita V38 antes de A08 y B/C. | `P=180,V=120` consumidos, trasladados de B10.3/B10.4 |
| A05.3a | `internal/adapters/agent/codex/appserver/`: codec JSONL acotado con IDs/correlación, inicialización oficial, capacidades nulas, métodos de cuota exactos, descarte de errores remotos y fallo terminal por exceso. No arranca procesos ni importa Firecracker/application/config. | Tras A02b; precede A05.3b y B05.3. | `P=278, V=179` |
| A05.3b | Traductor y controlador Codex anfitrión solo de cuota: un `app-server` persistente por perfil, lectura inicial, eventos, reconexión y rotación con cierre exacto; elimina el fichero y la ruta alternativa. | Integrada hasta `f2cb965b`; disjunta de la fuente física y no acreditante por sí sola. | Envolvente conjunta A05.3b+A05.4 consumida: `P=234,V=117`, dentro de `P≤237,V≤146` |
| A05.4 | Application/bootstrap inicia controladores antes del planificador, espera lectura inicial, ordena/deduplica candidatos y recoge todos los recursos ante fallo o shutdown. `e7f9e60e`+`993e3d03` conectan el productor renovable y `a6c326d7` lo consume en el claim. | `exercised`; A04 cerró el consumidor transaccional. No se eleva a `accredited` antes de la compuerta A. | La porción previa conserva `P=234,V=117`; la conexión final está medida en A05.2b |

El techo acumulado de A05 es `P=862,V=628`:
`49+30+14+21+53+278+237+180=862` y
`49+21+5+44+64+179+146+120=628`. A05.3b y A05.4 conservaron tareas causales
separadas y la porción existente consumió conjuntamente `P=234,V=117`. Las tres
líneas productivas y veintinueve de verificación nominalmente libres no bastaban
para el catálogo, los observadores vivos y la inyección. A05.2b consumió
exactamente la reasignación `P=180,V=120`; el contrato compartido del claim
consumió `P=44,V=107` de A04.2. No se añadió otro almacén, writer, planificador
o bucle de dominio.

### A06 — `P=350, V=350`

| Hijo | Write-set y resultado | Orden | Presupuesto |
|---|---|---|---:|
| A06.1 | `4e5a7ae3`: contrato neutral en `internal/ports/agent_environment.go` y modelo/validación en `internal/application/agent_environment.go`: estado único `preserved_pending_review`, ejecución e identidad externa exactas, cerca, workspace/base, change-set o bundle CAS, inventario, sello, configuración, rootfs y tiempos. No expone delete/GC. | `exercised`; reusa refs y comprobantes existentes, sin lifecycle ni acción adicional del scheduler. | Consumido `P=100,V=77` |
| A06.2 | `af0639de`: migración progresiva 025, método cohesivo del mismo `StateRepository`, SQLite, lectura y recovery. La repetición exacta es idempotente; proyecto, digest, referencia, cerca o payload distintos fallan cerrados. | `exercised`; serial sobre migraciones y estado. | Consumido `P=178,V=128` |
| A06.3 | `2831d3c2`: la aceptación de lanzamiento conserva de forma durable si el entorno exige preservación; la migración 026 lo hace write-once y toda escritura terminal aplica la misma compuerta de aplicación. Sin comprobante causal exacto no terminaliza. | `exercised`; abre A07/A08. B12 conectará después el efecto físico que produce el comprobante. | Consumido `P=72,V=42` |

La suma `100+178+72=350` y `100+150+100=350` conserva el techo A06. La
corrección presupuestaria del ADR V38 traslada `P=28` de A06.3 a A06.2: el
hecho durable necesita codificar y recuperar sus 27 campos causales sin JSON
opaco, mientras la compuerta terminal reutiliza ese lector y no añade otro
writer. Ambos hijos permanecen por debajo de 200 LOC. El comprobante se
persiste como hecho del `StateRepository`; no es otro agregado,
cola o estado de `Goal`, y no autoriza retirada automática.

A06 consume `P=350,V=247` de su techo `P=350,V=350`. Sus pruebas cubren
registro, repetición exacta, reinicio, alteración, proyecto y cerca ajenos,
material direccionable y rechazo de toda terminalización previa al
comprobante. La suite SQLite completa quedó verde sobre el mismo candidato.
La incidencia test-only `BUG-REBUILD-20260802-001` fija una barrera causal en
la prueba concurrente del atestador para que no confunda la acción posterior
con una violación de fencing. A06 sigue sin acreditar por sí sola `ORC-28` ni
V38: la siguiente dependencia causal es A07 y el cierre de la compuerta A
pertenece a A08.

### A07 — `P=165, V=695`

| Hijo | Write-set y resultado | Orden | Presupuesto |
|---|---|---|---:|
| A07.1 | `7ae98026`: separa `ClaimNextAction` de `ProcessClaim`, conserva el reclamo cercado exacto y permite filtrar lanzamientos sin frenar acciones de control. | Base caracterizada; dependía del cierre durable de A04. | Consumido `P=21,V=363` |
| A07.2 | `e4a6a98e`: un único despachador procesa en serie salvo `launch_agent`, que continúa en una gorutina corta ligada a su reclamo; los errores vuelven al mismo canal y la parada drena trabajo propio. | Base caracterizada; no cerraba mientras mantuviera el límite transitorio. | Consumido `P=78,V=332` |
| A07.3 | `fc5d920c`: retira del despachador `maxConcurrentLaunches`, la lectura de `governance.global_process_slots_budget` y el uso de `ExcludeLaunch`. Solo el claim, ya ligado a reserva física durable por A04, decide cuántos lanzamientos entrega. | `exercised`; abre A08 y no acredita por sí solo `ORC-28`. | Retirada neta `P=-15,V=-6` |

A07 consume por tanto `P=84,V=689` de `P=165,V=695`. Las cohortes físicas
de prueba `1/5/10/16/20` conservan una demanda lógica de 500, stop y observe
progresan durante saturación, el reclamo cercado llega intacto, no existe giro
activo y la parada espera la limpieza de todos los lanzamientos propios. La
suite bootstrap completa, cinco repeticiones focales con detector de carreras,
aplicación y `vet` quedaron verdes. La capacidad física y el presupuesto
durable permanecen autoridades distintas; bootstrap no vuelve a imponer
política central.

### A08 — `P=0, V=600`

| Hijo | Write-set y resultado | Orden | Presupuesto |
|---|---|---|---:|
| A08.1 | `e6352ec2`: aceptación neutral con SQLite y almacén de artefactos reales, agente sustituto contractual y cohortes lógicas `1/16/70/500`. Reclama solo la capacidad física `1/5/10/20`, conserva las reservas al reabrir y vuelve a ofrecer el mismo trabajo tras refrescar una observación obsoleta. | `exercised`; cierra la comprobación neutral y abre B03 sin arrancar KVM, Firecracker ni Agente MicroVM. | Consumido `P=0,V=348` |
| A08.2 | La misma prueba exige por nombre los contratos ejecutables de orden y ACK de buzón, parada, recuperación en fronteras causales, reserva exacta, preservación y despacho; además guarda que el núcleo neutral no contenga dependencias físicas. | Tres rondas normales, contratos focales y `-race` verdes. No acredita por sí sola `ORC-28` ni V38. | Incluido en `V=348` |

A08 queda `exercised` dentro de `P=0,V=600`. No emite evidencia canónica
mientras `AC-V38-AGENT-RUNTIME-ELASTIC` siga `planned`: el receipt ligado a
los resúmenes criptográficos de ambos candidatos se emite en C04 y solo C05
promueve `ORC-28` después de A+B+C. Esta separación evita que una compuerta
neutral, ejecutada sin KVM, atribuya prematuramente la composición física.

### B01 — `P=0, V=50`

| Hijo | Write-set y resultado | Orden | Presupuesto |
|---|---|---|---:|
| B01.1 | Compuerta compacta: verifica selección A04, fuentes/controlador A05 y frontera cruzada B04 sin reimplementarlos. Cubre `process` sin conexión al hermano, microVM sin ruta alternativa, controlador anfitrión solo de cuota, procesos separados y ausencia de límite Codex global. | Tras A05/A08/B03/B04; abre B10. | `P=0, V=50` |

### B02 — `P=200, V=250`

| Hijo | Conjunto de escritura y resultado | Orden | Presupuesto |
|---|---|---|---:|
| B02.1 | `9ad41b6`: esquema SQLite privado v11, reserva transaccional de ejecución y concesión CID, rango `3..=65535`, idempotencia, revisión, cerca, expiración y exclusividad activa. | `exercised`; el registro físico pertenece solo al binario hermano. | Delta hermano incluido en el total B02 |
| B02.2 | `b8fc520`: el motor, manifiesto, identidad y recuperación reciben el CID reservado; desaparece el CID constante y una respuesta física distinta falla cerrada. | `exercised`; no arranca KVM. | Delta hermano incluido en el total B02 |
| B02.3 | `178d519`: una detención exacta libera, una cerca obsoleta no libera y la recuperación pone en cuarentena al propietario de una reserva expirada antes de reutilizar el CID. | `exercised`; cubre reapertura, agotamiento y ocho reservas concurrentes. | Delta hermano incluido en el total B02 |
| B02.4 | `127c0a45`: retira de Orquesta el puerto CID, allocator y renderer sin consumidores, actualiza roadmap e inventario y conserva la brecha física pendiente. | Después de B02.1–B02.3; elimina la segunda autoridad. | Compensación `P=-1906,V=-2223` |

El candidato documentado del repositorio hermano es `3c827d9`. Sus gates
aportan 178 pruebas Rust correctas y dos smokes KVM ignorados, diez repeticiones
de la asignación concurrente, Clippy, formato y el cliente Go normal, con
detector de carreras y `vet`. El delta aislado es `P=372,V=314`; junto con la
retirada causal de Orquesta queda `P=-1534,V=-1909`, dentro del límite B02.

B02 queda `exercised`, no `accredited`: demuestra una única autoridad física,
pero todavía no el recorrido sellado de la release. La siguiente dependencia
serial elegida es B05; B06/B08/B09 siguen abiertas y B07 espera B05+B06.

### B03 — `P=250, V=300`

| Hijo | Conjunto de escritura y resultado | Orden | Presupuesto |
|---|---|---|---:|
| B03.1 | `27ca627`: caracteriza el repositorio hermano publicado, su crate/binario Rust, cliente Go y fronteras. La guarda ejecutable prohíbe dependencias de producto consumidoras, inversión hacia adaptadores, transporte TCP y rutas sin versión. El baseline `bdba503` se mide aparte en el ADR. | `exercised`; no atribuye el baseline ni el contrato cruzado a B03. | Consumido `P=0,V=71` de `P=60,V=80` |
| B03.2 | `8cdd2a9`: concentra nombres estables en `OperacionV1`, añade DTO neutrales para eventos/recuperación y exige que capacidades solo anuncie operaciones conocidas y realizadas. Las dos operaciones futuras siguen ausentes hasta sus tareas causales. | `exercised`; congela la semántica pública sin simular rutas. | Consumido `P=87,V=79` de `P=100,V=110` |
| B03.3 | `cfb6302` y `e84fd2a`: conserva `agentmicrovm.local.v1` sobre socket Unix y sus pruebas cliente/servidor; `servir` acepta solo `--config` absoluto y el cargador TOML estricto verifica identidad, propietario, modo, tamaño, schema y valores sin entorno. | `exercised`; un único proceso local técnico y ninguna escucha TCP. | Consumido neto `P=-53,V=45` de `P=90,V=110` |

La suma `60+100+90=250` y `80+110+110=300` limita el delta posterior al
baseline publicado `bdba503`. Sus 29.221 líneas físicas Rust/Go se inventarían
por separado y solo cuentan como conducta de una tarea cuando superan su gate;
no se atribuyen en bloque a B03 ni desaparecen de la medida. No existe reserva
posterior para «extraerlo», volver a publicarlo o convertirlo en servicio
remoto.

B03 queda `exercised` en el candidato hermano `f3ae2b4`, con delta neto total
`P=34,V=195` frente a `bdba503`. Pasaron formato, Clippy sin avisos, 172 pruebas
Rust —más dos smokes KVM ignorados—, pruebas Go normales y con detector de
carreras, `vet` y los contratos focales de independencia, configuración y
transporte. No acredita B04, `ORC-28` ni V38. B02 cerró después su siguiente
dependencia serial; B05/B06/B08/B09 quedan causalmente abiertas y no se
mezclan en un mismo conjunto de escritura.

### B05 — `P=490, V=320`

| Hijo | Write-set y resultado | Orden | Presupuesto |
|---|---|---|---:|
| B05.1 | `2295ee2`: `agentmicrovm.huesped.v1` fija inicio de sesión neutral, turnos idempotentes y eventos ordenados por cursor, sin proveedor, ruta host o variante abierta. | `exercised`; contrato estricto compartido solo entre host y huésped. | Incluido en el total B05 |
| B05.2 | `ce8ae2b` y `50d329a`: PID 1 aloja una única sesión por microVM, reconecta por nuevas conexiones vsock, cerca turnos, acota salida, mata el grupo y reutiliza el mismo supervisor para la orden corta. | `exercised`; no crea parada ni lifecycle paralelos. | Incluido en el total B05 |
| B05.3 | El perfil sellado conserva Codex 0.146.0 como ejecutor sustituible; el supervisor solo conoce un programa interno y puede alojar Claude, Gemini u otro perfil. B08/B10 traducen después la sesión pública. | `exercised` sin KVM; no importa el códec interno de Orquesta. | Baseline caracterizado, sin reatribuirlo |
| B05.4 | `0970fee`: el manifiesto embebido incorpora SPDX 2.3 con huésped, Linux, Codex y BusyBox. Dos initramfs y dos perfiles ext4 independientes resultaron idénticos, privados y modo `0400`. | `exercised`; activos temporales retirados exactamente. | Datos SPDX informados aparte como `A=70` |
| B05.5 | `ce8ae2b` y `0970fee`: incompatibilidad estricta, sesión distinta, turno conflictivo, cursor inválido, truncamiento, timeout, descendientes, rutas host ausentes y veinte/cien repeticiones focales. | Último; mismo candidato `1aec635`. | Total `P=480,V=319` |

El huésped musl queda sellado por
`dd0c0e8b94382603487b691d7ae775947a50b19439b5a1242c1d6392302d0ace`;
el initramfs reproducible por `7e858d7215d5efbfb0479da379fe4420c34c6aeb875f6295bae0f5eb23992c30`
y el perfil por `4c1daa5ad93b945b50416178436d9b6d636857de82422ff981936e04ea426ce8`.
Pasaron 181 pruebas Rust y quedaron ignorados los dos smokes KVM explícitos.

B05 queda `exercised`, no `accredited`: sus eventos viven aún en la frontera
huésped y B08/B10 deben exponerlos sin duplicar el supervisor; B07/B12 prueban
el mismo candidato físicamente. B06 cerró después la transferencia sellada y,
junto con B05, abrió B07.

### B06 — `P=400, V=350`

| Hijo | Write-set y resultado | Orden | Presupuesto |
|---|---|---|---:|
| B06.1 | El baseline se caracteriza mediante su recorrido API Unix → ledger SQLite → CAS privado → motor/vsock → huésped y vuelta; no recibe rutas, CAS, repositorios o tipos del consumidor. | `exercised` sobre el candidato exacto, sin reatribuir el baseline. | Sin delta |
| B06.2 | `54b80fb`: el cliente Go acredita base64, SHA-256, rutas, duplicados, límites y raíz canónica antes del socket; revalida todos los campos y blobs de la salida. Rust y Go fijan el mismo vector criptográfico. | Tras B06.1; es el conector público que B10 consumirá. | `P=193,V=115` netas |
| B06.3 | `b1824ecd`: retira de Orquesta `agent_microvm_bundle` y `firecracker/launchplan`, sin consumidores productivos; desaparece la ruta que solo proyectaba metadata `planned_not_applied`. | Tras B06.2; no deja bridge ni transporte alternativo. | Compensación `P=-475,V=-453` |
| B06.4 | Gates de API/CAS/SQLite/huésped, veinte rondas Go con `-race`, cross-project/CAS/memfd/write-set de Orquesta y suites completas sin KVM. | Último; B12 conserva el E2E físico continuo. | Incluido en B06.2 |

El delta neto causal es `P=-282,V=-338`, dentro de `P<=400,V<=350`. La carga
inline v1 queda acotada a 2.048 ficheros regulares y 8 MiB; no descomprime un
archivo. Enlaces, dispositivos, traversal y rutas del anfitrión no son
representables y fallan cerrados. `SCM_RIGHTS` era una optimización opcional
para cargas grandes, por lo que no se añadió otra ruta antes de necesitarla.

B06 queda `exercised`, no `accredited`: el recorrido contractual es ejecutable
y B05+B06 abren B07. La microVM física continua y la persistencia final del
conjunto de cambios en Orquesta pertenecen a B12 sobre el adaptador B10.

### B07 — `P=500, V=450`

| Hijo | Write-set y resultado | Orden | Presupuesto |
|---|---|---|---:|
| B07.1 | Repositorio hermano, `firecracker/launcher.go`: plan físico, recursos sellados, jailer/`cgroup` y una VM por `RunRef`. | Primero; no importa proveedor ni Orquesta. | `P=150, V=120` |
| B07.2 | Repositorio hermano, `firecracker/{engine,recovery}.go`: identificador opaco durable, registro físico/CID privado e idempotencia de lanzamiento. | Tras B02/B07.1; serial sobre `engine.go`. | `P=130, V=110` |
| B07.3 | Repositorio hermano, `firecracker/observer.go` y reconciliación exacta PID/hora de inicio/`cgroup`/CID tras reinicio. | Tras B07.2; conjunto de escritura disjunto del lanzador. | `P=120, V=110` |
| B07.4 | Fallos por etapa, PID reutilizado, KVM/agotamiento de memoria, limpieza exacta y conjunto contractual sin recursos huérfanos. | Último; no modifica API salvo incidencia causal separada. | `P=100, V=110` |

### B08 — `P=550, V=450`

| Hijo | Write-set y resultado | Orden | Presupuesto |
|---|---|---|---:|
| B08.1 | Repositorio hermano: intermediario físico enlaza la VM exacta a los servicios del protocolo local sin conocer comandos o ciclo de vida Orquesta. | Primero; fija la frontera del intermediario. | `P=120, V=100` |
| B08.2 | Caracterizar y retirar `agent_microvm_launch_auth` y `firecracker/networkauth`; definir/verificar la concesión neutral firmada en `agentmicrovm`, con audiencia, resumen criptográfico del plan, `RunRef`, cerca, expiración y uso único. | Tras B08.1; serial sobre autorización. | `P=160, V=130` |
| B08.3 | Conector Orquesta: aplica permisos/atestación, firma la concesión y traduce sesión, artefactos, MCP y buzón causal a mensajes/referencias; no exporta `CredentialStore`, secretos ni tipos internos. | Tras B08.2; no comparte almacén ni ruta. | `P=170, V=130` |
| B08.4 | Repetición/concurrencia, firma/audiencia/resumen criptográfico/cerca/expiración, reinicio independiente, alcance cruzado, orden posterior y comprobante perdido. | Después de B08.2–B08.3. | `P=100, V=90` |

### B09 — `P=600, V=550`

| Hijo | Write-set y resultado | Orden | Presupuesto |
|---|---|---|---:|
| B09.1 | Repositorio hermano, `firecracker/egress.go`: concesión neutral, servicio vsock y comprobante estable. | Primero; congela API pública. | `P=110, V=90` |
| B09.2 | Repositorio hermano, `firecracker/egress_resolver.go`: DNS, redirecciones e IPv4/IPv6; bloqueo de bucle local, red privada, ULA, enlace local y metadatos. | Tras B09.1; paralelo con B09.3. | `P=170, V=160` |
| B09.3 | Repositorio hermano, `firecracker/egress_receipts.go`: límites de tiempo/contenido, idempotencia y comprobantes ante éxito, rechazo y respuesta perdida. | Tras B09.1; paralelo con B09.2. | `P=140, V=130` |
| B09.4 | Conector Orquesta: permisos/presupuesto a `EgressGrant` neutral por protocolo, sin tipos internos o secretos en `agentmicrovm`. | Tras B09.1; disjunto de B09.2/B09.3. | `P=100, V=90` |
| B09.5 | Revinculación DNS, redirección privada, VM vecina, entrada e Internet directo negativos. | Último, sobre B09.2–B09.4. | `P=80, V=80` |

### B10 — `P=280, V=430`

| Hijo | Write-set y resultado | Orden | Presupuesto |
|---|---|---|---:|
| B10.1 | `internal/adapters/agent/agentmicrovm/agent.go`: `AgentLauncher` al cliente v1, referencias opacas y lanzamiento Codex real sin tipo Codex en el protocolo. | Tras B04/B06; paralelo con B10.2/B10.3. | `P=150, V=130` |
| B10.2 | `observer.go` y `recovery.go`: observación/control por socket y fronteras `definitely_not_applied`/`unknown_applied`, sin leer el registro físico hermano. | Paralelo con B10.1/B10.3. | `P=130, V=120` |
| B10.3 | Guarda de dependencia: fija revisión y contrato del módulo Go público de Agente MicroVM; prohíbe otro cliente HTTP, tipos físicos duplicados o acceso a su registro. | Paralelo con B10.1/B10.2; no añade producto. | `P=0, V=20` |
| B10.4 | `internal/bootstrap/agent_provider.go` y `cmd/orquesta/`: compone el adaptador sobre el cliente Go externo y el candidato hermano. Retira A05.1b al completar la negociación inicial v1; prueba candidato alcanzado, composición parcial cerrada y cero rutas alternativas Codex/proceso. | Después de B10.1–B10.3; serial en la composición de arranque y sobre A05.1b. | `P=0, V=80`; la retirada compensatoria financia el cambio neto. |
| B10.5 | Reinicio en cinco fronteras, cerca, duplicado, comprobante perdido, observación concurrente y modo `-race`. | Último; no ensancha composición durante la prueba. | `P=0, V=80`; es verificación sobre B10.1/B10.2. |

### B12 — `P=400, V=500`

| Hijo | Write-set y resultado | Orden | Presupuesto |
|---|---|---|---:|
| B12.1 | Repositorio hermano: método v1 de preservación transmite inventario neutral, sello, resúmenes criptográficos y contenido del identificador opaco exacto. | Primero; no conoce espacio de trabajo ni proveedor. | `P=100, V=100` |
| B12.2 | `internal/adapters/agent/agentmicrovm/preserve.go`: recibe contenido/resúmenes criptográficos, añade espacio de trabajo/configuración/herramientas y persiste el comprobante/CAS de A06. | Tras B12.1 y B06; serial sobre adaptador. | `P=120, V=130` |
| B12.3 | Orden parada→sello→inventario→persistencia Orquesta→confirmación de desmontaje/liberación y recuperación a ambos lados. | Tras B12.2/B11; serial sobre cierre. | `P=100, V=130` |
| B12.4 | `acceptance/v38_agent_firecracker_single_vm_test.go`: Codex real, orden posterior, parada, preservación y negativos. | Último; emite solo compuerta B. | `P=80, V=140` |

## Frontera pública reutilizable

Agente MicroVM ya existe como proyecto hermano independiente en
`https://github.com/aavidad/agente_microvm`. Posee crate y binario Rust, módulo
cliente Go, configuración, pruebas, documentación, versiones y almacenamiento
físico propios. B03 caracteriza y completa la frontera; B04 demuestra su
consumo cruzado antes de componerla en producto.

El único punto de integración es el protocolo local
`agentmicrovm.local.v1` sobre un socket Unix con propietario, grupo, modo, longitud
de trama, versión y tiempos explícitos. No se expone TCP/HTTP remoto. Un mismo
binario ofrece un modo `serve` local, supervisado o activado por socket, durante
las ejecuciones necesarias. Puede residir mientras existan ejecuciones para
admitir órdenes, eventos y recuperación, pero no es un proceso residente de
dominio: no mantiene `Goal`, DAG, planificador, permisos ni política de
admisión.

El contrato neutral incluye:

- referencias, solicitudes, eventos y comprobantes opacos y versionados para
  `Launch/Observe/SendOrder/Events/Stop/Preserve/Recover/Close`;
- configuración física tipada del proyecto hermano y un registro físico/CID
  privado;
- tramas de control y contenido acotados por resumen criptográfico/tipo de
  contenido; cargas grandes mediante descriptor anónimo sellado por
  `SCM_RIGHTS`, nunca por ruta;
- una concesión neutral firmada, de un solo uso, ligada a audiencia, `RunRef`,
  cerca, expiración y resumen criptográfico completo del plan;
- códigos máquina estables, negociación de versión y cierre/reconexión
  deterministas.

No atraviesan la frontera base de datos/cadena de conexión, sistema de archivos,
esquema, migración, transacción, bloqueo, secreto, `CredentialStore`,
credencial de proveedor, ruta, registro global, CAS ni importación Go. La clave
pública de confianza se configura de forma
independiente en `agentmicrovm`; Orquesta conserva el firmante privado dentro
de su propia autoridad. Cada lado rota, respalda y recupera su estado sin abrir
el del otro.

El paquete neutral tampoco contiene `Goal`, `WorkItem`, `Execution`, permisos,
presupuestos, Codex o un tipo de atestador. Orquesta implementa un conector bajo
`internal/adapters/agent/agentmicrovm` que adapta los puertos existentes al
protocolo. `application` conserva la decisión y escritura causal; la
composición de arranque solo selecciona punto de conexión y candidato
compatible. El motor Firecracker, registro físico, CID, intermediario y salida
se componen exclusivamente dentro del binario hermano.

B04 arranca un candidato real del repositorio hermano y comprueba la
compatibilidad con los dos resúmenes criptográficos. Guardas de módulos,
dependencias y entorno de ejecución
demuestran que el protocolo es la única arista: no hay `replace` productivo,
importación transitiva, acceso a ruta/almacén o secreto compartido. La eventual
publicación a terceros será gobierno ordinario de versiones del proyecto
hermano; no es una extracción pendiente ni una capability encubierta.

## Retirada de solapamientos y corrección P2 separada

- B02 comparó el puerto CID con el registro privado del proyecto hermano,
  reimplementó allí las invariantes útiles y eliminó en `127c0a45` el contrato,
  el asignador y el renderer físicos de Orquesta. No movió ni copió el SQL a
  otro paquete Orquesta y no mantiene un puente.
- B08 hace lo mismo con
  `internal/ports/agent_microvm_launch_auth.go` y caracteriza explícitamente
  `internal/adapters/agent/firecracker/networkauth/`, incluido `verifier.go` y
  sus errores `agent_firecracker_network_auth`. Sustituye desafío/prueba HMAC
  por una concesión neutral firmada: Orquesta autoriza y firma dentro del
  conector; `agentmicrovm` verifica con confianza pública y consume una sola
  vez. Tras migrar consumidores se eliminan el puerto, la ruta nominal y todo
  secreto compartido, sin conservar un segundo adaptador.
- El hallazgo en `internal/application/budget_policy.go:115` se registra como
  corrección propia de política de presupuesto, con capacidad, invariante,
  caracterización y prueba de regresión antes de modificar código. Si
  condiciona la admisión de A04, se resuelve en un conjunto de escritura serial
  previo; nunca se mezcla con la migración A03, no altera su esquema y no se
  oculta dentro de su presupuesto.

## Contratos de alto riesgo que deben mantenerse exactos

### Capacidad y cuotas

Una observación no reserva por sí misma. Hay dos contratos distintos:

- La capacidad física describe slots o recursos realmente excluyentes de un
  pool. `application`, con su reloj, acepta su frescura y el claim la reserva de
  forma atómica. Se consume al acreditarse el lanzamiento, se libera solo al
  terminal o `definitely_not_applied`, y `unknown_applied` la conserva en
  cuarentena hasta reconciliar.
- La cuota por cuenta/perfil describe estado explícito
  `available/exhausted/unknown`, ventana/reset, revisión, expiración y
  reintento. El porcentaje oficial, si existe, se conserva solo como evidencia
  y nunca se convierte en saldo absoluto. Es una compuerta durable: varios
  claims pueden ligar la misma revisión válida, pero Orquesta no debita,
  consume, devuelve ni libera segundos, mensajes, tokens o créditos.

En la capacidad física, `slots=0` es agotado y ausente es desconocido, nunca
ilimitado. En la cuota Codex no se inventa un restante que el protocolo oficial
no ofrece. Estado `unknown`, observación stale o fuente no disponible impiden
launches nuevos, pero no bloquean `observe_agent`, `stop_agent`, órdenes
admitidas ni preservación.
`RuntimeCodexMaxConcurrentExecutions` no participa en `BudgetPolicy`,
admisión o despacho globales. `governance.global_process_slots_budget`, con
default 70, expresa el guardarraíl genérico de presupuesto; la clave Codex
coexiste sin alias semántico entre ambas y queda solo dentro de su
conector/pool.

`runtime.capacity.observation_timeout` limita cuánto espera la aplicación a
`ObserveCapacity`; su contexto derivado se cancela siempre y el contrato del
observer debe respetarlo. Al vencer, el mismo ciclo continúa inmediatamente
con `ExcludeLaunch` para que stop/observe sigan progresando. Esta duración no
define la frescura: `runtime.capacity.observation_ttl` solo limita durante
cuánto tiempo una observación ya emitida puede aceptarse. Ninguna se deriva de
la otra ni de `runtime.codex.timeout`.

`WindowRef` evita perpetuar una cuota agotada al cruzar un reset. Una
observación válida de una ventana nueva puede reabrir la compuerta; no lo hace
un timer local sin observación. El legacy que ignoraba ceros con comparaciones
`>0`, persistía `exhausted` tras el reset o extraía cuota de awk/logs/tmux queda
como lección negativa. El lector actual de un fichero de informe se elimina:
solo el protocolo estructurado real del proveedor puede alimentar la
compuerta, sin fichero, tmux, logs, scraping, alias o fallback.

La consulta legacy se limita a caracterizar
`codex_usage_accounting_v0.go`, `codex_usage_accounting_json_v0.go`,
`codex_usage_accounting_wrapper_v0.go`,
`agent_usage_runtime_source_v0.go` y `capacity_decision_v0.go` en la copia de
consulta. Se preservan semánticas útiles —fuente/calidad/observación/reset,
agotamiento y reintento—, no sus supuestos restantes absolutos, parsers,
procesos, tablas, planificador ni código. Un porcentaje queda como evidencia;
no se convierte en unidades consumibles.

Antes del claim, `application` enumera candidatos opacos, los ordena de forma
determinista y los deduplica. El claim revalida por CAS y fija/reserva el primer
candidato todavía disponible en la misma transacción. La
`AgentPlacementRef` sí atraviesa application y el request/receipt de agente;
nombre, ruta, cuenta y secreto del perfil no lo hacen. Pool y launcher no
reseleccionan. Un reinicio o replay conserva colocación e idempotency key.

La capacidad física publicada debe ser bruta/exclusiva. Held se resta una sola
vez por `source+pool` para reservas `reserved`, `consumed` o `quarantined`; no
por `observation_ref` y nunca sobre una medida que ya haya descontado agentes
activos. La fuente declara esa semántica y una ambigüedad falla cerrada.

### Códec de `app-server`, esquema neutral y aislamiento de procesos

El códec interno específico de Codex solo conoce enmarcado, identificadores,
inicialización, correlación, listas permitidas y límites de protocolo. No
conoce Firecracker, `Goal`, espacio de trabajo, política de cuota ni ciclo de
vida. Orquesta deriva de él un artefacto neutral de esquema/versiones; no
comparte el paquete con `agentmicrovm`.

Por cada perfil de cuota configurado, la composición de arranque inicia
exactamente un proceso
persistente `codex app-server` y un lector técnico de protocolo acotado a esa
conexión. Completa `initialize`→`initialized`, solicita y persiste la primera
lectura válida y solo entonces arranca el planificador. El lector procesa
respuestas/eventos enmarcados; no sondea, planifica, elige colocación ni escribe
ciclo de vida. La conexión se reutiliza aunque haya cero agentes o una ola
`5/10/20`.

Ante pérdida de conexión, el controlador marca la cuota desconocida, cierra y
recolecta el hijo exacto y reconecta con límites. La rotación hace lo mismo y
no publica la sustitución hasta obtener su lectura inicial. La parada cancela
el lector, cierra `stdio`, espera el proceso y, si vence el plazo, mata solo su
PID verificado y lo recolecta antes de devolver. Un fallo durante el arranque
recoge todos los perfiles ya iniciados y no arranca el planificador.

Controlador anfitrión y agentes ejecutores huéspedes usan instancias separadas,
con `HOME`,
credenciales, `stdio`/socket, hilo, turno y permisos acotados. El codec no
migra el worker de proceso actual. El controlador no implementa
`AgentLauncher`, `AgentObserver` o `AgentController`, no abre turnos ni accede
a espacio de trabajo, buzón causal o artefactos. Es un conector técnico del
monolito modular, no un proceso residente de dominio, microservicio,
planificador, almacén o escritor. Ningún paquete del repositorio hermano
importa el códec Codex interno.

### Despachador y paralelismo

Hay un solo despachador y un solo protocolo de claim. La preferencia serial por
no-launch sirve para que observe/stop/mailbox avancen; no es una cola nueva.
Solo se crea una goroutine corta después de reclamar realmente un launch con
reserva física durable y solo mientras ejecuta ese claim. El despachador no
mantiene `maxConcurrentLaunches`; el límite físico del claim regula los
lanzamientos concurrentes y jamás trunca `ReadyWorkItems`, reprograma el DAG o
limita el número de Goals visibles.

Ese es el estado final de A07. Durante la transición A05.1d conserva
temporalmente `maxConcurrentLaunches` y `ExcludeLaunch`, alimentados por
`governance.global_process_slots_budget`, para no dejar ilimitado el modo
`process` antes de A04.2. La prueba transitoria acredita que un presupuesto
neutral N nunca supera N y que variar
`runtime.codex.max_concurrent_executions` no altera el despacho global.

### Identidad, red y credenciales

La identidad de operación contiene referencias de proyecto, `Goal`,
`WorkItem`, `Execution`, generación, acción/intento, motor externo y cerca. PID
y CID son datos de encaminamiento sujetos a reutilización, no identidad
suficiente.

El huésped no recibe NIC. Los únicos servicios vsock permitidos son
`orquesta_broker` y `controlled_egress_proxy`. Orquesta aplica permisos y
atestación detrás de sus puertos y emite una concesión neutral firmada, de un
solo uso, ligada a audiencia v1, `RunRef`, cerca, caducidad y resumen
criptográfico completo del plan. `agentmicrovm` verifica con confianza pública
configurada y registra
el consumo en su estado privado. No cruza `CredentialStore`, clave privada,
secreto HMAC, credencial de proveedor ni tipo de atestador. No se entregan
secretos en `rootfs`, configuración efectiva, `Goal`, instrucción, registro,
artefacto o comprobante.

El proxy vuelve a resolver DNS y revalida cada redirección. Debe negar Internet
directo, entrada, tráfico lateral, bucle local, redes privadas/enlace
local/metadatos y
todo destino no permitido. Un acuse del intermediario solo confirma admisión:
el efecto
se acredita mediante el comprobante del registro de efectos.

### Espacio de trabajo, parada y preservación

El anfitrión produce un paquete/instantánea ligado al espacio de trabajo y base
exactos. El conector Orquesta resuelve el CAS y transmite referencias,
resúmenes criptográficos, tipo de contenido y contenido verificado por el
socket Unix; el motor los entrega al huésped por
vsock. Para cargas grandes puede usar un descriptor anónimo sellado por
`SCM_RIGHTS`, nunca una ruta. Ni `agentmicrovm` ni el huésped montan `.git` o
sistema de archivos del anfitrión. El resultado vuelve por el protocolo y pasa por la
validación e integración en anfitrión ya acreditadas.

La parada cooperativa y forzada actúa solo sobre la identidad exacta. Si se
pierde el comprobante, se reconcilia antes de repetir; nunca se ejecuta una
terminación amplia.
Antes de liberar CID, socket, `cgroup` o montaje, `agentmicrovm` devuelve sello
e inventario; `application` los valida, persiste `preserved_pending_review` y
confirma el desmontaje por protocolo.
V38 no contiene API de borrado automático. La eliminación futura será otro
efecto autorizado, con alcance e idempotencia propios.

## Riesgos y ratchets

- **Concurrencia en fronteras compartidas:** A04 y A06 no editan a la vez
  reclamo, procesamiento o SQLite. El despachador A07 ya está integrado; A04
  deberá extenderlo sin crear otra ruta ni deshacer sus garantías.
- **Migraciones independientes:** A03 usa la cadena Orquesta y B02 la cadena
  privada del repositorio hermano. Un número coincidente no implica colisión;
  compartir esquema, cadena de conexión, transacción o fichero sí bloquea la
  tarea.
- **Autoridad de persistencia:** Orquesta escribe ciclo de vida por
  `StateRepository`; `agentmicrovm` escribe solo estado físico por su registro
  privado. Cada proyecto activa un conector propio y ninguno abre, replica,
  migra o usa como ruta alternativa el del otro. Solo
  referencias/cercas/comprobantes cruzan el protocolo.
- **Dirección hexagonal y entre módulos:** guardas de dependencias y entorno de
  ejecución impiden importaciones cruzadas y que Firecracker conozca
  SQL/CAS/credenciales de Orquesta. La composición de arranque de Orquesta solo
  ensambla el cliente local; `application` decide y el binario hermano compone
  sus conectores físicos.
- **Confundir TestAttestor con agente:** las guardas de
  importaciones/composición deben impedir compartir protocolo, `rootfs`,
  comprobantes o evidencia. Solo
  primitivas neutrales pequeñas, extraídas con dos consumidores y pruebas, son
  reutilizables.
- **Cuota falsa o stale:** fail-closed para launches nuevos y continuidad de
  control para ejecuciones existentes. Ningún «desconocido = ilimitado»,
  lector de fichero ni reserva contable de cuota.
- **Colocación no repetible:** application ordena/deduplica antes del claim; la
  transacción fija el primer candidato válido, lo persiste como ref opaca y el
  replay no cambia de perfil o pool. Bootstrap no toma esa decisión.
- **Procesos Codex mezclados:** esquema compatible no significa paquete,
  instancia ni estado común. Guardas prueban que el proyecto hermano no importa
  el códec interno, que el controlador anfitrión solo consulta cuota y que el
  trabajo microVM siempre se ejecuta dentro del huésped.
- **Ciclo de cuota incompleto:** todos los perfiles completan lectura inicial
  antes del planificador. Las olas 5/10/20 reutilizan conexiones; reconexión,
  rotación, fallo parcial de bootstrap y shutdown recolectan proceso y lector
  exactos. Cero agentes también prueba el ciclo completo.
- **Recursos huérfanos:** `agentmicrovm` inventaría PID, socket, CID, `cgroup` y
  temporales físicos propios; Orquesta inventaría sus referencias y artefactos.
  Cada proyecto limpia solo objetivos que posee y verifica; ante ambigüedad
  preserva e informa.
- **Anfitrión insuficiente:** no se reduce C de 20 a un número disponible ni se
  usa el resultado histórico `1+16`. Se documenta bloqueo y se repite en un
  anfitrión apto con la misma pareja de candidatos y resúmenes criptográficos.
- **Crecimiento arquitectónico:** Orquesta conserva un despachador, una fuente
  de ciclo de vida, un registro de efectos, uno de comandos y uno de
  configuración. `agentmicrovm` conserva un binario local, un registro físico
  privado y bucles técnicos acotados a protocolo/proceso. La separación fuera
  de proceso se justifica por KVM, Firecracker y aislamiento, no crea ciclo de
  vida, planificador,
  cola, permisos o política paralelos. No se aceptan servicios por proveedor,
  fase u operación, paquetes de una constante ni TCP/HTTP remoto.
- **Presupuesto:** la suma exacta de las 25 tareas V38 es
  `P=7.200` líneas no generadas y `V=10.083` líneas de verificación y arneses. El
  subtotal afectado A01+A02+A03+A04+A05+A08+B01+B05 queda
  en `P=2.505,V=3.158`: `0+170+334+649+862+0+0+490=2.505` y
  `138+238+537+647+628+600+50+320=3.158`. Sumado a B10 `280/430`, conserva
  exactamente el subtotal previo conjunto `P=2.785,V=3.588`; A02 incorpora Q1 una sola vez;
  A03 incorpora la base `139/270`; A04 incorpora base y vínculo `160/110`.
  A07 queda en `P=165,V=695`; sus pruebas integradas explican el aumento neto
  de `V=283`, y conserva `P=66` para retirar los límites transitorios.
  B03 limita a `P=250,V=300` el delta posterior al baseline hermano medido en
  `adr_v38_baseline_agente_microvm_2026-08-02.md`; B10 consume su cliente Go
  público en vez de duplicarlo y transfiere `P=180,V=120` a A05.2b. B02
  consume aisladamente `P=372,V=314`, pero
  la retirada causal de la autoridad duplicada deja su delta conjunto en
  `P=-1534,V=-1909`, dentro de `P=200,V=250`. Las demás tareas B contabilizan en sus
  techos el lado de Orquesta y el lado `agentmicrovm`. El código generado se informa aparte y
  ninguna holgura se descuenta dos veces. No existe contingencia, bolsa de
  extracción ni presupuesto posterior a V38 oculto. Un exceso exige ADR,
  compensación
  y autorización previos. Objetivo adicional: cero nuevos escritores de
  ciclo de vida, cero almacenes fuera de `StateRepository` y el registro físico
  privado, cero bucles de dominio/sondeo/planificación y cero comandos públicos
  Orquesta salvo el contrato de aceptación.

## Cierre operativo de cada encargo

Cada tarea entrega en forma compacta:

```text
hecho:
capability e invariante restaurado:
autoridad final:
write-set real y cambios ajenos preservados:
tests normales/negativos/race/E2E:
receipts y revisión acreditada:
función útil conservada:
complejidad retirada o evitada:
legacy retirado o bloqueo de retirada:
LOC netas/generadas y complejidad:
procesos, temporales, sockets, CIDs, cgroups y disco propios:
riesgos/P0/P1:
siguiente dependencia causal:
```

No se abre la siguiente tarea de integración si la anterior deja un P0/P1 en
su invariante, una contradicción de roadmap, un write-set ajeno sobrescrito o
un recurso físico propio sin clasificar. C05 es el único cierre de V38; todos
los cierres anteriores son compuertas parciales.
