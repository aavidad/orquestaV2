# Reconciliacion S13: auditoria estricta y retencion ampliada

Fecha: 2026-07-11. Alcance: documentacion y manifiestos; no mueve ni borra
artefactos.

## Ambitos

- `ambito_auditoria_estricta_s13_v0`: 68 rutas del inventario gobernado en
  `s13_artefactos_ejecucion_versionados_2026-07-10.json`.
- `ambito_retencion_ampliada_s13_v1`: 103 rutas del manifiesto de retencion
  `clasificacion_retencion_s13_2026-07-10.json`.

No son el mismo conjunto: interseccion `61`, exclusivas del ambito ampliado
`42` y omitidas por el ambito ampliado `7`. El detalle de rutas y el resultado
parseable estan en el JSON homonimo.

## Decision

Los 25 registros antes `candidato_borrar` pasan a `archivar_condicionado`.
Diez tienen cobertura adicional de la auditoria estricta con conteo de
referencias externas; quince solo estan catalogados por el manifiesto ampliado.
En ambos casos no se infiere que esten libres de referencias.

Siguiente paso seguro: abrir una ola gobernada distinta para comprobar cada
ruta, conservar sus referencias y preparar un destino trazable antes de
archivar. No borrar ni mover como consecuencia de esta reconciliacion.

## Reproduccion

```bash
python3 -m json.tool docs/auditorias/s13_artefactos_ejecucion_versionados_2026-07-10.json >/dev/null
python3 -m json.tool docs/clasificacion_retencion_s13_2026-07-10.json >/dev/null
python3 -m json.tool docs/auditorias/reconciliacion_s13_retencion_ampliada_2026-07-11.json >/dev/null
python3 - <<'PY'
import json
from pathlib import Path
a = json.loads(Path('docs/auditorias/s13_artefactos_ejecucion_versionados_2026-07-10.json').read_text())
b = json.loads(Path('docs/clasificacion_retencion_s13_2026-07-10.json').read_text())
A = {x['path'] for x in a['entries']}
B = {x['path'] for x in b['artefactos']}
print(f'estricto={len(A)} ampliado={len(B)} interseccion={len(A & B)} extras_ampliado={len(B - A)} omitidos_ampliado={len(A - B)} condicionales={sum(x["clasificacion"] == "archivar_condicionado" for x in b["artefactos"])}')
PY
```

Salida esperada: `estricto=68 ampliado=103 interseccion=61 extras_ampliado=42
omitidos_ampliado=7 condicionales=25`.
