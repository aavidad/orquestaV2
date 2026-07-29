# Decisión operativa: atestación Bubblewrap y microVM

Fecha: 2026-07-25. Estado: vigente para la frontera técnica; el orden de
activación fue corregido el 2026-07-26 y la autoridad de red de agentes se
ratificó el 2026-07-29. Autoridad: `AGENTS.md`,
`product/roadmap.json`, contrato V17 e
`inventario_bugs_orquesta_2026-06-30.md`. Esta decisión no acredita por sí sola
un nuevo adaptador ni cambia el lifecycle. El corte de alcance vigente está en
`docs/reconstruccion/corte_alcance_v23_firecracker_diferido_2026-07-26.md`.

## Decisión

Orquesta tiene un único puerto, `TestAttestor`. La aplicación entrega al puerto
un sujeto sellado, tests requeridos y política; recibe un resultado tipado con
veredicto, causa estructurada, digests de informe/salida y `policy_digest`.
No hay dos atestaciones, dos gates ni una ruta de integración especial.

```text
application -> TestAttestor -> BubblewrapTestAttestor
                         `-> MicroVMTestAttestor (Firecracker)
```

Ambos adaptadores cumplen el mismo contrato y preservan el mismo `SubjectDigest`.
Sus receipts no son intercambiables: el `policy_digest` incluye como mínimo
`provider_ref`, identidad verificada, límites, aislamiento de red, canal de E/S
y política de montaje. Un PASS Bubblewrap nunca satisface una política microVM
y viceversa. PASS solo habilita el gate existente; no integra ni cierra.

Contrato objetivo de configuración canónica de composición:

```toml
[test_attestor]
# disabled | bubblewrap | microvm
provider = "disabled"
```

La configuración implementada admite `disabled`, `bubblewrap` y `microvm`, con
`disabled` como default canónico. `bubblewrap` y `microvm` son opt-in
explícitos. No existe selección `auto` ni fallback entre proveedores. Está
prohibido cambiar de adaptador a mitad de una atestación: cambiaría el entorno
real bajo el mismo intento/receipt.

## Motivo arquitectónico y capacidad

La necesidad de aplicación es acreditar tests sobre bytes y política exactos,
no conocer namespaces o KVM. El puerto mantiene el core hexagonal y permite dos
adaptadores reales sin que Firecracker o Bubblewrap entren en dominio.

Bubblewrap es barato en equipos pequeños y sin KVM; Firecracker ofrece una
frontera de kernel más fuerte cuando KVM y recursos están disponibles. Por eso
la composición conserva ambos como adaptadores opt-in. Firecracker no
desbloquea, acredita ni forma parte de V23. Su activación real dentro de
`TestAttestor` queda como candidata V38, después de cerrar V23 y acreditar
Orquesta autoprogramable. No autoriza extender Firecracker al runtime general
de agentes. Bubblewrap queda congelado como deuda técnica hasta ese mismo
corte.

| Entorno | Selección |
|---|---|
| Equipo modesto o sin KVM | Bubblewrap, solo si su ruta ya acreditada pasa la sonda. |
| Host con KVM RW y guest acreditado | MicroVM/Firecracker. |
| Proveedor no seleccionado | `disabled`; V23 puede avanzar sin KVM, launcher ni guest. |

## Bubblewrap: evidencia retenida y congelación explícita

Los hechos vigentes son: Bubblewrap stock 0.11.1 tiene `MAX_ARGS=9000`; el
árbol observado contiene 8.497 ficheros y 534 directorios, con expansión de
43.821 argumentos; `NOFILE=32768`; AppArmor aplica `restrict userns=1`; el
perfil/memfd y Bubblewrap anidado fallan en `RTM_NEWADDR`; y el helper canónico
uid 0 queda bloqueado correctamente por `setgroups=deny` con 18 grupos
heredados. No se relajan grupos, no se añade setuid/capabilities y no se toca
`/usr/bin/bwrap`.

La mitigación documentada, **no activada ni perseguida en este frente**, es
compilar Bubblewrap 0.11.1 con `MAX_ARGS=65536` en ruta dedicada root-owned,
perfil AppArmor dedicado y sonda real. La tag anotada `v0.11.1` resuelve al
commit `124c4cdf4321f63ef17a1cb0ce8f9dd45bd7adbe` (tag object
`1e75470e57fb9d2e9028c1451d6bc856e0ac03ca`). Cuando se reanude, el instalador
deberá sellar URL fuente, SHA-256 publicado/calculado, firma/tag, diff del único
parche `9000 -> 65536`, hash del parche, hash del árbol resultante, toolchain,
binario final, perfil AppArmor y receipt de sonda. La ruta será dedicada,
`root:root 0755`, sin setuid y sin sustituir el binario del sistema.

`65536` únicamente deja margen al caso observado; no es una solución de diseño.
El argv por entrada crece linealmente y conserva un techo rígido. La solución
estructural futura es snapshot único O(1) por stream/descriptor con helper de
expansión aislado, reutilizable tanto por Bubblewrap como por microVM.

**Congelación:** no se investiga, instala, parchea ni activa Bubblewrap en este
ciclo. BUG-452/462/463 permanecen como evidencia y las deudas nuevas enlazadas
no autorizan trabajo derivado. El criterio único para reanudarlo es
**«Orquesta autoprogramable»**, acreditado en la revisión/composición que vaya a
retomar el frente. Entonces se abrirá un write-set y aceptación propios; hasta
ese momento no se interpreta esta documentación como orden de instalación.

## Firecracker/microVM diferido: frontera y requisitos

Firecracker 1.16.1 y `jailer` están presentes como binarios `root:root 0755`.
La tag `v1.16.1` resuelve al commit
`2038188f145fb81b8d098147a10e9d9f392fd22f` (tag object
`e527ccfc54495dabac96f1835db61a40afa15115`). La línea de trabajo conserva
únicamente materiales para la futura activación opt-in V38 de `TestAttestor`;
presencia de binarios no equivale a acreditar el proveedor ni a habilitar
Firecracker para agentes.

El startup preflight microVM debe fallar cerrado salvo que pruebe KVM RW para la
identidad runtime no-root; kernel e imagen guest mínimos digeridos/root-owned;
`jailer` y cgroup delegados con CPU/memoria/pids y cleanup observable. No hay
imagen guest acreditada aún. Este mínimo servirá para atestar sujetos sellados
cuando V38 se abra: no define workspaces de agentes, caches de agentes ni un
runtime general.

Firecracker 1.16.1 no soporta virtio-serial. El diseño provisional, todavía por
acreditar, no habilita red, TAP, DHCP, consola 8250 ni vsock: usa un drive
virtio raw de entrada read-only sellado y un drive raw de salida RW,
preasignado y acotado, para solicitud, stdout/stderr y reporte. Vsock queda
para una evolución futura únicamente después de ACL por CID/puerto,
autenticación y receipt. El guest recibe el snapshot sellado, no rutas host
escribibles, HOME, secretos ni estado del runtime.

La aceptación microVM exige PASS/FAIL/no-tests, timeout/cancelación, KVM/image/
jailer/cgroup/red/vsock negativos, boot y apagado real, subject/policy digest
exactos y cero VM, proceso, socket o cgroup residual. Un fallo no cae a
Bubblewrap en el mismo intento. El rollback selecciona un proveedor previamente
acreditado solo para intentos nuevos o detiene fail-closed.

Estas restricciones describen exclusivamente el adaptador
`MicroVMTestAttestor`: V38 continúa sin red y sin vsock. La autoridad distinta
para red de agentes sigue siendo `agent_microvm_network`, con estado
`planned_not_applied`; no modifica `TestAttestor` ni reutiliza su receipt.

La fixture neutral de una microVM solo caracteriza esa autoridad existente:
sin IP, TAP, bridge, NAT, inbound, east-west ni Internet directo; transporte
`vsock_only` a `orquesta_broker` y `controlled_egress_proxy`; y credencial de un
uso obtenida mediante `CredentialStore`. No añade wiring, E2E físico, receipt
ni afirmación sobre el contenido ejecutado dentro del guest.

## Orden de trabajo y deuda deliberadamente diferida

El orden vinculante corregido es:

```text
cerrar V23 por su contrato Wizard -> Orquesta autoprogramable
-> V38 candidata: activar TestAttestor Firecracker
-> V39 candidata: evaluar agentes, red acotada y RAM/tmpfs
```

La autorización `input:operator-authorization-2026-07-29` no altera ese orden
para V23, V38 o el runtime amplio. Solo permite conservar la caracterización
neutral anterior bajo `agent_microvm_network`; ningún verde actual cambia
`planned_not_applied`.

Mover todos los agentes a Firecracker no forma parte de V23 ni de este
adaptador. Tampoco se implementan ahora workspaces, rootfs o caches de agentes
en RAM/tmpfs, ni checkpoints frecuentes para dichos agentes. Son una deuda
posterior V39 que solo se evalúa después de cerrar V23, acreditar «Orquesta
autoprogramable» y resolver V38. Hasta entonces está prohibido ampliar este
write-set/objetivo
a runtime general, launchers de agentes, persistencia de checkpoints o cambios
de almacenamiento de agentes.

La futura evaluación decidirá si RAM/tmpfs y checkpoints mejoran el aislamiento,
coste y recuperación con evidencia de composición; no se presume que una
microVM de `TestAttestor` sea una plataforma válida para agentes. Si se aprueba,
será una nueva capability con contratos, presupuesto, persistencia de checkpoints
y pruebas de restart/recovery propias.

Para esa deuda futura de agentes, son requisitos obligatorios de aceptación:
filesystem del host nunca montado y LAN del host nunca enrutable desde la
microVM; red privada por microVM; sin east-west ni peer networking directo; y toda salida
exclusivamente por proxy de egress/controlador de búsquedas con política
explícita, sin inbound. La colaboración solo podrá viajar mediante gateway de
Orquesta, mailbox causal y CAS/artefactos autorizados, siempre ligados a la
identidad Goal/tarea/parent-child. Cada VM solo podrá alcanzar ese gateway y el
proxy. La política debe bloquear loopback host, RFC1918, ULA, link-local,
endpoints de metadata/SSRF y puertos no autorizados. NAT nunca será abierto:
solo puede existir subordinado al netns de la microVM y a reglas nftables
exactas. Esto no aplica al `TestAttestor`; V38 sigue sin red. Tampoco autoriza
implementar V39 antes de «Orquesta autoprogramable».

## Amenazas e invariantes

| Riesgo | Invariante |
|---|---|
| Bytes o política distintos | Subject y policy digests exactos, sin reutilización cruzada. |
| Bypass de privilegio | Runtime no-root; grupos/AppArmor no se relajan. |
| Red, secretos o host write | allowlist; sin red/vsock inicial en microVM; snapshots read-only. |
| Agotamiento/huérfanos | límites y cgroup/jailer observables; cleanup o preflight fallan cerrado. |
| Diagnóstico perdido | el receipt conserva causa estructurada antes de ejecutar; enlaza BUG-452. |

## Instalación y retirada futura de Bubblewrap

Cuando el criterio de reanudación se cumpla, la instalación deberá compilar en
temporal privado, verificar y sellar materiales, instalar atómicamente solo la
ruta dedicada y cargar el perfil dedicado; una sonda exacta comprobará límites,
AppArmor, namespaces, snapshot read-only, tmpfs, red vacía y Go pinneado. La
retirada desactiva primero el proveedor, espera intentos activos, conserva
manifests/receipts por retención y elimina únicamente los ficheros dedicados
identificados por manifest. Nunca borra paquetes ni `/usr/bin/bwrap`.

Las decisiones de configuración futuras se incorporan primero al registro
canónico y a la proyección efectiva; ningún adaptador añade variables de entorno
o defaults locales.
