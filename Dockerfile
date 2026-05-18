# syntax=docker/dockerfile:1

FROM golang:1.26-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/orquesta-server ./cmd/orquesta-server

FROM node:22-bookworm-slim AS runtime
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --system --gid 10001 orquesta \
    && useradd --system --uid 10001 --gid 10001 --home-dir /home/orquesta --create-home --shell /usr/sbin/nologin orquesta \
    && ln -s ../lib/node_modules/@openai/codex/bin/codex.js /usr/local/bin/codex \
    && rm -f /usr/local/bin/npm /usr/local/bin/npx /usr/local/bin/corepack \
    && rm -rf /usr/local/lib/node_modules/npm
COPY --from=builder /out/orquesta-server /usr/local/bin/orquesta-server
ENV HOME=/home/orquesta \
    CODEX_HOME=/home/orquesta/.codex \
    PATH=/usr/local/bin:/usr/bin:/bin
USER 10001:10001
ENTRYPOINT ["/usr/local/bin/orquesta-server"]
CMD ["run"]
