# orquesta-app-director-service

Servicio de aplicacion para arrancar una solicitud de app por director.

Flujo:

1. valida `AppSpecRequestV0` con factory;
2. prepara intake de director;
3. guarda el run inicial por `RunStorePortV0`;
4. compone observadores opcionales de entregas, progreso, leases y replans;
5. ejecuta el loop progresivo del nucleo con dispatchers inyectados;
6. devuelve un resultado compacto para REST/MCP/web.

El resultado incluye `director_tasks` para que la superficie publica pueda
mostrar el equipo preparado sin conocer scheduler, outbox ni decisiones internas.

No crea persistencia ni runtime por defecto. Todo efecto se inyecta. Si se
inyectan fuentes de observacion, el servicio las adapta a candidates del nucleo
sin que REST/MCP/web conozcan scheduler, outbox ni comandos internos.

Validacion:

```bash
go test -count=1 ./modulos/orquesta-app-director-service
```
