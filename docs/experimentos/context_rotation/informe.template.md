# APG-006 - resultado de rotacion de contexto

- Experimento: `EXPERIMENT_REF`
- Manifest SHA-256: `SHA256`
- Commit/snapshot: `COMMIT` / `SNAPSHOT_REF@SHA256`
- Dataset SHA-256: `SHA256`
- Decision mecanica: `adopt|keep_control|inconclusive`

## Pares

| Par | Control tokens/coste/tiempo | Treatment tokens/coste/tiempo | Tests/review | Handoffs | Receipts |
|---|---:|---:|---|---|---|
| `PAIR_REF` | `...` | `...` | `...` | `...` | `REF@SHA256` |

## Umbrales

- Paridad de aceptacion/tests y ausencia de falso verde, perdida causal,
  efecto externo adicional o defecto severo: `PASS|FAIL`.
- Mediana de reduccion tokens/coste >=20% y al menos dos pares mejores: `...`.
- Mediana de tiempo <=110% del control: `...`.
- Reworks, fallos e intervencion humana no aumentan: `...`.
- Continuidad solo con handoff compacto/refs y reproduccion ciega: `...`.

## Evidencia

- Decision JSON: `REF@SHA256`
- Revision independiente/reproduccion: `REF@SHA256`
- Incidencias, intervenciones y defectos posteriores: `REF@SHA256|none`

No se promociono codigo ni se cambio configuracion/default productivo.
