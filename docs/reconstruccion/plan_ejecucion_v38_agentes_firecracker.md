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
  `scripts/consultar_lecciones_legacy.sh` no tiene una entrada exacta para
  `ORC-28`; las consultas por `ORC-03`, `ORC-10`, `OPS-16`, `EXT-10`,
  `EVD-13` y `EXT-21` aportaron invariantes de caracterización, no autoridad
  de cierre.

## Lo que ya existe y no se vuelve a construir

- `Goal.RunnableWorkItems` y `Goal.ReadyWorkItems` derivan el conjunto completo,
  determinista y libre de conflictos. V38 lo caracteriza a escala; no crea
  otro planificador, ready queue, cohort manager ni state machine.
- `application.Orchestrator` es el único escritor del ciclo de vida;
  `StateRepository` y SQLite son la única fuente activa. Planificador, eventos,
  outbox y proyecciones no obtienen autoridad propia.
- El registro de efectos ya separa intent, aprobación, intento y comprobante, y ya
  trata `unknown_applied` como cuarentena. V38 lo extiende a capacidad y
  microVM; no crea un ledger paralelo.
- Los budgets durables y slots de proceso ya existen. Siguen gobernando
  presupuesto, pero no se disfrazan de observación de cuota/licencia externa.
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
  autenticación y allocator SQL. Se conectan y endurecen; no se duplican.
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

No existe todavía un motor persistente de agentes Firecracker, una fuente
neutral de capacidad viva, composición canónica de leases CID, rootfs de
agente, protocolo del huésped, broker/proxy, retorno sellado ni evidencia
física `1/5/10/16/20`. Tampoco existe aún la frontera pública importable del
motor físico. Esas son las brechas que siguen.

## Orden causal y write-sets

`A01 + A02 → A03 → A04 → A06 → A07`, con `A05` en paralelo tras A02,
`→ A08 →`
`B03 → [B02 | (B05 + B06 → B07) | B08 | B09]`;
`B07 + B09 → B04 → B01`; `B01 + B02 + B04 + B07 + B08 + B09 → B10 → B11 → B12 →`
`C01 → C02 → C03 → C04 → C05`.

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
| **A01 — cohortes lógicas** | `ORC-28`. El conjunto listo completo pertenece al `Goal`; el límite físico nunca elimina ni recrea `WorkItems`/`Executions`. Autoridad: `Goal` y `application`. | `internal/goal/v38_ready_set_test.go`, `internal/application/v38_schedule_ready_test.go`, dato estructurado en `acceptance/fixtures/`. Sin dependencia; paralela con A02. | Reusa `RunnableWorkItems`, `ReadyWorkItems`, `scheduleReady`. Preserva conflicto/write-set y orden determinista. Evita gestor de cohortes, paginador de DAG y segunda cola. | Pruebas normales y de propiedades para `1/16/70/500`; negativos de dependencia, ciclo/conflicto y presupuesto. La capacidad parcial no crea ni reemplaza una `Execution` o `Attempt` ya admitidos. Cierra cuando las cuatro cohortes se materializan completas sin tope implícito. `P≤0, V≤250`. |
| **A02 — contrato neutral de capacidad** | `ORC-28`. Capacidad es observación externa durable, no presupuesto ni texto de agente. Autoridad de frescura/política: `application`, usando su reloj. | `internal/application/agent_capacity.go`, `internal/application/agent_capacity_test.go`. Sin dependencia; paralela con A01. | Define necesidad consumidora `AgentCapacityObserver` y valores con presencia explícita: `SourceRef`, `PoolRef`, `ObservedAt`, `ExpiresAt`, `Quality`, `WindowRef`, `ResetAt`, límites/restantes presentes o ausentes, `RetryAt` y ref de artefacto opcional. Dos consumidores reales serán fuente configurada y fuente Codex. Evita almacén, servicio o planificador de cuotas. | Pruebas contractuales: cero presente no se pierde; ausente no equivale a ilimitado; `unknown/stale/unavailable` da cero reservas nuevas; nueva `WindowRef` válida puede reabrir; reloj del proveedor no decide frescura; payload/log no se hace autoridad. Cierra con contrato estable y sustituto contractual. `P≤200, V≤250`. |
| **A03 — persistencia canónica de capacidad** | `ORC-28`. Observación, reserva, consumo y liberación son hechos del mismo estado; revisión esperada, fencing e idempotencia son atómicos. Autoridad: `StateRepository`/SQLite. | `internal/adapters/state/sqlite/migrations/NNN_agent_capacity.sql`, pruebas de schema/migración en `internal/adapters/state/sqlite/agent_capacity_schema_test.go`. Depende A02; migración serial exclusiva. | Reusa la DB, transacciones y cadena de migración actuales. Solo tablas/índices necesarios ligados a refs opacas y revisión; no crea DB, outbox, journal ni almacén. | Actualización desde revisión anterior, rollback transaccional, FK/unique/fence, aislamiento de proyecto, reapertura y backup/restore. Cierra al migrar una sola vez y al no cambiar el ciclo de vida ni duplicar eventos. `P≤150, V≤250`. |
| **A04 — reserva durable en el claim** | `ORC-28`. Solo el claim global de `launch_agent` reserva capacidad; stop/observe siguen reclamables con capacidad cero. Reserva ligada a proyecto/goal/work/execution/action/generación/fence/idempotency/pool/ventana. Autoridad: `application`. | `internal/application/state.go`, `internal/application/agent_capacity_state.go`, `internal/adapters/state/sqlite/agent_capacity.go` y pruebas. Depende A02/A03; serial con A06/A07 por frontera compartida. | Extiende `ClaimNextAction`, el registro de efectos y la revisión esperada. Consume con comprobante de lanzamiento aceptado; libera solo al terminal o `definitely_not_applied`; `unknown_applied` retiene/cuarentena. Evita cola de capacidad, semáforo en memoria, reencolado privado y nuevo intento. | Normal, dobles claims, agotamiento, cero explícito, expiración entre claim/attempt, fence obsoleto, caída en cada frontera, reinicio, ventanas nuevas, carreras y negativo entre proyectos. Cierra cuando no hay sobre-reserva ni reintento ciego. `P≤500, V≤450`. |
| **A05 — fuentes y configuración V38** | `ORC-28`. El registro es la única definición; `runtime.provider=codex` conserva el proveedor y una clave ortogonal `runtime.isolation=process\|microvm` deja `process` por defecto. Una fuente específica del proveedor traduce datos, pero `application` decide política/frescura. | `config/registry.json`, salidas generadas por el generador vigente, `internal/adapters/agent/codex/capacity_observer.go`, `internal/adapters/agent/staticcapacity/`, `internal/bootstrap/agent_capacity.go` y pruebas. Depende A02; serializa todo cambio posterior del registro. | Caracteriza, no copia, `codex_usage_accounting.json`, `rate_limits`, `exhausted/try_again_at`, restantes y reset. No usa parser awk, scraping de logs/tmux ni wrapper del motor legacy. Una métrica solo produce slots con mapeo explícito; incompleta queda unknown. No reutiliza claves TestAttestor ni expande `runtime.provider`. Evita defaults/env ad hoc. | JSON estructurado: límite/restante `0` presente, reset que abandona exhausted, payload truncado, reloj sesgado, fuente desaparecida y nueva ventana. Paridad TOML/schema/docs/UI y redacción de secretos. Cierra con dos adaptadores contractuales, aislamiento tipado y sin `os.Getenv`. `P≤300, V≤300`. |
| **A06 — contrato neutral de preservación** | `ORC-28`. Antes de desmontar, el entorno entra en `preserved_pending_review` con sello e inventario; no es un nuevo estado del `Goal`. Autoridad de aceptación: `application`. | `internal/ports/agent_environment.go`, `internal/application/agent_environment.go`, migración `NNN_agent_environment_receipts.sql`, implementación SQLite específica y pruebas. Depende A03; migración y `state.go` seriales con A04. | El comprobante liga execution/external identity/fence, base del workspace, change-set/bundle CAS, inventario, configuración/rootfs y tiempos. Reusa CAS, refs, efectos y revisión. No expone operación delete/GC en V38. | Normal, comprobante duplicado, digest alterado, ref de otro proyecto, sello ausente, desmontaje prematuro, fence obsoleto y reinicio. Cierra cuando terminalizar un proveedor que exige preservación falla cerrado sin comprobante válido y el material sigue direccionable. `P≤350, V≤350`. |
| **A07 — despachador único con `launch_agent` concurrente** | `ORC-28`. Un despachador global; procesamiento serial salvo `launch_agent` ya reclamado. Observación y parada progresan aunque el lanzamiento esté saturado. Autoridad de reclamo y procesamiento: `application`; coordinación única: `bootstrap`, sin trabajadores residentes. | Base integrada en `7ae98026`; despachador y pruebas integrados en `e4a6a98e`. Depende de A04 para cerrar capacidad durable; los archivos compartidos se serializan. | Reclamo global mientras haya capacidad; al saturarse, reclamo global con `ExcludeLaunch`; cada lanzamiento admitido usa una gorutina corta ligada solo a su reclamo y toda acción restante se procesa en serie. Reusa reclamo, lease y fencing. Evita selector «solo launch», pool o gorutinas ociosas, cola por proveedor y segundo bucle. | Ya ratchea `1/5/10/16/20`, saturación, observación/parada, reclamo cercado íntegro, cierre de gorutinas, carrera y ausencia de giro activo. A07 solo cierra cuando A04 sustituya el límite estático por capacidad durable y A08 emita su comprobante. `P≤250, V≤400`. |
| **A08 — compuerta A neutral** | `ORC-28`. Acredita semántica sin KVM: cohortes, capacidad parcial, órdenes/mailbox, observación, parada, preservación y recuperación. Autoridad: contrato de aceptación V38, todavía sin promoción. | `acceptance/v38_agent_runtime_elastic_core_test.go`, datos V38 y sustituto contractual bajo la superficie de prueba existente. Depende A01–A07. | Reusa sustituto Agent, execution sessions/mailbox, CAS, reloj y almacén reales. No introduce sustituto productivo ni interpreta ACK/texto como efecto. | E2E neutral `1/16/70/500`; capacidad menor que conjunto listo; refresco; orden/ACK exactos posteriores al lanzamiento; parada; reinicio en fronteras de intento/efecto/comprobante; `-race`; suite sin `/dev/kvm`, binario Firecracker ni artefacto de huésped. Cierra solo con comprobante de compuerta A ligado a digests. `P≤100, V≤750`. |
| **B01 — selección explícita y composición Orquesta** | `ORC-28`. La dimensión tipada de aislamiento es `process\|microvm`; `microvm` solo se activa por valor explícito válido y un error de comprobación previa no cae a `process`, Codex ni Bubblewrap. Autoridad: bootstrap sobre A05. | `internal/bootstrap/agent_provider.go`, `cmd/orquesta/` en preparación de composición y pruebas. Depende A05/A08/B03/B04; se integra en serie antes de B10 por compartir composición. | Proyecta el registro de Orquesta a `agentmicrovm/v1.Config` y `Dependencies`, pero no conecta todavía implementaciones: B10 realiza el único wiring final. El motor público nunca recibe el registro global y `test_attestor.provider=microvm` permanece independiente. | Por defecto `process` no inicia Firecracker; `microvm` sin artefactos/KVM falla antes de admitir lanzamiento; configuración inválida/redactada. Cierra con selección/proyección tipadas listas para B10, sin ruta alternativa ni tipos Orquesta filtrados al motor. `P≤150, V≤200`. |
| **B02 — implementación de leases CID** | `ORC-28`. CID sirve para encaminamiento, nunca identidad; lease durable tiene revisión, fence, expiración y recuperación antes del enlace físico. Autoridad: misma SQLite mediante el puerto público de persistencia. | Solo migración `NNN_agent_vsock_cid.sql`, `internal/adapters/agent/firecracker/vsocklease/` y pruebas; no toca bootstrap ni composición. Depende A03/A08/B03; migración serial; paralela con B05/B06/B08/B09. | Hace que el allocator SQL existente implemente el puerto cohesivo `RuntimeJournal`/lease de `agentmicrovm/v1`; retira `SchemaStatements` duplicado o lo deriva de la migración solo para pruebas. B10 hará el wiring; el motor público no importa SQLite. | Asignación concurrente, wrap/agotamiento, liberación obsoleta, caída/reapertura, lease expirado, CID ocupado, recuperación exacta y carreras. Cierra con una definición de schema y un adaptador contractual todavía no compuesto. `P≤200, V≤250`. |
| **B03 — API pública versionada del motor** | `ORC-28`. El gestor físico es reutilizable por futuras aplicaciones e importable sin Orquesta. Su autoridad termina en recursos microVM y comprobantes físicos; nunca conoce `Goal`, `WorkItem`, `Execution`, ciclo de vida, permisos o presupuestos. | Nuevo árbol público `agentmicrovm/v1/{contract,config,ports,errors}.go`, suite contractual `agentmicrovm/v1/contracttest/` y pruebas. Depende A08; primera tarea B. | Expone una interfaz cohesiva `Engine` para lanzamiento, observación, envío, parada, preservación y cierre; `Config` tipada, refs opacas, `Handle` y códigos máquina estables. `Dependencies` agrupa una frontera por sistema real: reloj, artefactos, confianza/credenciales+atestación, journal durable/CID y servicios vsock. Sin interfaz por función, variables globales, i18n ni imports `orquesta/internal`. | Guardas de superficie/API v1, nulos, cancelación, idempotencia, códigos de error y suite contractual reutilizable por cualquier motor. Cierra cuando `go list -deps` del árbol público no contiene Orquesta interna y la API no exporta conceptos de producto. `P≤250, V≤300`. |
| **B04 — consumidor externo y prueba de extracción** | `ORC-28`. La importabilidad se demuestra, no se infiere por ubicación. La extracción física a otro módulo/repositorio se difiere hasta estabilizar v1. | `acceptance/agent_microvm_v1_external_consumer_test.go`, consumidor mínimo en `acceptance/fixtures/agent_microvm_external_consumer/` y guarda de API; no se registra otro `go.mod`. Depende B07/B09. | La prueba crea un módulo temporal externo, usa `replace` hacia este árbol, importa `orquesta/agentmicrovm/v1` y `.../firecracker`, instancia el motor con sustitutos contractuales y compila/ejecuta sin `internal`. Evita módulo, repositorio, microservicio o publicación prematuros. | Negativo que detecta import interno/transitivo, variables globales Orquesta y ruptura incompatible de v1; prueba mínima externa y presupuesto de dependencias. Cierra con consumidor verde y un informe de extracción, sin haber extraído ni publicado nada. `P≤50, V≤350`. |
| **B05 — rootfs y protocolo del huésped reproducibles** | `ORC-28`. El huésped inmutable contiene init, Git, Codex y cliente vsock; una sesión versionada transporta bootstrap, órdenes posteriores, estado y resultados sin decidir el ciclo de vida. | `agentmicrovm/v1/firecracker/internal/guestproto/`, cliente/init bajo `agentmicrovm/v1/firecracker/guestcmd/`, `tools/agentmicrovm-rootfs/`, manifest/SBOM/licencias y pruebas. Depende B03; paralela con B02/B06/B08/B09. | Protocolo privado al motor, distinto del one-shot TestAttestor; IDs/refs opacos e idempotencia, datos grandes por ref de artefacto. Reusa primitivas neutrales de hash/path solo con dos consumidores. Evita copiar rootfs TestAttestor, protocolo stdout, descargas implícitas y secretos embebidos. | Build reproducible, digest, owner/mode, sin red/device inesperado; versión incompatible, frame parcial/replay/desorden, reconexión tras reinicio, orden duplicada, contrapresión y límites. Cierra con kernel/rootfs/init sellados y orden posterior aplicada una vez. `P≤650, V≤550`. |
| **B06 — artefactos y bundle por CAS/vsock** | `ORC-28`. El motor público mueve medios por ref/digest y no interpreta repositorios; Orquesta conserva autoridad sobre workspace, base, snapshot, write-set e integración. | `agentmicrovm/v1/firecracker/artifacts.go`, `internal/adapters/agent/firecracker/bundleio/`, extensión mínima del productor Workspace/Git y pruebas. Depende A08/B03; paralela con B02/B05/B08/B09. | El puerto público de artefactos entrega bytes verificados y media type; el adaptador interno construye `AgentMicroVMBundleDescriptor`, valida traversal/links/devices/expansión y devuelve change-set CAS. El huésped nunca conoce rutas del anfitrión. Evita almacén de workspace o Git dentro del motor. | Bundle válido/corrupto/truncado, base errónea, traversal, links/device, expansión abusiva, ref entre proyectos, repetición idempotente y change-set fuera de write-set. Cierra con anfitrión→CAS→huésped→CAS→integración, sin tipos Orquesta en la API. `P≤400, V≤350`. |
| **B07 — motor físico Firecracker público** | `ORC-28`. `firecracker.New(Config, Dependencies)` crea una microVM por `RunRef`, observa y reconcilia recursos físicos; no cierra ni reabre trabajo de la aplicación llamante. | `agentmicrovm/v1/firecracker/{engine,launcher,observer,recovery}.go`, primitivas neutrales justificadas y suite `contracttest`. Depende solo de B03/B05/B06; no depende de B01 ni importa una implementación B02. | Implementa planes de lanzamiento/red y auditoría sin importar `internal`. Consume `RuntimeJournal` como puerto y persiste un handle durable con run ref opaca, PID/start-time, cgroup, CID lease/fence y digests. Sin NIC/TAP/bridge/NAT ni VM compartida. | Dos lanzamientos→dos VMs, fallo en cada paso, PID reutilizado, fugas, path/asset inválido, KVM ausente, OOM, observación presente/ausente/ambigua, reinicio y carreras; manifiesto físico de tiempo/RAM/disco/tokens respetado. Cierra con identidad exacta y cero recursos propios huérfanos. `P≤500, V≤450`. |
| **B08 — broker Orquesta sobre servicio público** | `ORC-28`. El broker Orquesta queda interno: traduce el servicio vsock neutral a execution session, CAS, MCP y mailbox de la ejecución exacta. No entra en el motor reutilizable. | `internal/adapters/agent/firecracker/broker/`, adaptador al `ServiceMux` público, `networkauth` existente y pruebas. Depende A08/B03; paralela con B02/B05/B06/B09. | Reusa CredentialStore, prueba de uso único, atestación transaccional y comandos existentes. El motor solo invoca el puerto de confianza/servicio con `RunRef`; Orquesta resuelve `ExecutionRef`, permisos y alcance. Evita nueva DB, registro de comandos, cola o socket entrante. | Replay/concurrencia de credencial, atestación errónea, alcance cruzado, reconexión, commit/rollback, reinicio que invalida challenge, orden posterior y payload CAS. Cierra cuando solo la VM exacta accede a `orquesta_broker`. `P≤550, V≤450`. |
| **B09 — egress controlado reutilizable** | `ORC-28`. El componente público aplica una concesión inmutable de red y emite comprobantes físicos; Orquesta decide autorización, presupuesto y ledger antes/después. Única salida: `controlled_egress_proxy` por vsock. | `agentmicrovm/v1/firecracker/egress.go`, `internal/adapters/agent/firecracker/egresspolicy.go` y pruebas. Depende B03; paralela con B02/B05/B06/B08. | API neutral `EgressGrant` con destinos, puertos, tiempo/bytes y ref opaca; el adaptador interno la deriva de permisos/budgets. Revalida DNS/redirects y bloquea loopback, RFC1918, ULA, link-local/metadata. Evita Internet directo, TAP/bridge/NAT y política Orquesta en el paquete público. | Permitir/negar, DNS rebinding, redirect privado, IPv4/IPv6, exceso, timeout, comprobante perdido, presupuesto agotado, VM vecina e inbound negativo. Cierra sin NIC del huésped y con comprobantes idempotentes/machine-readable. `P≤600, V≤550`. |
| **B10 — adaptador Agent, wiring único y continuidad** | `ORC-28`. Orquesta traduce `AgentLauncher/Observer` a `Engine` v1 y conserva toda autoridad de ciclo de vida, capacidad, permisos, efectos y refs de producto. | `internal/adapters/agent/firecracker/{agent,observer,recovery,dependencies}.go`, `internal/bootstrap/agent_provider.go`, `cmd/orquesta/` y pruebas. Depende B01/B02/B04/B06–B09; única tarea que conecta B02 y el motor a Orquesta, serial en composición. | Mapea `ExecutionRef`↔`RunRef` opaca; inyecta CAS, CredentialStore, SQLite/leases CID, reloj y servicios sin exportar Goal/WorkItem. Reusa recuperación de efectos en cinco fronteras y no crea otro compositor, writer o lifecycle. | Idempotencia, lanzamiento duplicado, fence obsoleto, proceso presente/ausente/ambiguo, CID reconciliado, reinicio en cada frontera, comprobante perdido, observación concurrente y `-race`. Solo `definitely_not_applied` relanza. Cierra con wiring real y continuidad determinista. `P≤650, V≤600`. |
| **B11 — parada y cierre exactos** | `ORC-28`. El motor detiene de forma cooperativa y luego forzada el handle físico exacto; Orquesta deja de admitir claims, detiene el tick, espera lanzamientos reclamados y solo después llama `agent.Shutdown`. | `agentmicrovm/v1/firecracker/{controller,shutdown}.go`, `internal/adapters/agent/firecracker/{controller,shutdown}.go`, corrección mínima en `internal/bootstrap/` y pruebas. Depende B10; serial con composición. | `Engine.Stop/Shutdown` usa RunRef, PID/start-time, cgroup, CID/fence; el adaptador implementa `AgentController/AgentShutdown` y reconciliación Orquesta. Corrige el riesgo actual de cerrar Agent antes de drenar el planificador. Evita `pkill`, kill por nombre/cohorte y uso del motor cerrado. | Cooperativa, timeout→forzada, PID reutilizado, comprobante perdido, repetición, fence obsoleto, peer intacto, reinicio durante parada, cierre con lanzamientos en vuelo y limpieza exacta. Cierra con comprobante y ningún uso posterior al cierre. `P≤300, V≤350`. |
| **B12 — sello, inventario y compuerta B** | `ORC-28`. Tras parada/terminal, huésped y broker devuelven change-set/manifiesto; Orquesta verifica, persiste `preserved_pending_review` y solo entonces desmonta/libera. No borra. | Método público de preservación en `agentmicrovm/v1/firecracker/`, `internal/adapters/agent/firecracker/preserve.go`, integración A06/B06 y `acceptance/v38_agent_firecracker_single_vm_test.go`. Depende B06/B10/B11. | El motor emite inventario físico neutral; el adaptador añade workspace/config/tooling y persiste comprobantes CAS. Reusa `agent_firecracker_single_vm`; no comparte evidencia/rootfs TestAttestor ni expone delete. | Una microVM real, orden posterior, observación, parada, sello, inventario, desmontaje; negativos de digest/write-set/ref/atestación y reinicio antes/después del sello. Cierra con comprobante de compuerta B; no promueve V38 por sí sola. `P≤400, V≤500`. |
| **C01 — arnés y comprobación previa de olas** | `ORC-28`. El anfitrión declara capacidad real antes de la ola y nunca rebaja silenciosamente el peldaño. Todos los recursos pertenecen a un run aislado. | `acceptance/v38_agent_firecracker_wave_harness_test.go`, script/runbook acotado, manifiesto de presupuesto y datos de recursos. Depende B12. | Reusa perfil de anfitrión solo como lección; mide CPU/RAM/disco/KVM/CID/FD/cgroup y digests. Sella por candidato los cuatro límites exactos `wall_time_limit_ms`, `ram_peak_limit_bytes`, `disk_peak_limit_bytes`, `agent_token_limit`. | Positivos/ausentes/excedidos, reserva de disco/tokens, RAM disponible, nombres/temporales exactos, aborto y limpieza inventariada. Si no caben 20 o falta un límite, la compuerta queda bloqueada. `P≤300, V≤300`. |
| **C02 — olas físicas del mismo candidato** | `ORC-28`. Ejecuta `1/5/10/16/20` con una microVM por agente y el mismo digest candidato; el conjunto listo lógico puede ser mayor. Es compuerta física independiente de B04 y TestAttestor. | Arnés C01 y resultados estructurados bajo `product/evidence/candidates/`; usa sin cambios el manifiesto de tiempo/RAM/disco/tokens y no edita roadmap. Depende C01. | Reusa despachador, capacidad, motor v1, adaptador Orquesta, broker/proxy, observación, mailbox, parada y preservación. Evita reconstrucción entre pasos, selección manual y métricas desde logs. | Por ola registra `present`, inicio, fin, duración, `RunRef`, peldaño y digest para siete latencias: **solicitud de capacidad, aprovisionamiento, lanzamiento, disponibilidad, parada cooperativa, parada forzada y sello/inventario**. Cada peldaño ejecuta subcasos separados cooperativo y forzado controlado; cero no significa ausencia. Registra también RAM, disco y tokens solicitados/observados. Cierra solo si pasan cinco peldaños dentro de presupuesto. `P≤150, V≤400`. |
| **C03 — matriz adversarial, carreras y recuperación** | `ORC-28`. Seguridad, causalidad e idempotencia se mantienen bajo fallos reales; estados ambiguos nunca disparan reintento ciego ni borrado. | Pruebas de aceptación adversarial V38 y datos de inyección de fallos; no producto salvo corrección causal separada. Depende C02. | Reusa casos A/B: fuente stale/unknown, nueva ventana, caída en intento/efecto/comprobante, replay de credencial, CID ABA, DNS rebinding, VM vecina, comprobante de parada perdido y sello incompleto. Evita una suite feliz como evidencia total. | `go test -race`, reinicio/reapertura, parada controlada de procesos propios, negativos de auth/proyecto/traversal/red y escaneo final. Toda corrección repite C02 con nuevo digest candidato. Cierra sin P0/P1 abierto. `P≤50, V≤650`. |
| **C04 — sello y revisión independiente** | `ORC-28`. Evidencia pertenece al mismo árbol fuente, binario, rootfs y configuración efectiva; los comprobantes quedan fuera del sujeto y lo referencian. Autoridad: pipeline de release/TestAttestor más revisión primaria y adversarial. | Manifiesto/índice inmutable del candidato, comprobantes de aceptación y notas de revisión; ningún cambio funcional. Depende C03. | Reusa TestAttestor sin darle red/vsock y el modelo de evidencia actual. Evita autodigest imposible, comprobante manual y mezcla de candidatos. | Compuertas globales: `git diff --check`, unitarias/contractuales/integración, `go test`, vet, carreras relevantes, arquitectura/config/auth, backup/restore/cierre y cero procesos propios. Cierra con dos revisiones y digests coincidentes A+B+C. `P≤50, V≤300`. |
| **C05 — promoción canónica atómica** | `ORC-28`. Solo tras C04, roadmap y evidencia de la misma revisión promueven la vertical completa; ningún read-model derivado puede acreditar por sí solo. | Obligatorios en un changeset: `product/roadmap.json`, command/fixture/test reales, comprobante/índice de `product/evidence/`, decisión `agent_microvm_network`, `product/knowledge/tooling_adoption_v1.json`, `product_roadmap_test.go` y guardas `acceptance/v38_agent_runtime_elastic_plan_*`. `product/capabilities.json` solo se regenera si la composición/release canónica lo exige y referencia la misma evidencia. Depende C04; serial final. | En `product/roadmap.json`, `AC-V38-AGENT-RUNTIME-ELASTIC` pasa `planned→executable` con los campos reales `command`, `test_ref`, `fixture` y `receipt`; `ORC-28` pasa `declared→accredited` con `evidence_refs`; `agent_microvm_network` pasa `planned_not_applied→applied`. `tooling_adoption_v1.json` cambia Firecracker de `candidate` al estado acreditado, soportado por la misma evidencia y sin volverse autoridad. | Sustituye las guardas vigentes que exigen contrato V38 `planned` y sin recibo, ausencia de evidencia V38, `ORC-28` `declared`, decisión `planned_not_applied` y Firecracker `candidate`; las nuevas guardas comprueban refs/digests coincidentes. Si falta un campo real, comando ejecutable o comprobante A+B+C, nada se promueve. Cierra V38 en una revisión indivisible. `P≤100, V≤250`. |

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
  repositorio de artefactos, confianza/credenciales y atestación, journal
  durable con leases CID, y multiplexor de servicios vsock. Cada puerto tendrá
  al menos un sustituto contractual y un consumidor real; no habrá interfaz
  por operación.
- Requests, handles, observaciones y comprobantes usan refs opacas,
  idempotency keys, fencing y códigos máquina estables. El paquete público no
  contiene texto humano, catálogos i18n, rutas Orquesta ni globals.

El motor no importa `internal`, SQLite ni paquetes de dominio. El adaptador
que permanece en `internal/adapters/agent/firecracker` traduce
`Goal`/`WorkItem`/`Execution`, permisos, presupuestos, ledger, CredentialStore,
CAS y estado SQLite a la API neutral. Así una aplicación futura puede aportar
otros adaptadores sin arrastrar el ciclo de vida de Orquesta.

B04 verifica esa separación desde un módulo temporal consumidor. La extracción
física solo se estudiará después de estabilizar v1 y medir sus dependencias; no
forma parte de la acreditación V38 y no autoriza publicar nada.

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

Una observación no reserva por sí misma. `application`, con su reloj inyectado,
acepta o rechaza su frescura y reserva de forma atómica al reclamar la acción
global. Los números necesitan presencia explícita: `remaining=0` es agotado;
`remaining` ausente es desconocido, nunca ilimitado. Una fuente
`unknown`, `stale` o `unavailable` impide reservas nuevas, pero no bloquea
`observe_agent`, `stop_agent`, órdenes ya admitidas ni preservación.

`WindowRef` evita perpetuar el estado agotado al cruzar un reset. Una
observación válida de una ventana nueva puede reabrir capacidad; no lo hace un
timer local sin observación. El legacy que ignoraba ceros con comparaciones
`>0`, persistía `exhausted` tras el reset o extraía cuota de awk/logs/tmux queda
como lección negativa. Solo datos estructurados y artefactos CAS pueden
caracterizar la traducción específica del proveedor.

La consulta legacy se limita a caracterizar
`codex_usage_accounting_v0.go`, `codex_usage_accounting_json_v0.go`,
`codex_usage_accounting_wrapper_v0.go`,
`agent_usage_runtime_source_v0.go` y `capacity_decision_v0.go` en la copia de
consulta. Se preservan semánticas útiles —fuente/calidad/observación/reset,
agotamiento, reintento y restantes de segundos/mensajes/tokens/créditos—, no
sus parsers, procesos, tablas, planificador ni código.

La reserva se consume con el comprobante que acredita lanzamiento aceptado. Se libera
solo con terminal exacto o `definitely_not_applied`. `unknown_applied`
conserva la reserva y exige reconciliar por external identity. Un reinicio no
crea otro intento ni otra reserva para la misma idempotency key.

### Despachador y paralelismo

Hay un solo despachador y un solo protocolo de claim. La preferencia serial por
no-launch sirve para que observe/stop/mailbox avancen; no es una cola nueva.
Solo se crea una goroutine corta después de reclamar realmente un launch y
solo mientras ejecuta ese claim. El límite físico regula claims launch
concurrentes; jamás trunca `ReadyWorkItems`, reprograma el DAG o limita el
número de Goals visibles.

### Identidad, red y credenciales

La identidad de operación contiene refs de proyecto, goal, work item,
execution, generación, action/attempt, motor externo y fence. PID y CID son
datos de routing sujetos a reutilización, no identidad suficiente.

El huésped no recibe NIC. Los únicos servicios vsock permitidos son
`orquesta_broker` y `controlled_egress_proxy`. La autorización exige proof
single-use desde `CredentialStore`, scope exacto y attestation; el challenge
en memoria se invalida tras reinicio. No se entregan secretos en rootfs,
configuración efectiva, Goal, prompt, log, artefacto o comprobante.

El proxy vuelve a resolver DNS y revalida cada redirect. Debe negar Internet
directo, inbound, east-west, loopback, redes privadas/link-local/metadata y
todo destino no permitido. Un ACK de broker solo confirma admisión: el efecto
se acredita mediante el comprobante del registro de efectos.

### Workspace, stop y preservación

El anfitrión produce un bundle/snapshot ligado al workspace y base exactos, lo
deposita en CAS y entrega descriptor más bytes verificados por vsock. El
huésped no monta `.git` ni filesystem del anfitrión. El resultado vuelve como
change-set/bundle CAS y pasa por la validación e integración en anfitrión ya
acreditadas.

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
- **Confundir TestAttestor con agente:** las guardas de imports/composición deben
  impedir compartir protocolo, rootfs, comprobantes o evidencia. Solo
  primitivas neutrales pequeñas, extraídas con dos consumidores y pruebas, son
  reutilizables.
- **Cuota falsa o stale:** fail-closed para launches nuevos y continuidad de
  control para ejecuciones existentes. Ningún «desconocido = ilimitado».
- **Recursos huérfanos:** todo run inventaría PID, socket, CID, cgroup,
  temporal y CAS propios. Limpia solo targets verificados; ante ambigüedad
  preserva y reporta.
- **Anfitrión insuficiente:** no se reduce C de 20 a un número disponible ni se
  usa el resultado histórico `1+16`. Se documenta bloqueo y se repite en un
  anfitrión apto con el mismo candidato.
- **Crecimiento arquitectónico:** máximo un despachador, una DB, un registro de efectos, un
  registro de comandos y un registro de configuración. Broker/proxy/huésped y
  motor son
  fronteras físicas o de seguridad del adaptador, no microservicios internos.
  El árbol público se limita a contrato v1, implementación Firecracker y suite
  contractual; no importa `internal`. No se aceptan paquetes de una constante,
  almacenes, planificadores, colas o bucles adicionales, ni otro módulo antes
  de estabilizar la API.
- **Presupuesto:** la suma base exacta de las 25 tareas V38 es `P=7.200` LOC no
  generadas y `V=9.800` LOC de verificación/arneses; generado se informa aparte.
  Existe una contingencia separada máxima de `P=300`, no asignada: usarla exige
  ADR y autorización explícita previas que nombren frontera inevitable y
  retirada/compensación concreta. D01 queda fuera y recibirá presupuesto propio
  al activarse. Objetivo adicional: cero nuevos escritores de ciclo de vida,
  cero almacenes, cero bucles residentes y cero comandos públicos salvo contrato
  de aceptación.

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
