BINARY      := phiocker
BINARY_DST  := /usr/local/bin/$(BINARY)
SERVICE     := $(BINARY).service
SYSTEMD_DIR := /etc/systemd/system

.PHONY: build install-service uninstall-service start-service stop-service status-service

build:
	go build -o $(BINARY_DST) ./cmd/phiocker

install-service: build
	sudo bash install-service.sh install

uninstall-service:
	sudo bash install-service.sh uninstall

start-service:
	sudo systemctl start $(BINARY)

stop-service:
	sudo systemctl stop $(BINARY)

status-service:
	systemctl status $(BINARY)

help:
	@grep -E '^## ' Makefile | sed 's/## //'
