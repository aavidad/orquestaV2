# syntax=docker/dockerfile:1

FROM golang:1.25-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/orquesta-server ./cmd/orquesta-server

FROM node:22-bookworm-slim AS runtime
ARG CODEX_NPM_VERSION=0.144.1
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tmux \
    && npm install -g "@openai/codex@${CODEX_NPM_VERSION}" \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --system --gid 10001 orquesta \
    && useradd --system --uid 10001 --gid 10001 --home-dir /workspace/home --create-home --shell /usr/sbin/nologin orquesta \
    && install -d -o 10001 -g 10001 /workspace/home /workspace/codex-home \
    && rm -f /usr/local/bin/npm /usr/local/bin/npx /usr/local/bin/corepack \
    && rm -rf /usr/local/lib/node_modules/npm
COPY --from=builder /out/orquesta-server /usr/local/bin/orquesta-server
ENV HOME=/workspace/home \
    CODEX_HOME=/workspace/codex-home \
    PATH=/usr/local/bin:/usr/bin:/bin
USER 10001:10001
ENTRYPOINT ["/usr/local/bin/orquesta-server"]
CMD ["run"]
