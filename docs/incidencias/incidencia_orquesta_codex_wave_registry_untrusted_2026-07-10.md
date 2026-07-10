# Incidencia: 208M monitor de ola Codex rechaza su propio registro

Fecha: 2026-07-10. Estado: abierto.

## Sintoma observado

La ola aislada `wave-ref-model-routing-legacy-208k-20260710` fue creada por
`orquesta-server codex-launch-wave` con
`--runtime-dir /tmp/orquesta-wave-model-routing-legacy-208k` y devolvio un
resumen con un agente en estado `running`, `process_proof_ref` y su digest.
Sin embargo, sus comandos de observacion devolvieron inmediatamente:

```text
codex-wave-status: blocked_registry_untrusted
codex-wave-tail: blocked_registry_untrusted
```

La comprobacion de proceso del mismo instante mostro vivo el wrapper y el
proceso `codex exec` bajo el runtime aislado
`/tmp/orquesta-wave-model-routing-legacy-208k/agent-01`.

La causa concreta está identificada: `codexWaveRuntimeUnderAllowedRootV0`
solo acepta raíces bajo el proyecto, su `runtime` hermano o
`ORQUESTA_CODEX_RUNTIME_WORKDIR/codex-waves`; el lanzador no aplica esa misma
validación a `--runtime-dir`. La prueba no demuestra corrupción de
`process_proof`: demuestra que la CLI permite crear una ola que su monitor no
podrá observar.

## Impacto

Una ola que Orquesta puede lanzar pero no observar no acredita autonomia:
impide confirmar progreso, resultado, fallo o parada cooperativa usando la
superficie gobernada. La lectura directa de logs se uso solo como diagnostico
de emergencia y no puede sustituir al contrato de observacion.

La invocacion del binario sin subcomando tambien produjo un panic
`slice bounds out of range [1:0]`; se conserva como sintoma de robustez de CLI
del mismo frente, sin asumir aun causa comun.

## Evidencia retenida

- Runtime: `/tmp/orquesta-wave-model-routing-legacy-208k`.
- Registro: `codex_wave_registry_v0.json`, con PID `2692011` en el momento de
  la observacion y `process_proof_digest`
  `sha256:a70576aaf3653bf853511c3918c852ec1ec0b83ab8944c32d285babfc21577fd`.
- El proceso hijo `codex exec` seguia vivo mientras los comandos gobernados
  devolvian `blocked_registry_untrusted`.

## Correccion y cierre requeridos

1. Aplicar al lanzamiento la misma validación de raíces que al control, o
   definir una raíz aislada `/tmp` canónica y validarla de forma idéntica en
   ambas operaciones. Nunca iniciar una ola real con ruta no observable.
2. Mantener rechazo para registros realmente manipulados o procesos fuera del
   runtime permitido.
3. Cubrir: ruta no permitida -> launch rechazado; ola creada en raíz permitida
   -> status/tail `running`; ola terminada -> resultado
   terminal; registro manipulado -> rechazo tipado sin lectura de logs.
4. Hacer que invocar el binario sin subcomando devuelva uso/exit no cero, sin
   panic.
5. Ejecutar pruebas focales aisladas y dejar recibo de una ola local observada
   y cerrada por la API/CLI de Orquesta.
