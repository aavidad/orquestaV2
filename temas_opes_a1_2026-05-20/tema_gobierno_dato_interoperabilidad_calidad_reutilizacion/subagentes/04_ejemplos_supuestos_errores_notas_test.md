# Bloque parcial: ejemplos, supuestos practicos, errores frecuentes y notas de test

Este material esta pensado para integrarse en un tema A1 sobre gobierno del dato, interoperabilidad semantica, calidad del dato y reutilizacion de la informacion publica. No sustituye al desarrollo teorico: aporta casos, cuadros de aplicacion, supuestos guiados, avisos de examen y muestras de preguntas que deben colocarse en apartados separados del texto principal.

## 1. Orientacion de examen para este bloque

En un examen A1, esta materia suele evaluarse por cruce de conceptos. La pregunta rara vez se limita a pedir una definicion aislada de dato abierto, metadato o interoperabilidad semantica. Lo habitual es que plantee una situacion administrativa y exija reconocer que instrumento juridico, organizativo o tecnico resuelve el problema.

La clave es distinguir cuatro planos. El primero es el gobierno del dato: quien decide, quien custodia, quien define, quien mide calidad, quien autoriza usos y quien responde ante riesgos. El segundo es la interoperabilidad, que permite que administraciones y sistemas se entiendan, no solo que se conecten. El tercer plano es la calidad, que convierte el dato en un activo util, trazable y fiable. El cuarto es la reutilizacion, que permite que la informacion publica se use de nuevo, con limites, condiciones y formatos adecuados.

Una buena respuesta no dice simplemente "hay que publicar los datos". Primero identifica si el dato debe circular dentro de la Administracion para tramitar un procedimiento, si debe ponerse a disposicion de otra administracion, si debe abrirse a terceros para reutilizacion, si contiene datos personales, si esta sometido a confidencialidad, si pertenece a una categoria de alto valor, si exige una API o si requiere un vocabulario comun para ser interpretado.

### Mapa mental aplicable a casos

| Si el caso habla de... | Pensar primero en... | Peligro tipico |
| --- | --- | --- |
| Un ciudadano al que se le pide un certificado ya disponible en otra administracion | Intermediacion de datos, derecho a no aportar documentos ya obrantes y transmision entre administraciones | Responder con "portal de datos abiertos" cuando el problema es de tramitacion |
| Varias consejerias que usan nombres distintos para el mismo concepto | Interoperabilidad semantica, vocabularios, modelo de datos comun, metadatos | Quedarse en integracion tecnica mediante API |
| Un portal que publica hojas de calculo con campos ambiguos | Calidad, metadatos, formatos reutilizables y DCAT-AP/DCAT-AP-ES | Confundir publicar un archivo con publicar datos reutilizables |
| Una empresa que quiere explotar datos meteorologicos o de movilidad | Reutilizacion, datos abiertos, conjuntos de alto valor, API, licencias | Olvidar limites por proteccion de datos, seguridad o derechos de terceros |
| Un cuadro de mando interno que da cifras distintas segun el departamento | Gobierno del dato, dato maestro, linaje, criterios de calidad y responsabilidades | Culpar solo a la herramienta de visualizacion |
| Una base con datos personales que se quiere abrir al publico | Minimizacion, anonimacion real, finalidad, base juridica y evaluacion de riesgos | Decir que basta con quitar el nombre |

## 2. Ejemplos integrables en el desarrollo teorico

### Ejemplo 1. Ayudas publicas y dato unico del solicitante

Una comunidad autonoma convoca ayudas para rehabilitacion energetica de viviendas. El formulario pide al solicitante que aporte DNI, volante de empadronamiento, certificado de discapacidad, acreditacion de familia numerosa, datos catastrales y certificado de estar al corriente de obligaciones tributarias. En la practica, muchos de esos datos ya obran en poder de administraciones publicas.

El enfoque correcto no es multiplicar documentos adjuntos, sino revisar el procedimiento desde el principio de no pedir lo que ya puede consultarse por medios electronicos con garantias. El dato no se convierte en reutilizable para cualquiera por el hecho de circular entre administraciones. En este caso se trata de interoperabilidad administrativa e intermediacion para una finalidad concreta, no de apertura general.

La unidad tramitadora debe identificar cada dato necesario, justificar su necesidad, consultar las plataformas o servicios habilitados cuando proceda, registrar la evidencia de la consulta y respetar los limites de proteccion de datos. Si el interesado puede oponerse a una consulta concreta, el procedimiento debe prever como se gestiona esa oposicion. Si la consulta se realiza en un contexto en el que la ley no permite oposicion, debe explicarse la base aplicable de forma clara.

Desde la perspectiva de gobierno del dato, el caso revela varias decisiones: que organo es responsable del procedimiento, que proveedor administrativo custodia cada dato, que definicion se usa para "unidad familiar", que version del dato se considera valida, cuanto tiempo se conserva la evidencia y como se resuelven discrepancias. Si el catastro, el padron y el registro de ayudas usan identificadores o fechas de referencia distintas, la interoperabilidad tecnica no basta; hace falta interoperabilidad semantica.

**Uso didactico.** Este ejemplo ayuda a explicar que la reutilizacion de informacion publica no agota la politica de datos. Hay datos que se intercambian para tramitar mejor y no para publicarse. Tambien permite introducir la calidad: si el certificado consultado esta desactualizado o el servicio devuelve una respuesta no trazable, el procedimiento gana velocidad pero pierde fiabilidad.

### Ejemplo 2. Portal de datos abiertos con contratos menores

Un ayuntamiento decide publicar un conjunto de datos sobre contratos menores. La primera version consiste en subir trimestralmente un PDF con una tabla exportada desde la aplicacion contable. El documento se puede leer, pero no permite filtrar, combinar ni automatizar el analisis. Los campos "proveedor", "objeto", "importe" y "fecha" no siguen criterios estables. Algunas filas tienen abreviaturas y otras incorporan datos personales en el campo de observaciones.

Una version madura del mismo conjunto exige un enfoque de reutilizacion. El ayuntamiento debe publicar datos en formato estructurado, con metadatos, periodicidad, licencia o condiciones de uso, fecha de actualizacion, responsable, diccionario de campos y una advertencia clara sobre limitaciones. Si el conjunto se ofrece mediante API, debe documentarse el acceso, los parametros, los codigos de respuesta y la version. Si hay campos que no deben publicarse por proteccion de datos o confidencialidad, se excluyen o se transforman antes de la publicacion.

La calidad no consiste solo en que el importe sea correcto. Tambien importa que el proveedor este normalizado, que la fecha tenga formato uniforme, que el identificador del expediente permita trazabilidad, que no existan duplicados, que las anulaciones se reflejen y que el historico no cambie sin rastro. El gobierno del dato define quien valida cada carga y que ocurre cuando se detecta un error.

**Uso didactico.** Este ejemplo sirve para distinguir transparencia, acceso a informacion publica y reutilizacion. La transparencia puede exigir publicar determinada informacion. El acceso permite pedir informacion. La reutilizacion exige que esa informacion pueda usarse de nuevo en condiciones tecnicas, juridicas y economicas adecuadas.

### Ejemplo 3. Vocabulario comun para servicios sociales

Tres administraciones colaboran en un programa de atencion a la dependencia. Una usa el campo "grado", otra "nivel de dependencia" y otra "situacion valorada". En una base, "G1" significa grado I; en otra, "1" significa prioridad alta; en otra, "leve" se usa como categoria interna. Cuando se integran los datos, los informes agregados ofrecen resultados inconsistentes.

El problema no se resuelve comprando una nueva herramienta. La herramienta puede transportar datos, pero no crea significado compartido. Hace falta una definicion comun de entidades, atributos, codigos, reglas de equivalencia y metadatos. Tambien se necesita decidir quien aprueba cambios en el vocabulario, como se versiona y como se comunica a las aplicaciones consumidoras.

La interoperabilidad semantica permite que el receptor entienda el dato como lo entiende el emisor. En un expediente administrativo, esa comprension afecta a derechos de las personas. Una categoria mal interpretada puede alterar una prioridad, generar una denegacion indebida o producir estadisticas publicas equivocadas.

**Uso didactico.** Este caso introduce la idea de que un catalogo de datos no es solo inventario tecnico. Debe recoger significado, origen, finalidad, restricciones, calidad y relaciones con otros conceptos.

### Ejemplo 4. Datos de movilidad como conjunto de alto valor

Una administracion metropolitana dispone de datos de paradas, lineas, horarios, incidencias y ocupacion del transporte publico. Varias empresas de movilidad y universidades piden acceso para crear aplicaciones, estudiar demanda y mejorar la planificacion urbana.

El caso conecta con los conjuntos de datos de alto valor previstos en el marco europeo de datos abiertos. La movilidad es una de las categorias relevantes. La administracion debe pensar en publicacion en formatos legibles por maquina, API cuando proceda, condiciones de reutilizacion claras, gratuidad en los terminos aplicables y actualizacion suficiente. Tambien debe diferenciar datos estaticos, como paradas o trazados, de datos dinamicos, como incidencias o tiempos estimados.

El riesgo aparece cuando se publican datos aparentemente impersonales que, combinados con otros, pueden revelar patrones individuales. Por eso el diseño debe valorar granularidad, agregacion, retraso temporal, anonimacion y finalidad. Abrir datos no significa publicar todo con maximo detalle.

**Uso didactico.** Este ejemplo permite explicar la relacion entre valor economico, interes publico, interoperabilidad tecnica, calidad temporal y proteccion de derechos.

### Ejemplo 5. Cuadro de mando de absentismo con cifras incompatibles

Un departamento de recursos humanos informa de un 6 % de absentismo. La intervencion general calcula un 4,8 %. La unidad de salud laboral obtiene un 7,1 %. Todas las cifras salen de sistemas oficiales, pero usan criterios distintos: dias naturales frente a dias laborables, inclusion o no de permisos, fecha de inicio frente a fecha de cierre y unidades administrativas no coincidentes.

La solucion de gobierno del dato empieza por definir el indicador. Debe fijarse el concepto, formula, poblacion incluida, exclusiones, periodo, fuente autorizada, responsable funcional, responsable tecnico y nivel de calidad exigido. Despues se documenta el linaje: de que sistemas procede la informacion, que transformaciones se aplican y que controles validan el resultado.

Este ejemplo es util para mostrar que la calidad no es una propiedad abstracta del dato aislado. Un dato puede ser exacto en origen y, aun asi, producir una decision equivocada si se interpreta fuera de contexto.

### Ejemplo 6. Reutilizacion de datos ambientales con licencia y metadatos

Una agencia ambiental publica mediciones de calidad del aire. Los datos se actualizan cada hora y se ofrecen para consulta ciudadana. Una empresa quiere reutilizarlos en una aplicacion que recomienda rutas saludables. La agencia puede fomentar ese uso, pero debe asegurar que el conjunto tenga metadatos, identificadores de estaciones, unidades de medida, frecuencia, advertencias sobre calibracion, historico y condiciones de reutilizacion.

Si una estacion esta en mantenimiento, el dato debe indicar su estado. Si se cambia el sensor o el metodo de medicion, el historico debe documentarlo. Si la agencia publica solo una imagen del mapa, la informacion puede ser visible, pero no plenamente reutilizable. La diferencia entre visualizar y reutilizar es una trampa clasica de examen.

## 3. Supuestos practicos guiados

### Supuesto 1. No pedir certificados que ya obran en poder de la Administracion

**Situacion.** Una diputacion provincial gestiona becas de transporte para estudiantes de municipios pequenos. El formulario exige a cada solicitante adjuntar certificado de empadronamiento, copia del DNI, justificante de matricula y certificado de discapacidad cuando proceda. La diputacion recibe quejas porque los ayuntamientos tardan en emitir certificados y porque algunos datos ya pueden consultarse.

**Pistas relevantes.**

| Pista del enunciado | Concepto que activa |
| --- | --- |
| "Certificado de empadronamiento" | Consulta a otra administracion, intermediacion, no aportacion de documentos ya disponibles |
| "Copia del DNI" | Verificacion de identidad o datos de identidad mediante servicios habilitados |
| "Certificado de discapacidad" | Dato especialmente sensible en sentido amplio para el procedimiento; necesidad, finalidad y garantias reforzadas |
| "Gestion de becas" | Procedimiento administrativo con base normativa y finalidad concreta |
| "Quejas por cargas administrativas" | Simplificacion, eficacia e interoperabilidad |

**Preguntas.**

1. ¿Debe resolverse el caso publicando un conjunto de datos abiertos?
2. ¿Que diferencia hay entre consultar un dato para tramitar una beca y reutilizar informacion publica?
3. ¿Que medidas de gobierno del dato deben existir antes de automatizar consultas?
4. ¿Que controles de calidad son necesarios?

**Resolucion paso a paso.**

Primero se identifica la finalidad: tramitar becas de transporte. Los datos se consultan para resolver un procedimiento concreto, no para ponerlos a disposicion de terceros. Por tanto, el instrumento principal es la interoperabilidad administrativa mediante consultas o transmisiones de datos entre administraciones, no la reutilizacion abierta.

Segundo se revisa la necesidad de cada dato. El empadronamiento puede ser necesario si la ayuda depende del municipio. La identidad puede verificarse sin exigir copia documental cuando existan servicios habilitados. La discapacidad solo debe consultarse si es relevante para un requisito, baremo o prioridad de la convocatoria. La matricula puede requerir interoperabilidad con administraciones educativas o, si no existe servicio, aportacion por el interesado.

Tercero se define el circuito de autorizacion y trazabilidad. La unidad gestora debe saber quien puede consultar, en que expediente, por que motivo, con que base normativa, que respuesta se obtiene y como se conserva la evidencia. La trazabilidad protege tanto al ciudadano como a la administracion.

Cuarto se fijan controles de calidad. No basta con recibir una respuesta "si/no". Hay que comprobar fecha de referencia, vigencia, organo emisor, correspondencia con el solicitante y tratamiento de errores. Un dato correcto pero no vigente puede causar una adjudicacion indebida.

Quinto se comunica al ciudadano de forma comprensible que documentos no debe aportar, que consultas se realizaran y como se ejercen los derechos que procedan. La simplificacion administrativa no debe convertirse en opacidad.

**Errores que debe evitar la respuesta.**

No debe afirmarse que todos los datos se pueden consultar siempre. Tampoco debe confundirse oposicion a la consulta con consentimiento universal. No debe proponerse publicar listados nominales de beneficiarios como solucion al problema de tramitacion. La publicidad activa de ayudas, cuando proceda, es otra cuestion y exige su propio analisis de datos personales, finalidad y normativa.

**Mini comprobacion.**

Una respuesta de calidad menciona finalidad, necesidad, plataforma o servicio de consulta, garantias de proteccion de datos, trazabilidad, calidad y diferencia entre intercambio administrativo y reutilizacion.

### Supuesto 2. Catalogo autonomico de datos abiertos

**Situacion.** Una comunidad autonoma quiere renovar su portal de datos abiertos. Actualmente cada consejeria publica ficheros en formatos diferentes. Algunos datasets no tienen fecha de actualizacion. Otros contienen columnas sin descripcion. La direccion general competente quiere alinearse con el marco estatal y europeo y preparar el portal para federacion con catalogos superiores.

**Pistas relevantes.**

| Pista del enunciado | Concepto que activa |
| --- | --- |
| "Formatos diferentes" | Normalizacion, formatos legibles por maquina, estandares abiertos |
| "Columnas sin descripcion" | Metadatos, diccionario de datos, interoperabilidad semantica |
| "Sin fecha de actualizacion" | Calidad temporal, vigencia, confianza |
| "Federacion con catalogos superiores" | DCAT-AP, DCAT-AP-ES, catalogacion interoperable |
| "Marco estatal y europeo" | ENI, NTI de reutilizacion, Directiva de datos abiertos, conjuntos de alto valor |

**Preguntas.**

1. ¿Que elementos minimos debe tener cada conjunto de datos?
2. ¿Como se diferencia el catalogo de datos de un simple repositorio de ficheros?
3. ¿Que papel cumple la interoperabilidad semantica?
4. ¿Que controles deben establecerse antes de publicar?

**Resolucion paso a paso.**

Primero se aprueba un modelo comun de catalogacion. Cada conjunto debe tener titulo, descripcion, responsable, publicador, fecha de creacion, fecha de actualizacion, frecuencia esperada, cobertura temporal y geografica, tema, palabras clave, licencia o condiciones de uso, formato, punto de acceso, nivel de calidad y contacto funcional.

Segundo se define un diccionario de datos por conjunto. Si un dataset contiene el campo "importe", debe aclarar si incluye impuestos, moneda, redondeo y momento contable. Si contiene "municipio", debe indicar codigo oficial, nombre normalizado y fecha de referencia. Esta es la parte semantica: el usuario externo debe entender el significado de lo publicado sin llamar al gestor.

Tercero se eligen formatos y mecanismos de acceso adecuados. Un CSV bien documentado puede ser mas reutilizable que un PDF visualmente limpio. Para datos dinamicos o de alto valor puede ser necesario un acceso mediante API. La interfaz debe tener documentacion y estabilidad suficiente.

Cuarto se incorporan controles previos a la publicacion. Deben detectarse datos personales, secretos, informacion protegida por propiedad intelectual de terceros, campos que permitan reidentificacion, duplicados, valores fuera de rango, codigos obsoletos y problemas de accesibilidad.

Quinto se establece gobierno permanente. Cada dataset necesita propietario funcional, responsable de actualizacion, circuito de incidencias, versionado y retirada. Un portal sin mantenimiento degrada rapidamente la confianza.

**Errores que debe evitar la respuesta.**

No basta con decir "usar datos abiertos". No debe confundirse DCAT-AP con una base de datos operativa. No debe proponerse publicar todo en tiempo real. No debe olvidarse que la federacion exige metadatos consistentes, no solo enlaces.

**Mini comprobacion.**

La respuesta debe mencionar catalogo, metadatos, formatos, licencias, calidad, proteccion de datos, actualizacion, DCAT-AP o perfil equivalente y responsabilidades.

### Supuesto 3. Interoperabilidad semantica en historia social unica

**Situacion.** Una comunidad autonoma quiere crear una historia social unica que agregue informacion procedente de servicios sociales municipales, dependencia, discapacidad, empleo y vivienda. Cada sistema usa categorias propias. En unos casos se habla de "persona usuaria"; en otros, de "beneficiario", "solicitante", "miembro de unidad convivencial" o "representante".

**Pistas relevantes.**

| Pista del enunciado | Concepto que activa |
| --- | --- |
| "Historia social unica" | Integracion de datos con impacto en derechos |
| "Categorias propias" | Vocabulario comun, ontologia o modelo conceptual |
| "Municipios y comunidad autonoma" | Interoperabilidad organizativa y juridica |
| "Dependencia, vivienda, empleo" | Finalidades distintas y minimizacion |
| "Unidad convivencial" | Concepto juridico-sectorial que debe definirse |

**Preguntas.**

1. ¿Por que no basta con conectar las bases de datos?
2. ¿Que elementos debe contener un modelo semantico comun?
3. ¿Que riesgos de calidad y proteccion de datos aparecen?
4. ¿Como se gobiernan los cambios de definicion?

**Resolucion paso a paso.**

Primero se delimita la finalidad y el alcance. No todos los datos de servicios sociales, empleo y vivienda pueden agregarse por conveniencia. Cada flujo necesita base juridica, finalidad, minimizacion y control de acceso.

Segundo se construye un modelo conceptual. Deben definirse entidades como persona, expediente, unidad convivencial, prestacion, valoracion, representante, domicilio, recurso y resolucion. Cada entidad requiere atributos, codigos, relaciones y reglas de negocio. Tambien se identifican sinonimos y equivalencias entre sistemas.

Tercero se acuerdan vocabularios y codigos. Si "unidad convivencial" no significa lo mismo en vivienda que en dependencia, no debe forzarse una equivalencia automatica. Se puede crear un concepto comun de nivel superior y conservar atributos especificos por sector.

Cuarto se establecen controles de calidad. La integracion debe detectar duplicados de personas, identificadores incompletos, direcciones inconsistentes, fechas imposibles, prestaciones cerradas que siguen activas y resoluciones pendientes mal clasificadas.

Quinto se crea un gobierno de cambios. Cualquier cambio en un codigo o definicion debe versionarse, comunicarse y probarse antes de entrar en produccion. La interoperabilidad semantica es una disciplina viva, no un documento inicial.

**Errores que debe evitar la respuesta.**

No debe equipararse historia social unica con expediente unico universal. No debe defenderse la acumulacion indiscriminada de datos "por si acaso". No debe tratarse la semantica como asunto menor de programadores. En este supuesto, una mala definicion puede afectar a baremos, prioridades y derechos.

**Mini comprobacion.**

La respuesta debe incluir finalidad, minimizacion, modelo conceptual, vocabularios, equivalencias, calidad, trazabilidad y gobierno de cambios.

### Supuesto 4. Incidente de calidad en datos de subvenciones

**Situacion.** Un organo gestor publica una estadistica mensual de subvenciones concedidas. Un medio de comunicacion detecta que una misma entidad aparece con tres nombres distintos y que las cifras no coinciden con las publicadas en el portal de transparencia. La unidad tecnica explica que los datos proceden de cargas manuales de distintas convocatorias.

**Pistas relevantes.**

| Pista del enunciado | Concepto que activa |
| --- | --- |
| "Tres nombres distintos" | Dato maestro, normalizacion, identificador unico |
| "Cifras no coinciden" | Linaje, version, criterio de computo |
| "Portal de transparencia" | Coherencia entre obligaciones de publicidad y reutilizacion |
| "Cargas manuales" | Riesgo operativo, controles de validacion |
| "Estadistica mensual" | Calidad temporal y reproducibilidad |

**Preguntas.**

1. ¿Que dimensiones de calidad estan afectadas?
2. ¿Que medidas inmediatas deben adoptarse?
3. ¿Que medidas estructurales corresponden al gobierno del dato?
4. ¿Como se comunica la correccion?

**Resolucion paso a paso.**

Primero se clasifica el incidente. Hay problemas de consistencia, unicidad, exactitud, trazabilidad y actualidad. Tambien puede haber un problema de definicion si una fuente computa concesiones y otra pagos.

Segundo se identifica la fuente autorizada. Debe decidirse que sistema tiene valor de referencia para cada dato: expediente, contabilidad, base nacional, portal de transparencia o registro sectorial. Si cada publicacion usa una fuente distinta, las diferencias se repetiran.

Tercero se corrige el dataset con versionado. La correccion debe mantener rastro: fecha de correccion, motivo, campos afectados y, si procede, nota metodologica. Borrar y sustituir sin explicacion deteriora la confianza.

Cuarto se implantan controles preventivos. Son utiles los identificadores unicos, listas controladas de beneficiarios, validacion de importes, reglas sobre estados del expediente, controles de duplicidad y conciliacion con la fuente contable.

Quinto se asignan responsabilidades. El responsable funcional define el criterio; el responsable tecnico implementa controles; la unidad de datos coordina catalogacion y calidad; el organo gestor responde del contenido.

**Errores que debe evitar la respuesta.**

No debe reducirse el problema a "un error informatico". No debe proponerse ocultar el dataset hasta que sea perfecto. La calidad se gestiona con niveles, controles y transparencia sobre limitaciones.

**Mini comprobacion.**

La respuesta debe hablar de dimensiones de calidad, dato maestro, linaje, fuente autorizada, controles y comunicacion de correcciones.

### Supuesto 5. Datos de alto valor y API de movilidad

**Situacion.** Una autoridad de transporte publica horarios de autobus en PDF. Varias aplicaciones privadas solicitan acceso automatizado para planificar rutas. La autoridad teme perder control sobre la informacion y argumenta que ya cumple porque los horarios estan en su web.

**Pistas relevantes.**

| Pista del enunciado | Concepto que activa |
| --- | --- |
| "Horarios en PDF" | Informacion visible pero poco reutilizable |
| "Acceso automatizado" | API, datos legibles por maquina |
| "Movilidad" | Categoria europea de alto valor |
| "Perder control" | Condiciones de reutilizacion, calidad, responsabilidad |
| "Aplicaciones privadas" | Reutilizacion con fines comerciales o no comerciales |

**Preguntas.**

1. ¿Publicar PDF equivale a facilitar reutilizacion?
2. ¿Que condiciones puede fijar la autoridad?
3. ¿Que requisitos de calidad son especialmente importantes?
4. ¿Que datos no deberian abrirse sin analisis adicional?

**Resolucion paso a paso.**

Primero se distingue consulta humana de reutilizacion automatizada. Un PDF en una web puede informar al viajero, pero no permite facilmente combinar, actualizar o integrar los horarios en servicios externos.

Segundo se valora el marco de datos abiertos y conjuntos de alto valor. La movilidad es una categoria en la que el acceso mediante formatos legibles por maquina y API resulta especialmente relevante. Si los datos son dinamicos, la frecuencia de actualizacion y la disponibilidad del servicio son parte de la calidad.

Tercero se fijan condiciones claras. La autoridad puede exigir atribucion, prohibir alteraciones que induzcan a error, informar de ausencia de garantia absoluta, establecer limites razonables de uso tecnico de la API y documentar versionado. Las condiciones no deben convertirse en barreras injustificadas a la reutilizacion.

Cuarto se separan datos publicables y datos sensibles. Horarios, paradas e incidencias agregadas pueden ser reutilizables. Datos de vigilancia, patrones individualizados de viajeros, informacion de seguridad o datos personales exigen exclusion, agregacion o analisis especifico.

Quinto se crea un canal de incidencias. Los reutilizadores deben poder reportar errores. La administracion mantiene la fuente y mejora calidad; el ecosistema reutilizador aporta deteccion temprana.

**Errores que debe evitar la respuesta.**

No debe afirmarse que una empresa privada no puede reutilizar informacion publica. Tampoco debe entenderse reutilizacion como cesion sin condiciones. El equilibrio esta en apertura por defecto cuando proceda, limites legales y garantias tecnicas.

**Mini comprobacion.**

La respuesta debe mencionar formato legible por maquina, API, movilidad como dato de alto valor, condiciones de reutilizacion, calidad temporal y limites por seguridad o privacidad.

## 4. Tabla de errores frecuentes y correccion pedagogica

| Error frecuente | Por que esta mal | Respuesta correcta |
| --- | --- | --- |
| "Dato abierto" es cualquier dato publicado en una web | Publicar no implica reutilizacion si no hay formato, metadatos, licencia y calidad | Dato abierto exige condiciones juridicas y tecnicas que permitan uso, redistribucion y combinacion |
| Interoperabilidad semantica es que dos sistemas se conecten por API | La API resuelve comunicacion tecnica; no asegura significado compartido | La semantica requiere vocabularios, definiciones, codigos y metadatos comunes |
| Calidad del dato significa que el dato sea verdadero | La exactitud es solo una dimension | Tambien importan completitud, actualidad, consistencia, unicidad, trazabilidad y adecuacion al uso |
| Reutilizacion y transparencia son lo mismo | La transparencia se centra en publicidad y acceso; la reutilizacion en nuevo uso de informacion | Pueden coincidir, pero tienen regimenes y objetivos distintos |
| Si se anonimiza quitando nombre y DNI, el dato ya no es personal | La reidentificacion puede producirse por combinacion de atributos | La anonimacion requiere analisis de riesgo, contexto y tecnicas adecuadas |
| El gobierno del dato es una funcion puramente informatica | Las decisiones sobre significado, calidad y uso son funcionales y juridicas | Es una responsabilidad transversal con roles, normas y controles |
| Un catalogo es una carpeta con ficheros | Sin metadatos y gestion de ciclo de vida no hay catalogo util | El catalogo describe, organiza, permite descubrir, evaluar y reutilizar datasets |
| Un dato de alto valor debe publicarse siempre con maximo detalle | El valor no elimina limites de privacidad, seguridad o confidencialidad | Deben aplicarse formatos, API y gratuidad cuando proceda, con garantias |
| La interoperabilidad elimina la necesidad de proteccion de datos | La circulacion de datos aumenta la necesidad de control | Debe haber finalidad, minimizacion, base juridica, seguridad y trazabilidad |
| Los metadatos son documentacion opcional | Sin metadatos, el reutilizador no conoce significado, vigencia ni condiciones | Los metadatos son infraestructura de confianza |
| La calidad se revisa al final del proyecto | Corregir al final es caro y poco fiable | La calidad se diseña desde captura, validacion, integracion, publicacion y archivo |
| Si el dato procede de una administracion, siempre es fiable | Tambien hay errores, retrasos y diferencias de criterio | Debe evaluarse fuente, vigencia, linaje y controles |
| La licencia puede prohibir usos comerciales por prudencia | La reutilizacion admite usos comerciales salvo limites justificados | Las condiciones deben ser claras, proporcionadas y no discriminatorias |
| El PDF firmado es el mejor formato para datos abiertos | Puede ser adecuado para documento, pero no para explotacion de datos | Para reutilizacion se prefieren formatos estructurados y legibles por maquina |
| La federacion de catalogos se logra copiando datasets | Federar exige metadatos compatibles y puntos de acceso estables | DCAT-AP y perfiles nacionales facilitan agregacion y descubrimiento |

## 5. Notas de test separadas

> **Nota de test.** Si el enunciado habla de "no pedir al ciudadano documentos ya aportados" o "consultar datos que obran en otra administracion", la respuesta apunta a interoperabilidad administrativa y transmision de datos, no a reutilizacion abierta.

> **Nota de test.** Si aparecen palabras como "vocabulario", "significado", "definicion comun", "codigos", "modelo conceptual" o "metadatos", la pista es interoperabilidad semantica. Si aparecen "API", "protocolo", "formato" o "servicio web", la pista puede ser interoperabilidad tecnica.

> **Nota de test.** Una pregunta sobre datos abiertos puede introducir datos personales para comprobar si el opositor aplica limites. La apertura no prevalece automaticamente sobre proteccion de datos, seguridad, secreto estadistico, propiedad intelectual de terceros o confidencialidad.

> **Nota de test.** La palabra "calidad" no debe activar solo "exactitud". En preguntas de caso, conviene recorrer al menos seis dimensiones: exactitud, completitud, actualidad, consistencia, unicidad y trazabilidad.

> **Nota de test.** "Conjunto de datos de alto valor" no significa "dato importante para mi organizacion". Es una categoria juridica europea vinculada a areas como geoespacial, observacion de la Tierra y medio ambiente, meteorologia, estadistica, sociedades y propiedad de sociedades, y movilidad.

> **Nota de test.** Si una opcion de respuesta dice que la reutilizacion solo puede ser sin fines comerciales, normalmente es falsa. El regimen de reutilizacion admite usos comerciales y no comerciales, salvo limites aplicables.

> **Nota de test.** DCAT-AP no es una aplicacion informatica concreta. Es un perfil de metadatos para describir catalogos y conjuntos de datos de forma interoperable.

> **Nota de test.** El gobierno del dato aparece cuando el caso pregunta "quien decide", "quien valida", "quien corrige", "que fuente prevalece", "como se versiona" o "como se mide la calidad".

> **Nota de test.** Si un dato se publica como imagen, PDF o tabla incrustada sin estructura, puede cumplir una finalidad informativa, pero sera debil para reutilizacion. La pregunta suele buscar la expresion "formato legible por maquina".

> **Nota de test.** El Reglamento sobre la Europa Interoperable refuerza la interoperabilidad transfronteriza del sector publico en la Union. No debe confundirse con una norma nacional sobre procedimiento administrativo ni con una simple guia tecnica voluntaria.

## 6. Muestra progresiva de preguntas tipo test

Estas preguntas son muestra de integracion y no deben sustituir al banco completo externo.

### Nivel base

**1. La interoperabilidad semantica se refiere principalmente a:**

A. La velocidad de transmision entre dos sistemas.
B. El significado compartido de la informacion intercambiada.
C. La contratacion de una unica plataforma tecnologica.
D. El cifrado de todas las comunicaciones.

**Respuesta correcta:** B.

**Diagnostico.** Si se elige A o D, se esta confundiendo semantica con aspectos tecnicos o de seguridad. Si se elige C, se confunde interoperabilidad con uniformidad tecnologica.

**2. En reutilizacion de informacion publica, un formato legible por maquina permite:**

A. Que el documento tenga firma manuscrita.
B. Que el contenido pueda ser procesado automaticamente por aplicaciones.
C. Que solo se pueda leer desde la sede electronica.
D. Que se impida todo uso comercial.

**Respuesta correcta:** B.

**Diagnostico.** La reutilizacion se apoya en tratamiento automatizado. Un PDF visual puede ser util para lectura, pero no siempre es adecuado para explotacion de datos.

**3. ¿Cual de las siguientes opciones es una dimension de calidad del dato?**

A. Unicidad.
B. Color corporativo.
C. Numero de pantallas de una aplicacion.
D. Lenguaje de programacion usado.

**Respuesta correcta:** A.

**Diagnostico.** La calidad se evalua sobre propiedades del dato y su adecuacion al uso, no sobre estetica o tecnologia de la aplicacion.

### Nivel aplicacion

**4. Una administracion pide al ciudadano que aporte un certificado de empadronamiento que puede consultar electronicamente. El enfoque mas adecuado es:**

A. Publicar el padron municipal como dato abierto.
B. Recabar o consultar el dato por los sistemas habilitados, con garantias y trazabilidad.
C. Exigir siempre el certificado para evitar errores.
D. Sustituir el procedimiento por una encuesta anonima.

**Respuesta correcta:** B.

**Diagnostico.** El caso trata de tramitacion administrativa, no de apertura de datos. Publicar el padron seria desproporcionado y contrario a garantias basicas.

**5. Un portal de datos abiertos permite descargar contratos menores solo en PDF escaneado. La principal debilidad para reutilizacion es:**

A. Que los datos son demasiado recientes.
B. Que el formato no facilita procesamiento automatico ni reutilizacion estructurada.
C. Que toda informacion contractual es secreta.
D. Que solo las administraciones pueden reutilizar datos.

**Respuesta correcta:** B.

**Diagnostico.** La trampa esta en confundir visibilidad con reutilizacion. Puede haber limites por datos personales, pero no toda informacion contractual es secreta.

**6. Tres departamentos usan codigos distintos para el mismo tipo de expediente. La medida mas directamente relacionada con interoperabilidad semantica es:**

A. Aumentar el ancho de banda.
B. Crear equivalencias, definiciones comunes y reglas de versionado de codigos.
C. Comprar monitores mas grandes.
D. Publicar capturas de pantalla del sistema.

**Respuesta correcta:** B.

**Diagnostico.** La pregunta no va de infraestructura, sino de significado. Los codigos solo interoperan si se documenta su equivalencia y contexto.

### Nivel examen real

**7. Una autoridad de transporte publica horarios en PDF y rechaza una API porque "la informacion ya esta disponible al publico". En el marco de reutilizacion, la critica mas solida es:**

A. La disponibilidad visual no garantiza reutilizacion efectiva en formato procesable.
B. Las empresas privadas nunca pueden reutilizar informacion publica.
C. Los datos de movilidad no tienen interes publico.
D. La API elimina todos los limites de proteccion de datos.

**Respuesta correcta:** A.

**Diagnostico.** La respuesta correcta combina reutilizacion y formato. La B es falsa porque la reutilizacion puede ser comercial. La D es falsa porque una API no elimina garantias.

**8. Un cuadro de mando de subvenciones ofrece importes distintos al portal de transparencia. La primera actuacion de gobierno del dato deberia ser:**

A. Cambiar el color del cuadro de mando para advertir al usuario.
B. Identificar fuente autorizada, definicion del indicador, linaje y reglas de conciliacion.
C. Eliminar todos los datos historicos.
D. Impedir toda descarga hasta que no exista error cero.

**Respuesta correcta:** B.

**Diagnostico.** La calidad se gestiona con definiciones, fuentes y controles. La transparencia de limitaciones es mejor que ocultacion indiscriminada.

**9. Un conjunto de datos contiene edad, codigo postal, fecha de atencion y diagnostico de salud, pero no nombre ni DNI. La afirmacion mas prudente es:**

A. Ya no hay ningun riesgo porque se han eliminado identificadores directos.
B. Puede persistir riesgo de reidentificacion por combinacion de atributos.
C. Debe publicarse siempre por ser informacion publica.
D. Solo hay riesgo si el fichero esta en PDF.

**Respuesta correcta:** B.

**Diagnostico.** La anonimacion no es una operacion cosmetica. Hay que valorar contexto, granularidad, rareza de combinaciones y posibles fuentes auxiliares.

**10. La federacion de un catalogo autonomico con catalogos europeos exige sobre todo:**

A. Usar nombres de ficheros largos.
B. Metadatos compatibles y puntos de acceso estables.
C. Eliminar la licencia de todos los datasets.
D. Publicar solo documentos ofimaticos.

**Respuesta correcta:** B.

**Diagnostico.** La federacion depende de metadatos normalizados. DCAT-AP y perfiles nacionales cumplen esa funcion.

## 7. Banco de frases utiles para modo tutor

Estas frases pueden integrarse como explicaciones adicionales separadas del texto base.

- **Que significa gobierno del dato.** Significa decidir y controlar como se define, produce, valida, comparte, protege, conserva y reutiliza el dato. No es un sinonimo de base de datos.
- **Por que importa.** Sin gobierno, dos unidades pueden usar el mismo dato con significados distintos y producir decisiones incompatibles.
- **Con que se confunde.** Se confunde con informatica, con analitica o con publicar datos. En realidad, incluye responsabilidades juridicas, funcionales, tecnicas y organizativas.
- **Como se reconoce en examen.** Aparecen preguntas sobre responsables, fuente oficial, calidad, linaje, reglas de acceso, versionado o ciclo de vida.
- **Que significa interoperabilidad semantica.** Significa que el dato conserva significado cuando pasa de un sistema u organizacion a otra.
- **Por que importa.** Una administracion puede recibir correctamente un campo y aun asi interpretarlo mal si no comparte definicion, codigo o contexto.
- **Con que se confunde.** Se confunde con conectividad tecnica. Conectar sistemas no implica que se entiendan.
- **Como se reconoce en examen.** El enunciado habla de vocabularios, codigos, definiciones, metadatos, unidades de medida o equivalencias.
- **Que significa calidad del dato.** Es el grado en que el dato sirve para el uso previsto con fiabilidad suficiente.
- **Por que importa.** En administracion publica, un dato deficiente puede causar retrasos, desigualdad, errores de pago, denegaciones indebidas o estadisticas engañosas.
- **Con que se confunde.** Se reduce indebidamente a exactitud. Tambien incluye completitud, actualidad, consistencia, unicidad y trazabilidad.
- **Como se reconoce en examen.** El caso menciona duplicados, campos vacios, fechas antiguas, discrepancias entre fuentes o imposibilidad de saber de donde sale una cifra.
- **Que significa reutilizacion.** Es el uso de documentos o datos del sector publico para fines distintos de la mision inicial para la que fueron producidos, dentro del marco juridico aplicable.
- **Por que importa.** Favorece transparencia, innovacion, rendicion de cuentas, investigacion y servicios de valor añadido.
- **Con que se confunde.** Se confunde con acceso a un expediente, publicidad activa o intercambio interno entre administraciones.
- **Como se reconoce en examen.** Aparecen terceros que quieren usar datos, licencias, formatos, API, datos abiertos, datos de alto valor o condiciones de reutilizacion.

## 8. Cuadros comparativos para integrar

### Transparencia, acceso, interoperabilidad y reutilizacion

| Dimension | Pregunta que responde | Destinatario tipico | Instrumentos habituales | Riesgo si se confunde |
| --- | --- | --- | --- | --- |
| Transparencia | ¿Que informacion debe publicarse para rendir cuentas? | Ciudadania en general | Publicidad activa, portales de transparencia | Publicar sin estructura reutilizable |
| Acceso a informacion publica | ¿Puede una persona solicitar informacion concreta? | Solicitante | Procedimiento de acceso, limites y ponderacion | Tratar toda solicitud como descarga abierta |
| Interoperabilidad administrativa | ¿Puede una administracion obtener datos necesarios para tramitar? | Organos administrativos | Plataformas de intermediacion, servicios, convenios o sistemas habilitados | Abrir datos personales cuando solo procedia consulta finalista |
| Reutilizacion | ¿Puede un tercero usar informacion publica para nuevos fines? | Ciudadania, empresas, investigacion, sector publico | Datos abiertos, licencias, formatos, APIs, metadatos | Creer que una web informativa ya es reutilizacion efectiva |

### Dimensiones de calidad con ejemplo administrativo

| Dimension | Pregunta practica | Ejemplo de control |
| --- | --- | --- |
| Exactitud | ¿El valor refleja correctamente la realidad o el acto administrativo? | Contraste con fuente autorizada |
| Completitud | ¿Faltan campos necesarios? | Reglas de obligatoriedad y porcentajes de nulos |
| Actualidad | ¿El dato esta vigente para decidir? | Fecha de ultima actualizacion y caducidad |
| Consistencia | ¿El dato coincide entre sistemas o dentro del mismo conjunto? | Reglas de conciliacion y validacion cruzada |
| Unicidad | ¿Existe mas de un registro para la misma entidad? | Identificador unico y deduplicacion |
| Trazabilidad | ¿Se sabe de donde viene y que transformaciones ha sufrido? | Linaje, logs, versionado y auditoria |
| Validez | ¿Cumple formato, rango y dominio permitido? | Catalogos de codigos y validaciones automaticas |
| Adecuacion al uso | ¿Sirve para la decision concreta? | Evaluacion por finalidad y nivel de calidad requerido |

### Tipos de metadatos que conviene mencionar

| Tipo de metadato | Funcion | Ejemplo |
| --- | --- | --- |
| Descriptivo | Facilita descubrimiento | Titulo, descripcion, palabras clave |
| Estructural | Explica organizacion del dato | Campos, tipos, relaciones, claves |
| Administrativo | Gestiona ciclo de vida | Responsable, licencia, fecha, version |
| Semantico | Aclara significado | Definicion, unidad, codigo, vocabulario |
| Calidad | Informa confianza | Completitud, errores conocidos, frecuencia |
| Acceso | Indica como obtenerlo | Formato, API, endpoint, restricciones |

## 9. Errores frecuentes redactados como avisos al opositor

1. No empieces una respuesta por la herramienta. Empieza por la finalidad administrativa y el regimen aplicable.
2. No escribas "se publicara en datos abiertos" si el caso habla de consultar datos personales para resolver un expediente.
3. No afirmes que el consentimiento es siempre la base de toda circulacion de datos en el sector publico. En muchos procedimientos la base vendra de la obligacion legal o del ejercicio de poderes publicos, con los matices que procedan.
4. No uses "anonimo" como sinonimo de "sin nombre". La anonimacion exige que la persona no sea identificable razonablemente en el contexto.
5. No confundas interoperabilidad con homogeneidad absoluta. Las administraciones pueden tener sistemas distintos si comparten reglas, estandares y significados suficientes.
6. No presentes la calidad como aspiracion estetica. Debe medirse con reglas, indicadores, umbrales y responsabilidades.
7. No olvides el ciclo de vida. Captura, validacion, uso, intercambio, publicacion, conservacion y retirada forman parte de la gestion del dato.
8. No trates los metadatos como apendice documental. En datos abiertos, los metadatos son lo que permite descubrir, comprender y reutilizar.
9. No ignores las condiciones de reutilizacion. La licencia o condiciones son parte del producto de datos.
10. No cites normas europeas como si sustituyeran automaticamente todas las obligaciones nacionales. En el examen conviene articular Union Europea, legislacion espanola, ENI y normas tecnicas.
11. No confundas "alto valor" con "alta sensibilidad". Un dato puede tener alto valor economico o social y, a la vez, exigir controles de privacidad o seguridad.
12. No prometas tiempo real si el dato no puede mantenerse con calidad. La frecuencia de actualizacion debe ser realista y documentada.
13. No uses ejemplos absurdos de distractores en notas de test. Las trampas reales son sutiles: transparencia frente a reutilizacion, API frente a semantica, exactitud frente a calidad, anonimacion frente a supresion de identificadores.

## 10. Supuesto corto para repaso final

**Enunciado.** Una consejeria publica datos de listas de espera sanitarias en un portal. El fichero se actualiza mensualmente en hoja de calculo. No contiene nombres, pero si hospital, especialidad, tramo de edad, sexo, diagnostico agrupado y municipio. Una asociacion pide mas detalle territorial y una empresa solicita una API. Al mismo tiempo, la intervencion detecta que los totales no coinciden con la memoria anual.

**Respuesta esperada en esquema.**

Debe separarse la finalidad estadistica y de transparencia de la reutilizacion. La peticion de mas detalle territorial exige valorar riesgo de reidentificacion, especialmente por combinacion de municipio, edad, sexo y diagnostico. La API puede ser adecuada si el conjunto es reutilizable y existe capacidad de mantenimiento, pero debe documentarse formato, metadatos, frecuencia y condiciones. La discrepancia con la memoria anual activa gobierno y calidad: definicion de indicador, fecha de corte, fuente autorizada, linaje, reglas de agregacion y versionado. No se debe publicar mas granularidad hasta cerrar el analisis de privacidad y calidad. La solucion madura combina metadatos, control de calidad, nota metodologica, posible agregacion adicional, canal de incidencias y condiciones claras de reutilizacion.

**Puntos que dan nota.**

- Distinguir transparencia, reutilizacion y calidad estadistica.
- Mencionar riesgo de reidentificacion aunque no haya nombres.
- Hablar de API como medio, no como obligacion absoluta sin contexto.
- Exigir definicion y linaje para resolver discrepancias.
- Documentar metodologia y limitaciones.

**Errores penalizables.**

- Decir que no hay dato personal porque no hay nombre.
- Publicar el maximo detalle por defecto.
- Rechazar la API solo porque la pide una empresa.
- Corregir totales sin versionado ni explicacion.
- Ignorar el papel del responsable funcional del dato.

## 11. Fuentes oficiales y tecnicas para citar editorialmente

Las referencias siguientes se han usado como base de contraste. En el tema final conviene citarlas de forma editorial, sin mostrar direcciones web en el cuerpo principal:

- Ley 39/2015, de 1 de octubre, del Procedimiento Administrativo Comun de las Administraciones Publicas, especialmente el regimen de documentos aportados por los interesados.
- Ley 40/2015, de 1 de octubre, de Regimen Juridico del Sector Publico, especialmente transmisiones de datos entre Administraciones Publicas.
- Ley 37/2007, de 16 de noviembre, sobre reutilizacion de la informacion del sector publico.
- Ley 19/2013, de 9 de diciembre, de transparencia, acceso a la informacion publica y buen gobierno.
- Real Decreto 4/2010, de 8 de enero, por el que se regula el Esquema Nacional de Interoperabilidad.
- Real Decreto 203/2021, de 30 de marzo, por el que se aprueba el Reglamento de actuacion y funcionamiento del sector publico por medios electronicos.
- Resolucion de 19 de febrero de 2013 por la que se aprueba la Norma Tecnica de Interoperabilidad de Reutilizacion de recursos de la informacion.
- Real Decreto 1112/2018, de 7 de septiembre, sobre accesibilidad de los sitios web y aplicaciones para dispositivos moviles del sector publico.
- Directiva (UE) 2019/1024 relativa a los datos abiertos y la reutilizacion de la informacion del sector publico.
- Reglamento de Ejecucion (UE) 2023/138 sobre conjuntos de datos de alto valor y modalidades de publicacion y reutilizacion.
- Reglamento (UE) 2022/868 relativo a la gobernanza europea de datos.
- Reglamento (UE) 2023/2854 sobre normas armonizadas para un acceso justo a los datos y su utilizacion.
- Reglamento (UE) 2024/903 sobre la Europa Interoperable.
- Perfil DCAT-AP y adaptacion DCAT-AP-ES para catalogos de datos abiertos.
