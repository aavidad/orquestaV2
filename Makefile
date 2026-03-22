BINARY   = orquesta
INSTALL  = $(HOME)/.local/bin/$(BINARY)
LDFLAGS  = -s -w
POSTGRES_BOOTSTRAP_TEST_SCRIPT = ./scripts/test-postgres-bootstrap.sh

.PHONY: build install clean test test-postgres-bootstrap

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

install: build
	mkdir -p $(HOME)/.local/bin
	cp $(BINARY) $(INSTALL)
	@echo "✓ Instalado en $(INSTALL)"

clean:
	rm -f $(BINARY)

test:
	go test ./...

test-postgres-bootstrap:
	$(POSTGRES_BOOTSTRAP_TEST_SCRIPT)
