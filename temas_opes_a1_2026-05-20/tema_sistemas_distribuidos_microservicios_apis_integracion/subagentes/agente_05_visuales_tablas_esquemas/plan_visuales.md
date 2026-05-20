# Plan de visuales utiles para el tema

Este plan esta pensado para una primera lectura continua. Los visuales deben
aparecer como apoyo, no como sustituto del desarrollo teorico. Cada figura debe
tener una lectura guiada: que muestra, por que importa y como se reconoce en un
supuesto de examen.

## Inventario propuesto

| ID | Visual | Ubicacion sugerida | Objetivo didactico | Asset borrador |
| --- | --- | --- | --- | --- |
| V01 | Mapa de capas de arquitectura e integracion | Tras el indice y la orientacion de examen | Dar una ruta mental: estrategia, servicios, integracion, datos, seguridad y operacion | `assets_borrador/mapa_capas_integracion.svg` |
| V02 | API gateway y microservicios en una administracion publica | Bloque de microservicios y APIs | Separar canal, contrato, orquestacion, servicios y sistemas de registro | `assets_borrador/flujo_api_gateway_microservicios.svg` |
| V03 | Integracion por eventos, observabilidad y resiliencia | Bloque de arquitectura distribuida | Explicar asincronia, trazabilidad, tolerancia a fallos y recuperacion | `assets_borrador/eventos_observabilidad_resiliencia.svg` |
| V04 | Intermediacion de datos entre administraciones | Bloque de interoperabilidad tecnica y servicios compartidos | Mostrar consulta con consentimiento/base juridica, plataforma intermediaria y organismo cedente | `assets_borrador/intermediacion_datos_aapp.svg` |
| V05 | Ciclo de vida de una API publica | Bloque de gobierno de APIs | Conectar diseno, versionado, seguridad, pruebas, publicacion, medicion y retirada | Tabla T03, sin SVG obligatorio |
| V06 | Comparativa de estilos de integracion | Bloque de integracion de sistemas | Distinguir REST, SOAP, eventos, batch y ficheros por uso adecuado | Tabla T02 |
| V07 | Patrones de resiliencia | Bloque de operacion y calidad de servicio | Reconocer circuit breaker, bulkhead, retry, timeout, idempotencia y colas | Tabla T04 |

## Criterios de uso

### V01. Mapa de capas de arquitectura e integracion

Lectura esperada:

1. La parte superior representa el marco de decision: estrategia publica,
   normativa, gobierno y orientacion al servicio.
2. La zona central representa los componentes que materializan el servicio:
   canales digitales, APIs, microservicios, integracion y datos.
3. La parte inferior representa las condiciones transversales: seguridad,
   observabilidad, continuidad, calidad y cumplimiento.

Texto alternativo sugerido:

> Mapa por capas que relaciona gobierno, canales, APIs, microservicios,
> integracion, datos, seguridad y operacion en una arquitectura distribuida de
> la Administracion publica.

Nota de tutor:

> No memorices el dibujo como una topologia fisica. Usalo para ordenar la
> pregunta: primero identifica el servicio publico, despues el contrato de
> interoperabilidad, despues el mecanismo tecnico y, por ultimo, las garantias
> de seguridad y operacion.

### V02. API gateway y microservicios

Lectura esperada:

- El canal no llama directamente a todos los sistemas internos.
- El gateway concentra politicas de entrada: autenticacion, autorizacion,
  limitacion de tasa, versionado y trazabilidad.
- Los microservicios exponen capacidades de negocio acotadas.
- Los sistemas de registro pueden seguir existiendo. La arquitectura moderna no
  exige sustituirlos de golpe, sino encapsularlos y gobernar sus contratos.

Texto alternativo sugerido:

> Esquema de una solicitud que entra por un canal digital, pasa por un API
> gateway, se distribuye a microservicios y consulta sistemas de registro con
> observabilidad transversal.

Nota de test:

> Trampa habitual: confundir gateway con bus de integracion. El gateway protege
> y gobierna el acceso a APIs. Un bus o plataforma de integracion media flujos
> entre sistemas, transforma mensajes o conecta aplicaciones heterogeneas.

### V03. Eventos, observabilidad y resiliencia

Lectura esperada:

- La publicacion de eventos desacopla productor y consumidores.
- La cola o broker ayuda a absorber picos y a reintentar.
- La observabilidad debe correlacionar trazas, logs y metricas.
- La resiliencia se disena antes del incidente: timeouts, circuit breaker,
  idempotencia, colas de errores y procedimientos de recuperacion.

Texto alternativo sugerido:

> Flujo de evento administrativo publicado por un servicio, consumido por otros
> servicios y supervisado mediante trazas, metricas, logs y mecanismos de
> recuperacion.

Nota de test:

> Si el enunciado habla de notificar cambios a varios consumidores sin bloquear
> el tramite principal, la respuesta suele apuntar a eventos o mensajeria
> asincrona, no a llamadas sincronas encadenadas.

### V04. Intermediacion de datos

Lectura esperada:

- Un organo tramitador solicita un dato necesario para el procedimiento.
- La plataforma de intermediacion aplica reglas de autorizacion, auditoria y
  trazabilidad.
- El organismo cedente responde con el dato o certificado disponible.
- El ciudadano no debe aportar documentos que la Administracion pueda consultar
  conforme al marco juridico aplicable.

Texto alternativo sugerido:

> Esquema de intermediacion de datos donde un organo tramitador consulta, a
> traves de una plataforma comun, la informacion de un organismo cedente con
> control de autorizacion y auditoria.

Nota de test:

> No confundir interoperabilidad con libre acceso indiscriminado. La consulta
> debe apoyarse en competencia, finalidad, base juridica, minimizacion y
> trazabilidad.

## Comportamiento responsive

- Cada SVG debe insertarse dentro de un contenedor con `overflow-x: auto`.
- El SVG debe conservar `viewBox`, `width: 100%`, `height: auto` y un ancho
  minimo razonable en pantallas estrechas.
- En movil debe permitirse scroll horizontal si el texto interno quedara
  ilegible.
- El texto alternativo debe resumir la idea, no repetir todos los rotulos.
- Las notas de tutor y de test deben ser bloques independientes y ocultables en
  HTML.

## Plan de sincronizacion Markdown/HTML

| Elemento | Markdown | HTML |
| --- | --- | --- |
| Figura | `![alt](assets/nombre.svg)` tras mover el asset | `<figure class="visual visual--scroll">` con `img` local |
| Caption | Parrafo breve bajo la imagen | `<figcaption>` equivalente |
| Nota tutor | Bloque separado con etiqueta "Modo tutor" | Panel ocultable o bloque de tutor |
| Nota test | Bloque separado, sin mezclarse con teoria | Panel azul ocultable conforme a politica HTML |
| Referencia | Fuente editorial en `fuentes.md` | Misma fuente editorial, sin URL visible |
