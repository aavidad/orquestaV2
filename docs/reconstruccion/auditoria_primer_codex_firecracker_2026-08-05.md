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
   sesiones de broker idempotentes y ligadas a identidad causal. El broker
   interactivo genérico no bloquea el primer Codex; PFC-04 sí necesita un
   listener one-shot estrecho sobre `control_broker`, ligado a la apertura
   física ya autenticada.
4. **Cerrada en código la cadena PFC-05a/b/c.**
   `orquestaV2@db1ac680` y `orquestaV2@8492a97c` conservan y recuperan la
   autoridad histórica de egreso; `011efb3a`, `11b27c85` y `fe7eefb2` la
   transportan como bytes exactos; `agente_microvm@17175ae`, `44814e5b` y
   `b6f0fafd` decodifican el contrato estricto, firman la concesión y proyectan
   únicamente `HTTP_PROXY`/`HTTPS_PROXY` hacia el loopback controlado. El gate
   amplio quedó verde tras `a74bfc26`. Falta acreditación física en PFC-06.
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
   `4f48195c`, `02b864a3` y `5a8d4bb6` fijan la autoridad física, su claim
   causal y la selección exacta de credencial por colocación. C4 sigue
   pendiente en la persistencia SQLite de esa autoridad, el broker host y el
   wiring residente.

KVM y Firecracker 1.16.1 están disponibles. El usuario ya puede acceder a
`/dev/kvm`; no hace falta sudo para ejecutar KVM. El sudo inevitable se limita
a instalar el servicio root, usuario/grupo, directorios y delegación cgroup.

## Minitareas de cierre

| ID | Trabajo | Criterio de cierre | Estimación |
| --- | --- | --- | ---: |
| PFC-01 | **Cerrado en código/arnés** por `5412c70e`: activos Codex actuales reproducibles | construcción doble exacta, manifiesto ligado y cancelación limpia; no acredita KVM | PFC-06 para evidencia física |
| PFC-02 | **Cerrado en código/arnés**: frontera root/no-root del UDS (`agente_microvm@989272d`) | daemon root; Orquesta por grupo+`peercred`; tercero rechazado; socket no público | PFC-06/PFC-07 para evidencia física |
| PFC-03 | **PFC-03a cerrado** por `ca78a490`; broker general aplazable | sesión exacta por `RunRef`+fence+atestación; el listener one-shot de credencial queda en PFC-04 | no bloquea por sí solo PFC-06 |
| PFC-04 | C1/C1b/C2/C3 y C4a-1/2/3b1 cerrados por `ec9304e`, `6d08deb`, `2f00b3d`, `d6ce92c1`, `5d00ede4`, `b96b3a2e`, `b25b1f58`, `7f04863d`, `4f48195c`, `02b864a3`, `5a8d4bb6` y pin `cc7cec3e` | secreto solo en memoria/tmpfs fuera de `/trabajo`, `0600`, principal exacto, consumo físico one-shot durable y claim causal por colocación | cerrar persistencia de autoridad, broker host y wiring |
| PFC-05 | **Cadena durable/transporte/compilación cerrada en código** hasta `b6f0fafd` y gate amplio verde tras `a74bfc26` | proxy explícito; concesión no llega al huésped; solo endpoint loopback | PFC-06 para evidencia física |
| PFC-06 | E2E físico acotado | una microVM, un Codex real, resultado terminal durable y limpieza exacta | 2–4 h |
| PFC-07 | **PFC-07a cerrado offline** por `agente_microvm@2acd300` | único sudo idempotente para servicio/grupo/directorios/cgroup, con status y rollback exactos | PFC-07b físico |

B11 (parada exacta) y B12 (preservación/sello/compuerta B) se mantienen como
cortes posteriores separados. No se engordan dentro de PFC-06 para fabricar un
"100%" prematuro.
