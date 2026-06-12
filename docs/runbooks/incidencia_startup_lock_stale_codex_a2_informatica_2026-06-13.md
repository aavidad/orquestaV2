# Incidencia startup lock stale Codex A2 Informatica - 2026-06-13

## Contexto

Durante la tanda `a2_informatica_2026-06-13` lanzada desde OPES contra Orquesta
aislado en `127.0.0.1:8792`, los wrappers Codex quedaron esperando el bloqueo
global de arranque:

```text
/home/alberto/.codex/.orquesta-codex-startup.lock
```

No habia ningun proceso `codex exec` propietario y el directorio de lock era
anterior a la tanda. Los padres materializados no llegaron a invocar Codex hasta
retirar manualmente el directorio vacio con `rmdir`.

## Impacto

- Padres A2 Informatica 13-27 agotaron o pudieron agotar
  `ORQUESTA_CODEX_STARTUP_LOCK_TIMEOUT_SECONDS` sin producir artefactos.
- Padres posteriores empezaron a ejecutar tras retirar el lock stale.
- No se detecto trabajo util perdido en `temas/` antes del desbloqueo.

## Correccion aplicada

Se ha actualizado `modulos/orquesta-runtime-codex/codex_wrapper_v0.go` para que
el wrapper, mientras espera el lock, retire de forma segura un directorio de
lock vacio si supera `ORQUESTA_CODEX_STARTUP_LOCK_STALE_SECONDS`.

Caracteristicas:

- valor por defecto: `900` segundos;
- `0` desactiva el reaping;
- solo se elimina con `rmdir`, por lo que no toca locks con contenido;
- conserva el timeout normal si el lock no puede retirarse.

Prueba focal:

```bash
go test -count=1 ./modulos/orquesta-runtime-codex
```

Resultado: pasada.

## Tarea derivada

Reiniciar los Orquesta vivos cuando no tengan agentes activos para que usen el
wrapper parcheado. Si una tanda ya estaba materializada antes del reinicio, los
scripts antiguos no incluyen esta mejora y deben tratarse como candidatos a
relaunch controlado si no produjeron `agent_ack.json` ni artefactos.

## Observacion de apagado y rearranque

Al cerrar el servidor aislado `127.0.0.1:8792` tras finalizar los padres
iniciales, el comando:

```bash
go run ./cmd/orquesta-server stop --force --reason "reinicio sin agentes activos para cargar parche stale startup lock antes de relanzar temas 013-016"
```

devolvio `shutdown_timeout`, pero aplico senal cooperativa y libero el puerto.
El estado publico quedo momentaneamente en `starting` al intentar arrancar de
nuevo de forma directa. La ejecucion en primer plano con `timeout` mostro que el
servidor no fallaba inmediatamente, sino que seguia vivo hasta ser cortado.

Arranque operativo que funciono:

```bash
setsid env ORQUESTA_SERVER_ADDR=127.0.0.1:8792 \
  ORQUESTA_CODEX_PROJECT_WORKDIR=/home/alberto/Trabajo/OPES \
  ORQUESTA_OPES_PROJECT_WORKDIR=/home/alberto/Trabajo/OPES \
  ORQUESTA_CODEX_RUNTIME_WORKDIR=/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/a2_informatica_2026-06-13/runtime \
  ORQUESTA_SERVER_STATE_DIR=/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/a2_informatica_2026-06-13/state \
  ORQUESTA_SERVER_AUDIT_FILE=audit.jsonl \
  ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED=false \
  go run ./cmd/orquesta-server run
```

Tarea tecnica: revisar por que `stop` puede terminar en `shutdown_timeout` aun
cuando no hay agentes activos y por que el arranque desacoplado requiere
`setsid` para quedar estable en este flujo. La solucion debe vivir en Orquesta,
no en instrucciones manuales de salida del paso.

## Observacion adicional

En la misma tanda, los `codex_stderr.log` muestran al inicio:

```text
failed to install system skills: io error while remove existing system skills dir: Permission denied
```

No se ha tratado como fallo bloqueante porque Codex continua la ejecucion y los
agentes escriben contenido. Queda como tarea tecnica separada: revisar el modo
de `CODEX_HOME` bajo `workspace-write`, preferiblemente con `CODEX_HOME`
aislado por agente/proyeccion controlada o con sandbox configurado por operador,
sin conceder escritura global a `~/.codex` por defecto.
