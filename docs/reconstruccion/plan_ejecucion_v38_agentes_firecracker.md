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
  ejecuta ni requiere KVM, Firecracker, rootfs o sockets vsock.
- **Compuerta B, adaptador Firecracker de activación explícita:** crea
  exactamente una microVM física por ejecución de agente, sin ruta alternativa;
  el huésped recibe el workspace por vsock, usa broker/proxy controlados, acepta
  órdenes posteriores al arranque, se observa y detiene por identidad exacta,
  y entrega sello e inventario antes de desmontar. Bubblewrap continúa siendo
  el aislamiento del `TestAttestor`.
- **Compuerta C, olas físicas:** el mismo árbol, binario, rootfs, configuración y
  datos de prueba sellados superan las olas `1/5/10/16/20`, sin reconstrucción entre
  peldaños y sin convertir esos límites físicos en un límite lógico del DAG.

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
  `RuntimeJournal` será el contrato neutral cohesivo del estado físico y leases
  CID y se compondrá sobre esa misma fuente activa. Planificador, eventos,
  outbox, bootstrap y proyecciones no obtienen autoridad propia.
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
  El adaptador Firecracker implementará esos puertos; no añadirá ciclo de vida
  de proveedor.
- Workspace/Git, CAS, integración en anfitrión, execution sessions, MCP y
  mailbox ya están acreditados. El huésped recibe refs y bytes verificados,
  nunca una ruta del anfitrión, y el broker transporta los comandos existentes
  sin crear otro mailbox.
- Ya existen contratos `agent_microvm_bundle`, `agent_microvm_network`,
  `agent_microvm_launch_auth`, `agent_microvm_vsock_cid` y adaptadores de plan,
  autenticación y allocator SQL. B02 y B08 deben mapearlos contra los nuevos
  puertos neutrales, migrar a sus consumidores y retirar las superficies
  solapadas en la misma vertical; no se conservan dos contratos, bridges ni
  adaptadores duplicados «por compatibilidad».
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

No existe todavía un motor persistente de agentes Firecracker, una fuente
neutral de capacidad física viva, controlador de cuota estructurada,
colocación opaca fijada dentro del claim, composición canónica de leases CID, rootfs de
agente, protocolo del huésped, broker/proxy, retorno sellado ni evidencia
física `1/5/10/16/20`. Tampoco existe aún la frontera pública importable del
motor físico. Esas son las brechas que siguen.

## Orden causal y write-sets

`A01 + A02`; `A02 → [A03.1 | A05.1 → (A05.1b + A05.1c + A05.1d)]`;
`A02 → A02b → [A04.1 → A03.2 → A03.3 | A05.2 | A05.3a → A05.3b]`;
`[A05.1b + A05.1c + A05.1d + A05.2 + A05.3b] → A05.4`;
`[A03.3 + A05.4] → A04.2 → A04.3 → A04.4 → A06 → A07 → A08 →`
`B03 → [B02 | (B05 + B06 → B07) | B08 | B09]`;
`B07 + B09 → B04 → B01`; `B01 + B02 + B04 + B07 + B08 + B09 → B10 → B11 → B12 →`
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

Después de C05, y fuera de la acreditación V38: `C05 → D01`, antes del primer
consumidor externo productivo.

El símbolo `NNN` significa «siguiente migración libre en el momento de
integrar». El número se asigna bajo exclusión mutua después de releer la cadena;
no se reserva anticipadamente `022` mientras haya trabajo SQLite concurrente.
Los write-sets separados por `+` pueden ejecutarse en paralelo. Toda tarea que
toque `internal/application/state.go`, `config/registry.json`, composición o
la cadena de migraciones se serializa aunque aparezca en otra ola.

En la columna LOC, `P` es código no generado de producto/scripts y `V` son
pruebas, datos y arneses. Es un techo, no un objetivo.

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
| **A02 — contrato neutral de capacidad** | `ORC-28`. Capacidad y cuota son observaciones externas durables, no presupuesto ni texto de agente. Autoridad de frescura/política: `application`, usando su reloj. | `internal/application/agent_capacity.go`, `internal/application/agent_placement.go` y pruebas. Sin dependencia; paralela con A01. | La fuente física expresa slots brutos presentes/ausentes por `SourceRef`+`PoolRef`. La compuerta de cuota expresa estado `available/exhausted/unknown`, `WindowRef`, reset, expiración y `RetryAt`; un porcentaje es solo evidencia, no saldo. Ambas conservan revisión, calidad y colocación opaca. Evita almacén, servicio o planificador de cuotas. | Pruebas: slot cero no se pierde; ausente no equivale a ilimitado; cuota no inventa restantes; unknown/stale/unavailable cierra launches; nueva ventana observada puede reabrir; reloj del proveedor no decide frescura; payload/log no se hace autoridad. `P≤159,V≤226`. |
| **A03 — persistencia canónica de capacidad y cuota** | `ORC-28`. Capacidad física y cuota son hechos durables con papeles distintos: solo la primera admite reserva/consumo/liberación; la cuota por perfil es una compuerta versionada sin saldo interno. `application` decide y escribe mediante `StateRepository`. | Conserva intacta `022_agent_capacity.sql`. Una migración progresiva posterior a A04.1 añade la observación completa de cuota y el binding 1:1; después el mismo `StateRepository` implementa lectura y append CAS. Orden `A02b → A04.1 → A03.2 → A03.3 → A04.2`. | No añade clase ni lifecycle de observación: `reservation.observation_ref` identifica el hecho físico reservable y `quota_observation_ref` conserva el binding inmutable. No existe store ni operación separada para escribir el binding: solo `ClaimNextAction` lo inserta con la reserva. SQLite es local/desarrollo/pruebas; PostgreSQL es el único adaptador productivo futuro V31; nunca están activos a la vez. | La base consumió 139/291; Q2 dispone de 100/150 y Q3 de 95/120. Prueba migración progresiva, CAS/idempotencia, rollback, FK diferida, aislamiento, reapertura, backup/restore, ausencia de reserva de cuota y fuente única. Techo `P≤334,V≤561`. |
| **A04 — colocación y reserva física durable en el claim** | `ORC-28`. `application` entrega candidatos opacos ordenados/deduplicados; el claim revalida y fija el primero disponible. Solo ese claim crea una `AgentCapacityReservation` física y liga cuota sin debitarla. Stop/observe siguen reclamables. | `internal/application/{agent_capacity,agent_capacity_state,state}.go`, `internal/adapters/state/sqlite/{claim,agent_capacity}.go` y pruebas. Orden único: `A02b → A04.1 → A03.2 → A03.3 → A04.2`; A04.2 depende además de A05.4 y es serial con A06/A07. | Multiperfil Codex: cada perfil es un placement/pool de capacidad 1. Perfil único: cada launch reserva 1 de N configurado. Dentro del `BEGIN`, CAS fija el primero válido y crea una reserva+binding 1:1. Held se resta una vez por `source+pool`. Firecracker host queda bajo presupuesto/preflight y leases CID/VM, sin segunda reserva. | Base física 114/53 y binding neutral 46/57 ya consumidos; quedan `P≤299,V≤337`. Prueba agotamiento, stale, carreras, replay, doble contabilidad y ausencia de segunda reserva Firecracker; migra los literales afectados. Techo `P≤459,V≤447`. |
| **A05 — fuentes, colocación, configuración e inyección V38** | `ORC-28`. Proveedor y aislamiento son ejes ortogonales. Una fuente física informa recursos reservables; un controlador persistente por perfil informa cuota. `application` decide frescura, orden/deduplicación y admisión; el claim fija colocación. | Registro/salidas, codec, controlador Codex, `internal/bootstrap/`, `cmd/orquesta/`, i18n y application. Depende A02b; A05.3a precede A05.3b y A05.4 precede A04.2. | Por perfil de cuota: un `app-server` persistente y un lector de protocolo acotado. La sesión fija capacidades nulas, solo admite métodos de cuota y descarta errores remotos sensibles. Bootstrap exige lectura inicial antes del planificador; la misma conexión sirve actualizaciones y olas 5/10/20. | Cero agentes, 5/10/20 sin proceso por launch, fallo inicial que aborta bootstrap y shutdown sin huérfanos. Fallo cerrado ante stale/timeout. Guardas impiden imports Firecracker y puertos de agente. `P≤608,V≤508`. |
| **A06 — contrato neutral de preservación** | `ORC-28`. Antes de desmontar, el entorno entra en `preserved_pending_review` con sello e inventario; no es un nuevo estado del `Goal`. Autoridad de aceptación: `application`. | `internal/ports/agent_environment.go`, `internal/application/agent_environment.go`, migración `NNN_agent_environment_receipts.sql`, implementación SQLite específica y pruebas. Depende A03; migración y `state.go` seriales con A04. | El comprobante liga execution/external identity/fence, base del workspace, change-set/bundle CAS, inventario, configuración/rootfs y tiempos. Reusa CAS, refs, efectos y revisión. No expone operación delete/GC en V38. | Normal, comprobante duplicado, digest alterado, ref de otro proyecto, sello ausente, desmontaje prematuro, fence obsoleto y reinicio. Cierra cuando terminalizar un proveedor que exige preservación falla cerrado sin comprobante válido y el material sigue direccionable. `P≤350, V≤350`. |
| **A07 — despachador único con `launch_agent` concurrente** | `ORC-28`. Un despachador global; procesamiento serial salvo `launch_agent` ya reclamado. Observación y parada progresan aunque el lanzamiento esté saturado. Autoridad de reclamo, admisión y procesamiento: `application`; bootstrap solo compone, arranca y drena. | Base integrada en `7ae98026`; despachador y pruebas integrados en `e4a6a98e`. Depende de A04 para cerrar capacidad durable; los archivos compartidos se serializan. | `ClaimNextAction` solo devuelve un lanzamiento con colocación y reserva física durables. El despachador no mantiene `maxConcurrentLaunches` ni lee `RuntimeCodexMaxConcurrentExecutions`; ante saturación reclama con `ExcludeLaunch`. Cada lanzamiento admitido usa una gorutina corta ligada a su claim. Evita pool global, cola por proveedor y segundo bucle. | Ratchets `1/5/10/16/20`, saturación, observación/parada, claim cercado, cierre, carrera y ausencia de giro activo. Negativos cambian proveedor/aislamiento sin leer `runtime.codex.*` y rechazan cualquier launch sin reserva. A07 cierra con A04 y el comprobante A08. `P≤250, V≤400`. |
| **A08 — compuerta A neutral** | `ORC-28`. Acredita semántica sin KVM: cohortes, capacidad parcial, órdenes/mailbox, observación, parada, preservación y recuperación. Autoridad: contrato de aceptación V38, todavía sin promoción. | `acceptance/v38_agent_runtime_elastic_core_test.go`, datos V38 y sustituto contractual bajo la superficie de prueba existente. Depende A01–A07. | Reusa sustituto Agent, execution sessions/mailbox, CAS, reloj y almacén reales. No introduce sustituto productivo ni interpreta ACK/texto como efecto. | E2E neutral `1/16/70/500`; capacidad menor que conjunto listo; refresco; orden/ACK exactos posteriores al lanzamiento; parada; reinicio en fronteras de intento/efecto/comprobante; `-race`; suite sin `/dev/kvm`, binario Firecracker ni artefacto de huésped. Cierra solo con comprobante de compuerta A ligado a digests. `P≤0, V≤750`. |
| **B01 — selección explícita y composición Orquesta** | `ORC-28`. Gate que verifica que proveedor, aislamiento, fuente física y cuota se componen por ejes distintos. La implementación pertenece a A04/A05/B10.4; B01 no vuelve a programarla. | Solo prueba contractual compacta después de A05/A08/B03/B04 y antes de B10. | Reutiliza la selección de A04, fuentes/controlador de A05 y composición de B10.4. Evita otra capa de bootstrap y matrices repetidas. | Negativos prueban que `process` no inicia Firecracker, que microVM incompleta no cae a proceso, que el controlador anfitrión solo observa cuota y que no reaparece un límite Codex global. `P≤0,V≤50`. |
| **B02 — implementación de leases CID** | `ORC-28`. CID sirve para encaminamiento, nunca identidad; lease durable tiene revisión, fence, expiración y recuperación antes del enlace físico. `RuntimeJournal`/`CIDLease` son contratos; SQLite es el conector local, de desarrollo y de pruebas actual, no autoridad. | Migración `NNN_agent_vsock_cid.sql`, `internal/adapters/state/sqlite/vsocklease/` y pruebas. Depende A03/A08/B03; migración serial; paralela con B05/B06/B08/B09. No se aloja SQL bajo el adaptador Firecracker. | Migra el allocator SQL existente al conector de estado y lo hace implementar el puerto cohesivo público. Mapea y retira `internal/ports/agent_microvm_vsock_cid.go` al migrar todos sus consumidores; no deja bridge. `SchemaStatements` se retira o deriva de la única migración de prueba. B10 compone el puerto; PostgreSQL será el conector productivo futuro de V31/`OPS-11`. | Asignación concurrente, wrap/agotamiento, liberación obsoleta, caída/reapertura, lease expirado, CID ocupado, recuperación exacta y carreras. Guardas `go list`/`rg` prohíben `database/sql`, SQLite y el adaptador SQL bajo `agentmicrovm/v1/firecracker/**`. Cierra con una definición de schema, un contrato y una sola fuente activa. `P≤200, V≤250`. |
| **B03 — API pública versionada del motor** | `ORC-28`. El gestor físico es reutilizable por futuras aplicaciones e importable sin Orquesta. Su autoridad termina en recursos microVM y comprobantes físicos; nunca conoce `Goal`, `WorkItem`, `Execution`, ciclo de vida, permisos o presupuestos. | Nuevo árbol público `agentmicrovm/v1/{contract,config,ports,errors}.go`, suite contractual `agentmicrovm/v1/contracttest/` y pruebas. Depende A08; primera tarea B. | Expone `Engine`, `Config`, refs/handles opacos y puertos cohesivos neutrales: `ArtifactSource`, `ArtifactSink`, `RuntimeJournal`/`CIDLease`, reloj, autorización/confianza y servicios vsock. Ni API ni motor reciben `CredentialStore`, atestador, CAS, SQLite, tipos de producto, permisos o presupuestos. Cada implementación concreta se inyecta por puerto; sin interfaz por función, globals, i18n ni imports `orquesta/internal`. | Guardas `go list` y `rg` sobre API, motor y adaptadores: sin dependencias internas, `database/sql`, SQLite, CAS, credenciales/atestación de Orquesta ni imports adaptador→adaptador. Cierra cuando el consumidor externo usa solo contratos neutrales y la API no exporta conceptos de producto. `P≤250, V≤300`. |
| **B04 — consumidor externo y prueba de extracción** | `ORC-28`. La importabilidad se demuestra, no se infiere por ubicación. La extracción física a otro módulo/repositorio se difiere hasta estabilizar v1. | `acceptance/agent_microvm_v1_external_consumer_test.go`, consumidor mínimo en `acceptance/fixtures/agent_microvm_external_consumer/` y guarda de API; no se registra otro `go.mod`. Depende B07/B09. | La prueba crea un módulo temporal externo, usa `replace` hacia este árbol, importa `orquesta/agentmicrovm/v1` y `.../firecracker`, instancia el motor con sustitutos contractuales y compila/ejecuta sin `internal`. Evita módulo, repositorio, microservicio o publicación prematuros. | Negativo que detecta import interno/transitivo, variables globales Orquesta y ruptura incompatible de v1; prueba mínima externa y presupuesto de dependencias. Cierra con consumidor verde y un informe de extracción, sin haber extraído ni publicado nada. `P≤50, V≤350`. |
| **B05 — rootfs y protocolo del huésped reproducibles** | `ORC-28`. Protocolo y supervisor del huésped son neutrales. El enlace Codex del huésped reutiliza el codec de A05, pero ejecuta una instancia distinta dentro de cada microVM; nunca comparte proceso, socket, HOME, hilo ni turno con el anfitrión. | `agentmicrovm/v1/{guestproto,guestsupervisor}/`, enlace fino y composición bajo `tools/agentmicrovm-rootfs/`, manifest/SBOM/licencias y pruebas. Depende B03 y A05.3a; paralela con B02/B06/B08/B09. | `guestproto` versiona bootstrap/control/eventos; `guestsupervisor` aporta reconexión, deduplicación y contrapresión. El enlace huésped importa solo el subpaquete codec Codex; ningún adaptador Firecracker lo importa. Arranca `app-server` dentro del huésped y no usa el controlador anfitrión para trabajar. | Build reproducible, digest, aislamiento de procesos, reinicialización, incompatibilidad, trama parcial/replay/desorden y ausencia de WebSocket/tmux/secretos. Cierra con kernel/rootfs/init sellados y worker Codex real independiente. `P≤490,V≤320`. |
| **B06 — artefactos y bundle por fuente/sumidero** | `ORC-28`. El motor público mueve contenido por ref/digest mediante `ArtifactSource`/`ArtifactSink` y no interpreta repositorios; Orquesta conserva autoridad sobre workspace, base, snapshot, write-set e integración. | `agentmicrovm/v1/artifacts.go`, `internal/adapters/agent/firecracker/bundleio/`, extensión mínima del productor Workspace/Git y pruebas. Depende A08/B03; paralela con B02/B05/B08/B09. | Los puertos públicos entregan y reciben bytes verificados con media type, sin conocer CAS. El adaptador interno construye `AgentMicroVMBundleDescriptor`, valida traversal/links/devices/expansión y traduce hacia el almacén configurado. El huésped nunca conoce rutas del anfitrión. Bootstrap compone fuente y sumidero; ni motor ni adaptador importan el conector concreto. | Bundle válido/corrupto/truncado, base errónea, traversal, links/device, expansión abusiva, ref entre proyectos, repetición idempotente y change-set fuera de write-set. Guardas de imports concretos. Cierra con anfitrión→artefacto→huésped→artefacto→integración, sin CAS ni tipos Orquesta en la API. `P≤400, V≤350`. |
| **B07 — motor físico Firecracker público** | `ORC-28`. `firecracker.New(Config, Dependencies)` crea una microVM por `RunRef`, observa y reconcilia recursos físicos; no conoce el proveedor alojado ni cierra o reabre trabajo de la aplicación llamante. | `agentmicrovm/v1/firecracker/{engine,launcher,observer,recovery}.go`, primitivas neutrales justificadas y suite `contracttest`. Depende solo de B03/B05/B06; no depende de B01 ni importa una implementación B02. | Implementa únicamente lanzamiento, vsock, observación y teardown físicos. Consume `RuntimeJournal`, `CIDLease`, `ArtifactSource/Sink` y autorización como puertos; persiste handle físico opaco sin conocer su conector. No importa `internal`, proveedor, protocolo de aplicación, `database/sql`, SQLite, CAS, credenciales ni atestación. Sin NIC/TAP/bridge/NAT ni VM compartida. | Dos lanzamientos→dos VMs, fallo por paso, PID reutilizado, fugas, path/asset inválido, KVM ausente, OOM, observación ambigua, reinicio y carreras; manifiesto físico respetado. Guardas `go list`/`rg` de dependencias y símbolos prohibidos. Cierra con identidad exacta, neutralidad y cero recursos propios huérfanos. `P≤500, V≤450`. |
| **B08 — broker Orquesta sobre servicio público** | `ORC-28`. El broker interno traduce el servicio vsock neutral a sesión, artefactos, MCP y mailbox exactos; `application` autoriza y conserva la causalidad. Ni broker ni motor obtienen autoridad de ciclo de vida. | Caracteriza `internal/adapters/agent/firecracker/networkauth/`, incluido `verifier.go`, y trabaja en `agentmicrovm/v1/launchauth/`, `internal/adapters/agent/launchauth/`, `internal/adapters/agent/firecracker/broker/` y pruebas. Depende A08/B03; paralela con B02/B05/B06/B09. | Mapea y retira `internal/ports/agent_microvm_launch_auth.go`. El código `agent_firecracker_network_auth` deja de ser un segundo adaptador nominal: desafío/prueba/transacción neutrales pasan a `agentmicrovm/v1/launchauth/`; la traducción Orquesta se mueve a `internal/adapters/agent/launchauth/`, sin dependencia Firecracker, y la ruta anterior se elimina tras migrar consumidores. El broker recibe `LaunchAuthorizer`, `ArtifactSource/Sink` y servicios compuestos; solo bootstrap ensambla. | Replay/concurrencia, atestación rechazada detrás del puerto, alcance cruzado, reconexión, commit/rollback, reinicio que invalida challenge, orden posterior y artefacto por ref. Guardas contra imports adaptador→adaptador, permanencia de `agent_firecracker_network_auth` y símbolos de producto en API. Cierra con un contrato, una implementación nominal y solo la VM exacta accediendo al broker. `P≤550, V≤450`. |
| **B09 — egress controlado reutilizable** | `ORC-28`. El componente público aplica una concesión inmutable de red y emite comprobantes físicos; Orquesta decide autorización, presupuesto y ledger antes/después. Única salida: `controlled_egress_proxy` por vsock. | `agentmicrovm/v1/firecracker/egress.go`, `internal/adapters/agent/firecracker/egresspolicy.go` y pruebas. Depende B03; paralela con B02/B05/B06/B08. | API neutral `EgressGrant` con destinos, puertos, tiempo/bytes y ref opaca; el adaptador interno la deriva de permisos/budgets. Revalida DNS/redirects y bloquea loopback, RFC1918, ULA, link-local/metadata. Evita Internet directo, TAP/bridge/NAT y política Orquesta en el paquete público. | Permitir/negar, DNS rebinding, redirect privado, IPv4/IPv6, exceso, timeout, comprobante perdido, presupuesto agotado, VM vecina e inbound negativo. Cierra sin NIC del huésped y con comprobantes idempotentes/machine-readable. `P≤600, V≤550`. |
| **B10 — adaptador Agent, cableado único y continuidad** | `ORC-28`. `application` decide ciclo de vida, capacidad, permisos y efectos; el adaptador traduce `AgentLauncher/Observer` a `Engine` v1 sin ganar autoridad. Une proveedor y aislamiento ya elegidos, sin clase por combinación. | `internal/adapters/agent/firecracker/{agent,observer,recovery,dependencies}.go`, `internal/bootstrap/agent_provider.go`, `cmd/orquesta/` y pruebas. Depende B01/B02/B04/B06–B09; única tarea de composición, serial sobre la compuerta A05.1b. | El adaptador recibe puertos neutrales y no importa conectores concretos. Bootstrap selecciona una fuente y ensambla conectores; no decide ni escribe. En el mismo cambio elimina el rechazo incondicional `bootstrap.runtime_isolation_not_composed` únicamente cuando la composición Firecracker real de `microvm` está completa; no conserva ruta paralela ni fallback a proceso. | Además de idempotencia, fence, CID, reinicio y `-race`: prueba explícita de retirada del rechazo, donde `process` sigue igual, `microvm` completo llega al motor y la composición parcial conserva el mismo código antes de construir Codex/proceso. Un espía de constructor acredita que no queda fallback al anfitrión. Cierra con Codex real dentro de la microVM y continuidad determinista. `P≤650, V≤600`. |
| **B11 — parada y cierre exactos** | `ORC-28`. El motor detiene de forma cooperativa y luego forzada el handle físico exacto; Orquesta deja de admitir claims, detiene el tick, espera lanzamientos reclamados y solo después llama `agent.Shutdown`. | `agentmicrovm/v1/firecracker/{controller,shutdown}.go`, `internal/adapters/agent/firecracker/{controller,shutdown}.go`, corrección mínima en `internal/bootstrap/` y pruebas. Depende B10; serial con composición. | `Engine.Stop/Shutdown` usa RunRef, PID/start-time, cgroup, CID/fence; el adaptador implementa `AgentController/AgentShutdown` y reconciliación Orquesta. Corrige el riesgo actual de cerrar Agent antes de drenar el planificador. Evita `pkill`, kill por nombre/cohorte y uso del motor cerrado. | Cooperativa, timeout→forzada, PID reutilizado, comprobante perdido, repetición, fence obsoleto, peer intacto, reinicio durante parada, cierre con lanzamientos en vuelo y limpieza exacta. Cierra con comprobante y ningún uso posterior al cierre. `P≤300, V≤350`. |
| **B12 — sello, inventario y compuerta B** | `ORC-28`. Tras parada/terminal, huésped y broker devuelven change-set/manifiesto; Orquesta verifica, persiste `preserved_pending_review` y solo entonces desmonta/libera. No borra. | Método público de preservación en `agentmicrovm/v1/firecracker/`, `internal/adapters/agent/firecracker/preserve.go`, integración A06/B06 y `acceptance/v38_agent_firecracker_single_vm_test.go`. Depende B06/B10/B11. | El motor emite inventario físico neutral; el adaptador añade workspace/config/tooling y persiste comprobantes CAS. Reusa `agent_firecracker_single_vm`; no comparte evidencia/rootfs TestAttestor ni expone delete. | Una microVM real, orden posterior, observación, parada, sello, inventario, desmontaje; negativos de digest/write-set/ref/atestación y reinicio antes/después del sello. Cierra con comprobante de compuerta B; no promueve V38 por sí sola. `P≤400, V≤500`. |
| **C01 — arnés y comprobación previa de olas** | `ORC-28`. El anfitrión declara capacidad real antes de la ola y nunca rebaja silenciosamente el peldaño. Todos los recursos pertenecen a un run aislado. | `acceptance/v38_agent_firecracker_wave_harness_test.go`, script/runbook acotado, manifiesto de presupuesto y datos de recursos. Depende B12. | Reusa perfil de anfitrión solo como lección; mide CPU/RAM/disco/KVM/CID/FD/cgroup y digests. Sella por candidato los cuatro límites exactos `wall_time_limit_ms`, `ram_peak_limit_bytes`, `disk_peak_limit_bytes`, `agent_token_limit`. | Positivos/ausentes/excedidos, reserva de disco/tokens, RAM disponible, nombres/temporales exactos, aborto y limpieza inventariada. Si no caben 20 o falta un límite, la compuerta queda bloqueada. `P≤300, V≤300`. |
| **C02 — olas físicas del mismo candidato** | `ORC-28`. Ejecuta `1/5/10/16/20` con una microVM por agente y el mismo digest candidato; el conjunto listo lógico puede ser mayor. Es compuerta física independiente de B04 y TestAttestor. | Arnés C01 y resultados estructurados bajo `product/evidence/candidates/`; usa sin cambios el manifiesto de tiempo/RAM/disco/tokens y no edita roadmap. Depende C01. | Reusa despachador, capacidad, motor v1, adaptador Orquesta, broker/proxy, observación, mailbox, parada y preservación. Evita reconstrucción entre pasos, selección manual y métricas desde logs. | Por ola registra `present`, inicio, fin, duración, `RunRef`, peldaño y digest para siete latencias: **solicitud de capacidad, aprovisionamiento, lanzamiento, disponibilidad, parada cooperativa, parada forzada y sello/inventario**. Cada peldaño ejecuta subcasos separados cooperativo y forzado controlado; cero no significa ausencia. Registra también RAM, disco y tokens solicitados/observados. Cierra solo si pasan cinco peldaños dentro de presupuesto. `P≤150, V≤400`. |
| **C03 — matriz adversarial, carreras y recuperación** | `ORC-28`. Seguridad, causalidad e idempotencia se mantienen bajo fallos reales; estados ambiguos nunca disparan reintento ciego ni borrado. | Pruebas de aceptación adversarial V38 y datos de inyección de fallos; no producto salvo corrección causal separada. Depende C02. | Reusa casos A/B: fuente stale/unknown, nueva ventana, caída en intento/efecto/comprobante, replay de credencial, CID ABA, DNS rebinding, VM vecina, comprobante de parada perdido y sello incompleto. Evita una suite feliz como evidencia total. | `go test -race`, reinicio/reapertura, parada controlada de procesos propios, negativos de auth/proyecto/traversal/red y escaneo final. Toda corrección repite C02 con nuevo digest candidato. Cierra sin P0/P1 abierto. `P≤50, V≤650`. |
| **C04 — sello y revisión independiente** | `ORC-28`. Evidencia pertenece al mismo árbol fuente, binario, rootfs y configuración efectiva; los comprobantes quedan fuera del sujeto y lo referencian. Autoridad: pipeline de release/TestAttestor más revisión primaria y adversarial. | Manifiesto/índice inmutable del candidato, comprobantes de aceptación y notas de revisión; ningún cambio funcional. Depende C03. | Reusa TestAttestor sin darle red/vsock y el modelo de evidencia actual. Evita autodigest imposible, comprobante manual y mezcla de candidatos. | Compuertas globales: `git diff --check`, unitarias/contractuales/integración, `go test`, vet, carreras relevantes, arquitectura/config/auth, backup/restore/cierre y cero procesos propios. Cierra con dos revisiones y digests coincidentes A+B+C. `P≤50, V≤300`. |
| **C05 — promoción canónica atómica** | `ORC-28`. Solo tras C04, roadmap y evidencia de la misma revisión promueven la vertical completa; ningún read-model derivado puede acreditar por sí solo. | Obligatorios en un changeset: `product/roadmap.json`, command/fixture/test reales, comprobante/índice de `product/evidence/`, decisión `agent_microvm_network`, `product/knowledge/tooling_adoption_v1.json`, `product_roadmap_test.go` y guardas `acceptance/v38_agent_runtime_elastic_plan_*`. `product/capabilities.json` solo se regenera si la composición/release canónica lo exige y referencia la misma evidencia. Depende C04; serial final. | En `product/roadmap.json`, `AC-V38-AGENT-RUNTIME-ELASTIC` pasa `planned→executable` con los campos reales `command`, `test_ref`, `fixture` y `receipt`; `ORC-28` pasa `declared→accredited` con `evidence_refs`; `agent_microvm_network` pasa `planned_not_applied→applied`. `tooling_adoption_v1.json` cambia Firecracker de `candidate` al estado acreditado, soportado por la misma evidencia y sin volverse autoridad. | Sustituye las guardas vigentes que exigen contrato V38 `planned` y sin recibo, ausencia de evidencia V38, `ORC-28` `declared`, decisión `planned_not_applied` y Firecracker `candidate`; las nuevas guardas comprueban refs/digests coincidentes. Si falta un campo real, comando ejecutable o comprobante A+B+C, nada se promueve. Cierra V38 en una revisión indivisible. `P≤100, V≤250`. |

## Ejes ortogonales y alcance real de proveedor

Proveedor, aislamiento, transporte, persistencia, artefactos y control son seis
conectores hexagonales ortogonales. El proveedor implementa el contrato neutral
de sesión, turnos y eventos; el aislamiento decide si la ejecución vive en
proceso o en una microVM; transporte mueve mensajes; persistencia conserva el
journal durable; artefactos mueve contenido por referencia y digest; control
observa, reorienta, interrumpe y detiene. Ninguno decide o incorpora a los
demás, y la composición los ensambla mediante puertos tipados. Para proveedor y
aislamiento elige una opción de cada eje, pero no crea adaptadores
`codex_process`, `codex_microvm`, `claude_process`, `claude_microvm` ni una
matriz N×M equivalente.

`application` decide admisión, autorización y mutaciones mediante puertos.
`StateRepository` y `RuntimeJournal` son contratos distintos y cohesivos sobre
la única fuente transaccional seleccionada; SQLite es el conector local, de
desarrollo y de pruebas actual. Bootstrap únicamente construye e inyecta la
composición y rechaza dos fuentes activas: no decide política, no escribe ciclo
de vida y no habilita escritura doble. PostgreSQL será el conector productivo
futuro de V31/`OPS-11`, fuera de V38, y deberá sustituir a SQLite en una
composición, no replicarlo como segunda autoridad.

El protocolo y supervisor neutrales viven respectivamente en
`agentmicrovm/v1/guestproto` y `agentmicrovm/v1/guestsupervisor`, fuera de
`firecracker`, para que los importen el traductor del rootfs y otros transportes
o aislamientos. `agentmicrovm/v1/firecracker` se limita al lanzamiento físico,
vsock, observación y desmontaje. La API pública no exporta enumeraciones, DTO,
configuración o errores propios de Codex, Claude, Gemini, Ollama o un agente
local. El traductor concreto queda en la imagen o adaptador del proveedor y se
sustituye sin cambiar el motor físico.

V38 debe ejecutar Codex real porque es el proveedor candidato configurado para
esta acreditación. Esa prueba demuestra únicamente `ORC-28` con la composición
Codex+Firecracker; no implementa ni acredita la matriz de proveedores de V25,
ni permite afirmar soporte de Claude, Gemini, Ollama o agentes locales. Esos
proveedores deberán superar posteriormente su propio contrato y pruebas de V25
sin modificar la API física neutral.

El codec común Codex de A05 fija explícitamente la versión y el schema
compatibles de `codex app-server`, las tramas JSONL, los identificadores de
petición, la correlación de eventos, los límites y
`initialize`→`initialized`. Lo reutilizan exactamente dos consumidores:
controlador anfitrión de cuota y enlace del huésped. No migra ni modifica el
worker de proceso actual. Reutilizar código no permite compartir estado vivo:
cada consumidor arranca una instancia separada, con proceso, `stdio`, socket,
`CODEX_HOME`, credenciales, hilo y turno propios. El controlador anfitrión
negocia capacidades nulas y tiene una lista cerrada de observación de
cuota/elegibilidad; no puede usar
métodos de hilo, turno, steering, parada de agente, workspace, mailbox ni
artefactos. Los errores remotos se validan y su contenido se descarta antes de
salir del codec. Si el protocolo real no ofrece cuota estructurada acreditable, A05
queda bloqueada y cerrada a lanzamientos.

B05 adapta `guestproto` al mismo codec y arranca `codex app-server` dentro del
huésped. Cada arranque y reinicio completa la inicialización antes de cualquier
operación. Una petición iniciada por el servidor sin manejador/autorización
explícitos no se confunde con una respuesta y falla cerrada. Quedan prohibidos
WebSocket, tmux, lectura heurística de terminal, scraping de logs, fichero de
cuota y órdenes iniciales sin canal posterior.

## Microencargos operativos de los padres grandes

Estas divisiones son hijos operativos de las tareas de la tabla anterior: no
añaden presupuesto, otra capability ni otro cierre. Cada hijo debe producir un
commit pequeño y una revisión propia. Los hijos que comparten fichero son
seriales; solo los declarados paralelos pueden abrirse a la vez. Cada suma
coincide exactamente con el techo de su padre. Ningún encargo pendiente supera
`P=200, V=200`; las bases ya integradas y A05.3a informan su medida real y no
autorizan otro write-set de ese tamaño.

### A02 — `P=159, V=226`

| Hijo | Write-set y resultado | Orden | Presupuesto |
|---|---|---|---:|
| A02.0 | Base `7e232072`: demanda y admisión multidimensionales sin perder ceros presentes. | Integrada, no acreditante por sí sola. | `P=77, V=120` consumidos |
| A02.1 | Separación neutral entre capacidad física, cuota no reservable y colocación opaca. | Integrada, no acreditante por sí sola. | `P=57, V=65` consumidos |
| A02.2 | Materialización CAS de la observación de cuota con revisión, idempotencia y colocación exactas. | Integrada en `fdfab8c7`; abre A03.2/A03.3. | `P=25, V=41` consumidos |

### A03 — `P=334, V=561`

| Hijo | Write-set y resultado | Orden | Presupuesto |
|---|---|---|---:|
| A03.1 | Base ya integrada en `71f4a827`: observación/reserva física y `022_agent_capacity.sql`, incluida su prueba canónica. Se conserva y se caracteriza; no se vuelve a presupuestar ni se edita la migración aplicada. | Completada parcial, no acreditante. Medición conservadora del diff ya integrado. | `P=139, V=291` consumidos |
| A03.2 / Q2 | Migración progresiva con el hecho completo `AgentQuotaObservationRecord` y `AgentPlacementBinding` 1:1 de cuatro columnas, FK diferida a reserva y cuota/colocación/revisión exactas; sin clase, saldo ni lifecycle de cuota. | Después de A04.1; exclusión sobre migraciones. No añade operación de escritura del binding. | `P=100, V=150` |
| A03.3 / Q3 | `StateRepository` obtiene la observación vigente por colocación y hace append CAS/idempotente. SQLite implementa ese mismo puerto para local/desarrollo/pruebas; no aparece otro store. | Después de Q2 y antes de A04.2. PostgreSQL implementará el mismo contrato productivo en V31. | `P=95, V=120` |

### A04 — `P=459, V=447`

| Hijo | Write-set y resultado | Orden | Presupuesto |
|---|---|---|---:|
| A04.1 | Base ya integrada en `33802c27`: estado de observación/reserva que ahora se caracteriza y migra al binding 1:1, sin segundo lifecycle. | Después de A02b; habilita A03.2, nunca A04.2 directamente. Completada parcial, no acreditante. | `P=114, V=53` consumidos |
| A04.1b | Binding neutral exacto ya integrado en `4ccfedc0`; no posee writer, store ni lifecycle propios. | Después de A04.1; caracteriza Q2/Q3/A04.2. | `P=46, V=57` consumidos |
| A04.2 / Q4 | `CapacityCandidates` ordenados/deduplicados en `ClaimRequest`; dentro del `BEGIN`, CAS revalida en orden y fija/reserva el primero disponible. Solo esta operación crea a la vez reserva física y binding 1:1. | Después de A03.3 y A05.4; exclusión sobre claim/SQLite. | `P=160, V=190` |
| A04.3 | Consume/libera/cuarentena la única reserva física. Multiperfil reserva 1 del pool unitario del placement; perfil único reserva 1 de N configurado. Held se resta una vez por `source+pool` en `reserved/consumed/quarantined`; nunca por `observation_ref` ni sobre capacidad ya neta. Cuota, preflight Firecracker y leases CID/VM no crean otra reserva. | Después de A04.2; serial con A06. | `P=100, V=80` restantes |
| A04.4 | Reinicio, lease expirado, recuperación ambigua, carrera del último candidato, replay sin reselección, doble contabilidad, proyecto, ausencia de límite Codex global y exactamente una `AgentCapacityReservation` por launch. | Último; no abre A07 hasta quedar verde. | `P=39, V=67` restantes |

### A05 — `P=608, V=508`

| Hijo | Write-set y resultado | Orden | Presupuesto |
|---|---|---|---:|
| A05.1 | `config/registry.json` y salidas generadas: ejes `provider`/`isolation`, fuente, `runtime.capacity.observation_ttl` y máximo de lectura del informe. Ya integrado en `87db76eb`; sus `P=49, V=49` están consumidos, pero el lector de informe y su clave se retirarán en A05.1d/A05.3b. | Primero y serial sobre registro; completado parcial, no acreditante. | `P=49, V=49` |
| A05.1b | `internal/bootstrap/runtime.go`, `cmd/orquesta/main.go`, catálogos `es.json`/`en.json`, manifest y pruebas compactas/reutilizadas: antes de construir renderizador, credenciales o Codex/proceso, `microvm` devuelve exactamente `bootstrap.runtime_isolation_not_composed`, presentado con `error.bootstrap.runtime_isolation_not_composed`; la CLI conserva el código y no lo degrada a `internal`. Cero fallback/recursos; `process` sigue verde. | Después de A05.1; antes de A05.4 y permanece serial con B01/B10 hasta que B10 lo sustituya. | `P=30, V=21` |
| A05.1c | `config/registry.json` y proyecciones generadas: añadir `runtime.capacity.observation_timeout` positivo, separado de `observation_ttl` y de `runtime.codex.*`; comprobar valor por defecto, valor TOML explícito, rechazo de cero y sincronización canónica de todas las proyecciones. | Después de A05.1; serial sobre registro y antes de A05.4. | `P=14, V=5` |
| A05.1d | Registro/bootstrap/scheduler: presupuesto global neutral, separación del límite Codex y tamaño máximo de trama `app-server`; retira la clave de informe sin alias ni fallback. | Integrada; serial sobre registro/bootstrap/scheduler. | `P=21, V=44` consumidos |
| A05.2 | Fuente física configurada y candidatos opacos: slots brutos por `source+pool`, ventanas, ceros presentes, frescura y `AgentPlacementRef`; declara si la medida es bruta y rechaza la doble resta. | Tras A02b; paralela con A05.3a/A05.3b. | `P=53, V=64` consumidos |
| A05.3a | `internal/adapters/agent/codex/appserver/`: codec JSONL acotado con IDs/correlación, inicialización oficial, capacidades nulas, métodos de cuota exactos, descarte de errores remotos y fallo terminal por exceso. No arranca procesos ni importa Firecracker/application/config. | Tras A02b; precede A05.3b y B05.3. | `P=278, V=179` |
| A05.3b | Controlador Codex anfitrión solo de cuota: un `app-server` persistente por perfil, lectura inicial/eventos, reconexión y rotación con cierre exacto; elimina fichero/fallback. | Tras A05.3a; disjunto de la fuente física. | Envolvente conjunta A05.3b+A05.4: `P≤163,V≤146` |
| A05.4 | Application/bootstrap inicia controladores antes del planificador, espera lectura inicial, ordena/deduplica candidatos y recoge todos los recursos ante fallo o shutdown. | Después de A05.1b–A05.3b; prerequisito de A04.2. | Misma envolvente conjunta; debe asignarse antes de abrir ambos write-sets |

El techo acumulado de A05 es `P=608,V=508`:
`49+30+14+21+53+278+163=608` y
`49+21+5+44+64+179+146=508`. A05.3b y A05.4 siguen siendo tareas causales
separadas; compartir envolvente evita inventar una precisión antes de medir su
frontera. No se usa contingencia ni se añade otro almacén, writer, planificador
o bucle de dominio.

### B01 — `P=0, V=50`

| Hijo | Write-set y resultado | Orden | Presupuesto |
|---|---|---|---:|
| B01.1 | Gate compacto: verifica selección A04, fuentes/controlador A05 y composición B10.4 sin reimplementarlos. Cubre microVM sin fallback, controlador anfitrión solo de cuota, procesos separados y ausencia de límite Codex global. | Tras A05/A08/B03/B04; abre B10. | `P=0, V=50` |

### B05 — `P=490, V=320`

| Hijo | Write-set y resultado | Orden | Presupuesto |
|---|---|---|---:|
| B05.1 | `agentmicrovm/v1/guestproto/`: schema y tramas neutrales versionadas para hilo, turno, control y eventos. | Primero; no importa Firecracker ni proveedor. | `P=100, V=70` |
| B05.2 | `agentmicrovm/v1/guestsupervisor/`: supervisor neutral, reconexión, deduplicación y contrapresión. | Tras B05.1; paralelo con B05.3. | `P=110, V=60` |
| B05.3 | Enlace fino huésped: adapta `guestproto` al codec A05.3a y arranca su propia instancia dentro de la microVM. Ningún adaptador Firecracker importa el codec. | Tras B05.1 y A05.3a; paralelo con B05.2. | `P=50, V=30` |
| B05.4 | Constructor de rootfs, kernel/init, manifiesto, SBOM, licencias, owner/modos y digest reproducible. | Después de B05.2–B05.3; serial sobre imagen. | `P=160, V=100` |
| B05.5 | Incompatibilidad, preinicialización, reinicio, petición no autorizada, replay/desorden, aislamiento host/huésped, límites y ausencia de WebSocket/tmux/secretos. | Último; mismo candidato de B05.4. | `P=70, V=60` |

### B07 — `P=500, V=450`

| Hijo | Write-set y resultado | Orden | Presupuesto |
|---|---|---|---:|
| B07.1 | `firecracker/launcher.go`: plan físico, assets sellados, jailer/cgroup y una VM por `RunRef`. | Primero; no importa proveedor ni `internal`. | `P=150, V=120` |
| B07.2 | `firecracker/{engine,recovery}.go`: handle durable, journal/CID como puerto e idempotencia de lanzamiento. | Tras B07.1; serial sobre `engine.go`. | `P=130, V=110` |
| B07.3 | `firecracker/observer.go` y reconciliación exacta PID/start-time/cgroup/CID tras reinicio. | Tras B07.2; write-set disjunto del launcher. | `P=120, V=110` |
| B07.4 | Fallos por etapa, PID reutilizado, KVM/OOM, limpieza exacta y suite contractual sin recursos huérfanos. | Último; no modifica API salvo incidencia causal separada. | `P=100, V=110` |

### B08 — `P=550, V=450`

| Hijo | Write-set y resultado | Orden | Presupuesto |
|---|---|---|---:|
| B08.1 | `internal/adapters/agent/firecracker/broker/`: enlace del `ServiceMux` neutral a una `RunRef`, recibiendo sus puertos por constructor y sin comandos privados. | Primero; fija la frontera del broker. | `P=120, V=100` |
| B08.2 | Caracterizar `internal/adapters/agent/firecracker/networkauth/verifier.go` y el código `agent_firecracker_network_auth`; mapear/retirar `agent_microvm_launch_auth`, mover lo neutral a `agentmicrovm/v1/launchauth/` y la traducción a `internal/adapters/agent/launchauth/`, sin dependencia Firecracker ni ruta nominal anterior. | Tras B08.1; serial sobre el contrato de autorización. | `P=160, V=130` |
| B08.3 | Traducción a sesión, `ArtifactSource/Sink`, MCP y mailbox exactos mediante puertos inyectados; contenido grande solo por ref. | Tras B08.2; no importa el adaptador de autorización ni el almacén concreto. | `P=170, V=130` |
| B08.4 | Replay/concurrencia, reinicio, alcance cruzado, orden posterior y comprobante perdido. | Después de B08.2–B08.3. | `P=100, V=90` |

### B09 — `P=600, V=550`

| Hijo | Write-set y resultado | Orden | Presupuesto |
|---|---|---|---:|
| B09.1 | `agentmicrovm/v1/firecracker/egress.go`: concesión neutral, servicio vsock y comprobante estable. | Primero; congela API pública. | `P=110, V=90` |
| B09.2 | `agentmicrovm/v1/firecracker/egress_resolver.go`: resolución DNS, redirecciones e identificación IPv4/IPv6; bloqueo de loopback, privada, ULA, link-local y metadata. | Tras B09.1; paralelo con B09.3 sobre fichero público disjunto. | `P=170, V=160` |
| B09.3 | `agentmicrovm/v1/firecracker/egress_receipts.go`: límites de tiempo/bytes, idempotencia y comprobantes ante éxito, rechazo y pérdida de respuesta. | Tras B09.1; paralelo con B09.2 sobre fichero público disjunto. | `P=140, V=130` |
| B09.4 | `internal/adapters/agent/firecracker/egresspolicy.go`: permisos/presupuesto a concesión neutral, sin política Orquesta en público. | Tras B09.1; disjunto de B09.2/B09.3. | `P=100, V=90` |
| B09.5 | DNS rebinding, redirect privado, VM vecina, inbound e Internet directo negativos. | Último, sobre B09.2–B09.4. | `P=80, V=80` |

### B10 — `P=650, V=600`

| Hijo | Write-set y resultado | Orden | Presupuesto |
|---|---|---|---:|
| B10.1 | `internal/adapters/agent/firecracker/agent.go`: `AgentLauncher` a `Engine`, refs opacas y lanzamiento Codex real sin tipo Codex público. | Tras B04/B06–B09; paralelo con B10.2/B10.3. | `P=150, V=130` |
| B10.2 | `observer.go` y `recovery.go`: observación, control y fronteras definitely/unknown applied. | Paralelo con B10.1/B10.3; ficheros disjuntos. | `P=130, V=120` |
| B10.3 | `dependencies.go`: constructor que solo recibe `ArtifactSource/Sink`, `RuntimeJournal`/`CIDLease`, `LaunchAuthorizer`, reloj y servicios cohesivos; sin imports de adaptadores. | Paralelo con B10.1/B10.2. | `P=120, V=100` |
| B10.4 | `internal/bootstrap/agent_provider.go` y `cmd/orquesta/`: única composición de proveedor+aislamiento y conectores concretos. Retira el rechazo incondicional A05.1b solo al inyectar Firecracker completo; prueba motor alcanzado, composición parcial aún cerrada y espía sin construcción Codex/proceso, por tanto sin fallback. | Después de B10.1–B10.3; serial en bootstrap y sobre A05.1b. | `P=130, V=120` |
| B10.5 | Reinicio en cinco fronteras, fence, duplicado, comprobante perdido, observación concurrente y `-race`. | Último; no ensancha composición durante la prueba. | `P=120, V=130` |

### B12 — `P=400, V=500`

| Hijo | Write-set y resultado | Orden | Presupuesto |
|---|---|---|---:|
| B12.1 | Método físico público de preservación: inventario neutral, sello, digests y handle exacto. | Primero; no conoce workspace ni proveedor. | `P=100, V=100` |
| B12.2 | `internal/adapters/agent/firecracker/preserve.go`: añade workspace/config/tooling y persiste comprobante CAS de A06. | Tras B12.1 y B06; serial sobre adaptador. | `P=120, V=130` |
| B12.3 | Orden parada→sello→inventario→persistencia→desmontaje/liberación y recuperación en cada frontera. | Tras B12.2/B11; serial sobre cierre. | `P=100, V=130` |
| B12.4 | `acceptance/v38_agent_firecracker_single_vm_test.go`: Codex real, orden posterior, stop, preservación y negativos. | Último; emite solo compuerta B. | `P=80, V=140` |

## Frontera pública reutilizable

El árbol público será `orquesta/agentmicrovm/v1` y su implementación física
`orquesta/agentmicrovm/v1/firecracker`. Permanecen dentro del módulo actual
durante V38: no se crea ahora otro módulo, repositorio, proceso residente ni
microservicio. La versión en la ruta permite estabilizar la superficie antes
de una posible extracción posterior.

La API pública solo expresa la gestión física:

- `Engine` agrupa lanzamiento, observación, canal de órdenes, parada,
  preservación y cierre sobre una `RunRef` opaca.
- `Config` contiene únicamente configuración tipada del motor: binario,
  kernel/rootfs por digest, límites físicos, jailer/cgroup, vsock, tiempos y
  política de red neutral. No recibe el registro global ni claves TOML.
- `Dependencies` recibe puertos cohesivos por sistema externo real: reloj,
  `ArtifactSource`, `ArtifactSink`, autorización/confianza neutral,
  `RuntimeJournal`/`CIDLease` y multiplexor de servicios vsock. Cada puerto
  tendrá al menos un sustituto contractual y un consumidor real; no habrá
  interfaz por operación.
- Requests, handles, observaciones y comprobantes usan refs opacas,
  idempotency keys, fencing y códigos máquina estables. El paquete público no
  contiene texto humano, catálogos i18n, rutas Orquesta ni globals.

El motor no importa `internal`, `database/sql`, SQLite, CAS, `CredentialStore`,
atestadores ni paquetes de dominio. Su API tampoco contiene `Goal`, `WorkItem`,
`Execution`, permisos o presupuestos. El adaptador de
`internal/adapters/agent/firecracker` recibe solo los puertos neutrales ya
construidos y no importa otros adaptadores. Bootstrap es el único lugar que
ensambla los conectores concretos con ese adaptador; `application` conserva la
decisión y escritura causal. Así una aplicación futura puede aportar otros
conectores sin arrastrar el ciclo de vida de Orquesta.

B04 verifica esa separación desde un módulo temporal consumidor. Guardas
`go list -deps` y `rg` comprueban tanto API/motor como la dirección de imports:
ningún adaptador importa SQLite, CAS, credenciales, atestación u otro adaptador
para componerlos. La extracción física solo se estudiará después de estabilizar
v1 y medir sus dependencias; no forma parte de V38 ni autoriza publicar nada.

## Retirada de solapamientos y corrección P2 separada

- B02 compara campo por campo `internal/ports/agent_microvm_vsock_cid.go` con
  `RuntimeJournal`/`CIDLease`, migra consumidores y elimina el contrato anterior
  al cerrar la tarea. El allocator SQL se mueve a
  `internal/adapters/state/sqlite/vsocklease/`; no queda bajo Firecracker ni se
  mantiene un bridge.
- B08 hace lo mismo con
  `internal/ports/agent_microvm_launch_auth.go` y caracteriza explícitamente
  `internal/adapters/agent/firecracker/networkauth/`, incluido `verifier.go` y
  sus errores `agent_firecracker_network_auth`. Las primitivas realmente
  neutrales de desafío, prueba y transacción pasan a
  `agentmicrovm/v1/launchauth/`; la traducción de credenciales y atestación de
  Orquesta pasa a `internal/adapters/agent/launchauth/`, recibe sus puertos y
  no depende de Firecracker. Tras migrar consumidores se elimina la ruta
  `firecracker/networkauth/` y su código nominal, sin conservar un segundo
  adaptador. Bootstrap inyecta la única implementación en broker y motor, que
  no la importan.
- El hallazgo en `internal/application/budget_policy.go:115` se registra como
  corrección propia de política de presupuesto, con capability, invariante,
  caracterización y test de regresión antes de modificar código. Si condiciona
  la admisión de A04, se resuelve en un write-set serial previo; nunca se mezcla
  con la migración A03, no altera su schema y no se oculta dentro de su
  presupuesto.

## Seguimiento post-V38: D01 — extracción versionable

D01 depende de C05 y debe cerrar antes del primer consumidor externo
productivo. Solo se activa cuando exista un segundo consumidor real y una
cadencia/owner de releases demostrados. Si falta cualquiera, permanece
`planned` y no se crea un repositorio o módulo vacío.

Su write-set y presupuesto se autorizarán en un encargo posterior. Extraerá la
superficie ya acreditada `agentmicrovm/v1` y `.../firecracker` a una librería,
módulo o repositorio versionable; conservará la misma suite contractual y
códigos máquina. No añadirá daemon, DB, almacén, planificador, ciclo de vida ni
compatibilidad privada con Orquesta.

El cierre de D01 exige ejecutar la suite de la librería extraída, compilar un
consumidor mínimo fuera del repositorio, revalidar el adaptador Orquesta contra
la versión publicada y repetir las pruebas de composición afectadas. D01 no
modifica la evidencia ni el estado acreditado de V38; cualquier incompatibilidad
abre una nueva versión, no reescribe v1.

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

### Codec de `app-server` y aislamiento de procesos

El codec común específico de Codex solo conoce framing, IDs, inicialización,
correlación, allowlists y límites de protocolo. No conoce Firecracker, Goal,
workspace, cuota política ni lifecycle. Evita codecs divergentes, pero no crea
un proceso compartido.

Por cada perfil de cuota configurado, bootstrap arranca exactamente un proceso
persistente `codex app-server` y un lector técnico de protocolo acotado a esa
conexión. Completa `initialize`→`initialized`, solicita y persiste la primera
lectura válida y solo entonces arranca el planificador. El lector procesa
respuestas/eventos enmarcados; no sondea, planifica, elige colocación ni escribe
lifecycle. La conexión se reutiliza aunque haya cero agentes o una ola
`5/10/20`.

Ante pérdida de conexión, el controlador marca la cuota desconocida, cierra y
recolecta el hijo exacto y reconecta con límites. La rotación hace lo mismo y
no publica la sustitución hasta obtener su lectura inicial. Shutdown cancela
el lector, cierra `stdio`, espera el proceso y, si vence el plazo, mata solo su
PID verificado y lo recolecta antes de devolver. Un fallo durante bootstrap
recoge todos los perfiles ya iniciados y no arranca el planificador.

Controlador anfitrión y workers huéspedes usan instancias separadas, con HOME,
credenciales, `stdio`/socket, hilo, turno y permisos acotados. El codec no
migra el worker de proceso actual. El controlador no implementa
`AgentLauncher`, `AgentObserver` o `AgentController`, no abre turnos ni accede
a workspace, mailbox o artefactos. Es un conector técnico del monolito modular,
no un daemon de dominio, microservicio, planificador, almacén o writer. Ningún
adaptador Firecracker importa el codec Codex.

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

La identidad de operación contiene refs de proyecto, goal, work item,
execution, generación, action/attempt, motor externo y fence. PID y CID son
datos de routing sujetos a reutilización, no identidad suficiente.

El huésped no recibe NIC. Los únicos servicios vsock permitidos son
`orquesta_broker` y `controlled_egress_proxy`. La aplicación exige prueba de un
uso, alcance exacto y atestación mediante sus puertos internos; el adaptador
Orquesta puede consumir `CredentialStore` y el atestador detrás de esos
puertos, pero el motor y su API solo reciben `LaunchAuthorizer`. El desafío en
memoria se invalida tras reinicio. No se entregan secretos en rootfs,
configuración efectiva, Goal, prompt, log, artefacto o comprobante.

El proxy vuelve a resolver DNS y revalida cada redirect. Debe negar Internet
directo, inbound, east-west, loopback, redes privadas/link-local/metadata y
todo destino no permitido. Un ACK de broker solo confirma admisión: el efecto
se acredita mediante el comprobante del registro de efectos.

### Workspace, stop y preservación

El anfitrión produce un bundle/snapshot ligado al workspace y base exactos. La
composición conecta el CAS actual con `ArtifactSource`/`ArtifactSink`; motor y
huésped solo ven refs, digests, media type y bytes verificados por vsock. El
huésped no monta `.git` ni filesystem del anfitrión. El resultado vuelve por el
sumidero y pasa por la validación e integración en anfitrión ya acreditadas.

La parada cooperativa y forzada actúa solo sobre la identidad exacta. Si se
pierde el comprobante, se reconcilia antes de repetir; nunca se ejecuta un kill
amplio.
Antes de liberar CID, socket, cgroup o mount, el adaptador obtiene sello e
inventario, application los valida y persiste `preserved_pending_review`.
V38 no contiene API de borrado automático. La eliminación futura será otro
efecto autorizado, con scope e idempotencia propios.

## Riesgos y ratchets

- **Concurrencia en fronteras compartidas:** A04 y A06 no editan a la vez
  reclamo, procesamiento o SQLite. El despachador A07 ya está integrado; A04
  deberá extenderlo sin crear otra ruta ni deshacer sus garantías.
- **Migraciones concurrentes:** A03 y B02 toman el siguiente número libre solo
  al integrar. Una colisión se resuelve reenumerando la migración propia, no
  sobrescribiendo trabajo ajeno.
- **Autoridad de persistencia:** composición rechaza dos fuentes
  transaccionales activas. El núcleo escribe por `StateRepository`; SQLite es
  el conector local, de desarrollo y de pruebas actual, y PostgreSQL será el
  conector productivo futuro de V31/`OPS-11`. PostgreSQL no entra en V38 y
  nunca se añade como escritura doble ni fallback.
- **Dirección hexagonal:** guardas `go list`/`rg` impiden que Firecracker
  conozca SQL/SQLite/CAS/credenciales/atestación y que un adaptador importe otro
  para componerlo. Solo bootstrap ensambla; `application` decide.
- **Confundir TestAttestor con agente:** las guardas de imports/composición deben
  impedir compartir protocolo, rootfs, comprobantes o evidencia. Solo
  primitivas neutrales pequeñas, extraídas con dos consumidores y pruebas, son
  reutilizables.
- **Cuota falsa o stale:** fail-closed para launches nuevos y continuidad de
  control para ejecuciones existentes. Ningún «desconocido = ilimitado»,
  lector de fichero ni reserva contable de cuota.
- **Colocación no repetible:** application ordena/deduplica antes del claim; la
  transacción fija el primer candidato válido, lo persiste como ref opaca y el
  replay no cambia de perfil o pool. Bootstrap no toma esa decisión.
- **Procesos Codex mezclados:** codec común no significa instancia común.
  Guardas prueban scopes separados y que el controlador anfitrión solo consulta
  cuota; el trabajo microVM siempre se ejecuta dentro del huésped.
- **Ciclo de cuota incompleto:** todos los perfiles completan lectura inicial
  antes del planificador. Las olas 5/10/20 reutilizan conexiones; reconexión,
  rotación, fallo parcial de bootstrap y shutdown recolectan proceso y lector
  exactos. Cero agentes también prueba el ciclo completo.
- **Recursos huérfanos:** todo run inventaría PID, socket, CID, cgroup,
  temporal y CAS propios. Limpia solo targets verificados; ante ambigüedad
  preserva y reporta.
- **Anfitrión insuficiente:** no se reduce C de 20 a un número disponible ni se
  usa el resultado histórico `1+16`. Se documenta bloqueo y se repite en un
  anfitrión apto con el mismo candidato.
- **Crecimiento arquitectónico:** máximo un despachador, una fuente
  transaccional activa por despliegue, un registro de efectos, un registro de
  comandos y un registro de configuración. Broker/proxy/huésped y
  motor son
  fronteras físicas o de seguridad del adaptador, no microservicios internos.
  El árbol público se limita a contrato v1, implementación Firecracker y suite
  contractual; no importa `internal`. No se aceptan paquetes de una constante,
  almacenes, planificadores, colas ni bucles de dominio, sondeo o planificación,
  ni otro módulo antes de estabilizar la API. Se permite solo un lector técnico
  acotado por conexión externa que bootstrap posee y cierra.
- **Presupuesto:** la suma base exacta de las 25 tareas V38 sigue siendo
  `P=7.200` LOC no generadas y `V=9.800` LOC de verificación/arneses. El
  subtotal afectado A01+A02+A03+A04+A05+A08+B01+B05 permanece
  `P=2.050,V=3.000`: `0+159+334+459+608+0+0+490=2.050` y
  `138+226+561+447+508+750+50+320=3.000`. A02 incorpora Q1 una sola vez;
  A03 incorpora la base `139/291`; A04 incorpora base y binding `160/110`.
  El código generado se informa aparte y ninguna holgura se descuenta dos veces.
  Existe una contingencia separada máxima de `P=300`, no asignada: usarla exige
  ADR y autorización explícita previas que nombren frontera inevitable y
  retirada/compensación concreta. D01 queda fuera y recibirá presupuesto propio
  al activarse. Objetivo adicional: cero nuevos escritores de ciclo de vida,
  cero almacenes, cero bucles de dominio/sondeo/planificación y cero comandos
  públicos salvo contrato de aceptación; los lectores técnicos de protocolo
  quedan acotados a su conexión y lifecycle de bootstrap.

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
