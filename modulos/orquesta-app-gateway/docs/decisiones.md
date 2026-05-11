# Decisiones: orquesta-app-gateway

## D-001: composition root sin servidor

El modulo devuelve `http.Handler` y no ejecuta `ListenAndServe`.

Motivo: abrir sockets, leer env o seleccionar puertos pertenece a un adaptador
superior. Aqui solo se ensamblan contratos.

## D-002: REST in-process para web

La web usa sus clientes REST contra un `apiMux` interno mediante un
`http.RoundTripper` in-process.

Motivo: conserva la frontera REST y evita loopback real, dependencia circular y
acceso directo de web al core.

## D-003: nil executor significa endpoint disponible pero no configurado

Los bridges MCP se montan aunque falte executor; el propio bridge devuelve error
publico 503.

Motivo: el operador ve una frontera estable y un fallo publico de configuracion,
no un 404 ambiguo.
