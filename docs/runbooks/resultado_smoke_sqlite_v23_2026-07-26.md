# Resultado del smoke SQLite V23

Fecha: 2026-07-26.

## Decisión y alcance

SQLite se conserva como adaptador concreto de estado de la composición actual.
No se incorpora al núcleo genérico ni se convierte en requisito de las apps
externas. Una sustitución futura debe entrar por los puertos de persistencia y
por el wiring de composición, sin compartir la base interna con consumidores.

Este smoke actuó únicamente sobre una copia privada del backup previo. No
detuvo, reconfiguró, migró ni escribió la unidad o la base viva de `Codex12`.
Tampoco acredita Firecracker, que no forma parte de V23. La promoción física
del perfil es una decisión operativa separada y exige su propio contrato
transaccional y rollback, no un E2E microVM.

## Sujeto exacto

- Revisión del binario: `644f620654ebb9707ad04bf9ed859affe932209d`.
- SHA-256 del binario: `4a0d1072835bce232e126a288416cfe4b3dfc1764f410cbcbd7e1dc604d68909`.
- Backup V16: `00f6d253334edaeba8579f9019c743a861c5c32cdc6577a26b02c0c22bdc298a`.
- Adaptador systemd durante el smoke: SHA-256
  `2ee321ad0c462329371510226673180e3846bb75b33cdf8dcd7fc5f3dc887d73`.
- Configuración aislada: SHA-256
  `57f7ffc8f24904cb4605af905308a571190a96c7248353cb77dd29255d41a0bc`.

El recibo completo, privado y con permisos `0600` está en:

`/home/alberto/Trabajo/.orquesta-runtime-v2-v23/audits/sqlite-final-2e714100-20260726/receipt.json`

## Resultado

| Comprobación | Antes | Primer arranque | Reinicio |
| --- | ---: | ---: | ---: |
| `PRAGMA user_version` | 16 | 19 | 19 |
| migraciones físicas nuevas | — | 17, 18 y 19 | ninguna duplicada |
| `quick_check` | `ok` | `ok` | `ok` |
| filas de `foreign_key_check` | 0 | 0 | 0 |
| goals | 8 | 8 | 8 |
| executions | 8 | 8 | 8 |
| work items | 14 | 14 | 14 |
| outbox | 29 | 29 | 29 |
| effect attempts | 17 | 17 | 17 |

El digest funcional, excluyendo `schema_migrations` y las tres tablas de
auditoría del propio health-check, fue idéntico antes, después y tras reinicio:

`1e604fe9391134392d80073e17fe9c2a7ee3981552895d7e74119a0ec292e46a`

Cada arranque ejecutó exactamente un `orquesta.system.status`. Por ello el
resultado contiene dos invocaciones, dos outcomes y dos autorizaciones
auditables adicionales. No son trabajo de agentes ni mutación funcional.

Las InvocationID fueron:

- migración: `8c7c7569d17042f88f87171915d2e7a6`;
- reinicio: `2fd05b9dc8e64f9b9dd0432800110ced`.

Al cerrar, la unidad estaba `not-found`, la base no tenía procesos abiertos,
los logs de daemon tenían cero bytes y no quedaban procesos del candidato. La
base V19 de evidencia se conserva privada con SHA-256
`27dd0d5eab81e03514fd4617525acce10c863c6965078909650e827fe35e7188`.

## Incidencias aprovechadas

El recorrido cerró tres clases de error sin debilitar guardas:

1. `systemd-run` expandía expresiones Bash antes de entregar el argv. Se cerró
   con `--expand-environment=no` en `644f6206`.
2. El arnés manual usó sucesivamente un symlink como ejecutable, el subgrupo
   cgroup equivocado y un token ficticio de 65 bytes. Las guardas rechazaron
   correctamente los tres; el token canónico mide 43 bytes.
3. El readiness mostraba falsos errores durante la transición Bash→sleep.
   `8f528a1a` silencia solo esos sondeos y conserva un único fallo terminal.

Las incidencias están en `BUG-ORQ-20260726-562` a `564`.

## Limpieza y siguiente gate de este frente

Se enviaron a la papelera el build diagnóstico de 239 MiB, su base auxiliar y
las credenciales ficticias. Se retuvieron la base final, configuración,
recibo, logs vacíos y evidencias textuales de los intentos.

Este smoke no fija el siguiente gate de V23. V23 continúa por su contrato
Wizard, generación editorial y flujos restantes. Firecracker `1 + 16` queda
diferido fuera de V23.

Si se promueve `Codex12`, debe hacerse como operación independiente: backup
nuevo, parada cooperativa, recreación systemd delegada, verificación V16→V19 y
rollback por identidad exacta si falla cualquier comprobación.
