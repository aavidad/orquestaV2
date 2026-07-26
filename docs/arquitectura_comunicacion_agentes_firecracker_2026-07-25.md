# Arquitectura de comunicación de agentes Firecracker

Fecha: 2026-07-25. Actualización: 2026-07-26. Estado: implementación del primer
corte autorizada; no cableada ni acreditada físicamente. Autoridad:
`AGENTS.md`, `product/roadmap.json`,
`decision_atestacion_bubblewrap_microvm_2026-07-25.md`,
BUG-ORQ-20260725-467 y BUG-ORQ-20260726-507.

## Alcance y precondición

Este documento no amplía el `TestAttestor` V23: su microVM sigue sin red ni
vsock. El operador adelantó el 2026-07-26 el contrato y adaptador de plan de red
para agentes. No autorizó ejecutar root, cambiar la red del host, cablear
agentes reales ni declarar acreditación sin el E2E físico separado.

Será una capability independiente, con contratos, presupuesto y acreditación
de composición propios. No presupone que workspaces, rootfs, caches, tmpfs/RAM
ni checkpoints frecuentes sean adecuados; cada uno requiere decisión y prueba
de coste, aislamiento, persistencia y recovery.

## Invariantes de aislamiento

Cada microVM de agente es un dominio aislado. Quedan prohibidos:

- red east-west, peer networking y direccionamiento IP mutuo entre agentes;
- SSH, shells remotas, montajes compartidos y acceso al filesystem, procesos,
  sockets, credenciales o metadata de otra VM, del host o de otra ejecución;
- rutas host escribibles, acceso a la LAN del host, loopback del host, RFC1918,
  ULA, link-local, endpoints de metadata y destinos/puertos no autorizados;
- tráfico inbound hacia la VM.

Un agente no descubre ni contacta a otro agente por IP, hostname, CID, puerto o
ruta. El acceso entre agentes solo puede aparecer en el futuro como una
capacidad explícita, autenticada, autorizada y auditada por Orquesta; nunca
como conectividad de red directa.

## Plano de colaboración

La única colaboración permitida pasa por un gateway/broker Orquesta autenticado.
El protocolo transporta refs opacas y no expone topología, paths ni identidades
de red:

```text
agent microVM -> gateway/broker Orquesta -> mailbox/eventos/CAS -> agent microVM
```

Las operaciones se ligan explícitamente a `goal_ref`, `task_ref` y `agent_ref`,
y, cuando exista delegación, a los refs parent/child. El broker es la autoridad
de acceso para:

- enviar y consumir mailbox/eventos causales;
- publicar, leer y entregar artefactos CAS únicamente por refs opacas;
- declarar receipts, progreso, bloqueos y resultados asociados al trabajo;
- conceder capacidades futuras entre agentes sin revelar direcciones mutuas.

La autorización aplica ACL por Goal y parentesco de tareas/agentes: una VM solo
puede observar, publicar o consumir aquello que corresponda a su Goal, tarea,
rol y relación causal autorizados. El broker valida generation/fencing cuando
proceda, idempotency key y precondiciones causales antes de persistir. Conserva
evento, decisión de autorización, intent, delivery/receipt y digests necesarios
para auditoría y replay sin duplicar efectos.

Cuotas por Goal, tarea, agente y operación limitan CPU/red/almacenamiento,
mensajes, bytes y concurrencia. El broker aplica backpressure explícito y
retry/idempotencia acotados; nunca se elude la presión mediante un canal directo
entre VMs.

## Transporte, egress y frontera de red

La microVM de agente no tiene NIC, TAP, bridge, rutas IP ni NAT. El único
transporte es vsock y la allowlist contiene:

1. el gateway/broker Orquesta autenticado;
2. un proxy o buscador controlado que aplique política, identidad, cuota y
   receipt.

No hay acceso IP directo a Internet, LAN, host ni metadata. El proxy
HTTP(S)/buscador vive en el host y valida destino, resolución, redirects,
puertos, identidad, scope y cuota antes de cualquier egress. Si Codex necesita
HTTP convencional dentro del guest, un adaptador local enlaza loopback guest
con el servicio vsock autorizado; no crea una interfaz IP fuera del guest.
NAT queda prohibido para este perfil: no existe fallback a TAP, bridge,
`iptables`/`nftables` o red compartida.

El host CID de vsock y los puertos exactos de broker/proxy se fijan en una
política sellada. La identidad de lanzamiento liga proyecto, Goal, WorkItem,
ejecución, generación, intento y agente; el broker/proxy no confían en el CID
como autenticación. Un digest de esas refs públicas solo liga causalidad: no
autentica la VM. Antes de exponer mailbox, CAS o proxy, el broker debe consumir
un challenge de un solo uso, verificar un proof contra la versión exacta de una
credencial leída mediante `CredentialStore` y verificar la atestación exacta
del lanzamiento. El authorizer reclama el challenge de forma atómica antes de
consultar el ledger de atestaciones o `CredentialStore`; missing, expirado
(`now >= expires_at`), consumido y la carrera perdedora terminan ahí. Una vez
reclamado, cualquier fallo posterior lo invalida y el agente debe pedir uno
nuevo. Solo después de atestación, credencial y HMAC válidos, el authorizer
inicia dentro de esa misma llamada una transacción de apertura. `Begin` tiene
cero efectos, `Open` solo prepara recursos internos y `Commit` es el único punto
atómico que puede hacer alcanzable la sesión. Cualquier error o cancelación
desde `Begin` hasta antes de un `Commit` correcto ejecuta `Rollback` con el
timeout de cleanup explícito de la composición. Si el rollback falla o agota su
plazo, se devuelve `cleanup_failed` sin receipt; puede quedar un residuo interno
para reparación, pero nunca una sesión alcanzable. El receipt posterior es
evidencia estructurada, nunca una credencial ni una autoridad reutilizable.
Solo refs, versión y digests entran en política/receipt; el secreto y el proof
permanecen en el callback de credenciales y en memoria transitoria. Política,
plan efectivo, autorización y receipt llevan digests independientes.

El CID guest tampoco es una identidad ni se elige libremente en el launcher.
Un puerto neutral reserva por lease un CID `>= 3`, un backend content-addressed
y un fencing token monotónico. La reserva liga pool, scope causal completo,
ejecución, agente, owner de lanzamiento e idempotency key. El ledger conserva
las generaciones después de liberar o expirar una reserva: si el mismo CID se
reutiliza, obtiene otro fencing token y otro `lease_ref`, por lo que una
liberación antigua no puede afectar al nuevo dueño (ABA). Renovar exige revisión
esperada; recuperar exige owner, scope y fencing exactos; liberar exige además
la revisión vigente. Los reintentos semánticamente distintos bajo la misma
idempotency key se rechazan.

El receipt de lease sigue sin ser autoridad por posesión. La composición que
pueda hacer alcanzable un backend debe consultar la reserva activa
inmediatamente antes del efecto y comparar `lease_ref`, fencing y revisión. El
plan renderizado conserva esos valores para evidenciar el vínculo, no para
sustituir la consulta. Expiración usa límite inclusivo (`now >= expires_at`) y
jamás revive una reserva; otra adquisición con la misma clave después de
expirar queda denegada como replay.

Este corte es una composición/adaptador alternativo de `AGT-01`/`AGT-03`, con
la evidencia de aislamiento exigida por `EVD-13` y el broker futuro relacionado
con `ORC-15`. No añade una capacidad 258 ni cambia el estado acreditado de esos
IDs: el plan declara `planned_not_applied`.

Gateway, proxy, política o su verificación indisponibles, ambiguos o inválidos
producen denegación fail-closed: no se crea una conectividad degradada, no se
abre fallback directo y el agente recibe una causa estructurada recuperable por
replan/retry autorizado.

## Estado del primer corte

El corte 2026-07-26 implementa:

- contrato neutral de política y scope exacto con digests causales;
- binding a una versión de credencial con owner/scope/purpose exactos y
  atestación del lanzamiento;
- authorizer host mínimo que reclama primero el challenge, exige después la
  atestación exacta, calcula HMAC-SHA256 solo dentro del callback de
  `CredentialStore`, compara con `hmac.Equal` y crea como máximo una transacción
  de apertura; `Open` prepara, `Commit` publica atómicamente y `Rollback`
  acotado limpia todo fallo o cancelación; el receipt resultante es solo
  evidencia sin secreto;
- almacén de challenges en memoria, acotado y ligado a política, lanzamiento,
  ejecución y agente; el consumo es atómico y una segunda presentación
  idéntica queda denegada sin consultar los ledgers. Proof, atestación,
  credencial o apertura fallidos gastan el challenge. Un reinicio invalida los
  challenges pendientes (fail-closed);
- contrato neutral de reserva CID con ownership, lease, revisión, fencing,
  recuperación, liberación e idempotencia; adaptador SQL transaccional
  Firecracker que usa la base canónica entregada por composición, conserva
  tombstones/generaciones y soporta concurrencia entre instancias y restart;
  el schema está descrito por el adaptador pero aún no pertenece a una
  migración canónica ni está cableado;
- render determinista `planned_not_applied` con cero interfaces, TAP, bridge,
  NAT, inbound, east-west o Internet directo, allowlist vsock exacta, lease CID
  y recibo ligado también a los bytes exactos del documento renderizado.

No implementa ni simula conectividad física. La siguiente dependencia causal
es un corte separado con:

1. adaptador físico de atestación y wiring del
   `AgentMicroVMLaunchAuthorizer` ya implementado, de modo que solo una
   transacción autorizada pueda hacer `Commit` de cada sesión del broker y del
   proxy;
2. incorporar las tablas de reserva CID a la migración de la fuente de estado
   canónica, cablear el allocator y exigir `Recover` vigente dentro de la
   transacción física que materialice un backend vsock distinto por VM, sin
   NIC;
3. bridge HTTP guest loopback→vsock y proxy host-side con política SSRF,
   resolución/redirect/puerto/cuota y receipts;
4. E2E multi-microVM físico con negativos y cleanup antes de cambiar
   `planned_not_applied`.

## Aceptación y pruebas de composición

La capability no se acredita con unitarios ni con una VM que simplemente
arranque. Debe incluir, como mínimo:

- pruebas negativas desde una VM contra otra VM, IP/hostname/CID/puerto mutuo,
  SSH, montajes y filesystem ajeno: todas denegadas;
- pruebas de que host filesystem, procesos/sockets host, LAN, loopback host,
  RFC1918, ULA, link-local y metadata no son alcanzables;
- pruebas de allowlist: solo los puertos vsock exactos del gateway y del
  proxy/buscador autorizado son alcanzables; inbound, CID/puertos no
  autorizados, Internet directo y cualquier NIC/TAP/NAT se deniegan;
- pruebas de reserva CID: 16 adquisiciones concurrentes sin duplicados,
  exclusión de CID `0/1/2` y `VMADDR_CID_ANY`, restart/recovery, expiración
  inclusiva, idempotencia, revisión de renovación, fencing monotónico, intento
  de liberación ajena y reutilización ABA;
- pruebas del gateway: autenticación, ACL por Goal y parentesco, aislamiento
  entre Goals, causalidad, fencing cuando aplique, idempotencia, auditoría y
  entrega de mailbox/CAS por refs opacas;
- pruebas de cuota y backpressure que no habiliten un bypass ni dupliquen
  mensajes, entregas o artefactos;
- fallos inducidos de gateway, proxy y política que prueben fail-closed, receipt
  estructurado y ausencia de rutas directas o residuales;
- E2E con varias microVMs, cleanup verificable de VM/netns/reglas/procesos y
  evidencia de que no quedaron sockets, rutas, artefactos o capacidades fuera
  del scope autorizado.

Los receipts de estas pruebas deben ligar la configuración efectiva, digests de
imagen/política/reglas, refs causales y sujeto acreditado. Un resultado de
`TestAttestor` o una prueba aislada de red no sustituye esta acreditación.

## Estado de validación de la rama

Los tests de reconstrucción de trazabilidad y los tests focales del contrato de
red pasan localmente. El gate raíz conserva únicamente dos fallos esperados,
`TestRealCodexReceiptMatchesCurrentProductSource` y
`TestV17RealCodexReceiptMatchesCurrentProductSource`, porque sus receipts reales
preceden al nuevo sujeto. No se reescriben ni se simulan en esta rama: solo
pueden re-sellarse mediante el smoke Codex real exacto después de integrar el
corte en `main`.
