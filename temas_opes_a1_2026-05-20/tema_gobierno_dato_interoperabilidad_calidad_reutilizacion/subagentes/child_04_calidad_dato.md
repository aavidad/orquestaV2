# Material parcial: calidad del dato en la Administracion publica

## 1. Alcance y encaje en el tema

Este bloque desarrolla la calidad del dato como condicion operativa para que el gobierno del dato, la interoperabilidad semantica y la reutilizacion de la informacion publica funcionen en la practica. No basta con disponer de datos, ni con abrirlos, ni con integrarlos tecnicamente. Para que un dato pueda sostener una decision administrativa, alimentar un expediente, compartirse entre organos, publicarse como informacion reutilizable o servir de evidencia en un procedimiento automatizado, debe cumplir unos requisitos minimos de calidad, estar gobernado y ser verificable.

La calidad del dato se puede definir como el grado en que un dato resulta adecuado para los usos legitimos previstos, conforme a reglas conocidas, dentro de un contexto administrativo determinado y con evidencia suficiente sobre su origen, significado, actualizacion y fiabilidad. Esta definicion es deliberadamente funcional: un dato no es de calidad en abstracto, sino para una finalidad. El mismo dato puede ser suficiente para estadistica agregada e insuficiente para resolver un expediente individual; puede servir para un cuadro de mando y no para una notificacion; puede ser valido tecnicamente y, sin embargo, carecer de valor juridico porque no procede de la fuente competente.

En la Administracion publica, la calidad del dato tiene una dimension adicional: conecta con legalidad, seguridad juridica, eficacia, transparencia, rendicion de cuentas, proteccion de datos, interoperabilidad y reutilizacion. Una direccion postal incompleta puede impedir una notificacion. Un identificador mal normalizado puede duplicar expedientes. Una fecha con formato ambiguo puede alterar un plazo. Un catalogo sin metadatos suficientes puede impedir la reutilizacion. Un registro sin trazabilidad puede debilitar la defensa de una actuacion administrativa.

Por eso la calidad del dato no debe verse como una tarea cosmetica de limpieza posterior. Es una funcion permanente de gobierno. Debe definirse en origen, medirse durante el ciclo de vida, observarse en los flujos de intercambio, corregirse con criterios de responsabilidad y documentarse de forma que los equipos juridicos, funcionales y tecnicos entiendan la misma realidad.

## 2. Ideas fuerza para integrar en el tema principal

- La calidad del dato es un requisito previo de la interoperabilidad real: si los datos no son correctos, completos, consistentes y entendibles, el intercambio tecnico solo propaga errores.
- La calidad se mide contra reglas, no contra impresiones. Las reglas deben ser explicitas, versionadas, verificables y vinculadas a procesos, datos maestros, catalogos o normas aplicables.
- La calidad en origen es mas eficaz que la depuracion al final del proceso. La correccion tardia suele ser mas costosa, menos trazable y menos segura.
- La observabilidad de datos permite detectar degradaciones, rupturas de contratos, anomalias y retrasos antes de que afecten a expedientes, servicios o publicacion.
- La remediacion exige responsables, prioridades y evidencia. No todo error tiene el mismo impacto: un error en un dato meramente descriptivo no equivale a un error en un dato que determina derechos, obligaciones o plazos.
- Los cuadros de mando de calidad deben combinar indicadores tecnicos y funcionales. No basta contar valores nulos si no se sabe que impacto tienen.
- En el expediente administrativo, la calidad del dato se relaciona con identificacion, integridad, autenticidad, trazabilidad, metadatos, interoperabilidad documental y conservacion.
- La reutilizacion de informacion publica requiere datos comprensibles, documentados, actualizados y con condiciones de uso claras. La apertura sin calidad reduce confianza y utilidad.

## 3. Definiciones operativas

### Calidad del dato

Es el conjunto de propiedades que permiten confiar en que un dato es apto para una finalidad concreta. Incluye dimensiones como exactitud, completitud, consistencia, actualidad, unicidad, validez, integridad, trazabilidad, disponibilidad, comprensibilidad y conformidad normativa o semantica.

### Regla de calidad

Es una condicion verificable que expresa como debe ser un dato para considerarse aceptable. Puede referirse a formato, rango, obligatoriedad, relacion con otros campos, correspondencia con un catalogo, fuente autorizada, temporalidad, cardinalidad, integridad referencial o coherencia semantica.

Ejemplo: "la fecha de resolucion no puede ser anterior a la fecha de solicitud"; "un expediente debe tener organo responsable"; "el codigo de municipio debe pertenecer al catalogo oficial vigente"; "si el procedimiento exige notificacion electronica, debe existir medio electronico habilitado o causa documentada de excepcion".

### Perfilado de datos

Es el analisis exploratorio y sistematico de un conjunto de datos para conocer su estructura real, distribuciones, valores frecuentes, valores atipicos, nulos, duplicados, patrones de formato, dependencias y posibles errores. Sirve para pasar de suposiciones a evidencia.

### Validacion

Es la comprobacion de un dato contra reglas definidas. Puede hacerse en la captura, en la entrada a un sistema, antes de un intercambio, durante una transformacion, antes de publicar un conjunto de datos o en auditorias periodicas.

### Observabilidad de datos

Es la capacidad de conocer el estado de los datos y de sus flujos mediante metricas, alertas, trazas, controles de esquema, controles de frescura, controles de volumen, linaje y seguimiento de incidencias. Permite detectar fallos de calidad de forma temprana.

### Calidad en origen

Es el principio segun el cual los datos deben nacer correctos, completos y gobernados en el proceso donde se generan o capturan, en lugar de confiar en limpiezas posteriores. Requiere formularios bien disenados, reglas de validacion, catalogos, datos maestros, responsabilidades y formacion.

### Remediacion

Es el conjunto de actuaciones para corregir, contener, explicar o compensar problemas de calidad. Puede incluir limpieza de registros, deduplicacion, enriquecimiento, reclasificacion, correccion de reglas, cambios de proceso, revision manual, bloqueo de uso o comunicacion a consumidores del dato.

### Dato maestro

Es un dato de referencia esencial y compartido por varios procesos o sistemas, como persona, entidad, organo, procedimiento, territorio, expediente, unidad administrativa o catalogo de servicios. Su mala calidad tiene efecto multiplicador.

### Linaje del dato

Es la informacion que permite saber de donde procede un dato, como se ha transformado, que sistemas lo han tratado, que reglas se le han aplicado y en que productos o decisiones se utiliza.

## 4. Dimensiones de calidad del dato

Las dimensiones son categorias para ordenar la medicion. No todas tienen la misma importancia en todos los casos. En un expediente sancionador, la exactitud, trazabilidad y validez juridica pueden ser criticas. En un portal de datos abiertos, la actualidad, la documentacion, la disponibilidad y la comprensibilidad pueden ser decisivas. En un intercambio entre administraciones, la consistencia semantica y la conformidad con modelos comunes cobran especial relevancia.

| Dimension | Pregunta de control | Ejemplo administrativo | Indicador posible | Riesgo si falla |
|---|---|---|---|---|
| Exactitud | El dato refleja correctamente la realidad o la fuente competente? | Importe concedido en una subvencion | Porcentaje de registros contrastados sin discrepancia | Resoluciones incorrectas, pagos indebidos, reclamaciones |
| Completitud | Estan todos los datos necesarios? | Expediente con interesado, organo, procedimiento, fecha y documentos esenciales | Campos obligatorios cumplimentados por tipo de procedimiento | Tramitacion incompleta, retrasos, imposibilidad de resolver |
| Validez | Cumple el formato, dominio o regla definida? | Codigo DIR3, NIF, fecha ISO, codigo de procedimiento | Registros que superan validaciones de formato y catalogo | Rechazos en intercambios, errores de carga, ambiguedad |
| Consistencia | No se contradice con otros datos? | Fecha de notificacion posterior a fecha de resolucion | Reglas cruzadas superadas | Incoherencia del expediente, perdida de confianza |
| Unicidad | El mismo objeto no aparece duplicado sin control? | Un mismo expediente registrado dos veces | Duplicados detectados por claves o similitud | Doble tramitacion, informes inflados, pagos duplicados |
| Actualidad | El dato esta vigente para su uso? | Domicilio de notificacion o estado de una licencia | Antiguedad media, registros vencidos | Notificaciones fallidas, decisiones con informacion obsoleta |
| Integridad referencial | Las relaciones entre entidades son validas? | Documento asociado a expediente existente | Enlaces rotos o referencias inexistentes | Expedientes incompletos, perdida documental |
| Trazabilidad | Puede reconstruirse origen, cambios y uso? | Correccion de un dato en un procedimiento | Registros con historico y usuario/proceso responsable | Debilidad probatoria, dificultad de auditoria |
| Comprensibilidad | El dato puede interpretarse correctamente? | Campo "estado" con codigos documentados | Campos con definicion, catalogo y ejemplo | Reutilizacion incorrecta, errores de analisis |
| Disponibilidad | El dato esta accesible cuando se necesita? | Servicio de consulta de datos de identidad | Tiempo de disponibilidad y respuesta | Paralizacion de tramites, reintentos, cargas manuales |
| Oportunidad | Llega a tiempo para el proceso? | Actualizacion de padron para una ayuda con plazo | Latencia desde origen hasta consumo | Decisiones tardias o fuera de plazo |
| Conformidad | Respeta normas, politicas y contratos de datos? | Metadatos ENI, esquema de intercambio, licencia de reutilizacion | Reglas normativas superadas | Incumplimiento, incompatibilidad, bloqueo de publicacion |

La tabla no debe usarse como lista cerrada. Una organizacion madura selecciona dimensiones prioritarias por dominio de datos. Por ejemplo, en datos economico-presupuestarios puede pesar mucho la exactitud y conciliacion; en datos geograficos, precision espacial y vigencia; en datos personales, minimizacion, licitud, exactitud y actualizacion; en datos abiertos, documentacion, formato reutilizable, licencia, actualizacion y estabilidad del identificador.

## 5. Reglas de calidad: como se disenan

Una regla de calidad util debe ser comprensible por el responsable funcional y ejecutable por el responsable tecnico. Si solo la entiende el area tecnica, puede medir algo irrelevante. Si solo la formula el area funcional sin precision, no se podra automatizar.

Una buena regla suele incluir:

- Nombre de la regla.
- Dato o conjunto de datos afectado.
- Dimension de calidad que mide.
- Condicion verificable.
- Fuente de autoridad o razon funcional.
- Severidad del incumplimiento.
- Umbral aceptable.
- Momento de aplicacion.
- Responsable de resolver incidencias.
- Evidencia que debe conservarse.

Ejemplo de regla simple:

| Elemento | Contenido |
|---|---|
| Nombre | Validez de fecha de solicitud |
| Dato | Fecha de solicitud en expediente |
| Dimension | Validez y consistencia |
| Condicion | Debe existir y no puede ser posterior a la fecha de registro |
| Fuente | Logica del procedimiento y registro administrativo |
| Severidad | Alta |
| Umbral | Cero errores en expedientes en tramitacion |
| Momento | Captura y revision previa a instruccion |
| Responsable | Unidad tramitadora |
| Evidencia | Registro de validacion y, si procede, subsanacion |

Ejemplo de regla semantica:

| Elemento | Contenido |
|---|---|
| Nombre | Clasificacion del tipo de tramite |
| Dato | Tipo de tramite asociado a procedimiento |
| Dimension | Conformidad semantica |
| Condicion | El valor debe pertenecer al vocabulario aprobado para el procedimiento |
| Fuente | Catalogo corporativo de procedimientos y tramites |
| Severidad | Media o alta segun impacto |
| Umbral | Al menos 99 por ciento de registros clasificados |
| Momento | Alta del expediente y cambios de fase |
| Responsable | Gestor del catalogo y unidad propietaria |
| Evidencia | Version del vocabulario aplicada |

No todas las reglas deben tener severidad maxima. Una gestion realista distingue entre errores bloqueantes, advertencias y oportunidades de mejora. Bloqueante es aquello que impide tramitar, decidir, intercambiar o publicar con seguridad. Advertencia es aquello que conviene revisar pero no invalida el proceso. Mejora es aquello que aumenta utilidad, analitica o reutilizacion sin afectar al nucleo juridico.

## 6. Perfilado de datos

El perfilado es el punto de partida para conocer la calidad real. Muchas organizaciones creen que sus datos cumplen reglas que nunca se han medido. El perfilado descubre distribuciones inesperadas, codificaciones antiguas, campos usados para fines distintos, valores libres donde deberia haber catalogo, duplicados y excepciones no documentadas.

El perfilado puede responder, entre otras, a estas preguntas:

- Que porcentaje de valores nulos tiene cada campo?
- Que formatos aparecen realmente en una fecha, identificador o codigo?
- Hay valores fuera del catalogo esperado?
- Existen duplicados exactos o aproximados?
- Que campos parecen depender de otros?
- Hay registros sin relacion con entidades obligatorias?
- Que valores son anomalos por frecuencia, rango o patron?
- Existen diferencias de calidad por unidad administrativa, periodo, canal o sistema de origen?

Ejemplo: en un conjunto de expedientes de ayudas, el perfilado detecta que el campo "municipio" contiene codigos oficiales, nombres en texto libre, abreviaturas y valores historicos. Aunque el campo parezca completo porque casi nunca esta vacio, su calidad semantica es baja. El problema no es de completitud, sino de validez, normalizacion y conformidad con catalogo.

El perfilado no debe interpretarse de forma aislada. Un valor atipico no siempre es un error. Una ayuda de importe muy alto puede ser excepcional pero correcta. Una fecha antigua puede ser coherente en expedientes de revision. Una baja tasa de nulos puede ocultar valores comodin como "desconocido", "N/A" o "9999". Por eso el perfilado debe combinar analisis automatico y conocimiento funcional.

### Tecnicas habituales de perfilado

| Tecnica | Para que sirve | Ejemplo |
|---|---|---|
| Conteo de nulos | Medir completitud | Expedientes sin fecha de inicio |
| Distribucion de valores | Detectar codigos dominantes o raros | Estados de tramitacion no documentados |
| Analisis de patrones | Revisar formatos | Identificadores con longitudes distintas |
| Deteccion de duplicados | Encontrar repeticion de entidades | Misma persona y mismo procedimiento con dos expedientes activos |
| Integridad referencial | Verificar enlaces | Documento sin expediente asociado |
| Reglas cruzadas | Detectar incoherencias | Resolucion anterior a informe preceptivo |
| Frescura | Medir actualizacion | Dataset publicado sin actualizacion desde el periodo comprometido |
| Comparacion con fuente autorizada | Comprobar exactitud | Codigo territorial contra catalogo oficial |

## 7. Validacion: controles en el ciclo de vida

La validacion debe distribuirse a lo largo del ciclo de vida del dato. Si se valida solo al final, el sistema acumula errores, se normalizan malas practicas y la correccion resulta mas cara. Si se valida solo al principio, pueden aparecer errores durante transformaciones, integraciones, migraciones o publicaciones.

### Validacion en captura

Es el control aplicado cuando el dato se introduce o recibe. Incluye campos obligatorios, mascaras de formato, seleccion por catalogo, comprobaciones basicas de coherencia y ayuda contextual. Su objetivo es evitar errores evitables sin hacer imposible la tramitacion.

Ejemplo: un formulario de solicitud no deberia permitir seleccionar una provincia inexistente ni introducir una fecha imposible. Pero debe contemplar excepciones legitimas cuando el procedimiento lo permita, documentando la causa y la responsabilidad.

### Validacion en intercambio

Antes de enviar o recibir datos entre sistemas, deben comprobarse contratos, esquemas, codificaciones, versiones de catalogo, obligatoriedad de campos y reglas de seguridad. En la interoperabilidad entre administraciones, un intercambio tecnicamente exitoso puede ser funcionalmente inutil si los datos no tienen significado compartido.

Ejemplo: dos sistemas pueden enviar el campo "estado" como texto. Si uno usa "pendiente" para expediente no iniciado y otro lo usa para expediente pendiente de subsanacion, el intercambio crea ambiguedad semantica.

### Validacion en transformacion

Los procesos de integracion, analitica, anonimizado, agregacion o publicacion pueden introducir errores. Por eso conviene validar antes y despues de transformar. Se deben comprobar perdidas de registros, cambios de tipo, truncamientos, agrupaciones erroneas, conversiones de fecha, codificaciones y redondeos.

Ejemplo: al publicar datos de contratos en formato reutilizable, una transformacion puede convertir importes decimales en texto o redondear cifras de forma inadecuada. El dato publicado seria menos reutilizable y podria inducir errores.

### Validacion antes de decision administrativa

Cuando un dato alimenta una decision, una propuesta de resolucion o una actuacion automatizada, el umbral debe ser mas exigente. Deben verificarse fuente, actualidad, correspondencia con expediente, trazabilidad y reglas materiales. La calidad aqui no es solo tecnica; afecta a derechos e intereses.

Ejemplo: antes de denegar una ayuda por superar un umbral de renta, la Administracion debe poder justificar que el dato usado procede de fuente competente, corresponde a la persona afectada, esta actualizado para el periodo aplicable y se ha interpretado conforme a la norma.

## 8. Observabilidad de datos

La observabilidad de datos traslada al dato la idea de que no basta con construir un flujo: hay que saber si funciona correctamente, si los datos llegan, si llegan a tiempo, si conservan estructura, si varian de forma razonable y si los consumidores pueden confiar en ellos.

En una organizacion publica, la observabilidad es especialmente importante porque muchos procesos dependen de cadenas de sistemas. Un registro de entrada, un gestor de expedientes, una plataforma de intermediacion, un archivo electronico, un sistema de notificaciones, un portal de transparencia y un portal de datos abiertos pueden compartir informacion. Si un cambio en origen rompe una regla, el impacto puede aparecer lejos del sistema que genero el error.

### Controles de observabilidad

| Control | Descripcion | Ejemplo de alerta |
|---|---|---|
| Esquema | Verifica campos, tipos y obligatoriedad | Desaparece el campo de codigo de procedimiento |
| Volumen | Compara numero de registros esperado y recibido | Caida brusca de expedientes recibidos en una integracion diaria |
| Frescura | Mide si el dato se actualiza dentro del plazo esperado | Dataset mensual sin actualizacion tras cierre del mes |
| Distribucion | Detecta cambios anomalos en valores | Aumento repentino de expedientes en estado "otros" |
| Duplicidad | Vigila repeticion de entidades | Incremento de expedientes duplicados por canal |
| Integridad | Revisa relaciones entre tablas o entidades | Documentos sin expediente o expedientes sin interesado |
| Linaje | Permite reconstruir transformaciones | No se puede identificar el proceso que cambio un campo |
| Calidad por dominio | Mide reglas especificas | Registros de subvenciones sin convocatoria asociada |

La observabilidad debe producir alertas accionables. Una alerta que nadie atiende o que se dispara continuamente pierde valor. Cada alerta relevante necesita severidad, responsable, plazo de respuesta, posible contencion e impacto. Tambien conviene separar los incidentes de calidad que afectan a la operacion de los hallazgos de mejora que pueden planificarse.

## 9. Calidad en origen

La calidad en origen es el enfoque mas eficiente. El dato nace en un proceso, con una finalidad y bajo responsabilidad. Si se captura mal, se propaga mal. Corregir despues suele requerir conciliaciones, cargas manuales, interpretaciones y excepciones que reducen la confianza.

Para asegurar calidad en origen se combinan medidas organizativas, funcionales y tecnicas:

- Formularios claros, con campos necesarios y ayuda contextual.
- Catalogos controlados en lugar de texto libre cuando exista dominio conocido.
- Validaciones proporcionales al impacto del dato.
- Integracion con fuentes autorizadas para evitar pedir datos ya disponibles.
- Responsables de dato con capacidad real de decision.
- Revision periodica de reglas cuando cambian procedimientos o normativa.
- Formacion de unidades tramitadoras sobre el significado de los datos.
- Mecanismos de correccion y subsanacion trazables.
- Diseno de procesos que evite duplicar captura de la misma informacion.

Ejemplo: si varias unidades capturan el tipo de procedimiento en texto libre, despues sera dificil comparar plazos, cargas y resultados. Si existe un catalogo corporativo de procedimientos, integrado en la herramienta de tramitacion, con reglas de seleccion y versionado, la calidad nace mucho mas controlada.

Calidad en origen no significa rigidez absoluta. En la Administracion hay excepciones, supuestos no previstos, procedimientos historicos y situaciones transitorias. La madurez consiste en permitir excepciones documentadas, no en ocultarlas dentro de campos libres. Una excepcion registrada como tal puede ser gobernada; una excepcion mezclada con texto libre se convierte en ruido.

## 10. Remediacion de problemas de calidad

La remediacion empieza cuando se detecta un problema y termina cuando se ha corregido, contenido, aceptado justificadamente o incorporado a un plan de mejora. No todos los problemas deben resolverse del mismo modo. Hay que priorizar segun impacto en derechos, cumplimiento normativo, continuidad de servicio, seguridad, reutilizacion e imagen institucional.

### Tipos de remediacion

| Tipo | Uso adecuado | Ejemplo |
|---|---|---|
| Correccion en origen | Cuando el sistema fuente mantiene el dato maestro | Corregir codigo de procedimiento en el catalogo corporativo |
| Limpieza puntual | Cuando hay un lote historico con errores acotados | Normalizar municipios en expedientes cerrados |
| Deduplicacion | Cuando existen registros repetidos | Fusionar entidades duplicadas con reglas de supervivencia |
| Enriquecimiento | Cuando falta informacion obtenible de fuente fiable | Completar codigos territoriales a partir de catalogo |
| Reclasificacion | Cuando una taxonomia cambio o se uso mal | Reasignar tipos de tramite a vocabulario vigente |
| Contencion | Cuando no se puede corregir de inmediato | Bloquear publicacion de un dataset con errores graves |
| Excepcion documentada | Cuando el dato es anomalo pero valido | Expediente con plazo especial justificado |
| Cambio de proceso | Cuando el error nace de una practica organizativa | Sustituir campo libre por catalogo validado |

La remediacion debe evitar dos errores: corregir silenciosamente y corregir solo el sintoma. La correccion silenciosa puede destruir trazabilidad. La correccion del sintoma deja el problema vivo. Si hay muchos expedientes sin organo responsable, no basta rellenar el campo en un lote; hay que saber por que se generaron asi y como evitar que vuelva a ocurrir.

### Criterios de priorizacion

Un modelo sencillo de priorizacion puede cruzar impacto y urgencia:

| Impacto | Ejemplos | Tratamiento |
|---|---|---|
| Critico | Afecta a derechos, plazos, pagos, sanciones, notificaciones o legalidad | Correccion inmediata, contencion y evidencia |
| Alto | Afecta a interoperabilidad, informes oficiales o publicacion relevante | Plan de remediacion con responsable y fecha |
| Medio | Afecta a analitica interna o eficiencia | Correccion planificada |
| Bajo | Mejora descriptiva o estetica sin efecto operativo | Acumular en backlog de calidad |

En datos publicos reutilizables, un error puede no afectar a un expediente individual, pero si a la confianza ciudadana y al ecosistema reutilizador. Si un dataset de presupuestos se publica con importes mal tipados, fechas ambiguas o metadatos pobres, reduce la capacidad de analisis externo y genera costes a terceros.

## 11. Medicion de la calidad

Lo que no se mide no se gobierna. Pero medir demasiadas cosas sin criterio puede crear burocracia inutil. La medicion debe centrarse en indicadores vinculados a riesgos y usos reales.

Un indicador de calidad debe incluir:

- Dimension medida.
- Formula.
- Unidad de analisis.
- Frecuencia de medicion.
- Umbral aceptable.
- Tendencia esperada.
- Responsable.
- Uso del indicador.
- Limitaciones de interpretacion.

Ejemplo:

| Indicador | Formula | Umbral | Uso |
|---|---|---|---|
| Completitud de metadatos obligatorios | Registros con todos los metadatos obligatorios / total de registros | 98 por ciento | Control de publicacion e interoperabilidad |
| Duplicidad de expedientes activos | Expedientes activos potencialmente duplicados / total de expedientes activos | Menos de 0,5 por ciento | Prevencion de doble tramitacion |
| Frescura de dataset reutilizable | Dias desde ultima actualizacion efectiva | Segun periodicidad comprometida | Cumplimiento de calendario de publicacion |
| Validez de codigos territoriales | Registros con codigo valido / total con codigo territorial | 99 por ciento | Normalizacion e integracion |
| Trazabilidad de cambios criticos | Cambios con usuario/proceso, fecha y motivo / total de cambios criticos | 100 por ciento | Auditoria y seguridad juridica |

La medicion debe permitir comparaciones utiles, pero con cuidado. Comparar unidades administrativas sin considerar volumen, complejidad, antiguedad de sistemas o tipo de procedimiento puede ser injusto. El objetivo no es castigar, sino mejorar. La calidad del dato es una responsabilidad compartida, aunque debe tener propietarios claros.

## 12. Cuadros de mando de calidad

Un cuadro de mando de calidad debe responder a tres preguntas: como estamos, donde esta el riesgo y que hay que hacer. No debe limitarse a graficos vistosos. Debe ayudar a decidir.

### Componentes recomendables

| Componente | Funcion |
|---|---|
| Semaforo por dominio de datos | Vista rapida de salud por expediente, personas, procedimientos, territorio, contratos, subvenciones o datasets |
| Tendencia temporal | Saber si la calidad mejora o empeora |
| Ranking de reglas incumplidas | Identificar problemas recurrentes |
| Impacto funcional | Relacionar errores con tramites, publicaciones o servicios afectados |
| Incidencias abiertas | Ver estado de remediacion |
| Calidad por sistema origen | Detectar donde nace el problema |
| Calidad por consumidor | Ver a quien afecta el problema |
| Alertas criticas | Priorizar fallos bloqueantes |
| Metadatos de confianza | Mostrar fecha de actualizacion, responsable y limitaciones |

Un buen cuadro de mando distingue entre calidad tecnica y calidad de negocio. Por ejemplo, un dataset puede tener todos los campos con formato valido y aun asi ser poco util porque usa categorias no documentadas, no indica periodicidad, mezcla ejercicios presupuestarios o carece de licencia clara. A la inversa, un dato puede incumplir una regla formal menor sin afectar a la decision.

### Ejemplo de lectura de cuadro de mando

Supongamos un cuadro de mando de expedientes de ayudas. Muestra completitud del 99 por ciento, duplicidad del 0,2 por ciento y trazabilidad de cambios criticos del 100 por ciento. A primera vista, la calidad parece alta. Sin embargo, el indicador de actualidad del domicilio de notificacion esta en 82 por ciento y la tasa de notificaciones fallidas ha subido. La conclusion no es que "los datos son buenos", sino que existe un riesgo concreto en una dimension concreta que afecta a un proceso concreto.

## 13. Calidad del dato y expediente administrativo

El expediente administrativo es el conjunto ordenado de documentos y actuaciones que sirven de antecedente y fundamento a la resolucion administrativa, asi como las diligencias encaminadas a ejecutarla. En el entorno electronico, el expediente no es solo una carpeta documental: tambien depende de metadatos, identificadores, relaciones, estados, trazas y datos estructurados que permiten tramitar, consultar, interoperar, conservar y acreditar.

La calidad del dato se relaciona con el expediente en varios niveles:

### Identificacion del expediente

Cada expediente debe tener identificador unico, organo responsable, procedimiento, interesado cuando proceda, fechas relevantes y estado. Si estos datos fallan, se dificulta la localizacion, el seguimiento, el intercambio, la acumulacion, el archivo y la rendicion de cuentas.

### Integridad documental

Los documentos deben estar asociados al expediente correcto, con metadatos suficientes y sin rupturas de relacion. Un documento valido en si mismo puede estar mal integrado si se vincula al expediente equivocado o carece de referencia a la actuacion que lo produjo.

### Trazabilidad de actuaciones

Las actuaciones administrativas deben poder reconstruirse: quien actuo, cuando, con que base, que dato se uso y que cambio se produjo. La trazabilidad de datos criticos refuerza la seguridad juridica y facilita auditorias, recursos, revisiones y control interno.

### Coherencia temporal

En un expediente importan los tiempos. Fechas de solicitud, registro, subsanacion, informe, audiencia, propuesta, resolucion, notificacion y recurso deben guardar coherencia. Errores de fechas pueden afectar al computo de plazos y a la validez de actuaciones.

### Interoperabilidad del expediente

Cuando el expediente se remite, consulta o archiva, sus datos y metadatos deben ser entendibles por otros sistemas. La interoperabilidad no se consigue solo enviando ficheros; exige estructura, metadatos, formatos, codigos y significado compartido.

### Conservacion y archivo

La calidad tambien afecta a la conservacion a largo plazo. Datos sin metadatos suficientes pierden contexto. Un expediente conservado sin informacion sobre procedencia, firmas, estados o relaciones puede ser dificil de interpretar en el futuro.

## 14. Relacion con interoperabilidad semantica

La interoperabilidad semantica busca que la informacion intercambiada tenga el mismo significado para emisor y receptor. La calidad del dato es su aliada natural. Si un concepto no esta definido, si un catalogo no esta versionado o si un campo mezcla significados, la interoperabilidad semantica se debilita.

Ejemplo: "fecha de alta" puede significar fecha de alta en el sistema, fecha de inicio del procedimiento, fecha de presentacion de solicitud o fecha de alta de una persona en un registro. Si no se define el concepto, dos sistemas pueden intercambiar el campo con exito tecnico y fracasar semanticamente.

Para mejorar la calidad semantica se usan:

- Glosarios de terminos.
- Vocabularios controlados.
- Catalogos de datos.
- Modelos conceptuales.
- Identificadores persistentes.
- Metadatos de definicion, origen, version y vigencia.
- Reglas de transformacion documentadas.
- Acuerdos entre productores y consumidores.

La calidad semantica no es un lujo academico. En administraciones con multiples organismos, niveles territoriales y sistemas historicos, la falta de significado comun produce errores de interpretacion, duplicidades y costes de integracion.

## 15. Relacion con reutilizacion de la informacion publica

La reutilizacion exige que terceros puedan encontrar, entender y usar informacion publica con seguridad juridica y tecnica. La calidad del dato condiciona esa reutilizacion. Un conjunto de datos puede estar publicado, pero ser poco reutilizable si no esta actualizado, si no explica campos, si mezcla formatos, si carece de licencia clara o si cambia sin aviso.

Desde la perspectiva de reutilizacion, son especialmente importantes:

- Metadatos descriptivos suficientes.
- Periodicidad de actualizacion.
- Formatos abiertos y procesables.
- Estabilidad de esquemas.
- Identificadores persistentes.
- Documentacion de cambios.
- Calidad de codificaciones.
- Ausencia de datos personales no pertinentes.
- Condiciones de uso claras.
- Contacto o canal de incidencias.

Ejemplo: publicar un listado de equipamientos publicos en PDF puede cumplir una funcion informativa minima, pero limita la reutilizacion. Publicarlo en formato estructurado, con coordenadas validas, codigos territoriales, fecha de actualizacion, licencia y definicion de campos multiplica su valor. Si ademas se mantiene la calidad en el tiempo, terceros pueden construir servicios, analisis y visualizaciones confiables.

## 16. Ejemplos trabajados

### Ejemplo 1: padron o registro territorial usado para una ayuda

Una ayuda se concede a personas residentes en determinados municipios. El sistema recibe municipio en texto libre. Aparecen variantes como "Sevilla", "Sev.", "SEVILLA", codigos numericos y nombres de barrios. El problema inicial parece de formato, pero en realidad afecta a validez, consistencia semantica y exactitud.

Solucion razonable: sustituir texto libre por catalogo territorial vigente, conservar historico cuando proceda, validar el municipio al capturar, registrar excepciones y revisar expedientes activos afectados. En cuadros de mando, medir porcentaje de registros con codigo territorial valido y numero de correcciones manuales por unidad.

### Ejemplo 2: duplicidad de expedientes por entrada multicanal

Una persona presenta solicitud por sede electronica y, dias despues, por registro presencial. Si el sistema no detecta similitud por interesado, procedimiento y periodo, puede abrir dos expedientes. La duplicidad afecta a unicidad, eficiencia y riesgo juridico.

Solucion razonable: reglas de deteccion temprana, alerta a unidad tramitadora, criterios de acumulacion o cierre, trazabilidad de la decision y actualizacion de indicadores de duplicidad. No debe hacerse una fusion automatica sin evaluar el procedimiento y las garantias.

### Ejemplo 3: portal de datos abiertos con actualizacion irregular

Un dataset indica periodicidad mensual, pero se actualiza de forma irregular. Los reutilizadores no saben si la ausencia de cambios significa que no hay datos nuevos o que la publicacion esta retrasada. El problema es de actualidad, oportunidad, metadatos y confianza.

Solucion razonable: medir frescura, publicar fecha de ultima actualizacion efectiva, registrar incidencias, ajustar periodicidad si el compromiso era irreal y evitar publicar series incompletas sin advertencia editorial.

### Ejemplo 4: actuacion automatizada basada en dato externo

Un procedimiento automatiza una comprobacion de requisitos consultando un dato de otra Administracion. Si la respuesta se interpreta sin version, sin fecha de obtencion o sin identificador claro de fuente, la decision puede ser dificil de justificar.

Solucion razonable: registrar fuente, fecha, finalidad, version del esquema, resultado recibido, regla aplicada y, cuando proceda, mecanismo de contraste o subsanacion. La calidad aqui se vincula directamente a garantias del procedimiento.

## 17. Supuestos practicos guiados

### Supuesto 1: expediente con errores de fechas

Situacion: en una revision de expedientes se detecta que un 3 por ciento tiene fecha de resolucion anterior a la fecha del informe preceptivo. El sistema permite guardar fechas sin control cruzado.

Pistas:

- No es solo un error de formato.
- Afecta a coherencia temporal y posible validez del procedimiento.
- Debe analizarse si el error esta en la fecha de resolucion, en el informe o en la asociacion documental.

Preguntas:

1. Que dimensiones de calidad estan afectadas?
2. Que controles deben aplicarse en origen?
3. Como se prioriza la remediacion?
4. Que evidencia debe conservarse?

Resolucion orientativa: estan afectadas consistencia, validez, trazabilidad e integridad del expediente. Debe incorporarse una regla que impida o advierta incoherencias temporales, diferenciando errores bloqueantes de excepciones justificadas. La remediacion debe priorizar expedientes vivos, expedientes con recursos o procedimientos donde el informe sea determinante. Debe conservarse historico de correccion, motivo, responsable y fecha.

Errores habituales: limitarse a corregir fechas sin revisar la regla de captura; asumir que toda fecha anomala es nula de pleno derecho; no distinguir entre dato mal introducido y documento mal asociado.

Mini comprobacion: si tras corregir el lote el sistema sigue permitiendo la misma incoherencia, la remediacion esta incompleta.

### Supuesto 2: publicacion reutilizable con codigos no documentados

Situacion: se publica un conjunto de datos sobre subvenciones. El campo "tipo_beneficiario" contiene codigos A, B, C y X, pero no existe diccionario. Los reutilizadores preguntan que significa cada valor.

Pistas:

- El dato puede ser tecnicamente valido y semanticamente opaco.
- El problema afecta a reutilizacion y comprensibilidad.
- La solucion no es solo cambiar nombres de columnas.

Preguntas:

1. Que dimensiones fallan?
2. Que metadatos deben anadirse?
3. Como se evita que vuelva a pasar?

Resolucion orientativa: fallan comprensibilidad, conformidad semantica y posiblemente documentacion de metadatos. Debe incorporarse diccionario de datos, definicion de cada codigo, version del catalogo, fecha de actualizacion y contacto o mecanismo de incidencias. Para evitar repeticion, la publicacion debe pasar por validacion previa de metadatos y catalogos.

Errores habituales: creer que publicar el fichero basta; incluir explicaciones en notas dispersas no versionadas; modificar codigos historicos sin equivalencias.

Mini comprobacion: una persona externa debe poder interpretar el campo sin consultar al equipo productor.

### Supuesto 3: datos maestros de organos administrativos

Situacion: varios sistemas mantienen su propia lista de organos. Un mismo organo aparece con nombres ligeramente distintos, codigos antiguos y relaciones jerarquicas divergentes. Los informes corporativos no cuadran.

Pistas:

- Hay un problema de dato maestro.
- La duplicidad no es solo tecnica; afecta a gobierno y responsabilidad.
- Puede requerir catalogo de autoridad y reglas de sincronizacion.

Preguntas:

1. Que riesgos produce?
2. Que medidas organizativas y tecnicas se recomiendan?
3. Que indicadores se podrian usar?

Resolucion orientativa: produce informes inconsistentes, errores de asignacion, problemas de interoperabilidad y baja confianza. Se recomienda fuente maestra, responsables de mantenimiento, reglas de alta/baja/modificacion, historico de vigencia, identificadores estables y sincronizacion controlada con sistemas consumidores. Indicadores: porcentaje de sistemas alineados, codigos obsoletos, organos duplicados, registros sin fecha de vigencia y tiempo medio de propagacion de cambios.

Errores habituales: intentar resolverlo con una limpieza unica; no definir quien tiene autoridad; borrar codigos historicos necesarios para interpretar expedientes antiguos.

Mini comprobacion: si dos sistemas pueden crear organos sin coordinacion, el problema reaparecera.

## 18. Notas de test: trampas y distractores frecuentes

> Nota de test. Calidad del dato no equivale a proteccion de datos, aunque se relacionan. La proteccion de datos exige, entre otros principios, exactitud y actualizacion cuando se tratan datos personales. Pero la calidad del dato tambien aplica a datos no personales, datos administrativos, datos abiertos, catalogos y metadatos.

> Nota de test. Interoperabilidad tecnica no garantiza interoperabilidad semantica. Que dos sistemas intercambien un fichero correctamente no significa que interpreten igual los campos.

> Nota de test. Completitud no es lo mismo que exactitud. Un campo puede estar lleno en el 100 por cien de registros y estar mal. Tambien puede estar incompleto pero ser exacto en los registros que si tienen valor.

> Nota de test. Validez formal no implica correccion material. Un NIF puede tener formato valido y no corresponder a la persona interesada. Una fecha puede tener formato correcto y ser incoherente con el expediente.

> Nota de test. La calidad no debe medirse solo con indicadores tecnicos. En examen puede aparecer una opcion que hable de "porcentaje de nulos" como unica medida. Es insuficiente si no se vincula al proceso y al impacto.

> Nota de test. La remediacion no siempre consiste en borrar o sobrescribir. En Administracion suele ser necesario conservar historico, justificar correcciones y documentar excepciones.

> Nota de test. Calidad en origen no significa impedir toda excepcion. Significa controlar las excepciones, hacerlas visibles y trazables.

> Nota de test. Un cuadro de mando con muchos graficos no es necesariamente un buen sistema de calidad. Lo relevante es que los indicadores sean accionables, tengan responsable y permitan priorizar.

> Nota de test. En reutilizacion, publicar datos no equivale a publicar datos reutilizables. La reutilizacion requiere formato, metadatos, licencia, actualizacion, documentacion y estabilidad suficientes.

> Nota de test. En expediente administrativo, los metadatos no son decorativos. Son parte esencial para identificar, contextualizar, conservar, intercambiar y probar actuaciones.

## 19. Errores frecuentes de enfoque

1. Tratar la calidad como limpieza final. La limpieza final puede ser necesaria, pero la estrategia madura es prevenir, medir y corregir en origen.
2. Confundir dato disponible con dato confiable. Que un dato exista en un sistema no demuestra que sea apto para decidir.
3. Medir sin actuar. Los indicadores que no generan decisiones producen falsa sensacion de control.
4. Aplicar la misma severidad a todos los errores. No todos los problemas de calidad tienen igual impacto.
5. Resolver duplicados sin criterio funcional. La fusion de entidades puede tener efectos juridicos y debe ser trazable.
6. Ignorar los datos historicos. Cambiar codigos o catalogos sin equivalencias puede romper series, expedientes y analisis.
7. Dejar la calidad solo en manos del area tecnica. La calidad requiere responsables funcionales, juridicos y tecnicos.
8. Documentar tarde. Si las reglas, metadatos y decisiones no se documentan al disenar el proceso, despues se reconstruyen con dificultad.
9. Abrir datos sin control de calidad. La transparencia y la reutilizacion necesitan confianza, no solo descarga.
10. No revisar reglas tras cambios normativos o procedimentales. Una regla correcta en una version del procedimiento puede quedar obsoleta.

## 20. Mini esquema para visual

Visual sugerido para el tema principal: ciclo de calidad del dato en seis pasos.

1. Definir: conceptos, responsables, reglas y usos.
2. Capturar: validaciones, catalogos y calidad en origen.
3. Medir: perfilado, indicadores y controles.
4. Observar: alertas, linaje, frescura y ruptura de contratos.
5. Remediar: correccion, contencion, excepciones y mejora de proceso.
6. Reutilizar o decidir: expediente, interoperabilidad, analitica y publicacion.

El visual debe mostrar que el ciclo vuelve a "definir", porque cada incidencia relevante puede obligar a revisar reglas, procesos o responsabilidades.

## 21. Fuentes oficiales y marco de referencia para citar editorialmente

- Ley 39/2015, del Procedimiento Administrativo Comun de las Administraciones Publicas.
- Ley 40/2015, de Regimen Juridico del Sector Publico.
- Real Decreto 203/2021, por el que se aprueba el Reglamento de actuacion y funcionamiento del sector publico por medios electronicos.
- Real Decreto 4/2010, por el que se regula el Esquema Nacional de Interoperabilidad.
- Real Decreto 311/2022, por el que se regula el Esquema Nacional de Seguridad.
- Normas Tecnicas de Interoperabilidad relativas a documento electronico, expediente electronico, politica de firma, catalogo de estandares, requisitos de conexion y procedimientos de copiado autentico y conversion.
- Ley 37/2007, sobre reutilizacion de la informacion del sector publico.
- Directiva europea sobre datos abiertos y reutilizacion de la informacion del sector publico.
- Reglamento europeo de Gobernanza de Datos.
- Reglamento General de Proteccion de Datos y normativa organica espanola de proteccion de datos, en cuanto a exactitud, responsabilidad proactiva y calidad en tratamientos de datos personales.
- Guias y criterios de administracion digital, interoperabilidad, datos abiertos, catalogos de datos y gobierno del dato publicados por administraciones publicas competentes.

## 22. Resumen para integracion

La calidad del dato es la disciplina que permite confiar en que los datos publicos son aptos para tramitar, decidir, intercambiar, publicar, conservar y reutilizar. Se concreta en dimensiones medibles, reglas explicitas, perfilado, validacion, observabilidad, remediacion y cuadros de mando. Su aplicacion en la Administracion publica exige conectar tecnica y derecho: un dato debe ser valido, consistente y trazable, pero tambien proceder de fuente adecuada, conservar contexto y respetar la finalidad del procedimiento. En el expediente administrativo, la calidad refuerza identificacion, integridad, coherencia temporal, trazabilidad y valor probatorio. En interoperabilidad semantica, asegura significado comun. En reutilizacion, transforma la publicacion formal en informacion realmente util.
