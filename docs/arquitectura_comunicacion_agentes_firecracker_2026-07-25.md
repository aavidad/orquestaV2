# Arquitectura futura: comunicación de agentes Firecracker

Fecha: 2026-07-25. Estado: decisión de arquitectura para la capability futura
de runtime general de agentes; no implementada ni acreditada. Autoridad:
`AGENTS.md`, la secuencia de
`decision_atestacion_bubblewrap_microvm_2026-07-25.md` y BUG-ORQ-20260725-467.

## Alcance y precondición

Este documento no amplía el `TestAttestor` V23: su microVM sigue sin red. La
evaluación de agentes Firecracker solo puede abrirse después de:

```text
TestAttestor Firecracker -> recuperar V23 -> Orquesta autoprogramable
```

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

## Egress y frontera de red

Por defecto la microVM no tiene egress. Cuando una capability aprobada requiera
salida, la allowlist mínima contiene solo:

1. el gateway/broker Orquesta autenticado;
2. un proxy o buscador controlado que aplique política, identidad, cuota y
   receipt.

No hay acceso directo a Internet, LAN, host ni metadata. Si se usa NAT, queda
encerrado en el netns de esa microVM y subordinado a reglas nftables exactas de
origen, destino, puerto y estado; no es un puente abierto ni una ruta compartida
para east-west. La política bloquea expresamente los rangos y endpoints vetados
de la sección anterior y todo lo que no esté permitido.

Gateway, proxy, política o su verificación indisponibles, ambiguos o inválidos
producen denegación fail-closed: no se crea una conectividad degradada, no se
abre fallback directo y el agente recibe una causa estructurada recuperable por
replan/retry autorizado.

## Aceptación y pruebas de composición

La capability no se acredita con unitarios ni con una VM que simplemente
arranque. Debe incluir, como mínimo:

- pruebas negativas desde una VM contra otra VM, IP/hostname/CID/puerto mutuo,
  SSH, montajes y filesystem ajeno: todas denegadas;
- pruebas de que host filesystem, procesos/sockets host, LAN, loopback host,
  RFC1918, ULA, link-local y metadata no son alcanzables;
- pruebas de allowlist: solo gateway y proxy/buscador autorizado alcanzables;
  inbound, Internet directo, puertos no autorizados y NAT fuera del netns
  denegados;
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
