# Incidencia: regresiones descubiertas por lote D3

Fecha: 2026-07-10. Estado: abierto.

## Evidencia comun

El runner aislado `scripts/orquesta_test_batches.sh` ejecuto dos pases de seis
paquetes. El stack paso dos veces; el lote raiz/servidor/goal/state/atestador
fallo dos veces con los mismos cinco tests. Recibo temporal:
`/tmp/orquesta-d3-batch-runs/receipt.json`,
`reason_code=one_or_more_batches_failed`, 12 ejecuciones y cero pases completos.
La reproduccion focal aislada confirma los cinco fallos.

## BUG-ORQ-20260710-208O: frontera Codex acepta PATH y no prueba version

`TestCodexCommandPathV0RechazaRutaRelativaAunqueEsteEnPath` demuestra que
`codexCommandPathV0` resuelve `codex` mediante `ORQUESTA_CODEX_PATH` aunque la
configuracion declara un comando relativo. `TestValidateCodexCommandAvailableV0PreflightVersion`
demuestra que el preflight solo comprueba que la ruta es absoluta: acepta un
ejecutable sin salida de version.

Impacto: contradice el contrato de ruta absoluta y hace que un binario de PATH
pueda cambiar la identidad del proveedor.

Cierre: comando explicito y absoluto; preflight con evidencia de version sin
exponer salida cruda; tests focales verdes. No reintroducir seleccion implicita
por PATH.

## BUG-ORQ-20260710-208P: Claude process rechaza un goal valido de servidor

`TestServerGoalBackendFromEnvV0ClaudeProcessLanzaYObservaResultadoV0` y
`TestServerGoalBackendFromEnvV0ClaudeProcessControlParaProcesoV0` fallan antes
de lanzar con `claude_goal_process_invalid` usando comandos falsos locales y
un spec valido. La frontera process no puede ejercer observacion ni control.

Cierre: recuperar la composicion de `claude_process` sin relajar refs, write
set ni control de proceso; ambos focales deben lanzar/observar o parar el
proceso falso aislado.

## BUG-ORQ-20260710-208Q: effective_config filtra argv de escalada sensible

`TestServerConfigFromEnvV0LeeDirectorEscaladaV0` publica en JSON valores de
argv configurado (`--model`, `sonnet`) aunque el setting esta marcado
`Sensitive`. El fallo sale al serializar `EffectiveConfig`, no al parsear la
configuracion.

Cierre: mantener presencia/ref redactada, nunca argumentos crudos de director
externo; conservar test de serializacion adversarial.

## Relacion con D3 y 208H

La separacion de metricas D3 pasa focalmente (`env_vars_orquesta=425`,
`env_vars_orquesta_test_only=103`), pero no se declara cierre global hasta que
las regresiones 208O/208P/208Q permitan dos pases aislados completos. 208H ya
tiene atestador activo en composicion; su evidencia de lotes hereda este
bloqueo, no un nuevo fallo de atestacion.
