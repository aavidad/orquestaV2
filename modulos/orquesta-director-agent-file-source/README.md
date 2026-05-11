# orquesta-director-agent-file-source

Adaptador periferico para convertir artefactos JSON del director en decisiones compactas de workflow.

Responsabilidad:
- recibir descriptores de artefactos por un puerto;
- leer el contenido por un puerto de lectura;
- decodificar una decision, una lista o un sobre versionado;
- validar cada decision con `orquesta-director-agent`;
- exponerlas como `DirectorAgentDecisionSourcePortV0`.

Fuera de alcance:
- elegir proveedor, modelo, HOME, identidad o cuota;
- escanear directorios por defecto;
- persistir estado;
- ejecutar agentes;
- conocer detalles internos del nucleo.

Prueba local:

```bash
go test ./modulos/orquesta-director-agent-file-source -count=1
```
