# Clasificacion de retencion S13 - nota de la tarea T9103

Fecha original: 2026-07-10. Reconciliada documentalmente: 2026-07-11.
Ejecutada por el revisor (Claude) fuera del piloto D2, segun el formato de la tarea T9103 de
`docs/backlog_piloto_autonomia_2026-07-10.md`.

Artefacto durable: `docs/clasificacion_retencion_s13_2026-07-10.json`
(validado con parser JSON; 103 artefactos: 3 `retener`, 75 `archivar` y
25 `archivar_condicionado`). La comparacion reproducible esta en
`docs/auditorias/reconciliacion_s13_retencion_ampliada_2026-07-11.json`.

Criterio aplicado (conservador):

- `retener`: documentacion (incidencias) y lo ya archivado en
  `docs/historico/`.
- `archivar`: receipts `orquesta_goal_result_*` (evidencia de cierres, no
  fuente) y cualquier marcador citado por documentacion; se mueven a
  `docs/historico/` en una ola gobernada, nunca se borran directamente.
- `archivar_condicionado`: los 25 marcadores antes etiquetados
  `candidato_borrar` estan catalogados por los manifiestos JSON S13. Diez
  pertenecen tambien al ambito estricto y declaran referencias externas; los
  otros quince solo aparecen en el ambito ampliado. Ninguno se declara libre de
  referencias ni apto para borrar.

En esta tarea NO se borro, movio ni modifico ningun artefacto historico.
La siguiente accion segura es una ola gobernada separada que compruebe
referencias por ruta y prepare un destino trazable antes de archivar. La poda
no esta autorizada por este manifiesto (autorizacion de poda del operador
2026-07-05 no sustituye esa comprobacion).

Reproduccion:

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

Salida esperada: `estricto=68 ampliado=103 interseccion=61
extras_ampliado=42 omitidos_ampliado=7 condicionales=25`.
