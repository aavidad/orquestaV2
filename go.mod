module orquesta

go 1.25.0

toolchain go1.25.11

require (
	github.com/aavidad/agente_microvm/conectores/orquesta v0.0.0-20260805221511-928ef309a173
	golang.org/x/text v0.38.0
)

require (
	github.com/coreos/go-oidc/v3 v3.20.0
	github.com/go-jose/go-jose/v4 v4.1.4
	github.com/google/jsonschema-go v0.4.3
	github.com/modelcontextprotocol/go-sdk v1.6.1
	github.com/pelletier/go-toml/v2 v2.4.3
	golang.org/x/oauth2 v0.36.0
	golang.org/x/sys v0.44.0
	modernc.org/sqlite v1.53.0
	pgregory.net/rapid v1.3.0
)

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/segmentio/asm v1.1.3 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	modernc.org/libc v1.73.4 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
)

replace pgregory.net/rapid => ./modulos/orquesta-estado-vivo/testdeps/rapid
