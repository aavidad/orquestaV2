# Decisión operativa: atestación Bubblewrap y microVM

Fecha: 2026-07-25. Estado: vigente. Autoridad: `AGENTS.md`,
`product/roadmap.json`, contrato V17 e
`inventario_bugs_orquesta_2026-06-30.md`. Esta decisión no acredita por sí sola
un nuevo adaptador ni cambia el lifecycle.

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
# bubblewrap | microvm | auto
provider = "microvm"
```

`bubblewrap` y `microvm` son selección explícita. `auto` solo decide una vez
durante startup, después de una sonda real, persiste la elección en configuración
efectiva/digest y falla cerrado si ningún proveedor pasa. Está prohibido hacer
fallback o cambiar de adaptador a mitad de una atestación: cambiarían el entorno
real bajo el mismo intento/receipt.

Esta clave todavía no acredita wiring implementado. Antes de leerla desde
cualquier adaptador debe incorporarse al registro canónico con tipo, default,
validación, proyección efectiva y requisito de reinicio.

## Motivo arquitectónico y capacidad

La necesidad de aplicación es acreditar tests sobre bytes y política exactos,
no conocer namespaces o KVM. El puerto mantiene el core hexagonal y permite dos
adaptadores reales sin que Firecracker o Bubblewrap entren en dominio.

Bubblewrap es barato en equipos pequeños y sin KVM; Firecracker ofrece una
frontera de kernel más fuerte cuando KVM y recursos están disponibles. Por eso
la composición conserva ambos. El frente activo está estrictamente acotado:
Firecracker sustituye Bubblewrap **solo dentro de `TestAttestor`** para
desbloquear V23. No autoriza extender Firecracker al runtime general de agentes.
Bubblewrap queda congelado como deuda técnica hasta que Orquesta sea
autoprogramable.

| Entorno | Selección |
|---|---|
| Equipo modesto o sin KVM | Bubblewrap, solo si su ruta ya acreditada pasa la sonda. |
| Host con KVM RW y guest acreditado | MicroVM/Firecracker. |
| `auto` | MicroVM si pasa todo el preflight; Bubblewrap solo si también está acreditado; si no, fail-closed. |

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

## Firecracker/microVM activo: frontera y requisitos

Firecracker 1.16.1 y `jailer` están presentes como binarios `root:root 0755`.
La tag `v1.16.1` resuelve al commit
`2038188f145fb81b8d098147a10e9d9f392fd22f` (tag object
`e527ccfc54495dabac96f1835db61a40afa15115`). La línea de trabajo activa es
únicamente el adaptador microVM de `TestAttestor` de V23; presencia de binarios
no equivale a acreditar el proveedor ni a habilitar Firecracker para agentes.

El startup preflight microVM debe fallar cerrado salvo que pruebe KVM RW para la
identidad runtime no-root; kernel e imagen guest mínimos digeridos/root-owned;
`jailer` y cgroup delegados con CPU/memoria/pids y cleanup observable. No hay
imagen guest acreditada aún. Este mínimo sirve solo para atestar el sujeto V23:
no define workspaces de agentes, caches de agentes ni un runtime general.

Inicialmente no se habilita red, TAP, DHCP ni vsock. La E/S se limita a
virtio-serial con framing acotado para request, stdout/stderr y reporte. Vsock
permanece deshabilitado hasta disponer de ACL por CID/puerto, autenticación y
receipt: actualmente no tiene ACL. El guest recibe el snapshot sellado, no
rutas host escribibles, HOME, secretos ni estado del runtime.

La aceptación microVM exige PASS/FAIL/no-tests, timeout/cancelación, KVM/image/
jailer/cgroup/red/vsock negativos, boot y apagado real, subject/policy digest
exactos y cero VM, proceso, socket o cgroup residual. Un fallo no cae a
Bubblewrap en el mismo intento. El rollback selecciona un proveedor previamente
acreditado solo para intentos nuevos o detiene fail-closed.

## Orden de trabajo y deuda deliberadamente diferida

El orden vinculante es:

```text
TestAttestor Firecracker -> recuperar V23 -> Orquesta autoprogramable
-> evaluar/migrar agentes y RAM/tmpfs
```

Mover todos los agentes a Firecracker no forma parte de V23 ni de este
adaptador. Tampoco se implementan ahora workspaces, rootfs o caches de agentes
en RAM/tmpfs, ni checkpoints frecuentes para dichos agentes. Son una deuda
posterior que solo se evalúa después de recuperar V23 y acreditar «Orquesta
autoprogramable». Hasta entonces está prohibido ampliar este write-set/objetivo
a runtime general, launchers de agentes, persistencia de checkpoints o cambios
de almacenamiento de agentes.

La futura evaluación decidirá si RAM/tmpfs y checkpoints mejoran el aislamiento,
coste y recuperación con evidencia de composición; no se presume que una
microVM de `TestAttestor` sea una plataforma válida para agentes. Si se aprueba,
será una nueva capability con contratos, presupuesto, persistencia de checkpoints
y pruebas de restart/recovery propias.

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
