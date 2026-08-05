# Auditoría del primer Codex físico en Firecracker

Fecha de corte: 2026-08-05.

## Veredicto

B10 ya dispone de recuperación durable application+SQLite: intento histórico,
fences, receipt perdido, retry, reapertura y carrera quedan acreditados sin un
segundo `Launch`. Esto no prueba que un Codex haya trabajado dentro de KVM.

Estimación del corte físico:

- B10 lógico/durable: 90–95%;
- Firecracker/microVM hasta primer Codex real: 65–70%;
- pendiente físico: 30–35%, unas 10–18 horas de trabajo con revisión y E2E.

El E2E histórico de una microVM y ola de 16 acredita `TestAttestor`, no agentes
Codex. Kernel, initramfs y perfil existentes son anteriores al candidato actual
y deben reconstruirse; no se reutilizan como evidencia nueva.

## Fronteras abiertas

1. El daemon/Jailer necesita privilegios, pero su UDS queda `0600` para el
   propietario del daemon. Orquesta no-root no tiene todavía una delegación por
   grupo y `peercred` que rechace a terceros.
2. La compilación del perfil omite `controlled_egress_proxy` y el executor
   Codex elimina las variables de proxy. El huésped no tiene una salida web
   gobernada reproducible.
3. No existe entrega sellada de OAuth/API key a `/trabajo/.codex`. El runner
   parte de `HOME`/`CODEX_HOME` vacíos; ningún secreto debe entrar en rootfs ni
   quedar durable después del cierre.
4. Falta el broker host real para control, credenciales, MCP/mailbox y
   artefactos ligados a `RunRef`, fence y atestación.

KVM y Firecracker 1.16.1 están disponibles. El usuario ya puede acceder a
`/dev/kvm`; no hace falta sudo para ejecutar KVM. El sudo inevitable se limita
a instalar el servicio root, usuario/grupo, directorios y delegación cgroup.

## Minitareas de cierre

| ID | Trabajo | Criterio de cierre | Estimación |
| --- | --- | --- | ---: |
| PFC-01 | Reconstruir kernel/initramfs/perfil/executor desde candidatos actuales | manifiesto, digests y build reproducible; cero `.tmp` antiguos como evidencia | 1–2 h |
| PFC-02 | Frontera root/no-root del UDS | daemon root; Orquesta por grupo+`peercred`; tercero rechazado; socket no público | 2–3 h |
| PFC-03 | Broker host mínimo | sesión exacta por `RunRef`+fence+atestación; control/MCP/mailbox/artefactos por refs opacas | 2–4 h |
| PFC-04 | Credencial Codex sellada | secreto solo en memoria/tmpfs `0600`, principal exacto y borrado terminal verificable | 2–4 h |
| PFC-05 | Egress controlado | proxy explícito funciona; Internet directo, LAN, RFC1918, inbound y east-west quedan denegados | 2–4 h |
| PFC-06 | E2E físico acotado | una microVM, un Codex real, resultado terminal durable y limpieza exacta | 2–4 h |
| PFC-07 | Instalador auditable único | único sudo idempotente para servicio/grupo/directorios/cgroup, con status y rollback documentados | 1–2 h |

B11 (parada exacta) y B12 (preservación/sello/compuerta B) se mantienen como
cortes posteriores separados. No se engordan dentro de PFC-06 para fabricar un
"100%" prematuro.
