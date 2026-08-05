# Auditoría del primer Codex físico en Firecracker

Fecha de corte: 2026-08-05.

## Veredicto

B10 ya dispone de recuperación durable application+SQLite: intento histórico,
fences, receipt perdido, retry, reapertura y carrera quedan acreditados sin un
segundo `Launch`. Esto no prueba que un Codex haya trabajado dentro de KVM.

Estimación del corte físico:

- B10 lógico/durable: 90–95%;
- Firecracker/microVM hasta primer Codex real: aproximadamente 78%;
- pendiente físico: aproximadamente 22%.

Este porcentaje mide solo el corte hasta el primer Codex físico. No representa
el porcentaje de cierre de todo Orquesta.

El E2E histórico de una microVM y ola de 16 acredita `TestAttestor`, no agentes
Codex. PFC-01 ya reconstruye kernel, initramfs, perfil, executor y huésped desde
los candidatos actuales con manifiesto reproducible. Sigue sin ser evidencia de
arranque KVM ni de ejecución Codex física.

## Estado de fronteras

1. **Cerrada en código y arnés (PFC-02).** El daemon/Jailer puede publicar el
   UDS para un GID efectivo exacto con directorio `0750`, socket `0660`,
   `SO_PEERCRED`, PID vivo y publicación atómica; el modo por defecto conserva
   `0700`/`0600`. `agente_microvm@989272d` pasó 354 pruebas y revisión xhigh.
   La instalación root y su E2E físico pertenecen a PFC-07/PFC-06.
2. **Cerrada en código y arnés (PFC-01).** `orquestaV2@5412c70e` construye los
   activos Codex actuales dos veces con igualdad exacta, manifiesto final
   `3bf07f6ad35df0acda0514580ab24a0e6c130e40103dda10e2e2b85537b84f0a`,
   fuentes Cargo desde 22 archivos `.crate` verificados, lock efectivo
   publicado, rutas locales remapeadas y limpieza ante cancelación. La revisión
   independiente cerró con P0=0/P1=0. No afirma cadena de suministro hermética,
   KVM ni Codex real.
3. **Cerrada la autoridad neutral de PFC-03a.** `orquestaV2@ca78a490` define
   sesiones de broker idempotentes y ligadas a identidad causal. Falta PFC-03b:
   adaptador host real y composición física contra la microVM.
4. **Cerrada la persistencia de autoridad de PFC-05a.**
   `orquestaV2@db1ac680` y `orquestaV2@8492a97c` conservan y recuperan la
   autoridad histórica de egreso. Falta transportarla al lanzamiento y
   compilar el proxy físico sin convertir configuración estática en permiso.
5. La compilación del perfil todavía omite `controlled_egress_proxy` y el executor
   Codex elimina las variables de proxy. El huésped no tiene una salida web
   gobernada reproducible.
6. No existe entrega sellada de OAuth/API key a `/trabajo/.codex`. El runner
   parte de `HOME`/`CODEX_HOME` vacíos; ningún secreto debe entrar en rootfs ni
   quedar durable después del cierre.
7. Falta el broker host real para control, credenciales, MCP/mailbox y
   artefactos ligados a `RunRef`, fence y atestación.

KVM y Firecracker 1.16.1 están disponibles. El usuario ya puede acceder a
`/dev/kvm`; no hace falta sudo para ejecutar KVM. El sudo inevitable se limita
a instalar el servicio root, usuario/grupo, directorios y delegación cgroup.

## Minitareas de cierre

| ID | Trabajo | Criterio de cierre | Estimación |
| --- | --- | --- | ---: |
| PFC-01 | **Cerrado en código/arnés** por `5412c70e`: activos Codex actuales reproducibles | construcción doble exacta, manifiesto ligado y cancelación limpia; no acredita KVM | PFC-06 para evidencia física |
| PFC-02 | **Cerrado en código/arnés**: frontera root/no-root del UDS (`agente_microvm@989272d`) | daemon root; Orquesta por grupo+`peercred`; tercero rechazado; socket no público | PFC-06/PFC-07 para evidencia física |
| PFC-03 | **PFC-03a cerrado** por `ca78a490`; falta adaptador/composición física | sesión exacta por `RunRef`+fence+atestación; control/MCP/mailbox/artefactos por refs opacas | PFC-03b pendiente |
| PFC-04 | Credencial Codex sellada | secreto solo en memoria/tmpfs `0600`, principal exacto y borrado terminal verificable | 2–4 h |
| PFC-05 | **PFC-05a durable cerrado** por `db1ac680` y `8492a97c`; faltan transporte y proxy | proxy explícito funciona; Internet directo, LAN, RFC1918, inbound y east-west quedan denegados | PFC-05b/PFC-05c pendientes |
| PFC-06 | E2E físico acotado | una microVM, un Codex real, resultado terminal durable y limpieza exacta | 2–4 h |
| PFC-07 | Instalador auditable único | único sudo idempotente para servicio/grupo/directorios/cgroup, con status y rollback documentados | 1–2 h |

B11 (parada exacta) y B12 (preservación/sello/compuerta B) se mantienen como
cortes posteriores separados. No se engordan dentro de PFC-06 para fabricar un
"100%" prematuro.
