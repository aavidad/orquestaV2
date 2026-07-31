# 01. Misión, límites y vocabulario

Responsabilidad: define qué problema resuelve una aplicación de orquestación,
qué entidades forman su núcleo y qué propiedades deben permanecer invariantes.

Alcance: lenguaje común para diseño, implementación, pruebas, operación y
auditoría.

No acredita: describir estas propiedades no demuestra que una implementación
las cumpla.

## Al terminar este capítulo, el lector sabrá…

- explicar qué diferencia a un orquestador de un lanzador de procesos;
- identificar las entidades y autoridades canónicas;
- usar un vocabulario técnico común sin mezclar términos equivalentes;
- enumerar los invariantes que permiten considerar correcto el núcleo.

## Glosario técnico común

Este glosario fija la forma castellana que usa todo el manual. El término entre
comillas invertidas solo se conserva cuando aparece como identificador literal
en código, contratos o documentación externa:

- **recibo (`receipt`):** hecho estructurado que acredita una admisión, intento,
  observación o efecto concretos;
- **huella criptográfica (`digest` o `hash`):** resumen verificable que liga
  contenido inmutable sin copiarlo;
- **arrendamiento (`lease`):** concesión temporal y exclusiva que caduca;
- **cerca (`fence`):** ordinal monotónico que impide escribir a un poseedor
  antiguo de un arrendamiento;
- **buzón causal (`mailbox`):** entrega durable y dirigida que liga emisor,
  destinatario, sujeto y resolución;
- **ciclo de vida (`lifecycle`):** conjunto autoritativo de estados y
  transiciones de una entidad;
- **grafo dirigido acíclico (DAG):** conjunto de trabajos y dependencias
  orientadas que no permite volver causalmente al punto de partida.

## 1. Qué es una aplicación de orquestación

Es un sistema que:

1. recibe una intención atribuida a una identidad y un proyecto;
2. la normaliza en una especificación confirmable;
3. mantiene un objetivo durable;
4. descompone el objetivo en un grafo dirigido acíclico (DAG) de trabajo;
5. determina qué trabajo está listo por causalidad;
6. coordina agentes, herramientas y conectores por puertos;
7. gobierna capacidad, presupuestos, permisos y efectos;
8. conserva artefactos, decisiones, pruebas y recibos;
9. recupera el trabajo tras fallos sin duplicarlo;
10. cierra únicamente cuando el resultado queda acreditado.

La propiedad esencial no es “lanzar muchos procesos”. Es conservar una decisión
causal y verificable desde la intención hasta el resultado.

## 2. Qué no es

No es:

- un guion que abre terminales;
- una cola por proveedor;
- un conjunto de agentes que se envían texto libre;
- una interfaz web que refleja procesos;
- un plan almacenado en la memoria del Director;
- un bucle de reintentos que interpreta registros;
- un enjambre sin autoridad de identidad y estado;
- una herramienta de integración continua renombrada;
- un motor distinto por fase, proveedor o dominio;
- un conjunto de microservicios creados antes de tener fronteras reales.

Esas piezas pueden ser adaptadores o consumidores. Ninguna sustituye al núcleo.

## 3. Entidades canónicas

### 3.1 Intención

Petición atribuida y normalizada. Contiene referencias opacas, no secretos ni
rutas físicas. Antes de confirmarse puede requerir preguntas, recomendaciones
o una vista previa. No posee un ciclo de vida paralelo.

### 3.2 `AppSpec`

Especificación confirmada de la aplicación o trabajo solicitado. Tiene
generación y resumen canónico. Una modificación posterior crea una nueva
generación causal; no reescribe silenciosamente el pasado.

### 3.3 `Goal`

Única autoridad de identidad, generación y ciclo de vida del objetivo. Solo el
escritor de aplicación puede transicionarlo. Una ejecución, un evento, una
cola, un panel o un Director no pueden cerrar, cancelar o reabrirlo por sí
solos.

### 3.4 `WorkItem`

Unidad del DAG interno del `Goal`. Declara objetivo local, dependencias,
escrituras permitidas, requisitos, presupuesto, pruebas y contrato de salida.
No es otro agregado de mando.

### 3.5 `PhaseInstance`

Metadato causal inmutable para agrupar trabajo. Su progreso se deriva de
`WorkItem` y recibos. No mantiene otra máquina de estados.

### 3.6 `Execution`

Intento concreto de ejecutar un `WorkItem`. Conserva identidad, intento,
generaciones, cercado, proveedor observado, recibos y tiempos. No decide el
estado terminal del `Goal`.

### 3.7 Acción

Trabajo reclamable que el motor ya ha validado: lanzar, observar, detener,
preparar espacio, integrar, atestar, entregar un mensaje u otra operación
registrada. Una acción es un intento operativo, no una autoridad superior.

### 3.8 Artefacto

Contenido inmutable y direccionado por resumen. Los objetos grandes viajan por
referencia. La aparición de un artefacto no demuestra por sí sola procedencia,
calidad ni aceptación.

### 3.9 Evidencia y recibo

Un recibo acredita que una frontera observó o realizó una operación exacta. La
evidencia liga sujeto, revisiones, configuración, binario o imagen, pruebas y
decisiones. Ninguno debe vivir dentro del mismo sujeto cuyo resumen pretende
contener.

## 4. Autoridades

| Decisión | Autoridad |
|---|---|
| Identidad y ciclo de vida | `Goal` escrito por aplicación |
| Trabajo listo | DAG y estado durable |
| Reclamación | repositorio transaccional con arrendamiento y cercado |
| Política | motor de aplicación |
| Propuesta de plan | Director con arrendamiento |
| Ejecución externa | adaptador detrás de puerto |
| Autenticación | adaptador que produce principal |
| Autorización | aplicación antes del caso de uso |
| Configuración | registro canónico y proyección tipada |
| Secretos | `CredentialStore` |
| Integración Git | caso de uso y adaptador con comparación causal |
| Acreditación | contrato, pruebas y evidencia del candidato exacto |

Si dos piezas pueden decidir la misma transición, el diseño está roto aunque
ambas coincidan durante las pruebas felices.

## 5. Arquitectura física predeterminada

La forma inicial es un monolito modular:

```text
interfaces -> aplicación -> núcleo
adaptadores -> puertos <- aplicación/núcleo
bootstrap  -> composición explícita
```

Reglas:

- un binario productivo principal;
- una fuente transaccional de estado activa por despliegue;
- un almacén de artefactos elegido por composición;
- un registro de comandos;
- un registro de configuración;
- proveedores y tecnologías fuera del núcleo;
- procesos separados solo por aislamiento, seguridad o tecnología;
- módulos pequeños por responsabilidad, no un fichero por símbolo.

## 6. Referencias opacas

Identidades como actor, proyecto, objetivo, ejecución, artefacto o repositorio
se transportan explícitamente. La aplicación:

- no infiere usuario desde variables globales;
- no deriva proyecto de una ruta;
- no acepta identidad de un cuerpo no autenticado;
- no expone credenciales en una referencia;
- no reutiliza una referencia de otra generación;
- no convierte un identificador externo en autoridad interna sin validarlo.

## 7. Hechos frente a proyecciones

Estado durable, decisiones y recibos son hechos. Paneles, cronologías, colas
visuales, contadores y búsquedas son proyecciones.

Una proyección puede retrasarse, reconstruirse o descartarse. Nunca debe:

- transicionar un `Goal`;
- liberar capacidad;
- decidir que un proceso murió;
- aprobar un efecto;
- ocultar una contradicción terminal;
- convertirse en segunda fuente de recuperación.

## 8. Invariantes de corrección

Una implementación correcta conserva al menos:

1. un ciclo de vida y un escritor;
2. una generación causal por cambio;
3. un DAG sin dependencias desconocidas ni ciclos;
4. un conjunto listo derivado, no mantenido en otra cola;
5. una reclamación exclusiva, acotada y cercada;
6. repetición causal por clave, no por tiempo;
7. ningún efecto sin identidad, permiso, alcance y presupuesto;
8. ninguna credencial en estado, texto o artefacto;
9. ningún cierre por respuesta textual del agente;
10. ninguna acreditación sin sujeto inmutable;
11. ningún borrado automático de datos pendientes de revisión;
12. ningún proveedor dentro del núcleo;
13. ninguna dependencia operativa del legado;
14. ninguna interfaz pública con reglas distintas;
15. ningún fallo local que congele trabajo causalmente independiente.

## 9. Presupuestos de diseño

Cada capacidad declara límites de:

- líneas y complejidad;
- escritores;
- bucles de fondo;
- almacenes;
- comandos públicos;
- CPU, memoria, disco, red y procesos;
- latencia y tiempo de recuperación;
- contexto y coste de modelos;
- alcance de permisos y secretos.

Superar un límite exige una decisión y, cuando proceda, retirar complejidad
equivalente. La ausencia de un límite no autoriza crecimiento indefinido.

## 10. Criterio de finalización

Una aplicación de orquestación está completa solo cuando:

- el catálogo exhaustivo no tiene capacidades pendientes;
- cada dependencia causal está satisfecha;
- la composición productiva pasa sus pruebas reales;
- recuperación, copia y restauración están ejercitadas;
- seguridad e aislamiento pasan negativos;
- no quedan procesos, escrituras dobles ni consumidores del legado;
- cada evidencia corresponde al mismo árbol, binario o imagen y configuración;
- el control global se repite desde una copia limpia.

Una cifra de líneas, pruebas, agentes o días no reemplaza este criterio.
