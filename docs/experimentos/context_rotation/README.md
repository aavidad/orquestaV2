# APG-006 context rotation harness

Piloto aislado para comparar una sesion continua (`control`) con workers frescos
por frontera semantica (`treatment`). No cambia defaults, no promociona cambios y
no conoce proveedores productivos.

## Contrato

- El manifest fija un commit/snapshot comun, tres o mas pares, modelo, effort,
  herramientas, write-set y presupuesto identicos por brazo.
- `usage_by_role` siempre declara `director`, `workers`, `handoffs`,
  `recoveries` y `reviewers`, incluso con cero. Sus contadores
  `input_uncached`, `output`, `cache_read` y `cache_write` son mutuamente
  excluyentes; el coste se registra en micros. La revision final se reparte de
  forma declarada entre los brazos y entra en `reviewers`.
- El reviewer recibe solo el paquete ciego. El mapping modo/candidato se guarda
  aparte con modo `0600`; no se transportan transcript ni razonamientos.
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

```bash
bash scripts/experimentos/context_rotation/test_harness.sh
```

`cleanup` conserva state/receipts y solo elimina worktrees limpias cuando la
decision durable, resultados, hashes y retencion minima son validos. Si falta
evidencia o hay cambios sin entregar, retiene todo.

Implementacion deliberadamente pequena: `domain.py` contiene solo reglas puras
de evaluacion; `harness.py` contiene el adaptador concreto Git/proceso. No hay
framework de puertos, registro de plugins ni configuracion productiva.
