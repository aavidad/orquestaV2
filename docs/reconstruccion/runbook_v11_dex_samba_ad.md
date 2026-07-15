# Smoke real V11: Dex sobre Samba AD por LDAPS

## Propósito y frontera

Este smoke opt-in acredita la frontera externa de `AC-V11-OIDC-AD`:

```text
Samba AD -- LDAPS validado --> Dex -- OIDC/PKCE --> IdentityProvider OIDC de Orquesta
```

Orquesta sigue siendo un resource server OIDC sin LDAP, contraseña de
directorio, sesión de navegador ni mapeo automático de grupos a RBAC. Dex es
el bridge reemplazable. Los grupos externos solo permiten o deniegan admisión;
los roles de proyecto siguen en el estado V10.

El probe construye el adaptador OIDC productivo con issuer, audiencia y grupo
requerido, y autentica el token firmado real. No atraviesa por sí solo
bootstrap, middleware bearer ni MCP: selección de provider y binding de
request quedan en la suite integrada de `AC-V11-OIDC-AD`. Por eso este smoke no
se presenta como E2E de servidor instalado. Dex publica OIDC por HTTP solo en
loopback aislado; el loader de producción exige issuer HTTPS. La pata de
directorio sí usa LDAPS con verificación estricta y CA propia.

El cliente HTTP productivo de discovery/JWKS es propio: usa el
`identity.oidc.upstream_timeout` del registro canónico (máximo 30 segundos),
no hereda `http.DefaultClient`, transport global ni proxies de entorno y limita
conexiones y cada fase de red. Un cliente HTTP inyectado solo existe como seam
explícito de composición/prueba.

El smoke no acredita un tenant AD/Entra/AD FS concreto ni operación permanente
de Dex. Esa validación corresponde a instalación. Tampoco añade capabilities:
V11 es una vertical transversal y posee cero IDs del catálogo.

Referencias primarias:

- [Dex: conector LDAP, LDAPS, CA y ejemplo Active Directory](https://dexidp.io/docs/connectors/ldap/)
- [Dex: clientes públicos](https://dexidp.io/docs/configuration/custom-scopes-claims-clients/)
- [Samba: parámetros TLS del AD DC](https://www.samba.org/samba/docs/current/man-html/smb.conf.5.html)

## Componentes fijados

- Dex oficial `v2.45.1`:
  `ghcr.io/dexidp/dex@sha256:8499afd690c437f52301efd2b05b2455da5bd2dfc20332cd697dc9937f808462`.
- Debian oficial:
  `debian@sha256:60eac759739651111db372c07be67863818726f754804b8707c90979bda511df`.
- Paquetes Samba Debian fijados a
  `2:4.17.12+dfsg-0+deb12u4` dentro de la imagen efímera.

No se usa imagen Samba comunitaria. Debian oficial aporta los binarios Samba;
es el límite externo inevitable y queda explícito. Docker conserva las capas
base en su cache normal, pero el script elimina imagen Samba derivada,
contenedores, red y runtime propios.

## Requisitos

- Docker operativo para el usuario actual;
- `go`, `python3`, `openssl`, `curl` y `grep`;
- vendor Go vigente;
- 3 GiB libres en `TMPDIR`;
- acceso de red al mirror Debian solo si la capa Samba no está en cache.

Samba AD necesita escribir ACL `security.NTACL` al provisionar SYSVOL. Solo su
contenedor temporal recibe `CAP_SYS_ADMIN`. Dex corre sin privilegios con UID/GID
del operador. Ningún puerto se publica fuera de `127.0.0.1`.

## Ejecución

Desde la raíz del rebuild:

```bash
./scripts/smoke_v11_dex_samba_ad.sh
```

El script tiene esperas acotadas, timeout por request y falla si Samba muere o
Dex no publica discovery. Genera CA, certificado y contraseñas aleatorias bajo un directorio
`0700`. El bind de Dex vive solo en el YAML temporal de Dex montado de solo
lectura; no entra en configuración Orquesta. ID tokens quedan en memoria del
harness y pasan al probe Go por stdin: no se escriben en fichero, argumentos,
logs ni evidencia.

En fallo, se muestran como máximo 80 líneas por contenedor y se sustituyen los
cuatro secretos generados por `[REDACTED]`. El trap elimina siempre los recursos
con nombres propios de la ejecución; nunca toca contenedores, redes o imágenes
ajenas.

## Aserciones ejercitadas

1. Cadena CA/SAN válida abre LDAPS y una CA distinta falla cerrada.
2. Login real de `alice` por formulario Dex produce Authorization Code y el
   cliente público lo canjea con PKCE `S256`, state y nonce.
3. Token firmado contiene `orquesta-users` y el adaptador OIDC productivo lo
   acepta con `RequiredGroups`.
4. Dos logins producen el mismo `sub`, `PrincipalRef` y `ActorRef`.
5. Contraseña errónea no produce callback/código.
6. Usuario AD deshabilitado no produce callback/código.
7. Tras retirar `alice` de `orquesta-users`, un token nuevo carece del grupo y
   el adaptador devuelve `oidc.group_denied`.
8. Token ya emitido antes de retirar el grupo sigue válido solo hasta su
   expiración firmada, coherente con resource server stateless.
9. Cleanup deja cero contenedores, redes y directorios runtime propios.

## Operación y límites explícitos

- Orquesta no convierte grupos OIDC en roles. Antes de cambiar una instalación
  existente de `local_token` a `oidc`, el propietario local debe crear por el
  camino RBAC auditado la membresía del `PrincipalRef` estable derivado de
  `oidc + issuer + sub`. La superficie guiada/multicanal para ese alta pertenece
  al Wizard/Web de V23–V24; V11 entrega proveedor, composición y pruebas, no un
  segundo comando administrativo ad hoc. Así no habrá CLI temporal que luego
  deba convertirse en legacy.
- El `RemoteKeySet` de go-oidc refresca ante un `kid` nuevo. La retirada urgente
  de un `kid` ya cacheado requiere reiniciar Orquesta tras retirar la clave del
  IdP; la expiración firmada limita tokens ya emitidos, pero V11 no afirma una
  lista de revocación local ni introspección.
- Tokens basura con muchos `kid` distintos pueden provocar consultas JWKS. La
  instalación actual permanece en loopback. Antes de exposición remota, el
  despliegue/operación debe aportar rate limit frontal y prueba de abuso; no se
  añade aquí un daemon o cache criptográfica casera.
- El IdP debe limitar la vida del ID token conforme a su política. El adaptador
  exige y valida `iat`, `nbf` y `exp`, pero no reescribe la duración firmada.

Salida final esperada:

```text
PASS ldaps_valid_chain_and_invalid_ca_rejected
PASS valid_login_and_required_group_admission
PASS stable_subject_principal_and_actor
PASS wrong_password_rejected
PASS disabled_user_rejected
PASS removed_group_rejects_new_token_old_token_lives_to_expiry
V11_DEX_SAMBA_AD_SMOKE=PASS
PASS cleanup_no_container_network_or_runtime_residue
```

## Evidencia de ejecución

Ejecutado con resultado verde el 2026-07-15 en el equipo de reconstrucción,
usando exactamente los dos digests y la versión Samba indicados arriba. El
output completo fue el bloque esperado. Después se verificó ausencia de
contenedores/red/runtime con prefijo propio. Esta nota orienta al operador; el
receipt sellado de V11 es la autoridad de acreditación de release.

## Diagnóstico rápido

- `missing command`: instalar solo el requisito nombrado y volver a ejecutar.
- `Samba AD container exited during startup`: revisar bloque redacted; no
  ampliar privilegios más allá de `CAP_SYS_ADMIN`.
- timeout de Dex: comprobar digest disponible, puerto loopback y CA montada.
- `v11.oidc_callback_missing`: revisar log Dex redacted; suele señalar bind DN,
  credencial, usuario deshabilitado o TLS.
- fallo de cleanup: no borrar recursos por patrón amplio; usar los nombres
  exactos que imprime el bloque de fallo.
