# APG-006 - resultado de rotacion de contexto

- Experimento: `EXPERIMENT_REF`
- Manifest SHA-256: `SHA256`
- Commit/snapshot: `COMMIT` / `SNAPSHOT_REF@SHA256`
- Semilla/algoritmo y plan: `SEED` / `sha256-balanced-block-v1` / `REF@SHA256`
- Dataset SHA-256: `SHA256`
- Decision mecanica: `adopt|keep_control|inconclusive`

## Pares

| Par | Control tokens/coste/tiempo | Treatment tokens/coste/tiempo | Calidad C/T | Lineas innecesarias C/T | Tests/handoffs | Receipts |
|---|---:|---:|---:|---:|---|---|
| `PAIR_REF` | `...` | `...` | `.../...` | `.../...` | `...` | `REF@SHA256` |

## Umbrales

- Paridad de aceptacion/tests y ausencia de falso verde, perdida causal,
  efecto externo adicional o defecto severo: `PASS|FAIL`.
- Seis pares completos; mediana pareada de reduccion tokens/coste >=20% y al
  menos dos tercios de pares mejores: `...`.
- Mediana de tiempo <=110% del control: `...`.
- Score treatment no inferior por par y lineas innecesarias menores en agregado,
  al menos dos tercios de pares, sin aumento por par: `...`.
- Reworks, fallos e intervencion humana no aumentan: `...`.
- Continuidad solo con handoff compacto/refs y reproduccion ciega: `...`.

## Evidencia

- Decision JSON: `REF@SHA256`
- Revision independiente/reproduccion: `REF@SHA256`
- Rubrica/instrucciones ciegas: `REF@SHA256`
- Incidencias, intervenciones y defectos posteriores: `REF@SHA256|none`

## Amenazas a validez

- Seleccion/representatividad de tareas: `...`.
- Deriva temporal, carga y fallos de proveedor: `...`.
- Contaminacion entre brazos/reviewer o descegado accidental: `...`.
- Sensibilidad de metricas y desacuerdo sobre codigo innecesario: `...`.
- Alcance de generalizacion y replica requerida: `...`.

No se promociono codigo ni se cambio configuracion/default productivo.
