# 08. Superficies, internacionalización y contratos de aplicación

> **Responsabilidad:** definir cómo HTTP, MCP, línea de
> órdenes, biblioteca cliente y web exponen los mismos casos de uso; cómo se
> conservan identidad, autorización, revisión esperada, idempotencia y errores;
> y cómo toda aplicación creada o modificada debe poder explicarse, operarse,
> traducirse y probarse.
>
> **Alcance:** registro único de órdenes, despacho común, autenticación,
> autorización, auditoría, errores máquina, BCP-47, catálogos, formatos,
> accesibilidad, aplicación web instalable y la regla transversal `APP-13` para
> documentación y cabeceras en aplicaciones Go y no Go.
>
> **No acredita:** este texto no demuestra una composición productiva. El
> servidor MCP real de `UI-02`, el registro de órdenes de V20 y la
> internacionalización total de `UI-18` están acreditados en su alcance. La API
> HTTP total (`UI-01`), la línea de órdenes fina total (`UI-03`), la web y sus
> superficies relacionadas siguen declaradas. `APP-13` también sigue
> declarado: su estado honesto es **NO-GO**, es decir, todavía no puede
> utilizarse para acreditar universalmente cualquier aplicación generada o
> modificada.

## Al terminar este capítulo, el lector sabrá…

- exponer un caso de uso por HTTP, MCP, línea de órdenes, biblioteca y web sin
  duplicar reglas;
- conservar autenticación, autorización, revisión esperada e idempotencia entre
  transportes;
- diseñar internacionalización BCP-47 y accesibilidad verificables;
- aplicar las ocho secciones y cabeceras exigidas por `APP-13`;
- distinguir las superficies acreditadas de las que continúan declaradas.

La autoridad viva se consulta en la base versionada
`HEAD:product/roadmap.json`, mediante `git show HEAD:product/roadmap.json`, en
[`product/capabilities.json`](../../../product/capabilities.json) y
[`product/evidence/`](../../../product/evidence/). Los contratos de V20 y V21
explican el diseño y la secuencia histórica, pero su narración no sustituye la
evidencia vigente.

## Índice del capítulo

- [8.1 Una aplicación, varios transportes](#81-una-aplicación-varios-transportes)
- [8.2 Estado real de las superficies](#82-estado-real-de-las-superficies)
- [8.3 Registro único de órdenes](#83-registro-único-de-órdenes)
- [8.4 Recorrido completo de una petición](#84-recorrido-completo-de-una-petición)
- [8.5 Autenticación y autorización](#85-autenticación-y-autorización)
- [8.6 Revisión esperada e idempotencia](#86-revisión-esperada-e-idempotencia)
- [8.7 Auditoría sin segundo ciclo de vida](#87-auditoría-sin-segundo-ciclo-de-vida)
- [8.8 Errores máquina y mensajes humanos](#88-errores-máquina-y-mensajes-humanos)
- [8.9 Internacionalización como contrato](#89-internacionalización-como-contrato)
- [8.10 Catálogos y parámetros](#810-catálogos-y-parámetros)
- [8.11 Plurales, números, moneda, fecha y zona horaria](#811-plurales-números-moneda-fecha-y-zona-horaria)
- [8.12 Activar una superficie nueva](#812-activar-una-superficie-nueva)
- [8.13 HTTP](#813-http)
- [8.14 MCP](#814-mcp)
- [8.15 Línea de órdenes y biblioteca cliente](#815-línea-de-órdenes-y-biblioteca-cliente)
- [8.16 Web instalable y accesible](#816-web-instalable-y-accesible)
- [8.17 Regla transversal APP-13](#817-regla-transversal-app-13)
- [8.18 Identificar la aplicación objetivo](#818-identificar-la-aplicación-objetivo)
- [8.19 Las ocho secciones obligatorias](#819-las-ocho-secciones-obligatorias)
- [8.20 Cabeceras de responsabilidad](#820-cabeceras-de-responsabilidad)
- [8.21 Go, otros lenguajes y archivos no comentables](#821-go-otros-lenguajes-y-archivos-no-comentables)
- [8.22 Planificación transversal de APP-13](#822-planificación-transversal-de-app-13)
- [8.23 Comprobador neutral](#823-comprobador-neutral)
- [8.24 Internacionalización dentro de APP-13](#824-internacionalización-dentro-de-app-13)
- [8.25 Pruebas de APP-13](#825-pruebas-de-app-13)
- [8.26 Por qué APP-13 está en NO-GO](#826-por-qué-app-13-está-en-no-go)
- [8.27 Secuencia completa para crear una aplicación](#827-secuencia-completa-para-crear-una-aplicación)
- [8.28 Diagnóstico de divergencia entre superficies](#828-diagnóstico-de-divergencia-entre-superficies)
- [8.29 Pruebas mínimas del registro y superficies](#829-pruebas-mínimas-del-registro-y-superficies)
- [8.30 Orden recomendado de construcción desde cero](#830-orden-recomendado-de-construcción-desde-cero)
- [8.31 Criterio de cierre](#831-criterio-de-cierre)

## 8.1 Una aplicación, varios transportes

Una superficie pública traduce una petición a un caso de uso; no decide el
negocio. La forma correcta es:

```text
HTTP ─────┐
MCP ──────┤
CLI ──────┼──► registro de órdenes ─► despachador ─► aplicación ─► dominio
SDK ──────┤                                  │
web ──────┘                                  └── puertos ─► adaptadores
```

`CLI` es la interfaz de línea de órdenes. `SDK` es una biblioteca cliente.
`MCP` es el protocolo de herramientas y recursos para clientes compatibles.
El [glosario técnico común](01_mision_limites_y_vocabulario.md#glosario-técnico-común)
define ciclo de vida, recibo y huella criptográfica. En este capítulo,
«esquema» designa el `schema`, «respaldo lingüístico» el `fallback`,
«configuración regional» el `locale` y «carga útil» el `payload`. Todos los
transportes deben resolver la misma definición de orden, pasar por la misma
autorización de aplicación y obtener el mismo código máquina.

La alternativa incorrecta es crear cinco implementaciones:

```text
HTTP escribe Goal por su cuenta
MCP llama directamente al almacén
CLI tiene reglas privadas
web reproduce el ciclo de vida
SDK inventa sus propios errores
```

Ese diseño genera cinco autoridades que divergen con el tiempo.

## 8.2 Estado real de las superficies

| Capacidad | Alcance resumido | Estado vivo al redactar |
|---|---|---|
| `UI-02` | servidor MCP real | acreditado |
| `UI-18` | internacionalización total en el alcance activado | acreditado |
| `UI-01` | API HTTP tipada total | declarado |
| `UI-03` | línea de órdenes fina total | declarado |
| `UI-04` y siguientes de web | experiencia web y superficies asociadas | declarado |
| `UI-09` | escritorio nativo separado | declarado; decisión `reject` y alcance `excluded` |
| `APP-13` | documentación transversal de aplicaciones | declarado, NO-GO |

V20 acreditó la paridad de sus órdenes registradas en HTTP, MCP, línea de
órdenes y biblioteca cliente. Eso no autoriza a marcar como terminadas todas
las capacidades futuras de esos canales. La acreditación es por contrato y
revisión, no por parecido.

Las evidencias específicas son
[`v20_command_registry.json`](../../../product/evidence/v20_command_registry.json)
y [`v21_i18n.json`](../../../product/evidence/v21_i18n.json). Deben leerse con
el estado de la hoja de ruta y con las huellas criptográficas del candidato que
acreditan.

La web debe construirse como cliente del mismo registro y de los mismos casos
de uso. No hace falta otro servicio de negocio ni una aplicación de escritorio
nativa para obtener una experiencia instalable.

## 8.3 Registro único de órdenes

El registro vigente se encuentra en
[`internal/commands/registry.json`](../../../internal/commands/registry.json).
El contrato de diseño está en
[`analisis_y_contrato_v20_registro_comandos.md`](../analisis_y_contrato_v20_registro_comandos.md).

Una definición de orden contiene, de forma conceptual:

```text
id estable
versión
clase: consulta o mutación
manejador de aplicación
permiso
audiencia
si queda ligada a una ejecución
modo de repetición
esquema estricto de entrada
esquema estricto de salida
clave de descripción traducible
códigos de error permitidos
enlace HTTP: método + ruta
enlace MCP: herramienta
enlace CLI: ruta de órdenes
```

Los nombres exactos del registro son parte del esquema productivo. No deben
duplicarse en una tabla manual o en constantes de cada transporte.

### Qué genera el registro

El registro puede producir proyecciones para:

- tipos Go;
- tabla del despachador;
- rutas HTTP;
- definiciones de herramientas MCP;
- árbol de órdenes de terminal;
- biblioteca cliente;
- documentación pública;
- comprobaciones de paridad;
- fuentes tipadas de claves traducibles.

El código generado se mantiene separado de la política. Una plantilla puede
generar enlaces; no debe decidir autorización, cierre o reintentos.

### Prohibiciones

- una orden pública que no figure en el registro;
- un alias permanente sin plan de retirada;
- un transporte con esquema más permisivo;
- un error nuevo decidido solo por HTTP;
- un manejador que importe un almacén concreto;
- una ruta web que escriba estado sin pasar por aplicación;
- un identificador localizado;
- una descripción humana codificada en la plantilla.

## 8.4 Recorrido completo de una petición

```text
1. el transporte autentica una credencial
2. obtiene un principal verificable
3. resuelve la definición registrada
4. construye la envolvente de autoridad
5. valida esquema, versión, proyecto y huellas criptográficas
6. autorización de aplicación comprueba permiso y alcance
7. se registra la invocación auditable
8. el despachador llama al manejador común
9. el caso de uso aplica revisión esperada e idempotencia
10. la transacción persiste instantánea, evento y salida cuando proceda
11. se registra el resultado auditable
12. una tabla única traduce el resultado al transporte
13. el catálogo produce el texto humano solicitado
```

El despachador no es otro orquestador. Resuelve definiciones, une autoridad,
valida, audita y llama al caso de uso.

### Estructura conceptual

```text
envolvente de orden (`CommandEnvelope`)
  identificador de orden (`CommandID`)
  versión (`Version`)
  referencia de proyecto (`ProjectRef`)
  referencia del principal (`PrincipalRef`) <- procede de autenticación
  vínculo de ejecución opcional (`ExecutionBinding`) <- procede de enlace acreditado
  referencia de petición (`RequestRef`)
  clave de idempotencia (`IdempotencyKey`)
  revisión esperada opcional (`ExpectedRevision`)
  configuración regional (`Locale`)
  contenido (`Payload`)
```

Es una representación didáctica. El esquema definitivo de cada orden vive en
el registro. En particular, el `Payload` no debe incluir:

- identidad del principal;
- funciones o permisos;
- recibo de autorización;
- un proyecto alternativo;
- una ejecución autoafirmada;
- secretos de transporte.

## 8.5 Autenticación y autorización

Autenticar responde «quién presenta la petición». Autorizar responde «qué
puede hacer ese principal sobre este proyecto y sujeto». Son pasos distintos.

### Autenticación en el borde

Cada adaptador traduce su credencial a una identidad neutral:

- token local;
- sesión web;
- OIDC;
- LDAP/LDAPS cuando sea el adaptador instalado;
- credencial de herramienta MCP;
- identidad de servicio.

El núcleo no conoce marcas, cabeceras HTTP ni formatos del proveedor.

### Autorización en aplicación

Antes de cada orden o consulta, la aplicación comprueba:

- principal;
- permiso exigido por la definición;
- proyecto explícito;
- sujeto concreto;
- pertenencia o ámbito;
- ejecución ligada, si la orden la exige;
- presupuesto o aprobación, si hay efectos;
- restricciones adicionales de seguridad.

La interfaz no puede sustituir esa comprobación con un botón oculto. El cliente
web tampoco es una frontera de confianza.

### Aislamiento entre proyectos

Se deben probar dos clases de negativo:

1. un principal sin permiso no puede actuar;
2. un principal autorizado en A no puede leer ni inferir recursos de B.

Según el contrato, una consulta cruzada puede responder `not_found` para no
revelar existencia en lugar de `forbidden`. Esa traducción debe ser única y
deliberada.

## 8.6 Revisión esperada e idempotencia

Una mutación durable necesita dos defensas diferentes.

### Revisión esperada

`expected_revision` expresa sobre qué revisión leyó y decidió el cliente:

```text
estado actual revision=12
cliente A envía expected_revision=12 ──► acepta y produce revision=13
cliente B envía expected_revision=12 ──► conflicto
```

Evita que una escritura tardía pise una decisión posterior.

### Idempotencia

La clave de idempotencia expresa que dos intentos son la misma intención:

```text
misma orden + mismo sujeto + misma clave + mismo contenido
  -> mismo recibo o resultado

misma clave + contenido o sujeto distinto
  -> conflicto
```

No debe deduplicarse solo por cadena. El ámbito incluye orden, versión,
principal, proyecto, sujeto y huella criptográfica de entrada conforme al
contrato.

### Consultas y mutaciones

- una mutación repite el recibo persistido por aplicación;
- una consulta solo lee una revisión existente: no incrementa revisiones, no
  cambia el estado del producto, no genera eventos de dominio y no escribe
  resultados en el repositorio autoritativo;
- repetir una consulta vuelve a calcular su resultado sobre la revisión leída,
  salvo que una caché no autoritativa pueda responder con la misma semántica;
- un adaptador no guarda una caché autoritativa que cambie esta semántica;
- los reintentos por red no producen dos efectos externos.

Marcador contractual: `consultas_sin_escritura_de_producto`.

## 8.7 Auditoría sin segundo ciclo de vida

Una mutación puede producir su auditoría durable mediante la transacción y la
bandeja de salida definidas por el caso de uso. Una consulta, en cambio, no
escribe auditoría dentro del repositorio de producto: hacerlo convertiría una
lectura en mutación y podría bloquearla por conflictos de revisión.

Si una política exige auditar el acceso de lectura, el adaptador de borde usa
un puerto separado después de autenticar y autorizar:

```text
AuditAccess (auditar acceso)
  referencia
  orden y versión
  principal y proyecto
  sujeto y huellas criptográficas
  instante
  transporte
  clave de idempotencia
  resultado permitido o denegado
```

El puerto de auditoría de acceso declara su propio recibo, tiempo límite,
repetición y fallo. Una política obligatoria falla de forma explícita y cerrada
si no puede registrar el acceso; una política informativa puede conservar una
métrica no durable. En ambos casos, el fallo nunca cambia el resultado
autoritativo de la consulta ni su revisión.

Esta auditoría no cierra el `Goal`, no entra en el repositorio de estado y no
reconstruye el producto. Su almacén es una fuente de cumplimiento o seguridad,
nunca una autoridad que pueda contradecir a la instantánea autoritativa.

Los datos sensibles se redactan antes de persistir. «No mostrar en la web» no
es una política de secreto suficiente.

## 8.8 Errores máquina y mensajes humanos

Los códigos máquina deben ser estables y no traducidos:

```text
invalid_request
unauthenticated
forbidden
not_found
conflict
unavailable
internal
```

Su texto humano se obtiene mediante una clave de catálogo. Una respuesta
neutral puede contener:

```text
code
message_key
parameters
correlation_ref
retryable
details permitidos
```

No debe contener una frase española fija como autoridad, ni el texto bruto de
una excepción interna.

### Correspondencia uniforme

| Código | HTTP orientativo | CLI | MCP y biblioteca |
|---|---:|---|---|
| `invalid_request` | 400 | salida de uso inválido | error tipado equivalente |
| `unauthenticated` | 401 | autenticación requerida | error tipado equivalente |
| `forbidden` | 403 | permiso insuficiente | error tipado equivalente |
| `not_found` | 404 | recurso no encontrado | error tipado equivalente |
| `conflict` | 409 | conflicto de revisión o idempotencia | error tipado equivalente |
| `unavailable` | 503 | indisponibilidad recuperable | error tipado equivalente |
| `internal` | 500 | fallo interno con referencia | error tipado equivalente |

La tabla es una política de adaptación, no una segunda taxonomía. Si un
transporte necesita más detalle, lo deriva del mismo error neutral sin cambiar
su significado.

## 8.9 Internacionalización como contrato

El manifiesto vigente está en
[`internal/i18n/manifest.json`](../../../internal/i18n/manifest.json), y el
contrato histórico de V21 en
[`analisis_y_contrato_v21_i18n_total.md`](../analisis_y_contrato_v21_i18n_total.md).

Internacionalizar no es ejecutar una sustitución de cadenas. Requiere:

- una única propiedad de cada texto visible;
- catálogos por idioma;
- claves estables;
- selección BCP-47;
- respaldo lingüístico definido;
- plurales y formatos;
- paridad de parámetros;
- pruebas por superficie.

### Autoridad de texto

Todo texto humano visible debe proceder de una fuente tipada de catálogo:

- etiquetas y ayudas;
- mensajes de error;
- estados explicados;
- descripciones de órdenes;
- salida de terminal destinada a personas;
- herramientas y recursos MCP presentados;
- web;
- Asistente;
- notificaciones;
- plantillas de instrucciones controladas;
- documentación pública localizada.

Los códigos, referencias y huellas criptográficas no se traducen.

### Resolución BCP-47

BCP-47 es el estándar de etiquetas como `es`, `es-ES`, `en` o `en-GB`.

La política exigida es:

1. español es idioma por defecto;
2. español es el respaldo lingüístico final;
3. una etiqueta instalada exacta se usa directamente;
4. una variante válida puede caer a su idioma base;
5. una etiqueta válida no disponible cae a español;
6. una etiqueta inválida se rechaza;
7. no se corrige silenciosamente una entrada mal formada.

Ejemplo:

```text
es-ES -> es, si no hay catálogo es-ES
en-GB -> en, si no hay catálogo en-GB
fr-FR -> es, si francés no está instalado
"english" -> invalid_request, no es una etiqueta BCP-47 válida
```

### Qué permanece invariante

- `command_id` y versión;
- códigos de error;
- `message_key`;
- `GoalRef`, `WorkItemRef`, `ExecutionRef` y `ProjectRef`;
- referencias de auditoría;
- huellas criptográficas;
- nombres de campos de un protocolo;
- estados máquina;
- identificadores de permisos.

Traducir un código rompe clientes y evidencia. Solo se traduce su presentación.

## 8.10 Catálogos y parámetros

Dos catálogos tienen paridad si contienen las mismas claves activas y cada
mensaje acepta los mismos parámetros tipados.

Ejemplo conceptual:

```text
clave: workitem.retry_scheduled
parámetros:
  attempt: entero
  at: instante

es: "Reintento {attempt} programado para {at}."
en: "Retry {attempt} scheduled for {at}."
```

El ejemplo inglés forma parte de un catálogo bilingüe, no de la conversación
con el operador.

Las pruebas deben detectar:

- clave ausente;
- clave sobrante no declarada;
- parámetro ausente;
- parámetro de tipo distinto;
- plural incompleto;
- cadena visible directa fuera del catálogo;
- respaldo lingüístico roto;
- superficie activa sin fuente de traducción.

No se recomienda una heurística que busque «palabras españolas» o «palabras
inglesas». Produce falsos positivos y no prueba propiedad tipada.

## 8.11 Plurales, números, moneda, fecha y zona horaria

El catálogo expresa categorías plurales; no concatena una `s`.

```text
0 elementos
1 elemento
2 elementos
```

Números y moneda se formatean con locale:

```text
es-ES: 1.234,50 €
en-US: €1,234.50
```

Un instante durable se almacena en un formato neutral y se presenta en la zona
horaria solicitada. El catálogo no debe incrustar una zona fija.

Las pruebas incluyen:

- cero, uno, dos y valores grandes;
- decimales y redondeo;
- moneda conocida y desconocida;
- cambio de día por zona;
- horario de verano;
- locale ausente;
- parámetros inválidos.

## 8.12 Activar una superficie nueva

Una superficie no pasa a activa solo porque compila. Antes debe:

1. declarar sus textos en el manifiesto;
2. generar o registrar sus claves;
3. completar catálogos español e inglés;
4. probar paridad de claves y parámetros;
5. probar español por defecto y respaldo lingüístico;
6. probar BCP-47 válido e inválido;
7. demostrar que no contiene cadenas visibles privadas;
8. enlazarse al registro común de órdenes;
9. acreditar autenticación y autorización;
10. acreditar su composición real.

Si una superficie sigue marcada como futura en el manifiesto, sus textos no
pueden contarse como cobertura productiva.

## 8.13 HTTP

Una API HTTP fina:

- traduce método, ruta, cabeceras y cuerpo;
- autentica;
- selecciona locale;
- resuelve una orden registrada;
- entrega la envolvente al despachador;
- traduce el resultado neutral a estado, cabeceras y cuerpo;
- propaga una referencia de correlación;
- no contiene decisiones del ciclo de vida.

### Negativos

- método o tipo de contenido incorrectos;
- campos adicionales bajo esquema estricto;
- cuerpo demasiado grande;
- revisión ausente cuando es obligatoria;
- clave de idempotencia ausente;
- proyecto contradictorio entre ruta y cuerpo;
- locale inválido;
- principal de otro proyecto;
- fuga de detalles internos.

`UI-01` sigue declarado. Que algunas rutas participaran en la evidencia de V20
no acredita todavía la API HTTP total prometida por esa capacidad.

## 8.14 MCP

El servidor MCP real expone herramientas, recursos y plantillas de
instrucciones (`prompts`) como contratos distintos:

- una herramienta ejecuta una orden registrada;
- un recurso proporciona datos direccionables y paginados;
- una plantilla de instrucciones (`prompt`) proporciona texto gobernado;
- una destreza conserva su propio ámbito, huella criptográfica y confianza;
- un complemento empaqueta capacidades sin entrar en dominio.

Una herramienta MCP no debe convertirse en un atajo al almacén. Su descripción
se localiza; su identificador y esquema permanecen estables.

`UI-02` está acreditado, pero cada nueva herramienta aún necesita:

- definición registrada;
- permiso;
- esquema;
- coste;
- idempotencia;
- recibo cuando produce efecto;
- pruebas de ámbito y errores;
- paridad con la aplicación.

## 8.15 Línea de órdenes y biblioteca cliente

La línea de órdenes es un traductor fino:

```text
argumentos -> entrada tipada -> despachador -> salida tipada -> presentación
```

Sus códigos de salida derivan del error neutral. La salida para máquinas debe
ser estable; la salida para personas puede localizarse.

La biblioteca cliente no duplica el dominio. Ofrece:

- tipos generados;
- construcción segura de peticiones;
- transporte;
- decodificación de errores;
- repetición de red compatible con idempotencia;
- selección de locale;
- ninguna autorización local fingida.

`UI-03` sigue declarado. La presencia de enlaces de órdenes acreditados en V20
no equivale al producto total de terminal.

## 8.16 Web instalable y accesible

La web debe consumir los mismos casos de uso. Puede ser una aplicación web
instalable, conocida por la sigla `PWA`, sin crear un escritorio nativo
separado.

### Arquitectura

```text
navegador
  ├── vistas y componentes
  ├── estado de interfaz no autoritativo
  ├── cliente generado
  ├── catálogo y formatos
  └── caché permitida
           │
           ▼
API registrada ─► aplicación ─► único estado
```

La caché del navegador no decide que un `Goal` esté cerrado. Una operación sin
red se conserva como intención local solo si el contrato define reconciliación,
idempotencia y visibilidad.

### Accesibilidad

La meta debe ser WCAG 2.2 nivel AA, con pruebas automáticas y manuales. Como
mínimo:

- HTML semántico;
- navegación completa por teclado;
- orden de foco predecible;
- foco visible;
- nombres y descripciones accesibles;
- contraste suficiente;
- estados no expresados solo con color;
- regiones vivas para cambios importantes;
- errores asociados a sus campos;
- ampliación y reflujo;
- preferencia de movimiento reducido;
- objetivos táctiles utilizables;
- tablas y gráficos con alternativa;
- idioma del documento y de fragmentos;
- tiempo ampliable cuando exista límite.

### Pruebas web

- matriz de teclado por flujo;
- lector de pantalla en rutas críticas;
- contraste;
- ampliación;
- tema claro y oscuro si ambos se ofrecen;
- conexión lenta, desconexión y reanudación;
- sesión expirada;
- doble envío;
- actualización concurrente;
- locale y zona horaria;
- límites de móvil y escritorio.

Estas obligaciones son diseño para las capacidades web pendientes. No deben
describirse como acreditadas hoy.

## 8.17 Regla transversal APP-13

`APP-13` exige que cualquier aplicación creada o modificada por Orquesta pueda
ser entendida y operada por una persona o por otro agente. La guía de trabajo
está en
[`GUIA_MAESTRA_AGENTES_INVENTARIO_Y_RECONSTRUCCION.md`](../GUIA_MAESTRA_AGENTES_INVENTARIO_Y_RECONSTRUCCION.md)
y el bloqueo vivo en
[`bloqueo_universalidad_app13_2026-07-30.md`](../bloqueo_universalidad_app13_2026-07-30.md).

La regla aplica a aplicaciones Go y no Go. No basta con generar un `README`, ni
con documentar solo aplicaciones nuevas.

### Cuatro casos obligatorios

La aceptación universal debe cubrir:

1. crear una aplicación Go;
2. modificar una aplicación Go existente;
3. crear una aplicación no Go;
4. modificar una aplicación no Go existente.

Todos deben ejecutarse y probarse en el mismo árbol o imagen candidata que se
pretende acreditar. Un archivo de ejemplo aislado no prueba la aplicación.

## 8.18 Identificar la aplicación objetivo

Antes de planificar, el expediente necesita un sujeto durable equivalente a
`ApplicationTarget`. El nombre definitivo requiere contrato, pero su contenido
mínimo es:

```text
modo: create | modify
identidad opaca de aplicación
raíz relativa canónica
perfil de calidad
referencia y huella criptográfica de política de documentación
referencia y huella criptográfica de política de internacionalización
```

`create` significa crear; `modify`, modificar. Son valores máquina.

La raíz no se deduce con una búsqueda oportunista. Debe:

- pertenecer al proyecto;
- ser relativa y canónica;
- resolver dentro del espacio autorizado;
- rechazar `..`, rutas absolutas y enlaces simbólicos que escapen;
- conservar identidad aunque cambie la sesión;
- formar parte del sujeto de atestación.

No se crea un segundo registro de aplicaciones, otro ciclo de vida ni otra puerta
de cierre. El sujeto se enlaza al `ProjectRef`, al `Goal` y al `AppSpec`.

## 8.19 Las ocho secciones obligatorias

Cada aplicación debe tener un manifiesto o documento canónico con ocho
secciones tipadas, no vacías y verificables.

### 1. Propósito y usuarios

Debe explicar:

- qué problema resuelve;
- quién la usa;
- cuáles son sus resultados observables;
- qué supuestos de uso mantiene.

No sirve «aplicación principal» ni repetir su nombre.

### 2. Alcance y exclusiones

Debe declarar:

- funciones incluidas;
- límites;
- responsabilidades de otros sistemas;
- comportamientos deliberadamente ausentes;
- estado experimental o condicionado.

Esta sección impide que una omisión parezca una capacidad terminada.

### 3. Entradas y salidas

Debe enumerar:

- órdenes y consultas;
- archivos y formatos;
- variables o configuración admitidas a través del registro;
- recursos y artefactos;
- salidas, errores y recibos;
- límites de tamaño y validación.

No se documentan secretos concretos.

### 4. Arquitectura y módulos

Debe mostrar:

- módulos y dependencias;
- fronteras hexagonales;
- casos de uso;
- puertos y adaptadores;
- procesos externos;
- flujo relevante de datos.

Una lista de carpetas sin responsabilidad no basta.

### 5. Autoridades de escritura

Debe indicar:

- quién puede mutar cada estado durable;
- qué componentes son solo lectura;
- dónde se aplican revisión esperada e idempotencia;
- qué proyecciones no son autoridad;
- qué efectos requieren aprobación.

Esta es una de las secciones más importantes para depurar.

### 6. Datos, permisos, secretos y efectos

Debe cubrir:

- clasificación de datos;
- aislamiento;
- permisos;
- almacenamiento y retención;
- credenciales por referencia;
- redacción;
- efectos externos;
- recibos y reversibilidad;
- amenazas y negativos.

### 7. Arranque, diagnóstico, recuperación y parada

Debe permitir a otra persona:

- arrancar la aplicación;
- comprobar salud;
- localizar registros y referencias;
- diagnosticar un atasco;
- recuperar estado;
- reiniciar sin duplicar efectos;
- hacer copia y restauración cuando aplique;
- detener y comprobar que no quedan procesos propios.

### 8. Contratos y pruebas

Debe enlazar:

- contratos de aceptación;
- pruebas unitarias;
- pruebas de contrato;
- negativos;
- mutaciones;
- pruebas de extremo a extremo;
- evidencias;
- límites conocidos y capacidades no acreditadas.

«Tiene pruebas» no es una descripción suficiente.

## 8.20 Cabeceras de responsabilidad

Cada módulo, herramienta, adaptador o archivo relevante debe explicar de forma
breve:

- qué responsabilidad tiene;
- qué no hace deliberadamente;
- si escribe o solo lee;
- qué contratos aplica;
- cómo se prueba.

Ejemplo conceptual en Go:

```go
// Package dispatch traduce órdenes registradas a casos de uso.
//
// No decide transiciones de Goal ni accede a un almacén concreto. La
// autorización y la escritura del ciclo de vida pertenecen a application.
package dispatch
```

Para una función exportada:

```go
// Dispatch valida la envolvente registrada y ejecuta su manejador común.
// No autentica credenciales de transporte ni traduce mensajes humanos.
func Dispatch(...) ...
```

En Go, la documentación del paquete puede cumplir la cabecera del módulo. No es
obligatorio repetir un bloque en cada archivo si no añade una frontera real.
Los símbolos exportados siguen las convenciones de Go.

Ejemplo conceptual en TypeScript:

```typescript
/**
 * Traduce el estado neutral del caso de uso a la vista de tareas.
 * No modifica el Goal ni decide permisos; esas autoridades viven en servidor.
 * Contratos: registro de órdenes, catálogo i18n y pruebas de accesibilidad.
 */
```

### Una cabecera útil no repite el nombre

Insuficiente:

```text
El planificador sirve para programar.
```

Útil:

```text
Reclama WorkItems listos con arrendamiento y cercado. No cambia el Goal fuera
del escritor de aplicación ni mantiene colas privadas por proveedor.
```

## 8.21 Go, otros lenguajes y archivos no comentables

La política debe adaptarse a la sintaxis:

- Go: documentación de paquete y símbolos exportados;
- JavaScript o TypeScript: comentarios de módulo y contratos públicos;
- Python: documento de módulo y API pública;
- Rust: documentación de módulo y elementos públicos;
- shell: propósito, entradas, efectos y código de salida;
- SQL: migración, precondición, reversión y autoridad;
- archivos de configuración: documento compañero o campos de metadatos
  permitidos por su esquema.

No se deben insertar comentarios en:

- binarios;
- formatos que no los admiten;
- código generado;
- dependencias vendorizadas;
- archivos cuya huella criptográfica externa exige inmutabilidad.

Esos casos requieren una exclusión tipada y una referencia desde el manifiesto:
tipo de archivo, razón, generador o propietario y forma de verificación. Una
exclusión no puede ser un patrón amplio para ocultar código propio.

## 8.22 Planificación transversal de APP-13

La regla no puede depender de que el agente recuerde documentar. Una política
determinista de planificación debe:

1. identificar la raíz de aplicación;
2. calcular si el conjunto de escritura (`write-set`) la toca;
3. inyectar las pruebas `APP-13` exigidas;
4. conservarlas en todo replanteamiento;
5. exigir actualización del manifiesto si cambia el contrato;
6. exigir cabeceras para archivos relevantes nuevos o modificados;
7. exigir internacionalización para texto visible;
8. impedir cierre sin atestación del sujeto exacto.

El solapamiento de rutas se calcula por segmentos canónicos. Una raíz
`aplicaciones/api` no coincide por prefijo textual con
`aplicaciones/apicultura`.

### Replanteamiento

Una replanificación no puede eliminar las pruebas transversales para hacer
pasar el candidato. Si cambia la raíz o el modo, se calcula un nuevo sujeto y
se invalida la atestación anterior.

## 8.23 Comprobador neutral

El comprobador de `APP-13` debe ser una herramienta gobernada:

- identificador y versión;
- esquema de entrada y salida;
- permisos;
- coste y límites;
- idempotencia;
- huella criptográfica del ejecutable o imagen;
- recibo;
- política de redacción;
- adaptadores por lenguaje si son necesarios.

No debe importar el dominio ni escribir el ciclo de vida. Recibe un sujeto autorizado,
analiza el árbol y devuelve un informe estructurado.

### Informe conceptual

```text
informe de documentación (`AppDocumentationReport`)
  sujeto (`Subject`)
    referencia de proyecto (`ProjectRef`)
    referencia de aplicación (`ApplicationRef`)
    raíz canónica (`CanonicalRoot`)
    referencia del objetivo (`GoalRef`)
    referencia de unidad de trabajo (`WorkItemRef`)
    generación del plan (`PlanGeneration`)
    generación de la especificación (`AppSpecGeneration`)
    huella del árbol (`TreeDigest`)
    huella del cambio (`ChangeDigest`)
    huella de la política (`PolicyDigest`)
  manifiesto (`Manifest`)
    localizado (`Located`)
    ocho secciones válidas (`EightSectionsValid`)
  cabeceras (`Headers`)
    ficheros pertinentes (`RelevantFiles`)
    ficheros cubiertos (`CoveredFiles`)
    exclusiones (`Exclusions`)
    hallazgos (`Findings`)
  internacionalización (`I18n`)
    referencia de política (`PolicyRef`)
    hallazgos de texto visible (`VisibleTextFindings`)
    hallazgos de catálogo (`CatalogFindings`)
  pruebas (`Tests`)
    exigidas (`Required`)
    observadas (`Observed`)
  veredicto (`Verdict`)
  referencia del recibo de herramienta (`ToolReceiptRef`)
  huella del informe (`ReportDigest`)
```

Los nombres son ilustrativos y no sustituyen el esquema que aún debe aprobarse.

### La huella criptográfica no puede autocontenerse

Un manifiesto dentro del árbol no puede contener de forma estable la huella
criptográfica del mismo árbol que lo contiene: escribirla cambia el árbol. La
atestación vive fuera del sujeto o en un índice posterior y referencia:

- árbol exacto;
- conjunto de cambios;
- aplicación;
- proyecto;
- `Goal`;
- `WorkItem`;
- generaciones;
- política;
- ejecutable del comprobador.

## 8.24 Internacionalización dentro de APP-13

Una aplicación bien documentada pero con texto visible codificado sigue
incumpliendo si su política exige internacionalización.

El manifiesto debe declarar:

- idiomas instalados;
- idioma por defecto;
- respaldo lingüístico;
- dueño de catálogos;
- fuentes tipadas;
- superficies activas;
- tratamiento de fecha, número, moneda y zona;
- archivos excluidos con razón;
- pruebas.

El comprobador debe entender la tecnología. En Go puede inspeccionar fuentes
tipadas; en una web puede comprobar componentes y catálogos; en una utilidad
shell puede existir un perfil más pequeño. «No Go» no significa «sin
contrato».

## 8.25 Pruebas de APP-13

### Positivas

- aplicación Go nueva con ocho secciones, cabeceras e i18n;
- aplicación Go existente modificada;
- aplicación no Go nueva;
- aplicación no Go existente modificada;
- exclusión válida de código generado;
- cambio funcional que actualiza su documentación;
- mismo árbol o imagen ejecutando pruebas funcionales y documentales.

### Negativas del manifiesto

- sección ausente;
- sección vacía;
- tipo incorrecto;
- contenido genérico que no identifica responsabilidad;
- raíz distinta de la acreditada;
- política o huella criptográfica obsoletos;
- modo `create` aplicado a una modificación.

### Negativas de cabecera

- archivo relevante omitido;
- cabecera que repite el nombre;
- cabecera en idioma incorrecto;
- autoridad de escritura falsa;
- exclusión demasiado amplia;
- comentario insertado en formato no permitido;
- cabecera correcta en un archivo distinto para engañar al comprobador.

### Negativas de seguridad y sujeto

- ruta absoluta;
- `..`;
- enlace simbólico que sale de raíz;
- raíz falsa;
- cruce de proyecto;
- secreto incluido en documentación;
- recibo de otro árbol;
- recibo de otra aplicación;
- recibo manipulado;
- herramienta no permitida o sin huella criptográfica.

### Negativas de cierre

- pruebas funcionales verdes sin atestación documental;
- atestación verde con pruebas funcionales fallidas;
- solo caso Go acreditado;
- aplicación creada sí, modificada no;
- replanificación que elimina la prueba;
- texto visible sin catálogo;
- documentación generada que no pertenece al candidato.

## 8.26 Por qué APP-13 está en NO-GO

El bloqueo actual no es una duda editorial. Faltan piezas causales:

1. reconciliar en las autoridades vigentes la aceptación universal de los
   cuatro casos;
2. modelar de forma durable la aplicación objetivo;
3. proyectarla desde el Asistente y el expediente;
4. inyectar las pruebas transversales desde una política central;
5. preservar esa obligación al replantear;
6. definir y componer un atestador neutral para aplicaciones no Go;
7. gobernar la herramienta de comprobación;
8. emitir informe y recibo ligados al sujeto exacto;
9. ejecutar los cuatro extremos a extremo;
10. sellar evidencia del mismo árbol, binario o imagen y configuración.

Además existe una discrepancia que no se debe resolver solo en este manual. El
contrato vigente de aplicaciones generadas exige cubrir Go y no Go en el mismo
candidato, mientras que la guía maestra explicita cuatro casos: crear y
modificar en ambos grupos tecnológicos. Antes de convertir esa exigencia en una
puerta automática hay que reconciliar de forma atómica la hoja de ruta, la ruta
causal y el contrato de aceptación. Hasta entonces, el manual conserva la
exigencia más completa como objetivo, pero no falsifica una acreditación.

Hay manifiestos locales útiles. Demuestran esas aplicaciones concretas, no una
propiedad universal de cualquier aplicación que Orquesta vaya a crear o
modificar.

Hasta completar la cadena, está prohibido:

- marcar `APP-13` como acreditado;
- usar un documento aislado como prueba;
- presentar solo Go como cobertura universal;
- afirmar que una cabecera existe sin comprobar el árbol;
- convertir la guía manual en una puerta productiva ficticia.

## 8.27 Secuencia completa para crear una aplicación

Una vez acreditada la regla, el flujo objetivo será:

```text
1. intención identifica aplicación, modo y usuarios
2. AppSpec fija raíz, perfil de calidad e i18n
3. planificación deriva casos de uso y pruebas
4. política APP-13 inyecta documentación y cabeceras
5. agentes implementan con conjuntos de escritura acotados
6. registro genera las superficies necesarias
7. catálogos cubren todo texto visible
8. pruebas funcionales y documentales corren en el mismo candidato
9. comprobador emite informe y recibo
10. autor, revisión primaria y adversarial evalúan el sujeto
11. Consejo interviene si la política lo exige
12. integración revalida todas las huellas criptográficas
13. evidencia acredita la revisión exacta
```

Para modificar una aplicación, se añade inventario previo:

- manifiesto existente;
- fronteras;
- autoridades;
- superficies;
- catálogos;
- deuda;
- pruebas;
- comportamiento del legado que se va a sustituir.

La modificación conserva material útil, pero no hereda afirmaciones sin prueba.

## 8.28 Diagnóstico de divergencia entre superficies

Si HTTP funciona y MCP no, o si una salida cambia de idioma de forma
incoherente:

1. identificar `command_id` y versión;
2. localizar su única definición en el registro;
3. comparar proyecciones generadas;
4. comprobar que ambas llegan al mismo manejador;
5. comparar la envolvente de identidad y proyecto;
6. revisar esquema y campos adicionales;
7. comprobar `expected_revision` e idempotencia;
8. comparar código máquina antes de la traducción;
9. resolver configuración regional y respaldo lingüístico;
10. comprobar clave y parámetros de catálogo;
11. revisar la evidencia de composición, no solo unitarios.

### Síntomas y causas probables

| Síntoma | Causa probable |
|---|---|
| HTTP acepta un campo que MCP rechaza | esquemas duplicados o generación obsoleta |
| CLI devuelve otro error | tabla privada de traducción |
| web muestra un estado imposible | estado local tratado como autoridad |
| el reintento duplica un efecto | idempotencia aplicada solo en transporte |
| un locale inválido cae a español | normalización silenciosa |
| una clave existe solo en español | paridad de catálogo rota |
| el texto está traducido pero el número no | formateo fuera del sistema i18n |
| APP-13 pasa en Go y falla fuera de Go | comprobador ligado al lenguaje |
| manifiesto verde en otro árbol | sujeto o recibo incompletos |
| cabeceras genéricas por todas partes | comprobación por presencia, no por contrato |

## 8.29 Pruebas mínimas del registro y superficies

### Registro

- identificador y versión únicos;
- esquema estricto;
- manejador existente;
- permiso y audiencia;
- código de error declarado;
- enlace por transporte;
- proyección reproducible;
- rechazo de una orden no registrada;
- ausencia de imports de dominio desde interfaces.

### Paridad

Para cada orden activa:

- misma entrada válida en HTTP, MCP, línea de órdenes y biblioteca;
- misma entrada inválida;
- mismo código máquina;
- mismo resultado neutral;
- mismo aislamiento;
- misma revisión esperada;
- misma idempotencia;
- misma localización de texto humano;
- mismo recibo para una mutación repetida.

La web se incorpora a esta matriz al activarse.

### Identidad

- credencial ausente;
- credencial inválida;
- principal sin permiso;
- proyecto ajeno;
- ejecución no ligada;
- carga útil intentando suplantar al principal;
- sesión caducada;
- revocación;
- repetición después de rotar credenciales.

### Internacionalización

- español por defecto;
- español como respaldo lingüístico;
- `es-ES` a `es`;
- `en-GB` a `en`;
- idioma válido no instalado;
- etiqueta inválida;
- paridad de claves;
- paridad de parámetros;
- plurales;
- números, moneda, fechas y zonas;
- códigos máquina idénticos;
- ausencia de cadenas visibles directas;
- documentación pública localizada.

### Accesibilidad y web

- teclado;
- foco;
- lector de pantalla;
- contraste;
- reflujo;
- movimiento reducido;
- estados dinámicos;
- error y recuperación;
- conexión lenta y sin conexión;
- instalación y actualización de la aplicación web;
- caché que no inventa autoridad.

## 8.30 Orden recomendado de construcción desde cero

1. Definir casos de uso y errores neutrales.
2. Implementar autorización en aplicación.
3. Añadir revisión esperada e idempotencia.
4. Crear el registro único de órdenes.
5. Generar despachador y proyecciones.
6. Componer primero un transporte mínimo y probarlo.
7. Añadir HTTP, MCP y línea de órdenes sobre el mismo contrato.
8. Crear manifiesto i18n, catálogos y selección BCP-47.
9. Añadir biblioteca cliente.
10. Construir web instalable y accesible como cliente.
11. Modelar la aplicación objetivo de `APP-13`.
12. Inyectar la política transversal en planificación.
13. implementar el comprobador neutral y adaptadores por tecnología.
14. acreditar crear y modificar en Go y no Go.

No conviene construir primero una interfaz rica y después intentar deducir qué
casos de uso ejecutaba. La autoridad debe existir antes que su presentación.

## 8.31 Criterio de cierre

Una superficie está cerrada solo si:

- usa una orden registrada;
- llama al mismo caso de uso;
- no escribe el ciclo de vida;
- autentica y autoriza correctamente;
- conserva proyecto e identidad;
- aplica revisión esperada e idempotencia;
- devuelve códigos máquina estables;
- localiza todo texto humano;
- prueba aislamiento y errores;
- funciona en la composición prometida.

Una aplicación cumple `APP-13` solo si:

- su sujeto y raíz son inequívocos;
- contiene las ocho secciones;
- los módulos relevantes explican responsabilidad y autoridad;
- Go y no Go obedecen políticas equivalentes;
- i18n está cubierta;
- el comprobador es gobernado;
- el recibo liga el árbol exacto;
- las pruebas funcionales y documentales corresponden al mismo candidato;
- los cuatro casos universales están acreditados.

El estado honesto actual de Orquesta es: registro V20 e internacionalización
V21 acreditados en su alcance; servidor MCP real acreditado; API total, línea
de órdenes total, web accesible e instalable y universalidad `APP-13` todavía
pendientes de sus propias verticales y evidencias.
