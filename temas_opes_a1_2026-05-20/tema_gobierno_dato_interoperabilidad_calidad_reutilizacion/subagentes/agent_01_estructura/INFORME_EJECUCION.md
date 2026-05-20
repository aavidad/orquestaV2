# Informe de ejecucion - subentrega de estructura

## Alcance

Se ha producido material parcial para la estructura, indice, mapa conceptual, plan de secciones, trazabilidad de fuentes y checklist de integracion del tema A1 sobre gobierno del dato, interoperabilidad semantica, calidad del dato y reutilizacion de la informacion publica.

No se ha editado `tema_a1.md`, porque la instruccion de esta subentrega limita el trabajo a materiales parciales dentro de `subagentes/`.

## Artefactos producidos

| Artefacto | Contenido |
| --- | --- |
| `01_estructura_indice_mapa.md` | Tesis del tema, indice A1 con pesos de palabras, ruta pedagogica, mapa conceptual textual, diferencias criticas y ubicacion de elementos obligatorios. |
| `02_plan_secciones_integracion.md` | Desarrollo previsto de las 12 secciones, con funcion, subapartados, tablas, ejemplos, notas de test, supuestos y plan de repaso. |
| `03_fuentes_base_y_trazabilidad.md` | Catalogo de fuentes oficiales por identificador editorial, uso por seccion, puntos de actualidad y advertencias de citacion. |
| `04_checklist_integracion_a1.md` | Checklist para ensamblar Markdown/HTML, validar palabras, politica editorial, fuentes, HTML y coherencia por seccion. |
| `assets_sugeridos/mapa_gobierno_dato.svg` | SVG local sugerido para el mapa conceptual del tema. |

## Fuentes verificadas

Se revisaron fuentes oficiales y tecnicas actuales para sostener la estructura:

- BOE: Ley 39/2015, Ley 40/2015, RD 203/2021, RD 4/2010 ENI, NTI-RISP, Ley 37/2007, RDL 24/2021, LOPDGDD y ENS.
- DOUE/EUR-Lex: Directiva (UE) 2019/1024, Reglamento (UE) 2022/868, Reglamento (UE) 2023/2854 y Reglamento de Ejecucion (UE) 2023/138.
- datos.gob.es: NTI-RISP, DCAT-AP, datos de alto valor y recursos formativos.
- Junta de Andalucia: gobierno del dato, marco de referencia de gobierno del dato, portal de datos abiertos y Ley 1/2014.
- W3C: DCAT version 3 como recomendacion tecnica para catalogos de datos.
- AEPD: referencias para anonimiza-cion, seudonimizacion y riesgos de reidentificacion.

## Verificaciones ejecutadas

| Verificacion | Comando | Resultado |
| --- | --- | --- |
| Suma de palabras objetivo | `awk -F'|' '/\\| [0-9]+ \\|/ {gsub(/[ .]/,"",$5); sum+=$5} END {print sum}' 01_estructura_indice_mapa.md` | 21.900 palabras objetivo, dentro del rango A1. |
| Ausencia de menciones internas en material editorial | Busqueda local de identificadores tecnicos, rutas de ejecucion y nombres de runtime dentro de la subentrega. | Sin coincidencias. |
| Cobertura de elementos estructurales obligatorios | `rg -n "Orientacion|Mapa inicial|Conceptos clave|Marco normativo|Gobierno del dato|Interoperabilidad semantica|Calidad del dato|Reutilizacion|Supuestos practicos|Errores frecuentes|Repaso|fuentes" .../agent_01_estructura` | Coincidencias presentes en los materiales. |
| Espacios finales | `rg -n "[[:blank:]]+$" .../agent_01_estructura || true` | Sin coincidencias. |
| SVG valido | `xmllint --noout assets_sugeridos/mapa_gobierno_dato.svg` | Sin errores. |

## Huecos pendientes para el ensamblador

- Redactar `tema_a1.md` completo con 20.250 a 22.500 palabras.
- Crear `tema_a1.html` sincronizado con el Markdown, barra lateral plegable, primera lectura activa, modo tutor y notas de test ocultables.
- Crear `fuentes.md` del tema con fuentes definitivas y fecha de consulta.
- Crear `checklist_a1.md`, `assets/` y `banco_preguntas_i18n/es/` en la raiz del tema.
- Ejecutar la validacion real de palabras sobre `tema_a1.md`.
- Ejecutar la validacion editorial completa sobre el tema final y no marcarlo como listo si no alcanza el minimo aplicable.
