# Entrega de visuales, tablas y esquemas responsivos

## Uso previsto

Este material aporta piezas visuales y comparativas para un tema A1 sobre
gobierno del dato, interoperabilidad semantica, calidad del dato y reutilizacion
de la informacion publica. No sustituye el desarrollo teorico. Cada tabla o
esquema esta pensado como apoyo a una primera lectura continua y como base para
modo tutor, repaso y enfoque de examen.

Los SVG incluidos son deterministas, locales y reutilizables. Se recomienda
insertarlos con un contenedor responsivo con desplazamiento horizontal en movil,
texto alternativo editorial y pie de figura breve. No contienen enlaces externos
ni dependencias remotas.

## Plan de visuales integrables

| Visual | Ubicacion sugerida | Objetivo didactico | Archivo propuesto | Alt text sugerido |
| --- | --- | --- | --- | --- |
| Mapa inicial del tema | Despues de la orientacion de examen | Presentar la relacion entre gobierno, interoperabilidad, calidad y reutilizacion | `assets/mapa_gobierno_dato_interoperabilidad.svg` | Mapa que conecta marco juridico, gobierno del dato, calidad, interoperabilidad y reutilizacion de informacion publica. |
| Capas de interoperabilidad | Bloque de interoperabilidad | Evitar confundir interoperabilidad tecnica con interoperabilidad completa | `assets/capas_interoperabilidad.svg` | Capas legal, organizativa, semantica y tecnica de la interoperabilidad en servicios publicos digitales. |
| Ciclo de calidad del dato | Bloque de calidad | Mostrar la calidad como ciclo permanente, no como revision final | `assets/ciclo_calidad_dato.svg` | Ciclo de perfilado, reglas, validacion, correccion, publicacion y monitorizacion de calidad del dato. |
| Cadena de reutilizacion | Bloque de reutilizacion de informacion publica | Explicar el paso de dato administrativo a activo reutilizable | `assets/pipeline_reutilizacion.svg` | Cadena desde inventario de datos hasta publicacion, acceso, reutilizacion y retroalimentacion. |
| Matriz de fuentes normativas | Cierre del bloque teorico | Ordenar las fuentes oficiales por funcion | Tabla markdown | Tabla de normas estatales y europeas aplicables al dato publico. |
| Cuadro de errores frecuentes | Enfoque de examen | Convertir confusiones tipicas en criterios de discriminacion | Tabla markdown | Tabla de errores frecuentes y forma correcta de reconocerlos. |

## Mapa inicial para el tema

El tema puede abrir con una idea rectora: una Administracion publica moderna no
gestiona "ficheros" aislados, sino activos de informacion sometidos a reglas de
responsabilidad, calidad, interoperabilidad, seguridad, proteccion de datos,
transparencia y reutilizacion. El gobierno del dato fija quien decide, quien
custodia, con que criterios se mide la calidad y como se documenta el ciclo de
vida. La interoperabilidad asegura que esos datos puedan entenderse y usarse
entre organos, Administraciones y terceros. La calidad hace que el dato sea
fiable para tramitar, decidir, automatizar o publicar. La reutilizacion permite
que la informacion publica, cuando procede, genere valor social, economico y
democratico fuera del expediente que la origino.

El mapa debe evitar una secuencia falsa en la que primero se "gobierna" y luego
se "publica". En la practica, los cuatro ejes se condicionan mutuamente. Un dato
sin responsable no tiene calidad sostenible. Un dato sin metadatos no se puede
interoperar. Un dato sin calidad degrada la reutilizacion. Un dato reutilizable
sin licencias, formatos y condiciones claras no es realmente abierto. Y un dato
personal o protegido no se publica por el simple hecho de existir en poder de una
Administracion.

## Tabla comparativa: cuatro ejes del tema

| Eje | Pregunta que responde | Producto visible | Riesgo si se descuida | Clave de examen |
| --- | --- | --- | --- | --- |
| Gobierno del dato | Quien decide, define, custodia, mide y rinde cuentas sobre el dato. | Politicas, roles, catalogos, linaje, reglas de calidad, comites y procedimientos. | Duplicidades, incoherencias, datos sin responsable, decisiones no trazables. | No es solo tecnologia: es organizacion, responsabilidad y ciclo de vida. |
| Interoperabilidad semantica | Como se asegura que dos sistemas entienden lo mismo cuando intercambian datos. | Modelos de datos, vocabularios, codigos comunes, metadatos, identificadores y definiciones compartidas. | Integraciones tecnicas que transmiten campos pero no significado. | Diferenciar semantica de formato, protocolo o conectividad. |
| Calidad del dato | Como se verifica que el dato es apto para su finalidad. | Reglas, metricas, controles, incidencias, planes de mejora y evidencias. | Automatizaciones erroneas, perdida de confianza, decisiones administrativas defectuosas. | La calidad se define respecto al uso y debe medirse. |
| Reutilizacion de informacion publica | Como se pone informacion publica a disposicion de terceros bajo condiciones claras. | Catalogos, descargas, APIs, licencias, formatos abiertos, metadatos y canales de soporte. | Publicar datos inutilizables, protegidos, desactualizados o juridicamente ambiguos. | Reutilizar no equivale a publicar cualquier dato ni a vulnerar proteccion de datos. |

### Modo tutor

El opositor debe asociar cada eje a una pregunta funcional. Si el enunciado
habla de "quien es responsable", "linaje", "calidad", "catalogo interno" o
"ciclo de vida", normalmente apunta a gobierno del dato. Si habla de significado
comun, vocabularios, modelos semanticos o codigos compartidos, apunta a
interoperabilidad semantica. Si habla de exactitud, completitud, actualidad o
consistencia, apunta a calidad. Si habla de formatos, licencias, catalogos
publicos, APIs y uso por terceros, apunta a reutilizacion.

### Nota de test

Trampa frecuente: una opcion puede mencionar "formato abierto" y parecer
interoperabilidad semantica. El formato abierto facilita acceso tecnico, pero no
garantiza que el significado de los campos sea comun. Para interoperabilidad
semantica debe aparecer definicion compartida, modelo de datos, vocabulario,
metadato o correspondencia de significado.

## Tabla normativa orientativa

| Fuente | Ambito principal | Aporta al tema | Como usarla en el desarrollo |
| --- | --- | --- | --- |
| Reglamento General de Proteccion de Datos y normativa espanola de proteccion de datos | Proteccion de datos personales | Principios, licitud, minimizacion, limitacion de finalidad, responsabilidad proactiva y derechos | Para explicar limites al dato abierto, calidad vinculada a exactitud y necesidad de gobierno responsable. |
| Ley 39/2015 | Procedimiento administrativo comun | Expediente, documentos, tramitacion electronica, derechos de las personas interesadas | Para conectar dato, documento, expediente y servicio publico digital. |
| Ley 40/2015 | Regimen juridico del sector publico | Funcionamiento electronico e interaccion entre Administraciones | Para situar cooperacion, relaciones interadministrativas y servicios compartidos. |
| Real Decreto 203/2021 | Actuacion y funcionamiento del sector publico por medios electronicos | Desarrollo reglamentario del funcionamiento electronico | Para vincular gestion administrativa, canales digitales, archivo, intercambio y expediente. |
| Esquema Nacional de Interoperabilidad | Interoperabilidad en el sector publico | Principios, dimensiones y normas tecnicas | Para estructurar interoperabilidad legal, organizativa, semantica y tecnica. |
| Normas Tecnicas de Interoperabilidad | Aplicacion tecnica y organizativa del ENI | Documento electronico, expediente, catalogos, reutilizacion, copiado autentico, politica de firma, etc. | Para aterrizar el ENI en instrumentos concretos, sin convertir el tema en una lista de resoluciones. |
| Ley 37/2007 sobre reutilizacion de la informacion del sector publico | Reutilizacion | Condiciones, modalidades, transparencia, formatos y obligaciones de los organismos | Para explicar que reutilizacion es un regimen juridico propio, no mera descarga de datos. |
| Ley 19/2013 | Transparencia y acceso a la informacion publica | Publicidad activa, derecho de acceso, buen gobierno | Para diferenciar transparencia, acceso y reutilizacion. |
| Directiva de datos abiertos y reutilizacion de informacion del sector publico | Marco europeo de open data | Datos dinamicos, conjuntos de alto valor, formatos y APIs | Para enlazar la politica espanola con la europea. |
| Reglamento de Gobernanza de Datos | Gobernanza europea del dato | Reutilizacion de categorias protegidas, intermediacion de datos, altruismo de datos | Para situar el dato publico en un ecosistema europeo de confianza. |
| Reglamento de Datos | Acceso y uso de datos en la economia europea | Acceso, intercambio, condiciones contractuales y servicios de tratamiento | Como contexto europeo reciente sobre espacios de datos, sin desplazar el foco administrativo. |
| Reglamento de la Europa Interoperable | Cooperacion europea en interoperabilidad | Evaluaciones, soluciones reutilizables y cooperacion transfronteriza | Para mostrar que interoperabilidad es politica publica y no solo integracion local. |
| Reglamento de Ejecucion sobre conjuntos de datos de alto valor | Open data europeo | Categorias de datos de alto valor y requisitos de publicacion | Para explicar por que algunos datos publicos tienen prioridad de apertura. |

### Modo tutor

La tabla no debe presentarse como inventario memoristico. Su utilidad es agrupar
funciones: unas fuentes protegen el dato, otras ordenan la tramitacion, otras
obligan a interoperar, otras permiten reutilizar y otras impulsan espacios
europeos de datos. En examen, conviene detectar la funcion juridica antes que
recordar una enumeracion completa.

### Nota de test

Trampa frecuente: confundir transparencia con reutilizacion. La transparencia
se centra en publicidad activa y acceso a informacion publica. La reutilizacion
se centra en el uso posterior de documentos o datos por personas fisicas o
juridicas, con condiciones y formatos que permitan crear valor o nuevos
servicios. Pueden relacionarse, pero no son sinonimos.

## Tabla: capas de interoperabilidad

| Capa | Que armoniza | Ejemplos en Administracion publica | Indicadores de que aparece en un supuesto | Confusion habitual |
| --- | --- | --- | --- | --- |
| Legal | Normas, competencias, bases juridicas y obligaciones | Proteccion de datos, transparencia, procedimiento, reutilizacion, archivo, firma electronica | El caso pregunta si puede cederse, publicarse, conservarse o reutilizarse un dato | Creer que una API legitima cualquier intercambio. |
| Organizativa | Procesos, responsabilidades y acuerdos entre entidades | Convenios, procedimientos comunes, ventanilla unica, organos responsables, acuerdos de nivel de servicio | El supuesto menciona organos distintos, responsabilidades, plazos o circuitos administrativos | Reducir el problema a "conectar sistemas". |
| Semantica | Significado de los datos intercambiados | Diccionarios de datos, modelos comunes, vocabularios, codigos territoriales, identificadores | El problema esta en que cada sistema llama distinto a la misma realidad o usa codigos incompatibles | Confundir semantica con XML, JSON o formato. |
| Tecnica | Protocolos, formatos, infraestructuras y seguridad tecnica | Servicios web, APIs, formatos abiertos, certificados, redes, mecanismos de intercambio | El supuesto se centra en transporte, formato, autenticacion o disponibilidad tecnica | Pensar que resolver la tecnica resuelve el significado. |
| Gobernanza transversal | Decision, priorizacion, control y mejora continua | Comites de interoperabilidad, catalogos, politicas de datos, medicion, auditoria | El enunciado pregunta quien mantiene, versiona, aprueba o revisa | Tratar la interoperabilidad como proyecto puntual. |

### Modo tutor

La interoperabilidad completa requiere que los sistemas puedan trabajar juntos
sin perder significado, validez juridica ni responsabilidad administrativa. Por
eso el examen suele castigar respuestas que reducen todo a conectividad. Si dos
organismos intercambian un campo llamado "estado", pero uno lo entiende como
estado de expediente y otro como estado civil, la integracion tecnica puede
funcionar y la interoperabilidad semantica estar rota.

## Tabla: roles de gobierno del dato

| Rol | Funcion principal | Decision que suele tomar | Evidencia esperable | Error frecuente |
| --- | --- | --- | --- | --- |
| Responsable funcional del dato | Define significado, finalidad administrativa y reglas de negocio | Que representa el dato, para que se usa y que nivel de calidad necesita | Definicion oficial, reglas funcionales, criterios de correccion | Delegar todo en tecnologia. |
| Propietario o titular del conjunto | Asume la responsabilidad de un conjunto de datos dentro de la organizacion | Prioridad de mejora, nivel de apertura, condiciones internas de uso | Ficha de dataset, politica, autorizaciones, revision periodica | Creer que propietario equivale a "dueno privado" del dato publico. |
| Custodio tecnico | Mantiene almacenamiento, disponibilidad, seguridad tecnica e integraciones | Arquitectura, backup, controles de acceso, monitorizacion | Registros, configuraciones, controles operativos | Que el custodio redefina el significado juridico o funcional. |
| Responsable de calidad | Define metricas, umbrales y seguimiento de incidencias | Que dimensiones medir y como corregir desviaciones | Informes de calidad, reglas, incidencias, plan de remediacion | Medir solo si hay un error visible. |
| Delegado o unidad de proteccion de datos | Evalua riesgos y cumplimiento cuando hay datos personales | Necesidad de evaluacion, minimizacion, base juridica, garantias | Informes, recomendaciones, registro de actividades, evaluacion de impacto si procede | Usarlo como aprobador general de cualquier dato, aunque no haya datos personales. |
| Responsable de seguridad | Define controles de seguridad y continuidad | Medidas, clasificacion, auditoria, gestion de riesgos | Politica de seguridad, analisis de riesgos, controles | Confundir seguridad con calidad o con apertura. |
| Unidad de transparencia o datos abiertos | Facilita publicacion, catalogacion y reutilizacion | Formatos, metadatos, licencias, actualizacion y canales | Catalogo, fichas, APIs, estadisticas de uso | Publicar sin coordinar con responsables funcionales y juridicos. |
| Comite de datos | Prioriza, resuelve conflictos y asegura coherencia transversal | Roadmap, estandares, excepciones, responsabilidades | Actas, acuerdos, politicas aprobadas | Crear un organo formal sin capacidad operativa. |

### Nota de test

Cuando una pregunta pide "quien garantiza que un dato tenga significado correcto
para el procedimiento", la respuesta mas probable no es el area tecnica sino el
responsable funcional o propietario del dato. Cuando pregunta por copias,
accesos, disponibilidad o integraciones, puede aparecer el custodio tecnico.

## Tabla: dimensiones de calidad del dato

| Dimension | Definicion operativa | Ejemplo publico | Metrica posible | Causa frecuente de baja calidad |
| --- | --- | --- | --- | --- |
| Exactitud | El dato representa correctamente la realidad que pretende describir | Domicilio administrativo correcto de una persona interesada | Porcentaje de registros verificados contra fuente autorizada | Captura manual sin validacion o falta de actualizacion. |
| Completitud | Estan presentes los campos necesarios para una finalidad | Expediente con datos minimos para tramitar una ayuda | Porcentaje de registros con campos obligatorios cumplimentados | Formularios incompletos o integraciones parciales. |
| Consistencia | No hay contradicciones internas o entre sistemas | Codigo de municipio coherente con provincia y comunidad autonoma | Numero de reglas de coherencia incumplidas por periodo | Modelos distintos o sincronizaciones asincronas sin control. |
| Actualidad | El dato esta actualizado para el uso previsto | Estado de una licencia, subvencion o autorizacion | Tiempo desde ultima actualizacion frente a umbral | Procesos batch lentos o ausencia de responsable de refresco. |
| Validez | El dato cumple formato, dominio y reglas permitidas | NIF, fecha, codigo DIR3, codigo postal o identificador de expediente | Porcentaje de valores fuera de dominio | Falta de catalogos de referencia o controles de entrada. |
| Unicidad | Una entidad no aparece duplicada de forma indebida | Mismo establecimiento inscrito dos veces con pequenas variaciones | Tasa de duplicados probables | Ausencia de identificador comun o reglas de resolucion de identidad. |
| Trazabilidad | Se conoce origen, transformaciones y responsable | Dataset estadistico derivado de registros administrativos | Porcentaje de campos con linaje documentado | Transformaciones manuales, hojas intermedias o cargas sin metadatos. |
| Accesibilidad controlada | El dato esta disponible para quien puede usarlo y protegido frente a quien no | Consulta interna de expediente por unidad competente | Incidencias de acceso, tiempos de respuesta, accesos denegados correctos | Permisos excesivos o demasiado restrictivos. |

### Modo tutor

La calidad no es un atributo absoluto. Un dato puede ser suficiente para una
estadistica agregada y deficiente para una resolucion individual. Por eso el
examen puede plantear escenarios donde no basta decir "el dato es correcto"; hay
que preguntar correcto para que finalidad, con que riesgo y con que evidencia.

## Tabla: interoperabilidad semantica aplicada

| Problema | Solucion semantica | Instrumento posible | Resultado esperado |
| --- | --- | --- | --- |
| Dos unidades usan nombres distintos para el mismo concepto | Definicion comun y glosario corporativo | Diccionario de datos | Disminuyen ambiguedades en informes e integraciones. |
| Dos sistemas usan codigos diferentes para una misma clasificacion | Tabla de correspondencias o adopcion de codigo comun | Catalogo de codigos, vocabulario controlado | Los datos pueden compararse y agregarse. |
| Un campo cambia de significado con el tiempo | Versionado semantico | Modelo de datos versionado y metadatos de vigencia | Se pueden interpretar historicos sin errores. |
| Un tercero reutilizador no entiende el contenido del dataset | Metadatos descriptivos y definiciones publicas | Ficha de catalogo, esquema, vocabulario | Aumenta reutilizacion real y baja necesidad de soporte. |
| Una integracion transmite JSON valido pero produce decisiones incorrectas | Reglas de negocio y validacion semantica | Contrato de datos, pruebas de validacion | La integracion valida significado, no solo sintaxis. |
| Varios organos publican datos parecidos sin homogeneidad | Estandar comun y gobernanza de cambios | Perfil de aplicacion, ontologia ligera, grupo de mantenimiento | Comparabilidad y escalabilidad interadministrativa. |

### Nota de test

Si el supuesto dice que "los datos se transmiten correctamente pero el organo
receptor interpreta mal los estados", la averia no es principalmente de red ni
de formato. Es de interoperabilidad semantica y de gobierno del modelo de datos.

## Tabla: reutilizacion de informacion publica

| Elemento | Por que importa | Buena practica | Senal de mala practica |
| --- | --- | --- | --- |
| Inventario de datos | Permite saber que existe antes de publicar | Catalogo interno con responsables, fuentes y condiciones | Publicaciones aisladas sin responsable ni mantenimiento. |
| Metadatos | Permiten localizar, entender y evaluar el dato | Titulo claro, descripcion, periodicidad, cobertura, formato, licencia, fecha de actualizacion | Dataset con columnas cripticas o sin fecha. |
| Formatos abiertos y legibles por maquina | Facilitan automatizacion y reutilizacion | CSV, JSON, XML, RDF u otros formatos adecuados al caso | PDF escaneado como unica forma de acceso a datos tabulares. |
| Licencia y condiciones | Dan seguridad juridica al reutilizador | Condiciones claras, no discriminatorias y compatibles con reutilizacion | Ambiguedad sobre permisos, tasas o atribucion. |
| API o descarga masiva | Ajusta acceso a usos diferentes | API para datos dinamicos y descarga para historicos o volumen | Solo consultas manuales pantalla a pantalla. |
| Calidad documentada | Evita usos erroneos y mejora confianza | Indicadores, avisos de limitacion, historial de cambios | Datos sin advertencias, aunque haya rupturas de serie. |
| Proteccion de datos y confidencialidad | Evita publicar lo que no debe publicarse | Anonimizacion, agregacion, exclusion o base juridica adecuada | Publicar datos personales por defecto. |
| Retroalimentacion | Permite mejorar con usuarios reales | Canal de incidencias, versionado, estadisticas de uso | Catalogo sin mantenimiento ni respuesta. |

### Modo tutor

La reutilizacion valiosa no consiste en subir archivos. Exige preparar el dato
para que otra persona pueda descubrirlo, entenderlo, descargarlo, integrarlo,
conocer sus limites y usarlo con seguridad juridica. En examen, una opcion que
solo dice "publicar en la web" suele ser insuficiente si no incluye metadatos,
formato, licencia, calidad y actualizacion.

## Tabla: diferencias que suelen caer en examen

| Pareja | Diferencia esencial | Ejemplo de discriminacion |
| --- | --- | --- |
| Transparencia y reutilizacion | La transparencia garantiza publicidad activa y acceso; la reutilizacion regula el uso posterior de informacion publica. | Solicitar un expediente por derecho de acceso no equivale a crear una API de datos reutilizables. |
| Interoperabilidad tecnica y semantica | La tecnica permite conexion y transporte; la semantica asegura significado compartido. | Enviar JSON valido no evita que "estado" signifique cosas distintas. |
| Calidad y seguridad | La calidad mira aptitud del dato para su finalidad; la seguridad protege confidencialidad, integridad, disponibilidad, autenticidad y trazabilidad tecnica. | Un dato puede estar cifrado y ser incorrecto. |
| Gobierno del dato y gestion documental | El gobierno del dato ordena activos de informacion y responsabilidades; la gestion documental ordena documentos, expedientes, archivo y evidencia administrativa. | Un dataset estadistico derivado de expedientes requiere ambos enfoques. |
| Dato abierto y dato publico | No todo dato en poder publico es abierto; puede estar limitado por proteccion de datos, secreto, seguridad u otros intereses. | Un registro con datos personales no se abre sin base, minimizacion o anonimizacion robusta. |
| Catalogo de datos y repositorio tecnico | El catalogo describe y gobierna; el repositorio almacena. | Un lago de datos sin metadatos no sustituye al catalogo. |
| Anonimizacion y seudonimizacion | La anonimizacion impide identificar razonablemente; la seudonimizacion reduce vinculacion directa pero sigue siendo dato personal si puede reidentificarse. | Sustituir DNI por codigo interno no siempre convierte el dato en anonimo. |

### Nota de test

Trampa frecuente: "dato publico" no significa "dato publicable". La titularidad
publica o la presencia en un expediente no eliminan limites de proteccion de
datos, seguridad, secreto estadistico, propiedad intelectual, intereses
economicos legitimos o confidencialidad.

## Supuesto practico guiado: catalogo de datos municipales

### Situacion

Un ayuntamiento quiere publicar datos de movilidad, licencias urbanisticas,
contratos menores y calidad del aire. Los datos proceden de aplicaciones
distintas, con responsables funcionales diferentes. Algunos campos contienen
datos personales indirectos, otros se actualizan en tiempo real y otros solo se
revisan mensualmente.

### Pistas para resolver

| Pista | Lectura correcta |
| --- | --- |
| Varias aplicaciones y responsables | Hace falta gobierno del dato y asignacion de responsabilidades. |
| Datos que se entienden de forma distinta | Aparece interoperabilidad semantica. |
| Actualizacion real o mensual | Deben definirse periodicidad, calidad y metadatos. |
| Posibles datos personales indirectos | Intervienen proteccion de datos, minimizacion y anonimizacion/agregacion. |
| Publicacion a terceros | Entra reutilizacion, licencia, formatos, API y catalogo. |

### Resolucion paso a paso

1. Inventariar los conjuntos de datos y asignar responsable funcional, custodio
   tecnico y responsable de calidad.
2. Clasificar cada conjunto segun sensibilidad, presencia de datos personales,
   valor de reutilizacion, periodicidad y dependencia de otros sistemas.
3. Definir modelo semantico minimo: campos, unidades, codigos, identificadores,
   cobertura temporal y territorial, versionado y glosario.
4. Establecer reglas de calidad: completitud minima, formatos validos,
   coherencia entre codigos, umbral de actualidad y controles de duplicidad.
5. Decidir modalidad de publicacion: descarga, API, datos dinamicos, historico,
   agregacion o no publicacion si hay limites juridicos no salvables.
6. Documentar metadatos, licencia, fecha de actualizacion, responsable, limites
   de uso y canal de incidencias.
7. Revisar periodicamente uso, incidencias, cambios normativos y nuevas
   necesidades de interoperabilidad.

### Errores a evitar

| Error | Por que es incorrecto | Correccion |
| --- | --- | --- |
| Publicar primero y gobernar despues | Genera datasets sin responsable ni mantenimiento | Gobernanza minima antes de publicar. |
| Convertir todo a CSV y darlo por abierto | El formato no resuelve significado, licencia ni calidad | Anadir metadatos, definiciones, licencia y controles. |
| Tratar seudonimizacion como anonimizacion | Puede seguir siendo dato personal | Evaluar riesgo de reidentificacion y aplicar garantias. |
| Usar codigos internos sin documentar | El reutilizador no puede interpretar el dato | Publicar catalogos de codigos o correspondencias. |
| No indicar periodicidad | El usuario no sabe si el dato esta vigente | Metadato de frecuencia y ultima actualizacion. |

### Mini comprobacion

Una respuesta completa debe mencionar al menos: inventario, responsables,
clasificacion juridica, modelo semantico, reglas de calidad, metadatos, formatos
o API, licencia, actualizacion y seguimiento. Si falta proteccion de datos en un
supuesto con informacion personal, la respuesta queda incompleta.

## Esquema de desarrollo teorico con puntos visuales

| Seccion del tema | Visual o tabla recomendada | Funcion en la lectura |
| --- | --- | --- |
| Orientacion de examen | Mapa inicial | Dar una ruta mental y evitar memorizar conceptos sueltos. |
| Conceptos y definiciones | Tabla de cuatro ejes | Situar cada definicion antes de desarrollarla. |
| Gobierno del dato | Tabla de roles | Mostrar que el gobierno es organizativo y responsable. |
| Interoperabilidad | Capas de interoperabilidad | Prevenir la reduccion tecnicista del concepto. |
| Interoperabilidad semantica | Tabla de problemas y soluciones | Conectar glosarios, modelos y vocabularios con casos reales. |
| Calidad del dato | Ciclo de calidad y tabla de dimensiones | Presentar calidad como sistema continuo medible. |
| Reutilizacion | Pipeline de reutilizacion y tabla de elementos | Aterrizar apertura en catalogo, metadatos, licencia, formatos y APIs. |
| Marco normativo | Matriz normativa | Ordenar fuentes por funcion, no por enumeracion. |
| Supuestos practicos | Tabla de pistas | Enseñar reconocimiento de patrones en examen. |
| Repaso final | Diferencias frecuentes | Consolidar discriminadores de test. |

## Notas de test separadas

1. Si el enunciado habla de "mismo significado en varios sistemas", la palabra
   clave es interoperabilidad semantica, no solo interoperabilidad tecnica.
2. Si el enunciado habla de "responsable, propietario, custodio, linaje o
   catalogo interno", la respuesta se mueve hacia gobierno del dato.
3. Si se plantea una publicacion de datos con personas identificables o
   reidentificables, no basta invocar transparencia o datos abiertos: hay que
   aplicar proteccion de datos, minimizacion, base juridica y, si procede,
   anonimizacion o agregacion.
4. Si un dato se usa para decidir sobre derechos de una persona, la calidad
   exigible debe ser mayor que si se usa para estadistica agregada.
5. Si la opcion dice "publicar en PDF" como solucion de reutilizacion, suele ser
   incompleta para datos tabulares o dinamicos.
6. Si una API devuelve datos sin metadatos, licencias ni definiciones, hay acceso
   tecnico pero reutilizacion debil.
7. Si dos organos tienen competencias distintas, la interoperabilidad tambien
   necesita acuerdos organizativos y base juridica, no solo desarrollo software.
8. Si aparece "conjuntos de alto valor", debe conectarse con open data europeo,
   prioridad de publicacion y requisitos de acceso, no con una etiqueta interna
   de importancia subjetiva.

## Propuesta de repaso visual final

| Idea de repaso | Frase de recuperacion activa |
| --- | --- |
| Gobierno | Quien responde por el dato, como se define, quien lo mantiene y como se decide su uso. |
| Interoperabilidad | Capacidad de cooperar sin perder validez juridica, sentido organizativo, significado ni soporte tecnico. |
| Semantica | Mismo dato entendido de la misma manera por sistemas y personas distintas. |
| Calidad | Aptitud medible del dato para una finalidad concreta. |
| Reutilizacion | Uso posterior de informacion publica con condiciones, formatos, metadatos y garantias. |
| Limites | No todo dato publico es abierto; hay proteccion de datos, seguridad, confidencialidad y proporcionalidad. |

## Criterios de integracion responsive

| Elemento | Requisito |
| --- | --- |
| SVG | Insertar en contenedor con `overflow-x: auto`, ancho minimo razonable y `max-width: none` en pantallas estrechas. |
| Tablas | Envolver en contenedor scrollable; no convertir tablas densas en imagen. |
| Notas de test | Separarlas visualmente de la teoria; no mezclar distractores con definiciones. |
| Modo tutor | Situarlo despues de bloques complejos y mantenerlo ocultable en HTML. |
| Pie de figura | Una frase funcional, no decorativa. |
| Accesibilidad | Usar `alt`, `figcaption` y contraste suficiente. |
