# Decisiones: orquesta-http-gateway

## D-001: mux fino sobre `net/http`

Se usa `http.NewServeMux` y handlers `net/http` inyectados.

Motivo: el gateway debe ser estable y no debe acoplarse a frameworks, runtime,
DB, web, MCP, factory ni cmd legacy.

## D-002: rutas sin handler no se registran

Una dependencia ausente deja su ruta sin configurar y por tanto responde 404.

Motivo: permite componer superficies parciales sin falsos endpoints ni stubs de
negocio dentro del gateway.

## D-003: sin filtro de metodo en el gateway

El mux no restringe metodos HTTP.

Motivo: el handler inyectado es el propietario del contrato operativo de cada
endpoint. El gateway solo decide presencia de ruta.

## D-004: panel de stats separado de API stats

`/director-stats` queda como ruta web y `/api/v0/director/stats` como REST API.

Motivo: la web necesita una superficie de operador y el director/MCP necesitan
un contrato JSON estable; mezclar ambos en la misma ruta fuerza acoplamiento y
dificulta pruebas verticales.

## D-005: manifiesto canonico de rutas

Las rutas publicas y overlays conocidos se declaran en `PublicRouteManifestV0`.
Las rutas exactas que viven bajo un prefijo registrado deben declarar el prefijo
que sombrean para que la precedencia sea intencional y testeable.

Motivo: evitar que nuevas rutas bajo `/api/v0/apps/` o overlays de composicion
cambien el dispatch por accidente.

## D-006: supervision es mutacion de control-plane

`/api/v0/runs/supervise` y `/api/v0/autoprogramming/supervise` se clasifican
como mutaciones publicas igual que indica el manifiesto.

Motivo: supervisar puede lanzar, drenar o avanzar trabajo; por tanto el guard
browser debe exigir mismo origen, cliente no-browser o token de intencion antes
de delegar al handler inyectado.
