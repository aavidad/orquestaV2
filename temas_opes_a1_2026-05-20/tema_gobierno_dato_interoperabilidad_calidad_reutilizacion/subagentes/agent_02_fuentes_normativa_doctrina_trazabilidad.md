# Agente 02 - Fuentes oficiales, normativa, doctrina y trazabilidad

Fecha de revision: 2026-05-20.

Tema: Gobierno del dato, interoperabilidad semantica, calidad del dato y reutilizacion de la informacion publica.

Rol cubierto: fuentes oficiales, normativa, doctrina tecnica y trazabilidad para integracion posterior por el responsable del tema.

Nota editorial interna: este documento conserva enlaces y referencias de trazabilidad para trabajo de redaccion. En el tema final y en el HTML publicable no deben mostrarse URL reales; alli conviene citar de forma editorial, por ejemplo "BOE, Ley 37/2007, texto consolidado" o "EUR-Lex, Reglamento de Ejecucion (UE) 2023/138". Tampoco debe copiarse esta cabecera ni ninguna mencion a la ejecucion interna.

## 1. Jerarquia recomendada de fuentes

Para un tema A1 conviene ordenar las fuentes de mayor a menor fuerza juridica y de lo general a lo especifico:

| Nivel | Fuente | Uso recomendado en el tema |
| --- | --- | --- |
| 1 | Derecho de la Union Europea directamente aplicable o armonizador | Abrir el marco: datos abiertos, gobernanza europea de datos, interoperabilidad transfronteriza, datos de alto valor, proteccion de datos y mercado unico digital. |
| 2 | Legislacion basica estatal | Explicar derechos, obligaciones y principios comunes de todas las Administraciones publicas espanolas: procedimiento, regimen juridico, funcionamiento electronico, interoperabilidad, reutilizacion, transparencia y proteccion de datos. |
| 3 | Normas tecnicas de interoperabilidad y esquemas nacionales | Bajar a la parte operativa: metadatos, modelos de datos, catalogos, intermediacion, estandares, documento y expediente electronico, seguridad y conservacion. |
| 4 | Normativa autonomica andaluza | Aterrizar el enfoque en Junta de Andalucia: administracion electronica, simplificacion, transparencia, datos abiertos, estrategia digital y gobierno del dato. |
| 5 | Guias oficiales y doctrina institucional | Convertir la norma en criterios pedagogicos: calidad de datos abiertos, DCAT-AP, NTI-RISP, interoperabilidad semantica, APIs y datos de alto valor. |
| 6 | Estandares tecnicos internacionales | Usar como apoyo, no como sustituto de la norma: RDF, DCAT, SKOS, OWL, SHACL, SPARQL, FAIR, ISO/IEC 25012, DAMA. |

Idea de examen: el opositor debe reconocer que el gobierno del dato no es solo una tecnica informatica. Es un sistema de responsabilidades, normas, procesos, vocabularios, calidad, seguridad, proteccion de datos y reutilizacion que permite que la informacion publica sea fiable, interoperable y util.

## 2. Matriz de fuentes prioritarias

### 2.1 Union Europea

| ID | Fuente | Naturaleza | Relevancia para el tema | Como usarla |
| --- | --- | --- | --- | --- |
| UE-01 | Directiva (UE) 2019/1024, sobre datos abiertos y reutilizacion de la informacion del sector publico | Directiva | Marco europeo de reutilizacion, datos dinamicos, formatos abiertos, API, datos de alto valor, investigacion financiada publicamente y principio de reutilizacion con restricciones minimas. | Base del bloque de reutilizacion y datos abiertos. Diferenciar reutilizacion de acceso a la informacion publica. |
| UE-02 | Reglamento de Ejecucion (UE) 2023/138, lista de conjuntos de datos de alto valor | Reglamento de ejecucion | Concreta categorias de datos de alto valor: geoespacial, observacion de la Tierra y medio ambiente, meteorologia, estadistica, sociedades y movilidad. Exige API, descarga masiva cuando proceda, metadatos y licencias abiertas. | Tabla de datos de alto valor y pregunta de examen sobre API/licencia/metadatos. |
| UE-03 | Reglamento (UE) 2024/903, Reglamento sobre la Europa Interoperable | Reglamento | Introduce evaluacion de interoperabilidad para requisitos vinculantes transfronterizos, soluciones interoperables europeas, sandbox de interoperabilidad y gobernanza europea. | Bloque de interoperabilidad legal, organizativa, semantica y tecnica. |
| UE-04 | Reglamento (UE) 2022/868, Reglamento de Gobernanza de Datos | Reglamento | Reutilizacion de categorias protegidas de datos del sector publico, servicios de intermediacion de datos, altruismo de datos y Comite Europeo de Innovacion en materia de Datos. | Bloque de gobierno europeo del dato y limites de reutilizacion. |
| UE-05 | Reglamento (UE) 2023/2854, Data Act | Reglamento | Reglas armonizadas de acceso y uso justo de datos, disponibilidad de datos de productos conectados, puesta a disposicion ante necesidad excepcional del sector publico, cambio entre servicios de tratamiento de datos e interoperabilidad. | Enfoque de economia del dato, no sustituye la normativa RISP. |
| UE-06 | Reglamento (UE) 2018/1724, pasarela digital unica | Reglamento | Procedimientos transfronterizos, principio de solo una vez y sistema tecnico de intercambio automatizado de pruebas. | Ejemplo fuerte de interoperabilidad orientada a servicios publicos. |
| UE-07 | Reglamento (UE) 2016/679, RGPD | Reglamento | Principios de licitud, minimizacion, exactitud, limitacion de finalidad, responsabilidad proactiva, privacidad desde el diseno y proteccion de datos personales. | Bloque de limites: no todo dato publico es dato abierto; anonimizar no es trivial. |
| UE-08 | Directiva 2007/2/CE INSPIRE y normativa de ejecucion | Directiva y reglamentos de ejecucion | Infraestructura europea de informacion espacial, metadatos, interoperabilidad de datos y servicios geograficos. | Ejemplo sectorial: geodatos como caso maduro de interoperabilidad semantica y tecnica. |
| UE-09 | Marco Europeo de Interoperabilidad, revision 2017, y portal Interoperable Europe | Comunicacion y doctrina institucional | Define niveles de interoperabilidad y principios como apertura, reutilizacion, neutralidad tecnologica, portabilidad, preservacion, seguridad, privacidad y multilinguismo. | Marco conceptual para explicar la diferencia entre capas. |

### 2.2 Estado espanol

| ID | Fuente | Naturaleza | Relevancia para el tema | Como usarla |
| --- | --- | --- | --- | --- |
| ES-01 | Constitucion Espanola, articulos 18.4, 103 y 105 b) | Constitucion | Proteccion frente al uso de la informatica, servicio objetivo a los intereses generales, eficacia administrativa y acceso a archivos/registros. | Apertura juridica breve del tema. |
| ES-02 | Ley 39/2015, del Procedimiento Administrativo Comun | Ley basica | Derechos de las personas, relacion electronica, documentos aportados, registros, archivo, copias, expedientes, simplificacion y garantias procedimentales. | Relacionar dato con procedimiento, prueba y expediente. |
| ES-03 | Ley 40/2015, de Regimen Juridico del Sector Publico | Ley basica | Principios de actuacion, relacion electronica entre Administraciones, actuaciones automatizadas, interoperabilidad, transmisiones de datos, ENI, ENS y reutilizacion de sistemas. | Base para interoperabilidad interadministrativa y "solo una vez" nacional. |
| ES-04 | Real Decreto 203/2021, Reglamento de actuacion y funcionamiento del sector publico por medios electronicos | Reglamento estatal | Desarrolla Ley 39/2015 y Ley 40/2015 en funcionamiento electronico, sedes, registros, notificaciones, archivo, interoperabilidad y relaciones interadministrativas. | Conectar normas generales con servicios digitales reales. |
| ES-05 | Real Decreto 4/2010, Esquema Nacional de Interoperabilidad | Real decreto | Marco espanol de interoperabilidad organizativa, semantica y tecnica. Desarrollado por NTI. Texto consolidado con modificacion publicada el 06/11/2024. | Fuente central del bloque de interoperabilidad. |
| ES-06 | Real Decreto 311/2022, Esquema Nacional de Seguridad | Real decreto | Seguridad de la informacion en sector publico, gestion de riesgos, conservacion, vigilancia continua y responsabilidades. | Fuente de limites: gobierno del dato fiable exige seguridad. |
| ES-07 | Ley 37/2007, sobre reutilizacion de la informacion del sector publico | Ley basica | Regimen juridico de reutilizacion, documentos reutilizables, condiciones, tarifas, formatos, licencias y limites. Texto modificado por RDL 24/2021 para transponer la Directiva 2019/1024. | Base espanola de RISP. |
| ES-08 | Real Decreto-ley 24/2021, Libro tercero | Real decreto-ley | Transpone la Directiva 2019/1024 modificando la Ley 37/2007: datos abiertos, datos de alto valor, datos dinamicos, empresas publicas y datos de investigacion. | Actualizar la explicacion historica de RISP. |
| ES-09 | Real Decreto 1495/2011 | Real decreto | Desarrollo de Ley 37/2007 para el sector publico estatal: modalidades, condiciones, documentos con propiedad intelectual o datos personales. | Marco estatal operativo para RISP. |
| ES-10 | Ley 19/2013, de transparencia, acceso a la informacion publica y buen gobierno | Ley basica | Publicidad activa, derecho de acceso, concepto de informacion publica y principio de reutilizacion de informacion publicada. | Diferenciar transparencia, acceso y reutilizacion. |
| ES-11 | Ley Organica 3/2018, de Proteccion de Datos Personales y garantia de los derechos digitales | Ley organica | Adaptacion nacional al RGPD, garantias de derechos digitales, tratamientos por Administraciones y regimen de proteccion. | Limites juridicos de apertura y cruce de datos. |
| ES-12 | Ley 14/2010, infraestructuras y servicios de informacion geografica en Espana | Ley | Transpone INSPIRE y regula geodatos, metadatos, servicios interoperables y puesta en comun. | Ejemplo sectorial de datos de alto valor e interoperabilidad. |
| ES-13 | Ley 12/1989, de la Funcion Estadistica Publica | Ley | Interes para codificaciones estadisticas, secreto estadistico, funcion estadistica y calidad/normalizacion de conceptos. | Apoyo en datos estadisticos y definiciones comunes. |

### 2.3 Normas tecnicas de interoperabilidad y guias PAe/datos.gob.es

| ID | Fuente | Naturaleza | Relevancia | Uso |
| --- | --- | --- | --- | --- |
| NTI-01 | NTI de Relacion de modelos de datos, Resolucion de 28 de junio de 2012 | Norma tecnica | Condiciones para establecer y publicar modelos de datos comunes y modelos sujetos a intercambio. Conecta con el Centro de Interoperabilidad Semantica. | Fuente principal de interoperabilidad semantica en Espana. |
| NTI-02 | NTI de Protocolos de intermediacion de datos, Resolucion de 28 de junio de 2012 | Norma tecnica | Especificaciones para intercambio intermediado de datos entre Administraciones y otros nodos/plataformas. | Explicar "no pedir al ciudadano lo que puede consultarse". |
| NTI-03 | NTI de Reutilizacion de recursos de la informacion, Resolucion de 19 de febrero de 2013 | Norma tecnica | Seleccion, identificacion, descripcion, formato, condiciones de uso y puesta a disposicion de recursos reutilizables. | Base de catalogos de datos, metadatos y URI. |
| NTI-04 | NTI de Catalogo de estandares, Resolucion de 3 de octubre de 2012 | Norma tecnica | Estandares aplicables para garantizar independencia tecnologica e interoperabilidad. | Fuente para hablar de formatos abiertos y estandares. |
| NTI-05 | NTI de Documento electronico, Expediente electronico, Digitalizacion, Copiado autentico y Gestion documental | Normas tecnicas | Metadatos, integridad, identificacion, conversion, expediente y gestion documental. | No convertir el tema en documental, pero usar como soporte de calidad y trazabilidad. |
| GUIA-01 | Guia de aplicacion de la NTI de Reutilizacion de recursos de informacion | Guia oficial | Explica aplicacion practica de NTI-RISP, politicas RISP, metadatos, catalogos y condiciones. | Material pedagogico para ejemplos. |
| GUIA-02 | DCAT-AP y NTI-RISP, datos.gob.es | Material formativo oficial | Relacion entre DCAT-AP europeo y NTI-RISP espanola; catalogos y metadatos semanticos. | Tabla comparativa de metadatos y catalogos. |
| GUIA-03 | Guia practica para la mejora de la calidad de datos abiertos, datos.gob.es | Guia oficial | Calidad de datos abiertos; dimensiones como exactitud, completitud, consistencia, actualidad, trazabilidad, comprensibilidad y disponibilidad. | Bloque de calidad. |
| GUIA-04 | Guia practica para migrar a DCAT-AP-ES, datos.gob.es, 2025/2026 | Guia oficial | Migracion hacia DCAT-AP-ES, alineacion con Directiva 2019/1024 y Reglamento 2023/138. Indica que la nueva NTI-RISP estaba en tramitacion en la fecha de publicacion de la guia. | Usarla como doctrina reciente, no como norma ya vigente si no se verifica su aprobacion. |
| GUIA-05 | Decalogo del reutilizador de datos del sector publico, edicion 2025 publicada en 2026 | Guia oficial | Reutilizacion responsable en contexto de economia del dato, IA, licencias, calidad, interoperabilidad y persistencia. | Cierre practico del bloque RISP. |

### 2.4 Andalucia

| ID | Fuente | Naturaleza | Relevancia | Uso |
| --- | --- | --- | --- | --- |
| AND-01 | Ley 1/2014, de Transparencia Publica de Andalucia | Ley autonomica | Publicidad activa, derecho de acceso autonomico, reutilizacion, Consejo de Transparencia y Proteccion de Datos de Andalucia. | Aterrizaje autonomico en transparencia y datos abiertos. |
| AND-02 | Decreto 622/2019, de administracion electronica, simplificacion de procedimientos y racionalizacion organizativa de la Junta de Andalucia | Decreto autonomico | Principios de administracion electronica, sede, portal, registro, notificaciones, gestion documental, comprobacion de datos y documentos, automatizacion. | Ejemplo andaluz transversal y muy examinable. |
| AND-03 | Portal de Datos Abiertos de la Junta de Andalucia | Portal oficial | Catalogo centralizado de conjuntos de datos abiertos del sector publico andaluz, con categorias y formatos reutilizables. | Caso real de RISP autonomico. |
| AND-04 | Pagina "Gobierno del dato" de la Agencia Digital de Andalucia | Doctrina institucional | Define el gobierno del dato como garantia de fuentes autorizadas, consistencia, interoperabilidad y difusion como datos abiertos cuando proceda. | Definicion institucional local del tema. |
| AND-05 | Estrategia Andaluza de Administracion Digital centrada en las personas 2030 | Plan/estrategia | Aprobada por Acuerdo de 8 de abril de 2026; orienta la transformacion digital de servicios publicos andaluces centrada en ciudadania. | Contexto actual, no sustituye a leyes. |
| AND-06 | Plan Plurianual de Actuacion 2025-2030 de la Agencia Digital de Andalucia | Plan estrategico | Busca gobierno 100% digital, administracion integrada, IA, simplificacion, servicios centrados en personas y objetivos estrategicos. | Contexto de politica publica. |

### 2.5 Estandares y doctrina tecnica de apoyo

| ID | Fuente | Naturaleza | Relevancia | Uso |
| --- | --- | --- | --- | --- |
| TEC-01 | W3C DCAT 3, Recomendacion de 22 de agosto de 2024 | Estandar web | Vocabulario para catalogos de datos, datasets, servicios de datos y distribuciones. | Explicar por que DCAT-AP se basa en DCAT. |
| TEC-02 | W3C RDF 1.1 Concepts | Estandar web | Modelo de grafos y tripletas para representar informacion en la web semantica. | Definir RDF sin entrar en excesivo detalle tecnico. |
| TEC-03 | W3C SKOS | Estandar web | Sistemas de organizacion del conocimiento: tesauros, esquemas de conceptos y relaciones semanticas. | Ejemplo de vocabularios controlados. |
| TEC-04 | W3C OWL 2 | Estandar web | Ontologias y definiciones formales de conceptos y relaciones. | Diferenciar vocabulario, taxonomia y ontologia. |
| TEC-05 | W3C SHACL | Estandar web | Validacion de grafos RDF mediante formas y restricciones. | Conectar calidad semantica con validacion automatica. |
| TEC-06 | SPARQL 1.1 | Estandar web | Consulta y manipulacion de grafos RDF. | Ejemplo de explotacion tecnica de datos enlazados. |
| TEC-07 | ISO/IEC 25012 e ISO/IEC 25024 | Normas ISO | Modelo de calidad de datos y medicion; citadas por guias oficiales de calidad. | Usar solo como apoyo, sin reproducir norma cerrada. |
| TEC-08 | FAIR principles | Principios de gestion de datos | Encontrables, accesibles, interoperables y reutilizables. | Util para relacionar metadatos, persistencia y reutilizacion. |
| TEC-09 | DAMA-DMBOK | Marco profesional | Gobierno, arquitectura, calidad, metadatos, datos maestros y seguridad. | Apoyo doctrinal, no fuente juridica. |

## 3. Definiciones de autoridad para integrar

Estas definiciones estan redactadas para ser integradas o adaptadas. Conviene mantener el tono explicativo y anadir despues ejemplos de Administracion publica.

### 3.1 Gobierno del dato

Gobierno del dato es el conjunto de normas, roles, decisiones, procesos y controles que aseguran que los datos de una organizacion publica son gestionados como un activo institucional. En una Administracion no basta con almacenar datos: debe saberse quien es responsable del dato, cual es la fuente autorizada, con que definicion se usa, que calidad tiene, quien puede acceder, durante cuanto tiempo se conserva, como se comparte y en que condiciones puede abrirse o reutilizarse.

Trazabilidad: AND-04, GUIA-03, TEC-09, RGPD y Ley 40/2015. La pagina de Gobierno del dato de la Agencia Digital de Andalucia ofrece una formulacion util: fuentes autorizadas, consistencia entre escenarios, interoperabilidad y difusion como datos abiertos cuando proceda.

Modo tutor:
- Que significa: pasar de "tener bases de datos" a gobernar informacion con responsabilidades y criterios comunes.
- Por que importa: sin gobierno del dato hay duplicidades, contradicciones, errores de tramitacion, mala interoperabilidad y poca confianza.
- Con que se confunde: con analitica, big data o una plataforma tecnologica. La tecnologia ayuda, pero no sustituye el marco de decision.
- Como se reconoce en examen: aparecen palabras como roles, fuente autorizada, ciclo de vida, calidad, metadatos, catalogo, linaje, seguridad, proteccion de datos y reutilizacion.

### 3.2 Dato, informacion, documento y recurso de informacion

Dato es una representacion de un hecho, atributo, medida o valor. Informacion es dato contextualizado que permite interpretar una situacion. Documento, en el marco de reutilizacion, comprende una representacion de actos, hechos o informacion y recopilaciones de estos, cualquiera que sea su soporte, siempre que entren en el ambito de la norma. Recurso de informacion es la unidad que se publica o gestiona para su reutilizacion, normalmente descrita por metadatos y ofrecida mediante distribuciones, servicios o API.

Trazabilidad: Ley 37/2007, NTI-RISP, DCAT, RD 203/2021 y normas de documento/expediente electronico.

Ejemplo: una tabla CSV con municipios y poblacion contiene datos; una ficha que explica fecha, fuente, periodicidad y metodologia aporta informacion; el expediente administrativo donde esos datos justifican una resolucion pertenece al ambito documental; el conjunto publicado en un catalogo de datos abiertos es un recurso reutilizable.

### 3.3 Informacion publica, acceso y reutilizacion

Informacion publica es la que obra en poder de sujetos obligados y ha sido elaborada o adquirida en el ejercicio de sus funciones. Acceso a la informacion publica es el derecho de conocer esa informacion conforme a la normativa de transparencia. Reutilizacion es el uso posterior de documentos o datos del sector publico por personas fisicas o juridicas, con fines comerciales o no comerciales, distinto del uso inicial para el que se produjeron en la funcion publica.

Trazabilidad: Ley 19/2013, Ley 1/2014 de Andalucia, Ley 37/2007 y Directiva 2019/1024.

Error frecuente: creer que todo lo accesible es automaticamente reutilizable sin condiciones. El acceso puede estar limitado por proteccion de datos, seguridad, propiedad intelectual, secreto estadistico, confidencialidad comercial o intereses publicos protegidos. La reutilizacion exige revisar licencias, formato, calidad y condiciones.

### 3.4 Datos abiertos

Datos abiertos son datos puestos a disposicion para que puedan ser consultados, usados, reutilizados y redistribuidos con restricciones minimas. En el sector publico no basta con publicar un PDF: el objetivo de datos abiertos exige formatos abiertos o legibles por maquina cuando proceda, metadatos, licencias claras, persistencia, actualizacion y calidad suficiente.

Trazabilidad: Directiva 2019/1024, Ley 37/2007, NTI-RISP, datos.gob.es y Reglamento 2023/138.

Ejemplo: un listado de centros administrativos publicado en PDF puede servir para lectura humana, pero un CSV, JSON o API con identificadores, coordenadas, fecha de actualizacion, licencia y metadatos favorece reutilizacion automatica.

### 3.5 Interoperabilidad

Interoperabilidad es la capacidad de organizaciones y sistemas para trabajar juntos compartiendo informacion y conocimiento mediante procesos y sistemas de informacion. En Administracion publica debe entenderse por capas: juridica, organizativa, semantica y tecnica. La capa juridica evita conflictos normativos; la organizativa alinea procesos y responsabilidades; la semantica asegura que los datos significan lo mismo; la tecnica permite el intercambio mediante estandares, protocolos, servicios y formatos.

Trazabilidad: EIF, Reglamento 2024/903, RD 4/2010, Ley 40/2015 y RD 203/2021.

Tabla integrable:

| Capa | Pregunta clave | Ejemplo publico |
| --- | --- | --- |
| Juridica | Podemos compartir este dato con base legal suficiente? | Consulta de residencia entre Administraciones para resolver una ayuda. |
| Organizativa | Quien pide, quien responde y en que proceso? | Convenio, plataforma de intermediacion o procedimiento definido. |
| Semantica | Significa lo mismo "domicilio", "unidad familiar" o "centro" para todos? | Modelo de datos comun y codificaciones normalizadas. |
| Tecnica | Como se intercambia de forma segura y automatizable? | API, servicio web, XML, JSON, RDF, certificados, red administrativa. |

### 3.6 Interoperabilidad semantica

Interoperabilidad semantica es la capacidad de intercambiar datos preservando su significado. Requiere modelos de datos, definiciones compartidas, vocabularios controlados, codificaciones, identificadores persistentes y metadatos. No se reduce a que dos sistemas puedan conectarse; exige que interpreten de forma coherente lo intercambiado.

Trazabilidad: RD 4/2010, NTI de Relacion de modelos de datos, Centro de Interoperabilidad Semantica, EIF, SEMIC, DCAT-AP, RDF, SKOS y OWL.

Ejemplo de examen: si una Administracion codifica sexo, municipio, tipo de via o clase de procedimiento de una forma distinta a otra, puede haber intercambio tecnico pero no verdadera interoperabilidad semantica. La solucion pasa por modelos comunes, codigos oficiales y metadatos, no solo por cambiar el conector.

### 3.7 Metadatos

Metadatos son datos que describen otros datos o recursos: titulo, descripcion, responsable, fecha de creacion, fecha de actualizacion, cobertura territorial, periodicidad, formato, licencia, calidad, vocabularios usados, API y condiciones de acceso. En datos abiertos, los metadatos son decisivos porque permiten encontrar, interpretar, comparar y reutilizar conjuntos de datos.

Trazabilidad: NTI-RISP, DCAT, DCAT-AP, Reglamento 2023/138, Ley 14/2010 para geodatos e INSPIRE.

Ejemplo: si un conjunto "Contratos menores" no indica periodo, organo responsable, definicion de importe, licencia y frecuencia de actualizacion, el reutilizador puede descargarlo pero no usarlo con confianza.

### 3.8 Calidad del dato

Calidad del dato es el grado en que un dato cumple los requisitos necesarios para el uso previsto. En sector publico deben considerarse dimensiones como exactitud, completitud, consistencia, actualidad, trazabilidad, comprensibilidad, disponibilidad, accesibilidad, conformidad y confidencialidad. La calidad no se arregla solo al publicar: debe gestionarse en origen, durante el ciclo de vida y en la explotacion.

Trazabilidad: Guia practica para la mejora de la calidad de datos abiertos, ISO/IEC 25012, ISO/IEC 25024, RGPD, ENS, NTI-RISP.

Tabla integrable:

| Dimension | Pregunta de control | Riesgo si falla |
| --- | --- | --- |
| Exactitud | El dato representa correctamente la realidad administrativa? | Resoluciones erroneas, estadisticas falsas, perdida de confianza. |
| Completitud | Faltan valores necesarios para el uso previsto? | Analisis sesgado o imposibilidad de automatizar. |
| Consistencia | El mismo dato coincide entre sistemas y periodos? | Contradicciones entre portales, expedientes y cuadros de mando. |
| Actualidad | Esta actualizado con la frecuencia declarada? | Decisiones basadas en informacion obsoleta. |
| Trazabilidad | Puede conocerse origen, transformacion y responsable? | No se puede auditar ni corregir el error. |
| Comprensibilidad | Estan explicadas variables, codigos y unidades? | Reutilizacion defectuosa por interpretacion incorrecta. |
| Disponibilidad | El dato y su servicio estan accesibles cuando se necesitan? | Fallos en servicios digitales y APIs. |
| Conformidad | Cumple norma, formato, metadatos y licencia? | Riesgo juridico y falta de interoperabilidad. |

### 3.9 Datos de alto valor

Los datos de alto valor son conjuntos cuya reutilizacion puede aportar beneficios socioeconomicos, medioambientales o de innovacion relevantes. La Union Europea los ha concretado en seis categorias: geoespacial, observacion de la Tierra y medio ambiente, meteorologia, estadistica, sociedades y propiedad de sociedades, y movilidad. Deben publicarse con condiciones armonizadas, formatos legibles por maquina, API adecuadas, descarga masiva cuando proceda y licencias abiertas.

Trazabilidad: Directiva 2019/1024 y Reglamento de Ejecucion 2023/138.

Nota de test: las seis categorias son muy preguntables. Conviene memorizarlas por bloques: territorio, medio ambiente, tiempo, estadistica, empresas y desplazamientos.

### 3.10 API y descarga masiva

Una API permite acceso automatizado, selectivo y actualizable a datos o servicios. La descarga masiva permite obtener un conjunto completo o una parte amplia para tratamiento local. En datos de alto valor, el Reglamento 2023/138 exige API que respondan a necesidades razonables de reutilizadores y, cuando se indique, tambien descarga masiva. Las condiciones de uso y criterios de calidad del servicio deben estar publicados.

Trazabilidad: Reglamento 2023/138, Directiva 2019/1024, NTI-RISP y datos.gob.es.

Ejemplo: para datos meteorologicos o movilidad, una API permite consultar datos actualizados; para un inventario historico, la descarga masiva facilita analisis completo.

### 3.11 Datos personales, anonimización y reutilizacion

La apertura o reutilizacion no elimina las obligaciones de proteccion de datos. Si un conjunto contiene datos personales o permite reidentificacion al combinarse con otras fuentes, debe aplicarse el RGPD y la LOPDGDD. La anonimizacion debe ser robusta; la seudonimizacion reduce riesgo pero no convierte automaticamente los datos en anonimos. Deben aplicarse minimizacion, limitacion de finalidad, seguridad, evaluacion de impacto cuando proceda y responsabilidad proactiva.

Trazabilidad: RGPD, LOPDGDD, Directiva 2019/1024, Reglamento 2023/138, ENS.

Error frecuente: "quitar el nombre" no siempre anonimiza. Codigo postal, fecha de nacimiento, sexo, centro, tramo temporal u otros atributos pueden permitir reidentificacion si se combinan con fuentes externas.

### 3.12 Datos maestros, datos de referencia y codificaciones

Datos maestros son entidades nucleares de una organizacion: personas, centros, organos, expedientes, procedimientos, servicios, direcciones o proveedores. Datos de referencia son listas y codigos normalizados usados para clasificar: municipios, paises, tipos de via, organos, unidades, categorias o estados. Ambos son clave para interoperabilidad y calidad porque reducen duplicidades y ambiguedades.

Trazabilidad: NTI de Relacion de modelos de datos, DIR3, INE, Ley 12/1989, DCAT/SKOS.

Ejemplo: si cada consejeria nombra un mismo centro de forma distinta, el cuadro de mando sera inconsistente. Un identificador comun y una fuente autorizada solucionan la divergencia.

## 4. Desarrollo teorico sugerido por bloques

### Bloque A. Del expediente al ecosistema de datos

Punto de partida: la Administracion siempre ha producido documentos y datos. La transformacion digital cambia la escala, velocidad y posibilidad de reutilizacion. El dato deja de ser un residuo de la tramitacion y pasa a ser un activo para derechos, simplificacion, analitica, transparencia, automatizacion y mejora de politicas publicas.

Normativa: Ley 39/2015, Ley 40/2015, RD 203/2021, Decreto 622/2019 andaluz.

Desarrollo recomendado:
1. Procedimiento electronico y expediente generan datos estructurados.
2. La actuacion administrativa automatizada exige datos fiables.
3. Las relaciones entre Administraciones requieren intercambio electronico.
4. El ciudadano no debe soportar cargas documentales innecesarias si la Administracion puede comprobar datos conforme a Derecho.
5. La reutilizacion transforma informacion publica en valor social y economico.

Ejemplo: una subvencion puede requerir identidad, domicilio, renta, discapacidad, situacion laboral y cuenta bancaria. Sin gobierno del dato, cada dato se pide, se copia y se interpreta manualmente. Con gobierno e interoperabilidad, se consulta en fuentes autorizadas, se registra trazabilidad, se minimiza documentacion y se conserva evidencia.

### Bloque B. Gobierno del dato en la Administracion publica

Ejes para explicar:

| Eje | Pregunta | Salida esperada |
| --- | --- | --- |
| Responsabilidad | Quien es dueno funcional y custodio tecnico del dato? | Roles, responsables y comites. |
| Definicion | Que significa cada dato? | Glosario, diccionario de datos, modelo semantico. |
| Fuente autorizada | Cual es el origen valido? | Registro maestro o sistema fuente. |
| Calidad | Como se mide y corrige? | Reglas, indicadores, validaciones y mejora continua. |
| Seguridad | Quien puede acceder y bajo que condiciones? | Controles, perfiles, auditoria, ENS. |
| Proteccion de datos | Hay datos personales o riesgo de reidentificacion? | Base juridica, minimizacion, EIPD, medidas. |
| Interoperabilidad | Como se comparte sin perder significado? | Modelos, codigos, APIs, estandares. |
| Reutilizacion | Puede abrirse o licenciarse? | Catalogo, metadatos, condiciones, API. |

Trazabilidad: AND-04, GUIA-03, ES-06, UE-07, NTI-01, NTI-03.

### Bloque C. Interoperabilidad: no solo tecnologia

Desarrollo recomendado:
- Interoperabilidad juridica: bases legales, competencias, limites, convenios y obligaciones.
- Interoperabilidad organizativa: procesos, acuerdos de nivel de servicio, responsables y gobierno.
- Interoperabilidad semantica: modelos de datos, vocabularios, codificaciones y metadatos.
- Interoperabilidad tecnica: APIs, servicios, formatos, seguridad, redes y certificados.

Punto fino de examen: el ENI espanol y el EIF europeo coinciden en que la interoperabilidad debe atender varias dimensiones. Si un caso practico solo habla de "conectar sistemas", la respuesta completa debe anadir significado, proceso, base legal, seguridad y calidad.

Trazabilidad: RD 4/2010, Reglamento 2024/903, EIF, Ley 40/2015, RD 203/2021.

### Bloque D. Interoperabilidad semantica y modelos de datos

Subideas:
1. El dato tiene significado dentro de un modelo.
2. El modelo debe publicar definiciones, codigos, identificadores, relaciones y versiones.
3. Los vocabularios controlados evitan sinonimos y ambiguedades.
4. Las ontologias y grafos ayudan en dominios complejos.
5. DCAT y DCAT-AP describen catalogos y datasets; no describen todo el dominio sustantivo, pero permiten federar y encontrar datos.
6. La validacion semantica puede automatizarse con reglas o lenguajes como SHACL.

Ejemplo: "centro educativo" puede referirse al edificio, al codigo administrativo, a la entidad juridica, a la sede fisica o a una unidad organizativa. El modelo de datos debe precisar que entidad se describe y con que identificador.

Trazabilidad: NTI-01, SEMIC, DCAT, SKOS, OWL, SHACL.

### Bloque E. Calidad del dato como condicion de confianza

Desarrollo recomendado:
- La calidad se define segun el uso: no hay calidad abstracta si no se sabe para que se necesita el dato.
- En Administracion hay usos de alto impacto: resolver derechos, pagar prestaciones, imponer sanciones, elaborar estadisticas, abrir datos y entrenar modelos.
- La calidad debe gestionarse desde la captura: formularios, validaciones, fuentes autorizadas, codigos, metadatos, controles y auditoria.
- La calidad tiene dimension juridica: exactitud en RGPD, responsabilidad proactiva, seguridad, transparencia y rendicion de cuentas.
- Los datos abiertos de baja calidad reducen reutilizacion y pueden inducir errores sociales o economicos.

Trazabilidad: GUIA-03, RGPD, ENS, Ley 37/2007, Reglamento 2023/138.

Mini tabla de problemas:

| Problema | Ejemplo | Correccion |
| --- | --- | --- |
| Duplicidad | Un mismo beneficiario aparece con variantes de nombre. | Identificador maestro y reglas de deduplicacion. |
| Formato heterogeneo | Fechas en dd/mm/aaaa y aaaa-mm-dd. | Estandar unico y validacion. |
| Codificacion local | Municipios escritos a mano. | Codigo INE y catalogo de referencia. |
| Falta de linaje | No se sabe de que sistema procede el dato. | Metadatos de origen y transformacion. |
| Obsolescencia | Dataset indica actualizacion mensual pero lleva un ano sin cambios. | Politica de actualizacion y alerta. |
| Riesgo de privacidad | Dataset granular permite reidentificar. | Agregacion, supresion, anonimizacion robusta y evaluacion. |

### Bloque F. Reutilizacion de la informacion publica

Desarrollo recomendado:
1. Finalidad: aprovechar informacion generada por el sector publico para transparencia, innovacion, investigacion, actividad economica y mejora social.
2. Diferenciar acceso, publicidad activa y reutilizacion.
3. Identificar limites: datos personales, propiedad intelectual, seguridad, confidencialidad, secretos, derechos de terceros.
4. Explicar condiciones: licencias, tarifas excepcionales, formatos, metadatos, medios electronicos, no discriminacion y ausencia de exclusividades salvo excepcion.
5. Datos dinamicos y datos de alto valor: API y actualizacion.
6. Papel de catalogos: datos.gob.es, portales autonomicos, DCAT-AP y federacion europea.

Trazabilidad: Ley 37/2007, RDL 24/2021, Directiva 2019/1024, Reglamento 2023/138, NTI-RISP.

### Bloque G. Andalucia: gobierno del dato, datos abiertos y administracion digital

Desarrollo recomendado:
- Ley 1/2014: transparencia autonomica, publicidad activa, derecho de acceso y reutilizacion.
- Decreto 622/2019: principios de administracion electronica andaluza, portal, sede, registro, notificaciones, gestion documental, comprobacion de datos y automatizacion.
- Agencia Digital de Andalucia: gobierno del dato, fuentes autorizadas, consistencia e interoperabilidad.
- Portal de Datos Abiertos: caso real de catalogo centralizado.
- Estrategia Andaluza de Administracion Digital 2030 y Plan Plurianual ADA 2025-2030: contexto estrategico actual.

Precaucion: no convertir planes estrategicos en obligaciones juridicas equivalentes a una ley. Sirven para explicar direccion politica y organizativa, no para sustituir el marco normativo.

## 5. Tablas listas para adaptacion al tema

### 5.1 Transparencia, acceso, reutilizacion y gobierno del dato

| Concepto | Pregunta que responde | Norma principal | Producto esperado |
| --- | --- | --- | --- |
| Transparencia activa | Que debe publicar la Administracion por iniciativa propia? | Ley 19/2013 y Ley 1/2014 Andalucia | Portal de transparencia, informacion institucional, economica, normativa y organizativa. |
| Derecho de acceso | Puede una persona solicitar informacion publica concreta? | Ley 19/2013 y normativa autonomica | Resolucion de acceso, con limites y ponderacion. |
| Reutilizacion | Puede usarse la informacion para crear nuevos productos, servicios o analisis? | Ley 37/2007 y Directiva 2019/1024 | Datos con condiciones de uso, licencia, formato y metadatos. |
| Gobierno del dato | Como se asegura que el dato es fiable, definido, protegido e interoperable? | Ley 40/2015, ENI, ENS, RGPD y doctrina institucional | Roles, catalogo, calidad, metadatos, seguridad, linaje y fuentes autorizadas. |

### 5.2 Normativa RISP en secuencia historica

| Hito | Aporte |
| --- | --- |
| Directiva 2003/98/CE | Primer marco europeo de reutilizacion de informacion del sector publico. |
| Ley 37/2007 | Transposicion espanola inicial y regimen basico de reutilizacion. |
| Real Decreto 1495/2011 | Desarrollo para el sector publico estatal. |
| Ley 18/2015 | Adaptacion a la Directiva 2013/37/UE y ampliacion del enfoque de reutilizacion. |
| Directiva (UE) 2019/1024 | Recast europeo: datos abiertos, datos dinamicos, alto valor, investigacion y empresas publicas. |
| Real Decreto-ley 24/2021 | Modificacion de Ley 37/2007 para transponer la Directiva 2019/1024. |
| Reglamento de Ejecucion (UE) 2023/138 | Lista y condiciones de publicacion de datos de alto valor. |

### 5.3 Datos de alto valor

| Categoria UE | Ejemplos posibles | Cautelas |
| --- | --- | --- |
| Geoespacial | Direcciones, unidades administrativas, parcelas, edificios, redes. | Coordinar con INSPIRE, calidad posicional y metadatos. |
| Observacion de la Tierra y medio ambiente | Calidad del aire, agua, emisiones, habitats. | Datos tecnicos con unidades y series temporales claras. |
| Meteorologia | Observaciones, predicciones, alertas, series historicas. | API y frecuencia de actualizacion son esenciales. |
| Estadistica | Indicadores oficiales, poblacion, economia, empleo. | Secreto estadistico, metodologia y versionado. |
| Sociedades y propiedad de sociedades | Registros mercantiles, identificadores, datos societarios. | Limites juridicos, exactitud e identidad. |
| Movilidad | Transporte publico, redes, horarios, trafico. | Datos dinamicos, estandares sectoriales y disponibilidad. |

### 5.4 Semantica: vocabulario, taxonomia, ontologia y modelo de datos

| Elemento | Que aporta | Ejemplo |
| --- | --- | --- |
| Glosario | Definiciones legibles por personas | "Dato de alto valor", "metadato", "licencia abierta". |
| Vocabulario controlado | Terminos permitidos y relaciones simples | Tipos de procedimiento o estados de tramitacion. |
| Taxonomia | Clasificacion jerarquica | Categorias de datasets por sector. |
| Ontologia | Conceptos y relaciones formalizadas | Personas, organos, expedientes, actos y competencias. |
| Modelo de datos | Estructura de entidades, atributos, relaciones y restricciones | Dataset de subvenciones con beneficiario, organo, importe, finalidad y fecha. |

## 6. Supuestos practicos guiados

### Supuesto 1. Apertura de datos de ayudas publicas

Situacion: una consejeria quiere publicar un dataset de ayudas concedidas para fomentar transparencia y reutilizacion.

Pistas normativas:
- Transparencia: Ley 19/2013 y Ley 1/2014.
- Reutilizacion: Ley 37/2007, NTI-RISP y Directiva 2019/1024.
- Proteccion de datos: RGPD y LOPDGDD.
- Calidad: guia de calidad de datos abiertos.
- Metadatos: DCAT-AP/NTI-RISP.

Resolucion:
1. Identificar finalidad de publicacion y base juridica.
2. Determinar si hay datos personales de beneficiarios y aplicar minimizacion o agregacion cuando proceda.
3. Definir campos: organo, convocatoria, beneficiario si legalmente publicable, importe, fecha, finalidad, municipio, estado.
4. Usar codigos normalizados para organos, municipios, procedimientos y sectores.
5. Publicar con metadatos: titulo, descripcion, responsable, fecha, periodicidad, licencia, cobertura, formato y API si procede.
6. Establecer control de calidad: exactitud, completitud, consistencia, trazabilidad y actualizacion.
7. Revisar limites: datos protegidos, confidencialidad, menores, violencia de genero, seguridad u otros supuestos.

Errores frecuentes:
- Publicar un PDF escaneado y llamarlo dato abierto.
- No indicar licencia.
- Mezclar importes concedidos, pagados y justificados sin definirlos.
- No explicar la fecha de referencia.
- Publicar datos personales innecesarios.

### Supuesto 2. Intercambio de datos de empadronamiento para una prestacion

Situacion: un procedimiento de ayuda necesita comprobar residencia. Se plantea pedir certificado al ciudadano.

Pistas normativas:
- Ley 39/2015: documentacion y derechos de las personas.
- Ley 40/2015 y RD 203/2021: relaciones electronicas entre Administraciones.
- NTI de Protocolos de intermediacion de datos.
- RGPD: base juridica, minimizacion y seguridad.

Resolucion:
1. Comprobar que el dato es necesario y proporcionado.
2. Identificar fuente competente y canal de consulta.
3. Usar plataforma o nodo de intermediacion si existe.
4. Registrar consentimiento o base juridica cuando proceda segun el regimen aplicable.
5. Dejar evidencia de consulta, fecha, respuesta y uso.
6. Evitar conservar mas datos de los necesarios.

Idea de examen: interoperabilidad no es "curiosidad administrativa"; se justifica por simplificacion y por evitar cargas al ciudadano, siempre con base juridica y garantias.

### Supuesto 3. Catalogo de datos autonomico con problemas de calidad

Situacion: un portal de datos abiertos tiene datasets duplicados, fechas de actualizacion falsas, enlaces rotos y columnas sin descripcion.

Pistas normativas y tecnicas:
- NTI-RISP.
- Guia de calidad de datos abiertos.
- Reglamento 2023/138 si hay datos de alto valor.
- DCAT-AP y DCAT-AP-ES como referencia doctrinal.

Resolucion:
1. Inventariar datasets y responsables.
2. Identificar fuentes autorizadas.
3. Normalizar metadatos obligatorios y recomendados.
4. Validar formatos, enlaces, licencias, periodicidad y cobertura.
5. Crear indicadores de salud del dato.
6. Aplicar versionado y politica de cambios.
7. Publicar API o descarga masiva segun naturaleza del dato.

### Supuesto 4. Ontologia para servicios sociales

Situacion: varias Administraciones usan conceptos distintos para unidad familiar, hogar, convivencia, beneficiario, representante y expediente.

Pistas:
- Interoperabilidad semantica.
- NTI de Relacion de modelos de datos.
- EIF y SEMIC.
- RGPD si aparecen datos personales.

Resolucion:
1. Acordar glosario juridico-funcional.
2. Distinguir definicion normativa de definicion tecnica.
3. Crear modelo conceptual con entidades y relaciones.
4. Asociar codigos y vocabularios controlados.
5. Versionar definiciones.
6. Validar reglas de calidad y consistencia.
7. Documentar limites de uso y finalidad.

Error frecuente: creer que una ontologia "resuelve" una contradiccion legal. Si dos prestaciones definen unidad familiar de manera distinta por norma, el modelo debe representar esa diferencia, no ocultarla.

## 7. Notas de test separadas

Estas notas no deben mezclarse con la teoria continua; van en bloques de repaso, test o enfoque de examen.

### 7.1 Preguntas base

1. Que diferencia hay entre transparencia activa y reutilizacion?
Respuesta: la transparencia activa obliga a publicar determinada informacion para control publico; la reutilizacion regula el uso posterior de documentos o datos para fines distintos, comerciales o no, bajo condiciones de formato, licencia y limites.

2. Cuales son las capas de interoperabilidad mas habituales en el marco europeo?
Respuesta: juridica, organizativa, semantica y tecnica, con gobierno de interoperabilidad como elemento transversal.

3. Que norma espanola regula el Esquema Nacional de Interoperabilidad?
Respuesta: Real Decreto 4/2010, desarrollado por Normas Tecnicas de Interoperabilidad.

4. Que norma europea lista los datos de alto valor?
Respuesta: Reglamento de Ejecucion (UE) 2023/138.

5. Que diferencia hay entre anonimizacion y seudonimizacion?
Respuesta: la anonimizacion impide identificar razonablemente a una persona; la seudonimizacion reduce la vinculacion directa pero sigue siendo dato personal si puede reidentificarse con informacion adicional.

### 7.2 Trampas tipicas

| Trampa | Correccion |
| --- | --- |
| "Publicar datos equivale a reutilizacion libre sin condiciones." | La reutilizacion requiere revisar ley, licencia, formato, limites y proteccion de datos. |
| "Interoperabilidad es solo conectar aplicaciones." | Tambien exige base legal, procesos, significado comun y seguridad. |
| "Un PDF publicado es siempre dato abierto." | Puede ser informacion publicada, pero no necesariamente dato abierto reutilizable y legible por maquina. |
| "Quitar nombres anonimiza." | Pueden quedar identificadores indirectos o riesgo por combinacion. |
| "Calidad del dato es tarea tecnica posterior." | Debe gestionarse desde captura, fuente autorizada y proceso. |
| "Los planes estrategicos crean obligaciones iguales que una ley." | Orientan politicas, pero la obligacion juridica deriva de normas aplicables. |
| "DCAT describe el contenido exacto de cada dominio." | DCAT describe catalogos, datasets, servicios y distribuciones; los modelos de dominio requieren vocabularios o esquemas propios. |

### 7.3 Preguntas de aplicacion

Pregunta: una Administracion publica un dataset de movilidad en CSV actualizado en tiempo real una vez al mes. Que problema hay?
Respuesta esperada: si los datos son dinamicos o de alto valor, puede ser necesario acceso por API y actualizacion adecuada. Ademas debe revisarse periodicidad, metadatos, calidad y licencia.

Pregunta: dos organismos usan el mismo campo "estado" con valores "A", "B" y "C", pero con significados distintos. Que capa falla?
Respuesta esperada: interoperabilidad semantica. Tambien puede fallar la gobernanza si no hay diccionario, modelo ni codificacion comun.

Pregunta: se va a abrir un dataset de expedientes sancionadores con identificadores indirectos. Que debe revisarse?
Respuesta esperada: RGPD, LOPDGDD, base juridica, minimizacion, anonimización robusta, riesgo de reidentificacion, finalidad, seguridad y limites de transparencia/RISP.

## 8. Plan de visuales utiles

1. Mapa de cuatro capas de interoperabilidad: juridica, organizativa, semantica y tecnica, con gobierno como capa envolvente. SVG local, sin imagen decorativa.
2. Flujo "del dato al dato abierto": captura -> fuente autorizada -> calidad -> metadatos -> catalogo -> API/descarga -> reutilizacion -> retroalimentacion.
3. Matriz "transparencia, acceso, reutilizacion, gobierno del dato" con cuatro columnas.
4. Diagrama de semantica: glosario -> vocabulario controlado -> modelo de datos -> catalogo DCAT -> API.
5. Ciclo de calidad del dato: definir, capturar, validar, monitorizar, corregir, publicar, auditar.
6. Tabla de datos de alto valor por categoria UE.
7. Caso practico visual: solicitud de ayuda publica y consulta intermediada de datos.

Requisito para HTML: los diagramas deben ser SVG determinista o HTML/CSS local, con texto localizable, scroll horizontal en movil si hay tablas anchas y posibilidad de ocultar visuales en primera lectura.

## 9. Esquema de fuentes para `fuentes.md`

Propuesta editorial sin URL visible:

### Normativa europea

- Union Europea. Directiva (UE) 2019/1024, de 20 de junio de 2019, relativa a los datos abiertos y la reutilizacion de la informacion del sector publico.
- Union Europea. Reglamento de Ejecucion (UE) 2023/138, de 21 de diciembre de 2022, por el que se establecen una lista de conjuntos de datos especificos de alto valor y modalidades de publicacion y reutilizacion.
- Union Europea. Reglamento (UE) 2024/903, de 13 de marzo de 2024, por el que se establecen medidas para garantizar un alto nivel de interoperabilidad del sector publico en toda la Union.
- Union Europea. Reglamento (UE) 2022/868, de 30 de mayo de 2022, relativo a la gobernanza europea de datos.
- Union Europea. Reglamento (UE) 2023/2854, de 13 de diciembre de 2023, sobre normas armonizadas para un acceso justo a los datos y su utilizacion.
- Union Europea. Reglamento (UE) 2018/1724, relativo a la pasarela digital unica.
- Union Europea. Reglamento (UE) 2016/679, Reglamento General de Proteccion de Datos.
- Union Europea. Directiva 2007/2/CE INSPIRE y normativa de ejecucion sobre informacion espacial.

### Normativa estatal

- Espana. Constitucion Espanola.
- Espana. Ley 39/2015, de 1 de octubre, del Procedimiento Administrativo Comun de las Administraciones Publicas.
- Espana. Ley 40/2015, de 1 de octubre, de Regimen Juridico del Sector Publico.
- Espana. Real Decreto 203/2021, de 30 de marzo, Reglamento de actuacion y funcionamiento del sector publico por medios electronicos.
- Espana. Real Decreto 4/2010, de 8 de enero, Esquema Nacional de Interoperabilidad.
- Espana. Real Decreto 311/2022, de 3 de mayo, Esquema Nacional de Seguridad.
- Espana. Ley 37/2007, de 16 de noviembre, sobre reutilizacion de la informacion del sector publico.
- Espana. Real Decreto-ley 24/2021, de 2 de noviembre, libro tercero, transposicion de la Directiva (UE) 2019/1024.
- Espana. Real Decreto 1495/2011, de 24 de octubre, desarrollo de la Ley 37/2007 para el sector publico estatal.
- Espana. Ley 19/2013, de transparencia, acceso a la informacion publica y buen gobierno.
- Espana. Ley Organica 3/2018, de Proteccion de Datos Personales y garantia de los derechos digitales.
- Espana. Ley 14/2010, sobre infraestructuras y servicios de informacion geografica en Espana.
- Espana. Ley 12/1989, de la Funcion Estadistica Publica.

### Normas tecnicas y guias oficiales

- Secretaria de Estado de Administraciones Publicas. Norma Tecnica de Interoperabilidad de Relacion de modelos de datos.
- Secretaria de Estado de Administraciones Publicas. Norma Tecnica de Interoperabilidad de Protocolos de intermediacion de datos.
- Secretaria de Estado de Administraciones Publicas. Norma Tecnica de Interoperabilidad de Reutilizacion de recursos de la informacion.
- Secretaria de Estado de Administraciones Publicas. Norma Tecnica de Interoperabilidad de Catalogo de estandares.
- Portal de Administracion Electronica. Guias de aplicacion del Esquema Nacional de Interoperabilidad.
- datos.gob.es. Guia de aplicacion de la Norma Tecnica de Interoperabilidad de Reutilizacion de Recursos de Informacion.
- datos.gob.es. Guia practica para la mejora de la calidad de datos abiertos.
- datos.gob.es. DCAT-AP y la Norma Tecnica de Interoperabilidad de Reutilizacion de Recursos de Informacion.
- datos.gob.es. Guia practica para migrar a DCAT-AP-ES.
- datos.gob.es. Decalogo del reutilizador de datos del sector publico.

### Andalucia

- Comunidad Autonoma de Andalucia. Ley 1/2014, de Transparencia Publica de Andalucia.
- Junta de Andalucia. Decreto 622/2019, de administracion electronica, simplificacion de procedimientos y racionalizacion organizativa.
- Junta de Andalucia. Portal de Datos Abiertos.
- Agencia Digital de Andalucia. Gobierno del dato.
- Junta de Andalucia. Estrategia Andaluza de Administracion Digital centrada en las personas 2030.
- Agencia Digital de Andalucia. Plan Plurianual de Actuacion 2025-2030.

### Estandares tecnicos de apoyo

- W3C. Data Catalog Vocabulary, DCAT version 3.
- W3C. RDF 1.1 Concepts and Abstract Syntax.
- W3C. SKOS Simple Knowledge Organization System.
- W3C. OWL 2 Web Ontology Language.
- W3C. Shapes Constraint Language SHACL.
- W3C. SPARQL 1.1.
- ISO/IEC. Familia 25012 y 25024 sobre calidad de datos y medicion, como apoyo doctrinal.

## 10. Enlaces de trazabilidad interna

No copiar estos enlaces al tema final ni al HTML visible. Sirven para que el integrador archive o compruebe fuentes.

| ID | Enlace oficial |
| --- | --- |
| UE-01 | https://eur-lex.europa.eu/eli/dir/2019/1024/oj?locale=es |
| UE-02 | https://eur-lex.europa.eu/eli/reg_impl/2023/138/oj?locale=es |
| UE-03 | https://eur-lex.europa.eu/eli/reg/2024/903/oj?locale=es |
| UE-04 | https://eur-lex.europa.eu/eli/reg/2022/868/oj?locale=es |
| UE-05 | https://eur-lex.europa.eu/eli/reg/2023/2854/oj?locale=es |
| UE-06 | https://eur-lex.europa.eu/eli/reg/2018/1724/oj?locale=es |
| UE-07 | https://eur-lex.europa.eu/eli/reg/2016/679/oj?locale=es |
| UE-09 | https://interoperable-europe.ec.europa.eu/collection/iopeu-monitoring/european-interoperability-framework |
| ES-02 | https://www.boe.es/buscar/act.php?id=BOE-A-2015-10565 |
| ES-03 | https://www.boe.es/buscar/act.php?id=BOE-A-2015-10566 |
| ES-04 | https://www.boe.es/buscar/act.php?id=BOE-A-2021-5032 |
| ES-05 | https://www.boe.es/buscar/act.php?id=BOE-A-2010-1331 |
| ES-06 | https://www.boe.es/buscar/act.php?id=BOE-A-2022-7191 |
| ES-07 | https://www.boe.es/buscar/act.php?id=BOE-A-2007-19814 |
| ES-08 | https://www.boe.es/buscar/act.php?id=BOE-A-2021-17910 |
| ES-09 | https://www.boe.es/buscar/act.php?id=BOE-A-2011-17560 |
| ES-10 | https://www.boe.es/buscar/act.php?id=BOE-A-2013-12887 |
| ES-11 | https://www.boe.es/buscar/act.php?id=BOE-A-2018-16673 |
| ES-12 | https://www.boe.es/buscar/act.php?id=BOE-A-2010-10707 |
| ES-13 | https://www.boe.es/buscar/act.php?id=BOE-A-1989-10767 |
| NTI-01 | https://www.boe.es/buscar/act.php?id=BOE-A-2012-10050 |
| NTI-02 | https://www.boe.es/buscar/act.php?id=BOE-A-2012-10049 |
| NTI-03 | https://www.boe.es/buscar/act.php?id=BOE-A-2013-2380 |
| NTI-04 | https://www.boe.es/buscar/act.php?id=BOE-A-2012-13501 |
| GUIA-01 | https://datos.gob.es/es/documentacion/guia-de-aplicacion-de-la-norma-tecnica-de-interoperabilidad-de-reutilizacion-de |
| GUIA-02 | https://datos.gob.es/es/documentacion/dcat-ap-y-la-norma-tecnica-de-interoperabilidad-de-reutilizacion-de-recursos-de |
| GUIA-03 | https://datos.gob.es/es/documentacion/guia-practica-para-la-mejora-de-la-calidad-de-datos-abiertos |
| GUIA-04 | https://datos.gob.es/es/conocimiento/guia-practica-para-migrar-dcat-ap-es |
| GUIA-05 | https://datos.gob.es/es/conocimiento/decalogo-del-reutilizador-de-datos-del-sector-publico |
| AND-01 | https://www.boe.es/buscar/act.php?id=BOE-A-2014-7534 |
| AND-02 | https://www.juntadeandalucia.es/boja/2019/250/1.html |
| AND-03 | https://www.juntadeandalucia.es/datosabiertos/portal/catalogo-datos |
| AND-04 | https://www.juntadeandalucia.es/organismos/ada/areas/estrategia/gobierno-dato.html |
| AND-05 | https://www.juntadeandalucia.es/organismos/industriaenergiayminas/consejeria/transparencia/planificacion-evaluacion-estadistica/planes/detalle/655954.html |
| AND-06 | https://www.juntadeandalucia.es/organismos/transparencia/planificacion-evaluacion-estadistica/planes/detalle/607849.html |
| TEC-01 | https://www.w3.org/TR/vocab-dcat-3/ |
| TEC-02 | https://www.w3.org/TR/rdf11-concepts/ |
| TEC-03 | https://www.w3.org/TR/skos-reference/ |
| TEC-04 | https://www.w3.org/TR/owl-overview/ |
| TEC-05 | https://www.w3.org/TR/shacl/ |
| TEC-06 | https://www.w3.org/TR/sparql11-overview/ |

## 11. Riesgos y puntos a verificar por el integrador

1. DCAT-AP-ES: datos.gob.es indica guias recientes y tramitacion/actualizacion. Antes de afirmar que una nueva NTI-RISP ya esta aprobada, verificar BOE. Formulacion segura: "las guias recientes impulsan la migracion hacia DCAT-AP-ES y la alineacion con el marco europeo".
2. Data Governance Act y Data Act: no convertirlos en normas especificas de RISP espanola. Son marco europeo mas amplio de economia y gobernanza de datos.
3. Datos personales: no afirmar que los datos publicos quedan fuera del RGPD. La regla es la contraria: si hay datos personales, RGPD y LOPDGDD aplican.
4. INSPIRE/geodatos: utilizar como ejemplo sectorial, no como marco general de todos los datos.
5. Planes de Andalucia 2025-2030 y Estrategia 2030: presentarlos como contexto vigente de politica publica, subordinado a leyes y reglamentos.
6. URLs: no incluir URLs reales en el markdown final visible ni en el HTML final; archivar referencias o usar citas editoriales.
7. Calidad: evitar listas sin explicacion. Cada dimension debe conectarse con consecuencias reales en procedimiento, transparencia, analitica o reutilizacion.
8. Interoperabilidad semantica: no limitarla a RDF/ontologias. En A1 debe explicarse tambien con glosarios, codigos, modelos y metadatos.
