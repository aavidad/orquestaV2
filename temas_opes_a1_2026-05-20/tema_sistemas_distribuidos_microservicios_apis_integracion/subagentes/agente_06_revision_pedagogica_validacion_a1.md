# Agente 06 - Revision pedagogica, ensamblado y validacion A1

## Alcance

Rol cubierto: revision pedagogica, apoyo al ensamblado, validacion de palabras y checklist A1 para el tema "Arquitecturas distribuidas, microservicios, APIs e integracion de sistemas en la Administracion publica".

Write-set usado: este fichero dentro de `subagentes/`. No se ha editado `tema_a1.md`, porque el ensamblado final corresponde al agente principal.

Estado observado al inicio de esta revision: no existian aun `tema_a1.md`, `tema_a1.html`, `fuentes.md`, `checklist_a1.md`, `assets/` ni `banco_preguntas_i18n/es/` dentro del tema. Por tanto, las validaciones finales de palabras y politica quedan bloqueadas hasta que exista el tema ensamblado.

## Criterio de no listo

El tema no debe marcarse como listo si ocurre cualquiera de estas condiciones:

| Control | Umbral o regla | Consecuencia |
| --- | --- | --- |
| Palabras del `tema_a1.md` | Menos de 20.250 palabras | Crear `INFORME_BLOQUEO.md`; no declarar listo |
| Palabras del `tema_a1.md` | Mas de 22.500 palabras | Recortar o mover material auxiliar fuera del tema |
| Evidencia minima | Falta algun artefacto raiz requerido | Bloqueo verificable |
| Texto visible | Menciona rutas, prompts, agentes, arquitectura de ejecucion o ids tecnicos | Corregir antes de publicacion |
| Fuentes | No prioriza BOE, UE y portales oficiales espanoles | Rehacer fuentes y trazabilidad |
| Test | Notas de test mezcladas con teoria | Separar en bloques identificables y ocultables en HTML |
| HTML | Referencia assets inexistentes o no parsea | Bloqueo de publicacion |

## Presupuesto recomendado de palabras

Objetivo de ensamblado: unas 21.600 palabras. Deja margen para ajustes sin caer por debajo de 20.250 ni superar 22.500.

| Bloque final | Palabras objetivo | Funcion pedagogica |
| --- | ---: | --- |
| Indice, orientacion de examen y ruta de estudio | 900 | Fijar alcance, nivel y forma de estudio |
| Mapa inicial del tema | 800 | Dar una estructura mental antes del detalle |
| Definiciones de autoridad y vocabulario base | 1.800 | Evitar ambiguedades tecnicas |
| Fundamentos de arquitecturas distribuidas | 2.300 | Explicar acoplamiento, latencia, consistencia y fallos |
| Microservicios y arquitectura cloud-native | 2.500 | Distinguir patron, organizacion y operacion |
| APIs: REST, HTTP, contratos y versionado | 2.500 | Convertir la integracion en contrato gobernable |
| Integracion de sistemas y patrones | 2.500 | Cubrir sincronismo, eventos, colas, ETL y legados |
| Marco publico: Leyes 39/2015 y 40/2015, RD 203/2021, ENI, ENS, UE | 2.700 | Conectar tecnologia con obligacion administrativa |
| Gobierno operativo: seguridad, observabilidad, resiliencia, DevSecOps | 2.200 | Preparar respuestas de madurez A1 |
| Ejemplos trabajados y tablas comparativas | 1.300 | Reforzar comprension sin sustituir teoria |
| Supuestos practicos guiados | 1.100 | Entrenar aplicacion a casos de administracion publica |
| Notas de test, errores frecuentes, repaso final, plan de visuales y fuentes | 1.000 | Consolidar y preparar examen |
| Total recomendado | 21.600 | Dentro del rango A1 |

## Secuencia de ensamblado recomendada

1. Abrir con la idea central: la arquitectura distribuida en administraciones publicas no es una moda tecnica, sino una forma de sostener servicios digitales interoperables, seguros, trazables y evolutivos.
2. Definir antes de desarrollar: sistema distribuido, servicio, microservicio, API, contrato, interoperabilidad tecnica, interoperabilidad semantica, acoplamiento, cohesion, resiliencia, disponibilidad, consistencia, idempotencia, autenticacion, autorizacion, trazabilidad y observabilidad.
3. Explicar primero los fundamentos neutrales y despues aterrizar en administracion publica. El lector debe entender por que una decision de arquitectura afecta a derecho de la ciudadania, continuidad del servicio, seguridad, proteccion de datos y reutilizacion.
4. Introducir microservicios con cautela. Deben presentarse como una opcion con costes de operacion, gobierno y observabilidad, no como solucion universal.
5. Separar API de microservicio. Una administracion puede exponer APIs bien gobernadas sobre sistemas modulares, monolitos modernizados, plataformas comunes o integraciones de legado.
6. Distinguir integracion sincronica y asincronica con ejemplos: consulta de datos de otra administracion, alta de expediente, notificacion, intercambio documental, eventos de cambio de estado y sincronizacion de catalogos maestros.
7. Conectar cada bloque tecnico con un marco publico: ENI para interoperabilidad, ENS para seguridad, RD 203/2021 para funcionamiento electronico, Ley 39/2015 y Ley 40/2015 para derechos, relaciones electronicas, cooperacion y reutilizacion, RGPD para datos personales, Reglamento de Europa Interoperable para dimension UE.
8. Reservar el banco completo de preguntas para `banco_preguntas_i18n/es/`; dentro del tema solo incluir muestra progresiva.

## Puntos de revision pedagogica

| Riesgo | Senal de alerta | Correccion recomendada |
| --- | --- | --- |
| Tema enciclopedico | Listas largas de tecnologias sin explicacion | Convertir cada lista en criterio de decision con ejemplo |
| Exceso de jerga | Uso de terminos antes de definirlos | Insertar definicion breve antes del primer uso |
| Microservicios idealizados | Se ocultan costes de red, datos, despliegue y observabilidad | Anadir bloque de trade-offs y antipatrones |
| Confusion API-servicio | Se presenta toda API como microservicio | Separar contrato externo, implementacion interna y gobierno |
| Teoria sin administracion publica | Ejemplos genericos de ecommerce o redes sociales | Usar expediente, sede, carpeta ciudadana, intermediacion, notificaciones, registros, DIR3, SIA |
| Test mezclado | Distractores dentro del desarrollo teorico | Mover trampas a notas de test o repaso |
| Tablas sustitutivas | La tabla contiene informacion no explicada antes | Explicar primero en parrafos y usar la tabla como sintesis |
| HTML no estudiable | Todo visible a la vez | Primera lectura activa por defecto y capas ocultables |

## Tablas que conviene incluir en el tema

| Tabla | Objetivo |
| --- | --- |
| Monolito modular, SOA, microservicios y arquitectura orientada a eventos | Evitar respuestas simplistas sobre modernizacion |
| API REST, API de eventos, intercambio documental, cola y ETL | Elegir patron segun latencia, consistencia y trazabilidad |
| API gateway, BFF, service mesh, ESB y bus de eventos | Diferenciar piezas de arquitectura y no mezclarlas |
| Interoperabilidad tecnica, semantica, organizativa y juridica | Alinear con ENI y Europa Interoperable |
| Principios de diseno de APIs publicas | Contrato, versionado, seguridad, trazabilidad, accesibilidad y documentacion |
| Riesgos de microservicios | Latencia, fallos parciales, datos distribuidos, observabilidad, despliegue y gobierno |
| Controles de seguridad | Identidad, autorizacion, cifrado, registro, monitorizacion, gestion de secretos y auditoria |
| Errores frecuentes de examen | Confusiones entre API, servicio, dato, interfaz, integracion y interoperabilidad |

## Ejemplos recomendados

Usar ejemplos publicos y neutros, sin rutas internas ni datos reales:

| Ejemplo | Conceptos que refuerza |
| --- | --- |
| Consulta de datos que la ciudadania no debe aportar de nuevo | Interoperabilidad, intermediacion, base juridica, trazabilidad |
| Registro electronico que envia asiento a un organo destino | Integracion entre sistemas, metadatos, disponibilidad |
| Notificacion electronica con acuse y puesta a disposicion | API, evento, evidencia, ciclo de vida |
| Carpeta ciudadana que agrega expedientes de varios sistemas | API composition, experiencia de usuario, desacoplamiento |
| Modernizacion de una aplicacion legacy de subvenciones | Estrangulamiento progresivo, APIs, colas, riesgos de datos |
| Catalogo maestro de unidades y oficinas | Dato de referencia, semantica comun, sincronizacion |

## Supuestos practicos que conviene desarrollar

### Supuesto 1: consulta interoperable de datos

Situacion: un procedimiento administrativo necesita verificar identidad, residencia o titulacion sin pedir documentos ya disponibles en otra administracion.

Pistas que debe reconocer el opositor: principio de no aportacion repetida cuando proceda, base juridica, interoperabilidad, consulta trazable, minimizacion de datos, registro de evidencias, autorizacion y control de acceso.

Resolucion esperada: definir sistema consumidor, sistema proveedor, contrato de datos, control de finalidad, autenticacion de sistema, autorizacion, registro de auditoria, gestion de errores, reintentos y mensaje comprensible al gestor.

### Supuesto 2: modernizacion con APIs

Situacion: una consejeria quiere exponer servicios de una aplicacion de expedientes sin sustituirla de golpe.

Pistas: no confundir API con reescritura completa, aplicar patron de estrangulamiento, proteger datos personales, versionar contratos, publicar documentacion tecnica, probar regresion y mantener trazabilidad.

Resolucion esperada: fachada API inicial, anticorruption layer si el modelo legacy no coincide, pruebas contractuales, observabilidad, migracion por capacidades y retirada progresiva de dependencias obsoletas.

### Supuesto 3: microservicios para alta demanda

Situacion: un servicio digital tiene picos de carga previsibles y equipos diferenciados por capacidades funcionales.

Pistas: microservicios solo si hay independencia real de dominio y capacidad operativa. Deben aparecer CI/CD, observabilidad, gestion de configuracion, seguridad servicio a servicio, gateway, tolerancia a fallos y gobierno de datos.

Resolucion esperada: partir por capacidades de negocio, evitar base de datos compartida sin control, definir eventos y contratos, usar colas cuando sea necesario, establecer SLO, logs correlados, metricas y trazas.

## Plan de visuales

Todos los visuales deben ser didacticos, locales, responsivos y legibles en movil.

| Visual | Formato recomendado | Uso |
| --- | --- | --- |
| Mapa del tema | SVG local | Mostrar relacion entre arquitectura, API, integracion, seguridad e interoperabilidad |
| Flujo de consulta interoperable | SVG local | Explicar consumidor, proveedor, autorizacion, auditoria y respuesta |
| Diagrama de gateway y servicios | SVG local | Diferenciar cliente, gateway, servicios, datos y observabilidad |
| Comparativa de patrones de integracion | HTML/CSS o tabla con scroll | Facilitar decision en supuestos |
| Ciclo de vida de una API publica | SVG local | Diseno, contrato, seguridad, versionado, pruebas, operacion y retirada |

## Fuentes oficiales verificadas para el ensamblado

No incluir URLs reales visibles en el `tema_a1.md` ni en el `tema_a1.html`. La cita editorial puede referir el titulo, organo y fecha. Para trazabilidad interna, usar claves de fuente en `fuentes.md`.

| Clave sugerida | Fuente | Uso en el tema |
| --- | --- | --- |
| BOE-L39-2015 | Ley 39/2015, de Procedimiento Administrativo Comun de las Administraciones Publicas | Derechos y funcionamiento del procedimiento administrativo electronico |
| BOE-L40-2015 | Ley 40/2015, de Regimen Juridico del Sector Publico | Regimen juridico, cooperacion, medios electronicos y reutilizacion |
| BOE-RD203-2021 | Real Decreto 203/2021, Reglamento de actuacion y funcionamiento del sector publico por medios electronicos | Funcionamiento electronico, sedes, registros, colaboracion y relaciones electronicas |
| BOE-RD4-2010 | Real Decreto 4/2010, Esquema Nacional de Interoperabilidad | Interoperabilidad tecnica, semantica y organizativa |
| BOE-RD311-2022 | Real Decreto 311/2022, Esquema Nacional de Seguridad | Seguridad de sistemas, servicios y administracion electronica |
| UE-REG-2024-903 | Reglamento (UE) 2024/903, Europa Interoperable | Interoperabilidad transfronteriza y soluciones comunes |
| UE-REG-2024-1183 | Reglamento (UE) 2024/1183, identidad digital europea | Identidad digital y servicios de confianza |
| UE-DIR-2022-2555 | Directiva (UE) 2022/2555, NIS2 | Ciberseguridad y gestion de riesgos |
| UE-REG-2016-679 | Reglamento (UE) 2016/679, RGPD | Proteccion de datos desde el diseno y minimizacion |
| PAE-ENI-NTI | Portal de Administracion Electronica, ENI y Normas Tecnicas de Interoperabilidad | Ejemplos y vocabulario de interoperabilidad espanola |
| PAE-PID | Plataforma de Intermediacion de Datos | Ejemplo de integracion e interoperabilidad entre administraciones |
| PAE-DIR3 | Directorio Comun DIR3 | Dato maestro comun para unidades y oficinas |
| PAE-SIA | Sistema de Informacion Administrativa | Inventario administrativo y referencia para procedimientos |
| NIST-SP800-204 | NIST SP 800-204, Security Strategies for Microservices-based Application Systems | Seguridad de microservicios |
| NIST-SP800-204A | NIST SP 800-204A, Service Mesh | Servicio a servicio, malla, autenticacion y observabilidad |
| IETF-RFC9110 | RFC 9110, HTTP Semantics | Semantica HTTP, recursos, metodos y representaciones |
| OAS-3-1 | OpenAPI Specification 3.1 | Contratos de API y documentacion tecnica |

## Checklist editorial para el agente principal

| Item | Validacion |
| --- | --- |
| Indice | Existe y coincide con los encabezados reales |
| Orientacion de examen | Explica como estudiar y que suele preguntarse |
| Mapa inicial | Incluye mapa conceptual o ruta visual |
| Definiciones | Cada termino tecnico clave aparece definido antes de usarse en profundidad |
| Desarrollo teorico | Parrafos explicativos suficientes; no solo esquemas |
| Tablas | Refuerzan teoria ya explicada |
| Ejemplos | Son de administracion publica y no de empresas genericas salvo apoyo tecnico |
| Supuestos | Incluyen situacion, pistas, preguntas, resolucion y errores |
| Notas de test | Separadas y reconocibles |
| Errores frecuentes | Incluyen confusion y forma de reconocerla |
| Repaso final | Resume 5-7 ideas, diferencias y preguntas de recuperacion |
| Fuentes | Priorizan BOE, UE y portales oficiales |
| HTML | Primera lectura activa, barra lateral plegable, modo tutor y notas ocultables |
| Banco externo | Preguntas completas fuera del tema, en `banco_preguntas_i18n/es/` |
| Texto visible | Sin ids tecnicos, rutas internas, prompts ni menciones de ejecucion |

## Validacion automatizable cuando exista `tema_a1.md`

Comprobacion de palabras:

```bash
tema="external/opes/a1/tema_sistemas_distribuidos_microservicios_apis_integracion/tema_a1.md"
words="$(wc -w < "$tema")"
printf 'palabras=%s\n' "$words"
test "$words" -ge 20250
test "$words" -le 22500
```

Comprobacion editorial minima:

```bash
tema="external/opes/a1/tema_sistemas_distribuidos_microservicios_apis_integracion/tema_a1.md"
rg -n "Indice|Orientacion|Mapa|Definicion|Supuesto|Nota de test|Errores frecuentes|Repaso|Fuentes" "$tema"
! rg -n "https?://|wave_ref|agent_ref|prompt|Codex|Orquesta|subagente|ruta interna" "$tema"
```

Comprobacion de HTML y assets:

```bash
html="external/opes/a1/tema_sistemas_distribuidos_microservicios_apis_integracion/tema_a1.html"
test -f "$html"
rg -n "sidebar|modo|tutor|nota-test|primera-lectura|banco_preguntas_i18n" "$html"
```

Estas comprobaciones no sustituyen la revision humana. Sirven para detectar incumplimientos obvios antes de cerrar.

## Resultado de validacion en este pase

| Test requerido | Resultado | Evidencia |
| --- | --- | --- |
| validar-palabras-a1-20250-22500 | Bloqueado | `tema_a1.md` no existe aun |
| validar-politica-editorial-opes-a1 | Bloqueado | No existe material final que revisar |

## Informe de bloqueo que debe emitirse si el tema queda incompleto

Si el agente principal no alcanza el minimo A1, crear `INFORME_BLOQUEO.md` con esta informacion:

| Campo | Contenido esperado |
| --- | --- |
| Palabras alcanzadas | Numero exacto de palabras de `tema_a1.md` |
| Secciones completas | Lista de bloques finalizados y revisados |
| Secciones pendientes | Lista de bloques ausentes o demasiado breves |
| Brecha hasta minimo | `20250 - palabras_alcanzadas` si es positiva |
| Evidencias faltantes | Artefactos raiz no creados o no sincronizados |
| Pasos para cerrar | Acciones concretas, en orden, con rutas |
| Estado | No listo para publicacion ni subida controlada |

## Prioridad de cierre

1. Crear `tema_a1.md` con rango A1 y estructura completa.
2. Crear `fuentes.md` con claves editoriales y fuentes oficiales priorizadas.
3. Crear visuales locales en `assets/` y referenciarlos desde Markdown/HTML.
4. Crear banco externo en `banco_preguntas_i18n/es/`.
5. Generar `tema_a1.html` sincronizado con Markdown.
6. Ejecutar validacion de palabras y politica editorial.
7. Completar `checklist_a1.md` e `INFORME_EJECUCION.md`.
