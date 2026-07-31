# Manual para crear una aplicación de orquestación

Responsabilidad: este índice organiza el conocimiento necesario para diseñar,
construir, probar, operar y continuar una aplicación de orquestación como
Orquesta.

Alcance: sirve a una persona o agente que empiece desde cero, continúe un
repositorio existente o audite una supuesta finalización.

No acredita: este manual no implementa ninguna capacidad, no cambia el catálogo
del producto y no demuestra que Orquesta esté terminada.

## Al terminar este índice, el lector sabrá…

- elegir la ruta de lectura según empiece, continúe o audite un orquestador;
- distinguir el manual de las autoridades que fijan el estado del producto;
- localizar el glosario común y la responsabilidad de cada capítulo;
- comprobar qué significa realmente que una capacidad esté acreditada.

## 1. Propósito

Una aplicación de orquestación recibe una intención, la convierte en trabajo
causal, coordina agentes y herramientas, gobierna efectos, conserva artefactos
y solo cierra cuando el resultado queda acreditado.

El manual existe para que esa definición no dependa de la memoria de un agente
ni se degrade al pasar de una versión a otra. Reúne:

- el modelo mínimo que debe existir;
- las fronteras que no deben cruzarse;
- la secuencia causal de construcción;
- las pruebas capaces de distinguir una implementación real de una simulación;
- los fallos históricos que no deben repetirse;
- el procedimiento para reanudar el trabajo sin inventar estado;
- el mapa que permite convertir ausencias en microtareas.

## 2. Autoridad

Este manual es explicativo. La hoja de ruta autoritativa es siempre la base
versionada `HEAD:product/roadmap.json`, que se lee con
`git show HEAD:product/roadmap.json`. La copia de trabajo
`product/roadmap.json` puede contener una propuesta aún no integrada y no
sustituye esa autoridad. Si el manual discrepa con una fuente vigente,
prevalece este orden:

1. [`AGENTS.md`](../../../AGENTS.md);
2. la base versionada `HEAD:product/roadmap.json`;
3. [`product/capabilities.json`](../../../product/capabilities.json) y
   [`product/evidence/`](../../../product/evidence/);
4. [`ruta_total_100.md`](../ruta_total_100.md);
5. los `AGENTS.md` locales;
6. este manual;
7. el legado, únicamente como fuente histórica de caracterización.

Una contradicción detiene solo el conjunto de escritura afectado. Se resuelve
en la autoridad correspondiente antes de programar esa parte.

## 3. Estado honesto

El manual usa siempre estos estados:

```text
declarada -> implementada -> conectada -> ejercitada -> acreditada
```

Solo `acreditada` cuenta como terminada. Además:

- una prueba unitaria no acredita una composición real;
- un adaptador alcanzable no demuestra un efecto;
- una respuesta de un agente no prueba que el trabajo esté bien;
- un proceso detenido no prueba que sus datos estén inventariados;
- un documento no crea una capacidad;
- un recibo de otra revisión no acredita el candidato actual;
- una capacidad histórica no se hereda por nombre o parecido.

## 4. Manifiesto del manual

| Capítulo | Responsabilidad |
|---|---|
| [01](01_mision_limites_y_vocabulario.md) | Misión, límites, glosario común y criterios de corrección. |
| [02](02_nucleo_lifecycle_y_dag.md) | Núcleo, `Goal`, DAG, planificador y Director. |
| [03](03_estado_transacciones_y_recuperacion.md) | Estado durable, transacciones, recuperación y contrato objetivo V31 multianfitrión. |
| [04](04_gestion_elastica_de_agentes.md) | Demanda, agentes elásticos, cuota, salud y retirada individual. |
| [05](05_aislamiento_firecracker_y_microvm.md) | Aislamiento físico, Firecracker y una microVM por agente. |
| [06](06_seguridad_configuracion_identidad_y_efectos.md) | Configuración, secretos, identidad, permisos y efectos. |
| [07](07_colaboracion_consejo_revision_y_contexto.md) | Buzón causal, subagentes, Consejo, revisión y contexto. |
| [08](08_superficies_i18n_y_contratos_de_aplicacion.md) | HTTP, MCP, línea de órdenes, web, internacionalización y manifiestos de aplicaciones. |
| [09](09_proceso_de_construccion_pruebas_y_acreditacion.md) | Construcción por cortes, pruebas, revisión y acreditación. |
| [10](10_operacion_observabilidad_y_continuidad.md) | Operación, diagnóstico, parada, actualización y continuidad. |
| [11](11_lecciones_antipatrones_y_diagnostico.md) | Lecciones históricas, antipatrones y guía de diagnóstico. |
| [12](12_auditoria_y_mapa_de_completitud.md) | Auditoría total, inventario y conversión de huecos en tareas. |

El fichero
[`manual_manifest.json`](manual_manifest.json) fija de forma mecánica el
conjunto de capítulos y sus temas obligatorios. Su prueba estructural impide
eliminar silenciosamente una responsabilidad, pero no acredita el producto.
El [glosario técnico común](01_mision_limites_y_vocabulario.md#glosario-técnico-común)
evita mezclar palabras castellanas e inglesas para el mismo concepto.

El manifiesto distingue literalmente la base autoritativa
`HEAD:product/roadmap.json` de la copia de trabajo local
`product/roadmap.json`. Esta última solo se comprueba como propuesta cuando
difiere de `HEAD`; su presencia nunca le concede autoridad ni acreditación.

## 5. Tres formas de usar el manual

### 5.1 Empezar desde cero

1. leer los capítulos 01, 02 y 03;
2. crear catálogo y contratos de aceptación antes del producto;
3. construir el monolito modular de dentro hacia fuera;
4. acreditar primero estado, causalidad y recuperación;
5. añadir agentes mediante puertos neutrales;
6. añadir seguridad y efectos antes de permitir mutaciones externas;
7. añadir superficies sobre los mismos casos de uso;
8. incorporar aislamiento fuerte como adaptador;
9. ejecutar la auditoría del capítulo 12.

No se empieza por una interfaz web, un proveedor de modelos o un lanzador de
procesos. Sin autoridad durable y causalidad, esas piezas crean otro conjunto
de automatizaciones, no un orquestador fiable.

### 5.2 Continuar una implementación

1. leer las instrucciones del repositorio y el último traspaso;
2. consultar estado Git, procesos, disco y tareas vivas;
3. comparar el estado acreditado con el catálogo, no con documentos narrativos;
4. identificar la siguiente dependencia causal;
5. consultar lecciones históricas por capacidad, ruta y operación;
6. declarar un conjunto de escritura estrecho;
7. ejecutar una microtarea y su contrarrevisión;
8. actualizar evidencia y traspaso sin atribuir más de lo demostrado.

### 5.3 Auditar una supuesta finalización

La auditoría empieza intentando refutar estas afirmaciones:

- existe un único `Goal` autoritativo;
- solo aplicación escribe su ciclo de vida;
- el planificador deriva el conjunto listo completo;
- los agentes se crean y retiran sin techo oculto ni borrado;
- un reinicio no duplica ejecuciones ni efectos;
- cada efecto conserva intención, aprobación, intento y recibo;
- las superficies comparten comandos, autorización e i18n;
- las pruebas ejercen la composición y el candidato exactos;
- no quedan procesos, escrituras dobles o dependencias del legado;
- cada capacidad acreditada tiene evidencia de la misma revisión.

Si una afirmación depende de “debería”, “parece”, una captura o un fichero
presente, se considera no demostrada.

## 6. Reglas de mantenimiento

Al modificar el manual:

- se preserva el orden de autoridad;
- se enlaza la capacidad o decisión relevante;
- se distingue regla, recomendación, estado actual y deuda;
- se añade la lección a la prueba estructural cuando sea universal;
- no se copia el catálogo de 257 capacidades como una segunda autoridad;
- no se presenta una propuesta histórica como diseño vigente;
- no se reescribe una sección para esconder un fallo observado;
- se mantiene cada capítulo con una responsabilidad localizable.

## 7. Resultado esperado

Después de leer y aplicar el manual, otro agente debe poder responder:

1. qué autoridad decide cada transición;
2. dónde se persiste y cómo se recupera;
3. cómo se calcula y satisface la demanda de agentes;
4. cómo se detiene uno sin dañar los demás;
5. qué datos se preservan y quién puede retirarlos;
6. cómo se gobiernan secretos, permisos y efectos;
7. cómo colaboran autores, revisores, consejo y Director;
8. qué prueba acredita cada promesa;
9. cómo se diagnostica un atasco;
10. qué falta realmente para terminar.
