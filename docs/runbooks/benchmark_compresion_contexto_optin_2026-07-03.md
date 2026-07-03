# Runbook: benchmark opt-in de compresion de contexto documental

Fecha: 2026-07-03.

Scope: CTX-TASK-801D / T284. Este runbook cubre solo evaluacion documental
opt-in de contexto no critico. No habilita un adaptador productivo.

## Regla de uso

El script exige confirmacion explicita:

```bash
ORQUESTA_CONTEXT_COMPRESSION_BENCHMARK_CONFIRM=CTX_COMPRESSION_BENCHMARK_OPT_IN \
  bash scripts/benchmark_context_compression_optin.sh \
  --input scripts/docs/context_compression_benchmark_sample_noncritical_t284.md \
  --output /tmp/orquesta-context-compression-benchmark.json \
  --compressed-output /tmp/orquesta-context-compression-preview.md \
  --target-ratio 0.45
```

El compresor por defecto es un baseline extractivo local sin paquetes externos.
Para evaluar LLMLingua u otra herramienta local, el operador puede pasar un
comando que lea stdin y escriba stdout:

```bash
ORQUESTA_CONTEXT_COMPRESSION_BENCHMARK_CONFIRM=CTX_COMPRESSION_BENCHMARK_OPT_IN \
ORQUESTA_CONTEXT_COMPRESSION_COMMAND='comando-local-de-compresion' \
  bash scripts/benchmark_context_compression_optin.sh \
  --input ruta/no/critica.md \
  --output /tmp/orquesta-context-compression-benchmark.json
```

## Material protegido

No usar este benchmark con material operativo critico. El script rechaza:

- `AGENTS.md`;
- rutas de `contratos` o `contracts`;
- bloques con marcadores de instrucciones de agente;
- contenido que declare reglas hard, hard rules o write-set.

Si aparece un rechazo, no lo rodees con copias editadas del material protegido.
Selecciona corpus no critico o registra que no procede benchmark.

## Lectura de resultado

El JSON `orquesta_context_compression_benchmark.v0` compara:

- `source_bytes` frente a `compressed_bytes`;
- `source_tokens_estimated` frente a `compressed_tokens_estimated`;
- `lost_refs_count` y `lost_refs`;
- `elapsed_ms`.

Un resultado con `lost_refs_count > 0` no bloquea el script, pero bloquea
cualquier decision de producto hasta revisar por que se perdieron refs.

## Verificacion manual

```bash
bash -n scripts/benchmark_context_compression_optin.sh
bash -n scripts/test_benchmark_context_compression_optin.sh
bash scripts/test_benchmark_context_compression_optin.sh
git diff --check
```

## Nota de scanner

El goal recibio `backlog_scan_ref=scan-ref-backlog-d438fd99b7a6` con foto
esperada `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md:line:16`
`sha256:49ddfa8cf3f609e42a9cec0342e28231a4bce588da6c8af8ba14f046c9e78d7a`.
En este worktree, la linea 16 actual es
`## T284 benchmark-compresion-contexto-optin` y su hash de linea observado
durante la ejecucion fue
`c294611add48932f0fb6ea5ee3c5f87e89d890d7e6ca20430c7c3cf69653d0c1`.
No se intento merge del backlog fuera del write-set; si el scanner necesita
insertar nuevos bloques, debe rebasear contra la foto actual.
