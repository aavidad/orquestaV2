# Subentrega 01 - estructura, indice y mapa conceptual

## Uso previsto

Este documento entrega una estructura de integracion para el tema A1 "Gobierno del dato, interoperabilidad semantica, calidad del dato y reutilizacion de la informacion publica". No es el tema final. Su funcion es que la persona que ensamble el tema disponga de una arquitectura editorial completa, pesos de palabras, orden pedagogico, mapa conceptual y ubicacion de los elementos obligatorios.

## Tesis del tema

El tema debe defender una idea central: en la Administracion publica el dato no es solo un subproducto de los expedientes, sino un activo institucional que debe gobernarse para que sea fiable, interoperable, seguro, reutilizable y orientado al valor publico. Esa tesis permite unir cuatro bloques que suelen estudiarse por separado:

- Gobierno del dato: decide responsabilidades, reglas, ciclo de vida, catalogos, metadatos, linaje y uso responsable.
- Interoperabilidad semantica: garantiza que el dato intercambiado conserve significado comun entre administraciones, sistemas y reutilizadores.
- Calidad del dato: mide y corrige si el dato sirve para decidir, tramitar, publicar, intercambiar o automatizar sin inducir errores.
- Reutilizacion de la informacion publica: abre datos y documentos del sector publico bajo condiciones juridicas, tecnicas y organizativas que permitan crear valor social, economico y democratico.

La linea narrativa recomendada es: dato fiable -> significado comun -> intercambio seguro -> publicacion reutilizable -> valor publico y control democratico.

## Indice propuesto con pesos de palabras

Objetivo editorial: 21.900 palabras aproximadas. Rango valido A1: 20.250 a 22.500 palabras. Si el ensamblaje queda por debajo de 20.250 palabras, no debe marcarse como listo.

| Num. | Seccion | Funcion dentro del tema | Palabras objetivo |
| --- | --- | --- | ---: |
| 1 | Orientacion de examen y enfoque de estudio | Situar lo que se pregunta, como reconocer trampas y que vocabulario dominar. | 900 |
| 2 | Mapa inicial: del dato administrativo al valor publico | Dar una vista de conjunto antes de entrar en normas y tecnica. | 900 |
| 3 | Conceptos clave y definiciones de autoridad | Fijar definiciones de dato, informacion, documento, metadato, activo semantico, calidad, reutilizacion, dato abierto y dato de alto valor. | 1.600 |
| 4 | Marco normativo y politico aplicable | Ordenar el marco espanol, europeo y andaluz sin convertirlo en lista seca de leyes. | 2.800 |
| 5 | Gobierno del dato en la Administracion publica | Desarrollar principios, roles, procesos, ciclo de vida, catalogo, linaje, stewardship y cultura del dato. | 2.600 |
| 6 | Interoperabilidad semantica y Esquema Nacional de Interoperabilidad | Explicar niveles de interoperabilidad, modelos de datos, vocabularios, codificaciones, metadatos y activos semanticos. | 2.600 |
| 7 | Calidad del dato: dimensiones, controles y mejora continua | Desarrollar exactitud, completitud, consistencia, actualidad, unicidad, trazabilidad, validez y utilidad. | 2.500 |
| 8 | Reutilizacion de la informacion publica y datos abiertos | Conectar RISP, datos abiertos, licencias, condiciones, datos dinamicos, API y conjuntos de alto valor. | 2.500 |
| 9 | Arquitectura operativa: servicios comunes, catalogos, API y espacios de datos | Mostrar como aterrizan los conceptos en portales, inventarios, plataformas de intermediacion, datos de referencia y espacios de datos. | 1.700 |
| 10 | Proteccion de datos, seguridad, etica y limites de la apertura | Integrar RGPD/LOPDGDD, ENS, anonimiza-cion, minimizacion, riesgos y no confusion entre apertura y exposicion. | 1.700 |
| 11 | Supuestos practicos guiados | Aplicar el tema a casos de publicacion, intercambio, calidad y gobierno del dato. | 1.200 |
| 12 | Errores frecuentes, repaso final, notas de test, visuales y fuentes | Cerrar con recuperacion activa, diferencias criticas, muestra test y plan visual. | 900 |
|  | Total |  | 21.900 |

## Ruta pedagogica de primera lectura

1. Empezar por una orientacion de examen breve: que suele confundirse y que verbos debe manejar la persona opositora.
2. Presentar el mapa inicial antes de la normativa para que el lector sepa por que cada norma aparece.
3. Definir conceptos antes de desarrollar. La definicion debe preceder al ejemplo.
4. Explicar el marco normativo por capas, no por cronologia pura: procedimiento y funcionamiento electronico, interoperabilidad, reutilizacion, proteccion de datos y gobierno/espacios de datos.
5. Desarrollar gobierno del dato como disciplina organizativa. No reducirlo a tecnologia.
6. Insertar interoperabilidad semantica como puente entre gobierno y reutilizacion.
7. Tratar la calidad como sistema de controles y evidencias, no como atributo retorico.
8. Cerrar con reutilizacion, porque es la salida publica del ciclo: publicar no basta si no hay metadatos, formato, licencia, API, persistencia y calidad.
9. Separar las notas de test de la teoria. Las trampas deben ir en cajas de nota de test o en el repaso.

## Mapa conceptual textual

El mapa principal debe poder convertirse en SVG local. Nodos y relaciones recomendadas:

| Capa | Nodo | Relacion clave | Riesgo si falla |
| --- | --- | --- | --- |
| Juridica | Ley 39/2015, Ley 40/2015, RD 203/2021 | Funcionamiento electronico y colaboracion administrativa | Expedientes y relaciones digitales inconexas |
| Interoperabilidad | ENI y normas tecnicas | Criterios comunes para intercambio, conservacion y reutilizacion | Sistemas tecnicamente conectados pero semanticamente ambiguos |
| Gobierno | Roles, politicas, catalogos, linaje | Responsabilidad sobre el dato durante todo su ciclo de vida | Datos sin dueno, duplicados o contradictorios |
| Semantica | Modelos de datos, vocabularios, codigos y metadatos | Significado comun para administraciones y aplicaciones | El mismo termino significa cosas distintas |
| Calidad | Reglas, validaciones, monitorizacion y correccion | Dato apto para tramitar, decidir, publicar y reutilizar | Decisiones erroneas y desconfianza |
| Apertura | RISP, datos abiertos, API, licencias, datos de alto valor | Reutilizacion con seguridad juridica y tecnica | Publicacion formal sin uso real |
| Garantias | Proteccion de datos, seguridad, etica y accesibilidad | Limites y salvaguardas del uso del dato | Exposicion indebida, sesgo o perdida de derechos |
| Valor publico | Transparencia, eficiencia, innovacion y mejores servicios | Resultado esperado del ciclo | Dato como coste, no como activo |

## Mapa inicial redactable para el tema

El texto final puede abrir el mapa con esta idea, adaptada y ampliada:

En un expediente administrativo el dato nace ligado a una finalidad concreta: identificar a una persona interesada, acreditar una circunstancia, resolver una ayuda, registrar una actuacion o medir un servicio publico. Pero ese mismo dato puede tener una segunda vida si se gobierna correctamente. Puede evitar que la ciudadania aporte documentos que ya obran en poder de la Administracion, alimentar cuadros de mando, permitir controles de legalidad, favorecer la transparencia, integrarse con otras administraciones y publicarse como dato abierto. La diferencia entre un dato util y un dato peligroso no esta solo en el formato. Esta en saber quien responde de el, que significa, de donde procede, que calidad tiene, que metadatos lo describen, bajo que base juridica se usa y que limites protegen derechos e intereses publicos.

Ese mapa explica por que el tema no debe estudiarse como cuatro epigrafes aislados. El gobierno del dato crea responsabilidad y metodo; la interoperabilidad semantica crea significado comun; la calidad del dato crea confianza; y la reutilizacion de la informacion publica transforma el dato en recurso para terceros. En la Administracion publica, esos cuatro elementos se apoyan en normas comunes, servicios compartidos, catalogos, vocabularios, codigos, API, medidas de seguridad y criterios de proteccion de datos.

## Diferencias criticas que deben aparecer

| Conceptos que se confunden | Diferencia que debe fijarse | Ejemplo de examen |
| --- | --- | --- |
| Transparencia y reutilizacion | La transparencia permite conocer informacion publica; la reutilizacion permite usar documentos o datos para fines distintos, comerciales o no, bajo condiciones. | Pregunta que mezcla derecho de acceso con licencia de reutilizacion. |
| Interoperabilidad tecnica y semantica | La tecnica conecta sistemas; la semantica asegura que el contenido intercambiado conserve significado. | Una API funciona, pero dos organismos interpretan "unidad" de forma distinta. |
| Dato abierto y dato publico | No todo dato publico puede abrirse; deben respetarse limites de proteccion de datos, seguridad, propiedad intelectual y otros intereses protegidos. | Publicar microdatos personales con una disociacion reversible. |
| Calidad de datos y calidad de software | La calidad de datos mide aptitud del dato; la calidad de software mide el producto tecnologico. Se relacionan, pero no son lo mismo. | Un portal estable con datos obsoletos no tiene buena calidad de datos. |
| Datos maestros y datos de referencia | Los maestros identifican entidades nucleares de negocio; los de referencia normalizan valores comunes usados por muchos sistemas. | Padron de entidades frente a codigos de provincia o municipio. |
| Metadatos y datos | Los metadatos describen, contextualizan y hacen localizable el recurso; no sustituyen al dato. | Dataset correcto pero imposible de encontrar por falta de metadatos. |
| Anonimizacion y seudonimizacion | La anonimizacion busca impedir la identificacion; la seudonimizacion reduce riesgos pero sigue dentro del regimen de datos personales si puede reidentificarse. | Reutilizacion de datos sanitarios o sociales. |
| Publicacion por descarga y API | La descarga masiva facilita copia completa; la API facilita acceso automatizado y actualizable. En datos dinamicos y de alto valor pueden ser complementarias. | Obligaciones de datos de alto valor. |

## Ubicacion de elementos obligatorios

| Elemento exigido | Ubicacion recomendada | Observacion editorial |
| --- | --- | --- |
| Indice | Inicio, tras titulo y orientacion breve | Debe coincidir con HTML y banco externo. |
| Orientacion de examen | Seccion 1 | Incluir que se pregunta, trampas y metodo de estudio. |
| Mapa inicial | Seccion 2 y asset SVG | Debe ser lectura, no solo imagen. |
| Definiciones | Seccion 3 y repaso final | Definir antes de explicar. |
| Desarrollo teorico | Secciones 4 a 10 | Parrafos cortos, una idea por parrafo. |
| Tablas markdown | En secciones 3, 4, 6, 7, 8 y 10 | Usarlas para comparar, no para sustituir teoria. |
| Ejemplos | Despues de cada concepto abstracto | Expediente, padron, subvencion, API, catalogo. |
| Supuestos practicos | Seccion 11 | Situacion, pistas, preguntas, resolucion, errores y comprobacion. |
| Notas de test | Bloques separados al final de secciones o seccion 12 | No meter distractores dentro de la teoria. |
| Errores frecuentes | Seccion 12 | Diferencias criticas y falsas equivalencias. |
| Repaso final | Seccion 12 | Mapa de 5 a 7 ideas, definiciones rapidas y preguntas activas. |
| Plan de visuales | Seccion 12 y archivo SVG sugerido | Visuales locales y utiles. |
| Fuentes oficiales | Fichero de fuentes y cierre editorial | Citar por norma/organismo, no con URL visible en tema final. |
