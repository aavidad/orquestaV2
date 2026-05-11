# Contexto local: orquesta-app-director-service

Lee primero este fichero y los documentos de `docs/`.

Reglas locales:

- servicio de aplicacion: compone factory, intake de director y nucleo;
- todos los efectos externos entran por puertos inyectados;
- REST, MCP, web y CLI son adaptadores, no viven aqui;
- no elegir proveedor, modelo, credenciales, HOME, runtime concreto ni DB;
- no importar `cmd` ni `db`;
- ficheros pequenos y pruebas focales.
