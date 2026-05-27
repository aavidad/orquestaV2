# Guardian break-glass de autoprogramacion

Estado: base implementada en `cmd/orquesta-guardian`.
El ciclo residente de promocion puede delegar en este guardian con
`ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_ENABLED=1`.

## Objetivo

Evitar que una automejora rota deje a Orquesta sin capacidad de repararse. El
guardian vive fuera del servidor Orquesta y trabaja con binario candidato,
`last_good`, healthcheck temporal y paquete de reparacion durable.

## Flujo

1. Compila el candidato en un path temporal.
2. Ejecuta tests requeridos.
3. Arranca el candidato contra estado/runtime temporal; comprueba `/healthz`
   solo como liveness y `/api/v0/server/readiness` antes de efectos externos.
4. Detiene el candidato con politica de proceso por plataforma y registra
   `stop_receipt` compacto. Si el stop queda ambiguo o vivo, el healthcheck
   falla y no hay promocion.
5. Si todo pasa, guarda el binario actual como `last_good` y promociona el
   candidato de forma atomica.
6. Si falla build, tests o healthcheck, no toca el binario vivo, escribe
   manifest y crea `repair_packet.json`.
7. Si se configura `--repair-command`, lo ejecuta con
   `ORQUESTA_GUARDIAN_REPAIR_PACKET` para lanzar un reparador externo opt-in.
   Si se usa `--repair-codex`, exige write-set y pruebas requeridas, escribe un
   packet de lanzamiento versionado y deja el cierre supeditado a
   `agent_ack.json`, no a stdout ni exit code del launcher.

## Ejemplo

```bash
go run ./cmd/orquesta-guardian check-promote \
  --project-dir /home/alberto/Trabajo/orquesta \
  --state-dir /home/alberto/Trabajo/.orquesta-control/orquesta/guardian \
  --current-bin /home/alberto/Trabajo/.orquesta-control/orquesta/autoprogramacion-servidor-2026-05-23/bin/orquesta-server-latest \
  --test-command 'go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server' \
  --test-command 'go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-runtime-worktree'
```

Los comandos `check-promote`, `restore-last-good` y `shutdown-server` rechazan
argumentos posicionales sobrantes. Los comandos shell deben declararse mediante
flags explicitas (`--build-command`, `--test-command`, `--repair-command`) o la
env canonica equivalente; un token suelto despues de `flag.Parse` bloquea con
`guardian_config_extra_args:positional_args:<n>` sin eco del valor crudo.

## Lease de estado

`check-promote`, `restore-last-good` y `shutdown-server` reclaman un lease
durable bajo `state_dir/leases` antes de tocar `current`, `last_good`, candidato
o senalizacion del servidor. El receipt publico usa
`orquesta_guardian_promotion_lease.v0` con `lease_ref`, owner/ref opaco,
`acquired_at`, `deadline_at`, refs de estado/objetivo y reason code compacto.

Si otro proceso conserva el lease activo, el guardian devuelve
`guardian_promotion_lease_busy` sin ejecutar comandos de build/test/repair ni
copiar artefactos. Si el lease desaparece antes de promover, restaurar o
senalizar, devuelve `guardian_promotion_lease_lost`. El servidor residente
consume ambos estados como bloqueo retryable; no los declara promocion exitosa
ni lanza reparador Codex por ese motivo.

## Reparador externo opt-in

El guardian no mete Codex ni proveedor en el nucleo. Para reparar con agente hay
dos opciones opt-in.

Opcion generica: configurar `--repair-command`. Ese comando recibe:

- `ORQUESTA_GUARDIAN_REPAIR_PACKET`
- `ORQUESTA_GUARDIAN_FAILURE_PHASE`
- `ORQUESTA_GUARDIAN_PROJECT_DIR`

Opcion Codex: pasar `--repair-codex` con write-set y pruebas requeridas. El
guardian crea, junto al `repair_packet.json`, un
`orquesta_guardian_repair_launch_packet.v0` con:

- `reason=break_glass`;
- `write_set` cerrado;
- `required_tests`;
- refs opacas de manifest/repair packet;
- `run_ref`/`promotion_ref` si se declaran;
- presupuesto de un agente;
- contrato terminal `codex_agent_ack.v0` en `agent_ack.json`;
- politica de sandbox.

Despues lanza un solo agente externo gobernado por Director con:

```bash
<current-bin> codex-launch-director-wave \
  --agents 1 \
  --objective-file <repair.md> \
  --write-set <paths> \
  --required-tests <tests> \
  --branch-ref <branch_ref_opaca> \
  --worktree-ref <worktree_ref_opaca> \
  --allow-unmanaged-launch \
  --confirm-unmanaged-launch <wave_ref>
```

El binario usado debe ser el ultimo bueno o el binario vivo que todavia tenga el
comando `codex-launch-director-wave`. Si el servidor residente esta sano, usar
la cola normal de automejora; `--repair-codex` queda reservado a break-glass de
build/startup/readiness rotos.

El default del reparador Codex es `workspace-write` y `medium`. Para
`danger-full-access` debe declararse `--repair-codex-allow-broad-sandbox` y
`--repair-codex-sandbox-evidence-ref`; sin esa evidencia el guardian bloquea la
configuracion. El prompt Markdown solo transporta refs del repair packet y del
launch packet: no incrusta logs, rutas absolutas, HOME, tokens, payloads HTTP ni
transcripts previos.

## Integracion residente

Cuando `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_ENABLED=1` y
`ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_ENABLED=1`, el puerto de
promocion del servidor ejecuta el guardian despues de que el staging Git quede
promocionado localmente y antes de considerar completo el efecto de promocion de
automejora. Si el guardian falla, el efecto publico queda `blocked` y retryable,
con evidencia compacta; no se sustituye el binario vivo.

Variables canonicas del servidor:

- `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_COMMAND`.
- `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_STATE_DIR`.
- `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_CURRENT_BIN`.
- `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_CANDIDATE_BIN`.
- `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_LAST_GOOD_BIN`.
- `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_BUILD_COMMAND`.
- `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_TEST_COMMANDS`.
- `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_HEALTH_TIMEOUT`.
- `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_COMMAND_TIMEOUT`.
- `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_ARTIFACT_ROOT`.
- `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_ARTIFACT_MAX_BYTES`.
- `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_REPAIR_COMMAND`.
- `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_REPAIR_CODEX`.
- `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_REPAIR_CODEX_WRITE_SET`.
- `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_REPAIR_CODEX_REQUIRED_TESTS`.
- `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_REPAIR_CODEX_WORKTREE_REF`.
- `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_REPAIR_CODEX_BRANCH_REF`.
- `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_REPAIR_CODEX_RUN_REF`.
- `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_REPAIR_CODEX_PROMOTION_REF`.
- `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_REPAIR_CODEX_SANDBOX`.
- `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_REPAIR_CODEX_REASONING_EFFORT`.
- `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_REPAIR_CODEX_RUNTIME_DIR`.
- `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_REPAIR_CODEX_ALLOW_BROAD_SANDBOX`.
- `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_REPAIR_CODEX_SANDBOX_EVIDENCE_REF`.

Estas variables del servidor se proyectan a las `ORQUESTA_GUARDIAN_REPAIR_CODEX_*`
del guardian solo por opt-in. Si no se declaran refs especificas del reparador,
el servidor conserva `run_ref`, `promotion_ref`, `worktree_ref` y `branch_ref`
del comando de promocion como refs opacas; no las convierte en rutas ni nombres
Git.

Variables directas del guardian para el reparador Codex:

- `ORQUESTA_GUARDIAN_REPAIR_CODEX_WRITE_SET`.
- `ORQUESTA_GUARDIAN_REPAIR_CODEX_REQUIRED_TESTS`.
- `ORQUESTA_GUARDIAN_REPAIR_CODEX_WORKTREE_REF`.
- `ORQUESTA_GUARDIAN_REPAIR_CODEX_BRANCH_REF`.
- `ORQUESTA_GUARDIAN_REPAIR_CODEX_RUN_REF`.
- `ORQUESTA_GUARDIAN_REPAIR_CODEX_PROMOTION_REF`.
- `ORQUESTA_GUARDIAN_REPAIR_CODEX_SANDBOX`.
- `ORQUESTA_GUARDIAN_REPAIR_CODEX_ALLOW_BROAD_SANDBOX`.
- `ORQUESTA_GUARDIAN_REPAIR_CODEX_SANDBOX_EVIDENCE_REF`.

## Politica de artefactos binarios

La promocion y restore de `candidate`, `current` y `last_good` usan copia
streaming con presupuesto configurable (`--artifact-max-bytes` o
`ORQUESTA_GUARDIAN_ARTIFACT_MAX_BYTES`). El guardian calcula SHA-256 y tamano,
bloquea symlinks en la ruta, destinos no regulares, hardlinks inseguros y
cambios entre `Lstat`, open y verificacion posterior. Si se declara
`ORQUESTA_GUARDIAN_ARTIFACT_ROOT`, `candidate`, `current` y `last_good` deben
quedar dentro de esa raiz.

La escritura usa tmp no colisionable, fsync de fichero/directorio y rename
atomico. El manifest local incluye `orquesta_guardian_artifact_manifest.v0` con
hashes de `candidate`, `current` previo, `last_good` y `current` final. Si falla
backup o promocion, publica `last_good_unverified` o `promotion_incomplete` y no
declara `candidate_promoted`.

El servidor residente no usa solo el exit code del guardian como evidencia de
promocion. Al consumir `orquesta_guardian_result.v0` registra
`promotion_guardian_receipt.v0` con refs opacas de promocion/run/worktree/branch,
`guardian_attempt_ref`, refs de manifest/evidencia, hash/tamano compacto del
candidato cuando existe y estado de staging aplicado. Si el guardian corre con
`--promote=false`, el receipt queda en `candidate_verified` y el servidor bloquea
la promocion para no tratar la verificacion como binario activo promovido.

## Retencion y replay

Desde T220, cada intento del guardian se indexa por `attempt_ref`; si no viene
explicito, se deriva de `promotion_ref`, `shutdown_ref` o del generador local.
Los nombres de manifest, repair packet, prompt y runtime temporal usan esa ref
opaca, no rutas ni nombres Git.

Manifest, repair packet, prompt y logs redactados se escriben con
tmp+fsync+rename. Repetir el mismo intento con payload distinto no sobrescribe:
la salida publica queda en `guardian_manifest_incomplete` y expone
`retention_status` con reason code compacto. La salida publica no muestra rutas
absolutas ni cuerpos de log; solo refs, contadores, reason codes y estado de
retencion.

La politica local conserva `last_good` y evidencias actuales. La limpieza opera
por categoria: manifests terminales, repair packets, logs redactados, logs
crudos opt-in, `candidate-state`, `candidate-runtime` y runtime del reparador
Codex.

## Shutdown

Existe shutdown controlado por `modulos/orquesta-server-shutdown` y endpoint
`POST /api/v0/server/shutdown`. Para un restart de guardian no debe usarse el
atajo `orquesta-server stop` como unica pieza, porque hoy solicita
`forced=true`.

`cmd/orquesta-guardian shutdown-server` pide shutdown cooperativo por defecto
(`forced=false`), espera `shutdown_ready=true` y despues puede senalizar el PID
si se pasa `--server-pid`. El `deadline` es la hora limite del cierre ordenado:
se calcula desde el inicio usando `--shutdown-timeout`. Hasta ese momento los
agentes pueden escribir checkpoint/handoff y drenar. Si vence el timeout y no
hay cierre suficiente, el guardian escala a `forced=true`. `--now` salta esa
espera y pide forzado desde el principio.

Autoridad: el shutdown se solicita como `orquesta-director`. Los agentes no
pueden iniciar shutdown; solo preparan checkpoint/ACK y el Director decide si
continua la espera o fuerza tras el deadline.

Ejemplo:

```bash
go run ./cmd/orquesta-guardian shutdown-server \
  --project-dir /home/alberto/Trabajo/orquesta \
  --state-dir /home/alberto/Trabajo/.orquesta-control/orquesta/guardian \
  --current-bin /home/alberto/Trabajo/.orquesta-control/orquesta/autoprogramacion-servidor-2026-05-23/bin/orquesta-server-latest \
  --server-addr 127.0.0.1:8090 \
  --server-pid 12345 \
  --shutdown-timeout 2m
```
