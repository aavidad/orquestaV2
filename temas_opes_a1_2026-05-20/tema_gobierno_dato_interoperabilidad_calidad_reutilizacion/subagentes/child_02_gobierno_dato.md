# Material parcial: gobierno del dato en una Administracion publica

## 1. Enfoque del bloque

El gobierno del dato es el conjunto de principios, responsabilidades, procesos, normas y mecanismos de control que permiten que los datos de una Administracion publica sean comprensibles, fiables, seguros, localizables, reutilizables y utiles para prestar servicios, rendir cuentas y tomar decisiones. No equivale a comprar una herramienta de catalogacion, ni a crear un almacen central de datos, ni a publicar datasets abiertos sin criterio. Es una funcion permanente de direccion y gestion que conecta la estrategia institucional con la operacion diaria de expedientes, registros, servicios digitales, estadisticas, indicadores, interoperabilidad y reutilizacion de informacion publica.

En una Administracion publica el gobierno del dato tiene una condicion especial: los datos no son solo un activo economico o tecnico, sino tambien evidencia de derechos, obligaciones, potestades, actuaciones administrativas y garantias ciudadanas. Un padron, un registro de ayudas, un expediente sancionador, un sistema tributario, una historia social, una base de datos de contratos o un catalogo de patrimonio publico no pueden tratarse como meros ficheros internos. Su calidad afecta a personas concretas, a la igualdad en el acceso a servicios, a la eficacia del gasto publico, a la transparencia, a la supervision y a la confianza institucional.

La idea central para examen es sencilla: el gobierno del dato define quien decide, quien responde, como se documenta, como se controla y como se mejora el dato durante todo su ciclo de vida. La gestion del dato ejecuta esas decisiones en sistemas, procesos y equipos. La arquitectura de datos proporciona modelos, integraciones, plataformas y estandares. La proteccion de datos, la seguridad, la transparencia, la interoperabilidad y la reutilizacion aportan limites juridicos y condiciones de uso. Ninguna de esas piezas sustituye a las demas.

## 2. Principios de gobierno del dato

El primer principio es la orientacion a finalidad publica. Un dato debe gobernarse segun la funcion administrativa que soporta: tramitar un procedimiento, acreditar una condicion, calcular una prestacion, inspeccionar, planificar, informar a la ciudadania, evaluar politicas publicas o facilitar reutilizacion. Esta finalidad permite distinguir entre datos operativos, datos estadisticos, datos maestros, datos abiertos, datos personales, datos protegidos, datos de referencia y datos de evidencia administrativa.

El segundo principio es responsabilidad identificable. Cada conjunto de datos relevante necesita una unidad responsable del significado, calidad, reglas de uso y ciclo de vida. Si nadie es responsable del dato, la organizacion acaba delegando decisiones sustantivas en tecnicos, proveedores o usos informales. La responsabilidad no significa que una persona introduzca todos los registros, sino que existe una autoridad funcional capaz de definir el dato correcto, resolver conflictos, priorizar mejoras y responder ante auditorias.

El tercer principio es calidad proporcional al riesgo y al uso. No todos los datos requieren el mismo nivel de control. Un dato usado para conceder una ayuda, imponer una sancion, identificar una persona o publicar indicadores oficiales exige controles mas estrictos que un dato exploratorio. La calidad no es un ideal abstracto; debe medirse respecto a dimensiones como exactitud, completitud, actualidad, consistencia, unicidad, validez, trazabilidad y disponibilidad.

El cuarto principio es interoperabilidad desde el diseno. El dato debe poder circular entre organos, Administraciones y sistemas respetando competencias, bases juridicas, seguridad y proteccion de datos. La interoperabilidad no empieza al construir una API, sino al definir conceptos comunes, codigos, identificadores, metadatos, formatos, reglas de intercambio y acuerdos organizativos. Si cada unidad define "beneficiario", "domicilio", "centro", "empresa" o "expediente" de manera distinta, la integracion tecnica solo mueve inconsistencias.

El quinto principio es minimizacion y necesidad. La Administracion debe evitar pedir, conservar o difundir mas datos de los necesarios para la finalidad legitima. Este principio es critico cuando hay datos personales o categorias especialmente sensibles, pero tambien sirve para reducir costes, deuda tecnica y exposicion a errores. Un buen gobierno del dato no acumula datos por reflejo; decide que datos son necesarios, durante cuanto tiempo, bajo que reglas y con que controles.

El sexto principio es transparencia interna y externa. Internamente, los equipos deben saber que datos existen, que significan, quien los gestiona, que calidad tienen y como pueden solicitar acceso. Externamente, cuando proceda, la ciudadania y los reutilizadores deben encontrar informacion publica comprensible, actualizada y en condiciones claras. La transparencia no elimina los limites por proteccion de datos, seguridad, secreto estadistico, propiedad intelectual o intereses publicos protegidos.

El septimo principio es trazabilidad. Debe ser posible reconstruir el origen de un dato, las transformaciones que ha sufrido, las reglas aplicadas, los intercambios realizados y las decisiones relevantes. La trazabilidad permite auditar, corregir errores, explicar resultados automatizados, depurar responsabilidades y mejorar procesos. Sin linaje, un indicador puede parecer objetivo aunque provenga de extracciones manuales, equivalencias no documentadas o reglas cambiadas sin control.

El octavo principio es reutilizacion segura y ordenada. La informacion publica tiene valor social y economico cuando puede ser reutilizada, pero esa reutilizacion debe respetar derechos, limites normativos, calidad minima, condiciones de licencia, metadatos adecuados y actualizacion sostenible. Publicar un conjunto de datos sin descripcion, sin fecha, sin responsable y sin formato estable no es apertura real; es traslado de incertidumbre al reutilizador.

## 3. Definiciones operativas

Dato es una representacion registrada de un hecho, atributo, medida, relacion o evento. En la Administracion puede representar una persona interesada, una solicitud, un acto administrativo, una resolucion, una fecha, una direccion, un pago, una infraccion, un contrato, un inmueble o un indicador. El dato por si solo no garantiza significado: necesita contexto, reglas y metadatos.

Informacion es dato interpretado en un contexto. Por ejemplo, una fecha aislada es un dato; la fecha de presentacion de una solicitud dentro de un expediente es informacion que puede determinar plazos, admision, silencio administrativo o efectos juridicos.

Conjunto de datos es una coleccion organizada de datos que se gestiona como unidad. Puede ser una tabla, un fichero, una vista, una API, un registro administrativo, una serie estadistica o un conjunto publicado para reutilizacion. En gobierno del dato interesa identificarlo por nombre, responsable, finalidad, campos principales, calidad, restricciones, ciclo de vida y condiciones de acceso.

Dato maestro es un dato nuclear, compartido y relativamente estable sobre entidades esenciales: personas, organos, unidades administrativas, expedientes, procedimientos, territorios, centros, proveedores, activos o codigos. Su mala gestion provoca duplicidades e incoherencias en muchos sistemas.

Dato de referencia es un conjunto de valores normalizados que otros sistemas usan para clasificar o validar informacion: codigos de municipios, tipos de via, catalogos de procedimientos, clasificaciones economicas, unidades organicas, estados de expediente o tipos documentales. Su valor esta en que reduce ambiguedad y facilita interoperabilidad.

Metadato es dato sobre el dato. Describe significado, estructura, origen, responsable, calidad, formato, licencia, fecha de actualizacion, periodo temporal, cobertura territorial, sensibilidad, restricciones, esquema, vocabulario o linaje. Un catalogo sin metadatos suficientes no permite localizar ni evaluar un conjunto de datos.

Linaje del dato es la descripcion de su recorrido desde el origen hasta su uso: fuentes, cargas, transformaciones, reglas de negocio, validaciones, agregaciones, cambios de formato, intercambios y consumos. El linaje puede documentarse a nivel de conjunto, tabla, campo, API, proceso o indicador.

Stewardship es la funcion de custodia funcional del dato. El data steward no suele ser el propietario juridico ni el administrador tecnico, sino la persona o equipo que vela por definiciones, reglas, calidad, incidencias, documentacion y uso correcto en un dominio concreto.

Dominio de datos es un area funcional o conceptual con datos relacionados y responsabilidades coherentes. En una Administracion pueden existir dominios como poblacion y territorio, expedientes y procedimientos, recursos humanos, contratacion, subvenciones, tributario, servicios sociales, salud publica, urbanismo, educacion, movilidad, medio ambiente o transparencia.

Catalogo de datos es el inventario gobernado de conjuntos de datos, fuentes, APIs, informes, indicadores y recursos relacionados. Debe permitir descubrir datos, conocer su significado, responsable, calidad, restricciones y condiciones de uso. Puede tener una parte interna para gestion y otra publica para reutilizacion.

Politica de datos es una norma interna de alto nivel que fija principios, roles, decisiones, obligaciones y controles sobre los datos. Puede desarrollarse en procedimientos, guias, modelos de metadatos, criterios de calidad, reglas de acceso, instrucciones de conservacion y acuerdos de intercambio.

## 4. Roles y responsabilidades

El gobierno del dato requiere una distribucion clara de responsabilidades. En una Administracion, la dificultad habitual es que los datos atraviesan organos con competencias distintas. La unidad que captura un dato puede no ser la que lo usa para un indicador, lo publica como dato abierto, lo intercambia con otra Administracion o responde ante una reclamacion. Por eso conviene separar propiedad funcional, custodia, administracion tecnica, seguridad, proteccion de datos y consumo.

La alta direccion o el organo de gobierno impulsa la estrategia de datos, aprueba la politica general, resuelve prioridades y garantiza recursos. Su papel no es definir cada campo, sino convertir el dato en asunto institucional. Sin patrocinio directivo, el gobierno del dato suele quedar reducido a iniciativas tecnicas aisladas.

El responsable funcional del dato, tambien llamado data owner en algunas organizaciones, decide sobre el significado, reglas de negocio, nivel de calidad esperado y usos permitidos dentro de su competencia. Debe poder validar definiciones, aceptar indicadores, priorizar correcciones y participar en acuerdos de intercambio. En el sector publico, esta figura debe alinearse con competencias organicas y con responsabilidad sobre procedimientos o servicios.

El data steward o custodio funcional trabaja en la gestion cotidiana del dato. Revisa definiciones, mantiene metadatos, analiza incidencias, propone reglas de calidad, coordina usuarios, documenta excepciones y ayuda a que los datos se usen correctamente. Es una figura especialmente util porque traduce entre el lenguaje del negocio publico y el lenguaje tecnico.

El responsable tecnico o administrador de plataforma gestiona sistemas, bases de datos, integraciones, rendimiento, copias, disponibilidad, despliegues y controles tecnicos. No deberia decidir por si solo el significado juridico o funcional de los datos, aunque su conocimiento es imprescindible para evaluar viabilidad, riesgos y automatizaciones.

El responsable de seguridad vela por confidencialidad, integridad, disponibilidad, autenticidad y trazabilidad tecnica. Interviene en clasificacion de activos, controles de acceso, registro de actividad, continuidad, cifrado, gestion de vulnerabilidades y requisitos de seguridad en intercambios.

El delegado o responsable de proteccion de datos asesora y supervisa el cumplimiento cuando hay tratamientos de datos personales. Su funcion no consiste en bloquear cualquier uso de datos, sino en asegurar que existe base juridica, informacion adecuada, minimizacion, limitacion de finalidad, medidas de seguridad, gestion de derechos y evaluacion de riesgos cuando proceda.

La unidad de transparencia, reutilizacion o datos abiertos participa en publicacion de informacion, licencias, formatos, metadatos publicos, demandas de reutilizadores, actualizacion de datasets y coherencia con portales de datos. Debe coordinarse con las unidades responsables para no publicar informacion descontextualizada o con riesgos no evaluados.

Los consumidores de datos son unidades, empleados publicos, sistemas, cuadros de mando, servicios externos, reutilizadores o ciudadanos que usan datos. Tambien tienen responsabilidades: respetar condiciones de uso, no alterar significado, comunicar errores, no crear copias opacas y no usar datos para finalidades incompatibles.

| Rol | Decision principal | Evidencia que debe producir | Riesgo si falta |
| --- | --- | --- | --- |
| Alta direccion | Prioridades, politica y recursos | Politica de datos, comite, hoja de ruta | Iniciativas aisladas sin autoridad |
| Responsable funcional | Significado, reglas y calidad esperada | Glosario, reglas de negocio, criterios de aceptacion | Definiciones contradictorias |
| Data steward | Custodia diaria y mejora | Metadatos, incidencias, reglas de calidad, propuestas | Catalogo desactualizado y errores recurrentes |
| Responsable tecnico | Operacion de sistemas e integraciones | Arquitectura, controles tecnicos, logs, disponibilidad | Dependencia de soluciones no gobernadas |
| Seguridad | Proteccion tecnica y continuidad | Clasificacion, controles de acceso, auditoria | Exposicion, manipulacion o indisponibilidad |
| Proteccion de datos | Cumplimiento en datos personales | Analisis de base juridica, riesgos y garantias | Tratamientos excesivos o incompatibles |
| Transparencia y reutilizacion | Publicacion y condiciones de uso | Fichas publicas, licencias, plan de actualizacion | Apertura inutil, insegura o no sostenible |
| Consumidores | Uso correcto y retorno de calidad | Solicitudes, reportes de error, justificacion de uso | Copias paralelas y decisiones no trazables |

## 5. Stewardship: la funcion de custodia

El stewardship es una de las piezas mas importantes porque evita que el gobierno del dato quede en organigramas formales. Un steward eficaz conoce el procedimiento, entiende como se captura y usa el dato, identifica errores frecuentes y puede hablar con juristas, tecnicos, gestores, estadisticos y responsables de transparencia. Su autoridad debe estar reconocida, aunque no siempre tenga jerarquia sobre todos los actores.

Sus tareas habituales son mantener el glosario de negocio, revisar fichas del catalogo, definir reglas de calidad, clasificar criticidad, validar cambios de campos, coordinar incidencias, proponer controles preventivos, documentar excepciones y participar en pruebas de integracion. Tambien debe detectar datos duplicados, obsoletos, ambiguos o usados fuera de contexto.

Un buen ejemplo es el dato "domicilio de notificacion". Puede capturarse en registros de entrada, sedes electronicas, padrones, sistemas tributarios o expedientes sectoriales. Si no existe stewardship, distintas unidades pueden usar el mismo nombre para conceptos distintos: domicilio fiscal, domicilio padronal, direccion postal, direccion electronica habilitada o direccion declarada en una solicitud. El resultado son notificaciones fallidas, duplicidad de peticiones a la ciudadania y problemas de interoperabilidad.

El steward no debe convertirse en cuello de botella. Para evitarlo, se pueden definir niveles de decision: cambios menores de metadatos, correcciones de calidad ordinarias, cambios de definicion, altas de conjuntos criticos, autorizaciones de acceso y publicaciones externas. Cada nivel requiere aprobacion distinta. Asi se mantiene control sin paralizar la operacion.

## 6. Dominios de datos y modelo federado

En organizaciones publicas grandes es inviable que una unica oficina central conozca todos los datos con detalle. Por eso suele ser adecuado un modelo federado: una unidad central de gobierno del dato define politica, estandares, metodologia, herramientas, indicadores y seguimiento; los dominios funcionales mantienen la responsabilidad sobre sus propios datos; y los comites resuelven conflictos transversales.

El dominio de datos agrupa informacion por sentido funcional. Por ejemplo, el dominio "procedimientos y expedientes" puede definir identificadores de expediente, estados, fases, organos responsables, tipos documentales y eventos administrativos. El dominio "territorio" puede mantener unidades geograficas, codigos, direcciones, parcelas o zonas. El dominio "subvenciones" puede gestionar convocatorias, beneficiarios, solicitudes, concesiones, justificaciones y reintegros.

El modelo federado exige reglas comunes. Si cada dominio actua de forma aislada, se reproduce la fragmentacion. Deben existir modelos comunes para metadatos, identificadores, clasificacion de sensibilidad, criterios de calidad, nomenclatura, versionado, APIs, licencias, conservacion y linaje. Tambien debe existir un mecanismo para resolver conceptos compartidos. "Persona", "empresa", "representante", "unidad organica" o "expediente" no pertenecen de forma exclusiva a un solo dominio.

Un error habitual es crear dominios segun sistemas informaticos. El sistema es una implementacion; el dominio es una responsabilidad funcional. Puede haber varios sistemas dentro de un dominio y un sistema puede contener datos de varios dominios. Para examen, conviene recordar que el gobierno del dato debe organizarse por significado y responsabilidad, no solo por tecnologia.

## 7. Catalogos de datos y metadatos

El catalogo es la puerta de entrada al gobierno del dato. Permite saber que datos existen, quien los gestiona, que significan, donde estan, que calidad tienen y como se pueden usar. En una Administracion publica, el catalogo interno sirve para gestion, interoperabilidad, analitica, seguridad y control. El catalogo publico sirve para transparencia y reutilizacion, pero solo debe mostrar lo que sea publicable y con el nivel de detalle adecuado.

Una ficha minima de catalogo deberia incluir nombre del conjunto, descripcion, dominio, responsable funcional, steward, sistema origen, finalidad, base o habilitacion de uso cuando proceda, cobertura temporal, cobertura territorial, frecuencia de actualizacion, formato, campos principales, identificadores, vocabularios usados, calidad conocida, restricciones de acceso, nivel de sensibilidad, condiciones de reutilizacion, relacion con otros conjuntos y contacto funcional.

Los metadatos deben ser utiles, no decorativos. Si el catalogo solo contiene nombres tecnicos de tablas, nadie ajeno al equipo de sistemas podra usarlo. Si solo contiene textos generales sin campos, reglas ni responsables, no servira para integracion. La buena practica es combinar metadatos descriptivos, estructurales, administrativos, juridicos, tecnicos y de calidad.

Los metadatos descriptivos explican que es el conjunto y para que sirve. Los estructurales describen campos, tipos, relaciones, codigos y formatos. Los administrativos recogen responsable, fechas, version, frecuencia y conservacion. Los juridicos indican restricciones, licencias, datos personales, confidencialidad o limites de reutilizacion. Los tecnicos describen ubicacion logica, API, esquema, carga o dependencias. Los de calidad muestran completitud, actualidad, errores, validaciones y advertencias.

La actualizacion del catalogo debe integrarse en procesos de cambio. Cada alta de sistema, modificacion de modelo, nueva API, publicacion de dataset o nuevo indicador deberia exigir revisar metadatos. Si el catalogo se mantiene mediante campanas manuales anuales, quedara obsoleto. La gobernanza madura convierte el catalogo en parte del ciclo de vida de soluciones y servicios.

## 8. Linaje y trazabilidad

El linaje permite explicar de donde procede un dato y como llega a un uso determinado. En el sector publico esto es especialmente relevante por tres razones. Primero, porque muchas decisiones afectan a derechos o cargas de la ciudadania. Segundo, porque los indicadores publicos deben ser explicables y comparables. Tercero, porque los intercambios entre Administraciones requieren confianza sobre origen, vigencia y transformaciones.

El linaje puede ser tecnico o funcional. El linaje tecnico muestra flujos entre bases, procesos ETL, APIs, colas, ficheros y transformaciones. El linaje funcional explica reglas de negocio: como se calcula un indicador, que expedientes entran, que fechas se toman, que exclusiones se aplican, que version de clasificacion se usa y quien aprobo el criterio. Ambos son necesarios. Un diagrama tecnico sin reglas de negocio no explica la decision; una descripcion funcional sin trazabilidad tecnica no permite auditar el recorrido real.

Un ejemplo: un cuadro de mando sobre tiempo medio de resolucion de licencias puede usar datos de registro, gestor de expedientes, notificaciones y archivo. Para que el indicador sea confiable debe documentarse que fecha inicia el computo, que expedientes se excluyen, como se tratan suspensiones, desistimientos o caducidades, que unidad valida la regla, que sistema aporta cada campo y cuando se actualiza. Sin linaje, dos informes pueden ofrecer cifras distintas y ambos parecer plausibles.

El linaje tambien ayuda en cambios normativos o funcionales. Si cambia la definicion de un estado de expediente, el linaje permite identificar informes, APIs, datasets y procesos afectados. Si una fuente tiene errores, permite saber que productos derivados deben revisarse. Por eso no es una documentacion secundaria, sino una herramienta de control institucional.

## 9. Ciclo de vida del dato

El gobierno del dato debe cubrir todo el ciclo de vida. La primera fase es planificacion: decidir que datos son necesarios, para que finalidad, con que base, con que calidad y bajo que responsabilidad. En esta fase se evitan muchos problemas posteriores, como pedir datos innecesarios, crear campos ambiguos o duplicar fuentes ya existentes.

La segunda fase es captura o adquisicion. El dato puede ser aportado por la ciudadania, generado por la Administracion, consultado a otra Administracion, recibido de un sensor, importado de un registro, obtenido de una declaracion responsable o derivado de otro dato. En esta fase importan validaciones de entrada, normalizacion, identificadores, controles de duplicidad, informacion al interesado cuando proceda y registro de origen.

La tercera fase es almacenamiento y organizacion. Incluye modelos de datos, repositorios, clasificacion, seguridad, copias, versionado y conservacion. No basta con guardar el dato; debe almacenarse de forma que conserve significado, integridad, disponibilidad y trazabilidad.

La cuarta fase es uso y transformacion. Aqui se producen consultas, tramitacion, calculos, informes, automatizaciones, intercambios, analitica o decisiones. Deben aplicarse reglas de acceso, calidad, minimizacion, control de cambios y documentacion de transformaciones.

La quinta fase es intercambio y publicacion. Puede haber comunicacion entre organos, intermediacion de datos, APIs, remision a otras Administraciones, publicacion de indicadores o apertura para reutilizacion. Esta fase requiere acuerdos, metadatos, formatos, medidas de seguridad, condiciones de uso y control de version.

La sexta fase es archivo, conservacion o eliminacion. Los datos publicos no se conservan indefinidamente por defecto ni se eliminan de manera arbitraria. Deben aplicarse calendarios de conservacion, normativa de archivos, necesidades de evidencia, limitaciones de finalidad, proteccion de datos, valor historico y reglas de expurgo cuando proceda.

| Fase | Pregunta de gobierno | Control recomendado | Ejemplo administrativo |
| --- | --- | --- | --- |
| Planificacion | Que dato se necesita y por que | Analisis de finalidad, responsable y calidad esperada | Disenar un nuevo procedimiento de ayudas |
| Captura | Como se obtiene y valida | Formularios con validaciones, consulta a fuentes autenticas | Alta de una solicitud en sede electronica |
| Almacenamiento | Donde se conserva y con que proteccion | Modelo, clasificacion, permisos, copias | Registro de expedientes y documentos |
| Uso | Quien lo usa y bajo que reglas | Perfiles, trazas, reglas de negocio, control de cambios | Calculo de prioridad en una inspeccion |
| Intercambio | Con quien se comparte | Acuerdo, API, metadatos, seguridad, logs | Consulta de datos de identidad o residencia |
| Publicacion | Puede reutilizarse | Anonimizacion si procede, licencia, ficha publica | Dataset de contratos o subvenciones |
| Conservacion | Cuanto tiempo permanece | Calendario, archivo, expurgo, bloqueo | Archivo de expedientes finalizados |

## 10. Politicas de gobierno del dato

La politica de datos es el documento marco que fija como se gobiernan los datos en la organizacion. Debe ser clara, operativa y aprobada con autoridad suficiente. Si es demasiado generica, no cambiara comportamientos. Si es demasiado tecnica, no sera asumida por las unidades funcionales. Su valor esta en conectar principios, roles, procesos y controles.

Una politica completa puede incluir ambito de aplicacion, principios, clasificacion de datos, roles, modelo de dominios, criterios de catalogacion, reglas de metadatos, calidad, acceso, intercambio, reutilizacion, proteccion de datos, seguridad, conservacion, linaje, gestion de incidencias, comites, indicadores y regimen de revision. Debe indicar tambien que decisiones se elevan al comite de datos y cuales se resuelven en cada dominio.

Las politicas especificas desarrollan el marco general. Por ejemplo, una politica de calidad de datos define dimensiones, umbrales, controles, responsables y tratamiento de errores. Una politica de metadatos define campos obligatorios, vocabularios y reglas de actualizacion. Una politica de acceso define perfiles, autorizaciones, segregacion de funciones y revision periodica. Una politica de datos abiertos define criterios de publicacion, formatos, licencias, actualizacion y retirada.

La politica debe acompanar al cambio cultural. Muchos problemas de datos no se deben a falta de tecnologia, sino a incentivos organizativos: cada unidad protege "sus" datos, se crean hojas de calculo paralelas, se corrigen errores solo en destino, se piden documentos que ya obran en poder de la Administracion o se publican indicadores sin responsable claro. El gobierno del dato transforma esas practicas mediante reglas, liderazgo y beneficios visibles.

## 11. Comites y mecanismos de decision

El comite de gobierno del dato es el foro donde se priorizan iniciativas, se resuelven conflictos y se supervisa la implantacion. Debe incluir perfiles directivos, funcionales, tecnologicos, juridicos, seguridad, proteccion de datos, transparencia/reutilizacion y, cuando proceda, archivo o estadistica. No debe ser una reunion informativa sin capacidad de decision.

Sus funciones pueden incluir aprobar estandares, priorizar dominios, validar politicas, resolver conflictos de definicion, impulsar catalogacion, revisar indicadores de calidad, aprobar planes de mejora, coordinar reutilizacion, supervisar riesgos y elevar decisiones a organos superiores. La frecuencia debe ser suficiente para mantener avance, pero no tan alta que convierta el gobierno del dato en burocracia.

Ademas del comite central pueden existir grupos de dominio. Estos grupos trabajan el detalle: glosarios, reglas de calidad, fuentes, incidencias, cambios de campos, necesidades de intercambio y propuestas de publicacion. El comite central no deberia decidir si un campo concreto admite un valor especifico salvo que exista conflicto relevante; esa decision corresponde al dominio.

Para que los comites funcionen, deben basarse en evidencias: fichas de catalogo, metricas de calidad, incidencias, riesgos, demanda de datos, duplicidades, costes y casos de uso. Un comite sin datos sobre los datos termina discutiendo percepciones. La gobernanza madura mide su propia eficacia.

## 12. Calidad del dato como responsabilidad gobernada

La calidad del dato no se corrige al final con depuraciones masivas. Debe incorporarse desde la captura, los procesos, las integraciones y las reglas de negocio. En una Administracion publica, la calidad debe medirse segun el impacto: no es lo mismo un error tipografico en una descripcion interna que un identificador incorrecto en una resolucion, una fecha mal calculada en un plazo o una duplicidad en beneficiarios.

Las dimensiones habituales son exactitud, completitud, consistencia, actualidad, unicidad, validez, integridad referencial, trazabilidad y comprensibilidad. Exactitud significa que el dato refleja correctamente la realidad o fuente autentica. Completitud significa que estan presentes los campos necesarios. Consistencia significa que no hay contradicciones entre sistemas o campos. Actualidad significa que el dato esta vigente para el uso previsto. Unicidad evita registros duplicados. Validez comprueba que el dato respeta formato, rango o vocabulario. Trazabilidad permite conocer origen y cambios. Comprensibilidad permite que usuarios autorizados interpreten el dato sin ambiguedad.

La calidad debe tener umbrales y responsables. Decir "mejorar la calidad" no basta. Es mas util fijar reglas: porcentaje maximo de registros sin identificador, plazo de correccion de incidencias, campos obligatorios, control de duplicados, comparacion con fuente de referencia o validacion automatica de codigos. Cuando una regla falla, debe existir un circuito: deteccion, registro, asignacion, correccion, validacion y aprendizaje.

Un aspecto importante para examen es distinguir calidad de dato y calidad de sistema. Un sistema puede funcionar tecnicamente y contener datos malos. Tambien puede haber datos correctos en un sistema dificil de explotar por falta de metadatos. El gobierno del dato se ocupa del valor y fiabilidad del dato, no solo de la disponibilidad de la aplicacion.

## 13. Gobierno del dato, interoperabilidad y reutilizacion

La interoperabilidad semantica depende directamente del gobierno del dato. Para intercambiar informacion de manera eficaz, las Administraciones deben compartir significados, identificadores, formatos, codigos y reglas. Una API tecnicamente correcta puede ser inutil si el campo "estado" no tiene una definicion comun o si la fecha de referencia cambia segun el sistema.

La interoperabilidad organizativa requiere acuerdos sobre responsabilidades, competencias, procedimientos y niveles de servicio. La interoperabilidad semantica exige modelos de informacion, vocabularios, glosarios y metadatos. La interoperabilidad tecnica aporta protocolos, formatos, servicios, APIs, seguridad y plataformas. El gobierno del dato cruza las tres dimensiones porque define quien mantiene significado, calidad, acceso y evolucion.

La reutilizacion de informacion publica exige datos localizables, comprensibles, actualizados y con condiciones claras. El catalogo publico, la ficha de metadatos, el formato abierto, la licencia, la frecuencia de actualizacion y la documentacion de campos son componentes de gobierno del dato. Sin ellos, el reutilizador no puede valorar si el conjunto sirve para crear servicios, investigacion, control ciudadano o analisis economico.

Tambien debe gobernarse la frontera entre apertura y proteccion. No todo dato publico es publicable. Hay que ponderar proteccion de datos personales, confidencialidad, seguridad, secreto estadistico, derechos de terceros y otros limites. Cuando se publiquen datos agregados, anonimizados o disociados, debe evaluarse el riesgo de reidentificacion, especialmente si se combinan con otros conjuntos disponibles.

## 14. Casos practicos guiados

### Caso 1: catalogar datos de subvenciones

Situacion: una consejeria quiere mejorar la transparencia de subvenciones y preparar un conjunto reutilizable con convocatorias, solicitudes, beneficiarios, importes concedidos, pagos y justificaciones. Actualmente cada servicio mantiene hojas auxiliares y el sistema de gestion no tiene metadatos completos.

Pistas: hay datos personales, informacion economica, obligaciones de transparencia, posible reutilizacion, fases de procedimiento y riesgo de inconsistencias entre concesion, pago y justificacion.

Resolucion: primero se identifica el dominio "subvenciones" y el responsable funcional. Despues se nombra un steward que documente conceptos: convocatoria, solicitud, beneficiario, entidad colaboradora, importe solicitado, importe concedido, pago, justificacion, reintegro y estado. A continuacion se crea una ficha de catalogo con origen, campos, frecuencia, cobertura temporal, restricciones y calidad conocida. Se definen reglas de calidad: identificador unico de convocatoria, relacion entre solicitud y beneficiario, importes no negativos, estados normalizados y fechas coherentes. Se separa la vista interna, que puede contener datos necesarios para gestion, de la vista publica, que debe respetar limites de proteccion y transparencia. Finalmente se establece un plan de actualizacion y un circuito de incidencias.

Errores a evitar: publicar una hoja exportada sin responsable, mezclar conceptos de concesion y pago, omitir fecha de actualizacion, usar nombres de campos incomprensibles o no evaluar datos personales.

### Caso 2: linaje de un indicador de tiempo medio de resolucion

Situacion: el organo directivo solicita un indicador de tiempo medio de resolucion de expedientes. Dos unidades ofrecen cifras distintas para el mismo periodo.

Pistas: puede haber diferencias en fecha de inicio, fecha final, expedientes suspendidos, desistimientos, anulaciones, cambios de estado o datos incompletos.

Resolucion: se crea una definicion funcional aprobada: que expedientes entran, que fecha inicia el computo, que fecha lo cierra, como se tratan suspensiones, que periodo se informa y que unidad es responsable. Se documenta el linaje tecnico: sistema origen, campos usados, transformaciones, filtros y proceso de calculo. Se comparan resultados con muestras de expedientes y se registran reglas de calidad. La ficha del indicador incluye version de definicion y fecha de aprobacion. Si cambia la norma o el procedimiento, se versiona el indicador para no romper series historicas.

Errores a evitar: elegir la cifra que "parece mejor", cambiar reglas sin versionado, no documentar exclusiones o confundir tiempo de tramitacion con tiempo de resolucion administrativa.

### Caso 3: dominio comun de personas interesadas

Situacion: varios sistemas tienen registros de personas interesadas con identificadores, nombres, domicilios y representantes. Hay duplicidades y notificaciones fallidas.

Pistas: existen datos maestros, datos personales, necesidad de exactitud, distintas fuentes y reglas sobre representacion y notificacion.

Resolucion: se define el dominio de personas interesadas y se identifican datos maestros: identificador, nombre, tipo de persona, domicilio a efectos concretos, medio de notificacion, representante y fecha de vigencia. Se aclara que no todos los domicilios son equivalentes. Se establecen fuentes de referencia, reglas de validacion, mecanismo de actualizacion, gestion de duplicados y trazabilidad de cambios. El steward coordina unidades afectadas y documenta excepciones. Los accesos se limitan segun finalidad y competencia.

Errores a evitar: fusionar registros sin criterio, usar un domicilio para finalidades distintas, tratar representante como simple campo textual o permitir copias locales sin actualizacion.

## 15. Notas de test separadas

Estas notas no sustituyen al desarrollo teorico. Sirven para reconocer preguntas de examen y evitar trampas habituales.

1. Si una pregunta contrapone gobierno del dato y gestion tecnica, la respuesta correcta suele destacar que el gobierno fija responsabilidades, reglas y decisiones, mientras la tecnologia las implementa.

2. Si se pregunta por el catalogo de datos, no basta con decir "inventario". Debe incluir metadatos, responsable, significado, condiciones de acceso, calidad, actualizacion y restricciones.

3. Si aparece "linaje", pensar en origen, transformaciones, reglas y usos. No confundirlo con copia de seguridad ni con historico de versiones de una aplicacion.

4. Si se pregunta por stewardship, la idea clave es custodia funcional del dato: definiciones, calidad, metadatos e incidencias. No es necesariamente el administrador de base de datos.

5. Si se pregunta por interoperabilidad semantica, la respuesta debe hablar de significados compartidos, modelos, vocabularios, codigos y metadatos, no solo de APIs o protocolos.

6. Si se pregunta por reutilizacion, recordar que la apertura requiere condiciones claras, formatos adecuados, metadatos y respeto a limites juridicos. Publicar sin contexto no es buena reutilizacion.

7. Si se pregunta por calidad, buscar dimensiones: exactitud, completitud, consistencia, actualidad, unicidad, validez y trazabilidad. La calidad depende del uso y del riesgo.

8. Si se plantea un caso con varios sistemas que usan conceptos distintos, la medida inicial no deberia ser "integrarlos todos" sin mas, sino acordar definiciones, responsables, datos maestros y reglas de calidad.

9. Si se plantea un dato personal, no basta invocar transparencia o reutilizacion. Hay que analizar finalidad, base juridica, minimizacion, limites de acceso, seguridad y posible anonimizacion o disociacion.

10. Si se pregunta por comites de datos, la opcion mas completa incluye decision, priorizacion, resolucion de conflictos, supervision de calidad y aprobacion de estandares.

## 16. Errores frecuentes

Confundir gobierno del dato con una herramienta. Un catalogo, una plataforma de calidad o un lago de datos pueden ayudar, pero no sustituyen roles, decisiones, politicas y responsabilidades.

Pensar que el dato pertenece al departamento que lo almacena. En la Administracion, la responsabilidad debe vincularse a competencias, finalidades y procesos, no solo al servidor o aplicacion donde reside.

Crear catalogos sin mantenimiento. Un catalogo desactualizado genera desconfianza y puede ser peor que no tenerlo, porque induce a usar datos bajo supuestos falsos.

Documentar solo nombres tecnicos. Si los metadatos no explican significado funcional, reglas, restricciones y calidad, no sirven para usuarios, auditores ni reutilizadores.

Corregir errores solo en destino. Arreglar manualmente un informe sin corregir la fuente, la regla o el proceso perpetua el problema y rompe la trazabilidad.

Publicar datos sin evaluar riesgos. La reutilizacion no elimina proteccion de datos, seguridad, confidencialidad ni limites legales. La apertura requiere analisis y diseno.

No versionar definiciones. Si cambia el calculo de un indicador o el significado de un estado, debe quedar constancia. Sin versionado no se pueden comparar periodos ni explicar discrepancias.

Confundir dato maestro con dato mas importante. Un dato maestro es compartido, estable y referencial para muchos procesos. Puede ser criticamente importante, pero la clave esta en su funcion transversal.

Organizar dominios por aplicaciones. Los dominios deben reflejar significado y responsabilidad funcional. Las aplicaciones cambian; los conceptos administrativos suelen tener continuidad.

Reducir calidad a "datos completos". La completitud es solo una dimension. Un dato puede estar completo y ser incorrecto, inconsistente, obsoleto o no trazable.

## 17. Mini esquema de repaso

El gobierno del dato responde a cinco preguntas: que datos existen, que significan, quien responde por ellos, con que calidad se gestionan y bajo que condiciones se usan o comparten.

La Administracion necesita gobierno del dato porque sus datos sostienen derechos, obligaciones, expedientes, servicios, transparencia, interoperabilidad, reutilizacion y decisiones publicas.

Los pilares practicos son politica de datos, roles, dominios, catalogo, metadatos, calidad, linaje, controles de acceso, comites y mejora continua.

El steward es el puente entre negocio publico y tecnologia. Mantiene definiciones, reglas, metadatos e incidencias del dominio.

El catalogo no es un listado de tablas. Es un instrumento de descubrimiento, responsabilidad, calidad, acceso y reutilizacion.

El linaje explica origen y transformaciones. Es imprescindible para auditar indicadores, decisiones, intercambios y datos publicados.

La reutilizacion solo es madura si los datos son comprensibles, actualizados, documentados, juridicamente utilizables y sostenibles.

## 18. Referencias normativas y de marco sin URL visible

Constituyen referencias utiles para integrar este bloque en un tema completo: Reglamento General de Proteccion de Datos; Ley Organica 3/2018, de Proteccion de Datos Personales y garantia de los derechos digitales; Ley 39/2015, del Procedimiento Administrativo Comun de las Administraciones Publicas; Ley 40/2015, de Regimen Juridico del Sector Publico; Real Decreto 4/2010, por el que se regula el Esquema Nacional de Interoperabilidad; Real Decreto 203/2021, de actuacion y funcionamiento del sector publico por medios electronicos; Ley 37/2007, sobre reutilizacion de la informacion del sector publico; Real Decreto 1495/2011, de desarrollo de la reutilizacion en el sector publico estatal; Reglamento europeo de gobernanza de datos; Reglamento europeo sobre la Europa Interoperable; normativa de transparencia estatal y autonomica; normativa de archivos y patrimonio documental que resulte aplicable.
