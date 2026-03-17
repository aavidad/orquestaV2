BINARY   = orquesta
INSTALL  = $(HOME)/.local/bin/$(BINARY)
LDFLAGS  = -s -w

.PHONY: build install clean test

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
