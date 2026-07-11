# APG-006 context rotation harness

Piloto aislado para comparar una sesion continua (`control`) con workers frescos
por frontera semantica (`treatment`). No cambia defaults, no promociona cambios y
no conoce proveedores productivos.

## Contrato

- El manifest fija antes de ejecutar un commit/snapshot comun, seis o mas pares,
  semilla/algoritmo de aleatorizacion, modelo, effort, herramientas, write-set y
  presupuesto total identicos por brazo. No se cambian umbrales tras observar
  resultados.
- Cada par es un bloque: ambos brazos parten del mismo commit, se lanzan
  concurrentemente y su orden queda contrabalanceado por
  `sha256-balanced-block-v1`. `plan` es determinista para manifest identico.
- Control recibe al inicio la union de refs disponible para todas las etapas;
  treatment recibe la ref comun y solo las refs de su etapa. Asi se prueba
  rotacion/resumen, no acceso desigual a informacion. El presupuesto treatment
  es compartido por el brazo completo, no se reinicia en cada etapa.
- `usage_by_role` siempre declara `director`, `workers`, `handoffs`,
  `recoveries` y `reviewers`, incluso con cero. Sus contadores
  `input_uncached`, `output`, `cache_read` y `cache_write` son mutuamente
  excluyentes; el coste se registra en micros. La revision final se reparte de
  forma declarada entre los brazos y entra en `reviewers`.
- El reviewer recibe solo el paquete ciego. El mapping modo/candidato se guarda
  aparte con modo `0600`; no se transportan transcript ni razonamientos.
- El `blind_candidate_ref` del dataset debe coincidir con el HMAC del paquete.
  Cada candidato se revisa en contexto fresco. La rubrica preregistrada usa una
  escala 0-100 y cuenta lineas cambiadas que no son necesarias para objetivo,
  criterios, tests, seguridad o mantenibilidad; las instrucciones quedan
  ligadas por SHA-256.
- La clave de `fixtures/` es solo sintetica. En una ejecucion real se crea fuera
  del repo/state visible al reviewer y se revela tras cerrar todas las reviews.
- Receipts idempotentes ligan operacion, input y resultado por SHA-256. Una
  etapa treatment posterior exige el receipt de la anterior.

## Uso

```bash
python3 scripts/experimentos/context_rotation/harness.py plan \
  --manifest /ruta/manifest.json

python3 scripts/experimentos/context_rotation/harness.py prepare \
  --manifest /ruta/manifest.json --apply --confirm EXPERIMENT_REF

python3 scripts/experimentos/context_rotation/harness.py run-stage \
  --manifest /ruta/manifest.json --pair PAIR --mode treatment --stage STAGE
```

Todo es dry-run salvo `--apply --confirm EXPERIMENT_REF`. Ejecutar un adaptador
externo exige ademas `--allow-provider --provider-command /ruta/absoluta`; se
invoca por argv, recibe el stage request JSON por stdin y debe devolver
`orquesta.context_rotation.provider_result.v1`. Este repositorio no incluye ni
ejecuta un proveedor real.

Orquesta invocara `plan`, lanzara los pares y ambos brazos en paralelo sobre las
`2*N` worktrees, y serializara solo las etapas causales dentro de cada treatment.
Despues generara paquetes `blind`, recogera el dataset y ejecutara `evaluate`.
Solo un veredicto `adopt` permite que Orquesta cree otra tarea causal para una
politica reversible; este harness nunca activa ni promociona esa politica.

## Protocolo y decision

1. Congelar manifest, commit, snapshot, corpus de seis o mas tareas y rubrica.
2. Generar `plan`; conservar hash y comprobar repeticion byte a byte.
3. Ejecutar cada bloque concurrente. No reutilizar proceso/contexto entre
   treatment stages; no ejecutar otras cargas deliberadas en un solo brazo.
4. Recoger uso por rol, duracion monotona, lecturas repetidas, rework, fallos,
   intervenciones y churn Git (`changed_lines`, `changed_files`).
5. Crear paquetes ciegos; revisar cada candidato en contexto fresco y conservar
   evidencia de score y lineas innecesarias. Descegar solo al cerrar reviews.
6. Reproducir decision desde dataset inmutable y ejecutar review final del
   informe.

`adopt` exige: seis pares completos; mediana pareada de tokens y coste con
reduccion >=20%; mejora simultanea en al menos dos tercios de pares; mediana de
ratio temporal <=110%; calidad treatment no inferior en ningun par; lineas
innecesarias menores en agregado y al menos dos tercios de pares, sin aumento en
ningun par; tests/review aceptados y ausencia de falso verde, perdida causal,
efecto externo adicional o defecto severo. Evidencia incompleta produce
`inconclusive`; evidencia completa que incumple un umbral produce
`keep_control`.

Seis pares permiten un piloto decisorio acotado, no una conclusion universal.
Las tareas deben fijarse antes de ejecutar y representar varios tipos/tamanos;
si se seleccionan por conveniencia o comparten demasiado codigo, debe constar
como amenaza a validez y no se generaliza fuera de ese corpus. Un segundo run
con nueva semilla/corpus sigue siendo necesario antes de cambiar un default
amplio.

```bash
bash scripts/experimentos/context_rotation/test_harness.sh
```

`cleanup` conserva state/receipts y solo elimina worktrees limpias cuando la
decision durable, resultados, hashes y retencion minima son validos. Si falta
evidencia o hay cambios sin entregar, retiene todo.

Implementacion deliberadamente pequena: `domain.py` contiene solo reglas puras
de evaluacion; `harness.py` contiene el adaptador concreto Git/proceso. No hay
framework de puertos, registro de plugins ni configuracion productiva.
