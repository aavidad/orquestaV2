# Revision pedagogica, ensamblado y validacion A1

## Alcance

Tema: Gobierno del dato, interoperabilidad semantica, calidad del dato y reutilizacion de la informacion publica.

Rol de esta entrega: revision pedagogica, apoyo al ensamblado final, validacion de palabras y checklist A1. Este material es parcial y trazable para integracion posterior. No sustituye a `tema_a1.md` ni debe copiarse literalmente en el texto visible del tema.

## Estado recibido

En el momento de esta entrega no existe `tema_a1.md` dentro del tema asignado. Por tanto:

- No se puede declarar cumplimiento del minimo A1 de 20.250 palabras.
- No se puede validar sincronizacion entre Markdown, HTML, fuentes, visuales y banco externo.
- No se puede cerrar la politica editorial como cumplida; solo se deja checklist y criterios de revision.

Resultado operativo: entrega bloqueada para cierre final hasta que el agente integrador ensamble el tema completo y ejecute las validaciones.

## Pruebas obligatorias

### validar-palabras-a1-20250-22500

Comando recomendado desde la raiz del tema:

```bash
words=$(wc -w < tema_a1.md)
test "$words" -ge 20250 -a "$words" -le 22500
```

Estado actual: bloqueado porque `tema_a1.md` no existe.

Condicion de aceptacion:

- 20.250 a 22.500 palabras en `tema_a1.md`.
- Si queda por debajo, crear `INFORME_BLOQUEO.md` de nivel tema con palabras alcanzadas, secciones completas, secciones pendientes y pasos exactos de cierre.
- No marcar el tema como listo si no alcanza el minimo.

### validar-politica-editorial-opes-a1

Comprobaciones minimas recomendadas:

```bash
test -f tema_a1.md
test -f tema_a1.html
test -f fuentes.md
test -f checklist_a1.md
test -d subagentes
test -d assets
test -d banco_preguntas_i18n/es
rg -n "^# |^## Indice|^## Orientacion de examen|^## Mapa inicial|^## Fuentes|Notas de test|Supuesto practico|Errores frecuentes|Repaso final" tema_a1.md
```

Estado actual: bloqueado porque los artefactos finales todavia no existen.

## Estructura de ensamblado recomendada

| Bloque | Objetivo pedagogico | Palabras orientativas |
|---|---:|---:|
| Titulo, indice y orientacion de examen | Situar el tema y explicar como estudiarlo | 900 |
| Mapa inicial | Relacionar gobierno, interoperabilidad, calidad y reutilizacion | 850 |
| Definiciones de autoridad | Fijar vocabulario antes del desarrollo complejo | 1.350 |
| Marco normativo y politico | Conectar Ley 40/2015, ENI, ENS, RISP, UE y datos personales | 3.000 |
| Gobierno del dato publico | Roles, ciclo de vida, estrategia, stewardship, catalogos y control | 3.000 |
| Interoperabilidad semantica | Significado comun, metadatos, vocabularios, DCAT, identificadores | 3.000 |
| Calidad del dato | Dimensiones, controles, linaje, metadatos, automatizacion y mejora | 2.750 |
| Reutilizacion de informacion publica | Apertura, licencias, datos de alto valor, API, limites y responsabilidad | 3.100 |
| Integracion aplicada en una administracion | Caso transversal desde inventario hasta publicacion reutilizable | 1.600 |
| Supuestos practicos guiados | Aplicar decisiones a casos de examen | 1.100 |
| Errores frecuentes, repaso y enfoque test | Consolidar discriminadores y trampas reales | 1.150 |

Total orientativo: 21.800 palabras. El integrador debe ajustar la redaccion final para quedar entre 20.250 y 22.500.

## Criterios de revision pedagogica

### Primera lectura

- El desarrollo teorico debe poder leerse sin tablas, test ni visuales.
- Cada apartado debe empezar con definicion o idea matriz antes de introducir normativa o detalle tecnico.
- Los parrafos deben contener una idea principal y cerrar con consecuencia practica o ejemplo cuando el concepto sea abstracto.
- Las listas solo deben aparecer cuando ordenen decisiones, dimensiones o requisitos. Si una lista contiene conceptos de examen, cada punto necesita explicacion breve.

### Modo tutor

Cada concepto nuclear debe poder responder cuatro preguntas:

| Concepto | Que significa | Por que importa | Con que se confunde | Como se reconoce en examen |
|---|---|---|---|---|
| Gobierno del dato | Sistema de responsabilidades, reglas y controles sobre el dato | Evita publicar o explotar datos sin titularidad, calidad o base juridica | Gestion tecnica de bases de datos | Preguntas sobre roles, ciclo de vida, decision y rendicion de cuentas |
| Interoperabilidad semantica | Conservacion del significado compartido al intercambiar datos | Permite que sistemas distintos entiendan lo mismo | Integracion tecnica o conectividad | Preguntas sobre vocabularios, metadatos, codificaciones y modelos comunes |
| Calidad del dato | Grado en que el dato sirve a su finalidad con precision, completitud y consistencia | Sin calidad no hay reutilizacion fiable ni decision publica defendible | Cantidad de datos o formato abierto | Preguntas sobre controles, linaje, validacion, duplicados y actualizacion |
| Reutilizacion | Uso de informacion publica por terceros bajo condiciones juridicas y tecnicas | Genera transparencia, valor economico y servicios derivados | Publicacion informativa no reutilizable | Preguntas sobre licencias, formatos, datos abiertos, limites y datos de alto valor |

### Separacion de teoria y notas de test

- La teoria debe explicar el concepto sin distractores.
- Las notas de test deben ir separadas y marcar trampas solo despues de explicar el contenido.
- Evitar distractores absurdos. Las trampas utiles son confundir interoperabilidad tecnica con semantica, apertura con reutilizacion, seguridad con calidad, catalogo con gobierno y anonimizar con seudonimizar.

## Checklist A1 para el integrador

| Area | Criterio | Estado actual |
|---|---|---|
| Extension | `tema_a1.md` entre 20.250 y 22.500 palabras | Bloqueado |
| Tono | Adulto, tecnico, pedagogico, sin relleno | Pendiente de revisar sobre texto final |
| Indice | Incluye indice navegable y numerado | Pendiente |
| Orientacion de examen | Explica peso, enfoque y forma de estudiar | Pendiente |
| Mapa inicial | Presenta ruta conceptual de 5-7 ideas | Pendiente |
| Definiciones | Define dato, metadato, gobierno, interoperabilidad, calidad, reutilizacion, dato abierto, dato de alto valor | Pendiente |
| Desarrollo teorico | Incluye marco nacional, UE y aplicacion administrativa | Pendiente |
| Tablas | Usa tablas para comparar dimensiones, normas, roles y controles | Pendiente |
| Ejemplos | Incluye ejemplos publicos realistas sin datos personales reales | Pendiente |
| Supuestos | Incluye al menos dos supuestos guiados con solucion separable | Pendiente |
| Notas de test | Separadas de la teoria y con formato diferenciado | Pendiente |
| Errores frecuentes | Incluye errores conceptuales y tecnicos | Pendiente |
| Repaso final | Incluye mapa final, definiciones rapidas y preguntas de recuperacion | Pendiente |
| Visuales | Planifica y referencia assets locales responsivos | Pendiente |
| Fuentes | Prioriza BOE, EUR-Lex, administracion electronica y datos.gob.es | Pendiente |
| Banco i18n | Preguntas externas en `banco_preguntas_i18n/es/` | Pendiente |
| HTML | Patron con primera lectura activa, barra lateral plegable y elementos ocultables | Pendiente |

## Fuentes oficiales que deben quedar cubiertas

No insertar URL visibles en el Markdown o HTML final. En `fuentes.md`, citar editorialmente por organismo, titulo, identificador y fecha de consulta o archivo local.

| Eje | Fuente oficial prioritaria | Uso didactico |
|---|---|---|
| Regimen juridico publico | Ley 40/2015, de Regimen Juridico del Sector Publico, especialmente el articulo 156 | Base del ENI y ENS |
| Procedimiento administrativo | Ley 39/2015, del Procedimiento Administrativo Comun | Derechos, documentos, archivos y actuacion electronica |
| Interoperabilidad nacional | Real Decreto 4/2010, Esquema Nacional de Interoperabilidad | Dimensiones, principios, normas tecnicas e interoperabilidad integral |
| Seguridad | Real Decreto 311/2022, Esquema Nacional de Seguridad | Relacion entre proteccion, trazabilidad, disponibilidad y datos |
| Reutilizacion nacional | Ley 37/2007, sobre reutilizacion de la informacion del sector publico | Condiciones juridicas de reutilizacion |
| Desarrollo RISP estatal | Real Decreto 1495/2011 | Modalidades, obligaciones y reutilizacion en el sector publico estatal |
| Datos abiertos UE | Directiva (UE) 2019/1024 | Datos abiertos, reutilizacion, datos dinamicos y datos de alto valor |
| Datos de alto valor | Reglamento de Ejecucion (UE) 2023/138 | Categorias HVD y obligaciones de publicacion |
| Gobernanza europea de datos | Reglamento (UE) 2022/868 | Intermediacion, altruismo de datos y espacios de datos |
| Acceso y uso de datos | Reglamento (UE) 2023/2854 | Interoperabilidad, acceso y uso justo de datos |
| Interoperabilidad UE | Reglamento (UE) 2024/903, Europa Interoperable | Cooperacion y evaluacion de interoperabilidad transfronteriza |
| Proteccion de datos | Reglamento (UE) 2016/679 y Ley Organica 3/2018 | Limites, licitud, minimizacion, anonimizado y seudonimizado |
| Metadatos y catalogos | Datos.gob.es, DCAT-AP, DCAT-AP-ES y NTI-RISP | Catalogacion, federacion, metadatos y publicacion reutilizable |
| Calidad | Guias oficiales de datos.gob.es sobre calidad de datos abiertos | Dimensiones, controles y mejora de datasets |

## Riesgos de contenido a corregir antes del cierre

- Tratar el gobierno del dato como una herramienta o una oficina aislada, en vez de como sistema de responsabilidades.
- Reducir la interoperabilidad semantica a formatos o APIs.
- Presentar la calidad como atributo absoluto y no como adecuacion a finalidad, uso y contexto.
- Confundir dato abierto con dato publico sin restricciones.
- Omitir limites por proteccion de datos, propiedad intelectual, secreto, seguridad o confidencialidad.
- Meter el banco completo de preguntas dentro del tema en lugar de dejarlo externo.
- Usar visuales decorativos que no expliquen relaciones, ciclos o decisiones.
- Poner URLs reales visibles en el tema final.

## Visuales utiles

| Asset local propuesto | Funcion |
|---|---|
| `assets/mapa_gobierno_interoperabilidad_calidad_reutilizacion.svg` | Mapa causal: gobierno define reglas, interoperabilidad preserva significado, calidad asegura utilidad, reutilizacion genera valor |
| `assets/ciclo_vida_dato_publico.svg` | Captura, validacion, catalogacion, publicacion, uso, feedback y mejora |
| `assets/matriz_interoperabilidad.svg` | Legal, organizativa, semantica y tecnica con ejemplos de cada nivel |
| `assets/flujo_dataset_alto_valor.svg` | De inventario interno a publicacion reutilizable con metadatos y API |

Todos los visuales deben ser locales, responsivos, con texto legible en movil y contenedor con desbordamiento horizontal cuando sea necesario.

## Banco de preguntas externo

El HTML final solo debe incluir muestra progresiva. El banco completo debe vivir en `banco_preguntas_i18n/es/` con estructura localizable.

Cobertura minima recomendada:

- Nivel base: definiciones, dimensiones de interoperabilidad, diferencia dato/dataset/metadato.
- Nivel aplicacion: elegir controles de calidad, catalogar dataset, identificar limites por proteccion de datos.
- Nivel examen real: casos con varias normas aplicables, conflicto entre apertura y restricciones, seleccion de formato/API/licencia.

Cada pregunta debe tener:

- Enunciado.
- Opciones plausibles.
- Respuesta correcta.
- Explicacion de por que es correcta.
- Diagnostico de cada opcion incorrecta.
- Referencia al apartado de repaso.

## Pasos exactos para desbloquear el cierre

1. Ensamblar `tema_a1.md` con la estructura anterior y los materiales del resto de subagentes.
2. Crear `fuentes.md` con referencias oficiales, sin URL visible en el texto final.
3. Crear `assets/` con los SVG didacticos locales y comprobar que el HTML no referencia assets inexistentes.
4. Crear `banco_preguntas_i18n/es/` con banco externo, no incrustado completo en el tema.
5. Generar `tema_a1.html` sincronizado con el Markdown, primera lectura activa, modo tutor y elementos ocultables.
6. Ejecutar conteo de palabras y ajustar hasta 20.250-22.500.
7. Revisar la checklist editorial completa y registrar resultado en `checklist_a1.md`.
8. Si algun punto no cumple, crear `INFORME_BLOQUEO.md` de nivel tema y no declarar listo.
