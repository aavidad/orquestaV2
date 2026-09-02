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
