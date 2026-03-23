# Multi-stage Dockerfile para Orquesta
# Autor: OSL - Diputación de Granada (Plataforma Municipal)

# --- Fase de construcción (Build) ---
FROM golang:1.21-alpine AS builder

# Instalamos dependencias básicas para Go/SQLite si hiciera falta (modernc no las requiere normalmente)
RUN apk add --no-cache git gcc musl-dev

WORKDIR /app

# Copiamos primero dependencias para cachear capas
COPY go.mod go.sum ./
RUN go mod download

# Copiamos el resto del código
COPY . .

# Construimos el binario estático
RUN CGO_ENABLED=0 go build -o orquesta-bin main.go

# --- Fase final (Run) ---
FROM alpine:latest

# Añadimos certificados para conexiones HTTPS salientes si fuera necesario (MCP/API)
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copiamos el binario desde la fase anterior
COPY --from=builder /app/orquesta-bin /app/orquesta

# Creamos directorio para persistencia de datos y logs
RUN mkdir -p /app/data /app/logs

# Variables de entorno por defecto
# Se recomienda sobreescribir ORQUESTA_DB_PATH al arrancar el contenedor
ENV ORQUESTA_DB_PATH=/app/data/orquesta.db
ENV PORT=8080

# Exponemos el puerto del panel web / API
EXPOSE 8080

# Punto de entrada por defecto: arrancar el servidor
# Podríamos permitir que el usuario pase argumentos personalizados
ENTRYPOINT ["/app/orquesta", "serve", "--puerto", "8080"]
