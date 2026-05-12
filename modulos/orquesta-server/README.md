# orquesta-server

Servidor residente de Orquesta.

Este modulo permite ejecutar Orquesta como proceso independiente de la consola
que lo lanza. Expone `healthz`, `/api/status` y delega el resto del trafico al
handler de aplicacion inyectado.

El supervisor global se ejecuta por pulsos acotados mediante un puerto. Si el
usuario cierra la sesion de Codex, el daemon sigue vivo y el operador puede
reengancharse leyendo el statefile y consultando la API.

