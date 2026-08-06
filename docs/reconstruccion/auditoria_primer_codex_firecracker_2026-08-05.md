# Auditoría del primer Codex físico en Firecracker

Fecha de corte: 2026-08-06.

## Veredicto

B10 ya dispone de recuperación durable application+SQLite: intento histórico,
fences, receipt perdido, retry, reapertura y carrera quedan acreditados sin un
segundo `Launch`. Esto no prueba que un Codex haya trabajado dentro de KVM.

Estimación del corte físico:

- B10 lógico/durable: 90–95%;
- Firecracker/microVM hasta primer Codex real: aproximadamente 89%;
- pendiente físico: aproximadamente 11%.

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
   sesiones de broker idempotentes y ligadas a identidad causal. El broker
   interactivo genérico no bloquea el primer Codex; PFC-04 sí necesita un
   listener one-shot estrecho sobre `control_broker`, ligado a la apertura
   física ya autenticada.
4. **Cerrada en código la cadena PFC-05a/b/c y su resolver compuesto.**
   `orquestaV2@db1ac680` y `orquestaV2@8492a97c` conservan y recuperan la
   autoridad histórica de egreso; `011efb3a`, `11b27c85` y `fe7eefb2` la
   transportan como bytes exactos; `agente_microvm@17175ae`, `44814e5b` y
   `b6f0fafd` decodifican el contrato estricto, firman la concesión y proyectan
   únicamente `HTTP_PROXY`/`HTTPS_PROXY` hacia el loopback controlado.
   `59113b5d` y `2c3ebe88` cargan y componen el resolver de política sellada.
   El gate amplio quedó verde tras `a74bfc26`. Falta que el Goal/smoke solicite
   una ref explícita y la acreditación física en PFC-06.
5. El asset huésped con egreso debe reconstruirse y fijarse atómicamente: un
   huésped anterior rechaza correctamente el campo nuevo del WorkPacket.
6. No existe entrega sellada de OAuth/API key a `/trabajo/.codex`. El runner
   parte de `HOME`/`CODEX_HOME` vacíos; ningún secreto debe entrar en rootfs ni
   quedar durable después del cierre.
7. **PFC-07a cerrado offline.** `agente_microvm@2acd300` instala una slice
   estable, conserva un único writer durante upgrade, usa temporales root
   confiables y liga artefactos, estados y rollback mediante receipts. Dos
   rondas de revisión y el arnés quedaron verdes. Sigue marcado
   `launch_ready=false`: no se ha ejecutado sudo ni se ha acreditado
   systemd/cgroup/KVM real; eso pertenece a PFC-07b.
8. **PFC-04 C1/C1b/C2/C3 y C4a-1/2/3b1 cerrados en código.** El protocolo público Rust y Go
   quedó fijado por `agente_microvm@ec9304e` y `6d08deb`, con pin exacto en
   `orquestaV2@cc7cec3e`. `agente_microvm@2f00b3d` proyecta `auth.json` en un
   tmpfs privado, purga antes del ACK terminal, hace fatal cualquier estado
   ambiguo e impide reutilizar la microVM o entrar por la ejecución legacy.
   Dos rondas de revisión corrigieron los falsos verdes antes del commit.
   `orquestaV2@5d00ede4` y `b96b3a2e` definen y persisten el consumo durable
   `UseOnce`; `b25b1f58` y `7f04863d` fijan la consulta de versión sin material;
   `4f48195c`, `02b864a3`, `5a8d4bb6` y `130961aa` fijan la autoridad física,
   su claim causal, la selección exacta por colocación y su enlace con la
   compilación firmada. `f02b811e`, `f6c07202` y `01f22436` añaden las
   migraciones SQLite v32/v33 y el registro durable
   `Prepare -> BindExternal -> Resolve`, con sesión exacta, carrera, rollback y
   reapertura acreditados. `2bf62d96` cierra el handler one-shot y `9c64e12b`
   compone resolver, registry y replay histórico en el arranque productivo.
   `911357d9` cierra además el listener UDS residente: publicación privada,
   `SO_PEERCRED`, concurrencia acotada, lifecycle, cleanup ligado al inodo y
   sustituciones concurrentes quedaron revisados sin P0/P1/P2. `4db7e8b6`
   completa su composición residente, el cierre broker -> cliente -> store ->
   SQLite y los rollbacks, con revisión final P0/P1=0. `1bc90fcd` añade una
   fuente segura de `auth.json` sin exponer material. `f9b86e00` deja el
   provisionado puro y reanudable de firma Ed25519+auth Codex con
   `ProvisionStore`, callbacks fail-closed, cancelación y adaptador local que
   reabre una respuesta `Create` perdida. Sigue faltando CLI/composición
   operativa y acreditación con el servicio host físico; no acredita KVM.
9. **Readiness PFC-06 incompleto.** El resolver productivo de egreso está
   compuesto, pero el Goal/smoke debe pedir su ref explícita.
   `agente_microvm@a8d1e76` lleva el perfil físico a config/motor/instalador,
   `agente_microvm@b67815c` construye el descriptor puro y
   `agente_microvm@7d3644a` proyecta JSON CLI canónico desde configuración
   efectiva+digest causal. Falta publicación root-owned con receipt/installer
   y no está acreditado el digest real del binario ni KVM. Son minitareas
   acotadas sobre contratos existentes, no motivos para ampliar PFC-04.
   Orquesta no debe inventar vCPU, memoria ni identidades de servicio mediante
   flags porque PFC-01 no contiene esos hechos.

KVM y Firecracker 1.16.1 están disponibles. El usuario ya puede acceder a
`/dev/kvm`; no hace falta sudo para ejecutar KVM. El sudo inevitable se limita
a instalar el servicio root, usuario/grupo, directorios y delegación cgroup.

## Minitareas de cierre

| ID | Trabajo | Criterio de cierre | Estimación |
| --- | --- | --- | ---: |
| PFC-01 | **Cerrado en código/arnés** por `5412c70e`: activos Codex actuales reproducibles | construcción doble exacta, manifiesto ligado y cancelación limpia; no acredita KVM | PFC-06 para evidencia física |
| PFC-02 | **Cerrado en código/arnés**: frontera root/no-root del UDS (`agente_microvm@989272d`) | daemon root; Orquesta por grupo+`peercred`; tercero rechazado; socket no público | PFC-06/PFC-07 para evidencia física |
| PFC-03 | **PFC-03a cerrado** por `ca78a490`; broker general aplazable | sesión exacta por `RunRef`+fence+atestación; el listener one-shot de credencial queda en PFC-04 | no bloquea por sí solo PFC-06 |
| PFC-04 | C1/C1b/C2/C3, handler one-shot, listener UDS residente y compuesto, replay productivo y autoridad SQLite v32/v33 cerrados hasta `4db7e8b6`, `911357d9`, `2bf62d96`, `9c64e12b` y `01f22436`; pin huésped `cc7cec3e`; fuente segura `auth.json` `1bc90fcd`; provisionado puro/reanudable Ed25519+auth con `ProvisionStore` y reapertura local por `f9b86e00` | secreto solo en memoria/tmpfs fuera de `/trabajo`, `0600`, principal exacto, consumo one-shot y autoridad preparada/bound durable | falta CLI/composición operativa y acreditación host física; no acredita KVM |
| PFC-05 | **Cadena durable/transporte/compilación y resolver compuesto cerrados en código** hasta `b6f0fafd`, `59113b5d` y `2c3ebe88` | proxy explícito; concesión no llega al huésped; solo endpoint loopback | Goal/smoke debe pedir ref; PFC-06 para evidencia física |
| PFC-06 | E2E físico acotado; solicitud de egreso y publicación root-owned de descriptor+receipt pendientes. `b67815c` construye el descriptor y `7d3644a` proyecta su JSON CLI canónico desde config efectiva+digest causal, sin acreditar el digest real del binario | una microVM, un Codex real, resultado terminal durable y limpieza exacta | 11% del hito físico |
| PFC-07 | **PFC-07a cerrado offline** por `agente_microvm@2acd300`; perfil físico en config/motor/instalador por `a8d1e76` | único sudo idempotente para servicio/grupo/directorios/cgroup, publicación root-owned, receipt/installer, status y rollback exactos | PFC-07b físico; no acredita KVM |

B11 (parada exacta) y B12 (preservación/sello/compuerta B) se mantienen como
cortes posteriores separados. No se engordan dentro de PFC-06 para fabricar un
"100%" prematuro.
