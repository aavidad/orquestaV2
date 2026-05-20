# Marco normativo y fuentes oficiales

## Enfoque general

Las arquitecturas distribuidas, los microservicios, las APIs y la integracion de sistemas en la Administracion publica no pueden explicarse solo como una decision tecnica. En el sector publico, la arquitectura es tambien una forma de cumplir obligaciones juridicas: relacion electronica con ciudadanos y empresas, cooperacion entre administraciones, interoperabilidad, seguridad, trazabilidad, conservacion documental, reutilizacion, accesibilidad, proteccion de datos y servicios publicos digitales transfronterizos. El marco normativo no impone una tecnologia concreta, pero si fija condiciones que favorecen arquitecturas modulares, interoperables, seguras y auditables.

Para un tema A1 conviene presentar este bloque como una cadena normativa. En la base estatal estan la Ley 39/2015, del Procedimiento Administrativo Comun, y la Ley 40/2015, de Regimen Juridico del Sector Publico. El Real Decreto 203/2021 desarrolla ambas leyes para la actuacion electronica. El Esquema Nacional de Interoperabilidad y el Esquema Nacional de Seguridad concretan requisitos tecnicos, organizativos y documentales. En el plano europeo, eIDAS y eIDAS2 ordenan identidad y servicios de confianza; NIS2 refuerza la ciberseguridad; y el Reglamento Europa Interoperable introduce gobernanza para servicios publicos digitales transfronterizos. La reutilizacion de informacion y de soluciones completa el cuadro.

## Ley 39/2015: procedimiento electronico y derechos de relacion

La Ley 39/2015, aprobada por la Jefatura del Estado y publicada en el Boletin Oficial del Estado, explica por que los sistemas administrativos deben tramitar electronicamente de extremo a extremo. Sus preceptos sobre identificacion y firma de los interesados, derecho y obligacion de relacionarse electronicamente, registros, documentos, copias, notificaciones y expediente electronico obligan a que las aplicaciones publicas no sean silos aislados, sino piezas conectadas de un procedimiento comun.

En terminos arquitectonicos, el articulo 14 diferencia entre personas fisicas, que en general pueden elegir el canal, y sujetos obligados a relacionarse electronicamente, como personas juridicas, entidades sin personalidad juridica, profesionales colegiados, representantes de obligados y empleados publicos en ciertos supuestos. Esto exige canales digitales robustos y asistencia a quienes no estan obligados o necesitan apoyo. La arquitectura debe contemplar servicios de identificacion, firma, registro, consulta de estado, notificacion y acceso a expediente, sin romper la unidad procedimental.

El articulo 28 expresa una idea central para la integracion: el ciudadano no debe cargar con documentos que ya obran en poder de las Administraciones o que estas pueden consultar, salvo oposicion o limites aplicables. Esa regla empuja a usar plataformas de intermediacion de datos, servicios de consulta, trazabilidad de consentimiento u oposicion y controles de finalidad. Desde el punto de vista de APIs, no basta con exponer datos; deben existir base juridica, minimizacion, control de acceso, auditoria y capacidad de acreditar que la consulta se produjo dentro de un procedimiento concreto.

Los articulos 41 a 43, sobre notificaciones, tambien condicionan el diseno. La notificacion electronica exige constancia de puesta a disposicion, acceso, fechas, contenido e identidad del emisor y destinatario. Por tanto, sedes, direccion electronica habilitada, avisos, repositorios y servicios de firma o sello deben coordinarse con garantias probatorias.

Fuente editorial principal: Ley 39/2015, de 1 de octubre, del Procedimiento Administrativo Comun de las Administraciones Publicas; Jefatura del Estado; Boletin Oficial del Estado.

## Ley 40/2015: funcionamiento electronico, cooperacion y reutilizacion

La Ley 40/2015 aporta la logica interna del sector publico y transforma la interoperabilidad en principio de funcionamiento ordinario. Las Administraciones deben relacionarse entre si y con sus organos y entidades por medios electronicos, asegurando interoperabilidad, seguridad y proteccion de datos. Esa exigencia justifica arquitecturas de integracion, servicios compartidos, directorios comunes, intercambio seguro de datos y mecanismos de cooperacion.

Los articulos 38 a 46 regulan sedes electronicas, portales, sistemas de identificacion de las Administraciones, actuacion administrativa automatizada, firma, intercambio electronico de datos en entornos cerrados y archivo electronico. Para microservicios y automatizacion, destaca la actuacion administrativa automatizada: si un acto se produce integramente por medios electronicos sin intervencion directa de empleado publico, deben estar definidos los organos responsables de especificacion, programacion, mantenimiento, supervision, control de calidad y, en su caso, auditoria del sistema y del codigo fuente. Esta regla conecta con la gobernanza del software publico: versionado, pruebas, responsabilidad funcional y control de cambios.

La Ley 40/2015 es igualmente relevante por sus articulos 155 a 158. Las transmisiones de datos entre Administraciones, el Esquema Nacional de Interoperabilidad, el Esquema Nacional de Seguridad, la reutilizacion de sistemas y aplicaciones y la transferencia de tecnologia son el soporte juridico de muchas plataformas comunes. La ley orienta hacia componentes compartidos, servicios horizontales y APIs comunes antes que desarrollos redundantes.

Fuente editorial principal: Ley 40/2015, de 1 de octubre, de Regimen Juridico del Sector Publico; Jefatura del Estado; Boletin Oficial del Estado.

## Real Decreto 203/2021: reglamento de actuacion electronica

El Real Decreto 203/2021 desarrolla las Leyes 39/2015 y 40/2015 en lo referido a la actuacion y funcionamiento electronico del sector publico. Es una norma puente entre el mandato juridico y la operativa tecnica. Ordena portales, Punto de Acceso General electronico, sedes electronicas, Carpeta Ciudadana, registros electronicos, identificacion, firma, representacion, notificaciones, expediente, archivo, interoperabilidad, transmisiones de datos, plataformas de intermediacion y reutilizacion de aplicaciones.

Para una arquitectura distribuida, define puntos de integracion institucionales. El Punto de Acceso General electronico facilita el acceso a servicios, tramites e informacion. La Carpeta Ciudadana actua como area personalizada para seguimiento de tramitacion, comunicaciones, notificaciones y datos en poder del sector publico estatal. Las sedes electronicas concentran actuaciones que requieren identificacion de la Administracion y, en su caso, de la persona interesada. Los registros electronicos deben ser interoperables y conectarse con el Sistema de Interconexion de Registros. Las transmisiones de datos y las plataformas de intermediacion reducen aportaciones documentales innecesarias.

El reglamento tambien trata la adhesion a plataformas, registros o servicios electronicos de la Administracion General del Estado. Esto explica los servicios comunes: una entidad publica puede integrarse en servicios horizontales en vez de construir soluciones propias. La integracion no elimina la responsabilidad del organismo sobre permisos, finalidad, evidencias, cumplimiento del ENS y del ENI, y calidad del servicio.

Fuente editorial principal: Real Decreto 203/2021, de 30 de marzo, por el que se aprueba el Reglamento de actuacion y funcionamiento del sector publico por medios electronicos; Gobierno de España; Boletin Oficial del Estado.

## Esquema Nacional de Interoperabilidad

El Esquema Nacional de Interoperabilidad, regulado por el Real Decreto 4/2010 y desarrollado mediante normas tecnicas de interoperabilidad, convierte la cooperacion digital en criterios concretos. Su finalidad es crear condiciones para que datos, documentos, expedientes, aplicaciones y servicios sean interoperables en dimensiones organizativa, semantica y tecnica. En un tema sobre microservicios y APIs, el ENI permite explicar que una API publica no es interoperable solo por usar HTTP o JSON: tambien necesita significado comun de datos, metadatos, formatos normalizados, reglas de conservacion, politicas de firma, directorios y acuerdos de integracion.

Sus principios especificos son la interoperabilidad como cualidad integral, el caracter multidimensional de la interoperabilidad y el enfoque de soluciones multilaterales. Esto encaja con arquitecturas orientadas a servicios: el objetivo no es conectar dos aplicaciones puntuales, sino crear capacidades reutilizables por multiples administraciones y procedimientos. El ENI atiende a estandares, infraestructuras y servicios comunes, firma y certificados, documento y expediente electronico, copias autenticas, gestion documental, catalogos de estandares y modelos de datos.

El ENI resulta especialmente util para justificar la separacion entre componentes de negocio y servicios transversales. Directorios como DIR3, inventarios como SIA, modelos semanticos comunes, registros interconectados o plataformas de intermediacion no son accesorios: son instrumentos para que una arquitectura distribuida mantenga coherencia institucional. Sin esa capa comun, cada microservicio podria funcionar tecnicamente, pero el conjunto seria opaco, dificil de auditar y poco reutilizable.

Fuente editorial principal: Real Decreto 4/2010, de 8 de enero, por el que se regula el Esquema Nacional de Interoperabilidad en el ambito de la Administracion Electronica; Gobierno de España; Boletin Oficial del Estado.

## Esquema Nacional de Seguridad

El Esquema Nacional de Seguridad, vigente en su regulacion por el Real Decreto 311/2022, establece la politica de seguridad en la utilizacion de medios electronicos. Es obligatorio para el sector publico y para los sistemas de entidades privadas que presten soluciones o servicios a aquel en los terminos aplicables. Su lectura es imprescindible para arquitecturas distribuidas porque estas incrementan superficies de exposicion: APIs, integraciones, proveedores, redes, identidad federada, servicios en la nube y cadenas de suministro.

El ENS se basa en principios como seguridad integral, gestion de la seguridad basada en riesgos, prevencion, deteccion, respuesta y conservacion, lineas de defensa, vigilancia continua, reevaluacion periodica y diferenciacion de responsabilidades. Tambien fija requisitos minimos: organizacion e implantacion del proceso de seguridad, analisis y gestion de riesgos, control de accesos, minimo privilegio, proteccion de instalaciones, adquisicion segura, gestion de incidentes, continuidad, registro de actividad, proteccion de informacion almacenada y en transito, interconexion de sistemas, servicios en la nube y cadena de suministro.

En terminos practicos, el ENS obliga a clasificar sistemas, categorizar medidas y documentar riesgos. Una API que expone datos administrativos debe autenticar, autorizar, registrar, cifrar cuando proceda, limitar privilegios, gestionar secretos, controlar cambios, monitorizar actividad y prever respuesta a incidentes. Un microservicio forma parte de un sistema categorizado, con responsables, medidas y evidencias de cumplimiento. El ENS es el puente entre seguridad por diseño y seguridad administrativa verificable.

Fuente editorial principal: Real Decreto 311/2022, de 3 de mayo, por el que se regula el Esquema Nacional de Seguridad; Gobierno de España; Boletin Oficial del Estado; Centro Criptologico Nacional como referencia tecnica especializada.

## eIDAS y eIDAS2: identidad y servicios de confianza europeos

El Reglamento eIDAS, Reglamento (UE) 910/2014 del Parlamento Europeo y del Consejo, regula la identificacion electronica y los servicios de confianza para transacciones electronicas en el mercado interior. Para la Administracion publica, su funcion es asegurar que determinados medios de identificacion electronica y servicios de confianza puedan operar con validez y reconocimiento transfronterizo. Afecta a firmas electronicas, sellos electronicos, sellos de tiempo, entrega electronica certificada, autenticacion de sitios web y certificados cualificados.

La reforma conocida como eIDAS2, aprobada mediante el Reglamento (UE) 2024/1183, introduce el Marco Europeo de Identidad Digital y la cartera europea de identidad digital. Su relevancia para APIs e integracion es clara: la identidad deja de ser solo un formulario de login nacional y pasa a integrarse con atributos, credenciales y servicios de confianza europeos. Las Administraciones deberan aceptar medios de identificacion reconocidos, validar atributos cuando proceda y mantener niveles de seguridad adecuados.

En un diseño distribuido, eIDAS aconseja separar autenticacion, firma, sellado, validacion de certificados y auditoria en servicios transversales. Tambien evita confundir autenticacion con autorizacion: saber quien es una persona o entidad no basta para decidir si puede consultar un expediente, actuar como representante o acceder a un dato. Esa decision exige reglas de procedimiento, representacion, finalidad, proteccion de datos y trazabilidad.

Fuente editorial principal: Reglamento (UE) 910/2014 y Reglamento (UE) 2024/1183; Parlamento Europeo y Consejo de la Union Europea; Diario Oficial de la Union Europea; Comision Europea como fuente explicativa del marco de identidad digital.

## NIS2: ciberseguridad y continuidad de servicios esenciales

La Directiva (UE) 2022/2555, conocida como NIS2 o Directiva SRI 2, eleva el nivel comun de ciberseguridad en la Union. Aunque su transposicion nacional requiere norma interna y en España consta en tramitacion mediante el marco de coordinacion y gobernanza de la ciberseguridad, su contenido ya es una referencia obligada para arquitecturas publicas: gestion de riesgos, medidas tecnicas y organizativas, notificacion de incidentes, gestion de crisis, seguridad de la cadena de suministro y responsabilidad directiva.

Para la Administracion, NIS2 debe leerse junto al ENS. El ENS da el marco nacional de seguridad para sistemas publicos; NIS2 incorpora una logica europea de resiliencia, supervision y coordinacion para sectores criticos, entidades esenciales e importantes y determinados servicios digitales. En arquitectura distribuida, esto refuerza practicas como segmentacion, inventario de activos, gestion de vulnerabilidades, continuidad, copias, observabilidad, registro, proteccion de integraciones, evaluacion de proveedores y capacidad de respuesta coordinada.

Fuente editorial principal: Directiva (UE) 2022/2555 del Parlamento Europeo y del Consejo; Diario Oficial de la Union Europea; Departamento de Seguridad Nacional y Centro Criptologico Nacional como fuentes institucionales españolas sobre su aplicacion y transposicion.

## Reglamento Europa Interoperable

El Reglamento (UE) 2024/903, conocido como Reglamento Europa Interoperable, establece medidas para un alto nivel de interoperabilidad del sector publico en la Union. Su entrada en vigor en 2024 marca un cambio importante: la interoperabilidad transfronteriza deja de depender solo de recomendaciones y pasa a tener una gobernanza europea mas estructurada. Introduce evaluaciones de interoperabilidad para cambios en sistemas relacionados con servicios publicos transfronterizos, un Consejo de Europa Interoperable, un portal europeo de soluciones reutilizables, espacios de pruebas regulatorios y cooperacion GovTech.

Para el tema, permite conectar el ENI español con el marco europeo. Las APIs y microservicios de una Administracion no deben pensarse solo para su organismo titular, sino para encajar en servicios publicos compuestos, nacionales y europeos. Esto exige contratos estables, semantica clara, documentacion, gestion de versiones, criterios de accesibilidad, seguridad, proteccion de datos y reutilizacion. La interoperabilidad europea no elimina las competencias nacionales, pero obliga a diseñar pensando en cooperacion, escalabilidad institucional y ausencia de barreras tecnicas injustificadas.

Fuente editorial principal: Reglamento (UE) 2024/903 del Parlamento Europeo y del Consejo; Diario Oficial de la Union Europea; Comision Europea, Direccion General de Servicios Digitales.

## Reutilizacion, datos abiertos y servicios comunes

La reutilizacion tiene dos vertientes. La primera es la reutilizacion de informacion del sector publico, regulada en España por la Ley 37/2007 y alineada con la Directiva (UE) 2019/1024 sobre datos abiertos y reutilizacion de la informacion del sector publico. Esta normativa fomenta licencias abiertas, formatos legibles por maquina, disponibilidad de documentos reutilizables, conjuntos de datos de alto valor y reduccion de restricciones. En arquitectura, esto se traduce en APIs de datos, catalogos, metadatos, calidad, anonimización o disociacion cuando haya datos personales, y condiciones claras de uso.

La segunda es la reutilizacion de sistemas, aplicaciones y servicios. La Ley 40/2015, el ENI y el Real Decreto 203/2021 respaldan directorios de aplicaciones, transferencia tecnologica y reutilizacion en modo producto o servicio. Esta logica favorece servicios comunes como Cl@ve para identificacion y firma, la suite @firma para validacion y firma electronica, la Plataforma de Intermediacion de Datos, el Sistema de Interconexion de Registros, DIR3 como directorio comun, SIA como inventario de procedimientos, Notific@ y la Direccion Electronica Habilitada unica para notificaciones, Carpeta Ciudadana, Punto de Acceso General electronico, FACe, INSIDE y la Red SARA.

Estos servicios comunes muestran la idea central del tema: la Administracion moderna no debe duplicar capacidades horizontales, sino integrarlas mediante contratos claros, seguridad comun, trazabilidad y gobierno de datos. Una arquitectura de microservicios publica sera correcta si mejora modularidad y evolucion, pero sera plenamente administrativa solo si respeta procedimiento, interoperabilidad, seguridad, identidad, archivo, reutilizacion y servicio al ciudadano.

Fuentes editoriales principales: Ley 37/2007, de reutilizacion de la informacion del sector publico; Jefatura del Estado; Boletin Oficial del Estado. Directiva (UE) 2019/1024; Parlamento Europeo y Consejo de la Union Europea; Diario Oficial de la Union Europea. Catalogo de Servicios de Administracion Digital, Portal de Administracion Electronica y Secretaria General de Administracion Digital.
