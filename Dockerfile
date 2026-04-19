# syntax=docker/dockerfile:1.7

# Multi-stage Dockerfile para Orquesta
# Perfil opcional de despliegue para el plano de control.

# --- Fase de construcción (Build) ---
FROM golang:1.25-alpine AS builder

# Git se mantiene para fallbacks de resolución de módulos fuera del proxy.
RUN apk add --no-cache git

WORKDIR /app

# Copiamos primero dependencias para cachear capas
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copiamos el resto del código
COPY . .

# Compilación reproducible con caches explícitas de módulos y build.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o orquesta-bin main.go

# --- Fase final (Run) ---
FROM alpine:latest

# Añadimos certificados para conexiones HTTPS salientes si fuera necesario (MCP/API)
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copiamos el binario desde la fase anterior
COPY --from=builder /app/orquesta-bin /app/orquesta

# Creamos directorios para logs y workspace montado.
RUN mkdir -p /app/logs /app/workspace

# Variables de entorno por defecto.
# El backend de persistencia debe declararse desde fuera del contenedor
# mediante ORQUESTA_DB_DRIVER + ORQUESTA_DB_DSN/ORQUESTA_DB.
ENV ORQUESTA_WORKSPACE_ROOT=/app/workspace

# Exponemos el puerto del panel web / API
EXPOSE 16543

# Punto de entrada por defecto.
# Se deja el comando separado para poder sobreescribirlo desde compose o docker run.
ENTRYPOINT ["/app/orquesta"]
CMD ["serve", "--puerto", "16543"]
