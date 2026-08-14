# Cliente Go para Orquesta

Este módulo consume `agentmicrovm.local.v1` por HTTP/1.1 sobre un socket Unix.
Es deliberadamente neutral: no importa paquetes de Orquesta ni conoce Goal,
WorkItem, permisos, presupuesto, proveedores o lifecycle del consumidor.

Orquesta debe declarar su propio puerto y mantener en su adaptador la
traducción entre sus referencias opacas y estos DTO JSON. El cliente aporta:

- salud y negociación de capacidades;
- lanzamiento firmado y observación;
- órdenes no interactivas y revisión de trabajo;
- sesiones asíncronas: inicio idempotente, entrada idempotente y eventos
  paginados por cursor;
- sincronización sellada de entrada y salida, con acreditación de cada árbol
  antes y después del socket;
- detención, preservación y cierre lógico;
- cabeceras de protocolo e idempotencia en todas las mutaciones;
- errores tipados y respuestas acotadas a 12 MiB.

## Backend Docker

`NegociarDocker` exige el selector `docker`, disponibilidad real del Engine y
las diez operaciones del recorrido; también rechaza campos desconocidos o una
forma JSON distinta. Tras negociar, la superficie específica es:

- `PrepararContenedor`, `CodificarPlanLanzamientoContenedorV1` y
  `CodificarSolicitudLanzarORecuperarContenedorV1` para producir y fijar la
  intención Ed25519 que el consumidor persiste antes del socket;
- `LanzarORecuperarContenedor` y
  `LanzarORecuperarContenedorCodificada` para enviar un DTO nuevo o reenviar
  sin reserialización el envelope persistido;
- `ConsultarOperacionContenedor` y `ObservarContenedor` para separar
  aceptación durable de liveness;
- `CodificarSolicitudDetenerContenedorV1`, `DetenerContenedor` y
  `DetenerContenedorCodificada` para fijar la intención, reenviarla sin
  reserializar y exigir el receipt terminal cercado;
- `EjecutarOrdenContenedor`, `SincronizarEntradaContenedor` y
  `SincronizarSalidaContenedor`, junto con sus codificadores y variantes
  `Codificada`, para trabajo acotado, contenido sellado y replay byte a byte;
- `IniciarSesionContenedor`, `EnviarEntradaSesionContenedor` y
  `LeerEventosSesionContenedor` para el curso vivo y terminal.

Inicio y entrada exponen además
`CodificarSolicitudInicioSesionContenedorV1`,
`CodificarSolicitudEntradaSesionContenedorV1`,
`IniciarSesionContenedorCodificada` y
`EnviarEntradaSesionContenedorCodificada`. El consumidor puede fijar y
persistir el cuerpo antes del primer intento y, tras una respuesta ambigua o
un reinicio propio, reenviar esos mismos bytes. La ruta codificada comprueba
forma cerrada, duplicados, semántica y límite de cuerpo antes del socket, pero
no reserializa una intención válida.

Todos los métodos que reciben una respuesta la validan con esquemas cerrados
independientes de los DTO públicos. Las fixtures `plan_contenedor_v1.json`,
`lanzamiento_contenedor_firmado_v1.json`, `trabajo_contenedor_v1.json` y
`sesion_contenedor_v1.json` son compartidas con Rust. El cliente no abre el
socket Docker, no conoce rutas del host y no convierte un ACK o un receipt
negativo en efecto físico correcto.

La semántica HTTP también es cerrada: lanzamiento e inicio de sesión exigen
`201 Created`; negociación, consultas, detención, orden, sincronizaciones,
entrada y eventos exigen `200 OK`. Otro `2xx`, incluido `202 Accepted`, se
devuelve como `api.estado_inesperado` antes de interpretar el JSON. Así, un
ACK con forma de receipt no se convierte en efecto confirmado.

Las respuestas no exitosas de esas mismas rutas sólo conservan `codigo` y
`detalle` cuando el cuerpo coincide exactamente con `ProblemaV1`. Campos
desconocidos, claves duplicadas, campos ausentes o tipos divergentes no pueden
inyectar un código máquina: se reducen al fallo cerrado
`api.respuesta_rechazada`. Las rutas históricas ajenas al selector Docker
mantienen su compatibilidad anterior.

El transporte UDS no sigue redirecciones HTTP. Una respuesta `3xx` se entrega
al validador como rechazo de la ruta original y nunca reenvía una mutación, su
cuerpo persistido o su clave de idempotencia a la ruta indicada por
`Location`.

Cada respuesta debe aportar exactamente una cabecera
`x-agentmicrovm-protocolo` con valor `agentmicrovm.local.v1`. Un valor
incompatible, un segundo valor divergente o incluso dos testigos `v1` no
constituyen una identidad de protocolo única y fallan antes de leer el cuerpo.

Las once llamadas Docker exigen además un único `Content-Type` cuyo media type
sea `application/json`; parámetros MIME válidos son compatibles. Ausencia,
otro tipo o cabeceras duplicadas fallan como
`microvm.tipo_contenido_incompatible` antes de clasificar status o problema.

Ejemplo mínimo Docker:

```go
capacidades, err := cliente.NegociarDocker(ctx)
if err != nil {
	return err
}
if capacidades.BackendEjecuciones != microvm.BackendEjecucionesDocker {
	return errors.New("backend inesperado")
}

preparada, err := firmante.PrepararContenedor(plan, ahora, vigencia)
if err != nil {
	return err
}
cuerpo, err := microvm.CodificarSolicitudLanzarORecuperarContenedorV1(preparada)
if err != nil {
	return err
}
if err := persistirIntento(plan.OperacionRef, cuerpo); err != nil {
	return err
}
_, err = cliente.LanzarORecuperarContenedorCodificada(
	ctx, plan.OperacionRef, cuerpo,
)
```

La integración del consumidor decide dónde persiste esa intención y cómo
proyecta sus propios estados. Este módulo no aporta una base de datos paralela
ni autoriza a reconstruir el cuerpo después de una respuesta ambigua.

La misma separación se aplica a las sesiones Docker:

```go
cuerpoInicio, err := microvm.CodificarSolicitudInicioSesionContenedorV1(inicio)
if err != nil {
	return err
}
if err := persistirIntento(claveInicio, cuerpoInicio); err != nil {
	return err
}
_, err = cliente.IniciarSesionContenedorCodificada(
	ctx, claveInicio, referencia, cuerpoInicio,
)
```

La persistencia mostrada pertenece al consumidor. Este módulo sólo acredita
que los bytes preparados y los de replay coinciden exactamente.

La misma pareja `CodificarSolicitud…`/método `…Codificada` existe para
detención, orden y sincronizaciones de entrada/salida. Sus límites de cuerpo
son los de las rutas servidoras: 16 KiB para detención y salida, 2 MiB para
orden y 12 MiB para entrada. Los métodos de DTO siguen disponibles para un
primer intento cuando el consumidor ya ha persistido el cuerpo producido por
el codificador correspondiente.

Las sesiones usan exclusivamente las rutas locales v1:

- `POST /v1/ejecuciones/{ref}/sesiones`;
- `POST /v1/ejecuciones/{ref}/sesiones/{sesion_ref}/entradas`;
- `GET /v1/ejecuciones/{ref}/sesiones/{sesion_ref}/eventos`.

`IniciarSesion` y `EnviarEntradaSesion` exigen una clave de idempotencia. La
identidad de la entrada es esa misma clave; no existe otra en el cuerpo.
`LeerEventosSesion` recibe cerca, cursor `despues_de` y límite de página. El
cliente verifica antes del socket referencias, revisiones, cerca, base64 y
límites; después verifica JSON estricto, identidad de ejecución/sesión,
continuidad del cursor, revisiones, terminalidad y bloques de eventos.

Ejemplo mínimo:

```go
cliente, err := microvm.Nuevo("/run/agente-microvm/api.sock")
if err != nil {
	return err
}
defer cliente.LiberarConexiones()

capacidades, err := cliente.Capacidades(ctx)
```

Ejemplo de sesión:

```go
sesion, err := cliente.IniciarSesion(ctx, "inicio:estable", "ejecucion:1",
	microvm.SolicitudIniciarSesionTrabajoV1{
		SesionRef:               "sesion:1",
		RevisionEsperada:        2,
		RevisionTrabajoEsperada: 1,
		Cerca:                   7,
		EjecutorRef:             descriptor.EjecutorRef,
		DirectorioTrabajo:       ".",
		PlazoTotalMilisegundos:  60_000,
		MaximoEventosBytes:      1_048_576,
	})
if err != nil {
	return err
}

pagina, err := cliente.LeerEventosSesion(ctx, sesion.EjecucionRef,
	sesion.SesionRef, microvm.ConsultaEventosSesionTrabajoV1{
		Cerca: sesion.Cerca, DespuesDe: 0, MaximoEventos: 64,
	})
```

`FirmanteConcesiones` traduce referencias autorizadas de sesión, artefactos,
MCP y buzón a un digest causal, fija audiencia, `RunRef`, cerca y vigencia, y
firma Ed25519 con una clave privada inyectada. Solo el identificador público de
la clave cruza el contrato; no se exportan el secreto, `CredentialStore` ni
tipos internos de Orquesta. El vector portable se verifica también desde Rust.

El lanzamiento final sigue recibiendo `plan` y `concesion` como
`json.RawMessage`. El adaptador no accede a SQLite, al CAS ni al runtime de
Firecracker, y un reintento debe conservar exactamente contexto, instante y
vigencia para reproducir la misma concesión.

`CalcularRaizContenido` verifica base64, SHA-256, rutas, duplicados y límites
y produce la raíz canónica compartida con el huésped. `SincronizarEntrada` y
`SincronizarSalida` aplican esa verificación automáticamente; una respuesta
corrupta nunca llega al consumidor como resultado correcto.

Verificación local:

```bash
go test ./...
go test -race ./...
go vet ./...
```
